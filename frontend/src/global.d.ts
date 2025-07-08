// TypeScript global declarations for WASM bridge globals

declare global {
  // Sends a message from JS/React to Go WASM (string or object)
  var wasmSendMessage: (msg: any) => void;
  // Exported by Go WASM, called by JS to send a message to Go
  var sendWasmMessage: ((msg: any) => void) | undefined;

  // Called by Go WASM to deliver a message to JS/React (string or object)
  var onWasmMessage: (msg: any) => void;

  // Called by Go WASM when WASM is fully ready (e.g., after WebSocket is connected)
  var onWasmReady: () => void;

  // Used by Go/JS for versioning/logging
  var __WASM_VERSION: string;

  // Used as a message queue for WASM concurrency interop
  var __WASM_EVENT_QUEUE: any[];

  interface Window {
    sendWasmMessage?: (msg: any) => void;
    wasmSendMessage?: (msg: any) => void;
    onWasmMessage?: (msg: any) => void;
    onWasmReady?: () => void;
    __WASM_EVENT_QUEUE?: any[];
    __WASM_VERSION?: string;
    /**
     * Returns a shared ArrayBuffer from WASM for JS/React to consume (if available)
     */
    getSharedBuffer?: () => ArrayBuffer;
  }
}

export {}; // Ensures this file is treated as a module
