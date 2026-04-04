package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := newTestServer("dev")
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
