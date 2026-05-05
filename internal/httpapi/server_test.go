package httpapi_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/memory"
)

func newTestServer(platform string) *httpapi.Server {
	repositories := application.Repositories{
		Users: memory.NewMemoryRepository(),
	}
	return httpapi.NewServer(platform, &application.ServerMetrics{}, &repositories, "./testdata")
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
