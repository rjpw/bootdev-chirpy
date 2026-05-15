package config

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq" // normally seen in main, we use the config package provision Postgres connections
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/operations"
	"github.com/rjpw/bootdev-chirpy/internal/postgres"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/schema"
)

// ----------------  Server ----------------

var _ application.Runnable = (*Service)(nil)

type Service struct {
	httpServer *http.Server
	close      func() error
	SecretKey  string
}

func (service *Service) Handler() http.Handler { return service.httpServer.Handler }

func (service *Service) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		err := service.httpServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("Error shutting down: %s", err.Error())
		}
	}()
	err := service.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (service *Service) Close() error {
	if service.close != nil {
		return service.close()
	}
	return nil
}

func NewService(env application.Environment, staticPath string) (*Service, error) {
	repo, db, err := postgres.NewRepositoryFromURL(env.DBURL)
	if err != nil {
		return nil, err
	}

	repos := &application.Repositories{
		Users:        repo,
		UserSessions: repo,
		Chirps:       repo,
	}
	metrics := &application.ServerMetrics{}
	handler := httpapi.NewRouter(env, metrics, repos, staticPath)

	return &Service{
		httpServer: &http.Server{
			Addr:              "0.0.0.0:8080",
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
		},
		SecretKey: env.SecretKey,
		close:     db.Close,
	}, nil
}

// ----------------  Migrator ----------------

func NewMigrator(env application.Environment, command string) (*operations.Migrator, error) {
	db, err := postgres.Open(env.DBURL)
	if err != nil {
		return nil, err
	}

	migrationsFS, err := fs.Sub(schema.Migrations, "migrations")
	if err != nil {
		db.Close()
		return nil, err
	}

	return operations.NewMigrator(db, migrationsFS, command), nil
}
