package domain

import "time"

// Event is a domain event in the outbox (docs/adr/0034): something that
// happened to a video or a job, recorded in the same transaction as the
// change that caused it and handled afterwards by a relay. Type is one of
// the jobcontracts.Event* names.
type Event struct {
	ID        int64
	Type      string
	AssetID   string
	JobID     string
	CreatedAt time.Time
	// HandledAt is when a relay handled the event, or nil while it waits.
	HandledAt *time.Time
	// Attempts counts the tries so far; LastError says why the last failed.
	Attempts  int
	LastError *string
}
