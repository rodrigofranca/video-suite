// Package store holds the SQL used by the API and the worker. PostgreSQL is
// the source of truth for assets, jobs, logs and artifacts.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ultramedia/media-core/domain"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// ---- Assets ----

const assetColumns = `id, filename, status, created_at, updated_at`

func scanAsset(row pgx.Row) (domain.Asset, error) {
	var a domain.Asset
	err := row.Scan(&a.ID, &a.Filename, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	return a, notFound(err)
}

// CreateAsset inserts a new asset (normally with status uploading).
func (s *Store) CreateAsset(ctx context.Context, a domain.Asset) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO media_assets (id, filename, status) VALUES ($1, $2, $3)`,
		a.ID, a.Filename, a.Status)
	return err
}

// DeleteAsset removes an asset and, by cascade, its source, jobs and logs.
func (s *Store) DeleteAsset(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM media_assets WHERE id = $1`, id)
	return err
}

func (s *Store) GetAsset(ctx context.Context, id string) (domain.Asset, error) {
	return scanAsset(s.db.QueryRow(ctx, `SELECT `+assetColumns+` FROM media_assets WHERE id = $1`, id))
}

// AssetListItem is an asset with what the video list shows next to it.
type AssetListItem struct {
	domain.Asset
	ThumbnailKey string          // empty until a thumbnail exists
	Summary      json.RawMessage // metadata summary; nil until extracted
}

// ListAssets returns assets newest first, optionally filtered by status,
// with their latest thumbnail and metadata summary.
func (s *Store) ListAssets(ctx context.Context, status string) ([]AssetListItem, error) {
	rows, err := s.db.Query(ctx,
		`SELECT a.id, a.filename, a.status, a.created_at, a.updated_at,
		        th.storage_key, tm.metadata_json->'summary'
		 FROM media_assets a
		 LEFT JOIN LATERAL (
		     SELECT storage_key FROM artifacts
		     WHERE asset_id = a.id AND type = $2
		     ORDER BY created_at DESC LIMIT 1
		 ) th ON true
		 LEFT JOIN technical_metadata tm ON tm.asset_id = a.id
		 WHERE $1 = '' OR a.status = $1
		 ORDER BY a.created_at DESC LIMIT 200`, status, domain.ArtifactThumbnail)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (AssetListItem, error) {
		var it AssetListItem
		var thumb *string
		var summary []byte
		err := r.Scan(&it.ID, &it.Filename, &it.Status, &it.CreatedAt, &it.UpdatedAt, &thumb, &summary)
		if thumb != nil {
			it.ThumbnailKey = *thumb
		}
		if summary != nil {
			it.Summary = summary
		}
		return it, err
	})
}

func (s *Store) SetAssetStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE media_assets SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

// CompleteUpload records the stored source file and the pipeline jobs, and
// moves the asset to processing, all in one transaction.
func (s *Store) CompleteUpload(ctx context.Context, src domain.Source, jobs []domain.Job) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO media_sources (id, asset_id, storage_key, mime_type, size_bytes, checksum)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			src.ID, src.AssetID, src.StorageKey, src.MimeType, src.SizeBytes, src.Checksum)
		if err != nil {
			return fmt.Errorf("insert source: %w", err)
		}
		for _, j := range jobs {
			_, err := tx.Exec(ctx,
				`INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				j.ID, j.AssetID, j.Type, j.InputJSON, j.Status, j.CreatedAt)
			if err != nil {
				return fmt.Errorf("insert job %s: %w", j.Type, err)
			}
		}
		_, err = tx.Exec(ctx,
			`UPDATE media_assets SET status = $2, updated_at = NOW() WHERE id = $1`,
			src.AssetID, domain.AssetProcessing)
		return err
	})
}

func (s *Store) GetSource(ctx context.Context, assetID string) (domain.Source, error) {
	var src domain.Source
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, storage_key, mime_type, size_bytes, checksum
		 FROM media_sources WHERE asset_id = $1 LIMIT 1`, assetID).
		Scan(&src.ID, &src.AssetID, &src.StorageKey, &src.MimeType, &src.SizeBytes, &src.Checksum)
	return src, notFound(err)
}

// ---- Jobs ----

const jobColumns = `id, asset_id, type, input_json, status, progress, error_message, created_at, started_at, completed_at`

func scanJob(row pgx.Row) (domain.Job, error) {
	var j domain.Job
	err := row.Scan(&j.ID, &j.AssetID, &j.Type, &j.InputJSON, &j.Status, &j.Progress,
		&j.ErrorMessage, &j.CreatedAt, &j.StartedAt, &j.CompletedAt)
	return j, notFound(err)
}

func (s *Store) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return scanJob(s.db.QueryRow(ctx, `SELECT `+jobColumns+` FROM processing_jobs WHERE id = $1`, id))
}

// ListJobs returns an asset's jobs in pipeline order (creation order).
func (s *Store) ListJobs(ctx context.Context, assetID string) ([]domain.Job, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+jobColumns+` FROM processing_jobs WHERE asset_id = $1 ORDER BY created_at, type`, assetID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Job, error) { return scanJob(r) })
}

// StartJob marks a job as running. It returns false when the job already
// finished (done or failed), e.g. when a message is delivered twice. A job
// left running by a crash can be started again.
func (s *Store) StartJob(ctx context.Context, id string) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE processing_jobs
		 SET status = $2, started_at = NOW(), completed_at = NULL, error_message = NULL, progress = 0
		 WHERE id = $1 AND status IN ($3, $2)`,
		id, domain.JobRunning, domain.JobPending)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// FinishJob marks a job done, or failed with jobErr's message.
func (s *Store) FinishJob(ctx context.Context, id string, jobErr error) error {
	status, progress := domain.JobDone, new(int)
	*progress = 100
	var msg *string
	if jobErr != nil {
		status, progress = domain.JobFailed, nil // keep the progress reached
		m := jobErr.Error()
		msg = &m
	}
	_, err := s.db.Exec(ctx,
		`UPDATE processing_jobs
		 SET status = $2, progress = COALESCE($3, progress), error_message = $4, completed_at = NOW()
		 WHERE id = $1`,
		id, status, progress, msg)
	return err
}

// ErrBusy is returned when an asset still has jobs pending or running.
var ErrBusy = errors.New("the asset still has jobs pending or running")

// CreateJobIfIdle adds a pending job for an asset whose jobs have all
// finished. When assetStatus is not empty the asset moves to that status in
// the same transaction. It returns ErrNotFound for an unknown asset and
// ErrBusy while any job of the asset is pending or running. The asset row is
// locked, so concurrent calls cannot both succeed.
func (s *Store) CreateJobIfIdle(ctx context.Context, job domain.Job, assetStatus string) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var current string
		err := tx.QueryRow(ctx, `SELECT status FROM media_assets WHERE id = $1 FOR UPDATE`, job.AssetID).Scan(&current)
		if err != nil {
			return notFound(err)
		}
		var busy bool
		err = tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM processing_jobs WHERE asset_id = $1 AND status IN ($2, $3))`,
			job.AssetID, domain.JobPending, domain.JobRunning).Scan(&busy)
		if err != nil {
			return err
		}
		if busy || current == domain.AssetUploading {
			return ErrBusy
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			job.ID, job.AssetID, job.Type, job.InputJSON, domain.JobPending, job.CreatedAt)
		if err != nil || assetStatus == "" {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE media_assets SET status = $2, updated_at = NOW() WHERE id = $1`, job.AssetID, assetStatus)
		return err
	})
}

// SetJobProgress records how far a running job is, from 0 to 100.
func (s *Store) SetJobProgress(ctx context.Context, id string, progress int) error {
	progress = max(0, min(progress, 100))
	_, err := s.db.Exec(ctx,
		`UPDATE processing_jobs SET progress = $2 WHERE id = $1 AND status = $3`,
		id, progress, domain.JobRunning)
	return err
}

// NextPendingJob returns the asset's pending job of the given type.
func (s *Store) NextPendingJob(ctx context.Context, assetID, jobType string) (domain.Job, error) {
	return scanJob(s.db.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM processing_jobs
		 WHERE asset_id = $1 AND type = $2 AND status = $3
		 ORDER BY created_at LIMIT 1`,
		assetID, jobType, domain.JobPending))
}

// ---- Job logs ----

func (s *Store) AddJobLog(ctx context.Context, jobID, level, message string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO job_logs (id, job_id, level, message, timestamp) VALUES ($1, $2, $3, $4, $5)`,
		domain.NewID(), jobID, level, message, time.Now())
	return err
}

func (s *Store) ListJobLogs(ctx context.Context, jobID string) ([]domain.JobLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, job_id, level, message, timestamp FROM job_logs
		 WHERE job_id = $1 ORDER BY timestamp, id LIMIT 1000`, jobID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.JobLog, error) {
		var l domain.JobLog
		err := r.Scan(&l.ID, &l.JobID, &l.Level, &l.Message, &l.Timestamp)
		return l, err
	})
}

// ---- Technical metadata ----

// SaveTechnicalMetadata stores an asset's metadata, replacing any earlier
// result. metadataJSON holds {"summary": {...}, "ffprobe": {...}}.
func (s *Store) SaveTechnicalMetadata(ctx context.Context, assetID string, metadataJSON []byte) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO technical_metadata (id, asset_id, metadata_json, extracted_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (asset_id) DO UPDATE
		 SET metadata_json = EXCLUDED.metadata_json, extracted_at = EXCLUDED.extracted_at`,
		domain.NewID(), assetID, metadataJSON)
	return err
}

// GetTechnicalMetadata returns the stored metadata JSON of an asset.
func (s *Store) GetTechnicalMetadata(ctx context.Context, assetID string) (domain.TechnicalMetadata, error) {
	var m domain.TechnicalMetadata
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, metadata_json, extracted_at FROM technical_metadata WHERE asset_id = $1`, assetID).
		Scan(&m.ID, &m.AssetID, &m.MetadataJSON, &m.ExtractedAt)
	return m, notFound(err)
}

// ---- Artifacts ----

func (s *Store) CreateArtifact(ctx context.Context, a domain.Artifact) error {
	meta := a.MetadataJSON
	if meta == nil {
		meta = []byte("{}")
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO artifacts (id, asset_id, job_id, type, storage_key, metadata_json)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.AssetID, a.JobID, a.Type, a.StorageKey, meta)
	return err
}

func (s *Store) ListArtifacts(ctx context.Context, assetID string) ([]domain.Artifact, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, asset_id, job_id, type, storage_key, metadata_json, created_at
		 FROM artifacts WHERE asset_id = $1 ORDER BY created_at`, assetID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Artifact, error) {
		var a domain.Artifact
		err := r.Scan(&a.ID, &a.AssetID, &a.JobID, &a.Type, &a.StorageKey, &a.MetadataJSON, &a.CreatedAt)
		return a, err
	})
}

// LatestArtifact returns the newest artifact of a type for an asset.
func (s *Store) LatestArtifact(ctx context.Context, assetID, artifactType string) (domain.Artifact, error) {
	var a domain.Artifact
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, job_id, type, storage_key, metadata_json, created_at
		 FROM artifacts WHERE asset_id = $1 AND type = $2
		 ORDER BY created_at DESC LIMIT 1`, assetID, artifactType).
		Scan(&a.ID, &a.AssetID, &a.JobID, &a.Type, &a.StorageKey, &a.MetadataJSON, &a.CreatedAt)
	return a, notFound(err)
}
