//go:build integration

package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/testdb"
)

var srv *config.Server

func TestMain(m *testing.M) {
	url, cleanup, err := testdb.SetupURL()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	env := application.Environment{
		DBURL:     url,
		Platform:  "dev",
		SecretKey: "test-secret-key",
	}

	srv, err = config.NewServer(env, "../../root")
	if err != nil {
		panic(err)
	}
	defer srv.Close()

	os.Exit(m.Run())
}

func issueRequest(method, path string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	return w
}

func TestHealthz(t *testing.T) {
	w := issueRequest("GET", "/api/healthz", "")
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}
