// Package jobs routes jobs to the handler registered for their type.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

// ErrUnsupported is returned when no handler is registered for a job type.
var ErrUnsupported = errors.New("unsupported job type")

// Job is what a handler receives.
type Job struct {
	ID      string
	AssetID string
	Type    string
	Payload json.RawMessage // JSON of the typed input from job-contracts
	// Log writes to the worker output and to the job's logs in the database.
	Log *slog.Logger
}

// HandlerFunc runs one job.
type HandlerFunc func(ctx context.Context, job Job) error

// Dispatcher maps job types to handlers.
type Dispatcher struct {
	handlers map[string]HandlerFunc
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: map[string]HandlerFunc{}}
}

// Register sets the handler for a job type, replacing any previous one.
func (d *Dispatcher) Register(jobType string, h HandlerFunc) {
	d.handlers[jobType] = h
}

// Handle registers a handler that receives the payload already decoded into
// the typed input struct from job-contracts.
func Handle[T any](d *Dispatcher, jobType string, h func(ctx context.Context, job Job, input T) error) {
	d.Register(jobType, func(ctx context.Context, job Job) error {
		var input T
		if err := json.Unmarshal(job.Payload, &input); err != nil {
			return fmt.Errorf("decode %s payload: %w", jobType, err)
		}
		return h(ctx, job, input)
	})
}

// Dispatch runs the handler for job.Type. It returns an error wrapping
// ErrUnsupported when the type has no handler.
func (d *Dispatcher) Dispatch(ctx context.Context, job Job) error {
	h, ok := d.handlers[job.Type]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnsupported, job.Type)
	}
	if job.Log == nil {
		job.Log = slog.New(slog.DiscardHandler)
	}
	return h(ctx, job)
}
