package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// GuestPINRepository provides data access for guest PINs.
type GuestPINRepository struct {
	BaseRepository
}

// NewGuestPINRepository creates a new guest PIN repository.
func NewGuestPINRepository(db *DB) *GuestPINRepository {
	return &GuestPINRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new guest PIN.
func (r *GuestPINRepository) Create(ctx context.Context, pin *models.GuestPIN) error {
	pin.ID = GenerateID()
	pin.CreatedAt = r.Now()
	pin.UpdatedAt = r.Now()

	_, err := r.DB().ExecContext(ctx, `
		INSERT INTO guest_pins (
			id, calendar_id, event_uid, event_summary, pin_code, generation_method,
			custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		pin.ID, pin.CalendarID, pin.EventUID, pin.EventSummary, pin.PINCode,
		pin.GenerationMethod, pin.CustomPIN, pin.ValidFrom, pin.ValidUntil,
		pin.Status, pin.RegenerationEligible, pin.CreatedAt, pin.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("inserting guest PIN: %w", err)
	}

	return nil
}

// GetByID retrieves a guest PIN by its ID.
func (r *GuestPINRepository) GetByID(ctx context.Context, id string) (*models.GuestPIN, error) {
	pin := &models.GuestPIN{}

	err := r.DB().QueryRowContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins WHERE id = ?
	`, id).Scan(
		&pin.ID, &pin.CalendarID, &pin.EventUID, &pin.EventSummary, &pin.PINCode,
		&pin.GenerationMethod, &pin.CustomPIN, &pin.ValidFrom, &pin.ValidUntil,
		&pin.Status, &pin.RegenerationEligible, &pin.CreatedAt, &pin.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying guest PIN: %w", err)
	}

	return pin, nil
}

// GetByEventUID retrieves a guest PIN by calendar ID and event UID.
func (r *GuestPINRepository) GetByEventUID(ctx context.Context, calendarID, eventUID string) (*models.GuestPIN, error) {
	pin := &models.GuestPIN{}

	err := r.DB().QueryRowContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins WHERE calendar_id = ? AND event_uid = ?
	`, calendarID, eventUID).Scan(
		&pin.ID, &pin.CalendarID, &pin.EventUID, &pin.EventSummary, &pin.PINCode,
		&pin.GenerationMethod, &pin.CustomPIN, &pin.ValidFrom, &pin.ValidUntil,
		&pin.Status, &pin.RegenerationEligible, &pin.CreatedAt, &pin.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying guest PIN by event: %w", err)
	}

	return pin, nil
}

// ListByCalendar retrieves all guest PINs for a calendar.
func (r *GuestPINRepository) ListByCalendar(ctx context.Context, calendarID string) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE calendar_id = ?
		ORDER BY valid_from DESC
	`, calendarID)
	if err != nil {
		return nil, fmt.Errorf("querying guest PINs: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

// ListByStatus retrieves all guest PINs with a specific status.
func (r *GuestPINRepository) ListByStatus(ctx context.Context, status string) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE status = ?
		ORDER BY valid_from DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("querying guest PINs by status: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

// ListActive retrieves all currently active guest PINs.
func (r *GuestPINRepository) ListActive(ctx context.Context) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE status = 'active'
		ORDER BY valid_from DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying active guest PINs: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

// ListPendingActivation retrieves PINs that should become active.
func (r *GuestPINRepository) ListPendingActivation(ctx context.Context) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE status = 'pending' AND valid_from <= datetime('now')
		ORDER BY valid_from
	`)
	if err != nil {
		return nil, fmt.Errorf("querying pending activation PINs: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

// ListExpired retrieves PINs that should be marked as expired.
func (r *GuestPINRepository) ListExpired(ctx context.Context) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE status = 'active' AND valid_until <= datetime('now')
		ORDER BY valid_until
	`)
	if err != nil {
		return nil, fmt.Errorf("querying expired PINs: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

func (r *GuestPINRepository) scanPINs(rows *sql.Rows) ([]models.GuestPIN, error) {
	var pins []models.GuestPIN
	for rows.Next() {
		var pin models.GuestPIN
		if err := rows.Scan(
			&pin.ID, &pin.CalendarID, &pin.EventUID, &pin.EventSummary, &pin.PINCode,
			&pin.GenerationMethod, &pin.CustomPIN, &pin.ValidFrom, &pin.ValidUntil,
			&pin.Status, &pin.RegenerationEligible, &pin.CreatedAt, &pin.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning guest PIN: %w", err)
		}
		pins = append(pins, pin)
	}
	return pins, rows.Err()
}

// Update updates an existing guest PIN.
func (r *GuestPINRepository) Update(ctx context.Context, pin *models.GuestPIN) error {
	pin.UpdatedAt = r.Now()

	result, err := r.DB().ExecContext(ctx, `
		UPDATE guest_pins SET
			event_summary = ?, pin_code = ?, generation_method = ?, custom_pin = ?,
			valid_from = ?, valid_until = ?, status = ?, regeneration_eligible = ?, updated_at = ?
		WHERE id = ?
	`,
		pin.EventSummary, pin.PINCode, pin.GenerationMethod, pin.CustomPIN,
		pin.ValidFrom, pin.ValidUntil, pin.Status, pin.RegenerationEligible,
		pin.UpdatedAt, pin.ID,
	)

	if err != nil {
		return fmt.Errorf("updating guest PIN: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("guest PIN not found: %s", pin.ID)
	}

	return nil
}

// UpdateStatus updates just the status of a guest PIN.
func (r *GuestPINRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.DB().ExecContext(ctx, `
		UPDATE guest_pins SET status = ?, updated_at = ? WHERE id = ?
	`, status, r.Now(), id)

	if err != nil {
		return fmt.Errorf("updating PIN status: %w", err)
	}

	return nil
}

// Delete removes a guest PIN by ID.
func (r *GuestPINRepository) Delete(ctx context.Context, id string) error {
	result, err := r.DB().ExecContext(ctx, "DELETE FROM guest_pins WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting guest PIN: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("guest PIN not found: %s", id)
	}

	return nil
}

// DeleteByCalendar removes all guest PINs for a calendar.
func (r *GuestPINRepository) DeleteByCalendar(ctx context.Context, calendarID string) error {
	_, err := r.DB().ExecContext(ctx, "DELETE FROM guest_pins WHERE calendar_id = ?", calendarID)
	if err != nil {
		return fmt.Errorf("deleting guest PINs for calendar: %w", err)
	}
	return nil
}

// FindConflicts finds PINs with the same code that have overlapping validity windows.
func (r *GuestPINRepository) FindConflicts(ctx context.Context, pinCode string, validFrom, validUntil string, excludeID string) ([]models.GuestPIN, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
		       custom_pin, valid_from, valid_until, status, regeneration_eligible, created_at, updated_at
		FROM guest_pins
		WHERE pin_code = ?
		  AND id != ?
		  AND valid_from < ?
		  AND valid_until > ?
		  AND status NOT IN ('expired', 'conflict')
	`, pinCode, excludeID, validUntil, validFrom)
	if err != nil {
		return nil, fmt.Errorf("querying PIN conflicts: %w", err)
	}
	defer rows.Close()

	return r.scanPINs(rows)
}

// AssignToLock creates or updates a guest PIN to lock assignment.
func (r *GuestPINRepository) AssignToLock(ctx context.Context, guestPINID, lockID string, slotNumber int) error {
	_, err := r.DB().ExecContext(ctx, `
		INSERT INTO guest_pin_locks (guest_pin_id, lock_id, slot_number, sync_status)
		VALUES (?, ?, ?, 'pending')
		ON CONFLICT(guest_pin_id, lock_id) DO UPDATE SET
			slot_number = ?, sync_status = 'pending'
	`, guestPINID, lockID, slotNumber, slotNumber)

	if err != nil {
		return fmt.Errorf("assigning PIN to lock: %w", err)
	}

	return nil
}

// UpdateLockSyncStatus updates the sync status for a PIN-lock assignment.
func (r *GuestPINRepository) UpdateLockSyncStatus(ctx context.Context, guestPINID, lockID, status string, errMsg *string) error {
	_, err := r.DB().ExecContext(ctx, `
		UPDATE guest_pin_locks SET
			sync_status = ?, last_synced_at = datetime('now'), error_message = ?
		WHERE guest_pin_id = ? AND lock_id = ?
	`, status, errMsg, guestPINID, lockID)

	if err != nil {
		return fmt.Errorf("updating lock sync status: %w", err)
	}

	return nil
}

// GetLockAssignments retrieves all lock assignments for a guest PIN.
func (r *GuestPINRepository) GetLockAssignments(ctx context.Context, guestPINID string) ([]models.GuestPINLock, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT guest_pin_id, lock_id, slot_number, sync_status, last_synced_at, error_message
		FROM guest_pin_locks WHERE guest_pin_id = ?
	`, guestPINID)
	if err != nil {
		return nil, fmt.Errorf("querying lock assignments: %w", err)
	}
	defer rows.Close()

	var assignments []models.GuestPINLock
	for rows.Next() {
		var a models.GuestPINLock
		if err := rows.Scan(&a.GuestPINID, &a.LockID, &a.SlotNumber, &a.SyncStatus, &a.LastSyncedAt, &a.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scanning lock assignment: %w", err)
		}
		assignments = append(assignments, a)
	}

	return assignments, rows.Err()
}

