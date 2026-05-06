package httpapi_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
)

// Define and initialize testdataFS
//
//go:embed testdata/*
var testdataFS embed.FS

func TestValidateChirpAPI(t *testing.T) {
	// Test cases for validating inbound API calls to /api/validate_chirp
	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "empty body", body: "", wantCode: 400},
		{name: "min body length = 1", body: strings.Repeat("x", 1), wantCode: 201},
		{name: "max body length = 140", body: strings.Repeat("x", 140), wantCode: 201},
		{name: "oversize body", body: strings.Repeat("x", 141), wantCode: 400},
	}

	srv := newTestServer("dev")

	// get a user to post with
	payload := fmt.Sprintf("{\"email\": \"%s\"}", "saul@bettercall.com")
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(payload))
	rep := httptest.NewRecorder()
	srv.ServeHTTP(rep, req)

	user, err := decodeEntity[domain.User](t, rep.Body.String())
	if err != nil {
		t.Fatalf("Could not create user: %s", err.Error())
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := httpapi.ChirpParams{Body: tc.body, UserID: user.ID}
			payload, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("json.Marshal(params) error: %v", err)
			}

			r := httptest.NewRequest(
				"POST",
				"/api/chirps",
				strings.NewReader(string(payload)),
			)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Errorf("POST /api/chirps: want status %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

// function to read raw test data from `internal/api/testdata` and return as a string
func readTestData(t *testing.T, filename string) string {
	t.Helper()
	data, err := testdataFS.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatalf("testdataFS.ReadFile(%q) error: %v", filename, err)
	}
	return string(data)
}

func decodeEntity[T any](t *testing.T, rawData string) (T, error) {
	t.Helper()
	var v T
	if err := json.NewDecoder(strings.NewReader(rawData)).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}
