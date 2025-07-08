// JS/WASM bridge for using the WASM WebSocket client as a single source of truth for all real-time communication.
// This assumes your WASM client exposes global functions for messaging.

// Type for messages sent/received via WASM bridge
export interface WasmBridgeMessage {
  type: string;
  payload?: any;
  [key: string]: any;
}

/**
 * Send a message to the server via the WASM WebSocket client.
 * @param msg - The message object to send (will be stringified if needed)
 */
export function wasmSendMessage(msg: WasmBridgeMessage) {
  if (typeof window !== 'undefined' && typeof (window as any).wasmSendMessage === 'function') {
    // Always send as a JSON string to Go WASM for robust cross-runtime compatibility
    (window as any).wasmSendMessage(JSON.stringify(msg));
  } else {
    throw new Error('WASM sendMessage bridge not available');
  }
}

/**
 * Register a callback for messages received from the server via the WASM WebSocket client.
 * @param cb - Callback to invoke with each message
 */
export function onWasmMessage(cb: (msg: WasmBridgeMessage) => void) {
  if (typeof window !== 'undefined') {
    (window as any).onWasmMessage = cb;
  }
}
