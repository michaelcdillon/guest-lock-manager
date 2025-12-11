package websocket

import (
	"encoding/json"
	"time"
)

// MessageType identifies the type of WebSocket message.
type MessageType string

const (
	// Server -> Client event types
	TypeLockStatusChanged     MessageType = "lock.status_changed"
	TypePinStatusChanged      MessageType = "pin.status_changed"
	TypePinSyncStatusChanged  MessageType = "pin.sync_status_changed"
	TypePinConflictDetected   MessageType = "pin.conflict_detected"
	TypeCalendarSyncCompleted MessageType = "calendar.sync_completed"
	TypeCalendarSyncError     MessageType = "calendar.sync_error"
	TypeSystemStatusChanged   MessageType = "system.status_changed"
	TypeNotification          MessageType = "notification"

	// Client -> Server command types
	TypeSubscribe   MessageType = "subscribe"
	TypeUnsubscribe MessageType = "unsubscribe"
	TypePing        MessageType = "ping"

	// Server -> Client response types
	TypeSubscribeAck MessageType = "subscribe.ack"
	TypePong         MessageType = "pong"
	TypeError        MessageType = "error"
)

// Message represents a WebSocket message envelope.
type Message struct {
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   any         `json:"payload"`
}

// NewMessage creates a new message with the current timestamp.
func NewMessage(msgType MessageType, payload any) Message {
	return Message{
		Type:      msgType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

// JSON serializes the message to JSON bytes.
func (m Message) JSON() ([]byte, error) {
	return json.Marshal(m)
}

// LockStatusPayload is the payload for lock.status_changed events.
type LockStatusPayload struct {
	LockID       string    `json:"lock_id"`
	EntityID     string    `json:"entity_id"`
	Online       bool      `json:"online"`
	BatteryLevel *int      `json:"battery_level,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

// PinStatusPayload is the payload for pin.status_changed events.
type PinStatusPayload struct {
	PinID          string `json:"pin_id"`
	PinType        string `json:"pin_type"` // "guest" or "static"
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	EventSummary   string `json:"event_summary,omitempty"`
}

// PinSyncStatusPayload is the payload for pin.sync_status_changed events.
type PinSyncStatusPayload struct {
	PinID          string `json:"pin_id"`
	PinType        string `json:"pin_type"`
	LockID         string `json:"lock_id"`
	LockName       string `json:"lock_name"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	SlotNumber     int    `json:"slot_number"`
}

// CalendarSyncPayload is the payload for calendar.sync_completed events.
type CalendarSyncPayload struct {
	CalendarID   string    `json:"calendar_id"`
	CalendarName string    `json:"calendar_name"`
	Status       string    `json:"status"`
	EventsFound  int       `json:"events_found"`
	PinsCreated  int       `json:"pins_created"`
	PinsUpdated  int       `json:"pins_updated"`
	PinsRemoved  int       `json:"pins_removed"`
	NextSyncAt   time.Time `json:"next_sync_at,omitempty"`
}

// CalendarSyncErrorPayload is the payload for calendar.sync_error events.
type CalendarSyncErrorPayload struct {
	CalendarID   string    `json:"calendar_id"`
	CalendarName string    `json:"calendar_name"`
	Error        string    `json:"error"`
	Message      string    `json:"message"`
	RetryAt      time.Time `json:"retry_at,omitempty"`
}

// NotificationPayload is the payload for notification events.
type NotificationPayload struct {
	Level       string             `json:"level"` // info, warning, error, success
	Title       string             `json:"title"`
	Message     string             `json:"message"`
	Action      *NotificationAction `json:"action,omitempty"`
	Dismissible bool               `json:"dismissible"`
}

// NotificationAction is an optional action button for notifications.
type NotificationAction struct {
	Type  string `json:"type"` // "link"
	Label string `json:"label"`
	URL   string `json:"url"`
}

// ErrorPayload is the payload for error messages.
type ErrorPayload struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	OriginalType string `json:"original_type,omitempty"`
}



