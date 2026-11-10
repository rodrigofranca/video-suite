package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Language model usage (docs/adr/0032). The AI worker writes one llm_usage
// row per call; these read the totals.

// Usage is what a set of calls to the language model used.
type Usage struct {
	Calls            int
	PromptTokens     int64
	CompletionTokens int64
	// CostUSD sums the calls with a known price; UnpricedCalls counts the
	// others, and FromLogCalls the calls read back from job logs.
	CostUSD       float64
	UnpricedCalls int
	FromLogCalls  int
}

// UsageGroup is the usage of one day, model, job type or video.
type UsageGroup struct {
	Key   string // the day (YYYY-MM-DD), model, job type or asset ID ("" for deleted videos)
	Label string // for videos, the file name
	Usage
}

// UsageSummary is the usage since a time, in total and grouped.
type UsageSummary struct {
	Total                             Usage
	ByDay, ByModel, ByAction, ByAsset []UsageGroup
}

const usageSums = `COUNT(*), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0),
	COALESCE(SUM(cost_usd), 0)::float8, COUNT(*) FILTER (WHERE cost_usd IS NULL), COUNT(*) FILTER (WHERE from_log)`

func scanUsage(row pgx.Row, extra ...any) (Usage, error) {
	var u Usage
	err := row.Scan(append(extra, &u.Calls, &u.PromptTokens, &u.CompletionTokens, &u.CostUSD, &u.UnpricedCalls, &u.FromLogCalls)...)
	return u, err
}

func (s *Store) usageGroups(ctx context.Context, query string, args ...any) ([]UsageGroup, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	groups, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (UsageGroup, error) {
		var g UsageGroup
		u, err := scanUsage(r, &g.Key, &g.Label)
		g.Usage = u
		return g, err
	})
	if groups == nil {
		groups = []UsageGroup{}
	}
	return groups, err
}

// AssetUsage returns an asset's usage, in total and per job type.
func (s *Store) AssetUsage(ctx context.Context, assetID string) (Usage, []UsageGroup, error) {
	total, err := scanUsage(s.db.QueryRow(ctx, `SELECT `+usageSums+` FROM llm_usage WHERE asset_id = $1`, assetID))
	if err != nil {
		return total, nil, err
	}
	byAction, err := s.usageGroups(ctx,
		`SELECT job_type, '', `+usageSums+` FROM llm_usage WHERE asset_id = $1 GROUP BY job_type ORDER BY job_type`, assetID)
	return total, byAction, err
}

// UsageSince sums the usage since a time: in total, per day (UTC), model,
// job type, and video (the 20 that used the most tokens; deleted videos
// together under an empty key).
func (s *Store) UsageSince(ctx context.Context, since time.Time) (UsageSummary, error) {
	var sum UsageSummary
	var err error
	if sum.Total, err = scanUsage(s.db.QueryRow(ctx, `SELECT `+usageSums+` FROM llm_usage WHERE created_at >= $1`, since)); err != nil {
		return sum, err
	}
	if sum.ByDay, err = s.usageGroups(ctx,
		`SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, '', `+usageSums+`
		 FROM llm_usage WHERE created_at >= $1 GROUP BY day ORDER BY day`, since); err != nil {
		return sum, err
	}
	if sum.ByModel, err = s.usageGroups(ctx,
		`SELECT model, '', `+usageSums+` FROM llm_usage WHERE created_at >= $1
		 GROUP BY model ORDER BY SUM(prompt_tokens + completion_tokens) DESC, model`, since); err != nil {
		return sum, err
	}
	if sum.ByAction, err = s.usageGroups(ctx,
		`SELECT job_type, '', `+usageSums+` FROM llm_usage WHERE created_at >= $1
		 GROUP BY job_type ORDER BY SUM(prompt_tokens + completion_tokens) DESC, job_type`, since); err != nil {
		return sum, err
	}
	sum.ByAsset, err = s.usageGroups(ctx,
		`SELECT COALESCE(u.asset_id::text, ''), COALESCE(MAX(a.filename), ''), `+usageSums+`
		 FROM llm_usage u LEFT JOIN media_assets a ON a.id = u.asset_id
		 WHERE u.created_at >= $1
		 GROUP BY u.asset_id ORDER BY SUM(u.prompt_tokens + u.completion_tokens) DESC LIMIT 20`, since)
	return sum, err
}

// AddUsage records a call; the AI worker does this, so it is here for tests.
func (s *Store) AddUsage(ctx context.Context, id, assetID, jobID, jobType, model string, prompt, completion int, cost *float64, at time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO llm_usage (id, asset_id, job_id, job_type, model, prompt_tokens, completion_tokens, cost_usd, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, orNull(assetID), orNull(jobID), jobType, model, prompt, completion, cost, at)
	return err
}
