package lock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// ZWaveJSUIClient provides direct PIN operations over the Z-Wave JS UI websocket API.
// This bypasses Home Assistant for battery-efficient writes when available.
type ZWaveJSUIClient struct {
	apiKey  string
	timeout time.Duration
}

// NewZWaveJSUIClient builds a client using environment defaults.
// ZWAVE_JS_UI_WS_URL defaults to ws://localhost:3000
// ZWAVE_JS_UI_API_KEY optionally sets an Authorization bearer token.
func NewZWaveJSUIClient() *ZWaveJSUIClient {
	return &ZWaveJSUIClient{
		apiKey:  getEnv("ZWAVE_JS_UI_API_KEY", ""),
		timeout: 5 * time.Second,
	}
}

// SetUserCode writes a user code directly via Z-Wave JS UI.
func (c *ZWaveJSUIClient) SetUserCode(ctx context.Context, nodeID, slot int, code string) error {
	return c.call(ctx, zwaveJSUICommand{
		Command:      "node.execute_command",
		NodeID:       nodeID,
		Endpoint:     0,
		CommandClass: 99, // USER_CODE
		MethodName:   "setUserCode",
		Args:         []any{slot, code},
	})
}

// ClearUserCode removes a user code directly via Z-Wave JS UI.
func (c *ZWaveJSUIClient) ClearUserCode(ctx context.Context, nodeID, slot int) error {
	return c.call(ctx, zwaveJSUICommand{
		Command:      "node.execute_command",
		NodeID:       nodeID,
		Endpoint:     0,
		CommandClass: 99, // USER_CODE
		MethodName:   "clearUserCode",
		Args:         []any{slot},
	})
}

type zwaveJSUICommand struct {
	Command      string `json:"command"`
	NodeID       int    `json:"nodeId"`
	Endpoint     int    `json:"endpoint"`
	CommandClass int    `json:"commandClass"`
	MethodName   string `json:"methodName"`
	// Args vary per method; we keep it generic.
	Args []any `json:"args"`
}

type zwaveJSUIResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	MessageID string `json:"messageId"`
}

func (c *ZWaveJSUIClient) call(ctx context.Context, cmd zwaveJSUICommand) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	wsURL := GetZWaveJSUIURL()
	start := time.Now()

	header := http.Header{}
	if c.apiKey != "" {
		header.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("connect to Z-Wave JS UI (%s): %w", wsURL, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	_ = conn.SetWriteDeadline(deadline)
	if err := conn.WriteJSON(cmd); err != nil {
		return fmt.Errorf("send command to %s: %w", wsURL, err)
	}

	_ = conn.SetReadDeadline(deadline)

	_, data, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read response from %s: %w", wsURL, err)
	}

	var resp zwaveJSUIResponse
	_ = json.Unmarshal(data, &resp)

	if !resp.Success {
		// Include raw response for diagnostics (no PIN code in response).
		return fmt.Errorf("zwave_js_ui error: %s response=%s", resp.Error, strings.TrimSpace(string(data)))
	}

	log.Printf("Z-Wave JS UI command success via %s in %v (node=%d slot=%v op=%s)", wsURL, time.Since(start), cmd.NodeID, cmd.Args, cmd.MethodName)
	return nil
}
