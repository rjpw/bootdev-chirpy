package config

import (
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type Config struct {
	Metrics *metrics.ServerMetrics
	Db      *database.Queries
	Users   store.UserStore
}
