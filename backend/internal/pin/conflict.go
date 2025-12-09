package pin

import (
	"context"
	"fmt"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// ConflictChecker detects PIN conflicts (same code, overlapping dates).
type ConflictChecker struct {
	// findConflicts is a function that queries for conflicting PINs
	findConflicts func(ctx context.Context, pinCode string, validFrom, validUntil string, excludeID string) ([]models.GuestPIN, error)
}

// NewConflictChecker creates a new conflict checker.
func NewConflictChecker(findFunc func(ctx context.Context, pinCode string, validFrom, validUntil string, excludeID string) ([]models.GuestPIN, error)) *ConflictChecker {
	return &ConflictChecker{
		findConflicts: findFunc,
	}
}

// Conflict represents a detected PIN conflict.
type Conflict struct {
	PINCode        string    `json:"pin_code"`
	ConflictingPIN string    `json:"conflicting_pin_id"`
	EventSummary   string    `json:"event_summary,omitempty"`
	OverlapStart   time.Time `json:"overlap_start"`
	OverlapEnd     time.Time `json:"overlap_end"`
}

// CheckConflicts checks if a PIN would conflict with existing PINs.
func (c *ConflictChecker) CheckConflicts(ctx context.Context, pinCode string, validFrom, validUntil time.Time, excludeID string) ([]Conflict, error) {
	conflicting, err := c.findConflicts(ctx, pinCode, validFrom.Format(time.RFC3339), validUntil.Format(time.RFC3339), excludeID)
	if err != nil {
		return nil, fmt.Errorf("checking conflicts: %w", err)
	}

	var conflicts []Conflict
	for _, pin := range conflicting {
		// Calculate overlap period
		overlapStart := validFrom
		if pin.ValidFrom.After(overlapStart) {
			overlapStart = pin.ValidFrom
		}

		overlapEnd := validUntil
		if pin.ValidUntil.Before(overlapEnd) {
			overlapEnd = pin.ValidUntil
		}

		summary := ""
		if pin.EventSummary != nil {
			summary = *pin.EventSummary
		}

		conflicts = append(conflicts, Conflict{
			PINCode:        pinCode,
			ConflictingPIN: pin.ID,
			EventSummary:   summary,
			OverlapStart:   overlapStart,
			OverlapEnd:     overlapEnd,
		})
	}

	return conflicts, nil
}

// HasConflict returns true if there are any conflicts.
func (c *ConflictChecker) HasConflict(ctx context.Context, pinCode string, validFrom, validUntil time.Time, excludeID string) (bool, error) {
	conflicts, err := c.CheckConflicts(ctx, pinCode, validFrom, validUntil, excludeID)
	if err != nil {
		return false, err
	}
	return len(conflicts) > 0, nil
}

// FindAlternativePIN tries to find a non-conflicting PIN by modifying the original.
// Returns empty string if no alternative can be found within reasonable attempts.
func (c *ConflictChecker) FindAlternativePIN(ctx context.Context, originalPIN string, validFrom, validUntil time.Time, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}

	pinLen := len(originalPIN)
	if pinLen < 4 {
		pinLen = 4
	}

	for i := 0; i < maxAttempts; i++ {
		// Modify the PIN by incrementing
		modified := incrementPIN(originalPIN, i+1)
		if len(modified) > pinLen {
			modified = modified[len(modified)-pinLen:]
		}

		hasConflict, err := c.HasConflict(ctx, modified, validFrom, validUntil, "")
		if err != nil {
			return "", err
		}

		if !hasConflict {
			return modified, nil
		}
	}

	return "", fmt.Errorf("could not find alternative PIN after %d attempts", maxAttempts)
}

// incrementPIN increments a PIN string by a value.
func incrementPIN(pin string, increment int) string {
	// Convert to number, add increment, convert back
	var num int
	for _, c := range pin {
		num = num*10 + int(c-'0')
	}
	num += increment

	// Format back to string with same length
	return fmt.Sprintf("%0*d", len(pin), num%pow10(len(pin)))
}


