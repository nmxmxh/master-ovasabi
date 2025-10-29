package main

import (
	"sync"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
)

// DeviceRegistry maintains a mapping from logical compute worker/device IDs
// to active WebSocket clients. This supports targeted delivery for
// compute:dispatch:v1:assigned events.
type DeviceRegistry struct {
	mu             sync.RWMutex
	deviceToClient map[string]*WSClient
}

func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{deviceToClient: make(map[string]*WSClient)}
}

func (r *DeviceRegistry) BindDevice(deviceID string, client *WSClient) {
	if deviceID == "" || client == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deviceToClient[deviceID] = client
}

// UnbindClient removes any device bindings that point to the given client.
func (r *DeviceRegistry) UnbindClient(client *WSClient) {
	if client == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, c := range r.deviceToClient {
		if c == client {
			delete(r.deviceToClient, id)
		}
	}
}

func (r *DeviceRegistry) GetClient(deviceID string) (*WSClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.deviceToClient[deviceID]
	return c, ok
}

// Global registry instance. main.go can choose to use or replace it during init.
var computeDeviceRegistry = NewDeviceRegistry()

// RegisterCapabilitiesFromClient binds a client to a device/worker ID using
// the Metadata.GlobalContext.Source field provided by the client when it
// emits a compute:capabilities:v1:update event. The capability payload is
// not inspected here; binding is metadata-driven.
func RegisterCapabilitiesFromClient(meta *commonpb.Metadata, client *WSClient, log *zap.Logger) {
	if meta == nil || meta.GlobalContext == nil || client == nil {
		return
	}
	deviceID := meta.GlobalContext.Source
	if deviceID == "" {
		return
	}
	computeDeviceRegistry.BindDevice(deviceID, client)
	if log != nil {
		log.Info("[COMPUTE] Bound device to WebSocket client",
			zap.String("device_id", deviceID),
			zap.String("user_id", client.userID),
			zap.String("campaign_id", client.campaignID))
	}
}

// ComputeTargetClientForEvent returns the target device ID and client for a
// compute:dispatch:v1:assigned event. It extracts
// metadata.service_specific.routing.target_worker_id.
func ComputeTargetClientForEvent(event *nexuspb.EventResponse) (string, *WSClient, bool) {
	if event == nil || event.Metadata == nil || event.Metadata.ServiceSpecific == nil {
		return "", nil, false
	}
	ss := event.Metadata.ServiceSpecific.AsMap()
	routingRaw, ok := ss["routing"].(map[string]interface{})
	if !ok {
		return "", nil, false
	}
	target, ok := routingRaw["target_worker_id"].(string)
	if !ok || target == "" {
		return "", nil, false
	}
	client, found := computeDeviceRegistry.GetClient(target)
	if !found || client == nil {
		return target, nil, false
	}
	return target, client, true
}
