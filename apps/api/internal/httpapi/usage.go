package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"ultramedia/media-core/store"
)

// Language model usage (docs/adr/0032).

type usageDTO struct {
	Calls            int   `json:"calls"`
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	// CostUSD sums the calls the gateway priced; UnpricedCalls counts the
	// others. FromLogCalls were read back from the logs of jobs that ran
	// before calls were recorded one by one.
	CostUSD       float64 `json:"costUsd"`
	UnpricedCalls int     `json:"unpricedCalls"`
	FromLogCalls  int     `json:"fromLogCalls"`
}

type usageGroupDTO struct {
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
	usageDTO
}

func toUsage(u store.Usage) usageDTO {
	return usageDTO{Calls: u.Calls, PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens, CostUSD: u.CostUSD,
		UnpricedCalls: u.UnpricedCalls, FromLogCalls: u.FromLogCalls}
}

func toUsageGroups(groups []store.UsageGroup) []usageGroupDTO {
	out := make([]usageGroupDTO, 0, len(groups))
	for _, g := range groups {
		out = append(out, usageGroupDTO{Key: g.Key, Label: g.Label, usageDTO: toUsage(g.Usage)})
	}
	return out
}

// assetUsageDTO is a video's usage, in total and per job type.
type assetUsageDTO struct {
	usageDTO
	ByAction []usageGroupDTO `json:"byAction"`
}

// usageDays are the periods GET /usage takes; 0 is everything.
var usageDays = map[int]bool{1: true, 7: true, 30: true, 90: true, 0: true}

// getUsage handles GET /usage?days=30 (1, 7, 30, 90, or 0 for everything):
// the language model usage of the period, in total and per day (UTC), model,
// job type and video.
func (a *api) getUsage(w http.ResponseWriter, r *http.Request) {
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || !usageDays[n] {
			writeError(w, http.StatusBadRequest, "days must be 1, 7, 30, 90 or 0 (everything)")
			return
		}
		days = n
	}
	since := time.Time{}
	if days > 0 {
		now := time.Now().UTC()
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1-days)
	}
	sum, err := a.Store.UsageSince(r.Context(), since)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	var sinceOut *time.Time
	if days > 0 {
		sinceOut = &since
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"days": days, "since": sinceOut, "total": toUsage(sum.Total),
		"byDay": toUsageGroups(sum.ByDay), "byModel": toUsageGroups(sum.ByModel),
		"byAction": toUsageGroups(sum.ByAction), "byVideo": toUsageGroups(sum.ByAsset),
	})
}
