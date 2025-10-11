//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

// Utility functions and shared helpers.

// --- Utility Functions ---
// Use main-thread-only logging from log.go

// --- AI/ML Functions (Optimized) ---
// Infer processes input using SIMD-like batch operations
func Infer(input []byte) []byte {
	// Reuse buffer from pool
	buf := resourcePool.Get().([]byte)[:0]
	defer resourcePool.Put(buf)

	buf = append(buf, bytes.ToUpper(input)...)
	return buf
}

// Embed generates vector embeddings (WebGPU compute would be better)
func Embed(input []byte) []float32 {
	vec := make([]float32, 8)
	for i := 0; i < 8 && i < len(input); i++ {
		vec[i] = float32(input[i])
	}
	return vec
}

//go:embed config/service_registration.json
var embeddedServiceRegistration embed.FS

func getEmbeddedServiceRegistration() []byte {
	data, err := embeddedServiceRegistration.ReadFile("config/service_registration.json")
	if err != nil {
		return nil
	}
	return data
}

// emitToNexus sends event results/state to the Nexus event bus
func emitToNexus(eventType string, payload interface{}, metadata json.RawMessage) {
	if !isMainThread() {
		wasmWarn("[NEXUS EMIT] Attempted to emit event from worker thread. Event emission is restricted to main thread.", eventType)
		return
	}

	// Ensure payload is wrapped in a 'data' field for canonical envelope
	var canonicalPayload map[string]interface{}
	switch v := payload.(type) {
	case map[string]interface{}:
		// If already has 'data' field, use as-is
		if _, ok := v["data"]; ok {
			canonicalPayload = v
		} else {
			canonicalPayload = map[string]interface{}{"data": v}
		}
	default:
		canonicalPayload = map[string]interface{}{"data": v}
	}
	payloadBytes, _ := json.Marshal(canonicalPayload)
	// Ensure metadata is a structured object, not empty or a string
	var canonicalMetadata map[string]interface{}
	if len(metadata) == 0 {
		canonicalMetadata = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(metadata, &canonicalMetadata); err != nil {
			// If metadata is not valid JSON, fallback to empty object
			canonicalMetadata = map[string]interface{}{}
		}
	}
	// Optionally, add required subfields if missing (e.g., versioning, campaign, user)
	// Example: if _, ok := canonicalMetadata["versioning"]; !ok { canonicalMetadata["versioning"] = map[string]interface{}{} }
	metadataBytes, _ := json.Marshal(canonicalMetadata)
	env := EventEnvelope{
		Type:     eventType,
		Payload:  payloadBytes,
		Metadata: metadataBytes,
	}

	// Use the established userID global and ID generation pattern from main.go
	currentUserID := userID
	if currentUserID == "" {
		currentUserID = "guest_unknown"
	}

	// Try to get campaign ID from incoming metadata, fallback to default
	currentCampaignID := "0" // Default campaign
	var correlationID string
	if len(metadata) > 0 {
		var metaMap map[string]interface{}
		if err := json.Unmarshal(metadata, &metaMap); err == nil {
			if globalContext, ok := metaMap["global_context"].(map[string]interface{}); ok {
				if campaignID, ok := globalContext["campaign_id"].(string); ok && campaignID != "" {
					currentCampaignID = campaignID
				}
				if cid, ok := globalContext["correlation_id"].(string); ok && cid != "" {
					correlationID = cid
				}
			}
		}
	}

	// Generate other IDs using the established pattern
	sessionID := generateSessionID()
	deviceID := generateDeviceID()
	if correlationID == "" {
		correlationID = generateCorrelationID()
	}

	// Create canonical envelope format for WebSocket gateway using established pattern
	canonicalEnvelope := map[string]interface{}{
		"type":           eventType,
		"correlation_id": correlationID,
		"timestamp":      time.Now().Format(time.RFC3339),
		"version":        "1.0.0",
		"environment":    "production",
		"source":         "wasm",
		"metadata": map[string]interface{}{
			"global_context": map[string]interface{}{
				"user_id":        currentUserID,
				"campaign_id":    currentCampaignID,
				"correlation_id": correlationID,
				"session_id":     sessionID,
				"device_id":      deviceID,
				"source":         "wasm",
			},
			"envelope_version": "1.0.0",
			"environment":      "production",
		},
		"payload": canonicalPayload,
	}

	envelopeBytes, err := json.Marshal(canonicalEnvelope)
	if err != nil {
		wasmError("[NEXUS ERROR] Failed to marshal canonical envelope:", err)
		return
	}
	sendWSMessage(envelopeBytes)
	wasmLog("[NEXUS EMIT] EventEnvelope:", env)
	wasmLog("[NEXUS EMIT] eventType:", eventType)
	wasmLog("[NEXUS EMIT] payload (type):", fmt.Sprintf("%T", payload), "payload (raw):", payload)
	wasmLog("[NEXUS EMIT] metadata (type):", fmt.Sprintf("%T", metadata), "metadata (raw):", metadata)
	wasmLog("[NEXUS EMIT] marshaled envelope:", string(envelopeBytes))
}

// MemoryPoolManager methods are defined in memorypool.go

func jsValueToMap(v js.Value) map[string]interface{} {
	result := make(map[string]interface{})
	keys := js.Global().Get("Object").Call("keys", v)
	length := keys.Get("length").Int()

	for i := 0; i < length; i++ {
		key := keys.Index(i).String()
		val := v.Get(key)

		if val.Type() == js.TypeObject && !val.IsNull() {
			result[key] = jsValueToMap(val)
		} else if val.Type() == js.TypeString {
			result[key] = val.String()
		} else if val.Type() == js.TypeNumber {
			result[key] = val.Float()
		} else if val.Type() == js.TypeBoolean {
			result[key] = val.Bool()
		} else if val.IsNull() {
			result[key] = nil
		} else {
			result[key] = val.String() // fallback
		}
	}

	return result
}
