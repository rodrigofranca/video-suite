package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"ultramedia/api/internal/db"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	conn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(nil)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			writeJSON(w, map[string]string{"status": "error", "error": err.Error()})
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /assets/upload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"error": "not implemented"})
	})

	mux.HandleFunc("GET /assets", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"assets": []any{}})
	})

	mux.HandleFunc("GET /assets/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "not found"})
	})

	mux.HandleFunc("GET /assets/{id}/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"error": "not implemented"})
	})

	mux.HandleFunc("POST /assets/{id}/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"error": "not implemented"})
	})

	mux.HandleFunc("GET /jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"error": "not implemented"})
	})

	mux.HandleFunc("GET /assets/{id}/artifacts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		writeJSON(w, map[string]string{"error": "not implemented"})
	})

	addr := ":8080"
	fmt.Printf("API listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
