package jobcontracts

// Job types. The Go media worker runs the FFmpeg jobs (metadata, thumbnail,
// HLS, audio and clip export); the Python AI worker (apps/ai-worker) runs
// transcribe-audio, analyze-transcript, suggest-clips, track-faces and
// translate-transcript. See StreamFor.
const (
	JobTypeExtractMetadata   = "extract-metadata"
	JobTypeGenerateThumbnail = "generate-thumbnail"
	JobTypeTranscodeHLS      = "transcode-hls"
	JobTypeExtractAudio      = "extract-audio"
	JobTypeTranscribeAudio   = "transcribe-audio"
	JobTypeAnalyzeTranscript = "analyze-transcript"
	JobTypeSuggestClips      = "suggest-clips"
	JobTypeTrackFaces        = "track-faces"
	// JobTypeExportClip runs on request, not in the Pipeline.
	JobTypeExportClip = "export-clip"
	// JobTypeTranslateTranscript runs before an export with captions in a
	// language the transcript has no translation into yet (docs/adr/0029).
	JobTypeTranslateTranscript = "translate-transcript"
)

// Clip export formats (ExportClipInput.Format).
const (
	// ClipFormatOriginal keeps the video's shape, at most 1920 px on the long side.
	ClipFormatOriginal = "original"
	// ClipFormatVertical crops the middle of the frame to 9:16 (1080x1920).
	ClipFormatVertical = "vertical"
	// ClipFormatVerticalFace crops 9:16 around the main face, following it
	// (the clip's face track); the middle when there is no face.
	ClipFormatVerticalFace = "vertical-face"
	// ClipFormatVerticalBlur fits the whole frame into 9:16 over a blurred copy.
	ClipFormatVerticalBlur = "vertical-blur"
)

// ClipFormats lists the export formats, in the order the dashboard shows them.
var ClipFormats = []string{ClipFormatOriginal, ClipFormatVertical, ClipFormatVerticalFace, ClipFormatVerticalBlur}

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

// Where burned-in captions sit (ExportClipInput.CaptionPosition; empty means
// the bottom), and the colors of the highlighted word, the box and the
// karaoke fill (ExportClipInput.CaptionColor; empty means the style's own:
// yellow, or violet for the box) (docs/adr/0025).
var (
	CaptionPositions = []string{"bottom", "middle", "top"}
	CaptionColors    = []string{"yellow", "green", "pink", "blue", "violet"}
)

// CaptionLanguages are the languages captions can be translated into
// (ExportClipInput.CaptionLanguage, TranslateTranscriptInput.Language), as
// ISO 639-1 codes. All are written in the Latin alphabet, which the caption
// font covers (docs/adr/0029).
var CaptionLanguages = []string{"en", "es", "pt", "fr", "de", "it"}

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

// TrackFacesInput asks the AI worker to follow the main face through each
// of the asset's clips, for vertical exports that keep it in frame
// (docs/adr/0022). It reads the source through a signed URL. With ClipID,
// only that clip is tracked, after it was edited (docs/adr/0024).
type TrackFacesInput struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
	ClipID     string `json:"clip_id,omitempty"`
}

// ExportClipInput asks the media worker to cut a clip out of the source as an
// MP4 at OutputKey. The clip's start and end are read from the database when
// the job runs. Captions burns the asset's speech into the picture, in
// CaptionStyle (empty means CaptionStyleHighlight), at CaptionPosition, with
// CaptionColor, in capitals when CaptionCaps is true (nil means only the
// one-word style), and in CaptionLanguage: empty means the transcript's own,
// another language its translation (docs/adr/0029).
type ExportClipInput struct {
	AssetID         string `json:"asset_id"`
	ClipID          string `json:"clip_id"`
	StorageKey      string `json:"storage_key"`
	Format          string `json:"format"`
	Captions        bool   `json:"captions"`
	CaptionStyle    string `json:"caption_style,omitempty"`
	CaptionPosition string `json:"caption_position,omitempty"`
	CaptionColor    string `json:"caption_color,omitempty"`
	CaptionCaps     *bool  `json:"caption_caps,omitempty"`
	CaptionLanguage string `json:"caption_language,omitempty"`
	OutputKey       string `json:"output_key"`
}

// TranslateTranscriptInput asks the AI worker to translate the asset's
// transcript into Language, one of CaptionLanguages, for captions in that
// language (docs/adr/0029). The translation replaces an earlier one into the
// same language; it is read from the database, not from the message.
type TranslateTranscriptInput struct {
	AssetID  string `json:"asset_id"`
	Language string `json:"language"`
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
	JobTypeTrackFaces,
}

// RerunWith lists, for a job run again by hand, the jobs that run again
// after it because they work on its result: new clip suggestions need their
// faces tracked again (docs/adr/0023). They follow it in Pipeline order.
var RerunWith = map[string][]string{
	JobTypeSuggestClips: {JobTypeTrackFaces},
}

// After names, for a job outside the Pipeline, the job that may wait for
// it: an export with translated captions waits for the translation
// (docs/adr/0029). The worker queues it only when one is pending.
var After = map[string]string{
	JobTypeTranslateTranscript: JobTypeExportClip,
}

// NextJobType returns the job type that follows jobType in the Pipeline, or
// in After.
func NextJobType(jobType string) (string, bool) {
	for i, t := range Pipeline {
		if t == jobType && i+1 < len(Pipeline) {
			return Pipeline[i+1], true
		}
	}
	next, ok := After[jobType]
	return next, ok
}
