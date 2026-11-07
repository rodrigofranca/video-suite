package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// These tests use the PostgreSQL and storage from `make up && make migrate`
// and are skipped when they are not running.

type env struct {
	t       *testing.T
	store   *store.Store
	storage *storage.Client
	handler http.Handler

	mu       sync.Mutex
	enqueued []jobcontracts.Message
	failNext bool
}

func newEnv(t *testing.T, maxUpload int64) *env {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable"
	}
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	t.Cleanup(pool.Close)

	cfg, err := storage.ConfigFromEnv()
	if err != nil {
		cfg = storage.Config{Endpoint: "http://localhost:9000", Region: "us-east-1", AccessKey: "ultramedia", SecretKey: "ultramedia", Bucket: "ultramedia", PublicURL: "http://localhost:9000"}
	}
	st := storage.New(cfg)
	if err := st.Check(ctx); err != nil {
		t.Skipf("storage not available: %v", err)
	}

	e := &env{t: t, store: store.New(pool), storage: st}
	e.handler = NewRouter(Deps{
		DB: pool, Store: e.store, Storage: st, MaxUploadBytes: maxUpload,
		Enqueue: func(_ context.Context, m jobcontracts.Message) error {
			e.mu.Lock()
			defer e.mu.Unlock()
			if e.failNext {
				e.failNext = false
				return errors.New("redis down")
			}
			e.enqueued = append(e.enqueued, m)
			return nil
		},
	})
	return e
}

// upload posts content as a multipart file and returns the response.
func (e *env) upload(filename string, content []byte) *httptest.ResponseRecorder {
	return e.uploadWithLength(filename, content, true)
}

// uploadWithLength can hide the request size, as a chunked upload would.
func (e *env) uploadWithLength(filename string, content []byte, knownLength bool) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filename)
	fw.Write(content)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/assets/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if !knownLength {
		req.ContentLength = -1
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *env) get(path string, out any) int {
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if out != nil {
		json.Unmarshal(rec.Body.Bytes(), out)
	}
	return rec.Code
}

// jobOfType returns the asset's newest job of a type.
func (e *env) jobOfType(assetID, jobType string) domain.Job {
	e.t.Helper()
	jobs, err := e.store.ListJobs(context.Background(), assetID)
	if err != nil {
		e.t.Fatal(err)
	}
	for i := len(jobs) - 1; i >= 0; i-- {
		if jobs[i].Type == jobType {
			return jobs[i]
		}
	}
	e.t.Fatalf("asset %s has no %s job", assetID, jobType)
	return domain.Job{}
}

// fakeMP4 starts with an ftyp box, which is all the upload checks.
func fakeMP4(size int) []byte {
	b := bytes.Repeat([]byte{0x42}, size)
	copy(b, "\x00\x00\x00\x20ftypisom")
	return b
}

func (e *env) cleanupAsset(id string) {
	e.t.Cleanup(func() {
		ctx := context.Background()
		e.storage.Delete(ctx, storage.SourceKey(id))
		e.store.DeleteAsset(ctx, id)
	})
}

func TestUploadCreatesAssetJobsAndEnqueuesFirst(t *testing.T) {
	e := newEnv(t, 0)
	content := fakeMP4(3 << 20)

	rec := e.upload("My Clip.mp4", content)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	var created struct{ AssetID, Status string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	if created.Status != "processing" || !domain.IsID(created.AssetID) {
		t.Fatalf("response = %+v", created)
	}

	var detail struct {
		Filename string
		Status   string
		Source   struct {
			StorageKey string
			SizeBytes  int64
			Checksum   string
		}
		Jobs     []struct{ ID, Type, Status string }
		Playback *struct{ ManifestURL string }
	}
	if code := e.get("/assets/"+created.AssetID, &detail); code != 200 {
		t.Fatalf("GET asset: %d", code)
	}
	sum := sha256.Sum256(content)
	if detail.Filename != "My Clip.mp4" || detail.Status != "processing" || detail.Playback != nil {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.Source.SizeBytes != int64(len(content)) || detail.Source.Checksum != hex.EncodeToString(sum[:]) {
		t.Fatalf("source = %+v", detail.Source)
	}
	if detail.Source.StorageKey != storage.SourceKey(created.AssetID) {
		t.Fatalf("storage key %q is not generated from the asset ID", detail.Source.StorageKey)
	}
	if ok, _ := e.storage.Exists(context.Background(), detail.Source.StorageKey); !ok {
		t.Fatal("source file not in storage")
	}

	if len(detail.Jobs) != len(jobcontracts.Pipeline) {
		t.Fatalf("jobs = %+v", detail.Jobs)
	}
	for i, typ := range jobcontracts.Pipeline {
		if detail.Jobs[i].Type != typ || detail.Jobs[i].Status != "pending" {
			t.Fatalf("job %d = %+v, want pending %s", i, detail.Jobs[i], typ)
		}
	}

	if len(e.enqueued) != 1 || e.enqueued[0].JobID != detail.Jobs[0].ID || e.enqueued[0].Type != jobcontracts.JobTypeExtractMetadata {
		t.Fatalf("enqueued = %+v, want only the first job", e.enqueued)
	}
	in, err := jobcontracts.Decode[jobcontracts.ExtractMetadataInput](e.enqueued[0])
	if err != nil || in.AssetID != created.AssetID || in.StorageKey != detail.Source.StorageKey {
		t.Fatalf("payload = %+v, %v", in, err)
	}

	var playback map[string]string
	if code := e.get("/assets/"+created.AssetID+"/playback", &playback); code != http.StatusConflict || playback["status"] != "processing" {
		t.Fatalf("playback before ready: %d %v", code, playback)
	}
}

func TestPlaybackOnceReady(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("ready.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)

	ctx := context.Background()
	jobs, _ := e.store.ListJobs(ctx, created.AssetID)
	hls := jobs[2]
	e.store.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: created.AssetID, JobID: hls.ID, Type: domain.ArtifactHLS, StorageKey: storage.HLSManifestKey(created.AssetID)})
	e.store.SetAssetStatus(ctx, created.AssetID, domain.AssetReady)

	var pb struct{ Type, ManifestURL string }
	if code := e.get("/assets/"+created.AssetID+"/playback", &pb); code != 200 {
		t.Fatalf("playback: %d", code)
	}
	if pb.Type != "hls" || pb.ManifestURL != e.storage.PublicURL(storage.HLSManifestKey(created.AssetID)) {
		t.Fatalf("playback = %+v", pb)
	}

	var job struct{ ID, Type string }
	if code := e.get("/jobs/"+hls.ID, &job); code != 200 || job.Type != "transcode-hls" {
		t.Fatalf("GET job: %d %+v", code, job)
	}
	e.store.AddJobLog(ctx, hls.ID, domain.LogInfo, "hello")
	var logs struct {
		Logs []struct{ Level, Message string }
	}
	if code := e.get("/jobs/"+hls.ID+"/logs", &logs); code != 200 || len(logs.Logs) != 1 || logs.Logs[0].Message != "hello" {
		t.Fatalf("GET logs: %d %+v", code, logs)
	}
	var list struct{ Assets []struct{ ID string } }
	e.get("/assets?status=ready", &list)
	found := false
	for _, a := range list.Assets {
		found = found || a.ID == created.AssetID
	}
	if !found {
		t.Fatal("ready asset missing from GET /assets?status=ready")
	}
}

func TestUploadRejections(t *testing.T) {
	e := newEnv(t, 1<<20) // 1MB limit
	suffix := time.Now().Format("150405.000000")

	cases := []struct {
		name     string
		filename string
		content  []byte
		want     int
	}{
		{"wrong extension", "clip-" + suffix + ".mov", fakeMP4(1024), http.StatusUnsupportedMediaType},
		{"not an mp4", "text-" + suffix + ".mp4", []byte("this is plain text, not video"), http.StatusUnsupportedMediaType},
		{"too large", "big-" + suffix + ".mp4", fakeMP4(2 << 20), http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		if rec := e.upload(tc.filename, tc.content); rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, rec.Code, tc.want, rec.Body)
		}
	}
	// Without Content-Length the limit is enforced while streaming to storage.
	streamed := "streamed-" + suffix + ".mp4"
	cases = append(cases, struct {
		name     string
		filename string
		content  []byte
		want     int
	}{"too large, streamed", streamed, nil, 0})
	if rec := e.uploadWithLength(streamed, fakeMP4(1<<20+1), false); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("too large, streamed: status %d, want 413 (%s)", rec.Code, rec.Body)
	}

	// Rejected uploads must not leave assets behind.
	assets, _ := e.store.ListAssets(context.Background(), "")
	for _, a := range assets {
		for _, tc := range cases {
			if a.Filename == tc.filename {
				t.Errorf("%s: asset %s was left behind", tc.name, a.ID)
			}
		}
	}
}

func TestUploadEnqueueFailureMarksAssetFailed(t *testing.T) {
	e := newEnv(t, 0)
	e.failNext = true
	name := "orphan-" + time.Now().Format("150405.000000") + ".mp4"
	rec := e.upload(name, fakeMP4(1024))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}

	assets, _ := e.store.ListAssets(context.Background(), domain.AssetFailed)
	var id string
	for _, a := range assets {
		if a.Filename == name {
			id = a.ID
		}
	}
	if id == "" {
		t.Fatal("asset should be kept and marked failed")
	}
	e.cleanupAsset(id)
	jobs, _ := e.store.ListJobs(context.Background(), id)
	if jobs[0].Status != domain.JobFailed || jobs[0].ErrorMessage == nil {
		t.Fatalf("first job = %+v, want failed with a message", jobs[0])
	}
}

func TestMetadataAndThumbnailInResponses(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("meta.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	id := created.AssetID

	if code := e.get("/assets/"+id+"/metadata", nil); code != http.StatusNotFound {
		t.Fatalf("metadata before extraction: %d, want 404", code)
	}
	var before struct {
		ThumbnailURL *string
		Metadata     json.RawMessage
	}
	e.get("/assets/"+id, &before)
	if before.ThumbnailURL != nil || string(before.Metadata) != "null" && before.Metadata != nil {
		t.Fatalf("before = %+v, want nulls", before)
	}

	ctx := context.Background()
	jobs, _ := e.store.ListJobs(ctx, id)
	e.store.SaveTechnicalMetadata(ctx, id, []byte(`{"summary":{"durationSeconds":42.5,"width":1920,"height":1080},"ffprobe":{"format":{"filename":"k"}}}`))
	e.store.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: id, JobID: jobs[1].ID, Type: domain.ArtifactThumbnail, StorageKey: storage.ThumbnailKey(id)})

	type summary struct {
		DurationSeconds float64
		Width, Height   int
	}
	var detail struct {
		ThumbnailURL *string
		Metadata     summary
	}
	e.get("/assets/"+id, &detail)
	wantThumb := e.storage.PublicURL(storage.ThumbnailKey(id))
	if detail.ThumbnailURL == nil || *detail.ThumbnailURL != wantThumb || detail.Metadata.DurationSeconds != 42.5 || detail.Metadata.Width != 1920 {
		t.Fatalf("detail = %+v", detail)
	}

	var list struct {
		Assets []struct {
			ID           string
			ThumbnailURL *string
			Metadata     *summary
		}
	}
	e.get("/assets", &list)
	found := false
	for _, a := range list.Assets {
		if a.ID == id {
			found = a.ThumbnailURL != nil && *a.ThumbnailURL == wantThumb && a.Metadata != nil && a.Metadata.Height == 1080
		}
	}
	if !found {
		t.Fatal("list item is missing the thumbnail or metadata")
	}

	var full struct {
		Summary summary
		FFprobe map[string]any
	}
	if code := e.get("/assets/"+id+"/metadata", &full); code != 200 || full.Summary.DurationSeconds != 42.5 || full.FFprobe["format"] == nil {
		t.Fatalf("GET metadata: %d %+v", code, full)
	}
}

func (e *env) post(path, body string, out any) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	e.handler.ServeHTTP(rec, req)
	if out != nil {
		json.Unmarshal(rec.Body.Bytes(), out)
	}
	return rec.Code
}

func TestRerunJob(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("rerun.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	id := created.AssetID
	ctx := context.Background()

	// While the upload's jobs are pending, nothing can be re-run.
	if code := e.post("/assets/"+id+"/jobs", `{"type":"generate-thumbnail"}`, nil); code != http.StatusConflict {
		t.Fatalf("re-run while pending: %d, want 409", code)
	}
	jobs, _ := e.store.ListJobs(ctx, id)
	for _, j := range jobs {
		e.store.StartJob(ctx, j.ID)
		e.store.FinishJob(ctx, j.ID, nil)
	}
	e.store.SetAssetStatus(ctx, id, domain.AssetReady)

	cases := []struct {
		body string
		want int
	}{
		{`{"type":"make-coffee"}`, http.StatusBadRequest},
		{`not json`, http.StatusBadRequest},
	}
	for _, c := range cases {
		if code := e.post("/assets/"+id+"/jobs", c.body, nil); code != c.want {
			t.Errorf("POST %s: %d, want %d", c.body, code, c.want)
		}
	}

	e.enqueued = nil
	var job struct{ ID, Type, Status string }
	if code := e.post("/assets/"+id+"/jobs", `{"type":"generate-thumbnail"}`, &job); code != http.StatusAccepted {
		t.Fatalf("re-run thumbnail: %d", code)
	}
	if job.Type != "generate-thumbnail" || job.Status != "pending" || len(e.enqueued) != 1 || e.enqueued[0].JobID != job.ID {
		t.Fatalf("job = %+v, enqueued = %+v", job, e.enqueued)
	}
	in, _ := jobcontracts.Decode[jobcontracts.GenerateThumbnailInput](e.enqueued[0])
	if in.AssetID != id || in.StorageKey != storage.SourceKey(id) || in.AtSecond != 3 {
		t.Fatalf("payload = %+v", in)
	}
	if a, _ := e.store.GetAsset(ctx, id); a.Status != domain.AssetReady {
		t.Fatalf("a thumbnail re-run must not change the asset status, got %s", a.Status)
	}

	// The history keeps both attempts.
	if jobs, _ := e.store.ListJobs(ctx, id); len(jobs) != len(jobcontracts.Pipeline)+1 {
		t.Fatalf("jobs = %d, want one more than the pipeline", len(jobs))
	}

	// Re-running HLS makes the asset processing again.
	e.store.StartJob(ctx, job.ID)
	e.store.FinishJob(ctx, job.ID, nil)
	if code := e.post("/assets/"+id+"/jobs", `{"type":"transcode-hls"}`, nil); code != http.StatusAccepted {
		t.Fatalf("re-run HLS: %d", code)
	}
	if a, _ := e.store.GetAsset(ctx, id); a.Status != domain.AssetProcessing {
		t.Fatalf("asset = %s after HLS re-run, want processing", a.Status)
	}
}

func TestTranscriptAndSubtitles(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("speech.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	id := created.AssetID
	ctx := context.Background()

	if code := e.get("/assets/"+id+"/transcript", nil); code != http.StatusNotFound {
		t.Fatalf("transcript before: %d, want 404", code)
	}
	var before struct{ Subtitles []any }
	e.get("/assets/"+id, &before)
	if before.Subtitles == nil || len(before.Subtitles) != 0 {
		t.Fatalf("subtitles before = %#v, want an empty list", before.Subtitles)
	}

	transcribe := e.jobOfType(id, jobcontracts.JobTypeTranscribeAudio)
	e.store.SaveTranscript(ctx, domain.Transcript{ID: domain.NewID(), AssetID: id, JobID: transcribe.ID, Language: "en",
		Text: "hello world", Segments: []byte(`[{"start":0,"end":1.2,"text":"hello world"}]`), Model: "base"})
	for _, lang := range []string{"en", "en", "pt"} { // two English attempts: the newest wins
		e.store.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: id, JobID: transcribe.ID, Type: domain.ArtifactSubtitles,
			StorageKey: storage.SubtitlesKey(id, lang), MetadataJSON: []byte(`{"language":"` + lang + `"}`)})
	}

	var tr struct {
		Language, Model, Text string
		Segments              []struct{ Start, End float64 }
	}
	if code := e.get("/assets/"+id+"/transcript", &tr); code != 200 || tr.Text != "hello world" || tr.Language != "en" || len(tr.Segments) != 1 || tr.Segments[0].End != 1.2 {
		t.Fatalf("transcript: %d %+v", code, tr)
	}
	var detail struct {
		Subtitles []struct{ Language, URL string }
	}
	e.get("/assets/"+id, &detail)
	if len(detail.Subtitles) != 2 {
		t.Fatalf("subtitles = %+v, want one per language", detail.Subtitles)
	}
	for _, s := range detail.Subtitles {
		if s.URL != e.storage.PublicURL(storage.SubtitlesKey(id, s.Language)) {
			t.Fatalf("subtitles URL = %q", s.URL)
		}
	}
}

func TestAnalysis(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("talk.mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	id := created.AssetID
	ctx := context.Background()

	if code := e.get("/assets/"+id+"/analysis", nil); code != http.StatusNotFound {
		t.Fatalf("analysis before: %d, want 404", code)
	}
	analyze := e.jobOfType(id, jobcontracts.JobTypeAnalyzeTranscript)
	var input jobcontracts.AnalyzeTranscriptInput
	if err := json.Unmarshal(analyze.InputJSON, &input); err != nil || input.AssetID != id {
		t.Fatalf("analyze input = %s (%v)", analyze.InputJSON, err)
	}

	save := func(summary string, tags []string) {
		t.Helper()
		err := e.store.SaveAnalysis(ctx, domain.Analysis{ID: domain.NewID(), AssetID: id, JobID: analyze.ID, Language: "pt",
			Summary: summary, Tags: tags, Chapters: []byte(`[{"start":0,"title":"Abertura"},{"start":42.5,"title":"Demonstração"}]`),
			Model: "claude-haiku-5-5"})
		if err != nil {
			t.Fatal(err)
		}
	}
	save("primeira versão", []string{"old"})
	save("Um resumo.", []string{"vídeo", "demo"}) // a re-run replaces the analysis

	var got struct {
		Language, Model, Summary string
		Tags                     []string
		Chapters                 []struct {
			Start float64
			Title string
		}
	}
	if code := e.get("/assets/"+id+"/analysis", &got); code != 200 {
		t.Fatalf("analysis: %d", code)
	}
	if got.Summary != "Um resumo." || got.Language != "pt" || got.Model != "claude-haiku-5-5" ||
		strings.Join(got.Tags, ",") != "vídeo,demo" || len(got.Chapters) != 2 || got.Chapters[1].Start != 42.5 {
		t.Fatalf("analysis = %+v", got)
	}

	// No tags or chapters: empty lists, not null.
	e.store.SaveAnalysis(ctx, domain.Analysis{ID: domain.NewID(), AssetID: id, JobID: analyze.ID, Language: "en", Summary: "Short.", Model: "m"})
	var raw map[string]json.RawMessage
	e.get("/assets/"+id+"/analysis", &raw)
	if string(raw["tags"]) != "[]" || string(raw["chapters"]) != "[]" {
		t.Fatalf("empty analysis: tags %s, chapters %s", raw["tags"], raw["chapters"])
	}
}

func TestClips(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.upload("My Talk (final).mp4", fakeMP4(1024))
	var created struct{ AssetID string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	e.cleanupAsset(created.AssetID)
	id := created.AssetID
	ctx := context.Background()

	var list struct {
		WordTimings bool
		Clips       []struct {
			ID, Title, Reason    string
			Start, End, Duration float64
			Score, Rank          int
			Exports              []struct {
				Format, Status, CaptionStyle string
				Ready, Captions              bool
				Width, Height                int
				SizeBytes                    int64
			}
		}
	}
	if code := e.get("/assets/"+id+"/clips", &list); code != 200 || list.Clips == nil || len(list.Clips) != 0 || list.WordTimings {
		t.Fatalf("clips before: %d %+v", code, list)
	}
	suggest := e.jobOfType(id, jobcontracts.JobTypeSuggestClips)
	second, best := domain.NewID(), domain.NewID()
	err := e.store.ReplaceClips(ctx, id, []domain.Clip{
		{ID: second, JobID: suggest.ID, Start: 60, End: 85.5, Title: "Second", Score: 6, Rank: 2, Model: "m"},
		{ID: best, JobID: suggest.ID, Start: 12, End: 47.5, Title: "Best", Reason: "Strong hook.", Score: 9, Rank: 1, Model: "m"},
	})
	if err != nil {
		t.Fatal(err)
	}
	e.get("/assets/"+id+"/clips", &list)
	if len(list.Clips) != 2 || list.Clips[0].ID != best || list.Clips[0].Duration != 35.5 || list.Clips[0].Reason != "Strong hook." || len(list.Clips[0].Exports) != 0 {
		t.Fatalf("clips = %+v", list.Clips)
	}

	exportPath := "/assets/" + id + "/clips/" + best + "/exports"
	if code := e.post(exportPath, `{"format": "vertical"}`, nil); code != http.StatusConflict {
		t.Fatalf("export while the pipeline runs: %d, want 409", code)
	}
	jobs, _ := e.store.ListJobs(ctx, id)
	for _, j := range jobs {
		e.store.FinishJob(ctx, j.ID, nil)
	}
	if code := e.post(exportPath, `{"format": "square"}`, nil); code != http.StatusBadRequest {
		t.Fatalf("bad format: %d", code)
	}
	if code := e.post(exportPath, `{"format": "vertical", "captionStyle": "comic-sans"}`, nil); code != http.StatusBadRequest {
		t.Fatalf("bad caption style: %d", code)
	}
	if code := e.post("/assets/"+id+"/clips/"+domain.NewID()+"/exports", `{"format": "vertical"}`, nil); code != http.StatusNotFound {
		t.Fatalf("unknown clip: %d", code)
	}
	if code := e.post("/assets/"+id+"/clips/not-a-uuid/exports", `{"format": "vertical"}`, nil); code != http.StatusNotFound {
		t.Fatalf("bad clip id: %d", code)
	}

	var job struct{ ID, Type, Status string }
	if code := e.post(exportPath, `{"format": "vertical", "captions": false}`, &job); code != http.StatusAccepted || job.Type != jobcontracts.JobTypeExportClip {
		t.Fatalf("export: %d %+v", code, job)
	}
	msg := e.enqueued[len(e.enqueued)-1]
	in, err := jobcontracts.Decode[jobcontracts.ExportClipInput](msg)
	if err != nil || msg.Stream() != jobcontracts.StreamJobs || in.ClipID != best || in.Format != "vertical" || in.Captions || in.CaptionStyle != "highlight" ||
		in.StorageKey != storage.SourceKey(id) || in.OutputKey != storage.ClipKey(id, best, "vertical") {
		t.Fatalf("queued %+v (%v) on %s", in, err, msg.Stream())
	}
	e.get("/assets/"+id+"/clips", &list)
	if ex := list.Clips[0].Exports; len(ex) != 1 || ex[0].Format != "vertical" || ex[0].Status != "pending" || ex[0].Ready {
		t.Fatalf("exports while pending = %+v", ex)
	}
	if code := e.get("/clips/"+best+"/download?format=vertical", nil); code != http.StatusNotFound {
		t.Fatalf("download before the export: %d", code)
	}

	// The worker finishes the export.
	e.store.FinishJob(ctx, job.ID, nil)
	e.store.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: id, JobID: job.ID, Type: domain.ArtifactClip,
		StorageKey:   storage.ClipKey(id, best, "vertical"),
		MetadataJSON: []byte(`{"clipId":"` + best + `","format":"vertical","captions":true,"captionStyle":"lines","width":1080,"height":1920,"sizeBytes":4096}`)})
	e.store.SaveTranscript(ctx, domain.Transcript{ID: domain.NewID(), AssetID: id, JobID: e.jobOfType(id, jobcontracts.JobTypeTranscribeAudio).ID,
		Language: "en", Text: "hi", Words: []byte(`[{"start":0,"end":0.4,"word":"hi"}]`), Model: "base"})
	e.get("/assets/"+id+"/clips", &list)
	if ex := list.Clips[0].Exports; len(ex) != 1 || ex[0].Status != "done" || !ex[0].Ready || ex[0].Width != 1080 || ex[0].SizeBytes != 4096 || ex[0].CaptionStyle != "lines" {
		t.Fatalf("exports when done = %+v", ex)
	}
	if !list.WordTimings {
		t.Fatal("wordTimings = false with a transcript that has words")
	}

	rec = httptest.NewRecorder()
	e.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/clips/"+best+"/download?format=vertical", nil))
	loc := rec.Header().Get("Location")
	if rec.Code != http.StatusFound || !strings.Contains(loc, "X-Amz-Signature") ||
		!strings.Contains(loc, "My-Talk-final-clip-1-vertical.mp4") || !strings.Contains(loc, "/clips/"+best+"/vertical.mp4") {
		t.Fatalf("download: %d %s", rec.Code, loc)
	}
	rec = httptest.NewRecorder()
	e.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/clips/"+best+"/download?format=vertical&inline=1", nil))
	if loc := rec.Header().Get("Location"); rec.Code != http.StatusFound || strings.Contains(loc, "attachment") {
		t.Fatalf("inline: %d %s", rec.Code, loc)
	}
	for _, path := range []string{"/clips/" + best + "/download?format=original", "/clips/" + best + "/download?format=x", "/clips/nope/download?format=vertical"} {
		if code := e.get(path, nil); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, code)
		}
	}
}
