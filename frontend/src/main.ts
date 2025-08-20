// Ensure WASM-dependent logic only runs after wasmReady event
// @ts-ignore: no types for wasm-feature-detect
import { threads } from 'wasm-feature-detect';

// --- Global Bridge Functions ---
// These functions are defined here so the application can use them,
// but they will be replaced by the actual functions exposed by the Go WASM module.
// The Zustand store will handle queueing messages until `onWasmReady` is called.

// Override default handlers to use pendingWasmReady logic
let pendingMessages: any[] = [];

window.onWasmMessage = (msg: any) => {
  // Integration: update campaign state from WASM/WebSocket events
  if (msg.type === 'campaign:state:v1:success' || msg.type === 'campaign:state:v1:completed') {
    // Update Zustand global store with new campaign state
    try {
      // Dynamically import global store to avoid circular deps
      import('./store/global').then(mod => {
        if (mod && mod.useGlobalStore) {
          const setCampaignState = mod.useGlobalStore.getState().setServiceState;
          // Use 'campaign' as the service name
          setCampaignState('campaign', msg.payload || {});
        }
      });
    } catch (err) {
      console.error('[WASM] Failed to update campaign state in store:', err);
    }
  }
  // Existing logic: keep pending messages for initialization
  console.log('[Global State] WASM Message (before store init)', msg);
  pendingMessages.push(msg);
};

// Helper to load Go WASM and wire up the JS <-> WASM bridge
async function loadGoWasm(wasmUrl: string) {
  console.log('[WASM-LOADER] loadGoWasm called with URL:', wasmUrl);
  // @ts-ignore
  if (!window.Go) {
    console.error(
      '[WASM-LOADER] Go WASM bootstrap script not loaded. Make sure wasm_exec.js is included in your index.html.'
    );
    return;
  }
  // @ts-ignore
  const go = new window.Go();
  let attempts = 0;
  const maxAttempts = 3;
  const requiredExports = ['initWebGPU', 'runGPUCompute', 'getGPUMetricsBuffer'];
  while (attempts < maxAttempts) {
    try {
      console.log('[WASM-LOADER] Attempting WASM instantiateStreaming...');
      const result = await WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject);
      console.log('[WASM-LOADER] WASM instantiateStreaming succeeded, running Go WASM...');
      go.run(result.instance);
      console.log('[WASM-LOADER] go.run completed. Checking exports...');
      // Check for required exports on window
      const missing = requiredExports.filter(fn => typeof (window as any)[fn] !== 'function');
      if (missing.length === 0) {
        console.log('[WASM-LOADER] All required exports attached to window:', requiredExports);
        // Dispatch wasmReady event
        window.dispatchEvent(new Event('wasmReady'));
        break;
      } else {
        console.warn('[WASM-LOADER] Missing exports after load:', missing);
        attempts++;
        if (attempts < maxAttempts) {
          console.log('[WASM-LOADER] Retrying WASM load...');
        } else {
          console.error(
            '[WASM-LOADER] Failed to attach all exports after maximum attempts:',
            missing
          );
        }
      }
    } catch (error) {
      console.error('[WASM-LOADER] Error instantiating WASM from', wasmUrl, error);
      attempts++;
    }
  }
}

// --- Main Execution ---

// Wait for service worker to signal readiness before starting WASM/compute-worker
function startWasmAndComputeWorker() {
  console.log('[WASM-LOADER] startWasmAndComputeWorker called');
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
        console.warn(
          '[WASM] WebAssembly threads are NOT supported. Will load single-threaded WASM.'
        );
      }
      const wasmUrl = supportsThreads ? '/main.threads.wasm' : '/main.wasm';
      // Set global variable for Go WASM to log
      (window as any).__WASM_VERSION = wasmUrl;
      await loadGoWasm(wasmUrl);
      console.log(`[WASM] Loaded ${wasmUrl} (threads: ${supportsThreads})`);
      // Start compute-worker after WASM is loaded
      if (window.Worker) {
        try {
          const worker = new Worker('/workers/compute-worker.js');
          worker.onmessage = event => {
            if (event.data && event.data.type === 'worker-ready') {
              console.log('[ComputeWorker] Ready:', event.data.capabilities);
            }
            if (event.data && event.data.type === 'worker-error') {
              console.error('[ComputeWorker] Error:', event.data.error);
            }
          };
          (window as any).__COMPUTE_WORKER = worker;
          console.log('[ComputeWorker] Started and listening for messages.');
        } catch (err) {
          console.error('[ComputeWorker] Failed to start:', err);
        }
      } else {
        console.warn('[ComputeWorker] Web Workers are not supported in this browser.');
      }
    } catch (error) {
      console.error('[WASM] Failed to detect features or load WASM module:', error);
    }
  })();
}

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.addEventListener('message', event => {
    if (event.data && event.data.type === 'sw-ready') {
      console.log('[SW] Ready, starting WASM and compute-worker');
      startWasmAndComputeWorker();
    }
  });
  // On every page load, always start WASM and compute-worker
  window.addEventListener('load', () => {
    console.log('[WASM-LOADER] window load event fired');
    startWasmAndComputeWorker();
  });
} else {
  // Fallback: start immediately if no SW
  startWasmAndComputeWorker();
}

window.addEventListener('wasmReady', () => {
  // Only start polling after wasmReady event
  console.log('[Frontend] wasmReady event received, checking for WASM exports...');
  // Log the WASM export summary for inspection
  const exportSummary = (window as any).__WASM_GLOBAL_METADATA;
  console.log('[Frontend] __WASM_GLOBAL_METADATA:', exportSummary);
  if (typeof (window as any).getWasmExportSummary === 'function') {
    console.log('[Frontend] getWasmExportSummary():', (window as any).getWasmExportSummary());
  }
  // Check for required exports and their types
  const requiredExports = ['initWebGPU', 'runGPUCompute', 'getGPUMetricsBuffer'];
  requiredExports.forEach(fn => {
    const typ = exportSummary && exportSummary[fn];
    if (typ !== 'function') {
      console.warn(`[Frontend] WASM export '${fn}' missing or not a function. Type:`, typ);
    }
  });
  // Continue with polling logic as before
  type WasmFunctions = {
    initWebGPU: boolean;
    runGPUCompute: boolean;
    getGPUMetricsBuffer: boolean;
  };
  const checkWasmFunctions = (): WasmFunctions => ({
    initWebGPU: typeof (window as any).initWebGPU === 'function',
    runGPUCompute: typeof (window as any).runGPUCompute === 'function',
    getGPUMetricsBuffer: typeof (window as any).getGPUMetricsBuffer === 'function'
  });

  let attempts = 0;
  const maxAttempts = 10;
  let timer: ReturnType<typeof setInterval>;
  const delayedStartMs = 200;
  const intervalMs = 1000;
  const updateStoreWithWasmFunctions = (wasmFunctions: WasmFunctions) => {
    import('./store/global').then(mod => {
      if (mod && mod.useGlobalStore) {
        const store = mod.useGlobalStore.getState();
        if (store.setConnectionState) store.setConnectionState({ wasmReady: true });
        if (store.setWasmFunctions) store.setWasmFunctions(wasmFunctions);
      }
    });
  };

  const pollWasmFunctions = () => {
    const wasmFunctions = checkWasmFunctions();
    console.log('[Frontend] WASM functions available (poll):', wasmFunctions);
    updateStoreWithWasmFunctions(wasmFunctions);
    attempts++;
    if (Object.values(wasmFunctions).every(Boolean) || attempts >= maxAttempts) {
      clearInterval(timer);
      console.log('[Frontend] WASM function polling complete:', wasmFunctions);
    }
  };
  // Delay polling to allow WASM exports to attach
  setTimeout(() => {
    pollWasmFunctions();
    timer = setInterval(pollWasmFunctions, intervalMs);
  }, delayedStartMs);
});
