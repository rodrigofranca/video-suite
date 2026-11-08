package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
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

func TestLadder(t *testing.T) {
	names := func(rs []Rendition) string {
		var out []string
		for _, r := range rs {
			out = append(out, fmt.Sprintf("%s %dx%d %dk", r.Name, r.Width, r.Height, r.MaxKbps))
		}
		return strings.Join(out, ", ")
	}
	for _, tc := range []struct {
		w, h int
		want string
	}{
		{1920, 1080, "1080p 1920x1080 5000k, 720p 1280x720 2800k, 480p 852x480 1400k, 360p 640x360 800k"},
		{3840, 2160, "1080p 1920x1080 5000k, 720p 1280x720 2800k, 480p 852x480 1400k, 360p 640x360 800k"}, // no 4K
		{1080, 1920, "1080p 1080x1920 5000k, 720p 720x1280 2800k, 480p 480x852 1400k, 360p 360x640 800k"}, // portrait
		{1366, 768, "768p 1366x768 3093k, 480p 854x480 1400k, 360p 640x360 800k"},                         // 720p is within 10%
		{640, 360, "360p 640x360 800k"},
		{321, 241, "240p 320x240 535k"}, // smaller than the ladder: its own size
	} {
		if got := names(Ladder(tc.w, tc.h)); got != tc.want {
			t.Errorf("Ladder(%d, %d) =\n  %s\nwant\n  %s", tc.w, tc.h, got, tc.want)
		}
	}
	if Ladder(0, 0) != nil {
		t.Error("Ladder(0, 0) should be empty")
	}
}

func TestHLSOutput(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mp4")
	testvideo.Make(t, src, 13, "1280x720")
	out := filepath.Join(dir, "hls")
	os.Mkdir(out, 0o755)

	var logs bytes.Buffer
	var progress []time.Duration
	record := func(d time.Duration) { progress = append(progress, d) }
	renditions := Ladder(1280, 720)
	if err := Run(context.Background(), HLSArgs(src, out, renditions, true), slog.New(slog.NewTextHandler(&logs, nil)), Options{Progress: record}); err != nil {
		t.Fatalf("Run: %v\n%s", err, logs.String())
	}

	master, err := os.ReadFile(filepath.Join(out, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"#EXTM3U", "RESOLUTION=1280x720", "720p/index.m3u8", "RESOLUTION=852x480", "480p/index.m3u8", "RESOLUTION=640x360", "360p/index.m3u8", "mp4a.40.2"} {
		if !strings.Contains(string(master), want) {
			t.Errorf("master playlist missing %q:\n%s", want, master)
		}
	}
	for _, r := range renditions {
		playlist, err := os.ReadFile(filepath.Join(out, r.Name, ManifestName))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"#EXT-X-PLAYLIST-TYPE:VOD", "#EXT-X-ENDLIST", "segment_000.ts", "segment_002.ts"} {
			if !strings.Contains(string(playlist), want) {
				t.Errorf("%s playlist missing %q", r.Name, want)
			}
		}
		if segments, _ := filepath.Glob(filepath.Join(out, r.Name, "segment_*.ts")); len(segments) != 3 { // 13s in 6s segments
			t.Errorf("%s: %d segments, want 3", r.Name, len(segments))
		}
	}
	// The renditions switch at the same points: their segments start at the same times.
	var durations []string
	for _, r := range renditions {
		playlist, _ := os.ReadFile(filepath.Join(out, r.Name, ManifestName))
		var d []string
		for _, line := range strings.Split(string(playlist), "\n") {
			if strings.HasPrefix(line, "#EXTINF:") {
				d = append(d, line)
			}
		}
		durations = append(durations, strings.Join(d, " "))
	}
	if durations[0] != durations[1] || durations[1] != durations[2] {
		t.Errorf("segment lengths differ between renditions: %q", durations)
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

// A silent video with an odd size: one quality, scaled to even, without audio.
func TestHLSWithoutAudio(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "silent.mp4")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "testsrc2=s=321x241:r=25:d=3",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", src).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	out := filepath.Join(dir, "hls")
	if err := Run(context.Background(), HLSArgs(src, out, Ladder(321, 241), false), slog.Default(), Options{}); err != nil {
		t.Fatal(err)
	}
	master, _ := os.ReadFile(filepath.Join(out, ManifestName))
	if !strings.Contains(string(master), "240p/index.m3u8") || !strings.Contains(string(master), "RESOLUTION=320x240") || strings.Contains(string(master), "mp4a") {
		t.Fatalf("master playlist without audio:\n%s", master)
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
	err := Run(context.Background(), HLSArgs(bad, dir, Ladder(640, 360), true), slog.New(slog.NewTextHandler(&logs, nil)), Options{})
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
