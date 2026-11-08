-- The vignette library and export presets (docs/adr/0030).
-- A vignette is an uploaded MP4, an opening or closing joined to exported
-- clips; its file is private, at storage_key.
CREATE TABLE vignettes (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    filename    TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A preset is a named set of export choices: the captions' look and
-- language, and the opening and closing, each with a horizontal and a
-- vertical version. A vignette in use cannot be deleted.
CREATE TABLE presets (
    id                  UUID PRIMARY KEY,
    name                TEXT NOT NULL,
    captions            BOOLEAN NOT NULL DEFAULT TRUE,
    caption_style       TEXT NOT NULL,
    caption_position    TEXT NOT NULL DEFAULT 'bottom',
    caption_color       TEXT NOT NULL DEFAULT '',
    caption_caps        BOOLEAN,
    caption_language    TEXT NOT NULL DEFAULT '',
    intro_horizontal_id UUID REFERENCES vignettes(id) ON DELETE RESTRICT,
    intro_vertical_id   UUID REFERENCES vignettes(id) ON DELETE RESTRICT,
    outro_horizontal_id UUID REFERENCES vignettes(id) ON DELETE RESTRICT,
    outro_vertical_id   UUID REFERENCES vignettes(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX presets_name_key ON presets (lower(name));
CREATE INDEX idx_presets_intro_horizontal_id ON presets (intro_horizontal_id);
CREATE INDEX idx_presets_intro_vertical_id ON presets (intro_vertical_id);
CREATE INDEX idx_presets_outro_horizontal_id ON presets (outro_horizontal_id);
CREATE INDEX idx_presets_outro_vertical_id ON presets (outro_vertical_id);
