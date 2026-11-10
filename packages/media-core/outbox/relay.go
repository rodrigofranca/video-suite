// Package outbox handles the domain events recorded in the outbox table
// (docs/adr/0034). The API and the media worker each run a Relay; they share
// the work safely, since each event is claimed by one relay at a time.
package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
)

// Handler reacts to an event, in the transaction that marks it handled: if
// it fails, what it changed is undone and the event is tried again later.
type Handler func(ctx context.Context, tx store.EventTx, e domain.Event) error

// Defaults for a Relay.
const (
	DefaultPoll    = 5 * time.Second
	DefaultKeepFor = 7 * 24 * time.Hour
	pruneEvery     = time.Hour
)

// Relay hands each event to the handlers for its type, oldest first. It
// wakes as soon as events are recorded, and looks every Poll anyway, for
// retries and in case a wake-up was missed.
type Relay struct {
	Store    *store.Store
	Handlers map[string][]Handler
	Log      *slog.Logger
	// Poll is the longest wait between two looks (DefaultPoll).
	Poll time.Duration
	// KeepFor is how long handled events stay, as a history (DefaultKeepFor).
	KeepFor time.Duration
	// Assets limits the relay to these videos' events; tests use it to
	// leave other tests' events alone.
	Assets []string
}

// Run handles events until ctx ends.
func (r *Relay) Run(ctx context.Context) {
	var lastPrune time.Time
	for ctx.Err() == nil {
		listener, err := r.Store.Listen(ctx)
		if err != nil {
			if ctx.Err() == nil {
				r.log().Error("outbox relay cannot listen; it will look every poll", "error", err)
			}
		}
		for ctx.Err() == nil {
			r.Drain(ctx)
			if time.Since(lastPrune) > pruneEvery {
				r.prune(ctx)
				lastPrune = time.Now()
			}
			if listener == nil {
				sleep(ctx, r.poll())
				break // try to listen again
			}
			if err := listener.Wait(ctx, r.poll()); err != nil {
				if ctx.Err() == nil {
					r.log().Warn("outbox relay lost its connection; listening again", "error", err)
				}
				break
			}
		}
		if listener != nil {
			listener.Close()
		}
	}
}

// Drain handles the events that are due until none is left, and returns how
// many it handled. Failed events wait for their next try.
func (r *Relay) Drain(ctx context.Context) int {
	handled := 0
	for ctx.Err() == nil {
		e, found, err := r.Store.HandleNextEvent(ctx, r.Assets, r.handle)
		switch {
		case errors.Is(err, store.ErrEventFailed):
			r.log().Warn("event failed; it will be tried again", "event_id", e.ID, "type", e.Type,
				"asset_id", e.AssetID, "job_id", e.JobID, "attempts", e.Attempts, "error", err)
		case err != nil:
			if ctx.Err() == nil {
				r.log().Error("outbox relay could not read events", "error", err)
			}
			return handled
		case !found:
			return handled
		default:
			handled++
		}
	}
	return handled
}

func (r *Relay) handle(ctx context.Context, tx store.EventTx, e domain.Event) error {
	handlers, ok := r.Handlers[e.Type]
	if !ok {
		// Trying again would not help: mark it handled, and say so.
		r.log().Warn("no handler for event", "event_id", e.ID, "type", e.Type)
		return nil
	}
	for _, h := range handlers {
		if err := h(ctx, tx, e); err != nil {
			return err
		}
	}
	return nil
}

func (r *Relay) prune(ctx context.Context) {
	keep := r.KeepFor
	if keep <= 0 {
		keep = DefaultKeepFor
	}
	if n, err := r.Store.PruneEvents(ctx, keep); err != nil {
		r.log().Warn("could not prune handled events", "error", err)
	} else if n > 0 {
		r.log().Info("pruned handled events", "count", n)
	}
}

func (r *Relay) poll() time.Duration {
	if r.Poll > 0 {
		return r.Poll
	}
	return DefaultPoll
}

func (r *Relay) log() *slog.Logger {
	if r.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return r.Log
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
