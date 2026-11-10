package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Problem says what is wrong with a clip changed by hand, or "" when
// nothing is (docs/adr/0024). videoDuration is 0 when unknown: the end is
// then not checked against it.
func (c Clip) Problem(videoDuration float64) string {
	switch {
	case c.Start < 0 || c.End <= c.Start:
		return "the start must be at least 0 and before the end"
	case c.End-c.Start < MinClipSeconds || c.End-c.Start > MaxClipSeconds:
		return fmt.Sprintf("a clip lasts from %.0f to %.0f seconds", MinClipSeconds, MaxClipSeconds)
	case videoDuration > 0 && c.End > videoDuration+0.05:
		return fmt.Sprintf("the end is after the end of the video (%.1f s)", videoDuration)
	case c.Title == "":
		return "the title cannot be empty"
	case len([]rune(c.Title)) > MaxClipTitle:
		return fmt.Sprintf("the title is longer than %d characters", MaxClipTitle)
	}
	return ""
}

// Said is what was said between start and end (seconds): the words whose
// middle falls inside, or, for a transcript without word timings, the
// segments whose middle does. It is a quote's text (docs/adr/0039).
func (t Transcript) Said(start, end float64) string {
	type piece struct {
		Start, End float64
		Word, Text string
	}
	inside := func(list []byte, text func(piece) string) string {
		var pieces []piece
		json.Unmarshal(list, &pieces)
		var out []string
		for _, p := range pieces {
			if mid := (p.Start + p.End) / 2; mid >= start && mid <= end {
				out = append(out, strings.TrimSpace(text(p)))
			}
		}
		return strings.Join(strings.Fields(strings.Join(out, " ")), " ")
	}
	if said := inside(t.Words, func(p piece) string { return p.Word }); said != "" {
		return said
	}
	return inside(t.Segments, func(p piece) string { return p.Text })
}

// QuoteTitle makes a clip title from a quote: the quote itself, cut at a
// word with "…" when it is longer than MaxClipTitle.
func QuoteTitle(quote string) string {
	quote = strings.Join(strings.Fields(quote), " ")
	runes := []rune(quote)
	if len(runes) <= MaxClipTitle {
		return quote
	}
	cut := string(runes[:MaxClipTitle-1])
	if i := strings.LastIndex(cut, " "); i > MaxClipTitle/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:.-") + "…"
}
