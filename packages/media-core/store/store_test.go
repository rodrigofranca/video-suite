package store

import (
	"context"
	"errors"
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
