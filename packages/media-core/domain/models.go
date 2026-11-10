package domain

import "time"

type Source struct {
	ID         string
	AssetID    string
	StorageKey string
	MimeType   string
	SizeBytes  int64
	Checksum   string
}

type Job struct {
	ID           string
	AssetID      string
	Type         string
	InputJSON    []byte
	Status       string
	Progress     int
	ErrorMessage *string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

type JobLog struct {
	ID        string
	JobID     string
	Level     string
	Message   string
	Timestamp time.Time
}

type Artifact struct {
	ID           string
	AssetID      string
	JobID        string
	Type         string
	StorageKey   string
	MetadataJSON []byte
	CreatedAt    time.Time
}

type TechnicalMetadata struct {
	ID           string
	AssetID      string
	MetadataJSON []byte
	ExtractedAt  time.Time
}

// Job statuses, matching the CHECK constraint on processing_jobs.status.
const (
	JobPending = "pending"
	JobRunning = "running"
	JobDone    = "done"
	JobFailed  = "failed"
)

// Artifact types (artifacts.type).
const (
	ArtifactHLS       = "hls"       // storage_key points to the HLS playlist (index.m3u8)
	ArtifactThumbnail = "thumbnail" // storage_key points to a JPEG image
	ArtifactAudio     = "audio"     // 16 kHz mono FLAC for transcription
	ArtifactSubtitles = "subtitles" // WebVTT; metadata has the language
	ArtifactClip      = "clip"      // exported clip MP4; metadata has clipId and format
)

// Job log levels (job_logs.level).
const (
	LogInfo    = "info"
	LogWarning = "warning"
	LogError   = "error"
)

// Transcript is the speech found in an asset, written by the AI worker.
type Transcript struct {
	ID        string
	AssetID   string
	JobID     string
	Language  string // ISO 639-1 code, detected or requested
	Text      string
	Segments  []byte // JSON: [{"start": 0.0, "end": 2.5, "text": "..."}]
	Words     []byte // JSON: [{"start": 0.1, "end": 0.4, "word": "Hello"}]; [] for older transcripts
	Model     string // Whisper model used, e.g. "base"
	CreatedAt time.Time
}

// Translation is an asset's transcript in another language, written by a
// language model for captions in that language (docs/adr/0029).
type Translation struct {
	ID             string
	AssetID        string
	JobID          string
	Language       string // ISO 639-1 code of the translation
	SourceLanguage string // the transcript's language
	Segments       []byte // JSON: the transcript's segments, same times, translated text
	Words          []byte // JSON: [{"start", "end", "word"}], the translated words placed on the speech
	Model          string // model that answered, as reported by the gateway
	CreatedAt      time.Time
}

// Analysis is what a language model wrote about an asset's transcript,
// saved by the AI worker (docs/adr/0019).
type Analysis struct {
	ID        string
	AssetID   string
	JobID     string
	Language  string // language of the summary, tags and titles (the transcript's)
	Summary   string
	Tags      []string
	Chapters  []byte // JSON: [{"start": 0.0, "title": "..."}], sorted by start
	Model     string // model that answered, as reported by the gateway
	CreatedAt time.Time
}

// Clip is a moment of an asset that works as a short standalone video,
// suggested by a language model from the transcript (docs/adr/0020) or made
// by someone from a quote in it (docs/adr/0039).
type Clip struct {
	ID        string
	AssetID   string
	JobID     string  // the suggest-clips job; empty for a clip made from a quote
	Origin    string  // ClipSuggested or ClipQuote
	Start     float64 // seconds
	End       float64
	Title     string
	Reason    string // why the moment works on its own
	Score     int    // 1 to 10, from the model; 0 for a clip made from a quote
	Rank      int    // 1 is the best
	Model     string
	CreatedAt time.Time
	// FaceTrack is where to centre a vertical crop over time, from the
	// track-faces job; nil until it has run (see migration 0013).
	FaceTrack []byte
	// EditedAt is when someone last changed the clip by hand; nil for a
	// clip as suggested (see migration 0014).
	EditedAt *time.Time
}

// Where a clip came from (see migration 0021).
const (
	ClipSuggested = "suggested" // picked by the language model
	ClipQuote     = "quote"     // made by someone from a quote in the transcript
)

// Limits of a clip edited by hand, in seconds.
const (
	MinClipSeconds = 3.0
	MaxClipSeconds = 180.0
	// MaxClipTitle is the longest title, in characters.
	MaxClipTitle = 120
)

// Vignette is an opening or closing in the library, an uploaded MP4 joined
// to exported clips by a Preset (docs/adr/0030).
type Vignette struct {
	ID         string
	Name       string
	Filename   string // as uploaded
	StorageKey string
	SizeBytes  int64
	CreatedAt  time.Time
}

// Preset is a named set of export choices: the captions' look and language,
// and the opening and closing, each in a horizontal and a vertical version
// (docs/adr/0030).
type Preset struct {
	ID              string
	Name            string
	Captions        bool
	CaptionStyle    string
	CaptionPosition string
	CaptionColor    string // empty for the style's own
	CaptionCaps     *bool  // nil for the style's own
	CaptionLanguage string // empty for the transcript's own
	// Vignette IDs; empty for none.
	IntroHorizontal, IntroVertical string
	OutroHorizontal, OutroVertical string
	// The watermark image's ID (empty for none), its corner, its width as a
	// percentage of the clip's shorter side, and its opacity in percent
	// (docs/adr/0031).
	Watermark         string
	WatermarkPosition string
	WatermarkSize     int
	WatermarkOpacity  int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Watermark is an image in the library, laid over exported clips by a
// Preset (docs/adr/0031).
type Watermark struct {
	ID            string
	Name          string
	Filename      string // as uploaded
	StorageKey    string
	SizeBytes     int64
	Width, Height int
	CreatedAt     time.Time
}
