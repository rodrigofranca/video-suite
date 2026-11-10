package ffmpeg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The export stages (docs/adr/0035), in the order the export job uses them.

// Vertical clips are 1080x1920 (9:16), the size social apps expect.
const (
	VerticalWidth  = 1080
	VerticalHeight = 1920
	// maxSide is the longest side of a clip that keeps the video's shape.
	maxSide = 1920
)

// ClipSize returns the width and height of a clip exported in a format.
func ClipSize(format string, srcW, srcH int) (int, int) {
	if format != "original" {
		return VerticalWidth, VerticalHeight
	}
	w, h := srcW, srcH
	if long := max(w, h); long > maxSide {
		w, h = w*maxSide/long, h*maxSide/long
	}
	return even(w), even(h)
}

// ---- Frame ----

// Frame gives the picture the format's shape and size (docs/adr/0020, 0022):
//   - original: the video's shape, up to 1920 px on the longer side;
//   - vertical: the middle of the frame, 9:16;
//   - vertical-face: 9:16 following Track (the middle without one);
//   - vertical-blur: the whole frame over a blurred, zoomed copy of itself.
type Frame struct {
	Format string
	Track  []TrackPoint
}

// TrackPoint is the centre of the 9:16 window at a time of the clip.
type TrackPoint struct {
	T float64 `json:"t"` // seconds from the clip start
	X float64 `json:"x"` // share of the source width
}

func (f Frame) String() string {
	switch f.Format {
	case "original":
		return "keep the shape"
	case "vertical-face":
		if len(f.Track) > 0 {
			return "crop to 9:16 following the face"
		}
		return "crop to 9:16"
	case "vertical-blur":
		return "fit into 9:16 over a blur"
	}
	return "crop to 9:16"
}

func (f Frame) Apply(g *Graph, m Media) (Media, error) {
	w, h := ClipSize(f.Format, m.Width, m.Height)
	switch f.Format {
	case "original":
		m.Video = g.Pipe(m.Video, F("scale", w, h), F("setsar", 1))
	case "vertical", "vertical-face":
		// As wide as 9:16 allows: the middle of the frame, or following the track.
		cw := even(min(m.Width, m.Height*9/16))
		ch := even(min(m.Height, m.Width*16/9))
		x := strconv.Itoa((m.Width - cw) / 2)
		if f.Format == "vertical-face" && len(f.Track) > 0 {
			x = "'" + trackExpr(f.Track, m.Width, cw) + "'"
		}
		m.Video = g.Pipe(m.Video, F("crop", fmt.Sprintf("w=%d", cw), fmt.Sprintf("h=%d", ch), "x="+x, fmt.Sprintf("y=%d", (m.Height-ch)/2)),
			F("scale", w, h), F("setsar", 1))
	case "vertical-blur":
		m.Video = fitOverBlur(g, m.Video, w, h)
	default:
		return m, fmt.Errorf("unknown clip format %q", f.Format)
	}
	m.Width, m.Height = w, h
	return m, nil
}

// fitOverBlur fits a picture whole into w x h, centred over a blurred,
// darker copy of itself that fills the frame.
func fitOverBlur(g *Graph, in Stream, w, h int) Stream {
	copies := g.Split(in, 2)
	bg := g.Pipe(copies[0], F("scale", w, h, "force_original_aspect_ratio=increase"), F("crop", w, h),
		F("gblur", "sigma=30"), F("eq", "brightness=-0.08"))
	fg := g.Pipe(copies[1], F("scale", w, h, "force_original_aspect_ratio=decrease", "force_divisible_by=2"), F("setsar", 1))
	return g.Apply([]Stream{bg, fg}, F("overlay", "(W-w)/2", "(H-h)/2"), F("setsar", 1))
}

// trackExpr returns an FFmpeg expression of t (seconds) for the crop's left
// edge, in pixels: straight lines between the track's points, held before
// the first and after the last, always inside the frame. FFmpeg evaluates it
// for every frame.
func trackExpr(points []TrackPoint, srcW, cropW int) string {
	px := func(x float64) float64 {
		return min(max(math.Round(x*float64(srcW)-float64(cropW)/2), 0), float64(srcW-cropW))
	}
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
	first, last := points[0], points[len(points)-1]
	if len(points) == 1 {
		return f(px(first.X))
	}
	terms := []string{"lt(t," + f(first.T) + ")*" + f(px(first.X))}
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		if b.T <= a.T {
			continue
		}
		slope := (px(b.X) - px(a.X)) / (b.T - a.T)
		terms = append(terms, fmt.Sprintf("gte(t,%s)*lt(t,%s)*(%s+(t-%s)*%s)", f(a.T), f(b.T), f(px(a.X)), f(a.T), f(slope)))
	}
	terms = append(terms, "gte(t,"+f(last.T)+")*"+f(px(last.X)))
	return strings.Join(terms, "+")
}

// ---- Watermark ----

// Watermark lays an image over the picture (docs/adr/0031): in a corner
// ("top-right", "top-left", "bottom-right", "bottom-left"), as wide as Size
// percent of the shorter side, at Opacity percent, a margin of 4% of the
// shorter side from the edges. It goes before the captions, which stay
// readable over it.
type Watermark struct {
	Path     string
	Position string
	Size     int
	Opacity  int
}

func (wm Watermark) String() string { return "watermark" }

func (wm Watermark) Apply(g *Graph, m Media) (Media, error) {
	short := min(m.Width, m.Height)
	margin := short * 4 / 100
	x, y := strconv.Itoa(margin), strconv.Itoa(margin)
	if strings.HasSuffix(wm.Position, "right") {
		x = fmt.Sprintf("main_w-overlay_w-%d", margin)
	}
	if strings.HasPrefix(wm.Position, "bottom") {
		y = fmt.Sprintf("main_h-overlay_h-%d", margin)
	}
	logo := g.Pipe(g.Video(g.Input(wm.Path)), F("format", "rgba"), F("scale", even(short*wm.Size/100), -1),
		F("colorchannelmixer", fmt.Sprintf("aa=%.2f", float64(wm.Opacity)/100)))
	m.Video = g.Apply([]Stream{m.Video, logo}, F("overlay", "x="+x, "y="+y))
	return m, nil
}

// ---- Captions ----

// Captions burns an ASS subtitle file into the picture (docs/adr/0021).
type Captions struct {
	File string
}

func (c Captions) String() string { return "captions" }

func (c Captions) Apply(g *Graph, m Media) (Media, error) {
	m.Video = g.Pipe(m.Video, F("ass", "filename="+Escape(c.File)))
	return m, nil
}

// ---- Vignettes ----

// Vignettes joins an opening before the clip and a closing after it
// (docs/adr/0030); either may be nil. Each is fitted whole into the clip's
// size over a blurred copy of itself, at the clip's frame rate (30 when
// unknown), with its sound or silence. The clip gets silence too when the
// source has none. It changes the clip's length, so it goes last.
type Vignettes struct {
	Intro, Outro *VignetteInput
}

// VignetteInput is an opening or closing to join to a clip.
type VignetteInput struct {
	Path     string
	Duration float64 // seconds
	Audio    bool
}

func (v Vignettes) String() string {
	switch {
	case v.Intro != nil && v.Outro != nil:
		return "opening and closing"
	case v.Intro != nil:
		return "opening"
	}
	return "closing"
}

func (v Vignettes) Apply(g *Graph, m Media) (Media, error) {
	if v.Intro == nil && v.Outro == nil {
		return m, nil
	}
	fps := m.FrameRate
	if fps <= 0 {
		fps = 30
	}
	rate := strconv.FormatFloat(fps, 'f', 3, 64)
	sound := func(in Stream, has bool, duration float64) Stream {
		format := []Filter{F("aresample", 48000), F("aformat", "sample_fmts=fltp", "channel_layouts=stereo")}
		if has {
			return g.Pipe(in, format...)
		}
		return g.Apply(nil, append([]Filter{F("anullsrc", "r=48000", "cl=stereo"), F("atrim", "duration="+seconds(duration))}, format...)...)
	}
	part := func(vi *VignetteInput) []Stream {
		in := g.Input(vi.Path)
		picture := g.Pipe(fitOverBlur(g, g.Pipe(g.Video(in), F("fps", rate)), m.Width, m.Height), F("format", "yuv420p"))
		return []Stream{picture, sound(g.Audio(in), vi.Audio, vi.Duration)}
	}
	clip := []Stream{g.Pipe(m.Video, F("fps", rate), F("format", "yuv420p")), sound(m.Audio, m.HasAudio, m.Duration)}
	var parts []Stream
	if v.Intro != nil {
		parts = append(parts, part(v.Intro)...)
		m.Duration += v.Intro.Duration
	}
	parts = append(parts, clip...)
	if v.Outro != nil {
		parts = append(parts, part(v.Outro)...)
		m.Duration += v.Outro.Duration
	}
	joined := g.Fork(parts, 2, F("concat", fmt.Sprintf("n=%d", len(parts)/2), "v=1", "a=1"))
	m.Video, m.Audio, m.HasAudio = joined[0], joined[1], true
	return m, nil
}
