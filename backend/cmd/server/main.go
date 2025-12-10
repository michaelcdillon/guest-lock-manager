// Package main is the entry point for the Guest Lock PIN Manager server.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guest-lock-manager/backend/internal/api"
	"github.com/guest-lock-manager/backend/internal/calendar"
	"github.com/guest-lock-manager/backend/internal/lock"
	"github.com/guest-lock-manager/backend/internal/pin"
	"github.com/guest-lock-manager/backend/internal/storage"
	"github.com/guest-lock-manager/backend/internal/websocket"
)

// loadSetting fetches a single setting value by key, returning an empty string if missing.
func loadSetting(ctx context.Context, db *storage.DB, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query setting %s: %w", key, err)
	}
	return value, nil
}

// addonOptions represents the subset of add-on options we care about.
type addonOptions struct {
	ZWaveJSUIWSURL string `json:"zwave_js_ui_ws_url"`
}

// loadAddonOptions loads options from /data/options.json when running as an add-on.
func loadAddonOptions() addonOptions {
	data, err := os.ReadFile("/data/options.json")
	if err != nil {
		return addonOptions{}
	}

	var opts addonOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return addonOptions{}
	}

	return opts
}

// version is set at build time via -ldflags "-X main.version=x.y.z".
// Defaults to "dev" when not provided.
var version = "dev"

func main() {
	// Parse command-line flags
	addr := flag.String("addr", ":8099", "HTTP server address")
	dataDir := flag.String("data", "/data", "Data directory for SQLite database")
	staticDir := flag.String("static", "./static", "Directory for static frontend files")
	healthCheck := flag.Bool("health-check", false, "Run health check and exit")
	flag.Parse()

	// Health check mode for Docker HEALTHCHECK
	if *healthCheck {
		if err := runHealthCheck(*addr); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
		os.Exit(0)
	}

	// Allow overriding version via environment (e.g., injected by container build/runtime)
	// Prefer VERSION, then BUILD_VERSION (provided in some addon environments).
	if envVer := os.Getenv("VERSION"); envVer != "" {
		version = envVer
	} else if buildVer := os.Getenv("BUILD_VERSION"); buildVer != "" {
		version = buildVer
	}

	log.Printf("Starting Guest Lock PIN Manager (version: %s)...", version)

	// Initialize database
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("Failed to create data directory %q: %v", *dataDir, err)
	}
	dbPath := *dataDir + "/guest-lock-manager.db"
	db, err := storage.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := storage.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations complete")

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize repositories
	calendarRepo := storage.NewCalendarRepository(db)
	guestPINRepo := storage.NewGuestPINRepository(db)
	lockRepo := storage.NewLockRepository(db)
	staticPINRepo := storage.NewStaticPINRepository(db)

	// Configure Z-Wave JS UI URL in priority order: add-on option -> persisted setting -> env/default.
	if opts := loadAddonOptions(); opts.ZWaveJSUIWSURL != "" {
		lock.SetZWaveJSUIURL(opts.ZWaveJSUIWSURL)
	} else if url, err := loadSetting(context.Background(), db, "zwave_js_ui_ws_url"); err == nil && url != "" {
		lock.SetZWaveJSUIURL(url)
	}

	// Log Z-Wave JS UI reachability at startup
	zwURL := lock.GetZWaveJSUIURL()
	if ok := lock.IsZWaveJSUIAvailable(context.Background()); ok {
		log.Printf("Z-Wave JS UI reachable at %s", zwURL)
	} else {
		log.Printf("Z-Wave JS UI NOT reachable at %s", zwURL)
	}

	// Initialize services with default settings
	// TODO: Load these from settings table
	checkinTime := "15:00"
	checkoutTime := "11:00"
	minPIN := 4
	maxPIN := 6
	batchWindowSeconds := 30
	defaultSyncIntervalMin := 15

	// Initialize sync service
	syncService := calendar.NewSyncService(
		db,
		calendarRepo,
		guestPINRepo,
		lockRepo,
		checkinTime, checkoutTime,
		minPIN, maxPIN,
	)

	// Initialize lock manager
	lockManager := lock.NewManager(db, lockRepo, guestPINRepo, batchWindowSeconds)

	// Initialize schedulers
	calendarScheduler := calendar.NewScheduler(
		syncService,
		calendarRepo,
		hub,
		defaultSyncIntervalMin,
	)

	guestPINScheduler := pin.NewStatusScheduler(guestPINRepo, lockManager, hub)
	staticPINScheduler := pin.NewStaticPINScheduler(staticPINRepo, lockManager, hub)

	// Start schedulers
	if err := calendarScheduler.Start(context.Background()); err != nil {
		log.Printf("Warning: Failed to start calendar scheduler: %v", err)
	}
	guestPINScheduler.Start()
	staticPINScheduler.Start()

	// Initialize static PIN states
	if err := staticPINScheduler.InitializeStates(context.Background()); err != nil {
		log.Printf("Warning: Failed to initialize static PIN states: %v", err)
	}

	// Initialize HTTP router with services
	router := api.NewRouterWithServices(db, hub, *staticDir, syncService, calendarScheduler)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("Server listening on %s", *addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop schedulers
	calendarScheduler.Stop()
	guestPINScheduler.Stop()
	staticPINScheduler.Stop()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// runHealthCheck performs a health check against the running server.
func runHealthCheck(addr string) error {
	url := "http://localhost" + addr + "/api/health"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return http.ErrAbortHandler
	}
	return nil
}
