package captions

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

// Word is one spoken word and when it was said, in seconds, as stored in
// the transcript (transcripts.words).
type Word struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"word"`
}

// ParseWords reads a transcript's words.
func ParseWords(raw []byte) ([]Word, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var words []Word
	if err := json.Unmarshal(raw, &words); err != nil {
		return nil, fmt.Errorf("read transcript words: %w", err)
	}
	return words, nil
}

// CutWords returns the words said between start and end, with times
// relative to start. A word at an edge is kept when its middle is inside.
func CutWords(words []Word, start, end float64) []Word {
	var out []Word
	for _, w := range words {
		mid := (w.Start + w.End) / 2
		if strings.TrimSpace(w.Text) == "" || mid < start || mid > end {
			continue
		}
		out = append(out, Word{
			Start: max(w.Start, start) - start,
			End:   min(max(w.End, w.Start), end) - start,
			Text:  strings.TrimSpace(w.Text),
		})
	}
	return out
}

// maxPause is the longest silence inside a phrase, in seconds.
const maxPause = 0.8

// Phrases groups words into the short phrases shown together: at most
// maxWords words and maxChars characters, broken at sentence ends and pauses.
func Phrases(words []Word, maxWords, maxChars int) [][]Word {
	var phrases [][]Word
	var cur []Word
	chars := 0
	for _, w := range words {
		if len(cur) > 0 {
			prev := cur[len(cur)-1]
			if len(cur) >= maxWords || chars+1+len(w.Text) > maxChars ||
				w.Start-prev.End > maxPause || strings.ContainsAny(prev.Text[len(prev.Text)-1:], ".!?…") {
				phrases = append(phrases, cur)
				cur, chars = nil, 0
			}
		}
		if len(cur) > 0 {
			chars++ // the space
		}
		cur = append(cur, w)
		chars += len(w.Text)
	}
	if len(cur) > 0 {
		phrases = append(phrases, cur)
	}
	return phrases
}

// WordStyleFor returns the style of word-by-word captions: bigger than line
// captions, and higher on vertical video, where social apps put them.
func WordStyleFor(width, height int) Style {
	if height > width { // vertical
		return Style{Width: width, Height: height, FontSize: width * 80 / 1080, MarginV: height * 28 / 100}
	}
	return Style{Width: width, Height: height, FontSize: max(height*64/1080, 18), MarginV: height * 8 / 100}
}

// Text colors (ASS writes them as &HAABBGGRR). The accent comes from the
// style: yellow by default, violet for the box (see Style.With).
const (
	white = "&H00FFFFFF"
	black = "&H00000000"
)

// accent is the style's accent, yellow when none was set.
func (st Style) accent(fallback string) string {
	if st.Accent != "" {
		return st.Accent
	}
	return Colors[fallback]
}

// inline turns an ASS style color (&H00BBGGRR) into an override tag color (&HBBGGRR&).
func inline(color string) string { return "&H" + strings.TrimPrefix(color, "&H00") + "&" }

// textOn is the text color that reads on a box of the given color: black on
// light colors, white on dark ones.
func textOn(color string) string {
	var b, g, r int
	fmt.Sscanf(strings.TrimPrefix(color, "&H00"), "%02X%02X%02X", &b, &g, &r)
	if 0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(b) > 150 {
		return black
	}
	return white
}

// lingerSeconds is how long a phrase stays after its last word when the
// next one does not start sooner.
const lingerSeconds = 0.4

// phraseEnd is when a phrase leaves the screen: a little after its last
// word, but never after the next phrase starts.
func phraseEnd(phrases [][]Word, p int) float64 {
	last := phrases[p][len(phrases[p])-1].End
	end := last + lingerSeconds
	if p+1 < len(phrases) {
		end = min(end, phrases[p+1][0].Start)
	}
	return max(end, last)
}

// wordWindows calls fn for each word of a phrase with when it is the word
// being said: from its start until the next word starts, the last one until
// the phrase ends. Words with no time of their own are skipped.
func wordWindows(phrase []Word, end float64, fn func(i int, from, to float64) error) error {
	for i, word := range phrase {
		to := end
		if i+1 < len(phrase) {
			to = phrase[i+1].Start
		}
		if to-word.Start < 0.01 {
			continue
		}
		if err := fn(i, word.Start, to); err != nil {
			return err
		}
	}
	return nil
}

func dialogue(w io.Writer, layer int, from, to float64, style, text string) error {
	_, err := fmt.Fprintf(w, "Dialogue: %d,%s,%s,%s,,0,0,0,,%s\n", layer, assTime(from), assTime(to), style, text)
	return err
}

// WriteHighlight writes the "highlight" style: the phrase in white, with the
// word being said in the accent color (yellow by default). One event per word.
func WriteHighlight(w io.Writer, phrases [][]Word, st Style) error {
	if err := writeHeader(w, st); err != nil {
		return err
	}
	for p, phrase := range phrases {
		err := wordWindows(phrase, phraseEnd(phrases, p), func(i int, from, to float64) error {
			parts := make([]string, len(phrase))
			for j, other := range phrase {
				parts[j] = st.text(other.Text)
				if j == i {
					parts[j] = `{\c` + inline(st.accent("yellow")) + `}` + parts[j] + `{\r}`
				}
			}
			return dialogue(w, 0, from, to, "Default", strings.Join(parts, " "))
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteBox writes the "box" style: the phrase in white, with a box in the
// accent color (violet by default) behind the word being said, the word in
// black or white, whichever reads on it. The box is a second layer with the
// same text, where every other word is transparent, in an opaque-box style.
func WriteBox(w io.Writer, phrases [][]Word, st Style) error {
	pad := max(st.FontSize/6, 4)
	margin := st.Width * 6 / 100
	color := st.accent("violet")
	box := fmt.Sprintf("Style: Box,DejaVu Sans,%d,%s,%s,%s,%s,-1,0,0,0,100,100,0,0,3,%d,0,%d,%d,%d,%d,1",
		st.FontSize, textOn(color), textOn(color), color, color, pad, st.align(), margin, margin, st.MarginV)
	if err := writeHeader(w, st, box); err != nil {
		return err
	}
	for p, phrase := range phrases {
		end := phraseEnd(phrases, p)
		texts := make([]string, len(phrase))
		for j, word := range phrase {
			texts[j] = st.text(word.Text)
		}
		if err := dialogue(w, 0, phrase[0].Start, end, "Default", strings.Join(texts, " ")); err != nil {
			return err
		}
		err := wordWindows(phrase, end, func(i int, from, to float64) error {
			var b strings.Builder
			for j, text := range texts {
				if j > 0 {
					b.WriteString(`{\alpha&HFF&} `)
				}
				if j == i {
					b.WriteString(`{\alpha&H00&}` + text)
				} else {
					b.WriteString(`{\alpha&HFF&}` + text)
				}
			}
			return dialogue(w, 1, from, to, "Box", b.String())
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteKaraoke writes the "karaoke" style: the phrase appears in white and
// each word fills with the accent color (yellow by default), left to right,
// while it is said. It uses the ASS karaoke tag \kf, one event per phrase.
func WriteKaraoke(w io.Writer, phrases [][]Word, st Style) error {
	outline := max(st.FontSize/12, 2)
	margin := st.Width * 6 / 100
	// Karaoke fills from the secondary color (not yet said) to the primary.
	kara := fmt.Sprintf("Style: Karaoke,DejaVu Sans,%d,%s,%s,%s,&H80000000,-1,0,0,0,100,100,0,0,1,%d,%d,%d,%d,%d,%d,1",
		st.FontSize, st.accent("yellow"), white, black, outline, outline/2, st.align(), margin, margin, st.MarginV)
	if err := writeHeader(w, st, kara); err != nil {
		return err
	}
	for p, phrase := range phrases {
		var b strings.Builder
		for i, word := range phrase {
			until := word.End
			if i+1 < len(phrase) {
				until = phrase[i+1].Start
			}
			if i > 0 {
				b.WriteString(" ")
			}
			fmt.Fprintf(&b, `{\kf%d}%s`, max(int(math.Round((until-word.Start)*100)), 1), st.text(word.Text))
		}
		if err := dialogue(w, 0, phrase[0].Start, phraseEnd(phrases, p), "Karaoke", b.String()); err != nil {
			return err
		}
	}
	return nil
}

// oneWordMaxChars is the longest word shown at full size; longer words are
// shrunk to fit the width.
const oneWordMaxChars = 10

// OneWordStyleFor returns the style of the "one-word" captions: one big word
// at a time, in the lower middle of the picture.
func OneWordStyleFor(width, height int) Style {
	if height > width { // vertical
		return Style{Width: width, Height: height, FontSize: width * 110 / 1080, MarginV: height * 35 / 100}
	}
	return Style{Width: width, Height: height, FontSize: max(height*90/1080, 24), MarginV: height * 12 / 100}
}

// WriteOneWord writes the "one-word" style: only the word being said (in
// capitals unless turned off), with a short pop as it appears.
func WriteOneWord(w io.Writer, words []Word, st Style) error {
	if err := writeHeader(w, st); err != nil {
		return err
	}
	for i, word := range words {
		to := word.End + lingerSeconds
		if i+1 < len(words) {
			to = min(to, words[i+1].Start)
		}
		to = max(to, word.End)
		if to-word.Start < 0.01 {
			continue
		}
		text := st.text(word.Text)
		size := ""
		if n := len([]rune(text)); n > oneWordMaxChars {
			size = fmt.Sprintf(`\fs%d`, st.FontSize*oneWordMaxChars/n)
		}
		pop := `{` + size + `\fscx115\fscy115\t(0,120,\fscx100\fscy100)}`
		if err := dialogue(w, 0, word.Start, to, "Default", pop+text); err != nil {
			return err
		}
	}
	return nil
}
