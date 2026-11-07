package jobcontracts

// Job types. The Go media worker runs the first four; the Python AI worker
// (apps/ai-worker) runs transcribe-audio. See StreamFor.
const (
	JobTypeExtractMetadata   = "extract-metadata"
	JobTypeGenerateThumbnail = "generate-thumbnail"
	JobTypeTranscodeHLS      = "transcode-hls"
	JobTypeExtractAudio      = "extract-audio"
	JobTypeTranscribeAudio   = "transcribe-audio"
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

// ExtractAudioInput asks for the source's first audio track as 16 kHz mono
// FLAC, the format Whisper works with, stored at AudioKey.
type ExtractAudioInput struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
	AudioKey   string `json:"audio_key"`
}

// TranscribeAudioInput asks the AI worker to transcribe the audio at AudioKey.
// Language is an ISO 639-1 code; empty means detect it.
type TranscribeAudioInput struct {
	AssetID  string `json:"asset_id"`
	AudioKey string `json:"audio_key"`
	Language string `json:"language,omitempty"`
}

// Pipeline is the order in which an upload's jobs run. The API creates one
// job row per entry and enqueues the first; when a job ends (done or failed),
// the worker that ran it enqueues the next one. Transcription comes after HLS
// so the video is playable before the slower AI steps (docs/adr/0017).
var Pipeline = []string{
	JobTypeExtractMetadata,
	JobTypeGenerateThumbnail,
	JobTypeTranscodeHLS,
	JobTypeExtractAudio,
	JobTypeTranscribeAudio,
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
