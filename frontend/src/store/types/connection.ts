// Connection-related type definitions
export interface ConnectionState {
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

export interface MediaStreamingState {
  connected: boolean;
  connecting: boolean;
  campaignId: string;
  contextId: string;
  peerId: string;
  url: string;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
  streamInfo?: any;
  error?: string;
  lastConnectAttempt?: string;
}
