package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/worker/internal/ffmpeg"

	jobcontracts "ultramedia/job-contracts"
)

// addClip stores one suggested clip for the fixture's asset and, when vtt is
// not empty, subtitles for the whole video.
func (f *fixture) addClip(start, end float64, vtt string) domain.Clip {
	f.t.Helper()
	clip := domain.Clip{ID: domain.NewID(), JobID: f.jobs[jobcontracts.JobTypeSuggestClips].ID, Start: start, End: end,
		Title: "A good moment", Score: 8, Rank: 1, Model: "test"}
	if err := f.st.ReplaceClips(f.ctx, f.asset, []domain.Clip{clip}); err != nil {
		f.t.Fatal(err)
	}
	if vtt != "" {
		key := storage.SubtitlesKey(f.asset, "en")
		if err := f.files.Upload(f.ctx, key, strings.NewReader(vtt), "text/vtt"); err != nil {
			f.t.Fatal(err)
		}
		f.t.Cleanup(func() { f.files.Delete(f.ctx, key) })
		f.st.CreateArtifact(f.ctx, domain.Artifact{ID: domain.NewID(), AssetID: f.asset, JobID: f.jobs[jobcontracts.JobTypeTranscribeAudio].ID,
			Type: domain.ArtifactSubtitles, StorageKey: key, MetadataJSON: []byte(`{"language":"en"}`)})
	}
	return clip
}

func (f *fixture) exportClip(clip domain.Clip, format string, withCaptions bool) (string, error) {
	f.t.Helper()
	key := storage.ClipKey(f.asset, clip.ID, format)
	f.t.Cleanup(func() { f.files.Delete(f.ctx, key) })
	err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
		AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: format, Captions: withCaptions, OutputKey: key,
	})
	if err != nil {
		return "", err
	}
	local := filepath.Join(f.t.TempDir(), format+".mp4")
	if err := f.files.DownloadFile(f.ctx, key, local); err != nil {
		f.t.Fatal(err)
	}
	return local, nil
}

const clipVTT = "WEBVTT\n\n00:00:01.000 --> 00:00:05.000\nThese words are burned into the picture.\n\n00:00:09.000 --> 00:00:11.000\nAfter the clip.\n"

func TestExportClipFormats(t *testing.T) {
	f := newFixture(t, 12, "640x360")
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips] // any job row works for logs
	clip := f.addClip(2, 6.5, clipVTT)

	for _, tc := range []struct {
		format string
		w, h   int
	}{
		{jobcontracts.ClipFormatOriginal, 640, 360},
		{jobcontracts.ClipFormatVertical, 1080, 1920},
		{jobcontracts.ClipFormatVerticalBlur, 1080, 1920},
	} {
		path, err := f.exportClip(clip, tc.format, true)
		if err != nil {
			t.Fatalf("%s: %v\nlogs:\n%s", tc.format, err, f.logs.String())
		}
		_, s, err := ffmpeg.Probe(f.ctx, path, path)
		if err != nil {
			t.Fatal(err)
		}
		if s.Width != tc.w || s.Height != tc.h || s.VideoCodec != "h264" || s.AudioCodec != "aac" ||
			s.DurationSeconds < 4.3 || s.DurationSeconds > 4.8 {
			t.Errorf("%s: got %+v, want %dx%d, h264/aac, about 4.5 s", tc.format, s, tc.w, tc.h)
		}
	}

	arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
	var formats []string
	for _, a := range arts {
		if a.Type != domain.ArtifactClip {
			continue
		}
		var meta struct {
			ClipID, Format, Language string
			Captions                 bool
			Width, Height            int
			SizeBytes                int64
		}
		json.Unmarshal(a.MetadataJSON, &meta)
		if meta.ClipID != clip.ID || !meta.Captions || meta.Language != "en" || meta.SizeBytes == 0 || a.StorageKey != storage.ClipKey(f.asset, clip.ID, meta.Format) {
			t.Errorf("artifact = %s %s", a.StorageKey, a.MetadataJSON)
		}
		formats = append(formats, meta.Format)
	}
	if len(formats) != 3 {
		t.Fatalf("clip artifacts for formats %v, want 3", formats)
	}
	if !strings.Contains(f.logs.String(), "burning in 1 line captions (en)") {
		t.Errorf("logs do not mention the captions:\n%s", f.logs.String())
	}
	if p := f.progress; len(p) == 0 || p[len(p)-1] > 99 {
		t.Errorf("progress = %v", p)
	}
}

// The captions really end up in the picture: the bottom part of a frame
// differs between exports with and without them.
func TestExportClipBurnsCaptions(t *testing.T) {
	f := newFixture(t, 8, "640x360")
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, clipVTT)

	with, err := f.exportClip(clip, jobcontracts.ClipFormatVertical, true)
	if err != nil {
		t.Fatal(err)
	}
	withFrame := grayFrame(t, with, 2)
	without, err := f.exportClip(clip, jobcontracts.ClipFormatVertical, false)
	if err != nil {
		t.Fatal(err)
	}
	withoutFrame := grayFrame(t, without, 2)

	// 108x192 frames: captions sit about 22% above the bottom.
	const w, top, bottom = 108, 120, 160
	diff := 0
	for i := top * w; i < bottom*w; i++ {
		if d := int(withFrame[i]) - int(withoutFrame[i]); d > 40 || d < -40 {
			diff++
		}
	}
	if diff < 50 {
		t.Fatalf("only %d pixels differ in the caption area; captions were not burned in", diff)
	}
}

func TestExportClipRejectsBadInput(t *testing.T) {
	f := newFixture(t, 4, "320x240")
	clip := f.addClip(0, 2, "")
	for _, in := range []jobcontracts.ExportClipInput{
		{AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: "square", OutputKey: storage.ClipKey(f.asset, clip.ID, "square")},
		{AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: "vertical", OutputKey: "assets/other/clips/x/vertical.mp4"},
		{AssetID: f.asset, ClipID: domain.NewID(), StorageKey: f.src, Format: "vertical", OutputKey: "x"},
	} {
		if err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeSuggestClips), in); err == nil {
			t.Errorf("accepted %+v", in)
		}
	}
	other := domain.NewID()
	err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeSuggestClips), jobcontracts.ExportClipInput{
		AssetID: f.asset, ClipID: other, StorageKey: f.src, Format: "vertical", OutputKey: storage.ClipKey(f.asset, other, "vertical"),
	})
	if err == nil || !strings.Contains(err.Error(), "suggested again") {
		t.Errorf("unknown clip: %v", err)
	}
}

// grayFrame returns the frame at second at, scaled to 108x192, as 8-bit gray.
func grayFrame(t *testing.T, path string, at float64) []byte {
	t.Helper()
	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-v", "error", "-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", path, "-frames:v", "1",
		"-vf", "scale=108:192", "-f", "rawvideo", "-pix_fmt", "gray", "-")
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 108*192 {
		t.Fatalf("frame has %d bytes", out.Len())
	}
	return out.Bytes()
}

// grayVideo makes a plain gray video with sound, so colors in a frame can
// only come from the captions.
func grayVideo(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=0x404040:s=640x360:r=30:d=6",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v %s", err, out)
	}
}

// colors counts the strongly yellow, violet and white pixels of the frame at
// second at, scaled to 216x384, in the rows where vertical captions sit.
func colors(t *testing.T, path string, at float64) (yellow, violet, white int) {
	t.Helper()
	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-v", "error", "-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", path, "-frames:v", "1",
		"-vf", "scale=216:384", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	px := out.Bytes()
	for i := 150 * 216 * 3; i < 330*216*3 && i+2 < len(px); i += 3 {
		r, g, b := px[i], px[i+1], px[i+2]
		switch {
		case r > 180 && g > 150 && b < 100:
			yellow++
		case r > 100 && r < 200 && g < 140 && b > 200:
			violet++
		case r > 220 && g > 220 && b > 220:
			white++
		}
	}
	return yellow, violet, white
}

func (f *fixture) saveWords(words string) {
	f.t.Helper()
	err := f.st.SaveTranscript(f.ctx, domain.Transcript{ID: domain.NewID(), AssetID: f.asset, JobID: f.jobs[jobcontracts.JobTypeTranscribeAudio].ID,
		Language: "en", Text: "These words are burned into the picture.", Words: []byte(words), Model: "test"})
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) clipArtifactStyle(clipID, format string) string {
	f.t.Helper()
	arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
	style := "missing"
	for _, a := range arts {
		var meta struct{ ClipID, Format, CaptionStyle string }
		json.Unmarshal(a.MetadataJSON, &meta)
		if a.Type == domain.ArtifactClip && meta.ClipID == clipID && meta.Format == format {
			style = meta.CaptionStyle
		}
	}
	return style
}

// Each style burns its own look: at 2.6 s "burned" is being said.
func TestExportClipCaptionStyles(t *testing.T) {
	f := newFixtureWith(t, func(path string) { grayVideo(t, path) })
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, clipVTT)
	f.saveWords(`[{"start":1.0,"end":1.4,"word":"These"},{"start":1.5,"end":1.9,"word":"words"},{"start":2.0,"end":2.4,"word":"are"},` +
		`{"start":2.5,"end":2.9,"word":"burned"},{"start":3.0,"end":3.3,"word":"into"},{"start":3.4,"end":3.6,"word":"the"},{"start":3.7,"end":4.4,"word":"picture."}]`)

	key := storage.ClipKey(f.asset, clip.ID, jobcontracts.ClipFormatVertical)
	f.t.Cleanup(func() { f.files.Delete(f.ctx, key) })
	for _, tc := range []struct {
		style                          string
		minYellow, minViolet, minWhite int
		log                            string
	}{
		{"", 40, 0, 0, "burning in highlight captions: 7 words in 2 phrases (en)"}, // the default
		{jobcontracts.CaptionStyleBox, 0, 200, 30, "burning in box captions"},
		{jobcontracts.CaptionStyleKaraoke, 100, 0, 30, "burning in karaoke captions"},
		{jobcontracts.CaptionStyleOneWord, 0, 0, 50, "burning in one-word captions"},
		{jobcontracts.CaptionStyleLines, 0, 0, 30, "burning in 2 line captions"},
	} {
		f.logs.Reset()
		err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
			AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, Captions: true,
			CaptionStyle: tc.style, OutputKey: key,
		})
		if err != nil {
			t.Fatalf("%q: %v\nlogs:\n%s", tc.style, err, f.logs.String())
		}
		if !strings.Contains(f.logs.String(), tc.log) {
			t.Errorf("%q: logs miss %q:\n%s", tc.style, tc.log, f.logs.String())
		}
		want := tc.style
		if want == "" {
			want = jobcontracts.CaptionStyleHighlight
		}
		if got := f.clipArtifactStyle(clip.ID, jobcontracts.ClipFormatVertical); got != want {
			t.Errorf("%q: caption style = %q, want %q", tc.style, got, want)
		}
		local := filepath.Join(t.TempDir(), "clip.mp4")
		if err := f.files.DownloadFile(f.ctx, key, local); err != nil {
			t.Fatal(err)
		}
		yellow, violet, white := colors(t, local, 2.6)
		ok := yellow >= tc.minYellow && violet >= tc.minViolet && white >= tc.minWhite &&
			(tc.minYellow > 0 || yellow <= 2) && (tc.minViolet > 0 || violet <= 2)
		if !ok {
			t.Errorf("%q: yellow %d, violet %d, white %d; want at least %d, %d, %d (and none of the colors not used)",
				tc.style, yellow, violet, white, tc.minYellow, tc.minViolet, tc.minWhite)
		}
	}
}

// Captions in another language come from the transcript's translation
// (docs/adr/0029); without one, the export fails and says why.
func TestExportClipTranslatedCaptions(t *testing.T) {
	f := newFixtureWith(t, func(path string) { grayVideo(t, path) })
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, clipVTT)
	f.saveWords(`[{"start":1.0,"end":1.4,"word":"These"},{"start":1.5,"end":1.9,"word":"words"},{"start":2.0,"end":2.4,"word":"are"},` +
		`{"start":2.5,"end":2.9,"word":"burned"},{"start":3.0,"end":3.3,"word":"into"},{"start":3.4,"end":3.6,"word":"the"},{"start":3.7,"end":4.4,"word":"picture."}]`)
	err := f.st.SaveTranslation(f.ctx, domain.Translation{ID: domain.NewID(), AssetID: f.asset, JobID: f.jobs[jobcontracts.JobTypeSuggestClips].ID,
		Language: "es", SourceLanguage: "en", Model: "test",
		Segments: []byte(`[{"start":1.0,"end":4.4,"text":"Estas palabras quedan en la imagen."}]`),
		Words: []byte(`[{"start":1.0,"end":1.5,"word":"Estas"},{"start":1.5,"end":2.4,"word":"palabras"},{"start":2.5,"end":2.9,"word":"quedan"},` +
			`{"start":3.0,"end":3.3,"word":"en"},{"start":3.4,"end":3.6,"word":"la"},{"start":3.7,"end":4.4,"word":"imagen."}]`)})
	if err != nil {
		t.Fatal(err)
	}
	// The translated subtitles are newer than the original ones.
	esKey := storage.SubtitlesKey(f.asset, "es")
	if err := f.files.Upload(f.ctx, esKey, strings.NewReader("WEBVTT\n\n00:00:01.000 --> 00:00:04.400\nEstas palabras quedan en la imagen.\n"), "text/vtt"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.files.Delete(f.ctx, esKey) })
	f.st.CreateArtifact(f.ctx, domain.Artifact{ID: domain.NewID(), AssetID: f.asset, JobID: f.jobs[jobcontracts.JobTypeSuggestClips].ID,
		Type: domain.ArtifactSubtitles, StorageKey: esKey, MetadataJSON: []byte(`{"language":"es","translatedFrom":"en"}`)})

	key := storage.ClipKey(f.asset, clip.ID, jobcontracts.ClipFormatVertical)
	t.Cleanup(func() { f.files.Delete(f.ctx, key) })
	for _, tc := range []struct {
		style, language, log, translatedFrom string
	}{
		{"", "es", "burning in highlight captions: 6 words in 2 phrases (es, translated from en)", "en"},
		{jobcontracts.CaptionStyleLines, "es", "burning in 2 line captions (es, translated from en)", "en"}, // one line, split for 9:16,
		// The original, though the translated subtitles are newer.
		{jobcontracts.CaptionStyleLines, "", "burning in 2 line captions (en)", ""},
		// The transcript's own language is the original.
		{jobcontracts.CaptionStyleKaraoke, "en", "burning in karaoke captions: 7 words in 2 phrases (en)", ""},
	} {
		f.logs.Reset()
		err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
			AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, Captions: true,
			CaptionStyle: tc.style, CaptionLanguage: tc.language, OutputKey: key,
		})
		if err != nil {
			t.Fatalf("%s %q: %v\nlogs:\n%s", tc.style, tc.language, err, f.logs.String())
		}
		if !strings.Contains(f.logs.String(), tc.log) {
			t.Errorf("%s %q: logs miss %q:\n%s", tc.style, tc.language, tc.log, f.logs.String())
		}
		arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
		var meta struct{ Language, TranslatedFrom string }
		json.Unmarshal(arts[len(arts)-1].MetadataJSON, &meta)
		wantLanguage := tc.language
		if wantLanguage == "" {
			wantLanguage = "en"
		}
		if meta.Language != wantLanguage || meta.TranslatedFrom != tc.translatedFrom {
			t.Errorf("%s %q: file metadata says %+v", tc.style, tc.language, meta)
		}
	}

	err = f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
		AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, Captions: true,
		CaptionLanguage: "fr", OutputKey: key,
	})
	if err == nil || !strings.Contains(err.Error(), "no fr translation of the transcript") {
		t.Errorf("without a translation: err = %v", err)
	}
	err = f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
		AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, Captions: true,
		CaptionLanguage: "ja", OutputKey: key,
	})
	if err == nil || !strings.Contains(err.Error(), `caption_language "ja"`) {
		t.Errorf("an unknown language: err = %v", err)
	}
}

// count counts the pixels of rows [from, to) of the frame at second at,
// scaled to 216x384, that match a color.
func count(t *testing.T, path string, at float64, from, to int, match func(r, g, b byte) bool) int {
	t.Helper()
	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-v", "error", "-ss", strconv.FormatFloat(at, 'f', 2, 64), "-i", path, "-frames:v", "1",
		"-vf", "scale=216:384", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	px, n := out.Bytes(), 0
	for i := from * 216 * 3; i < to*216*3 && i+2 < len(px); i += 3 {
		if match(px[i], px[i+1], px[i+2]) {
			n++
		}
	}
	return n
}

// Position, color and capitals: at 2.6 s "burned" is being said.
func TestExportClipCaptionOptions(t *testing.T) {
	f := newFixtureWith(t, func(path string) { grayVideo(t, path) })
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, clipVTT)
	f.saveWords(`[{"start":1.0,"end":1.4,"word":"These"},{"start":1.5,"end":1.9,"word":"words"},{"start":2.0,"end":2.4,"word":"are"},` +
		`{"start":2.5,"end":2.9,"word":"burned"},{"start":3.0,"end":3.3,"word":"into"},{"start":3.4,"end":3.6,"word":"the"},{"start":3.7,"end":4.4,"word":"picture."}]`)
	key := storage.ClipKey(f.asset, clip.ID, jobcontracts.ClipFormatVertical)
	f.t.Cleanup(func() { f.files.Delete(f.ctx, key) })
	green := func(r, g, b byte) bool { return g > 180 && r < 120 && b < 150 }
	pink := func(r, g, b byte) bool { return r > 200 && g < 130 && b > 140 }
	white := func(r, g, b byte) bool { return r > 220 && g > 220 && b > 220 }
	yes := true

	for _, tc := range []struct {
		name, style, position, color string
		caps                         *bool
		accent                       func(r, g, b byte) bool
		rows                         [2]int // where the captions must be
		meta                         string // position, color and capitals recorded
		log                          string
	}{
		{"green at the top", jobcontracts.CaptionStyleHighlight, "top", "green", nil, green, [2]int{30, 120},
			"top green false", "at the top, green"},
		{"pink box in the middle, in capitals", jobcontracts.CaptionStyleBox, "middle", "pink", &yes, pink, [2]int{150, 235},
			"middle pink true", "at the middle, pink, in capitals"},
	} {
		f.logs.Reset()
		err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
			AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, Captions: true,
			CaptionStyle: tc.style, CaptionPosition: tc.position, CaptionColor: tc.color, CaptionCaps: tc.caps, OutputKey: key,
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !strings.Contains(f.logs.String(), tc.log) {
			t.Errorf("%s: logs miss %q:\n%s", tc.name, tc.log, f.logs.String())
		}
		arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
		var meta struct {
			CaptionPosition, CaptionColor string
			CaptionCaps                   bool
		}
		json.Unmarshal(arts[len(arts)-1].MetadataJSON, &meta)
		if got := fmt.Sprint(meta.CaptionPosition, " ", meta.CaptionColor, " ", meta.CaptionCaps); got != tc.meta {
			t.Errorf("%s: recorded %q, want %q", tc.name, got, tc.meta)
		}
		local := filepath.Join(t.TempDir(), "clip.mp4")
		if err := f.files.DownloadFile(f.ctx, key, local); err != nil {
			t.Fatal(err)
		}
		inside := count(t, local, 2.6, tc.rows[0], tc.rows[1], tc.accent)
		outside := count(t, local, 2.6, 0, tc.rows[0], tc.accent) + count(t, local, 2.6, tc.rows[1], 384, tc.accent) +
			count(t, local, 2.6, 0, tc.rows[0], white) + count(t, local, 2.6, tc.rows[1], 384, white)
		if inside < 40 || outside > 5 {
			t.Errorf("%s: %d accent pixels where the captions should be, %d caption pixels elsewhere", tc.name, inside, outside)
		}
	}
}

// Transcripts made before word timings were saved get line captions.
func TestExportClipWordCaptionsFallBackToLines(t *testing.T) {
	f := newFixture(t, 6, "320x240")
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, clipVTT)
	f.saveWords(`[]`)
	if _, err := f.exportClip(clip, jobcontracts.ClipFormatOriginal, true); err != nil {
		t.Fatal(err)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, "has no word timings") || !strings.Contains(logs, "line captions (en)") {
		t.Errorf("logs:\n%s", logs)
	}
	if got := f.clipArtifactStyle(clip.ID, jobcontracts.ClipFormatOriginal); got != jobcontracts.CaptionStyleLines {
		t.Errorf("caption style = %q, want lines", got)
	}
	bad := jobcontracts.ExportClipInput{AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: "vertical",
		CaptionStyle: "comic-sans", OutputKey: storage.ClipKey(f.asset, clip.ID, "vertical")}
	if err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), bad); err == nil {
		t.Error("accepted an unknown caption style")
	}
}

// whiteCenter returns the mean column of the white pixels in the frame at
// second at, scaled to 108x192, or -1 when there are none.
func whiteCenter(t *testing.T, path string, at float64) float64 {
	t.Helper()
	px := grayFrame(t, path, at)
	sum, n := 0, 0
	for i, v := range px {
		if v > 200 {
			sum += i % 108
			n++
		}
	}
	if n == 0 {
		return -1
	}
	return float64(sum) / float64(n)
}

// The "vertical-face" crop follows the track: a white square moving right
// stays in the middle of the clip, where the centre crop would lose it.
func TestExportClipFollowsTheFaceTrack(t *testing.T) {
	f := newFixtureWith(t, func(path string) {
		// overlay, not drawbox: in drawbox, t is the box's thickness, not the time.
		cmd := exec.Command("ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=0x404040:s=1280x720:r=25:d=6",
			"-f", "lavfi", "-i", "color=c=white:s=120x120:r=25:d=6", "-f", "lavfi", "-i", "sine=duration=6",
			"-filter_complex", "[0][1]overlay=x='300+t*130':y=300", "-map", "2:a",
			"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", path)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("ffmpeg not available: %v %s", err, out)
		}
	})
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 5, "")
	// The square's centre: (300 + 60 + 130 t) / 1280 of the width.
	track := `{"coverage":1,"width":1280,"height":720,"points":[{"t":0,"x":0.28125},{"t":5,"x":0.7890625}]}`
	if err := f.st.SetFaceTrack(f.ctx, clip.ID, []byte(track)); err != nil {
		t.Fatal(err)
	}
	clip, _ = f.st.GetClip(f.ctx, clip.ID)

	face, err := f.exportClip(clip, jobcontracts.ClipFormatVerticalFace, false)
	if err != nil {
		t.Fatalf("%v\nlogs:\n%s", err, f.logs.String())
	}
	for _, at := range []float64{0.5, 4.5} {
		if c := whiteCenter(t, face, at); c < 46 || c > 62 {
			t.Errorf("at %.1fs the square is at column %.0f of 108, want the middle (54)", at, c)
		}
	}
	if !strings.Contains(f.logs.String(), "following the face (found in 100% of the clip, 2 camera points)") {
		t.Errorf("logs:\n%s", f.logs.String())
	}
	var meta struct{ Framing string }
	arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
	for _, a := range arts {
		if a.Type == domain.ArtifactClip {
			json.Unmarshal(a.MetadataJSON, &meta)
		}
	}
	if meta.Framing != "face" {
		t.Errorf("framing = %q, want face", meta.Framing)
	}

	// The centre crop does not keep the square in the middle.
	center, err := f.exportClip(clip, jobcontracts.ClipFormatVertical, false)
	if err != nil {
		t.Fatal(err)
	}
	if c := whiteCenter(t, center, 4.5); c >= 0 && c > 40 && c < 68 {
		t.Errorf("the centre crop also centres the square (column %.0f); the test proves nothing", c)
	}
}

// Without a track, or with too little face, the face crop uses the centre.
func TestExportClipFaceFallsBackToCentre(t *testing.T) {
	f := newFixture(t, 4, "640x360")
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(0, 3, "")
	if _, err := f.exportClip(clip, jobcontracts.ClipFormatVerticalFace, false); err != nil {
		t.Fatal(err)
	}
	f.st.SetFaceTrack(f.ctx, clip.ID, []byte(`{"coverage":0.1,"width":640,"height":360,"points":[]}`))
	clip, _ = f.st.GetClip(f.ctx, clip.ID)
	if _, err := f.exportClip(clip, jobcontracts.ClipFormatVerticalFace, false); err != nil {
		t.Fatal(err)
	}
	logs := f.logs.String()
	for _, want := range []string{"has no face track", "a face was found in only 10% of the clip; using the centre"} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs miss %q:\n%s", want, logs)
		}
	}
}

// putVignette makes a plain-colored video, stores it in the vignette
// library and returns its key.
func (f *fixture) putVignette(color, size string, seconds float64) string {
	f.t.Helper()
	path := filepath.Join(f.t.TempDir(), color+".mp4")
	out, err := exec.Command("ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i",
		fmt.Sprintf("color=c=%s:s=%s:r=30:d=%.2f", color, size, seconds), "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", path).CombinedOutput()
	if err != nil {
		f.t.Fatalf("%v: %s", err, out)
	}
	key := storage.VignetteKey(domain.NewID())
	if err := f.files.UploadFile(f.ctx, key, path, "video/mp4"); err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { f.files.Delete(context.Background(), key) })
	return key
}

// The opening and closing in the clip's orientation are used when there is
// one, else the other version, fitted; the file records them.
func TestExportClipWithVignettes(t *testing.T) {
	f := newFixture(t, 6, "640x360")
	f.jobs[jobcontracts.JobTypeExportClip] = f.jobs[jobcontracts.JobTypeSuggestClips]
	clip := f.addClip(1, 4, "")
	introH := f.putVignette("red", "1280x720", 1)
	outroH, outroV := f.putVignette("green", "1280x720", 1.5), f.putVignette("blue", "720x1280", 2)

	key := storage.ClipKey(f.asset, clip.ID, jobcontracts.ClipFormatVertical)
	t.Cleanup(func() { f.files.Delete(context.Background(), key) })
	err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), jobcontracts.ExportClipInput{
		AssetID: f.asset, ClipID: clip.ID, StorageKey: f.src, Format: jobcontracts.ClipFormatVertical, OutputKey: key,
		Intro: &jobcontracts.Vignette{Horizontal: introH}, Outro: &jobcontracts.Vignette{Horizontal: outroH, Vertical: outroV},
		Preset: "My channel",
	})
	if err != nil {
		t.Fatalf("%v\nlogs:\n%s", err, f.logs.String())
	}
	for _, want := range []string{`with the preset \"My channel\"`, "joining the opening: 1280x720, 1.0 s, silent (horizontal version, fitted)",
		"joining the closing: 720x1280, 2.0 s, silent (vertical version)"} {
		if !strings.Contains(f.logs.String(), want) {
			t.Errorf("logs miss %q:\n%s", want, f.logs.String())
		}
	}
	arts, _ := f.st.ListArtifacts(f.ctx, f.asset)
	var meta struct {
		Preset                        string
		Intro, Outro                  bool
		DurationSeconds, TotalSeconds float64
	}
	json.Unmarshal(arts[len(arts)-1].MetadataJSON, &meta)
	if meta.Preset != "My channel" || !meta.Intro || !meta.Outro || meta.DurationSeconds != 3 || meta.TotalSeconds != 6 {
		t.Errorf("file metadata = %+v", meta)
	}
	local := filepath.Join(t.TempDir(), "clip.mp4")
	f.files.DownloadFile(f.ctx, key, local)
	_, info, _ := ffmpeg.Probe(f.ctx, local, "clip.mp4")
	if info.Width != 1080 || info.Height != 1920 || info.DurationSeconds < 5.9 || info.DurationSeconds > 6.15 || info.AudioCodec == "" {
		t.Errorf("clip = %+v; want 1080x1920, 6 s, with sound", info)
	}

	for name, tc := range map[string]struct {
		in   jobcontracts.ExportClipInput
		want string
	}{
		"a key outside the library": {jobcontracts.ExportClipInput{Intro: &jobcontracts.Vignette{Vertical: f.src}}, "invalid input: opening vignette"},
		"no version":                {jobcontracts.ExportClipInput{Outro: &jobcontracts.Vignette{}}, "invalid input: closing vignette"},
		"a deleted vignette":        {jobcontracts.ExportClipInput{Intro: &jobcontracts.Vignette{Vertical: storage.VignetteKey(domain.NewID())}}, "could not get the opening vignette"},
		"too long":                  {jobcontracts.ExportClipInput{Outro: &jobcontracts.Vignette{Vertical: f.putVignette("white", "720x1280", 31)}}, "the closing vignette lasts 31.0 s; vignettes can last up to 30 s"},
	} {
		in := tc.in
		in.AssetID, in.ClipID, in.StorageKey, in.Format, in.OutputKey = f.asset, clip.ID, f.src, jobcontracts.ClipFormatVertical, key
		if err := f.media.ExportClip(f.ctx, f.job(jobcontracts.JobTypeExportClip), in); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.want)
		}
	}
}
