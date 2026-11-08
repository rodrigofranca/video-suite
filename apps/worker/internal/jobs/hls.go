package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
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

// TranscodeHLS converts an asset's source into HLS in several qualities,
// stored under assets/{id}/hls/: a master playlist at hls/index.m3u8 and one
// folder per rendition (docs/adr/0028). It reports progress while FFmpeg
// runs, and removes files of an earlier run that the new one does not use.
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

	// The local probe is instant, and does not depend on the metadata job.
	_, info, err := ffmpeg.Probe(ctx, src, in.StorageKey)
	if err != nil {
		return fmt.Errorf("could not read the video: %w", err)
	}
	renditions := ffmpeg.Ladder(info.Width, info.Height)
	if len(renditions) == 0 {
		return fmt.Errorf("the video has no picture size (%dx%d)", info.Width, info.Height)
	}
	names := make([]string, len(renditions))
	for i, r := range renditions {
		names[i] = r.Name
	}
	job.Log.Info(fmt.Sprintf("encoding %d qualities: %s", len(renditions), strings.Join(names, ", ")))

	out := filepath.Join(dir, "hls")
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	args := ffmpeg.HLSArgs(src, out, renditions, info.AudioCodec != "")
	job.Log.Info("running " + ffmpeg.CommandLine(args))
	start := time.Now()
	opts := ffmpeg.Options{}
	if duration := time.Duration(info.DurationSeconds * float64(time.Second)); duration > 0 {
		opts.Progress = func(written time.Duration) {
			// 99 at most: the job reaches 100 only once the files are uploaded.
			job.Progress(min(int(written*100/duration), 99))
		}
	} else {
		job.Log.Warn("unknown duration; progress will not be shown")
	}
	if err := ffmpeg.Run(ctx, args, job.Log, opts); err != nil {
		return err
	}
	job.Log.Info("ffmpeg finished", "seconds", time.Since(start).Round(100*time.Millisecond).Seconds())

	uploaded, segments, err := m.uploadHLS(ctx, out, in.AssetID)
	if err != nil {
		return err
	}
	job.Log.Info("uploaded HLS output", "files", len(uploaded), "segments", segments, "prefix", storage.HLSPrefix(in.AssetID))
	if removed, err := m.removeStaleHLS(ctx, in.AssetID, uploaded); err != nil {
		// The new stream is complete; old files only take space.
		job.Log.Warn("could not remove the files of an earlier encoding: " + err.Error())
	} else if removed > 0 {
		job.Log.Info(fmt.Sprintf("removed %d files of an earlier encoding", removed))
	}

	meta, _ := json.Marshal(map[string]any{
		"segments": segments, "segmentSeconds": ffmpeg.SegmentSeconds, "renditions": renditions,
	})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactHLS,
		StorageKey: storage.HLSManifestKey(in.AssetID), MetadataJSON: meta,
	})
}

// uploadHLS sends the segments first, then each rendition's playlist, and
// the master playlist last, so a player never sees a playlist that points to
// missing files. It returns the keys it wrote and how many were segments.
func (m Media) uploadHLS(ctx context.Context, dir, assetID string) (map[string]bool, int, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	if !slices.Contains(files, ffmpeg.ManifestName) {
		return nil, 0, fmt.Errorf("ffmpeg produced no %s", ffmpeg.ManifestName)
	}
	order := func(name string) int {
		switch {
		case name == ffmpeg.ManifestName:
			return 2 // the master playlist
		case strings.HasSuffix(name, ".m3u8"):
			return 1
		}
		return 0
	}
	slices.SortFunc(files, func(a, b string) int {
		if d := order(a) - order(b); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})

	uploaded := map[string]bool{}
	segments := 0
	for _, name := range files {
		contentType := "video/mp2t"
		if strings.HasSuffix(name, ".m3u8") {
			contentType = "application/vnd.apple.mpegurl"
		} else {
			segments++
		}
		key := storage.HLSPrefix(assetID) + name
		if err := m.Storage.UploadFile(ctx, key, filepath.Join(dir, filepath.FromSlash(name)), contentType); err != nil {
			return uploaded, segments, err
		}
		uploaded[key] = true
	}
	return uploaded, segments, nil
}

// removeStaleHLS deletes the files under the asset's HLS folder that this
// run did not write: segments of a longer earlier encoding, or the files of
// a single-quality one.
func (m Media) removeStaleHLS(ctx context.Context, assetID string, keep map[string]bool) (int, error) {
	keys, err := m.Storage.List(ctx, storage.HLSPrefix(assetID))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, key := range keys {
		if keep[key] {
			continue
		}
		if err := m.Storage.Delete(ctx, key); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
