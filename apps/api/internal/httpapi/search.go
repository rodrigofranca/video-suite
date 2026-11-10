package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"ultramedia/media-core/store"
)

// Search over what was said in every video (docs/adr/0038).
const (
	maxSearchLength = 200   // characters of a query
	searchPage      = 20    // videos per page
	searchMoments   = 3     // moments shown per video
	maxSearchOffset = 10000 // deepest page
)

type searchDTO struct {
	Query string `json:"query"`
	// Searched is how many videos have a transcript: those a search looks through.
	Searched int              `json:"searched"`
	Videos   []searchVideoDTO `json:"videos"`
	// More says there are more videos after these; ask with offset.
	More bool `json:"more"`
}

type searchVideoDTO struct {
	Asset    assetDTO `json:"asset"`
	Language string   `json:"language"`
	// Matches is how many moments of the video match; Moments are the
	// best of them, in the order they were said.
	Matches int         `json:"matches"`
	Moments []momentDTO `json:"moments"`
}

type momentDTO struct {
	Start float64   `json:"start"`
	End   float64   `json:"end"`
	Parts []partDTO `json:"parts"`
}

// partDTO is a piece of what was said; match marks the words that matched.
type partDTO struct {
	Text  string `json:"text"`
	Match bool   `json:"match,omitempty"`
}

// search answers GET /search?q=...&offset=N: the videos whose speech
// matches, best first, with their best moments.
func (a *api) search(w http.ResponseWriter, r *http.Request) {
	q := strings.Join(strings.Fields(r.URL.Query().Get("q")), " ")
	if q == "" {
		writeError(w, http.StatusBadRequest, "type what to search for")
		return
	}
	if utf8.RuneCountInString(q) > maxSearchLength {
		writeError(w, http.StatusBadRequest, "a search has up to "+strconv.Itoa(maxSearchLength)+" characters")
		return
	}
	offset := 0
	if s := r.URL.Query().Get("offset"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > maxSearchOffset {
			writeError(w, http.StatusBadRequest, "offset must be a number from 0 to "+strconv.Itoa(maxSearchOffset))
			return
		}
		offset = n
	}

	ctx := r.Context()
	found, err := a.Store.SearchSpeech(ctx, store.SpeechSearch{Query: q, Limit: searchPage + 1, Offset: offset, Moments: searchMoments})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	searched, err := a.Store.CountTranscripts(ctx)
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	out := searchDTO{Query: q, Searched: searched, Videos: []searchVideoDTO{}, More: len(found) > searchPage}
	for _, v := range found[:min(len(found), searchPage)] {
		asset := toAsset(v.Asset)
		asset.ThumbnailURL = a.publicURL(v.ThumbnailKey)
		asset.Metadata = v.Summary
		video := searchVideoDTO{Asset: asset, Language: v.Language, Matches: v.Matches, Moments: []momentDTO{}}
		for _, m := range v.Moments {
			moment := momentDTO{Start: m.Start, End: m.End, Parts: []partDTO{}}
			for _, p := range m.Parts {
				moment.Parts = append(moment.Parts, partDTO{Text: p.Text, Match: p.Match})
			}
			video.Moments = append(video.Moments, moment)
		}
		out.Videos = append(out.Videos, video)
	}
	writeJSON(w, http.StatusOK, out)
}
