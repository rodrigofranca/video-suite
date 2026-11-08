package captions

import (
	"strings"
	"testing"
)

func texts(phrases [][]Word) string {
	var out []string
	for _, p := range phrases {
		var ws []string
		for _, w := range p {
			ws = append(ws, w.Text)
		}
		out = append(out, strings.Join(ws, " "))
	}
	return strings.Join(out, " | ")
}

func TestParseAndCutWords(t *testing.T) {
	words, err := ParseWords([]byte(`[{"start":0.5,"end":0.9,"word":"Before"},{"start":0.95,"end":1.4,"word":" edge"},` +
		`{"start":2,"end":2.3,"word":"inside"},{"start":4.8,"end":5.4,"word":"late"},{"start":3,"end":3.1,"word":" "}]`))
	if err != nil || len(words) != 5 {
		t.Fatalf("ParseWords = %v, %v", words, err)
	}
	got := CutWords(words, 1, 5)
	if texts([][]Word{got}) != "edge inside" || got[0].Start != 0 || got[0].End < 0.39 || got[1].Start != 1 {
		t.Fatalf("CutWords = %+v", got)
	}
	if w, err := ParseWords(nil); w != nil || err != nil {
		t.Fatalf("no words: %v %v", w, err)
	}
	if _, err := ParseWords([]byte("{")); err == nil {
		t.Fatal("want an error for bad JSON")
	}
}

func TestPhrases(t *testing.T) {
	at := 0.0
	var words []Word
	for _, s := range strings.Fields("This is how streaming works. The player downloads a playlist first then segments") {
		words = append(words, Word{Start: at, End: at + 0.3, Text: s})
		at += 0.35
	}
	for i := 10; i < len(words); i++ { // a long pause before "first"
		words[i].Start += 1.5
		words[i].End += 1.5
	}
	got := texts(Phrases(words, 4, 22))
	want := "This is how streaming | works. | The player downloads a | playlist | first then segments"
	if got != want {
		t.Fatalf("Phrases =\n%s\nwant\n%s", got, want)
	}
	if got := texts(Phrases(words[:3], 4, 8)); got != "This is | how" {
		t.Fatalf("by length: %s", got)
	}
}

func TestWriteHighlight(t *testing.T) {
	phrases := [][]Word{
		{{0, 0.4, "Hello"}, {0.5, 0.9, "{world}"}},
		{{1.0, 1.4, "Next"}},
	}
	var b strings.Builder
	if err := WriteHighlight(&b, phrases, WordStyleFor(1080, 1920)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"Style: Default,DejaVu Sans,80,",
		",2,64,64,537,1\n",
		// The phrase stays up until the next one starts; each word lights up in turn.
		"Dialogue: 0,0:00:00.00,0:00:00.50,Default,,0,0,0,,{\\c&H00E6FF&}Hello{\\r} (world)\n",
		"Dialogue: 0,0:00:00.50,0:00:01.00,Default,,0,0,0,,Hello {\\c&H00E6FF&}(world){\\r}\n",
		// The last phrase lingers after its last word.
		"Dialogue: 0,0:00:01.00,0:00:01.80,Default,,0,0,0,,{\\c&H00E6FF&}Next{\\r}\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ASS missing %q:\n%s", want, out)
		}
	}
	if st := WordStyleFor(1280, 720); st.FontSize != 42 || st.MarginV != 57 {
		t.Errorf("landscape word style = %+v", st)
	}
}

func TestWriteBox(t *testing.T) {
	var b strings.Builder
	phrases := [][]Word{{{0, 0.4, "Hello"}, {0.5, 0.9, "world"}}}
	if err := WriteBox(&b, phrases, WordStyleFor(1080, 1920)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		// An opaque-box style (BorderStyle 3) whose box is violet.
		"Style: Box,DejaVu Sans,80,&H00FFFFFF,&H00FFFFFF,&H00F65C8B,&H00F65C8B,-1,0,0,0,100,100,0,0,3,13,0,2,64,64,537,1\n",
		// The whole phrase underneath...
		"Dialogue: 0,0:00:00.00,0:00:01.30,Default,,0,0,0,,Hello world\n",
		// ...and above it, only the word being said, in its box.
		"Dialogue: 1,0:00:00.00,0:00:00.50,Box,,0,0,0,,{\\alpha&H00&}Hello{\\alpha&HFF&} {\\alpha&HFF&}world\n",
		"Dialogue: 1,0:00:00.50,0:00:01.30,Box,,0,0,0,,{\\alpha&HFF&}Hello{\\alpha&HFF&} {\\alpha&H00&}world\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ASS missing %q:\n%s", want, out)
		}
	}
}

func TestWriteKaraoke(t *testing.T) {
	var b strings.Builder
	phrases := [][]Word{{{1.0, 1.4, "Sing"}, {1.6, 2.25, "along"}}}
	if err := WriteKaraoke(&b, phrases, WordStyleFor(1080, 1920)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	// Yellow once said, white before; each word fills until the next starts.
	for _, want := range []string{
		"Style: Karaoke,DejaVu Sans,80,&H0000E6FF,&H00FFFFFF,&H00000000,",
		"Dialogue: 0,0:00:01.00,0:00:02.65,Karaoke,,0,0,0,,{\\kf60}Sing {\\kf65}along\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ASS missing %q:\n%s", want, out)
		}
	}
}

func TestWriteOneWord(t *testing.T) {
	var b strings.Builder
	words := []Word{{0, 0.3, "fast"}, {0.35, 0.8, "transcription,"}, {2, 2.2, "ok"}}
	if err := WriteOneWord(&b, words, OneWordStyleFor(1080, 1920).With(Options{Caps: true}, "")); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"Style: Default,DejaVu Sans,110,",
		",2,64,64,672,1\n",
		// Capitals, a pop, and each word until the next one starts.
		"Dialogue: 0,0:00:00.00,0:00:00.35,Default,,0,0,0,,{\\fscx115\\fscy115\\t(0,120,\\fscx100\\fscy100)}FAST\n",
		// A long word is shrunk to fit; after it, a pause: it stays a moment, then goes.
		"Dialogue: 0,0:00:00.35,0:00:01.20,Default,,0,0,0,,{\\fs78\\fscx115\\fscy115\\t(0,120,\\fscx100\\fscy100)}TRANSCRIPTION,\n",
		"Dialogue: 0,0:00:02.00,0:00:02.60,Default,,0,0,0,,{\\fscx115\\fscy115\\t(0,120,\\fscx100\\fscy100)}OK\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ASS missing %q:\n%s", want, out)
		}
	}
}

func TestOptions(t *testing.T) {
	words := []Word{{0, 0.3, "Hello"}, {0.4, 0.8, "world"}}
	phrases := [][]Word{words}
	write := func(fn func(*strings.Builder, Style) error, st Style) string {
		t.Helper()
		var b strings.Builder
		if err := fn(&b, st); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	vertical := WordStyleFor(1080, 1920)

	// Top: alignment 8, clear of the bars at the top of vertical video; middle: 5.
	top := write(func(b *strings.Builder, st Style) error { return WriteHighlight(b, phrases, st) }, vertical.With(Options{Position: "top"}, "yellow"))
	if !strings.Contains(top, ",1,6,3,8,64,64,268,1\n") {
		t.Errorf("top:\n%s", top)
	}
	middle := write(func(b *strings.Builder, st Style) error { return WriteKaraoke(b, phrases, st) }, vertical.With(Options{Position: "middle"}, "yellow"))
	if !strings.Contains(middle, "Style: Karaoke,DejaVu Sans,80,&H0000E6FF,&H00FFFFFF,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,6,3,5,64,64,0,1") {
		t.Errorf("middle karaoke:\n%s", middle)
	}

	// The accent colors the highlighted word, the karaoke fill and the box;
	// on a light box the word is black, on a dark one white.
	green := write(func(b *strings.Builder, st Style) error { return WriteHighlight(b, phrases, st) }, vertical.With(Options{Color: "green"}, "yellow"))
	if !strings.Contains(green, `{\c&H5EE83C&}Hello{\r} world`) {
		t.Errorf("green highlight:\n%s", green)
	}
	yellowBox := write(func(b *strings.Builder, st Style) error { return WriteBox(b, phrases, st) }, vertical.With(Options{Color: "yellow"}, "violet"))
	if !strings.Contains(yellowBox, "Style: Box,DejaVu Sans,80,&H00000000,&H00000000,&H0000E6FF,&H0000E6FF,") {
		t.Errorf("yellow box:\n%s", yellowBox)
	}
	pinkKaraoke := write(func(b *strings.Builder, st Style) error { return WriteKaraoke(b, phrases, st) }, vertical.With(Options{Color: "pink"}, "yellow"))
	if !strings.Contains(pinkKaraoke, "Style: Karaoke,DejaVu Sans,80,&H00B84FFF,") {
		t.Errorf("pink karaoke:\n%s", pinkKaraoke)
	}
	if st := vertical.With(Options{}, "violet"); st.ColorName() != "violet" || textOn(st.Accent) != white {
		t.Errorf("default box color = %q, text %s", st.ColorName(), textOn(st.Accent))
	}
	if st := vertical.With(Options{Color: "nope"}, "yellow"); st.ColorName() != "yellow" {
		t.Errorf("unknown color = %q, want the style's own", st.ColorName())
	}

	// Capitals for every style, lines included.
	caps := write(func(b *strings.Builder, st Style) error { return WriteKaraoke(b, phrases, st) }, vertical.With(Options{Caps: true}, "yellow"))
	if !strings.Contains(caps, `{\kf40}HELLO {\kf40}WORLD`) {
		t.Errorf("karaoke in capitals:\n%s", caps)
	}
	var lines strings.Builder
	WriteASS(&lines, []Cue{{0, 1, "Hello world"}}, StyleFor(1920, 1080).With(Options{Caps: true, Position: "top"}, ""))
	if !strings.Contains(lines.String(), ",Default,,0,0,0,,HELLO WORLD\n") || !strings.Contains(lines.String(), ",8,115,115,64,1\n") {
		t.Errorf("lines in capitals at the top:\n%s", lines.String())
	}
	var lower strings.Builder
	WriteOneWord(&lower, words, OneWordStyleFor(1080, 1920))
	if !strings.Contains(lower.String(), "}Hello\n") {
		t.Errorf("one word without capitals:\n%s", lower.String())
	}
}
