// Ensure WASM-dependent logic only runs after wasmReady event
// @ts-ignore: no types for wasm-feature-detect
import { threads } from 'wasm-feature-detect';

// WASM loader initialized

// --- Global Bridge Functions ---
// These functions are defined here so the application can use them,
// but they will be replaced by the actual functions exposed by the Go WASM module.
// The Zustand store will handle queueing messages until `onWasmReady` is called.

// Override default handlers to use pendingWasmReady logic
// let pendingMessages: any[] = []; // Removed - no longer needed

// Note: onWasmMessage is now handled by wasmBridge.ts to avoid duplication
// The wasmBridge.ts handler routes messages to the event store and notifies listeners

// Global error handler for WASM errors
(window as any).onWasmError = (error: any) => {
  console.error('[WASM] Error:', error);
};

// Handle user ID changes from WASM (guest → authenticated migration)
window.onUserIDChanged = (newUserId: string) => {
  console.log('[WASM] User ID changed:', newUserId);

  // Update metadata store with new user ID
  try {
    import('./store/stores/metadataStore').then(mod => {
      if (mod && mod.useMetadataStore) {
        const store = mod.useMetadataStore.getState();
        if (store.handleUserIDChange) {
          store.handleUserIDChange(newUserId);
        }
      }
    });
  } catch (err) {
    console.error('[WASM] Failed to update metadata store with new user ID:', err);
  }
};

// Global error handler for WASM errors
(window as any).onWasmError = (errorMsg: any) => {
  console.error('[WASM] Error received:', errorMsg);

  // Handle validation errors specifically
  if (errorMsg.type === 'error:validation_error') {
    console.error('[WASM] Validation error:', errorMsg.payload);
    // You could show a user-friendly error message here
  } else if (errorMsg.type === 'error:parse_error') {
    console.error('[WASM] Parse error:', errorMsg.payload);
  } else if (errorMsg.type === 'error:invalid_event_type') {
    console.error('[WASM] Invalid event type:', errorMsg.payload);
  }
};

// Helper to load Go WASM and wire up the JS <-> WASM bridge
async function loadGoWasm(wasmUrl: string) {
  // Loading WASM from URL
  // @ts-ignore
  if (!window.Go) {
    console.error(
      '[WASM-LOADER] Go WASM bootstrap script not loaded. Make sure wasm_exec.js is included in your index.html.'
    );
    return;
  }

  // Try the requested WASM file first, then fallback to main.wasm
  const wasmUrls = [wasmUrl];
  if (wasmUrl !== '/main.wasm') {
    wasmUrls.push('/main.wasm');
  }

  for (const currentWasmUrl of wasmUrls) {
    // Attempting WASM file load

    // @ts-ignore
    const go = new window.Go();
    let attempts = 0;
    const maxAttempts = 3;
    const requiredExports = [
      'initWebGPU',
      'getWebGPUDevice',
      'runGPUCompute',
      'runConcurrentCompute',
      'getGPUMetricsBuffer'
    ];

    while (attempts < maxAttempts) {
      try {
        // Attempting WASM instantiation

        // Try to fetch the WASM file first to check if it's accessible
        const response = await fetch(currentWasmUrl);
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const result = await WebAssembly.instantiateStreaming(response, go.importObject);
        // WASM instantiation succeeded, running Go WASM
        go.run(result.instance);
        // Go WASM execution completed

        // Check for required exports on window
        const missing = requiredExports.filter(fn => typeof (window as any)[fn] !== 'function');
        if (missing.length === 0) {
          // All required exports attached to window
          wasmInitializationComplete = true;
          // Dispatch wasmReady event
          window.dispatchEvent(new Event('wasmReady'));
          return; // Success!
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
        console.error('[WASM-LOADER] Error instantiating WASM from', currentWasmUrl, error);
        attempts++;

        // If this is the last attempt for this URL, try the next URL
        if (attempts >= maxAttempts && currentWasmUrl !== wasmUrls[wasmUrls.length - 1]) {
          console.log(`[WASM-LOADER] Failed to load ${currentWasmUrl}, trying next URL...`);
          break; // Break out of the attempts loop to try next URL
        }
      }
    }
  }

  console.error('[WASM-LOADER] Failed to load any WASM file after trying all URLs');
  // Reset the flag so we can retry if needed
  wasmInitializationStarted = false;
}

// --- Main Execution ---

// Guard to prevent multiple WASM initializations
let wasmInitializationStarted = false;
let wasmInitializationComplete = false;

// Wait for service worker to signal readiness before starting WASM/compute-worker
function startWasmAndComputeWorker() {
  // Prevent multiple initializations
  if (wasmInitializationStarted || wasmInitializationComplete) {
    console.log('[WASM-LOADER] WASM initialization already started or complete, skipping...');
    return;
  }

  wasmInitializationStarted = true;
  // Starting WASM and compute worker
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
      // Loading Go WASM
      await loadGoWasm(wasmUrl);
      console.log(`✅ WASM loaded successfully (threads: ${supportsThreads})`);
      // WASM functions loaded
      // Start compute-worker after WASM is loaded
      if (window.Worker) {
        try {
          const worker = new Worker('/workers/compute-worker.js');
          worker.onmessage = event => {
            if (event.data && event.data.type === 'worker-ready') {
              // Compute worker ready
            }
            if (event.data && event.data.type === 'worker-error') {
              console.error('[ComputeWorker] Error:', event.data.error);
            }
          };
          (window as any).__COMPUTE_WORKER = worker;
          // Compute worker started
        } catch (err) {
          console.error('[ComputeWorker] Failed to start:', err);
        }
      } else {
        console.warn('[ComputeWorker] Web Workers are not supported in this browser.');
      }
    } catch (error) {
      console.error('[WASM] Failed to detect features or load WASM module:', error);
      // Reset the flag so we can retry if needed
      wasmInitializationStarted = false;
    }
  })();
}

// Single WASM initialization to prevent conflicts with GPU setup
let wasmStartAttempted = false;
let wasmInitializationInProgress = false;

// Streamlined service worker registration
function registerServiceWorker() {
  if (!('serviceWorker' in navigator)) {
    console.log('[SW] Service Worker not supported');
    return;
  }

  // Registering service worker

  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/sw.js', { scope: '/' })
      .then(registration => {
        // Service worker registered successfully

        // Handle updates silently
        registration.addEventListener('updatefound', () => {
          const newWorker = registration.installing;
          if (newWorker) {
            newWorker.addEventListener('statechange', () => {
              if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                console.log('[SW] New version available, auto-reloading...');
                window.location.reload();
              }
            });
          }
        });
      })
      .catch(error => {
        console.error('[SW] Registration failed:', error);
      });
  });
}

// Register service worker
registerServiceWorker();

function attemptWasmStart() {
  if (wasmStartAttempted || wasmInitializationInProgress) {
    console.log('[WASM-LOADER] WASM start already attempted or in progress, skipping');
    return;
  }

  // Additional check: if WASM is already ready, don't start again
  if (typeof window !== 'undefined' && (window as any).wasmReady) {
    console.log('[WASM-LOADER] WASM already ready, skipping initialization');
    return;
  }

  wasmStartAttempted = true;
  wasmInitializationInProgress = true;
  console.log('[WASM-LOADER] Starting WASM initialization...');

  try {
    startWasmAndComputeWorker();
  } catch (error) {
    console.error('[WASM-LOADER] Error starting WASM:', error);
    wasmStartAttempted = false; // Reset on failure to allow retry
    wasmInitializationInProgress = false;
  }
}

if ('serviceWorker' in navigator) {
  // Listen for service worker ready signal
  navigator.serviceWorker.addEventListener('message', event => {
    if (event.data && event.data.type === 'sw-ready') {
      // Service worker ready, starting WASM
      attemptWasmStart();
    }
  });

  // Fallback: start on window load if SW doesn't signal ready
  window.addEventListener('load', () => {
    // Window loaded, attempting WASM start
    attemptWasmStart();
  });

  // Immediate fallback for development or if SW is slow
  setTimeout(() => {
    if (!wasmStartAttempted) {
      // Timeout fallback - starting WASM
      attemptWasmStart();
    }
  }, 1000);
} else {
  // No service worker - start immediately
  // No service worker, starting WASM immediately
  attemptWasmStart();
}

window.addEventListener('wasmReady', () => {
  // Only start polling after wasmReady event
  // WASM ready event received

  // Clear initialization progress flag
  wasmInitializationInProgress = false;

  // Clear any cached user IDs from localStorage to ensure fresh WASM user ID
  if (typeof window !== 'undefined' && window.localStorage) {
    // Clearing cached user IDs
    // Clear any cached metadata that might have stale user IDs
    window.localStorage.removeItem('metadata-store');
  }

  // Immediately set event store as ready when wasmReady event fires
  import('./store/stores/eventStore').then(mod => {
    if (mod && mod.useEventStore) {
      const store = mod.useEventStore.getState();
      if (store.setWasmReady) {
        // Setting event store WASM ready
        store.setWasmReady(true);
      }
    }
  });

  // Log the WASM export summary for inspection
  const exportSummary = (window as any).__WASM_GLOBAL_METADATA;
  // WASM global metadata available
  // Check for required exports and their types
  const requiredExports = [
    'initWebGPU',
    'runGPUCompute',
    'runGPUComputeWithOffset',
    'runConcurrentCompute',
    'getGPUMetricsBuffer',
    'getWebGPUDevice',
    'checkWebGPUAvailability',
    'getWasmWebGPUStatus',
    'checkWebGPUDeviceValidity',
    'sendWasmMessage'
  ];
  requiredExports.forEach(fn => {
    const windowFn = (window as any)[fn];
    const metadataFn = exportSummary && exportSummary[fn];

    // Check if function is available on window object
    if (typeof windowFn !== 'function') {
      console.warn(
        `[Frontend] WASM export '${fn}' missing or not a function on window. Type:`,
        typeof windowFn
      );
    } else {
      // WASM export available
    }

    // Also check metadata for debugging
    if (metadataFn && typeof metadataFn !== 'function') {
      console.log(
        `[Frontend] WASM export '${fn}' in metadata is not a function. Type:`,
        typeof metadataFn
      );
    }
  });
  // Continue with polling logic as before
  type WasmFunctions = {
    initWebGPU: boolean;
    runGPUCompute: boolean;
    runGPUComputeWithOffset: boolean;
    runConcurrentCompute: boolean;
    getGPUMetricsBuffer: boolean;
    sendWasmMessage: boolean;
  };
  const checkWasmFunctions = (): WasmFunctions => ({
    initWebGPU: typeof (window as any).initWebGPU === 'function',
    runGPUCompute: typeof (window as any).runGPUCompute === 'function',
    runGPUComputeWithOffset: typeof (window as any).runGPUComputeWithOffset === 'function',
    runConcurrentCompute: typeof (window as any).runConcurrentCompute === 'function',
    getGPUMetricsBuffer: typeof (window as any).getGPUMetricsBuffer === 'function',
    sendWasmMessage: typeof (window as any).sendWasmMessage === 'function'
  });

  let attempts = 0;
  const maxAttempts = 10;
  let timer: ReturnType<typeof setInterval>;
  const delayedStartMs = 200;
  const intervalMs = 1000;
  let lastWasmFunctions: WasmFunctions | null = null;

  const updateStoreWithWasmFunctions = (wasmFunctions: WasmFunctions) => {
    // Only update if functions have actually changed
    if (JSON.stringify(wasmFunctions) === JSON.stringify(lastWasmFunctions)) {
      return;
    }
    lastWasmFunctions = wasmFunctions;

    import('./store/stores/connectionStore').then(mod => {
      if (mod && mod.useConnectionStore) {
        const store = mod.useConnectionStore.getState();
        if (store.setConnectionState) store.setConnectionState({ wasmReady: true });
        if (store.setWasmFunctions) store.setWasmFunctions(wasmFunctions);
      }
    });

    // Also update event store when WASM is ready
    import('./store/stores/eventStore').then(mod => {
      if (mod && mod.useEventStore) {
        const store = mod.useEventStore.getState();
        if (store.setWasmReady) {
          // Setting event store WASM ready
          store.setWasmReady(true);
        }
      }
    });

    // Initialize metadata store with WASM-generated IDs
    import('./store/stores/metadataStore').then(mod => {
      if (mod && mod.useMetadataStore) {
        const store = mod.useMetadataStore.getState();
        if (store.initializeMetadata && store.initializeUserId) {
          console.log('[WASM] Initializing metadata store with WASM IDs');
          // Initialize both metadata and user ID
          store.initializeMetadata();
          store.initializeUserId();
        }
      }
    });
  };

  // Removed verbose WASM polling logs

  const pollWasmFunctions = () => {
    const wasmFunctions = checkWasmFunctions();

    // Removed verbose WASM polling logs

    updateStoreWithWasmFunctions(wasmFunctions);
    attempts++;

    // Check if core WASM functions are available (don't require runConcurrentCompute)
    const coreFunctions = {
      initWebGPU: wasmFunctions.initWebGPU,
      runGPUCompute: wasmFunctions.runGPUCompute,
      runGPUComputeWithOffset: wasmFunctions.runGPUComputeWithOffset,
      getGPUMetricsBuffer: wasmFunctions.getGPUMetricsBuffer
    };
    const coreFunctionsReady = Object.values(coreFunctions).every(Boolean);

    if (coreFunctionsReady || attempts >= maxAttempts) {
      clearInterval(timer);
      // Only log completion
      console.log('✅ WASM functions ready:', coreFunctionsReady);
    }
  };
  // Delay polling to allow WASM exports to attach
  setTimeout(() => {
    pollWasmFunctions();
    timer = setInterval(pollWasmFunctions, intervalMs);
  }, delayedStartMs);

  // Also start periodic connection status checking
  const checkConnectionStatus = () => {
    // Check if WebSocket is connected by looking at global state
    const wsConnected = (window as any).wsConnected;
    const wasmReady = (window as any).wasmReady;

    if (typeof wsConnected === 'boolean' || typeof wasmReady === 'boolean') {
      import('./store/stores/connectionStore').then(mod => {
        if (mod && mod.useConnectionStore) {
          const store = mod.useConnectionStore.getState();
          if (store.checkGlobalConnectionStatus) {
            store.checkGlobalConnectionStatus();
          }
        }
      });
    }
  };

  // Check connection status every 2 seconds
  setInterval(checkConnectionStatus, 2000);

  // Also check for connection timeouts every 10 seconds
  setInterval(() => {
    import('./store/stores/connectionStore').then(mod => {
      if (mod && mod.useConnectionStore) {
        const store = mod.useConnectionStore.getState();
        if (store.checkConnectionTimeout) {
          store.checkConnectionTimeout();
        }
      }
    });
  }, 10000);
});
