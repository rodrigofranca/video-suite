package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"time"
	"ultramedia/api/internal/app"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
)

// The vignette library and export presets (docs/adr/0030); app.SavePreset
// and app.Library do the work (docs/adr/0033).

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
	WatermarkID       *string `json:"watermarkId"`
	WatermarkPosition string  `json:"watermarkPosition"`
	WatermarkSize     int     `json:"watermarkSize"`
	WatermarkOpacity  int     `json:"watermarkOpacity"`
	// JumpCuts cuts the pauses and filler sounds out (docs/adr/0040).
	JumpCuts  bool      `json:"jumpCuts"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
		WatermarkOpacity: p.WatermarkOpacity, JumpCuts: p.JumpCuts, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
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

// ---- Vignettes ----

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
	JumpCuts          bool    `json:"jumpCuts"`
}

var presetUsage = `expected a JSON body like {"name": "My channel", "captionStyle": "highlight", "introVerticalId": "<vignette id>"}`

// readPreset decodes a preset body; app.SavePreset checks it.
func readPreset(w http.ResponseWriter, r *http.Request) (app.PresetInput, bool) {
	var b presetBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, presetUsage)
		return app.PresetInput{}, false
	}
	return app.PresetInput{
		Name: b.Name, Captions: b.Captions, CaptionStyle: b.CaptionStyle, CaptionPosition: b.CaptionPosition,
		CaptionColor: b.CaptionColor, CaptionCaps: b.CaptionCaps, CaptionLanguage: b.CaptionLanguage,
		IntroHorizontal: b.IntroHorizontal, IntroVertical: b.IntroVertical, OutroHorizontal: b.OutroHorizontal, OutroVertical: b.OutroVertical,
		WatermarkID: b.WatermarkID, WatermarkPosition: b.WatermarkPosition, WatermarkSize: b.WatermarkSize, WatermarkOpacity: b.WatermarkOpacity,
		JumpCuts: b.JumpCuts,
	}, true
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
	in, ok := readPreset(w, r)
	if !ok {
		return
	}
	p, err := a.presets.Create(r.Context(), in)
	a.writePreset(w, r, p, err, http.StatusCreated)
}

// updatePreset handles PUT /presets/{id}, which replaces the preset's
// choices. Exports made before keep their files.
func (a *api) updatePreset(w http.ResponseWriter, r *http.Request) {
	in, ok := readPreset(w, r)
	if !ok {
		return
	}
	p, err := a.presets.Update(r.Context(), r.PathValue("id"), in)
	a.writePreset(w, r, p, err, http.StatusOK)
}

func (a *api) writePreset(w http.ResponseWriter, r *http.Request, p domain.Preset, err error, status int) {
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	writeJSON(w, status, toPreset(p))
}

// deletePreset handles DELETE /presets/{id}: 204. Its files stay in the
// library, and exports made with it keep their files.
func (a *api) deletePreset(w http.ResponseWriter, r *http.Request) {
	a.noContent(w, r, a.presetDeletes.Run(r.Context(), r.PathValue("id")))
}
