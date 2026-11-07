package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// ExtractAudio saves the source's first audio track as 16 kHz mono FLAC for
// the transcription job. A video without audio is not an error: the job logs
// it and stores nothing, and transcription then has nothing to do.
func (m Media) ExtractAudio(ctx context.Context, job Job, in jobcontracts.ExtractAudioInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	if in.AudioKey != storage.AudioKey(in.AssetID) {
		return fmt.Errorf("invalid input: audio_key %q", in.AudioKey)
	}
	url, err := m.Storage.PresignGet(ctx, in.StorageKey, sourceURLExpiry)
	if err != nil {
		return err
	}
	_, s, err := ffmpeg.Probe(ctx, url, in.StorageKey)
	if err != nil {
		return err
	}
	if s.AudioCodec == "" {
		job.Log.Info("the video has no audio track; there is nothing to transcribe")
		return nil
	}

	dir, err := os.MkdirTemp(m.TempDir, "audio-"+in.AssetID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "audio.flac")

	args := ffmpeg.AudioArgs(url, out)
	job.Log.Info("running " + ffmpeg.CommandLine(args, url))
	duration := time.Duration(s.DurationSeconds * float64(time.Second))
	opts := ffmpeg.Options{Redact: []string{url}}
	if duration > 0 {
		opts.Progress = func(written time.Duration) {
			job.Progress(min(int(written*100/duration), 99))
		}
	}
	if err := ffmpeg.Run(ctx, args, job.Log, opts); err != nil {
		return err
	}
	info, err := os.Stat(out)
	if err != nil {
		return fmt.Errorf("ffmpeg produced no audio file: %w", err)
	}

	if err := m.Storage.UploadFile(ctx, in.AudioKey, out, "audio/flac"); err != nil {
		return err
	}
	job.Log.Info("uploaded audio", "key", in.AudioKey, "bytes", info.Size())

	meta, _ := json.Marshal(map[string]any{
		"sampleRate": 16000, "channels": 1, "durationSeconds": s.DurationSeconds, "sizeBytes": info.Size(),
	})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactAudio,
		StorageKey: in.AudioKey, MetadataJSON: meta,
	})
}
