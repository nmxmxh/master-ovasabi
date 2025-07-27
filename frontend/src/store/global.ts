import { create } from 'zustand';
import { persist, subscribeWithSelector, devtools } from 'zustand/middleware';
import { subscribeToWasmMessages, wasmSendMessage } from '../lib/wasmBridge';
import { merge, cloneDeep, isEmpty, debounce } from 'lodash';
import { useMemo, useCallback } from 'react';

// --- Type Definitions (Communication Standards Compliant) ---
export interface EventEnvelope {
  type: string; // {service}:{action}:v{version}:{state}
  payload?: any;
  metadata: Metadata;
  correlationId?: string;
  timestamp: string; // ISO string with timezone
}

export interface Metadata {
  campaign: CampaignMetadata;
  user: UserMetadata;
  device: DeviceMetadata;
  session: SessionMetadata;
  correlation_id?: string; // Backend expects this field
  [key: string]: any;
}

export interface CampaignMetadata {
  campaignId: number;
  campaignName?: string;
  slug?: string;
  features: string[];
  serviceSpecific?: {
    campaign?: Record<string, any>;
    localization?: {
      scripts?: Record<string, ScriptBlock>;
      scripts_translations?: Record<string, any>;
      scripts_translated?: Record<string, ScriptBlock>;
    };
    [key: string]: any;
  };
  scheduling?: Record<string, any>;
  versioning?: Record<string, any>;
  audit?: Record<string, any>;
  gdpr?: {
    consentRequired: boolean;
    privacyPolicyUrl?: string;
    termsUrl?: string;
    consentGiven?: boolean;
    consentTimestamp?: string; // ISO string with timezone
  };
}

export interface ScriptBlock {
  main_text: string;
  options_title: string;
  options_subtitle: string;
  question_subtitle: string;
  questions: Array<{
    question: string;
    why_this_matters: string;
    options: string[];
    accessibility?: {
      ariaLabel?: string;
      altText?: string;
    };
  }>;
}

export interface UserMetadata {
  userId?: string;
  username?: string;
  privileges?: string[];
  referralCode?: string;
}

export interface DeviceMetadata {
  deviceId: string;
  userAgent?: string;
  userAgentData?: {
    brands?: Array<{ brand: string; version: string }>;
    mobile?: boolean;
    platform?: string;
  };
  language?: string;
  timezone?: string;
  consentGiven: boolean;
  gdprConsentTimestamp?: string; // ISO string with timezone
  gdprConsentRequired?: boolean;
}

export interface SessionMetadata {
  sessionId: string;
  guestId?: string;
  authenticated?: boolean;
}

// --- Connection State ---
interface ConnectionState {
  connected: boolean;
  connecting: boolean;
  lastPing: string; // ISO string with timezone
  reconnectAttempts: number;
  maxReconnectAttempts: number;
  reconnectDelay: number;
  wasmReady: boolean;
}

// --- Generic Feature State ---
interface FeatureState {
  [key: string]: any;
}

// Generic state manager for different features/services
interface ServiceStates {
  [serviceName: string]: FeatureState;
}

// --- Utility Functions ---
const generateId = (): string => {
  return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
};

const getTimezoneAwareTimestamp = (): string => {
  return new Date().toISOString();
};

const getSessionStorage = (key: string): string | null => {
  if (typeof window === 'undefined') return null;
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
};

const setSessionStorage = (key: string, value: string): void => {
  if (typeof window === 'undefined') return;
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // Storage not available
  }
};

// Get modern device info using User-Agent Client Hints API
const getDeviceInfo = () => {
  if (typeof navigator === 'undefined') {
    return {
      userAgent: '',
      userAgentData: undefined,
      language: 'en',
      timezone: 'UTC'
    };
  }

  const userAgentData = (navigator as any).userAgentData;

  return {
    userAgent: navigator.userAgent,
    userAgentData: userAgentData
      ? {
          brands: userAgentData.brands || [],
          mobile: userAgentData.mobile || false,
          platform: userAgentData.platform || ''
        }
      : undefined,
    language: navigator.language,
    timezone: typeof Intl !== 'undefined' ? Intl.DateTimeFormat().resolvedOptions().timeZone : 'UTC'
  };
};

// --- Initial State ---
const createInitialMetadata = (): Metadata => {
  const deviceId = getSessionStorage('device_id') || `device_${generateId()}`;
  const sessionId = `session_${generateId()}`;
  const guestId = getSessionStorage('guest_id') || `guest_${generateId()}`;
  const deviceInfo = getDeviceInfo();

  setSessionStorage('device_id', deviceId);
  setSessionStorage('guest_id', guestId);

  return {
    campaign: {
      campaignId: 0,
      features: [],
      gdpr: {
        consentRequired: false,
        consentGiven: false
      }
    },
    user: {},
    device: {
      deviceId,
      ...deviceInfo,
      consentGiven: false
    },
    session: {
      sessionId,
      guestId,
      authenticated: false
    },
    correlation_id: generateId()
  };
};

// --- Store Interface ---
interface PendingRequestEntry {
  expectedEventType: string;
  resolve: (event: EventEnvelope) => void;
  reject?: (reason?: any) => void;
}

interface GlobalState {
  // Core state
  metadata: Metadata;
  events: EventEnvelope[];
  eventStates: Record<string, string>; // eventType -> current state
  eventPayloads: Record<string, any>; // eventType -> payload for proactive state
  queuedMessages: EventEnvelope[];
  connection: ConnectionState;
  serviceStates: ServiceStates; // Generic state for different services/features
  pendingRequests: Record<string, PendingRequestEntry>;

  // Actions
  setMetadata: (meta: Partial<Metadata>) => void;
  updateMetadata: (updater: (current: Metadata) => Metadata) => void;
  emitEvent: (
    event: Omit<EventEnvelope, 'timestamp' | 'correlationId'>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateEventState: (eventType: string, state: string) => void;
  setConnectionState: (state: Partial<ConnectionState>) => void;
  handleWasmMessage: (msg: any) => void;
  processQueuedMessages: () => void;
  reconnect: () => void;
  clearHistory: () => void;
  reset: () => void;

  // Generic service state actions
  setServiceState: (serviceName: string, state: Partial<FeatureState>) => void;
  updateServiceState: (
    serviceName: string,
    updater: (current: FeatureState) => FeatureState
  ) => void;
  clearServiceState: (serviceName: string) => void;

  // Getters
  getEventsByType: (eventType: string) => EventEnvelope[];
  getLatestEvent: (eventType?: string) => EventEnvelope | undefined;
  getCurrentState: (eventType: string) => string | undefined;
  getServiceState: (serviceName: string) => FeatureState;
  isConnected: () => boolean;
}

// Debounced metadata merger for performance
const debouncedMetadataMerge = debounce(
  (set: any, newMeta: Partial<Metadata>, currentMeta: Metadata) => {
    const mergedMetadata = merge(cloneDeep(currentMeta), newMeta, {
      correlation_id: newMeta.correlation_id || generateId()
    });

    set({ metadata: mergedMetadata }, false, 'setMetadata');
  },
  100
);

// --- Zustand Store ---
export const useGlobalStore = create<GlobalState>()(
  devtools(
    subscribeWithSelector(
      persist(
        (set, get) => ({
          // Initial state
          metadata: createInitialMetadata(),
          events: [],
          eventStates: {},
          queuedMessages: [],
          campaignState: null, // Dedicated field for latest campaign state (legacy, see eventPayloads)
          eventPayloads: {}, // Map of eventType -> payload for proactive state
          connection: {
            connected: false,
            connecting: false,
            lastPing: '', // Don't initialize with current time
            reconnectAttempts: 0,
            maxReconnectAttempts: 5,
            reconnectDelay: 1000,
            wasmReady: false
          },
          serviceStates: {}, // Generic state for all services
          pendingRequests: {},

          // Actions
          setMetadata: meta => {
            if (isEmpty(meta)) return;
            const currentState = get();
            debouncedMetadataMerge(set, meta, currentState.metadata);
          },

          updateMetadata: updater =>
            set(
              state => ({
                metadata: updater(cloneDeep(state.metadata))
              }),
              false,
              'updateMetadata'
            ),

          emitEvent: (event, onResponse) => {
            const state = get();
            const correlationId = generateId();
            // Compute expected success event type
            let expectedEventType = event.type;
            if (event.type.endsWith(':request')) {
              expectedEventType = event.type.replace(/:request$/, ':success');
            }
            const fullEvent: EventEnvelope = {
              ...event,
              timestamp: getTimezoneAwareTimestamp(),
              correlationId,
              metadata: merge(cloneDeep(state.metadata), event.metadata)
            };

            // Store pending request if callback provided
            if (onResponse) {
              set(current => ({
                pendingRequests: {
                  ...current.pendingRequests,
                  [correlationId]: {
                    expectedEventType,
                    resolve: onResponse
                  }
                }
              }));
            }

            // Extract state from event type (follows communication standards)
            const eventState = event.type.split(':').pop() || 'unknown';

            set(
              current => ({
                events: [...current.events, fullEvent],
                eventStates: {
                  ...current.eventStates,
                  [event.type]: eventState
                }
              }),
              false,
              'emitEvent'
            );

            // Send via WASM bridge if connected
            if (state.connection.connected && state.connection.wasmReady) {
              // Create clean message object for WASM bridge
              const wasmMessage = {
                type: fullEvent.type,
                payload: fullEvent.payload || {},
                metadata: fullEvent.metadata || {},
                correlationId: fullEvent.correlationId,
                timestamp: fullEvent.timestamp
              };

              console.log('[Global State] Sending to WASM:', {
                type: wasmMessage.type,
                correlationId: wasmMessage.correlationId,
                hasPayload: !!wasmMessage.payload,
                hasMetadata: !!wasmMessage.metadata
              });

              wasmSendMessage(wasmMessage);
            } else {
              // Queue for later if not connected
              set(
                current => ({
                  queuedMessages: [...current.queuedMessages, fullEvent]
                }),
                false,
                'queueMessage'
              );
            }
          },

          updateEventState: (eventType, state) =>
            set(
              current => ({
                eventStates: {
                  ...current.eventStates,
                  [eventType]: state
                }
              }),
              false,
              'updateEventState'
            ),

          setConnectionState: connectionState =>
            set(
              state => ({
                connection: merge(cloneDeep(state.connection), connectionState)
              }),
              false,
              'setConnectionState'
            ),

          handleWasmMessage: msg => {
            // WASM bridge now handles type conversion, so we expect proper EventEnvelope structure
            const event: EventEnvelope = {
              type: msg.type || 'unknown',
              payload: msg.payload,
              metadata: msg.metadata || {},
              timestamp: msg.timestamp || getTimezoneAwareTimestamp(),
              correlationId: msg.metadata?.correlation_id || msg.correlationId || generateId()
            };

            // Log the received event
            console.log('[Global Store] Received WASM event:', {
              type: event.type,
              hasPayload: !!event.payload,
              hasMetadata: !!event.metadata,
              correlationId: event.correlationId
            });

            // --- Robust request/response: check for pending request match ---
            const state = get();
            if (
              event.correlationId &&
              state.pendingRequests[event.correlationId] &&
              state.pendingRequests[event.correlationId].expectedEventType === event.type
            ) {
              // Call the resolve callback and remove from pending
              state.pendingRequests[event.correlationId].resolve(event);
              set(current => {
                // Defensive: ensure correlationId is string
                const pending = { ...current.pendingRequests };
                if (event.correlationId && typeof event.correlationId === 'string') {
                  delete pending[event.correlationId];
                }
                return { pendingRequests: pending };
              });
              return;
            }

            // Handle echo events for connection heartbeat (with metadata payload)
            if (event.type === 'echo') {
              console.log('[Global Store] Received echo heartbeat:', event.payload);
              get().setConnectionState({
                connected: true,
                lastPing: event.timestamp,
                reconnectAttempts: 0
              });
              return;
            }

            // Handle legacy ping events (fallback)
            if (event.type === 'ping' || event.type === 'system:heartbeat:v1:received') {
              get().setConnectionState({
                connected: true,
                lastPing: event.timestamp,
                reconnectAttempts: 0
              });
              return;
            }

            // Handle connection closed events
            if (
              event.type === 'connection:closed' ||
              event.type === 'system:connection:v1:closed'
            ) {
              console.log('[Global Store] WebSocket connection lost, updating state');
              get().setConnectionState({
                connected: false,
                wasmReady: false
              });
              // Don't trigger immediate reconnection - let the monitoring handle it
              return;
            }

            // Add event to history and update state
            const eventState = event.type.split(':').pop() || 'unknown';

            set(
              current => {
                const update: any = {
                  events: [...current.events, event],
                  eventStates: {
                    ...current.eventStates,
                    [event.type]: eventState
                  },
                  metadata: merge(cloneDeep(current.metadata), event.metadata)
                };
                // Proactively update eventPayloads for any *:v1:success event
                if (event.type && event.type.endsWith(':v1:success')) {
                  update.eventPayloads = {
                    ...current.eventPayloads,
                    [event.type]: event.payload
                  };
                }
                // For backward compatibility, keep campaignState updated
                if (event.type === 'campaign:state:v1:success') {
                  update.campaignState = event.payload;
                }
                return update;
              },
              false,
              'handleWasmMessage'
            );
            // --- Convenience selectors for generic eventPayloads ---
          },

          processQueuedMessages: () => {
            const state = get();
            if (!state.connection.connected || !state.connection.wasmReady) return;

            state.queuedMessages.forEach(event => {
              // Create clean message object for WASM bridge
              const wasmMessage = {
                type: event.type,
                payload: event.payload || {},
                metadata: event.metadata || {},
                correlationId: event.correlationId,
                timestamp: event.timestamp
              };

              console.log('[Global State] Sending queued message to WASM:', {
                type: wasmMessage.type,
                correlationId: wasmMessage.correlationId
              });

              wasmSendMessage(wasmMessage);
            });

            set({ queuedMessages: [] }, false, 'processQueuedMessages');
          },

          reconnect: () => {
            const state = get();

            // Check if we should even attempt to reconnect
            if (state.connection.connecting) {
              console.log('[Global Store] Already attempting to reconnect, skipping');
              return;
            }

            if (state.connection.reconnectAttempts >= state.connection.maxReconnectAttempts) {
              console.log('[Global Store] Max reconnection attempts reached, giving up');
              return;
            }

            // Check if we're actually disconnected
            if (state.connection.connected && state.connection.wasmReady) {
              console.log('[Global Store] Already connected, skipping reconnection');
              return;
            }

            console.log(
              `[Global Store] Attempting reconnection (attempt ${state.connection.reconnectAttempts + 1}/${state.connection.maxReconnectAttempts})`
            );

            get().setConnectionState({
              connecting: true,
              reconnectAttempts: state.connection.reconnectAttempts + 1
            });

            const delay = Math.min(
              state.connection.reconnectDelay * Math.pow(2, state.connection.reconnectAttempts),
              30000 // Max 30 seconds
            );

            setTimeout(() => {
              // Use WASM reconnection method if available
              if (typeof window !== 'undefined' && (window as any).reconnectWebSocket) {
                console.log('[Global Store] Triggering WASM WebSocket reconnection');
                (window as any).reconnectWebSocket();
              } else if (typeof window !== 'undefined' && (window as any).initWebSocket) {
                console.log('[Global Store] Fallback to initWebSocket');
                (window as any).initWebSocket();
              } else {
                console.warn('[Global Store] No WASM reconnection method available');
              }

              get().setConnectionState({ connecting: false });
            }, delay);
          },

          clearHistory: () => set({ events: [] }, false, 'clearHistory'),

          // Generic service state actions
          setServiceState: (serviceName, serviceState) =>
            set(
              state => ({
                serviceStates: {
                  ...state.serviceStates,
                  [serviceName]: merge(
                    cloneDeep(state.serviceStates[serviceName] || {}),
                    serviceState
                  )
                }
              }),
              false,
              'setServiceState'
            ),

          updateServiceState: (serviceName, updater) =>
            set(
              state => ({
                serviceStates: {
                  ...state.serviceStates,
                  [serviceName]: updater(cloneDeep(state.serviceStates[serviceName] || {}))
                }
              }),
              false,
              'updateServiceState'
            ),

          clearServiceState: serviceName =>
            set(
              state => ({
                serviceStates: {
                  ...state.serviceStates,
                  [serviceName]: {}
                }
              }),
              false,
              'clearServiceState'
            ),

          getServiceState: serviceName => {
            const state = get();
            return state.serviceStates[serviceName] || {};
          },

          reset: () =>
            set(
              {
                metadata: createInitialMetadata(),
                events: [],
                eventStates: {},
                queuedMessages: [],
                serviceStates: {}, // Reset all service states
                connection: {
                  connected: false,
                  connecting: false,
                  lastPing: '', // Don't initialize with current time
                  reconnectAttempts: 0,
                  maxReconnectAttempts: 5,
                  reconnectDelay: 1000,
                  wasmReady: false
                }
              },
              false,
              'reset'
            ),

          // Getters
          getEventsByType: eventType => {
            const { events } = get();
            return events.filter(event => event.type === eventType);
          },

          getLatestEvent: eventType => {
            const { events } = get();
            const filtered = eventType ? events.filter(event => event.type === eventType) : events;
            return filtered[filtered.length - 1];
          },

          getCurrentState: eventType => {
            const { eventStates } = get();
            return eventStates[eventType];
          },

          isConnected: () => {
            const { connection } = get();
            return connection.connected && connection.wasmReady;
          }
        }),
        {
          name: 'ovasabi-global-state',
          partialize: state => ({
            metadata: state.metadata,
            events: state.events.slice(-50), // Keep last 50 events
            eventStates: state.eventStates
          })
        }
      )
    ),
    { name: 'GlobalStore' }
  )
);

// --- WASM Bridge Integration ---
let wasmUnsubscribe: (() => void) | null = null;
let pendingWasmReady = false;
let pendingMessages: any[] = [];
let initializationLock = false; // Prevent multiple initializations

// Initialize WASM handlers immediately when this module loads
if (typeof window !== 'undefined') {
  // Override the default handlers from main.ts immediately
  window.onWasmReady = () => {
    console.log('[Global State] WASM Ready (before store init)');
    pendingWasmReady = true;
  };

  window.onWasmMessage = (msg: any) => {
    console.log('[Global State] WASM Message (before store init)', msg);
    pendingMessages.push(msg);
  };
}

export function initializeGlobalState() {
  // Prevent multiple initializations (useful in StrictMode)
  if (initializationLock) {
    console.log('[Global State] Already initialized, skipping duplicate initialization');
    return () => {}; // Return empty cleanup function
  }

  initializationLock = true;
  console.log('[Global State] Initializing global state...');

  const store = useGlobalStore.getState();

  // Subscribe to WASM messages
  if (wasmUnsubscribe) {
    wasmUnsubscribe();
  }

  wasmUnsubscribe = subscribeToWasmMessages(msg => {
    store.handleWasmMessage(msg);
  });

  // Setup WASM ready handler (now that store is ready)
  if (typeof window !== 'undefined') {
    window.onWasmReady = () => {
      console.log('[Global State] WASM Ready');
      store.setConnectionState({
        wasmReady: true,
        connected: true,
        lastPing: getTimezoneAwareTimestamp(),
        reconnectAttempts: 0
      });
      store.processQueuedMessages();
    };

    window.onWasmMessage = (msg: any) => {
      store.handleWasmMessage(msg);
    };

    // Handle any pending WASM ready state
    if (pendingWasmReady) {
      console.log('[Global State] Processing pending WASM ready');
      store.setConnectionState({
        wasmReady: true,
        connected: true,
        lastPing: getTimezoneAwareTimestamp(),
        reconnectAttempts: 0
      });
      pendingWasmReady = false;
    }

    // Handle any pending messages
    if (pendingMessages.length > 0) {
      console.log('[Global State] Processing pending WASM messages', pendingMessages.length);
      pendingMessages.forEach(msg => store.handleWasmMessage(msg));
      pendingMessages = [];
    }
  }

  // Setup connection monitoring with timezone-aware timestamps
  const connectionMonitor = setInterval(() => {
    const state = useGlobalStore.getState();

    // Only start monitoring after WASM is ready and we have received at least one ping
    if (!state.connection.wasmReady || !state.connection.lastPing || !state.connection.connected) {
      return;
    }

    const now = new Date();
    const lastPing = new Date(state.connection.lastPing);
    const timeSinceLastPing = now.getTime() - lastPing.getTime();

    // Consider disconnected if no ping for 60 seconds (increased from 30)
    if (timeSinceLastPing > 60000) {
      console.warn('[Global State] Connection timeout detected, marking as disconnected');
      state.setConnectionState({
        connected: false,
        wasmReady: false // Also mark WASM as not ready to trigger full reconnection
      });

      // Don't immediately attempt reconnection - let the user-triggered events handle it
      // This prevents the monitoring from causing a reconnection loop
    }
  }, 15000); // Check every 15 seconds (increased from 10)

  // Setup window visibility change and focus event handlers for reconnection
  // Use debounced handlers to prevent aggressive reconnection attempts
  let reconnectionTimeout: NodeJS.Timeout | null = null;

  const scheduleReconnection = (reason: string) => {
    const state = useGlobalStore.getState();

    // Don't schedule if already connected or already scheduled
    if (state.connection.connected || reconnectionTimeout) {
      return;
    }

    // Don't reconnect if we've exceeded max attempts
    if (state.connection.reconnectAttempts >= state.connection.maxReconnectAttempts) {
      console.log('[Global State] Max reconnection attempts reached, not scheduling reconnection');
      return;
    }

    console.log(`[Global State] Scheduling reconnection due to: ${reason}`);

    reconnectionTimeout = setTimeout(() => {
      reconnectionTimeout = null;
      const currentState = useGlobalStore.getState();

      // Double-check we're still disconnected before attempting reconnection
      if (!currentState.connection.connected || !currentState.connection.wasmReady) {
        console.log(`[Global State] Executing scheduled reconnection for: ${reason}`);
        currentState.reconnect();
      }
    }, 2000); // 2 second delay to prevent rapid reconnection attempts
  };

  const handleVisibilityChange = () => {
    if (!document.hidden) {
      console.log('[Global State] Window became visible, checking connection...');
      scheduleReconnection('visibility change');
    }
  };

  const handleWindowFocus = () => {
    console.log('[Global State] Window gained focus, checking connection...');
    scheduleReconnection('window focus');
  };

  const handleOnline = () => {
    console.log('[Global State] Network came online, checking connection...');
    scheduleReconnection('network online');
  };

  // Add event listeners for window state changes
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleVisibilityChange);
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('focus', handleWindowFocus);
    window.addEventListener('online', handleOnline);
  }

  // Cleanup function
  return () => {
    console.log('[Global State] Cleaning up global state...');
    initializationLock = false; // Allow re-initialization

    if (wasmUnsubscribe) {
      wasmUnsubscribe();
      wasmUnsubscribe = null;
    }
    clearInterval(connectionMonitor);

    // Clear any pending reconnection timeout
    if (reconnectionTimeout) {
      clearTimeout(reconnectionTimeout);
      reconnectionTimeout = null;
    }

    // Remove event listeners
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    }

    if (typeof window !== 'undefined') {
      window.removeEventListener('focus', handleWindowFocus);
      window.removeEventListener('online', handleOnline);
    }
  };
}

// --- Utility Hooks ---
export function useEventHistory(eventType?: string, limit?: number) {
  const events = useGlobalStore(state => state.events);

  return useMemo(() => {
    const filtered = eventType ? events.filter(e => e.type === eventType) : events;
    return limit ? filtered.slice(-limit) : filtered;
  }, [events, eventType, limit]);
}

export function useEventState(eventType: string) {
  return useGlobalStore(state => state.eventStates[eventType]);
}

export function useMetadata() {
  const metadata = useGlobalStore(state => state.metadata);
  const setMetadata = useGlobalStore(state => state.setMetadata);
  const updateMetadata = useGlobalStore(state => state.updateMetadata);

  return useMemo(
    () => ({
      metadata,
      setMetadata,
      updateMetadata
    }),
    [metadata, setMetadata, updateMetadata]
  );
}

export function useConnectionStatus() {
  const connected = useGlobalStore(state => state.connection.connected);
  const connecting = useGlobalStore(state => state.connection.connecting);
  const wasmReady = useGlobalStore(state => state.connection.wasmReady);
  const lastPing = useGlobalStore(state => state.connection.lastPing);
  const reconnectAttempts = useGlobalStore(state => state.connection.reconnectAttempts);

  return useMemo(() => {
    const isConnected = connected && wasmReady;
    return {
      connected,
      connecting,
      wasmReady,
      lastPing,
      reconnectAttempts,
      isConnected
    };
  }, [connected, connecting, wasmReady, lastPing, reconnectAttempts]);
}

export function useEmitEvent() {
  return useGlobalStore(state => state.emitEvent);
}

// --- Generic Service State Hooks ---
export function useServiceState<T extends FeatureState = FeatureState>(serviceName: string) {
  // Use a simple, stable selector
  const serviceState = useGlobalStore(state => state.serviceStates[serviceName]);

  // Create stable selectors for the actions
  const setServiceState = useGlobalStore(state => state.setServiceState);
  const updateServiceState = useGlobalStore(state => state.updateServiceState);
  const clearServiceState = useGlobalStore(state => state.clearServiceState);

  // Return a simple object without complex memoization
  return {
    state: (serviceState || {}) as T,
    setState: useCallback(
      (newState: Partial<T>) => setServiceState(serviceName, newState),
      [setServiceState, serviceName]
    ),
    updateState: useCallback(
      (updater: (current: T) => T) =>
        updateServiceState(
          serviceName,
          updater as unknown as (current: FeatureState) => FeatureState
        ),
      [updateServiceState, serviceName]
    ),
    clearState: useCallback(() => clearServiceState(serviceName), [clearServiceState, serviceName])
  };
}

// Convenience hooks for common services
export function useSearchState() {
  return useServiceState<{
    currentQuery?: string | null;
    loading?: boolean;
    results?: any[];
    error?: string | null;
    lastSearchTimestamp?: string | null;
  }>('search');
}

export function useNexusState() {
  return useServiceState('nexus');
}

export function useCampaignState() {
  return useServiceState('campaign');
}

// --- Type exports ---
export type { GlobalState, FeatureState, ServiceStates };
