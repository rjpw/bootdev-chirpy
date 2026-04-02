package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

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
			srv := newTestServer()

			r := httptest.NewRequest("POST", "/api/validate_chirp", strings.NewReader(string(payload)))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Errorf("POST /api/validate_chirp: want status %d, got %d", tc.wantCode, w.Code)
			}

		})
	}

}
