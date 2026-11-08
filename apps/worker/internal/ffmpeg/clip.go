package ffmpeg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Vertical clips are 1080x1920 (9:16), the size social apps expect.
const (
	VerticalWidth  = 1080
	VerticalHeight = 1920
	// maxSide is the longest side of a clip that keeps the video's shape.
	maxSide = 1920
)

// ClipSpec describes one clip export.
type ClipSpec struct {
	Input         string  // file path or signed URL of the source
	Start         float64 // seconds
	Duration      float64 // seconds
	Format        string  // "original", "vertical" or "vertical-blur"
	SourceWidth   int
	SourceHeight  int
	SubtitlesFile string // ASS file to burn in; empty for none
	Output        string
	// Track is where to centre a "vertical-face" crop over time; empty
	// means the middle of the frame.
	Track []TrackPoint
	// Intro and Outro are vignettes joined before and after the clip
	// (docs/adr/0030); nil for none. They are converted to the clip's
	// size and FrameRate (30 when unknown), and SourceAudio says whether
	// the clip itself has sound.
	Intro, Outro *VignetteInput
	FrameRate    float64
	SourceAudio  bool
	// Watermark is an image laid over the clip, below the captions and not
	// over the vignettes (docs/adr/0031); nil for none.
	Watermark *WatermarkInput
}

// WatermarkInput is an image over a clip: its file, corner ("top-right",
// "top-left", "bottom-right", "bottom-left"), width as a percentage of the
// clip's shorter side, and opacity in percent.
type WatermarkInput struct {
	Path     string
	Position string
	Size     int
	Opacity  int
}

// watermarkFilters lays the image at input over the labelled picture base,
// a margin of 4% of the shorter side away from the corner.
func watermarkFilters(wm *WatermarkInput, input, w, h int) string {
	short := min(w, h)
	margin := short * 4 / 100
	x, y := strconv.Itoa(margin), strconv.Itoa(margin)
	if strings.HasSuffix(wm.Position, "right") {
		x = fmt.Sprintf("main_w-overlay_w-%d", margin)
	}
	if strings.HasPrefix(wm.Position, "bottom") {
		y = fmt.Sprintf("main_h-overlay_h-%d", margin)
	}
	return fmt.Sprintf("[base];[%d:v:0]format=rgba,scale=%d:-1,colorchannelmixer=aa=%.2f[wm];[base][wm]overlay=x=%s:y=%s",
		input, even(short*wm.Size/100), float64(wm.Opacity)/100, x, y)
}

// VignetteInput is an opening or closing to join to a clip.
type VignetteInput struct {
	Path     string
	Duration float64 // seconds
	Audio    bool
}

// TrackPoint is the centre of the 9:16 window at a time of the clip.
type TrackPoint struct {
	T float64 `json:"t"` // seconds from the clip start
	X float64 `json:"x"` // share of the source width
}

// ClipSize returns the width and height of the exported clip.
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

// ClipArgs returns the arguments to cut and reframe a clip as H.264 + AAC
// MP4. Seeking before the input is fast and, since the video is re-encoded,
// frame accurate; timestamps then start at 0, which the subtitles rely on.
func ClipArgs(c ClipSpec) ([]string, error) {
	if c.SourceWidth <= 0 || c.SourceHeight <= 0 {
		return nil, fmt.Errorf("unknown source size %dx%d", c.SourceWidth, c.SourceHeight)
	}
	w, h := ClipSize(c.Format, c.SourceWidth, c.SourceHeight)
	var graph string
	switch c.Format {
	case "original":
		graph = fmt.Sprintf("[0:v:0]scale=%d:%d,setsar=1", w, h)
	case "vertical", "vertical-face":
		// As wide as 9:16 allows: the middle of the frame, or following the track.
		cw := even(min(c.SourceWidth, c.SourceHeight*9/16))
		ch := even(min(c.SourceHeight, c.SourceWidth*16/9))
		x := strconv.Itoa((c.SourceWidth - cw) / 2)
		if c.Format == "vertical-face" && len(c.Track) > 0 {
			x = "'" + trackExpr(c.Track, c.SourceWidth, cw) + "'"
		}
		graph = fmt.Sprintf("[0:v:0]crop=w=%d:h=%d:x=%s:y=%d,scale=%d:%d,setsar=1", cw, ch, x, (c.SourceHeight-ch)/2, w, h)
	case "vertical-blur":
		// The whole frame, centered over a blurred, zoomed copy of itself.
		graph = fmt.Sprintf("[0:v:0]split=2[bg][fg];"+
			"[bg]scale=%[1]d:%[2]d:force_original_aspect_ratio=increase,crop=%[1]d:%[2]d,gblur=sigma=30,eq=brightness=-0.08[bgb];"+
			"[fg]scale=%[1]d:%[2]d:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[fgs];"+
			"[bgb][fgs]overlay=(W-w)/2:(H-h)/2,setsar=1", w, h)
	default:
		return nil, fmt.Errorf("unknown clip format %q", c.Format)
	}
	// Seek and length are options of the source input, so they apply to it alone.
	inputs := []string{"-ss", seconds(c.Start), "-t", seconds(c.Duration), "-i", c.Input}
	next := 1 // the index of the next input
	if c.Watermark != nil {
		// Below the captions, so they stay readable.
		inputs = append(inputs, "-i", c.Watermark.Path)
		graph += watermarkFilters(c.Watermark, next, w, h)
		next++
	}
	if c.SubtitlesFile != "" {
		graph += ",ass=filename=" + escapeFilterValue(c.SubtitlesFile)
	}
	maps := []string{"-map", "[v]", "-map", "0:a:0?"}
	if c.Intro != nil || c.Outro != nil {
		graph, inputs = joinVignettes(c, graph, inputs, next, w, h)
		maps = []string{"-map", "[v]", "-map", "[a]"}
	} else {
		graph += "[v]"
	}

	args := []string{"-hide_banner", "-nostdin", "-y", "-loglevel", "warning"}
	args = append(args, inputs...)
	args = append(args, "-filter_complex", graph)
	args = append(args, maps...)
	return append(args,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "160k", "-ac", "2",
		"-movflags", "+faststart", // playable while it downloads
		c.Output,
	), nil
}

// joinVignettes extends a clip's filter graph (its picture chain, not yet
// labelled) with the vignettes: each is fitted whole into the clip's size
// over a blurred copy of itself, at the clip's frame rate, with its sound or
// silence, and the parts are joined in order into [v] and [a]. The clip gets
// silence too when the source has no sound.
func joinVignettes(c ClipSpec, clipGraph string, inputs []string, next, w, h int) (string, []string) {
	fps := c.FrameRate
	if fps <= 0 {
		fps = 30
	}
	rate := strconv.FormatFloat(fps, 'f', 3, 64)
	const audioFormat = "aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo"
	audio := func(input int, has bool, duration float64, label string) string {
		if has {
			return fmt.Sprintf("[%d:a:0]%s[%s]", input, audioFormat, label)
		}
		return fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%s,%s[%s]", seconds(duration), audioFormat, label)
	}
	parts := []string{clipGraph + ",fps=" + rate + ",format=yuv420p[cv]", audio(0, c.SourceAudio, c.Duration, "ca")}
	var order []string
	vignette := func(v *VignetteInput, name string) {
		input := next
		next++
		inputs = append(inputs, "-i", v.Path)
		parts = append(parts,
			fmt.Sprintf("[%[1]d:v:0]fps=%[2]s,split=2[%[3]sbg][%[3]sfg];"+
				"[%[3]sbg]scale=%[4]d:%[5]d:force_original_aspect_ratio=increase,crop=%[4]d:%[5]d,gblur=sigma=30,eq=brightness=-0.08[%[3]sbgb];"+
				"[%[3]sfg]scale=%[4]d:%[5]d:force_original_aspect_ratio=decrease:force_divisible_by=2,setsar=1[%[3]sfgs];"+
				"[%[3]sbgb][%[3]sfgs]overlay=(W-w)/2:(H-h)/2,setsar=1,format=yuv420p[%[3]sv]", input, rate, name, w, h),
			audio(input, v.Audio, v.Duration, name+"a"))
		order = append(order, "["+name+"v]["+name+"a]")
	}
	if c.Intro != nil {
		vignette(c.Intro, "i")
	}
	order = append(order, "[cv][ca]")
	if c.Outro != nil {
		vignette(c.Outro, "o")
	}
	parts = append(parts, fmt.Sprintf("%sconcat=n=%d:v=1:a=1[v][a]", strings.Join(order, ""), len(order)))
	return strings.Join(parts, ";"), inputs
}

func seconds(s float64) string { return strconv.FormatFloat(s, 'f', 3, 64) }

func even(n int) int { return max(n-n%2, 2) }

// escapeFilterValue escapes a value for a filtergraph option: the characters
// that separate options, filters and labels.
var filterEscaper = strings.NewReplacer(`\`, `\\\\`, `'`, `\\\'`, `:`, `\\:`, `,`, `\,`, `;`, `\;`, `[`, `\[`, `]`, `\]`)

func escapeFilterValue(s string) string { return filterEscaper.Replace(s) }

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
