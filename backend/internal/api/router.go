// Package api provides HTTP routing and handlers for the REST API.
package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guest-lock-manager/backend/internal/api/handlers"
	"github.com/guest-lock-manager/backend/internal/api/middleware"
	"github.com/guest-lock-manager/backend/internal/calendar"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// NewRouter creates and configures the HTTP router with all API routes.
// This is a convenience wrapper that creates a router without sync services.
func NewRouter(db *storage.DB, hub *websocket.Hub, staticDir string) *mux.Router {
	return NewRouterWithServices(db, hub, staticDir, nil, nil)
}

// NewRouterWithServices creates and configures the HTTP router with all API routes
// and injects the sync service and scheduler for calendar sync operations.
func NewRouterWithServices(
	db *storage.DB,
	hub *websocket.Hub,
	staticDir string,
	syncService *calendar.SyncService,
	calendarScheduler *calendar.Scheduler,
) *mux.Router {
	r := mux.NewRouter()

	// Apply global middleware
	r.Use(middleware.Logging)
	r.Use(middleware.ErrorRecovery)

	// API subrouter
	api := r.PathPrefix("/api").Subrouter()

	// Health and status endpoints
	api.HandleFunc("/health", handlers.HealthCheck(db)).Methods("GET")
	api.HandleFunc("/status", handlers.Status(db, hub)).Methods("GET")

	// WebSocket endpoint
	api.HandleFunc("/ws", handlers.WebSocketUpgrade(hub)).Methods("GET")

	// Calendar endpoints
	api.HandleFunc("/calendars", handlers.ListCalendars(db)).Methods("GET")
	api.HandleFunc("/calendars", handlers.CreateCalendar(db, calendarScheduler)).Methods("POST")
	api.HandleFunc("/calendars/{id}", handlers.GetCalendar(db)).Methods("GET")
	api.HandleFunc("/calendars/{id}", handlers.UpdateCalendar(db, calendarScheduler)).Methods("PUT")
	api.HandleFunc("/calendars/{id}", handlers.DeleteCalendar(db, calendarScheduler)).Methods("DELETE")
	api.HandleFunc("/calendars/{id}/sync", handlers.SyncCalendar(db, hub, syncService)).Methods("POST")
	api.HandleFunc("/calendars/{id}/locks", handlers.GetCalendarLocks(db)).Methods("GET")
	api.HandleFunc("/calendars/{id}/locks", handlers.UpdateCalendarLocks(db)).Methods("PUT")

	// Lock endpoints
	api.HandleFunc("/locks", handlers.ListLocks(db)).Methods("GET")
	api.HandleFunc("/locks/discover", handlers.DiscoverLocks(db)).Methods("POST")
	api.HandleFunc("/locks/{id}", handlers.GetLock(db)).Methods("GET")
	api.HandleFunc("/locks/{id}", handlers.UpdateLock(db)).Methods("PUT")
	api.HandleFunc("/locks/{id}", handlers.DeleteLock(db)).Methods("DELETE")
	api.HandleFunc("/locks/{id}/pins", handlers.GetLockPins(db)).Methods("GET")

	// Guest PIN endpoints
	api.HandleFunc("/guest-pins", handlers.ListGuestPins(db)).Methods("GET")
	api.HandleFunc("/guest-pins/{id}", handlers.GetGuestPin(db)).Methods("GET")
	api.HandleFunc("/guest-pins/{id}", handlers.UpdateGuestPin(db, hub)).Methods("PATCH")
	api.HandleFunc("/guest-pins/{id}/regenerate", handlers.RegenerateGuestPin(db, hub)).Methods("POST")

	// Static PIN endpoints
	api.HandleFunc("/static-pins", handlers.ListStaticPins(db)).Methods("GET")
	api.HandleFunc("/static-pins", handlers.CreateStaticPin(db, hub)).Methods("POST")
	api.HandleFunc("/static-pins/{id}", handlers.GetStaticPin(db)).Methods("GET")
	api.HandleFunc("/static-pins/{id}", handlers.UpdateStaticPin(db, hub)).Methods("PUT")
	api.HandleFunc("/static-pins/{id}", handlers.DeleteStaticPin(db, hub)).Methods("DELETE")

	// Settings endpoints
	api.HandleFunc("/settings", handlers.GetSettings(db)).Methods("GET")
	api.HandleFunc("/settings", handlers.UpdateSettings(db)).Methods("PUT")

	// Serve static frontend files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	return r
}

