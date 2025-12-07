package models

import (
	"time"
)

// GuestPIN represents a PIN code generated for a guest from a calendar event.
type GuestPIN struct {
	ID                   string     `json:"id"`
	CalendarID           string     `json:"calendar_id"`
	EventUID             string     `json:"event_uid"`
	EventSummary         *string    `json:"event_summary,omitempty"`
	PINCode              string     `json:"pin_code"`
	GenerationMethod     string     `json:"generation_method"`
	CustomPIN            *string    `json:"custom_pin,omitempty"`
	ValidFrom            time.Time  `json:"valid_from"`
	ValidUntil           time.Time  `json:"valid_until"`
	Status               string     `json:"status"`
	RegenerationEligible bool       `json:"regeneration_eligible"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// PIN generation method constants - priority order (highest first)
const (
	GenerationMethodCustom           = "custom"            // Owner-specified custom PIN
	GenerationMethodPhoneLast4       = "phone_last4"       // Last 4 digits from phone pattern
	GenerationMethodDescriptionRandom = "description_random" // Deterministic from description
	GenerationMethodDateBased        = "date_based"        // Check-in + check-out days
)

// PIN status constants
const (
	PINStatusPending  = "pending"  // Not yet active
	PINStatusActive   = "active"   // Currently valid
	PINStatusExpired  = "expired"  // Past validity window
	PINStatusConflict = "conflict" // Conflicts with another PIN
)

// GuestPINLock represents the M:N relationship between guest PINs and locks,
// with sync status tracking.
type GuestPINLock struct {
	GuestPINID   string     `json:"guest_pin_id"`
	LockID       string     `json:"lock_id"`
	SlotNumber   int        `json:"slot_number"`
	SyncStatus   string     `json:"sync_status"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

// Lock sync status constants
const (
	LockSyncPending = "pending"
	LockSyncSynced  = "synced"
	LockSyncFailed  = "failed"
	LockSyncRemoved = "removed"
)

// GuestPINWithLocks combines a guest PIN with its lock assignments.
type GuestPINWithLocks struct {
	GuestPIN
	Locks []GuestPINLock `json:"locks"`
}

// IsActive returns true if the PIN is currently within its validity window.
func (p *GuestPIN) IsActive(now time.Time) bool {
	return now.After(p.ValidFrom) && now.Before(p.ValidUntil)
}

// CanRegenerate returns true if the PIN can be regenerated.
// PINs can only be regenerated if the start date is at least 1 day in the future.
func (p *GuestPIN) CanRegenerate(now time.Time) bool {
	if !p.RegenerationEligible {
		return false
	}
	// Must be at least 1 day before valid_from
	return p.ValidFrom.Sub(now) >= 24*time.Hour
}

