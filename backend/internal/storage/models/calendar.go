// Package models contains the domain models for the application.
package models

import (
	"time"
)

// CalendarSubscription represents a rental calendar subscription (iCal feed).
type CalendarSubscription struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	SyncIntervalMin int        `json:"sync_interval_min"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus      string     `json:"sync_status"`
	SyncError       *string    `json:"sync_error,omitempty"`
	Enabled         bool       `json:"enabled"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// SyncStatus constants
const (
	SyncStatusPending = "pending"
	SyncStatusSyncing = "syncing"
	SyncStatusSuccess = "success"
	SyncStatusError   = "error"
)

// CalendarLockMapping represents the M:N relationship between calendars and locks.
type CalendarLockMapping struct {
	CalendarID string `json:"calendar_id"`
	LockID     string `json:"lock_id"`
}

// CalendarEvent represents a parsed event from an iCal feed.
type CalendarEvent struct {
	UID         string    `json:"uid"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Location    string    `json:"location,omitempty"`
}

// CalendarSyncResult contains the results of a calendar sync operation.
type CalendarSyncResult struct {
	CalendarID   string    `json:"calendar_id"`
	CalendarName string    `json:"calendar_name"`
	EventsFound  int       `json:"events_found"`
	PINsCreated  int       `json:"pins_created"`
	PINsUpdated  int       `json:"pins_updated"`
	PINsRemoved  int       `json:"pins_removed"`
	Error        error     `json:"-"`
	SyncedAt     time.Time `json:"synced_at"`
}



