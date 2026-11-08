package ffmpeg

import (
	"context"
	"log/slog"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"ultramedia/worker/internal/testvideo"
)

// colorVideo writes a plain-colored MP4 of a size, frame rate and length,
// with a tone or silent.
func colorVideo(t *testing.T, path, color, size string, rate int, seconds float64, audio bool) {
	t.Helper()
	d := strconv.FormatFloat(seconds, 'f', 2, 64)
	args := []string{"-v", "error", "-y", "-f", "lavfi", "-i", "color=c=" + color + ":s=" + size + ":r=" + strconv.Itoa(rate) + ":d=" + d}
	if audio {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency=880:duration="+d, "-c:a", "aac", "-shortest")
	}
	args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", path)
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

// centre returns the color of a small square in the middle of the frame at second at.
func centre(t *testing.T, path string, at float64) (r, g, b int) {
	t.Helper()
	out, err := exec.Command("ffmpeg", "-v", "error", "-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", path, "-frames:v", "1",
		"-vf", "crop=iw/4:ih/8,scale=1:1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-").Output()
	if err != nil || len(out) < 3 {
		t.Fatalf("frame at %.2f: %v", at, err)
	}
	return int(out[0]), int(out[1]), int(out[2])
}

// An opening and a closing in other sizes and frame rates, one silent, are
// fitted to a vertical clip and joined in order, with sound throughout.
func TestClipWithVignettes(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src, intro, outro := filepath.Join(dir, "src.mp4"), filepath.Join(dir, "intro.mp4"), filepath.Join(dir, "outro.mp4")
	testvideo.Make(t, src, 5, "640x360")
	colorVideo(t, intro, "red", "1280x720", 30, 1.5, true) // horizontal, fitted over its blur
	colorVideo(t, outro, "blue", "720x1280", 24, 1, false) // vertical, silent
	out := filepath.Join(dir, "clip.mp4")

	args, err := ClipArgs(ClipSpec{
		Input: src, Start: 1, Duration: 3, Format: "vertical", SourceWidth: 640, SourceHeight: 360, Output: out,
		Intro: &VignetteInput{Path: intro, Duration: 1.5, Audio: true}, Outro: &VignetteInput{Path: outro, Duration: 1},
		FrameRate: 25, SourceAudio: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), args, slog.Default(), Options{}); err != nil {
		t.Fatalf("%v\n%s", err, CommandLine(args))
	}
	_, info, err := Probe(context.Background(), out, "clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 1080 || info.Height != 1920 || info.AudioCodec != "aac" || math.Abs(info.DurationSeconds-5.5) > 0.15 {
		t.Fatalf("clip = %dx%d, audio %q, %.2f s; want 1080x1920 with AAC, 5.5 s", info.Width, info.Height, info.AudioCodec, info.DurationSeconds)
	}
	if r, g, b := centre(t, out, 0.7); r < 200 || g > 60 || b > 60 {
		t.Errorf("at 0.7 s the opening should show (red), got %d,%d,%d", r, g, b)
	}
	if r, g, b := centre(t, out, 2.5); r > 200 && g < 60 && b < 60 || b > 200 && r < 60 && g < 60 {
		t.Errorf("at 2.5 s the clip should show, got the plain color %d,%d,%d", r, g, b)
	}
	if r, g, b := centre(t, out, 5.2); b < 200 || r > 60 || g > 60 {
		t.Errorf("at 5.2 s the closing should show (blue), got %d,%d,%d", r, g, b)
	}
}

// A silent source with a silent opening still gets a sound track, and the
// original shape keeps the source's size.
func TestClipWithVignetteAndNoSound(t *testing.T) {
	needFFmpeg(t)
	dir := t.TempDir()
	src, intro := filepath.Join(dir, "src.mp4"), filepath.Join(dir, "intro.mp4")
	testvideo.MakeWithoutAudio(t, src, 3, "640x360")
	colorVideo(t, intro, "green", "1080x1920", 30, 1, false)
	out := filepath.Join(dir, "clip.mp4")
	args, err := ClipArgs(ClipSpec{
		Input: src, Start: 0, Duration: 2, Format: "original", SourceWidth: 640, SourceHeight: 360, Output: out,
		Intro: &VignetteInput{Path: intro, Duration: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), args, slog.Default(), Options{}); err != nil {
		t.Fatalf("%v\n%s", err, CommandLine(args))
	}
	_, info, _ := Probe(context.Background(), out, "clip.mp4")
	if info.Width != 640 || info.Height != 360 || info.AudioCodec != "aac" || math.Abs(info.DurationSeconds-3) > 0.15 {
		t.Fatalf("clip = %dx%d, audio %q, %.2f s; want 640x360 with AAC, 3 s", info.Width, info.Height, info.AudioCodec, info.DurationSeconds)
	}
}
