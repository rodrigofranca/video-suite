package jobcontracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessageRoundTrip(t *testing.T) {
	in := GenerateThumbnailInput{AssetID: "a1", StorageKey: "assets/a1/source/input.mp4", AtSecond: 3}
	msg, err := NewMessage("j1", JobTypeGenerateThumbnail, in)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseMessage(msg.Values())
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if got.JobID != "j1" || got.Type != JobTypeGenerateThumbnail {
		t.Fatalf("unexpected message: %+v", got)
	}

	out, err := Decode[GenerateThumbnailInput](got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out != in {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestParseMessageRejectsBadInput(t *testing.T) {
	cases := map[string]map[string]any{
		"missing job_id":  {FieldType: JobTypeTranscodeHLS, FieldPayload: "{}"},
		"missing type":    {FieldJobID: "j1", FieldPayload: "{}"},
		"missing payload": {FieldJobID: "j1", FieldType: JobTypeTranscodeHLS},
		"invalid JSON":    {FieldJobID: "j1", FieldType: JobTypeTranscodeHLS, FieldPayload: "{"},
	}
	for name, values := range cases {
		if _, err := ParseMessage(values); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestNextJobType(t *testing.T) {
	cases := map[string]string{
		JobTypeExtractMetadata:   JobTypeGenerateThumbnail,
		JobTypeGenerateThumbnail: JobTypeTranscodeHLS,
		JobTypeTranscodeHLS:      JobTypeExtractAudio,
		JobTypeExtractAudio:      JobTypeTranscribeAudio,
		JobTypeTranscribeAudio:   JobTypeAnalyzeTranscript,
		JobTypeAnalyzeTranscript: JobTypeSuggestClips,
		JobTypeSuggestClips:      JobTypeTrackFaces,
		JobTypeTrackFaces:        "",
		JobTypeExportClip:        "", // on request, not in the pipeline
		// An export in another language waits for the translation.
		JobTypeTranslateTranscript: JobTypeExportClip,
		"unknown":                  "",
	}
	for in, want := range cases {
		got, ok := NextJobType(in)
		if got != want || ok != (want != "") {
			t.Errorf("NextJobType(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

func TestTrackFacesInputLeavesOutAnEmptyClip(t *testing.T) {
	all, _ := json.Marshal(TrackFacesInput{AssetID: "a", StorageKey: "k"})
	one, _ := json.Marshal(TrackFacesInput{AssetID: "a", StorageKey: "k", ClipID: "c"})
	if strings.Contains(string(all), "clip_id") || !strings.Contains(string(one), `"clip_id":"c"`) {
		t.Fatalf("all clips: %s; one clip: %s", all, one)
	}
}

func TestExportClipInputLeavesOutUnchosenCaptionOptions(t *testing.T) {
	plain, _ := json.Marshal(ExportClipInput{AssetID: "a", Format: "vertical", Captions: true})
	for _, field := range []string{"caption_position", "caption_color", "caption_caps"} {
		if strings.Contains(string(plain), field) {
			t.Fatalf("%s in %s", field, plain)
		}
	}
	off := false
	chosen, _ := json.Marshal(ExportClipInput{CaptionPosition: "top", CaptionColor: "green", CaptionCaps: &off})
	if !strings.Contains(string(chosen), `"caption_position":"top","caption_color":"green","caption_caps":false`) {
		t.Fatalf("chosen options: %s", chosen)
	}
}
