package ffmpeg

import (
	"fmt"
	"strconv"
	"strings"
)

// A clip export is a source and a list of stages over one FFmpeg run
// (docs/adr/0035). Each stage takes the clip as the stages before left it,
// adds its filters to the graph, and passes the clip on. Nothing is encoded
// twice, whatever the stages.

// Media is the clip as the stages pass it on: its picture and sound in the
// graph, the picture's size, and the clip's length and frame rate.
type Media struct {
	Video, Audio  Stream
	HasAudio      bool // false: Audio is the source's missing sound
	Width, Height int
	Duration      float64
	FrameRate     float64 // 0 when unknown
}

// Stage is one step of an export. String names it for the job log.
type Stage interface {
	Apply(g *Graph, m Media) (Media, error)
	String() string
}

// Source is the part of the video to export: the file or signed URL, where
// to start and for how long, and what the probe found.
type Source struct {
	Path            string
	Start, Duration float64
	Width, Height   int
	FrameRate       float64
	Audio           bool
}

// Export is one clip export. Stages run in the order given: the export job
// changes the picture first (Frame), then draws over it (Watermark, then
// Captions, which stay on top), and joins the vignettes last, so they get
// none of it.
type Export struct {
	Source Source
	Stages []Stage
	Output string
}

// Args returns the FFmpeg arguments for the export: H.264 + AAC in an MP4
// that plays while it downloads. Seeking before the input is fast and, since
// the video is re-encoded, frame accurate; timestamps then start at 0, which
// the captions rely on. Seek and length are options of the source input, so
// they apply to it alone.
func (e Export) Args() ([]string, error) {
	s := e.Source
	if s.Width <= 0 || s.Height <= 0 {
		return nil, fmt.Errorf("unknown source size %dx%d", s.Width, s.Height)
	}
	var g Graph
	in := g.Input(s.Path, "-ss", seconds(s.Start), "-t", seconds(s.Duration))
	m := Media{Video: g.Video(in), Audio: g.Audio(in), HasAudio: s.Audio, Width: s.Width, Height: s.Height,
		Duration: s.Duration, FrameRate: s.FrameRate}
	for _, stage := range e.Stages {
		var err error
		if m, err = stage.Apply(&g, m); err != nil {
			return nil, fmt.Errorf("%s: %w", stage, err)
		}
	}
	io, err := g.Args(m.Video, m.Audio)
	if err != nil {
		return nil, err
	}
	args := append([]string{"-hide_banner", "-nostdin", "-y", "-loglevel", "warning"}, io...)
	return append(args,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "160k", "-ac", "2",
		"-movflags", "+faststart",
		e.Output,
	), nil
}

// Describe lists the stages for the job log: "crop to 9:16 → watermark".
func Describe(stages []Stage) string {
	names := make([]string, len(stages))
	for i, s := range stages {
		names[i] = s.String()
	}
	return strings.Join(names, " → ")
}

func seconds(s float64) string { return strconv.FormatFloat(s, 'f', 3, 64) }

func even(n int) int { return max(n-n%2, 2) }
