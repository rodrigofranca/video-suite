package jobs

import (
	"context"
	"fmt"
	"log/slog"

	jobcontracts "ultramedia/job-contracts"
)

// Stubs are placeholder handlers for the MVP1 pipeline. They only log and
// succeed; real FFmpeg/ffprobe handlers replace them one by one.
type Stubs struct {
	Log *slog.Logger
}

func (s Stubs) ExtractMetadata(_ context.Context, jobID string, in jobcontracts.ExtractMetadataInput) error {
	s.called(jobcontracts.JobTypeExtractMetadata, jobID, in.AssetID)
	return nil
}

func (s Stubs) GenerateThumbnail(_ context.Context, jobID string, in jobcontracts.GenerateThumbnailInput) error {
	s.called(jobcontracts.JobTypeGenerateThumbnail, jobID, in.AssetID)
	return nil
}

func (s Stubs) TranscodeHLS(_ context.Context, jobID string, in jobcontracts.TranscodeHLSInput) error {
	s.called(jobcontracts.JobTypeTranscodeHLS, jobID, in.AssetID)
	return nil
}

func (s Stubs) called(jobType, jobID, assetID string) {
	s.Log.Info(fmt.Sprintf("stub: %s called for asset %s", jobType, assetID), "job_id", jobID)
}

// RegisterStubs registers the stub handlers for every MVP1 job type.
func RegisterStubs(d *Dispatcher, log *slog.Logger) {
	s := Stubs{Log: log}
	Handle(d, jobcontracts.JobTypeExtractMetadata, s.ExtractMetadata)
	Handle(d, jobcontracts.JobTypeGenerateThumbnail, s.GenerateThumbnail)
	Handle(d, jobcontracts.JobTypeTranscodeHLS, s.TranscodeHLS)
}
