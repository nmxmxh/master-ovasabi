import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import {
  validateCanonicalEventEnvelope,
  transformToCanonicalMetadata
} from '../../types/canonicalEvents';
import type { EventEnvelope, EventState } from '../types/events';
import { useMetadataStore } from './metadataStore';

// --- Pub/Sub types ---
type EventType = string;
type SubscriberCallback = (event: EventEnvelope) => void;
type Subscribers = Map<EventType, Set<SubscriberCallback>>;

interface EventStore extends EventState {
  // Actions
  emitEvent: (
    event: Omit<
      EventEnvelope,
      'timestamp' | 'correlation_id' | 'version' | 'environment' | 'source'
    > & { correlation_id?: string },
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateEventState: (eventType: string, state: string) => void;
  handleWasmMessage: (msg: any) => void;
  processQueuedMessages: () => void;
  clearHistory: () => void;
  getEventsByType: (eventType: string) => EventEnvelope[];
  getLatestEvent: (eventType?: string) => EventEnvelope | undefined;
  getCurrentState: (eventType: string) => string | undefined;

  // Pub/Sub
  subscribe: (eventType: EventType, callback: SubscriberCallback) => () => void;
  unsubscribe: (eventType: EventType, callback: SubscriberCallback) => void;
  subscribers: Subscribers;

  // WASM readiness
  isWasmReady: boolean;
  setWasmReady: (ready: boolean) => void;
  queuedEvents: Array<{ event: any; onResponse?: (event: EventEnvelope) => void }>;
}

// Utility function to generate correlation ID
const generateCorrelationId = (): string => {
  return `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

export const useEventStore = create<EventStore>()(
  devtools(
    (set, get) => ({
      // Initial event state
      events: [],
      eventStates: {},
      eventPayloads: {},
      queuedMessages: [],
      pendingRequests: {},
      lastMessageTime: null,
      eventsByType: new Map(),
      subscribers: new Map(),

      // WASM readiness state
      isWasmReady: false,
      queuedEvents: [],

      // --- Pub/Sub Actions ---
      subscribe: (eventType, callback) => {
        const subscribers = get().subscribers;
        if (!subscribers.has(eventType)) {
          subscribers.set(eventType, new Set());
        }
        subscribers.get(eventType)!.add(callback);
        set({ subscribers: new Map(subscribers) });

        // Return an unsubscribe function
        return () => get().unsubscribe(eventType, callback);
      },

      unsubscribe: (eventType, callback) => {
        const subscribers = get().subscribers;
        if (subscribers.has(eventType)) {
          subscribers.get(eventType)!.delete(callback);
          if (subscribers.get(eventType)!.size === 0) {
            subscribers.delete(eventType);
          }
          set({ subscribers: new Map(subscribers) });
        }
      },

      // --- Core Actions ---
      emitEvent: (event, onResponse) => {
        // Check if WASM is ready, if not queue the event
        if (!get().isWasmReady) {
          console.log('[EventStore] WASM not ready, queuing event:', event.type);
          set(
            state => ({
              queuedEvents: [...state.queuedEvents, { event, onResponse }]
            }),
            false,
            'queueEvent'
          );
          return;
        }

        // Additional check: ensure userID is available from WASM and metadata store
        if (typeof window !== 'undefined' && !(window as any).userID) {
          console.log('[EventStore] WASM userID not available, queuing event:', event.type);
          set(
            state => ({
              queuedEvents: [...state.queuedEvents, { event, onResponse }]
            }),
            false,
            'queueEvent'
          );
          return;
        }

        // Check if metadata store has been initialized with WASM user ID
        const metadataStore = useMetadataStore.getState();
        if (
          metadataStore.userId === 'loading' ||
          metadataStore.metadata?.user?.userId === 'loading'
        ) {
          console.log(
            '[EventStore] Metadata store not initialized with WASM user ID, queuing event:',
            event.type
          );
          set(
            state => ({
              queuedEvents: [...state.queuedEvents, { event, onResponse }]
            }),
            false,
            'queueEvent'
          );
          return;
        }

        const correlationId = event.correlation_id || generateCorrelationId();

        // Transform metadata to canonical format if needed
        let canonicalMetadata = event.metadata;
        if (event.metadata && !event.metadata.global_context) {
          // Transform flat metadata structure to canonical format using helper function
          canonicalMetadata = transformToCanonicalMetadata(event.metadata, correlationId);
          console.log('[EventStore] Transformed metadata to canonical format:', {
            original: event.metadata,
            canonical: canonicalMetadata
          });
        } else {
          console.log('[EventStore] Using existing canonical metadata:', canonicalMetadata);
        }

        // Create canonical event envelope with proper structure
        const fullEvent: EventEnvelope = {
          ...event,
          correlation_id: correlationId,
          timestamp: new Date().toISOString(),
          version: '1.0.0',
          environment: process.env.NODE_ENV || 'development',
          source: 'frontend',
          metadata: canonicalMetadata,
          // Use payload directly as data - this matches the Go structpb.Struct format
          payload: event.payload
            ? {
                data: event.payload
              }
            : undefined
        };

        // Validate event if it's a canonical event
        if (validateCanonicalEventEnvelope(fullEvent)) {
          console.log('[EventStore] Emitting canonical event:', fullEvent.type);
        } else {
          console.warn('[EventStore] Event validation failed:', fullEvent);
        }

        // Add to events list
        set(
          state => ({
            events: [...state.events, fullEvent].slice(-100), // Keep last 100 events
            eventStates: {
              ...state.eventStates,
              [fullEvent.type]: 'emitted'
            }
          }),
          false,
          'emitEvent'
        );

        // If there's a response handler, store it
        if (onResponse) {
          set(
            state => ({
              pendingRequests: {
                ...state.pendingRequests,
                [correlationId]: {
                  expectedEventType: fullEvent.type.replace(/:requested$/, ':success'),
                  eventType: fullEvent.type,
                  timestamp: Date.now(),
                  resolve: onResponse
                }
              }
            }),
            false,
            'storePendingRequest'
          );
        }

        // Send to WASM WebSocket
        try {
          // Check if WASM is ready and send message directly
          if (typeof window.sendWasmMessage === 'function') {
            window.sendWasmMessage(fullEvent);
            console.log('[EventStore] Event sent to WASM:', fullEvent.type, fullEvent);
          } else {
            console.warn('[EventStore] sendWasmMessage not available, WASM not ready');
            // Queue the event for later processing
            set(
              state => ({
                queuedEvents: [...state.queuedEvents, { event: fullEvent, onResponse }]
              }),
              false,
              'queueEventForWasm'
            );
          }
        } catch (error) {
          console.error('[EventStore] Failed to send event to WASM:', error);
        }

        console.log('[EventStore] Event emitted:', fullEvent);
      },

      updateEventState: (eventType, state) => {
        set(
          currentState => ({
            eventStates: {
              ...currentState.eventStates,
              [eventType]: state
            }
          }),
          false,
          'updateEventState'
        );
      },

      handleWasmMessage: msg => {
        console.log('[EventStore] Handling WASM message:', {
          type: msg.type,
          correlationId: msg.correlationId || msg.correlation_id,
          payload: msg.payload,
          metadata: msg.metadata,
          timestamp: new Date().toISOString()
        });

        if (msg.type === 'search:search:v1:success') {
          console.log('[EventStore] SEARCH SUCCESS EVENT RECEIVED:', msg);
        }

        // Update connection status when we receive any message (indicates WebSocket is connected)
        if (msg.type && msg.type !== 'connection:status') {
          // Update last message time
          set(
            state => ({
              ...state,
              lastMessageTime: new Date().toISOString()
            }),
            false,
            'updateLastMessageTime'
          );

          import('./connectionStore').then(mod => {
            if (mod && mod.useConnectionStore) {
              const store = mod.useConnectionStore.getState();
              if (store.handleConnectionStatus) {
                store.handleConnectionStatus(true, 'message_received');
              }
            }
          });
        }

        // Handle connection status messages specially
        if (msg.type === 'connection:status') {
          // Update connection store with WebSocket status
          import('./connectionStore').then(mod => {
            if (mod && mod.useConnectionStore) {
              const store = mod.useConnectionStore.getState();
              if (store.handleConnectionStatus) {
                store.handleConnectionStatus(
                  msg.payload?.connected || false,
                  msg.payload?.reason || 'unknown'
                );
              }
            }
          });
          return; // Don't add connection status messages to event history
        }

        // Handle forced campaign switch
        if (msg.type === 'reconnect:campaign_switch') {
          console.log('[EventStore] Received reconnect:campaign_switch event', msg);
          const newCampaignId = msg.payload?.newCampaignId;
          if (newCampaignId) {
            import('./connectionStore').then(mod => {
              if (mod && mod.useConnectionStore) {
                mod.useConnectionStore.getState().handleCampaignSwitch(newCampaignId);
              }
            });
          }
          return; // Stop further processing for this event
        }

        // Process the message and potentially emit events
        if (msg.type && msg.payload) {
          console.log(
            '[EventStore] Processing message with type:',
            msg.type,
            'and payload:',
            typeof msg.payload
          );
          if (msg.type === 'search:search:v1:success') {
            console.log('[EventStore] Processing search success event:', msg);
          }

          // Parse payload if it's a JSON string
          let parsedPayload = msg.payload;
          if (typeof msg.payload === 'string') {
            try {
              parsedPayload = JSON.parse(msg.payload);
              console.log('[EventStore] Parsed JSON payload:', parsedPayload);
            } catch (error) {
              console.warn('[EventStore] Failed to parse payload as JSON:', error);
              // Keep as string if parsing fails
            }
          }

          // Handle nested payload structure (payload.data contains the actual data)
          // Backend sends payloads wrapped in {data: {...}}, so we need to unwrap it
          if (parsedPayload && typeof parsedPayload === 'object') {
            // Check for nested data structure (common with protobuf serialization)
            if (parsedPayload.data && typeof parsedPayload.data === 'object') {
              console.log('[EventStore] Found nested payload.data structure, extracting:', {
                hasData: !!parsedPayload.data,
                dataKeys: Object.keys(parsedPayload.data || {}),
                originalKeys: Object.keys(parsedPayload || {})
              });
              parsedPayload = parsedPayload.data;
            }
            // Also check if campaigns are directly in payload (for backward compatibility)
            if (
              msg.type === 'campaign:list:v1:success' &&
              !parsedPayload.campaigns &&
              parsedPayload.data?.campaigns
            ) {
              console.log('[EventStore] Campaigns found in payload.data.campaigns, extracting');
              parsedPayload = parsedPayload.data;
            }
          }

          // Parse metadata if it's a JSON string
          let parsedMetadata = msg.metadata;
          if (typeof msg.metadata === 'string') {
            try {
              parsedMetadata = JSON.parse(msg.metadata);
              console.log('[EventStore] Parsed JSON metadata:', parsedMetadata);
            } catch (error) {
              console.warn('[EventStore] Failed to parse metadata as JSON:', error);
            }
          }

          // Extract correlation ID from multiple sources
          let extractedCorrelationId;

          // Prioritize correlationId from the payload
          if (
            parsedPayload &&
            typeof parsedPayload === 'object' &&
            'correlationId' in parsedPayload
          ) {
            extractedCorrelationId = parsedPayload.correlationId;
          }

          // Fallback to the message-level correlationId
          if (!extractedCorrelationId) {
            extractedCorrelationId = msg.correlation_id || msg.correlationId;
          }

          // Generate a new one if not found anywhere
          if (!extractedCorrelationId) {
            extractedCorrelationId = `corr_${Date.now()}`;
          }

          console.log('[EventStore] Correlation ID extraction:', {
            fromMsg: msg.correlation_id || msg.correlationId,
            fromPayload:
              parsedPayload && typeof parsedPayload === 'object'
                ? parsedPayload.correlationId
                : 'not_object',
            final: extractedCorrelationId,
            msgType: msg.type
          });

          const event: EventEnvelope = {
            type: msg.type,
            payload: parsedPayload,
            metadata: parsedMetadata || {},
            correlation_id: extractedCorrelationId,
            timestamp: msg.timestamp || new Date().toISOString(),
            version: msg.version || '1.0.0',
            environment: msg.environment || 'development',
            source: msg.source || 'wasm'
          };

          console.log('[EventStore] Creating event envelope:', {
            type: event.type,
            correlation_id: event.correlation_id,
            timestamp: event.timestamp
          });

          // --- Notify Subscribers ---
          const { subscribers } = get();
          // Notify wildcard subscribers
          if (subscribers.has('*')) {
            subscribers.get('*')!.forEach(callback => callback(event));
          }
          // Notify event-specific subscribers
          if (subscribers.has(event.type)) {
            subscribers.get(event.type)!.forEach(callback => callback(event));
          }
          // --- End Notify ---

          if (event.type === 'search:search:v1:success') {
            console.log('[EventStore] Creating search success event envelope:', event);
          }

          set(
            state => {
              const newEvents = [...state.events, event].slice(-100);
              console.log('[EventStore] Adding event to store:', {
                type: event.type,
                totalEvents: newEvents.length,
                eventAdded: true,
                searchEvents: newEvents.filter(e => e.type === 'search:search:v1:success').length
              });

              if (event.type === 'search:search:v1:success') {
                console.log('[EventStore] Adding search success event to store:', event);
              }

              if (event.type === 'campaign:switch:v1:success') {
                console.log('[EventStore] Processing campaign switch success event:', event);

                // Update campaign store with the new campaign data
                import('./campaignStore').then(({ useCampaignStore }) => {
                  const campaignStore = useCampaignStore.getState();
                  if (campaignStore.updateCampaignFromResponse) {
                    campaignStore.updateCampaignFromResponse(event.payload);
                    console.log('[EventStore] Updated campaign store with switch success data');
                  }
                });

                // Trigger WebSocket reconnection to subscribe to new campaign
                if (window.handleCampaignSwitchSuccess) {
                  const campaignId =
                    event.payload.campaign_id || event.payload.campaignId || 'unknown';
                  const reason = event.payload.reason || 'user_initiated';
                  console.log(
                    '[EventStore] Triggering WebSocket reconnection for campaign:',
                    campaignId
                  );
                  window.handleCampaignSwitchSuccess(campaignId, reason);
                }
              }

              if (event.type === 'campaign:state:v1:success') {
                console.log('[EventStore] Adding campaign state success event to store:', event);
                // Update campaign store with the received campaign data
                import('./campaignStore').then(({ useCampaignStore }) => {
                  useCampaignStore.getState().updateCampaignFromResponse(event.payload);
                });
              }

              if (event.type === 'campaign:list:v1:success') {
                // Ensure campaigns are at the top level of payload
                let campaignPayload = event.payload;
                if (campaignPayload && typeof campaignPayload === 'object') {
                  // If campaigns are nested in data, extract them
                  if (!campaignPayload.campaigns && campaignPayload.data?.campaigns) {
                    console.log(
                      '[EventStore] ⚠️ Campaigns found in payload.data, extracting to top level'
                    );
                    campaignPayload = campaignPayload.data;
                  }
                }

                console.log('[EventStore] ✅ Adding campaign list success event to store:', {
                  type: event.type,
                  correlation_id: event.correlation_id,
                  payload: campaignPayload,
                  payloadKeys: campaignPayload ? Object.keys(campaignPayload) : [],
                  hasCampaigns: !!campaignPayload?.campaigns,
                  campaignsCount: campaignPayload?.campaigns?.length || 0,
                  campaignsPreview: campaignPayload?.campaigns?.slice(0, 2) // First 2 for debugging
                });

                // Update campaign store with the received campaign list
                import('./campaignStore')
                  .then(({ useCampaignStore }) => {
                    console.log(
                      '[EventStore] ✅ Calling updateCampaignsFromResponse with payload:',
                      campaignPayload
                    );
                    useCampaignStore.getState().updateCampaignsFromResponse(campaignPayload);
                  })
                  .catch(err => {
                    console.error('[EventStore] ❌ Error updating campaign store:', err);
                  });
              }
              return {
                events: newEvents,
                eventStates: {
                  ...state.eventStates,
                  [event.type]: 'received'
                }
              };
            },
            false,
            'handleWasmMessage'
          );

          const correlationId = msg.correlation_id || msg.correlationId || event.correlation_id;
          let matchedCorrelationId = correlationId;

          // Attempt to find an exact match by correlation ID first
          if (correlationId && get().pendingRequests[correlationId]) {
            matchedCorrelationId = correlationId;
          }

          if (matchedCorrelationId && get().pendingRequests[matchedCorrelationId]) {
            const pendingRequest = get().pendingRequests[matchedCorrelationId];
            // Ensure the event type matches what this request expects to prevent cross-resolution collisions
            const expectedType = pendingRequest.expectedEventType;
            if (expectedType && event.type !== expectedType) {
              console.warn(
                '[EventStore] Pending request type mismatch; not resolving.',
                {
                  correlationId: matchedCorrelationId,
                  expectedType,
                  receivedType: event.type
                }
              );
            } else {
              console.log(
                '[EventStore] Resolving pending request for correlation ID:',
                matchedCorrelationId,
                'with event type:',
                event.type
              );
              pendingRequest.resolve(event);
              set(
                state => {
                  const newPendingRequests = { ...state.pendingRequests };
                  delete newPendingRequests[matchedCorrelationId];
                  return { pendingRequests: newPendingRequests };
                },
                false,
                'resolvePendingRequest'
              );
              return;
            }

          } else {
            console.log(
              '[EventStore] No pending request found for correlation ID:',
              correlationId,
              'or matching event type. Available pending requests:',
              Object.keys(get().pendingRequests)
            );
          }
        }
      },

      processQueuedMessages: () => {
        const state = get();
        console.log('[EventStore] Processing queued messages:', state.queuedMessages.length);

        // Process all queued messages
        state.queuedMessages.forEach(msg => {
          get().handleWasmMessage(msg);
        });

        // Clear the queue
        set({ queuedMessages: [] }, false, 'processQueuedMessages');
      },

      clearHistory: () => {
        set(
          {
            events: [],
            eventStates: {},
            eventPayloads: {},
            queuedMessages: [],
            pendingRequests: {}
          },
          false,
          'clearHistory'
        );
      },

      getEventsByType: eventType => {
        return get().events.filter(event => event.type === eventType);
      },

      getLatestEvent: eventType => {
        const events = get().events;
        if (eventType) {
          const filteredEvents = events.filter(event => event.type === eventType);
          return filteredEvents[filteredEvents.length - 1];
        }
        return events[events.length - 1];
      },

      getCurrentState: eventType => {
        return get().eventStates[eventType];
      },

      // WASM readiness management
      setWasmReady: (ready: boolean) => {
        set({ isWasmReady: ready }, false, 'setWasmReady');

        // If WASM just became ready, process queued events
        if (ready) {
          const { queuedEvents, emitEvent } = get();
          if (queuedEvents.length > 0) {
            console.log(`[EventStore] WASM ready, processing ${queuedEvents.length} queued events`);
            // Create a copy and clear the queue before processing to avoid infinite loops
            const eventsToProcess = [...queuedEvents];
            set({ queuedEvents: [] }, false, 'clearQueuedEvents');

            // Re-emit events to ensure they are fully processed and formatted
            eventsToProcess.forEach(({ event, onResponse }) => {
              emitEvent(event, onResponse);
            });
          }
        }
      }
    }),
    {
      name: 'event-store'
    }
  )
);

// --- Centralized WASM Message Handling ---
(function initializeWasmListener() {
  if (typeof window === 'undefined') {
    return;
  }

  // Assign the handler from the store to the global window object.
  // This ensures there is only ONE handler for all incoming WASM messages.
  (window as any).onWasmMessage = (msg: any) => {
    useEventStore.getState().handleWasmMessage(msg);
  };

  // Signal to WASM that the frontend event store is ready to process messages.
  // This should be done after the store is created, but WASM might not be ready yet.
  // Retry with exponential backoff until WASM is ready.
  const trySignal = () => {
    if (typeof (window as any).setFrontendReady === 'function') {
      console.log('[EventStore] Signaling to WASM that frontend is ready.');
      (window as any).setFrontendReady();
    } else {
      // Retry after a short delay - WASM might still be initializing
      setTimeout(trySignal, 100);
    }
  };

  // Initial attempt
  trySignal();
})();
