// Package lock provides Home Assistant lock integration and management.
package lock

import (
	"os"
	"sync/atomic"
	"time"
)

// Config holds the configuration for Home Assistant API access.
type Config struct {
	// BaseURL is the Home Assistant API base URL
	BaseURL string

	// Token is the long-lived access token for API authentication
	Token string

	// SupervisorToken is the Supervisor API token (for addon mode)
	SupervisorToken string

	// Timeout for API requests
	Timeout time.Duration
}

// DefaultConfig returns the default configuration, reading from environment variables.
func DefaultConfig() Config {
	return Config{
		BaseURL:         getEnv("HA_URL", "http://supervisor/core"),
		Token:           getEnv("HA_TOKEN", ""),
		SupervisorToken: getEnv("SUPERVISOR_TOKEN", ""),
		Timeout:         30 * time.Second,
	}
}

// getEnv returns an environment variable value or a default if not set.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

var zwaveJSUIWSURL atomic.Value

func init() {
	zwaveJSUIWSURL.Store(defaultZWaveJSUIWSURL())
}

func defaultZWaveJSUIWSURL() string {
	return getEnv("ZWAVE_JS_UI_WS_URL", "ws://localhost:3000")
}

// SetZWaveJSUIURL overrides the runtime Z-Wave JS UI websocket URL.
// Accepts ws:// or wss:// URLs; empty values reset to the default.
func SetZWaveJSUIURL(url string) {
	if url == "" {
		url = defaultZWaveJSUIWSURL()
	}
	zwaveJSUIWSURL.Store(url)
}

// GetZWaveJSUIURL returns the currently configured Z-Wave JS UI websocket URL.
func GetZWaveJSUIURL() string {
	if v := zwaveJSUIWSURL.Load(); v != nil {
		return v.(string)
	}
	return defaultZWaveJSUIWSURL()
}

// IsAddonMode returns true if running as a Home Assistant addon.
func (c Config) IsAddonMode() bool {
	return c.SupervisorToken != ""
}

// AuthToken returns the appropriate authentication token.
func (c Config) AuthToken() string {
	if c.IsAddonMode() {
		return c.SupervisorToken
	}
	return c.Token
}
