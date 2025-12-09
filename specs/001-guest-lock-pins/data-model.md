# Data Model: Guest Lock PIN Manager

**Feature**: 001-guest-lock-pins  
**Date**: 2025-12-07  
**Storage**: SQLite 3

## Entity Relationship Diagram

```
┌─────────────────────┐       ┌─────────────────────┐
│  CalendarSubscription│       │    ManagedLock      │
├─────────────────────┤       ├─────────────────────┤
│ id (PK)             │       │ id (PK)             │
│ name                │       │ entity_id           │
│ url                 │       │ name                │
│ sync_interval_min   │       │ protocol            │
│ last_sync_at        │       │ total_slots         │
│ sync_status         │       │ online              │
│ enabled             │       │ battery_level       │
│ created_at          │       │ last_seen_at        │
│ updated_at          │       │ created_at          │
└─────────┬───────────┘       │ updated_at          │
          │                   └──────────┬──────────┘
          │ M:N                          │
          ▼                              │ 1:N
┌─────────────────────┐                  │
│ CalendarLockMapping │                  │
├─────────────────────┤                  │
│ calendar_id (FK)    │                  │
│ lock_id (FK)        │                  │
└─────────────────────┘                  │
                                         │
┌─────────────────────┐                  │
│      GuestPIN       │◄─────────────────┤
├─────────────────────┤                  │
│ id (PK)             │                  │
│ calendar_id (FK)    │                  │
│ event_uid           │                  │
│ event_summary       │                  │
│ pin_code            │                  │
│ generation_method   │                  │
│ valid_from          │                  │
│ valid_until         │                  │
│ status              │                  │
│ regeneration_eligible│                 │
│ created_at          │                  │
│ updated_at          │                  │
└─────────┬───────────┘                  │
          │                              │
          │ M:N                          │
          ▼                              │
┌─────────────────────┐                  │
│   GuestPINLock      │                  │
├─────────────────────┤                  │
│ guest_pin_id (FK)   │                  │
│ lock_id (FK)        │                  │
│ slot_number         │                  │
│ sync_status         │                  │
│ last_synced_at      │                  │
└─────────────────────┘                  │
                                         │
┌─────────────────────┐                  │
│     StaticPIN       │◄─────────────────┘
├─────────────────────┤
│ id (PK)             │
│ name                │
│ pin_code            │
│ enabled             │
│ always_active       │
│ created_at          │
│ updated_at          │
└─────────┬───────────┘
          │
          │ 1:N
          ▼
┌─────────────────────┐
│  StaticPINSchedule  │
├─────────────────────┤
│ id (PK)             │
│ static_pin_id (FK)  │
│ day_of_week         │
│ start_time          │
│ end_time            │
└─────────────────────┘
          │
          │ (StaticPIN → Lock is also M:N)
          ▼
┌─────────────────────┐
│   StaticPINLock     │
├─────────────────────┤
│ static_pin_id (FK)  │
│ lock_id (FK)        │
│ slot_number         │
│ sync_status         │
│ last_synced_at      │
└─────────────────────┘
```

## Entity Definitions

### CalendarSubscription

Represents a subscribed rental calendar (iCal feed).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PK, UUID | Unique identifier |
| name | TEXT | NOT NULL | Display name (e.g., "Airbnb Unit A") |
| url | TEXT | NOT NULL, UNIQUE | iCal feed URL |
| sync_interval_min | INTEGER | NOT NULL, DEFAULT 15 | Sync frequency in minutes |
| last_sync_at | DATETIME | NULL | Last successful sync timestamp |
| sync_status | TEXT | NOT NULL | 'pending', 'syncing', 'success', 'error' |
| sync_error | TEXT | NULL | Error message if sync_status='error' |
| enabled | BOOLEAN | NOT NULL, DEFAULT 1 | Whether calendar is active |
| created_at | DATETIME | NOT NULL | Creation timestamp |
| updated_at | DATETIME | NOT NULL | Last modification timestamp |

**Validation Rules**:
- URL must be valid HTTP/HTTPS URL
- sync_interval_min must be >= 5 (prevent excessive polling)
- name must be 1-100 characters

### ManagedLock

A lock device under addon management.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PK, UUID | Unique identifier |
| entity_id | TEXT | NOT NULL, UNIQUE | Home Assistant entity ID (e.g., lock.front_door) |
| name | TEXT | NOT NULL | Display name from HA or user-defined |
| protocol | TEXT | NOT NULL | 'zwave', 'zigbee', 'wifi', 'unknown' |
| total_slots | INTEGER | NOT NULL | Available PIN code slots |
| guest_slots | INTEGER | NOT NULL, DEFAULT 0 | Slots reserved for guest PINs |
| static_slots | INTEGER | NOT NULL, DEFAULT 0 | Slots reserved for static PINs |
| online | BOOLEAN | NOT NULL, DEFAULT 0 | Current online status |
| battery_level | INTEGER | NULL | Battery percentage (0-100), null if mains powered |
| last_seen_at | DATETIME | NULL | Last communication timestamp |
| direct_integration | TEXT | NULL | 'zwave_js_ui', 'zigbee2mqtt', null for HA-only |
| created_at | DATETIME | NOT NULL | Creation timestamp |
| updated_at | DATETIME | NOT NULL | Last modification timestamp |

**Validation Rules**:
- entity_id must match pattern `lock\.[a-z0-9_]+`
- total_slots must be >= 1
- guest_slots + static_slots <= total_slots
- battery_level must be 0-100 or null

**State Transitions**:
- `online`: Updated via HA WebSocket state changes
- `battery_level`: Updated on each lock communication

### GuestPIN

Auto-generated temporary PIN derived from calendar event.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PK, UUID | Unique identifier |
| calendar_id | TEXT | FK → CalendarSubscription | Source calendar |
| event_uid | TEXT | NOT NULL | iCal event UID for deduplication |
| event_summary | TEXT | NULL | Event title for display |
| pin_code | TEXT | NOT NULL | Generated PIN (4-8 digits) |
| generation_method | TEXT | NOT NULL | 'custom', 'phone_last4', 'description_random', 'date_based' |
| custom_pin | TEXT | NULL | Owner-specified override (highest priority) |
| valid_from | DATETIME | NOT NULL | Check-in time |
| valid_until | DATETIME | NOT NULL | Check-out time |
| status | TEXT | NOT NULL | 'pending', 'active', 'expired', 'conflict' |
| regeneration_eligible | BOOLEAN | NOT NULL | True if valid_from > now + 24h |
| created_at | DATETIME | NOT NULL | Creation timestamp |
| updated_at | DATETIME | NOT NULL | Last modification timestamp |

**Validation Rules**:
- pin_code must be 4-8 digits only
- valid_from must be before valid_until
- generation_method must be one of enum values
- Unique constraint on (calendar_id, event_uid)

**State Transitions**:
```
pending → active      (valid_from reached, synced to lock)
pending → conflict    (duplicate PIN detected)
active → expired      (valid_until reached, removed from lock)
conflict → pending    (PIN regenerated)
```

### StaticPIN

User-defined recurring PIN for service personnel.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PK, UUID | Unique identifier |
| name | TEXT | NOT NULL | Display name (e.g., "Cleaner - Maria") |
| pin_code | TEXT | NOT NULL | User-defined PIN (4-8 digits) |
| enabled | BOOLEAN | NOT NULL, DEFAULT 1 | Whether PIN is active |
| always_active | BOOLEAN | NOT NULL, DEFAULT 0 | True = no schedule restrictions |
| created_at | DATETIME | NOT NULL | Creation timestamp |
| updated_at | DATETIME | NOT NULL | Last modification timestamp |

**Validation Rules**:
- pin_code must be 4-8 digits only
- name must be 1-100 characters
- If always_active=false, must have at least one schedule entry

### StaticPINSchedule

Time restriction for a static PIN.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PK, UUID | Unique identifier |
| static_pin_id | TEXT | FK → StaticPIN | Parent static PIN |
| day_of_week | INTEGER | NOT NULL | 0=Sunday, 6=Saturday |
| start_time | TEXT | NOT NULL | Time in HH:MM format (24h) |
| end_time | TEXT | NOT NULL | Time in HH:MM format (24h) |

**Validation Rules**:
- day_of_week must be 0-6
- start_time and end_time must be valid HH:MM
- start_time must be before end_time (or handle overnight spans)

### Junction Tables

#### CalendarLockMapping
Maps calendars to locks (M:N).

| Field | Type | Constraints |
|-------|------|-------------|
| calendar_id | TEXT | FK → CalendarSubscription, PK |
| lock_id | TEXT | FK → ManagedLock, PK |

#### GuestPINLock
Maps guest PINs to locks with sync status.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| guest_pin_id | TEXT | FK → GuestPIN, PK | |
| lock_id | TEXT | FK → ManagedLock, PK | |
| slot_number | INTEGER | NOT NULL | Assigned slot on lock |
| sync_status | TEXT | NOT NULL | 'pending', 'synced', 'failed', 'removed' |
| last_synced_at | DATETIME | NULL | Last successful sync |
| error_message | TEXT | NULL | Error details if failed |

#### StaticPINLock
Maps static PINs to locks with sync status.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| static_pin_id | TEXT | FK → StaticPIN, PK | |
| lock_id | TEXT | FK → ManagedLock, PK | |
| slot_number | INTEGER | NOT NULL | Assigned slot on lock |
| sync_status | TEXT | NOT NULL | 'pending', 'synced', 'failed' |
| last_synced_at | DATETIME | NULL | Last successful sync |

## Indexes

```sql
-- Fast calendar sync lookups
CREATE INDEX idx_calendar_next_sync ON CalendarSubscription(enabled, last_sync_at);

-- Fast PIN validity checks
CREATE INDEX idx_guest_pin_validity ON GuestPIN(status, valid_from, valid_until);

-- Fast lock entity lookups
CREATE INDEX idx_lock_entity ON ManagedLock(entity_id);

-- Event deduplication
CREATE UNIQUE INDEX idx_guest_pin_event ON GuestPIN(calendar_id, event_uid);

-- Schedule lookups by day
CREATE INDEX idx_schedule_day ON StaticPINSchedule(day_of_week, start_time);
```

## Migration Strategy

All migrations stored in `backend/internal/storage/migrations/` as numbered SQL files:

```
001_initial_schema.sql
002_add_direct_integration.sql
...
```

Migration runner executes on startup, tracks applied migrations in `_migrations` table.


