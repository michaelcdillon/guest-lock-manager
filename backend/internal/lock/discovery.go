package lock

import (
	"context"
	"strings"
)

// DiscoveredLock represents a lock found during discovery.
type DiscoveredLock struct {
	EntityID          string  `json:"entity_id"`
	Name              string  `json:"name"`
	Protocol          string  `json:"protocol"`
	SupportsPIN       bool    `json:"supports_pin"`
	Online            bool    `json:"online"`
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

		lock := DiscoveredLock{
			EntityID:    entity.EntityID,
			Name:        entity.Attributes.FriendlyName,
			Protocol:    detectProtocol(entity),
			SupportsPIN: supportsPINCode(entity),
			Online:      entity.State != "unavailable",
			BatteryLevel: func() *int {
				if entity.Attributes.Battery != nil {
					return entity.Attributes.Battery
				}
				return nil
			}(),
			DirectIntegration: directIntegration,
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

// supportsPINCode checks if the lock supports user code management.
func supportsPINCode(entity LockEntity) bool {
	// Check supported features bitmask
	// Feature 1: Lock/Unlock
	// Feature 2: Open (unlatch)
	// Feature 4: User codes (what we need)
	const userCodeFeature = 4
	return entity.Attributes.Supported&userCodeFeature != 0
}
