package jobs

import (
	"context"
	"fmt"

	jobcontracts "ultramedia/job-contracts"
)

// Stubs are placeholder handlers for pipeline steps that are not built yet.
// They log and succeed, so the pipeline moves on to the next job.
type Stubs struct{}

func (Stubs) ExtractMetadata(_ context.Context, job Job, in jobcontracts.ExtractMetadataInput) error {
	called(job, in.AssetID)
	return nil
}

func (Stubs) GenerateThumbnail(_ context.Context, job Job, in jobcontracts.GenerateThumbnailInput) error {
	called(job, in.AssetID)
	return nil
}

func (Stubs) TranscodeHLS(_ context.Context, job Job, in jobcontracts.TranscodeHLSInput) error {
	called(job, in.AssetID)
	return nil
}

func called(job Job, assetID string) {
	job.Log.Info(fmt.Sprintf("stub: %s called for asset %s (not implemented yet)", job.Type, assetID))
}

// RegisterStubs registers stub handlers for every MVP1 job type. Real
// handlers registered afterwards replace them.
func RegisterStubs(d *Dispatcher) {
	s := Stubs{}
	Handle(d, jobcontracts.JobTypeExtractMetadata, s.ExtractMetadata)
	Handle(d, jobcontracts.JobTypeGenerateThumbnail, s.GenerateThumbnail)
	Handle(d, jobcontracts.JobTypeTranscodeHLS, s.TranscodeHLS)
}
