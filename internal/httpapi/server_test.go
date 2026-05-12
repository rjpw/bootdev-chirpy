package httpapi_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/memory"
)

// Define and initialize testdataFS
//
//go:embed testdata/*
var testdataFS embed.FS

func newTestServer() *httpapi.Server {
	// note: this loads internal/httpapi/testdata/.env
	env := application.LoadEnvironment()

	repo := memory.NewMemoryRepository()
	repositories := application.Repositories{
		Users:        repo,
		UserSessions: repo,
		Chirps:       repo,
	}
	return httpapi.NewServer(
		env,
		&application.ServerMetrics{},
		&repositories,
		"./testdata")
}

func parseHitCount(t *testing.T, body string) int {
	t.Helper()
	re := regexp.MustCompile(`visited (\d+) times!`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatalf("no metric parsable from body: %q", body)
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil {
		t.Fatalf("could not parse hit count: %v", err)
	}
	return count
}

func decodeEntity[T any](t *testing.T, rawData string) (T, error) {
	t.Helper()
	var v T
	if err := json.NewDecoder(strings.NewReader(rawData)).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

func getFileReader(t *testing.T, filename string) fs.File {
	t.Helper()

	file, err := testdataFS.Open(fmt.Sprintf("testdata/%s", filename))
	if err != nil {
		t.Fatalf("testdataFS.Open(%q) error: %v", filename, err)
	}
	return file

}
