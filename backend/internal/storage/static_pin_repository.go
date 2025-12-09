package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// StaticPINRepository handles database operations for static PINs.
type StaticPINRepository struct {
	db *DB
}

// NewStaticPINRepository creates a new static PIN repository.
func NewStaticPINRepository(db *DB) *StaticPINRepository {
	return &StaticPINRepository{db: db}
}

// Create creates a new static PIN.
func (r *StaticPINRepository) Create(ctx context.Context, pin *models.StaticPIN) error {
	if pin.ID == "" {
		pin.ID = GenerateID()
	}
	pin.CreatedAt = time.Now().UTC()
	pin.UpdatedAt = pin.CreatedAt

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO static_pins (id, name, pin_code, enabled, always_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, pin.ID, pin.Name, pin.PINCode, pin.Enabled, pin.AlwaysActive, pin.CreatedAt, pin.UpdatedAt)

	return err
}

// GetByID retrieves a static PIN by ID with its schedules.
func (r *StaticPINRepository) GetByID(ctx context.Context, id string) (*models.StaticPINWithSchedules, error) {
	var pin models.StaticPINWithSchedules
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, pin_code, enabled, always_active, created_at, updated_at
		FROM static_pins WHERE id = ?
	`, id).Scan(&pin.ID, &pin.Name, &pin.PINCode, &pin.Enabled, &pin.AlwaysActive,
		&pin.CreatedAt, &pin.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load schedules
	schedules, err := r.GetSchedules(ctx, id)
	if err != nil {
		return nil, err
	}
	pin.Schedules = schedules

	return &pin, nil
}

// List retrieves all static PINs.
func (r *StaticPINRepository) List(ctx context.Context) ([]models.StaticPINWithSchedules, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, pin_code, enabled, always_active, created_at, updated_at
		FROM static_pins ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pins []models.StaticPINWithSchedules
	for rows.Next() {
		var pin models.StaticPINWithSchedules
		if err := rows.Scan(&pin.ID, &pin.Name, &pin.PINCode, &pin.Enabled, &pin.AlwaysActive,
			&pin.CreatedAt, &pin.UpdatedAt); err != nil {
			continue
		}
		pins = append(pins, pin)
	}

	// Load schedules for all PINs
	for i := range pins {
		schedules, err := r.GetSchedules(ctx, pins[i].ID)
		if err == nil {
			pins[i].Schedules = schedules
		}
	}

	return pins, nil
}

// ListEnabled retrieves all enabled static PINs.
func (r *StaticPINRepository) ListEnabled(ctx context.Context) ([]models.StaticPINWithSchedules, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, pin_code, enabled, always_active, created_at, updated_at
		FROM static_pins WHERE enabled = 1 ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pins []models.StaticPINWithSchedules
	for rows.Next() {
		var pin models.StaticPINWithSchedules
		if err := rows.Scan(&pin.ID, &pin.Name, &pin.PINCode, &pin.Enabled, &pin.AlwaysActive,
			&pin.CreatedAt, &pin.UpdatedAt); err != nil {
			continue
		}
		pins = append(pins, pin)
	}

	// Load schedules for all PINs
	for i := range pins {
		schedules, err := r.GetSchedules(ctx, pins[i].ID)
		if err == nil {
			pins[i].Schedules = schedules
		}
	}

	return pins, nil
}

// Update updates a static PIN.
func (r *StaticPINRepository) Update(ctx context.Context, pin *models.StaticPIN) error {
	pin.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		UPDATE static_pins SET
			name = ?, pin_code = ?, enabled = ?, always_active = ?, updated_at = ?
		WHERE id = ?
	`, pin.Name, pin.PINCode, pin.Enabled, pin.AlwaysActive, pin.UpdatedAt, pin.ID)

	return err
}

// Delete deletes a static PIN and its schedules (cascade).
func (r *StaticPINRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM static_pins WHERE id = ?", id)
	return err
}

// GetSchedules retrieves all schedules for a static PIN.
func (r *StaticPINRepository) GetSchedules(ctx context.Context, staticPINID string) ([]models.StaticPINSchedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, static_pin_id, day_of_week, start_time, end_time
		FROM static_pin_schedules WHERE static_pin_id = ?
		ORDER BY day_of_week, start_time
	`, staticPINID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.StaticPINSchedule
	for rows.Next() {
		var s models.StaticPINSchedule
		if err := rows.Scan(&s.ID, &s.StaticPINID, &s.DayOfWeek, &s.StartTime, &s.EndTime); err != nil {
			continue
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}

// SetSchedules replaces all schedules for a static PIN.
func (r *StaticPINRepository) SetSchedules(ctx context.Context, staticPINID string, schedules []models.StaticPINSchedule) error {
	// Delete existing schedules
	if _, err := r.db.ExecContext(ctx, "DELETE FROM static_pin_schedules WHERE static_pin_id = ?", staticPINID); err != nil {
		return err
	}

	// Insert new schedules
	for _, s := range schedules {
		id := GenerateID()
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO static_pin_schedules (id, static_pin_id, day_of_week, start_time, end_time)
			VALUES (?, ?, ?, ?, ?)
		`, id, staticPINID, s.DayOfWeek, s.StartTime, s.EndTime); err != nil {
			return err
		}
	}

	return nil
}

// AssignToLock assigns a static PIN to a lock.
func (r *StaticPINRepository) AssignToLock(ctx context.Context, staticPINID, lockID string, slotNumber int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO static_pin_locks (static_pin_id, lock_id, slot_number, sync_status)
		VALUES (?, ?, ?, ?)
	`, staticPINID, lockID, slotNumber, models.StaticPINSyncPending)
	return err
}

// RemoveFromLock removes a static PIN from a lock.
func (r *StaticPINRepository) RemoveFromLock(ctx context.Context, staticPINID, lockID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM static_pin_locks WHERE static_pin_id = ? AND lock_id = ?
	`, staticPINID, lockID)
	return err
}

// GetLockAssignments retrieves all lock assignments for a static PIN.
func (r *StaticPINRepository) GetLockAssignments(ctx context.Context, staticPINID string) ([]models.StaticPINLock, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT static_pin_id, lock_id, slot_number, sync_status, last_synced_at
		FROM static_pin_locks WHERE static_pin_id = ?
	`, staticPINID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []models.StaticPINLock
	for rows.Next() {
		var a models.StaticPINLock
		if err := rows.Scan(&a.StaticPINID, &a.LockID, &a.SlotNumber, &a.SyncStatus, &a.LastSyncedAt); err != nil {
			continue
		}
		assignments = append(assignments, a)
	}

	return assignments, nil
}

// UpdateLockSyncStatus updates the sync status for a static PIN on a lock.
func (r *StaticPINRepository) UpdateLockSyncStatus(ctx context.Context, staticPINID, lockID, status string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE static_pin_locks SET
			sync_status = ?,
			last_synced_at = ?
		WHERE static_pin_id = ? AND lock_id = ?
	`, status, now, staticPINID, lockID)
	return err
}

// ListPendingSync retrieves all static PIN/lock combinations that need syncing.
func (r *StaticPINRepository) ListPendingSync(ctx context.Context) ([]models.StaticPINLock, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT spl.static_pin_id, spl.lock_id, spl.slot_number, spl.sync_status, spl.last_synced_at
		FROM static_pin_locks spl
		JOIN static_pins sp ON sp.id = spl.static_pin_id
		WHERE sp.enabled = 1 AND spl.sync_status = 'pending'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []models.StaticPINLock
	for rows.Next() {
		var a models.StaticPINLock
		if err := rows.Scan(&a.StaticPINID, &a.LockID, &a.SlotNumber, &a.SyncStatus, &a.LastSyncedAt); err != nil {
			continue
		}
		assignments = append(assignments, a)
	}

	return assignments, nil
}


