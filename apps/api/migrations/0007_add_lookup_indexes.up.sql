-- PostgreSQL does not index foreign keys automatically. These cover the
-- lookups the API and worker make: jobs/files/logs per asset or job, and
-- the dashboard filter by asset status.
CREATE INDEX idx_media_assets_status ON media_assets (status);
CREATE INDEX idx_media_sources_asset_id ON media_sources (asset_id);
CREATE INDEX idx_processing_jobs_asset_id ON processing_jobs (asset_id);
CREATE INDEX idx_processing_jobs_status ON processing_jobs (status);
CREATE INDEX idx_job_logs_job_id ON job_logs (job_id);
CREATE INDEX idx_artifacts_asset_id ON artifacts (asset_id);
CREATE INDEX idx_artifacts_job_id ON artifacts (job_id);
CREATE INDEX idx_technical_metadata_asset_id ON technical_metadata (asset_id);
