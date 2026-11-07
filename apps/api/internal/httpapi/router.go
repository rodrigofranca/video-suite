// Package httpapi wires the HTTP routes of the API.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
)

// Pinger reports whether the database is reachable (satisfied by *pgxpool.Pool).
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewRouter returns the API routes. Most are still stubs (MVP1 foundation).
func NewRouter(db Pinger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /assets/upload", notImplemented)
	mux.HandleFunc("GET /assets", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"assets": []any{}})
	})
	mux.HandleFunc("GET /assets/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	mux.HandleFunc("GET /assets/{id}/jobs", notImplemented)
	mux.HandleFunc("POST /assets/{id}/jobs", notImplemented)
	mux.HandleFunc("GET /jobs/{id}", notImplemented)
	mux.HandleFunc("GET /assets/{id}/artifacts", notImplemented)

	return mux
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
