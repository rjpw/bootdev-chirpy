package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsInitiallyZero(t *testing.T) {
	srv := newTestServer("dev")
	r := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}

	hitCount := parseHitCount(t, w.Body.String())
	if hitCount != 0 {
		t.Errorf("want 'visited 0 times!', got '%d'", hitCount)
	}
}

func TestMetricsReflectHits(t *testing.T) {
	srv := newTestServer("dev")

	cases := []struct {
		method      string
		path        string
		contentType string
		body        string
	}{
		{"GET", "/app/hello.txt", "text/plain; charset=utf-8", "Hello world!"},
		{"GET", "/app/hello.txt", "text/plain; charset=utf-8", "Hello world!"},
		{"GET", "/app/hello.txt", "text/plain; charset=utf-8", "Hello world!"},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Body.String() != tc.body {
			t.Errorf("want %q, got %q", tc.body, w.Body.String())
		}
	}

	r := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}

	hitCount := parseHitCount(t, w.Body.String())
	if hitCount != len(cases) {
		t.Errorf("want %d hits, got %d", len(cases), hitCount)
	}
}

func TestResetClearsHits(t *testing.T) {
	srv := newTestServer("dev")

	r := httptest.NewRequest("GET", "/app/hello.txt", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Body.String() != "Hello world!" {
		t.Errorf("want 'Hello world!', got %q", w.Body.String())
	}
	if srv.cfg.Metrics.FileserverHits() != 1 {
		t.Fatalf("expected 1 hit after routed call")
	}

	r = httptest.NewRequest("POST", "/admin/reset", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if srv.cfg.Metrics.FileserverHits() != 0 {
		t.Errorf("expected 0 hits after reset, got %d", srv.cfg.Metrics.FileserverHits())
	}
}
