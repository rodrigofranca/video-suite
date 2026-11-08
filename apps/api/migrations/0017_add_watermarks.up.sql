-- Watermark images and their place in presets (docs/adr/0031). An image is
-- a PNG or JPEG, private at storage_key; its size is read on upload.
CREATE TABLE watermarks (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    filename    TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    width       INTEGER NOT NULL,
    height      INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The corner, the width as a percentage of the clip's shorter side, and the
-- opacity in percent. An image in use cannot be deleted.
ALTER TABLE presets
    ADD COLUMN watermark_id UUID REFERENCES watermarks(id) ON DELETE RESTRICT,
    ADD COLUMN watermark_position TEXT NOT NULL DEFAULT 'top-right',
    ADD COLUMN watermark_size INTEGER NOT NULL DEFAULT 15,
    ADD COLUMN watermark_opacity INTEGER NOT NULL DEFAULT 80;
CREATE INDEX idx_presets_watermark_id ON presets (watermark_id);
