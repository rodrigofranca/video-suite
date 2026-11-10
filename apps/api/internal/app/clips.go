package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strings"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// EditClip changes a clip's start, end or title for the exports from then
// on (docs/adr/0024). The clip must keep to domain.Clip.Problem. New times
// clear its face track and queue a track-faces job for this clip only, in
// the same transaction. Like an export, it waits until no job of the asset
// is pending or running.
type EditClip struct {
	Assets   Assets
	Clips    Clips
	Metadata Metadata
	Writer   ClipWriter
	Log      *slog.Logger
	clock
}

// EditRequest holds the changes; nil fields stay as they are. Times are
// seconds from the start of the video, kept to the millisecond.
type EditRequest struct {
	AssetID, ClipID string
	Start, End      *float64
	Title           *string
}

// Run returns the clip as saved.
func (uc EditClip) Run(ctx context.Context, req EditRequest) (domain.Clip, error) {
	clip, err := clipOf(ctx, uc.Assets, uc.Clips, req.AssetID, req.ClipID)
	if err != nil {
		return clip, err
	}
	if req.Start == nil && req.End == nil && req.Title == nil {
		return clip, fail(Invalid, "nothing to change: give a start, an end or a title")
	}
	edited := clip
	if req.Start != nil {
		edited.Start = math.Round(*req.Start*1000) / 1000
	}
	if req.End != nil {
		edited.End = math.Round(*req.End*1000) / 1000
	}
	if req.Title != nil {
		edited.Title = strings.Join(strings.Fields(*req.Title), " ")
	}
	duration, err := uc.videoDuration(ctx, req.AssetID)
	if err != nil {
		return clip, err
	}
	if problem := edited.Problem(duration); problem != "" {
		return clip, fail(Invalid, problem)
	}
	if edited.Start == clip.Start && edited.End == clip.End && edited.Title == clip.Title {
		return clip, nil // nothing changed
	}

	var retrack *domain.Job
	if edited.Start != clip.Start || edited.End != clip.End {
		src, err := uc.Assets.GetSource(ctx, req.AssetID)
		if err != nil {
			return clip, err
		}
		input, _ := json.Marshal(jobcontracts.TrackFacesInput{AssetID: req.AssetID, StorageKey: src.StorageKey, ClipID: clip.ID})
		retrack = &domain.Job{ID: uc.newID(), AssetID: req.AssetID, Type: jobcontracts.JobTypeTrackFaces, InputJSON: input,
			Status: domain.JobPending, CreatedAt: uc.now()}
	}
	err = uc.Writer.EditClipIfIdle(ctx, edited, retrack)
	switch {
	case errors.Is(err, domain.ErrBusy):
		return clip, &Error{Kind: Conflict, Msg: err.Error(), Err: err}
	case isNotFound(err):
		return clip, fail(NotFound, "clip not found")
	case err != nil:
		return clip, err
	}
	orDiscard(uc.Log).Info("clip edited", "asset_id", req.AssetID, "clip_id", clip.ID, "start", edited.Start, "end", edited.End)
	return uc.Clips.GetClip(ctx, clip.ID)
}

// videoDuration is the video's length from its metadata, or 0 when the
// metadata job has not run.
func (uc EditClip) videoDuration(ctx context.Context, assetID string) (float64, error) {
	m, err := uc.Metadata.GetTechnicalMetadata(ctx, assetID)
	if isNotFound(err) {
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

// clipOf loads a clip of an asset, answering NotFound for a malformed or
// unknown asset or clip, or a clip of another asset.
func clipOf(ctx context.Context, assets Assets, clips Clips, assetID, clipID string) (domain.Clip, error) {
	if err := assetExists(ctx, assets, assetID); err != nil {
		return domain.Clip{}, err
	}
	if !domain.IsID(clipID) {
		return domain.Clip{}, fail(NotFound, "clip not found")
	}
	clip, err := clips.GetClip(ctx, clipID)
	if isNotFound(err) || (err == nil && clip.AssetID != assetID) {
		return domain.Clip{}, fail(NotFound, "clip not found")
	}
	return clip, err
}

// DeleteClip removes a clip and its exported files; its export jobs stay
// in the history (docs/adr/0027). Only while no job of the asset runs.
type DeleteClip struct {
	Assets Assets
	Writer ClipWriter
	Files  Files
	Log    *slog.Logger
}

func (uc DeleteClip) Run(ctx context.Context, assetID, clipID string) error {
	if err := assetExists(ctx, uc.Assets, assetID); err != nil {
		return err
	}
	if !domain.IsID(clipID) {
		return fail(NotFound, "clip not found")
	}
	if err := deleteProblem(uc.Writer.DeleteClipIfIdle(ctx, assetID, clipID), "the clip"); err != nil {
		return err
	}
	n, err := uc.Files.DeletePrefix(context.WithoutCancel(ctx), storage.ClipPrefix(assetID, clipID))
	logDeleted(orDiscard(uc.Log), err, "clip deleted", "asset_id", assetID, "clip_id", clipID, "files", n)
	return nil
}
