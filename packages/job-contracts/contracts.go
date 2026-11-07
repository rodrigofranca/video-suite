package jobcontracts

const (
	JobTypeExtractMetadata   = "extract-metadata"
	JobTypeGenerateThumbnail = "generate-thumbnail"
	JobTypeTranscodeHLS      = "transcode-hls"
)

type ExtractMetadataInput struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
}

type GenerateThumbnailInput struct {
	AssetID    string  `json:"asset_id"`
	StorageKey string  `json:"storage_key"`
	AtSecond   float64 `json:"at_second"`
}

type TranscodeHLSInput struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
}

// Pipeline is the order in which an upload's jobs run. The API creates one
// job row per entry and enqueues the first; when a job ends (done or failed),
// the worker enqueues the next one.
var Pipeline = []string{
	JobTypeExtractMetadata,
	JobTypeGenerateThumbnail,
	JobTypeTranscodeHLS,
}

// NextJobType returns the job type that follows jobType in the Pipeline.
func NextJobType(jobType string) (string, bool) {
	for i, t := range Pipeline {
		if t == jobType && i+1 < len(Pipeline) {
			return Pipeline[i+1], true
		}
	}
	return "", false
}
