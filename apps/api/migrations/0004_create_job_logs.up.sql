CREATE TABLE job_logs (
    id        UUID PRIMARY KEY,
    job_id    UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    level     TEXT NOT NULL,
    message   TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
