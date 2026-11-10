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
	"io"

	"ultramedia/media-core/domain"
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
	// TooLarge: a file is bigger than allowed.
	TooLarge
	// Unsupported: a file is not of a kind the use case accepts.
	Unsupported
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

// AssetWriter creates, completes and deletes videos. CompleteUpload queues
// the first job in the same transaction (the outbox, docs/adr/0034). The
// IfIdle methods return domain.ErrBusy while the asset is uploading or has a
// job pending or running, and domain.ErrNotFound for an unknown one.
type AssetWriter interface {
	CreateAsset(ctx context.Context, a domain.Asset) error
	CompleteUpload(ctx context.Context, src domain.Source, jobs []domain.Job) error
	DeleteAsset(ctx context.Context, id string) error
	DeleteAssetIfIdle(ctx context.Context, id string) error
}

// Metadata reads what the metadata job found in a video.
type Metadata interface {
	GetTechnicalMetadata(ctx context.Context, assetID string) (domain.TechnicalMetadata, error)
}

// Clips reads clips.
type Clips interface {
	GetClip(ctx context.Context, id string) (domain.Clip, error)
}

// ClipWriter changes and deletes clips, only while the asset is idle
// (domain.ErrBusy otherwise). EditClipIfIdle queues retrack, when given, in
// the same transaction.
type ClipWriter interface {
	EditClipIfIdle(ctx context.Context, c domain.Clip, retrack *domain.Job) error
	DeleteClipIfIdle(ctx context.Context, assetID, clipID string) error
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

// PresetWriter saves presets. CreatePreset and UpdatePreset return
// domain.ErrNameTaken when another preset has the name.
type PresetWriter interface {
	CreatePreset(ctx context.Context, p domain.Preset) error
	UpdatePreset(ctx context.Context, p domain.Preset) error
	DeletePreset(ctx context.Context, id string) error
}

// LibraryWriter records and removes the library's files. The deletes
// return a *domain.InUseError while a preset uses the file.
type LibraryWriter interface {
	CreateVignette(ctx context.Context, v domain.Vignette) error
	DeleteVignette(ctx context.Context, id string) error
	CreateWatermark(ctx context.Context, w domain.Watermark) error
	DeleteWatermark(ctx context.Context, id string) error
}

// Files keeps stored files (S3).
type Files interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) (int, error)
}

// Jobs records processing jobs. CreateJobIfIdle queues the job in the same
// transaction (the outbox, docs/adr/0034), and returns domain.ErrBusy while
// the asset has a job pending or running.
type Jobs interface {
	CreateJobIfIdle(ctx context.Context, job domain.Job, assetStatus string, then ...domain.Job) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
}

// isNotFound reports a port's "does not exist".
func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
