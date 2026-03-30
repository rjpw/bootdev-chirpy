package api

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
)

func newTestServer() *Server {
	cfg := &config.Config{Metrics: &metrics.ServerMetrics{}}
	return NewServer(cfg, "./testdata")
}

func parseHitCount(t *testing.T, body string) int {
	t.Helper()
	re := regexp.MustCompile(`visited (\d+) times!`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatalf("no metric parsable from body: %q", body)
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil {
		t.Fatalf("could not parse hit count: %v", err)
	}
	return count
}

func TestHealthz(t *testing.T) {
	srv := newTestServer()
	r := httptest.NewRequest("GET", "/api/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("want OK, got %q", w.Body.String())
	}
}

func TestMetricsInitiallyZero(t *testing.T) {
	srv := newTestServer()
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
	// note that the state of metrics is being counted
	// so we define this server outside the loop of calls to /app
	srv := newTestServer()

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
			t.Errorf("want '%q`, got %q", tc.body, w.Body.String())
		}
	}

	// now look at the metrics value
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
	srv := newTestServer()

	// hit the root app once
	r := httptest.NewRequest("GET", "/app/hello.txt", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Body.String() != "Hello world!" {
		t.Errorf("want 'Hello world!`, got %q", w.Body.String())
	}

	if srv.cfg.Metrics.FileserverHits() != 1 {
		t.Fatalf("expected 1 hits after routed call")
	}

	r = httptest.NewRequest("POST", "/admin/reset", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if srv.cfg.Metrics.FileserverHits() != 0 {
		t.Errorf("expected 0 hits after reset, got %d", srv.cfg.Metrics.FileserverHits())
	}
}

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
		srv := newTestServer() // ensure stateless operations
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
