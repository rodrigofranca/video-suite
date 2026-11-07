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
	if c.SubtitlesFile != "" {
		graph += ",ass=filename=" + escapeFilterValue(c.SubtitlesFile)
	}
	graph += "[v]"

	return []string{
		"-hide_banner", "-nostdin", "-y", "-loglevel", "warning",
		"-ss", seconds(c.Start), "-i", c.Input, "-t", seconds(c.Duration),
		"-filter_complex", graph,
		"-map", "[v]", "-map", "0:a:0?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "160k", "-ac", "2",
		"-movflags", "+faststart", // playable while it downloads
		c.Output,
	}, nil
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
