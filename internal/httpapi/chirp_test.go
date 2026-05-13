package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
)

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

	srv := newTestServer()

	// get a user to post with
	req := httptest.NewRequest("POST", "/api/users", getFileReader(t, "UserParams_with_expiry.json"))
	rep := httptest.NewRecorder()
	srv.ServeHTTP(rep, req)

	// get a token to use
	req = httptest.NewRequest("POST", "/api/login", getFileReader(t, "UserParams_with_expiry.json"))
	rep = httptest.NewRecorder()
	srv.ServeHTTP(rep, req)

	user, err := decodeEntity[httpapi.PostLoginResponse](t, rep.Body.String())
	if err != nil {
		t.Fatalf("Could not create user: %s", err.Error())
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := httpapi.ChirpParams{Body: tc.body}
			payload, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("json.Marshal(params) error: %v", err)
			}

			r := httptest.NewRequest(
				"POST",
				"/api/chirps",
				strings.NewReader(string(payload)),
			)
			r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", user.AccessToken))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Errorf("POST /api/chirps: want status %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}
