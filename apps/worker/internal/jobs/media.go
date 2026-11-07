package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// sourceURLExpiry is how long a signed source URL stays valid. It is longer
// than the job timeout, so a URL never expires mid-job.
const sourceURLExpiry = time.Hour

// Media runs the video processing jobs of the pipeline.
type Media struct {
	Storage *storage.Client
	Store   *store.Store
	TempDir string // where work files go; defaults to the OS temp dir
}

// Register adds the handler of every pipeline job type to d.
func (m Media) Register(d *Dispatcher) {
	Handle(d, jobcontracts.JobTypeExtractMetadata, m.ExtractMetadata)
	Handle(d, jobcontracts.JobTypeGenerateThumbnail, m.GenerateThumbnail)
	Handle(d, jobcontracts.JobTypeTranscodeHLS, m.TranscodeHLS)
}

// validInput rejects inputs that could point outside the asset's folder.
func validInput(assetID, storageKey string) error {
	if !domain.IsID(assetID) || storageKey == "" {
		return fmt.Errorf("invalid input: asset_id %q, storage_key %q", assetID, storageKey)
	}
	return nil
}

// knownDuration returns the duration found by the metadata job, or 0 when it
// is not known (for example because that job failed).
func (m Media) knownDuration(ctx context.Context, assetID string) float64 {
	meta, err := m.Store.GetTechnicalMetadata(ctx, assetID)
	if err != nil {
		return 0
	}
	var doc struct {
		Summary struct {
			DurationSeconds float64 `json:"durationSeconds"`
		} `json:"summary"`
	}
	if json.Unmarshal(meta.MetadataJSON, &doc) != nil {
		return 0
	}
	return doc.Summary.DurationSeconds
}

var errNoImage = errors.New("ffmpeg produced no image")
