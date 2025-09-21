import { useConnectionStore } from '../stores/connectionStore';

// Connection status hook
export function useConnectionStatus() {
  const {
    connected,
    connecting,
    lastPing,
    reconnectAttempts,
    maxReconnectAttempts,
    reconnectDelay,
    wasmReady,
    wasmFunctions,
    setConnectionState,
    setWasmFunctions,
    reconnect,
    handleConnectionStatus,
    checkGlobalConnectionStatus
  } = useConnectionStore();

  return {
    connected,
    connecting,
    lastPing,
    reconnectAttempts,
    maxReconnectAttempts,
    reconnectDelay,
    wasmReady,
    wasmFunctions,
    isConnected: connected && wasmReady,
    setConnectionState,
    setWasmFunctions,
    reconnect,
    handleConnectionStatus,
    checkGlobalConnectionStatus
  };
}

// Media streaming hook
export function useMediaStreamingState() {
  const { mediaStreaming, setMediaStreamingState, clearMediaStreamingState } = useConnectionStore();

  return {
    mediaStreaming,
    ...mediaStreaming,
    setState: setMediaStreamingState,
    clearState: clearMediaStreamingState
  };
}
