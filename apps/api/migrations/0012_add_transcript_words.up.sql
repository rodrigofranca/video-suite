-- When each word of a transcript was said, for word-by-word captions in
-- exported clips (docs/adr/0021): [{"start": 1.2, "end": 1.5, "word": "Hello"}].
-- Transcripts made before this migration have none until transcribed again.
ALTER TABLE transcripts ADD COLUMN words JSONB NOT NULL DEFAULT '[]';
