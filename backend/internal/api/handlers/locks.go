package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/lock"
	"github.com/guest-lock-manager/backend/internal/storage"
)

// LockResponse represents a lock in API responses.
type LockResponse struct {
	ID                string  `json:"id"`
	EntityID          string  `json:"entity_id"`
	Name              string  `json:"name"`
	Protocol          string  `json:"protocol"`
	TotalSlots        int     `json:"total_slots"`
	GuestSlots        int     `json:"guest_slots"`
	StaticSlots       int     `json:"static_slots"`
	Online            bool    `json:"online"`
	BatteryLevel      *int    `json:"battery_level,omitempty"`
	LastSeenAt        *string `json:"last_seen_at,omitempty"`
	DirectIntegration *string `json:"direct_integration,omitempty"`
}

// ListLocks returns all managed locks.
func ListLocks(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := db.QueryContext(ctx, `
			SELECT id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			       online, battery_level, last_seen_at, direct_integration
			FROM managed_locks ORDER BY name
		`)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query locks")
			return
		}
		defer rows.Close()

		var locks []LockResponse
		for rows.Next() {
			var l LockResponse
			if err := rows.Scan(&l.ID, &l.EntityID, &l.Name, &l.Protocol, &l.TotalSlots, &l.GuestSlots, &l.StaticSlots, &l.Online, &l.BatteryLevel, &l.LastSeenAt, &l.DirectIntegration); err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to scan lock")
				return
			}
			locks = append(locks, l)
		}

		if locks == nil {
			locks = []LockResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(locks)
	}
}

// DiscoverLocks finds and adds locks from Home Assistant.
func DiscoverLocks(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Initialize HA client
		config := lock.DefaultConfig()
		haClient := lock.NewHAClient(config)
		discovery := lock.NewDiscovery(haClient)

		// Discover locks
		discovered, err := discovery.DiscoverLocks(ctx)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to discover locks: "+err.Error())
			return
		}

		// Add new locks to database
		var added []LockResponse
		for _, d := range discovered {
			// Check if already exists
			var exists int
			if err := db.QueryRowContext(ctx, "SELECT 1 FROM managed_locks WHERE entity_id = ?", d.EntityID).Scan(&exists); err == nil && exists == 1 {
				continue
			}

			id := storage.GenerateID()
			_, err := db.ExecContext(ctx, `
				INSERT INTO managed_locks (id, entity_id, name, protocol, online, battery_level, direct_integration)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, id, d.EntityID, d.Name, d.Protocol, d.Online, d.BatteryLevel, d.DirectIntegration)

			if err != nil {
				continue
			}

			added = append(added, LockResponse{
				ID:                id,
				EntityID:          d.EntityID,
				Name:              d.Name,
				Protocol:          d.Protocol,
				TotalSlots:        10,
				GuestSlots:        5,
				StaticSlots:       5,
				Online:            d.Online,
				BatteryLevel:      d.BatteryLevel,
				DirectIntegration: d.DirectIntegration,
			})
		}

		if added == nil {
			added = []LockResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(added)
	}
}

// GetLock returns a single lock by ID.
func GetLock(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var l LockResponse
		err := db.QueryRowContext(ctx, `
			SELECT id, entity_id, name, protocol, total_slots, guest_slots, static_slots,
			       online, battery_level, last_seen_at, direct_integration
			FROM managed_locks WHERE id = ?
		`, id).Scan(&l.ID, &l.EntityID, &l.Name, &l.Protocol, &l.TotalSlots, &l.GuestSlots, &l.StaticSlots, &l.Online, &l.BatteryLevel, &l.LastSeenAt, &l.DirectIntegration)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Lock not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(l)
	}
}

// UpdateLock updates a lock configuration.
func UpdateLock(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var req struct {
			Name        string `json:"name"`
		TotalSlots  int    `json:"total_slots"`
			GuestSlots  int    `json:"guest_slots"`
			StaticSlots int    `json:"static_slots"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

	if req.TotalSlots <= 0 {
		middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "total_slots must be greater than zero")
		return
	}
	if req.GuestSlots < 0 || req.StaticSlots < 0 {
		middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "guest_slots and static_slots must be non-negative")
		return
	}
	if req.GuestSlots+req.StaticSlots > req.TotalSlots {
		middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "guest_slots + static_slots cannot exceed total_slots")
		return
	}

		result, err := db.ExecContext(ctx, `
			UPDATE managed_locks SET
			name = ?, total_slots = ?, guest_slots = ?, static_slots = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
	`, req.Name, req.TotalSlots, req.GuestSlots, req.StaticSlots, id)

		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update lock")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Lock not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteLock removes a lock from management.
func DeleteLock(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		result, err := db.ExecContext(ctx, "DELETE FROM managed_locks WHERE id = ?", id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to delete lock")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Lock not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetLockPins returns all PINs assigned to a lock.
func GetLockPins(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		// Get guest PINs
		guestRows, err := db.QueryContext(ctx, `
			SELECT gp.id, gp.event_summary, gp.pin_code, gp.status, gpl.slot_number
			FROM guest_pins gp
			JOIN guest_pin_locks gpl ON gp.id = gpl.guest_pin_id
			WHERE gpl.lock_id = ?
		`, id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query guest PINs")
			return
		}
		defer guestRows.Close()

		var pins []map[string]any
		for guestRows.Next() {
			var pin struct {
				ID      string
				Summary *string
				Code    string
				Status  string
				Slot    int
			}
			if err := guestRows.Scan(&pin.ID, &pin.Summary, &pin.Code, &pin.Status, &pin.Slot); err != nil {
				continue
			}
			pins = append(pins, map[string]any{
				"id":          pin.ID,
				"type":        "guest",
				"name":        pin.Summary,
				"pin_code":    pin.Code,
				"status":      pin.Status,
				"slot_number": pin.Slot,
			})
		}

		// Get static PINs
		staticRows, err := db.QueryContext(ctx, `
			SELECT sp.id, sp.name, sp.pin_code, sp.enabled, spl.slot_number
			FROM static_pins sp
			JOIN static_pin_locks spl ON sp.id = spl.static_pin_id
			WHERE spl.lock_id = ?
		`, id)
		if err == nil {
			defer staticRows.Close()
			for staticRows.Next() {
				var pin struct {
					ID      string
					Name    string
					Code    string
					Enabled bool
					Slot    int
				}
				if err := staticRows.Scan(&pin.ID, &pin.Name, &pin.Code, &pin.Enabled, &pin.Slot); err != nil {
					continue
				}
				status := "disabled"
				if pin.Enabled {
					status = "active"
				}
				pins = append(pins, map[string]any{
					"id":          pin.ID,
					"type":        "static",
					"name":        pin.Name,
					"pin_code":    pin.Code,
					"status":      status,
					"slot_number": pin.Slot,
				})
			}
		}

		if pins == nil {
			pins = []map[string]any{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pins)
	}
}
