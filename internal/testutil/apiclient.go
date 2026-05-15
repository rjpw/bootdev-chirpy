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

// A stateful API client for use in longer testing workflows.
type InternalAPIClient struct {
	t           *testing.T
	Handler     http.Handler
	AccessToken string
	SessionID   string
	Email       string
}

func NewAPIClient(t *testing.T, handler http.Handler) *InternalAPIClient {
	return &InternalAPIClient{t: t, Handler: handler}
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
	handler http.Handler,
	method, path, authValue string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	r.Header.Add("Authorization", authValue)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func IssueRequest(
	handler http.Handler,
	method, path string,
	body io.Reader,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func Require(t *testing.T, i1, i2 int) {
	t.Helper()
	if i1 != i2 {
		t.Errorf("Expected response code %v and got %v", i2, i1)
	}
}

// Happy path Chirp assumes we're okay changing state
func (apiClient *InternalAPIClient) Chirp(body string) *httptest.ResponseRecorder {
	w := IssueAuthorizedRequest(apiClient.Handler, "POST", "/api/chirps",
		Bearer(apiClient.AccessToken),
		Marshal(apiClient.t, httpapi.ChirpParams{Body: body}))
	Require(apiClient.t, w.Code, http.StatusCreated)
	return w
}

// "Try" version of Chirp is provisional, used to validate failure modes
func (apiClient *InternalAPIClient) TryChirp(body string) *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(apiClient.Handler, "POST", "/api/chirps",
		Bearer(apiClient.AccessToken),
		Marshal(apiClient.t, httpapi.ChirpParams{Body: body}))
}

func (apiClient *InternalAPIClient) CreateUser(email, password string) *httptest.ResponseRecorder {
	w := IssueRequest(apiClient.Handler, "POST", "/api/users",
		Marshal(apiClient.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	Require(apiClient.t, w.Code, http.StatusCreated)
	apiClient.Email = email
	return w
}

func (apiClient *InternalAPIClient) TryCreateUser(
	email, password string,
) *httptest.ResponseRecorder {
	return IssueRequest(apiClient.Handler, "POST", "/api/users",
		Marshal(apiClient.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (apiClient *InternalAPIClient) Login(email, password string) *httptest.ResponseRecorder {
	w := IssueRequest(apiClient.Handler, "POST", "/api/login",
		Marshal(apiClient.t, httpapi.PostLoginRequest{Email: email, Password: password}))
	Require(apiClient.t, w.Code, http.StatusOK)
	resp := Decode[httpapi.PostLoginResponse](apiClient.t, w.Body.String())
	apiClient.AccessToken = resp.AccessToken
	apiClient.SessionID = resp.SessionID
	apiClient.Email = email
	return w
}

func (apiClient *InternalAPIClient) TryLogin(email, password string) *httptest.ResponseRecorder {
	return IssueRequest(apiClient.Handler, "POST", "/api/login",
		Marshal(apiClient.t, httpapi.PostLoginRequest{Email: email, Password: password}))
}

func (apiClient *InternalAPIClient) Refresh() *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(apiClient.Handler, "POST", "/api/refresh",
		Bearer(apiClient.SessionID), strings.NewReader(""))
}

func (apiClient *InternalAPIClient) Revoke() *httptest.ResponseRecorder {
	return IssueAuthorizedRequest(apiClient.Handler, "POST", "/api/revoke",
		Bearer(apiClient.SessionID), strings.NewReader(""))
}
