package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

// These tests need no database, storage or queue: the ports are fakes.

const (
	assetID = "11111111-1111-4111-8111-111111111111"
	clipID  = "22222222-2222-4222-8222-222222222222"
	otherID = "33333333-3333-4333-8333-333333333333"
)

var now = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// fakeStore keeps every port's rows in maps; a missing row is domain.ErrNotFound.
type fakeStore struct {
	assets       map[string]domain.Asset
	sources      map[string]domain.Source
	clips        map[string]domain.Clip
	transcripts  map[string]domain.Transcript
	translations map[string]bool // asset/language
	presets      map[string]domain.Preset
	vignettes    map[string]domain.Vignette
	watermarks   map[string]domain.Watermark
	busy         bool
	failReads    error // returned by GetTranscript, to see errors pass through
	created      []domain.Job
	finished     map[string]error
}

func newFake() *fakeStore {
	return &fakeStore{
		assets:       map[string]domain.Asset{assetID: {ID: assetID, Filename: "talk.mp4"}},
		sources:      map[string]domain.Source{assetID: {AssetID: assetID, StorageKey: "assets/" + assetID + "/source/input.mp4"}},
		clips:        map[string]domain.Clip{clipID: {ID: clipID, AssetID: assetID, Start: 10, End: 40}},
		transcripts:  map[string]domain.Transcript{assetID: {AssetID: assetID, Language: "pt"}},
		translations: map[string]bool{},
		presets:      map[string]domain.Preset{},
		vignettes:    map[string]domain.Vignette{},
		watermarks:   map[string]domain.Watermark{},
		finished:     map[string]error{},
	}
}

func get[T any](m map[string]T, key string) (T, error) {
	v, ok := m[key]
	if !ok {
		return v, domain.ErrNotFound
	}
	return v, nil
}

func (f *fakeStore) GetAsset(_ context.Context, id string) (domain.Asset, error) {
	return get(f.assets, id)
}
func (f *fakeStore) GetSource(_ context.Context, id string) (domain.Source, error) {
	return get(f.sources, id)
}
func (f *fakeStore) GetClip(_ context.Context, id string) (domain.Clip, error) {
	return get(f.clips, id)
}
func (f *fakeStore) GetTranscript(_ context.Context, id string) (domain.Transcript, error) {
	if f.failReads != nil {
		return domain.Transcript{}, f.failReads
	}
	return get(f.transcripts, id)
}
func (f *fakeStore) GetTranslation(_ context.Context, id, language string) (domain.Translation, error) {
	if !f.translations[id+"/"+language] {
		return domain.Translation{}, domain.ErrNotFound
	}
	return domain.Translation{AssetID: id, Language: language}, nil
}
func (f *fakeStore) GetPreset(_ context.Context, id string) (domain.Preset, error) {
	return get(f.presets, id)
}
func (f *fakeStore) GetVignette(_ context.Context, id string) (domain.Vignette, error) {
	return get(f.vignettes, id)
}
func (f *fakeStore) GetWatermark(_ context.Context, id string) (domain.Watermark, error) {
	return get(f.watermarks, id)
}
func (f *fakeStore) CreateJobIfIdle(_ context.Context, job domain.Job, _ string, then ...domain.Job) error {
	if f.busy {
		return domain.ErrBusy
	}
	f.created = append(append(f.created, job), then...)
	return nil
}
func (f *fakeStore) FinishJob(_ context.Context, id string, jobErr error) error {
	f.finished[id] = jobErr
	return nil
}
func (f *fakeStore) GetJob(_ context.Context, id string) (domain.Job, error) {
	for _, j := range f.created {
		if j.ID == id {
			return j, nil
		}
	}
	return domain.Job{}, domain.ErrNotFound
}

type fakeQueue struct {
	sent []jobcontracts.Message
	err  error
}

func (q *fakeQueue) Enqueue(_ context.Context, msg jobcontracts.Message) error {
	if q.err != nil {
		return q.err
	}
	q.sent = append(q.sent, msg)
	return nil
}

// seq numbers job IDs across use cases, so a fake shared by two runs never
// holds two jobs with the same ID.
var seq int

func useCase(f *fakeStore, q *fakeQueue) ExportClip {
	return ExportClip{
		Assets: f, Clips: f, Speech: f, Brand: f, Jobs: f, Queue: q,
		Now: func() time.Time { return now },
		NewID: func() string {
			seq++
			return fmt.Sprintf("job-%d", seq)
		},
	}
}

func str(s string) *string { return &s }
func yes() *bool           { b := true; return &b }
func no() *bool            { b := false; return &b }

func input(t *testing.T, j domain.Job) jobcontracts.ExportClipInput {
	t.Helper()
	var in jobcontracts.ExportClipInput
	if err := json.Unmarshal(j.InputJSON, &in); err != nil {
		t.Fatal(err)
	}
	return in
}

func kindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return 0
}

func TestExportQueuesTheExport(t *testing.T) {
	f, q := newFake(), &fakeQueue{}
	res, err := useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Translate != nil || len(f.created) != 1 || len(q.sent) != 1 || q.sent[0].JobID != res.Job.ID ||
		q.sent[0].Type != jobcontracts.JobTypeExportClip || !res.Job.CreatedAt.Equal(now) {
		t.Fatalf("result %+v, created %d, sent %+v", res, len(f.created), q.sent)
	}
	in := input(t, res.Job)
	want := jobcontracts.ExportClipInput{AssetID: assetID, ClipID: clipID, StorageKey: "assets/" + assetID + "/source/input.mp4",
		Format: "vertical", Captions: true, CaptionStyle: "highlight",
		OutputKey: "assets/" + assetID + "/clips/" + clipID + "/vertical.mp4"}
	if fmt.Sprint(in) != fmt.Sprint(want) {
		t.Fatalf("input = %+v\nwant    %+v", in, want)
	}
}

func TestExportNamesWhatIsMissing(t *testing.T) {
	f := newFake()
	f.clips[otherID] = domain.Clip{ID: otherID, AssetID: otherID}
	for name, req := range map[string]ExportRequest{
		"a malformed asset":       {AssetID: "nope", ClipID: clipID, Format: "vertical"},
		"an unknown asset":        {AssetID: otherID, ClipID: clipID, Format: "vertical"},
		"a malformed clip":        {AssetID: assetID, ClipID: "nope", Format: "vertical"},
		"an unknown clip":         {AssetID: assetID, ClipID: "44444444-4444-4444-8444-444444444444", Format: "vertical"},
		"a clip of another video": {AssetID: assetID, ClipID: otherID, Format: "vertical"},
	} {
		_, err := useCase(f, &fakeQueue{}).Run(context.Background(), req)
		if kindOf(err) != NotFound {
			t.Errorf("%s: %v, want NotFound", name, err)
		}
	}
	delete(f.sources, assetID)
	_, err := useCase(f, &fakeQueue{}).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical"})
	if kindOf(err) != Conflict || !strings.Contains(err.Error(), "no source file") {
		t.Errorf("without a source: %v", err)
	}
}

func TestExportRejectsChoicesNotOnOffer(t *testing.T) {
	for name, req := range map[string]ExportRequest{
		"format":   {Format: "square"},
		"style":    {Format: "vertical", CaptionStyle: str("comic")},
		"position": {Format: "vertical", CaptionPosition: str("left")},
		"color":    {Format: "vertical", CaptionColor: str("red")},
		"language": {Format: "vertical", CaptionLanguage: str("ja")},
	} {
		req.AssetID, req.ClipID = assetID, clipID
		f := newFake()
		_, err := useCase(f, &fakeQueue{}).Run(context.Background(), req)
		if kindOf(err) != Invalid || !errors.Is(err, ErrInvalidChoice) || !strings.Contains(err.Error(), "formats: original") || len(f.created) != 0 {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestExportWithAPreset(t *testing.T) {
	f, q := newFake(), &fakeQueue{}
	caps := true
	f.presets[otherID] = domain.Preset{ID: otherID, Name: "My channel", Captions: true, CaptionStyle: "box", CaptionPosition: "top",
		CaptionColor: "pink", CaptionCaps: &caps, CaptionLanguage: "pt", IntroVertical: "v1", Watermark: "w1",
		WatermarkPosition: "bottom-left", WatermarkSize: 20, WatermarkOpacity: 60}
	f.vignettes["v1"] = domain.Vignette{ID: "v1", StorageKey: "library/vignettes/v1.mp4"}
	f.watermarks["w1"] = domain.Watermark{ID: "w1", StorageKey: "library/watermarks/w1.png"}

	// The request's color wins; the rest comes from the preset; Portuguese is the transcript's own.
	res, err := useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical-face",
		CaptionColor: str("green"), PresetID: otherID})
	if err != nil {
		t.Fatal(err)
	}
	in := input(t, res.Job)
	if in.Preset != "My channel" || in.CaptionStyle != "box" || in.CaptionPosition != "top" || in.CaptionColor != "green" ||
		in.CaptionCaps == nil || !*in.CaptionCaps || in.CaptionLanguage != "" || res.Translate != nil {
		t.Fatalf("captions = %+v", in)
	}
	if in.Intro == nil || in.Intro.Vertical != "library/vignettes/v1.mp4" || in.Intro.Horizontal != "" || in.Outro != nil {
		t.Fatalf("vignettes = %+v, %+v", in.Intro, in.Outro)
	}
	if in.Watermark == nil || *in.Watermark != (jobcontracts.Watermark{Key: "library/watermarks/w1.png", Position: "bottom-left", Size: 20, Opacity: 60}) {
		t.Fatalf("watermark = %+v", in.Watermark)
	}

	// A preset without captions turns them off unless the request turns them on.
	f.presets[otherID] = domain.Preset{ID: otherID, Name: "Clean", Captions: false, CaptionStyle: "lines"}
	res, _ = useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "original", PresetID: otherID})
	if input(t, res.Job).Captions {
		t.Fatal("the preset's captions off was not applied")
	}
	res, _ = useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "original", PresetID: otherID, Captions: yes()})
	if !input(t, res.Job).Captions {
		t.Fatal("the request's captions on lost to the preset")
	}

	for _, id := range []string{"nope", "55555555-5555-4555-8555-555555555555"} {
		_, err := useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "original", PresetID: id})
		if kindOf(err) != Invalid || err.Error() != "unknown preset" {
			t.Errorf("preset %q: %v", id, err)
		}
	}
}

func TestExportTranslatesFirstWhenNeeded(t *testing.T) {
	f, q := newFake(), &fakeQueue{}
	req := ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical", CaptionLanguage: str("en")}
	res, err := useCase(f, q).Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Translate == nil || len(f.created) != 2 || f.created[0].Type != jobcontracts.JobTypeTranslateTranscript ||
		len(q.sent) != 1 || q.sent[0].JobID != res.Translate.ID {
		t.Fatalf("created %+v, sent %+v", f.created, q.sent)
	}
	if !res.Job.CreatedAt.After(res.Translate.CreatedAt) || input(t, res.Job).CaptionLanguage != "en" {
		t.Fatalf("the export does not wait for the translation: %+v", res.Job)
	}
	var tin jobcontracts.TranslateTranscriptInput
	json.Unmarshal(res.Translate.InputJSON, &tin)
	if tin != (jobcontracts.TranslateTranscriptInput{AssetID: assetID, Language: "en"}) {
		t.Fatalf("translation input = %+v", tin)
	}

	// Once translated, the export goes alone.
	f, q = newFake(), &fakeQueue{}
	f.translations[assetID+"/en"] = true
	res, _ = useCase(f, q).Run(context.Background(), req)
	if res.Translate != nil || len(f.created) != 1 || input(t, res.Job).CaptionLanguage != "en" {
		t.Fatalf("with a translation: %+v", f.created)
	}

	// Without captions, the language does not matter.
	f, q = newFake(), &fakeQueue{}
	req.Captions = no()
	res, _ = useCase(f, q).Run(context.Background(), req)
	if res.Translate != nil || input(t, res.Job).CaptionLanguage != "" {
		t.Fatalf("without captions: %+v", input(t, res.Job))
	}

	// Without a transcript there is nothing to translate.
	f = newFake()
	delete(f.transcripts, assetID)
	_, err = useCase(f, &fakeQueue{}).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical", CaptionLanguage: str("en")})
	if kindOf(err) != Conflict || !strings.Contains(err.Error(), "no transcript") {
		t.Fatalf("without a transcript: %v", err)
	}
}

func TestExportWaitsForTheAssetsJobs(t *testing.T) {
	f, q := newFake(), &fakeQueue{}
	f.busy = true
	_, err := useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical"})
	if kindOf(err) != Conflict || !errors.Is(err, domain.ErrBusy) || len(q.sent) != 0 {
		t.Fatalf("busy: %v, sent %v", err, q.sent)
	}
}

// Failures that the HTTP tests cannot cause.
func TestExportWhenTheQueueOrTheStoreFails(t *testing.T) {
	f, q := newFake(), &fakeQueue{err: errors.New("redis down")}
	_, err := useCase(f, q).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical", CaptionLanguage: str("en")})
	if kindOf(err) != Unavailable || err.(*Error).Msg != "the job could not be queued" {
		t.Fatalf("queue down: %v", err)
	}
	// Both the translation and the export waiting for it are closed, with the reason.
	if len(f.finished) != 2 || !strings.Contains(f.finished[f.created[0].ID].Error(), "redis down") ||
		!strings.Contains(f.finished[f.created[1].ID].Error(), "could not be queued") {
		t.Fatalf("finished = %v", f.finished)
	}

	f = newFake()
	f.failReads = errors.New("connection reset")
	_, err = useCase(f, &fakeQueue{}).Run(context.Background(), ExportRequest{AssetID: assetID, ClipID: clipID, Format: "vertical", CaptionLanguage: str("en")})
	if err == nil || kindOf(err) != 0 || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("a store failure should pass through as an internal error: %v", err)
	}
}
