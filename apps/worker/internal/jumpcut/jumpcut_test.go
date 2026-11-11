package jumpcut

import (
	"math"
	"testing"

	"ultramedia/worker/internal/captions"
)

func w(start, end float64, text string) captions.Word {
	return captions.Word{Start: start, End: end, Text: text}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCut(t *testing.T) {
	words := []captions.Word{
		w(0.5, 0.9, "So,"),
		w(1.0, 1.4, "today"), // 0.1 s after: same breath
		w(3.0, 3.3, "uhm…"),  // a filler, left out
		w(4.0, 4.5, "we"),
		w(4.55, 5.0, "start."),
		w(9.8, 10.0, "Done"), // its tail would go past the end
	}
	p := Cut(words, 10)
	want := []Span{{0.4, 1.55}, {3.9, 5.15}, {9.7, 10}}
	if len(p.Keep) != len(want) {
		t.Fatalf("keep = %+v, want %+v", p.Keep, want)
	}
	for i := range want {
		if !near(p.Keep[i].Start, want[i].Start) || !near(p.Keep[i].End, want[i].End) {
			t.Fatalf("keep = %+v, want %+v", p.Keep, want)
		}
	}
	if p.Fillers != 1 || !near(p.Length, 1.15+1.25+0.3) {
		t.Fatalf("fillers %d, length %v", p.Fillers, p.Length)
	}
}

func TestShortPausesStay(t *testing.T) {
	// 0.5 s between the words: 0.25 s of it is room around them, and the
	// 0.25 s left is under MinPause, so the sentence keeps its rhythm.
	p := Cut([]captions.Word{w(1, 1.5, "one"), w(2, 2.5, "two")}, 5)
	if len(p.Keep) != 1 || !near(p.Keep[0].Start, 0.9) || !near(p.Keep[0].End, 2.65) {
		t.Fatalf("keep = %+v", p.Keep)
	}
	// 0.6 s: cut down to the room around the words.
	p = Cut([]captions.Word{w(1, 1.5, "one"), w(2.1, 2.5, "two")}, 5)
	if len(p.Keep) != 2 {
		t.Fatalf("keep = %+v", p.Keep)
	}
}

func TestNothingToKeep(t *testing.T) {
	if p := Cut(nil, 10); len(p.Keep) != 0 || p.Length != 0 {
		t.Fatalf("no words: %+v", p)
	}
	if p := Cut([]captions.Word{w(1, 2, "Uhh")}, 10); len(p.Keep) != 0 || p.Fillers != 1 {
		t.Fatalf("only a filler: %+v", p)
	}
}

func TestIsFiller(t *testing.T) {
	for _, s := range []string{"uh", "Uhm,", "Ummm…", "hmm", "Mhm.", "ãh", "Hã?", "ähm", "euh", "eh!"} {
		if !IsFiller(s) {
			t.Errorf("%q is a filler", s)
		}
	}
	for _, s := range []string{"é", "em", "er", "like", "tipo", "a", "e", "home", "humano", "ahead"} {
		if IsFiller(s) {
			t.Errorf("%q is a word", s)
		}
	}
}
