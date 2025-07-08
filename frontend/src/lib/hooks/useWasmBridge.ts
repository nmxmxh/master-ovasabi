// React hook for production-grade WASM WebSocket bridge communication
// Handles concurrent messages, queueing, and optional GPU/AI task offload via WASM

import { useEffect, useState, useCallback } from 'react';
import { wasmSendMessage, onWasmMessage } from '../wasmBridge';
import type { WasmBridgeMessage } from '../wasmBridge';
import type { EventRequest } from '../../../protos/nexus/v1/nexus';
import { useGlobalStore } from '../../store/global';

export interface UseWasmBridgeOptions {
  onMessage?: (msg: WasmBridgeMessage) => void;
  onError?: (err: any) => void;
  autoConnect?: boolean;
}

// Singleton bridge state to prevent reinitialization
const bridgeState = {
  initialized: false,
  sendQueue: [] as WasmBridgeMessage[],
  isReady: false,
  setReady: (_: boolean) => {},
  setConnected: (_: boolean) => {},
  setLastMessage: (_: WasmBridgeMessage | null) => {},
  onMessage: undefined as ((msg: WasmBridgeMessage) => void) | undefined
};

export function useWasmBridge({ onMessage, autoConnect = true }: UseWasmBridgeOptions) {
  const [connected, setConnected] = useState(false);
  const [ready, setReady] = useState(false); // explicit ready state
  const [lastMessage, setLastMessage] = useState<WasmBridgeMessage | null>(null);
  // Store setters in singleton for bridge callbacks
  bridgeState.setReady = setReady;
  bridgeState.setConnected = setConnected;
  bridgeState.setLastMessage = setLastMessage;
  bridgeState.onMessage = onMessage;

  // Register message handler
  useEffect(() => {
    if (!autoConnect) return;
    if (bridgeState.initialized) return;
    bridgeState.initialized = true;
    console.log('[WasmBridge] Initializing WASM bridge (singleton)...');
    onWasmMessage((msg: WasmBridgeMessage) => {
      console.log('[WasmBridge] Received message:', msg);
      bridgeState.setLastMessage(msg);
      if (bridgeState.onMessage) bridgeState.onMessage(msg);
      if (msg.type === 'gpu_result' && msg.taskId) {
        // ...handle GPU result...
      }
    });
    (window as any).onWasmReady = () => {
      bridgeState.isReady = true;
      bridgeState.setReady(true);
      bridgeState.setConnected(true);
      console.log(
        '[WasmBridge] WASM bridge is READY. Flushing queued messages:',
        bridgeState.sendQueue.length
      );
      // Flush queued messages
      bridgeState.sendQueue.forEach(m => {
        if (typeof window.wasmSendMessage === 'function') {
          console.log('[WasmBridge] Flushing queued message:', m);
          wasmSendMessage(m);
        } else {
          console.error(
            '[WasmBridge] wasmSendMessage is not available on window during flush! Message not sent:',
            m
          );
        }
      });
      bridgeState.sendQueue = [];
    };
    // Debug: check if WASM is already ready (hot reload/dev)
    if ((window as any).wasmReady) {
      console.log('[WasmBridge] WASM bridge was already ready on mount.');
      (window as any).onWasmReady();
    }
    return () => {
      // Do not reset bridgeState.initialized, keep singleton
    };
  }, [autoConnect]);

  // Send message, queue if not ready
  const send = useCallback((msg: WasmBridgeMessage) => {
    if (bridgeState.isReady) {
      if (typeof window.wasmSendMessage === 'function') {
        console.log('[WasmBridge] Sending message (bridge ready):', msg);
        wasmSendMessage(msg);
      } else {
        console.error(
          '[WasmBridge] wasmSendMessage is not available on window! Message not sent:',
          msg
        );
      }
    } else {
      console.log('[WasmBridge] Bridge not ready, queueing message:', msg);
      bridgeState.sendQueue.push(msg);
    }
  }, []);

  // Example: offload a GPU/AI task via WASM
  const runGpuTask = useCallback(
    (task: Record<string, any>) => {
      send({ type: 'gpu_task', ...task });
    },
    [send]
  );

  // Load canonical event types from backend/registry and store in Zustand
  useEffect(() => {
    async function loadEventTypes() {
      try {
        const res = await fetch('/config/service_registration.json');
        if (!res.ok) throw new Error('Failed to fetch event types');
        const data = await res.json();
        // Canonical states per standards
        const states = ['requested', 'started', 'success', 'failed', 'completed'];
        // Generate canonical event types for all services/actions/states
        const eventTypes = Array.isArray(data)
          ? data.flatMap(service => {
              const serviceName = service.name;
              const version = service.version;
              const actionMap = service.action_map || {};
              return Object.keys(actionMap).flatMap(action =>
                states.map(state => `${serviceName}:${action}:${version}:${state}`)
              );
            })
          : [];
        if (useGlobalStore.getState().setEventTypes) {
          useGlobalStore.getState().setEventTypes(eventTypes);
        }
        console.log('[WasmBridge] Loaded canonical service event types:', eventTypes);
      } catch (e) {
        console.warn('[WasmBridge] Could not load canonical event types:', e);
      }
    }
    loadEventTypes();
  }, []);

  // --- Canonical Nexus event helpers ---
  /**
   * Send a canonical Nexus event to the backend (matches EventRequest proto contract)
   * @param event Partial<EventRequest> (eventType required, payload optional, others auto-filled if missing)
   */
  // Helper for UUID generation using crypto.randomUUID() with fallback
  const getEventId = () => {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return crypto.randomUUID();
    }
    // Fallback: simple random string (not RFC4122, but unique enough for events)
    return 'evt_' + Math.random().toString(36).slice(2) + Date.now();
  };

  // Helper to build canonical metadata for backend/proto compliance

  const sendNexusEvent = useCallback(
    (event: Partial<EventRequest> & { eventType: string }) => {
      // Always get latest metadata from global store and merge with event.metadata
      const globalMetadata = useGlobalStore.getState().metadata;
      const canonicalEvent: EventRequest = {
        eventId: event.eventId || getEventId(),
        eventType: event.eventType,
        entityId: event.entityId || 'frontend',
        campaignId: event.campaignId || 'ovasabi_website',
        payload: event.payload || {},
        metadata: buildCanonicalMetadata(globalMetadata, event.metadata)
      };
      // Set top-level type to canonical eventType for WASM compatibility
      send({ type: canonicalEvent.eventType, ...canonicalEvent });
    },
    [send]
  );

  /**
   * Helper to build a canonical Nexus EventRequest (does not send)
   */
  const buildNexusEvent = (
    eventType: string,
    payload?: any,
    entityId?: string,
    campaignId?: string,
    metadata?: any
  ): EventRequest => ({
    eventId: getEventId(),
    eventType,
    entityId: entityId || 'frontend',
    campaignId: campaignId || '0',
    payload: payload || {},
    metadata
  });

  // Listen for all incoming events/messages (optionally filter by type)
  // Usage: pass onMessage to hook, or use lastMessage

  return {
    connected,
    ready,
    lastMessage,
    send,
    runGpuTask,
    sendNexusEvent,
    buildNexusEvent,
    buildCanonicalMetadata
  };
}

export function buildCanonicalMetadata(meta: any, eventMeta?: any): any {
  return {
    features: (eventMeta?.features ?? meta?.features) || [],
    tags: (eventMeta?.tags ?? meta?.tags) || [],
    aiConfidence: (eventMeta?.aiConfidence ?? meta?.aiConfidence) || 0,
    embeddingId: (eventMeta?.embeddingId ?? meta?.embeddingId) || '',
    // Add any other required fields with safe defaults here
    ...meta,
    ...eventMeta
  };
}
