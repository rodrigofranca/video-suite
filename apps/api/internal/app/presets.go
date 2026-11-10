package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"ultramedia/media-core/domain"

	jobcontracts "ultramedia/job-contracts"
)

// SavePreset creates and replaces export presets (docs/adr/0030, 0031):
// a name, the caption choices, the openings and closings, and the
// watermark. The files a preset names must be in the library.
type SavePreset struct {
	Brand  Brand
	Writer PresetWriter
	clock
}

// PresetInput is a preset as asked for. Omitted caption choices get the
// defaults (captions on, the first style, at the bottom), and omitted
// watermark choices too (top right, 15%, 80%).
type PresetInput struct {
	Name                                                           string
	Captions                                                       *bool
	CaptionStyle, CaptionPosition, CaptionColor, CaptionLanguage   string
	CaptionCaps                                                    *bool
	IntroHorizontal, IntroVertical, OutroHorizontal, OutroVertical *string
	WatermarkID                                                    *string
	WatermarkPosition                                              string
	WatermarkSize, WatermarkOpacity                                *int
}

// Create saves a new preset and returns it as saved.
func (uc SavePreset) Create(ctx context.Context, in PresetInput) (domain.Preset, error) {
	p, err := uc.check(ctx, in)
	if err != nil {
		return p, err
	}
	p.ID = uc.newID()
	return uc.save(ctx, p, uc.Writer.CreatePreset)
}

// Update replaces a preset's choices. Exports made before keep their files.
func (uc SavePreset) Update(ctx context.Context, id string, in PresetInput) (domain.Preset, error) {
	if !domain.IsID(id) {
		return domain.Preset{}, fail(NotFound, "not found")
	}
	if _, err := uc.Brand.GetPreset(ctx, id); err != nil {
		if isNotFound(err) {
			return domain.Preset{}, fail(NotFound, "not found")
		}
		return domain.Preset{}, err
	}
	p, err := uc.check(ctx, in)
	if err != nil {
		return p, err
	}
	p.ID = id
	return uc.save(ctx, p, uc.Writer.UpdatePreset)
}

func (uc SavePreset) save(ctx context.Context, p domain.Preset, write func(context.Context, domain.Preset) error) (domain.Preset, error) {
	err := write(ctx, p)
	switch {
	case errors.Is(err, domain.ErrNameTaken):
		return p, &Error{Kind: Conflict, Msg: fmt.Sprintf("another preset is called %q", p.Name), Err: err}
	case isNotFound(err):
		return p, fail(NotFound, "not found")
	case err != nil:
		return p, err
	}
	return uc.Brand.GetPreset(ctx, p.ID)
}

// check fills in the defaults and checks the choices and the files.
func (uc SavePreset) check(ctx context.Context, in PresetInput) (domain.Preset, error) {
	name, ok := cleanName(in.Name)
	if !ok {
		return domain.Preset{}, fail(Invalid, fmt.Sprintf("the name must have 1 to %d characters", maxName))
	}
	p := domain.Preset{Name: name, Captions: in.Captions == nil || *in.Captions, CaptionStyle: in.CaptionStyle,
		CaptionPosition: in.CaptionPosition, CaptionColor: in.CaptionColor, CaptionCaps: in.CaptionCaps, CaptionLanguage: in.CaptionLanguage}
	if p.CaptionStyle == "" {
		p.CaptionStyle = jobcontracts.CaptionStyles[0]
	}
	if p.CaptionPosition == "" {
		p.CaptionPosition = jobcontracts.CaptionPositions[0]
	}
	if !slices.Contains(jobcontracts.CaptionStyles, p.CaptionStyle) || !slices.Contains(jobcontracts.CaptionPositions, p.CaptionPosition) ||
		(p.CaptionColor != "" && !slices.Contains(jobcontracts.CaptionColors, p.CaptionColor)) ||
		(p.CaptionLanguage != "" && !slices.Contains(jobcontracts.CaptionLanguages, p.CaptionLanguage)) {
		return domain.Preset{}, fail(Invalid, "caption styles: "+strings.Join(jobcontracts.CaptionStyles, ", ")+
			"; positions: "+strings.Join(jobcontracts.CaptionPositions, ", ")+"; colors (or empty): "+strings.Join(jobcontracts.CaptionColors, ", ")+
			"; languages (or empty): "+strings.Join(jobcontracts.CaptionLanguages, ", "))
	}

	slots := []struct {
		in  *string
		out *string
	}{{in.IntroHorizontal, &p.IntroHorizontal}, {in.IntroVertical, &p.IntroVertical}, {in.OutroHorizontal, &p.OutroHorizontal}, {in.OutroVertical, &p.OutroVertical}}
	for _, s := range slots {
		if s.in == nil || *s.in == "" {
			continue
		}
		if err := inLibrary(ctx, *s.in, "vignette", func(id string) error { _, err := uc.Brand.GetVignette(ctx, id); return err }); err != nil {
			return domain.Preset{}, err
		}
		*s.out = *s.in
	}

	p.WatermarkPosition, p.WatermarkSize, p.WatermarkOpacity = in.WatermarkPosition, 15, 80
	if p.WatermarkPosition == "" {
		p.WatermarkPosition = jobcontracts.WatermarkPositions[0]
	}
	if in.WatermarkSize != nil {
		p.WatermarkSize = *in.WatermarkSize
	}
	if in.WatermarkOpacity != nil {
		p.WatermarkOpacity = *in.WatermarkOpacity
	}
	if !slices.Contains(jobcontracts.WatermarkPositions, p.WatermarkPosition) ||
		p.WatermarkSize < jobcontracts.MinWatermarkSize || p.WatermarkSize > jobcontracts.MaxWatermarkSize ||
		p.WatermarkOpacity < jobcontracts.MinWatermarkOpacity || p.WatermarkOpacity > jobcontracts.MaxWatermarkOpacity {
		return domain.Preset{}, fail(Invalid, fmt.Sprintf("watermark positions: %s; size: %d to %d (%% of the shorter side); opacity: %d to %d (%%)",
			strings.Join(jobcontracts.WatermarkPositions, ", "), jobcontracts.MinWatermarkSize, jobcontracts.MaxWatermarkSize,
			jobcontracts.MinWatermarkOpacity, jobcontracts.MaxWatermarkOpacity))
	}
	if in.WatermarkID != nil && *in.WatermarkID != "" {
		if err := inLibrary(ctx, *in.WatermarkID, "watermark", func(id string) error { _, err := uc.Brand.GetWatermark(ctx, id); return err }); err != nil {
			return domain.Preset{}, err
		}
		p.Watermark = *in.WatermarkID
	}
	return p, nil
}

// inLibrary answers Invalid for a file ID that is malformed or not in the library.
func inLibrary(ctx context.Context, id, what string, get func(string) error) error {
	if !domain.IsID(id) {
		return fail(Invalid, fmt.Sprintf("unknown %s %q", what, id))
	}
	err := get(id)
	if isNotFound(err) {
		return fail(Invalid, fmt.Sprintf("unknown %s %q", what, id))
	}
	return err
}

// DeletePreset removes a preset. Its files stay in the library, and exports
// made with it keep their files.
type DeletePreset struct {
	Writer PresetWriter
}

func (uc DeletePreset) Run(ctx context.Context, id string) error {
	if !domain.IsID(id) {
		return fail(NotFound, "not found")
	}
	err := uc.Writer.DeletePreset(ctx, id)
	if isNotFound(err) {
		return fail(NotFound, "not found")
	}
	return err
}
