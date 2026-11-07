// Package captions turns an asset's WebVTT subtitles into the styled ASS
// file FFmpeg burns into an exported clip.
package captions

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Cue is one subtitle: its text and when it shows, in seconds.
type Cue struct {
	Start, End float64
	Text       string // one line; the renderer wraps it
}

var tag = regexp.MustCompile(`<[^>]*>`)

// ParseVTT reads the cues of a WebVTT file. Cue settings, NOTE and STYLE
// blocks and inline tags are ignored; lines of a cue are joined with spaces.
func ParseVTT(r io.Reader) ([]Cue, error) {
	var cues []Cue
	var cur *Cue
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		switch {
		case line == "":
			if cur != nil && cur.Text != "" {
				cues = append(cues, *cur)
			}
			cur = nil
		case strings.Contains(line, "-->"):
			start, end, err := timing(line)
			if err != nil {
				return nil, err
			}
			cur = &Cue{Start: start, End: end}
		case cur != nil:
			text := strings.Join(strings.Fields(tag.ReplaceAllString(line, "")), " ")
			cur.Text = strings.TrimSpace(cur.Text + " " + text)
		}
	}
	if cur != nil && cur.Text != "" {
		cues = append(cues, *cur)
	}
	return cues, sc.Err()
}

func timing(line string) (float64, float64, error) {
	parts := strings.SplitN(line, "-->", 2)
	start, err := timestamp(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	end, err := timestamp(strings.Fields(parts[1])[0]) // settings may follow
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

// timestamp parses "HH:MM:SS.mmm" or "MM:SS.mmm".
func timestamp(s string) (float64, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("invalid WebVTT timestamp %q", s)
	}
	var total float64
	for _, p := range parts {
		n, err := strconv.ParseFloat(p, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid WebVTT timestamp %q", s)
		}
		total = total*60 + n
	}
	return total, nil
}

// minCue is the shortest cue worth showing, in seconds.
const minCue = 0.2

// Cut returns the cues between start and end, with times relative to start.
func Cut(cues []Cue, start, end float64) []Cue {
	var out []Cue
	for _, c := range cues {
		s, e := max(c.Start, start)-start, min(c.End, end)-start
		if e-s >= minCue {
			out = append(out, Cue{Start: s, End: e, Text: c.Text})
		}
	}
	return out
}

// Split breaks cues longer than maxChars into consecutive cues of whole
// words, sharing the time in proportion to their length. Vertical video has
// room for fewer words per line.
func Split(cues []Cue, maxChars int) []Cue {
	var out []Cue
	for _, c := range cues {
		chunks := chunk(c.Text, maxChars)
		total := 0
		for _, ch := range chunks {
			total += len(ch)
		}
		at := c.Start
		for i, ch := range chunks {
			end := at + (c.End-c.Start)*float64(len(ch))/float64(total)
			if i == len(chunks)-1 {
				end = c.End
			}
			out = append(out, Cue{Start: at, End: end, Text: ch})
			at = end
		}
	}
	return out
}

func chunk(text string, maxChars int) []string {
	var chunks []string
	var cur string
	for _, w := range strings.Fields(text) {
		if cur != "" && len(cur)+1+len(w) > maxChars {
			chunks = append(chunks, cur)
			cur = w
			continue
		}
		cur = strings.TrimSpace(cur + " " + w)
	}
	if cur != "" {
		chunks = append(chunks, cur)
	}
	return chunks
}

// Style sizes the captions for a frame: bold white text with a dark outline,
// centered near the bottom. Vertical video keeps them higher, clear of the
// buttons that social apps draw at the bottom.
type Style struct {
	Width, Height int
	FontSize      int
	MarginV       int // distance from the bottom edge
}

// StyleFor returns the caption style for a frame of the given size.
func StyleFor(width, height int) Style {
	if height > width { // vertical
		return Style{Width: width, Height: height, FontSize: width * 64 / 1080, MarginV: height * 22 / 100}
	}
	return Style{Width: width, Height: height, FontSize: max(height*52/1080, 16), MarginV: height * 6 / 100}
}

// WriteASS writes cues as an ASS subtitle file in the given style.
func WriteASS(w io.Writer, cues []Cue, st Style) error {
	outline := max(st.FontSize/12, 2)
	margin := st.Width * 6 / 100
	_, err := fmt.Fprintf(w, `[Script Info]
ScriptType: v4.00+
PlayResX: %d
PlayResY: %d
WrapStyle: 0
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,DejaVu Sans,%d,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,%d,%d,2,%d,%d,%d,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`, st.Width, st.Height, st.FontSize, outline, outline/2, margin, margin, st.MarginV)
	if err != nil {
		return err
	}
	for _, c := range cues {
		if _, err := fmt.Fprintf(w, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", assTime(c.Start), assTime(c.End), escape(c.Text)); err != nil {
			return err
		}
	}
	return nil
}

// assTime formats seconds as H:MM:SS.cc.
func assTime(sec float64) string {
	cs := int(math.Round(sec * 100))
	h, cs := cs/360000, cs%360000
	m, cs := cs/6000, cs%6000
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, cs/100, cs%100)
}

// escape keeps text from being read as ASS override codes.
var escaper = strings.NewReplacer(`\`, "", "{", "(", "}", ")", "\n", " ")

func escape(s string) string { return escaper.Replace(s) }
