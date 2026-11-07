package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestRoutes(t *testing.T) {
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
		{"upload stub", fakeDB{}, "POST", "/assets/upload", 501, `{"error":"not implemented"}`},
		{"list assets", fakeDB{}, "GET", "/assets", 200, `{"assets":[]}`},
		{"unknown asset", fakeDB{}, "GET", "/assets/00000000-0000-0000-0000-000000000000", 404, `{"error":"not found"}`},
		{"asset jobs stub", fakeDB{}, "GET", "/assets/x/jobs", 501, `not implemented`},
		{"wrong method", fakeDB{}, "DELETE", "/assets", 405, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(tc.db).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestJSONContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}
