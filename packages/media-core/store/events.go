package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

// ---- Outbox (docs/adr/0034) ----
//
// A change that leads to more work records a domain event in outbox_events,
// in its own transaction, and a relay handles the event afterwards. The work
// is never lost between the database and the queue: either both the change
// and its event are saved, or neither is.

// recordEvent adds an event in the caller's transaction. A trigger wakes the
// relays once the transaction commits.
func recordEvent(ctx context.Context, tx pgx.Tx, eventType, assetID, jobID string) error {
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events (type, asset_id, job_id) VALUES ($1, $2, $3)`,
		eventType, orNull(assetID), orNull(jobID))
	return err
}

// queueJob records that a pending job may run: the relay sends it to its stream.
func queueJob(ctx context.Context, tx pgx.Tx, job domain.Job) error {
	return recordEvent(ctx, tx, jobcontracts.EventJobQueued, job.AssetID, job.ID)
}

const eventColumns = `id, type, COALESCE(asset_id::text, ''), COALESCE(job_id::text, ''), created_at, handled_at, attempts, last_error`

func scanEvent(row pgx.Row) (domain.Event, error) {
	var e domain.Event
	err := row.Scan(&e.ID, &e.Type, &e.AssetID, &e.JobID, &e.CreatedAt, &e.HandledAt, &e.Attempts, &e.LastError)
	return e, notFound(err)
}

// Events lists an asset's events, oldest first, handled or not.
func (s *Store) Events(ctx context.Context, assetID string) ([]domain.Event, error) {
	rows, err := s.db.Query(ctx, `SELECT `+eventColumns+` FROM outbox_events WHERE asset_id = $1 ORDER BY id`, assetID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Event, error) { return scanEvent(r) })
}

// EventTx is what an event handler may read and change. It runs in the
// transaction that marks the event handled, so a handler that fails leaves
// nothing behind, and the event is tried again.
type EventTx struct{ tx pgx.Tx }

func (t EventTx) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return scanJob(t.tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM processing_jobs WHERE id = $1`, id))
}

// NextPendingJob returns the asset's oldest pending job of a type.
func (t EventTx) NextPendingJob(ctx context.Context, assetID, jobType string) (domain.Job, error) {
	return scanJob(t.tx.QueryRow(ctx, nextPendingJob, assetID, jobType, domain.JobPending))
}

func (t EventTx) SetAssetStatus(ctx context.Context, assetID, status string) error {
	_, err := t.tx.Exec(ctx, `UPDATE media_assets SET status = $2, updated_at = NOW() WHERE id = $1`, assetID, status)
	return err
}

// AddJobLog adds a line to a job's log.
func (t EventTx) AddJobLog(ctx context.Context, jobID, level, message string) error {
	_, err := t.tx.Exec(ctx,
		`INSERT INTO job_logs (id, job_id, level, message, timestamp) VALUES ($1, $2, $3, $4, $5)`,
		domain.NewID(), jobID, level, message, time.Now())
	return err
}

// QueueJob records that a pending job may run.
func (t EventTx) QueueJob(ctx context.Context, job domain.Job) error {
	return queueJob(ctx, t.tx, job)
}

// ErrEventFailed marks an event whose handler failed: its changes were
// undone and it waits for another try.
var ErrEventFailed = errors.New("event handler failed")

// maxRetryDelay caps the wait between two tries of an event.
const maxRetryDelay = 5 * time.Minute

// retryDelay is the wait after a failed try: 1 s, 2 s, 4 s and so on, up to
// maxRetryDelay.
func retryDelay(attempts int) time.Duration {
	if attempts > 9 {
		return maxRetryDelay
	}
	return min(time.Second<<max(attempts-1, 0), maxRetryDelay)
}

// HandleNextEvent claims the oldest event that is due and that no other
// relay holds, runs handle on it and marks it handled, in one transaction.
// When handle fails, its changes are undone, the event waits retryDelay and
// the error wraps ErrEventFailed. It reports whether there was an event.
// With assets, only those videos' events are claimed.
func (s *Store) HandleNextEvent(ctx context.Context, assets []string, handle func(context.Context, EventTx, domain.Event) error) (domain.Event, bool, error) {
	var (
		e      domain.Event
		found  bool
		failed error
	)
	if len(assets) == 0 {
		assets = nil // NULL: every video
	}
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		e, err = scanEvent(tx.QueryRow(ctx, `SELECT `+eventColumns+` FROM outbox_events
			WHERE handled_at IS NULL AND next_attempt_at <= NOW() AND ($1::uuid[] IS NULL OR asset_id = ANY($1::uuid[]))
			ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, assets))
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		// A savepoint: the handler's changes go if it fails, the event's row stays.
		failed = pgx.BeginFunc(ctx, tx, func(sp pgx.Tx) error { return handle(ctx, EventTx{sp}, e) })
		if failed == nil {
			_, err = tx.Exec(ctx, `UPDATE outbox_events SET handled_at = NOW(), attempts = attempts + 1, last_error = NULL WHERE id = $1`, e.ID)
			return err
		}
		wait := retryDelay(e.Attempts + 1)
		_, err = tx.Exec(ctx,
			`UPDATE outbox_events SET attempts = attempts + 1, last_error = $2, next_attempt_at = NOW() + make_interval(secs => $3) WHERE id = $1`,
			e.ID, failed.Error(), wait.Seconds())
		return err
	})
	if err != nil {
		return e, found, err
	}
	if failed != nil {
		e.Attempts++
		return e, true, fmt.Errorf("%w: %s event %d: %w", ErrEventFailed, e.Type, e.ID, failed)
	}
	return e, found, nil
}

// PruneEvents deletes the events handled longer than keep ago.
func (s *Store) PruneEvents(ctx context.Context, keep time.Duration) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM outbox_events WHERE handled_at < NOW() - make_interval(secs => $1)`, keep.Seconds())
	return tag.RowsAffected(), err
}

// Listener tells a relay when events are recorded, through the PostgreSQL
// channel the outbox trigger notifies. It holds its own connection.
type Listener struct{ conn *pgx.Conn }

// Listen starts listening on jobcontracts.OutboxChannel.
func (s *Store) Listen(ctx context.Context) (*Listener, error) {
	c, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	conn := c.Hijack() // out of the pool: it keeps the LISTEN
	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{jobcontracts.OutboxChannel}.Sanitize()); err != nil {
		conn.Close(ctx)
		return nil, err
	}
	return &Listener{conn: conn}, nil
}

// Wait returns when events were recorded or after timeout, whichever comes
// first. An error means the connection is gone and the listener must be
// replaced.
func (l *Listener) Wait(ctx context.Context, timeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := l.conn.WaitForNotification(waitCtx)
	if err != nil && ctx.Err() == nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded) && !l.conn.IsClosed() {
		return nil // timed out: time to look anyway
	}
	return err
}

func (l *Listener) Close() { l.conn.Close(context.Background()) }
