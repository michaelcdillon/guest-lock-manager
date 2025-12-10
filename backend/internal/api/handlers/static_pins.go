package handlers

import (
	"database/sql"
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
	SlotNumber   int           `json:"slot_number,omitempty"`
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
			SELECT id, name, pin_code, enabled, always_active, slot_number
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
			if err := rows.Scan(&p.ID, &p.Name, &p.PinCode, &p.Enabled, &p.AlwaysActive, &p.SlotNumber); err != nil {
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

			// Slot number from first assignment (if any)
			var slot sql.NullInt64
			_ = db.QueryRowContext(ctx, `
				SELECT slot_number FROM static_pin_locks WHERE static_pin_id = ? LIMIT 1
			`, p.ID).Scan(&slot)
			if slot.Valid {
				p.SlotNumber = int(slot.Int64)
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
			SlotNumber   int           `json:"slot_number"`
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
		if req.SlotNumber <= 0 {
			req.SlotNumber = 1
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

		// Validate slot is free across existing static pin assignments
		var slotConflict int
		err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM static_pin_locks WHERE slot_number = ?
	`, req.SlotNumber).Scan(&slotConflict)
		if err == nil && slotConflict > 0 {
			middleware.WriteError(w, http.StatusConflict, middleware.ErrConflict, "Slot already in use by another static PIN")
			return
		}

		id := storage.GenerateID()

		_, err = db.ExecContext(ctx, `
			INSERT INTO static_pins (id, name, pin_code, enabled, always_active, slot_number)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, req.Name, req.PinCode, req.Enabled, req.AlwaysActive, req.SlotNumber)

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

		// Assign to all managed locks with the chosen slot number
		lockRows, err := db.QueryContext(ctx, `SELECT id FROM managed_locks`)
		if err == nil {
			for lockRows.Next() {
				var lockID string
				if err := lockRows.Scan(&lockID); err != nil {
					continue
				}
				db.ExecContext(ctx, `
					INSERT OR REPLACE INTO static_pin_locks (static_pin_id, lock_id, slot_number, sync_status)
					VALUES (?, ?, ?, 'pending')
				`, id, lockID, req.SlotNumber)
			}
			lockRows.Close()
		}

		response := StaticPinResponse{
			ID:           id,
			Name:         req.Name,
			PinCode:      req.PinCode,
			Enabled:      req.Enabled,
			AlwaysActive: req.AlwaysActive,
			SlotNumber:   req.SlotNumber,
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
			SELECT id, name, pin_code, enabled, always_active, slot_number
			FROM static_pins WHERE id = ?
		`, id).Scan(&p.ID, &p.Name, &p.PinCode, &p.Enabled, &p.AlwaysActive, &p.SlotNumber)

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

		// Slot number
		var slot sql.NullInt64
		_ = db.QueryRowContext(ctx, `
			SELECT slot_number FROM static_pin_locks WHERE static_pin_id = ? LIMIT 1
		`, p.ID).Scan(&slot)
		if slot.Valid {
			p.SlotNumber = int(slot.Int64)
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
			SlotNumber   *int          `json:"slot_number"`
			Schedules    []PinSchedule `json:"schedules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		// Load current slot number (for conflict detection and default)
		var currentSlot int
		if err := db.QueryRowContext(ctx, `SELECT slot_number FROM static_pins WHERE id = ?`, id).Scan(&currentSlot); err != nil {
			if err == sql.ErrNoRows {
				middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Static PIN not found")
				return
			}
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to load static PIN")
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

		// Determine desired slot and validate conflicts if changing
		newSlot := currentSlot
		if req.SlotNumber != nil {
			if *req.SlotNumber <= 0 {
				newSlot = 1
			} else {
				newSlot = *req.SlotNumber
			}

			var slotConflict int
			err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM static_pin_locks
			WHERE slot_number = ? AND static_pin_id != ?
		`, newSlot, id).Scan(&slotConflict)
			if err == nil && slotConflict > 0 {
				middleware.WriteError(w, http.StatusConflict, middleware.ErrConflict, "Slot already in use by another static PIN")
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
		if req.SlotNumber != nil {
			query += ", slot_number = ?"
			args = append(args, newSlot)
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

		// Update slot assignments if provided
		if req.SlotNumber != nil && newSlot > 0 {
			db.ExecContext(ctx, `
				UPDATE static_pin_locks
				SET slot_number = ?, sync_status = 'pending'
				WHERE static_pin_id = ?
		`, newSlot, id)
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
