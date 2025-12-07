// Package handlers provides HTTP request handlers for the API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/guest-lock-manager/backend/internal/lock"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status      string `json:"status"`
	HAConnected bool   `json:"ha_connected"`
	DBConnected bool   `json:"db_connected"`
}

// HealthCheck returns a handler that performs a health check.
func HealthCheck(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		dbConnected := db.Ping() == nil

		// Determine overall status
		status := "healthy"
		if !dbConnected {
			status = "degraded"
		}

		response := HealthResponse{
			Status:      status,
			HAConnected: true, // TODO: Implement actual HA connection check
			DBConnected: dbConnected,
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	}
}

// StatusResponse represents the system status response.
type StatusResponse struct {
	HAConnected          bool   `json:"ha_connected"`
	HAVersion            string `json:"ha_version,omitempty"`
	ZWaveJSUIAvailable   bool   `json:"zwave_js_ui_available"`
	Zigbee2MQTTAvailable bool   `json:"zigbee2mqtt_available"`
	CalendarsCount       int    `json:"calendars_count"`
	LocksCount           int    `json:"locks_count"`
	ActiveGuestPins      int    `json:"active_guest_pins"`
	ActiveStaticPins     int    `json:"active_static_pins"`
	NextSyncAt           string `json:"next_sync_at,omitempty"`
	PendingOperations    int    `json:"pending_operations"`
}

// Status returns a handler that provides system status information.
func Status(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zwaveAvailable := lock.IsZWaveJSUIAvailable(ctx)
		zigbeeAvailable := lock.IsZigbee2MQTTAvailable(ctx)

		// Count calendars
		var calendarsCount int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM calendar_subscriptions").Scan(&calendarsCount)

		// Count locks
		var locksCount int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM managed_locks").Scan(&locksCount)

		// Count active guest PINs
		var activeGuestPins int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM guest_pins WHERE status = 'active'").Scan(&activeGuestPins)

		// Count enabled static PINs
		var activeStaticPins int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM static_pins WHERE enabled = 1").Scan(&activeStaticPins)

		// Count pending operations
		var pendingOps int
		db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM guest_pin_locks WHERE sync_status = 'pending'
			UNION ALL
			SELECT COUNT(*) FROM static_pin_locks WHERE sync_status = 'pending'
		`).Scan(&pendingOps)

		response := StatusResponse{
			HAConnected:          true, // TODO: Implement actual HA connection check
			ZWaveJSUIAvailable:   zwaveAvailable,
			Zigbee2MQTTAvailable: zigbeeAvailable,
			CalendarsCount:       calendarsCount,
			LocksCount:           locksCount,
			ActiveGuestPins:      activeGuestPins,
			ActiveStaticPins:     activeStaticPins,
			PendingOperations:    pendingOps,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
