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
	ArtifactHLS = "hls" // storage_key points to the HLS playlist (index.m3u8)
)

// Job log levels (job_logs.level).
const (
	LogInfo    = "info"
	LogWarning = "warning"
	LogError   = "error"
)
