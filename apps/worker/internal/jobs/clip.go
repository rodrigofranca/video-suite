package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
	"ultramedia/worker/internal/captions"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// Captions per screen: vertical video has room for fewer words per line.
const (
	captionCharsVertical = 32
	captionCharsOriginal = 84
)

// ExportClip cuts one suggested clip out of the source as an MP4 in the
// requested format, optionally with the subtitles burned in (docs/adr/0020).
// The clip's start and end come from the database, read when the job runs.
func (m Media) ExportClip(ctx context.Context, job Job, in jobcontracts.ExportClipInput) error {
	if err := validInput(in.AssetID, in.StorageKey); err != nil {
		return err
	}
	if !domain.IsID(in.ClipID) || !slices.Contains(jobcontracts.ClipFormats, in.Format) ||
		in.OutputKey != storage.ClipKey(in.AssetID, in.ClipID, in.Format) ||
		(in.CaptionStyle != "" && !slices.Contains(jobcontracts.CaptionStyles, in.CaptionStyle)) {
		return fmt.Errorf("invalid input: clip_id %q, format %q, caption_style %q, output_key %q", in.ClipID, in.Format, in.CaptionStyle, in.OutputKey)
	}
	clip, err := m.Store.GetClip(ctx, in.ClipID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && clip.AssetID != in.AssetID) {
		return fmt.Errorf("clip %s not found for asset %s; the clips were probably suggested again", in.ClipID, in.AssetID)
	}
	if err != nil {
		return err
	}
	duration := clip.End - clip.Start
	job.Log.Info(fmt.Sprintf("exporting clip %q, %.1fs to %.1fs (%.1fs), format %s", clip.Title, clip.Start, clip.End, duration, in.Format))

	url, err := m.Storage.PresignGet(ctx, in.StorageKey, sourceURLExpiry)
	if err != nil {
		return err
	}
	_, src, err := ffmpeg.Probe(ctx, url, in.StorageKey)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(m.TempDir, "clip-"+in.ClipID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	width, height := ffmpeg.ClipSize(in.Format, src.Width, src.Height)
	spec := ffmpeg.ClipSpec{
		Input: url, Start: clip.Start, Duration: duration, Format: in.Format,
		SourceWidth: src.Width, SourceHeight: src.Height, Output: filepath.Join(dir, "clip.mp4"),
	}
	framing := ""
	if in.Format == jobcontracts.ClipFormatVerticalFace {
		spec.Track, framing = faceTrack(job, clip, src.Width, src.Height)
	}
	var subs burned
	if in.Captions {
		subs, err = m.clipCaptions(ctx, job, in, clip, width, height, dir)
		if err != nil {
			return err
		}
		spec.SubtitlesFile = subs.Path
	}

	args, err := ffmpeg.ClipArgs(spec)
	if err != nil {
		return err
	}
	job.Log.Info("running " + ffmpeg.CommandLine(args, url))
	started := time.Now()
	total := time.Duration(duration * float64(time.Second))
	err = ffmpeg.Run(ctx, args, job.Log, ffmpeg.Options{
		Redact:   []string{url},
		Progress: func(written time.Duration) { job.Progress(min(int(written*100/total), 99)) },
	})
	if err != nil {
		return err
	}
	info, err := os.Stat(spec.Output)
	if err != nil {
		return err
	}
	job.Log.Info(fmt.Sprintf("encoded %dx%d in %s, %d KB", width, height, time.Since(started).Round(100*time.Millisecond), info.Size()/1024))

	if err := m.Storage.UploadFile(ctx, in.OutputKey, spec.Output, "video/mp4"); err != nil {
		return err
	}
	job.Log.Info("uploaded clip", "key", in.OutputKey)
	meta, _ := json.Marshal(map[string]any{
		"clipId": clip.ID, "format": in.Format, "framing": framing,
		"captions": subs.Path != "", "captionStyle": subs.Style, "language": subs.Language,
		"captionPosition": subs.Position, "captionColor": subs.Color, "captionCaps": subs.Caps,
		"start": clip.Start, "end": clip.End, "durationSeconds": duration,
		"width": width, "height": height, "sizeBytes": info.Size(),
	})
	return m.Store.CreateArtifact(ctx, domain.Artifact{
		ID: domain.NewID(), AssetID: in.AssetID, JobID: job.ID, Type: domain.ArtifactClip,
		StorageKey: in.OutputKey, MetadataJSON: meta,
	})
}

// Word-by-word captions: phrases per screen.
const (
	phraseWordsVertical, phraseCharsVertical = 4, 22
	phraseWordsOriginal, phraseCharsOriginal = 7, 40
)

// clipCaptions writes the clip's captions as an ASS file sized for the clip,
// in the requested style: word by word from the transcript's word timings,
// or the subtitle lines. Word by word falls back to lines for transcripts
// made before word timings were saved. It returns the file, the language and
// the style used, or an empty path, not an error, when there is nothing to
// burn in.
// burned is what was burned into a clip: the ASS file and the choices it
// ended up with, after falling back to line captions and to the defaults.
type burned struct {
	Path, Language, Style string
	Position, Color       string // Color is empty for styles without an accent
	Caps                  bool
}

// accentOf is each style's own accent color; lines and one-word have none.
var accentOf = map[string]string{
	jobcontracts.CaptionStyleHighlight: "yellow",
	jobcontracts.CaptionStyleKaraoke:   "yellow",
	jobcontracts.CaptionStyleBox:       "violet",
}

// clipCaptions writes the clip's captions in the chosen style, position,
// color and case (docs/adr/0021, docs/adr/0025). Word styles fall back to
// lines when the transcript has no word timings; the path is empty when
// there is nothing to burn.
func (m Media) clipCaptions(ctx context.Context, job Job, in jobcontracts.ExportClipInput, clip domain.Clip, width, height int, dir string) (burned, error) {
	style := in.CaptionStyle
	if style == "" {
		style = jobcontracts.CaptionStyles[0]
	}
	position := in.CaptionPosition
	if position == "" {
		position = jobcontracts.CaptionPositions[0]
	}
	options := func(style string) (captions.Options, string) {
		caps := style == jobcontracts.CaptionStyleOneWord // its look, unless turned off
		if in.CaptionCaps != nil {
			caps = *in.CaptionCaps
		}
		color := ""
		if accentOf[style] != "" {
			color = accentOf[style]
			if slices.Contains(jobcontracts.CaptionColors, in.CaptionColor) {
				color = in.CaptionColor
			}
		}
		return captions.Options{Position: position, Color: color, Caps: caps}, color
	}
	if style != jobcontracts.CaptionStyleLines {
		opts, color := options(style)
		path, language, err := m.wordCaptions(ctx, job, in.AssetID, style, clip, width, height, dir, opts)
		if err != nil || path != "" {
			return burned{path, language, style, position, color, opts.Caps}, err
		}
	}
	opts, _ := options(jobcontracts.CaptionStyleLines)
	path, language, err := m.lineCaptions(ctx, job, in.AssetID, clip, width, height, dir, opts)
	if path == "" {
		return burned{}, err
	}
	return burned{path, language, jobcontracts.CaptionStyleLines, position, "", opts.Caps}, err
}

// wordCaptions writes captions that follow the transcript's word timings,
// in one of the word styles. It returns an empty path when the transcript has
// no word timings for the clip, so the caller can fall back to line captions.
func (m Media) wordCaptions(ctx context.Context, job Job, assetID, style string, clip domain.Clip, width, height int, dir string, opts captions.Options) (string, string, error) {
	tr, err := m.Store.GetTranscript(ctx, assetID)
	if errors.Is(err, store.ErrNotFound) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	all, err := captions.ParseWords(tr.Words)
	if err != nil {
		return "", "", err
	}
	if len(all) == 0 {
		job.Log.Warn("this transcript has no word timings (it was made before they were saved); using line captions. Run the transcription again for the " + style + " style")
		return "", "", nil
	}
	maxWords, maxChars := phraseWordsOriginal, phraseCharsOriginal
	if height > width {
		maxWords, maxChars = phraseWordsVertical, phraseCharsVertical
	}
	words := captions.CutWords(all, clip.Start, clip.End)
	phrases := captions.Phrases(words, maxWords, maxChars)
	if len(phrases) == 0 {
		return "", "", nil
	}
	path := filepath.Join(dir, "captions.ass")
	err = writeFile(path, func(f *os.File) error {
		phraseStyle := captions.WordStyleFor(width, height).With(opts, opts.Color)
		switch style {
		case jobcontracts.CaptionStyleBox:
			return captions.WriteBox(f, phrases, phraseStyle)
		case jobcontracts.CaptionStyleKaraoke:
			return captions.WriteKaraoke(f, phrases, phraseStyle)
		case jobcontracts.CaptionStyleOneWord:
			return captions.WriteOneWord(f, words, captions.OneWordStyleFor(width, height).With(opts, ""))
		default:
			return captions.WriteHighlight(f, phrases, phraseStyle)
		}
	})
	if err != nil {
		return "", "", err
	}
	job.Log.Info(fmt.Sprintf("burning in %s captions: %d words in %d phrases (%s)%s", style, len(words), len(phrases), tr.Language, describe(opts)))
	return path, tr.Language, nil
}

func (m Media) lineCaptions(ctx context.Context, job Job, assetID string, clip domain.Clip, width, height int, dir string, opts captions.Options) (string, string, error) {
	art, err := m.Store.LatestArtifact(ctx, assetID, domain.ArtifactSubtitles)
	if errors.Is(err, store.ErrNotFound) {
		job.Log.Warn("no subtitles for this video; exporting without captions")
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var meta struct {
		Language string `json:"language"`
	}
	json.Unmarshal(art.MetadataJSON, &meta)

	vttPath := filepath.Join(dir, "subtitles.vtt")
	if err := m.Storage.DownloadFile(ctx, art.StorageKey, vttPath); err != nil {
		return "", "", err
	}
	f, err := os.Open(vttPath)
	if err != nil {
		return "", "", err
	}
	all, err := captions.ParseVTT(f)
	f.Close()
	if err != nil {
		return "", "", err
	}
	maxChars := captionCharsOriginal
	if height > width {
		maxChars = captionCharsVertical
	}
	cues := captions.Split(captions.Cut(all, clip.Start, clip.End), maxChars)
	if len(cues) == 0 {
		job.Log.Warn("no subtitles in this part of the video; exporting without captions")
		return "", "", nil
	}

	assPath := filepath.Join(dir, "captions.ass")
	if err := writeFile(assPath, func(f *os.File) error {
		return captions.WriteASS(f, cues, captions.StyleFor(width, height).With(opts, ""))
	}); err != nil {
		return "", "", err
	}
	job.Log.Info(fmt.Sprintf("burning in %d line captions (%s)%s", len(cues), meta.Language, describe(opts)))
	return assPath, meta.Language, nil
}

// describe lists the caption choices that differ from the plain look, for the log.
func describe(o captions.Options) string {
	var parts []string
	if o.Position != "" && o.Position != "bottom" {
		parts = append(parts, "at the "+o.Position)
	}
	if o.Color != "" {
		parts = append(parts, o.Color)
	}
	if o.Caps {
		parts = append(parts, "in capitals")
	}
	if len(parts) == 0 {
		return ""
	}
	return ", " + strings.Join(parts, ", ")
}

func writeFile(path string, write func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// minFaceCoverage is the share of a clip in which a face must be found for
// the crop to follow it (the track-faces job leaves the points empty below it).
const minFaceCoverage = 0.3

// faceTrack returns the points a "vertical-face" crop follows and the framing
// used: "face", or "center" when there is nothing to follow, with the reason
// in the log.
func faceTrack(job Job, clip domain.Clip, width, height int) ([]ffmpeg.TrackPoint, string) {
	if len(clip.FaceTrack) == 0 {
		job.Log.Warn("this clip has no face track (the Face tracking job has not run since the clips were suggested); using the centre")
		return nil, "center"
	}
	var track struct {
		Coverage      float64             `json:"coverage"`
		Points        []ffmpeg.TrackPoint `json:"points"`
		Width, Height int
	}
	if err := json.Unmarshal(clip.FaceTrack, &track); err != nil {
		job.Log.Warn("unreadable face track; using the centre: " + err.Error())
		return nil, "center"
	}
	switch {
	case track.Width != width || track.Height != height:
		job.Log.Warn(fmt.Sprintf("the face track was made for a %dx%d picture, not %dx%d; using the centre", track.Width, track.Height, width, height))
		return nil, "center"
	case len(track.Points) == 0 && track.Coverage < minFaceCoverage:
		job.Log.Warn(fmt.Sprintf("a face was found in only %.0f%% of the clip; using the centre", track.Coverage*100))
		return nil, "center"
	case len(track.Points) == 0:
		job.Log.Info("the picture is already narrower than 9:16; using the centre")
		return nil, "center"
	}
	job.Log.Info(fmt.Sprintf("following the face (found in %.0f%% of the clip, %d camera points)", track.Coverage*100, len(track.Points)))
	return track.Points, "face"
}
