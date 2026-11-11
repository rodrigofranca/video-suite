// Package jumpcut finds what a clip keeps with jump cuts (docs/adr/0040):
// the speech, without the pauses between words and without filler sounds,
// from the transcript's word timings.
package jumpcut

import (
	"strings"
	"unicode"

	"ultramedia/worker/internal/captions"
)

// Span is a part of the clip, in seconds from the clip's start.
type Span struct {
	Start, End float64
}

// Room kept around each word, and the shortest pause that is cut: speech
// starts a little before a word's timing and fades a little after it, and
// a shorter pause is the rhythm of the sentence. A pause longer than
// Lead+Tail+MinPause comes out as Lead+Tail of quiet.
const (
	Lead     = 0.10
	Tail     = 0.15
	MinPause = 0.30
)

// Plan is what a clip keeps.
type Plan struct {
	Keep    []Span
	Fillers int     // filler words left out
	Length  float64 // seconds kept
}

// Cut returns the spans of a clip of the given length to keep: each word,
// with Lead before and Tail after, joined to the next when the pause between
// them is shorter than MinPause. Words are relative to the clip's start (as
// captions.CutWords gives them). Fillers are left out. Without words, it
// keeps nothing, and the caller exports the clip whole.
func Cut(words []captions.Word, length float64) Plan {
	var p Plan
	for _, w := range words {
		if IsFiller(w.Text) {
			p.Fillers++
			continue
		}
		s := Span{Start: max(w.Start-Lead, 0), End: min(max(w.End, w.Start)+Tail, length)}
		if s.End <= s.Start {
			continue
		}
		if n := len(p.Keep); n > 0 && s.Start-p.Keep[n-1].End < MinPause {
			p.Keep[n-1].End = max(p.Keep[n-1].End, s.End)
			continue
		}
		p.Keep = append(p.Keep, s)
	}
	for _, s := range p.Keep {
		p.Length += s.End - s.Start
	}
	return p
}

// fillers are sounds that carry no words in any of the transcripts'
// languages, after IsFiller has lowered them and squeezed repeated letters
// ("Ummm" is "um"). Words that mean something somewhere stay out: "er" is
// "he" in German, "em" is "in" in Portuguese, "é" is "is", and "like",
// "tipo" or "you know" are words.
var fillers = map[string]bool{
	"um": true, "uh": true, "uhm": true, "umh": true, "erm": true, "ehm": true, "eh": true, "ah": true,
	"hm": true, "mhm": true, "m": true, "hum": true, "ahn": true,
	"ã": true, "ãh": true, "hã": true, "éh": true, // Portuguese
	"äh": true, "ähm": true, // German
	"euh": true, // French
}

// IsFiller tells a filler sound ("uh", "Ummm…", "ãh") from a word.
func IsFiller(word string) bool {
	var b strings.Builder
	var last rune
	for _, r := range strings.ToLower(word) {
		if !unicode.IsLetter(r) || r == last {
			continue
		}
		b.WriteRune(r)
		last = r
	}
	return fillers[b.String()]
}
