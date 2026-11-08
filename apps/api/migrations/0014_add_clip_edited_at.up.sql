-- When someone last changed a clip's start, end or title by hand
-- (docs/adr/0024); NULL for a clip as the model suggested it.
ALTER TABLE clips ADD COLUMN edited_at TIMESTAMPTZ;
