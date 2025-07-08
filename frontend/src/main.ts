// @ts-ignore: no types for wasm-feature-detect
import { threads } from 'wasm-feature-detect';

// --- Ensure WASM onWasmReady handler is defined before any WASM is loaded ---
(window as any).onWasmReady = function () {
  console.log('[frontend] WASM is ready!');
  // You can trigger any app logic here, e.g. set a React state, dispatch an event, etc.
};

// --- WASM concurrency interop demo ---
// We'll use a JS array as a message queue, and Go can pull from it via syscall/js
(window as any).__WASM_EVENT_QUEUE = [];

// --- PRODUCTION-GRADE WASM BRIDGE GLUE ---
// Helper to load Go WASM and wire up the JS <-> WASM bridge
async function loadGoWasm(wasmUrl: string) {
  // @ts-ignore
  const go = new window.Go();
  const result = await WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject);

  // Robust: Track when sendWasmMessage is set, and queue messages until available
  let wasmSendQueue: any[] = [];
  let sendWasmMessageReady = false;
  Object.defineProperty(window, 'sendWasmMessage', {
    configurable: true,
    set(fn) {
      console.log('[WASM] window.sendWasmMessage has been set by Go WASM runtime');
      Object.defineProperty(window, 'sendWasmMessage', {
        value: fn,
        writable: true,
        configurable: true
      });
      sendWasmMessageReady = true;
      // Flush queued messages
      if (wasmSendQueue.length > 0) {
        console.log('[WASM] Flushing queued messages to sendWasmMessage:', wasmSendQueue.length);
        wasmSendQueue.forEach(msg => {
          try {
            fn(msg);
          } catch (e) {
            console.error('[WASM] Error sending queued message:', msg, e);
          }
        });
        wasmSendQueue = [];
      }
    },
    get() {
      return undefined;
    }
  });

  // Expose JS -> WASM message send (called by frontend)
  window.wasmSendMessage = function (msg) {
    if (sendWasmMessageReady && typeof window.sendWasmMessage === 'function') {
      console.log('[WASM] window.sendWasmMessage is available, sending message:', msg);
      window.sendWasmMessage(msg);
    } else {
      console.warn('[WASM] sendWasmMessage not available, queueing message:', msg);
      wasmSendQueue.push(msg);
    }
  };

  // Expose WASM -> JS message receive (called by Go)
  // Go should call window.onWasmMessage(msg) to deliver messages to JS/React
  if (typeof window.onWasmMessage !== 'function') {
    window.onWasmMessage = function (msg) {
      // This will be set by the React bridge (see useWasmBridge)
      // No-op fallback
      console.warn('[WASM] onWasmMessage called but no handler registered', msg);
    };
  }

  // Go should call window.onWasmReady() when ready (after WebSocket is connected)
  if (typeof window.onWasmReady !== 'function') {
    window.onWasmReady = function () {
      // This will be set by the React bridge (see useWasmBridge)
      // No-op fallback
      if (!sendWasmMessageReady) {
        console.warn(
          '[WASM] onWasmReady called but sendWasmMessage is not set yet. Will wait until it is set.'
        );
        // When sendWasmMessage is set, the queue will be flushed and bridge will be ready.
      } else {
        console.warn('[WASM] onWasmReady called but no handler registered');
      }
    };
  }

  // Actually run the Go WASM instance
  go.run(result.instance);
}

(async () => {
  const supportsThreads = await threads();
  const wasmUrl = supportsThreads ? '/main.threads.wasm' : '/main.wasm';
  // Set global variable for Go WASM to log
  (window as any).__WASM_VERSION = wasmUrl;
  await loadGoWasm(wasmUrl);
  console.log(`[WASM] Loaded ${wasmUrl} (threads: ${supportsThreads})`);
})();
