package store

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
)

// testStore connects to the migrated database from `make up && make migrate`,
// or skips the test.
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT 1 FROM media_assets LIMIT 1"); err != nil {
		pool.Close()
		t.Skipf("database not migrated: %v", err)
	}
	t.Cleanup(pool.Close)
	return New(pool)
}

// newUpload creates an asset with its source and the three pipeline jobs.
func newUpload(t *testing.T, s *Store) (domain.Asset, []domain.Job) {
	t.Helper()
	ctx := context.Background()
	asset := domain.Asset{ID: domain.NewID(), Filename: "clip.mp4", Status: domain.AssetUploading}
	if err := s.CreateAsset(ctx, asset); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DeleteAsset(context.Background(), asset.ID) })

	base := time.Now()
	var jobs []domain.Job
	for i, typ := range []string{"extract-metadata", "generate-thumbnail", "transcode-hls"} {
		jobs = append(jobs, domain.Job{
			ID: domain.NewID(), AssetID: asset.ID, Type: typ, InputJSON: []byte(`{"asset_id":"x"}`),
			Status: domain.JobPending, CreatedAt: base.Add(time.Duration(i) * time.Millisecond),
		})
	}
	src := domain.Source{ID: domain.NewID(), AssetID: asset.ID, StorageKey: "k", MimeType: "video/mp4", SizeBytes: 10, Checksum: "abc"}
	if err := s.CompleteUpload(ctx, src, jobs); err != nil {
		t.Fatal(err)
	}
	return asset, jobs
}

func TestUploadCreatesProcessingAssetWithJobs(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)

	got, err := s.GetAsset(ctx, asset.ID)
	if err != nil || got.Status != domain.AssetProcessing {
		t.Fatalf("GetAsset = %+v, %v; want status processing", got, err)
	}
	if src, err := s.GetSource(ctx, asset.ID); err != nil || src.SizeBytes != 10 {
		t.Fatalf("GetSource = %+v, %v", src, err)
	}
	list, err := s.ListJobs(ctx, asset.ID)
	if err != nil || len(list) != 3 {
		t.Fatalf("ListJobs = %d jobs, %v", len(list), err)
	}
	for i := range list {
		if list[i].ID != jobs[i].ID || list[i].Status != domain.JobPending {
			t.Fatalf("job %d = %s/%s, want %s pending", i, list[i].Type, list[i].Status, jobs[i].Type)
		}
	}

	processing, err := s.ListAssets(ctx, domain.AssetProcessing)
	if err != nil || !containsAsset(processing, asset.ID) {
		t.Fatalf("ListAssets(processing) missing asset: %v", err)
	}
	ready, _ := s.ListAssets(ctx, domain.AssetReady)
	if containsAsset(ready, asset.ID) {
		t.Fatal("ListAssets(ready) should not include a processing asset")
	}
}

func TestJobLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, jobs := newUpload(t, s)
	job := jobs[0]

	if ok, err := s.StartJob(ctx, job.ID); !ok || err != nil {
		t.Fatalf("StartJob = %v, %v", ok, err)
	}
	// A job left running (worker crash) can be started again.
	if ok, _ := s.StartJob(ctx, job.ID); !ok {
		t.Fatal("restarting a running job should succeed")
	}
	if err := s.FinishJob(ctx, job.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetJob(ctx, job.ID)
	if got.Status != domain.JobDone || got.Progress != 100 || got.StartedAt == nil || got.CompletedAt == nil {
		t.Fatalf("finished job = %+v", got)
	}
	// A duplicate delivery of a finished job is skipped.
	if ok, _ := s.StartJob(ctx, job.ID); ok {
		t.Fatal("starting a done job should be refused")
	}

	failed := jobs[2]
	s.StartJob(ctx, failed.ID)
	s.FinishJob(ctx, failed.ID, errors.New("ffmpeg exited with status 1"))
	got, _ = s.GetJob(ctx, failed.ID)
	if got.Status != domain.JobFailed || got.ErrorMessage == nil || *got.ErrorMessage != "ffmpeg exited with status 1" {
		t.Fatalf("failed job = %+v", got)
	}

	next, err := s.NextPendingJob(ctx, job.AssetID, "generate-thumbnail")
	if err != nil || next.ID != jobs[1].ID {
		t.Fatalf("NextPendingJob = %+v, %v", next, err)
	}
	if _, err := s.NextPendingJob(ctx, job.AssetID, "transcode-hls"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("NextPendingJob for a failed job: err = %v, want ErrNotFound", err)
	}
}

func TestLogsAndArtifacts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)

	s.AddJobLog(ctx, jobs[2].ID, domain.LogInfo, "first")
	s.AddJobLog(ctx, jobs[2].ID, domain.LogError, "second")
	logs, err := s.ListJobLogs(ctx, jobs[2].ID)
	if err != nil || len(logs) != 2 || logs[0].Message != "first" || logs[1].Level != domain.LogError {
		t.Fatalf("ListJobLogs = %+v, %v", logs, err)
	}

	if _, err := s.LatestArtifact(ctx, asset.ID, domain.ArtifactHLS); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LatestArtifact before creation: %v", err)
	}
	art := domain.Artifact{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[2].ID, Type: domain.ArtifactHLS, StorageKey: "assets/x/hls/index.m3u8"}
	if err := s.CreateArtifact(ctx, art); err != nil {
		t.Fatal(err)
	}
	got, err := s.LatestArtifact(ctx, asset.ID, domain.ArtifactHLS)
	if err != nil || got.StorageKey != art.StorageKey || string(got.MetadataJSON) != "{}" {
		t.Fatalf("LatestArtifact = %+v, %v", got, err)
	}
}

func TestDeleteAssetCascades(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	if err := s.DeleteAsset(ctx, asset.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAsset(ctx, asset.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAsset after delete: %v", err)
	}
	if _, err := s.GetJob(ctx, jobs[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetJob after delete: %v", err)
	}
}

func containsAsset(list []AssetListItem, id string) bool {
	for _, a := range list {
		if a.ID == id {
			return true
		}
	}
	return false
}

func TestMetadataThumbnailAndProgress(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)

	if _, err := s.GetTechnicalMetadata(ctx, asset.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetTechnicalMetadata before extraction: %v", err)
	}
	// Saving twice keeps a single row with the latest result.
	s.SaveTechnicalMetadata(ctx, asset.ID, []byte(`{"summary":{"durationSeconds":1},"ffprobe":{}}`))
	if err := s.SaveTechnicalMetadata(ctx, asset.ID, []byte(`{"summary":{"durationSeconds":12.5},"ffprobe":{}}`)); err != nil {
		t.Fatal(err)
	}
	m, err := s.GetTechnicalMetadata(ctx, asset.ID)
	if err != nil || !strings.Contains(string(m.MetadataJSON), "12.5") {
		t.Fatalf("GetTechnicalMetadata = %s, %v", m.MetadataJSON, err)
	}

	s.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[1].ID, Type: domain.ArtifactThumbnail, StorageKey: "assets/x/thumbnails/thumbnail.jpg"})
	list, err := s.ListAssets(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var item *AssetListItem
	for i := range list {
		if list[i].ID == asset.ID {
			item = &list[i]
		}
	}
	if item == nil || item.ThumbnailKey != "assets/x/thumbnails/thumbnail.jpg" || !strings.Contains(string(item.Summary), "12.5") {
		t.Fatalf("list item = %+v", item)
	}

	// Progress only changes while the job runs, and is clamped to 0-100.
	s.SetJobProgress(ctx, jobs[2].ID, 50)
	if got, _ := s.GetJob(ctx, jobs[2].ID); got.Progress != 0 {
		t.Fatalf("pending job progress = %d, want 0", got.Progress)
	}
	s.StartJob(ctx, jobs[2].ID)
	s.SetJobProgress(ctx, jobs[2].ID, 140)
	if got, _ := s.GetJob(ctx, jobs[2].ID); got.Progress != 100 {
		t.Fatalf("progress = %d, want clamped to 100", got.Progress)
	}
}

func TestCreateJobIfIdle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	rerun := func(typ, status string) error {
		return s.CreateJobIfIdle(ctx, domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: typ, InputJSON: []byte("{}"), CreatedAt: time.Now()}, status)
	}

	// The upload's jobs are still pending.
	if err := rerun("generate-thumbnail", ""); !errors.Is(err, ErrBusy) {
		t.Fatalf("while pending: err = %v, want ErrBusy", err)
	}
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	s.SetAssetStatus(ctx, asset.ID, domain.AssetReady)

	if err := rerun("transcode-hls", domain.AssetProcessing); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListJobs(ctx, asset.ID)
	if len(list) != 4 || list[3].Type != "transcode-hls" || list[3].Status != domain.JobPending {
		t.Fatalf("jobs after re-run = %d, last %+v", len(list), list[len(list)-1])
	}
	if a, _ := s.GetAsset(ctx, asset.ID); a.Status != domain.AssetProcessing {
		t.Fatalf("asset = %s, want processing", a.Status)
	}
	// A second re-run waits for the first one.
	if err := rerun("transcode-hls", domain.AssetProcessing); !errors.Is(err, ErrBusy) {
		t.Fatalf("second re-run: err = %v, want ErrBusy", err)
	}
	if err := s.CreateJobIfIdle(ctx, domain.Job{ID: domain.NewID(), AssetID: domain.NewID(), Type: "x", InputJSON: []byte("{}"), CreatedAt: time.Now()}, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown asset: err = %v, want ErrNotFound", err)
	}
}

func TestCreateJobIfIdleWithJobsAfterIt(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	job := func(typ string) domain.Job {
		return domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: typ, InputJSON: []byte("{}"), CreatedAt: time.Now()}
	}
	clips, faces := job("suggest-clips"), job("track-faces")
	if err := s.CreateJobIfIdle(ctx, clips, "", faces); err != nil {
		t.Fatal(err)
	}
	for _, j := range []domain.Job{clips, faces} {
		if got, err := s.GetJob(ctx, j.ID); err != nil || got.Status != domain.JobPending || got.Type != j.Type {
			t.Fatalf("job %s = %+v, %v; want pending", j.Type, got, err)
		}
	}
	if next, err := s.NextPendingJob(ctx, asset.ID, "track-faces"); err != nil || next.ID != faces.ID {
		t.Fatalf("next track-faces = %+v, %v", next, err)
	}
}

func TestEditClipIfIdle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	clip := domain.Clip{ID: domain.NewID(), JobID: jobs[0].ID, Start: 10, End: 30, Title: "A", Score: 5, Rank: 1, Model: "m"}
	if err := s.ReplaceClips(ctx, asset.ID, []domain.Clip{clip}); err != nil {
		t.Fatal(err)
	}
	s.SetFaceTrack(ctx, clip.ID, []byte(`{"coverage":1}`))
	edited := clip
	edited.Title = "B"
	if err := s.EditClipIfIdle(ctx, edited, nil); !errors.Is(err, ErrBusy) {
		t.Fatalf("while the upload's jobs are pending: %v, want ErrBusy", err)
	}
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	if err := s.EditClipIfIdle(ctx, edited, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetClip(ctx, clip.ID)
	if got.Title != "B" || got.EditedAt == nil || got.FaceTrack == nil {
		t.Fatalf("title edit: %+v", got)
	}
	edited.Start, edited.End = 5, 20
	retrack := domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: "track-faces", InputJSON: []byte("{}"), CreatedAt: time.Now()}
	if err := s.EditClipIfIdle(ctx, edited, &retrack); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetClip(ctx, clip.ID)
	if got.Start != 5 || got.End != 20 || got.FaceTrack != nil {
		t.Fatalf("times edit: %+v", got)
	}
	if j, err := s.GetJob(ctx, retrack.ID); err != nil || j.Status != domain.JobPending {
		t.Fatalf("retrack job = %+v, %v", j, err)
	}
	if err := s.EditClipIfIdle(ctx, domain.Clip{ID: domain.NewID(), Start: 1, End: 9, Title: "x"}, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown clip: %v, want ErrNotFound", err)
	}
	// New suggestions replace the others but keep the edited clip.
	if err := s.ReplaceClips(ctx, asset.ID, []domain.Clip{{ID: domain.NewID(), JobID: jobs[0].ID, Start: 40, End: 60, Title: "New", Score: 6, Rank: 1, Model: "m"}}); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListClips(ctx, asset.ID); len(list) != 2 {
		t.Fatalf("after new suggestions: %d clips, want the new one and the edited one", len(list))
	}
	edited.End = edited.Start // the database refuses an empty range
	s.FinishJob(ctx, retrack.ID, nil)
	if err := s.EditClipIfIdle(ctx, edited, nil); err == nil {
		t.Fatal("an empty range was saved")
	}
}

func TestTranscripts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)

	if _, err := s.GetTranscript(ctx, asset.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("before: err = %v, want ErrNotFound", err)
	}
	first := domain.Transcript{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: "en", Text: "hello", Model: "tiny"}
	if err := s.SaveTranscript(ctx, first); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetTranscript(ctx, asset.ID); string(got.Words) != "[]" {
		t.Fatalf("words of a transcript saved without them = %s, want []", got.Words)
	}
	second := domain.Transcript{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: "pt", Text: "olá mundo",
		Segments: []byte(`[{"start":0,"end":1.5,"text":"olá mundo"}]`),
		Words:    []byte(`[{"start":0,"end":0.6,"word":"olá"},{"start":0.7,"end":1.5,"word":"mundo"}]`), Model: "base"}
	if err := s.SaveTranscript(ctx, second); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTranscript(ctx, asset.ID)
	if err != nil || got.Language != "pt" || got.Text != "olá mundo" || got.Model != "base" || !strings.Contains(string(got.Segments), `"end": 1.5`) ||
		!strings.Contains(string(got.Words), `"word": "mundo"`) {
		t.Fatalf("GetTranscript = %+v (%s), %v", got, got.Segments, err)
	}
}

func TestTranslations(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	transcript := domain.Transcript{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: "pt", Text: "olá mundo", Model: "base"}
	if err := s.SaveTranscript(ctx, transcript); err != nil {
		t.Fatal(err)
	}
	if langs, err := s.TranslationLanguages(ctx, asset.ID); err != nil || len(langs) != 0 {
		t.Fatalf("before: languages = %v, %v; want none", langs, err)
	}
	if _, err := s.GetTranslation(ctx, asset.ID, "en"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("before: err = %v, want ErrNotFound", err)
	}
	for _, tr := range []domain.Translation{
		{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: "en", SourceLanguage: "pt", Model: "m1"},
		{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Language: "es", SourceLanguage: "pt", Model: "m1"},
		// Again into English: replaces the first.
		{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[1].ID, Language: "en", SourceLanguage: "pt", Model: "m2",
			Segments: []byte(`[{"start":0,"end":1.5,"text":"hello world"}]`),
			Words:    []byte(`[{"start":0,"end":0.7,"word":"hello"},{"start":0.7,"end":1.5,"word":"world"}]`)},
	} {
		if err := s.SaveTranslation(ctx, tr); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.GetTranslation(ctx, asset.ID, "en")
	if err != nil || got.Model != "m2" || got.JobID != jobs[1].ID || got.SourceLanguage != "pt" ||
		!strings.Contains(string(got.Words), `"word": "world"`) || !strings.Contains(string(got.Segments), "hello world") {
		t.Fatalf("GetTranslation = %+v (%s), %v", got, got.Words, err)
	}
	if langs, err := s.TranslationLanguages(ctx, asset.ID); err != nil || strings.Join(langs, ",") != "en,es" {
		t.Fatalf("languages = %v, %v; want en,es", langs, err)
	}

	// A new transcript: the translations no longer match it.
	transcript.ID, transcript.Text = domain.NewID(), "olá de novo"
	if err := s.SaveTranscript(ctx, transcript); err != nil {
		t.Fatal(err)
	}
	if langs, err := s.TranslationLanguages(ctx, asset.ID); err != nil || len(langs) != 0 {
		t.Fatalf("after a new transcript: languages = %v, %v; want none", langs, err)
	}
}

func TestDeleteAssetIfIdle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	if err := s.DeleteAssetIfIdle(ctx, asset.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("with pending jobs: %v, want ErrBusy", err)
	}
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	if err := s.DeleteAssetIfIdle(ctx, asset.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAsset(ctx, asset.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after deleting: %v, want ErrNotFound", err)
	}
	if list, _ := s.ListJobs(ctx, asset.ID); len(list) != 0 {
		t.Fatalf("jobs left: %d", len(list))
	}
	if err := s.DeleteAssetIfIdle(ctx, asset.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("twice: %v, want ErrNotFound", err)
	}

	// An upload in progress waits; one that stopped over an hour ago can go.
	uploading := domain.Asset{ID: domain.NewID(), Filename: "big.mp4", Status: domain.AssetUploading}
	s.CreateAsset(ctx, uploading)
	t.Cleanup(func() { s.DeleteAsset(context.Background(), uploading.ID) })
	if err := s.DeleteAssetIfIdle(ctx, uploading.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("while uploading: %v, want ErrBusy", err)
	}
	s.db.Exec(ctx, `UPDATE media_assets SET updated_at = NOW() - INTERVAL '2 hours' WHERE id = $1`, uploading.ID)
	if err := s.DeleteAssetIfIdle(ctx, uploading.ID); err != nil {
		t.Fatalf("abandoned upload: %v", err)
	}
}

func TestDeleteClipIfIdle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	gone, kept := domain.NewID(), domain.NewID()
	s.ReplaceClips(ctx, asset.ID, []domain.Clip{
		{ID: gone, JobID: jobs[0].ID, Start: 0, End: 20, Title: "A", Score: 5, Rank: 1, Model: "m"},
		{ID: kept, JobID: jobs[0].ID, Start: 30, End: 50, Title: "B", Score: 4, Rank: 2, Model: "m"},
	})
	for _, id := range []string{gone, kept} {
		s.CreateArtifact(ctx, domain.Artifact{ID: domain.NewID(), AssetID: asset.ID, JobID: jobs[0].ID, Type: domain.ArtifactClip,
			StorageKey: "k/" + id, MetadataJSON: []byte(`{"clipId":"` + id + `"}`)})
	}
	if err := s.DeleteClipIfIdle(ctx, asset.ID, gone); err != nil {
		t.Fatal(err)
	}
	clips, _ := s.ListClips(ctx, asset.ID)
	arts, _ := s.ListArtifacts(ctx, asset.ID)
	if len(clips) != 1 || clips[0].ID != kept || len(arts) != 1 || !strings.Contains(string(arts[0].MetadataJSON), kept) {
		t.Fatalf("left: %d clips, %d artifacts", len(clips), len(arts))
	}
	if err := s.DeleteClipIfIdle(ctx, asset.ID, gone); !errors.Is(err, ErrNotFound) {
		t.Fatalf("twice: %v, want ErrNotFound", err)
	}
	if err := s.DeleteClipIfIdle(ctx, domain.NewID(), kept); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another asset's clip: %v, want ErrNotFound", err)
	}
	s.CreateJobIfIdle(ctx, domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: "export-clip", InputJSON: []byte("{}"), CreatedAt: time.Now()}, "")
	if err := s.DeleteClipIfIdle(ctx, asset.ID, kept); !errors.Is(err, ErrBusy) {
		t.Fatalf("while a job is pending: %v, want ErrBusy", err)
	}
}

func TestVignettesAndPresets(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newVignette := func(name string) domain.Vignette {
		t.Helper()
		v := domain.Vignette{ID: domain.NewID(), Name: name, Filename: name + ".mp4", SizeBytes: 1000}
		v.StorageKey = "library/vignettes/" + v.ID + ".mp4"
		if err := s.CreateVignette(ctx, v); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.db.Exec(context.Background(), `DELETE FROM vignettes WHERE id = $1`, v.ID) })
		return v
	}
	intro, outro := newVignette("Logo"), newVignette("Ending")
	list, err := s.ListVignettes(ctx)
	if err != nil || len(list) < 2 {
		t.Fatalf("ListVignettes = %d, %v", len(list), err)
	}
	if got, err := s.GetVignette(ctx, intro.ID); err != nil || got.Name != "Logo" || got.StorageKey != intro.StorageKey || got.SizeBytes != 1000 {
		t.Fatalf("GetVignette = %+v, %v", got, err)
	}

	caps := true
	name := "Channel " + domain.NewID()[:8]
	p := domain.Preset{ID: domain.NewID(), Name: name, Captions: true, CaptionStyle: "box", CaptionPosition: "top",
		CaptionColor: "pink", CaptionCaps: &caps, CaptionLanguage: "en", IntroVertical: intro.ID, OutroHorizontal: outro.ID}
	if err := s.CreatePreset(ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DeletePreset(context.Background(), p.ID) })
	same := domain.Preset{ID: domain.NewID(), Name: strings.ToUpper(name), CaptionStyle: "lines"}
	if err := s.CreatePreset(ctx, same); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("a second preset with the name in other case: %v, want ErrNameTaken", err)
	}
	got, err := s.GetPreset(ctx, p.ID)
	if err != nil || got.Name != name || got.CaptionStyle != "box" || got.CaptionCaps == nil || !*got.CaptionCaps ||
		got.CaptionLanguage != "en" || got.IntroVertical != intro.ID || got.IntroHorizontal != "" || got.OutroHorizontal != outro.ID {
		t.Fatalf("GetPreset = %+v, %v", got, err)
	}

	// The vignettes in use cannot be deleted.
	var inUse *InUseError
	if err := s.DeleteVignette(ctx, intro.ID); !errors.As(err, &inUse) || inUse.Presets[0] != name {
		t.Fatalf("delete a vignette in use: %v", err)
	}
	p.IntroVertical, p.CaptionCaps, p.Name = "", nil, name+" v2"
	if err := s.UpdatePreset(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetPreset(ctx, p.ID)
	if got.IntroVertical != "" || got.CaptionCaps != nil || got.Name != name+" v2" || !got.UpdatedAt.After(got.CreatedAt) {
		t.Fatalf("after the update: %+v", got)
	}
	if err := s.DeleteVignette(ctx, intro.ID); err != nil {
		t.Fatalf("delete a vignette no longer used: %v", err)
	}
	if err := s.DeleteVignette(ctx, intro.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete it again: %v", err)
	}
	if err := s.UpdatePreset(ctx, domain.Preset{ID: domain.NewID(), Name: "x", CaptionStyle: "lines"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update an unknown preset: %v", err)
	}
	presets, err := s.ListPresets(ctx)
	if err != nil || len(presets) == 0 {
		t.Fatalf("ListPresets = %v, %v", presets, err)
	}
	if err := s.DeletePreset(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetPreset(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestWatermarks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := domain.Watermark{ID: domain.NewID(), Name: "Logo", Filename: "logo.png", SizeBytes: 500, Width: 400, Height: 200}
	w.StorageKey = "library/watermarks/" + w.ID + ".png"
	if err := s.CreateWatermark(ctx, w); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.db.Exec(context.Background(), `DELETE FROM watermarks WHERE id = $1`, w.ID) })
	if got, err := s.GetWatermark(ctx, w.ID); err != nil || got.Width != 400 || got.Height != 200 || got.StorageKey != w.StorageKey {
		t.Fatalf("GetWatermark = %+v, %v", got, err)
	}
	if list, err := s.ListWatermarks(ctx); err != nil || len(list) == 0 {
		t.Fatalf("ListWatermarks = %v, %v", list, err)
	}

	// Without watermark choices, a preset gets the defaults.
	plain := domain.Preset{ID: domain.NewID(), Name: "Plain " + domain.NewID()[:8], CaptionStyle: "lines", CaptionPosition: "bottom"}
	if err := s.CreatePreset(ctx, plain); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DeletePreset(context.Background(), plain.ID) })
	if got, _ := s.GetPreset(ctx, plain.ID); got.Watermark != "" || got.WatermarkPosition != "top-right" || got.WatermarkSize != 15 || got.WatermarkOpacity != 80 {
		t.Fatalf("defaults = %+v", got)
	}

	p := domain.Preset{ID: domain.NewID(), Name: "Marked " + domain.NewID()[:8], CaptionStyle: "lines", CaptionPosition: "bottom",
		Watermark: w.ID, WatermarkPosition: "bottom-left", WatermarkSize: 25, WatermarkOpacity: 60}
	if err := s.CreatePreset(ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DeletePreset(context.Background(), p.ID) })
	got, _ := s.GetPreset(ctx, p.ID)
	if got.Watermark != w.ID || got.WatermarkPosition != "bottom-left" || got.WatermarkSize != 25 || got.WatermarkOpacity != 60 {
		t.Fatalf("GetPreset = %+v", got)
	}
	var inUse *InUseError
	if err := s.DeleteWatermark(ctx, w.ID); !errors.As(err, &inUse) || inUse.What != "watermark" || inUse.Presets[0] != p.Name ||
		!strings.HasPrefix(err.Error(), "the watermark is used by") {
		t.Fatalf("delete an image in use: %v", err)
	}
	p.Watermark = ""
	if err := s.UpdatePreset(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteWatermark(ctx, w.ID); err != nil {
		t.Fatalf("delete a free image: %v", err)
	}
	if err := s.DeleteWatermark(ctx, w.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete it again: %v", err)
	}
}

func TestUsage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	// Far in the future, so the totals since then hold only these rows.
	day1 := time.Date(2090, 1, 1, 10, 0, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)
	cost := func(v float64) *float64 { return &v }
	var ids []string
	for _, r := range []struct {
		asset, jobType, model string
		in, out               int
		cost                  *float64
		at                    time.Time
	}{
		{asset.ID, "analyze-transcript", "claude-haiku-5-5", 1000, 200, cost(0.0002), day1},
		{asset.ID, "analyze-transcript", "claude-haiku-5-5", 1100, 250, cost(0.000235), day1}, // a repair
		{asset.ID, "suggest-clips", "claude-haiku-5-5", 3000, 500, nil, day2},                 // no price
		{"", "translate-transcript", "gpt-6-luna", 400, 400, cost(0.001), day2},               // a deleted video
		{asset.ID, "suggest-clips", "claude-haiku-5-5", 9, 9, cost(1), day1.Add(-48 * time.Hour)},
	} {
		id := domain.NewID()
		ids = append(ids, id)
		job := jobs[0].ID
		if r.asset == "" {
			job = ""
		}
		if err := s.AddUsage(ctx, id, r.asset, job, r.jobType, r.model, r.in, r.out, r.cost, r.at); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, id := range ids {
			s.db.Exec(context.Background(), `DELETE FROM llm_usage WHERE id = $1`, id)
		}
	})

	total, byAction, err := s.AssetUsage(ctx, asset.ID)
	if err != nil || total.Calls != 4 || total.PromptTokens != 5109 || total.CompletionTokens != 959 || total.UnpricedCalls != 1 ||
		math.Abs(total.CostUSD-1.000435) > 1e-9 || len(byAction) != 2 || byAction[0].Key != "analyze-transcript" || byAction[0].Calls != 2 {
		t.Fatalf("AssetUsage = %+v %+v, %v", total, byAction, err)
	}

	sum, err := s.UsageSince(ctx, day1.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total.Calls != 4 || sum.Total.PromptTokens != 5500 || math.Abs(sum.Total.CostUSD-0.001435) > 1e-9 {
		t.Fatalf("total = %+v", sum.Total)
	}
	if len(sum.ByDay) != 2 || sum.ByDay[0].Key != "2090-01-01" || sum.ByDay[0].Calls != 2 || sum.ByDay[1].Key != "2090-01-02" {
		t.Fatalf("by day = %+v", sum.ByDay)
	}
	if len(sum.ByModel) != 2 || sum.ByModel[0].Key != "claude-haiku-5-5" || sum.ByModel[1].Key != "gpt-6-luna" {
		t.Fatalf("by model = %+v", sum.ByModel)
	}
	if len(sum.ByAction) != 3 || sum.ByAction[0].Key != "suggest-clips" {
		t.Fatalf("by action = %+v", sum.ByAction)
	}
	if len(sum.ByAsset) != 2 || sum.ByAsset[0].Key != asset.ID || sum.ByAsset[0].Label != "clip.mp4" || sum.ByAsset[1].Key != "" {
		t.Fatalf("by video = %+v", sum.ByAsset)
	}
}

func TestCreateClipIfIdle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	asset, jobs := newUpload(t, s)
	suggested := domain.Clip{ID: domain.NewID(), JobID: jobs[0].ID, Start: 40, End: 60, Title: "Suggested", Score: 7, Rank: 1, Model: "m"}
	if err := s.ReplaceClips(ctx, asset.ID, []domain.Clip{suggested}); err != nil {
		t.Fatal(err)
	}
	quote := func(start float64, title string) (domain.Clip, domain.Job) {
		c := domain.Clip{ID: domain.NewID(), AssetID: asset.ID, Start: start, End: start + 5, Title: title, CreatedAt: time.Now()}
		j := domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: "track-faces", InputJSON: []byte(`{"clip_id":"` + c.ID + `"}`), CreatedAt: time.Now()}
		return c, j
	}
	first, track := quote(10, "First quote")
	if err := s.CreateClipIfIdle(ctx, first, track); !errors.Is(err, ErrBusy) {
		t.Fatalf("while the upload's jobs are pending: %v, want ErrBusy", err)
	}
	for _, j := range jobs {
		s.StartJob(ctx, j.ID)
		s.FinishJob(ctx, j.ID, nil)
	}
	if err := s.CreateClipIfIdle(ctx, first, track); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetClip(ctx, first.ID)
	if err != nil || got.Origin != domain.ClipQuote || got.JobID != "" || got.Score != 0 || got.Rank != 1 || got.EditedAt != nil || got.Title != "First quote" {
		t.Fatalf("quote clip = %+v, %v", got, err)
	}
	if j, err := s.GetJob(ctx, track.ID); err != nil || j.Status != domain.JobPending || j.Type != "track-faces" {
		t.Fatalf("track job = %+v, %v", j, err)
	}
	if next, err := s.NextPendingJob(ctx, asset.ID, "track-faces"); err != nil || next.ID != track.ID {
		t.Fatalf("queued track-faces = %+v, %v", next, err)
	}
	// While it tracks faces, another quote waits.
	second, track2 := quote(20, "Second quote")
	if err := s.CreateClipIfIdle(ctx, second, track2); !errors.Is(err, ErrBusy) {
		t.Fatalf("while tracking: %v, want ErrBusy", err)
	}
	s.FinishJob(ctx, track.ID, nil)
	if err := s.CreateClipIfIdle(ctx, second, track2); err != nil {
		t.Fatal(err)
	}
	s.FinishJob(ctx, track2.ID, nil)

	// The newest quote first, then the earlier one, then the suggestion.
	list, _ := s.ListClips(ctx, asset.ID)
	var order []string
	for _, c := range list {
		order = append(order, c.Title)
	}
	if strings.Join(order, ",") != "Second quote,First quote,Suggested" {
		t.Fatalf("order = %v", order)
	}
	if list[2].Origin != domain.ClipSuggested || list[2].JobID != jobs[0].ID || list[2].Score != 7 {
		t.Fatalf("suggested clip = %+v", list[2])
	}

	// New suggestions keep the quotes.
	if err := s.ReplaceClips(ctx, asset.ID, []domain.Clip{{ID: domain.NewID(), JobID: jobs[0].ID, Start: 70, End: 90, Title: "New", Score: 6, Rank: 3, Model: "m"}}); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListClips(ctx, asset.ID); len(list) != 3 {
		t.Fatalf("after new suggestions: %d clips, want the two quotes and the new one", len(list))
	}

	unknown, track3 := quote(1, "x")
	unknown.AssetID = domain.NewID()
	if err := s.CreateClipIfIdle(ctx, unknown, track3); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown asset: %v, want ErrNotFound", err)
	}
	// A suggested clip needs its job and score.
	_, err = s.db.Exec(ctx, `INSERT INTO clips (id, asset_id, start_seconds, end_seconds, title, rank, model) VALUES ($1, $2, 1, 5, 'x', 9, 'm')`,
		domain.NewID(), asset.ID)
	if err == nil {
		t.Fatal("a suggested clip without a job was saved")
	}
}
