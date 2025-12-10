package lock

import (
	"context"
	"strconv"
	"strings"
)

// DiscoveredLock represents a lock found during discovery.
type DiscoveredLock struct {
	EntityID          string  `json:"entity_id"`
	Name              string  `json:"name"`
	Protocol          string  `json:"protocol"`
	SupportsPIN       bool    `json:"supports_pin"`
	Online            bool    `json:"online"`
	State             string  `json:"state"`
	BatteryLevel      *int    `json:"battery_level,omitempty"`
	DirectIntegration *string `json:"direct_integration,omitempty"`
}

// Discovery provides lock discovery functionality.
type Discovery struct {
	haClient *HAClient
}

// NewDiscovery creates a new lock discovery service.
func NewDiscovery(haClient *HAClient) *Discovery {
	return &Discovery{haClient: haClient}
}

// DiscoverLocks finds all compatible locks in Home Assistant.
func (d *Discovery) DiscoverLocks(ctx context.Context) ([]DiscoveredLock, error) {
	entities, err := d.haClient.GetLocks(ctx)
	if err != nil {
		return nil, err
	}

	zwaveAvailable := IsZWaveJSUIAvailable(ctx)
	zigbeeAvailable := IsZigbee2MQTTAvailable(ctx)

	var locks []DiscoveredLock
	for _, entity := range entities {
		var directIntegration *string
		var nodeOnline *bool

		switch proto := detectProtocol(entity); proto {
		case "zwave":
			if zwaveAvailable {
				val := "zwave_js_ui"
				directIntegration = &val
			}
		case "zigbee":
			if zigbeeAvailable {
				val := "zigbee2mqtt"
				directIntegration = &val
			}
		}

		battery := firstBattery(entity.Attributes)
		if battery == nil {
			battery = d.lookupBatterySensor(ctx, entity.EntityID)
		}

		if online := d.lookupNodeStatus(ctx, entity.EntityID); online != nil {
			nodeOnline = online
		}

		lock := DiscoveredLock{
			EntityID:    entity.EntityID,
			Name:        entity.Attributes.FriendlyName,
			Protocol:    detectProtocol(entity),
			SupportsPIN: supportsPINCode(entity),
			Online: func() bool {
				if nodeOnline != nil {
					return *nodeOnline
				}
				return entity.State != "unavailable"
			}(),
			State:             normalizeState(entity.State),
			BatteryLevel:      battery,
			DirectIntegration: directIntegration,
		}

		// If protocol unknown but node status sensor exists, assume zwave
		if lock.Protocol == "unknown" && nodeOnline != nil {
			lock.Protocol = "zwave"
		}

		locks = append(locks, lock)
	}

	return locks, nil
}

// detectProtocol infers the lock protocol from the entity metadata.
func detectProtocol(entity LockEntity) string {
	id := strings.ToLower(entity.EntityID)
	name := strings.ToLower(entity.Attributes.FriendlyName)
	deviceClass := strings.ToLower(entity.Attributes.DeviceClass)

	switch {
	case strings.Contains(id, "zwave") || strings.Contains(name, "z-wave") || strings.Contains(deviceClass, "zwave"):
		return "zwave"
	case strings.Contains(id, "zigbee") || strings.Contains(id, "z2m") || strings.Contains(deviceClass, "zigbee"):
		return "zigbee"
	case strings.Contains(id, "wifi") || strings.Contains(id, "august") || strings.Contains(id, "yale"):
		return "wifi"
	default:
		return "unknown"
	}
}

func normalizeState(state string) string {
	switch strings.ToLower(state) {
	case "locked":
		return "locked"
	case "unlocked":
		return "unlocked"
	case "jammed":
		return "jammed"
	default:
		return "unknown"
	}
}

// supportsPINCode checks if the lock supports user code management.
func supportsPINCode(entity LockEntity) bool {
	// Check supported features bitmask
	// Feature 1: Lock/Unlock
	// Feature 2: Open (unlatch)
	// Feature 4: User codes (what we need)
	const userCodeFeature = 4
	return entity.Attributes.Supported&userCodeFeature != 0
}

func firstBattery(attr LockAttributes) *int {
	if attr.Battery != nil {
		return attr.Battery
	}
	if attr.BatteryLevel != nil {
		return attr.BatteryLevel
	}
	return nil
}

// lookupBatterySensor tries companion battery sensors like sensor.<lock>_battery_level or _battery.
func (d *Discovery) lookupBatterySensor(ctx context.Context, lockEntityID string) *int {
	base := strings.TrimPrefix(lockEntityID, "lock.")
	if base == "" {
		return nil
	}
	candidates := []string{
		"sensor." + base + "_battery_level",
		"sensor." + base + "_battery",
	}

	for _, cid := range candidates {
		state, err := d.haClient.GetEntityState(ctx, cid)
		if err != nil || state == nil {
			continue
		}

		// Prefer attributes battery_level/battery, else parse state
		if val := parseBatteryValue(state.Attributes["battery_level"]); val != nil {
			return val
		}
		if val := parseBatteryValue(state.Attributes["battery"]); val != nil {
			return val
		}
		if val := parseBatteryValue(state.State); val != nil {
			return val
		}
	}
	return nil
}

func parseBatteryValue(v any) *int {
	switch t := v.(type) {
	case float64:
		iv := int(t)
		return &iv
	case int:
		iv := t
		return &iv
	case int64:
		iv := int(t)
		return &iv
	case string:
		if t == "" {
			return nil
		}
		if iv, err := strconv.Atoi(t); err == nil {
			return &iv
		}
	}
	return nil
}

// lookupNodeStatus reads sensor.<lock>_node_status to improve online/protocol detection.
func (d *Discovery) lookupNodeStatus(ctx context.Context, lockEntityID string) *bool {
	base := strings.TrimPrefix(lockEntityID, "lock.")
	if base == "" {
		return nil
	}
	entityID := "sensor." + base + "_node_status"
	state, err := d.haClient.GetEntityState(ctx, entityID)
	if err != nil || state == nil {
		return nil
	}
	val := strings.ToLower(state.State)
	switch val {
	case "alive", "awake", "ready":
		return boolPtr(true)
	case "dead", "asleep", "sleeping":
		return boolPtr(false)
	default:
		return nil
	}
}

func boolPtr(b bool) *bool { return &b }
