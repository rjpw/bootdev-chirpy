package httpapi_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/memory"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
)

func newTestServer(platform string) *httpapi.Server {
	cfg := &config.Config{
		Platform: platform,
		Metrics:  &metrics.ServerMetrics{},
		Users:    memory.NewMemoryRepository(),
	}
	return httpapi.NewServer(cfg, "./testdata")
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
