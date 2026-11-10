package domain

import "fmt"

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
