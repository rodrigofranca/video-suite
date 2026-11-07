package ffmpeg

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ultramedia/worker/internal/testvideo"
)

func needFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
}

func TestHLSOutput(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mp4")
	testvideo.Make(t, src, 13, "321x241") // odd size: must be scaled to even
	out := filepath.Join(dir, "hls")
	os.Mkdir(out, 0o755)

	var logs bytes.Buffer
	if err := Run(context.Background(), HLSArgs(src, out), slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatalf("Run: %v\n%s", err, logs.String())
	}

	playlist, err := os.ReadFile(filepath.Join(out, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	p := string(playlist)
	for _, want := range []string{"#EXTM3U", "#EXT-X-PLAYLIST-TYPE:VOD", "#EXT-X-ENDLIST", "segment_000.ts", "segment_002.ts"} {
		if !strings.Contains(p, want) {
			t.Errorf("playlist missing %q:\n%s", want, p)
		}
	}
	segments, _ := filepath.Glob(filepath.Join(out, "segment_*.ts"))
	if len(segments) != 3 { // 13s in 6s segments
		t.Errorf("got %d segments, want 3", len(segments))
	}
}

func TestRunReportsFFmpegError(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.mp4")
	os.WriteFile(bad, []byte("\x00\x00\x00\x20ftypisom but not really a video"), 0o644)

	var logs bytes.Buffer
	err := Run(context.Background(), HLSArgs(bad, dir), slog.New(slog.NewTextHandler(&logs, nil)))
	if err == nil {
		t.Fatal("want an error for a broken file")
	}
	if !strings.Contains(err.Error(), "ffmpeg failed") || !strings.Contains(logs.String(), "ffmpeg: ") {
		t.Fatalf("error %q / logs %q should include FFmpeg's message", err, logs.String())
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	needFFmpeg(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(ctx, []string{"-f", "lavfi", "-i", "testsrc2", "-f", "null", "-"}, slog.New(slog.DiscardHandler))
	if err == nil {
		t.Fatal("want an error when the context is canceled")
	}
}
