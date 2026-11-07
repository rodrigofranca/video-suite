-- Speech transcripts written by the AI worker, one per asset (a re-run
-- replaces it). segments: [{"start": 0.0, "end": 2.5, "text": "..."}].
CREATE TABLE transcripts (
    id         UUID PRIMARY KEY,
    asset_id   UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id     UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    language   TEXT NOT NULL,
    text       TEXT NOT NULL,
    segments   JSONB NOT NULL DEFAULT '[]',
    model      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX transcripts_asset_id_key ON transcripts (asset_id);
CREATE INDEX idx_transcripts_job_id ON transcripts (job_id);
