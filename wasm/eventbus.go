//go:build js && wasm
// +build js,wasm

package main

import (
	"sync"
)

type EventEnvelope struct {
	Type     string `json:"type"`
	Payload  []byte `json:"payload"`
	Metadata []byte `json:"metadata"`
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
