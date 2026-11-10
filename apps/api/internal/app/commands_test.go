package app

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// The other commands, on the same fakes as the export, plus the writes and
// the files.

// fakeWriter records the writes on top of fakeStore's reads.
type fakeWriter struct {
	*fakeStore
	status    map[string]string // asset ID → status
	completed []domain.Job
	source    domain.Source
	deleted   []string // asset, clip and library IDs
	edited    *domain.Clip
	retrack   *domain.Job
	metadata  []byte
	nameTaken bool
	inUse     bool
}

func newWriter() *fakeWriter {
	return &fakeWriter{fakeStore: newFake(), status: map[string]string{}}
}

func (f *fakeWriter) CreateAsset(_ context.Context, a domain.Asset) error {
	f.assets[a.ID] = a
	f.status[a.ID] = a.Status
	return nil
}
func (f *fakeWriter) CompleteUpload(_ context.Context, src domain.Source, jobs []domain.Job) error {
	f.source, f.completed = src, jobs
	f.created = append(f.created, jobs...)
	f.queued = append(f.queued, jobs[0])
	return nil
}
func (f *fakeWriter) DeleteAsset(_ context.Context, id string) error {
	delete(f.assets, id)
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeWriter) DeleteAssetIfIdle(ctx context.Context, id string) error {
	if f.busy {
		return domain.ErrBusy
	}
	return f.DeleteAsset(ctx, id)
}
func (f *fakeWriter) GetTechnicalMetadata(_ context.Context, id string) (domain.TechnicalMetadata, error) {
	if f.metadata == nil {
		return domain.TechnicalMetadata{}, domain.ErrNotFound
	}
	return domain.TechnicalMetadata{AssetID: id, MetadataJSON: f.metadata}, nil
}
func (f *fakeWriter) EditClipIfIdle(_ context.Context, c domain.Clip, retrack *domain.Job) error {
	if f.busy {
		return domain.ErrBusy
	}
	f.edited, f.retrack, f.clips[c.ID] = &c, retrack, c
	if retrack != nil {
		f.created = append(f.created, *retrack)
		f.queued = append(f.queued, *retrack)
	}
	return nil
}
func (f *fakeWriter) DeleteClipIfIdle(_ context.Context, _, clipID string) error {
	if f.busy {
		return domain.ErrBusy
	}
	if _, ok := f.clips[clipID]; !ok {
		return domain.ErrNotFound
	}
	delete(f.clips, clipID)
	f.deleted = append(f.deleted, clipID)
	return nil
}
func (f *fakeWriter) CreatePreset(_ context.Context, p domain.Preset) error {
	if f.nameTaken {
		return domain.ErrNameTaken
	}
	f.presets[p.ID] = p
	return nil
}
func (f *fakeWriter) UpdatePreset(ctx context.Context, p domain.Preset) error {
	return f.CreatePreset(ctx, p)
}
func (f *fakeWriter) DeletePreset(_ context.Context, id string) error {
	if _, ok := f.presets[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.presets, id)
	return nil
}
func (f *fakeWriter) CreateVignette(_ context.Context, v domain.Vignette) error {
	f.vignettes[v.ID] = v
	return nil
}
func (f *fakeWriter) DeleteVignette(_ context.Context, id string) error {
	if f.inUse {
		return &domain.InUseError{What: "vignette", Presets: []string{"Channel"}}
	}
	delete(f.vignettes, id)
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeWriter) CreateWatermark(_ context.Context, w domain.Watermark) error {
	f.watermarks[w.ID] = w
	return nil
}
func (f *fakeWriter) DeleteWatermark(_ context.Context, id string) error {
	delete(f.watermarks, id)
	f.deleted = append(f.deleted, id)
	return nil
}

// fakeFiles keeps uploaded files in memory.
type fakeFiles struct {
	files    map[string][]byte
	types    map[string]string
	prefixes []string
}

func newFiles() *fakeFiles { return &fakeFiles{files: map[string][]byte{}, types: map[string]string{}} }

func (f *fakeFiles) Upload(_ context.Context, key string, body io.Reader, contentType string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.files[key], f.types[key] = b, contentType
	return nil
}
func (f *fakeFiles) Delete(_ context.Context, key string) error {
	delete(f.files, key)
	return nil
}
func (f *fakeFiles) DeletePrefix(_ context.Context, prefix string) (int, error) {
	f.prefixes = append(f.prefixes, prefix)
	return 1, nil
}

var mp4 = []byte("\x00\x00\x00\x20ftypisom-and-then-the-video")

// ---- Upload ----

func uploader(f *fakeWriter, files *fakeFiles) UploadVideo {
	return UploadVideo{Assets: f, Files: files, MaxBytes: 1 << 10, clock: fixedClock()}
}

func TestUploadStoresTheFileAndStartsThePipeline(t *testing.T) {
	f, files := newWriter(), newFiles()
	id, err := uploader(f, files).Run(context.Background(), UploadRequest{Filename: `C:\videos\talk.MP4`, Body: bytes.NewReader(mp4)})
	if err != nil {
		t.Fatal(err)
	}
	if f.assets[id].Filename != "talk.MP4" || !bytes.Equal(files.files[storage.SourceKey(id)], mp4) {
		t.Fatalf("asset %+v, files %v", f.assets[id], files.files)
	}
	if f.source.SizeBytes != int64(len(mp4)) || len(f.source.Checksum) != 64 || len(f.completed) != len(jobcontracts.Pipeline) {
		t.Fatalf("source %+v, %d jobs", f.source, len(f.completed))
	}
	if len(f.queued) != 1 || f.queued[0].ID != f.completed[0].ID || f.queued[0].Type != jobcontracts.Pipeline[0] {
		t.Fatalf("queued %+v, want only the first job", f.queued)
	}
}

func TestUploadRejectsWhatItCannotTake(t *testing.T) {
	for name, c := range map[string]struct {
		filename string
		body     []byte
		want     Kind
	}{
		"not .mp4":    {"talk.mov", mp4, Unsupported},
		"not an MP4":  {"talk.mp4", []byte("plain text, not a video"), Unsupported},
		"over 1 KiB":  {"talk.mp4", append(mp4, make([]byte, 1<<10)...), TooLarge},
		"empty video": {"talk.mp4", nil, Unsupported},
	} {
		f, files := newWriter(), newFiles()
		before := len(f.assets)
		_, err := uploader(f, files).Run(context.Background(), UploadRequest{Filename: c.filename, Body: bytes.NewReader(c.body)})
		if kindOf(err) != c.want || len(f.assets) != before || len(files.files) != 0 || len(f.queued) != 0 {
			t.Errorf("%s: %v (kind %d), %d assets, %d files", name, err, kindOf(err), len(f.assets), len(files.files))
		}
	}
}

// ---- Delete and run again ----

func TestDeleteVideo(t *testing.T) {
	f, files := newWriter(), newFiles()
	uc := DeleteVideo{Assets: f, Writer: f, Files: files}
	f.busy = true
	if err := uc.Run(context.Background(), assetID); kindOf(err) != Conflict || !strings.Contains(err.Error(), "the video cannot be deleted") {
		t.Fatalf("busy: %v", err)
	}
	f.busy = false
	if err := uc.Run(context.Background(), assetID); err != nil || len(files.prefixes) != 1 || files.prefixes[0] != storage.AssetPrefix(assetID) {
		t.Fatalf("err %v, prefixes %v", err, files.prefixes)
	}
	for _, id := range []string{assetID, "nope"} {
		if err := uc.Run(context.Background(), id); kindOf(err) != NotFound {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestRerunJob(t *testing.T) {
	f := newWriter()
	uc := RerunJob{Assets: f, Jobs: f, clock: fixedClock()}
	if _, err := uc.Run(context.Background(), assetID, "make-coffee"); kindOf(err) != Invalid || !errors.Is(err, ErrUnknownJobType) {
		t.Fatalf("unknown type: %v", err)
	}
	if _, err := uc.Run(context.Background(), assetID, jobcontracts.JobTypeExportClip); kindOf(err) != Invalid ||
		!errors.Is(err, ErrNotRerunnable) || err.(*Error).Msg != "Clip export jobs cannot be run again by hand: they run when a clip is exported" {
		t.Fatalf("an export: %v", err)
	}
	job, err := uc.Run(context.Background(), assetID, jobcontracts.JobTypeSuggestClips)
	if err != nil || job.Type != jobcontracts.JobTypeSuggestClips || len(f.queued) != 1 || f.queued[0].ID != job.ID {
		t.Fatalf("job %+v, err %v, queued %+v", job, err, f.queued)
	}
	// The jobs that work on its result are created too, and wait for it.
	if len(f.created) != 1+len(jobcontracts.RerunWith[job.Type]) || f.created[1].Type != jobcontracts.JobTypeTrackFaces {
		t.Fatalf("created %+v, want the job and the ones after it", f.created)
	}
	delete(f.sources, assetID)
	if _, err := uc.Run(context.Background(), assetID, jobcontracts.JobTypeTranscodeHLS); kindOf(err) != Conflict {
		t.Fatalf("no source: %v", err)
	}
}

// ---- Clips ----

func editor(f *fakeWriter) EditClip {
	return EditClip{Assets: f, Clips: f, Metadata: f, Writer: f, clock: fixedClock()}
}

func ptr[T any](v T) *T { return &v }

func TestEditClip(t *testing.T) {
	f := newWriter()
	f.metadata = []byte(`{"summary":{"durationSeconds":60}}`)
	f.clips[clipID] = domain.Clip{ID: clipID, AssetID: assetID, Start: 10, End: 40, Title: "Best"}
	uc := editor(f)
	edit := func(r EditRequest) (domain.Clip, error) {
		r.AssetID, r.ClipID = assetID, clipID
		return uc.Run(context.Background(), r)
	}

	for why, r := range map[string]EditRequest{
		"nothing":        {},
		"after the end":  {End: ptr(61.0)},
		"too short":      {End: ptr(11.0)},
		"empty title":    {Title: ptr("   ")},
		"start past end": {Start: ptr(45.0)},
	} {
		if _, err := edit(r); kindOf(err) != Invalid {
			t.Errorf("%s: %v", why, err)
		}
	}
	if f.edited != nil {
		t.Fatal("an invalid edit was saved")
	}

	// The title alone: no new face track.
	c, err := edit(EditRequest{Title: ptr("  The  hook ")})
	if err != nil || c.Title != "The hook" || f.retrack != nil || len(f.queued) != 0 {
		t.Fatalf("clip %+v, err %v, retrack %v", c, err, f.retrack)
	}
	// New times, kept to the millisecond: this clip is tracked again.
	c, err = edit(EditRequest{Start: ptr(12.34567)})
	if err != nil || c.Start != 12.346 || f.retrack == nil || len(f.queued) != 1 || f.queued[0].Type != jobcontracts.JobTypeTrackFaces {
		t.Fatalf("clip %+v, err %v, queued %+v", c, err, f.queued)
	}
	// The same values again change nothing.
	f.edited = nil
	if _, err := edit(EditRequest{Start: ptr(12.346)}); err != nil || f.edited != nil {
		t.Fatalf("no-op edit: %v, saved %v", err, f.edited)
	}
	// While a job runs, edits wait.
	f.busy = true
	if _, err := edit(EditRequest{Title: ptr("Again")}); kindOf(err) != Conflict {
		t.Fatalf("busy: %v", err)
	}
	// A clip of another video is not found.
	f.busy = false
	f.clips[otherID] = domain.Clip{ID: otherID, AssetID: "elsewhere", Start: 1, End: 20, Title: "x"}
	if _, err := uc.Run(context.Background(), EditRequest{AssetID: assetID, ClipID: otherID, Title: ptr("y")}); kindOf(err) != NotFound {
		t.Fatalf("other video's clip: %v", err)
	}
}

func TestDeleteClip(t *testing.T) {
	f, files := newWriter(), newFiles()
	uc := DeleteClip{Assets: f, Writer: f, Files: files}
	if err := uc.Run(context.Background(), assetID, clipID); err != nil || files.prefixes[0] != storage.ClipPrefix(assetID, clipID) {
		t.Fatalf("err %v, prefixes %v", err, files.prefixes)
	}
	if err := uc.Run(context.Background(), assetID, clipID); kindOf(err) != NotFound {
		t.Fatalf("again: %v", err)
	}
}

// ---- Presets ----

func TestSavePreset(t *testing.T) {
	f := newWriter()
	uc := SavePreset{Brand: f, Writer: f, clock: fixedClock()}
	ctx := context.Background()

	p, err := uc.Create(ctx, PresetInput{Name: "  My   channel "})
	if err != nil || p.Name != "My channel" || !p.Captions || p.CaptionStyle != jobcontracts.CaptionStyles[0] ||
		p.WatermarkPosition != jobcontracts.WatermarkPositions[0] || p.WatermarkSize != 15 || p.WatermarkOpacity != 80 {
		t.Fatalf("defaults: %+v, %v", p, err)
	}
	for why, in := range map[string]PresetInput{
		"no name":           {Name: " "},
		"unknown style":     {Name: "x", CaptionStyle: "comic-sans"},
		"unknown vignette":  {Name: "x", IntroVertical: ptr(otherID)},
		"malformed file id": {Name: "x", WatermarkID: ptr("logo")},
		"watermark too big": {Name: "x", WatermarkSize: ptr(90)},
	} {
		if _, err := uc.Create(ctx, in); kindOf(err) != Invalid {
			t.Errorf("%s: %v", why, err)
		}
	}
	f.vignettes[otherID] = domain.Vignette{ID: otherID}
	if p, err = uc.Update(ctx, p.ID, PresetInput{Name: "Mine", IntroVertical: ptr(otherID), Captions: no()}); err != nil ||
		p.IntroVertical != otherID || p.Captions {
		t.Fatalf("update: %+v, %v", p, err)
	}
	if _, err := uc.Update(ctx, clipID, PresetInput{Name: "x"}); kindOf(err) != NotFound {
		t.Fatalf("unknown preset: %v", err)
	}
	f.nameTaken = true
	if _, err := uc.Create(ctx, PresetInput{Name: "Mine"}); kindOf(err) != Conflict || !strings.Contains(err.Error(), `"Mine"`) {
		t.Fatalf("name taken: %v", err)
	}
	del := DeletePreset{Writer: f}
	if err := del.Run(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := del.Run(ctx, p.ID); kindOf(err) != NotFound {
		t.Fatalf("again: %v", err)
	}
}

// ---- Library ----

func pngOf(t *testing.T, w, h int) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestLibraryAddsEachKindByItsRules(t *testing.T) {
	f, files := newWriter(), newFiles()
	lib := Library{Store: f, Files: files, clock: fixedClock()}
	ctx := context.Background()

	id, err := lib.Add(ctx, Vignettes, "", "Intro (final).mp4", bytes.NewReader(mp4))
	if err != nil || f.vignettes[id].Name != "Intro (final)" || f.vignettes[id].SizeBytes != int64(len(mp4)) ||
		files.types[storage.VignetteKey(id)] != "video/mp4" {
		t.Fatalf("vignette %+v, %v", f.vignettes[id], err)
	}
	id, err = lib.Add(ctx, Watermarks, "Logo", "logo.png", bytes.NewReader(pngOf(t, 64, 32)))
	w := f.watermarks[id]
	if err != nil || w.Width != 64 || w.Height != 32 || w.StorageKey != storage.WatermarkKey(id, ".png") || files.types[w.StorageKey] != "image/png" {
		t.Fatalf("watermark %+v, %v", w, err)
	}

	for why, c := range map[string]struct {
		kind     FileKind
		filename string
		body     []byte
		want     Kind
	}{
		"vignette not .mp4":   {Vignettes, "intro.mov", mp4, Unsupported},
		"vignette not an MP4": {Vignettes, "intro.mp4", []byte("not a video at all"), Unsupported},
		"watermark not image": {Watermarks, "logo.png", []byte("hello"), Unsupported},
		"watermark tiny":      {Watermarks, "logo.png", pngOf(t, 4, 4), Invalid},
		"watermark too big":   {Watermarks, "logo.png", make([]byte, 5<<20+1), TooLarge},
	} {
		before := len(files.files)
		if _, err := lib.Add(ctx, c.kind, "", c.filename, bytes.NewReader(c.body)); kindOf(err) != c.want || len(files.files) != before {
			t.Errorf("%s: %v (kind %d)", why, err, kindOf(err))
		}
	}
	if _, err := lib.Add(ctx, Vignettes, strings.Repeat("n", 61), "intro.mp4", bytes.NewReader(mp4)); kindOf(err) != Invalid {
		t.Errorf("long name: %v", err)
	}
}

func TestLibraryRemovesOnlyWhatNoPresetUses(t *testing.T) {
	f, files := newWriter(), newFiles()
	lib := Library{Store: f, Files: files, clock: fixedClock()}
	ctx := context.Background()
	id, err := lib.Add(ctx, Vignettes, "Intro", "intro.mp4", bytes.NewReader(mp4))
	if err != nil {
		t.Fatal(err)
	}
	f.inUse = true
	if err := lib.Remove(ctx, Vignettes, id); kindOf(err) != Conflict || !strings.Contains(err.Error(), "Channel") || len(files.files) != 1 {
		t.Fatalf("in use: %v, %d files", err, len(files.files))
	}
	f.inUse = false
	if err := lib.Remove(ctx, Vignettes, id); err != nil || len(files.files) != 0 {
		t.Fatalf("remove: %v, %d files", err, len(files.files))
	}
	for _, missing := range []string{id, "nope"} {
		if err := lib.Remove(ctx, Watermarks, missing); kindOf(err) != NotFound {
			t.Errorf("%s: %v", missing, err)
		}
	}
}
