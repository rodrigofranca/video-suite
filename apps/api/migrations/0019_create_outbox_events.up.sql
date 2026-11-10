-- The outbox (docs/adr/0034): domain events recorded in the same transaction
-- as the change that caused them, and handled afterwards by a relay. A
-- job.queued event sends a job to its Redis stream; a job.finished event
-- moves the pipeline on. Events go with their video and job.
CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,
    type            TEXT NOT NULL,
    asset_id        UUID REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id          UUID REFERENCES processing_jobs(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Set once a relay has handled the event; handled events are kept for a
    -- while, as a history, and then pruned.
    handled_at      TIMESTAMPTZ,
    -- A failed attempt waits before the next one, longer each time.
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- What a relay looks for: events not handled yet, oldest first.
CREATE INDEX idx_outbox_events_due ON outbox_events (next_attempt_at, id) WHERE handled_at IS NULL;
CREATE INDEX idx_outbox_events_asset_id ON outbox_events (asset_id);
CREATE INDEX idx_outbox_events_job_id ON outbox_events (job_id);
CREATE INDEX idx_outbox_events_handled_at ON outbox_events (handled_at) WHERE handled_at IS NOT NULL;

-- Wake the relays when a transaction that recorded events commits, so they
-- do not wait for their next poll. Whoever writes events need not know.
CREATE FUNCTION notify_outbox_events() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('outbox_events', '');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER outbox_events_notify
AFTER INSERT ON outbox_events
FOR EACH STATEMENT EXECUTE FUNCTION notify_outbox_events();
