package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
	FaceCoverage *float64 `json:"faceCoverage"`
	// FaceFollows is what a "vertical-face" export follows: "speaker" (who
	// is speaking, with several people on screen) or "face" (the biggest);
	// FacePeople is how many people are on screen at once. Both are null
	// while the clip has no face track.
	FaceFollows *string `json:"faceFollows"`
	FacePeople  *int    `json:"facePeople"`
	// EditedAt is when the clip was last changed by hand, or null.
	EditedAt *time.Time      `json:"editedAt"`
	Exports  []clipExportDTO `json:"exports"`
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
	// CaptionStyle is the style the file's captions ended up in, with its
	// position, accent color (empty for styles without one) and case.
	CaptionStyle    string `json:"captionStyle,omitempty"`
	CaptionPosition string `json:"captionPosition,omitempty"`
	CaptionColor    string `json:"captionColor,omitempty"`
	CaptionCaps     bool   `json:"captionCaps,omitempty"`
	// CaptionLanguage is the language of the captions: of the file once it
	// is ready, the one asked for before. TranslatedFrom is the transcript's
	// language when the captions were translated (docs/adr/0029).
	CaptionLanguage string `json:"captionLanguage,omitempty"`
	TranslatedFrom  string `json:"translatedFrom,omitempty"`
	// Preset is the name of the preset the export used (of the file once it
	// is ready, the one asked for before); Intro and Outro say whether the
	// file has an opening and a closing (docs/adr/0030).
	Preset    string `json:"preset,omitempty"`
	Intro     bool   `json:"intro,omitempty"`
	Outro     bool   `json:"outro,omitempty"`
	Watermark bool   `json:"watermark,omitempty"`
	// Framing is how a "vertical-face" file was framed: "face" or "center".
	Framing   string `json:"framing,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	Ready     bool   `json:"ready"` // a file can be downloaded
	// Outdated is true when the file was cut from another start or end:
	// the clip was edited after this export.
	Outdated bool `json:"outdated,omitempty"`
}

type clipFileMeta struct {
	ClipID          string  `json:"clipId"`
	Format          string  `json:"format"`
	Captions        bool    `json:"captions"`
	CaptionStyle    string  `json:"captionStyle"`
	CaptionPosition string  `json:"captionPosition"`
	CaptionColor    string  `json:"captionColor"`
	CaptionCaps     bool    `json:"captionCaps"`
	Language        string  `json:"language"`
	TranslatedFrom  string  `json:"translatedFrom"`
	Preset          string  `json:"preset"`
	Intro           bool    `json:"intro"`
	Outro           bool    `json:"outro"`
	Watermark       bool    `json:"watermark"`
	Framing         string  `json:"framing"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	SizeBytes       int64   `json:"sizeBytes"`
	Start           float64 `json:"start"`
	End             float64 `json:"end"`
}

// listClips returns an asset's suggested clips, best first, each with the
// state of its exports; whether the transcript has word timings (needed for
// word-by-word captions); its language; and the languages it has been
// translated into, which exports can use without translating first.
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
		out = append(out, toClipDTO(c, jobs, arts))
	}
	wordTimings, language := false, ""
	if tr, err := a.Store.GetTranscript(ctx, asset.ID); err == nil {
		var words []json.RawMessage
		wordTimings = json.Unmarshal(tr.Words, &words) == nil && len(words) > 0
		language = tr.Language
	} else if !errors.Is(err, store.ErrNotFound) {
		a.internalError(w, r, err)
		return
	}
	translations, err := a.Store.TranslationLanguages(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clips": out, "wordTimings": wordTimings, "transcriptLanguage": language, "translations": translations,
	})
}

func toClipDTO(c domain.Clip, jobs []domain.Job, arts []domain.Artifact) clipDTO {
	dto := clipDTO{
		ID: c.ID, Start: c.Start, End: c.End, Duration: c.End - c.Start, Title: c.Title, Reason: c.Reason,
		Score: c.Score, Rank: c.Rank, Model: c.Model, CreatedAt: c.CreatedAt, EditedAt: c.EditedAt,
		Exports: clipExports(c, jobs, arts),
	}
	dto.FaceCoverage, dto.FaceFollows, dto.FacePeople = faceSummary(c.FaceTrack)
	return dto
}

// faceSummary reads a clip's face track (docs/adr/0022, 0023). Tracks made
// before speakers were followed have neither "follows" nor "people".
func faceSummary(track []byte) (*float64, *string, *int) {
	var t struct {
		Coverage float64 `json:"coverage"`
		Follows  string  `json:"follows"`
		People   *int    `json:"people"`
	}
	if len(track) == 0 || json.Unmarshal(track, &t) != nil {
		return nil, nil, nil
	}
	if t.Follows == "" {
		t.Follows = "face"
	}
	return &t.Coverage, &t.Follows, t.People
}

// clipExports finds, per format, the newest export job of a clip and the
// newest file it produced.
func clipExports(clip domain.Clip, jobs []domain.Job, arts []domain.Artifact) []clipExportDTO {
	clipID := clip.ID
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
		var in jobcontracts.ExportClipInput
		json.Unmarshal(job.InputJSON, &in)
		e.CaptionLanguage, e.Preset = in.CaptionLanguage, in.Preset // asked for; the file's below
		if file, ok := latestClipFile(arts, clipID, format); ok {
			e.Captions, e.CaptionStyle, e.Framing, e.Width, e.Height, e.SizeBytes, e.Ready = file.Captions, file.CaptionStyle, file.Framing, file.Width, file.Height, file.SizeBytes, true
			e.CaptionLanguage, e.Preset, e.Intro, e.Outro, e.Watermark = "", file.Preset, file.Intro, file.Outro, file.Watermark
			if file.Captions {
				e.CaptionPosition, e.CaptionColor, e.CaptionCaps = file.CaptionPosition, file.CaptionColor, file.CaptionCaps
				e.CaptionLanguage, e.TranslatedFrom = file.Language, file.TranslatedFrom
			}
			e.Outdated = math.Abs(file.Start-clip.Start) > 0.001 || math.Abs(file.End-clip.End) > 0.001
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

// editClip handles PATCH /assets/{id}/clips/{clipId} with any of {"start",
// "end", "title"}: it changes the clip for the exports from then on
// (docs/adr/0024). New times must stay inside the video and give a clip of
// domain.MinClipSeconds to domain.MaxClipSeconds; they clear the face track
// and queue a track-faces job for this clip. Like an export, an edit waits
// until no job of the asset is pending or running.
func (a *api) editClip(w http.ResponseWriter, r *http.Request) {
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
	var body struct {
		Start *float64 `json:"start"`
		End   *float64 `json:"end"`
		Title *string  `json:"title"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || (body.Start == nil && body.End == nil && body.Title == nil) {
		writeError(w, http.StatusBadRequest, `expected a JSON body with any of {"start": 12.5, "end": 40, "title": "..."} (seconds from the start of the video)`)
		return
	}
	edited := clip
	if body.Start != nil {
		edited.Start = math.Round(*body.Start*1000) / 1000
	}
	if body.End != nil {
		edited.End = math.Round(*body.End*1000) / 1000
	}
	if body.Title != nil {
		edited.Title = strings.Join(strings.Fields(*body.Title), " ")
	}
	duration, err := a.videoDuration(ctx, asset.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if msg := checkClip(edited, duration); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	var retrack *domain.Job
	if edited.Start != clip.Start || edited.End != clip.End {
		src, err := a.Store.GetSource(ctx, asset.ID)
		if err != nil {
			a.internalError(w, r, err)
			return
		}
		input, _ := json.Marshal(jobcontracts.TrackFacesInput{AssetID: asset.ID, StorageKey: src.StorageKey, ClipID: clip.ID})
		retrack = &domain.Job{
			ID: domain.NewID(), AssetID: asset.ID, Type: jobcontracts.JobTypeTrackFaces, InputJSON: input,
			Status: domain.JobPending, CreatedAt: time.Now(),
		}
	}
	if edited.Start == clip.Start && edited.End == clip.End && edited.Title == clip.Title {
		a.writeClip(w, r, clip)
		return
	}
	err = a.Store.EditClipIfIdle(ctx, edited, retrack)
	if errors.Is(err, store.ErrBusy) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	a.Log.Info("clip edited", "asset_id", asset.ID, "clip_id", clip.ID, "start", edited.Start, "end", edited.End)
	if retrack != nil {
		msg := jobcontracts.Message{JobID: retrack.ID, Type: retrack.Type, Payload: retrack.InputJSON}
		if err := a.Enqueue(ctx, msg); err != nil {
			// The edit stands; the clip stays without a face track (centred) and the job shows why.
			a.Store.FinishJob(context.WithoutCancel(ctx), retrack.ID, fmt.Errorf("could not enqueue job: %w", err))
			a.Log.Error("enqueue failed", "asset_id", asset.ID, "job_id", retrack.ID, "error", err)
		}
	}
	saved, err := a.Store.GetClip(ctx, clip.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	a.writeClip(w, r, saved)
}

// checkClip returns why an edited clip is not valid, or "".
func checkClip(c domain.Clip, duration float64) string {
	switch {
	case c.Start < 0 || c.End <= c.Start:
		return "the start must be at least 0 and before the end"
	case c.End-c.Start < domain.MinClipSeconds || c.End-c.Start > domain.MaxClipSeconds:
		return fmt.Sprintf("a clip lasts from %.0f to %.0f seconds", domain.MinClipSeconds, domain.MaxClipSeconds)
	case duration > 0 && c.End > duration+0.05:
		return fmt.Sprintf("the end is after the end of the video (%.1f s)", duration)
	case c.Title == "":
		return "the title cannot be empty"
	case len([]rune(c.Title)) > domain.MaxClipTitle:
		return fmt.Sprintf("the title is longer than %d characters", domain.MaxClipTitle)
	}
	return ""
}

// videoDuration is the source's duration from its metadata, or 0 when unknown.
func (a *api) videoDuration(ctx context.Context, assetID string) (float64, error) {
	m, err := a.Store.GetTechnicalMetadata(ctx, assetID)
	if errors.Is(err, store.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var doc struct {
		Summary struct {
			DurationSeconds float64 `json:"durationSeconds"`
		} `json:"summary"`
	}
	json.Unmarshal(m.MetadataJSON, &doc)
	return doc.Summary.DurationSeconds, nil
}

// writeClip answers with one clip as listClips shows it.
func (a *api) writeClip(w http.ResponseWriter, r *http.Request, c domain.Clip) {
	ctx := r.Context()
	jobs, err := a.Store.ListJobs(ctx, c.AssetID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	arts, err := a.Store.ListArtifacts(ctx, c.AssetID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClipDTO(c, jobs, arts))
}

// exportClip handles POST /assets/{id}/clips/{clipId}/exports with
// {"format": "vertical", "captions": true}: it queues an export-clip job.
// Like a re-run, it waits until no other job of the asset is pending or
// running. Captions in a captionLanguage the transcript has no translation
// into yet first queue a translate-transcript job, which the export waits
// for (docs/adr/0029).
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
	var choice exportBody
	err = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&choice)
	if err != nil {
		writeError(w, http.StatusBadRequest, exportUsage)
		return
	}
	var preset domain.Preset
	var intro, outro *jobcontracts.Vignette
	var watermark *jobcontracts.Watermark
	if choice.PresetID != "" {
		if domain.IsID(choice.PresetID) {
			preset, err = a.Store.GetPreset(ctx, choice.PresetID)
		}
		if !domain.IsID(choice.PresetID) || errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown preset")
			return
		}
		if err == nil {
			intro, outro, err = a.exportVignettes(ctx, preset)
		}
		if err == nil {
			watermark, err = a.exportWatermark(ctx, preset)
		}
		if err != nil {
			a.internalError(w, r, err)
			return
		}
	}
	body := choice.resolve(preset)
	if !slices.Contains(jobcontracts.ClipFormats, body.Format) || !slices.Contains(jobcontracts.CaptionStyles, body.CaptionStyle) ||
		(body.CaptionPosition != "" && !slices.Contains(jobcontracts.CaptionPositions, body.CaptionPosition)) ||
		(body.CaptionColor != "" && !slices.Contains(jobcontracts.CaptionColors, body.CaptionColor)) ||
		(body.CaptionLanguage != "" && !slices.Contains(jobcontracts.CaptionLanguages, body.CaptionLanguage)) {
		writeError(w, http.StatusBadRequest, exportUsage)
		return
	}
	captions := body.Captions
	language, translate, err := a.captionLanguage(ctx, asset.ID, captions, body.CaptionLanguage)
	if errors.Is(err, errNothingToTranslate) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		a.internalError(w, r, err)
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
		Captions: captions, CaptionStyle: body.CaptionStyle,
		CaptionPosition: body.CaptionPosition, CaptionColor: body.CaptionColor, CaptionCaps: body.CaptionCaps,
		CaptionLanguage: language, Intro: intro, Outro: outro, Watermark: watermark, Preset: preset.Name,
		OutputKey: storage.ClipKey(asset.ID, clip.ID, body.Format),
	})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	now := time.Now()
	job := domain.Job{
		ID: domain.NewID(), AssetID: asset.ID, Type: jobcontracts.JobTypeExportClip, InputJSON: input,
		Status: domain.JobPending, CreatedAt: now,
	}
	first, then := job, []domain.Job(nil)
	if translate {
		in, _ := json.Marshal(jobcontracts.TranslateTranscriptInput{AssetID: asset.ID, Language: language})
		first = domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: jobcontracts.JobTypeTranslateTranscript, InputJSON: in,
			Status: domain.JobPending, CreatedAt: now}
		job.CreatedAt = now.Add(time.Millisecond) // listed after the translation it waits for
		then = []domain.Job{job}
	}
	err = a.Store.CreateJobIfIdle(ctx, first, "", then...)
	if errors.Is(err, store.ErrBusy) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if err := a.Enqueue(ctx, jobcontracts.Message{JobID: first.ID, Type: first.Type, Payload: first.InputJSON}); err != nil {
		bg := context.WithoutCancel(ctx)
		a.Store.FinishJob(bg, first.ID, fmt.Errorf("could not enqueue job: %w", err))
		if translate {
			a.Store.FinishJob(bg, job.ID, errors.New("not run: the translation before it could not be queued"))
		}
		a.Log.Error("enqueue failed", "asset_id", asset.ID, "job_id", first.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "the job could not be queued")
		return
	}
	saved, err := a.Store.GetJob(ctx, job.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	if translate {
		a.Log.Info("clip export queued after a translation", "asset_id", asset.ID, "clip_id", clip.ID, "format", body.Format,
			"language", language, "job_id", job.ID, "translate_job_id", first.ID)
	} else {
		a.Log.Info("clip export queued", "asset_id", asset.ID, "clip_id", clip.ID, "format", body.Format, "job_id", job.ID)
	}
	writeJSON(w, http.StatusAccepted, toJob(saved))
}

var errNothingToTranslate = errors.New("this video has no transcript to translate the captions from")

var exportUsage = `expected a JSON body like {"format": "vertical", "captions": true, "captionStyle": "highlight", "captionPosition": "bottom", "captionColor": "yellow", "captionCaps": false, "captionLanguage": "en", "presetId": "<id>"}; formats: ` +
	strings.Join(jobcontracts.ClipFormats, ", ") + "; caption styles: " + strings.Join(jobcontracts.CaptionStyles, ", ") +
	"; positions: " + strings.Join(jobcontracts.CaptionPositions, ", ") + "; colors: " + strings.Join(jobcontracts.CaptionColors, ", ") +
	"; languages (empty for the transcript's own): " + strings.Join(jobcontracts.CaptionLanguages, ", ")

// exportBody is what POST .../exports takes. With presetId, the preset
// gives its openings and closings, and the caption choices the body leaves
// out (docs/adr/0030).
type exportBody struct {
	Format          string  `json:"format"`
	Captions        *bool   `json:"captions"`
	CaptionStyle    *string `json:"captionStyle"`
	CaptionPosition *string `json:"captionPosition"`
	CaptionColor    *string `json:"captionColor"`
	CaptionCaps     *bool   `json:"captionCaps"`
	CaptionLanguage *string `json:"captionLanguage"`
	PresetID        string  `json:"presetId"`
}

// exportChoice is an export's caption choices once the preset and the
// defaults have filled in what the body left out.
type exportChoice struct {
	Format                                                       string
	Captions                                                     bool
	CaptionStyle, CaptionPosition, CaptionColor, CaptionLanguage string
	CaptionCaps                                                  *bool
}

// resolve fills the choices the body left out from the preset (a zero
// Preset when none was given) and the defaults.
func (b exportBody) resolve(p domain.Preset) exportChoice {
	pick := func(v *string, fromPreset string) string {
		if v != nil {
			return *v
		}
		return fromPreset
	}
	c := exportChoice{
		Format: b.Format, Captions: p.ID == "" || p.Captions,
		CaptionStyle: pick(b.CaptionStyle, p.CaptionStyle), CaptionPosition: pick(b.CaptionPosition, p.CaptionPosition),
		CaptionColor: pick(b.CaptionColor, p.CaptionColor), CaptionLanguage: pick(b.CaptionLanguage, p.CaptionLanguage),
		CaptionCaps: p.CaptionCaps,
	}
	if b.Captions != nil {
		c.Captions = *b.Captions
	}
	if b.CaptionCaps != nil {
		c.CaptionCaps = b.CaptionCaps
	}
	if c.CaptionStyle == "" {
		c.CaptionStyle = jobcontracts.CaptionStyles[0]
	}
	if c.CaptionPosition == jobcontracts.CaptionPositions[0] {
		c.CaptionPosition = "" // the default, as before presets
	}
	return c
}

// captionLanguage returns the language an export's captions are burned in
// (empty for the transcript's own) and whether the transcript must be
// translated into it first. Asking for the transcript's own language is
// asking for no translation.
func (a *api) captionLanguage(ctx context.Context, assetID string, captions bool, asked string) (string, bool, error) {
	if !captions || asked == "" {
		return "", false, nil
	}
	tr, err := a.Store.GetTranscript(ctx, assetID)
	if errors.Is(err, store.ErrNotFound) {
		return "", false, errNothingToTranslate
	}
	if err != nil {
		return "", false, err
	}
	if tr.Language == asked {
		return "", false, nil
	}
	_, err = a.Store.GetTranslation(ctx, assetID, asked)
	if errors.Is(err, store.ErrNotFound) {
		return asked, true, nil
	}
	return asked, false, err
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
	file, ok := latestClipFile(arts, clip.ID, format)
	if !ok {
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
		filename = clipFilename(asset.Filename, clip.Rank, format, file.TranslatedFrom != "" && file.Captions, file.Language)
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
// "my-talk-clip-2-vertical.mp4", in plain ASCII; with translated captions,
// their language follows: "my-talk-clip-2-vertical-en.mp4".
func clipFilename(videoName string, rank int, format string, translated bool, language string) string {
	base := strings.Trim(unsafeFilename.ReplaceAllString(strings.TrimSuffix(videoName, path.Ext(videoName)), "-"), "-")
	if len(base) > 60 {
		base = base[:60]
	}
	if base == "" {
		base = "video"
	}
	if translated {
		return fmt.Sprintf("%s-clip-%d-%s-%s.mp4", base, rank, format, language)
	}
	return fmt.Sprintf("%s-clip-%d-%s.mp4", base, rank, format)
}
