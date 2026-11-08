package httpapi

import (
	"context"
	"errors"
	"net/http"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"
)

// deleteAsset handles DELETE /assets/{id}: it removes the video, everything
// recorded about it and all its stored files (docs/adr/0027). Like a re-run,
// it waits until no job of the asset is pending or running (409), and an
// upload in progress cannot be deleted.
func (a *api) deleteAsset(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	err := a.Store.DeleteAssetIfIdle(r.Context(), asset.ID)
	if a.deleteFailed(w, r, err, "the video") {
		return
	}
	n, err := a.Storage.DeletePrefix(context.WithoutCancel(r.Context()), storage.AssetPrefix(asset.ID))
	a.logDeleted(err, "asset deleted", "asset_id", asset.ID, "files", n, "prefix", storage.AssetPrefix(asset.ID))
	w.WriteHeader(http.StatusNoContent)
}

// deleteClip handles DELETE /assets/{id}/clips/{clipId}: it removes the clip
// and its exported files. The export jobs stay in the history.
func (a *api) deleteClip(w http.ResponseWriter, r *http.Request) {
	asset, ok := a.loadAsset(w, r)
	if !ok {
		return
	}
	clipID := r.PathValue("clipId")
	if !domain.IsID(clipID) {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	err := a.Store.DeleteClipIfIdle(r.Context(), asset.ID, clipID)
	if a.deleteFailed(w, r, err, "the clip") {
		return
	}
	n, err := a.Storage.DeletePrefix(context.WithoutCancel(r.Context()), storage.ClipPrefix(asset.ID, clipID))
	a.logDeleted(err, "clip deleted", "asset_id", asset.ID, "clip_id", clipID, "files", n)
	w.WriteHeader(http.StatusNoContent)
}

// deleteFailed answers a failed delete and reports whether it did.
func (a *api) deleteFailed(w http.ResponseWriter, r *http.Request, err error, what string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrBusy):
		writeError(w, http.StatusConflict, what+" cannot be deleted while it is uploading or a job is pending or running")
	default:
		a.internalError(w, r, err)
	}
	return true
}

// logDeleted logs a deletion. The records are already gone, so a storage
// error only leaves files behind: it is logged, with where they are, rather
// than failing the request.
func (a *api) logDeleted(err error, msg string, args ...any) {
	if err != nil {
		a.Log.Error(msg+", but some files could not be removed", append(args, "error", err)...)
		return
	}
	a.Log.Info(msg, args...)
}
