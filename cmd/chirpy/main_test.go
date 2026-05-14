//go:build integration

package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/testdb"
	"github.com/rjpw/bootdev-chirpy/internal/testutil"
)

var service *config.Service

func TestMain(m *testing.M) {
	url, cleanup, err := testdb.SetupURL()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	env := application.Environment{
		DBURL:     url,
		Platform:  "dev",
		SecretKey: "test-secret-key",
	}

	service, err = config.NewService(env, "../../root")
	if err != nil {
		panic(err)
	}
	defer service.Close()

	os.Exit(m.Run())
}

func issueRequest(method, path string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	service.Handler().ServeHTTP(w, r)
	return w
}

func TestHealthz(t *testing.T) {
	w := issueRequest("GET", "/api/healthz", "")
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHappyPath(t *testing.T) {

	saulsPassword := "honour among thieves"

	saul := testutil.NewAPIClient(t, service.Handler())

	// happy path - create an account
	saul.CreateUser("saul@bettercall.com", saulsPassword)

	// try duplicating with the same email -- should fail with status conflict
	testutil.AssertStatus(t, "Create account for Saul",
		saul.TryCreateUser("saul@bettercall.com", "some other password"),
		http.StatusConflict)

	// try chirping before login -- should fail
	testutil.AssertStatus(t, "Premature chirping", saul.TryChirp("Better call Saul!"), http.StatusUnauthorized)

	// so log in ... incorrectly the first time ...
	testutil.AssertStatus(
		t,
		"Login with bad password",
		saul.TryLogin("saul@bettercall.com", "was it 'password'?"),
		http.StatusUnauthorized,
	)

	// ... remember the password ...
	testutil.AssertStatus(
		t,
		"Login with good password",
		saul.TryLogin("saul@bettercall.com", saulsPassword),
		http.StatusOK,
	)

	// ... actually log in this time -- changes our client state, required for chirp
	saul.Login("saul@bettercall.com", saulsPassword)

	// ... chirping after login should succeed
	testutil.AssertStatus(t, "Good times", saul.Chirp("Better call Saul!"), http.StatusCreated)
}

// This is an aspirational test for the moment
func TestMITMTokenTheft(t *testing.T) {
	t.Skip(
		"Bearer tokens are proof-of-possession — server cannot detect theft without DPoP or similar",
	)

	saul := testutil.NewAPIClient(t, service.Handler())
	mike := testutil.NewAPIClient(t, service.Handler())

	saul.CreateUser("saul@bettercall.com", "password")
	mike.CreateUser("mike@bettercall.com", "password")

	saul.Login("saul@bettercall.com", "password")
	mike.Login("mike@bettercall.com", "password")

	// Each user chirps with their own token
	testutil.AssertStatus(t, "Chirp from Saul", saul.Chirp("Better call Saul!"), http.StatusCreated)
	testutil.AssertStatus(t, "Chirp from Mike", mike.Chirp("No half measures"), http.StatusCreated)

	// Saul steals Mike's token
	saul.AccessToken = mike.AccessToken
	testutil.AssertStatus(t, "Saul goes rogue", saul.Chirp("I am Mike now"), http.StatusForbidden)
}
