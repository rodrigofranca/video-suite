// Package store holds the SQL used by the API and the worker. PostgreSQL is
// the source of truth for assets, jobs, logs and artifacts.
package store

import (
	"context"
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

// ListAssets returns assets newest first, optionally filtered by status.
func (s *Store) ListAssets(ctx context.Context, status string) ([]domain.Asset, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+assetColumns+` FROM media_assets
		 WHERE $1 = '' OR status = $1
		 ORDER BY created_at DESC LIMIT 200`, status)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Asset, error) { return scanAsset(r) })
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
