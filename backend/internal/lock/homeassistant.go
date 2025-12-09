package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	EntityID   string            `json:"entity_id"`
	State      string            `json:"state"`
	Attributes LockAttributes    `json:"attributes"`
}

// LockAttributes represents lock entity attributes.
type LockAttributes struct {
	FriendlyName string `json:"friendly_name"`
	DeviceClass  string `json:"device_class,omitempty"`
	Supported    int    `json:"supported_features"`
	Battery      *int   `json:"battery,omitempty"`
}

// GetLocks retrieves all lock entities from Home Assistant.
func (c *HAClient) GetLocks(ctx context.Context) ([]LockEntity, error) {
	states, err := c.getStates(ctx)
	if err != nil {
		return nil, err
	}

	var locks []LockEntity
	for _, state := range states {
		if len(state.EntityID) > 5 && state.EntityID[:5] == "lock." {
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	return nil
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


