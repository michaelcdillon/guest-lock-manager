// Package calendar provides iCal parsing and calendar sync functionality.
package calendar

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// Parser parses iCal/ICS calendar feeds.
type Parser struct {
	httpClient *http.Client
}

// NewParser creates a new iCal parser.
func NewParser() *Parser {
	return &Parser{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAndParse downloads and parses an iCal feed from a URL.
func (p *Parser) FetchAndParse(url string) ([]models.CalendarEvent, error) {
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar returned status %d", resp.StatusCode)
	}

	return p.Parse(resp.Body)
}

// Parse reads and parses iCal data from a reader.
func (p *Parser) Parse(r io.Reader) ([]models.CalendarEvent, error) {
	var events []models.CalendarEvent
	var currentEvent *models.CalendarEvent
	var currentField string
	var multilineValue strings.Builder

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Handle line continuation (lines starting with space or tab)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if currentField != "" {
				multilineValue.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, " "), "\t"))
			}
			continue
		}

		// Process previous multiline field
		if currentField != "" && currentEvent != nil {
			p.setEventField(currentEvent, currentField, multilineValue.String())
			currentField = ""
			multilineValue.Reset()
		}

		// Parse field:value
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		field := line[:colonIdx]
		value := line[colonIdx+1:]

		// Handle property parameters (e.g., DTSTART;VALUE=DATE:20231215)
		if semicolonIdx := strings.Index(field, ";"); semicolonIdx != -1 {
			field = field[:semicolonIdx]
		}

		switch field {
		case "BEGIN":
			if value == "VEVENT" {
				currentEvent = &models.CalendarEvent{}
			}
		case "END":
			if value == "VEVENT" && currentEvent != nil {
				// Only include events with valid dates
				if !currentEvent.Start.IsZero() && !currentEvent.End.IsZero() {
					events = append(events, *currentEvent)
				}
				currentEvent = nil
			}
		case "UID", "SUMMARY", "DESCRIPTION", "LOCATION", "DTSTART", "DTEND":
			if currentEvent != nil {
				currentField = field
				multilineValue.WriteString(value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading calendar: %w", err)
	}

	return events, nil
}

// setEventField sets a field on a CalendarEvent.
func (p *Parser) setEventField(event *models.CalendarEvent, field, value string) {
	// Unescape common iCal escape sequences
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\,", ",")
	value = strings.ReplaceAll(value, "\\;", ";")
	value = strings.ReplaceAll(value, "\\\\", "\\")

	switch field {
	case "UID":
		event.UID = value
	case "SUMMARY":
		event.Summary = value
	case "DESCRIPTION":
		event.Description = value
	case "LOCATION":
		event.Location = value
	case "DTSTART":
		event.Start = p.parseDateTime(value)
	case "DTEND":
		event.End = p.parseDateTime(value)
	}
}

// parseDateTime parses an iCal date/time value.
func (p *Parser) parseDateTime(value string) time.Time {
	// Common iCal date formats
	formats := []string{
		"20060102T150405Z",     // UTC datetime
		"20060102T150405",      // Local datetime
		"20060102",             // Date only
		"2006-01-02T15:04:05Z", // ISO 8601 with dashes
		"2006-01-02",           // ISO 8601 date
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t
		}
	}

	return time.Time{}
}

// FilterFutureEvents returns only events that haven't ended yet.
func FilterFutureEvents(events []models.CalendarEvent, now time.Time) []models.CalendarEvent {
	var future []models.CalendarEvent
	for _, e := range events {
		if e.End.After(now) {
			future = append(future, e)
		}
	}
	return future
}

// FilterByDateRange returns events that overlap with the given date range.
func FilterByDateRange(events []models.CalendarEvent, start, end time.Time) []models.CalendarEvent {
	var filtered []models.CalendarEvent
	for _, e := range events {
		// Event overlaps if it starts before range ends and ends after range starts
		if e.Start.Before(end) && e.End.After(start) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}



