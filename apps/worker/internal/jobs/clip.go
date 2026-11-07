package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/captions"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// Captions per screen: vertical video has room for fewer words per line.
const (
	captionCharsVertical = 32
	captionCharsOriginal = 84
)

// ExportClip cuts one suggested clip out of the source as an MP4 in the
// requested format, optionally with the subtitles burned in (docs/adr/0020).
// The clip's start and end come from the database, read when the job runs.
func (m Media) ExportClip(ctx context.Context, job Job, in jobcontracts.ExportClipInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	if !domain.IsID(in.ClipID) || !slices.Contains(jobcontracts.ClipFormats, in.Format) ||
		in.OutputKey != storage.ClipKey(in.AssetID, in.ClipID, in.Format) {
		return fmt.Errorf("invalid input: clip_id %q, format %q, output_key %q", in.ClipID, in.Format, in.OutputKey)
	}
	clip, err := m.Store.GetClip(ctx, in.ClipID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && clip.AssetID != in.AssetID) {
		return fmt.Errorf("clip %s not found for asset %s; the clips were probably suggested again", in.ClipID, in.AssetID)
	}
	if err != nil {
		return err
	}
	duration := clip.End - clip.Start
	job.Log.Info(fmt.Sprintf("exporting clip %q, %.1fs to %.1fs (%.1fs), format %s", clip.Title, clip.Start, clip.End, duration, in.Format))

	url, err := m.Storage.PresignGet(ctx, in.StorageKey, sourceURLExpiry)
	if err != nil {
		return err
	}
	_, src, err := ffmpeg.Probe(ctx, url, in.StorageKey)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(m.TempDir, "clip-"+in.ClipID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	width, height := ffmpeg.ClipSize(in.Format, src.Width, src.Height)
	spec := ffmpeg.ClipSpec{
		Input: url, Start: clip.Start, Duration: duration, Format: in.Format,
		SourceWidth: src.Width, SourceHeight: src.Height, Output: filepath.Join(dir, "clip.mp4"),
	}
	language := ""
	if in.Captions {
		spec.SubtitlesFile, language, err = m.clipCaptions(ctx, job, in.AssetID, clip, width, height, dir)
		if err != nil {
			return err
		}
	}

	args, err := ffmpeg.ClipArgs(spec)
	if err != nil {
		return err
	}
	job.Log.Info("running " + ffmpeg.CommandLine(args, url))
	started := time.Now()
	total := time.Duration(duration * float64(time.Second))
	err = ffmpeg.Run(ctx, args, job.Log, ffmpeg.Options{
		Redact:   []string{url},
		Progress: func(written time.Duration) { job.Progress(min(int(written*100/total), 99)) },
	})
	if err != nil {
		return err
	}
	info, err := os.Stat(spec.Output)
	if err != nil {
		return err
	}
	job.Log.Info(fmt.Sprintf("encoded %dx%d in %s, %d KB", width, height, time.Since(started).Round(100*time.Millisecond), info.Size()/1024))

	if err := m.Storage.UploadFile(ctx, in.OutputKey, spec.Output, "video/mp4"); err != nil {
		return err
	}
	job.Log.Info("uploaded clip", "key", in.OutputKey)
	meta, _ := json.Marshal(map[string]any{
		"clipId": clip.ID, "format": in.Format, "captions": spec.SubtitlesFile != "", "language": language,
		"start": clip.Start, "end": clip.End, "durationSeconds": duration,
		"width": width, "height": height, "sizeBytes": info.Size(),
	})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactClip,
		StorageKey: in.OutputKey, MetadataJSON: meta,
	})
}

// clipCaptions writes the clip's part of the asset's subtitles as an ASS file
// sized for the clip. It returns an empty path, not an error, when there is
// nothing to burn in: no subtitles, or no speech in the clip.
func (m Media) clipCaptions(ctx context.Context, job Job, assetID string, clip domain.Clip, width, height int, dir string) (string, string, error) {
	art, err := m.Store.LatestArtifact(ctx, assetID, domain.ArtifactSubtitles)
	if errors.Is(err, store.ErrNotFound) {
		job.Log.Warn("no subtitles for this video; exporting without captions")
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var meta struct {
		Language string `json:"language"`
	}
	json.Unmarshal(art.MetadataJSON, &meta)

	vttPath := filepath.Join(dir, "subtitles.vtt")
	if err := m.Storage.DownloadFile(ctx, art.StorageKey, vttPath); err != nil {
		return "", "", err
	}
	f, err := os.Open(vttPath)
	if err != nil {
		return "", "", err
	}
	all, err := captions.ParseVTT(f)
	f.Close()
	if err != nil {
		return "", "", err
	}
	maxChars := captionCharsOriginal
	if height > width {
		maxChars = captionCharsVertical
	}
	cues := captions.Split(captions.Cut(all, clip.Start, clip.End), maxChars)
	if len(cues) == 0 {
		job.Log.Warn("no subtitles in this part of the video; exporting without captions")
		return "", "", nil
	}

	assPath := filepath.Join(dir, "captions.ass")
	out, err := os.Create(assPath)
	if err != nil {
		return "", "", err
	}
	if err := captions.WriteASS(out, cues, captions.StyleFor(width, height)); err != nil {
		out.Close()
		return "", "", err
	}
	if err := out.Close(); err != nil {
		return "", "", err
	}
	job.Log.Info(fmt.Sprintf("burning in %d captions (%s)", len(cues), meta.Language))
	return assPath, meta.Language, nil
}
