// @ts-ignore: no types for wasm-feature-detect
import { threads } from 'wasm-feature-detect';

// This file contains the low-level "glue" code for loading the Go WASM module
// and establishing the global functions for communication.

// --- Global Bridge Functions ---
// These functions are defined here so the application can use them,
// but they will be replaced by the actual functions exposed by the Go WASM module.
// The Zustand store will handle queueing messages until `onWasmReady` is called.

window.onWasmReady = () => {
  console.log('[WASM] onWasmReady called, but no store listener is attached yet.');
};

window.onWasmMessage = msg => {
  console.warn('[WASM] onWasmMessage called, but no store listener is attached yet.', msg);
};

// Helper to load Go WASM and wire up the JS <-> WASM bridge
async function loadGoWasm(wasmUrl: string) {
  // @ts-ignore
  if (!window.Go) {
    console.error(
      'Go WASM bootstrap script not loaded. Make sure wasm_exec.js is included in your index.html.'
    );
    return;
  }
  // @ts-ignore
  const go = new window.Go();
  try {
    const result = await WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject);
    // Run the Go WASM instance. This is a blocking call, so it should be last.
    go.run(result.instance);
  } catch (error) {
    console.error(`[WASM] Error instantiating WASM from ${wasmUrl}:`, error);
  }
}

// --- Main Execution ---
(async () => {
  try {
    // Extra debug: check SharedArrayBuffer and COOP/COEP
    const hasSharedArrayBuffer = typeof SharedArrayBuffer !== 'undefined';
    if (!hasSharedArrayBuffer) {
      console.warn('[WASM] SharedArrayBuffer is NOT available. Threads will not be supported.');
    } else {
      console.log('[WASM] SharedArrayBuffer is available.');
    }
    // Check for COOP/COEP headers (required for threads)
    if (hasSharedArrayBuffer && !crossOriginIsolated) {
      console.warn(
        '[WASM] crossOriginIsolated is FALSE. COOP/COEP headers are likely missing. Threads will not be supported.'
      );
    } else if (hasSharedArrayBuffer && crossOriginIsolated) {
      console.log('[WASM] crossOriginIsolated is TRUE. COOP/COEP headers are set.');
    }
    // Feature detect threads
    const supportsThreads = await threads();
    if (supportsThreads) {
      console.log('[WASM] WebAssembly threads are SUPPORTED.');
    } else {
      console.warn('[WASM] WebAssembly threads are NOT supported. Will load single-threaded WASM.');
    }
    const wasmUrl = supportsThreads ? '/main.threads.wasm' : '/main.wasm';
    // Set global variable for Go WASM to log
    (window as any).__WASM_VERSION = wasmUrl;
    await loadGoWasm(wasmUrl);
    console.log(`[WASM] Loaded ${wasmUrl} (threads: ${supportsThreads})`);
  } catch (error) {
    console.error('[WASM] Failed to detect features or load WASM module:', error);
  }
})();
