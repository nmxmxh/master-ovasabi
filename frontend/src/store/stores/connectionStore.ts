import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { ConnectionState, MediaStreamingState } from '../types/connection';

interface ConnectionStore extends ConnectionState {
  // Actions
  setConnectionState: (state: Partial<ConnectionState>) => void;
  setWasmFunctions: (funcs: { [key: string]: boolean }) => void;
  reconnect: () => void;
  handleConnectionStatus: (connected: boolean, reason: string) => void;
  checkGlobalConnectionStatus: () => void;
  checkConnectionTimeout: () => void;

  // Media Streaming
  mediaStreaming: MediaStreamingState;
  setMediaStreamingState: (state: Partial<MediaStreamingState>) => void;
  clearMediaStreamingState: () => void;
}

export const useConnectionStore = create<ConnectionStore>()(
  devtools(
    (set, get) => ({
      // Initial connection state
      connected: false,
      connecting: false,
      lastPing: '',
      reconnectAttempts: 0,
      maxReconnectAttempts: 5,
      reconnectDelay: 1000,
      wasmReady: false,
      wasmFunctions: {
        initWebGPU: false,
        runGPUCompute: false,
        getGPUMetricsBuffer: false
      },

      // Initial media streaming state
      mediaStreaming: {
        connected: false,
        peerId: '',
        streamInfo: null,
        error: undefined,
        lastConnectAttempt: ''
      },

      // Actions
      setConnectionState: newState => {
        set(
          (state: ConnectionStore) => ({
            ...state,
            ...newState,
            lastPing: newState.lastPing || state.lastPing
          }),
          false,
          'setConnectionState'
        );
      },

      // Handle WebSocket connection status updates from WASM
      handleConnectionStatus: (connected: boolean, reason: string) => {
        set(
          (state: ConnectionStore) => ({
            ...state,
            connected,
            connecting: false,
            lastPing: connected ? new Date().toISOString() : state.lastPing,
            reconnectAttempts: connected ? 0 : state.reconnectAttempts
          }),
          false,
          'handleConnectionStatus'
        );

        // Log connection status changes for debugging
        console.log(
          `[ConnectionStore] WebSocket status: ${connected ? 'CONNECTED' : 'DISCONNECTED'} (${reason})`
        );
      },

      // Check global WebSocket status as fallback
      checkGlobalConnectionStatus: () => {
        const globalConnected = (window as any).wsConnected;
        const globalWasmReady = (window as any).wasmReady;

        if (typeof globalConnected === 'boolean' && typeof globalWasmReady === 'boolean') {
          set(
            (state: ConnectionStore) => ({
              ...state,
              connected: globalConnected,
              wasmReady: globalWasmReady,
              lastPing: globalConnected ? new Date().toISOString() : state.lastPing
            }),
            false,
            'checkGlobalConnectionStatus'
          );

          console.log(
            `[ConnectionStore] Global status check: connected=${globalConnected}, wasmReady=${globalWasmReady}`
          );
        }
      },

      // Check connection timeout based on last message received
      checkConnectionTimeout: () => {
        // Check if we haven't received a message in the last 30 seconds
        import('./eventStore').then(mod => {
          if (mod && mod.useEventStore) {
            const eventStore = mod.useEventStore.getState();
            const lastMessageTime = (eventStore as any).lastMessageTime;

            if (lastMessageTime) {
              const timeSinceLastMessage = Date.now() - new Date(lastMessageTime).getTime();
              const timeoutThreshold = 30000; // 30 seconds

              if (timeSinceLastMessage > timeoutThreshold) {
                console.log(
                  '[ConnectionStore] Connection timeout detected, no messages received in',
                  timeSinceLastMessage,
                  'ms'
                );
                set(
                  (state: ConnectionStore) => ({
                    ...state,
                    connected: false,
                    lastPing: state.lastPing
                  }),
                  false,
                  'connectionTimeout'
                );
              }
            }
          }
        });
      },

      setWasmFunctions: funcs => {
        set(
          (state: ConnectionStore) => ({
            wasmFunctions: {
              ...(state.wasmFunctions || {}),
              ...funcs
            } as ConnectionStore['wasmFunctions']
          }),
          false,
          'setWasmFunctions'
        );
      },

      reconnect: () => {
        const state = get();
        if (state.connecting) return;

        set(
          {
            connecting: true,
            reconnectAttempts: state.reconnectAttempts + 1
          },
          false,
          'reconnect'
        );

        // Call WASM reconnect function if available
        if (typeof (window as any).reconnectWebSocket === 'function') {
          console.log('[ConnectionStore] Calling WASM reconnect function');
          (window as any).reconnectWebSocket();
        } else {
          console.warn('[ConnectionStore] WASM reconnect function not available');
          // Fallback: simulate reconnection
          setTimeout(() => {
            set(
              {
                connecting: false,
                connected: true,
                lastPing: new Date().toISOString()
              },
              false,
              'reconnectComplete'
            );
          }, 1000);
        }
      },

      // Media Streaming Actions
      setMediaStreamingState: newState => {
        set(
          (state: ConnectionStore) => ({
            mediaStreaming: { ...state.mediaStreaming, ...newState }
          }),
          false,
          'setMediaStreamingState'
        );
      },

      clearMediaStreamingState: () => {
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
        );
      }
    }),
    {
      name: 'connection-store'
    }
  )
);
