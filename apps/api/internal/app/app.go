// Package app holds the API's use cases (docs/adr/0033): what the platform
// does when someone asks for it, apart from how they asked. A use case
// checks the request, applies the rules and drives the ports below; the
// HTTP handlers only translate between HTTP and it, and the store, the
// storage and the queue are adapters behind the ports.
//
// Reads that only list or show data stay in the HTTP layer and read the
// store directly; commands, which change state, come here.
package app

import (
	"context"
	"errors"
	"fmt"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

// ---- Errors ----

// Kind says what went wrong from the caller's side, so each adapter can
// answer in its own terms (an HTTP status, an exit code).
type Kind int

const (
	// Invalid: the request is wrong; asking again the same way fails again.
	Invalid Kind = iota + 1
	// NotFound: something the request names does not exist.
	NotFound
	// Conflict: the request is fine but the current state does not allow
	// it (a job is running, there is nothing to translate).
	Conflict
	// Unavailable: a service the use case needs failed; trying later may work.
	Unavailable
)

// Error is a use case's answer to a request it cannot carry out. Msg is
// meant for the person who asked; Err, when set, is the cause, for logs.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

func fail(kind Kind, msg string) error { return &Error{Kind: kind, Msg: msg} }

// ---- Ports ----
//
// Each port is as small as the use cases that need it; *store.Store
// satisfies the store ports, so wiring them is passing the same value.

// Assets reads videos.
type Assets interface {
	GetAsset(ctx context.Context, id string) (domain.Asset, error)
	GetSource(ctx context.Context, assetID string) (domain.Source, error)
}

// Clips reads clips.
type Clips interface {
	GetClip(ctx context.Context, id string) (domain.Clip, error)
}

// Speech reads what was said: the transcript and its translations.
type Speech interface {
	GetTranscript(ctx context.Context, assetID string) (domain.Transcript, error)
	GetTranslation(ctx context.Context, assetID, language string) (domain.Translation, error)
}

// Brand reads presets and the files they use.
type Brand interface {
	GetPreset(ctx context.Context, id string) (domain.Preset, error)
	GetVignette(ctx context.Context, id string) (domain.Vignette, error)
	GetWatermark(ctx context.Context, id string) (domain.Watermark, error)
}

// Jobs records processing jobs. CreateJobIfIdle returns domain.ErrBusy
// while the asset has a job pending or running.
type Jobs interface {
	CreateJobIfIdle(ctx context.Context, job domain.Job, assetStatus string, then ...domain.Job) error
	FinishJob(ctx context.Context, id string, jobErr error) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
}

// Queue hands a job to the workers.
type Queue interface {
	Enqueue(ctx context.Context, msg jobcontracts.Message) error
}

// QueueFunc lets a function be the Queue.
type QueueFunc func(ctx context.Context, msg jobcontracts.Message) error

func (f QueueFunc) Enqueue(ctx context.Context, msg jobcontracts.Message) error { return f(ctx, msg) }

// isNotFound reports a port's "does not exist".
func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
