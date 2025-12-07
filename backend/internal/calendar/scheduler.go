package calendar

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// Scheduler manages periodic calendar sync jobs.
type Scheduler struct {
	cron         *cron.Cron
	syncService  *SyncService
	calendarRepo *storage.CalendarRepository
	broadcaster  *websocket.EventBroadcaster
	
	// Track jobs per calendar
	jobs   map[string]cron.EntryID
	jobsMu sync.RWMutex
	
	// Default sync interval if calendar doesn't specify
	defaultInterval time.Duration
}

// NewScheduler creates a new calendar sync scheduler.
func NewScheduler(
	syncService *SyncService,
	calendarRepo *storage.CalendarRepository,
	hub *websocket.Hub,
	defaultIntervalMin int,
) *Scheduler {
	if defaultIntervalMin <= 0 {
		defaultIntervalMin = 15
	}

	var broadcaster *websocket.EventBroadcaster
	if hub != nil {
		broadcaster = websocket.NewEventBroadcaster(hub)
	}

	return &Scheduler{
		cron:            cron.New(cron.WithSeconds()),
		syncService:     syncService,
		calendarRepo:    calendarRepo,
		broadcaster:     broadcaster,
		jobs:            make(map[string]cron.EntryID),
		defaultInterval: time.Duration(defaultIntervalMin) * time.Minute,
	}
}

// Start begins the scheduler and loads all enabled calendars.
func (s *Scheduler) Start(ctx context.Context) error {
	log.Println("Starting calendar sync scheduler...")

	// Load all enabled calendars and schedule them
	calendars, err := s.calendarRepo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	for _, cal := range calendars {
		s.ScheduleCalendar(cal)
	}

	// Add a job to check for PIN status updates every minute
	s.cron.AddFunc("@every 1m", func() {
		s.updatePINStatuses()
	})

	// Add a job to refresh calendar schedules every 5 minutes
	// This catches any newly added or modified calendars
	s.cron.AddFunc("@every 5m", func() {
		s.refreshSchedules(context.Background())
	})

	s.cron.Start()
	log.Printf("Calendar scheduler started with %d calendars", len(calendars))

	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() {
	log.Println("Stopping calendar sync scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Calendar scheduler stopped")
}

// ScheduleCalendar adds or updates a calendar's sync schedule.
func (s *Scheduler) ScheduleCalendar(cal models.CalendarSubscription) {
	if !cal.Enabled {
		s.UnscheduleCalendar(cal.ID)
		return
	}

	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	// Remove existing job if any
	if existingID, exists := s.jobs[cal.ID]; exists {
		s.cron.Remove(existingID)
		delete(s.jobs, cal.ID)
	}

	// Calculate interval
	interval := time.Duration(cal.SyncIntervalMin) * time.Minute
	if interval < time.Minute {
		interval = s.defaultInterval
	}

	// Create cron expression: run every N minutes
	spec := minutesToCronSpec(cal.SyncIntervalMin)

	entryID, err := s.cron.AddFunc(spec, func() {
		s.syncCalendar(cal.ID, cal.Name)
	})

	if err != nil {
		log.Printf("Failed to schedule calendar %s: %v", cal.ID, err)
		return
	}

	s.jobs[cal.ID] = entryID
	log.Printf("Scheduled calendar %s (%s) every %d minutes", cal.ID, cal.Name, cal.SyncIntervalMin)
}

// UnscheduleCalendar removes a calendar from the sync schedule.
func (s *Scheduler) UnscheduleCalendar(calendarID string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	if entryID, exists := s.jobs[calendarID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, calendarID)
		log.Printf("Unscheduled calendar %s", calendarID)
	}
}

// TriggerSync manually triggers an immediate sync for a calendar.
func (s *Scheduler) TriggerSync(calendarID string) {
	go func() {
		ctx := context.Background()
		cal, err := s.calendarRepo.GetByID(ctx, calendarID)
		if err != nil || cal == nil {
			log.Printf("Calendar not found for sync: %s", calendarID)
			return
		}
		s.syncCalendar(cal.ID, cal.Name)
	}()
}

// syncCalendar performs the actual sync operation.
func (s *Scheduler) syncCalendar(calendarID, calendarName string) {
	ctx := context.Background()
	log.Printf("Syncing calendar: %s (%s)", calendarID, calendarName)

	result, err := s.syncService.SyncCalendar(ctx, calendarID)
	if err != nil {
		log.Printf("Calendar sync failed for %s: %v", calendarID, err)
		if s.broadcaster != nil {
			s.broadcaster.BroadcastCalendarSyncError(calendarID, calendarName, err)
		}
		return
	}

	log.Printf("Calendar sync completed for %s: %d events, %d PINs created, %d updated, %d removed",
		calendarID, result.EventsFound, result.PINsCreated, result.PINsUpdated, result.PINsRemoved)

	if s.broadcaster != nil {
		s.broadcaster.BroadcastCalendarSyncCompleted(*result)
	}
}

// updatePINStatuses checks and updates PIN activation/expiration.
func (s *Scheduler) updatePINStatuses() {
	ctx := context.Background()
	if err := s.syncService.UpdatePINStatuses(ctx); err != nil {
		log.Printf("Failed to update PIN statuses: %v", err)
	}
}

// refreshSchedules reloads calendar schedules from the database.
func (s *Scheduler) refreshSchedules(ctx context.Context) {
	calendars, err := s.calendarRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("Failed to refresh calendar schedules: %v", err)
		return
	}

	// Build set of current calendar IDs
	currentIDs := make(map[string]bool)
	for _, cal := range calendars {
		currentIDs[cal.ID] = true
		s.ScheduleCalendar(cal)
	}

	// Remove jobs for calendars that no longer exist or are disabled
	s.jobsMu.Lock()
	for calID := range s.jobs {
		if !currentIDs[calID] {
			s.cron.Remove(s.jobs[calID])
			delete(s.jobs, calID)
			log.Printf("Removed schedule for calendar %s (no longer enabled)", calID)
		}
	}
	s.jobsMu.Unlock()
}

// minutesToCronSpec converts minutes to a cron spec.
func minutesToCronSpec(minutes int) string {
	if minutes <= 0 {
		minutes = 15
	}

	// Use @every syntax for the interval
	duration := time.Duration(minutes) * time.Minute
	return "@every " + duration.String()
}

// GetScheduledCalendars returns a list of currently scheduled calendar IDs.
func (s *Scheduler) GetScheduledCalendars() []string {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	return ids
}

// GetNextRun returns the next scheduled run time for a calendar.
func (s *Scheduler) GetNextRun(calendarID string) *time.Time {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	if entryID, exists := s.jobs[calendarID]; exists {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			return &entry.Next
		}
	}
	return nil
}

