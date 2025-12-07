// Package models contains the domain models for the application.
package models

import (
	"time"
)

// ManagedLock represents a lock device under addon management.
type ManagedLock struct {
	ID                string     `json:"id"`
	EntityID          string     `json:"entity_id"`
	Name              string     `json:"name"`
	Protocol          string     `json:"protocol"`
	TotalSlots        int        `json:"total_slots"`
	GuestSlots        int        `json:"guest_slots"`
	StaticSlots       int        `json:"static_slots"`
	Online            bool       `json:"online"`
	BatteryLevel      *int       `json:"battery_level,omitempty"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty"`
	DirectIntegration *string    `json:"direct_integration,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// AvailableSlots returns the number of slots not reserved for guest or static PINs.
func (l *ManagedLock) AvailableSlots() int {
	return l.TotalSlots - l.GuestSlots - l.StaticSlots
}

// LockSummary is a minimal lock representation for list views.
type LockSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

// LockProtocol represents the communication protocol for a lock.
type LockProtocol string

const (
	ProtocolZWave   LockProtocol = "zwave"
	ProtocolZigbee  LockProtocol = "zigbee"
	ProtocolWiFi    LockProtocol = "wifi"
	ProtocolUnknown LockProtocol = "unknown"
)

// DirectIntegrationType represents the type of direct protocol integration.
type DirectIntegrationType string

const (
	DirectZWaveJSUI   DirectIntegrationType = "zwave_js_ui"
	DirectZigbee2MQTT DirectIntegrationType = "zigbee2mqtt"
)

