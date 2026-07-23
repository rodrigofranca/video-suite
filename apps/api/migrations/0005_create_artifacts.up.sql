CREATE TABLE artifacts (
    id            UUID PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id        UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    type          TEXT NOT NULL,
    storage_key   TEXT NOT NULL,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
