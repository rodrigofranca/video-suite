package ffmpeg

import (
	"bytes"
	"context"
	"image/jpeg"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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
	var progress []time.Duration
	record := func(d time.Duration) { progress = append(progress, d) }
	if err := Run(context.Background(), HLSArgs(src, out), slog.New(slog.NewTextHandler(&logs, nil)), Options{Progress: record}); err != nil {
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

	if len(progress) < 2 || progress[len(progress)-1] < 12*time.Second {
		t.Fatalf("progress = %v, want reports up to about 13s", progress)
	}
	for i := 1; i < len(progress); i++ {
		if progress[i] < progress[i-1] {
			t.Fatalf("progress went backwards: %v", progress)
		}
	}
}

func TestThumbnail(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mp4")
	testvideo.Make(t, src, 3, "1920x1080")

	out := filepath.Join(dir, "thumb.jpg")
	if err := Run(context.Background(), ThumbnailArgs(src, 1.5, out), slog.New(slog.DiscardHandler), Options{}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil || cfg.Width != 1280 || cfg.Height != 720 {
		t.Fatalf("thumbnail = %dx%d, %v; want 1280x720", cfg.Width, cfg.Height, err)
	}

	// Past the end FFmpeg succeeds but writes nothing: callers must check.
	late := filepath.Join(dir, "late.jpg")
	Run(context.Background(), ThumbnailArgs(src, 10, late), slog.New(slog.DiscardHandler), Options{})
	if _, err := os.Stat(late); err == nil {
		t.Fatal("expected no image for a time past the end")
	}
}

func TestCommandLineRedacts(t *testing.T) {
	got := CommandLine([]string{"-i", "http://s3/x?sig=abc", "out.jpg"}, "http://s3/x?sig=abc")
	if got != "ffmpeg -i <source> out.jpg" {
		t.Fatalf("CommandLine = %q", got)
	}
}

func TestRunReportsFFmpegError(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.mp4")
	os.WriteFile(bad, []byte("\x00\x00\x00\x20ftypisom but not really a video"), 0o644)

	var logs bytes.Buffer
	err := Run(context.Background(), HLSArgs(bad, dir), slog.New(slog.NewTextHandler(&logs, nil)), Options{})
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
	err := Run(ctx, []string{"-f", "lavfi", "-i", "testsrc2", "-f", "null", "-"}, slog.New(slog.DiscardHandler), Options{})
	if err == nil {
		t.Fatal("want an error when the context is canceled")
	}
}

func TestTrackExpr(t *testing.T) {
	// 1280-wide source, 404-wide crop: the left edge is x*1280 - 202, kept in 0..876.
	got := trackExpr([]TrackPoint{{0, 0.1}, {2, 0.5}, {4, 0.95}}, 1280, 404)
	want := "lt(t,0.000)*0.000+" +
		"gte(t,0.000)*lt(t,2.000)*(0.000+(t-0.000)*219.000)+" +
		"gte(t,2.000)*lt(t,4.000)*(438.000+(t-2.000)*219.000)+" +
		"gte(t,4.000)*876.000"
	if got != want {
		t.Fatalf("trackExpr =\n%s\nwant\n%s", got, want)
	}
	if one := trackExpr([]TrackPoint{{1, 0.5}}, 1280, 404); one != "438.000" {
		t.Fatalf("one point = %s", one)
	}
	args, _ := ClipArgs(ClipSpec{Input: "in.mp4", Duration: 4, Format: "vertical-face", SourceWidth: 1280, SourceHeight: 720,
		Output: "out.mp4", Track: []TrackPoint{{0, 0.5}, {4, 0.6}}})
	if graph := args[slices.Index(args, "-filter_complex")+1]; !strings.HasPrefix(graph, "[0:v:0]crop=w=404:h=720:x='lt(t,0.000)*438.000+") {
		t.Fatalf("graph = %s", graph)
	}
}
