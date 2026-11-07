package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// GenerateThumbnail saves one frame of the video as a JPEG. It takes the
// frame at in.AtSecond, or at the middle of the video when the video is
// shorter than that, and falls back to the first frame if needed.
func (m Media) GenerateThumbnail(ctx context.Context, job Job, in jobcontracts.GenerateThumbnailInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	duration := m.knownDuration(ctx, in.AssetID)
	at := thumbnailSecond(in.AtSecond, duration)

	url, err := m.Storage.PresignGet(ctx, in.StorageKey, sourceURLExpiry)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(m.TempDir, "thumbnail-"+in.AssetID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "thumbnail.jpg")

	if duration > 0 {
		job.Log.Info(fmt.Sprintf("taking the frame at %.2fs of %.2fs", at, duration))
	} else {
		job.Log.Info(fmt.Sprintf("taking the frame at %.2fs (duration unknown)", at))
	}
	err = grabFrame(ctx, job, url, at, out)
	if err == errNoImage && at > 0 {
		job.Log.Warn(fmt.Sprintf("no frame at %.2fs; using the first frame", at))
		at = 0
		err = grabFrame(ctx, job, url, at, out)
	}
	if err != nil {
		return err
	}

	width, height, err := jpegSize(out)
	if err != nil {
		return err
	}
	key := storage.ThumbnailKey(in.AssetID)
	if err := m.Storage.UploadFile(ctx, key, out, "image/jpeg"); err != nil {
		return err
	}
	job.Log.Info("uploaded thumbnail", "key", key, "size", fmt.Sprintf("%dx%d", width, height))

	meta, _ := json.Marshal(map[string]any{"atSecond": at, "width": width, "height": height})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactThumbnail,
		StorageKey: key, MetadataJSON: meta,
	})
}

// thumbnailSecond picks where to take the thumbnail: the requested second, or
// the middle of the video when the video is not that long.
func thumbnailSecond(requested, duration float64) float64 {
	requested = max(requested, 0)
	if duration > 0 && requested >= duration {
		return duration / 2
	}
	return requested
}

// grabFrame writes the frame at the given second to out, or returns
// errNoImage when FFmpeg found no frame there.
func grabFrame(ctx context.Context, job Job, url string, at float64, out string) error {
	args := ffmpeg.ThumbnailArgs(url, at, out)
	job.Log.Info("running " + ffmpeg.CommandLine(args, url))
	if err := ffmpeg.Run(ctx, args, job.Log, ffmpeg.Options{Redact: []string{url}}); err != nil {
		return err
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		return errNoImage
	}
	return nil
}

func jpegSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("read thumbnail: %w", err)
	}
	return cfg.Width, cfg.Height, nil
}
