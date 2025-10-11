import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import {
  validateCanonicalEventEnvelope,
  transformToCanonicalMetadata
} from '../../types/canonicalEvents';
import type { EventEnvelope, EventState } from '../types/events';
import { useMetadataStore } from './metadataStore';

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

      // WASM readiness state
      isWasmReady: false,
      queuedEvents: [],

      // Actions
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
            metadata: msg.metadata || {},
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
                console.log('[EventStore] Adding campaign list success event to store:', event);
                // Update campaign store with the received campaign list
                import('./campaignStore').then(({ useCampaignStore }) => {
                  useCampaignStore.getState().updateCampaignsFromResponse(event.payload);
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

          // Check if this resolves a pending request
          const correlationId = msg.correlation_id || msg.correlationId || event.correlation_id;
          console.log('[EventStore] Checking correlation ID match:', {
            receivedCorrelationId: correlationId,
            availablePendingRequests: Object.keys(get().pendingRequests),
            exactMatch: correlationId && get().pendingRequests[correlationId],
            partialMatch:
              correlationId &&
              Object.keys(get().pendingRequests).find(
                key => key.includes(correlationId) || correlationId.includes(key)
              )
          });

          // If no correlation ID from backend, try to match by event type and recent timing
          let matchedCorrelationId = correlationId;
          if (!correlationId || !get().pendingRequests[correlationId]) {
            // Try to find a matching pending request by event type and timing
            const pendingKeys = Object.keys(get().pendingRequests);
            const matchingKey = pendingKeys.find(key => {
              const pendingRequest = get().pendingRequests[key];
              const timeDiff = Date.now() - (pendingRequest as any).timestamp;
              // Match if it's the same event type and within 5 seconds
              return (pendingRequest as any).eventType === event.type && timeDiff < 5000;
            });

            if (matchingKey) {
              matchedCorrelationId = matchingKey;
              console.log('[EventStore] Found matching request by type and timing:', {
                eventType: event.type,
                matchedKey: matchingKey,
                timeDiff: Date.now() - (get().pendingRequests[matchingKey] as any).timestamp
              });
            }
          }

          if (event.type === 'search:search:v1:success') {
            console.log(
              '[EventStore] Processing search success event correlation ID:',
              correlationId
            );
          }

          if (matchedCorrelationId && get().pendingRequests[matchedCorrelationId]) {
            const pendingRequest = get().pendingRequests[matchedCorrelationId];
            console.log(
              '[EventStore] Resolving pending request for correlation ID:',
              matchedCorrelationId
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
          } else if (correlationId) {
            // Try partial matching for correlation IDs that might have been modified
            const partialMatch = Object.keys(get().pendingRequests).find(
              key => key.includes(correlationId) || correlationId.includes(key)
            );

            if (partialMatch) {
              console.log('[EventStore] Found partial correlation ID match:', {
                received: correlationId,
                stored: partialMatch
              });
              const pendingRequest = get().pendingRequests[partialMatch];
              pendingRequest.resolve(event);

              set(
                state => {
                  const newPendingRequests = { ...state.pendingRequests };
                  delete newPendingRequests[partialMatch];
                  return { pendingRequests: newPendingRequests };
                },
                false,
                'resolvePendingRequestPartial'
              );
            } else {
              console.log(
                '[EventStore] No pending request found for correlation ID:',
                correlationId,
                'Available pending requests:',
                Object.keys(get().pendingRequests)
              );
            }
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
          const { queuedEvents } = get();
          console.log(`[EventStore] WASM ready, processing ${queuedEvents.length} queued events`);

          queuedEvents.forEach(({ event }) => {
            // Send directly to WASM
            try {
              if (typeof window.sendWasmMessage === 'function') {
                window.sendWasmMessage(event);
                console.log('[EventStore] Queued event sent to WASM:', event.type, event);
              } else {
                console.warn('[EventStore] sendWasmMessage not available for queued event');
              }
            } catch (error) {
              console.error('[EventStore] Failed to send queued event to WASM:', error);
            }
          });

          // Clear queued events
          set({ queuedEvents: [] }, false, 'clearQueuedEvents');
        }
      }
    }),
    {
      name: 'event-store'
    }
  )
);
