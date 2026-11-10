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

	jobcontracts "ultramedia/job-contracts"
)

// ErrNotFound is returned when a row does not exist (domain.ErrNotFound).
var ErrNotFound = domain.ErrNotFound

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

// abandonedUpload is how long an asset can stay "uploading" before it counts
// as an upload that stopped (the API restarted mid-upload) and can be deleted.
const abandonedUpload = time.Hour

// DeleteAssetIfIdle removes an asset and, by cascade, everything recorded
// about it, while none of its jobs is pending or running and it is not being
// uploaded (docs/adr/0027). It returns ErrNotFound for an unknown asset and
// ErrBusy otherwise. The caller removes the stored files.
func (s *Store) DeleteAssetIfIdle(ctx context.Context, id string) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var status string
		var updated time.Time
		err := tx.QueryRow(ctx, `SELECT status, updated_at FROM media_assets WHERE id = $1 FOR UPDATE`, id).Scan(&status, &updated)
		if err != nil {
			return notFound(err)
		}
		if status == domain.AssetUploading && time.Since(updated) < abandonedUpload {
			return ErrBusy
		}
		if err := checkIdle(ctx, tx, id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM media_assets WHERE id = $1`, id)
		return err
	})
}

// DeleteClipIfIdle removes a clip and the records of its exported files,
// while none of its asset's jobs is pending or running. The export jobs stay
// in the history. It returns ErrNotFound for a clip that is not the asset's.
func (s *Store) DeleteClipIfIdle(ctx context.Context, assetID, clipID string) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM media_assets WHERE id = $1 FOR UPDATE`, assetID); err != nil {
			return err
		}
		if err := checkIdle(ctx, tx, assetID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM clips WHERE id = $1 AND asset_id = $2`, clipID, assetID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_, err = tx.Exec(ctx,
			`DELETE FROM artifacts WHERE asset_id = $1 AND type = $2 AND metadata_json->>'clipId' = $3`,
			assetID, domain.ArtifactClip, clipID)
		return err
	})
}

// checkIdle returns ErrBusy while any job of the asset is pending or running.
func checkIdle(ctx context.Context, tx pgx.Tx, assetID string) error {
	var busy bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM processing_jobs WHERE asset_id = $1 AND status IN ($2, $3))`,
		assetID, domain.JobPending, domain.JobRunning).Scan(&busy)
	if err != nil {
		return err
	}
	if busy {
		return ErrBusy
	}
	return nil
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

// CompleteUpload records the stored source file and the pipeline jobs,
// queues the first job and moves the asset to processing, all in one
// transaction. The worker that runs a job queues the next one when it ends
// (the job.finished event).
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
		if len(jobs) > 0 {
			if err := queueJob(ctx, tx, jobs[0]); err != nil {
				return err
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

// FinishJob marks a job done, or failed with jobErr's message, and records
// the job.finished event that moves the pipeline on, in one transaction. An
// unknown job is left alone.
func (s *Store) FinishJob(ctx context.Context, id string, jobErr error) error {
	status, progress := domain.JobDone, new(int)
	*progress = 100
	var msg *string
	if jobErr != nil {
		status, progress = domain.JobFailed, nil // keep the progress reached
		m := jobErr.Error()
		msg = &m
	}
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var assetID string
		err := tx.QueryRow(ctx,
			`UPDATE processing_jobs
			 SET status = $2, progress = COALESCE($3, progress), error_message = $4, completed_at = NOW()
			 WHERE id = $1 RETURNING asset_id`,
			id, status, progress, msg).Scan(&assetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return recordEvent(ctx, tx, jobcontracts.EventJobFinished, assetID, id)
	})
}

// ErrBusy is returned when an asset still has jobs pending or running.
var ErrBusy = domain.ErrBusy

// CreateJobIfIdle adds a pending job for an asset whose jobs have all
// finished and queues it, with the pending jobs in then, which the pipeline
// queues after it.
// When assetStatus is not empty the asset moves to that status in the same
// transaction. It returns ErrNotFound for an unknown asset and ErrBusy while
// any job of the asset is pending or running. The asset row is locked, so
// concurrent calls cannot both succeed.
func (s *Store) CreateJobIfIdle(ctx context.Context, job domain.Job, assetStatus string, then ...domain.Job) error {
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
		for _, j := range append([]domain.Job{job}, then...) {
			_, err = tx.Exec(ctx,
				`INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				j.ID, job.AssetID, j.Type, j.InputJSON, domain.JobPending, j.CreatedAt)
			if err != nil {
				return err
			}
		}
		if err := queueJob(ctx, tx, job); err != nil {
			return err
		}
		if assetStatus == "" {
			return nil
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

const nextPendingJob = `SELECT ` + jobColumns + ` FROM processing_jobs
	WHERE asset_id = $1 AND type = $2 AND status = $3 ORDER BY created_at LIMIT 1`

// NextPendingJob returns the asset's oldest pending job of a type.
func (s *Store) NextPendingJob(ctx context.Context, assetID, jobType string) (domain.Job, error) {
	return scanJob(s.db.QueryRow(ctx, nextPendingJob, assetID, jobType, domain.JobPending))
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

// ---- Transcripts ----

// SaveTranscript stores an asset's transcript, replacing any earlier one
// and removing the translations of the earlier one.
func (s *Store) SaveTranscript(ctx context.Context, t domain.Transcript) error {
	segments, words := t.Segments, t.Words
	if segments == nil {
		segments = []byte("[]")
	}
	if words == nil {
		words = []byte("[]")
	}
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO transcripts (id, asset_id, job_id, language, text, segments, words, model)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (asset_id) DO UPDATE
			 SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, language = EXCLUDED.language,
			     text = EXCLUDED.text, segments = EXCLUDED.segments, words = EXCLUDED.words,
			     model = EXCLUDED.model, created_at = NOW()`,
			t.ID, t.AssetID, t.JobID, t.Language, t.Text, segments, words, t.Model)
		if err != nil {
			return err
		}
		// Translations of the earlier transcript no longer match the speech.
		_, err = tx.Exec(ctx, `DELETE FROM transcript_translations WHERE asset_id = $1`, t.AssetID)
		return err
	})
}

// GetTranscript returns an asset's transcript.
func (s *Store) GetTranscript(ctx context.Context, assetID string) (domain.Transcript, error) {
	var t domain.Transcript
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, job_id, language, text, segments, words, model, created_at
		 FROM transcripts WHERE asset_id = $1`, assetID).
		Scan(&t.ID, &t.AssetID, &t.JobID, &t.Language, &t.Text, &t.Segments, &t.Words, &t.Model, &t.CreatedAt)
	return t, notFound(err)
}

// ---- Translations ----

// SaveTranslation stores a translation of an asset's transcript, replacing
// an earlier one into the same language.
func (s *Store) SaveTranslation(ctx context.Context, t domain.Translation) error {
	segments, words := t.Segments, t.Words
	if segments == nil {
		segments = []byte("[]")
	}
	if words == nil {
		words = []byte("[]")
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO transcript_translations (id, asset_id, job_id, language, source_language, segments, words, model)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (asset_id, language) DO UPDATE
		 SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, source_language = EXCLUDED.source_language,
		     segments = EXCLUDED.segments, words = EXCLUDED.words, model = EXCLUDED.model, created_at = NOW()`,
		t.ID, t.AssetID, t.JobID, t.Language, t.SourceLanguage, segments, words, t.Model)
	return err
}

// GetTranslation returns the translation of an asset's transcript into a language.
func (s *Store) GetTranslation(ctx context.Context, assetID, language string) (domain.Translation, error) {
	var t domain.Translation
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, job_id, language, source_language, segments, words, model, created_at
		 FROM transcript_translations WHERE asset_id = $1 AND language = $2`, assetID, language).
		Scan(&t.ID, &t.AssetID, &t.JobID, &t.Language, &t.SourceLanguage, &t.Segments, &t.Words, &t.Model, &t.CreatedAt)
	return t, notFound(err)
}

// TranslationLanguages lists the languages an asset's transcript has been
// translated into, in alphabetical order.
func (s *Store) TranslationLanguages(ctx context.Context, assetID string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT language FROM transcript_translations WHERE asset_id = $1 ORDER BY language`, assetID)
	if err != nil {
		return nil, err
	}
	languages, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if languages == nil {
		languages = []string{}
	}
	return languages, err
}

// ---- Analyses ----

// SaveAnalysis stores an asset's analysis, replacing any earlier one.
func (s *Store) SaveAnalysis(ctx context.Context, a domain.Analysis) error {
	tags, chapters := a.Tags, a.Chapters
	if tags == nil {
		tags = []string{}
	}
	if chapters == nil {
		chapters = []byte("[]")
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO analyses (id, asset_id, job_id, language, summary, tags, chapters, model)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (asset_id) DO UPDATE
		 SET id = EXCLUDED.id, job_id = EXCLUDED.job_id, language = EXCLUDED.language,
		     summary = EXCLUDED.summary, tags = EXCLUDED.tags, chapters = EXCLUDED.chapters,
		     model = EXCLUDED.model, created_at = NOW()`,
		a.ID, a.AssetID, a.JobID, a.Language, a.Summary, tags, chapters, a.Model)
	return err
}

// GetAnalysis returns an asset's analysis.
func (s *Store) GetAnalysis(ctx context.Context, assetID string) (domain.Analysis, error) {
	var a domain.Analysis
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_id, job_id, language, summary, tags, chapters, model, created_at
		 FROM analyses WHERE asset_id = $1`, assetID).
		Scan(&a.ID, &a.AssetID, &a.JobID, &a.Language, &a.Summary, &a.Tags, &a.Chapters, &a.Model, &a.CreatedAt)
	return a, notFound(err)
}

// ---- Clips ----

const clipColumns = `id, asset_id, job_id, origin, start_seconds, end_seconds, title, reason, score, rank, model, created_at, face_track, edited_at`

func scanClip(row pgx.Row) (domain.Clip, error) {
	var c domain.Clip
	var jobID *string
	var score *int
	err := row.Scan(&c.ID, &c.AssetID, &jobID, &c.Origin, &c.Start, &c.End, &c.Title, &c.Reason, &score, &c.Rank, &c.Model,
		&c.CreatedAt, &c.FaceTrack, &c.EditedAt)
	if jobID != nil {
		c.JobID = *jobID
	}
	if score != nil {
		c.Score = *score
	}
	return c, notFound(err)
}

// ReplaceClips stores an asset's suggested clips, removing the earlier
// suggestions but keeping the clips made or edited by hand, as the
// suggest-clips job does (docs/adr/0026, 0039; the job also drops new clips
// that repeat a kept one and ranks them all, which this method leaves to the
// caller).
func (s *Store) ReplaceClips(ctx context.Context, assetID string, clips []domain.Clip) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM clips WHERE asset_id = $1 AND edited_at IS NULL AND origin = $2`, assetID, domain.ClipSuggested)
		if err != nil {
			return err
		}
		for _, c := range clips {
			_, err := tx.Exec(ctx,
				`INSERT INTO clips (id, asset_id, job_id, start_seconds, end_seconds, title, reason, score, rank, model)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				c.ID, assetID, c.JobID, c.Start, c.End, c.Title, c.Reason, c.Score, c.Rank, c.Model)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ListClips returns an asset's clips, best first.
func (s *Store) ListClips(ctx context.Context, assetID string) ([]domain.Clip, error) {
	rows, err := s.db.Query(ctx, `SELECT `+clipColumns+` FROM clips WHERE asset_id = $1 ORDER BY rank, start_seconds`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	clips := []domain.Clip{}
	for rows.Next() {
		c, err := scanClip(rows)
		if err != nil {
			return nil, err
		}
		clips = append(clips, c)
	}
	return clips, rows.Err()
}

// GetClip returns one clip.
func (s *Store) GetClip(ctx context.Context, id string) (domain.Clip, error) {
	return scanClip(s.db.QueryRow(ctx, `SELECT `+clipColumns+` FROM clips WHERE id = $1`, id))
}

// CreateClipIfIdle adds a clip made from a quote (docs/adr/0039) at the top
// of the asset's clips, while none of its jobs is pending or running, and
// adds track as a pending job and queues it, in the same transaction, to
// find the faces through the clip. It returns ErrNotFound for an unknown
// asset and ErrBusy while the asset is busy.
func (s *Store) CreateClipIfIdle(ctx context.Context, c domain.Clip, track domain.Job) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockIdleAsset(ctx, tx, c.AssetID); err != nil {
			return err
		}
		// The newest clip made by hand comes first.
		if _, err := tx.Exec(ctx, `UPDATE clips SET rank = rank + 1 WHERE asset_id = $1`, c.AssetID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO clips (id, asset_id, origin, start_seconds, end_seconds, title, reason, rank, model, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '', $8)`,
			c.ID, c.AssetID, domain.ClipQuote, c.Start, c.End, c.Title, c.Reason, c.CreatedAt)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			track.ID, c.AssetID, track.Type, track.InputJSON, domain.JobPending, track.CreatedAt)
		if err != nil {
			return err
		}
		track.AssetID = c.AssetID
		return queueJob(ctx, tx, track)
	})
}

// lockIdleAsset locks an asset's row, as CreateJobIfIdle does, so no job
// starts until the transaction ends, and returns ErrBusy while one of its
// jobs is pending or running (ErrNotFound for an unknown asset).
func lockIdleAsset(ctx context.Context, tx pgx.Tx, assetID string) error {
	var id string
	if err := tx.QueryRow(ctx, `SELECT id FROM media_assets WHERE id = $1 FOR UPDATE`, assetID).Scan(&id); err != nil {
		return notFound(err)
	}
	return checkIdle(ctx, tx, assetID)
}

// EditClipIfIdle sets a clip's start, end and title, and marks it edited,
// while none of its asset's jobs is pending or running. With retrack, the
// clip's face track is cleared and retrack is added as a pending job and
// queued, in the same transaction, to track the new range. It returns
// ErrNotFound for an unknown clip and ErrBusy while the asset is busy.
func (s *Store) EditClipIfIdle(ctx context.Context, c domain.Clip, retrack *domain.Job) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var assetID string
		err := tx.QueryRow(ctx, `SELECT asset_id FROM clips WHERE id = $1`, c.ID).Scan(&assetID)
		if err != nil {
			return notFound(err)
		}
		if err := lockIdleAsset(ctx, tx, assetID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE clips SET start_seconds = $2, end_seconds = $3, title = $4, edited_at = NOW(),
			 face_track = CASE WHEN $5 THEN NULL ELSE face_track END
			 WHERE id = $1`,
			c.ID, c.Start, c.End, c.Title, retrack != nil)
		if err != nil || retrack == nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO processing_jobs (id, asset_id, type, input_json, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			retrack.ID, assetID, retrack.Type, retrack.InputJSON, domain.JobPending, retrack.CreatedAt)
		if err != nil {
			return err
		}
		retrack.AssetID = assetID
		return queueJob(ctx, tx, *retrack)
	})
}

// SetFaceTrack stores where the main face is through a clip.
func (s *Store) SetFaceTrack(ctx context.Context, clipID string, track []byte) error {
	_, err := s.db.Exec(ctx, `UPDATE clips SET face_track = $2 WHERE id = $1`, clipID, track)
	return err
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
