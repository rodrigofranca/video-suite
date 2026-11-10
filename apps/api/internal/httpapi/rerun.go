package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

// createJob handles POST /assets/{id}/jobs with {"type": "<job type>"}:
// app.RerunJob runs one pipeline job again for an asset whose jobs have all
// finished, with the jobs that work on its result (docs/adr/0015).
func (a *api) createJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, `expected a JSON body like {"type": "transcode-hls"}`)
		return
	}
	job, err := a.reruns.Run(r.Context(), r.PathValue("id"), body.Type)
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toJob(job))
}
