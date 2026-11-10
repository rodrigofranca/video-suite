package ffmpeg

import (
	"slices"
	"strings"
	"testing"
)

// These tests check the graphs as text; the tests that run FFmpeg check
// the pictures and sound they make.

func TestGraphChainsAndLabels(t *testing.T) {
	var g Graph
	src := g.Input("in.mp4", "-ss", "1.000", "-t", "3.000")
	logo := g.Input("logo.png")
	v := g.Pipe(g.Video(src), F("scale", 1080, 1920), F("setsar", 1))
	v = g.Pipe(v, F("eq", "brightness=0.1")) // continues the same chain
	mark := g.Pipe(g.Video(logo), F("format", "rgba"))
	v = g.Apply([]Stream{v, mark}, F("overlay", "x=10", "y=10"))
	silence := g.Apply(nil, F("anullsrc", "r=48000"), F("atrim", "duration="+seconds(2)))

	want := "[0:v:0]scale=1080:1920,setsar=1,eq=brightness=0.1[s1];" +
		"[1:v:0]format=rgba[s2];" +
		"[s1][s2]overlay=x=10:y=10[s3];" +
		"anullsrc=r=48000,atrim=duration=2.000[s4]"
	if got := g.String(); got != want {
		t.Fatalf("graph =\n%s\nwant\n%s", got, want)
	}
	args, err := g.Args(v, silence, g.Audio(src))
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"-ss", "1.000", "-t", "3.000", "-i", "in.mp4", "-i", "logo.png", "-filter_complex", want,
		"-map", "[s3]", "-map", "[s4]", "-map", "0:a:0?"}
	if !slices.Equal(args, wantArgs) {
		t.Fatalf("args =\n%q\nwant\n%q", args, wantArgs)
	}
}

func TestGraphSplitAndConcat(t *testing.T) {
	var g Graph
	in := g.Input("in.mp4")
	copies := g.Split(g.Pipe(g.Video(in), F("fps", 25.0)), 2)
	a := g.Pipe(copies[0], F("hflip"))
	b := g.Pipe(copies[1], F("vflip"))
	out := g.Fork([]Stream{a, b}, 1, F("concat", "n=2", "v=1", "a=0"))
	want := "[0:v:0]fps=25.000,split=2[s1][s2];[s1]hflip[s3];[s2]vflip[s4];[s3][s4]concat=n=2:v=1:a=0[s5]"
	if got := g.String(); got != want || len(out) != 1 {
		t.Fatalf("graph =\n%s\nwant\n%s", got, want)
	}
}

func TestGraphRefusesAStreamReadTwice(t *testing.T) {
	var g Graph
	in := g.Input("in.mp4")
	v := g.Pipe(g.Video(in), F("hflip"))
	g.Pipe(v, F("vflip"))
	g.Pipe(v, F("negate")) // FFmpeg would refuse it: a stage bug
	if _, err := g.Args(v); err == nil || !strings.Contains(err.Error(), "[s0] is read twice") {
		t.Fatalf("err = %v", err)
	}
}

func TestGraphWithoutFilters(t *testing.T) {
	var g Graph
	in := g.Input("in.mp4")
	args, _ := g.Args(g.Video(in), g.Audio(in))
	if want := []string{"-i", "in.mp4", "-map", "0:v:0", "-map", "0:a:0?"}; !slices.Equal(args, want) {
		t.Fatalf("args = %q", args)
	}
}

func TestEscape(t *testing.T) {
	if got := Escape(`/tmp/a:b,c'd.ass`); got != `/tmp/a\\:b\,c\\\'d.ass` {
		t.Fatalf("Escape = %s", got)
	}
}

// graphOf builds an export and returns its filter graph.
func graphOf(t *testing.T, src Source, stages ...Stage) string {
	t.Helper()
	args, err := Export{Source: src, Stages: stages, Output: "out.mp4"}.Args()
	if err != nil {
		t.Fatal(err)
	}
	return args[slices.Index(args, "-filter_complex")+1]
}

var hd = Source{Path: "in.mp4", Start: 2, Duration: 10, Width: 1920, Height: 1080, FrameRate: 25, Audio: true}

func TestFrameStage(t *testing.T) {
	cases := map[string]string{
		"original":      "[0:v:0]scale=1920:1080,setsar=1[s0]",
		"vertical":      "[0:v:0]crop=w=606:h=1080:x=657:y=0,scale=1080:1920,setsar=1[s0]",
		"vertical-face": "[0:v:0]crop=w=606:h=1080:x=657:y=0,scale=1080:1920,setsar=1[s0]", // no track: the middle
		"vertical-blur": "[0:v:0]split=2[s0][s1];" +
			"[s0]scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,gblur=sigma=30,eq=brightness=-0.08[s2];" +
			"[s1]scale=1080:1920:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[s3];" +
			"[s2][s3]overlay=(W-w)/2:(H-h)/2,setsar=1[s4]",
	}
	for format, want := range cases {
		if got := graphOf(t, hd, Frame{Format: format}); got != want {
			t.Errorf("%s:\n%s\nwant\n%s", format, got, want)
		}
	}
	if _, err := (Export{Source: hd, Stages: []Stage{Frame{Format: "square"}}}).Args(); err == nil ||
		err.Error() != `crop to 9:16: unknown clip format "square"` {
		t.Fatalf("unknown format: %v", err)
	}
	if _, err := (Export{Source: Source{Path: "in.mp4"}}).Args(); err == nil {
		t.Fatal("want an error for an unknown source size")
	}
}

func TestWatermarkAndCaptionsDrawInOrder(t *testing.T) {
	got := graphOf(t, hd, Frame{Format: "vertical"},
		Watermark{Path: "logo.png", Position: "bottom-left", Size: 20, Opacity: 80}, Captions{File: "/tmp/x:y.ass"})
	// 20% of the 1080 px side, 43 px (4%) from the left and the bottom; the
	// captions after the overlay, so they are drawn over the logo.
	want := "[0:v:0]crop=w=606:h=1080:x=657:y=0,scale=1080:1920,setsar=1[s0];" +
		"[1:v:0]format=rgba,scale=216:-1,colorchannelmixer=aa=0.80[s1];" +
		"[s0][s1]overlay=x=43:y=main_h-overlay_h-43,ass=filename=/tmp/x\\\\:y.ass[s3]"
	if got != want {
		t.Fatalf("graph =\n%s\nwant\n%s", got, want)
	}
}

func TestVignettesStage(t *testing.T) {
	src := hd
	src.Audio = false // the clip gets silence for its length
	stages := []Stage{Frame{Format: "original"},
		Vignettes{Intro: &VignetteInput{Path: "intro.mp4", Duration: 2, Audio: true}, Outro: &VignetteInput{Path: "outro.mp4", Duration: 1}}}
	args, err := Export{Source: src, Stages: stages, Output: "out.mp4"}.Args()
	if err != nil {
		t.Fatal(err)
	}
	graph := args[slices.Index(args, "-filter_complex")+1]
	for _, want := range []string{
		"[0:v:0]scale=1920:1080,setsar=1,fps=25.000,format=yuv420p[s1]",          // the clip, at its frame rate
		"anullsrc=r=48000:cl=stereo,atrim=duration=10.000,aresample=48000",       // its silence
		"[1:v:0]fps=25.000,split=2",                                              // the opening, fitted over its blur
		"[1:a:0]aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo", // with its sound
		"anullsrc=r=48000:cl=stereo,atrim=duration=1.000",                        // the closing is silent
		"concat=n=3:v=1:a=1",
	} {
		if !strings.Contains(graph, want) {
			t.Errorf("graph lacks %q:\n%s", want, graph)
		}
	}
	// Opening, clip, closing: in that order into the concat.
	concat := graph[strings.LastIndex(graph, ";")+1:]
	if !strings.HasPrefix(concat, "[s9][s10][s1][s2][s17][s18]concat") {
		t.Errorf("concat inputs = %s", concat)
	}
	if maps := strings.Join(args[slices.Index(args, "-map"):slices.Index(args, "-c:v")], " "); maps != "-map [s19] -map [s20]" {
		t.Errorf("maps = %s", maps)
	}
	if d := Describe(stages); d != "keep the shape → opening and closing" {
		t.Errorf("Describe = %q", d)
	}
}

func TestStagesWithoutVignettesKeepTheSourceSound(t *testing.T) {
	args, err := Export{Source: hd, Stages: []Stage{Frame{Format: "original"}, Captions{File: "c.ass"}}, Output: "out.mp4"}.Args()
	if err != nil {
		t.Fatal(err)
	}
	if maps := strings.Join(args[slices.Index(args, "-map"):slices.Index(args, "-c:v")], " "); maps != "-map [s1] -map 0:a:0?" {
		t.Fatalf("maps = %s", maps)
	}
}
