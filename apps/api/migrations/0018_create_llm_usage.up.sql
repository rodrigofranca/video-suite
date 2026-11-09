-- Language model usage (docs/adr/0032): one row per call to the model, with
-- its tokens and the cost the gateway reported (NULL when it did not know the
-- price). Rows outlive their video and job, so the totals stay right after a
-- video is deleted.
CREATE TABLE llm_usage (
    id                UUID PRIMARY KEY,
    asset_id          UUID REFERENCES media_assets(id) ON DELETE SET NULL,
    job_id            UUID REFERENCES processing_jobs(id) ON DELETE SET NULL,
    job_type          TEXT NOT NULL,
    model             TEXT NOT NULL,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd          NUMERIC(14, 8),
    -- Read back from a job's log line, for jobs that ran before this table:
    -- one row per job with its last call's (or, for translations, all) tokens.
    from_log          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_llm_usage_asset_id ON llm_usage (asset_id);
CREATE INDEX idx_llm_usage_job_id ON llm_usage (job_id);
CREATE INDEX idx_llm_usage_created_at ON llm_usage (created_at);

INSERT INTO llm_usage (id, asset_id, job_id, job_type, model, prompt_tokens, completion_tokens, from_log, created_at)
SELECT gen_random_uuid(), j.asset_id, j.id, j.type, m[1], m[2]::integer, m[3]::integer, TRUE, l.timestamp
FROM job_logs l
JOIN processing_jobs j ON j.id = l.job_id
CROSS JOIN LATERAL regexp_match(l.message, 'model (\S+), (\d+) tokens in, (\d+) out') AS m
WHERE m IS NOT NULL;
