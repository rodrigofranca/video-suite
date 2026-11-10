package domain

import (
	"errors"
	"strings"
)

// Errors the layers share (docs/adr/0033): adapters such as the store return
// them, and use cases and the HTTP layer check them with errors.Is, so no
// layer depends on another's package for its errors.
var (
	// ErrNotFound: the row or file does not exist.
	ErrNotFound = errors.New("not found")
	// ErrBusy: the asset still has a job pending or running.
	ErrBusy = errors.New("the asset still has jobs pending or running")
)

// ErrNameTaken: another preset has the name (in any case).
var ErrNameTaken = errors.New("another preset has this name")

// InUseError: a vignette or watermark image cannot be deleted because
// presets use it.
type InUseError struct {
	What    string // "vignette" or "watermark"
	Presets []string
}

func (e *InUseError) Error() string {
	return "the " + e.What + " is used by the preset(s) " + strings.Join(e.Presets, ", ") + "; take it out of them first"
}
