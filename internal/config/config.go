package config

import (
	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
)

type Config struct {
	Platform string
	Metrics  *metrics.ServerMetrics
	Users    domain.UserRepository
}

func NewConfig(
	platform string,
	metrics *metrics.ServerMetrics,
	users domain.UserRepository,
) *Config {
	return &Config{
		Platform: platform,
		Metrics:  metrics,
		Users:    users,
	}
}
