-- Translations of an asset's transcript, written by a language model for
-- captions in another language (docs/adr/0029). One per asset and language;
-- a new transcript removes them, since they no longer match its speech.
-- segments: the transcript's segments, same times, translated text.
-- words: [{"start", "end", "word"}], the translated words placed on the
-- original speech, for word-by-word captions.
CREATE TABLE transcript_translations (
    id              UUID PRIMARY KEY,
    asset_id        UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    job_id          UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    language        TEXT NOT NULL,
    source_language TEXT NOT NULL,
    segments        JSONB NOT NULL DEFAULT '[]',
    words           JSONB NOT NULL DEFAULT '[]',
    model           TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX transcript_translations_asset_language_key ON transcript_translations (asset_id, language);
CREATE INDEX idx_transcript_translations_job_id ON transcript_translations (job_id);
