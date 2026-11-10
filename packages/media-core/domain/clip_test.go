package domain

import (
	"strings"
	"testing"
)

func TestClipProblem(t *testing.T) {
	ok := Clip{Start: 10, End: 40, Title: "A good moment"}
	for name, tc := range map[string]struct {
		clip     Clip
		duration float64
		want     string
	}{
		"fine":                  {ok, 60, ""},
		"unknown duration":      {Clip{Start: 10, End: 400, Title: "x"}, 0, "a clip lasts from 3 to 180 seconds"},
		"end before start":      {Clip{Start: 40, End: 10, Title: "x"}, 60, "before the end"},
		"negative start":        {Clip{Start: -1, End: 10, Title: "x"}, 60, "at least 0"},
		"too short":             {Clip{Start: 10, End: 12, Title: "x"}, 60, "from 3 to 180"},
		"past the video":        {Clip{Start: 10, End: 61, Title: "x"}, 60, "after the end of the video (60.0 s)"},
		"a rounding hair after": {Clip{Start: 10, End: 60.04, Title: "x"}, 60, ""},
		"no title":              {Clip{Start: 10, End: 40}, 60, "cannot be empty"},
		"long title":            {Clip{Start: 10, End: 40, Title: strings.Repeat("é", MaxClipTitle+1)}, 60, "longer than 120"},
	} {
		got := tc.clip.Problem(tc.duration)
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: %q, want %q", name, got, tc.want)
		}
	}
}

func TestSaid(t *testing.T) {
	tr := Transcript{
		Segments: []byte(`[{"start": 0, "end": 4, "text": " Bom dia a todos."}, {"start": 4, "end": 9, "text": " A reforma foi aprovada."}]`),
		Words: []byte(`[{"start": 0, "end": 0.5, "word": "Bom"}, {"start": 0.5, "end": 1, "word": "dia"},
			{"start": 1, "end": 1.4, "word": "a"}, {"start": 1.4, "end": 2, "word": "todos."},
			{"start": 4.2, "end": 4.4, "word": "A"}, {"start": 4.4, "end": 5, "word": "reforma"},
			{"start": 5, "end": 5.4, "word": "foi"}, {"start": 5.4, "end": 6.2, "word": "aprovada."}]`),
	}
	for _, c := range []struct {
		start, end float64
		want       string
	}{
		{4.1, 6.4, "A reforma foi aprovada."},
		{0.4, 1.3, "dia a"}, // a word counts when its middle is inside
		{2.5, 3.9, ""},
	} {
		if got := tr.Said(c.start, c.end); got != c.want {
			t.Errorf("Said(%v, %v) = %q, want %q", c.start, c.end, got, c.want)
		}
	}
	// Without word timings: whole segments.
	tr.Words = []byte("[]")
	if got := tr.Said(0, 9); got != "Bom dia a todos. A reforma foi aprovada." {
		t.Errorf("segments: %q", got)
	}
}

func TestQuoteTitle(t *testing.T) {
	if got := QuoteTitle("  A reforma   foi aprovada. "); got != "A reforma foi aprovada." {
		t.Errorf("short: %q", got)
	}
	long := strings.Repeat("palavra ", 30) // 240 characters
	got := QuoteTitle(long)
	if n := len([]rune(got)); n > MaxClipTitle || !strings.HasSuffix(got, "palavra…") {
		t.Errorf("long: %d characters, %q", n, got)
	}
}
