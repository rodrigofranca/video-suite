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
	var jobs []map[string]any
	for _, t := range JobTypes {
		routes[t.Name] = StreamFor(t.Name)
		jobs = append(jobs, map[string]any{"name": t.Name, "label": t.Label, "worker": t.Worker, "rerun": t.Rerun})
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
		"jobs":             jobs,
		"after":            After,
		"events":           map[string]string{"jobQueued": EventJobQueued, "jobFinished": EventJobFinished, "channel": OutboxChannel},
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

// The table agrees with the rest of the contract: the Pipeline comes first,
// in order, and every job type a rule names is in it.
func TestJobTypesTable(t *testing.T) {
	seen := map[string]bool{}
	for i, jt := range JobTypes {
		if seen[jt.Name] || jt.Label == "" || (jt.Worker != WorkerMedia && jt.Worker != WorkerAI) {
			t.Fatalf("bad entry %+v", jt)
		}
		seen[jt.Name] = true
		if i < len(Pipeline) && (Pipeline[i] != jt.Name || !jt.Rerun) {
			t.Fatalf("entry %d is %s; the Pipeline comes first, in order, and its jobs can run again", i, jt.Name)
		}
	}
	for first, then := range RerunWith {
		for _, name := range append([]string{first}, then...) {
			if !seen[name] {
				t.Fatalf("RerunWith names %s, which is not in JobTypes", name)
			}
		}
	}
	for first, then := range After {
		if !seen[first] || !seen[then] {
			t.Fatalf("After names %s and %s; both must be in JobTypes", first, then)
		}
	}
	if Label(JobTypeTranscodeHLS) != "HLS encoding" || Label("make-coffee") != "make-coffee" {
		t.Fatal("Label")
	}
	if got := TypesOf(WorkerAI); !slices.Equal(got, []string{JobTypeTranscribeAudio, JobTypeAnalyzeTranscript,
		JobTypeSuggestClips, JobTypeTrackFaces, JobTypeTranslateTranscript}) {
		t.Fatalf("TypesOf(ai) = %v", got)
	}
}
