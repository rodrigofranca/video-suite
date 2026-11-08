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
		(in.CaptionStyle != "" && !slices.Contains(jobcontracts.CaptionStyles, in.CaptionStyle)) ||
		(in.CaptionLanguage != "" && !slices.Contains(jobcontracts.CaptionLanguages, in.CaptionLanguage)) {
		return fmt.Errorf("invalid input: clip_id %q, format %q, caption_style %q, caption_language %q, output_key %q",
			in.ClipID, in.Format, in.CaptionStyle, in.CaptionLanguage, in.OutputKey)
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
		"captions": subs.Path != "", "captionStyle": subs.Style, "language": subs.Language, "translatedFrom": subs.TranslatedFrom,
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

// burned is what was burned into a clip: the ASS file and the choices it
// ended up with, after falling back to line captions and to the defaults.
type burned struct {
	Path, Language  string
	TranslatedFrom  string // the transcript's language, for translated captions
	Style           string
	Position, Color string // Color is empty for styles without an accent
	Caps            bool
}

// speech is what captions say: the transcript's words, or its translation's
// (docs/adr/0029).
type speech struct {
	Words    []byte // JSON words, as in transcripts.words; nil without a transcript
	Language string
	From     string // the transcript's language, for a translation
}

// captionSpeech returns the words captions in language show: the
// transcript's when language is empty (or the transcript's own), else its
// translation, which the translate-transcript job wrote before this export.
func (m Media) captionSpeech(ctx context.Context, assetID, language string) (speech, error) {
	if language != "" {
		tr, err := m.Store.GetTranslation(ctx, assetID, language)
		if err == nil {
			return speech{tr.Words, tr.Language, tr.SourceLanguage}, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return speech{}, err
		}
	}
	tr, err := m.Store.GetTranscript(ctx, assetID)
	switch {
	case errors.Is(err, store.ErrNotFound) && language == "":
		return speech{}, nil // line captions may still exist
	case errors.Is(err, store.ErrNotFound):
		return speech{}, fmt.Errorf("there is no transcript to translate into %s", language)
	case err != nil:
		return speech{}, err
	case language != "" && language != tr.Language:
		// The translate-transcript job before this one failed, or a new
		// transcript replaced the translation since.
		return speech{}, fmt.Errorf("there is no %s translation of the transcript: the translation failed (see the translate-transcript job) or the transcript was made again since; export again to translate it", language)
	}
	return speech{tr.Words, tr.Language, ""}, nil
}

// accentOf is each style's own accent color; lines and one-word have none.
var accentOf = map[string]string{
	jobcontracts.CaptionStyleHighlight: "yellow",
	jobcontracts.CaptionStyleKaraoke:   "yellow",
	jobcontracts.CaptionStyleBox:       "violet",
}

// clipCaptions writes the clip's captions in the chosen style, position,
// color, case and language (docs/adr/0021, 0025, 0029). Word styles fall
// back to lines when the transcript has no word timings; the path is empty
// when there is nothing to burn.
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
	said, err := m.captionSpeech(ctx, in.AssetID, in.CaptionLanguage)
	if err != nil {
		return burned{}, err
	}
	if style != jobcontracts.CaptionStyleLines {
		opts, color := options(style)
		path, err := m.wordCaptions(job, said, style, clip, width, height, dir, opts)
		if err != nil || path != "" {
			return burned{path, said.Language, said.From, style, position, color, opts.Caps}, err
		}
	}
	opts, _ := options(jobcontracts.CaptionStyleLines)
	path, language, err := m.lineCaptions(ctx, job, in.AssetID, said, clip, width, height, dir, opts)
	if path == "" {
		return burned{}, err
	}
	return burned{path, language, said.From, jobcontracts.CaptionStyleLines, position, "", opts.Caps}, err
}

// wordCaptions writes captions that follow the words' timings, in one of
// the word styles. It returns an empty path when there are no word timings
// for the clip, so the caller can fall back to line captions.
func (m Media) wordCaptions(job Job, said speech, style string, clip domain.Clip, width, height int, dir string, opts captions.Options) (string, error) {
	if said.Words == nil {
		return "", nil // no transcript
	}
	all, err := captions.ParseWords(said.Words)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		job.Log.Warn("this transcript has no word timings (it was made before they were saved); using line captions. Run the transcription again for the " + style + " style")
		return "", nil
	}
	maxWords, maxChars := phraseWordsOriginal, phraseCharsOriginal
	if height > width {
		maxWords, maxChars = phraseWordsVertical, phraseCharsVertical
	}
	words := captions.CutWords(all, clip.Start, clip.End)
	phrases := captions.Phrases(words, maxWords, maxChars)
	if len(phrases) == 0 {
		return "", nil
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
		return "", err
	}
	job.Log.Info(fmt.Sprintf("burning in %s captions: %d words in %d phrases (%s)%s", style, len(words), len(phrases), said.describe(), describe(opts)))
	return path, nil
}

// describe names the language for the log: "en", or "en, translated from pt".
func (s speech) describe() string {
	if s.From != "" {
		return s.Language + ", translated from " + s.From
	}
	return s.Language
}

// lineCaptions writes the subtitle lines of the clip: the original
// subtitles, or the translated ones for a translation.
func (m Media) lineCaptions(ctx context.Context, job Job, assetID string, said speech, clip domain.Clip, width, height int, dir string, opts captions.Options) (string, string, error) {
	art, meta, err := m.subtitlesFor(ctx, assetID, said)
	if errors.Is(err, store.ErrNotFound) {
		job.Log.Warn("no subtitles for this video; exporting without captions")
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}

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
	job.Log.Info(fmt.Sprintf("burning in %d line captions (%s)%s", len(cues), speech{Language: meta.Language, From: meta.TranslatedFrom}.describe(), describe(opts)))
	return assPath, meta.Language, nil
}

type subtitlesMeta struct {
	Language       string `json:"language"`
	TranslatedFrom string `json:"translatedFrom"`
}

// subtitlesFor finds the newest subtitles file for the captions: the
// original's, or for a translation the one in its language.
func (m Media) subtitlesFor(ctx context.Context, assetID string, said speech) (domain.Artifact, subtitlesMeta, error) {
	arts, err := m.Store.ListArtifacts(ctx, assetID)
	if err != nil {
		return domain.Artifact{}, subtitlesMeta{}, err
	}
	for i := len(arts) - 1; i >= 0; i-- { // arts are oldest first: look from the newest
		var meta subtitlesMeta
		if arts[i].Type != domain.ArtifactSubtitles || json.Unmarshal(arts[i].MetadataJSON, &meta) != nil {
			continue
		}
		if said.From == "" && meta.TranslatedFrom == "" || said.From != "" && meta.TranslatedFrom != "" && meta.Language == said.Language {
			return arts[i], meta, nil
		}
	}
	return domain.Artifact{}, subtitlesMeta{}, store.ErrNotFound
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
