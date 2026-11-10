package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// The vignette library and export presets (docs/adr/0030).

// maxName is the longest preset or vignette name, in characters.
const maxName = 60

type vignetteDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	// UsedBy names the presets that use the vignette; it cannot be deleted
	// while they do.
	UsedBy []string `json:"usedBy"`
}

type presetDTO struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Captions        bool    `json:"captions"`
	CaptionStyle    string  `json:"captionStyle"`
	CaptionPosition string  `json:"captionPosition"`
	CaptionColor    string  `json:"captionColor"`
	CaptionCaps     *bool   `json:"captionCaps"`
	CaptionLanguage string  `json:"captionLanguage"`
	IntroHorizontal *string `json:"introHorizontalId"`
	IntroVertical   *string `json:"introVerticalId"`
	OutroHorizontal *string `json:"outroHorizontalId"`
	OutroVertical   *string `json:"outroVerticalId"`
	// The watermark image (or null), its corner, its width as a percentage
	// of the clip's shorter side, and its opacity in percent (docs/adr/0031).
	WatermarkID       *string   `json:"watermarkId"`
	WatermarkPosition string    `json:"watermarkPosition"`
	WatermarkSize     int       `json:"watermarkSize"`
	WatermarkOpacity  int       `json:"watermarkOpacity"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func idOrNull(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func toPreset(p domain.Preset) presetDTO {
	return presetDTO{
		ID: p.ID, Name: p.Name, Captions: p.Captions, CaptionStyle: p.CaptionStyle, CaptionPosition: p.CaptionPosition,
		CaptionColor: p.CaptionColor, CaptionCaps: p.CaptionCaps, CaptionLanguage: p.CaptionLanguage,
		IntroHorizontal: idOrNull(p.IntroHorizontal), IntroVertical: idOrNull(p.IntroVertical),
		OutroHorizontal: idOrNull(p.OutroHorizontal), OutroVertical: idOrNull(p.OutroVertical),
		WatermarkID: idOrNull(p.Watermark), WatermarkPosition: p.WatermarkPosition, WatermarkSize: p.WatermarkSize,
		WatermarkOpacity: p.WatermarkOpacity, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// presetVignettes lists the vignette IDs a preset uses.
func presetVignettes(p domain.Preset) []string {
	var ids []string
	for _, id := range []string{p.IntroHorizontal, p.IntroVertical, p.OutroHorizontal, p.OutroVertical} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// cleanName trims a name and checks its length.
func cleanName(name string) (string, bool) {
	name = strings.Join(strings.Fields(name), " ")
	return name, name != "" && utf8.RuneCountInString(name) <= maxName
}

// ---- Vignettes ----

// uploadVignette handles POST /vignettes/upload (multipart/form-data: an
// optional "name" field, then the "file" field with an MP4 of up to
// domain.MaxVignetteBytes). The file is streamed to the private library.
func (a *api) uploadVignette(w http.ResponseWriter, r *http.Request) {
	const limit = domain.MaxVignetteBytes
	tooLarge := func() {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("a vignette can be up to %d MB", limit>>20))
	}
	if r.ContentLength > limit+multipartOverhead {
		tooLarge()
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+multipartOverhead)
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
				tooLarge()
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
		filename := cleanFilename(part.FileName())
		if name == "" {
			name = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
		clean, ok := cleanName(name)
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("the name must have 1 to %d characters", maxName))
			return
		}
		head := make([]byte, 12)
		if !strings.EqualFold(filepath.Ext(filename), ".mp4") {
			writeError(w, http.StatusUnsupportedMediaType, "only .mp4 files are accepted")
			return
		}
		if _, err := io.ReadFull(part, head); err != nil || !isMP4(head) {
			writeError(w, http.StatusUnsupportedMediaType, "the file is not an MP4 video")
			return
		}
		v := domain.Vignette{ID: domain.NewID(), Name: clean, Filename: filename}
		v.StorageKey = storage.VignetteKey(v.ID)
		ctx := r.Context()
		body := &countingReader{r: io.MultiReader(strings.NewReader(string(head)), part), hash: sha256.New(), limit: limit}
		err = a.Storage.Upload(ctx, v.StorageKey, body, "video/mp4")
		if err == nil && body.err != nil {
			err = body.err
		}
		if err == nil {
			v.SizeBytes = body.n
			err = a.Store.CreateVignette(ctx, v)
		}
		if err != nil {
			a.removeFile(ctx, v.StorageKey)
			var maxErr *http.MaxBytesError
			if errors.Is(body.err, errTooLarge) || errors.As(body.err, &maxErr) {
				tooLarge()
				return
			}
			a.internalError(w, r, err)
			return
		}
		saved, err := a.Store.GetVignette(ctx, v.ID)
		if err != nil {
			a.internalError(w, r, err)
			return
		}
		a.Log.Info("vignette uploaded", "vignette_id", v.ID, "bytes", v.SizeBytes)
		writeJSON(w, http.StatusCreated, vignetteDTO{ID: saved.ID, Name: saved.Name, Filename: saved.Filename,
			SizeBytes: saved.SizeBytes, CreatedAt: saved.CreatedAt, UsedBy: []string{}})
		return
	}
}

// removeFile deletes a stored file, even if the client went away; a failure
// only leaves an orphan file, so it is logged.
func (a *api) removeFile(ctx context.Context, key string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := a.Storage.Delete(ctx, key); err != nil {
		a.Log.Warn("could not delete a stored file", "key", key, "error", err)
	}
}

// listVignettes handles GET /vignettes: the library, newest first, each
// with the presets that use it.
func (a *api) listVignettes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vignettes, err := a.Store.ListVignettes(ctx)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	presets, err := a.Store.ListPresets(ctx)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]vignetteDTO, 0, len(vignettes))
	for _, v := range vignettes {
		used := []string{}
		for _, p := range presets {
			if slices.Contains(presetVignettes(p), v.ID) {
				used = append(used, p.Name)
			}
		}
		out = append(out, vignetteDTO{ID: v.ID, Name: v.Name, Filename: v.Filename, SizeBytes: v.SizeBytes, CreatedAt: v.CreatedAt, UsedBy: used})
	}
	writeJSON(w, http.StatusOK, map[string]any{"vignettes": out})
}

// vignetteFile handles GET /vignettes/{id}/file: it redirects to a
// short-lived signed link, for previews. The library is private.
func (a *api) vignetteFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	v, err := a.Store.GetVignette(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	url, err := a.Storage.PresignForBrowser(r.Context(), v.StorageKey, "", downloadLinkExpiry)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // the link expires
	http.Redirect(w, r, url, http.StatusFound)
}

// deleteVignette handles DELETE /vignettes/{id}: 204, or 409 while a preset
// uses it. Exports made with it keep their files.
func (a *api) deleteVignette(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ctx := r.Context()
	v, err := a.Store.GetVignette(ctx, id)
	if err == nil {
		err = a.Store.DeleteVignette(ctx, id)
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
	a.removeFile(ctx, v.StorageKey)
	a.Log.Info("vignette deleted", "vignette_id", id)
	w.WriteHeader(http.StatusNoContent)
}

// ---- Presets ----

// presetBody is what POST and PUT /presets take. Omitted caption fields get
// the defaults: captions on, the first style, at the bottom.
type presetBody struct {
	Name            string  `json:"name"`
	Captions        *bool   `json:"captions"`
	CaptionStyle    string  `json:"captionStyle"`
	CaptionPosition string  `json:"captionPosition"`
	CaptionColor    string  `json:"captionColor"`
	CaptionCaps     *bool   `json:"captionCaps"`
	CaptionLanguage string  `json:"captionLanguage"`
	IntroHorizontal *string `json:"introHorizontalId"`
	IntroVertical   *string `json:"introVerticalId"`
	OutroHorizontal *string `json:"outroHorizontalId"`
	OutroVertical   *string `json:"outroVerticalId"`
	// Omitted watermark choices get the defaults: top right, 15%, 80%.
	WatermarkID       *string `json:"watermarkId"`
	WatermarkPosition string  `json:"watermarkPosition"`
	WatermarkSize     *int    `json:"watermarkSize"`
	WatermarkOpacity  *int    `json:"watermarkOpacity"`
}

// readPreset decodes and checks a preset body. It returns the message for a
// 400 when the body is wrong.
func (a *api) readPreset(r *http.Request) (domain.Preset, string, error) {
	var b presetBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&b); err != nil {
		return domain.Preset{}, `expected a JSON body like {"name": "My channel", "captionStyle": "highlight", "introVerticalId": "<vignette id>"}`, nil
	}
	name, ok := cleanName(b.Name)
	if !ok {
		return domain.Preset{}, fmt.Sprintf("the name must have 1 to %d characters", maxName), nil
	}
	p := domain.Preset{Name: name, Captions: b.Captions == nil || *b.Captions, CaptionStyle: b.CaptionStyle,
		CaptionPosition: b.CaptionPosition, CaptionColor: b.CaptionColor, CaptionCaps: b.CaptionCaps, CaptionLanguage: b.CaptionLanguage}
	if p.CaptionStyle == "" {
		p.CaptionStyle = jobcontracts.CaptionStyles[0]
	}
	if p.CaptionPosition == "" {
		p.CaptionPosition = jobcontracts.CaptionPositions[0]
	}
	if !slices.Contains(jobcontracts.CaptionStyles, p.CaptionStyle) || !slices.Contains(jobcontracts.CaptionPositions, p.CaptionPosition) ||
		(p.CaptionColor != "" && !slices.Contains(jobcontracts.CaptionColors, p.CaptionColor)) ||
		(p.CaptionLanguage != "" && !slices.Contains(jobcontracts.CaptionLanguages, p.CaptionLanguage)) {
		return domain.Preset{}, "caption styles: " + strings.Join(jobcontracts.CaptionStyles, ", ") +
			"; positions: " + strings.Join(jobcontracts.CaptionPositions, ", ") + "; colors (or empty): " + strings.Join(jobcontracts.CaptionColors, ", ") +
			"; languages (or empty): " + strings.Join(jobcontracts.CaptionLanguages, ", "), nil
	}
	slots := []struct {
		in  *string
		out *string
	}{{b.IntroHorizontal, &p.IntroHorizontal}, {b.IntroVertical, &p.IntroVertical}, {b.OutroHorizontal, &p.OutroHorizontal}, {b.OutroVertical, &p.OutroVertical}}
	for _, s := range slots {
		if s.in == nil || *s.in == "" {
			continue
		}
		if !domain.IsID(*s.in) {
			return domain.Preset{}, fmt.Sprintf("unknown vignette %q", *s.in), nil
		}
		if _, err := a.Store.GetVignette(r.Context(), *s.in); errors.Is(err, store.ErrNotFound) {
			return domain.Preset{}, fmt.Sprintf("unknown vignette %q", *s.in), nil
		} else if err != nil {
			return domain.Preset{}, "", err
		}
		*s.out = *s.in
	}
	p.WatermarkPosition, p.WatermarkSize, p.WatermarkOpacity = b.WatermarkPosition, 15, 80
	if p.WatermarkPosition == "" {
		p.WatermarkPosition = jobcontracts.WatermarkPositions[0]
	}
	if b.WatermarkSize != nil {
		p.WatermarkSize = *b.WatermarkSize
	}
	if b.WatermarkOpacity != nil {
		p.WatermarkOpacity = *b.WatermarkOpacity
	}
	if !slices.Contains(jobcontracts.WatermarkPositions, p.WatermarkPosition) ||
		p.WatermarkSize < jobcontracts.MinWatermarkSize || p.WatermarkSize > jobcontracts.MaxWatermarkSize ||
		p.WatermarkOpacity < jobcontracts.MinWatermarkOpacity || p.WatermarkOpacity > jobcontracts.MaxWatermarkOpacity {
		return domain.Preset{}, fmt.Sprintf("watermark positions: %s; size: %d to %d (%% of the shorter side); opacity: %d to %d (%%)",
			strings.Join(jobcontracts.WatermarkPositions, ", "), jobcontracts.MinWatermarkSize, jobcontracts.MaxWatermarkSize,
			jobcontracts.MinWatermarkOpacity, jobcontracts.MaxWatermarkOpacity), nil
	}
	if b.WatermarkID != nil && *b.WatermarkID != "" {
		id := *b.WatermarkID
		if !domain.IsID(id) {
			return domain.Preset{}, fmt.Sprintf("unknown watermark %q", id), nil
		}
		if _, err := a.Store.GetWatermark(r.Context(), id); errors.Is(err, store.ErrNotFound) {
			return domain.Preset{}, fmt.Sprintf("unknown watermark %q", id), nil
		} else if err != nil {
			return domain.Preset{}, "", err
		}
		p.Watermark = id
	}
	return p, "", nil
}

// listPresets handles GET /presets, by name.
func (a *api) listPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := a.Store.ListPresets(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]presetDTO, 0, len(presets))
	for _, p := range presets {
		out = append(out, toPreset(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"presets": out})
}

func (a *api) getPreset(w http.ResponseWriter, r *http.Request) {
	p, ok := a.loadPreset(w, r)
	if ok {
		writeJSON(w, http.StatusOK, toPreset(p))
	}
}

func (a *api) loadPreset(w http.ResponseWriter, r *http.Request) (domain.Preset, bool) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return domain.Preset{}, false
	}
	p, err := a.Store.GetPreset(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return p, false
	}
	if err != nil {
		a.internalError(w, r, err)
		return p, false
	}
	return p, true
}

// createPreset handles POST /presets: 201 with the preset, 409 when another
// preset has the name.
func (a *api) createPreset(w http.ResponseWriter, r *http.Request) {
	p, problem, err := a.readPreset(r)
	if a.presetProblem(w, r, problem, err) {
		return
	}
	p.ID = domain.NewID()
	a.savePreset(w, r, p, a.Store.CreatePreset, http.StatusCreated)
}

// updatePreset handles PUT /presets/{id}, which replaces the preset's
// choices. Exports made before keep their files.
func (a *api) updatePreset(w http.ResponseWriter, r *http.Request) {
	old, ok := a.loadPreset(w, r)
	if !ok {
		return
	}
	p, problem, err := a.readPreset(r)
	if a.presetProblem(w, r, problem, err) {
		return
	}
	p.ID = old.ID
	a.savePreset(w, r, p, a.Store.UpdatePreset, http.StatusOK)
}

func (a *api) presetProblem(w http.ResponseWriter, r *http.Request, problem string, err error) bool {
	switch {
	case err != nil:
		a.internalError(w, r, err)
	case problem != "":
		writeError(w, http.StatusBadRequest, problem)
	default:
		return false
	}
	return true
}

func (a *api) savePreset(w http.ResponseWriter, r *http.Request, p domain.Preset, save func(context.Context, domain.Preset) error, status int) {
	err := save(r.Context(), p)
	switch {
	case errors.Is(err, store.ErrNameTaken):
		writeError(w, http.StatusConflict, fmt.Sprintf("another preset is called %q", p.Name))
		return
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
		return
	case err != nil:
		a.internalError(w, r, err)
		return
	}
	saved, err := a.Store.GetPreset(r.Context(), p.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, status, toPreset(saved))
}

// deletePreset handles DELETE /presets/{id}: 204. Its vignettes stay in
// the library, and exports made with it keep their files.
func (a *api) deletePreset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	err := a.Store.DeletePreset(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
