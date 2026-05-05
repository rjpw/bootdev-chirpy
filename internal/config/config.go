package config

import (
	"github.com/rjpw/bootdev-chirpy/internal/application"
)

type Config struct {
	Platform     string
	Metrics      *application.ServerMetrics
	Repositories *application.Repositories
}

func NewConfig(
	platform string,
	metrics *application.ServerMetrics,
	repositories *application.Repositories,
) *Config {
	return &Config{
		Platform:     platform,
		Metrics:      metrics,
		Repositories: repositories,
	}
}
