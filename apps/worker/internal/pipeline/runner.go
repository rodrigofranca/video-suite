// Package pipeline runs one job end to end: it tracks the job's status in
// PostgreSQL, records its logs, and starts the next pipeline step.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/jobs"
	"ultramedia/worker/internal/queue"

	jobcontracts "ultramedia/job-contracts"
)

// DefaultTimeout is the longest a job may run (see CONTEXT.md).
const DefaultTimeout = 30 * time.Minute

type Runner struct {
	Store      *store.Store
	Dispatcher *jobs.Dispatcher
	// Enqueue adds a message to the jobs stream.
	Enqueue func(ctx context.Context, msg jobcontracts.Message) error
	Log     *slog.Logger
	Timeout time.Duration // defaults to DefaultTimeout
}

// Process runs the job a queue message points to. The job row in PostgreSQL
// is the source of truth: the message only carries its ID.
func (r *Runner) Process(ctx context.Context, msg jobcontracts.Message) error {
	job, err := r.Store.GetJob(ctx, msg.JobID)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("job %s not found in database", msg.JobID)
	}
	if err != nil {
		return fmt.Errorf("%w: load job: %v", queue.ErrRetry, err)
	}
	log := r.jobLogger(job)

	started, err := r.Store.StartJob(ctx, job.ID)
	if err != nil {
		return fmt.Errorf("%w: start job: %v", queue.ErrRetry, err)
	}
	if !started {
		// Delivered twice (e.g. the worker stopped before acking). Do not run
		// it again, but make sure the pipeline moved on.
		r.log().Info("job already finished, skipping", "job_id", job.ID, "status", job.Status)
		var prevErr error
		if job.Status == domain.JobFailed {
			prevErr = errors.New("failed earlier")
		}
		r.advance(context.WithoutCancel(ctx), job, prevErr, log)
		return nil
	}

	log.Info("job started")
	runErr := r.run(ctx, job, log)

	if ctx.Err() != nil {
		// Worker shutting down: leave the job running and the message unacked,
		// so it runs again when the worker restarts.
		log.Warn("interrupted by worker shutdown; it will run again on restart")
		return fmt.Errorf("%w: %v", queue.ErrRetry, ctx.Err())
	}

	bg := context.WithoutCancel(ctx)
	if runErr != nil {
		log.Error("job failed: " + runErr.Error())
	} else {
		log.Info("job finished")
	}
	if err := r.Store.FinishJob(bg, job.ID, runErr); err != nil {
		r.log().Error("could not save job result", "job_id", job.ID, "error", err)
	}
	r.advance(bg, job, runErr, log)
	return runErr
}

// run calls the handler with the job timeout and turns a panic into an error.
func (r *Runner) run(ctx context.Context, job domain.Job, log *slog.Logger) (err error) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("handler panic: %v", p)
		}
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			err = fmt.Errorf("timed out after %s", timeout)
		}
	}()
	return r.Dispatcher.Dispatch(runCtx, jobs.Job{
		ID: job.ID, AssetID: job.AssetID, Type: job.Type, Payload: job.InputJSON, Log: log,
	})
}

// advance updates the asset and enqueues the next pipeline step. An asset is
// ready when its HLS job succeeds and failed when it fails (docs/adr/0003);
// metadata and thumbnail failures do not stop the pipeline (docs/adr/0009).
func (r *Runner) advance(ctx context.Context, job domain.Job, runErr error, log *slog.Logger) {
	if job.Type == jobcontracts.JobTypeTranscodeHLS {
		status := domain.AssetReady
		if runErr != nil {
			status = domain.AssetFailed
		}
		if err := r.Store.SetAssetStatus(ctx, job.AssetID, status); err != nil {
			r.log().Error("could not update asset status", "asset_id", job.AssetID, "error", err)
		}
		return
	}

	nextType, ok := jobcontracts.NextJobType(job.Type)
	if !ok {
		return
	}
	next, err := r.Store.NextPendingJob(ctx, job.AssetID, nextType)
	if errors.Is(err, store.ErrNotFound) {
		return // already started by an earlier delivery
	}
	if err != nil {
		log.Error("could not load next job: " + err.Error())
		return
	}
	msg := jobcontracts.Message{JobID: next.ID, Type: next.Type, Payload: next.InputJSON}
	if err := r.Enqueue(ctx, msg); err != nil {
		// Nothing would ever run the rest of the pipeline: fail it visibly.
		log.Error("could not enqueue " + next.Type + ": " + err.Error())
		r.Store.FinishJob(ctx, next.ID, fmt.Errorf("could not enqueue job: %w", err))
		r.Store.SetAssetStatus(ctx, job.AssetID, domain.AssetFailed)
		return
	}
	log.Info("queued next job", "next_type", next.Type, "next_job_id", next.ID)
}

func (r *Runner) log() *slog.Logger {
	if r.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return r.Log
}

// jobLogger writes to the worker output and to the job's rows in job_logs.
func (r *Runner) jobLogger(job domain.Job) *slog.Logger {
	next := r.log().With("job_id", job.ID, "type", job.Type).Handler()
	return slog.New(&jobLogHandler{next: next, store: r.Store, jobID: job.ID, fallback: r.log()})
}
