package app

import (
	"context"
	"errors"
	"hash"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// What uploads have in common: names, the MP4 check and a reader that
// counts, hashes and stops at a limit.

// maxName is the longest preset or library file name, in characters.
const maxName = 60

// cleanName trims a name and checks its length.
func cleanName(name string) (string, bool) {
	name = strings.Join(strings.Fields(name), " ")
	return name, name != "" && utf8.RuneCountInString(name) <= maxName
}

// CleanFilename keeps only the base name, for display. It is never used to
// build a storage key.
func CleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "." || name == "/" {
		name = "upload.mp4"
	}
	if len(name) > 255 {
		name = name[len(name)-255:]
	}
	return name
}

// isMP4 checks for the "ftyp" box that starts every MP4 (ISO BMFF) file.
func isMP4(head []byte) bool {
	return len(head) >= 8 && string(head[4:8]) == "ftyp"
}

var errTooLarge = errors.New("file too large")

// countingReader hashes (when hash is set) and counts what passes through
// it, stops after limit bytes and remembers the first read error, to tell
// "too large" apart from storage errors.
type countingReader struct {
	r     io.Reader
	hash  hash.Hash
	n     int64
	limit int64
	err   error
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.hash != nil {
		c.hash.Write(p[:n])
	}
	if c.n > c.limit {
		c.err = errTooLarge
		return n, errTooLarge
	}
	if err != nil && err != io.EOF && c.err == nil {
		c.err = err
	}
	return n, err
}

// removeFile deletes a stored file even if the caller went away; a failure
// only leaves an orphan file, so it is logged.
func removeFile(ctx context.Context, files Files, log *slog.Logger, key string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := files.Delete(ctx, key); err != nil {
		log.Warn("could not delete a stored file", "key", key, "error", err)
	}
}

func orDiscard(log *slog.Logger) *slog.Logger {
	if log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return log
}
