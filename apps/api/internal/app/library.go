package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // watermark formats
	_ "image/png"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
)

// Library adds files to the brand library and removes them: the vignettes
// joined to exports and the watermark images laid over them (docs/adr/0030,
// 0031). Adding and removing work the same for every kind of file; what
// differs, the checks, the limits and where the file goes, comes from the
// FileKind (a strategy).
type Library struct {
	Store LibraryStore
	Files Files
	Log   *slog.Logger
	clock
}

// LibraryStore reads and writes the library's records.
type LibraryStore interface {
	Brand
	LibraryWriter
}

// FileKind is one kind of library file.
type FileKind interface {
	// What names the kind in logs ("vignette").
	What() string
	// MaxBytes is the largest file of the kind, and TooLarge the message
	// for a bigger one.
	MaxBytes() int64
	TooLarge() string
	// accept checks an uploaded file and returns what to store: the body
	// to stream, its content type and key, and its picture size when known.
	accept(id, filename string, body io.Reader) (incoming, error)
	// create records a stored file; key reads its storage key; remove
	// deletes its record (a *domain.InUseError while presets use it).
	create(ctx context.Context, s LibraryStore, f libraryFile) error
	key(ctx context.Context, s LibraryStore, id string) (string, error)
	remove(ctx context.Context, s LibraryStore, id string) error
}

// incoming is an accepted file on its way to storage.
type incoming struct {
	body          *countingReader
	contentType   string
	key           string
	width, height int
}

// libraryFile is a stored file, to record.
type libraryFile struct {
	ID, Name, Filename, Key string
	Size                    int64
	Width, Height           int
}

// The kinds on offer.
var (
	Vignettes  FileKind = vignettes{}
	Watermarks FileKind = watermarks{}
)

// Add stores a file of a kind with a name (the file name without its
// extension when empty) and returns its ID.
func (l Library) Add(ctx context.Context, kind FileKind, name, filename string, body io.Reader) (string, error) {
	filename = CleanFilename(filename)
	if name == "" {
		name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	clean, ok := cleanName(name)
	if !ok {
		return "", fail(Invalid, fmt.Sprintf("the name must have 1 to %d characters", maxName))
	}
	id := l.newID()
	in, err := kind.accept(id, filename, body)
	if err != nil {
		return "", err
	}
	log := orDiscard(l.Log)
	err = l.Files.Upload(ctx, in.key, in.body, in.contentType)
	if err == nil && in.body.err != nil {
		err = in.body.err
	}
	if err == nil {
		err = kind.create(ctx, l.Store, libraryFile{ID: id, Name: clean, Filename: filename, Key: in.key, Size: in.body.n,
			Width: in.width, Height: in.height})
	}
	if err != nil {
		removeFile(ctx, l.Files, log, in.key)
		if errors.Is(in.body.err, errTooLarge) {
			return "", fail(TooLarge, kind.TooLarge())
		}
		if in.body.err != nil {
			return "", &Error{Kind: Unavailable, Msg: "upload failed", Err: in.body.err}
		}
		return "", err
	}
	log.Info(kind.What()+" uploaded", kind.What()+"_id", id, "bytes", in.body.n)
	return id, nil
}

// Remove deletes a file from the library and storage. It answers Conflict
// while a preset uses it; exports made with it keep their files.
func (l Library) Remove(ctx context.Context, kind FileKind, id string) error {
	if !domain.IsID(id) {
		return fail(NotFound, "not found")
	}
	key, err := kind.key(ctx, l.Store, id)
	if err == nil {
		err = kind.remove(ctx, l.Store, id)
	}
	var inUse *domain.InUseError
	switch {
	case isNotFound(err):
		return fail(NotFound, "not found")
	case errors.As(err, &inUse):
		return &Error{Kind: Conflict, Msg: err.Error(), Err: err}
	case err != nil:
		return err
	}
	log := orDiscard(l.Log)
	removeFile(ctx, l.Files, log, key)
	log.Info(kind.What()+" deleted", kind.What()+"_id", id)
	return nil
}

// ---- Vignettes: MP4 files, streamed ----

type vignettes struct{}

func (vignettes) What() string    { return "vignette" }
func (vignettes) MaxBytes() int64 { return domain.MaxVignetteBytes }
func (k vignettes) TooLarge() string {
	return fmt.Sprintf("a vignette can be up to %d MB", k.MaxBytes()>>20)
}

func (vignettes) accept(id, filename string, body io.Reader) (incoming, error) {
	if !strings.EqualFold(filepath.Ext(filename), ".mp4") {
		return incoming{}, fail(Unsupported, "only .mp4 files are accepted")
	}
	head := make([]byte, 12)
	if _, err := io.ReadFull(body, head); err != nil || !isMP4(head) {
		return incoming{}, fail(Unsupported, "the file is not an MP4 video")
	}
	return incoming{
		body:        &countingReader{r: io.MultiReader(bytes.NewReader(head), body), limit: domain.MaxVignetteBytes},
		contentType: "video/mp4", key: storage.VignetteKey(id),
	}, nil
}

func (vignettes) create(ctx context.Context, s LibraryStore, f libraryFile) error {
	return s.CreateVignette(ctx, domain.Vignette{ID: f.ID, Name: f.Name, Filename: f.Filename, StorageKey: f.Key, SizeBytes: f.Size})
}

func (vignettes) key(ctx context.Context, s LibraryStore, id string) (string, error) {
	v, err := s.GetVignette(ctx, id)
	return v.StorageKey, err
}

func (vignettes) remove(ctx context.Context, s LibraryStore, id string) error {
	return s.DeleteVignette(ctx, id)
}

// ---- Watermarks: PNG or JPEG images, read whole to check them ----

const (
	maxWatermarkBytes = 5 << 20
	maxWatermarkSide  = 4096
)

type watermarks struct{}

func (watermarks) What() string    { return "watermark" }
func (watermarks) MaxBytes() int64 { return maxWatermarkBytes }
func (k watermarks) TooLarge() string {
	return fmt.Sprintf("a watermark image can be up to %d MB", k.MaxBytes()>>20)
}

func (k watermarks) accept(id, _ string, body io.Reader) (incoming, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxWatermarkBytes+1))
	if len(data) > maxWatermarkBytes {
		return incoming{}, fail(TooLarge, k.TooLarge())
	}
	if err != nil {
		return incoming{}, &Error{Kind: Invalid, Msg: "could not read the file", Err: err}
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return incoming{}, fail(Unsupported, "the file is not a PNG or JPEG image")
	}
	if cfg.Width > maxWatermarkSide || cfg.Height > maxWatermarkSide || cfg.Width < 8 || cfg.Height < 8 {
		return incoming{}, fail(Invalid, fmt.Sprintf("the image is %dx%d; it must be 8 to %d px a side", cfg.Width, cfg.Height, maxWatermarkSide))
	}
	ext, contentType := ".png", "image/png"
	if format == "jpeg" {
		ext, contentType = ".jpg", "image/jpeg"
	}
	return incoming{
		body:        &countingReader{r: bytes.NewReader(data), limit: maxWatermarkBytes},
		contentType: contentType, key: storage.WatermarkKey(id, ext), width: cfg.Width, height: cfg.Height,
	}, nil
}

func (watermarks) create(ctx context.Context, s LibraryStore, f libraryFile) error {
	return s.CreateWatermark(ctx, domain.Watermark{ID: f.ID, Name: f.Name, Filename: f.Filename, StorageKey: f.Key,
		SizeBytes: f.Size, Width: f.Width, Height: f.Height})
}

func (watermarks) key(ctx context.Context, s LibraryStore, id string) (string, error) {
	w, err := s.GetWatermark(ctx, id)
	return w.StorageKey, err
}

func (watermarks) remove(ctx context.Context, s LibraryStore, id string) error {
	return s.DeleteWatermark(ctx, id)
}
