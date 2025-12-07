package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/calendar"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/storage/models"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// Calendar request/response types

type CreateCalendarRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	SyncIntervalMin int    `json:"sync_interval_min"`
	Enabled         bool   `json:"enabled"`
}

type CalendarResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	URL             string  `json:"url"`
	SyncIntervalMin int     `json:"sync_interval_min"`
	LastSyncAt      *string `json:"last_sync_at,omitempty"`
	SyncStatus      string  `json:"sync_status"`
	SyncError       *string `json:"sync_error,omitempty"`
	Enabled         bool    `json:"enabled"`
}

// ListCalendars returns all calendar subscriptions.
func ListCalendars(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := db.QueryContext(ctx, `
			SELECT id, name, url, sync_interval_min, last_sync_at, sync_status, sync_error, enabled
			FROM calendar_subscriptions ORDER BY name
		`)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query calendars")
			return
		}
		defer rows.Close()

		var calendars []CalendarResponse
		for rows.Next() {
			var c CalendarResponse
			if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.SyncIntervalMin, &c.LastSyncAt, &c.SyncStatus, &c.SyncError, &c.Enabled); err != nil {
				middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to scan calendar")
				return
			}
			calendars = append(calendars, c)
		}

		if calendars == nil {
			calendars = []CalendarResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendars)
	}
}

// CreateCalendar adds a new calendar subscription.
func CreateCalendar(db *storage.DB, scheduler *calendar.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateCalendarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" || req.URL == "" {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrValidation, "Name and URL are required")
			return
		}

		if req.SyncIntervalMin < 5 {
			req.SyncIntervalMin = 15
		}

		id := storage.GenerateID()
		ctx := r.Context()

		_, err := db.ExecContext(ctx, `
			INSERT INTO calendar_subscriptions (id, name, url, sync_interval_min, enabled)
			VALUES (?, ?, ?, ?, ?)
		`, id, req.Name, req.URL, req.SyncIntervalMin, req.Enabled)

		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to create calendar")
			return
		}

		// Schedule the calendar for syncing if enabled
		if scheduler != nil && req.Enabled {
			scheduler.ScheduleCalendar(models.CalendarSubscription{
				ID:              id,
				Name:            req.Name,
				URL:             req.URL,
				SyncIntervalMin: req.SyncIntervalMin,
				Enabled:         req.Enabled,
			})
		}

		response := CalendarResponse{
			ID:              id,
			Name:            req.Name,
			URL:             req.URL,
			SyncIntervalMin: req.SyncIntervalMin,
			SyncStatus:      "pending",
			Enabled:         req.Enabled,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// GetCalendar returns a single calendar by ID.
func GetCalendar(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var c CalendarResponse
		err := db.QueryRowContext(ctx, `
			SELECT id, name, url, sync_interval_min, last_sync_at, sync_status, sync_error, enabled
			FROM calendar_subscriptions WHERE id = ?
		`, id).Scan(&c.ID, &c.Name, &c.URL, &c.SyncIntervalMin, &c.LastSyncAt, &c.SyncStatus, &c.SyncError, &c.Enabled)

		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Calendar not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
	}
}

// UpdateCalendar updates an existing calendar.
func UpdateCalendar(db *storage.DB, scheduler *calendar.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var req CreateCalendarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		result, err := db.ExecContext(ctx, `
			UPDATE calendar_subscriptions SET
				name = ?, url = ?, sync_interval_min = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, req.Name, req.URL, req.SyncIntervalMin, req.Enabled, id)

		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update calendar")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Calendar not found")
			return
		}

		// Update the scheduler with the new calendar settings
		if scheduler != nil {
			scheduler.ScheduleCalendar(models.CalendarSubscription{
				ID:              id,
				Name:            req.Name,
				URL:             req.URL,
				SyncIntervalMin: req.SyncIntervalMin,
				Enabled:         req.Enabled,
			})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteCalendar removes a calendar subscription.
func DeleteCalendar(db *storage.DB, scheduler *calendar.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		result, err := db.ExecContext(ctx, "DELETE FROM calendar_subscriptions WHERE id = ?", id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to delete calendar")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Calendar not found")
			return
		}

		// Unschedule the calendar
		if scheduler != nil {
			scheduler.UnscheduleCalendar(id)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SyncCalendar triggers a manual sync for a calendar.
func SyncCalendar(db *storage.DB, hub *websocket.Hub, syncService *calendar.SyncService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		// Verify calendar exists
		var calName string
		err := db.QueryRowContext(r.Context(), "SELECT name FROM calendar_subscriptions WHERE id = ?", id).Scan(&calName)
		if err != nil {
			middleware.WriteError(w, http.StatusNotFound, middleware.ErrNotFound, "Calendar not found")
			return
		}

		// Return immediately with syncing status
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "syncing"})

		// Trigger sync in background
		go func() {
			ctx := context.Background()

			if syncService != nil {
				// Use the sync service for full sync
				result, err := syncService.SyncCalendar(ctx, id)
				if err != nil {
					if hub != nil {
						broadcaster := websocket.NewEventBroadcaster(hub)
						broadcaster.BroadcastCalendarSyncError(id, calName, err)
					}
					return
				}

				// Broadcast completion event
				if hub != nil {
					broadcaster := websocket.NewEventBroadcaster(hub)
					broadcaster.BroadcastCalendarSyncCompleted(*result)
				}
			} else {
				// Fallback: just update status (no sync service available)
				db.ExecContext(ctx, `
					UPDATE calendar_subscriptions SET 
						sync_status = 'success', 
						last_sync_at = CURRENT_TIMESTAMP,
						updated_at = CURRENT_TIMESTAMP
					WHERE id = ?
				`, id)

				if hub != nil {
					broadcaster := websocket.NewEventBroadcaster(hub)
					broadcaster.BroadcastNotification("success", "Calendar Synced", "Calendar sync completed successfully")
				}
			}
		}()
	}
}

// GetCalendarLocks returns locks assigned to a calendar.
func GetCalendarLocks(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		rows, err := db.QueryContext(ctx, `
			SELECT lock_id FROM calendar_lock_mappings WHERE calendar_id = ?
		`, id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to query lock mappings")
			return
		}
		defer rows.Close()

		var lockIds []string
		for rows.Next() {
			var lockId string
			if err := rows.Scan(&lockId); err != nil {
				continue
			}
			lockIds = append(lockIds, lockId)
		}

		if lockIds == nil {
			lockIds = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lockIds)
	}
}

// UpdateCalendarLocks updates the locks assigned to a calendar.
func UpdateCalendarLocks(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		ctx := r.Context()

		var req struct {
			LockIDs []string `json:"lock_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, middleware.ErrBadRequest, "Invalid request body")
			return
		}

		// Delete existing mappings
		_, err := db.ExecContext(ctx, "DELETE FROM calendar_lock_mappings WHERE calendar_id = ?", id)
		if err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, middleware.ErrInternalError, "Failed to update lock mappings")
			return
		}

		// Insert new mappings
		for _, lockId := range req.LockIDs {
			_, err := db.ExecContext(ctx, `
				INSERT INTO calendar_lock_mappings (calendar_id, lock_id) VALUES (?, ?)
			`, id, lockId)
			if err != nil {
				continue // Skip duplicates
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
