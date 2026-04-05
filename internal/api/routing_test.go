package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMethodResponseCodes(t *testing.T) {
	cases := []struct {
		method string
		path   string
		code   int
		body   string
	}{
		{"GET", "/app/cant-touch-this.txt", 404, ""},
		{"GET", "/admin/metrics", 200, ""},
		{"POST", "/admin/metrics", 405, ""},
		{"DELETE", "/admin/metrics", 405, ""},
		{"GET", "/api/healthz", 200, ""},
		{"POST", "/api/healthz", 405, ""},
		{"GET", "/admin/reset", 405, ""},
		{"POST", "/admin/reset", 200, ""}, // forbidden in production, but 200 in platform "dev"
		{"GET", "/api/validate_chirp", 405, ""},
		{"PUT", "/api/validate_chirp", 405, ""},
		{"POST", "/api/validate_chirp", 200, "{\"body\":\"hello world\"}"},
		{"POST", "/api/validate_chirp", 400, "{body: \"hello world\"}"},
		{"POST", "/api/validate_chirp", 400, "{}"},
	}
	for _, tc := range cases {
		srv := newTestServer("dev")
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(string(tc.body)))
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
		body        string
	}{
		{"GET", "/api/healthz", "text/plain; charset=utf-8", ""},
		{"GET", "/admin/metrics", "text/html; charset=utf-8", ""},
		{"POST", "/admin/reset", "text/plain; charset=utf-8", ""},
		{
			"POST",
			"/api/validate_chirp",
			"application/json; charset=utf-8",
			"{body: \"hello world\"}",
		},
	}
	for _, tc := range cases {
		srv := newTestServer("dev")
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
	}
}
