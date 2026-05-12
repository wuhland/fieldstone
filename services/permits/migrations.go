package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations executes all *.sql files from the embedded FS in lexical order.
// SQL files use IF NOT EXISTS so the operation is idempotent.
// TODO(fieldstone): replace with golang-migrate for proper versioned migrations.
func runMigrations(ctx context.Context, pool *pgxpool.Pool, files embed.FS) error {
	entries, err := fs.ReadDir(files, "db/migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		data, err := files.ReadFile("db/migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		slog.Info("migration applied", "file", name)
	}
	return nil
}
