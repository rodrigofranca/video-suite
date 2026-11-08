package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ultramedia/media-core/domain"
)

// ErrNameTaken is returned when another preset has the name (in any case).
var ErrNameTaken = errors.New("another preset has this name")

// InUseError is returned when a vignette cannot be deleted because presets use it.
type InUseError struct{ Presets []string }

func (e *InUseError) Error() string {
	return "the vignette is used by the preset(s) " + strings.Join(e.Presets, ", ") + "; take it out of them first"
}

// ---- Vignettes ----

const vignetteColumns = `id, name, filename, storage_key, size_bytes, created_at`

func scanVignette(row pgx.Row) (domain.Vignette, error) {
	var v domain.Vignette
	err := row.Scan(&v.ID, &v.Name, &v.Filename, &v.StorageKey, &v.SizeBytes, &v.CreatedAt)
	return v, notFound(err)
}

// CreateVignette adds an uploaded vignette to the library.
func (s *Store) CreateVignette(ctx context.Context, v domain.Vignette) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO vignettes (id, name, filename, storage_key, size_bytes) VALUES ($1, $2, $3, $4, $5)`,
		v.ID, v.Name, v.Filename, v.StorageKey, v.SizeBytes)
	return err
}

func (s *Store) GetVignette(ctx context.Context, id string) (domain.Vignette, error) {
	return scanVignette(s.db.QueryRow(ctx, `SELECT `+vignetteColumns+` FROM vignettes WHERE id = $1`, id))
}

// ListVignettes returns the library, newest first.
func (s *Store) ListVignettes(ctx context.Context) ([]domain.Vignette, error) {
	rows, err := s.db.Query(ctx, `SELECT `+vignetteColumns+` FROM vignettes ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Vignette, error) { return scanVignette(r) })
}

// DeleteVignette removes a vignette from the library. It returns an
// *InUseError while a preset uses it, and ErrNotFound for an unknown one.
// The caller removes the stored file.
func (s *Store) DeleteVignette(ctx context.Context, id string) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT name FROM presets
			 WHERE $1 IN (intro_horizontal_id, intro_vertical_id, outro_horizontal_id, outro_vertical_id)
			 ORDER BY lower(name) FOR UPDATE`, id)
		if err != nil {
			return err
		}
		names, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		if len(names) > 0 {
			return &InUseError{Presets: names}
		}
		tag, err := tx.Exec(ctx, `DELETE FROM vignettes WHERE id = $1`, id)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
}

// ---- Presets ----

const presetColumns = `id, name, captions, caption_style, caption_position, caption_color, caption_caps, caption_language,
	COALESCE(intro_horizontal_id::text, ''), COALESCE(intro_vertical_id::text, ''),
	COALESCE(outro_horizontal_id::text, ''), COALESCE(outro_vertical_id::text, ''), created_at, updated_at`

func scanPreset(row pgx.Row) (domain.Preset, error) {
	var p domain.Preset
	err := row.Scan(&p.ID, &p.Name, &p.Captions, &p.CaptionStyle, &p.CaptionPosition, &p.CaptionColor, &p.CaptionCaps,
		&p.CaptionLanguage, &p.IntroHorizontal, &p.IntroVertical, &p.OutroHorizontal, &p.OutroVertical, &p.CreatedAt, &p.UpdatedAt)
	return p, notFound(err)
}

// orNull turns an empty ID into SQL NULL.
func orNull(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// nameTaken maps a unique violation on the preset name to ErrNameTaken.
func nameTaken(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "presets_name_key" {
		return ErrNameTaken
	}
	return err
}

// CreatePreset saves a new preset. It returns ErrNameTaken when another
// preset has the name.
func (s *Store) CreatePreset(ctx context.Context, p domain.Preset) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO presets (id, name, captions, caption_style, caption_position, caption_color, caption_caps, caption_language,
		                      intro_horizontal_id, intro_vertical_id, outro_horizontal_id, outro_vertical_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		p.ID, p.Name, p.Captions, p.CaptionStyle, p.CaptionPosition, p.CaptionColor, p.CaptionCaps, p.CaptionLanguage,
		orNull(p.IntroHorizontal), orNull(p.IntroVertical), orNull(p.OutroHorizontal), orNull(p.OutroVertical))
	return nameTaken(err)
}

// UpdatePreset replaces a preset's choices. It returns ErrNotFound for an
// unknown preset and ErrNameTaken when another preset has the name.
func (s *Store) UpdatePreset(ctx context.Context, p domain.Preset) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE presets SET name = $2, captions = $3, caption_style = $4, caption_position = $5, caption_color = $6,
		        caption_caps = $7, caption_language = $8, intro_horizontal_id = $9, intro_vertical_id = $10,
		        outro_horizontal_id = $11, outro_vertical_id = $12, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.Name, p.Captions, p.CaptionStyle, p.CaptionPosition, p.CaptionColor, p.CaptionCaps, p.CaptionLanguage,
		orNull(p.IntroHorizontal), orNull(p.IntroVertical), orNull(p.OutroHorizontal), orNull(p.OutroVertical))
	if err != nil {
		return nameTaken(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetPreset(ctx context.Context, id string) (domain.Preset, error) {
	return scanPreset(s.db.QueryRow(ctx, `SELECT `+presetColumns+` FROM presets WHERE id = $1`, id))
}

// ListPresets returns the presets by name.
func (s *Store) ListPresets(ctx context.Context) ([]domain.Preset, error) {
	rows, err := s.db.Query(ctx, `SELECT `+presetColumns+` FROM presets ORDER BY lower(name), id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Preset, error) { return scanPreset(r) })
}

// DeletePreset removes a preset; exports made with it keep their files.
func (s *Store) DeletePreset(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM presets WHERE id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
