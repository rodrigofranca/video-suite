// Package jobs routes queue messages to the handler registered for their type.
package jobs

import (
	"context"
	"errors"
	"fmt"

	jobcontracts "ultramedia/job-contracts"
)

// ErrUnsupported is returned when no handler is registered for a job type.
var ErrUnsupported = errors.New("unsupported job type")

// HandlerFunc runs one job.
type HandlerFunc func(ctx context.Context, msg jobcontracts.Message) error

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
func Handle[T any](d *Dispatcher, jobType string, h func(ctx context.Context, jobID string, input T) error) {
	d.Register(jobType, func(ctx context.Context, msg jobcontracts.Message) error {
		input, err := jobcontracts.Decode[T](msg)
		if err != nil {
			return err
		}
		return h(ctx, msg.JobID, input)
	})
}

// Dispatch runs the handler for msg.Type. It returns an error wrapping
// ErrUnsupported when the type has no handler.
func (d *Dispatcher) Dispatch(ctx context.Context, msg jobcontracts.Message) error {
	h, ok := d.handlers[msg.Type]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnsupported, msg.Type)
	}
	return h(ctx, msg)
}
