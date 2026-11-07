package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// ExtractMetadata runs ffprobe on the source and stores the result. FFprobe
// reads the file through a signed URL, fetching only the parts it needs.
func (m Media) ExtractMetadata(ctx context.Context, job Job, in jobcontracts.ExtractMetadataInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	url, err := m.Storage.PresignGet(ctx, in.StorageKey, sourceURLExpiry)
	if err != nil {
		return err
	}

	job.Log.Info("running ffprobe on the source", "key", in.StorageKey)
	raw, s, err := ffmpeg.Probe(ctx, url, in.StorageKey)
	if err != nil {
		return err
	}
	job.Log.Info("metadata extracted",
		"duration", fmt.Sprintf("%.2fs", s.DurationSeconds),
		"resolution", fmt.Sprintf("%dx%d", s.Width, s.Height),
		"video", s.VideoCodec, "audio", s.AudioCodec, "fps", s.FrameRate)

	doc, err := json.Marshal(map[string]any{"summary": s, "ffprobe": json.RawMessage(raw)})
	if err != nil {
		return err
	}
	return m.Store.SaveTechnicalMetadata(ctx, in.AssetID, doc)
}
