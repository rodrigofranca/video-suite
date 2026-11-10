package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// CreateClip makes a clip from a quote in the transcript (docs/adr/0039):
// a range someone chose, which the dashboard snaps to the words of the
// quote. Without a title, the title is what was said in the range. The clip
// keeps to domain.Clip.Problem and comes first in the asset's clips. Like an
// edit, it waits until no job of the asset is pending or running, and it
// queues a track-faces job for the clip in the same transaction.
type CreateClip struct {
	Assets   Assets
	Clips    Clips
	Metadata Metadata
	Speech   Speech
	Writer   ClipWriter
	Log      *slog.Logger
	clock
}

// CreateRequest is the range, in seconds from the start of the video (kept
// to the millisecond), and an optional title.
type CreateRequest struct {
	AssetID    string
	Start, End float64
	Title      string
}

// Run returns the clip as saved.
func (uc CreateClip) Run(ctx context.Context, req CreateRequest) (domain.Clip, error) {
	if err := assetExists(ctx, uc.Assets, req.AssetID); err != nil {
		return domain.Clip{}, err
	}
	clip := domain.Clip{
		ID: uc.newID(), AssetID: req.AssetID, Origin: domain.ClipQuote,
		Start: math.Round(req.Start*1000) / 1000, End: math.Round(req.End*1000) / 1000,
		Title: strings.Join(strings.Fields(req.Title), " "), CreatedAt: uc.now(),
	}
	if clip.Title == "" {
		said, err := uc.said(ctx, clip)
		if err != nil {
			return clip, err
		}
		clip.Title = domain.QuoteTitle(said)
	}
	if clip.Title == "" && clip.End > clip.Start {
		clip.Title = "Clip at " + clockTime(clip.Start)
	}
	duration, err := videoDuration(ctx, uc.Metadata, req.AssetID)
	if err != nil {
		return clip, err
	}
	if problem := clip.Problem(duration); problem != "" {
		return clip, fail(Invalid, problem)
	}

	src, err := uc.Assets.GetSource(ctx, req.AssetID)
	if err != nil {
		return clip, err
	}
	input, _ := json.Marshal(jobcontracts.TrackFacesInput{AssetID: req.AssetID, StorageKey: src.StorageKey, ClipID: clip.ID})
	track := domain.Job{ID: uc.newID(), AssetID: req.AssetID, Type: jobcontracts.JobTypeTrackFaces, InputJSON: input,
		Status: domain.JobPending, CreatedAt: uc.now()}
	err = uc.Writer.CreateClipIfIdle(ctx, clip, track)
	switch {
	case errors.Is(err, domain.ErrBusy):
		return clip, &Error{Kind: Conflict, Msg: err.Error(), Err: err}
	case isNotFound(err):
		return clip, fail(NotFound, "not found")
	case err != nil:
		return clip, err
	}
	orDiscard(uc.Log).Info("clip made from a quote", "asset_id", req.AssetID, "clip_id", clip.ID, "start", clip.Start, "end", clip.End)
	return uc.Clips.GetClip(ctx, clip.ID)
}

// said is what was said in the clip's range; "" without a transcript.
func (uc CreateClip) said(ctx context.Context, c domain.Clip) (string, error) {
	t, err := uc.Speech.GetTranscript(ctx, c.AssetID)
	if isNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return t.Said(c.Start, c.End), nil
}

// clockTime writes seconds as m:ss.
func clockTime(seconds float64) string {
	s := int(math.Max(seconds, 0))
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

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
	duration, err := videoDuration(ctx, uc.Metadata, req.AssetID)
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
func videoDuration(ctx context.Context, metadata Metadata, assetID string) (float64, error) {
	m, err := metadata.GetTechnicalMetadata(ctx, assetID)
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
