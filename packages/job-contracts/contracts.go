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
