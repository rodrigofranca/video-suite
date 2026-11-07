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
// suggested by a language model from the transcript (docs/adr/0020).
type Clip struct {
	ID        string
	AssetID   string
	JobID     string
	Start     float64 // seconds
	End       float64
	Title     string
	Reason    string // why the moment works on its own
	Score     int    // 1 to 10, from the model
	Rank      int    // 1 is the best
	Model     string
	CreatedAt time.Time
	// FaceTrack is where to centre a vertical crop over time, from the
	// track-faces job; nil until it has run (see migration 0013).
	FaceTrack []byte
}
