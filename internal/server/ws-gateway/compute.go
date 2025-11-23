package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
	// Optional: track per-device security info and policies
	deviceSecurity map[string]PeerSecurityInfo
	devicePolicy   map[string]MediaPolicy
}

func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		deviceToClient: make(map[string]*WSClient),
		deviceSecurity: make(map[string]PeerSecurityInfo),
		devicePolicy:   make(map[string]MediaPolicy),
	}
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
			delete(r.deviceSecurity, id)
			delete(r.devicePolicy, id)
		}
	}
}

func (r *DeviceRegistry) GetClient(deviceID string) (*WSClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.deviceToClient[deviceID]
	return c, ok
}

// --- Security and Policy scaffolding ---
type PeerSecurityInfo struct {
	DeviceID        string
	DTLSFingerprint string
	SAS             string
	Verified        bool
	LastUpdated     time.Time
}

type MediaPolicy struct {
	// Future: server-side hints used by orchestration to downgrade slow peers
	MaxBitrateKbps int
	PreferredCodec string // e.g., "vp8", "vp9", "h264", "av1"
	Priority       string // e.g., "low", "normal", "high"
}

func (r *DeviceRegistry) SetSecurityInfo(info PeerSecurityInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deviceSecurity[info.DeviceID] = info
}

func (r *DeviceRegistry) GetSecurityInfo(deviceID string) (PeerSecurityInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.deviceSecurity[deviceID]
	return info, ok
}

func (r *DeviceRegistry) MarkSecurityVerified(deviceID string, verified bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.deviceSecurity[deviceID]; ok {
		info.Verified = verified
		info.LastUpdated = time.Now()
		r.deviceSecurity[deviceID] = info
	}
}

func (r *DeviceRegistry) SetPolicy(deviceID string, policy MediaPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devicePolicy[deviceID] = policy
}

func (r *DeviceRegistry) GetPolicy(deviceID string) (MediaPolicy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.devicePolicy[deviceID]
	return p, ok
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
	deviceID := meta.GlobalContext.DeviceId
	if deviceID == "" {
		return
	}
	computeDeviceRegistry.BindDevice(deviceID, client)
	// Attempt to derive DTLS fingerprint from metadata (if the client provided it)
	fpr := extractDTLSFingerprint(meta)
	sas := ""
	if fpr != "" {
		sas = computeSASFromFingerprint(fpr)
		computeDeviceRegistry.SetSecurityInfo(PeerSecurityInfo{
			DeviceID:        deviceID,
			DTLSFingerprint: fpr,
			SAS:             sas,
			Verified:        false,
			LastUpdated:     time.Now(),
		})
		// Emit a SAS message to the peer for out-of-band human verification if desired
		emitSecuritySAS(client, deviceID, sas, log)
	}
	if log != nil {
		if sas != "" {
			log.Info("[COMPUTE] Bound device to WebSocket client (SAS available)",
				zap.String("device_id", deviceID),
				zap.String("user_id", client.userID),
				zap.String("campaign_id", client.campaignID),
				zap.String("sas", sas))
		} else {
			log.Info("[COMPUTE] Bound device to WebSocket client",
				zap.String("device_id", deviceID),
				zap.String("user_id", client.userID),
				zap.String("campaign_id", client.campaignID))
		}
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

// --- Helpers ---
func extractDTLSFingerprint(meta *commonpb.Metadata) string {
	// Preferred: metadata.service_specific.dtls.fingerprint
	if meta.ServiceSpecific != nil {
		m := meta.ServiceSpecific.AsMap()
		if dtlsRaw, ok := m["dtls"].(map[string]interface{}); ok {
			if fpr, ok2 := dtlsRaw["fingerprint"].(string); ok2 && fpr != "" {
				return fpr
			}
		}
	}
	// Fallback: metadata.global_context.dtls_fingerprint
	return ""
}

func computeSASFromFingerprint(fpr string) string {
	// Normalize fingerprint by removing colons/spaces and lowercasing
	norm := ""
	for i := 0; i < len(fpr); i++ {
		ch := fpr[i]
		if ch == ':' || ch == ' ' {
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			ch = ch + 32
		}
		norm += string(ch)
	}
	sum := sha256.Sum256([]byte(norm))
	// Use first 4 bytes for short auth string
	s := hex.EncodeToString(sum[:4])
	// Format as xx-xx-xx-xx
	if len(s) >= 8 {
		return fmt.Sprintf("%s-%s-%s-%s", s[0:2], s[2:4], s[4:6], s[6:8])
	}
	return s
}

func emitSecuritySAS(client *WSClient, deviceID, sas string, log *zap.Logger) {
	if client == nil || sas == "" {
		return
	}
	wsEvent := WebSocketEvent{
		Type: "security:sas:v1:broadcast",
		Payload: map[string]interface{}{
			"device_id": deviceID,
			"sas":       sas,
			"note":      "Verify this code with the remote peer out-of-band.",
		},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Version:     "1.0.0",
		Environment: "development",
		Source:      "ws-gateway",
	}
	b, err := json.Marshal(wsEvent)
	if err != nil {
		if log != nil {
			log.Warn("[SECURITY] Failed to marshal SAS event", zap.Error(err))
		}
		return
	}
	select {
	case client.send <- b:
		if log != nil {
			log.Info("[SECURITY] Emitted SAS to client", zap.String("user_id", client.userID), zap.String("device_id", deviceID), zap.String("sas", sas))
		}
	default:
		if log != nil {
			log.Warn("[SECURITY] Dropped SAS event: client buffer full", zap.String("user_id", client.userID))
		}
	}
}
