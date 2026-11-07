package pipeline

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
)

// jobLogHandler is a slog.Handler that saves each record as a job_logs row
// and passes it on to the regular worker output.
type jobLogHandler struct {
	next     slog.Handler
	store    *store.Store
	jobID    string
	attrs    []slog.Attr // added with Logger.With, included in saved messages
	fallback *slog.Logger
}

func (h *jobLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (h *jobLogHandler) Handle(ctx context.Context, rec slog.Record) error {
	var b strings.Builder
	b.WriteString(rec.Message)
	write := func(a slog.Attr) bool {
		v := a.Value.Resolve().String()
		if strings.ContainsAny(v, " \t\"") {
			v = strconv.Quote(v)
		}
		b.WriteString(" " + a.Key + "=" + v)
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	rec.Attrs(write)

	level := domain.LogInfo
	switch {
	case rec.Level >= slog.LevelError:
		level = domain.LogError
	case rec.Level >= slog.LevelWarn:
		level = domain.LogWarning
	}
	// Save even while the worker shuts down, so the last lines are kept.
	if err := h.store.AddJobLog(context.WithoutCancel(ctx), h.jobID, level, b.String()); err != nil {
		h.fallback.Warn("could not save job log", "job_id", h.jobID, "error", err)
	}
	return h.next.Handle(ctx, rec)
}

func (h *jobLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	c.next = h.next.WithAttrs(attrs)
	return &c
}

func (h *jobLogHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.next = h.next.WithGroup(name)
	return &c
}
