package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// CalendarRepository provides data access for calendar subscriptions.
type CalendarRepository struct {
	BaseRepository
}

// NewCalendarRepository creates a new calendar repository.
func NewCalendarRepository(db *DB) *CalendarRepository {
	return &CalendarRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new calendar subscription.
func (r *CalendarRepository) Create(ctx context.Context, cal *models.CalendarSubscription) error {
	cal.ID = GenerateID()
	cal.CreatedAt = r.Now()
	cal.UpdatedAt = r.Now()
	cal.SyncStatus = models.SyncStatusPending

	_, err := r.DB().ExecContext(ctx, `
		INSERT INTO calendar_subscriptions (
			id, name, url, sync_interval_min, sync_status, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		cal.ID, cal.Name, cal.URL, cal.SyncIntervalMin,
		cal.SyncStatus, cal.Enabled, cal.CreatedAt, cal.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("inserting calendar: %w", err)
	}

	return nil
}

// GetByID retrieves a calendar by its ID.
func (r *CalendarRepository) GetByID(ctx context.Context, id string) (*models.CalendarSubscription, error) {
	cal := &models.CalendarSubscription{}

	err := r.DB().QueryRowContext(ctx, `
		SELECT id, name, url, sync_interval_min, last_sync_at, sync_status, 
		       sync_error, enabled, created_at, updated_at
		FROM calendar_subscriptions WHERE id = ?
	`, id).Scan(
		&cal.ID, &cal.Name, &cal.URL, &cal.SyncIntervalMin,
		&cal.LastSyncAt, &cal.SyncStatus, &cal.SyncError,
		&cal.Enabled, &cal.CreatedAt, &cal.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying calendar: %w", err)
	}

	return cal, nil
}

// List retrieves all calendar subscriptions.
func (r *CalendarRepository) List(ctx context.Context) ([]models.CalendarSubscription, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, name, url, sync_interval_min, last_sync_at, sync_status,
		       sync_error, enabled, created_at, updated_at
		FROM calendar_subscriptions
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("querying calendars: %w", err)
	}
	defer rows.Close()

	var calendars []models.CalendarSubscription
	for rows.Next() {
		var cal models.CalendarSubscription
		if err := rows.Scan(
			&cal.ID, &cal.Name, &cal.URL, &cal.SyncIntervalMin,
			&cal.LastSyncAt, &cal.SyncStatus, &cal.SyncError,
			&cal.Enabled, &cal.CreatedAt, &cal.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning calendar: %w", err)
		}
		calendars = append(calendars, cal)
	}

	return calendars, rows.Err()
}

// ListEnabled retrieves all enabled calendars that need syncing.
func (r *CalendarRepository) ListEnabled(ctx context.Context) ([]models.CalendarSubscription, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, name, url, sync_interval_min, last_sync_at, sync_status,
		       sync_error, enabled, created_at, updated_at
		FROM calendar_subscriptions
		WHERE enabled = 1
		ORDER BY last_sync_at ASC NULLS FIRST
	`)
	if err != nil {
		return nil, fmt.Errorf("querying enabled calendars: %w", err)
	}
	defer rows.Close()

	var calendars []models.CalendarSubscription
	for rows.Next() {
		var cal models.CalendarSubscription
		if err := rows.Scan(
			&cal.ID, &cal.Name, &cal.URL, &cal.SyncIntervalMin,
			&cal.LastSyncAt, &cal.SyncStatus, &cal.SyncError,
			&cal.Enabled, &cal.CreatedAt, &cal.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning calendar: %w", err)
		}
		calendars = append(calendars, cal)
	}

	return calendars, rows.Err()
}

// Update updates an existing calendar.
func (r *CalendarRepository) Update(ctx context.Context, cal *models.CalendarSubscription) error {
	cal.UpdatedAt = r.Now()

	result, err := r.DB().ExecContext(ctx, `
		UPDATE calendar_subscriptions SET
			name = ?, url = ?, sync_interval_min = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`,
		cal.Name, cal.URL, cal.SyncIntervalMin, cal.Enabled, cal.UpdatedAt, cal.ID,
	)

	if err != nil {
		return fmt.Errorf("updating calendar: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("calendar not found: %s", cal.ID)
	}

	return nil
}

// UpdateSyncStatus updates the sync status of a calendar.
func (r *CalendarRepository) UpdateSyncStatus(ctx context.Context, id string, status string, syncError *string) error {
	now := time.Now().UTC()
	var lastSyncAt *time.Time
	if status == models.SyncStatusSuccess {
		lastSyncAt = &now
	}

	_, err := r.DB().ExecContext(ctx, `
		UPDATE calendar_subscriptions SET
			sync_status = ?, sync_error = ?, last_sync_at = COALESCE(?, last_sync_at), updated_at = ?
		WHERE id = ?
	`, status, syncError, lastSyncAt, now, id)

	if err != nil {
		return fmt.Errorf("updating sync status: %w", err)
	}

	return nil
}

// Delete removes a calendar by ID.
func (r *CalendarRepository) Delete(ctx context.Context, id string) error {
	result, err := r.DB().ExecContext(ctx, "DELETE FROM calendar_subscriptions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting calendar: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("calendar not found: %s", id)
	}

	return nil
}

// GetLockIDs retrieves all lock IDs assigned to a calendar.
func (r *CalendarRepository) GetLockIDs(ctx context.Context, calendarID string) ([]string, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT lock_id FROM calendar_lock_mappings WHERE calendar_id = ?
	`, calendarID)
	if err != nil {
		return nil, fmt.Errorf("querying lock mappings: %w", err)
	}
	defer rows.Close()

	var lockIDs []string
	for rows.Next() {
		var lockID string
		if err := rows.Scan(&lockID); err != nil {
			return nil, fmt.Errorf("scanning lock ID: %w", err)
		}
		lockIDs = append(lockIDs, lockID)
	}

	return lockIDs, rows.Err()
}

// SetLockIDs replaces all lock assignments for a calendar.
func (r *CalendarRepository) SetLockIDs(ctx context.Context, calendarID string, lockIDs []string) error {
	return r.Transaction(func(tx *sql.Tx) error {
		// Delete existing mappings
		_, err := tx.ExecContext(ctx, "DELETE FROM calendar_lock_mappings WHERE calendar_id = ?", calendarID)
		if err != nil {
			return fmt.Errorf("deleting lock mappings: %w", err)
		}

		// Insert new mappings
		for _, lockID := range lockIDs {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO calendar_lock_mappings (calendar_id, lock_id) VALUES (?, ?)
			`, calendarID, lockID)
			if err != nil {
				return fmt.Errorf("inserting lock mapping: %w", err)
			}
		}

		return nil
	})
}



