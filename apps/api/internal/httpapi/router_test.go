package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Tests in this file need no database or storage.

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestRoutesWithoutServices(t *testing.T) {
	cases := []struct {
		name       string
		db         Pinger
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"health ok", fakeDB{}, "GET", "/healthz", 200, `{"status":"ok"}`},
		{"health db down", fakeDB{errors.New("down")}, "GET", "/healthz", 503, `"status":"error"`},
		{"malformed asset id", fakeDB{}, "GET", "/assets/not-a-uuid", 404, `{"error":"not found"}`},
		{"malformed job id", fakeDB{}, "GET", "/jobs/123/logs", 404, `{"error":"not found"}`},
		{"bad status filter", fakeDB{}, "GET", "/assets?status=weird", 400, `unknown status`},
		{"re-run with malformed id", fakeDB{}, "POST", "/assets/nope/jobs", 404, `not found`},
		{"upload without multipart", fakeDB{}, "POST", "/assets/upload", 400, `multipart/form-data`},
		{"wrong method", fakeDB{}, "DELETE", "/assets", 405, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(Deps{DB: tc.db}).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantStatus, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	h := NewRouter(Deps{DB: fakeDB{}, AllowedOrigin: "http://localhost:5173"})

	req := httptest.NewRequest(http.MethodOptions, "/assets/upload", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("preflight: status %d, headers %v", rec.Code, rec.Header())
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.test")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("other origin got Access-Control-Allow-Origin %q", got)
	}
}

func TestIsMP4(t *testing.T) {
	if !isMP4([]byte("\x00\x00\x00\x20ftypisom")) {
		t.Error("MP4 header not recognised")
	}
	if isMP4([]byte("GIF89a......")) || isMP4([]byte("ftyp")) {
		t.Error("non-MP4 accepted")
	}
}

func TestCleanFilename(t *testing.T) {
	cases := map[string]string{
		"clip.mp4":                        "clip.mp4",
		"../../etc/passwd.mp4":            "passwd.mp4",
		`C:\Users\me\video.mp4`:           "video.mp4",
		"":                                "upload.mp4",
		strings.Repeat("a", 300) + ".mp4": strings.Repeat("a", 251) + ".mp4",
	}
	for in, want := range cases {
		if got := cleanFilename(in); got != want {
			t.Errorf("cleanFilename(%.20q) = %.20q, want %.20q", in, got, want)
		}
	}
}

func TestClipFilename(t *testing.T) {
	for name, want := range map[string]string{
		"My Talk (final).mp4": "My-Talk-final-clip-2-vertical.mp4",
		"😀.mp4":               "video-clip-2-vertical.mp4",
		"aula_01-intro.MP4":   "aula_01-intro-clip-2-vertical.mp4",
	} {
		if got := clipFilename(name, 2, "vertical"); got != want {
			t.Errorf("clipFilename(%q) = %q, want %q", name, got, want)
		}
	}
}
