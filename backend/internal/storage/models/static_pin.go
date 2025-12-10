// Package models defines data structures for storage entities.
package models

import "time"

// StaticPIN represents a user-defined, recurring PIN code.
type StaticPIN struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PINCode      string    `json:"pin_code"`
	Enabled      bool      `json:"enabled"`
	AlwaysActive bool      `json:"always_active"`
	SlotNumber   int       `json:"slot_number"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StaticPINSchedule represents a day/time window when a static PIN is active.
type StaticPINSchedule struct {
	ID          string `json:"id"`
	StaticPINID string `json:"static_pin_id"`
	DayOfWeek   int    `json:"day_of_week"` // 0 = Sunday, 6 = Saturday
	StartTime   string `json:"start_time"`  // Format: "15:04"
	EndTime     string `json:"end_time"`    // Format: "15:04"
}

// StaticPINLock represents the assignment of a static PIN to a lock.
type StaticPINLock struct {
	StaticPINID  string     `json:"static_pin_id"`
	LockID       string     `json:"lock_id"`
	SlotNumber   int        `json:"slot_number"`
	SyncStatus   string     `json:"sync_status"` // pending, synced, failed
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

// StaticPINWithSchedules combines a static PIN with its schedules.
type StaticPINWithSchedules struct {
	StaticPIN
	Schedules []StaticPINSchedule `json:"schedules"`
}

// StaticPINWithLocks combines a static PIN with its lock assignments.
type StaticPINWithLocks struct {
	StaticPINWithSchedules
	Locks []StaticPINLock `json:"locks"`
}

// Sync status constants for static PINs
const (
	StaticPINSyncPending = "pending"
	StaticPINSyncSynced  = "synced"
	StaticPINSyncFailed  = "failed"
)

// IsActiveAt checks if the static PIN should be active at the given time.
// If AlwaysActive is true, returns true regardless of schedules.
// If no schedules exist and not always active, returns false.
func (p *StaticPINWithSchedules) IsActiveAt(t time.Time) bool {
	if !p.Enabled {
		return false
	}
	if p.AlwaysActive {
		return true
	}
	if len(p.Schedules) == 0 {
		return false
	}

	weekday := int(t.Weekday())
	currentTime := t.Format("15:04")

	for _, schedule := range p.Schedules {
		if schedule.DayOfWeek == weekday {
			if currentTime >= schedule.StartTime && currentTime <= schedule.EndTime {
				return true
			}
		}
	}

	return false
}

// IsActiveNow checks if the static PIN should be active right now.
func (p *StaticPINWithSchedules) IsActiveNow() bool {
	return p.IsActiveAt(time.Now())
}
