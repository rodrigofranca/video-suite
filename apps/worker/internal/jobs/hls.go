package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// HLS transcodes an asset's source into a single-rendition HLS stream and
// stores it under assets/{id}/hls/.
type HLS struct {
	Storage *storage.Client
	Store   *store.Store
	TempDir string // where work files go; defaults to the OS temp dir
}

func (h HLS) Transcode(ctx context.Context, job Job, in jobcontracts.TranscodeHLSInput) error {
	if !domain.IsID(in.AssetID) || in.StorageKey == "" {
		return fmt.Errorf("invalid input: asset_id %q, storage_key %q", in.AssetID, in.StorageKey)
	}
	dir, err := os.MkdirTemp(h.TempDir, "transcode-"+in.AssetID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "source.mp4")
	job.Log.Info("downloading source", "key", in.StorageKey)
	if err := h.Storage.DownloadFile(ctx, in.StorageKey, src); err != nil {
		return err
	}

	out := filepath.Join(dir, "hls")
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	args := ffmpeg.HLSArgs(src, out)
	job.Log.Info("running ffmpeg " + strings.Join(args, " "))
	start := time.Now()
	if err := ffmpeg.Run(ctx, args, job.Log); err != nil {
		return err
	}
	job.Log.Info("ffmpeg finished", "seconds", time.Since(start).Round(100*time.Millisecond).Seconds())

	segments, err := h.upload(ctx, out, in.AssetID)
	if err != nil {
		return err
	}
	job.Log.Info("uploaded HLS output", "segments", segments, "prefix", storage.HLSPrefix(in.AssetID))

	meta, _ := json.Marshal(map[string]any{"segments": segments, "segmentSeconds": ffmpeg.SegmentSeconds})
	return h.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactHLS,
		StorageKey: storage.HLSManifestKey(in.AssetID), MetadataJSON: meta,
	})
}

// upload sends the segments first and the playlist last, so a player never
// sees a playlist that points to missing segments.
func (h HLS) upload(ctx context.Context, dir, assetID string) (segments int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	slices.SortFunc(names, func(a, b string) int {
		if a == ffmpeg.ManifestName {
			return 1
		}
		if b == ffmpeg.ManifestName {
			return -1
		}
		return strings.Compare(a, b)
	})
	if !slices.Contains(names, ffmpeg.ManifestName) {
		return 0, fmt.Errorf("ffmpeg produced no %s", ffmpeg.ManifestName)
	}

	for _, name := range names {
		contentType := "video/mp2t"
		if name == ffmpeg.ManifestName {
			contentType = "application/vnd.apple.mpegurl"
		} else {
			segments++
		}
		if err := h.Storage.UploadFile(ctx, storage.HLSPrefix(assetID)+name, filepath.Join(dir, name), contentType); err != nil {
			return segments, err
		}
	}
	return segments, nil
}
