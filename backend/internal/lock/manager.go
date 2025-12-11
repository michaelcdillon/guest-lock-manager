package lock

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// Manager orchestrates lock operations and PIN synchronization.
type Manager struct {
	db           *storage.DB
	lockRepo     *storage.LockRepository
	guestPINRepo *storage.GuestPINRepository
	haClient     *HAClient
	zwaveClient  *ZWaveJSUIClient

	// Batching for battery efficiency
	pendingOps  map[string][]PINOperation
	batchMu     sync.Mutex
	batchWindow time.Duration
	batchTimer  *time.Timer
}

// PINOperation represents a pending PIN operation on a lock.
type PINOperation struct {
	LockID     string
	PINCode    string
	SlotNumber int
	Operation  string // "set" or "clear"
	GuestPINID string
}

// NewManager creates a new lock manager.
func NewManager(db *storage.DB, lockRepo *storage.LockRepository, guestPINRepo *storage.GuestPINRepository, batchWindowSeconds int) *Manager {
	config := DefaultConfig()
	haClient := NewHAClient(config)
	zwaveClient := NewZWaveJSUIClient()

	if batchWindowSeconds <= 0 {
		batchWindowSeconds = 30
	}

	return &Manager{
		db:           db,
		lockRepo:     lockRepo,
		guestPINRepo: guestPINRepo,
		haClient:     haClient,
		zwaveClient:  zwaveClient,
		pendingOps:   make(map[string][]PINOperation),
		batchWindow:  time.Duration(batchWindowSeconds) * time.Second,
	}
}

// pinWriter abstracts how PIN operations are sent (HA API or direct protocol).
type pinWriter interface {
	Set(ctx context.Context, slot int, code string) error
	Clear(ctx context.Context, slot int) error
	Name() string
}

type haPinWriter struct {
	client   *HAClient
	entityID string
}

func (w haPinWriter) Set(ctx context.Context, slot int, code string) error {
	return w.client.SetUserCode(ctx, w.entityID, slot, code)
}

func (w haPinWriter) Clear(ctx context.Context, slot int) error {
	return w.client.ClearUserCode(ctx, w.entityID, slot)
}

func (w haPinWriter) Name() string {
	return "home_assistant"
}

type zwavePinWriter struct {
	client *ZWaveJSUIClient
	nodeID int
}

func (w zwavePinWriter) Set(ctx context.Context, slot int, code string) error {
	return w.client.SetUserCode(ctx, w.nodeID, slot, code)
}

func (w zwavePinWriter) Clear(ctx context.Context, slot int) error {
	return w.client.ClearUserCode(ctx, w.nodeID, slot)
}

func (w zwavePinWriter) Name() string {
	return "zwave_js_ui"
}

// SetPIN queues a PIN to be set on a lock.
func (m *Manager) SetPIN(ctx context.Context, lockID, pinCode string, slotNumber int, guestPINID string) error {
	op := PINOperation{
		LockID:     lockID,
		PINCode:    pinCode,
		SlotNumber: slotNumber,
		Operation:  "set",
		GuestPINID: guestPINID,
	}

	m.queueOperation(op)
	return nil
}

// ClearPIN queues a PIN to be cleared from a lock.
func (m *Manager) ClearPIN(ctx context.Context, lockID string, slotNumber int, guestPINID string) error {
	op := PINOperation{
		LockID:     lockID,
		SlotNumber: slotNumber,
		Operation:  "clear",
		GuestPINID: guestPINID,
	}

	m.queueOperation(op)
	return nil
}

// queueOperation adds an operation to the batch queue.
func (m *Manager) queueOperation(op PINOperation) {
	m.batchMu.Lock()
	defer m.batchMu.Unlock()

	m.pendingOps[op.LockID] = append(m.pendingOps[op.LockID], op)

	// Start or reset the batch timer
	if m.batchTimer == nil {
		m.batchTimer = time.AfterFunc(m.batchWindow, m.flushBatch)
	}
}

// flushBatch executes all pending operations.
func (m *Manager) flushBatch() {
	m.batchMu.Lock()
	ops := m.pendingOps
	m.pendingOps = make(map[string][]PINOperation)
	m.batchTimer = nil
	m.batchMu.Unlock()

	ctx := context.Background()

	// Preload current lock states so we can fetch node_ids for direct writes.
	stateMap := make(map[string]LockEntity)
	if haLocks, err := m.haClient.GetLocks(ctx); err == nil {
		for _, l := range haLocks {
			stateMap[l.EntityID] = l
		}
	} else {
		log.Printf("Warning: failed to fetch HA lock states for direct integration: %v", err)
	}

	for lockID, lockOps := range ops {
		lock, err := m.lockRepo.GetByID(ctx, lockID)
		if err != nil || lock == nil {
			log.Printf("Lock not found: %s", lockID)
			continue
		}

		haWriter := haPinWriter{client: m.haClient, entityID: lock.EntityID}
		primary := pinWriter(haWriter)
		var fallback pinWriter

		if lock.DirectIntegration != nil && *lock.DirectIntegration == string(models.DirectZWaveJSUI) {
			if entity, ok := stateMap[lock.EntityID]; ok && entity.Attributes.NodeID != nil {
				primary = zwavePinWriter{client: m.zwaveClient, nodeID: *entity.Attributes.NodeID}
				fallback = haWriter
				log.Printf("Lock %s (%s): using direct zwave_js_ui with node_id=%d (fallback=home_assistant)", lockID, lock.EntityID, *entity.Attributes.NodeID)
			} else {
				log.Printf("Direct integration requested but node_id missing for lock %s (%s); using HA", lockID, lock.EntityID)
			}
		}

		for _, op := range lockOps {
			var err error
			if op.Operation == "set" {
				err = primary.Set(ctx, op.SlotNumber, op.PINCode)
				if err != nil && fallback != nil {
					log.Printf("Direct PIN set failed via %s for lock %s slot %d; falling back: %v", primary.Name(), lockID, op.SlotNumber, err)
					err = fallback.Set(ctx, op.SlotNumber, op.PINCode)
				}
			} else {
				err = primary.Clear(ctx, op.SlotNumber)
				if err != nil && fallback != nil {
					log.Printf("Direct PIN clear failed via %s for lock %s slot %d; falling back: %v", primary.Name(), lockID, op.SlotNumber, err)
					err = fallback.Clear(ctx, op.SlotNumber)
				}
			}

			// Update sync status
			status := models.LockSyncSynced
			var errMsg *string
			if err != nil {
				status = models.LockSyncFailed
				msg := err.Error()
				errMsg = &msg
				log.Printf("Failed to %s PIN on lock %s slot %d: %v", op.Operation, lockID, op.SlotNumber, err)
			}

			if op.GuestPINID != "" {
				m.guestPINRepo.UpdateLockSyncStatus(ctx, op.GuestPINID, lockID, status, errMsg)
			}
		}
	}
}

// FlushNow immediately executes all pending operations.
func (m *Manager) FlushNow() {
	m.batchMu.Lock()
	if m.batchTimer != nil {
		m.batchTimer.Stop()
	}
	m.batchMu.Unlock()

	m.flushBatch()
}

// SyncGuestPINs synchronizes all pending guest PIN changes to locks.
func (m *Manager) SyncGuestPINs(ctx context.Context) error {
	// Get all active PINs with pending sync status
	rows, err := m.db.QueryContext(ctx, `
		SELECT gpl.guest_pin_id, gpl.lock_id, gpl.slot_number, gp.pin_code, gp.status
		FROM guest_pin_locks gpl
		JOIN guest_pins gp ON gp.id = gpl.guest_pin_id
		WHERE gpl.sync_status = 'pending'
	`)
	if err != nil {
		return fmt.Errorf("querying pending syncs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var guestPINID, lockID, pinCode, status string
		var slotNumber int
		if err := rows.Scan(&guestPINID, &lockID, &slotNumber, &pinCode, &status); err != nil {
			continue
		}

		if status == models.PINStatusActive {
			m.SetPIN(ctx, lockID, pinCode, slotNumber, guestPINID)
		} else if status == models.PINStatusExpired {
			m.ClearPIN(ctx, lockID, slotNumber, guestPINID)
		}
	}

	return rows.Err()
}

// SyncStaticPINs synchronizes all pending static PIN changes to locks.
func (m *Manager) SyncStaticPINs(ctx context.Context) error {
	// Get all enabled static PINs with pending sync status
	rows, err := m.db.QueryContext(ctx, `
		SELECT spl.static_pin_id, spl.lock_id, spl.slot_number, sp.pin_code, sp.enabled
		FROM static_pin_locks spl
		JOIN static_pins sp ON sp.id = spl.static_pin_id
		WHERE spl.sync_status = 'pending'
	`)
	if err != nil {
		return fmt.Errorf("querying pending static syncs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var staticPINID, lockID, pinCode string
		var slotNumber int
		var enabled bool
		if err := rows.Scan(&staticPINID, &lockID, &slotNumber, &pinCode, &enabled); err != nil {
			continue
		}

		if enabled {
			m.SetPIN(ctx, lockID, pinCode, slotNumber, "")
		} else {
			m.ClearPIN(ctx, lockID, slotNumber, "")
		}
	}

	return rows.Err()
}

// RefreshLockStatus updates the status of all managed locks.
func (m *Manager) RefreshLockStatus(ctx context.Context) error {
	locks, err := m.lockRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("listing locks: %w", err)
	}

	// Get current states from HA
	haLocks, err := m.haClient.GetLocks(ctx)
	if err != nil {
		return fmt.Errorf("getting HA locks: %w", err)
	}

	// Build map of entity states
	stateMap := make(map[string]LockEntity)
	for _, l := range haLocks {
		stateMap[l.EntityID] = l
	}

	// Update each managed lock
	for _, lock := range locks {
		if state, ok := stateMap[lock.EntityID]; ok {
			online := state.State != "unavailable"
			m.lockRepo.UpdateStatus(ctx, lock.ID, online, state.Attributes.Battery)
		} else {
			m.lockRepo.UpdateStatus(ctx, lock.ID, false, nil)
		}
	}

	return nil
}

// SetStaticPIN queues a static PIN to be set on a lock.
func (m *Manager) SetStaticPIN(ctx context.Context, lockID, pinCode string, slotNumber int, staticPINID string) error {
	op := PINOperation{
		LockID:     lockID,
		PINCode:    pinCode,
		SlotNumber: slotNumber,
		Operation:  "set",
		// Note: GuestPINID is repurposed here for staticPINID tracking
	}

	log.Printf("Queueing static PIN set: lock=%s slot=%d", lockID, slotNumber)
	m.queueOperation(op)
	return nil
}

// ClearStaticPIN queues a static PIN to be cleared from a lock.
func (m *Manager) ClearStaticPIN(ctx context.Context, lockID string, slotNumber int, staticPINID string) error {
	op := PINOperation{
		LockID:     lockID,
		SlotNumber: slotNumber,
		Operation:  "clear",
	}

	log.Printf("Queueing static PIN clear: lock=%s slot=%d", lockID, slotNumber)
	m.queueOperation(op)
	return nil
}
