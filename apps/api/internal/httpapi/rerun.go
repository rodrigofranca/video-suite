package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// createJob handles POST /assets/{id}/jobs with {"type": "<job type>"}: it
// runs one pipeline job again for an asset whose jobs have all finished. The
// new job is a new row, so earlier attempts stay in the history
// (docs/adr/0015). Only that job runs, and the jobs that work on its result
// (jobcontracts.RerunWith), which the worker queues after it.
func (a *api) createJob(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, `expected a JSON body like {"type": "transcode-hls"}`)
		return
	}
	ctx := r.Context()
	src, err := a.Store.GetSource(ctx, asset.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "the asset has no source file")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	input, err := jobInput(body.Type, asset.ID, src.StorageKey)
	if errors.Is(err, errUnknownJobType) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	job := domain.Job{
		ID: domain.NewID(), AssetID: asset.ID, Type: body.Type, InputJSON: input,
		Status: domain.JobPending, CreatedAt: time.Now(),
	}
	var then []domain.Job
	for _, typ := range jobcontracts.RerunWith[body.Type] {
		in, err := jobInput(typ, asset.ID, src.StorageKey)
		if err != nil {
			a.internalError(w, r, err)
			return
		}
		then = append(then, domain.Job{ID: domain.NewID(), AssetID: asset.ID, Type: typ, InputJSON: in, CreatedAt: job.CreatedAt})
	}
	// The HLS job decides whether the asset is ready (docs/adr/0003), so the
	// asset is processing again until it ends.
	assetStatus := ""
	if job.Type == jobcontracts.JobTypeTranscodeHLS {
		assetStatus = domain.AssetProcessing
	}
	err = a.Store.CreateJobIfIdle(ctx, job, assetStatus, then...)
	if errors.Is(err, store.ErrBusy) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	if err := a.Enqueue(ctx, jobcontracts.Message{JobID: job.ID, Type: job.Type, Payload: job.InputJSON}); err != nil {
		bg := context.WithoutCancel(ctx)
		a.Store.FinishJob(bg, job.ID, fmt.Errorf("could not enqueue job: %w", err))
		for _, j := range then {
			a.Store.FinishJob(bg, j.ID, errors.New("not run: the job before it could not be queued"))
		}
		if assetStatus != "" {
			a.Store.SetAssetStatus(bg, asset.ID, domain.AssetFailed)
		}
		a.Log.Error("enqueue failed", "asset_id", asset.ID, "job_id", job.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "the job could not be queued")
		return
	}

	saved, err := a.Store.GetJob(ctx, job.ID)
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	a.Log.Info("job queued again", "asset_id", asset.ID, "job_id", job.ID, "type", job.Type)
	writeJSON(w, http.StatusAccepted, toJob(saved))
}
