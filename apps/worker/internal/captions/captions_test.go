package captions

import (
	"math"
	"strings"
	"testing"
)

const vtt = "\ufeffWEBVTT\n\nNOTE made by the AI worker\n\n" +
	"00:00:01.000 --> 00:00:04.500\nHello <b>there</b>,\neveryone.\n\n" +
	"00:05.000 --> 00:09.000 align:start\nSecond cue\n\n" +
	"00:00:12.000 --> 00:00:15.000\nThird {cue} with \\N codes\n"

func TestParseVTT(t *testing.T) {
	cues, err := ParseVTT(strings.NewReader(vtt))
	if err != nil {
		t.Fatal(err)
	}
	want := []Cue{{1, 4.5, "Hello there, everyone."}, {5, 9, "Second cue"}, {12, 15, `Third {cue} with \N codes`}}
	if len(cues) != len(want) {
		t.Fatalf("cues = %+v", cues)
	}
	for i := range want {
		if cues[i] != want[i] {
			t.Errorf("cue %d = %+v, want %+v", i, cues[i], want[i])
		}
	}
	if _, err := ParseVTT(strings.NewReader("WEBVTT\n\nxx:00 --> 00:01\nhi\n")); err == nil {
		t.Error("want an error for a bad timestamp")
	}
}

func TestCut(t *testing.T) {
	cues := []Cue{{1, 4.5, "a"}, {5, 9, "b"}, {12, 15, "c"}}
	got := Cut(cues, 4.4, 12.1) // keeps 0.1 s of "a" (dropped), all of "b", 0.1 s of "c" (dropped)
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if len(got) != 1 || !near(got[0].Start, 0.6) || !near(got[0].End, 4.6) || got[0].Text != "b" {
		t.Fatalf("Cut = %+v", got)
	}
}

func TestSplit(t *testing.T) {
	got := Split([]Cue{{0, 6, "one two three four five six"}}, 10)
	texts := []string{}
	for _, c := range got {
		texts = append(texts, c.Text)
	}
	if strings.Join(texts, "|") != "one two|three four|five six" {
		t.Fatalf("chunks = %v", texts)
	}
	if got[0].Start != 0 || got[len(got)-1].End != 6 || got[1].Start != got[0].End {
		t.Fatalf("times = %+v", got)
	}
	if one := Split([]Cue{{0, 1, "short"}}, 10); len(one) != 1 || one[0] != (Cue{0, 1, "short"}) {
		t.Fatalf("short cue = %+v", one)
	}
}

func TestWriteASS(t *testing.T) {
	var b strings.Builder
	if err := WriteASS(&b, []Cue{{0.5, 61.25, `Third {cue} with \N codes`}}, StyleFor(1080, 1920)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"PlayResX: 1080\nPlayResY: 1920",
		"Style: Default,DejaVu Sans,64,",
		",2,64,64,422,1\n", // bottom center, side margins, raised above app buttons
		"Dialogue: 0,0:00:00.50,0:01:01.25,Default,,0,0,0,,Third (cue) with N codes\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ASS missing %q:\n%s", want, out)
		}
	}
	if st := StyleFor(1920, 1080); st.FontSize != 52 || st.MarginV != 64 {
		t.Errorf("landscape style = %+v", st)
	}
}
