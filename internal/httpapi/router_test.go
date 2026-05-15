package httpapi_test

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/memory"
)

// Define and initialize testdataFS
//
//go:embed testdata/*
var testdataFS embed.FS

func newTestServer() *httpapi.ChirpyAPIRouter {
	// note: this loads internal/httpapi/testdata/.env
	env := application.LoadEnvironment()

	repo := memory.NewMemoryRepository()
	repositories := application.Repositories{
		Users:        repo,
		UserSessions: repo,
		Chirps:       repo,
	}
	return httpapi.NewRouter(
		env,
		&application.ServerMetrics{},
		&repositories,
		"./testdata")
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

func issueAuthorizedRequest(
	srv *httpapi.ChirpyAPIRouter,
	method, path, authValue string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	r.Header.Add("Authorization", authValue)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func issueRequest(
	srv *httpapi.ChirpyAPIRouter,
	method, path string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func marshalEntity[T any](t *testing.T, params T) string {
	t.Helper()
	entity, err := httpapi.MarshalEntity(params)
	if err != nil {
		t.Fatalf("Error decoding entity %q", err)
	}
	return entity
}

func decodeEntity[T any](t *testing.T, rawData string) (T, error) {
	t.Helper()
	v, err := httpapi.DecodeEntity[T](rawData)
	if err != nil {
		t.Fatalf("Error decoding entity %q", err)
		return v, err
	}
	return v, nil
}

func getFileReader(t *testing.T, filename string) fs.File {
	t.Helper()

	file, err := testdataFS.Open(fmt.Sprintf("testdata/%s", filename))
	if err != nil {
		t.Fatalf("testdataFS.Open(%q) error: %v", filename, err)
	}
	return file
}

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
		{"PUT", "/api/chirps", 405, ""},
	}
	for _, tc := range cases {
		srv := newTestServer()
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
			"/api/chirps",
			"application/json; charset=utf-8",
			"{body: \"hello world\"}",
		},
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
