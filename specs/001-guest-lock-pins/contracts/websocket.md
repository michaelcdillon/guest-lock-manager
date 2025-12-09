# WebSocket API Contract

**Feature**: 001-guest-lock-pins  
**Date**: 2025-12-07  
**Endpoint**: `ws://[host]/api/ws`

## Overview

The WebSocket API provides real-time updates for lock status, PIN changes, and 
system events. The frontend maintains a persistent connection to receive push 
notifications instead of polling the REST API.

## Connection

### Handshake

```
GET /api/ws HTTP/1.1
Upgrade: websocket
Connection: Upgrade
```

The server accepts the connection and begins sending events.

### Heartbeat

- **Server → Client**: `ping` frame every 30 seconds
- **Client → Server**: Must respond with `pong` within 10 seconds
- Connection closed if pong not received

### Reconnection

Client should implement exponential backoff reconnection:
- Initial delay: 1 second
- Max delay: 30 seconds
- Multiplier: 2x per attempt
- Reset on successful connection

## Message Format

All messages are JSON with a standard envelope:

```json
{
  "type": "event_type",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": { ... }
}
```

## Server → Client Events

### `lock.status_changed`

Sent when a lock's online status or battery level changes.

```json
{
  "type": "lock.status_changed",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "lock_id": "uuid",
    "entity_id": "lock.front_door",
    "online": true,
    "battery_level": 75,
    "last_seen_at": "2025-12-07T15:30:00Z"
  }
}
```

### `pin.status_changed`

Sent when a PIN's status changes (pending → active, active → expired, etc.).

```json
{
  "type": "pin.status_changed",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "pin_id": "uuid",
    "pin_type": "guest",
    "previous_status": "pending",
    "new_status": "active",
    "event_summary": "John Doe - Dec 15-18"
  }
}
```

### `pin.sync_status_changed`

Sent when a PIN's sync status to a specific lock changes.

```json
{
  "type": "pin.sync_status_changed",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "pin_id": "uuid",
    "pin_type": "guest",
    "lock_id": "uuid",
    "lock_name": "Front Door",
    "previous_status": "pending",
    "new_status": "synced",
    "slot_number": 3
  }
}
```

### `pin.conflict_detected`

Sent when a PIN collision is detected.

```json
{
  "type": "pin.conflict_detected",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "pin_id": "uuid",
    "conflicting_pin_id": "uuid",
    "pin_code": "1234",
    "overlap_start": "2025-12-15T15:00:00Z",
    "overlap_end": "2025-12-18T11:00:00Z",
    "suggested_action": "regenerate"
  }
}
```

### `calendar.sync_completed`

Sent when a calendar sync finishes (success or error).

```json
{
  "type": "calendar.sync_completed",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "calendar_id": "uuid",
    "calendar_name": "Airbnb Unit A",
    "status": "success",
    "events_found": 12,
    "pins_created": 2,
    "pins_updated": 1,
    "pins_removed": 0,
    "next_sync_at": "2025-12-07T15:45:00Z"
  }
}
```

### `calendar.sync_error`

Sent when a calendar sync fails.

```json
{
  "type": "calendar.sync_error",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "calendar_id": "uuid",
    "calendar_name": "Airbnb Unit A",
    "error": "fetch_failed",
    "message": "Connection timeout after 30s",
    "retry_at": "2025-12-07T15:35:00Z"
  }
}
```

### `system.status_changed`

Sent when overall system status changes.

```json
{
  "type": "system.status_changed",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "ha_connected": true,
    "zwave_js_ui_available": true,
    "zigbee2mqtt_available": false,
    "pending_operations": 3
  }
}
```

### `notification`

Sent for user-facing notifications (errors, warnings, info).

```json
{
  "type": "notification",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "level": "warning",
    "title": "Lock Capacity Warning",
    "message": "Front Door has only 1 slot remaining",
    "action": {
      "type": "link",
      "label": "Manage Lock",
      "url": "/locks/uuid"
    },
    "dismissible": true
  }
}
```

**Notification Levels**: `info`, `warning`, `error`, `success`

## Client → Server Commands

The client can send commands to request specific actions or subscribe to 
filtered events.

### `subscribe`

Subscribe to specific event types (default: all events).

```json
{
  "type": "subscribe",
  "payload": {
    "events": ["lock.status_changed", "pin.status_changed"],
    "lock_ids": ["uuid1", "uuid2"],
    "calendar_ids": ["uuid3"]
  }
}
```

**Response**:

```json
{
  "type": "subscribe.ack",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "subscribed_events": ["lock.status_changed", "pin.status_changed"],
    "filtered_locks": 2,
    "filtered_calendars": 1
  }
}
```

### `unsubscribe`

Remove event subscriptions.

```json
{
  "type": "unsubscribe",
  "payload": {
    "events": ["lock.status_changed"]
  }
}
```

### `ping`

Client-initiated ping (in addition to protocol-level ping/pong).

```json
{
  "type": "ping",
  "payload": {
    "client_time": "2025-12-07T15:30:00Z"
  }
}
```

**Response**:

```json
{
  "type": "pong",
  "timestamp": "2025-12-07T15:30:00.050Z",
  "payload": {
    "client_time": "2025-12-07T15:30:00Z",
    "server_time": "2025-12-07T15:30:00.050Z"
  }
}
```

## Error Handling

### Error Response Format

```json
{
  "type": "error",
  "timestamp": "2025-12-07T15:30:00Z",
  "payload": {
    "code": "invalid_command",
    "message": "Unknown command type: foobar",
    "original_type": "foobar"
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `invalid_command` | Unknown command type |
| `invalid_payload` | Malformed JSON or missing required fields |
| `subscription_failed` | Could not subscribe to requested events |
| `rate_limited` | Too many commands in short period |

## Implementation Notes

### Backend (Go)

- Use `gorilla/websocket` Upgrader with origin check
- Maintain hub pattern for broadcasting to multiple clients
- Buffer outgoing messages to handle slow clients
- Close connections cleanly on shutdown

### Frontend (TypeScript)

- Single WebSocket connection per session
- Queue messages during reconnection
- Deduplicate rapid updates (debounce UI refreshes)
- Show connection status indicator

### Message Ordering

- Events are sent in order of occurrence
- No guaranteed delivery (client should re-sync via REST on reconnect)
- Large payloads are chunked if exceeding 64KB

### Rate Limits

- Max 10 client commands per second
- Max 100 events per second per client (events are dropped if exceeded)


