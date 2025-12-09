// Package lock provides Home Assistant lock integration and management.
package lock

import (
	"os"
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


