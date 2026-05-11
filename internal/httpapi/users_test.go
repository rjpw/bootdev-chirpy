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

		params := httpapi.UserParams{Email: tc.email, Password: tc.password}
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
		payload := fmt.Sprintf("{\"password\": \"%s\", \"email\": \"%s\"}", tc.password, tc.email)
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
				} else if len(user.Token) == 0 {
					t.Errorf("%s -- Expecting a populated token and got none", tc.name)
				}

			} else {
				if err == nil && len(user.Token) > 0 {
					t.Errorf("%s -- Expecting no token and got %s", tc.name, user.Token)
				}
			}

		}

	}
}

func _TestMITMTokenTheftScenario(t *testing.T) {
	srv := newTestServer()

	cases := []struct {
		name          string
		method        string
		path          string
		email         string
		password      string
		chirpBody     string
		mitmAttempted bool
		responseCode  int
	}{
		{
			"create user saul",
			"POST",
			"/api/users",
			"saul@bettercall.com",
			"123456",
			"",
			false,
			http.StatusCreated,
		},
		{
			"create user mike",
			"POST",
			"/api/users",
			"mike@bettercall.com",
			"987654",
			"",
			false,
			http.StatusCreated,
		},
		{
			"login with correct password",
			"POST",
			"/api/login",
			"saul@bettercall.com",
			"123456",
			"",
			false,
			http.StatusOK,
		},
		{
			"login with incorrect password",
			"POST",
			"/api/login",
			"saul@bettercall.com",
			"000111222",
			"",
			false,
			http.StatusUnauthorized,
		},
		{
			"saul chirps as saul",
			"POST",
			"/api/chirp",
			"saul@bettercall.com",
			"123456",
			"Yo Adrian, Rocky called… he wants his face back!",
			false,
			http.StatusOK,
		},
		{
			"mike chirps as mike",
			"POST",
			"/api/chirp",
			"mike@bettercall.com",
			"123456",
			"You can be on one side of the law or the other, but if you make a deal with somebody, you keep your word.",
			false,
			http.StatusOK,
		},
		{
			"saul chirps as mike",
			"POST",
			"/api/chirp",
			"mike@bettercall.com",
			"123456",
			"Is this thing on?",
			true,
			http.StatusOK, // this should succeed in lesson 7 and fail when we fix it later
		},
	}

	tokenCache := make(map[string]string)
	userCache := make(map[string]domain.User)

	for _, tc := range cases {

		var payload string
		var user domain.User

		switch tc.path {
		case "/api/login":
			params := httpapi.UserParams{Email: tc.email, Password: tc.password}
			b, err := json.Marshal(params)
			if err != nil {
				t.Errorf("Error creating user to post: %v", err)
			} else {
				payload = string(b)
				fmt.Printf("User to POST: %s\n", payload)
			}
		case "/api/chirps":
			if tc.mitmAttempted && tc.email == "" {
				user = userCache[tc.email]
			}
			params := httpapi.ChirpParams{Body: tc.chirpBody, UserID: user.ID}
			b, err := json.Marshal(params)
			if err != nil {
				t.Errorf("Error creating chirp: %v", err)
			} else {
				payload = string(b)
				fmt.Printf("Chirp to POST: %s\n", payload)
			}
		}

		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(payload))
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, r)

		// keep tokens for later MITM attempt by Saul
		if w.Code == http.StatusOK && tc.path == "/api/login" {
			user, _ := decodeEntity[domain.User](t, w.Body.String())
			tokenCache[tc.email] = user.Token
			userCache[tc.email] = user
		}

		// expect rejection on bad password
		if w.Code != tc.responseCode {
			t.Errorf("%s -- %s -- Expected response code %d, got %d", tc.name, tc.path, tc.responseCode, w.Code)
		}

	}
}
