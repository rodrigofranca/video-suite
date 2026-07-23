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
