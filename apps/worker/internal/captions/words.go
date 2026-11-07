package captions

import (
	"encoding/json"
	"fmt"
	"io"
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

// highlight is the color of the word being said (ASS colors are BGR): yellow.
const highlight = `{\c&H00E6FF&}`

// lingerSeconds is how long a phrase stays after its last word when the
// next one does not start sooner.
const lingerSeconds = 0.4

// WriteASSWords writes phrases as an ASS file with one event per word: the
// whole phrase, with the word being said highlighted, from the moment it is
// said until the next word starts.
func WriteASSWords(w io.Writer, phrases [][]Word, st Style) error {
	if err := writeHeader(w, st); err != nil {
		return err
	}
	for p, phrase := range phrases {
		end := phrase[len(phrase)-1].End + lingerSeconds
		if p+1 < len(phrases) {
			end = min(end, phrases[p+1][0].Start)
		}
		end = max(end, phrase[len(phrase)-1].End)
		for i, word := range phrase {
			from, to := word.Start, end
			if i+1 < len(phrase) {
				to = phrase[i+1].Start
			}
			if to-from < 0.01 {
				continue // a word with no time of its own: the next event shows it
			}
			parts := make([]string, len(phrase))
			for j, other := range phrase {
				parts[j] = escape(other.Text)
				if j == i {
					parts[j] = highlight + parts[j] + `{\r}`
				}
			}
			if _, err := fmt.Fprintf(w, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", assTime(from), assTime(to), strings.Join(parts, " ")); err != nil {
				return err
			}
		}
	}
	return nil
}
