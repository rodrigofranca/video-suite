package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ultramedia/media-core/db"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/testvideo"

	jobcontracts "ultramedia/job-contracts"
)

// TestTranscodeHLS runs the real handler against the database, storage and
// ffmpeg from the local environment. It is skipped if any is missing.
func TestTranscodeHLS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://ultramedia:ultramedia@localhost:5432/ultramedia?sslmode=disable"
	}
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	// Registered first so it runs last: the cleanups below still need the pool.
	t.Cleanup(pool.Close)
	cfg, err := storage.ConfigFromEnv()
	if err != nil {
		cfg = storage.Config{Endpoint: "http://localhost:9000", Region: "us-east-1", AccessKey: "ultramedia", SecretKey: "ultramedia", Bucket: "ultramedia", PublicURL: "http://localhost:9000"}
	}
	files := storage.New(cfg)
	if err := files.Check(ctx); err != nil {
		t.Skipf("storage not available: %v", err)
	}
	st := store.New(pool)

	// An asset with its transcode job, and a real 8-second video as source.
	asset := domain.Asset{ID: domain.NewID(), Filename: "test.mp4", Status: domain.AssetUploading}
	st.CreateAsset(ctx, asset)
	t.Cleanup(func() { st.DeleteAsset(context.Background(), asset.ID) })
	job := domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: jobcontracts.JobTypeTranscodeHLS, InputJSON: []byte("{}"), Status: domain.JobPending, CreatedAt: time.Now()}
	src := domain.Source{ID: domain.NewID(), AssetID: asset.ID, StorageKey: storage.SourceKey(asset.ID), MimeType: "video/mp4", SizeBytes: 1, Checksum: "x"}
	if err := st.CompleteUpload(ctx, src, []domain.Job{job}); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(t.TempDir(), "in.mp4")
	testvideo.Make(t, video, 8, "640x360")
	if err := files.UploadFile(ctx, src.StorageKey, video, "video/mp4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		files.Delete(bg, src.StorageKey)
		files.Delete(bg, storage.HLSManifestKey(asset.ID))
		for i := range 5 {
			files.Delete(bg, fmt.Sprintf("%ssegment_%03d.ts", storage.HLSPrefix(asset.ID), i))
		}
	})

	h := HLS{Storage: files, Store: st}
	in := jobcontracts.TranscodeHLSInput{AssetID: asset.ID, StorageKey: src.StorageKey}
	if err := h.Transcode(ctx, Job{ID: job.ID, AssetID: asset.ID, Type: job.Type, Log: slog.New(slog.DiscardHandler)}, in); err != nil {
		t.Fatalf("Transcode: %v", err)
	}

	art, err := st.LatestArtifact(ctx, asset.ID, domain.ArtifactHLS)
	if err != nil || art.StorageKey != storage.HLSManifestKey(asset.ID) || art.JobID != job.ID {
		t.Fatalf("artifact = %+v, %v", art, err)
	}
	var meta struct{ Segments int }
	json.Unmarshal(art.MetadataJSON, &meta)
	if meta.Segments != 2 { // 8s in 6s segments
		t.Fatalf("artifact metadata = %s, want 2 segments", art.MetadataJSON)
	}

	// The playlist and its segments are readable without credentials.
	playlist := httpGet(t, files.PublicURL(art.StorageKey))
	if !strings.Contains(playlist, "#EXT-X-ENDLIST") {
		t.Fatalf("playlist:\n%s", playlist)
	}
	if seg := httpGet(t, files.PublicURL(storage.HLSPrefix(asset.ID)+"segment_001.ts")); len(seg) == 0 || seg[0] != 0x47 {
		t.Fatal("segment is not an MPEG-TS file")
	}
}

func TestTranscodeRejectsBadInput(t *testing.T) {
	err := HLS{}.Transcode(context.Background(), Job{Log: slog.New(slog.DiscardHandler)}, jobcontracts.TranscodeHLSInput{AssetID: "../../etc", StorageKey: "k"})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("want invalid input error, got %v", err)
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
