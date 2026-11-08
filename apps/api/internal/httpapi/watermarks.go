package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // accepted watermark formats
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
)

// Watermark images (docs/adr/0031): PNG or JPEG files in the private
// library, laid over exported clips by presets.

const (
	maxWatermarkBytes = 5 << 20
	maxWatermarkSide  = 4096
)

type watermarkDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"sizeBytes"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
	// UsedBy names the presets that use the image; it cannot be deleted
	// while they do.
	UsedBy []string `json:"usedBy"`
}

func toWatermark(w domain.Watermark, usedBy []string) watermarkDTO {
	return watermarkDTO{ID: w.ID, Name: w.Name, Filename: w.Filename, SizeBytes: w.SizeBytes, Width: w.Width, Height: w.Height,
		CreatedAt: w.CreatedAt, UsedBy: usedBy}
}

// uploadWatermark handles POST /watermarks/upload (multipart/form-data: an
// optional "name" field, then the "file" field with a PNG or JPEG of up to
// 5 MB and 4096 px a side).
func (a *api) uploadWatermark(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxWatermarkBytes+multipartOverhead)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data with a file field")
		return
	}
	name := ""
	for {
		part, err := mr.NextPart()
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "a watermark image can be up to 5 MB")
				return
			}
			writeError(w, http.StatusBadRequest, "missing file field")
			return
		}
		switch part.FormName() {
		case "name":
			b, _ := io.ReadAll(io.LimitReader(part, 1024))
			name = string(b)
			continue
		case "file":
		default:
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, maxWatermarkBytes+1))
		var maxErr *http.MaxBytesError
		if len(data) > maxWatermarkBytes || errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "a watermark image can be up to 5 MB")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "could not read the file")
			return
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "png" && format != "jpeg") {
			writeError(w, http.StatusUnsupportedMediaType, "the file is not a PNG or JPEG image")
			return
		}
		if cfg.Width > maxWatermarkSide || cfg.Height > maxWatermarkSide || cfg.Width < 8 || cfg.Height < 8 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("the image is %dx%d; it must be 8 to %d px a side", cfg.Width, cfg.Height, maxWatermarkSide))
			return
		}
		filename := cleanFilename(part.FileName())
		if name == "" {
			name = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
		clean, ok := cleanName(name)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("the name must have 1 to %d characters", maxName))
			return
		}
		ext, contentType := ".png", "image/png"
		if format == "jpeg" {
			ext, contentType = ".jpg", "image/jpeg"
		}
		wm := domain.Watermark{ID: domain.NewID(), Name: clean, Filename: filename, SizeBytes: int64(len(data)), Width: cfg.Width, Height: cfg.Height}
		wm.StorageKey = storage.WatermarkKey(wm.ID, ext)
		ctx := r.Context()
		if err := a.Storage.Upload(ctx, wm.StorageKey, bytes.NewReader(data), contentType); err != nil {
			a.internalError(w, r, err)
			return
		}
		if err := a.Store.CreateWatermark(ctx, wm); err != nil {
			a.removeFile(ctx, wm.StorageKey)
			a.internalError(w, r, err)
			return
		}
		saved, err := a.Store.GetWatermark(ctx, wm.ID)
		if err != nil {
			a.internalError(w, r, err)
			return
		}
		a.Log.Info("watermark uploaded", "watermark_id", wm.ID, "width", cfg.Width, "height", cfg.Height)
		writeJSON(w, http.StatusCreated, toWatermark(saved, []string{}))
		return
	}
}

// listWatermarks handles GET /watermarks: the images, newest first, each
// with the presets that use it.
func (a *api) listWatermarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	images, err := a.Store.ListWatermarks(ctx)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	presets, err := a.Store.ListPresets(ctx)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]watermarkDTO, 0, len(images))
	for _, img := range images {
		used := []string{}
		for _, p := range presets {
			if p.Watermark == img.ID {
				used = append(used, p.Name)
			}
		}
		out = append(out, toWatermark(img, used))
	}
	writeJSON(w, http.StatusOK, map[string]any{"watermarks": out})
}

// watermarkFile handles GET /watermarks/{id}/file: it redirects to a
// short-lived signed link, for previews. The library is private.
func (a *api) watermarkFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	img, err := a.Store.GetWatermark(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	url, err := a.Storage.PresignForBrowser(r.Context(), img.StorageKey, "", downloadLinkExpiry)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // the link expires
	http.Redirect(w, r, url, http.StatusFound)
}

// deleteWatermark handles DELETE /watermarks/{id}: 204, or 409 while a
// preset uses it. Exports made with it keep their files.
func (a *api) deleteWatermark(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ctx := r.Context()
	img, err := a.Store.GetWatermark(ctx, id)
	if err == nil {
		err = a.Store.DeleteWatermark(ctx, id)
	}
	var inUse *store.InUseError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
		return
	case errors.As(err, &inUse):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		a.internalError(w, r, err)
		return
	}
	a.removeFile(ctx, img.StorageKey)
	a.Log.Info("watermark deleted", "watermark_id", id)
	w.WriteHeader(http.StatusNoContent)
}
