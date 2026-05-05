package operations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/rjpw/bootdev-chirpy/internal/application"
)

var _ application.Runnable = (*Migrator)(nil)

type Migrator struct {
	db      *sql.DB
	fsys    fs.FS
	command string
}

func NewMigrator(db *sql.DB, fsys fs.FS, command string) *Migrator {
	return &Migrator{db: db, fsys: fsys, command: command}
}

func (m *Migrator) Run(ctx context.Context) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, m.db, m.fsys)
	if err != nil {
		return err
	}

	switch m.command {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Printf("applied: %s (%s)\n", r.Source.Path, r.Duration)
		}
	case "status":
		results, err := provider.Status(ctx)
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Printf("%-5s %s\n", r.State, r.Source.Path)
		}
	default:
		return fmt.Errorf("unknown migrate command: %s", m.command)
	}
	return nil
}

func (m *Migrator) Close() error {
	return m.db.Close()
}
