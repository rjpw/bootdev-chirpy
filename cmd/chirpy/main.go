package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/api"
	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/store/postgres"
)

type AppEnvironment struct {
	DBName   string
	DBURL    string
	Platform string
}

func main() {
	env, err := createEnvironment()
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	store, db, err := postgres.NewPostgresStoreFromURL(env.DBURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	cfg := config.NewConfig(
		env.Platform,
		&metrics.ServerMetrics{},
		store,
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

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func createEnvironment() (*AppEnvironment, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
		return nil, err
	}
	return &AppEnvironment{
		DBName:   os.Getenv("DBNAME"),
		DBURL:    os.Getenv("DB_URL"),
		Platform: os.Getenv("PLATFORM"),
	}, nil
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
