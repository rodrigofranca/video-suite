package httpapi

import (
	"context"
	"net/http"

	"ultramedia/api/internal/app"
)

// The brand library's uploads and deletes, the same for every kind of file:
// app.Library checks and stores the file, by its kind (docs/adr/0033).

// uploadVignette handles POST /vignettes/upload (multipart/form-data: an
// optional "name" field, then the "file" field with an MP4 of up to
// domain.MaxVignetteBytes).
func (a *api) uploadVignette(w http.ResponseWriter, r *http.Request) {
	a.addToLibrary(w, r, app.Vignettes, func(ctx context.Context, id string) (any, error) {
		v, err := a.Store.GetVignette(ctx, id)
		return vignetteDTO{ID: v.ID, Name: v.Name, Filename: v.Filename, SizeBytes: v.SizeBytes, CreatedAt: v.CreatedAt, UsedBy: []string{}}, err
	})
}

// uploadWatermark handles POST /watermarks/upload (multipart/form-data: an
// optional "name" field, then the "file" field with a PNG or JPEG of up to
// 5 MB and 4096 px a side).
func (a *api) uploadWatermark(w http.ResponseWriter, r *http.Request) {
	a.addToLibrary(w, r, app.Watermarks, func(ctx context.Context, id string) (any, error) {
		img, err := a.Store.GetWatermark(ctx, id)
		return toWatermark(img, []string{}), err
	})
}

// deleteVignette handles DELETE /vignettes/{id}: 204, or 409 while a
// preset uses it. Exports made with it keep their files.
func (a *api) deleteVignette(w http.ResponseWriter, r *http.Request) {
	a.noContent(w, r, a.library.Remove(r.Context(), app.Vignettes, r.PathValue("id")))
}

// deleteWatermark handles DELETE /watermarks/{id}, like deleteVignette.
func (a *api) deleteWatermark(w http.ResponseWriter, r *http.Request) {
	a.noContent(w, r, a.library.Remove(r.Context(), app.Watermarks, r.PathValue("id")))
}

// addToLibrary answers 201 with the new file as read reads it.
func (a *api) addToLibrary(w http.ResponseWriter, r *http.Request, kind app.FileKind, read func(context.Context, string) (any, error)) {
	tooLarge := func() { writeError(w, http.StatusRequestEntityTooLarge, kind.TooLarge()) }
	name, part, ok := fileField(w, r, kind.MaxBytes(), tooLarge)
	if !ok {
		return
	}
	id, err := a.library.Add(r.Context(), kind, name, part.FileName(), part)
	if isMaxBytes(err) {
		tooLarge()
		return
	}
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	saved, err := read(r.Context(), id)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}
