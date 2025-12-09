// Package pin provides PIN generation and scheduling functionality.
package pin

import (
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// ScheduleEvaluator evaluates whether PINs should be active based on schedules.
type ScheduleEvaluator struct {
	location *time.Location
}

// NewScheduleEvaluator creates a new schedule evaluator.
// Uses local time zone by default.
func NewScheduleEvaluator() *ScheduleEvaluator {
	return &ScheduleEvaluator{
		location: time.Local,
	}
}

// NewScheduleEvaluatorWithLocation creates a schedule evaluator with a specific timezone.
func NewScheduleEvaluatorWithLocation(loc *time.Location) *ScheduleEvaluator {
	if loc == nil {
		loc = time.Local
	}
	return &ScheduleEvaluator{location: loc}
}

// IsStaticPINActive checks if a static PIN should be active at the given time.
func (e *ScheduleEvaluator) IsStaticPINActive(pin *models.StaticPINWithSchedules, at time.Time) bool {
	if pin == nil || !pin.Enabled {
		return false
	}

	// Always active PINs are always on
	if pin.AlwaysActive {
		return true
	}

	// No schedules means not active
	if len(pin.Schedules) == 0 {
		return false
	}

	// Convert to configured timezone
	localTime := at.In(e.location)
	weekday := int(localTime.Weekday())
	currentTime := localTime.Format("15:04")

	// Check each schedule
	for _, schedule := range pin.Schedules {
		if e.isScheduleActive(schedule, weekday, currentTime) {
			return true
		}
	}

	return false
}

// IsStaticPINActiveNow checks if a static PIN should be active right now.
func (e *ScheduleEvaluator) IsStaticPINActiveNow(pin *models.StaticPINWithSchedules) bool {
	return e.IsStaticPINActive(pin, time.Now())
}

// isScheduleActive checks if a single schedule is active for the given day/time.
func (e *ScheduleEvaluator) isScheduleActive(schedule models.StaticPINSchedule, weekday int, currentTime string) bool {
	if schedule.DayOfWeek != weekday {
		return false
	}

	// Handle overnight schedules (e.g., 22:00 to 06:00)
	if schedule.StartTime > schedule.EndTime {
		// Active if current time is after start OR before end
		return currentTime >= schedule.StartTime || currentTime <= schedule.EndTime
	}

	// Normal same-day schedule
	return currentTime >= schedule.StartTime && currentTime <= schedule.EndTime
}

// GetNextActiveWindow returns the next time the PIN will become active.
// Returns nil if the PIN is always active or has no schedules.
func (e *ScheduleEvaluator) GetNextActiveWindow(pin *models.StaticPINWithSchedules, from time.Time) *time.Time {
	if pin == nil || !pin.Enabled {
		return nil
	}

	if pin.AlwaysActive || len(pin.Schedules) == 0 {
		return nil
	}

	localFrom := from.In(e.location)
	
	// Check each day for the next 7 days
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		checkDate := localFrom.AddDate(0, 0, dayOffset)
		weekday := int(checkDate.Weekday())

		for _, schedule := range pin.Schedules {
			if schedule.DayOfWeek == weekday {
				// Parse start time
				startHour, startMin := parseTime(schedule.StartTime)
				
				// Create the next occurrence
				nextActive := time.Date(
					checkDate.Year(), checkDate.Month(), checkDate.Day(),
					startHour, startMin, 0, 0,
					e.location,
				)

				// If this is in the future, return it
				if nextActive.After(from) {
					return &nextActive
				}
			}
		}
	}

	return nil
}

// GetNextInactiveWindow returns the next time the PIN will become inactive.
// Returns nil if the PIN is always active or has no schedules.
func (e *ScheduleEvaluator) GetNextInactiveWindow(pin *models.StaticPINWithSchedules, from time.Time) *time.Time {
	if pin == nil || !pin.Enabled {
		return nil
	}

	if pin.AlwaysActive || len(pin.Schedules) == 0 {
		return nil
	}

	// If not currently active, return nil
	if !e.IsStaticPINActive(pin, from) {
		return nil
	}

	localFrom := from.In(e.location)
	weekday := int(localFrom.Weekday())
	currentTime := localFrom.Format("15:04")

	// Find the currently active schedule
	for _, schedule := range pin.Schedules {
		if schedule.DayOfWeek == weekday && currentTime >= schedule.StartTime && currentTime <= schedule.EndTime {
			// Found the active schedule, return its end time
			endHour, endMin := parseTime(schedule.EndTime)
			nextInactive := time.Date(
				localFrom.Year(), localFrom.Month(), localFrom.Day(),
				endHour, endMin, 0, 0,
				e.location,
			)
			return &nextInactive
		}
	}

	return nil
}

// parseTime parses a "15:04" formatted time string.
func parseTime(timeStr string) (hour, minute int) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, 0
	}
	return t.Hour(), t.Minute()
}

// ShouldSyncToLock determines if a static PIN should currently be synced to a lock.
// This considers the schedule - only active PINs should be on the lock.
func (e *ScheduleEvaluator) ShouldSyncToLock(pin *models.StaticPINWithSchedules) bool {
	return e.IsStaticPINActiveNow(pin)
}

// FilterActivePINs returns only the PINs that are currently active.
func (e *ScheduleEvaluator) FilterActivePINs(pins []models.StaticPINWithSchedules, at time.Time) []models.StaticPINWithSchedules {
	var active []models.StaticPINWithSchedules
	for _, pin := range pins {
		if e.IsStaticPINActive(&pin, at) {
			active = append(active, pin)
		}
	}
	return active
}

// FilterActivePINsNow returns only the PINs that are currently active.
func (e *ScheduleEvaluator) FilterActivePINsNow(pins []models.StaticPINWithSchedules) []models.StaticPINWithSchedules {
	return e.FilterActivePINs(pins, time.Now())
}


