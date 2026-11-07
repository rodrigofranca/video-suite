package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	jobcontracts "ultramedia/job-contracts"
)

func TestDispatchDecodesTypedInput(t *testing.T) {
	d := NewDispatcher()
	var got jobcontracts.TranscodeHLSInput
	var gotJobID string
	Handle(d, jobcontracts.JobTypeTranscodeHLS, func(_ context.Context, jobID string, in jobcontracts.TranscodeHLSInput) error {
		gotJobID, got = jobID, in
		return nil
	})

	want := jobcontracts.TranscodeHLSInput{AssetID: "a1", StorageKey: "assets/a1/source/input.mp4"}
	msg, _ := jobcontracts.NewMessage("j1", jobcontracts.JobTypeTranscodeHLS, want)
	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if gotJobID != "j1" || got != want {
		t.Fatalf("handler got job %q input %+v", gotJobID, got)
	}
}

func TestDispatchUnknownType(t *testing.T) {
	err := NewDispatcher().Dispatch(context.Background(), jobcontracts.Message{Type: "unknown-type", Payload: []byte("{}")})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
	if !strings.Contains(err.Error(), "unknown-type") {
		t.Fatalf("error should name the type: %v", err)
	}
}

func TestDispatchPropagatesErrors(t *testing.T) {
	d := NewDispatcher()
	boom := errors.New("boom")
	Handle(d, "fails", func(context.Context, string, struct{}) error { return boom })

	if err := d.Dispatch(context.Background(), jobcontracts.Message{Type: "fails", Payload: []byte("{}")}); !errors.Is(err, boom) {
		t.Fatalf("want handler error, got %v", err)
	}
	if err := d.Dispatch(context.Background(), jobcontracts.Message{Type: "fails", Payload: []byte(`"not an object"`)}); err == nil {
		t.Fatal("want decode error for bad payload")
	}
}

func TestStubsLogAndSucceed(t *testing.T) {
	var buf bytes.Buffer
	d := NewDispatcher()
	RegisterStubs(d, slog.New(slog.NewTextHandler(&buf, nil)))

	for _, jobType := range []string{
		jobcontracts.JobTypeExtractMetadata,
		jobcontracts.JobTypeGenerateThumbnail,
		jobcontracts.JobTypeTranscodeHLS,
	} {
		msg := jobcontracts.Message{JobID: "j1", Type: jobType, Payload: []byte(`{"asset_id":"a1","storage_key":"k"}`)}
		if err := d.Dispatch(context.Background(), msg); err != nil {
			t.Fatalf("%s: %v", jobType, err)
		}
		if want := "stub: " + jobType + " called for asset a1"; !strings.Contains(buf.String(), want) {
			t.Fatalf("log missing %q:\n%s", want, buf.String())
		}
	}
}
