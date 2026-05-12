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
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
)

func marshalEntity[T any](t *testing.T, params T) string {
	b, err := json.Marshal(params)
	if err != nil {
		t.Errorf("Error creating user to post: %v", err)
	}
	return string(b)
}

func getUserPayload(t *testing.T, email, password string) string {
	t.Helper()
	params := httpapi.PostLoginRequest{Email: email, Password: password}
	return marshalEntity(t, params)
}

func TestUserFromRawString(t *testing.T) {
	cases := []struct {
		name         string
		method       string
		path         string
		contentType  string
		email        string
		password     string
		responseCode int
	}{
		{
			"reset server",
			"POST",
			"/admin/reset",
			"text/plain; charset=utf-8",
			"",
			"",
			200,
		},
		{
			"create user",
			"POST",
			"/api/users",
			"application/json; charset=utf-8",
			"user@example.com",
			"123456",
			201,
		},
	}
	for _, tc := range cases {
		srv := newTestServer()
		payload := fmt.Sprintf("{\"password\": \"%s\", \"email\": \"%s\"}", tc.password, tc.email)
		fmt.Printf("User to POST: %s\n", payload)

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

func TestUserFromParams(t *testing.T) {
	cases := []struct {
		name         string
		method       string
		path         string
		contentType  string
		email        string
		password     string
		responseCode int
	}{
		{
			"reset server",
			"POST",
			"/admin/reset",
			"text/plain; charset=utf-8",
			"",
			"",
			200,
		},
		{
			"create user",
			"POST",
			"/api/users",
			"application/json; charset=utf-8",
			"user@example.com",
			"123456",
			201,
		},
	}
	for _, tc := range cases {
		srv := newTestServer()

		params := httpapi.PostLoginRequest{Email: tc.email, Password: tc.password}
		b, err := json.Marshal(params)
		if err != nil {
			t.Errorf("Error creating user to post: %v", err)
		}
		payload := string(b)
		fmt.Printf("User to POST: %s\n", payload)

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
	srv := newTestServer()

	cases := []struct {
		name         string
		method       string
		path         string
		contentType  string
		email        string
		password     string
		responseCode int
	}{
		{
			"reset server",
			"POST",
			"/admin/reset",
			"text/plain; charset=utf-8",
			"",
			"",
			http.StatusOK,
		},
		{
			"create new user",
			"POST",
			"/api/users",
			"application/json; charset=utf-8",
			"user@example.com",
			"123456",
			http.StatusCreated,
		},
		{
			"create conflicting user",
			"POST",
			"/api/users",
			"text/plain; charset=utf-8",
			"user@example.com",
			"123456",
			http.StatusConflict,
		},
	}
	for _, tc := range cases {
		payload := fmt.Sprintf("{\"password\": \"%s\", \"email\": \"%s\"}", tc.password, tc.email)
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

func TestLoginProducesToken(t *testing.T) {
	srv := newTestServer()

	cases := []struct {
		name         string
		method       string
		path         string
		email        string
		password     string
		hasGoodToken bool
	}{
		{
			"create user",
			"POST",
			"/api/users",
			"user@example.com",
			"123456",
			false,
		},
		{
			"login with correct password",
			"POST",
			"/api/login",
			"user@example.com",
			"123456",
			true,
		},
		{
			"login with incorrect password",
			"POST",
			"/api/login",
			"user@example.com",
			"000111222",
			false,
		},
	}
	for _, tc := range cases {

		payload := getUserPayload(t, tc.email, tc.password)

		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(payload))
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, r)
		if tc.path == "/api/login" {

			var user domain.User
			err := json.NewDecoder(strings.NewReader(w.Body.String())).Decode(&user)

			// if login returned OK we expect a user with a good token
			if w.Code == http.StatusOK {

				if err != nil {
					t.Errorf("%s -- Error decoding user %v", tc.name, err)
					continue
				} else if len(user.AccessToken) == 0 {
					t.Errorf("%s -- Expecting a populated token and got none", tc.name)
				}

			} else {
				if err == nil && len(user.AccessToken) > 0 {
					t.Errorf("%s -- Expecting no token and got %s", tc.name, user.AccessToken)
				}
			}

		}

	}
}

func TestMITMTokenTheftScenario(t *testing.T) {
	srv := newTestServer()

	cases := []struct {
		name          string
		path          string
		email         string
		mitmAttempted bool
		responseCode  int
	}{
		{
			"create user saul",
			"/api/users",
			"saul@bettercall.com",
			false,
			http.StatusCreated,
		},
		{
			"create user mike",
			"/api/users",
			"mike@bettercall.com",
			false,
			http.StatusCreated,
		},
		{
			"saul logs in",
			"/api/login",
			"saul@bettercall.com",
			false,
			http.StatusOK,
		},
		{
			"mike logs in",
			"/api/login",
			"mike@bettercall.com",
			false,
			http.StatusOK,
		},
		{
			"saul chirps as saul",
			"/api/chirps",
			"saul@bettercall.com",
			false,
			http.StatusCreated,
		},
		{
			"mike chirps as mike",
			"/api/chirps",
			"mike@bettercall.com",
			false,
			http.StatusCreated,
		},
		// {
		// 	"saul chirps as mike",
		// 	"/api/chirps",
		// 	"mitm@sneaky.com",
		// 	true,
		// 	http.StatusCreated, // this should succeed in lesson 7 and fail when we fix it later
		// },
	}

	tokenCache := make(map[string]string)
	userCache := make(map[string]domain.User)

	// reset the server for good measure
	r := httptest.NewRequest(http.MethodPost, "/admin/reset", strings.NewReader(""))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	for _, tc := range cases {

		var r *http.Request

		switch tc.path {
		case "/api/users":
			fallthrough
		case "/api/login":
			r = httptest.NewRequest(http.MethodPost, tc.path, getFileReader(t, fmt.Sprintf("UserParams_%s.json", tc.email)))
		case "/api/chirps":
			r = httptest.NewRequest(http.MethodPost, tc.path, getFileReader(t, fmt.Sprintf("ChirpParams_%s.json", tc.email)))
			r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenCache[tc.email]))
		}

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)

		// keep tokens for later MITM attempt by Saul
		if tc.path == "/api/login" && w.Code == http.StatusOK {
			user, _ := decodeEntity[domain.User](t, w.Body.String())
			fmt.Printf("Retaining user %v for later use ...\n", user)
			tokenCache[tc.email] = user.AccessToken
			userCache[tc.email] = user
		}

		// expect rejection on bad password
		if w.Code != tc.responseCode {
			fmt.Printf("\n\nResponse Body: %s\n\n", w.Body.String())
			t.Errorf("%s -- %s -- Expected response code %d, got %d", tc.name, tc.path, tc.responseCode, w.Code)
		}

	}
}
