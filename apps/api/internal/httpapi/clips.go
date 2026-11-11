package httpapi

import (
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

	"ultramedia/api/internal/app"
	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// downloadLinkExpiry is how long a clip download link works.
const downloadLinkExpiry = 15 * time.Minute

type clipDTO struct {
	ID       string  `json:"id"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
	Title    string  `json:"title"`
	Reason   string  `json:"reason"`
	// Origin is "suggested" (by the language model) or "quote" (made from
	// the transcript, docs/adr/0039); Score is null for a quote.
	Origin    string    `json:"origin"`
	Score     *int      `json:"score"`
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
	// JumpCuts says the file's pauses and filler sounds were cut out
	// (docs/adr/0040), and DurationSeconds is how long the clip is in it,
	// without the vignettes.
	JumpCuts        bool    `json:"jumpCuts,omitempty"`
	DurationSeconds float64 `json:"durationSeconds,omitempty"`
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
	JumpCuts        bool    `json:"jumpCuts"`
	KeptSeconds     float64 `json:"keptSeconds"`
	DurationSeconds float64 `json:"durationSeconds"`
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
		Origin: c.Origin, Rank: c.Rank, Model: c.Model, CreatedAt: c.CreatedAt, EditedAt: c.EditedAt,
		Exports: clipExports(c, jobs, arts),
	}
	if c.Score > 0 {
		dto.Score = &c.Score
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
			e.JumpCuts, e.DurationSeconds = file.JumpCuts, file.DurationSeconds
			if file.JumpCuts {
				e.DurationSeconds = file.KeptSeconds
			}
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
// "end", "title"}: app.EditClip changes the clip for the exports from then
// on (docs/adr/0024).
func (a *api) editClip(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Start *float64 `json:"start"`
		End   *float64 `json:"end"`
		Title *string  `json:"title"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, `expected a JSON body with any of {"start": 12.5, "end": 40, "title": "..."} (seconds from the start of the video)`)
		return
	}
	clip, err := a.clipEdits.Run(r.Context(), app.EditRequest{AssetID: r.PathValue("id"), ClipID: r.PathValue("clipId"),
		Start: body.Start, End: body.End, Title: body.Title})
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	a.writeClip(w, r, clip)
}

// createClip handles POST /assets/{id}/clips with {"start": 12.1,
// "end": 18.4, "title": "..."}: a clip made from a quote (docs/adr/0039).
func (a *api) createClip(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Start *float64 `json:"start"`
		End   *float64 `json:"end"`
		Title string   `json:"title"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || body.Start == nil || body.End == nil {
		writeError(w, http.StatusBadRequest, `expected a JSON body with {"start": 12.1, "end": 18.4} (seconds from the start of the video) and an optional "title"`)
		return
	}
	clip, err := a.clipCreates.Run(r.Context(), app.CreateRequest{AssetID: r.PathValue("id"), Start: *body.Start, End: *body.End, Title: body.Title})
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	a.writeClipStatus(w, r, clip, http.StatusCreated)
}

// writeClip answers with one clip as listClips shows it.
func (a *api) writeClip(w http.ResponseWriter, r *http.Request, c domain.Clip) {
	a.writeClipStatus(w, r, c, http.StatusOK)
}

func (a *api) writeClipStatus(w http.ResponseWriter, r *http.Request, c domain.Clip, status int) {
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
	writeJSON(w, status, toClipDTO(c, jobs, arts))
}

// exportClip handles POST /assets/{id}/clips/{clipId}/exports with
// {"format": "vertical", "captions": true}: it queues an export-clip job.
// Like a re-run, it waits until no other job of the asset is pending or
// running. Captions in a captionLanguage the transcript has no translation
// into yet first queue a translate-transcript job, which the export waits
// for (docs/adr/0029).
func (a *api) exportClip(w http.ResponseWriter, r *http.Request) {
	var body exportBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, exportUsage)
		return
	}
	res, err := a.exports.Run(r.Context(), app.ExportRequest{
		AssetID: r.PathValue("id"), ClipID: r.PathValue("clipId"), Format: body.Format,
		Captions: body.Captions, CaptionStyle: body.CaptionStyle, CaptionPosition: body.CaptionPosition,
		CaptionColor: body.CaptionColor, CaptionCaps: body.CaptionCaps, CaptionLanguage: body.CaptionLanguage,
		JumpCuts: body.JumpCuts, PresetID: body.PresetID,
	})
	if errors.Is(err, app.ErrInvalidChoice) {
		writeError(w, http.StatusBadRequest, exportUsage)
		return
	}
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toJob(res.Job))
}

var exportUsage = `expected a JSON body like {"format": "vertical", "captions": true, "captionStyle": "highlight", "captionPosition": "bottom", "captionColor": "yellow", "captionCaps": false, "captionLanguage": "en", "jumpCuts": true, "presetId": "<id>"}; ` +
	app.ChoicesOnOffer

// exportBody is what POST .../exports takes; app.ExportClip does the rest
// (docs/adr/0033). With presetId, the preset gives its openings and closings,
// and the caption choices the body leaves out (docs/adr/0030).
type exportBody struct {
	Format          string  `json:"format"`
	Captions        *bool   `json:"captions"`
	CaptionStyle    *string `json:"captionStyle"`
	CaptionPosition *string `json:"captionPosition"`
	CaptionColor    *string `json:"captionColor"`
	CaptionCaps     *bool   `json:"captionCaps"`
	CaptionLanguage *string `json:"captionLanguage"`
	JumpCuts        *bool   `json:"jumpCuts"`
	PresetID        string  `json:"presetId"`
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
