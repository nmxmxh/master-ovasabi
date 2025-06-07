// @ts-ignore: no types for wasm-feature-detect
import { threads } from 'wasm-feature-detect';

// --- WASM concurrency interop demo ---
// We'll use a JS array as a message queue, and Go can pull from it via syscall/js
(window as any).__WASM_EVENT_QUEUE = [];

// Helper to load Go WASM
async function loadGoWasm(wasmUrl: string) {
  // @ts-ignore
  const go = new window.Go();
  const result = await WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject);
  go.run(result.instance);
}

(async () => {
  const supportsThreads = await threads();
  const wasmUrl = supportsThreads ? '/main.threads.wasm' : '/main.wasm';
  // Set global variable for Go WASM to log
  (window as any).__WASM_VERSION = wasmUrl;
  await loadGoWasm(wasmUrl);
  console.log(`[WASM] Loaded ${wasmUrl} (threads: ${supportsThreads})`);

  // Simulate concurrent event arrival (e.g., from WebSocket)
  setInterval(() => {
    (window as any).__WASM_EVENT_QUEUE.push({
      type: 'demo_event',
      payload: { ts: Date.now() }
    });
  }, 1000);

  // Optionally, expose a JS function to trigger Go event processing
  (window as any).triggerGoProcessEvent = () => {
    // Go should export a function (via syscall/js) to process the next event
    if ((window as any).processNextWasmEvent) {
      (window as any).processNextWasmEvent();
    }
  };
})();
