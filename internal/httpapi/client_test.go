package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
)

// A stateful client for use in longer workflows.
type testClient struct {
	t           *testing.T
	srv         *httpapi.Server
	accessToken string
	sessionID   string
	email       string
}

func newTestClient(t *testing.T, srv *httpapi.Server) *testClient {
	return &testClient{t: t, srv: srv}
}

func bearer(s string) string {
	return fmt.Sprintf("Bearer %s", s)
}

func assertStatus(t *testing.T, testContext string, reply *httptest.ResponseRecorder, i int) {
	t.Helper()
	if reply.Code != i {
		t.Errorf("%s -- Expected response code: %v and got %v", testContext, i, reply.Code)
	}
}

func decode[T any](t *testing.T, rawData string) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(strings.NewReader(rawData)).Decode(&v); err != nil {
		t.Errorf("Error decoding entity %q", err)
	}
	return v
}

func marshal[T any](t *testing.T, entity T) io.Reader {
	t.Helper()

	b, err := json.Marshal(entity)
	if err != nil {
		t.Errorf("Error creating entity: %v", err)
	}
	return bytes.NewReader(b)
}

func require(t *testing.T, i1, i2 int) {
	t.Helper()
	if i1 != i2 {
		t.Errorf("Expected response code %v and got %v", i2, i1)
	}
}

// Happy path Chirp assumes we're okay changing state
func (c *testClient) Chirp(body string) *httptest.ResponseRecorder {
	w := issueAuthorizedRequest(c.srv, "POST", "/api/chirps",
		bearer(c.accessToken),
		marshal(c.t, httpapi.ChirpParams{Body: body}))
	require(c.t, w.Code, http.StatusCreated)
	return w
}

// "Try" version of Chirp is provisional, used to validate failure modes
func (c *testClient) TryChirp(body string) *httptest.ResponseRecorder {
	return issueAuthorizedRequest(c.srv, "POST", "/api/chirps",
		bearer(c.accessToken),
		marshal(c.t, httpapi.ChirpParams{Body: body}))
}

func (c *testClient) CreateUser(email, password string) *httptest.ResponseRecorder {
	w := issueRequest(c.srv, "POST", "/api/users",
		marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	require(c.t, w.Code, http.StatusCreated)
	c.email = email
	return w
}

func (c *testClient) TryCreateUser(email, password string) *httptest.ResponseRecorder {
	return issueRequest(c.srv, "POST", "/api/users",
		marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (c *testClient) Login(email, password string) *httptest.ResponseRecorder {
	w := issueRequest(c.srv, "POST", "/api/login",
		marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	require(c.t, w.Code, http.StatusOK)
	resp := decode[httpapi.PostLoginResponse](c.t, w.Body.String())
	c.accessToken = resp.AccessToken
	c.sessionID = resp.SessionID
	c.email = email
	return w
}

func (c *testClient) TryLogin(email, password string) *httptest.ResponseRecorder {
	return issueRequest(c.srv, "POST", "/api/login",
		marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (c *testClient) Refresh() *httptest.ResponseRecorder {
	return issueAuthorizedRequest(c.srv, "POST", "/api/refresh",
		bearer(c.sessionID), strings.NewReader(""))
}

func (c *testClient) Revoke() *httptest.ResponseRecorder {
	return issueAuthorizedRequest(c.srv, "POST", "/api/revoke",
		bearer(c.sessionID), strings.NewReader(""))
}

func TestHappyPath(t *testing.T) {
	srv := newTestServer()

	saulsPassword := "honour among thieves"

	saul := newTestClient(t, srv)

	// happy path - create an account
	saul.CreateUser("saul@bettercall.com", saulsPassword)

	// try duplicating with the same email -- should fail with status conflict
	assertStatus(t, "Create account for Saul",
		saul.TryCreateUser("saul@bettercall.com", "some other password"),
		http.StatusConflict)

	// try chirping before login -- should fail
	assertStatus(t, "Premature chirping", saul.TryChirp("Better call Saul!"), http.StatusUnauthorized)

	// so log in ... incorrectly the first time ...
	assertStatus(
		t,
		"Login with bad password",
		saul.TryLogin("saul@bettercall.com", "was it 'password'?"),
		http.StatusUnauthorized,
	)

	// ... remember the password ...
	assertStatus(
		t,
		"Login with good password",
		saul.TryLogin("saul@bettercall.com", saulsPassword),
		http.StatusOK,
	)

	// ... actually log in this time -- changes our client state, required for chirp
	saul.Login("saul@bettercall.com", saulsPassword)

	// ... chirping after login should succeed
	assertStatus(t, "Good times", saul.Chirp("Better call Saul!"), http.StatusCreated)
}

// This is an aspirational test for the moment
func TestMITMTokenTheft(t *testing.T) {
	t.Skip(
		"Bearer tokens are proof-of-possession — server cannot detect theft without DPoP or similar",
	)

	srv := newTestServer()
	saul := newTestClient(t, srv)
	mike := newTestClient(t, srv)

	saul.CreateUser("saul@bettercall.com", "password")
	mike.CreateUser("mike@bettercall.com", "password")

	saul.Login("saul@bettercall.com", "password")
	mike.Login("mike@bettercall.com", "password")

	// Each user chirps with their own token
	assertStatus(t, "Chirp from Saul", saul.Chirp("Better call Saul!"), http.StatusCreated)
	assertStatus(t, "Chirp from Mike", mike.Chirp("No half measures"), http.StatusCreated)

	// Saul steals Mike's token
	saul.accessToken = mike.accessToken
	assertStatus(t, "Saul goes rogue", saul.Chirp("I am Mike now"), http.StatusForbidden)
}
