// Package storage provides SQLite database connectivity and data access.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection with application-specific methods.
type DB struct {
	*sql.DB
	path string
}

// NewDB creates a new database connection to the SQLite file at the given path.
// It creates the directory structure if it doesn't exist.
func NewDB(path string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// Open database with appropriate settings:
	// - _foreign_keys=on: Enable foreign key constraints
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _busy_timeout=5000: Wait up to 5 seconds if database is locked
	// - _synchronous=NORMAL: Balance between safety and performance
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Configure connection pool - WAL mode allows concurrent reads
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	return &DB{DB: db, path: path}, nil
}

// Path returns the filesystem path to the database file.
func (db *DB) Path() string {
	return db.path
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
func (db *DB) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rolling back transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

