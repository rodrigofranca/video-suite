package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"

	jobcontracts "ultramedia/job-contracts"
)

// multipartOverhead is the room left for multipart headers and boundaries on
// top of the file size limit.
const multipartOverhead = 1 << 20

// thumbnailAtSecond is where the thumbnail is taken until the metadata job
// provides the duration.
const thumbnailAtSecond = 3

var errTooLarge = errors.New("file too large")

// upload handles POST /assets/upload (multipart/form-data, field "file").
// The file is streamed to storage without being buffered in memory or on disk.
func (a *api) upload(w http.ResponseWriter, r *http.Request) {
	limit := a.MaxUploadBytes
	if r.ContentLength > limit+multipartOverhead {
		a.tooLarge(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+multipartOverhead)

	part, err := filePart(r)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			a.tooLarge(w)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	filename := cleanFilename(part.FileName())
	if !strings.EqualFold(filepath.Ext(filename), ".mp4") {
		writeError(w, http.StatusUnsupportedMediaType, "only .mp4 files are accepted")
		return
	}
	head := make([]byte, 12)
	if _, err := io.ReadFull(part, head); err != nil || !isMP4(head) {
		writeError(w, http.StatusUnsupportedMediaType, "the file is not an MP4 video")
		return
	}

	ctx := r.Context()
	asset := domain.Asset{ID: domain.NewID(), Filename: filename, Status: domain.AssetUploading}
	if err := a.Store.CreateAsset(ctx, asset); err != nil {
		a.internalError(w, r, err)
		return
	}

	key := storage.SourceKey(asset.ID)
	body := &countingReader{r: io.MultiReader(bytes.NewReader(head), part), hash: sha256.New(), limit: limit}
	err = a.Storage.Upload(ctx, key, body, "video/mp4")
	if err == nil && body.err != nil {
		err = body.err
	}
	if err != nil {
		a.discardUpload(ctx, asset.ID, key)
		var maxErr *http.MaxBytesError
		if errors.Is(body.err, errTooLarge) || errors.As(body.err, &maxErr) {
			a.tooLarge(w)
			return
		}
		a.Log.Error("upload failed", "asset_id", asset.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	src := domain.Source{
		ID: domain.NewID(), AssetID: asset.ID, StorageKey: key, MimeType: "video/mp4",
		SizeBytes: body.n, Checksum: hex.EncodeToString(body.hash.Sum(nil)),
	}
	jobs, err := pipelineJobs(asset.ID, key)
	if err == nil {
		err = a.Store.CompleteUpload(ctx, src, jobs)
	}
	if err != nil {
		a.discardUpload(ctx, asset.ID, key)
		a.internalError(w, r, err)
		return
	}

	first := jobs[0]
	msg := jobcontracts.Message{JobID: first.ID, Type: first.Type, Payload: first.InputJSON}
	if err := a.Enqueue(ctx, msg); err != nil {
		// The file is stored but nothing will process it: say so in the job.
		bg := context.WithoutCancel(ctx)
		a.Store.FinishJob(bg, first.ID, fmt.Errorf("could not enqueue job: %w", err))
		a.Store.SetAssetStatus(bg, asset.ID, domain.AssetFailed)
		a.Log.Error("enqueue failed", "asset_id", asset.ID, "job_id", first.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "the file was stored but processing could not start")
		return
	}

	a.Log.Info("asset uploaded", "asset_id", asset.ID, "bytes", src.SizeBytes)
	writeJSON(w, http.StatusCreated, map[string]string{"assetId": asset.ID, "status": domain.AssetProcessing})
}

// filePart returns the multipart part named "file".
func filePart(r *http.Request) (*multipart.Part, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, errors.New("expected multipart/form-data with a file field")
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return nil, errors.New("missing file field")
		}
		if err != nil {
			return nil, err
		}
		if part.FormName() == "file" {
			return part, nil
		}
	}
}

// pipelineJobs builds one pending job per pipeline step, in order.
func pipelineJobs(assetID, sourceKey string) ([]domain.Job, error) {
	base := time.Now()
	jobs := make([]domain.Job, 0, len(jobcontracts.Pipeline))
	for i, jobType := range jobcontracts.Pipeline {
		payload, err := jobInput(jobType, assetID, sourceKey)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, domain.Job{
			ID: domain.NewID(), AssetID: assetID, Type: jobType, InputJSON: payload,
			Status: domain.JobPending,
			// Distinct timestamps keep the pipeline order when listing jobs.
			CreatedAt: base.Add(time.Duration(i) * time.Millisecond),
		})
	}
	return jobs, nil
}

// errUnknownJobType is returned for a job type outside the pipeline.
var errUnknownJobType = errors.New("unknown job type")

// jobInput builds the JSON input of a pipeline job for an asset.
func jobInput(jobType, assetID, sourceKey string) ([]byte, error) {
	var input any
	switch jobType {
	case jobcontracts.JobTypeExtractMetadata:
		input = jobcontracts.ExtractMetadataInput{AssetID: assetID, StorageKey: sourceKey}
	case jobcontracts.JobTypeGenerateThumbnail:
		input = jobcontracts.GenerateThumbnailInput{AssetID: assetID, StorageKey: sourceKey, AtSecond: thumbnailAtSecond}
	case jobcontracts.JobTypeTranscodeHLS:
		input = jobcontracts.TranscodeHLSInput{AssetID: assetID, StorageKey: sourceKey}
	case jobcontracts.JobTypeExtractAudio:
		input = jobcontracts.ExtractAudioInput{AssetID: assetID, StorageKey: sourceKey, AudioKey: storage.AudioKey(assetID)}
	case jobcontracts.JobTypeTranscribeAudio:
		// Empty language: Whisper detects it.
		input = jobcontracts.TranscribeAudioInput{AssetID: assetID, AudioKey: storage.AudioKey(assetID)}
	default:
		return nil, fmt.Errorf("%w %q", errUnknownJobType, jobType)
	}
	return json.Marshal(input)
}

// discardUpload removes what a failed upload left behind. It runs even if
// the client went away.
func (a *api) discardUpload(ctx context.Context, assetID, key string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := a.Storage.Delete(ctx, key); err != nil {
		a.Log.Warn("could not delete partial upload", "key", key, "error", err)
	}
	if err := a.Store.DeleteAsset(ctx, assetID); err != nil {
		a.Log.Warn("could not delete asset of failed upload", "asset_id", assetID, "error", err)
	}
}

func (a *api) tooLarge(w http.ResponseWriter) {
	writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file is larger than %d MB", a.MaxUploadBytes>>20))
}

// isMP4 checks for the "ftyp" box that starts every MP4 (ISO BMFF) file.
func isMP4(head []byte) bool {
	return len(head) >= 8 && string(head[4:8]) == "ftyp"
}

// cleanFilename keeps only the base name, for display. It is never used to
// build a storage key.
func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "." || name == "/" {
		name = "upload.mp4"
	}
	if len(name) > 255 {
		name = name[len(name)-255:]
	}
	return name
}

// countingReader hashes and counts what passes through it, stops after limit
// bytes and remembers the first read error (to tell "too large" apart from
// storage errors).
type countingReader struct {
	r     io.Reader
	hash  hash.Hash
	n     int64
	limit int64
	err   error
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	c.hash.Write(p[:n])
	if c.n > c.limit {
		c.err = errTooLarge
		return n, errTooLarge
	}
	if err != nil && err != io.EOF && c.err == nil {
		c.err = err
	}
	return n, err
}
