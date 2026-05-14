package testutil

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
type TestClient struct {
	t           *testing.T
	srv         *httpapi.ChirpyAPIRouter
	Handler     http.Handler
	AccessToken string
	SessionID   string
	Email       string
}

func NewTestClient(t *testing.T, handler http.Handler) *TestClient {
	return &TestClient{t: t, Handler: handler}
}

func Bearer(s string) string {
	return fmt.Sprintf("Bearer %s", s)
}

func AssertStatus(t *testing.T, testContext string, reply *httptest.ResponseRecorder, i int) {
	t.Helper()
	if reply.Code != i {
		t.Errorf("%s -- Expected response code: %v and got %v", testContext, i, reply.Code)
	}
}

func Decode[T any](t *testing.T, rawData string) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(strings.NewReader(rawData)).Decode(&v); err != nil {
		t.Errorf("Error decoding entity %q", err)
	}
	return v
}

func Marshal[T any](t *testing.T, entity T) io.Reader {
	t.Helper()

	b, err := json.Marshal(entity)
	if err != nil {
		t.Errorf("Error creating entity: %v", err)
	}
	return bytes.NewReader(b)
}

func IssueAuthorizedRequest(
	srv *httpapi.ChirpyAPIRouter,
	method, path, authValue string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	r.Header.Add("Authorization", authValue)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func IssueRequest(
	srv *httpapi.ChirpyAPIRouter,
	method, path string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func Require(t *testing.T, i1, i2 int) {
	t.Helper()
	if i1 != i2 {
		t.Errorf("Expected response code %v and got %v", i2, i1)
	}
}

// Happy path Chirp assumes we're okay changing state
func (c *TestClient) Chirp(body string) *httptest.ResponseRecorder {
	w := IssueAuthorizedRequest(c.srv, "POST", "/api/chirps",
		Bearer(c.AccessToken),
		Marshal(c.t, httpapi.ChirpParams{Body: body}))
	Require(c.t, w.Code, http.StatusCreated)
	return w
}

// "Try" version of Chirp is provisional, used to validate failure modes
func (c *TestClient) TryChirp(body string) *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(c.srv, "POST", "/api/chirps",
		Bearer(c.AccessToken),
		Marshal(c.t, httpapi.ChirpParams{Body: body}))
}

func (c *TestClient) CreateUser(email, password string) *httptest.ResponseRecorder {
	w := IssueRequest(c.srv, "POST", "/api/users",
		Marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	Require(c.t, w.Code, http.StatusCreated)
	c.Email = email
	return w
}

func (c *TestClient) TryCreateUser(email, password string) *httptest.ResponseRecorder {
	return IssueRequest(c.srv, "POST", "/api/users",
		Marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (c *TestClient) Login(email, password string) *httptest.ResponseRecorder {
	w := IssueRequest(c.srv, "POST", "/api/login",
		Marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	Require(c.t, w.Code, http.StatusOK)
	resp := Decode[httpapi.PostLoginResponse](c.t, w.Body.String())
	c.AccessToken = resp.AccessToken
	c.SessionID = resp.SessionID
	c.Email = email
	return w
}

func (c *TestClient) TryLogin(email, password string) *httptest.ResponseRecorder {
	return IssueRequest(c.srv, "POST", "/api/login",
		Marshal(c.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (c *TestClient) Refresh() *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(c.srv, "POST", "/api/refresh",
		Bearer(c.SessionID), strings.NewReader(""))
}

func (c *TestClient) Revoke() *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(c.srv, "POST", "/api/revoke",
		Bearer(c.SessionID), strings.NewReader(""))
}
