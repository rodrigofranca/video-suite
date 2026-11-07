package jobcontracts

// Job types. The Go media worker runs the FFmpeg jobs (metadata, thumbnail,
// HLS, audio and clip export); the Python AI worker (apps/ai-worker) runs
// transcribe-audio, analyze-transcript and suggest-clips. See StreamFor.
const (
	JobTypeExtractMetadata   = "extract-metadata"
	JobTypeGenerateThumbnail = "generate-thumbnail"
	JobTypeTranscodeHLS      = "transcode-hls"
	JobTypeExtractAudio      = "extract-audio"
	JobTypeTranscribeAudio   = "transcribe-audio"
	JobTypeAnalyzeTranscript = "analyze-transcript"
	JobTypeSuggestClips      = "suggest-clips"
	// JobTypeExportClip runs on request, not in the Pipeline.
	JobTypeExportClip = "export-clip"
)

// Clip export formats (ExportClipInput.Format).
const (
	// ClipFormatOriginal keeps the video's shape, at most 1920 px on the long side.
	ClipFormatOriginal = "original"
	// ClipFormatVertical crops the middle of the frame to 9:16 (1080x1920).
	ClipFormatVertical = "vertical"
	// ClipFormatVerticalBlur fits the whole frame into 9:16 over a blurred copy.
	ClipFormatVerticalBlur = "vertical-blur"
)

// ClipFormats lists the export formats, in the order the dashboard shows them.
var ClipFormats = []string{ClipFormatOriginal, ClipFormatVertical, ClipFormatVerticalBlur}

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

// AnalyzeTranscriptInput asks the AI worker for a summary, tags and chapters
// of the asset's transcript, written by a language model (docs/adr/0019).
// The transcript is read from the database, not from the message.
type AnalyzeTranscriptInput struct {
	AssetID string `json:"asset_id"`
}

// SuggestClipsInput asks the AI worker for the moments of the transcript that
// work as short standalone clips (docs/adr/0020). The suggestions replace the
// asset's earlier ones.
type SuggestClipsInput struct {
	AssetID string `json:"asset_id"`
}

// ExportClipInput asks the media worker to cut a clip out of the source as an
// MP4 at OutputKey. The clip's start and end are read from the database when
// the job runs. Captions burns the asset's subtitles into the picture.
type ExportClipInput struct {
	AssetID    string `json:"asset_id"`
	ClipID     string `json:"clip_id"`
	StorageKey string `json:"storage_key"`
	Format     string `json:"format"`
	Captions   bool   `json:"captions"`
	OutputKey  string `json:"output_key"`
}

// Pipeline is the order in which an upload's jobs run. The API creates one
// job row per entry and enqueues the first; when a job ends (done or failed),
// the worker that ran it enqueues the next one. Transcription and analysis
// come after HLS so the video is playable before the slower AI steps
// (docs/adr/0017).
var Pipeline = []string{
	JobTypeExtractMetadata,
	JobTypeGenerateThumbnail,
	JobTypeTranscodeHLS,
	JobTypeExtractAudio,
	JobTypeTranscribeAudio,
	JobTypeAnalyzeTranscript,
	JobTypeSuggestClips,
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
