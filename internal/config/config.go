package config

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/memory"
	"github.com/rjpw/bootdev-chirpy/internal/operations"
	"github.com/rjpw/bootdev-chirpy/internal/postgres"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/schema"
)

// Server

var _ application.Runnable = (*Server)(nil)

type Server struct {
	httpServer *http.Server
	close      func() error
}

func (s *Server) Handler() http.Handler { return s.httpServer.Handler }

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.httpServer.Shutdown(context.Background())
	}()
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Close() error {
	if s.close != nil {
		return s.close()
	}
	return nil
}

func NewServer(env application.Environment, staticPath string) (*Server, error) {
	repo, db, err := postgres.NewRepositoryFromURL(env.DBURL)
	if err != nil {
		return nil, err
	}

	repos := &application.Repositories{
		Users:  repo,
		Chirps: nil,
	}
	metrics := &application.ServerMetrics{}
	handler := httpapi.NewServer(env.Platform, metrics, repos, staticPath)

	return &Server{
		httpServer: &http.Server{
			Addr:              "0.0.0.0:8080",
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
		},
		close: db.Close,
	}, nil
}

func NewTestServer() *Server {
	repos := &application.Repositories{
		Users: memory.NewMemoryRepository(),
	}
	metrics := &application.ServerMetrics{}
	handler := httpapi.NewServer("dev", metrics, repos, "./root")

	return &Server{
		httpServer: &http.Server{Handler: handler},
	}
}

// Migrator

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
