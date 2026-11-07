package jobcontracts

import "testing"

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
		JobTypeTranscribeAudio:   "",
		"unknown":                "",
	}
	for in, want := range cases {
		got, ok := NextJobType(in)
		if got != want || ok != (want != "") {
			t.Errorf("NextJobType(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}
