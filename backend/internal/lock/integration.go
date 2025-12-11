package lock

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Detects availability of direct protocol services with fast, non-blocking probes.
// We keep timeouts short to avoid delaying discovery/status endpoints.

// IsZWaveJSUIAvailable returns true if a Z-Wave JS UI endpoint responds.
func IsZWaveJSUIAvailable(ctx context.Context) bool {
	wsURL := GetZWaveJSUIURL()
	// Prefer a short websocket handshake; fall back to HTTP GET probe.
	if probeWebsocket(ctx, wsURL) {
		return true
	}
	httpURL := wsToHTTP(wsURL)
	return probeURL(ctx, httpURL)
}

// IsZigbee2MQTTAvailable returns true if a Zigbee2MQTT frontend responds.
func IsZigbee2MQTTAvailable(ctx context.Context) bool {
	url := getEnv("ZIGBEE2MQTT_URL", "http://localhost:8080") // UI default
	return probeURL(ctx, url)
}

func probeURL(ctx context.Context, url string) bool {
	// Some services (like Z-Wave JS UI) may not respond properly to HEAD,
	// so use GET with a short timeout.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

// wsToHTTP converts ws/wss URLs to http/https for probing, leaving others unchanged.
func wsToHTTP(raw string) string {
	if raw == "" {
		return "http://localhost:3000"
	}

	u, err := url.Parse(raw)
	if err != nil {
		// fallback: best-effort replacement
		return strings.Replace(raw, "ws://", "http://", 1)
	}

	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	case "":
		u.Scheme = "http"
	}
	return u.String()
}

// probeWebsocket attempts a short websocket handshake to the given ws/wss URL.
func probeWebsocket(ctx context.Context, wsURL string) bool {
	if wsURL == "" {
		wsURL = "ws://localhost:3000"
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
