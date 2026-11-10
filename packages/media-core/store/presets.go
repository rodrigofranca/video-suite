package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ultramedia/media-core/domain"
)

// ErrNameTaken is returned when another preset has the name (in any case).
var ErrNameTaken = domain.ErrNameTaken

// InUseError is returned when a vignette or watermark image cannot be
// deleted because presets use it.
type InUseError = domain.InUseError

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
			return &InUseError{What: "vignette", Presets: names}
		}
		tag, err := tx.Exec(ctx, `DELETE FROM vignettes WHERE id = $1`, id)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
}

// ---- Watermark images ----

const watermarkColumns = `id, name, filename, storage_key, size_bytes, width, height, created_at`

func scanWatermark(row pgx.Row) (domain.Watermark, error) {
	var w domain.Watermark
	err := row.Scan(&w.ID, &w.Name, &w.Filename, &w.StorageKey, &w.SizeBytes, &w.Width, &w.Height, &w.CreatedAt)
	return w, notFound(err)
}

// CreateWatermark adds an uploaded image to the library.
func (s *Store) CreateWatermark(ctx context.Context, w domain.Watermark) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO watermarks (id, name, filename, storage_key, size_bytes, width, height) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		w.ID, w.Name, w.Filename, w.StorageKey, w.SizeBytes, w.Width, w.Height)
	return err
}

func (s *Store) GetWatermark(ctx context.Context, id string) (domain.Watermark, error) {
	return scanWatermark(s.db.QueryRow(ctx, `SELECT `+watermarkColumns+` FROM watermarks WHERE id = $1`, id))
}

// ListWatermarks returns the images, newest first.
func (s *Store) ListWatermarks(ctx context.Context) ([]domain.Watermark, error) {
	rows, err := s.db.Query(ctx, `SELECT `+watermarkColumns+` FROM watermarks ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Watermark, error) { return scanWatermark(r) })
}

// DeleteWatermark removes an image from the library. It returns an
// *InUseError while a preset uses it, and ErrNotFound for an unknown one.
// The caller removes the stored file.
func (s *Store) DeleteWatermark(ctx context.Context, id string) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT name FROM presets WHERE watermark_id = $1 ORDER BY lower(name) FOR UPDATE`, id)
		if err != nil {
			return err
		}
		names, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		if len(names) > 0 {
			return &InUseError{What: "watermark", Presets: names}
		}
		tag, err := tx.Exec(ctx, `DELETE FROM watermarks WHERE id = $1`, id)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
}

// ---- Presets ----

const presetColumns = `id, name, captions, caption_style, caption_position, caption_color, caption_caps, caption_language,
	COALESCE(intro_horizontal_id::text, ''), COALESCE(intro_vertical_id::text, ''),
	COALESCE(outro_horizontal_id::text, ''), COALESCE(outro_vertical_id::text, ''),
	COALESCE(watermark_id::text, ''), watermark_position, watermark_size, watermark_opacity, created_at, updated_at`

func scanPreset(row pgx.Row) (domain.Preset, error) {
	var p domain.Preset
	err := row.Scan(&p.ID, &p.Name, &p.Captions, &p.CaptionStyle, &p.CaptionPosition, &p.CaptionColor, &p.CaptionCaps,
		&p.CaptionLanguage, &p.IntroHorizontal, &p.IntroVertical, &p.OutroHorizontal, &p.OutroVertical,
		&p.Watermark, &p.WatermarkPosition, &p.WatermarkSize, &p.WatermarkOpacity, &p.CreatedAt, &p.UpdatedAt)
	return p, notFound(err)
}

// watermarkDefaults fills the watermark choices a preset left at zero.
func watermarkDefaults(p domain.Preset) domain.Preset {
	if p.WatermarkPosition == "" {
		p.WatermarkPosition = "top-right"
	}
	if p.WatermarkSize == 0 {
		p.WatermarkSize = 15
	}
	if p.WatermarkOpacity == 0 {
		p.WatermarkOpacity = 80
	}
	return p
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
	p = watermarkDefaults(p)
	_, err := s.db.Exec(ctx,
		`INSERT INTO presets (id, name, captions, caption_style, caption_position, caption_color, caption_caps, caption_language,
		                      intro_horizontal_id, intro_vertical_id, outro_horizontal_id, outro_vertical_id,
		                      watermark_id, watermark_position, watermark_size, watermark_opacity)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		p.ID, p.Name, p.Captions, p.CaptionStyle, p.CaptionPosition, p.CaptionColor, p.CaptionCaps, p.CaptionLanguage,
		orNull(p.IntroHorizontal), orNull(p.IntroVertical), orNull(p.OutroHorizontal), orNull(p.OutroVertical),
		orNull(p.Watermark), p.WatermarkPosition, p.WatermarkSize, p.WatermarkOpacity)
	return nameTaken(err)
}

// UpdatePreset replaces a preset's choices. It returns ErrNotFound for an
// unknown preset and ErrNameTaken when another preset has the name.
func (s *Store) UpdatePreset(ctx context.Context, p domain.Preset) error {
	p = watermarkDefaults(p)
	tag, err := s.db.Exec(ctx,
		`UPDATE presets SET name = $2, captions = $3, caption_style = $4, caption_position = $5, caption_color = $6,
		        caption_caps = $7, caption_language = $8, intro_horizontal_id = $9, intro_vertical_id = $10,
		        outro_horizontal_id = $11, outro_vertical_id = $12, watermark_id = $13, watermark_position = $14,
		        watermark_size = $15, watermark_opacity = $16, updated_at = NOW()
		 WHERE id = $1`,
		p.ID, p.Name, p.Captions, p.CaptionStyle, p.CaptionPosition, p.CaptionColor, p.CaptionCaps, p.CaptionLanguage,
		orNull(p.IntroHorizontal), orNull(p.IntroVertical), orNull(p.OutroHorizontal), orNull(p.OutroVertical),
		orNull(p.Watermark), p.WatermarkPosition, p.WatermarkSize, p.WatermarkOpacity)
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
