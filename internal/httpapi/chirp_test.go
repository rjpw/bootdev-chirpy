package httpapi

import (
	"embed"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
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
		{name: "min body length = 1", body: strings.Repeat("x", 1), wantCode: 200},
		{name: "max body length = 140", body: strings.Repeat("x", 140), wantCode: 200},
		{name: "oversize body", body: strings.Repeat("x", 141), wantCode: 400},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := parameters{Body: tc.body}
			payload, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("json.Marshal(params) error: %v", err)
			}
			srv := newTestServer("dev")

			r := httptest.NewRequest(
				"POST",
				"/api/validate_chirp",
				strings.NewReader(string(payload)),
			)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Errorf("POST /api/validate_chirp: want status %d, got %d", tc.wantCode, w.Code)
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

func TestValidateChirpFilter(t *testing.T) {
	// Test cases for validating inbound API calls to /api/validate_chirp
	cases := []struct {
		name     string
		body     string
		reply    string
		wantCode int
	}{
		{
			name:     "clean message",
			body:     readTestData(t, "tc_filter_001.json"),
			reply:    readTestData(t, "tc_filter_001_reply.json"),
			wantCode: 200,
		},
		{
			name:     "extra element",
			body:     readTestData(t, "tc_filter_002.json"),
			reply:    readTestData(t, "tc_filter_002_reply.json"),
			wantCode: 200,
		},
		{
			name:     "blue streak",
			body:     readTestData(t, "tc_filter_003.json"),
			reply:    readTestData(t, "tc_filter_003_reply.json"),
			wantCode: 200,
		},
		{
			name:     "long lorem ipsum",
			body:     readTestData(t, "tc_filter_004.json"),
			reply:    readTestData(t, "tc_filter_004_reply.json"),
			wantCode: 400,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer("dev")
			r := httptest.NewRequest("POST", "/api/validate_chirp", strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Errorf("POST /api/validate_chirp: want status %d, got %d", tc.wantCode, w.Code)
			}

			if w.Body.String() != tc.reply {
				t.Errorf(
					"POST /api/validate_chirp: want body %q, got %q",
					tc.reply,
					w.Body.String(),
				)
			}
		})
	}
}
