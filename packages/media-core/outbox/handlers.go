package outbox

import (
	"context"
	"errors"
	"fmt"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// What the platform does with each event. A job.finished event has two
// reactions, kept apart: the asset's status and the next job.

// Sender sends a job's message to its stream (Redis XADD).
type Sender func(ctx context.Context, msg jobcontracts.Message) error

// Handlers maps each event type to its reactions.
func Handlers(send Sender) map[string][]Handler {
	return map[string][]Handler{
		jobcontracts.EventJobQueued:   {Publish(send)},
		jobcontracts.EventJobFinished: {SettleAsset, QueueNext},
	}
}

// Publish sends a queued job to its stream. A job that is no longer pending
// was already sent and has run, or was deleted with its video: it is not
// sent again. A job may still be sent twice when the relay stops between
// sending and marking the event; the workers skip a job that already ended.
func Publish(send Sender) Handler {
	return func(ctx context.Context, tx store.EventTx, e domain.Event) error {
		job, err := tx.GetJob(ctx, e.JobID)
		if errors.Is(err, store.ErrNotFound) || (err == nil && job.Status != domain.JobPending) {
			return nil
		}
		if err != nil {
			return err
		}
		return send(ctx, jobcontracts.Message{JobID: job.ID, Type: job.Type, Payload: job.InputJSON})
	}
}

// SettleAsset makes the asset ready when its HLS job is done, and failed
// when it failed (docs/adr/0003). Other jobs leave the asset as it is.
func SettleAsset(ctx context.Context, tx store.EventTx, e domain.Event) error {
	job, err := tx.GetJob(ctx, e.JobID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && job.Type != jobcontracts.JobTypeTranscodeHLS) {
		return nil
	}
	if err != nil {
		return err
	}
	status := domain.AssetReady
	if job.Status == domain.JobFailed {
		status = domain.AssetFailed
	}
	return tx.SetAssetStatus(ctx, job.AssetID, status)
}

// QueueNext queues the asset's next pending job once a job ends, done or
// failed: the next step of the pipeline, or a job waiting for it
// (jobcontracts.NextJobType). A failure does not stop the pipeline
// (docs/adr/0009).
func QueueNext(ctx context.Context, tx store.EventTx, e domain.Event) error {
	job, err := tx.GetJob(ctx, e.JobID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	nextType, ok := jobcontracts.NextJobType(job.Type)
	if !ok {
		return nil
	}
	next, err := tx.NextPendingJob(ctx, job.AssetID, nextType)
	if errors.Is(err, store.ErrNotFound) {
		return nil // nothing waits, as after a job run again on its own
	}
	if err != nil {
		return err
	}
	if err := tx.QueueJob(ctx, next); err != nil {
		return err
	}
	// Said in the finished job's log, where the dashboard shows it.
	return tx.AddJobLog(ctx, job.ID, domain.LogInfo, fmt.Sprintf("queued next job next_type=%s next_job_id=%s", next.Type, next.ID))
}
