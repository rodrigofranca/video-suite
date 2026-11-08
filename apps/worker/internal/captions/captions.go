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
// buttons that social apps draw at the bottom. With applies a person's
// choices: another position, an accent color, capitals.
type Style struct {
	Width, Height int
	FontSize      int
	MarginV       int    // distance from the bottom edge (from the top edge at the top)
	Align         int    // ASS alignment: 2 bottom center (also when 0), 5 middle, 8 top
	Accent        string // ASS color of the highlighted word, the box and the karaoke fill
	Caps          bool   // all capitals
}

// Options are the choices made for one export (docs/adr/0025).
type Options struct {
	Position string // "bottom" (or empty), "middle" or "top"
	Color    string // see Colors; empty for the style's own
	Caps     bool
}

// Colors are the accent colors on offer, as ASS colors (&HAABBGGRR).
var Colors = map[string]string{
	"yellow": "&H0000E6FF", // #FFE600
	"green":  "&H005EE83C", // #3CE85E
	"pink":   "&H00B84FFF", // #FF4FB8
	"blue":   "&H00FFB533", // #33B5FF
	"violet": "&H00F65C8B", // #8B5CF6
}

// With returns the style moved and colored as chosen. defaultColor is the
// style's own accent, used when no color is chosen.
func (st Style) With(o Options, defaultColor string) Style {
	switch o.Position {
	case "top":
		// Clear of the bars apps draw at the top of vertical video.
		st.Align, st.MarginV = 8, st.Height*6/100
		if st.Height > st.Width {
			st.MarginV = st.Height * 14 / 100
		}
	case "middle":
		st.Align, st.MarginV = 5, 0
	default:
		st.Align = 2
	}
	st.Accent = Colors[defaultColor]
	if c, ok := Colors[o.Color]; ok {
		st.Accent = c
	}
	st.Caps = o.Caps
	return st
}

// ColorName is the name of the accent color the style ends up with, or "".
func (st Style) ColorName() string {
	for name, c := range Colors {
		if c == st.Accent {
			return name
		}
	}
	return ""
}

func (st Style) align() int {
	if st.Align == 0 {
		return 2
	}
	return st.Align
}

// text escapes a caption and puts it in capitals when chosen.
func (st Style) text(s string) string {
	if st.Caps {
		s = strings.ToUpper(s)
	}
	return escape(s)
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
	if err := writeHeader(w, st); err != nil {
		return err
	}
	for _, c := range cues {
		if _, err := fmt.Fprintf(w, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", assTime(c.Start), assTime(c.End), st.text(c.Text)); err != nil {
			return err
		}
	}
	return nil
}

// writeHeader writes the ASS script info, the Default style (bold white
// DejaVu Sans with a dark outline, at the style's alignment) and any extra
// style lines.
func writeHeader(w io.Writer, st Style, extraStyles ...string) error {
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
Style: Default,DejaVu Sans,%d,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,%d,%d,%d,%d,%d,%d,1
%s
[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`, st.Width, st.Height, st.FontSize, outline, outline/2, st.align(), margin, margin, st.MarginV, extraLines(extraStyles))
	return err
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

func extraLines(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	return b.String()
}
