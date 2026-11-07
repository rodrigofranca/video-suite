package ffmpeg

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"ultramedia/worker/internal/testvideo"
)

// A trimmed ffprobe output of a phone video: cover art stream, odd frame rate.
const sampleProbe = `{
  "streams": [
    {"codec_type": "video", "codec_name": "mjpeg", "width": 300, "height": 300, "avg_frame_rate": "0/0", "disposition": {"attached_pic": 1}},
    {"codec_type": "video", "codec_name": "h264", "width": 1920, "height": 1080, "avg_frame_rate": "30000/1001", "r_frame_rate": "30000/1001", "disposition": {"attached_pic": 0}},
    {"codec_type": "audio", "codec_name": "aac"}
  ],
  "format": {"filename": "x", "format_name": "mov,mp4,m4a,3gp,3g2,mj2", "duration": "12.512000", "size": "23456789", "bit_rate": "14998123"}
}`

func TestSummarize(t *testing.T) {
	s, err := Summarize([]byte(sampleProbe))
	if err != nil {
		t.Fatal(err)
	}
	want := Summary{DurationSeconds: 12.512, Width: 1920, Height: 1080, VideoCodec: "h264", AudioCodec: "aac",
		BitRate: 14998123, FrameRate: 29.97, Container: "mov,mp4,m4a,3gp,3g2,mj2", SizeBytes: 23456789}
	if s != want {
		t.Fatalf("got %+v\nwant %+v", s, want)
	}
}

func TestSummarizeWithoutVideo(t *testing.T) {
	_, err := Summarize([]byte(`{"streams":[{"codec_type":"audio","codec_name":"mp3"}],"format":{"duration":"3"}}`))
	if err == nil || !strings.Contains(err.Error(), "no video stream") {
		t.Fatalf("want no-video error, got %v", err)
	}
}

func TestProbeRealFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.mp4")
	testvideo.Make(t, path, 4, "640x360")

	raw, s, err := Probe(context.Background(), path, "assets/x/source/input.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if s.Width != 640 || s.Height != 360 || s.VideoCodec != "h264" || s.AudioCodec != "aac" || s.FrameRate != 25 {
		t.Fatalf("summary = %+v", s)
	}
	if s.DurationSeconds < 3.9 || s.DurationSeconds > 4.2 {
		t.Fatalf("duration = %v, want about 4", s.DurationSeconds)
	}
	if strings.Contains(string(raw), path) || !strings.Contains(string(raw), `"filename":"assets/x/source/input.mp4"`) {
		t.Fatal("the input path must be replaced by the given name in the stored output")
	}
}

func TestProbeErrorHidesInput(t *testing.T) {
	needFFmpeg(t)
	secret := "http://127.0.0.1:1/file.mp4?X-Amz-Signature=SECRET"
	_, _, err := Probe(context.Background(), secret, "k")
	if err == nil {
		t.Fatal("want an error for an unreachable URL")
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error leaks the signed URL: %v", err)
	}
}

func TestFrameRate(t *testing.T) {
	cases := map[string]float64{"30/1": 30, "30000/1001": 29.97, "24000/1001": 23.976, "0/0": 0, "": 0}
	for in, want := range cases {
		if got := frameRate(in); got != want {
			t.Errorf("frameRate(%q) = %v, want %v", in, got, want)
		}
	}
}
