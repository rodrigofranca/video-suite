package jobcontracts

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"slices"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/contracts.json")

// contractsDoc is the part of the contract other languages must agree on.
// The Python AI worker reads testdata/contracts.json in its own tests, so a
// change here that is not mirrored there fails one of the two test suites.
func contractsDoc() map[string]any {
	routes := map[string]string{}
	for _, t := range slices.Concat(Pipeline, []string{JobTypeExportClip, JobTypeTranslateTranscript}) {
		routes[t] = StreamFor(t)
	}
	example := func(v any) json.RawMessage {
		b, _ := json.Marshal(v)
		return b
	}
	return map[string]any{
		"streams": map[string]string{"media": StreamJobs, "ai": StreamAIJobs},
		"consumerGroups": map[string]string{
			StreamJobs:   ConsumerGroup,
			StreamAIJobs: ConsumerGroupAI,
		},
		"fields":           map[string]string{"jobId": FieldJobID, "type": FieldType, "payload": FieldPayload},
		"pipeline":         Pipeline,
		"after":            After,
		"captionLanguages": CaptionLanguages,
		"routes":           routes,
		"examples": map[string]json.RawMessage{
			JobTypeExtractAudio:        example(ExtractAudioInput{AssetID: "11111111-1111-4111-8111-111111111111", StorageKey: "assets/11111111-1111-4111-8111-111111111111/source/input.mp4", AudioKey: "assets/11111111-1111-4111-8111-111111111111/audio/audio.flac"}),
			JobTypeTranscribeAudio:     example(TranscribeAudioInput{AssetID: "11111111-1111-4111-8111-111111111111", AudioKey: "assets/11111111-1111-4111-8111-111111111111/audio/audio.flac", Language: "pt"}),
			JobTypeAnalyzeTranscript:   example(AnalyzeTranscriptInput{AssetID: "11111111-1111-4111-8111-111111111111"}),
			JobTypeSuggestClips:        example(SuggestClipsInput{AssetID: "11111111-1111-4111-8111-111111111111"}),
			JobTypeTranslateTranscript: example(TranslateTranscriptInput{AssetID: "11111111-1111-4111-8111-111111111111", Language: "en"}),
			JobTypeTrackFaces:          example(TrackFacesInput{AssetID: "11111111-1111-4111-8111-111111111111", StorageKey: "assets/11111111-1111-4111-8111-111111111111/source/input.mp4", ClipID: "22222222-2222-4222-8222-222222222222"}),
		},
	}
}

func TestContractsGoldenFile(t *testing.T) {
	got, err := json.MarshalIndent(contractsDoc(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	const path = "testdata/contracts.json"
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run: go test ./packages/job-contracts -update)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is out of date; run: go test ./packages/job-contracts -update\nand update apps/ai-worker to match", path)
	}
}

func TestStreamFor(t *testing.T) {
	if StreamFor(JobTypeTranscribeAudio) != StreamAIJobs || StreamFor(JobTypeAnalyzeTranscript) != StreamAIJobs ||
		StreamFor(JobTypeSuggestClips) != StreamAIJobs || StreamFor(JobTypeTrackFaces) != StreamAIJobs || StreamFor(JobTypeExportClip) != StreamJobs ||
		StreamFor(JobTypeTranslateTranscript) != StreamAIJobs ||
		StreamFor(JobTypeExtractAudio) != StreamJobs || StreamFor("unknown") != StreamJobs {
		t.Fatal("unexpected routing")
	}
	if (Message{Type: JobTypeTranscribeAudio}).Stream() != StreamAIJobs {
		t.Fatal("Message.Stream does not use StreamFor")
	}
}
