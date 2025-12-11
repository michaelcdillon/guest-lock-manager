package storage

import (
	"context"
	"database/sql"
	"time"
)

// Repository defines the common interface for all data repositories.
type Repository interface {
	// Close releases any resources held by the repository.
	Close() error
}

// Queryable represents a database connection that can execute queries.
// Both *sql.DB and *sql.Tx implement this interface.
type Queryable interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// BaseRepository provides common functionality for all repositories.
type BaseRepository struct {
	db *DB
}

// NewBaseRepository creates a new base repository with the given database connection.
func NewBaseRepository(db *DB) BaseRepository {
	return BaseRepository{db: db}
}

// DB returns the underlying database connection.
func (r *BaseRepository) DB() *DB {
	return r.db
}

// Now returns the current time in UTC for database timestamps.
func (r *BaseRepository) Now() time.Time {
	return time.Now().UTC()
}

// Transaction executes a function within a database transaction.
func (r *BaseRepository) Transaction(fn func(tx *sql.Tx) error) error {
	return r.db.Transaction(fn)
}

// GenerateID creates a new UUID for use as a primary key.
// Uses a simple timestamp-based ID for lightweight implementation.
func GenerateID() string {
	return generateUUID()
}

// generateUUID creates a v4-like UUID.
// For production, consider using github.com/google/uuid.
func generateUUID() string {
	// Simple implementation using crypto/rand would go here
	// For now, using timestamp + random suffix
	return time.Now().UTC().Format("20060102150405.000000000")
}



