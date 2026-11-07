package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/testvideo"

	jobcontracts "ultramedia/job-contracts"
)

// These tests run the real handlers against the database, storage and FFmpeg
// from the local environment, and are skipped if any is missing.

type fixture struct {
	t     *testing.T
	ctx   context.Context
	files *storage.Client
	st    *store.Store
	media Media
	asset string
	jobs  map[string]domain.Job
	src   string

	mu       sync.Mutex
	logs     bytes.Buffer
	progress []int
}

func newFixture(t *testing.T, seconds int, size string) *fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable"
	}
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	t.Cleanup(pool.Close) // registered first so it runs last
	cfg, err := storage.ConfigFromEnv()
	if err != nil {
		cfg = storage.Config{Endpoint: "http://localhost:9000", Region: "us-east-1", AccessKey: "ultramedia", SecretKey: "ultramedia", Bucket: "ultramedia", PublicURL: "http://localhost:9000"}
	}
	files := storage.New(cfg)
	if err := files.Check(ctx); err != nil {
		t.Skipf("storage not available: %v", err)
	}
	st := store.New(pool)

	f := &fixture{t: t, ctx: ctx, files: files, st: st, media: Media{Storage: files, Store: st}, asset: domain.NewID(), jobs: map[string]domain.Job{}}
	f.src = storage.SourceKey(f.asset)
	if err := st.CreateAsset(ctx, domain.Asset{ID: f.asset, Filename: "test.mp4", Status: domain.AssetUploading}); err != nil {
		t.Fatal(err)
	}
	var list []domain.Job
	for i, typ := range jobcontracts.Pipeline {
		j := domain.Job{ID: domain.NewID(), AssetID: f.asset, Type: typ, InputJSON: []byte("{}"), Status: domain.JobPending, CreatedAt: time.Now().Add(time.Duration(i) * time.Millisecond)}
		f.jobs[typ] = j
		list = append(list, j)
	}
	if err := st.CompleteUpload(ctx, domain.Source{ID: domain.NewID(), AssetID: f.asset, StorageKey: f.src, MimeType: "video/mp4", SizeBytes: 1, Checksum: "x"}, list); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(t.TempDir(), "in.mp4")
	testvideo.Make(t, video, seconds, size)
	if err := files.UploadFile(ctx, f.src, video, "video/mp4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		st.DeleteAsset(bg, f.asset)
		for _, key := range []string{f.src, storage.ThumbnailKey(f.asset), storage.HLSManifestKey(f.asset)} {
			files.Delete(bg, key)
		}
		for i := range 10 {
			files.Delete(bg, fmt.Sprintf("%ssegment_%03d.ts", storage.HLSPrefix(f.asset), i))
		}
	})
	return f
}

func (f *fixture) job(jobType string) Job {
	return Job{
		ID: f.jobs[jobType].ID, AssetID: f.asset, Type: jobType,
		Log: slog.New(slog.NewTextHandler(&f.logs, nil)),
		Progress: func(p int) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.progress = append(f.progress, p)
		},
	}
}

func (f *fixture) extractMetadata() {
	f.t.Helper()
	in := jobcontracts.ExtractMetadataInput{AssetID: f.asset, StorageKey: f.src}
	if err := f.media.ExtractMetadata(f.ctx, f.job(jobcontracts.JobTypeExtractMetadata), in); err != nil {
		f.t.Fatalf("ExtractMetadata: %v", err)
	}
}

func (f *fixture) thumbnail() (atSecond float64, width, height int) {
	f.t.Helper()
	in := jobcontracts.GenerateThumbnailInput{AssetID: f.asset, StorageKey: f.src, AtSecond: 3}
	if err := f.media.GenerateThumbnail(f.ctx, f.job(jobcontracts.JobTypeGenerateThumbnail), in); err != nil {
		f.t.Fatalf("GenerateThumbnail: %v\n%s", err, f.logs.String())
	}
	art, err := f.st.LatestArtifact(f.ctx, f.asset, domain.ArtifactThumbnail)
	if err != nil || art.StorageKey != storage.ThumbnailKey(f.asset) {
		f.t.Fatalf("thumbnail artifact = %+v, %v", art, err)
	}
	var meta struct {
		AtSecond      float64
		Width, Height int
	}
	json.Unmarshal(art.MetadataJSON, &meta)
	return meta.AtSecond, meta.Width, meta.Height
}

func TestExtractMetadata(t *testing.T) {
	f := newFixture(t, 8, "640x360")
	f.extractMetadata()

	m, err := f.st.GetTechnicalMetadata(f.ctx, f.asset)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Summary struct {
			DurationSeconds float64
			Width, Height   int
			VideoCodec      string
			AudioCodec      string
		}
		FFprobe struct {
			Format struct{ Filename string }
		}
	}
	json.Unmarshal(m.MetadataJSON, &doc)
	s := doc.Summary
	if s.Width != 640 || s.Height != 360 || s.VideoCodec != "h264" || s.AudioCodec != "aac" || s.DurationSeconds < 7.9 || s.DurationSeconds > 8.2 {
		t.Fatalf("summary = %+v", s)
	}
	if doc.FFprobe.Format.Filename != f.src {
		t.Fatalf("stored filename = %q, want the storage key", doc.FFprobe.Format.Filename)
	}
	if strings.Contains(string(m.MetadataJSON), "X-Amz") || strings.Contains(f.logs.String(), "X-Amz") {
		t.Fatal("the signed URL leaked into stored metadata or logs")
	}
}

func TestThumbnailUsesDuration(t *testing.T) {
	f := newFixture(t, 2, "1920x1080")
	f.extractMetadata()

	at, w, h := f.thumbnail() // 3s requested, video is 2s long
	if at < 0.9 || at > 1.1 || w != 1280 || h != 720 {
		t.Fatalf("thumbnail at %.2fs, %dx%d; want about 1s, 1280x720", at, w, h)
	}
	resp, err := http.Get(f.files.PublicURL(storage.ThumbnailKey(f.asset)))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/jpeg" || len(body) < 1000 {
		t.Fatalf("public thumbnail: %d %s, %d bytes", resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
	}
	if strings.Contains(f.logs.String(), "X-Amz") {
		t.Fatal("the signed URL leaked into the job logs")
	}
}

func TestThumbnailFallsBackToFirstFrame(t *testing.T) {
	f := newFixture(t, 2, "640x360")
	// No metadata: the duration is unknown and 3s is past the end.
	at, w, _ := f.thumbnail()
	if at != 0 || w != 640 {
		t.Fatalf("thumbnail at %.2fs, width %d; want the first frame at 640", at, w)
	}
	if !strings.Contains(f.logs.String(), "using the first frame") {
		t.Fatalf("missing fallback log:\n%s", f.logs.String())
	}
}

func TestTranscodeHLS(t *testing.T) {
	f := newFixture(t, 8, "640x360")
	in := jobcontracts.TranscodeHLSInput{AssetID: f.asset, StorageKey: f.src}
	if err := f.media.TranscodeHLS(f.ctx, f.job(jobcontracts.JobTypeTranscodeHLS), in); err != nil {
		t.Fatalf("TranscodeHLS: %v", err)
	}

	art, err := f.st.LatestArtifact(f.ctx, f.asset, domain.ArtifactHLS)
	if err != nil || art.StorageKey != storage.HLSManifestKey(f.asset) {
		t.Fatalf("artifact = %+v, %v", art, err)
	}
	var meta struct{ Segments int }
	json.Unmarshal(art.MetadataJSON, &meta)
	if meta.Segments != 2 { // 8s in 6s segments
		t.Fatalf("artifact metadata = %s, want 2 segments", art.MetadataJSON)
	}
	if playlist := httpGet(t, f.files.PublicURL(art.StorageKey)); !strings.Contains(playlist, "#EXT-X-ENDLIST") {
		t.Fatalf("playlist:\n%s", playlist)
	}
	if seg := httpGet(t, f.files.PublicURL(storage.HLSPrefix(f.asset)+"segment_001.ts")); len(seg) == 0 || seg[0] != 0x47 {
		t.Fatal("segment is not an MPEG-TS file")
	}

	if len(f.progress) == 0 {
		t.Fatal("no progress reported")
	}
	for i, p := range f.progress {
		if p < 0 || p > 99 || (i > 0 && p < f.progress[i-1]) {
			t.Fatalf("progress = %v, want increasing values up to 99", f.progress)
		}
	}
	if last := f.progress[len(f.progress)-1]; last < 80 {
		t.Fatalf("last progress = %d, want close to the end", last)
	}
}

func TestRejectsBadInput(t *testing.T) {
	m := Media{}
	job := Job{Log: slog.New(slog.DiscardHandler), Progress: func(int) {}}
	bad := "../../etc"
	errs := []error{
		m.ExtractMetadata(context.Background(), job, jobcontracts.ExtractMetadataInput{AssetID: bad, StorageKey: "k"}),
		m.GenerateThumbnail(context.Background(), job, jobcontracts.GenerateThumbnailInput{AssetID: bad, StorageKey: "k"}),
		m.TranscodeHLS(context.Background(), job, jobcontracts.TranscodeHLSInput{AssetID: bad, StorageKey: "k"}),
	}
	for i, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "invalid input") {
			t.Errorf("handler %d: want invalid input error, got %v", i, err)
		}
	}
}

func TestThumbnailSecond(t *testing.T) {
	cases := []struct{ requested, duration, want float64 }{
		{3, 60, 3},  // long video: as requested
		{3, 2, 1},   // short video: middle
		{3, 0, 3},   // unknown duration: as requested
		{-1, 10, 0}, // negative: start
	}
	for _, c := range cases {
		if got := thumbnailSecond(c.requested, c.duration); got != c.want {
			t.Errorf("thumbnailSecond(%v, %v) = %v, want %v", c.requested, c.duration, got, c.want)
		}
	}
}

func httpGet(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d %s", url, resp.StatusCode, body)
	}
	return string(body)
}
