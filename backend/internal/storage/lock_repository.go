package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// LockRepository provides data access for managed locks.
type LockRepository struct {
	BaseRepository
}

// NewLockRepository creates a new lock repository.
func NewLockRepository(db *DB) *LockRepository {
	return &LockRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new managed lock.
func (r *LockRepository) Create(ctx context.Context, lock *models.ManagedLock) error {
	lock.ID = GenerateID()
	lock.CreatedAt = r.Now()
	lock.UpdatedAt = r.Now()

	_, err := r.DB().ExecContext(ctx, `
		INSERT INTO managed_locks (
			id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			online, state, battery_level, last_seen_at, direct_integration, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		lock.ID, lock.EntityID, lock.Name, lock.Protocol,
		lock.TotalSlots, lock.GuestSlots, lock.StaticSlots,
		lock.Online, lock.State, lock.BatteryLevel, lock.LastSeenAt,
		lock.DirectIntegration, lock.CreatedAt, lock.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("inserting lock: %w", err)
	}

	return nil
}

// GetByID retrieves a lock by its ID.
func (r *LockRepository) GetByID(ctx context.Context, id string) (*models.ManagedLock, error) {
	lock := &models.ManagedLock{}

	err := r.DB().QueryRowContext(ctx, `
		SELECT id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			   online, state, battery_level, last_seen_at, direct_integration, created_at, updated_at
		FROM managed_locks WHERE id = ?
	`, id).Scan(
		&lock.ID, &lock.EntityID, &lock.Name, &lock.Protocol,
		&lock.TotalSlots, &lock.GuestSlots, &lock.StaticSlots,
		&lock.Online, &lock.State, &lock.BatteryLevel, &lock.LastSeenAt,
		&lock.DirectIntegration, &lock.CreatedAt, &lock.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying lock: %w", err)
	}

	return lock, nil
}

// GetByEntityID retrieves a lock by its Home Assistant entity ID.
func (r *LockRepository) GetByEntityID(ctx context.Context, entityID string) (*models.ManagedLock, error) {
	lock := &models.ManagedLock{}

	err := r.DB().QueryRowContext(ctx, `
		SELECT id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			   online, state, battery_level, last_seen_at, direct_integration, created_at, updated_at
		FROM managed_locks WHERE entity_id = ?
	`, entityID).Scan(
		&lock.ID, &lock.EntityID, &lock.Name, &lock.Protocol,
		&lock.TotalSlots, &lock.GuestSlots, &lock.StaticSlots,
		&lock.Online, &lock.State, &lock.BatteryLevel, &lock.LastSeenAt,
		&lock.DirectIntegration, &lock.CreatedAt, &lock.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying lock: %w", err)
	}

	return lock, nil
}

// List retrieves all managed locks.
func (r *LockRepository) List(ctx context.Context) ([]models.ManagedLock, error) {
	rows, err := r.DB().QueryContext(ctx, `
		SELECT id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			   online, state, battery_level, last_seen_at, direct_integration, created_at, updated_at
		FROM managed_locks
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("querying locks: %w", err)
	}
	defer rows.Close()

	var locks []models.ManagedLock
	for rows.Next() {
		var lock models.ManagedLock
		if err := rows.Scan(
			&lock.ID, &lock.EntityID, &lock.Name, &lock.Protocol,
			&lock.TotalSlots, &lock.GuestSlots, &lock.StaticSlots,
			&lock.Online, &lock.State, &lock.BatteryLevel, &lock.LastSeenAt,
			&lock.DirectIntegration, &lock.CreatedAt, &lock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning lock: %w", err)
		}
		locks = append(locks, lock)
	}

	return locks, rows.Err()
}

// Update updates an existing lock.
func (r *LockRepository) Update(ctx context.Context, lock *models.ManagedLock) error {
	lock.UpdatedAt = r.Now()

	result, err := r.DB().ExecContext(ctx, `
		UPDATE managed_locks SET
			name = ?, protocol = ?, total_slots = ?, guest_slots = ?, static_slots = ?,
			online = ?, state = ?, battery_level = ?, last_seen_at = ?, direct_integration = ?,
			updated_at = ?
		WHERE id = ?
	`,
		lock.Name, lock.Protocol, lock.TotalSlots, lock.GuestSlots, lock.StaticSlots,
		lock.Online, lock.State, lock.BatteryLevel, lock.LastSeenAt, lock.DirectIntegration,
		lock.UpdatedAt, lock.ID,
	)

	if err != nil {
		return fmt.Errorf("updating lock: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("lock not found: %s", lock.ID)
	}

	return nil
}

// Delete removes a lock by ID.
func (r *LockRepository) Delete(ctx context.Context, id string) error {
	result, err := r.DB().ExecContext(ctx, "DELETE FROM managed_locks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting lock: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("lock not found: %s", id)
	}

	return nil
}

// UpdateStatus updates the online status and battery level of a lock.
func (r *LockRepository) UpdateStatus(ctx context.Context, id string, online bool, batteryLevel *int) error {
	now := time.Now().UTC()

	_, err := r.DB().ExecContext(ctx, `
		UPDATE managed_locks SET
			online = ?, battery_level = ?, last_seen_at = ?, updated_at = ?
		WHERE id = ?
	`, online, batteryLevel, now, now, id)

	if err != nil {
		return fmt.Errorf("updating lock status: %w", err)
	}

	return nil
}
