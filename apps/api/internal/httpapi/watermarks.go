package httpapi

import (
	"errors"
	_ "image/jpeg" // accepted watermark formats
	_ "image/png"
	"net/http"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
)

// Watermark images (docs/adr/0031): PNG or JPEG files in the private
// library, laid over exported clips by presets.

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
