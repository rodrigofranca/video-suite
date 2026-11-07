package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// Summary is the technical metadata shown in the dashboard.
type Summary struct {
	DurationSeconds float64 `json:"durationSeconds"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	VideoCodec      string  `json:"videoCodec,omitempty"`
	AudioCodec      string  `json:"audioCodec,omitempty"`
	BitRate         int64   `json:"bitRate,omitempty"` // bits per second, whole file
	FrameRate       float64 `json:"frameRate,omitempty"`
	Container       string  `json:"container,omitempty"`
	SizeBytes       int64   `json:"sizeBytes,omitempty"`
}

// Probe runs ffprobe on input (a file path or URL). It returns ffprobe's JSON
// output and a summary of it. name replaces the input in the output's
// "filename" field and in errors, so signed URLs are never stored or logged.
func Probe(ctx context.Context, input, name string) ([]byte, Summary, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error", "-print_format", "json", "-show_format", "-show_streams", input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, Summary{}, ctx.Err()
		}
		msg := strings.TrimSpace(redact(stderr.String(), []string{input}))
		return nil, Summary{}, fmt.Errorf("ffprobe failed (%v): %s", err, msg)
	}

	raw, err := replaceFilename(stdout.Bytes(), name)
	if err != nil {
		return nil, Summary{}, err
	}
	summary, err := Summarize(raw)
	return raw, summary, err
}

// ffprobe JSON, only the fields the summary uses.
type probeOutput struct {
	Format struct {
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
		Size       string `json:"size"`
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
		RFrameRate   string `json:"r_frame_rate"`
		Disposition  struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
		SideData []struct {
			Rotation float64 `json:"rotation"`
		} `json:"side_data_list"`
	} `json:"streams"`
}

// Summarize extracts the summary from ffprobe's JSON output. It fails when
// the file has no video stream.
func Summarize(raw []byte) (Summary, error) {
	var p probeOutput
	if err := json.Unmarshal(raw, &p); err != nil {
		return Summary{}, fmt.Errorf("parse ffprobe output: %w", err)
	}
	s := Summary{
		DurationSeconds: parseFloat(p.Format.Duration),
		BitRate:         int64(parseFloat(p.Format.BitRate)),
		SizeBytes:       int64(parseFloat(p.Format.Size)),
		Container:       p.Format.FormatName,
	}
	for _, st := range p.Streams {
		switch {
		case st.CodecType == "video" && st.Disposition.AttachedPic == 0 && s.VideoCodec == "":
			s.VideoCodec, s.Width, s.Height = st.CodecName, st.Width, st.Height
			// Phone videos are stored sideways with a rotation to apply when
			// shown, as FFmpeg does when decoding: report the size as shown.
			for _, d := range st.SideData {
				if int(math.Abs(d.Rotation)+0.5)%180 == 90 {
					s.Width, s.Height = s.Height, s.Width
				}
			}
			s.FrameRate = frameRate(st.AvgFrameRate)
			if s.FrameRate == 0 {
				s.FrameRate = frameRate(st.RFrameRate)
			}
		case st.CodecType == "audio" && s.AudioCodec == "":
			s.AudioCodec = st.CodecName
		}
	}
	if s.VideoCodec == "" {
		return s, errors.New("the file has no video stream")
	}
	return s, nil
}

// replaceFilename sets format.filename in ffprobe's output to name.
func replaceFilename(raw []byte, name string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}
	if format, ok := doc["format"].(map[string]any); ok {
		format["filename"] = name
	}
	return json.Marshal(doc)
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// frameRate turns "30000/1001" into 29.97 (rounded to 3 decimals).
func frameRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		return 0
	}
	n, d := parseFloat(num), parseFloat(den)
	if d == 0 {
		return 0
	}
	return float64(int(n/d*1000+0.5)) / 1000
}
