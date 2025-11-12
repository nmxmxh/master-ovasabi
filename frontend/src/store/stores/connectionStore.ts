import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { ConnectionState, MediaStreamingState } from '../types/connection';
import { storeRegistry } from '../utils/storeActions';

interface ConnectionStore extends ConnectionState {
  // Actions
  setConnectionState: (state: Partial<ConnectionState>) => void;
  setWasmFunctions: (funcs: { [key: string]: boolean }) => void;
  reconnect: () => void;
  handleConnectionStatus: (connected: boolean, reason: string) => void;
  handleCampaignSwitch: (newCampaignId: string) => void;
  checkGlobalConnectionStatus: () => void;
  checkConnectionTimeout: () => void;

  // Media Streaming
  mediaStreaming: MediaStreamingState;
  setMediaStreamingState: (state: Partial<MediaStreamingState>) => void;
  clearMediaStreamingState: () => void;
  connectMediaStreaming: (campaignId?: string, contextId?: string) => void;
  disconnectMediaStreaming: () => void;
}

// Removed verbose logging - only critical events are logged

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
        connecting: false,
        campaignId: '0',
        contextId: 'webgpu-particles',
        peerId: '',
        url: '',
        localStream: null,
        remoteStream: null,
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

        // Only log critical connection changes
        if (reason === 'error' || reason === 'failed') {
          console.log(
            `[ConnectionStore] WebSocket status: ${connected ? 'CONNECTED' : 'DISCONNECTED'} (${reason})`
          );
        }
      },

      handleCampaignSwitch: (newCampaignId: string) => {
        console.log(`[ConnectionStore] Switching campaign to ${newCampaignId}`);
        if (typeof (window as any).switchCampaign === 'function') {
          (window as any).switchCampaign(newCampaignId);
        } else {
          console.warn('[ConnectionStore] WASM switchCampaign function not available');
        }
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

          // Removed verbose status logging
        }
      },

      // Check connection timeout based on last message received
      checkConnectionTimeout: () => {
        // Note: Connection timeout checking is now handled by the event store
        // to avoid circular dependencies. This method is kept for compatibility.
        console.log('[ConnectionStore] Connection timeout check delegated to event store');
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
              connecting: false,
              campaignId: '0',
              contextId: 'webgpu-particles',
              peerId: '',
              url: '',
              localStream: null,
              remoteStream: null,
              streamInfo: null,
              error: undefined,
              lastConnectAttempt: ''
            }
          },
          false,
          'clearMediaStreamingState'
        );
      },

      connectMediaStreaming: (campaignId?: string, contextId?: string) => {
        const now = new Date().toISOString();
        const mediaAPI =
          typeof window !== 'undefined' ? (window as any).mediaStreaming : undefined;
        if (!mediaAPI || typeof mediaAPI.connect !== 'function') {
          console.warn('[ConnectionStore] mediaStreaming.connect() not available');
          set(
            (state: ConnectionStore) => ({
              mediaStreaming: {
                ...state.mediaStreaming,
                connecting: false
              }
            }),
            false,
            'connectMediaStreamingUnavailable'
          );
          return;
        }
        set(
          (state: ConnectionStore) => ({
            mediaStreaming: {
              ...state.mediaStreaming,
              connecting: true,
              connected: false,
              campaignId: campaignId ?? state.mediaStreaming.campaignId,
              contextId: contextId ?? state.mediaStreaming.contextId,
              lastConnectAttempt: now,
              error: undefined
            }
          }),
          false,
          'connectMediaStreaming'
        );
        mediaAPI.connect();
      },

      disconnectMediaStreaming: () => {
        const mediaAPI =
          typeof window !== 'undefined' ? (window as any).mediaStreaming : undefined;
        set(
          (state: ConnectionStore) => ({
            mediaStreaming: {
              ...state.mediaStreaming,
              connecting: false,
              connected: false,
              peerId: '',
              url: ''
            }
          }),
          false,
          'disconnectMediaStreaming'
        );
        if (!mediaAPI || typeof mediaAPI.shutdown !== 'function') {
          console.warn('[ConnectionStore] mediaStreaming.shutdown() not available');
          return;
        }
        mediaAPI.shutdown();
      }
    }),
    {
      name: 'connection-store'
    }
  )
);

// Register store actions to break circular dependencies
storeRegistry.register('connection', {
  handleConnectionStatus: (connected: boolean, reason: string) =>
    useConnectionStore.getState().handleConnectionStatus(connected, reason),
  handleCampaignSwitch: (newCampaignId: string) =>
    useConnectionStore.getState().handleCampaignSwitch(newCampaignId)
});
