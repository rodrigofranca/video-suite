package domain

import "errors"

// Errors the layers share (docs/adr/0033): adapters such as the store return
// them, and use cases and the HTTP layer check them with errors.Is, so no
// layer depends on another's package for its errors.
var (
	// ErrNotFound: the row or file does not exist.
	ErrNotFound = errors.New("not found")
	// ErrBusy: the asset still has a job pending or running.
	ErrBusy = errors.New("the asset still has jobs pending or running")
)
