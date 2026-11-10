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
