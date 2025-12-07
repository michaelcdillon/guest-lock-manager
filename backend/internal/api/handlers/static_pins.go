package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// StaticPinResponse represents a static PIN in API responses.
type StaticPinResponse struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	PinCode      string        `json:"pin_code"`
	Enabled      bool          `json:"enabled"`
	AlwaysActive bool          `json:"always_active"`
	Schedules    []PinSchedule `json:"schedules,omitempty"`
}

// PinSchedule represents a day/time schedule.
type PinSchedule struct {
	ID        string `json:"id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// ListStaticPins returns all static PINs.
func ListStaticPins(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := db.QueryContext(ctx, `
			SELECT id, name, pin_code, enabled, always_active
			FROM static_pins ORDER BY name
		`)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query static PINs")
			return
		}
		defer rows.Close()

		var pins []StaticPinResponse
		for rows.Next() {
			var p StaticPinResponse
			if err := rows.Scan(&p.ID, &p.Name, &p.PinCode, &p.Enabled, &p.AlwaysActive); err != nil {
				continue
			}

			// Get schedules for this PIN
			scheduleRows, err := db.QueryContext(ctx, `
				SELECT id, day_of_week, start_time, end_time
				FROM static_pin_schedules WHERE static_pin_id = ?
			`, p.ID)
			if err == nil {
				for scheduleRows.Next() {
					var s PinSchedule
					if err := scheduleRows.Scan(&s.ID, &s.DayOfWeek, &s.StartTime, &s.EndTime); err != nil {
						continue
					}
					p.Schedules = append(p.Schedules, s)
				}
				scheduleRows.Close() // Close immediately, not defer
			}

			pins = append(pins, p)
		}

		if pins == nil {
			pins = []StaticPinResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pins)
	}
}

// CreateStaticPin creates a new static PIN.
func CreateStaticPin(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req struct {
			Name         string        `json:"name"`
			PinCode      string        `json:"pin_code"`
			Enabled      bool          `json:"enabled"`
			AlwaysActive bool          `json:"always_active"`
			Schedules    []PinSchedule `json:"schedules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" || req.PinCode == "" {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "Name and PIN code are required")
			return
		}

		// Check for duplicate name (case-insensitive)
		var existingCount int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM static_pins WHERE LOWER(name) = LOWER(?)
		`, req.Name).Scan(&existingCount)
		if err == nil && existingCount > 0 {
			middleware.WriteError(w, http.StatusConflict, middleware.ErrConflict, "A static PIN with this name already exists")
			return
		}

		id := storage.GenerateID()

		_, err = db.ExecContext(ctx, `
			INSERT INTO static_pins (id, name, pin_code, enabled, always_active)
			VALUES (?, ?, ?, ?, ?)
		`, id, req.Name, req.PinCode, req.Enabled, req.AlwaysActive)

		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to create static PIN")
			return
		}

		// Add schedules
		for _, s := range req.Schedules {
			scheduleID := storage.GenerateID()
			db.ExecContext(ctx, `
				INSERT INTO static_pin_schedules (id, static_pin_id, day_of_week, start_time, end_time)
				VALUES (?, ?, ?, ?, ?)
			`, scheduleID, id, s.DayOfWeek, s.StartTime, s.EndTime)
		}

		response := StaticPinResponse{
			ID:           id,
			Name:         req.Name,
			PinCode:      req.PinCode,
			Enabled:      req.Enabled,
			AlwaysActive: req.AlwaysActive,
			Schedules:    req.Schedules,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// GetStaticPin returns a single static PIN by ID.
func GetStaticPin(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var p StaticPinResponse
		err := db.QueryRowContext(ctx, `
			SELECT id, name, pin_code, enabled, always_active
			FROM static_pins WHERE id = ?
		`, id).Scan(&p.ID, &p.Name, &p.PinCode, &p.Enabled, &p.AlwaysActive)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Static PIN not found")
			return
		}

		// Get schedules
		scheduleRows, err := db.QueryContext(ctx, `
			SELECT id, day_of_week, start_time, end_time
			FROM static_pin_schedules WHERE static_pin_id = ?
		`, p.ID)
		if err == nil {
			defer scheduleRows.Close()
			for scheduleRows.Next() {
				var s PinSchedule
				if err := scheduleRows.Scan(&s.ID, &s.DayOfWeek, &s.StartTime, &s.EndTime); err != nil {
					continue
				}
				p.Schedules = append(p.Schedules, s)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

// UpdateStaticPin updates a static PIN.
func UpdateStaticPin(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var req struct {
			Name         *string       `json:"name"`
			PinCode      *string       `json:"pin_code"`
			Enabled      *bool         `json:"enabled"`
			AlwaysActive *bool         `json:"always_active"`
			Schedules    []PinSchedule `json:"schedules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		// Check for duplicate name if name is being updated (exclude current PIN)
		if req.Name != nil && *req.Name != "" {
			var existingCount int
			err := db.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM static_pins WHERE LOWER(name) = LOWER(?) AND id != ?
			`, *req.Name, id).Scan(&existingCount)
			if err == nil && existingCount > 0 {
				middleware.WriteError(w, http.StatusConflict, middleware.ErrConflict, "A static PIN with this name already exists")
				return
			}
		}

		// Build update query dynamically
		query := "UPDATE static_pins SET updated_at = CURRENT_TIMESTAMP"
		var args []any

		if req.Name != nil {
			query += ", name = ?"
			args = append(args, *req.Name)
		}
		if req.PinCode != nil {
			query += ", pin_code = ?"
			args = append(args, *req.PinCode)
		}
		if req.Enabled != nil {
			query += ", enabled = ?"
			args = append(args, *req.Enabled)
		}
		if req.AlwaysActive != nil {
			query += ", always_active = ?"
			args = append(args, *req.AlwaysActive)
		}

		query += " WHERE id = ?"
		args = append(args, id)

		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update static PIN")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Static PIN not found")
			return
		}

		// Update schedules if provided
		if req.Schedules != nil {
			// Delete existing schedules
			db.ExecContext(ctx, "DELETE FROM static_pin_schedules WHERE static_pin_id = ?", id)

			// Add new schedules
			for _, s := range req.Schedules {
				scheduleID := storage.GenerateID()
				db.ExecContext(ctx, `
					INSERT INTO static_pin_schedules (id, static_pin_id, day_of_week, start_time, end_time)
					VALUES (?, ?, ?, ?, ?)
				`, scheduleID, id, s.DayOfWeek, s.StartTime, s.EndTime)
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteStaticPin removes a static PIN.
func DeleteStaticPin(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		result, err := db.ExecContext(ctx, "DELETE FROM static_pins WHERE id = ?", id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to delete static PIN")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Static PIN not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

