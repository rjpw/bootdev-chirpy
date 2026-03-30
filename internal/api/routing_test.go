package api

import (
	"net/http/httptest"
	"testing"
)

func TestMethodResponseCodes(t *testing.T) {
	cases := []struct {
		method string
		path   string
		code   int
	}{
		{"GET", "/app/cant-touch-this.txt", 404},
		{"GET", "/admin/metrics", 200},
		{"POST", "/admin/metrics", 405},
		{"DELETE", "/admin/metrics", 405},
		{"GET", "/api/healthz", 200},
		{"POST", "/api/healthz", 405},
		{"GET", "/admin/reset", 405},
		{"POST", "/admin/reset", 200},
	}
	for _, tc := range cases {
		srv := newTestServer()
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Errorf("%s %s: want %d, got %d", tc.method, tc.path, tc.code, w.Code)
		}
	}
}

func TestContentType(t *testing.T) {
	cases := []struct {
		method      string
		path        string
		contentType string
	}{
		{"GET", "/api/healthz", "text/plain; charset=utf-8"},
		{"GET", "/admin/metrics", "text/html; charset=utf-8"},
		{"POST", "/admin/reset", "text/plain; charset=utf-8"},
	}
	for _, tc := range cases {
		srv := newTestServer()
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
	}
}
