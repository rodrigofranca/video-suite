-- One technical_metadata row per asset, so a re-run of the metadata job
-- replaces the previous result (INSERT ... ON CONFLICT (asset_id)).
DELETE FROM technical_metadata t
USING technical_metadata newer
WHERE t.asset_id = newer.asset_id
  AND (t.extracted_at, t.id) < (newer.extracted_at, newer.id);

DROP INDEX IF EXISTS idx_technical_metadata_asset_id;
CREATE UNIQUE INDEX technical_metadata_asset_id_key ON technical_metadata (asset_id);
