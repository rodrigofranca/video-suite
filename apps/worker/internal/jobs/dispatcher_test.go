package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	jobcontracts "ultramedia/job-contracts"
)

func TestDispatchDecodesTypedInput(t *testing.T) {
	d := NewDispatcher()
	var got jobcontracts.TranscodeHLSInput
	var gotJobID string
	Handle(d, jobcontracts.JobTypeTranscodeHLS, func(_ context.Context, job Job, in jobcontracts.TranscodeHLSInput) error {
		gotJobID, got = job.ID, in
		return nil
	})

	want := jobcontracts.TranscodeHLSInput{AssetID: "a1", StorageKey: "assets/a1/source/input.mp4"}
	msg, _ := jobcontracts.NewMessage("j1", jobcontracts.JobTypeTranscodeHLS, want)
	if err := d.Dispatch(context.Background(), Job{ID: msg.JobID, Type: msg.Type, Payload: msg.Payload}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if gotJobID != "j1" || got != want {
		t.Fatalf("handler got job %q input %+v", gotJobID, got)
	}
}

func TestDispatchUnknownType(t *testing.T) {
	err := NewDispatcher().Dispatch(context.Background(), Job{Type: "unknown-type", Payload: []byte("{}")})
	if !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "unknown-type") {
		t.Fatalf("want ErrUnsupported naming the type, got %v", err)
	}
}

func TestDispatchPropagatesErrors(t *testing.T) {
	d := NewDispatcher()
	boom := errors.New("boom")
	Handle(d, "fails", func(context.Context, Job, struct{}) error { return boom })

	if err := d.Dispatch(context.Background(), Job{Type: "fails", Payload: []byte("{}")}); !errors.Is(err, boom) {
		t.Fatalf("want handler error, got %v", err)
	}
	if err := d.Dispatch(context.Background(), Job{Type: "fails", Payload: []byte(`"not an object"`)}); err == nil {
		t.Fatal("want decode error for bad payload")
	}
}
