// Package ffmpeg builds and runs FFmpeg commands. Commands are always run
// with an argument list, never through a shell.
package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
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

// HLSArgs returns the arguments to transcode input into a single-rendition
// HLS stream (H.264 + AAC) inside outDir.
func HLSArgs(input, outDir string) []string {
	seg := strconv.Itoa(SegmentSeconds)
	return []string{
		"-hide_banner", "-nostdin", "-y", "-loglevel", "warning",
		"-i", input,
		"-map", "0:v:0", "-map", "0:a:0?", // first video stream; audio if present
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "21",
		"-pix_fmt", "yuv420p", // 8-bit 4:2:0, playable in every browser
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", // H.264 needs even dimensions
		"-force_key_frames", "expr:gte(t,n_forced*" + seg + ")", // a keyframe at each segment start
		"-c:a", "aac", "-b:a", "128k", "-ac", "2",
		"-f", "hls",
		"-hls_time", seg,
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(outDir, "segment_%03d.ts"),
		filepath.Join(outDir, ManifestName),
	}
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
