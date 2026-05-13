package httpapi_test

// This file contains tests for the API server's user-related endpoints.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
)

type localTestCache struct {
	tokenCache map[string]string
	userCache  map[string]httpapi.PostLoginResponse
}

var testCache localTestCache

func initializeUsers(t *testing.T, srv *httpapi.Server) {
	t.Helper()

	// make the empty structures for workflows to be run later
	testCache = localTestCache{
		tokenCache: make(map[string]string),
		userCache:  make(map[string]httpapi.PostLoginResponse),
	}

	users := []struct {
		name  string
		email string
	}{
		{
			"create user saul",
			"saul@bettercall.com",
		},
		{
			"create user mike",
			"mike@bettercall.com",
		},
	}

	// reset the server first (redundant, but hey, we're initializing!)
	issueRequest(srv, http.MethodPost, "/admin/reset", strings.NewReader(""))

	for _, tc := range users {
		issueRequest(srv, http.MethodPost, "/api/users", getFileReader(t, fmt.Sprintf("UserParams_%s.json", tc.email)))
	}

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
		payload := marshalEntity(t, httpapi.PostLoginRequest{Email: tc.email, Password: tc.password})
		w := issueRequest(srv, tc.method, tc.path, strings.NewReader(payload))

		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Code != tc.responseCode {
			t.Errorf("Expected response code %d, got %d", tc.responseCode, w.Code)
		}

		if tc.path == "/api/users" {
			user, _ := decodeEntity[domain.User](t, w.Body.String())
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

		payload := marshalEntity(t, httpapi.PostLoginRequest{Email: tc.email, Password: tc.password})
		w := issueRequest(srv, tc.method, tc.path, strings.NewReader(payload))

		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Code != tc.responseCode {
			t.Errorf("Expected response code %d, got %d", tc.responseCode, w.Code)
		}

		if tc.path == "/api/users" {
			user, _ := decodeEntity[domain.User](t, w.Body.String())
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

		payload := marshalEntity(t, httpapi.PostLoginRequest{Email: tc.email, Password: tc.password})
		w := issueRequest(srv, tc.method, tc.path, strings.NewReader(payload))

		if got := w.Header().Get("Content-Type"); got != tc.contentType {
			t.Errorf("%s %s: want Content-Type %q, got %q", tc.method, tc.path, tc.contentType, got)
		}
		if w.Code != tc.responseCode {
			t.Errorf("Expected response code %d, got %d", tc.responseCode, w.Code)
		}

	}
}

func TestMITMTokenTheftScenario(t *testing.T) {
	srv := newTestServer()

	initializeUsers(t, srv)

	cases := []struct {
		name          string
		path          string
		email         string
		mitmAttempted bool
		responseCode  int
	}{
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

	for _, tc := range cases {

		var r *http.Request

		switch tc.path {
		case "/api/users":
			fallthrough
		case "/api/login":
			r = httptest.NewRequest(http.MethodPost, tc.path, getFileReader(t, fmt.Sprintf("UserParams_%s.json", tc.email)))
		case "/api/chirps":
			r = httptest.NewRequest(http.MethodPost, tc.path, getFileReader(t, fmt.Sprintf("ChirpParams_%s.json", tc.email)))
			r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testCache.tokenCache[tc.email]))
		}

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)

		// keep tokens for later MITM attempt by Saul
		if tc.path == "/api/login" && w.Code == http.StatusOK {
			user, _ := decodeEntity[httpapi.PostLoginResponse](t, w.Body.String())
			fmt.Printf("Retaining user %v for later use ...\n", user)
			testCache.tokenCache[tc.email] = user.AccessToken
			testCache.userCache[tc.email] = user
		}

		// expect rejection on bad password
		if w.Code != tc.responseCode {
			fmt.Printf("\n\nResponse Body: %s\n\n", w.Body.String())
			t.Errorf("%s -- %s -- Expected response code %d, got %d", tc.name, tc.path, tc.responseCode, w.Code)
		}

	}
}

func TestTokenRefreshScenarios(t *testing.T) {
	srv := newTestServer()

	initializeUsers(t, srv)

	cases := []struct {
		name         string
		path         string
		email        string
		responseCode int
	}{
		{
			"user logs in",
			"/api/login",
			"saul@bettercall.com",
			http.StatusOK,
		},
		{
			"user chirps with initial token",
			"/api/chirps",
			"saul@bettercall.com",
			http.StatusCreated,
		},
		{
			"user requests new token",
			"/api/refresh",
			"mike@bettercall.com",
			http.StatusOK,
		},
		{
			"user chirps with new token",
			"/api/chirps",
			"saul@bettercall.com",
			http.StatusCreated,
		},
		{
			"user revokes refresh token",
			"/api/revoke",
			"mike@bettercall.com",
			http.StatusOK,
		},
		{
			"revoked token refresh",
			"/api/refresh",
			"mike@bettercall.com",
			http.StatusNotFound,
		},
	}

	cachedRefreshToken := ""

	for _, tc := range cases {

		switch tc.path {
		case "/api/login":

			w := issueRequest(srv, http.MethodPost, tc.path, getFileReader(t, fmt.Sprintf("UserParams_%s.json", tc.email)))

			if w.Code == http.StatusOK {
				user, _ := decodeEntity[httpapi.PostLoginResponse](t, w.Body.String())
				fmt.Printf("Retaining user %v for later use ...\n", user)
				testCache.tokenCache[tc.email] = user.AccessToken
				cachedRefreshToken = user.SessionID
			} else {
				t.Errorf("%s -- Error logging in: %s", tc.name, w.Body.String())
			}
		case "/api/refresh":
			if len(cachedRefreshToken) == 0 {
				t.Errorf("%s -- Error retrieving refresh token: %s", tc.name, "No cached refresh token to use")
			} else {
				fmt.Printf("Using refresh token: %s ...\n", cachedRefreshToken[:8])
			}

			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path,
				fmt.Sprintf("Bearer %s", cachedRefreshToken),
				strings.NewReader(""))

			if w.Code == http.StatusOK {

				rt, err := decodeEntity[httpapi.SessionRefreshResponse](t, w.Body.String())
				if err != nil {
					t.Errorf("%s -- Error getting refresh token from header: %v", tc.name, err)
				} else {
					cachedRefreshToken = rt.AccessToken
				}
			}

		case "/api/revoke":
			if len(cachedRefreshToken) == 0 {
				t.Errorf("%s -- Error retrieving refresh token: %s", tc.name, "No cached refresh token to use")
			} else {
				fmt.Printf("Using refresh token: %s ...\n", cachedRefreshToken[:8])
			}
			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path, fmt.Sprintf("Bearer %s", cachedRefreshToken), strings.NewReader(""))

			if w.Code == http.StatusOK {

				rt, err := decodeEntity[httpapi.SessionRefreshResponse](t, w.Body.String())
				if err != nil {
					t.Errorf("%s -- Error getting refresh token from header: %v", tc.name, err)
				} else {
					cachedRefreshToken = rt.AccessToken
				}
			}

		case "/api/chirps":
			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path,
				fmt.Sprintf("Bearer %s", testCache.tokenCache[tc.email]),
				getFileReader(t, fmt.Sprintf("ChirpParams_%s.json", tc.email)))

			// expect rejection on bad password (for example)
			if w.Code != tc.responseCode {
				fmt.Printf("\n\nResponse Body: %s\n\n", w.Body.String())
				t.Errorf("%s -- %s -- Expected response code %d, got %d", tc.name, tc.path, tc.responseCode, w.Code)
			}
		}

	}
}

func TestAuthorizationScenarios(t *testing.T) {
	srv := newTestServer()

	initializeUsers(t, srv)

	cases := []struct {
		name         string
		method       string
		path         string
		email        string
		responseCode int
	}{
		{
			"saul logs in",
			"POST",
			"/api/login",
			"saul@bettercall.com",
			http.StatusOK,
		},
		{
			"mike logs in",
			"POST",
			"/api/login",
			"saul@bettercall.com",
			http.StatusOK,
		},
		{
			"saul updates his password",
			"PUT",
			"/api/users",
			"saul@bettercall.com",
			http.StatusCreated,
		},
	}

	cachedRefreshToken := ""

	for _, tc := range cases {

		switch tc.path {
		case "/api/login":
			w := issueRequest(srv, http.MethodPost, tc.path,
				getFileReader(t, fmt.Sprintf("UserParams_%s.json", tc.email)))

			if w.Code == http.StatusOK {
				user, _ := decodeEntity[httpapi.PostLoginResponse](t, w.Body.String())
				fmt.Printf("Retaining user %v for later use ...\n", user)
				testCache.tokenCache[tc.email] = user.AccessToken
				cachedRefreshToken = user.SessionID
			} else {
				t.Errorf("%s -- Error logging in: %s", tc.name, w.Body.String())
			}
		case "/api/refresh":
			if len(cachedRefreshToken) == 0 {
				t.Errorf("%s -- Error retrieving refresh token: %s", tc.name, "No cached refresh token to use")
			} else {
				fmt.Printf("Using refresh token: %s ...\n", cachedRefreshToken[:8])
			}

			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path,
				fmt.Sprintf("Bearer %s", cachedRefreshToken),
				strings.NewReader(""))

			if w.Code == http.StatusOK {

				rt, err := decodeEntity[httpapi.SessionRefreshResponse](t, w.Body.String())
				if err != nil {
					t.Errorf("%s -- Error getting refresh token from header: %v", tc.name, err)
				} else {
					cachedRefreshToken = rt.AccessToken
				}
			}

		case "/api/revoke":
			if len(cachedRefreshToken) == 0 {
				t.Errorf("%s -- Error retrieving refresh token: %s", tc.name, "No cached refresh token to use")
			} else {
				fmt.Printf("Using refresh token: %s ...\n", cachedRefreshToken[:8])
			}

			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path,
				fmt.Sprintf("Bearer %s", cachedRefreshToken),
				strings.NewReader(""))

			if w.Code == http.StatusOK {

				rt, err := decodeEntity[httpapi.SessionRefreshResponse](t, w.Body.String())
				if err != nil {
					t.Errorf("%s -- Error getting refresh token from header: %v", tc.name, err)
				} else {
					cachedRefreshToken = rt.AccessToken
				}
			}

		case "/api/chirps":

			w := issueAuthorizedRequest(srv, http.MethodPost, tc.path,
				fmt.Sprintf("Bearer %s", testCache.tokenCache[tc.email]),
				getFileReader(t, fmt.Sprintf("ChirpParams_%s.json", tc.email)))

			// expect rejection on bad password (for example)
			if w.Code != tc.responseCode {
				fmt.Printf("\n\nResponse Body: %s\n\n", w.Body.String())
				t.Errorf("%s -- %s -- Expected response code %d, got %d", tc.name, tc.path, tc.responseCode, w.Code)
			}
		}

	}
}
