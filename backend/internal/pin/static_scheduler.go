package pin

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/guest-lock-manager/backend/internal/lock"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// StaticPINScheduler manages static PIN activation based on day/time schedules.
type StaticPINScheduler struct {
	cron          *cron.Cron
	staticPINRepo *storage.StaticPINRepository
	lockManager   *lock.Manager
	evaluator     *ScheduleEvaluator
	broadcaster   *websocket.EventBroadcaster

	// Track current state of each static PIN (active on lock or not)
	pinStates   map[string]bool // pinID -> isActiveOnLock
	pinStatesMu sync.RWMutex
}

// NewStaticPINScheduler creates a new static PIN scheduler.
func NewStaticPINScheduler(
	staticPINRepo *storage.StaticPINRepository,
	lockManager *lock.Manager,
	hub *websocket.Hub,
) *StaticPINScheduler {
	var broadcaster *websocket.EventBroadcaster
	if hub != nil {
		broadcaster = websocket.NewEventBroadcaster(hub)
	}

	return &StaticPINScheduler{
		cron:          cron.New(cron.WithSeconds()),
		staticPINRepo: staticPINRepo,
		lockManager:   lockManager,
		evaluator:     NewScheduleEvaluator(),
		broadcaster:   broadcaster,
		pinStates:     make(map[string]bool),
	}
}

// Start begins the static PIN scheduler.
func (s *StaticPINScheduler) Start() {
	log.Println("Starting static PIN scheduler...")

	// Check for schedule changes every minute
	s.cron.AddFunc("@every 1m", func() {
		s.evaluateSchedules()
	})

	// Initial evaluation
	go s.evaluateSchedules()

	s.cron.Start()
	log.Println("Static PIN scheduler started")
}

// Stop gracefully shuts down the scheduler.
func (s *StaticPINScheduler) Stop() {
	log.Println("Stopping static PIN scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Static PIN scheduler stopped")
}

// evaluateSchedules checks all static PINs and syncs/unsyncs them from locks as needed.
func (s *StaticPINScheduler) evaluateSchedules() {
	ctx := context.Background()

	// Get all enabled static PINs
	pins, err := s.staticPINRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("Failed to list static PINs: %v", err)
		return
	}

	now := time.Now()

	for _, pin := range pins {
		shouldBeActive := s.evaluator.IsStaticPINActive(&pin, now)
		wasActive := s.getPinState(pin.ID)

		if shouldBeActive && !wasActive {
			// PIN just became active - sync to locks
			s.activatePin(ctx, &pin)
		} else if !shouldBeActive && wasActive {
			// PIN just became inactive - remove from locks
			s.deactivatePin(ctx, &pin)
		}
	}
}

// activatePin syncs a static PIN to all assigned locks.
func (s *StaticPINScheduler) activatePin(ctx context.Context, pin *models.StaticPINWithSchedules) {
	log.Printf("Activating static PIN: %s (%s)", pin.ID, pin.Name)

	assignments, err := s.staticPINRepo.GetLockAssignments(ctx, pin.ID)
	if err != nil {
		log.Printf("Failed to get lock assignments for PIN %s: %v", pin.ID, err)
		return
	}

	for _, assignment := range assignments {
		if s.lockManager != nil {
			if err := s.lockManager.SetStaticPIN(ctx, assignment.LockID, pin.PINCode, assignment.SlotNumber, pin.ID); err != nil {
				log.Printf("Failed to set static PIN on lock %s: %v", assignment.LockID, err)
				s.staticPINRepo.UpdateLockSyncStatus(ctx, pin.ID, assignment.LockID, models.StaticPINSyncFailed)
				continue
			}
			s.staticPINRepo.UpdateLockSyncStatus(ctx, pin.ID, assignment.LockID, models.StaticPINSyncSynced)
		}
	}

	s.setPinState(pin.ID, true)

	// Broadcast activation
	if s.broadcaster != nil {
		s.broadcaster.BroadcastPINStatusChanged(pin.ID, "static", "inactive", "active", pin.Name)
	}
}

// deactivatePin removes a static PIN from all assigned locks.
func (s *StaticPINScheduler) deactivatePin(ctx context.Context, pin *models.StaticPINWithSchedules) {
	log.Printf("Deactivating static PIN: %s (%s)", pin.ID, pin.Name)

	assignments, err := s.staticPINRepo.GetLockAssignments(ctx, pin.ID)
	if err != nil {
		log.Printf("Failed to get lock assignments for PIN %s: %v", pin.ID, err)
		return
	}

	for _, assignment := range assignments {
		if s.lockManager != nil {
			if err := s.lockManager.ClearStaticPIN(ctx, assignment.LockID, assignment.SlotNumber, pin.ID); err != nil {
				log.Printf("Failed to clear static PIN from lock %s: %v", assignment.LockID, err)
				continue
			}
			s.staticPINRepo.UpdateLockSyncStatus(ctx, pin.ID, assignment.LockID, models.StaticPINSyncPending)
		}
	}

	s.setPinState(pin.ID, false)

	// Broadcast deactivation
	if s.broadcaster != nil {
		s.broadcaster.BroadcastPINStatusChanged(pin.ID, "static", "active", "inactive", pin.Name)
	}
}

// getPinState returns the current active state of a PIN.
func (s *StaticPINScheduler) getPinState(pinID string) bool {
	s.pinStatesMu.RLock()
	defer s.pinStatesMu.RUnlock()
	return s.pinStates[pinID]
}

// setPinState updates the current active state of a PIN.
func (s *StaticPINScheduler) setPinState(pinID string, active bool) {
	s.pinStatesMu.Lock()
	defer s.pinStatesMu.Unlock()
	s.pinStates[pinID] = active
}

// ForceEvaluate triggers an immediate schedule evaluation.
// Useful after creating or updating a static PIN.
func (s *StaticPINScheduler) ForceEvaluate() {
	go s.evaluateSchedules()
}

// InitializeStates loads the initial state for all enabled static PINs.
// Should be called after starting the scheduler.
func (s *StaticPINScheduler) InitializeStates(ctx context.Context) error {
	pins, err := s.staticPINRepo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, pin := range pins {
		isActive := s.evaluator.IsStaticPINActive(&pin, now)
		s.setPinState(pin.ID, isActive)
	}

	return nil
}

