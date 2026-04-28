package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/rjpw/bootdev-chirpy/internal/config"
	"github.com/rjpw/bootdev-chirpy/internal/httpapi"
	"github.com/rjpw/bootdev-chirpy/internal/metrics"
	"github.com/rjpw/bootdev-chirpy/internal/postgres"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/schema"
)

type AppEnvironment struct {
	DBName   string
	DBURL    string
	Platform string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(os.Args[2:])
		return
	}

	env, err := createEnvironment()
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	store, db, err := postgres.NewPostgresRepositoryFromURL(env.DBURL)
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
		Handler:           httpapi.NewServer(cfg, "./root"),
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

func runMigrate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: chirpy migrate [up|status]\n")
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stderr, "DB_URL is required\n")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationsFS, err := fs.Sub(schema.Migrations, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrations fs: %v\n", err)
		os.Exit(1)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose provider: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch args[0] {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Printf("applied: %s (%s)\n", r.Source.Path, r.Duration)
		}
	case "status":
		results, err := provider.Status(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate status: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Printf("%-5s %s\n", r.State, r.Source.Path)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown migrate command: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "usage: chirpy migrate [up|status]\n")
		os.Exit(1)
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
