package config

import (
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type Config struct {
	Platform string
	Metrics  *metrics.ServerMetrics
	Users    store.UserStore
}

func NewConfig(platform string, metrics *metrics.ServerMetrics, users store.UserStore) *Config {
	return &Config{
		Platform: platform,
		Metrics:  metrics,
		Users:    users,
	}
}
