package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// HAClient is a client for the Home Assistant API.
type HAClient struct {
	config     Config
	httpClient *http.Client
}

// NewHAClient creates a new Home Assistant API client.
func NewHAClient(config Config) *HAClient {
	return &HAClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// LockEntity represents a lock entity from Home Assistant.
type LockEntity struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes LockAttributes `json:"attributes"`
}

// LockAttributes represents lock entity attributes.
type LockAttributes struct {
	FriendlyName string `json:"friendly_name"`
	DeviceClass  string `json:"device_class,omitempty"`
	Supported    int    `json:"supported_features"`
	Battery      *int   `json:"battery,omitempty"`
	BatteryLevel *int   `json:"battery_level,omitempty"`
	NodeID       *int   `json:"node_id,omitempty"`
}

// EntityState represents a generic HA entity state.
type EntityState struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
}

// GetLocks retrieves all lock entities from Home Assistant.
func (c *HAClient) GetLocks(ctx context.Context) ([]LockEntity, error) {
	states, err := c.getStates(ctx)
	if err != nil {
		return nil, err
	}

	// Try to enrich with node_id via registry (add-on mode only).
	nodeIDs := c.getRegistryNodeIDs(ctx)

	var locks []LockEntity
	for _, state := range states {
		if len(state.EntityID) > 5 && state.EntityID[:5] == "lock." {
			if nodeIDs != nil {
				if nid, ok := nodeIDs[state.EntityID]; ok {
					state.Attributes.NodeID = &nid
				}
			}
			locks = append(locks, state)
		}
	}

	return locks, nil
}

// SetUserCode sets a user code on a lock.
func (c *HAClient) SetUserCode(ctx context.Context, entityID string, slot int, code string) error {
	data := map[string]any{
		"entity_id": entityID,
		"code_slot": slot,
		"usercode":  code,
	}

	return c.callService(ctx, "lock", "set_usercode", data)
}

// ClearUserCode removes a user code from a lock.
func (c *HAClient) ClearUserCode(ctx context.Context, entityID string, slot int) error {
	data := map[string]any{
		"entity_id": entityID,
		"code_slot": slot,
	}

	return c.callService(ctx, "lock", "clear_usercode", data)
}

// getStates retrieves all entity states from Home Assistant.
func (c *HAClient) getStates(ctx context.Context) ([]LockEntity, error) {
	req, err := c.newRequest(ctx, "GET", "/api/states", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var states []LockEntity
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return states, nil
}

// GetEntityState retrieves a specific entity state by ID.
func (c *HAClient) GetEntityState(ctx context.Context, entityID string) (*EntityState, error) {
	path := fmt.Sprintf("/api/states/%s", entityID)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var state EntityState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &state, nil
}

// callService calls a Home Assistant service.
func (c *HAClient) callService(ctx context.Context, domain, service string, data any) error {
	path := fmt.Sprintf("/api/services/%s/%s", domain, service)

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := c.newRequest(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d) calling %s with payload %s: %s",
			resp.StatusCode, path, string(body), strings.TrimSpace(string(respBody)))
	}

	return nil
}

// getRegistryNodeIDs fetches entity->node_id mappings from the HA registry via websocket.
// Only works in add-on mode (Supervisor token). Returns nil on any error.
func (c *HAClient) getRegistryNodeIDs(ctx context.Context) map[string]int {
	if !c.config.IsAddonMode() {
		log.Printf("Registry lookup skipped: not running as addon (no Supervisor token).")
		return nil
	}

	wsURL := strings.Replace(c.config.BaseURL, "http", "ws", 1) + "/websocket"
	dialer := websocket.Dialer{
		HandshakeTimeout: 3 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		log.Printf("Registry lookup failed: websocket dial error: %v", err)
		return nil
	}
	defer conn.Close()

	// auth
	authMsg := map[string]any{
		"type":         "auth",
		"access_token": c.config.AuthToken(),
	}
	if err := conn.WriteJSON(authMsg); err != nil {
		log.Printf("Registry lookup failed: auth send error: %v", err)
		return nil
	}
	var resp map[string]any
	if err := conn.ReadJSON(&resp); err != nil {
		log.Printf("Registry lookup failed: auth read error: %v", err)
		return nil
	}
	if resp["type"] != "auth_ok" {
		log.Printf("Registry lookup failed: auth not ok (resp=%v)", resp)
		return nil
	}

	// entity registry list
	if err := conn.WriteJSON(map[string]any{"id": 1, "type": "config/entity_registry/list"}); err != nil {
		log.Printf("Registry lookup failed: entity_registry request error: %v", err)
		return nil
	}
	var ents map[string]any
	for {
		if err := conn.ReadJSON(&ents); err != nil {
			log.Printf("Registry lookup failed: entity_registry read error: %v", err)
			return nil
		}
		if id, ok := ents["id"].(float64); ok && int(id) == 1 {
			break
		}
	}
	entResult, ok := ents["result"].([]any)
	if !ok {
		log.Printf("Registry lookup failed: entity_registry result missing or invalid")
		return nil
	}
	entityToDevice := make(map[string]string)
	for _, e := range entResult {
		if m, ok := e.(map[string]any); ok {
			eid, _ := m["entity_id"].(string)
			did, _ := m["device_id"].(string)
			if eid != "" && did != "" {
				entityToDevice[eid] = did
			}
		}
	}

	if len(entityToDevice) == 0 {
		log.Printf("Registry lookup: no entities found in registry result")
		return nil
	}

	// device registry list
	if err := conn.WriteJSON(map[string]any{"id": 2, "type": "config/device_registry/list"}); err != nil {
		log.Printf("Registry lookup failed: device_registry request error: %v", err)
		return nil
	}
	var devs map[string]any
	for {
		if err := conn.ReadJSON(&devs); err != nil {
			log.Printf("Registry lookup failed: device_registry read error: %v", err)
			return nil
		}
		if id, ok := devs["id"].(float64); ok && int(id) == 2 {
			break
		}
	}
	devResult, ok := devs["result"].([]any)
	if !ok {
		log.Printf("Registry lookup failed: device_registry result missing or invalid")
		return nil
	}

	deviceToNode := make(map[string]int)
	for _, d := range devResult {
		m, ok := d.(map[string]any)
		if !ok {
			continue
		}
		did, _ := m["id"].(string)
		idents, _ := m["identifiers"].([]any)
		for _, ident := range idents {
			if pair, ok := ident.([]any); ok && len(pair) == 2 {
				provider, _ := pair[0].(string)
				val, _ := pair[1].(string)
				if provider == "zwave_js" {
					parts := strings.Split(val, "-")
					if len(parts) > 0 {
						if nid, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
							deviceToNode[did] = nid
							break
						}
					}
				}
			}
		}
	}

	if len(deviceToNode) == 0 {
		log.Printf("Registry lookup: no zwave_js identifiers found in devices")
		return nil
	}

	out := make(map[string]int)
	for eid, did := range entityToDevice {
		if nid, ok := deviceToNode[did]; ok {
			out[eid] = nid
		}
	}
	if len(out) == 0 {
		log.Printf("Registry lookup: no entity->node_id matches found")
		return nil
	}
	log.Printf("Registry lookup: mapped %d entities to zwave node_ids", len(out))
	return out
}

// newRequest creates a new HTTP request with authentication.
func (c *HAClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.config.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.AuthToken())
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
