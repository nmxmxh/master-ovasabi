import { useEventStore } from '../stores/eventStore';
import { useConnectionStore } from '../stores/connectionStore';

// Initialize compute event integration
export function initComputeEvents() {
  // Get store actions
  const eventStore = useEventStore.getState();
  const connectionStore = useConnectionStore.getState();

  // Register global WASM message handler
  if (typeof window !== 'undefined') {
    window.addEventListener('compute:capabilities:v1:update', (event: any) => {
      console.warn('[WASM] ✅ Received compute capabilities update:', {
        detail: event.detail,
        correlation_id: event.detail?.correlation_id,
        device_id: event.detail?.device_id
      });

      // Handle compute capabilities announcement
      eventStore.handleWasmMessage({
        type: 'compute:capabilities:v1:update',
        payload: event.detail,
        metadata: {
          global_context: {
            source: event.detail?.device_id || 'unknown',
            device_id: event.detail?.device_id || 'unknown',
            correlation_id: event.detail?.correlation_id
          }
        },
        timestamp: new Date().toISOString()
      });
    });

    // Listen for compute task events
    window.addEventListener('compute:task', (event: any) => {
      console.warn('[WASM] ✅ Received compute:task event:', {
        status: event.detail?.status,
        task_id: event.detail?.task_id,
        worker_id: event.detail?.worker_id,
        correlation_id: event.detail?.correlation_id
      });

      eventStore.handleWasmMessage({
        type: `compute:dispatch:v1:${event.detail.status}`,
        payload: event.detail,
        metadata: {
          global_context: {
            source: event.detail?.worker_id,
            task_id: event.detail?.task_id,
            correlation_id: event.detail?.correlation_id
          }
        },
        timestamp: new Date().toISOString()
      });
    });

    // Listen for compute worker status
    window.addEventListener('compute:worker:status', () => {
      console.warn('[WASM] ✅ Compute worker ready');
      connectionStore.setConnectionState({
        wasmReady: true,
        connected: true
      });
    });
  }
}
