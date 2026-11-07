DROP INDEX IF EXISTS technical_metadata_asset_id_key;
CREATE INDEX idx_technical_metadata_asset_id ON technical_metadata (asset_id);
