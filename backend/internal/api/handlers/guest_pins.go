package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// GuestPinResponse represents a guest PIN in API responses.
type GuestPinResponse struct {
	ID                    string  `json:"id"`
	CalendarID            string  `json:"calendar_id"`
	EventUID              string  `json:"event_uid"`
	EventSummary          *string `json:"event_summary,omitempty"`
	PinCode               string  `json:"pin_code"`
	GenerationMethod      string  `json:"generation_method"`
	CustomPin             *string `json:"custom_pin,omitempty"`
	ValidFrom             string  `json:"valid_from"`
	ValidUntil            string  `json:"valid_until"`
	Status                string  `json:"status"`
	RegenerationEligible  bool    `json:"regeneration_eligible"`
}

// ListGuestPins returns all guest PINs with optional filtering.
func ListGuestPins(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		calendarID := r.URL.Query().Get("calendar_id")
		status := r.URL.Query().Get("status")

		query := `
			SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
			       custom_pin, valid_from, valid_until, status, regeneration_eligible
			FROM guest_pins WHERE 1=1
		`
		var args []any

		if calendarID != "" {
			query += " AND calendar_id = ?"
			args = append(args, calendarID)
		}
		if status != "" {
			query += " AND status = ?"
			args = append(args, status)
		}

		query += " ORDER BY valid_from DESC"

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query guest PINs")
			return
		}
		defer rows.Close()

		var pins []GuestPinResponse
		for rows.Next() {
			var p GuestPinResponse
			if err := rows.Scan(&p.ID, &p.CalendarID, &p.EventUID, &p.EventSummary, &p.PinCode,
				&p.GenerationMethod, &p.CustomPin, &p.ValidFrom, &p.ValidUntil, &p.Status, &p.RegenerationEligible); err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to scan guest PIN")
				return
			}
			pins = append(pins, p)
		}

		if pins == nil {
			pins = []GuestPinResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pins)
	}
}

// GetGuestPin returns a single guest PIN by ID.
func GetGuestPin(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var p GuestPinResponse
		err := db.QueryRowContext(ctx, `
			SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
			       custom_pin, valid_from, valid_until, status, regeneration_eligible
			FROM guest_pins WHERE id = ?
		`, id).Scan(&p.ID, &p.CalendarID, &p.EventUID, &p.EventSummary, &p.PinCode,
			&p.GenerationMethod, &p.CustomPin, &p.ValidFrom, &p.ValidUntil, &p.Status, &p.RegenerationEligible)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Guest PIN not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

// UpdateGuestPin updates a guest PIN (e.g., set custom PIN or status).
func UpdateGuestPin(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var req struct {
			CustomPin *string `json:"custom_pin"`
			Status    *string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		// If custom PIN is provided, update it
		if req.CustomPin != nil {
			_, err := db.ExecContext(ctx, `
				UPDATE guest_pins SET
					custom_pin = ?,
					pin_code = ?,
					generation_method = 'custom',
					updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, req.CustomPin, req.CustomPin, id)

			if err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update guest PIN")
				return
			}
		}

		// If status is provided, update it (for manual activation/deactivation)
		if req.Status != nil {
			validStatuses := map[string]bool{"pending": true, "active": true, "expired": true}
			if !validStatuses[*req.Status] {
				middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "Invalid status. Must be: pending, active, or expired")
				return
			}

			_, err := db.ExecContext(ctx, `
				UPDATE guest_pins SET
					status = ?,
					updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, req.Status, id)

			if err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update guest PIN status")
				return
			}

			// Broadcast status change
			if hub != nil {
				broadcaster := websocket.NewEventBroadcaster(hub)
				broadcaster.BroadcastPINStatusChanged(id, "guest", "manual", *req.Status, "")
			}
		}

		// Return updated PIN
		var p GuestPinResponse
		err := db.QueryRowContext(ctx, `
			SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
			       custom_pin, valid_from, valid_until, status, regeneration_eligible
			FROM guest_pins WHERE id = ?
		`, id).Scan(&p.ID, &p.CalendarID, &p.EventUID, &p.EventSummary, &p.PinCode,
			&p.GenerationMethod, &p.CustomPin, &p.ValidFrom, &p.ValidUntil, &p.Status, &p.RegenerationEligible)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Guest PIN not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

// RegenerateGuestPin regenerates a PIN using the next available method.
func RegenerateGuestPin(db *storage.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		// Check if regeneration is eligible
		var eligible bool
		err := db.QueryRowContext(ctx, `
			SELECT regeneration_eligible FROM guest_pins WHERE id = ?
		`, id).Scan(&eligible)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Guest PIN not found")
			return
		}

		if !eligible {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "PIN is not eligible for regeneration")
			return
		}

		// TODO: Implement actual PIN regeneration logic
		// For now, just mark as regenerated with a placeholder
		newPin := "0000" // Placeholder - should use PIN generator

		_, err = db.ExecContext(ctx, `
			UPDATE guest_pins SET
				pin_code = ?,
				generation_method = 'date_based',
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, newPin, id)

		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to regenerate PIN")
			return
		}

		// Return updated PIN
		var p GuestPinResponse
		db.QueryRowContext(ctx, `
			SELECT id, calendar_id, event_uid, event_summary, pin_code, generation_method,
			       custom_pin, valid_from, valid_until, status, regeneration_eligible
			FROM guest_pins WHERE id = ?
		`, id).Scan(&p.ID, &p.CalendarID, &p.EventUID, &p.EventSummary, &p.PinCode,
			&p.GenerationMethod, &p.CustomPin, &p.ValidFrom, &p.ValidUntil, &p.Status, &p.RegenerationEligible)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

