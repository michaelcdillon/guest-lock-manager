package lock

import (
	"context"
	"fmt"
	"net/http"
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

	header := http.Header{}
	if c.apiKey != "" {
		header.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("connect to Z-Wave JS UI: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	_ = conn.SetWriteDeadline(deadline)
	if err := conn.WriteJSON(cmd); err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	_ = conn.SetReadDeadline(deadline)

	var resp zwaveJSUIResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if !resp.Success {
		if resp.Error != "" {
			return fmt.Errorf("zwave_js_ui error: %s", resp.Error)
		}
		return fmt.Errorf("zwave_js_ui error: unknown failure")
	}

	return nil
}
