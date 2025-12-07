package lock

import (
	"context"
	"net/http"
	"time"
)

// Detects availability of direct protocol services with fast, non-blocking probes.
// We keep timeouts short to avoid delaying discovery/status endpoints.

// IsZWaveJSUIAvailable returns true if a Z-Wave JS UI endpoint responds.
func IsZWaveJSUIAvailable(ctx context.Context) bool {
	url := getEnv("ZWAVE_JS_UI_URL", "http://localhost:3000") // UI default
	return probeURL(ctx, url)
}

// IsZigbee2MQTTAvailable returns true if a Zigbee2MQTT frontend responds.
func IsZigbee2MQTTAvailable(ctx context.Context) bool {
	url := getEnv("ZIGBEE2MQTT_URL", "http://localhost:8080") // UI default
	return probeURL(ctx, url)
}

func probeURL(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	// Consider any non-error HTTP status as available
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}
