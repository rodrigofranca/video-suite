package app

import (
	"strings"
	"testing"
)

func TestIsMP4(t *testing.T) {
	if !isMP4([]byte("\x00\x00\x00\x20ftypisom")) {
		t.Error("MP4 header not recognised")
	}
	if isMP4([]byte("GIF89a......")) || isMP4([]byte("ftyp")) {
		t.Error("non-MP4 accepted")
	}
}

func TestCleanFilename(t *testing.T) {
	cases := map[string]string{
		"clip.mp4":                        "clip.mp4",
		"../../etc/passwd.mp4":            "passwd.mp4",
		`C:\Users\me\video.mp4`:           "video.mp4",
		"":                                "upload.mp4",
		strings.Repeat("a", 300) + ".mp4": strings.Repeat("a", 251) + ".mp4",
	}
	for in, want := range cases {
		if got := CleanFilename(in); got != want {
			t.Errorf("CleanFilename(%.20q) = %.20q, want %.20q", in, got, want)
		}
	}
}
