package calendar

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guest-lock-manager/backend/internal/pin"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// SyncService handles calendar synchronization and PIN generation.
type SyncService struct {
	db          *storage.DB
	calendarRepo *storage.CalendarRepository
	guestPINRepo *storage.GuestPINRepository
	lockRepo     *storage.LockRepository
	parser       *Parser
	generator    *pin.Generator
	checkinTime  string // Format: "15:04"
	checkoutTime string
}

// NewSyncService creates a new calendar sync service.
func NewSyncService(
	db *storage.DB,
	calendarRepo *storage.CalendarRepository,
	guestPINRepo *storage.GuestPINRepository,
	lockRepo *storage.LockRepository,
	checkinTime, checkoutTime string,
	minPIN, maxPIN int,
) *SyncService {
	return &SyncService{
		db:           db,
		calendarRepo: calendarRepo,
		guestPINRepo: guestPINRepo,
		lockRepo:     lockRepo,
		parser:       NewParser(),
		generator:    pin.NewGenerator(minPIN, maxPIN),
		checkinTime:  checkinTime,
		checkoutTime: checkoutTime,
	}
}

// SyncCalendar synchronizes a single calendar and returns the result.
func (s *SyncService) SyncCalendar(ctx context.Context, calendarID string) (*models.CalendarSyncResult, error) {
	// Get calendar details
	calendar, err := s.calendarRepo.GetByID(ctx, calendarID)
	if err != nil {
		return nil, fmt.Errorf("getting calendar: %w", err)
	}
	if calendar == nil {
		return nil, fmt.Errorf("calendar not found: %s", calendarID)
	}

	result := &models.CalendarSyncResult{
		CalendarID:   calendar.ID,
		CalendarName: calendar.Name,
		SyncedAt:     time.Now().UTC(),
	}

	// Update status to syncing
	if err := s.calendarRepo.UpdateSyncStatus(ctx, calendarID, models.SyncStatusSyncing, nil); err != nil {
		log.Printf("Failed to update sync status: %v", err)
	}

	// Fetch and parse the calendar
	events, err := s.parser.FetchAndParse(calendar.URL)
	if err != nil {
		errMsg := err.Error()
		s.calendarRepo.UpdateSyncStatus(ctx, calendarID, models.SyncStatusError, &errMsg)
		result.Error = err
		return result, err
	}

	result.EventsFound = len(events)

	// Filter to future events only
	now := time.Now().UTC()
	events = FilterFutureEvents(events, now)

	// Get locks assigned to this calendar
	lockIDs, err := s.calendarRepo.GetLockIDs(ctx, calendarID)
	if err != nil {
		log.Printf("Failed to get lock IDs: %v", err)
		lockIDs = []string{}
	}

	// Process each event
	for _, event := range events {
		created, updated, err := s.processEvent(ctx, calendar.ID, event, lockIDs)
		if err != nil {
			log.Printf("Error processing event %s: %v", event.UID, err)
			continue
		}
		if created {
			result.PINsCreated++
		} else if updated {
			result.PINsUpdated++
		}
	}

	// Mark expired PINs
	removed, err := s.markExpiredPINs(ctx, calendarID, events)
	if err != nil {
		log.Printf("Error marking expired PINs: %v", err)
	}
	result.PINsRemoved = removed

	// Update calendar status to success
	if err := s.calendarRepo.UpdateSyncStatus(ctx, calendarID, models.SyncStatusSuccess, nil); err != nil {
		log.Printf("Failed to update sync status: %v", err)
	}

	return result, nil
}

// processEvent processes a single calendar event, creating or updating the PIN.
func (s *SyncService) processEvent(ctx context.Context, calendarID string, event models.CalendarEvent, lockIDs []string) (created, updated bool, err error) {
	// Check if PIN already exists for this event
	existing, err := s.guestPINRepo.GetByEventUID(ctx, calendarID, event.UID)
	if err != nil {
		return false, false, fmt.Errorf("checking existing PIN: %w", err)
	}

	// Calculate validity window with check-in/check-out times
	validFrom := s.applyCheckinTime(event.Start)
	validUntil := s.applyCheckoutTime(event.End)

	if existing != nil {
		// Update existing PIN if dates changed
		if !existing.ValidFrom.Equal(validFrom) || !existing.ValidUntil.Equal(validUntil) {
			existing.ValidFrom = validFrom
			existing.ValidUntil = validUntil
			existing.EventSummary = &event.Summary

			// Regenerate PIN if using date-based method and dates changed
			if existing.GenerationMethod == models.GenerationMethodDateBased {
				result := s.generator.GenerateFromEvent(event, "")
				existing.PINCode = result.PINCode
			}

			if err := s.guestPINRepo.Update(ctx, existing); err != nil {
				return false, false, fmt.Errorf("updating PIN: %w", err)
			}
			return false, true, nil
		}
		return false, false, nil
	}

	// Generate new PIN
	result := s.generator.GenerateFromEvent(event, "")

	// Create new guest PIN
	guestPIN := &models.GuestPIN{
		CalendarID:           calendarID,
		EventUID:             event.UID,
		EventSummary:         &event.Summary,
		PINCode:              result.PINCode,
		GenerationMethod:     result.Method,
		ValidFrom:            validFrom,
		ValidUntil:           validUntil,
		Status:               models.PINStatusPending,
		RegenerationEligible: true,
	}

	// Check if PIN should be active now
	now := time.Now().UTC()
	if guestPIN.IsActive(now) {
		guestPIN.Status = models.PINStatusActive
	}

	if err := s.guestPINRepo.Create(ctx, guestPIN); err != nil {
		return false, false, fmt.Errorf("creating PIN: %w", err)
	}

	// Assign to locks
	slotNumber := 1 // TODO: Implement slot allocation
	for _, lockID := range lockIDs {
		if err := s.guestPINRepo.AssignToLock(ctx, guestPIN.ID, lockID, slotNumber); err != nil {
			log.Printf("Failed to assign PIN to lock %s: %v", lockID, err)
		}
		slotNumber++
	}

	return true, false, nil
}

// markExpiredPINs marks PINs as expired if they're no longer in the calendar.
func (s *SyncService) markExpiredPINs(ctx context.Context, calendarID string, currentEvents []models.CalendarEvent) (int, error) {
	// Get all PINs for this calendar
	pins, err := s.guestPINRepo.ListByCalendar(ctx, calendarID)
	if err != nil {
		return 0, fmt.Errorf("listing PINs: %w", err)
	}

	// Build set of current event UIDs
	currentUIDs := make(map[string]bool)
	for _, e := range currentEvents {
		currentUIDs[e.UID] = true
	}

	// Mark PINs as expired if event is no longer in calendar
	removed := 0
	for _, pin := range pins {
		if !currentUIDs[pin.EventUID] && pin.Status != models.PINStatusExpired {
			if err := s.guestPINRepo.UpdateStatus(ctx, pin.ID, models.PINStatusExpired); err != nil {
				log.Printf("Failed to expire PIN %s: %v", pin.ID, err)
				continue
			}
			removed++
		}
	}

	return removed, nil
}

// applyCheckinTime applies the check-in time to a date.
func (s *SyncService) applyCheckinTime(date time.Time) time.Time {
	hour, minute := parseTimeString(s.checkinTime, 15, 0)
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
}

// applyCheckoutTime applies the check-out time to a date.
func (s *SyncService) applyCheckoutTime(date time.Time) time.Time {
	hour, minute := parseTimeString(s.checkoutTime, 11, 0)
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
}

// parseTimeString parses a time string "HH:MM" and returns hour and minute.
func parseTimeString(s string, defaultHour, defaultMinute int) (int, int) {
	if len(s) < 5 {
		return defaultHour, defaultMinute
	}

	var hour, minute int
	_, err := fmt.Sscanf(s, "%d:%d", &hour, &minute)
	if err != nil {
		return defaultHour, defaultMinute
	}

	return hour, minute
}

// SyncAllEnabled synchronizes all enabled calendars.
func (s *SyncService) SyncAllEnabled(ctx context.Context) ([]models.CalendarSyncResult, error) {
	calendars, err := s.calendarRepo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing enabled calendars: %w", err)
	}

	var results []models.CalendarSyncResult
	for _, cal := range calendars {
		result, err := s.SyncCalendar(ctx, cal.ID)
		if err != nil {
			log.Printf("Error syncing calendar %s: %v", cal.ID, err)
			if result == nil {
				result = &models.CalendarSyncResult{
					CalendarID:   cal.ID,
					CalendarName: cal.Name,
					Error:        err,
					SyncedAt:     time.Now().UTC(),
				}
			}
		}
		results = append(results, *result)
	}

	return results, nil
}

// UpdatePINStatuses updates PIN statuses based on current time.
func (s *SyncService) UpdatePINStatuses(ctx context.Context) error {
	// Activate pending PINs that should now be active
	pending, err := s.guestPINRepo.ListPendingActivation(ctx)
	if err != nil {
		return fmt.Errorf("listing pending PINs: %w", err)
	}

	for _, pin := range pending {
		if err := s.guestPINRepo.UpdateStatus(ctx, pin.ID, models.PINStatusActive); err != nil {
			log.Printf("Failed to activate PIN %s: %v", pin.ID, err)
		}
	}

	// Expire active PINs that have passed their validity window
	expired, err := s.guestPINRepo.ListExpired(ctx)
	if err != nil {
		return fmt.Errorf("listing expired PINs: %w", err)
	}

	for _, pin := range expired {
		if err := s.guestPINRepo.UpdateStatus(ctx, pin.ID, models.PINStatusExpired); err != nil {
			log.Printf("Failed to expire PIN %s: %v", pin.ID, err)
		}
	}

	return nil
}

