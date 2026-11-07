-- Moments of an asset that work as short standalone videos, suggested by a
-- language model from the transcript (docs/adr/0020). A new suggestion run
-- replaces an asset's clips. Exported files are artifacts of type "clip".
CREATE TABLE clips (
    id            UUID PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id        UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    start_seconds DOUBLE PRECISION NOT NULL CHECK (start_seconds >= 0),
    end_seconds   DOUBLE PRECISION NOT NULL CHECK (end_seconds > start_seconds),
    title         TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    score         INTEGER NOT NULL CHECK (score BETWEEN 1 AND 10),
    rank          INTEGER NOT NULL,
    model         TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clips_asset_id ON clips (asset_id, rank);
CREATE INDEX idx_clips_job_id ON clips (job_id);
