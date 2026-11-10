package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// thumbnailAtSecond is where the thumbnail is taken until the metadata job
// provides the duration.
const thumbnailAtSecond = 3

// ErrUnknownJobType is returned for a job type that is not in the job
// table; ErrNotRerunnable for one that runs only with an export.
var (
	ErrUnknownJobType = errors.New("unknown job type")
	ErrNotRerunnable  = errors.New("job type cannot be run again by hand")
)

// inputs builds, for each job type that can run again by hand (Rerun in
// jobcontracts.JobTypes), its input from the video alone. A test checks
// that it covers exactly those types.
var inputs = map[string]func(assetID, sourceKey string) any{
	jobcontracts.JobTypeExtractMetadata: func(a, k string) any { return jobcontracts.ExtractMetadataInput{AssetID: a, StorageKey: k} },
	jobcontracts.JobTypeGenerateThumbnail: func(a, k string) any {
		return jobcontracts.GenerateThumbnailInput{AssetID: a, StorageKey: k, AtSecond: thumbnailAtSecond}
	},
	jobcontracts.JobTypeTranscodeHLS: func(a, k string) any { return jobcontracts.TranscodeHLSInput{AssetID: a, StorageKey: k} },
	jobcontracts.JobTypeExtractAudio: func(a, k string) any {
		return jobcontracts.ExtractAudioInput{AssetID: a, StorageKey: k, AudioKey: storage.AudioKey(a)}
	},
	// Empty language: Whisper detects it.
	jobcontracts.JobTypeTranscribeAudio: func(a, _ string) any {
		return jobcontracts.TranscribeAudioInput{AssetID: a, AudioKey: storage.AudioKey(a)}
	},
	jobcontracts.JobTypeAnalyzeTranscript: func(a, _ string) any { return jobcontracts.AnalyzeTranscriptInput{AssetID: a} },
	jobcontracts.JobTypeSuggestClips:      func(a, _ string) any { return jobcontracts.SuggestClipsInput{AssetID: a} },
	jobcontracts.JobTypeTrackFaces:        func(a, k string) any { return jobcontracts.TrackFacesInput{AssetID: a, StorageKey: k} },
}

// JobInput builds the JSON input of a pipeline job for an asset.
func JobInput(jobType, assetID, sourceKey string) ([]byte, error) {
	build, ok := inputs[jobType]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownJobType, jobType)
	}
	return json.Marshal(build(assetID, sourceKey))
}

// pipelineJobs builds one pending job per pipeline step, in order, with
// distinct times that keep the order when the jobs are listed.
func pipelineJobs(assetID, sourceKey string, base time.Time, newID func() string) ([]domain.Job, error) {
	jobs := make([]domain.Job, 0, len(jobcontracts.Pipeline))
	for i, jobType := range jobcontracts.Pipeline {
		payload, err := JobInput(jobType, assetID, sourceKey)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, domain.Job{ID: newID(), AssetID: assetID, Type: jobType, InputJSON: payload,
			Status: domain.JobPending, CreatedAt: base.Add(time.Duration(i) * time.Millisecond)})
	}
	return jobs, nil
}

// clock and IDs, fixed by tests.
type clock struct {
	Now   func() time.Time
	NewID func() string
}

func (c clock) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c clock) newID() string {
	if c.NewID != nil {
		return c.NewID()
	}
	return domain.NewID()
}

// ---- Upload ----

// UploadVideo stores an uploaded MP4 and starts its pipeline (docs/adr/0009):
// it creates the asset, streams the file to storage while counting and
// hashing it, and records the source with one job per pipeline step, the
// first one queued (docs/adr/0034).
type UploadVideo struct {
	Assets   AssetWriter
	Files    Files
	Log      *slog.Logger
	MaxBytes int64
	clock
}

// UploadRequest is the file as it arrives.
type UploadRequest struct {
	Filename string
	Body     io.Reader
}

// Run returns the new asset's ID.
func (uc UploadVideo) Run(ctx context.Context, req UploadRequest) (string, error) {
	log := orDiscard(uc.Log)
	filename := CleanFilename(req.Filename)
	if !strings.EqualFold(filepath.Ext(filename), ".mp4") {
		return "", fail(Unsupported, "only .mp4 files are accepted")
	}
	head := make([]byte, 12)
	if _, err := io.ReadFull(req.Body, head); err != nil || !isMP4(head) {
		return "", fail(Unsupported, "the file is not an MP4 video")
	}

	asset := domain.Asset{ID: uc.newID(), Filename: filename, Status: domain.AssetUploading}
	if err := uc.Assets.CreateAsset(ctx, asset); err != nil {
		return "", err
	}
	key := storage.SourceKey(asset.ID)
	body := &countingReader{r: io.MultiReader(strings.NewReader(string(head)), req.Body), hash: sha256.New(), limit: uc.MaxBytes}
	err := uc.Files.Upload(ctx, key, body, "video/mp4")
	if err == nil && body.err != nil {
		err = body.err
	}
	if err != nil {
		uc.discard(ctx, asset.ID, key)
		if errors.Is(body.err, errTooLarge) {
			return "", fail(TooLarge, fmt.Sprintf("file is larger than %d MB", uc.MaxBytes>>20))
		}
		cause := err
		if body.err != nil {
			cause = body.err // the request's own error, such as a body over its size limit
		}
		log.Error("upload failed", "asset_id", asset.ID, "error", err)
		return "", &Error{Kind: Unavailable, Msg: "upload failed", Err: cause}
	}

	src := domain.Source{ID: uc.newID(), AssetID: asset.ID, StorageKey: key, MimeType: "video/mp4",
		SizeBytes: body.n, Checksum: hex.EncodeToString(body.hash.Sum(nil))}
	jobs, err := pipelineJobs(asset.ID, key, uc.now(), uc.newID)
	if err == nil {
		err = uc.Assets.CompleteUpload(ctx, src, jobs)
	}
	if err != nil {
		uc.discard(ctx, asset.ID, key)
		return "", err
	}
	log.Info("asset uploaded", "asset_id", asset.ID, "bytes", src.SizeBytes)
	return asset.ID, nil
}

// discard removes what a failed upload left behind, even if the caller went away.
func (uc UploadVideo) discard(ctx context.Context, assetID, key string) {
	log := orDiscard(uc.Log)
	removeFile(ctx, uc.Files, log, key)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := uc.Assets.DeleteAsset(ctx, assetID); err != nil {
		log.Warn("could not delete asset of failed upload", "asset_id", assetID, "error", err)
	}
}

// ---- Delete ----

// DeleteVideo removes a video, everything recorded about it and all its
// stored files (docs/adr/0027). The database goes first: once it is done the
// video is gone for everyone, and a storage failure only leaves files
// behind, which are logged with where they are. It waits until no job of the
// asset is pending or running, and an upload in progress cannot be deleted.
type DeleteVideo struct {
	Assets Assets
	Writer AssetWriter
	Files  Files
	Log    *slog.Logger
}

func (uc DeleteVideo) Run(ctx context.Context, assetID string) error {
	if err := assetExists(ctx, uc.Assets, assetID); err != nil {
		return err
	}
	if err := deleteProblem(uc.Writer.DeleteAssetIfIdle(ctx, assetID), "the video"); err != nil {
		return err
	}
	prefix := storage.AssetPrefix(assetID)
	n, err := uc.Files.DeletePrefix(context.WithoutCancel(ctx), prefix)
	logDeleted(orDiscard(uc.Log), err, "asset deleted", "asset_id", assetID, "files", n, "prefix", prefix)
	return nil
}

// assetExists answers NotFound for a malformed or unknown asset.
func assetExists(ctx context.Context, assets Assets, id string) error {
	if !domain.IsID(id) {
		return fail(NotFound, "not found")
	}
	_, err := assets.GetAsset(ctx, id)
	if isNotFound(err) {
		return fail(NotFound, "not found")
	}
	return err
}

// deleteProblem turns a failed delete into the answer for the caller.
func deleteProblem(err error, what string) error {
	switch {
	case err == nil:
		return nil
	case isNotFound(err):
		return fail(NotFound, "not found")
	case errors.Is(err, domain.ErrBusy):
		return &Error{Kind: Conflict, Msg: what + " cannot be deleted while it is uploading or a job is pending or running", Err: err}
	}
	return err
}

// logDeleted logs a deletion. The records are already gone, so a storage
// error only leaves files behind: it is logged rather than failing.
func logDeleted(log *slog.Logger, err error, msg string, args ...any) {
	if err != nil {
		log.Error(msg+", but some files could not be removed", append(args, "error", err)...)
		return
	}
	log.Info(msg, args...)
}

// ---- Run again ----

// RerunJob runs one pipeline job of an asset again, as a new job row so the
// earlier attempts stay in the history (docs/adr/0015), with the jobs that
// work on its result after it (jobcontracts.RerunWith). Only when every job
// of the asset has finished. Running HLS again makes the asset processing
// until it ends (docs/adr/0003). The job is queued with it (docs/adr/0034).
type RerunJob struct {
	Assets Assets
	Jobs   Jobs
	Log    *slog.Logger
	clock
}

func (uc RerunJob) Run(ctx context.Context, assetID, jobType string) (domain.Job, error) {
	kind, ok := jobcontracts.Lookup(jobType)
	switch {
	case !ok:
		err := fmt.Errorf("%w %q", ErrUnknownJobType, jobType)
		return domain.Job{}, &Error{Kind: Invalid, Msg: err.Error(), Err: err}
	case !kind.Rerun:
		return domain.Job{}, &Error{Kind: Invalid, Err: ErrNotRerunnable,
			Msg: fmt.Sprintf("%s jobs cannot be run again by hand: they run when a clip is exported", kind.Label)}
	}
	if err := assetExists(ctx, uc.Assets, assetID); err != nil {
		return domain.Job{}, err
	}
	src, err := uc.Assets.GetSource(ctx, assetID)
	if isNotFound(err) {
		return domain.Job{}, fail(Conflict, "the asset has no source file")
	}
	if err != nil {
		return domain.Job{}, err
	}
	input, err := JobInput(jobType, assetID, src.StorageKey)
	if errors.Is(err, ErrUnknownJobType) {
		return domain.Job{}, &Error{Kind: Invalid, Msg: err.Error(), Err: err}
	}
	if err != nil {
		return domain.Job{}, err
	}
	now := uc.now()
	job := domain.Job{ID: uc.newID(), AssetID: assetID, Type: jobType, InputJSON: input, Status: domain.JobPending, CreatedAt: now}
	var then []domain.Job
	for _, typ := range jobcontracts.RerunWith[jobType] {
		in, err := JobInput(typ, assetID, src.StorageKey)
		if err != nil {
			return domain.Job{}, err
		}
		then = append(then, domain.Job{ID: uc.newID(), AssetID: assetID, Type: typ, InputJSON: in, CreatedAt: now})
	}
	assetStatus := ""
	if jobType == jobcontracts.JobTypeTranscodeHLS {
		assetStatus = domain.AssetProcessing
	}
	err = uc.Jobs.CreateJobIfIdle(ctx, job, assetStatus, then...)
	if errors.Is(err, domain.ErrBusy) {
		return domain.Job{}, &Error{Kind: Conflict, Msg: err.Error(), Err: err}
	}
	if err != nil {
		return domain.Job{}, err
	}
	saved, err := uc.Jobs.GetJob(ctx, job.ID)
	if err != nil {
		return domain.Job{}, err
	}
	orDiscard(uc.Log).Info("job queued again", "asset_id", assetID, "job_id", job.ID, "type", job.Type)
	return saved, nil
}
