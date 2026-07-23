CREATE TABLE technical_metadata (
    id            UUID PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    extracted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
