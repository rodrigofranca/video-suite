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

// Burned-in caption styles (ExportClipInput.CaptionStyle). All but
// CaptionStyleLines follow the transcript's word timings; without them the
// export falls back to CaptionStyleLines (docs/adr/0021).
const (
	// CaptionStyleHighlight shows a short phrase with the word being said in yellow.
	CaptionStyleHighlight = "highlight"
	// CaptionStyleBox shows a short phrase with a box behind the word being said.
	CaptionStyleBox = "box"
	// CaptionStyleKaraoke shows a short phrase whose words fill with color as they are said.
	CaptionStyleKaraoke = "karaoke"
	// CaptionStyleOneWord shows only the word being said, big and in capitals.
	CaptionStyleOneWord = "one-word"
	// CaptionStyleLines shows the subtitle lines, as in the player.
	CaptionStyleLines = "lines"
)

// CaptionStyles lists the caption styles, in the order the dashboard shows
// them; the first is the default.
var CaptionStyles = []string{CaptionStyleHighlight, CaptionStyleBox, CaptionStyleKaraoke, CaptionStyleOneWord, CaptionStyleLines}

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
// the job runs. Captions burns the asset's speech into the picture, in
// CaptionStyle (empty means CaptionStyleHighlight).
type ExportClipInput struct {
	AssetID      string `json:"asset_id"`
	ClipID       string `json:"clip_id"`
	StorageKey   string `json:"storage_key"`
	Format       string `json:"format"`
	Captions     bool   `json:"captions"`
	CaptionStyle string `json:"caption_style,omitempty"`
	OutputKey    string `json:"output_key"`
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
