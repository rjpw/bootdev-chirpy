package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/api"
	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store/postgres"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		platform = "production"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	cfg := config.NewConfig(platform,
		&metrics.ServerMetrics{},
		postgres.NewPostgresStore(database.New(db)),
	)

	srv := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           api.NewServer(cfg, "./root"),
		ReadHeaderTimeout: time.Millisecond * 30000,
	}

	if err := runUntilInterrupt(srv); err != nil {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}

func runUntilInterrupt(srv *http.Server) error {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed

	return nil
}
