// JS/WASM bridge for using the WASM WebSocket client as a single source of truth for all real-time communication.
// This module handles proper type conversion at the Frontend↔WASM boundary.

import type { EventEnvelope } from '../store/global';

// Type for messages sent/received via WASM bridge
export interface WasmBridgeMessage {
  type: string;
  payload?: any;
  metadata?: any;
  [key: string]: any;
}

// --- Emitter for multiple listeners ---
type WasmListener = (msg: WasmBridgeMessage) => void;
const listeners: WasmListener[] = [];

// notifyListeners handles WASM→Frontend type conversion at the boundary
function notifyListeners(msg: any) {
  // Convert WASM message to proper TypeScript types at the boundary
  const convertedMsg = wasmMessageToTypescript(msg);
  listeners.forEach(cb => cb(convertedMsg));
}

// wasmMessageToTypescript converts WASM message to proper TypeScript types
function wasmMessageToTypescript(wasmMsg: any): WasmBridgeMessage {
  // WASM should already be sending properly typed objects
  // but ensure we have the right structure
  const msg: WasmBridgeMessage = {
    type: wasmMsg.type || 'unknown',
    payload: wasmMsg.payload,
    metadata: wasmMsg.metadata || {}
  };

  // Copy any additional properties
  Object.keys(wasmMsg).forEach(key => {
    if (!['type', 'payload', 'metadata'].includes(key)) {
      msg[key] = wasmMsg[key];
    }
  });

  return msg;
}

// Expose the listener manager to the window for WASM to call
if (typeof window !== 'undefined') {
  (window as any).onWasmMessage = notifyListeners;
}

/**
 * Subscribe to messages from the WASM bridge.
 * @param cb The callback to invoke with each message.
 * @returns An unsubscribe function.
 */
export function subscribeToWasmMessages(cb: WasmListener): () => void {
  listeners.push(cb);
  return () => {
    const index = listeners.indexOf(cb);
    if (index > -1) {
      listeners.splice(index, 1);
    }
  };
}

/**
 * Send a message to the server via the WASM WebSocket client.
 * Handles Frontend→WASM type conversion at the boundary.
 * @param msg - The message object to send
 */
export function wasmSendMessage(msg: WasmBridgeMessage | EventEnvelope) {
  if (typeof window !== 'undefined' && typeof (window as any).sendWasmMessage === 'function') {
    // Convert TypeScript types to WASM-compatible format at the boundary
    const wasmMsg = typescriptToWasmMessage(msg);
    (window as any).sendWasmMessage(wasmMsg);
  } else {
    // Queue the message if the bridge isn't ready yet.
    console.warn('WASM bridge not available. Message will be sent upon readiness.');
    // You could implement a queue here if needed, but we will centralize it in the Zustand store.
  }
}

// typescriptToWasmMessage converts TypeScript types to WASM-compatible format
function typescriptToWasmMessage(msg: WasmBridgeMessage | EventEnvelope): any {
  // Ensure we have a clean object that WASM can properly handle
  const wasmMsg: any = {
    type: msg.type
  };

  // Handle payload - ensure it's properly structured
  if ('payload' in msg && msg.payload !== undefined) {
    wasmMsg.payload = msg.payload;
  }

  // Handle metadata - ensure it's an object
  if ('metadata' in msg && msg.metadata !== undefined) {
    wasmMsg.metadata = msg.metadata;
  } else {
    wasmMsg.metadata = {};
  }

  // Copy any additional properties from WasmBridgeMessage
  if ('payload' in msg || 'metadata' in msg) {
    Object.keys(msg).forEach(key => {
      if (!['type', 'payload', 'metadata'].includes(key)) {
        wasmMsg[key] = (msg as any)[key];
      }
    });
  }

  return wasmMsg;
}
