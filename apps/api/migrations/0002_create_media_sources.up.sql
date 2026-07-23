CREATE TABLE media_sources (
    id          UUID PRIMARY KEY,
    asset_id    UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    mime_type   TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    checksum    TEXT NOT NULL
);
