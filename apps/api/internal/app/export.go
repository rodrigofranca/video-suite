package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// ExportClip queues the export of a clip in one format (docs/adr/0020),
// with captions as chosen or as a preset says (docs/adr/0030, 0031), in the
// transcript's language or another one: the first export into a language
// queues its translation first, and the export waits for it (docs/adr/0029).
// Like every command on an asset, it waits until no job of the asset is
// pending or running.
type ExportClip struct {
	Assets Assets
	Clips  Clips
	Speech Speech
	Brand  Brand
	Jobs   Jobs
	Queue  Queue
	Log    *slog.Logger
	clock
}

// ExportRequest is what someone asked for. A nil choice was left out: the
// preset, when there is one, fills it in, else the default does.
type ExportRequest struct {
	AssetID, ClipID string
	Format          string
	Captions        *bool
	CaptionStyle    *string
	CaptionPosition *string
	CaptionColor    *string
	CaptionCaps     *bool
	CaptionLanguage *string
	PresetID        string
}

// ExportResult is the queued export job and, when the captions must be
// translated first, the translation job it waits for.
type ExportResult struct {
	Job       domain.Job
	Translate *domain.Job
}

// ErrInvalidChoice marks a request whose format or caption choices are not
// on offer; the error's message lists what is.
var ErrInvalidChoice = errors.New("invalid export choice")

// ChoicesOnOffer lists the formats and caption choices an export takes.
var ChoicesOnOffer = "formats: " + strings.Join(jobcontracts.ClipFormats, ", ") +
	"; caption styles: " + strings.Join(jobcontracts.CaptionStyles, ", ") +
	"; positions: " + strings.Join(jobcontracts.CaptionPositions, ", ") +
	"; colors: " + strings.Join(jobcontracts.CaptionColors, ", ") +
	"; languages (empty for the transcript's own): " + strings.Join(jobcontracts.CaptionLanguages, ", ")

// Run checks the request, works out the captions and the vignettes, and
// queues the jobs.
func (uc ExportClip) Run(ctx context.Context, req ExportRequest) (ExportResult, error) {
	clip, err := clipOf(ctx, uc.Assets, uc.Clips, req.AssetID, req.ClipID)
	if err != nil {
		return ExportResult{}, err
	}

	var preset domain.Preset
	if req.PresetID != "" {
		if preset, err = uc.preset(ctx, req.PresetID); err != nil {
			return ExportResult{}, err
		}
	}
	choice := resolveChoices(req, preset)
	if !choice.valid() {
		return ExportResult{}, &Error{Kind: Invalid, Msg: ChoicesOnOffer, Err: ErrInvalidChoice}
	}
	language, translate, err := uc.captionLanguage(ctx, req.AssetID, choice)
	if err != nil {
		return ExportResult{}, err
	}
	intro, outro, err := uc.vignettes(ctx, preset)
	if err != nil {
		return ExportResult{}, err
	}
	watermark, err := uc.watermark(ctx, preset)
	if err != nil {
		return ExportResult{}, err
	}
	src, err := uc.Assets.GetSource(ctx, req.AssetID)
	if isNotFound(err) {
		return ExportResult{}, fail(Conflict, "the asset has no source file")
	}
	if err != nil {
		return ExportResult{}, err
	}

	input, err := json.Marshal(jobcontracts.ExportClipInput{
		AssetID: req.AssetID, ClipID: clip.ID, StorageKey: src.StorageKey, Format: choice.Format,
		Captions: choice.Captions, CaptionStyle: choice.CaptionStyle,
		CaptionPosition: choice.CaptionPosition, CaptionColor: choice.CaptionColor, CaptionCaps: choice.CaptionCaps,
		CaptionLanguage: language, Intro: intro, Outro: outro, Watermark: watermark, Preset: preset.Name,
		OutputKey: storage.ClipKey(req.AssetID, clip.ID, choice.Format),
	})
	if err != nil {
		return ExportResult{}, err
	}
	now := uc.now()
	job := domain.Job{ID: uc.newID(), AssetID: req.AssetID, Type: jobcontracts.JobTypeExportClip, InputJSON: input,
		Status: domain.JobPending, CreatedAt: now}
	first, then := job, []domain.Job(nil)
	var translation *domain.Job
	if translate {
		in, _ := json.Marshal(jobcontracts.TranslateTranscriptInput{AssetID: req.AssetID, Language: language})
		first = domain.Job{ID: uc.newID(), AssetID: req.AssetID, Type: jobcontracts.JobTypeTranslateTranscript, InputJSON: in,
			Status: domain.JobPending, CreatedAt: now}
		job.CreatedAt = now.Add(time.Millisecond) // listed after the translation it waits for
		then, translation = []domain.Job{job}, &first
	}

	err = uc.Jobs.CreateJobIfIdle(ctx, first, "", then...)
	if errors.Is(err, domain.ErrBusy) {
		return ExportResult{}, &Error{Kind: Conflict, Msg: err.Error(), Err: err}
	}
	if err != nil {
		return ExportResult{}, err
	}
	if err := uc.Queue.Enqueue(ctx, jobcontracts.Message{JobID: first.ID, Type: first.Type, Payload: first.InputJSON}); err != nil {
		bg := context.WithoutCancel(ctx)
		uc.Jobs.FinishJob(bg, first.ID, fmt.Errorf("could not enqueue job: %w", err))
		if translate {
			uc.Jobs.FinishJob(bg, job.ID, errors.New("not run: the translation before it could not be queued"))
		}
		uc.log().Error("enqueue failed", "asset_id", req.AssetID, "job_id", first.ID, "error", err)
		return ExportResult{}, &Error{Kind: Unavailable, Msg: "the job could not be queued", Err: err}
	}

	saved, err := uc.Jobs.GetJob(ctx, job.ID)
	if err != nil {
		return ExportResult{}, err
	}
	if translate {
		uc.log().Info("clip export queued after a translation", "asset_id", req.AssetID, "clip_id", clip.ID, "format", choice.Format,
			"language", language, "job_id", job.ID, "translate_job_id", first.ID)
	} else {
		uc.log().Info("clip export queued", "asset_id", req.AssetID, "clip_id", clip.ID, "format", choice.Format, "job_id", job.ID)
	}
	return ExportResult{Job: saved, Translate: translation}, nil
}

func (uc ExportClip) preset(ctx context.Context, id string) (domain.Preset, error) {
	if !domain.IsID(id) {
		return domain.Preset{}, fail(Invalid, "unknown preset")
	}
	p, err := uc.Brand.GetPreset(ctx, id)
	if isNotFound(err) {
		return p, fail(Invalid, "unknown preset")
	}
	return p, err
}

// exportChoice is what the export will do once the preset and the defaults
// have filled in what the request left out.
type exportChoice struct {
	Format                                                       string
	Captions                                                     bool
	CaptionStyle, CaptionPosition, CaptionColor, CaptionLanguage string
	CaptionCaps                                                  *bool
}

// resolveChoices fills the choices the request left out from the preset (a
// zero Preset when there is none) and the defaults: captions on, the first
// style, at the bottom.
func resolveChoices(req ExportRequest, p domain.Preset) exportChoice {
	pick := func(v *string, fromPreset string) string {
		if v != nil {
			return *v
		}
		return fromPreset
	}
	c := exportChoice{
		Format: req.Format, Captions: p.ID == "" || p.Captions,
		CaptionStyle: pick(req.CaptionStyle, p.CaptionStyle), CaptionPosition: pick(req.CaptionPosition, p.CaptionPosition),
		CaptionColor: pick(req.CaptionColor, p.CaptionColor), CaptionLanguage: pick(req.CaptionLanguage, p.CaptionLanguage),
		CaptionCaps: p.CaptionCaps,
	}
	if req.Captions != nil {
		c.Captions = *req.Captions
	}
	if req.CaptionCaps != nil {
		c.CaptionCaps = req.CaptionCaps
	}
	if c.CaptionStyle == "" {
		c.CaptionStyle = jobcontracts.CaptionStyles[0]
	}
	if c.CaptionPosition == jobcontracts.CaptionPositions[0] {
		c.CaptionPosition = "" // the default, as before presets
	}
	return c
}

func (c exportChoice) valid() bool {
	in := func(list []string, v string) bool { return v == "" || slices.Contains(list, v) }
	return slices.Contains(jobcontracts.ClipFormats, c.Format) && slices.Contains(jobcontracts.CaptionStyles, c.CaptionStyle) &&
		in(jobcontracts.CaptionPositions, c.CaptionPosition) && in(jobcontracts.CaptionColors, c.CaptionColor) &&
		in(jobcontracts.CaptionLanguages, c.CaptionLanguage)
}

// captionLanguage returns the language the captions are burned in (empty
// for the transcript's own) and whether the transcript must be translated
// into it first. Asking for the transcript's own language is asking for no
// translation, and without captions the language does not matter.
func (uc ExportClip) captionLanguage(ctx context.Context, assetID string, c exportChoice) (string, bool, error) {
	if !c.Captions || c.CaptionLanguage == "" {
		return "", false, nil
	}
	tr, err := uc.Speech.GetTranscript(ctx, assetID)
	if isNotFound(err) {
		return "", false, fail(Conflict, "this video has no transcript to translate the captions from")
	}
	if err != nil {
		return "", false, err
	}
	if tr.Language == c.CaptionLanguage {
		return "", false, nil
	}
	_, err = uc.Speech.GetTranslation(ctx, assetID, c.CaptionLanguage)
	if isNotFound(err) {
		return c.CaptionLanguage, true, nil
	}
	return c.CaptionLanguage, false, err
}

// vignettes resolves a preset's openings and closings into the storage keys
// the export joins: nil when it has none.
func (uc ExportClip) vignettes(ctx context.Context, p domain.Preset) (intro, outro *jobcontracts.Vignette, err error) {
	key := func(id string) (string, error) {
		if id == "" {
			return "", nil
		}
		v, err := uc.Brand.GetVignette(ctx, id)
		return v.StorageKey, err
	}
	pair := func(h, v string) (*jobcontracts.Vignette, error) {
		hk, err := key(h)
		if err != nil {
			return nil, err
		}
		vk, err := key(v)
		if err != nil || hk == "" && vk == "" {
			return nil, err
		}
		return &jobcontracts.Vignette{Horizontal: hk, Vertical: vk}, nil
	}
	if intro, err = pair(p.IntroHorizontal, p.IntroVertical); err != nil {
		return nil, nil, err
	}
	outro, err = pair(p.OutroHorizontal, p.OutroVertical)
	return intro, outro, err
}

// watermark resolves a preset's watermark into what the export lays over
// the clip: nil when it has none.
func (uc ExportClip) watermark(ctx context.Context, p domain.Preset) (*jobcontracts.Watermark, error) {
	if p.Watermark == "" {
		return nil, nil
	}
	img, err := uc.Brand.GetWatermark(ctx, p.Watermark)
	if err != nil {
		return nil, err
	}
	return &jobcontracts.Watermark{Key: img.StorageKey, Position: p.WatermarkPosition, Size: p.WatermarkSize, Opacity: p.WatermarkOpacity}, nil
}

func (uc ExportClip) log() *slog.Logger { return orDiscard(uc.Log) }
