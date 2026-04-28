package httpapi_test

// This file contains tests for the API server's user-related endpoints.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

func TestCreateOneUser(t *testing.T) {
	cases := []struct {
		method       string
		path         string
		contentType  string
		email        string
		responseCode int
	}{
		{
			"POST",
			"/admin/reset",
			"text/plain; charset=utf-8",
			"",
			200,
		},
		{
			"POST",
			"/api/users",
			"application/json; charset=utf-8",
			"user@example.com",
			201,
		},
	}
	for _, tc := range cases {
		srv := newTestServer("dev")
		payload := fmt.Sprintf("{\"email\": \"%s\"}", tc.email)
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(payload))
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, r)
		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Code != tc.responseCode {
			t.Errorf("Expected response code %d, got %d", tc.responseCode, w.Code)
		}

		if tc.path == "/api/users" {
			// parse user from w.Body
			var user domain.User
			data, err := io.ReadAll(w.Body)
			if err != nil {
				t.Errorf("Error %s reading reply", err.Error())
			}
			if err := json.Unmarshal(data, &user); err != nil {
				t.Errorf("Error %s decoding reply", err.Error())
			}
			if user.Email != tc.email {
				t.Errorf("Want new user returned email %q, got %q", tc.email, user.Email)
			}
		}
	}

}

func TestCreateUserConflict(t *testing.T) {

	srv := newTestServer("dev")

	cases := []struct {
		method       string
		path         string
		contentType  string
		email        string
		responseCode int
	}{
		{
			"POST",
			"/admin/reset",
			"text/plain; charset=utf-8",
			"",
			http.StatusOK,
		},
		{
			"POST",
			"/api/users",
			"application/json; charset=utf-8",
			"user@example.com",
			http.StatusCreated,
		},
		{
			"POST",
			"/api/users",
			"text/plain; charset=utf-8",
			"user@example.com",
			http.StatusConflict,
		},
	}
	for _, tc := range cases {
		payload := fmt.Sprintf("{\"email\": \"%s\"}", tc.email)
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(payload))
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, r)
		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Code != tc.responseCode {
			t.Errorf("Expected response code %d, got %d", tc.responseCode, w.Code)
		}

	}

}
