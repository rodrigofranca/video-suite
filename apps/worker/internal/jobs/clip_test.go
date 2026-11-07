package jobs

import (
	"bytes"
	"encoding/json"
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
	if !strings.Contains(f.logs.String(), "burning in 1 captions (en)") {
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
