// Package ffmpeg builds and runs FFmpeg commands. Commands are always run
// with an argument list, never through a shell.
package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SegmentSeconds is the target length of each HLS segment.
const SegmentSeconds = 6

// ManifestName is the HLS playlist file name.
const ManifestName = "index.m3u8"

// Rendition is one quality of the HLS ladder (docs/adr/0028).
type Rendition struct {
	Name    string `json:"name"` // the short side, e.g. "720p"
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	MaxKbps int    `json:"maxKbps"` // the video bitrate cap
}

// ladder is the qualities offered, by the picture's short side, with the
// video bitrate cap of each (H.264, up to 30 fps).
var ladder = []struct{ short, kbps int }{{1080, 5000}, {720, 2800}, {480, 1400}, {360, 800}}

// maxShort is the top quality: bigger sources are scaled down to it.
const maxShort = 1080

// Ladder returns the qualities to encode a width x height picture in, best
// first: the source's own size (at most 1080p) and every step of the ladder
// below it, leaving out a step within 10% of the one above. Sizes keep the
// picture's shape, so a portrait video's "720p" is 720 wide.
func Ladder(width, height int) []Rendition {
	short := min(width, height)
	if short <= 0 {
		return nil
	}
	sizes := []int{min(short, maxShort)}
	for _, step := range ladder {
		if float64(step.short) < float64(sizes[len(sizes)-1])*0.9 {
			sizes = append(sizes, step.short)
		}
	}
	out := make([]Rendition, 0, len(sizes))
	for _, size := range sizes {
		w := even(int(math.Round(float64(width) * float64(size) / float64(short))))
		h := even(int(math.Round(float64(height) * float64(size) / float64(short))))
		out = append(out, Rendition{Name: fmt.Sprintf("%dp", min(w, h)), Width: w, Height: h, MaxKbps: kbpsFor(size)})
	}
	return out
}

// kbpsFor is the bitrate cap for a short side: the ladder's, in a straight
// line between its steps, and in proportion below the lowest.
func kbpsFor(short int) int {
	for i, step := range ladder {
		if short >= step.short {
			if i == 0 {
				return step.kbps
			}
			up := ladder[i-1]
			return step.kbps + (up.kbps-step.kbps)*(short-step.short)/(up.short-step.short)
		}
	}
	low := ladder[len(ladder)-1]
	return max(low.kbps*short/low.short, 200)
}

// HLSArgs returns the arguments to transcode input into HLS (H.264 + AAC) in
// several qualities inside outDir: a master playlist, ManifestName, that
// lists one folder per rendition, each with its playlist and segments. The
// renditions share keyframes at each segment start, so players can switch
// between them. Quality is constant (CRF) up to each rendition's cap.
func HLSArgs(input, outDir string, renditions []Rendition, audio bool) []string {
	seg := strconv.Itoa(SegmentSeconds)
	split := fmt.Sprintf("[0:v:0]split=%d", len(renditions))
	var scales, streams []string
	args := []string{"-hide_banner", "-nostdin", "-y", "-loglevel", "warning", "-i", input}
	var maps []string
	for i, r := range renditions {
		split += fmt.Sprintf("[v%d]", i)
		scales = append(scales, fmt.Sprintf("[v%d]scale=%d:%d[o%d]", i, r.Width, r.Height, i))
		maps = append(maps, "-map", fmt.Sprintf("[o%d]", i))
		stream := fmt.Sprintf("v:%d", i)
		if audio {
			maps = append(maps, "-map", "0:a:0")
			stream += fmt.Sprintf(",a:%d", i)
		}
		streams = append(streams, stream+",name:"+r.Name)
	}
	args = append(args, "-filter_complex", split+";"+strings.Join(scales, ";"))
	args = append(args, maps...)
	args = append(args,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "21",
		"-pix_fmt", "yuv420p", // 8-bit 4:2:0, playable in every browser
		"-force_key_frames", "expr:gte(t,n_forced*"+seg+")", // a keyframe at each segment start, in every rendition
	)
	for i, r := range renditions {
		args = append(args, fmt.Sprintf("-maxrate:v:%d", i), fmt.Sprintf("%dk", r.MaxKbps),
			fmt.Sprintf("-bufsize:v:%d", i), fmt.Sprintf("%dk", 2*r.MaxKbps))
	}
	if audio {
		args = append(args, "-c:a", "aac", "-b:a", "128k", "-ac", "2")
	}
	return append(args,
		"-f", "hls",
		"-hls_time", seg,
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-master_pl_name", ManifestName,
		"-hls_segment_filename", filepath.Join(outDir, "%v", "segment_%03d.ts"),
		"-var_stream_map", strings.Join(streams, " "),
		filepath.Join(outDir, "%v", ManifestName),
	)
}

// maxLoggedLines caps how many FFmpeg output lines go to the job log.
const maxLoggedLines = 50

// Options adjusts how Run executes FFmpeg.
type Options struct {
	// Redact lists strings, such as signed URLs, that are replaced by
	// "<source>" in logs and error messages.
	Redact []string
	// Progress, when set, receives how much media FFmpeg has written so far.
	Progress func(written time.Duration)
}

// Run executes ffmpeg with args. Each line FFmpeg prints (warnings and errors
// only, given -loglevel warning) goes to log; on failure the error includes
// the last lines so the job's error message explains what went wrong.
func Run(ctx context.Context, args []string, log *slog.Logger, opts Options) error {
	if opts.Progress != nil {
		args = append([]string{"-progress", "pipe:1", "-nostats"}, args...)
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	var progressDone chan struct{}
	if opts.Progress != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		progressDone = make(chan struct{})
		go func() {
			defer close(progressDone)
			readProgress(stdout, opts.Progress)
		}()
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	var last []string
	logged := 0
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := redact(strings.TrimSpace(scanner.Text()), opts.Redact)
		if line == "" {
			continue
		}
		last = append(last, line)
		if len(last) > 5 {
			last = last[1:]
		}
		switch {
		case logged < maxLoggedLines:
			log.Warn("ffmpeg: " + line)
		case logged == maxLoggedLines:
			log.Warn("ffmpeg: further output not logged")
		}
		logged++
	}
	if progressDone != nil {
		<-progressDone
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(last) == 0 {
			return fmt.Errorf("ffmpeg failed: %w", err)
		}
		return fmt.Errorf("ffmpeg failed (%v): %s", err, strings.Join(last, " | "))
	}
	return nil
}

// readProgress parses the key=value lines of "-progress pipe:1".
func readProgress(r io.Reader, report func(time.Duration)) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		value, ok := strings.CutPrefix(scanner.Text(), "out_time_us=")
		if !ok {
			continue
		}
		if us, err := strconv.ParseInt(value, 10, 64); err == nil && us >= 0 {
			report(time.Duration(us) * time.Microsecond)
		}
	}
}

// CommandLine formats args for a log line, hiding redacted values.
func CommandLine(args []string, redacted ...string) string {
	return redact("ffmpeg "+strings.Join(args, " "), redacted)
}

func redact(s string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "<source>")
		}
	}
	return s
}

// ThumbnailArgs returns the arguments to save the frame at the given second
// of input as a JPEG at most 1280 pixels wide.
func ThumbnailArgs(input string, atSecond float64, output string) []string {
	return []string{
		"-hide_banner", "-nostdin", "-y", "-loglevel", "error",
		"-ss", strconv.FormatFloat(atSecond, 'f', 3, 64), // before -i: fast seek
		"-i", input,
		"-frames:v", "1",
		"-vf", "scale='min(1280,iw)':-2",
		"-q:v", "3",
		output,
	}
}

// AudioArgs returns the arguments to save the first audio track of input as
// 16 kHz mono FLAC, the format Whisper works with (about 1 MB per minute).
func AudioArgs(input, output string) []string {
	return []string{
		"-hide_banner", "-nostdin", "-y", "-loglevel", "error",
		"-i", input,
		"-map", "0:a:0", "-vn",
		"-ac", "1", "-ar", "16000",
		"-c:a", "flac",
		output,
	}
}
