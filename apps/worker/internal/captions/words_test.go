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

func TestWriteASSWords(t *testing.T) {
	phrases := [][]Word{
		{{0, 0.4, "Hello"}, {0.5, 0.9, "{world}"}},
		{{1.0, 1.4, "Next"}},
	}
	var b strings.Builder
	if err := WriteASSWords(&b, phrases, WordStyleFor(1080, 1920)); err != nil {
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
