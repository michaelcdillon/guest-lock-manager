package websocket

import (
	"log"
	"time"

	"github.com/guest-lock-manager/backend/internal/storage/models"
)

// EventBroadcaster handles broadcasting WebSocket events.
type EventBroadcaster struct {
	hub *Hub
}

// NewEventBroadcaster creates a new event broadcaster.
func NewEventBroadcaster(hub *Hub) *EventBroadcaster {
	return &EventBroadcaster{hub: hub}
}

// BroadcastCalendarSyncCompleted sends a calendar sync completed event.
func (b *EventBroadcaster) BroadcastCalendarSyncCompleted(result models.CalendarSyncResult) {
	payload := CalendarSyncPayload{
		CalendarID:   result.CalendarID,
		CalendarName: result.CalendarName,
		Status:       "success",
		EventsFound:  result.EventsFound,
		PinsCreated:  result.PINsCreated,
		PinsUpdated:  result.PINsUpdated,
		PinsRemoved:  result.PINsRemoved,
	}

	if result.Error != nil {
		payload.Status = "error"
	}

	msg := NewMessage(TypeCalendarSyncCompleted, payload)
	b.broadcast(msg)
}

// BroadcastCalendarSyncError sends a calendar sync error event.
func (b *EventBroadcaster) BroadcastCalendarSyncError(calendarID, calendarName string, err error) {
	payload := CalendarSyncErrorPayload{
		CalendarID:   calendarID,
		CalendarName: calendarName,
		Error:        "sync_error",
		Message:      err.Error(),
	}

	msg := NewMessage(TypeCalendarSyncError, payload)
	b.broadcast(msg)
}

// BroadcastPINStatusChanged sends a PIN status changed event.
func (b *EventBroadcaster) BroadcastPINStatusChanged(pinID, pinType, previousStatus, newStatus string, eventSummary string) {
	payload := PinStatusPayload{
		PinID:          pinID,
		PinType:        pinType,
		PreviousStatus: previousStatus,
		NewStatus:      newStatus,
		EventSummary:   eventSummary,
	}

	msg := NewMessage(TypePinStatusChanged, payload)
	b.broadcast(msg)
}

// BroadcastPINSyncStatusChanged sends a PIN sync status changed event.
func (b *EventBroadcaster) BroadcastPINSyncStatusChanged(pinID, pinType, lockID, lockName, previousStatus, newStatus string, slotNumber int) {
	payload := PinSyncStatusPayload{
		PinID:          pinID,
		PinType:        pinType,
		LockID:         lockID,
		LockName:       lockName,
		PreviousStatus: previousStatus,
		NewStatus:      newStatus,
		SlotNumber:     slotNumber,
	}

	msg := NewMessage(TypePinSyncStatusChanged, payload)
	b.broadcast(msg)
}

// BroadcastLockStatusChanged sends a lock status changed event.
func (b *EventBroadcaster) BroadcastLockStatusChanged(lockID, entityID string, online bool, batteryLevel *int) {
	payload := LockStatusPayload{
		LockID:       lockID,
		EntityID:     entityID,
		Online:       online,
		BatteryLevel: batteryLevel,
		LastSeenAt:   time.Now().UTC(),
	}

	msg := NewMessage(TypeLockStatusChanged, payload)
	b.broadcast(msg)
}

// BroadcastNotification sends a notification to all connected clients.
func (b *EventBroadcaster) BroadcastNotification(level, title, message string) {
	payload := NotificationPayload{
		Level:       level,
		Title:       title,
		Message:     message,
		Dismissible: true,
	}

	msg := NewMessage(TypeNotification, payload)
	b.broadcast(msg)
}

// BroadcastSystemStatusChanged sends a system status change event.
func (b *EventBroadcaster) BroadcastSystemStatusChanged(status map[string]interface{}) {
	msg := NewMessage(TypeSystemStatusChanged, status)
	b.broadcast(msg)
}

// broadcast sends a message to all connected clients.
func (b *EventBroadcaster) broadcast(msg Message) {
	data, err := msg.JSON()
	if err != nil {
		log.Printf("Error encoding WebSocket message: %v", err)
		return
	}

	b.hub.Broadcast(data)
}


