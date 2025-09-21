//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"syscall/js"
	"time"
)

// MemoryPoolManager is defined in memorypool.go

// UserState represents the user state structure
type UserState struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	DeviceID    string                 `json:"device_id"`
	Timestamp   int64                  `json:"timestamp"`
	IsTemporary bool                   `json:"is_temporary"`
	Version     string                 `json:"version"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ComputeState represents compute-intensive state
type ComputeState struct {
	ID             string             `json:"id"`
	Type           string             `json:"type"`
	Data           []float32          `json:"data"`
	Params         map[string]float64 `json:"params"`
	Timestamp      int64              `json:"timestamp"`
	ProcessingTime float64            `json:"processing_time"`
	MemoryUsage    int                `json:"memory_usage"`
}

// StateManager handles multi-layer state management with memory pools
type StateManager struct {
	version        string
	storageKeys    map[string]string
	memoryPools    *MemoryPoolManager
	stateCache     map[string]interface{}
	computeCache   map[string]*ComputeState
	userStateCache map[string]*UserState
	mutex          sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		version: "1.0.0",
		storageKeys: map[string]string{
			"temp":       "temp_user_state",
			"persistent": "persistent_user_state",
			"migration":  "user_state_migration",
			"compute":    "compute_state_cache",
		},
		memoryPools:    NewMemoryPoolManager(),
		stateCache:     make(map[string]interface{}),
		computeCache:   make(map[string]*ComputeState),
		userStateCache: make(map[string]*UserState),
	}
}

// InitializeState initializes user state with multi-layer fallback
func (sm *StateManager) InitializeState() js.Value {
	// 1. Try to get from existing userID global
	if js.Global().Get("userID").Truthy() {
		existingUserID := js.Global().Get("userID").String()
		state := UserState{
			UserID:      existingUserID,
			SessionID:   sm.getOrGenerateSessionID(),
			DeviceID:    sm.getOrGenerateDeviceID(),
			Timestamp:   time.Now().UnixMilli(),
			IsTemporary: false,
			Version:     sm.version,
		}
		return sm.stateToJSValue(state)
	}

	// 2. Try session storage (survives refresh)
	sessionState := sm.getFromSessionStorage()
	if sessionState != nil {
		sm.setGlobalUserID(sessionState.UserID)
		return sm.stateToJSValue(*sessionState)
	}

	// 3. Try persistent storage
	persistentState := sm.getFromPersistentStorage()
	if persistentState != nil {
		sm.setGlobalUserID(persistentState.UserID)
		// Copy to session storage for current session
		sm.saveToSessionStorage(*persistentState)
		return sm.stateToJSValue(*persistentState)
	}

	// 4. Generate new state
	newState := sm.generateNewState()
	sm.setGlobalUserID(newState.UserID)
	sm.saveToSessionStorage(newState)
	sm.saveToPersistentStorage(newState)
	return sm.stateToJSValue(newState)
}

// MigrateOldState migrates old state format to new format
func (sm *StateManager) MigrateOldState() {
	// Check for old guest_id format (8 characters or less)
	oldGuestID := sm.getFromLocalStorage("guest_id")
	if oldGuestID != "" && len(oldGuestID) < 32 {
		wasmLog("[WASM] Migrating old guest ID format:", oldGuestID)
		sm.ClearAllStorage()
		// Force regeneration
		sm.InitializeState()
	}
}

// ClearAllStorage clears all storage
func (sm *StateManager) ClearAllStorage() {
	sm.clearStorage("sessionStorage")
	sm.clearStorage("localStorage")
}

// ClearPersistentStorage clears only persistent storage
func (sm *StateManager) ClearPersistentStorage() {
	sm.clearStorage("localStorage")
}

// GetState returns current state
func (sm *StateManager) GetState() js.Value {
	if js.Global().Get("userID").Truthy() {
		userID := js.Global().Get("userID").String()
		state := UserState{
			UserID:      userID,
			SessionID:   sm.getOrGenerateSessionID(),
			DeviceID:    sm.getOrGenerateDeviceID(),
			Timestamp:   time.Now().UnixMilli(),
			IsTemporary: false,
			Version:     sm.version,
		}
		return sm.stateToJSValue(state)
	}
	return js.Null()
}

// StoreComputeState stores compute state with memory pool optimization
func (sm *StateManager) StoreComputeState(state *ComputeState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Store in memory cache
	sm.computeCache[state.ID] = state

	// Store compute state in a separate storage key
	sm.saveComputeStateToStorage(state)
}

// GetComputeState retrieves compute state from cache
func (sm *StateManager) GetComputeState(id string) *ComputeState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.computeCache[id]
}

// GetMemoryPoolStats returns memory pool statistics
func (sm *StateManager) GetMemoryPoolStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := make(map[string]interface{})
	for size := range sm.memoryPools.float32Pools {
		stats[fmt.Sprintf("pool_%d", size)] = map[string]interface{}{
			"size":  size,
			"count": 0, // Pool count would require additional tracking
		}
	}

	return stats
}

// UpdateState updates the current state
func (sm *StateManager) UpdateState(updates js.Value) {
	if !updates.Truthy() {
		return
	}

	// Parse updates
	var updateMap map[string]interface{}
	if err := json.Unmarshal([]byte(updates.String()), &updateMap); err != nil {
		wasmLog("[WASM] Failed to parse state updates:", err)
		return
	}

	// Get current state
	currentState := sm.getFromSessionStorage()
	if currentState == nil {
		currentState = sm.getFromPersistentStorage()
	}
	if currentState == nil {
		wasmLog("[WASM] No current state to update")
		return
	}

	// Apply updates
	if userID, ok := updateMap["user_id"].(string); ok {
		currentState.UserID = userID
		sm.setGlobalUserID(userID)
	}
	if sessionID, ok := updateMap["session_id"].(string); ok {
		currentState.SessionID = sessionID
	}
	if deviceID, ok := updateMap["device_id"].(string); ok {
		currentState.DeviceID = deviceID
	}
	if isTemporary, ok := updateMap["is_temporary"].(bool); ok {
		currentState.IsTemporary = isTemporary
	}

	currentState.Timestamp = time.Now().UnixMilli()

	// Save updated state
	sm.saveToSessionStorage(*currentState)
	sm.saveToPersistentStorage(*currentState)
}

// Helper methods

func (sm *StateManager) getFromSessionStorage() *UserState {
	return sm.getFromStorage("sessionStorage", sm.storageKeys["temp"])
}

func (sm *StateManager) getFromPersistentStorage() *UserState {
	return sm.getFromStorage("localStorage", sm.storageKeys["persistent"])
}

func (sm *StateManager) getFromStorage(storageType, key string) *UserState {
	storage := js.Global().Get(storageType)
	if !storage.Truthy() {
		return nil
	}

	item := storage.Call("getItem", key)
	if !item.Truthy() {
		return nil
	}

	var state UserState
	if err := json.Unmarshal([]byte(item.String()), &state); err != nil {
		wasmLog("[WASM] Failed to parse state from", storageType, ":", err)
		return nil
	}

	// Validate state
	if state.Version != sm.version || state.UserID == "" {
		return nil
	}

	return &state
}

func (sm *StateManager) saveToSessionStorage(state UserState) {
	sm.saveToStorage("sessionStorage", sm.storageKeys["temp"], state)
}

func (sm *StateManager) saveToPersistentStorage(state UserState) {
	sm.saveToStorage("localStorage", sm.storageKeys["persistent"], state)
}

func (sm *StateManager) saveToStorage(storageType, key string, state UserState) {
	storage := js.Global().Get(storageType)
	if !storage.Truthy() {
		return
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		wasmLog("[WASM] Failed to marshal state:", err)
		return
	}

	storage.Call("setItem", key, string(stateJSON))
}

func (sm *StateManager) saveComputeStateToStorage(state *ComputeState) {
	storage := js.Global().Get("localStorage")
	if !storage.Truthy() {
		return
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		wasmLog("[WASM] Failed to marshal compute state:", err)
		return
	}

	key := sm.storageKeys["compute"] + "_" + state.ID
	storage.Call("setItem", key, string(stateJSON))
}

func (sm *StateManager) clearStorage(storageType string) {
	storage := js.Global().Get(storageType)
	if !storage.Truthy() {
		return
	}

	for _, key := range sm.storageKeys {
		storage.Call("removeItem", key)
	}
}

func (sm *StateManager) getFromLocalStorage(key string) string {
	storage := js.Global().Get("localStorage")
	if !storage.Truthy() {
		return ""
	}

	item := storage.Call("getItem", key)
	if !item.Truthy() {
		return ""
	}

	return item.String()
}

func (sm *StateManager) generateNewState() UserState {
	return UserState{
		UserID:      generateGuestID(),
		SessionID:   generateSessionID(),
		DeviceID:    generateDeviceID(),
		Timestamp:   time.Now().UnixMilli(),
		IsTemporary: true,
		Version:     sm.version,
	}
}

func (sm *StateManager) getOrGenerateSessionID() string {
	sessionID := sm.getFromLocalStorage("session_id")
	if sessionID == "" {
		sessionID = generateSessionID()
		sm.setInLocalStorage("session_id", sessionID)
	}
	return sessionID
}

func (sm *StateManager) getOrGenerateDeviceID() string {
	deviceID := sm.getFromLocalStorage("device_id")
	if deviceID == "" {
		deviceID = generateDeviceID()
		sm.setInLocalStorage("device_id", deviceID)
	}
	return deviceID
}

func (sm *StateManager) setInLocalStorage(key, value string) {
	storage := js.Global().Get("localStorage")
	if storage.Truthy() {
		storage.Call("setItem", key, value)
	}
}

func (sm *StateManager) setGlobalUserID(userID string) {
	js.Global().Set("userID", js.ValueOf(userID))
}

func (sm *StateManager) stateToJSValue(state UserState) js.Value {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		wasmLog("[WASM] Failed to marshal state to JS:", err)
		return js.Null()
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stateJSON, &result); err != nil {
		wasmLog("[WASM] Failed to unmarshal state:", err)
		return js.Null()
	}

	return js.ValueOf(result)
}

// Global state manager instance
var globalStateManager = NewStateManager()

// JS-exposed functions
func jsInitializeState(this js.Value, args []js.Value) interface{} {
	return globalStateManager.InitializeState()
}

func jsMigrateOldState(this js.Value, args []js.Value) interface{} {
	globalStateManager.MigrateOldState()
	return nil
}

func jsClearAllStorage(this js.Value, args []js.Value) interface{} {
	globalStateManager.ClearAllStorage()
	return nil
}

func jsClearPersistentStorage(this js.Value, args []js.Value) interface{} {
	globalStateManager.ClearPersistentStorage()
	return nil
}

func jsGetState(this js.Value, args []js.Value) interface{} {
	return globalStateManager.GetState()
}

func jsUpdateState(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		globalStateManager.UpdateState(args[0])
	}
	return nil
}

func jsStoreComputeState(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		var state ComputeState
		if err := json.Unmarshal([]byte(args[0].String()), &state); err == nil {
			globalStateManager.StoreComputeState(&state)
		}
	}
	return nil
}

func jsGetComputeState(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		state := globalStateManager.GetComputeState(args[0].String())
		if state != nil {
			// Convert ComputeState to JS value
			stateJSON, err := json.Marshal(state)
			if err != nil {
				wasmLog("[WASM] Failed to marshal compute state:", err)
				return js.Null()
			}

			var result map[string]interface{}
			if err := json.Unmarshal(stateJSON, &result); err != nil {
				wasmLog("[WASM] Failed to unmarshal compute state:", err)
				return js.Null()
			}

			return js.ValueOf(result)
		}
	}
	return js.Null()
}

func jsGetMemoryPoolStats(this js.Value, args []js.Value) interface{} {
	return globalStateManager.GetMemoryPoolStats()
}
