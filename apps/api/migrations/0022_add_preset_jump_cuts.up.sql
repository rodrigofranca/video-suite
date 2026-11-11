-- Whether a preset's exports cut the pauses and filler sounds out of the
-- clip (docs/adr/0040).
ALTER TABLE presets ADD COLUMN jump_cuts BOOLEAN NOT NULL DEFAULT false;
