package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

func TestClipFromAQuote(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("talk.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.AssetID
	e.cleanupAsset(id)
	ctx := context.Background()
	path := "/assets/" + id + "/clips"

	transcribe := e.jobOfType(id, jobcontracts.JobTypeTranscribeAudio)
	e.store.SaveTranscript(ctx, domain.Transcript{ID: domain.NewID(), AssetID: id, JobID: transcribe.ID, Language: "pt", Text: "-",
		Segments: []byte(`[{"start": 12, "end": 16, "text": " A reforma foi aprovada."}]`),
		Words: []byte(`[{"start": 12.1, "end": 12.5, "word": "A"}, {"start": 12.5, "end": 13.4, "word": "reforma"},
			{"start": 13.4, "end": 13.8, "word": "foi"}, {"start": 13.8, "end": 15.0, "word": "aprovada."}]`), Model: "base"})
	e.store.SaveTechnicalMetadata(ctx, id, []byte(`{"summary":{"durationSeconds":120}}`))
	suggest := e.jobOfType(id, jobcontracts.JobTypeSuggestClips)
	e.store.ReplaceClips(ctx, id, []domain.Clip{{ID: domain.NewID(), JobID: suggest.ID, Start: 40, End: 70, Title: "Suggested", Score: 8, Rank: 1, Model: "m"}})

	// The transcript has its words.
	var tr struct {
		Words []struct {
			Start, End float64
			Word       string
		}
	}
	if code := e.get("/assets/"+id+"/transcript", &tr); code != http.StatusOK || len(tr.Words) != 4 || tr.Words[1].Word != "reforma" {
		t.Fatalf("transcript words: %d %+v", code, tr)
	}

	if code := e.send(http.MethodPost, path, `{"start": 12, "end": 15.2}`, nil); code != http.StatusConflict {
		t.Fatalf("while the pipeline runs: %d, want 409", code)
	}
	jobs, _ := e.store.ListJobs(ctx, id)
	for _, j := range jobs {
		e.store.FinishJob(ctx, j.ID, nil)
	}
	for _, c := range []struct{ body, why string }{
		{`{}`, "no range"},
		{`{"start": 12}`, "no end"},
		{`{"start": 12, "end": 14}`, "shorter than 3 s"},
		{`{"start": 100, "end": 121}`, "after the end of the video"},
		{`{"start": 12, "end": 16, "rank": 1}`, "unknown field"},
	} {
		if code := e.send(http.MethodPost, path, c.body, nil); code != http.StatusBadRequest {
			t.Errorf("%s (%s): %d, want 400", c.body, c.why, code)
		}
	}
	if code := e.send(http.MethodPost, "/assets/"+domain.NewID()+"/clips", `{"start": 12, "end": 16}`, nil); code != http.StatusNotFound {
		t.Fatalf("unknown video: %d", code)
	}

	e.markQueue()
	type clipJSON struct {
		ID, Title, Origin string
		Start, End        float64
		Score             *int
		Rank              int
		EditedAt          *string
	}
	var clip clipJSON
	if code := e.send(http.MethodPost, path, `{"start": 12.0, "end": 15.2}`, &clip); code != http.StatusCreated {
		t.Fatalf("create: %d", code)
	}
	if clip.Title != "A reforma foi aprovada." || clip.Origin != "quote" || clip.Score != nil || clip.Rank != 1 || clip.EditedAt != nil {
		t.Fatalf("created: %+v", clip)
	}
	if q := e.queued(); len(q) != 1 || q[0].Type != jobcontracts.JobTypeTrackFaces {
		t.Fatalf("queued %+v, want one track-faces", q)
	}

	var list struct{ Clips []clipJSON }
	e.get(path, &list)
	if len(list.Clips) != 2 || list.Clips[0].ID != clip.ID || list.Clips[1].Origin != "suggested" || list.Clips[1].Score == nil || *list.Clips[1].Score != 8 {
		t.Fatalf("list: %+v", list.Clips)
	}
}
