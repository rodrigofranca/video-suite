// Package testvideo generates small MP4 files for tests.
package testvideo

import (
	"os/exec"
	"strconv"
	"testing"
)

// MakeWithoutAudio writes an MP4 with a video track only.
func MakeWithoutAudio(t *testing.T, path string, seconds int, size string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=duration="+strconv.Itoa(seconds)+":size="+size+":rate=25",
		"-c:v", "libx264", "-preset", "ultrafast", path).CombinedOutput()
	if err != nil {
		t.Fatalf("make test video: %v: %s", err, out)
	}
}

// Make writes an MP4 (H.264 + AAC test pattern) of the given length and size
// to path. It skips the test when ffmpeg is not installed.
func Make(t *testing.T, path string, seconds int, size string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	d := strconv.Itoa(seconds)
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=duration="+d+":size="+size+":rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration="+d,
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest", path).CombinedOutput()
	if err != nil {
		t.Fatalf("make test video: %v: %s", err, out)
	}
}
