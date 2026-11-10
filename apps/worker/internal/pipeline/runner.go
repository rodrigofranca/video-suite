// Package pipeline runs one job end to end: it tracks the job's status in
// PostgreSQL and records its logs. Finishing a job records the job.finished
// event, and the outbox relay moves the pipeline on (docs/adr/0034).
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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
	Log        *slog.Logger
	Timeout    time.Duration // defaults to DefaultTimeout
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
		// it again; the pipeline moved on when it finished.
		r.log().Info("job already finished, skipping", "job_id", job.ID, "status", job.Status)
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
	// Saving the result also records the job.finished event, which settles
	// the asset and queues the next job.
	if err := r.Store.FinishJob(bg, job.ID, runErr); err != nil {
		r.log().Error("could not save job result", "job_id", job.ID, "error", err)
	}
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
		Progress: r.progressReporter(ctx, job.ID),
	})
}

// progressInterval is the shortest time between two progress writes.
const progressInterval = time.Second

// progressReporter returns a func that saves a job's progress, skipping
// repeated values and writing at most once per progressInterval.
func (r *Runner) progressReporter(ctx context.Context, jobID string) func(int) {
	var mu sync.Mutex
	last, lastAt := -1, time.Time{}
	return func(percent int) {
		mu.Lock()
		defer mu.Unlock()
		if percent == last || time.Since(lastAt) < progressInterval {
			return
		}
		if err := r.Store.SetJobProgress(context.WithoutCancel(ctx), jobID, percent); err != nil {
			r.log().Warn("could not save job progress", "job_id", jobID, "error", err)
			return
		}
		last, lastAt = percent, time.Now()
	}
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
