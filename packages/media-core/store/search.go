package store

import (
	"context"
	"strings"

	"ultramedia/media-core/domain"
)

// ---- Search (docs/adr/0038) ----

// TextPart is a piece of a transcript segment; Match marks the words that
// matched the search.
type TextPart struct {
	Text  string
	Match bool
}

// SpeechMoment is a transcript segment that matches a search.
type SpeechMoment struct {
	Start, End float64 // seconds
	Parts      []TextPart
}

// SpeechResult is a video whose speech matches a search.
type SpeechResult struct {
	AssetListItem
	Language string // the transcript's
	Matches  int    // segments that match
	Moments  []SpeechMoment
}

// SpeechSearch says what to find and how much of it to return.
type SpeechSearch struct {
	// Query is as someone typed it: words, "a phrase", -word to leave a
	// word out, "or" between alternatives. Accents and word endings do not
	// matter.
	Query string
	// Limit and Offset page through the videos.
	Limit, Offset int
	// Moments is how many matching segments to return per video, the best
	// ones; they come in the order they were said.
	Moments int
}

// Markers ts_headline puts around a matching word; control characters, so
// they are never in a transcript.
const (
	matchStart = "\x02"
	matchEnd   = "\x03"
)

// SearchSpeech finds the videos whose transcript matches the search, best
// match first and, among equal ones, the newest first. The search_query and
// search_config functions of migration 0020 read the query and choose each
// transcript's language rules.
func (s *Store) SearchSpeech(ctx context.Context, q SpeechSearch) ([]SpeechResult, error) {
	rows, err := s.db.Query(ctx,
		`WITH q AS MATERIALIZED (SELECT search_query($1) AS query),
		 hits AS (
		     SELECT s.asset_id, s.start_seconds, s.end_seconds, s.text, s.config,
		            ts_rank_cd(s.tsv, q.query) AS rank
		     FROM transcript_segments s, q
		     WHERE s.tsv @@ q.query
		 ),
		 ranked AS (
		     SELECT h.*, row_number() OVER (PARTITION BY h.asset_id ORDER BY h.rank DESC, h.start_seconds) AS n,
		            max(h.rank) OVER (PARTITION BY h.asset_id) AS best,
		            count(*) OVER (PARTITION BY h.asset_id) AS matches
		     FROM hits h
		 ),
		 videos AS (
		     SELECT r.asset_id, r.best, r.matches, a.created_at
		     FROM ranked r JOIN media_assets a ON a.id = r.asset_id
		     WHERE r.n = 1
		     ORDER BY r.best DESC, a.created_at DESC, r.asset_id
		     LIMIT $2 OFFSET $3
		 )
		 SELECT a.id, a.filename, a.status, a.created_at, a.updated_at,
		        th.storage_key, tm.metadata_json->'summary', t.language, v.matches,
		        m.start_seconds, m.end_seconds, ts_headline(m.config, m.text, q.query, $5)
		 FROM videos v
		 JOIN media_assets a ON a.id = v.asset_id
		 JOIN transcripts t ON t.asset_id = v.asset_id
		 JOIN ranked m ON m.asset_id = v.asset_id AND m.n <= $4
		 CROSS JOIN q
		 LEFT JOIN LATERAL (
		     SELECT storage_key FROM artifacts
		     WHERE asset_id = a.id AND type = $6
		     ORDER BY created_at DESC LIMIT 1
		 ) th ON true
		 LEFT JOIN technical_metadata tm ON tm.asset_id = a.id
		 ORDER BY v.best DESC, v.created_at DESC, v.asset_id, m.start_seconds`,
		q.Query, q.Limit, q.Offset, q.Moments,
		"HighlightAll=true, StartSel="+matchStart+", StopSel="+matchEnd, domain.ArtifactThumbnail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SpeechResult
	for rows.Next() {
		var it SpeechResult
		var thumb *string
		var summary []byte
		var m SpeechMoment
		var text string
		err := rows.Scan(&it.ID, &it.Filename, &it.Status, &it.CreatedAt, &it.UpdatedAt,
			&thumb, &summary, &it.Language, &it.Matches, &m.Start, &m.End, &text)
		if err != nil {
			return nil, err
		}
		m.Parts = textParts(text)
		if n := len(out); n > 0 && out[n-1].ID == it.ID {
			out[n-1].Moments = append(out[n-1].Moments, m)
			continue
		}
		if thumb != nil {
			it.ThumbnailKey = *thumb
		}
		if summary != nil {
			it.Summary = summary
		}
		it.Moments = []SpeechMoment{m}
		out = append(out, it)
	}
	return out, rows.Err()
}

// textParts splits a ts_headline result at its markers.
func textParts(marked string) []TextPart {
	var parts []TextPart
	for marked != "" {
		start := strings.Index(marked, matchStart)
		if start < 0 {
			parts = append(parts, TextPart{Text: marked})
			break
		}
		if start > 0 {
			parts = append(parts, TextPart{Text: marked[:start]})
		}
		rest := marked[start+len(matchStart):]
		end := strings.Index(rest, matchEnd)
		if end < 0 {
			end = len(rest)
		}
		parts = append(parts, TextPart{Text: rest[:end], Match: true})
		marked = strings.TrimPrefix(rest[end:], matchEnd)
	}
	return parts
}

// CountTranscripts is how many videos have a transcript, which is what a
// search looks through.
func (s *Store) CountTranscripts(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM transcripts`).Scan(&n)
	return n, err
}
