package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"
)

// Response shapes. Field names are camelCase for the dashboard.

type assetDTO struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// ThumbnailURL is the public URL of the thumbnail, or null.
	ThumbnailURL *string `json:"thumbnailUrl"`
	// Metadata is the summary from the metadata job, or null.
	Metadata json.RawMessage `json:"metadata"`
}

type assetDetailDTO struct {
	assetDTO
	Source    *sourceDTO     `json:"source"`
	Jobs      []jobDTO       `json:"jobs"`
	Artifacts []artifactDTO  `json:"artifacts"`
	Playback  *playbackDTO   `json:"playback"`
	Subtitles []subtitlesDTO `json:"subtitles"`
}

type subtitlesDTO struct {
	Language string `json:"language"`
	URL      string `json:"url"`
}

type transcriptDTO struct {
	Language  string          `json:"language"`
	Model     string          `json:"model"`
	Text      string          `json:"text"`
	Segments  json.RawMessage `json:"segments"`
	CreatedAt time.Time       `json:"createdAt"`
}

type analysisDTO struct {
	Language  string          `json:"language"`
	Model     string          `json:"model"`
	Summary   string          `json:"summary"`
	Tags      []string        `json:"tags"`
	Chapters  json.RawMessage `json:"chapters"`
	CreatedAt time.Time       `json:"createdAt"`
}

type sourceDTO struct {
	StorageKey string `json:"storageKey"`
	MimeType   string `json:"mimeType"`
	SizeBytes  int64  `json:"sizeBytes"`
	Checksum   string `json:"checksum"`
}

type jobDTO struct {
	ID          string          `json:"id"`
	AssetID     string          `json:"assetId"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	Error       *string         `json:"error"`
	Input       json.RawMessage `json:"input"`
	CreatedAt   time.Time       `json:"createdAt"`
	StartedAt   *time.Time      `json:"startedAt"`
	CompletedAt *time.Time      `json:"completedAt"`
}

type logDTO struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type artifactDTO struct {
	ID         string          `json:"id"`
	JobID      string          `json:"jobId"`
	Type       string          `json:"type"`
	StorageKey string          `json:"storageKey"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type playbackDTO struct {
	Type        string `json:"type"`
	ManifestURL string `json:"manifestUrl"`
	// Renditions are the qualities in the stream, best first (docs/adr/0028);
	// empty for videos encoded before there were several.
	Renditions []renditionDTO `json:"renditions"`
}

type renditionDTO struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func toAsset(a domain.Asset) assetDTO {
	return assetDTO{ID: a.ID, Filename: a.Filename, Status: a.Status, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}

func toJob(j domain.Job) jobDTO {
	return jobDTO{
		ID: j.ID, AssetID: j.AssetID, Type: j.Type, Status: j.Status, Progress: j.Progress,
		Error: j.ErrorMessage, Input: json.RawMessage(j.InputJSON),
		CreatedAt: j.CreatedAt, StartedAt: j.StartedAt, CompletedAt: j.CompletedAt,
	}
}

func toJobs(jobs []domain.Job) []jobDTO {
	out := make([]jobDTO, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toJob(j))
	}
	return out
}

func toArtifacts(arts []domain.Artifact) []artifactDTO {
	out := make([]artifactDTO, 0, len(arts))
	for _, a := range arts {
		out = append(out, artifactDTO{ID: a.ID, JobID: a.JobID, Type: a.Type, StorageKey: a.StorageKey, Metadata: json.RawMessage(a.MetadataJSON), CreatedAt: a.CreatedAt})
	}
	return out
}

var assetStatuses = map[string]bool{
	"": true, domain.AssetUploading: true, domain.AssetProcessing: true, domain.AssetReady: true, domain.AssetFailed: true,
}

func (a *api) listAssets(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if !assetStatuses[status] {
		writeError(w, http.StatusBadRequest, "unknown status "+status)
		return
	}
	assets, err := a.Store.ListAssets(r.Context(), status)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]assetDTO, 0, len(assets))
	for _, item := range assets {
		dto := toAsset(item.Asset)
		dto.ThumbnailURL = a.publicURL(item.ThumbnailKey)
		dto.Metadata = item.Summary
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": out})
}

// loadAsset reads the {id} asset or writes a 404.
func (a *api) loadAsset(w http.ResponseWriter, r *http.Request) (domain.Asset, bool) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return domain.Asset{}, false
	}
	asset, err := a.Store.GetAsset(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return asset, false
	}
	if err != nil {
		a.internalError(w, r, err)
		return asset, false
	}
	return asset, true
}

func (a *api) getAsset(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	detail := assetDetailDTO{assetDTO: toAsset(asset)}

	src, err := a.Store.GetSource(ctx, asset.ID)
	switch {
	case err == nil:
		detail.Source = &sourceDTO{StorageKey: src.StorageKey, MimeType: src.MimeType, SizeBytes: src.SizeBytes, Checksum: src.Checksum}
	case !errors.Is(err, store.ErrNotFound):
		a.internalError(w, r, err)
		return
	}

	jobs, err := a.Store.ListJobs(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	detail.Jobs = toJobs(jobs)

	arts, err := a.Store.ListArtifacts(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	detail.Artifacts = toArtifacts(arts)
	detail.Playback = a.playbackFor(asset, arts)
	detail.ThumbnailURL = a.publicURL(latestKey(arts, domain.ArtifactThumbnail))
	detail.Subtitles = a.subtitlesOf(arts)

	meta, err := a.Store.GetTechnicalMetadata(ctx, asset.ID)
	switch {
	case err == nil:
		detail.Metadata = summaryOf(meta.MetadataJSON)
	case !errors.Is(err, store.ErrNotFound):
		a.internalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// getMetadata returns the full metadata: the summary and ffprobe's output.
func (a *api) getMetadata(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	meta, err := a.Store.GetTechnicalMetadata(r.Context(), asset.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "metadata not extracted yet")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(meta.MetadataJSON)
}

// subtitlesOf lists the newest subtitles file per language.
func (a *api) subtitlesOf(arts []domain.Artifact) []subtitlesDTO {
	out := []subtitlesDTO{}
	seen := map[string]bool{}
	for i := len(arts) - 1; i >= 0; i-- {
		if arts[i].Type != domain.ArtifactSubtitles {
			continue
		}
		var meta struct {
			Language string `json:"language"`
		}
		json.Unmarshal(arts[i].MetadataJSON, &meta)
		if meta.Language == "" || seen[meta.Language] {
			continue
		}
		seen[meta.Language] = true
		out = append(out, subtitlesDTO{Language: meta.Language, URL: a.Storage.PublicURL(arts[i].StorageKey)})
	}
	return out
}

// getTranscript returns the speech transcript of an asset.
func (a *api) getTranscript(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	t, err := a.Store.GetTranscript(r.Context(), asset.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "transcript not available yet")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, transcriptDTO{
		Language: t.Language, Model: t.Model, Text: t.Text, Segments: json.RawMessage(t.Segments), CreatedAt: t.CreatedAt,
	})
}

// getAnalysis returns the summary, tags and chapters written from the
// transcript by a language model.
func (a *api) getAnalysis(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	an, err := a.Store.GetAnalysis(r.Context(), asset.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "analysis not available yet")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, analysisDTO{
		Language: an.Language, Model: an.Model, Summary: an.Summary, Tags: an.Tags,
		Chapters: json.RawMessage(an.Chapters), CreatedAt: an.CreatedAt,
	})
}

// summaryOf picks the "summary" object out of stored metadata.
func summaryOf(metadataJSON []byte) json.RawMessage {
	var doc struct {
		Summary json.RawMessage `json:"summary"`
	}
	if json.Unmarshal(metadataJSON, &doc) != nil {
		return nil
	}
	return doc.Summary
}

// latestKey returns the storage key of the newest artifact of a type.
func latestKey(arts []domain.Artifact, artifactType string) string {
	for i := len(arts) - 1; i >= 0; i-- {
		if arts[i].Type == artifactType {
			return arts[i].StorageKey
		}
	}
	return ""
}

// publicURL returns the browser URL of a public object, or nil for no key.
func (a *api) publicURL(key string) *string {
	if key == "" {
		return nil
	}
	u := a.Storage.PublicURL(key)
	return &u
}

// playbackFor returns the HLS playback info of a ready asset, or nil.
func (a *api) playbackFor(asset domain.Asset, arts []domain.Artifact) *playbackDTO {
	if asset.Status != domain.AssetReady {
		return nil
	}
	for i := len(arts) - 1; i >= 0; i-- {
		if arts[i].Type != domain.ArtifactHLS {
			continue
		}
		var meta struct {
			Renditions []renditionDTO `json:"renditions"`
		}
		json.Unmarshal(arts[i].MetadataJSON, &meta)
		if meta.Renditions == nil {
			meta.Renditions = []renditionDTO{}
		}
		return &playbackDTO{Type: "hls", ManifestURL: a.Storage.PublicURL(arts[i].StorageKey), Renditions: meta.Renditions}
	}
	return nil
}

func (a *api) playback(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	arts, err := a.Store.ListArtifacts(r.Context(), asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	pb := a.playbackFor(asset, arts)
	if pb == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "asset is not ready", "status": asset.Status})
		return
	}
	writeJSON(w, http.StatusOK, pb)
}

func (a *api) listAssetJobs(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	jobs, err := a.Store.ListJobs(r.Context(), asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": toJobs(jobs)})
}

func (a *api) listArtifacts(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	arts, err := a.Store.ListArtifacts(r.Context(), asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": toArtifacts(arts)})
}

func (a *api) getJob(w http.ResponseWriter, r *http.Request) {
	job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toJob(job))
}

func (a *api) listJobLogs(w http.ResponseWriter, r *http.Request) {
	job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	logs, err := a.Store.ListJobLogs(r.Context(), job.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]logDTO, 0, len(logs))
	for _, l := range logs {
		out = append(out, logDTO{ID: l.ID, Level: l.Level, Message: l.Message, Timestamp: l.Timestamp})
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": out})
}

func (a *api) loadJob(w http.ResponseWriter, r *http.Request) (domain.Job, bool) {
	id := r.PathValue("id")
	if !domain.IsID(id) {
		writeError(w, http.StatusNotFound, "not found")
		return domain.Job{}, false
	}
	job, err := a.Store.GetJob(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return job, false
	}
	if err != nil {
		a.internalError(w, r, err)
		return job, false
	}
	return job, true
}

func (a *api) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.Log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
