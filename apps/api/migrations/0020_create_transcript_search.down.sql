DROP FUNCTION IF EXISTS search_query(TEXT);
DROP AGGREGATE IF EXISTS search_or (tsquery);
DROP TRIGGER IF EXISTS transcripts_search_index ON transcripts;
DROP FUNCTION IF EXISTS transcripts_search_index();
DROP FUNCTION IF EXISTS index_transcript(UUID, TEXT, JSONB);
DROP TABLE IF EXISTS transcript_segments;
DROP FUNCTION IF EXISTS search_config(TEXT);

DO $$
DECLARE
    c RECORD;
BEGIN
    FOR c IN SELECT cfgname FROM pg_ts_config WHERE cfgname LIKE 'search\_%' LOOP
        EXECUTE format('DROP TEXT SEARCH CONFIGURATION %I', c.cfgname);
    END LOOP;
END $$;

DROP EXTENSION IF EXISTS unaccent;
