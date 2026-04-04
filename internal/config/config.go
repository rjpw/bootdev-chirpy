package config

import (
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type Config struct {
	Platform string
	Metrics  *metrics.ServerMetrics
	Db       *database.Queries
	Users    store.UserStore
}

func NewConfig(platform string, metrics *metrics.ServerMetrics, db *database.Queries, users store.UserStore) *Config {
	return &Config{
		Platform: platform,
		Metrics:  metrics,
		Db:       db,
		Users:    users,
	}
}
