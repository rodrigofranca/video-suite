package store

import (
	"context"
	"reflect"
	"testing"

	"ultramedia/media-core/domain"
)

// saveSpeech gives a new video a transcript with these segments.
func saveSpeech(t *testing.T, s *Store, language, segments string) domain.Asset {
	t.Helper()
	asset, jobs := newUpload(t, s)
	tr := domain.Transcript{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: language, Text: "-",
		Segments: []byte(segments), Model: "base"}
	if err := s.SaveTranscript(context.Background(), tr); err != nil {
		t.Fatal(err)
	}
	return asset
}

// search returns the videos found among want's, in order, with their moments' times.
func search(t *testing.T, s *Store, query string, moments int, among ...string) []SpeechResult {
	t.Helper()
	all, err := s.SearchSpeech(context.Background(), SpeechSearch{Query: query, Limit: 100, Moments: moments})
	if err != nil {
		t.Fatalf("SearchSpeech(%q): %v", query, err)
	}
	ours := map[string]bool{}
	for _, id := range among {
		ours[id] = true
	}
	var out []SpeechResult
	for _, r := range all {
		if ours[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

func TestSearchSpeech(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	pt := saveSpeech(t, s, "pt", `[
		{"start": 0, "end": 4.5, "text": " A votação do quokkário foi adiada."},
		{"start": 4.5, "end": 9, "text": " O ministro disse que os quokkários votam amanhã."},
		{"start": 9, "end": 10, "text": "  "},
		{"start": 10, "end": 14, "text": " Quokkário, quokkário e mais quokkário."}
	]`)
	en := saveSpeech(t, s, "en", `[{"start": 1, "end": 3, "text": "The quokkarios voted again."}]`)
	ids := []string{pt.ID, en.ID}

	// Accents and word endings do not matter. Each transcript is read with
	// its language's rules, and the query in every language the archive
	// has, so the English video matches too. The video with the best
	// segment comes first; its best moments come in the order they were said.
	got := search(t, s, "quokkario", 2, ids...)
	if len(got) != 2 || got[0].ID != pt.ID || got[1].ID != en.ID {
		t.Fatalf("quokkario: %+v", got)
	}
	if r := got[0]; r.Language != "pt" || r.Matches != 3 || r.Filename != "clip.mp4" || len(r.Moments) != 2 {
		t.Fatalf("quokkario, Portuguese video: %+v", r)
	}
	var starts []float64
	for _, m := range got[0].Moments {
		starts = append(starts, m.Start)
	}
	if !reflect.DeepEqual(starts, []float64{0, 10}) {
		t.Fatalf("moments start at %v, want the best and, of equal ones, the earliest, in time order [0 10]", starts)
	}
	wantParts := []TextPart{{Text: "A votação do "}, {Text: "quokkário", Match: true}, {Text: " foi adiada."}}
	if !reflect.DeepEqual(got[0].Moments[0].Parts, wantParts) {
		t.Fatalf("parts = %+v, want %+v", got[0].Moments[0].Parts, wantParts)
	}

	// A phrase, a word left out, and alternatives.
	if got := search(t, s, `"votação do quokkário"`, 5, ids...); len(got) != 1 || got[0].Matches != 1 || got[0].Moments[0].Start != 0 {
		t.Fatalf("phrase: %+v", got)
	}
	if got := search(t, s, "quokkário -amanhã", 5, pt.ID); len(got) != 1 || got[0].Matches != 2 {
		t.Fatalf("left out: %+v", got)
	}
	if got := search(t, s, "ministro or voted", 5, ids...); len(got) != 2 {
		t.Fatalf("or: %d videos, want both", len(got))
	}
	if got := search(t, s, "zzzquokka", 5, ids...); len(got) != 0 {
		t.Fatalf("no match: %+v", got)
	}

	// A new transcript replaces the old one's segments; deleting the video
	// removes them.
	tr, _ := s.GetTranscript(ctx, pt.ID)
	tr.ID, tr.Segments = domain.NewID(), []byte(`[{"start": 0, "end": 2, "text": "Nada a ver."}]`)
	if err := s.SaveTranscript(ctx, tr); err != nil {
		t.Fatal(err)
	}
	if got := search(t, s, "quokkario", 5, ids...); len(got) != 1 || got[0].ID != en.ID {
		t.Fatalf("after a new transcript: %+v", got)
	}
	if err := s.DeleteAsset(ctx, en.ID); err != nil {
		t.Fatal(err)
	}
	if got := search(t, s, "quokkarios", 5, ids...); len(got) != 0 {
		t.Fatalf("after deleting the video: %+v", got)
	}
}

func TestSearchSpeechPages(t *testing.T) {
	s := testStore(t)
	var ids []string
	for range 3 {
		ids = append(ids, saveSpeech(t, s, "en", `[{"start": 0, "end": 2, "text": "A wallabyesque moment."}]`).ID)
	}
	all := search(t, s, "wallabyesque", 1, ids...)
	if len(all) != 3 {
		t.Fatalf("%d videos, want 3", len(all))
	}
	// Equal matches: the newest video first.
	if all[0].ID != ids[2] || all[2].ID != ids[0] {
		t.Fatalf("order %s %s %s, want newest first", all[0].ID, all[1].ID, all[2].ID)
	}
	// The other tests' videos may match too, so page through ours only.
	page, err := s.SearchSpeech(context.Background(), SpeechSearch{Query: "wallabyesque", Limit: 1, Offset: 1, Moments: 1})
	if err != nil || len(page) != 1 {
		t.Fatalf("page: %+v, %v", page, err)
	}
}

func TestTextParts(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []TextPart
	}{
		{"", nil},
		{"no match", []TextPart{{Text: "no match"}}},
		{"\x02all\x03", []TextPart{{Text: "all", Match: true}}},
		{"a \x02b\x03 \x02c\x03.", []TextPart{{Text: "a "}, {Text: "b", Match: true}, {Text: " "}, {Text: "c", Match: true}, {Text: "."}}},
		{"cut \x02off", []TextPart{{Text: "cut "}, {Text: "off", Match: true}}},
	} {
		if got := textParts(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("textParts(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}
