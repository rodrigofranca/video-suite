// Package domain holds the core media entities shared by the API and the worker.
// Terms follow CONTEXT.md.
package domain

import "time"

type Asset struct {
	ID        string
	Filename  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Asset statuses, matching the CHECK constraint on media_assets.status.
// An asset is ready once its HLS rendition plays (docs/adr/0003).
const (
	AssetUploading  = "uploading"
	AssetProcessing = "processing"
	AssetReady      = "ready"
	AssetFailed     = "failed"
)
