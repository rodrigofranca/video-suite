package httpapi

import "net/http"

// deleteAsset handles DELETE /assets/{id}: app.DeleteVideo removes the
// video, everything recorded about it and all its stored files
// (docs/adr/0027); 409 while it is uploading or a job is pending or running.
func (a *api) deleteAsset(w http.ResponseWriter, r *http.Request) {
	a.noContent(w, r, a.videoDeletes.Run(r.Context(), r.PathValue("id")))
}

// deleteClip handles DELETE /assets/{id}/clips/{clipId}: app.DeleteClip
// removes the clip and its exported files. The export jobs stay in the history.
func (a *api) deleteClip(w http.ResponseWriter, r *http.Request) {
	a.noContent(w, r, a.clipDeletes.Run(r.Context(), r.PathValue("id"), r.PathValue("clipId")))
}

// noContent answers 204, or the use case's error.
func (a *api) noContent(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
