package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

func TestSearch(t *testing.T) {
	e := newEnv(t, 0)
	ctx := context.Background()
	rec := e.upload("sessao.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.AssetID
	e.cleanupAsset(id)
	transcribe := e.jobOfType(id, jobcontracts.JobTypeTranscribeAudio)
	err := e.store.SaveTranscript(ctx, domain.Transcript{ID: domain.NewID(), AssetID: id, JobID: transcribe.ID, Language: "pt", Text: "-",
		Segments: []byte(`[{"start": 2, "end": 5, "text": " A votação do numbatório foi adiada."},
			{"start": 5, "end": 8, "text": " Sem nada a ver."}]`), Model: "base"})
	if err != nil {
		t.Fatal(err)
	}

	type video struct {
		Asset    struct{ ID, Filename string }
		Language string
		Matches  int
		Moments  []struct {
			Start, End float64
			Parts      []map[string]any
		}
	}
	type result struct {
		Query    string
		Searched int
		More     bool
		Videos   []video
	}
	var got result
	if code := e.get("/search?q=++numbatorio++adiada", &got); code != http.StatusOK {
		t.Fatalf("search: %d", code)
	}
	if got.Query != "numbatorio adiada" || got.Searched < 1 || got.More {
		t.Fatalf("search: %+v", got)
	}
	var ours *video
	for i := range got.Videos {
		if got.Videos[i].Asset.ID == id {
			ours = &got.Videos[i]
		}
	}
	if ours == nil || ours.Asset.Filename != "sessao.mp4" || ours.Language != "pt" || ours.Matches != 1 || len(ours.Moments) != 1 {
		t.Fatalf("our video: %+v", ours)
	}
	m := ours.Moments[0]
	if m.Start != 2 || m.End != 5 || len(m.Parts) != 5 || m.Parts[1]["text"] != "numbatório" || m.Parts[1]["match"] != true ||
		m.Parts[3]["text"] != "adiada" || m.Parts[0]["match"] != nil {
		t.Fatalf("moment: %+v", m)
	}

	var none result
	if code := e.get("/search?q=numbatorio+-adiada", &none); code != http.StatusOK || none.Videos == nil {
		t.Fatalf("no match: %d %+v", code, none)
	}
	for _, v := range none.Videos {
		if v.Asset.ID == id {
			t.Fatalf("the left-out word still matched: %+v", v)
		}
	}
}
