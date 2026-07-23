CREATE TABLE media_assets (
    id          UUID PRIMARY KEY,
    filename    TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('uploading', 'processing', 'ready', 'failed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
