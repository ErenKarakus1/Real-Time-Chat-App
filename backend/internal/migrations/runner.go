package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct {
	db  *pgxpool.Pool
	dir string
}

func NewRunner(db *pgxpool.Pool, dir string) *Runner {
	return &Runner{
		db:  db,
		dir: dir,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.ensureSchemaMigrations(ctx); err != nil {
		return err
	}

	files, err := migrationFiles(r.dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		name := filepath.Base(file)
		applied, err := r.isApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := r.apply(ctx, name, file); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) ensureSchemaMigrations(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *Runner) isApplied(ctx context.Context, name string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE name = $1
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, name).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *Runner) apply(ctx context.Context, name string, file string) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func migrationFiles(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}
