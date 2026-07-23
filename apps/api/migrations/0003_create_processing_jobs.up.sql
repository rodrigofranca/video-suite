CREATE TABLE processing_jobs (
    id            UUID PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    type          TEXT NOT NULL,
    input_json    JSONB NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL CHECK (status IN ('pending', 'running', 'done', 'failed')),
    progress      INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ
);
