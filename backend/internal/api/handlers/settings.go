package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/storage"
)

// SettingsResponse represents settings in API responses.
type SettingsResponse struct {
	DefaultSyncIntervalMin string `json:"default_sync_interval_min"`
	MinPinLength           string `json:"min_pin_length"`
	MaxPinLength           string `json:"max_pin_length"`
	CheckinTime            string `json:"checkin_time"`
	CheckoutTime           string `json:"checkout_time"`
	BatteryEfficientMode   string `json:"battery_efficient_mode"`
	BatchWindowSeconds     string `json:"batch_window_seconds"`
}

// GetSettings returns all settings.
func GetSettings(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := db.QueryContext(ctx, "SELECT key, value FROM settings")
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query settings")
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				continue
			}
			settings[key] = value
		}

		response := SettingsResponse{
			DefaultSyncIntervalMin: settings["default_sync_interval_min"],
			MinPinLength:           settings["min_pin_length"],
			MaxPinLength:           settings["max_pin_length"],
			CheckinTime:            settings["checkin_time"],
			CheckoutTime:           settings["checkout_time"],
			BatteryEfficientMode:   settings["battery_efficient_mode"],
			BatchWindowSeconds:     settings["batch_window_seconds"],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// UpdateSettings updates settings.
func UpdateSettings(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req SettingsResponse
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		// Update each setting
		settings := map[string]string{
			"default_sync_interval_min": req.DefaultSyncIntervalMin,
			"min_pin_length":            req.MinPinLength,
			"max_pin_length":            req.MaxPinLength,
			"checkin_time":              req.CheckinTime,
			"checkout_time":             req.CheckoutTime,
			"battery_efficient_mode":    req.BatteryEfficientMode,
			"batch_window_seconds":      req.BatchWindowSeconds,
		}

		for key, value := range settings {
			if value != "" {
				_, err := db.ExecContext(ctx, `
					INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
					ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
				`, key, value, value)
				if err != nil {
					middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update settings")
					return
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

