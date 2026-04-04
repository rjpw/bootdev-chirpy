package api

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store/memory"
)

func newTestServer() *Server {
	cfg := &config.Config{
		Metrics: &metrics.ServerMetrics{},
		Users:   memory.NewMemoryStore(),
	}
	return NewServer(cfg, "./testdata")
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
