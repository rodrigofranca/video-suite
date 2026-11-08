DROP INDEX IF EXISTS idx_presets_watermark_id;
ALTER TABLE presets
    DROP COLUMN IF EXISTS watermark_id,
    DROP COLUMN IF EXISTS watermark_position,
    DROP COLUMN IF EXISTS watermark_size,
    DROP COLUMN IF EXISTS watermark_opacity;
DROP TABLE IF EXISTS watermarks;
