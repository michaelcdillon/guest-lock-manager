package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations executes all pending database migrations.
// Migrations are SQL files in the migrations/ directory, named with a numeric prefix.
func RunMigrations(db *DB) error {
	// Create migrations tracking table
	if err := createMigrationsTable(db.DB); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(db.DB)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	// Get available migrations
	migrations, err := getMigrationFiles()
	if err != nil {
		return fmt.Errorf("reading migration files: %w", err)
	}

	// Apply pending migrations in order
	for _, m := range migrations {
		if applied[m.Name] {
			continue
		}

		log.Printf("Applying migration: %s", m.Name)
		if err := applyMigration(db.DB, m); err != nil {
			return fmt.Errorf("applying migration %s: %w", m.Name, err)
		}
		log.Printf("Migration applied: %s", m.Name)
	}

	return nil
}

type migration struct {
	Name    string
	Content string
}

func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			name TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT name FROM _migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

func getMigrationFiles() ([]migration, error) {
	var migrations []migration

	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		migrations = append(migrations, migration{
			Name:    filepath.Base(path),
			Content: string(content),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by filename (numeric prefix ensures correct order)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(m.Content); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO _migrations (name) VALUES (?)", m.Name); err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	return tx.Commit()
}



