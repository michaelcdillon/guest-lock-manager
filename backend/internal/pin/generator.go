// Package pin provides PIN generation and management functionality.
package pin

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// Generator generates PIN codes using various methods.
type Generator struct {
	minLength int
	maxLength int
}

// NewGenerator creates a new PIN generator.
func NewGenerator(minLength, maxLength int) *Generator {
	if minLength < 4 {
		minLength = 4
	}
	if maxLength < minLength {
		maxLength = minLength
	}
	if maxLength > 8 {
		maxLength = 8
	}

	return &Generator{
		minLength: minLength,
		maxLength: maxLength,
	}
}

// GenerationResult contains the generated PIN and metadata.
type GenerationResult struct {
	PINCode string
	Method  string
	Success bool
}

// GenerateFromEvent generates a PIN for a calendar event using the priority chain:
// 1. Custom PIN (if provided)
// 2. Phone Last-4 (if pattern found in description)
// 3. Description-based random
// 4. Date-based (fallback, always succeeds)
func (g *Generator) GenerateFromEvent(event models.CalendarEvent, customPIN string) GenerationResult {
	// 1. Custom PIN (highest priority)
	if customPIN != "" {
		if g.isValidPIN(customPIN) {
			return GenerationResult{
				PINCode: customPIN,
				Method:  models.GenerationMethodCustom,
				Success: true,
			}
		}
	}

	// 2. Phone Last-4 extraction
	if pin := g.extractPhoneLast4(event.Description); pin != "" {
		return GenerationResult{
			PINCode: pin,
			Method:  models.GenerationMethodPhoneLast4,
			Success: true,
		}
	}

	// 3. Description-based random (deterministic hash)
	if event.Description != "" {
		pin := g.generateFromDescription(event.Description, event.UID)
		return GenerationResult{
			PINCode: pin,
			Method:  models.GenerationMethodDescriptionRandom,
			Success: true,
		}
	}

	// 4. Date-based (fallback - always succeeds)
	pin := g.generateFromDates(event.Start, event.End)
	return GenerationResult{
		PINCode: pin,
		Method:  models.GenerationMethodDateBased,
		Success: true,
	}
}

// extractPhoneLast4 extracts the last 4 digits from a phone pattern in the description.
// Looks for patterns like "(Last 4 Digits): XXXX" or "Last 4 Digits: XXXX"
func (g *Generator) extractPhoneLast4(description string) string {
	// Pattern: (Last 4 Digits): XXXX or Last 4 Digits: XXXX
	patterns := []string{
		`\(Last 4 Digits\):\s*(\d{4})`,
		`Last 4 Digits:\s*(\d{4})`,
		`last 4 digits:\s*(\d{4})`,
		`\(last 4 digits\):\s*(\d{4})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(description); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// generateFromDescription generates a deterministic PIN from the event description.
// The PIN will change if the description changes.
func (g *Generator) generateFromDescription(description, uid string) string {
	// Create a hash from description + UID for uniqueness
	data := description + "|" + uid
	hash := sha256.Sum256([]byte(data))

	// Convert first bytes to a number and extract digits
	num := uint64(hash[0])<<24 | uint64(hash[1])<<16 | uint64(hash[2])<<8 | uint64(hash[3])

	// Generate PIN of desired length
	pin := fmt.Sprintf("%0*d", g.minLength, num%uint64(pow10(g.minLength)))

	return pin
}

// generateFromDates generates a PIN from check-in and check-out dates.
// Format: check-in day (2 digits) + check-out day (2 digits)
func (g *Generator) generateFromDates(checkIn, checkOut time.Time) string {
	inDay := checkIn.Day()
	outDay := checkOut.Day()

	// Basic format: DDDD (in-day + out-day)
	pin := fmt.Sprintf("%02d%02d", inDay, outDay)

	// If we need more digits, add month info
	if g.minLength > 4 {
		pin = fmt.Sprintf("%02d%02d%02d", checkIn.Month(), inDay, outDay)
	}

	// Ensure minimum length
	for len(pin) < g.minLength {
		pin = "0" + pin
	}

	// Truncate if longer than max (but don't try to extend)
	if len(pin) > g.maxLength {
		return pin[:g.maxLength]
	}

	return pin
}

// isValidPIN checks if a PIN meets the length requirements.
func (g *Generator) isValidPIN(pin string) bool {
	if len(pin) < g.minLength || len(pin) > g.maxLength {
		return false
	}

	// Must be all digits
	for _, c := range pin {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// ValidatePIN checks if a PIN is valid and returns an error message if not.
func (g *Generator) ValidatePIN(pin string) error {
	if len(pin) < g.minLength {
		return fmt.Errorf("PIN must be at least %d digits", g.minLength)
	}
	if len(pin) > g.maxLength {
		return fmt.Errorf("PIN must be at most %d digits", g.maxLength)
	}

	for _, c := range pin {
		if c < '0' || c > '9' {
			return fmt.Errorf("PIN must contain only digits")
		}
	}

	return nil
}

// RegeneratePIN generates a new PIN using the next available method after the current one.
func (g *Generator) RegeneratePIN(event models.CalendarEvent, currentMethod string) GenerationResult {
	switch currentMethod {
	case models.GenerationMethodCustom:
		// Try phone last-4 first
		if pin := g.extractPhoneLast4(event.Description); pin != "" {
			return GenerationResult{PINCode: pin, Method: models.GenerationMethodPhoneLast4, Success: true}
		}
		// Fall through to description-based
		fallthrough
	case models.GenerationMethodPhoneLast4:
		// Try description-based
		if event.Description != "" {
			pin := g.generateFromDescription(event.Description, event.UID)
			return GenerationResult{PINCode: pin, Method: models.GenerationMethodDescriptionRandom, Success: true}
		}
		// Fall through to date-based
		fallthrough
	case models.GenerationMethodDescriptionRandom:
		// Use date-based as final fallback
		pin := g.generateFromDates(event.Start, event.End)
		return GenerationResult{PINCode: pin, Method: models.GenerationMethodDateBased, Success: true}
	default:
		// Already at date-based, regenerate with slight variation
		pin := g.generateFromDates(event.Start, event.End)
		return GenerationResult{PINCode: pin, Method: models.GenerationMethodDateBased, Success: true}
	}
}

// pow10 returns 10^n.
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// ParsePINSettings parses PIN settings from string values.
func ParsePINSettings(minStr, maxStr string) (min, max int) {
	min = 4
	max = 8

	if v, err := strconv.Atoi(strings.TrimSpace(minStr)); err == nil && v >= 4 && v <= 8 {
		min = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(maxStr)); err == nil && v >= min && v <= 8 {
		max = v
	}

	return min, max
}

