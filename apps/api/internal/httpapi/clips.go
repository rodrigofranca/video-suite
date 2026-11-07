package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// downloadLinkExpiry is how long a clip download link works.
const downloadLinkExpiry = 15 * time.Minute

type clipDTO struct {
	ID        string    `json:"id"`
	Start     float64   `json:"start"`
	End       float64   `json:"end"`
	Duration  float64   `json:"duration"`
	Title     string    `json:"title"`
	Reason    string    `json:"reason"`
	Score     int       `json:"score"`
	Rank      int       `json:"rank"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"createdAt"`
	// FaceCoverage is the share of the clip in which a face was found, or
	// null while the clip has no face track.
	FaceCoverage *float64        `json:"faceCoverage"`
	Exports      []clipExportDTO `json:"exports"`
}

// clipExportDTO is the newest export of a clip in one format: its job's
// state and, once done, the file's details.
type clipExportDTO struct {
	Format   string  `json:"format"`
	JobID    string  `json:"jobId"`
	Status   string  `json:"status"`
	Progress int     `json:"progress"`
	Error    *string `json:"error"`
	Captions bool    `json:"captions"`
	// CaptionStyle is the style the file's captions ended up in.
	CaptionStyle string `json:"captionStyle,omitempty"`
	// Framing is how a "vertical-face" file was framed: "face" or "center".
	Framing   string `json:"framing,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	Ready     bool   `json:"ready"` // a file can be downloaded
}

type clipFileMeta struct {
	ClipID       string `json:"clipId"`
	Format       string `json:"format"`
	Captions     bool   `json:"captions"`
	CaptionStyle string `json:"captionStyle"`
	Framing      string `json:"framing"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SizeBytes    int64  `json:"sizeBytes"`
}

// listClips returns an asset's suggested clips, best first, each with the
// state of its exports, and whether the transcript has word timings (needed
// for word-by-word captions).
func (a *api) listClips(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	clips, err := a.Store.ListClips(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	jobs, err := a.Store.ListJobs(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	arts, err := a.Store.ListArtifacts(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	out := make([]clipDTO, 0, len(clips))
	for _, c := range clips {
		out = append(out, clipDTO{
			ID: c.ID, Start: c.Start, End: c.End, Duration: c.End - c.Start, Title: c.Title, Reason: c.Reason,
			Score: c.Score, Rank: c.Rank, Model: c.Model, CreatedAt: c.CreatedAt, Exports: clipExports(c.ID, jobs, arts),
			FaceCoverage: faceCoverage(c.FaceTrack),
		})
	}
	wordTimings := false
	if tr, err := a.Store.GetTranscript(ctx, asset.ID); err == nil {
		var words []json.RawMessage
		wordTimings = json.Unmarshal(tr.Words, &words) == nil && len(words) > 0
	} else if !errors.Is(err, store.ErrNotFound) {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clips": out, "wordTimings": wordTimings})
}

func faceCoverage(track []byte) *float64 {
	var t struct {
		Coverage float64 `json:"coverage"`
	}
	if len(track) == 0 || json.Unmarshal(track, &t) != nil {
		return nil
	}
	return &t.Coverage
}

// clipExports finds, per format, the newest export job of a clip and the
// newest file it produced.
func clipExports(clipID string, jobs []domain.Job, arts []domain.Artifact) []clipExportDTO {
	out := []clipExportDTO{}
	for _, format := range jobcontracts.ClipFormats {
		var job *domain.Job
		for i := range jobs {
			var in jobcontracts.ExportClipInput
			if jobs[i].Type != jobcontracts.JobTypeExportClip || json.Unmarshal(jobs[i].InputJSON, &in) != nil ||
				in.ClipID != clipID || in.Format != format {
				continue
			}
			if job == nil || !jobs[i].CreatedAt.Before(job.CreatedAt) {
				job = &jobs[i]
			}
		}
		if job == nil {
			continue
		}
		e := clipExportDTO{Format: format, JobID: job.ID, Status: job.Status, Progress: job.Progress, Error: job.ErrorMessage}
		if file, ok := latestClipFile(arts, clipID, format); ok {
			e.Captions, e.CaptionStyle, e.Framing, e.Width, e.Height, e.SizeBytes, e.Ready = file.Captions, file.CaptionStyle, file.Framing, file.Width, file.Height, file.SizeBytes, true
		}
		out = append(out, e)
	}
	return out
}

func latestClipFile(arts []domain.Artifact, clipID, format string) (clipFileMeta, bool) {
	var best clipFileMeta
	var bestAt time.Time
	found := false
	for _, art := range arts {
		var m clipFileMeta
		if art.Type != domain.ArtifactClip || json.Unmarshal(art.MetadataJSON, &m) != nil || m.ClipID != clipID || m.Format != format {
			continue
		}
		if !found || !art.CreatedAt.Before(bestAt) {
			best, bestAt, found = m, art.CreatedAt, true
		}
	}
	return best, found
}

// exportClip handles POST /assets/{id}/clips/{clipId}/exports with
// {"format": "vertical", "captions": true}: it queues an export-clip job.
// Like a re-run, it waits until no other job of the asset is pending or running.
func (a *api) exportClip(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	clipID := r.PathValue("clipId")
	if !domain.IsID(clipID) {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	clip, err := a.Store.GetClip(ctx, clipID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && clip.AssetID != asset.ID) {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	body := struct {
		Format       string `json:"format"`
		Captions     *bool  `json:"captions"`
		CaptionStyle string `json:"captionStyle"`
	}{}
	err = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	if body.CaptionStyle == "" {
		body.CaptionStyle = jobcontracts.CaptionStyles[0]
	}
	if err != nil || !slices.Contains(jobcontracts.ClipFormats, body.Format) || !slices.Contains(jobcontracts.CaptionStyles, body.CaptionStyle) {
		writeError(w, http.StatusBadRequest, `expected a JSON body like {"format": "vertical", "captions": true, "captionStyle": "highlight"}; formats: `+
			strings.Join(jobcontracts.ClipFormats, ", ")+"; caption styles: "+strings.Join(jobcontracts.CaptionStyles, ", "))
		return
	}
	src, err := a.Store.GetSource(ctx, asset.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "the asset has no source file")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	input, err := json.Marshal(jobcontracts.ExportClipInput{
		AssetID: asset.ID, ClipID: clip.ID, StorageKey: src.StorageKey, Format: body.Format,
		Captions: body.Captions == nil || *body.Captions, CaptionStyle: body.CaptionStyle,
		OutputKey: storage.ClipKey(asset.ID, clip.ID, body.Format),
	})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	job := domain.Job{
		ID: domain.NewID(), AssetID: asset.ID, Type: jobcontracts.JobTypeExportClip, InputJSON: input,
		Status: domain.JobPending, CreatedAt: time.Now(),
	}
	err = a.Store.CreateJobIfIdle(ctx, job, "")
	if errors.Is(err, store.ErrBusy) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if err := a.Enqueue(ctx, jobcontracts.Message{JobID: job.ID, Type: job.Type, Payload: job.InputJSON}); err != nil {
		a.Store.FinishJob(context.WithoutCancel(ctx), job.ID, fmt.Errorf("could not enqueue job: %w", err))
		a.Log.Error("enqueue failed", "asset_id", asset.ID, "job_id", job.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "the job could not be queued")
		return
	}
	saved, err := a.Store.GetJob(ctx, job.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	a.Log.Info("clip export queued", "asset_id", asset.ID, "clip_id", clip.ID, "format", body.Format, "job_id", job.ID)
	writeJSON(w, http.StatusAccepted, toJob(saved))
}

// downloadClip handles GET /clips/{id}/download?format=vertical: it redirects
// to a short-lived signed link to the exported file. Clips are private
// (docs/adr/0020). With inline=1 the browser plays the file instead of
// saving it.
func (a *api) downloadClip(w http.ResponseWriter, r *http.Request) {
	id, format := r.PathValue("id"), r.URL.Query().Get("format")
	if !domain.IsID(id) || !slices.Contains(jobcontracts.ClipFormats, format) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ctx := r.Context()
	clip, err := a.Store.GetClip(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	arts, err := a.Store.ListArtifacts(ctx, clip.AssetID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if _, ok := latestClipFile(arts, clip.ID, format); !ok {
		writeError(w, http.StatusNotFound, "this clip has not been exported in this format yet")
		return
	}
	filename := "" // inline: the browser plays the file
	if r.URL.Query().Get("inline") != "1" {
		asset, err := a.Store.GetAsset(ctx, clip.AssetID)
		if err != nil {
			a.internalError(w, r, err)
			return
		}
		filename = clipFilename(asset.Filename, clip.Rank, format)
	}
	url, err := a.Storage.PresignForBrowser(ctx, storage.ClipKey(clip.AssetID, clip.ID, format), filename, downloadLinkExpiry)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // the link expires
	http.Redirect(w, r, url, http.StatusFound)
}

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// clipFilename names a downloaded clip after its video, e.g.
// "my-talk-clip-2-vertical.mp4", in plain ASCII.
func clipFilename(videoName string, rank int, format string) string {
	base := strings.Trim(unsafeFilename.ReplaceAllString(strings.TrimSuffix(videoName, path.Ext(videoName)), "-"), "-")
	if len(base) > 60 {
		base = base[:60]
	}
	if base == "" {
		base = "video"
	}
	return fmt.Sprintf("%s-clip-%d-%s.mp4", base, rank, format)
}
