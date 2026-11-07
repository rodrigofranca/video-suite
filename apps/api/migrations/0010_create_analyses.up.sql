-- What a language model wrote about an asset's transcript: a summary, topic
-- tags and chapters (docs/adr/0019). One per asset; a re-run replaces it.
-- chapters: [{"start": 0.0, "title": "..."}], sorted by start (seconds).
CREATE TABLE analyses (
    id         UUID PRIMARY KEY,
    asset_id   UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id     UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    language   TEXT NOT NULL,
    summary    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    chapters   JSONB NOT NULL DEFAULT '[]',
    model      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX analyses_asset_id_key ON analyses (asset_id);
CREATE INDEX idx_analyses_job_id ON analyses (job_id);
