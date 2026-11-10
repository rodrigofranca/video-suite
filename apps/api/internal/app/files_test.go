package app

import (
	"strings"
	"testing"

	jobcontracts "ultramedia/job-contracts"
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

// The API builds an input for exactly the job types the table says can run
// again by hand (docs/adr/0036).
func TestInputsMatchTheJobTable(t *testing.T) {
	for _, kind := range jobcontracts.JobTypes {
		_, ok := inputs[kind.Name]
		if ok != kind.Rerun {
			t.Errorf("%s: input builder %v, rerun %v", kind.Name, ok, kind.Rerun)
		}
	}
	if len(inputs) != len(jobcontracts.Pipeline) {
		t.Errorf("%d input builders for %d pipeline steps", len(inputs), len(jobcontracts.Pipeline))
	}
}
