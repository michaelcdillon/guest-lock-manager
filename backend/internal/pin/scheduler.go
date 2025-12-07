package pin

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
	"github.com/guest-lock-manager/backend/internal/lock"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// StatusScheduler manages PIN activation and expiration.
type StatusScheduler struct {
	cron         *cron.Cron
	guestPINRepo *storage.GuestPINRepository
	lockManager  *lock.Manager
	broadcaster  *websocket.EventBroadcaster
}

// NewStatusScheduler creates a new PIN status scheduler.
func NewStatusScheduler(
	guestPINRepo *storage.GuestPINRepository,
	lockManager *lock.Manager,
	hub *websocket.Hub,
) *StatusScheduler {
	var broadcaster *websocket.EventBroadcaster
	if hub != nil {
		broadcaster = websocket.NewEventBroadcaster(hub)
	}

	return &StatusScheduler{
		cron:         cron.New(cron.WithSeconds()),
		guestPINRepo: guestPINRepo,
		lockManager:  lockManager,
		broadcaster:  broadcaster,
	}
}

// Start begins the PIN status scheduler.
func (s *StatusScheduler) Start() {
	log.Println("Starting PIN status scheduler...")

	// Check for PIN activations every minute
	s.cron.AddFunc("@every 1m", func() {
		s.activatePendingPINs()
	})

	// Check for PIN expirations every minute
	s.cron.AddFunc("@every 1m", func() {
		s.expireOldPINs()
	})

	// Sync pending lock operations every 30 seconds
	s.cron.AddFunc("@every 30s", func() {
		s.syncPendingOperations()
	})

	s.cron.Start()
	log.Println("PIN status scheduler started")
}

// Stop gracefully shuts down the scheduler.
func (s *StatusScheduler) Stop() {
	log.Println("Stopping PIN status scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("PIN status scheduler stopped")
}

// activatePendingPINs activates PINs that should now be active.
func (s *StatusScheduler) activatePendingPINs() {
	ctx := context.Background()

	// Get PINs that should be activated
	pins, err := s.guestPINRepo.ListPendingActivation(ctx)
	if err != nil {
		log.Printf("Failed to list pending activation PINs: %v", err)
		return
	}

	for _, pin := range pins {
		oldStatus := pin.Status
		if err := s.guestPINRepo.UpdateStatus(ctx, pin.ID, models.PINStatusActive); err != nil {
			log.Printf("Failed to activate PIN %s: %v", pin.ID, err)
			continue
		}

		log.Printf("Activated PIN %s for event: %s", pin.ID, safeString(pin.EventSummary))

		// Queue PIN to be synced to locks
		s.queueLockSync(ctx, &pin)

		// Broadcast status change
		if s.broadcaster != nil {
			s.broadcaster.BroadcastPINStatusChanged(
				pin.ID,
				"guest",
				oldStatus,
				models.PINStatusActive,
				safeString(pin.EventSummary),
			)
		}
	}
}

// expireOldPINs marks expired PINs and removes them from locks.
func (s *StatusScheduler) expireOldPINs() {
	ctx := context.Background()

	// Get PINs that should be expired
	pins, err := s.guestPINRepo.ListExpired(ctx)
	if err != nil {
		log.Printf("Failed to list expired PINs: %v", err)
		return
	}

	for _, pin := range pins {
		oldStatus := pin.Status
		if err := s.guestPINRepo.UpdateStatus(ctx, pin.ID, models.PINStatusExpired); err != nil {
			log.Printf("Failed to expire PIN %s: %v", pin.ID, err)
			continue
		}

		log.Printf("Expired PIN %s for event: %s", pin.ID, safeString(pin.EventSummary))

		// Queue PIN removal from locks
		s.queueLockClear(ctx, &pin)

		// Broadcast status change
		if s.broadcaster != nil {
			s.broadcaster.BroadcastPINStatusChanged(
				pin.ID,
				"guest",
				oldStatus,
				models.PINStatusExpired,
				safeString(pin.EventSummary),
			)
		}
	}
}

// queueLockSync queues a PIN to be synced to its assigned locks.
func (s *StatusScheduler) queueLockSync(ctx context.Context, pin *models.GuestPIN) {
	if s.lockManager == nil {
		return
	}

	assignments, err := s.guestPINRepo.GetLockAssignments(ctx, pin.ID)
	if err != nil {
		log.Printf("Failed to get lock assignments for PIN %s: %v", pin.ID, err)
		return
	}

	for _, assignment := range assignments {
		if err := s.lockManager.SetPIN(ctx, assignment.LockID, pin.PINCode, assignment.SlotNumber, pin.ID); err != nil {
			log.Printf("Failed to queue PIN sync for lock %s: %v", assignment.LockID, err)
		}
	}
}

// queueLockClear queues a PIN to be cleared from its assigned locks.
func (s *StatusScheduler) queueLockClear(ctx context.Context, pin *models.GuestPIN) {
	if s.lockManager == nil {
		return
	}

	assignments, err := s.guestPINRepo.GetLockAssignments(ctx, pin.ID)
	if err != nil {
		log.Printf("Failed to get lock assignments for PIN %s: %v", pin.ID, err)
		return
	}

	for _, assignment := range assignments {
		if err := s.lockManager.ClearPIN(ctx, assignment.LockID, assignment.SlotNumber, pin.ID); err != nil {
			log.Printf("Failed to queue PIN clear for lock %s: %v", assignment.LockID, err)
		}
	}
}

// syncPendingOperations syncs any pending lock operations.
func (s *StatusScheduler) syncPendingOperations() {
	if s.lockManager == nil {
		return
	}

	ctx := context.Background()
	
	// Sync guest PINs
	if err := s.lockManager.SyncGuestPINs(ctx); err != nil {
		log.Printf("Failed to sync guest PINs: %v", err)
	}

	// Sync static PINs
	if err := s.lockManager.SyncStaticPINs(ctx); err != nil {
		log.Printf("Failed to sync static PINs: %v", err)
	}
}

// safeString returns the string value or empty string if nil.
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

