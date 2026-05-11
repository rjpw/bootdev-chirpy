package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/config"
)

func main() {
	env := application.LoadEnvironment()

	var runnable application.Runnable

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: chirpy migrate [up|status]\n")
			os.Exit(1)
		}
		m, err := config.NewMigrator(env, os.Args[2])
		if err != nil {
			log.Fatalf("Failed to create migrator: %v", err)
		}
		runnable = m
	} else {
		srv, err := config.NewServer(env, "./root")
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		runnable = srv
	}
	defer runnable.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := runnable.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
