//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"sync"
)

type EventEnvelope struct {
	Type          string          `json:"type"`
	CorrelationID string          `json:"correlation_id"`
	Payload       json.RawMessage `json:"payload"`
	Metadata      json.RawMessage `json:"metadata"`
}

type WASMEventBus struct {
	sync.RWMutex
	handlers map[string]func(EventEnvelope)
}

// NewWASMEventBus creates a new WASMEventBus
func NewWASMEventBus() *WASMEventBus {
	return &WASMEventBus{
		handlers: make(map[string]func(EventEnvelope)),
	}
}

func (eb *WASMEventBus) RegisterHandler(msgType string, handler func(EventEnvelope)) {
	eb.Lock()
	defer eb.Unlock()
	eb.handlers[msgType] = handler
}

func (eb *WASMEventBus) GetHandler(msgType string) func(EventEnvelope) {
	eb.RLock()
	defer eb.RUnlock()
	return eb.handlers[msgType]
}

func (eb *WASMEventBus) GetRegisteredHandlers() map[string]func(EventEnvelope) {
	eb.RLock()
	defer eb.RUnlock()
	// Return a copy to avoid race conditions
	handlers := make(map[string]func(EventEnvelope))
	for k, v := range eb.handlers {
		handlers[k] = v
	}
	return handlers
}
