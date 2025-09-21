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
    >,
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

        const correlationId = generateCorrelationId();

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
          // Import the WASM bridge function
          import('../../lib/wasmBridge')
            .then(mod => {
              if (mod && mod.wasmSendMessage) {
                mod.wasmSendMessage(fullEvent);
                console.log('[EventStore] Event sent to WASM:', fullEvent.type, fullEvent);
              } else {
                console.warn('[EventStore] wasmSendMessage not available');
              }
            })
            .catch(error => {
              console.error('[EventStore] Failed to import WASM bridge:', error);
            });
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

          const event: EventEnvelope = {
            type: msg.type,
            payload: parsedPayload,
            metadata: msg.metadata || {},
            correlation_id: msg.correlation_id || msg.correlationId || `corr_${Date.now()}`,
            timestamp: new Date().toISOString(),
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
          const correlationId = msg.correlationId || msg.correlation_id || event.correlation_id;
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

          if (event.type === 'search:search:v1:success') {
            console.log(
              '[EventStore] Processing search success event correlation ID:',
              correlationId
            );
          }

          if (correlationId && get().pendingRequests[correlationId]) {
            const pendingRequest = get().pendingRequests[correlationId];
            console.log(
              '[EventStore] Resolving pending request for correlation ID:',
              correlationId
            );
            pendingRequest.resolve(event);

            set(
              state => {
                const newPendingRequests = { ...state.pendingRequests };
                delete newPendingRequests[correlationId];
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

          queuedEvents.forEach(({ event, onResponse }) => {
            get().emitEvent(event, onResponse);
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
