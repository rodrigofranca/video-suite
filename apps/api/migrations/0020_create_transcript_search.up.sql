-- Search over what was said in every video (docs/adr/0038). Each transcript
-- segment becomes a row with its words indexed for full-text search; a
-- trigger keeps the rows in step with the transcript, whoever writes it.

CREATE EXTENSION IF NOT EXISTS unaccent;

-- One text search configuration per transcript language, named
-- search_<ISO 639-1 code>: the language's stemmer with accents removed
-- first, so "votacao" finds "votação" and "votações". ts_headline uses the
-- same configuration, so the words it marks are the words that matched.
DO $$
DECLARE
    l RECORD;
BEGIN
    FOR l IN SELECT * FROM (VALUES
        ('ar', 'arabic'), ('ca', 'catalan'), ('da', 'danish'), ('de', 'german'), ('el', 'greek'),
        ('en', 'english'), ('es', 'spanish'), ('eu', 'basque'), ('fi', 'finnish'), ('fr', 'french'),
        ('ga', 'irish'), ('hi', 'hindi'), ('hu', 'hungarian'), ('hy', 'armenian'), ('id', 'indonesian'),
        ('it', 'italian'), ('lt', 'lithuanian'), ('ne', 'nepali'), ('nl', 'dutch'), ('no', 'norwegian'),
        ('pt', 'portuguese'), ('ro', 'romanian'), ('ru', 'russian'), ('sr', 'serbian'), ('sv', 'swedish'),
        ('ta', 'tamil'), ('tr', 'turkish'), ('yi', 'yiddish')
    ) AS v (code, name) LOOP
        EXECUTE format('CREATE TEXT SEARCH CONFIGURATION search_%s (COPY = %s)', l.code, l.name);
        EXECUTE format('ALTER TEXT SEARCH CONFIGURATION search_%s ALTER MAPPING FOR hword, hword_part, word WITH unaccent, %s_stem',
                       l.code, l.name);
    END LOOP;
END $$;

-- Any other language: words as they are, without accents.
CREATE TEXT SEARCH CONFIGURATION search_simple (COPY = simple);
ALTER TEXT SEARCH CONFIGURATION search_simple ALTER MAPPING FOR hword, hword_part, word WITH unaccent, simple;

-- The configuration for a transcript language such as "pt" or "pt-BR".
CREATE FUNCTION search_config(language TEXT) RETURNS regconfig
LANGUAGE sql STABLE
RETURN COALESCE(
    (SELECT oid::regconfig FROM pg_ts_config WHERE cfgname = 'search_' || lower(split_part(language, '-', 1))),
    'search_simple'::regconfig);

CREATE TABLE transcript_segments (
    asset_id      UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    position      INTEGER NOT NULL, -- 1 for the transcript's first segment
    start_seconds DOUBLE PRECISION NOT NULL,
    end_seconds   DOUBLE PRECISION NOT NULL,
    text          TEXT NOT NULL,
    config        REGCONFIG NOT NULL,
    tsv           TSVECTOR NOT NULL,
    PRIMARY KEY (asset_id, position)
);

CREATE INDEX idx_transcript_segments_tsv ON transcript_segments USING GIN (tsv);

-- index_transcript replaces an asset's rows with the segments of its
-- transcript; segments without words are left out.
CREATE FUNCTION index_transcript(asset UUID, language TEXT, segments JSONB) RETURNS void
LANGUAGE sql AS $$
    DELETE FROM transcript_segments WHERE asset_id = asset;
    INSERT INTO transcript_segments (asset_id, position, start_seconds, end_seconds, text, config, tsv)
    SELECT asset, s.position, (s.segment->>'start')::float8, (s.segment->>'end')::float8,
           btrim(s.segment->>'text'), c.config, to_tsvector(c.config, btrim(s.segment->>'text'))
    FROM jsonb_array_elements(segments) WITH ORDINALITY AS s (segment, position),
         (SELECT search_config(language) AS config) AS c
    WHERE btrim(coalesce(s.segment->>'text', '')) <> '';
$$;

CREATE FUNCTION transcripts_search_index() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM transcript_segments WHERE asset_id = OLD.asset_id;
    ELSE
        PERFORM index_transcript(NEW.asset_id, NEW.language, NEW.segments);
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER transcripts_search_index
AFTER INSERT OR UPDATE OF language, segments OR DELETE ON transcripts
FOR EACH ROW EXECUTE FUNCTION transcripts_search_index();

-- The transcripts written before this migration.
SELECT index_transcript(asset_id, language, segments) FROM transcripts;

-- search_query reads what someone typed (words, "a phrase", -left out, or)
-- in each configuration the transcripts use, and matches any of them: the
-- archive mixes languages and the query does not say which it is in.
CREATE AGGREGATE search_or (tsquery) (SFUNC = tsquery_or, STYPE = tsquery);

CREATE FUNCTION search_query(typed TEXT) RETURNS tsquery
LANGUAGE sql STABLE
RETURN (SELECT search_or(websearch_to_tsquery(c.config, typed))
        FROM (SELECT DISTINCT search_config(language) AS config FROM transcripts) AS c);
