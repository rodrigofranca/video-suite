-- Clips someone makes from a quote in the transcript (docs/adr/0039): no
-- job suggested them and no model scored them. origin says where a clip
-- came from; a suggested clip still has its job and score.
ALTER TABLE clips ADD COLUMN origin TEXT NOT NULL DEFAULT 'suggested' CHECK (origin IN ('suggested', 'quote'));
ALTER TABLE clips ALTER COLUMN job_id DROP NOT NULL;
ALTER TABLE clips ALTER COLUMN score DROP NOT NULL;
ALTER TABLE clips ADD CONSTRAINT clips_suggested_by_a_job
    CHECK (origin <> 'suggested' OR (job_id IS NOT NULL AND score IS NOT NULL));
