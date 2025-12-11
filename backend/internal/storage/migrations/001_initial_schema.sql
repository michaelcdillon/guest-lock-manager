-- Initial schema for Guest Lock PIN Manager
-- Creates all core tables for calendar subscriptions, locks, and PINs

-- Managed locks from Home Assistant
CREATE TABLE managed_locks (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'unknown',
    total_slots INTEGER NOT NULL DEFAULT 10,
    guest_slots INTEGER NOT NULL DEFAULT 5,
    static_slots INTEGER NOT NULL DEFAULT 5,
    online INTEGER NOT NULL DEFAULT 0,
    battery_level INTEGER,
    last_seen_at DATETIME,
    direct_integration TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (protocol IN ('zwave', 'zigbee', 'wifi', 'unknown')),
    CHECK (guest_slots + static_slots <= total_slots),
    CHECK (battery_level IS NULL OR (battery_level >= 0 AND battery_level <= 100))
);

-- Calendar subscriptions (iCal feeds)
CREATE TABLE calendar_subscriptions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    sync_interval_min INTEGER NOT NULL DEFAULT 15,
    last_sync_at DATETIME,
    sync_status TEXT NOT NULL DEFAULT 'pending',
    sync_error TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (sync_interval_min >= 5),
    CHECK (sync_status IN ('pending', 'syncing', 'success', 'error'))
);

-- Calendar to lock mapping (M:N)
CREATE TABLE calendar_lock_mappings (
    calendar_id TEXT NOT NULL,
    lock_id TEXT NOT NULL,
    PRIMARY KEY (calendar_id, lock_id),
    FOREIGN KEY (calendar_id) REFERENCES calendar_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY (lock_id) REFERENCES managed_locks(id) ON DELETE CASCADE
);

-- Guest PINs (auto-generated from calendar events)
CREATE TABLE guest_pins (
    id TEXT PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    event_uid TEXT NOT NULL,
    event_summary TEXT,
    pin_code TEXT NOT NULL,
    generation_method TEXT NOT NULL,
    custom_pin TEXT,
    valid_from DATETIME NOT NULL,
    valid_until DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    regeneration_eligible INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (calendar_id) REFERENCES calendar_subscriptions(id) ON DELETE CASCADE,
    UNIQUE (calendar_id, event_uid),
    CHECK (generation_method IN ('custom', 'phone_last4', 'description_random', 'date_based')),
    CHECK (status IN ('pending', 'active', 'expired', 'conflict')),
    CHECK (valid_from < valid_until)
);

-- Guest PIN to lock mapping with sync status
CREATE TABLE guest_pin_locks (
    guest_pin_id TEXT NOT NULL,
    lock_id TEXT NOT NULL,
    slot_number INTEGER NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'pending',
    last_synced_at DATETIME,
    error_message TEXT,
    PRIMARY KEY (guest_pin_id, lock_id),
    FOREIGN KEY (guest_pin_id) REFERENCES guest_pins(id) ON DELETE CASCADE,
    FOREIGN KEY (lock_id) REFERENCES managed_locks(id) ON DELETE CASCADE,
    CHECK (sync_status IN ('pending', 'synced', 'failed', 'removed'))
);

-- Static PINs (user-defined, recurring)
CREATE TABLE static_pins (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    pin_code TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    always_active INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Static PIN schedules (day/time restrictions)
CREATE TABLE static_pin_schedules (
    id TEXT PRIMARY KEY,
    static_pin_id TEXT NOT NULL,
    day_of_week INTEGER NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    FOREIGN KEY (static_pin_id) REFERENCES static_pins(id) ON DELETE CASCADE,
    CHECK (day_of_week >= 0 AND day_of_week <= 6)
);

-- Static PIN to lock mapping with sync status
CREATE TABLE static_pin_locks (
    static_pin_id TEXT NOT NULL,
    lock_id TEXT NOT NULL,
    slot_number INTEGER NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'pending',
    last_synced_at DATETIME,
    PRIMARY KEY (static_pin_id, lock_id),
    FOREIGN KEY (static_pin_id) REFERENCES static_pins(id) ON DELETE CASCADE,
    FOREIGN KEY (lock_id) REFERENCES managed_locks(id) ON DELETE CASCADE,
    CHECK (sync_status IN ('pending', 'synced', 'failed'))
);

-- Settings (key-value store for addon configuration)
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Default settings
INSERT INTO settings (key, value) VALUES
    ('default_sync_interval_min', '15'),
    ('min_pin_length', '4'),
    ('max_pin_length', '8'),
    ('checkin_time', '15:00'),
    ('checkout_time', '11:00'),
    ('battery_efficient_mode', 'true'),
    ('batch_window_seconds', '30');

-- Indexes for common queries
CREATE INDEX idx_calendar_next_sync ON calendar_subscriptions(enabled, last_sync_at);
CREATE INDEX idx_guest_pin_validity ON guest_pins(status, valid_from, valid_until);
CREATE INDEX idx_guest_pin_calendar ON guest_pins(calendar_id);
CREATE INDEX idx_lock_entity ON managed_locks(entity_id);
CREATE INDEX idx_schedule_day ON static_pin_schedules(day_of_week, start_time);



