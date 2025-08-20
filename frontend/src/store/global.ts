import { create } from 'zustand';
import { persist, subscribeWithSelector, devtools } from 'zustand/middleware';
import { subscribeToWasmMessages, wasmSendMessage } from '../lib/wasmBridge';
import { merge, cloneDeep, isEmpty, debounce } from 'lodash';
import { useMemo, useCallback } from 'react';
// --- WASM Readiness and Message Queue Globals ---
// These are used for WASM readiness and message queuing before Zustand store is initialized

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
  // GPU and performance information
  gpuCapabilities?: any; // Will be populated by WASM GPU Bridge
  wasmGPUBridge?: {
    initialized: boolean;
    backend: string;
    workerCount: number;
    version: string;
  };
  gpuDetectedAt?: string;
  [key: string]: any; // Allow additional device properties
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
  wasmFunctions?: {
    initWebGPU: boolean;
    runGPUCompute: boolean;
    getGPUMetricsBuffer: boolean;
    [key: string]: boolean;
  };
}

// --- Generic Feature State ---
interface FeatureState {
  [key: string]: any;
}

// Generic state manager for different features/services
interface ServiceStates {
  [serviceName: string]: FeatureState;
}

// --- Media Streaming State ---
export interface MediaStreamingState {
  connected: boolean;
  peerId: string;
  streamInfo?: any;
  error?: string;
  lastConnectAttempt?: string;
}

// --- Utility Functions ---
const generateId = (): string => {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
    const array = new Uint8Array(16);
    window.crypto.getRandomValues(array);
    // Convert to base64, remove non-alphanumeric chars, and trim length
    return btoa(String.fromCharCode(...array))
      .replace(/[^a-zA-Z0-9]/g, '')
      .substring(0, 24);
  }
  // Fallback to Math.random if crypto is unavailable
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
  // Always use window.userID as the source of truth
  // Always use the guest ID from localStorage, as managed by WASM. Fallback to window.userID, then generate if missing.
  let guestId = '';
  if (typeof window !== 'undefined') {
    guestId = localStorage.getItem('guest_id') || window.userID || '';
    if (!guestId) {
      guestId = `guest_${generateId()}`;
      localStorage.setItem('guest_id', guestId);
    }
    window.userID = guestId;
  } else {
    guestId = `guest_${generateId()}`;
  }
  const deviceInfo = getDeviceInfo();

  setSessionStorage('device_id', deviceId);
  // Do not set guest_id in sessionStorage; WASM manages userID

  return {
    campaign: {
      campaignId: 0,
      features: [],
      gdpr: {
        consentRequired: false,
        consentGiven: false
      }
    },
    user: {
      userId: guestId
    },
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
  // Selectors for UI integration
  userId?: string;
  campaignId?: string | number;
  lastEvent?: EventEnvelope;
  messageLog?: string[];
  // Core state
  metadata: Metadata;
  events: EventEnvelope[];
  eventStates: Record<string, string>; // eventType -> current state
  eventPayloads: Record<string, any>; // eventType -> payload for proactive state
  queuedMessages: EventEnvelope[];
  connection: ConnectionState;
  serviceStates: ServiceStates; // Generic state for different services/features
  pendingRequests: Record<string, PendingRequestEntry>;
  campaignState?: any; // Added for campaign state integration
  mediaStreaming?: MediaStreamingState;

  // Actions
  setMetadata: (meta: Partial<Metadata>) => void;
  updateMetadata: (updater: (current: Metadata) => Metadata) => void;
  emitEvent: (
    event: Omit<EventEnvelope, 'timestamp' | 'correlationId'>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateEventState: (eventType: string, state: string) => void;
  setConnectionState: (state: Partial<ConnectionState>) => void;
  setWasmFunctions: (funcs: { [key: string]: boolean }) => void;
  handleWasmMessage: (msg: any) => void;
  processQueuedMessages: () => void;
  reconnect: () => void;
  clearHistory: () => void;
  reset: () => void;
  switchCampaign: (
    campaignId: number,
    slug?: string,
    onResponse?: (event: EventEnvelope) => void
  ) => void;

  // Campaign update actions
  updateCampaign: (
    updates: Record<string, any>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaignFeatures: (
    features: string[],
    action?: 'add' | 'remove' | 'set',
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaignConfig: (
    configType: 'ui_content' | 'scripts' | 'communication',
    config: Record<string, any>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;

  // Generic service state actions
  setServiceState: (serviceName: string, state: Partial<FeatureState>) => void;
  updateServiceState: (
    serviceName: string,
    updater: (current: FeatureState) => FeatureState
  ) => void;
  clearServiceState: (serviceName: string) => void;

  // Media streaming actions
  setMediaStreamingState: (state: Partial<MediaStreamingState>) => void;
  clearMediaStreamingState: () => void;

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
          // Selectors for UI integration
          get userId(): string {
            return (
              (this.metadata?.user?.userId as string) ||
              (this.metadata?.session?.guestId as string) ||
              ''
            );
          },
          get campaignId(): string | number {
            return (
              (this.metadata?.campaign?.slug as string) ||
              (this.metadata?.campaign?.campaignId as number) ||
              ''
            );
          },
          get lastEvent(): EventEnvelope | undefined {
            const events = this.events as EventEnvelope[];
            return events.length > 0 ? events[events.length - 1] : undefined;
          },
          get messageLog(): string[] {
            const events = this.events as EventEnvelope[];
            return events
              .slice(-50)
              .map(e => `[${e.timestamp}] ${e.type} ${e.payload ? JSON.stringify(e.payload) : ''}`);
          },
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
            wasmReady: false,
            wasmFunctions: {
              initWebGPU: false,
              runGPUCompute: false,
              getGPUMetricsBuffer: false
            }
          },
          serviceStates: {}, // Generic state for all services
          pendingRequests: {},
          mediaStreaming: {
            connected: false,
            peerId: '',
            streamInfo: null,
            error: undefined,
            lastConnectAttempt: ''
          },

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
          // Switch active campaign and emit state request
          switchCampaign: (
            campaignId: number,
            slug?: string,
            onResponse?: (event: EventEnvelope) => void
          ) => {
            const state = get();
            set(
              current => ({
                metadata: {
                  ...current.metadata,
                  campaign: {
                    ...current.metadata.campaign,
                    campaignId,
                    slug: slug || current.metadata.campaign.slug
                  }
                }
              }),
              false,
              'switchCampaign'
            );

            // Trigger WebSocket reconnection with new campaign ID
            if (
              typeof window !== 'undefined' &&
              typeof (window as any).reconnectWebSocket === 'function'
            ) {
              console.log(
                `[Global Store] Triggering WebSocket reconnection for campaign ${campaignId}`
              );
              (window as any).reconnectWebSocket();
            }

            // Only emit campaign:state:request if not already pending and payload is not null/empty
            const alreadyPending =
              state.pendingRequests &&
              Object.values(state.pendingRequests).some(
                req => req.expectedEventType === 'campaign:state:v1:success'
              );
            if (!alreadyPending) {
              state.emitEvent(
                {
                  type: 'campaign:state:v1:request',
                  payload: {}, // Always use empty object, never null
                  metadata: {
                    ...state.metadata,
                    campaign: {
                      ...state.metadata.campaign,
                      campaignId,
                      slug: slug || state.metadata.campaign.slug
                    }
                  }
                },
                onResponse
              );
            }
          },

          // Direct campaign update function
          updateCampaign: (
            updates: Record<string, any>,
            onResponse?: (event: EventEnvelope) => void
          ) => {
            const state = get();
            const campaignId =
              state.metadata.campaign.slug || state.metadata.campaign.campaignId.toString();

            state.emitEvent(
              {
                type: 'campaign:update:v1:requested',
                payload: {
                  campaignId,
                  updates
                },
                metadata: state.metadata
              },
              onResponse
            );
          },

          // Update campaign features
          updateCampaignFeatures: (
            features: string[],
            action: 'add' | 'remove' | 'set' = 'set',
            onResponse?: (event: EventEnvelope) => void
          ) => {
            const state = get();
            const campaignId =
              state.metadata.campaign.slug || state.metadata.campaign.campaignId.toString();

            state.emitEvent(
              {
                type: 'campaign:feature:v1:requested',
                payload: {
                  campaignId,
                  features,
                  action
                },
                metadata: state.metadata
              },
              onResponse
            );
          },

          // Update campaign configuration (UI content, scripts, etc.)
          updateCampaignConfig: (
            configType: 'ui_content' | 'scripts' | 'communication',
            config: Record<string, any>,
            onResponse?: (event: EventEnvelope) => void
          ) => {
            const state = get();
            const campaignId =
              state.metadata.campaign.slug || state.metadata.campaign.campaignId.toString();

            state.emitEvent(
              {
                type: 'campaign:config:v1:requested',
                payload: {
                  campaignId,
                  configType,
                  config
                },
                metadata: state.metadata
              },
              onResponse
            );
          },

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
            if (state.connection.wasmReady) {
              // Create clean message object for WASM bridge
              const wasmMessage = {
                type: fullEvent.type,
                payload: fullEvent.payload || {},
                metadata: fullEvent.metadata || {},
                correlationId: fullEvent.correlationId,
                timestamp: fullEvent.timestamp
              };

              console.log('FULLLLL EVENT >>>>>>', fullEvent);

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

          setWasmFunctions: funcs =>
            set(
              state =>
                ({
                  connection: {
                    ...state.connection,
                    wasmFunctions: {
                      ...state.connection.wasmFunctions,
                      ...funcs
                    }
                  }
                }) as Partial<GlobalState>,
              false,
              'setWasmFunctions'
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

            // Handle health events for service status updates
            if (event.type.includes(':health:v1:')) {
              console.log('[Global Store] Processing health event:', event.type, event.payload);

              // Extract service name from health event type (format: service:health:v1:state)
              const serviceName = event.type.split(':')[0];
              const healthState = event.type.split(':').pop() || 'unknown';

              if (event.payload && serviceName) {
                // Update service state with health information
                const currentServiceState = get().serviceStates[serviceName] || {};
                const healthData = {
                  ...currentServiceState,
                  health: {
                    status:
                      event.payload.status ||
                      (healthState === 'success'
                        ? 'healthy'
                        : healthState === 'failed'
                          ? 'down'
                          : 'unknown'),
                    responseTime: event.payload.response_time || 0,
                    lastCheck: event.payload.checked_at || Date.now(),
                    dependencies: event.payload.dependencies || {},
                    metrics: event.payload.metrics || {},
                    errorMessage: event.payload.error_message
                  }
                };

                set(
                  current => ({
                    serviceStates: {
                      ...current.serviceStates,
                      [serviceName]: healthData
                    }
                  }),
                  false,
                  'healthUpdate'
                );

                console.log(`[Global Store] Updated health for ${serviceName}:`, healthData.health);
              }
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

                // Enhanced campaign state handling
                if (
                  event.type === 'campaign:state:v1:success' ||
                  event.type === 'campaign:update:v1:success' ||
                  event.type === 'campaign:feature:v1:success' ||
                  event.type === 'campaign:config:v1:success'
                ) {
                  // Update campaignState for backward compatibility
                  update.campaignState = event.payload;

                  // Update campaign service state
                  const currentCampaignState = current.serviceStates.campaign || {};
                  update.serviceStates = {
                    ...current.serviceStates,
                    campaign: merge(cloneDeep(currentCampaignState), event.payload || {})
                  };

                  // Update metadata.campaign.serviceSpecific if payload contains campaign data
                  if (event.payload && typeof event.payload === 'object') {
                    const updatedMetadata = cloneDeep(update.metadata);
                    if (!updatedMetadata.campaign.serviceSpecific) {
                      updatedMetadata.campaign.serviceSpecific = {};
                    }
                    if (!updatedMetadata.campaign.serviceSpecific.campaign) {
                      updatedMetadata.campaign.serviceSpecific.campaign = {};
                    }

                    // Merge campaign state into metadata
                    updatedMetadata.campaign.serviceSpecific.campaign = merge(
                      updatedMetadata.campaign.serviceSpecific.campaign,
                      event.payload
                    );

                    // Update features array if present in payload
                    if (event.payload.features && Array.isArray(event.payload.features)) {
                      updatedMetadata.campaign.features = event.payload.features;
                    }

                    update.metadata = updatedMetadata;
                  }
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
            // --- Architectural comment: Only WASM should manage actual WebSocket connection/reconnection. Frontend must only request via window.reconnectWebSocket and never call initWebSocket directly. ---
            // Defensive: Prevent redundant reconnection attempts
            if (state.connection.connecting) {
              console.log('[Global Store] Already attempting to reconnect, skipping');
              return;
            }
            if (state.connection.reconnectAttempts >= state.connection.maxReconnectAttempts) {
              console.log('[Global Store] Max reconnection attempts reached, giving up');
              return;
            }
            if (state.connection.connected && state.connection.wasmReady) {
              console.log('[Global Store] Already connected, skipping reconnection');
              return;
            }
            if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
              console.warn('[Global Store] Shutdown in progress, reconnection blocked.');
              return;
            }
            // Redundancy prevention is handled by WASM (wsReconnectInProgress). Do not use window.__ovasabiReconnectionScheduled.
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
              if (
                typeof window !== 'undefined' &&
                typeof window.reconnectWebSocket === 'function'
              ) {
                console.log(
                  '[Global Store] Triggering WASM WebSocket reconnection via reconnectWebSocket'
                );
                window.reconnectWebSocket();
              } else {
                console.warn('[Global Store] No WASM reconnection method available');
              }
              get().setConnectionState({ connecting: false });
              // No need to reset any JS-side reconnection flag; WASM handles redundancy guard.
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

          // --- Media Streaming Actions ---
          setMediaStreamingState: (state: Partial<MediaStreamingState>) =>
            set(
              current => {
                const prev = current.mediaStreaming || {
                  connected: false,
                  peerId: '',
                  streamInfo: null,
                  error: undefined,
                  lastConnectAttempt: ''
                };
                return {
                  mediaStreaming: {
                    connected: state.connected !== undefined ? state.connected : prev.connected,
                    peerId: state.peerId !== undefined ? state.peerId : prev.peerId,
                    streamInfo: state.streamInfo !== undefined ? state.streamInfo : prev.streamInfo,
                    error: state.error !== undefined ? state.error : prev.error,
                    lastConnectAttempt:
                      state.lastConnectAttempt !== undefined
                        ? state.lastConnectAttempt
                        : prev.lastConnectAttempt
                  }
                };
              },
              false,
              'setMediaStreamingState'
            ),
          clearMediaStreamingState: () =>
            set(
              {
                mediaStreaming: {
                  connected: false,
                  peerId: '',
                  streamInfo: null,
                  error: undefined,
                  lastConnectAttempt: ''
                }
              },
              false,
              'clearMediaStreamingState'
            ),

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
                },
                mediaStreaming: {
                  connected: false,
                  peerId: '',
                  streamInfo: null,
                  error: undefined,
                  lastConnectAttempt: ''
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
            eventStates: state.eventStates,
            mediaStreaming: state.mediaStreaming
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
  // Accept status object from WASM (wasmReady, connected)
  window.onWasmReady = (status?: { wasmReady?: boolean; connected?: boolean }) => {
    console.log('[Global State] WASM Ready (before store init)', status);
    if (status && typeof status === 'object') {
      pendingWasmReady = true;
      // Store status for later use
      window.__pendingWasmStatus = status;
    } else {
      pendingWasmReady = true;
      window.__pendingWasmStatus = { wasmReady: true, connected: true };
    }
  };

  window.onWasmMessage = (msg: any) => {
    // Integration: update campaign state from WASM/WebSocket events
    if (msg.type === 'campaign:state:v1:success' || msg.type === 'campaign:state:v1:completed') {
      useGlobalStore.getState().updateMetadata(current => ({
        ...current,
        campaign: {
          ...current.campaign,
          ...msg.payload
        }
      }));
      useGlobalStore.setState({ campaignState: msg.payload });
    }
    // Existing logic: keep pending messages for initialization
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
    window.onWasmReady = (status?: { wasmReady?: boolean; connected?: boolean }) => {
      console.log('[Global State] WASM Ready', status);
      const ready =
        status && typeof status === 'object' ? status : { wasmReady: true, connected: true };
      store.setConnectionState({
        wasmReady: !!ready.wasmReady,
        connected: !!ready.connected,
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
      const ready = window.__pendingWasmStatus || { wasmReady: true, connected: true };
      store.setConnectionState({
        wasmReady: !!ready.wasmReady,
        connected: !!ready.connected,
        lastPing: getTimezoneAwareTimestamp(),
        reconnectAttempts: 0
      });
      store.processQueuedMessages();
      pendingWasmReady = false;
      window.__pendingWasmStatus = undefined;
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
    // Block reconnection if shutdown is in progress
    if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
      console.log('[Global State] Shutdown in progress, blocking reconnection attempt');
      return;
    }

    const state = useGlobalStore.getState();

    // Don't schedule if already connected or already scheduled
    if (state.connection.connected || reconnectionTimeout) {
      return;
    }

    // Don't reconnect if we've exceeded max attempts
    if (state.connection.reconnectAttempts >= state.connection.maxReconnectAttempts) {
      console.log('[Global Store] Max reconnection attempts reached, not scheduling reconnection');
      return;
    }

    console.log(`[Global Store] Scheduling reconnection due to: ${reason}`);

    reconnectionTimeout = setTimeout(() => {
      // Block reconnection if shutdown is in progress
      if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
        console.log('[Global State] Shutdown in progress, blocking scheduled reconnection');
        reconnectionTimeout = null;
        return;
      }
      reconnectionTimeout = null;
      const currentState = useGlobalStore.getState();

      // Double-check we're still disconnected before attempting reconnection
      if (!currentState.connection.connected || !currentState.connection.wasmReady) {
        console.log(`[Global Store] Executing scheduled reconnection for: ${reason}`);
        currentState.reconnect();
      }
    }, 2000); // 2 second delay to prevent rapid reconnection attempts
  };

  const handleVisibilityChange = () => {
    if (!document.hidden) {
      if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
        console.log('[Global State] Shutdown in progress, ignoring visibility change event');
        return;
      }
      console.log('[Global State] Window became visible, checking connection...');
      scheduleReconnection('visibility change');
    }
  };

  const handleWindowFocus = () => {
    if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
      console.log('[Global State] Shutdown in progress, ignoring window focus event');
      return;
    }
    console.log('[Global State] Window gained focus, checking connection...');
    scheduleReconnection('window focus');
  };

  const handleOnline = () => {
    if (typeof window !== 'undefined' && (window as any).isShuttingDown) {
      console.log('[Global State] Shutdown in progress, ignoring network online event');
      return;
    }
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

    // Notify frontend lifecycle manager about global state cleanup
    try {
      if (typeof window !== 'undefined' && (window as any).frontendLifecycleManager) {
        console.log('[Global State] Coordinating with frontend lifecycle manager...');
        (window as any).frontendLifecycleManager.shutdown();
      }
    } catch (error) {
      console.warn('[Global State] Failed to coordinate with frontend lifecycle manager:', error);
    }

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
// --- WASM WebSocket Connection Status ---
// Use this hook to reflect the current WebSocket connection state managed by WASM.
// Zustand store's connection state is updated via WASM events (onWasmReady, onWasmMessage).
export function useWasmWebSocketStatus() {
  return useGlobalStore(state => ({
    connected: state.connection.connected,
    connecting: state.connection.connecting,
    wasmReady: state.connection.wasmReady,
    lastPing: state.connection.lastPing,
    reconnectAttempts: state.connection.reconnectAttempts
  }));
}

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

// Campaign update hooks
export function useCampaignUpdates() {
  const updateCampaign = useGlobalStore(state => state.updateCampaign);
  const updateCampaignFeatures = useGlobalStore(state => state.updateCampaignFeatures);
  const updateCampaignConfig = useGlobalStore(state => state.updateCampaignConfig);
  const switchCampaign = useGlobalStore(state => state.switchCampaign);

  return useMemo(
    () => ({
      updateCampaign,
      updateCampaignFeatures,
      updateCampaignConfig,
      switchCampaign
    }),
    [updateCampaign, updateCampaignFeatures, updateCampaignConfig, switchCampaign]
  );
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

// Health state hook for monitoring service health
export function useHealthState() {
  const serviceStates = useGlobalStore(state => state.serviceStates);
  const emitEvent = useGlobalStore(state => state.emitEvent);

  // Extract health information from all services
  const healthStates = useMemo(() => {
    const health: Record<string, any> = {};

    Object.keys(serviceStates).forEach(serviceName => {
      const serviceState = serviceStates[serviceName];
      if (serviceState?.health) {
        health[serviceName] = serviceState.health;
      } else {
        // Default state if no health data available
        health[serviceName] = {
          status: 'down',
          responseTime: 0,
          lastCheck: 0,
          dependencies: {},
          metrics: {}
        };
      }
    });

    return health;
  }, [serviceStates]);

  // Function to request health check for a specific service
  const requestHealthCheck = useCallback(
    (serviceName: string) => {
      emitEvent({
        type: `${serviceName}:health:v1:requested`,
        payload: {},
        metadata: {} as any
      });
    },
    [emitEvent]
  );

  // Function to request health check for all services
  const requestAllHealthChecks = useCallback(() => {
    const serviceNames = [
      'admin',
      'analytics',
      'campaign',
      'commerce',
      'content',
      'search',
      'media',
      'messaging',
      'notification',
      'security',
      'nexus',
      'user'
    ];

    serviceNames.forEach(serviceName => {
      requestHealthCheck(serviceName);
    });
  }, [requestHealthCheck]);

  return {
    healthStates,
    requestHealthCheck,
    requestAllHealthChecks
  };
}

export function useNexusState() {
  return useServiceState('nexus');
}

export function useCampaignState() {
  const campaignServiceState = useServiceState('campaign');
  const campaignState = useGlobalStore(state => state.campaignState);
  const metadata = useGlobalStore(state => state.metadata.campaign);
  const { updateCampaign, updateCampaignFeatures, updateCampaignConfig } = useCampaignUpdates();

  // Get the most current campaign state from multiple sources
  const currentState = useMemo(() => {
    // Priority: service state > legacy campaignState > metadata
    const state = campaignServiceState.state || campaignState || {};

    // Merge in metadata for completeness
    return {
      ...state,
      campaignId: metadata.campaignId,
      slug: metadata.slug,
      features: state.features || metadata.features || [],
      serviceSpecific: metadata.serviceSpecific || state.serviceSpecific || {}
    };
  }, [campaignServiceState.state, campaignState, metadata]);

  return useMemo(
    () => ({
      state: currentState,
      metadata,
      setState: campaignServiceState.setState,
      updateState: campaignServiceState.updateState,
      clearState: campaignServiceState.clearState,
      // Campaign-specific update functions
      updateCampaign,
      updateFeatures: updateCampaignFeatures,
      updateConfig: updateCampaignConfig
    }),
    [
      currentState,
      metadata,
      campaignServiceState.setState,
      campaignServiceState.updateState,
      campaignServiceState.clearState,
      updateCampaign,
      updateCampaignFeatures,
      updateCampaignConfig
    ]
  );
}

// --- Type exports ---
export type { GlobalState, FeatureState, ServiceStates };

// GPU capabilities hook for easy access to GPU information in components
export function useGPUCapabilities() {
  const deviceMetadata = useGlobalStore(state => state.metadata.device);

  const gpuCapabilities = useMemo(() => {
    return deviceMetadata.gpuCapabilities || null;
  }, [deviceMetadata.gpuCapabilities]);

  const wasmGPUBridge = useMemo(() => {
    return deviceMetadata.wasmGPUBridge || null;
  }, [deviceMetadata.wasmGPUBridge]);

  const refreshGPUCapabilities = useCallback(async () => {
    try {
      // Dynamically import WASM GPU bridge to avoid circular dependencies
      const { wasmGPU } = await import('../lib/wasmBridge.js');
      await wasmGPU.updateMetadataWithGPUInfo();
    } catch (error) {
      console.error('[GPU Capabilities Hook] Failed to refresh GPU capabilities:', error);
    }
  }, []);

  const isWebGPUAvailable = useMemo(() => {
    return gpuCapabilities?.webgpu?.available || false;
  }, [gpuCapabilities]);

  const isWebGLAvailable = useMemo(() => {
    return gpuCapabilities?.webgl?.available || false;
  }, [gpuCapabilities]);

  const recommendedRenderer = useMemo(() => {
    return gpuCapabilities?.three?.recommendedRenderer || 'webgl';
  }, [gpuCapabilities]);

  const performanceScore = useMemo(() => {
    return gpuCapabilities?.performance?.score || 0;
  }, [gpuCapabilities]);

  return {
    gpuCapabilities,
    wasmGPUBridge,
    refreshGPUCapabilities,
    isWebGPUAvailable,
    isWebGLAvailable,
    recommendedRenderer,
    performanceScore,
    detectedAt: deviceMetadata.gpuDetectedAt
  };
}

// --- Media Streaming Hook ---
export function useMediaStreamingState() {
  const mediaStreaming = useGlobalStore(state => state.mediaStreaming);
  const setMediaStreamingState = useGlobalStore(state => state.setMediaStreamingState);
  const clearMediaStreamingState = useGlobalStore(state => state.clearMediaStreamingState);
  return {
    mediaStreaming,
    setMediaStreamingState,
    clearMediaStreamingState
  };
}
