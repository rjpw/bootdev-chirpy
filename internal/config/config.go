package config

import "github.com/rjpw/bootdev-chirpy/internal/metrics"

type Config struct {
	Metrics *metrics.ServerMetrics
}
