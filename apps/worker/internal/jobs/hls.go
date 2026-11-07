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
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// TranscodeHLS converts an asset's source into a single-rendition HLS stream
// stored under assets/{id}/hls/, reporting progress while FFmpeg runs.
func (m Media) TranscodeHLS(ctx context.Context, job Job, in jobcontracts.TranscodeHLSInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(m.TempDir, "transcode-"+in.AssetID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "source.mp4")
	job.Log.Info("downloading source", "key", in.StorageKey)
	if err := m.Storage.DownloadFile(ctx, in.StorageKey, src); err != nil {
		return err
	}

	// The local probe is instant and keeps progress working even if the
	// metadata job failed.
	var duration time.Duration
	if _, s, err := ffmpeg.Probe(ctx, src, in.StorageKey); err == nil {
		duration = time.Duration(s.DurationSeconds * float64(time.Second))
	} else {
		job.Log.Warn("could not read the duration; progress will not be shown: " + err.Error())
	}

	out := filepath.Join(dir, "hls")
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	args := ffmpeg.HLSArgs(src, out)
	job.Log.Info("running " + ffmpeg.CommandLine(args))
	start := time.Now()
	opts := ffmpeg.Options{}
	if duration > 0 {
		opts.Progress = func(written time.Duration) {
			// 99 at most: the job reaches 100 only once the files are uploaded.
			job.Progress(min(int(written*100/duration), 99))
		}
	}
	if err := ffmpeg.Run(ctx, args, job.Log, opts); err != nil {
		return err
	}
	job.Log.Info("ffmpeg finished", "seconds", time.Since(start).Round(100*time.Millisecond).Seconds())

	segments, err := m.uploadHLS(ctx, out, in.AssetID)
	if err != nil {
		return err
	}
	job.Log.Info("uploaded HLS output", "segments", segments, "prefix", storage.HLSPrefix(in.AssetID))

	meta, _ := json.Marshal(map[string]any{"segments": segments, "segmentSeconds": ffmpeg.SegmentSeconds})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactHLS,
		StorageKey: storage.HLSManifestKey(in.AssetID), MetadataJSON: meta,
	})
}

// upload sends the segments first and the playlist last, so a player never
// sees a playlist that points to missing segments.
func (m Media) uploadHLS(ctx context.Context, dir, assetID string) (segments int, err error) {
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
		if err := m.Storage.UploadFile(ctx, storage.HLSPrefix(assetID)+name, filepath.Join(dir, name), contentType); err != nil {
			return segments, err
		}
	}
	return segments, nil
}
