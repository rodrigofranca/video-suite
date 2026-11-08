// Package httpapi wires the HTTP routes of the API.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"ultramedia/media-core/domain"
	"ultramedia/media-core/storage"
	"ultramedia/media-core/store"

	jobcontracts "ultramedia/job-contracts"
)

// Pinger reports whether the database is reachable (satisfied by *pgxpool.Pool).
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps are the services the handlers use.
type Deps struct {
	DB      Pinger
	Store   *store.Store
	Storage *storage.Client
	// Enqueue adds a message to the jobs stream.
	Enqueue func(ctx context.Context, msg jobcontracts.Message) error
	// MaxUploadBytes limits the size of an uploaded file. Defaults to
	// domain.MaxUploadBytes.
	MaxUploadBytes int64
	// AllowedOrigin is the dashboard origin allowed by CORS (e.g. http://localhost:5173).
	AllowedOrigin string
	Log           *slog.Logger
}

type api struct {
	Deps
}

// NewRouter returns the API routes.
func NewRouter(deps Deps) http.Handler {
	if deps.MaxUploadBytes <= 0 {
		deps.MaxUploadBytes = domain.MaxUploadBytes
	}
	if deps.Log == nil {
		deps.Log = slog.New(slog.DiscardHandler)
	}
	a := &api{deps}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", a.health)

	mux.HandleFunc("POST /assets/upload", a.upload)
	mux.HandleFunc("GET /assets", a.listAssets)
	mux.HandleFunc("GET /assets/{id}", a.getAsset)
	mux.HandleFunc("DELETE /assets/{id}", a.deleteAsset)
	mux.HandleFunc("GET /assets/{id}/jobs", a.listAssetJobs)
	mux.HandleFunc("GET /assets/{id}/artifacts", a.listArtifacts)
	mux.HandleFunc("GET /assets/{id}/playback", a.playback)
	mux.HandleFunc("GET /assets/{id}/metadata", a.getMetadata)
	mux.HandleFunc("GET /assets/{id}/transcript", a.getTranscript)
	mux.HandleFunc("GET /assets/{id}/analysis", a.getAnalysis)
	mux.HandleFunc("GET /assets/{id}/clips", a.listClips)
	mux.HandleFunc("PATCH /assets/{id}/clips/{clipId}", a.editClip)
	mux.HandleFunc("DELETE /assets/{id}/clips/{clipId}", a.deleteClip)
	mux.HandleFunc("POST /assets/{id}/clips/{clipId}/exports", a.exportClip)
	mux.HandleFunc("GET /clips/{id}/download", a.downloadClip)
	mux.HandleFunc("POST /assets/{id}/jobs", a.createJob)

	mux.HandleFunc("GET /jobs/{id}", a.getJob)
	mux.HandleFunc("GET /jobs/{id}/logs", a.listJobLogs)

	return cors(deps.AllowedOrigin, mux)
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// cors lets the dashboard, served from another origin, call the API.
func cors(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == allowedOrigin {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
