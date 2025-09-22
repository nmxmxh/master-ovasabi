// JS/WASM bridge for using the WASM WebSocket client as a single source of truth for all real-time communication.
// This module handles proper type conversion at the Frontend↔WASM boundary.

import type { EventEnvelope } from '../store/types/events';

// Type for messages sent/received via WASM bridge
export interface WasmBridgeMessage {
  type: string;
  payload?: any;
  metadata?: any;
}

// Type definitions for WASM functions exposed to JavaScript
interface WASMFunctions {
  sendWasmMessage: (message: any) => void;
  getSharedBuffer: () => ArrayBuffer;
  getGPUMetricsBuffer: () => ArrayBuffer;
  getGPUComputeBuffer: () => ArrayBuffer;
  initWebGPU: () => boolean;
  runGPUCompute: (
    inputData: Float32Array,
    operation: number,
    callback: (result: Float32Array) => void
  ) => boolean;
  runGPUComputeWithOffset: (
    inputData: Float32Array,
    elapsedTime: number,
    globalParticleOffset: number,
    callback: (result: Float32Array) => void
  ) => boolean;
  runConcurrentCompute: (
    inputData: Float32Array,
    deltaTime: number,
    animationMode: number,
    callback: (result: Float32Array, metadata: any) => void
  ) => void;
  submitComputeTask: (
    taskType: string,
    data: Float32Array,
    callback: (result: Float32Array) => void,
    params?: { deltaTime?: number; animationMode?: number; priority?: number }
  ) => string | boolean;
  registerWasmPendingRequest: (correlationId: string, callback: (response: any) => void) => void;
  sendBinary: (type: string, payload: Uint8Array | ArrayBuffer, metadata: any) => void;
  infer: (input: Uint8Array) => Uint8Array;
  migrateUser: (newId: string) => void;
  reconnectWebSocket: () => void;
  submitGPUTask: (taskFn: () => void, callbackFn: () => void) => void;
  mediaStreaming: {
    connect: () => void;
    connectToCampaign: (campaignId: string, contextId: string, peerId: string) => void;
    send: (message: any) => void;
    onMessage: (callback: (data: any) => void) => void;
    onState: (callback: (state: string) => void) => void;
    isConnected: () => boolean;
    getURL: () => string;
  };
}

// Global WASM instance with proper typing
declare global {
  interface Window extends WASMFunctions {
    onWasmMessage?: (message: any) => void;
    processGPUFrame?: (buffer: Uint8Array) => void;
  }
}

// GPU operation types for centralized WASM GPU access
export enum GPUOperationType {
  PERFORMANCE_TEST = 0,
  PARTICLE_COMPUTE = 1,
  AI_INFERENCE = 2,
  DOM_INTERACTIVE = 3, // For real-time DOM sync operations
  BULK_PROCESSING = 4 // For large async computations
}

// GPU backend preference based on operation type
export enum GPUBackend {
  WEBGPU = 'webgpu',
  WEBGL = 'webgl',
  AUTO = 'auto'
}

// Enhanced GPU metrics interface with backend information
export interface GPUMetrics {
  timestamp: number;
  operation: GPUOperationType;
  backend: GPUBackend;
  dataSize: number;
  completionStatus: number;
  lastOperationTime: number;
  throughput: number;
  domSyncLatency?: number; // For DOM-interactive operations
}

// GPU capabilities interface for metadata integration
export interface GPUCapabilities {
  webgpu: {
    available: boolean;
    adapter?: {
      vendor?: string;
      architecture?: string;
      device?: string;
      description?: string;
    };
    features: string[];
    limits: {
      maxBufferSize?: number;
      maxComputeWorkgroupSize?: number;
      maxStorageBufferBindingSize?: number;
    };
  };
  webgl: {
    available: boolean;
    version: '1' | '2' | null;
    vendor?: string;
    renderer?: string;
    extensions: string[];
  };
  three: {
    optimized: boolean;
    recommendedRenderer: 'webgpu' | 'webgl2' | 'webgl';
    loadingMetrics: {
      totalLoadTime: number;
      memoryUsage: number;
      success: boolean;
    };
  };
  performance: {
    score: number;
    recommendation: string;
    benchmark?: {
      webgpuScore: number;
      webglScore: number;
      throughput?: number;
      duration?: number;
    };
  };
}

export class WasmGPUBridge {
  private initialized = false;
  private initPromise: Promise<boolean> | null = null;

  constructor() {
    // Wait for WASM to be ready before initializing GPU bridge
    if (typeof window !== 'undefined') {
      window.addEventListener('wasmReady', () => {
        this.initializeWASMGPU();
      });
    } else {
      // Fallback for non-browser environments
      this.initializeWASMGPU();
    }
  }

  private async initializeWASMGPU(): Promise<boolean> {
    if (this.initialized) return true;

    if (this.initPromise) {
      return this.initPromise;
    }

    this.initPromise = this.performWASMInitialization();
    return this.initPromise;
  }

  private async performWASMInitialization(): Promise<boolean> {
    try {
      // Wait for WASM functions to be available
      await this.waitForWASMFunctions();

      this.initialized = true;
      return true;
    } catch (error) {
      console.error('[WASM-GPU-Bridge] Initialization failed:', error);
      return false;
    }
  }

  private async waitForWASMFunctions(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('WASM functions not available after timeout'));
      }, 10000);

      const checkWASM = () => {
        if (this.areWASMFunctionsAvailable()) {
          clearTimeout(timeout);
          resolve();
        } else {
          setTimeout(checkWASM, 100);
        }
      };

      checkWASM();
    });
  }

  private areWASMFunctionsAvailable(): boolean {
    return !!(
      typeof window.sendWasmMessage === 'function' &&
      typeof window.getSharedBuffer === 'function' &&
      typeof window.initWebGPU === 'function'
    );
  }

  isInitialized(): boolean {
    return this.initialized;
  }

  async waitForInitialization(): Promise<boolean> {
    return this.initializeWASMGPU();
  }

  async getGPUCapabilities(): Promise<GPUCapabilities> {
    if (!this.initialized) {
      await this.waitForInitialization();
    }

    // Basic capability detection
    const webgpu = {
      available: typeof navigator !== 'undefined' && 'gpu' in navigator,
      features: [],
      limits: {}
    };

    const webgl = {
      available:
        typeof document !== 'undefined' && !!document.createElement('canvas').getContext('webgl'),
      version: null as '1' | '2' | null,
      extensions: []
    };

    return {
      webgpu,
      webgl,
      three: {
        optimized: true,
        recommendedRenderer: webgpu.available ? 'webgpu' : 'webgl2',
        loadingMetrics: {
          totalLoadTime: 0,
          memoryUsage: 0,
          success: true
        }
      },
      performance: {
        score: 85,
        recommendation: 'Good performance expected'
      }
    };
  }

  async updateMetadataWithGPUInfo(): Promise<void> {
    try {
      const capabilities = await this.getGPUCapabilities();

      // Update global metadata if available
      if (typeof window !== 'undefined' && (window as any).__WASM_GLOBAL_METADATA) {
        (window as any).__WASM_GLOBAL_METADATA.gpuCapabilities = capabilities;
      }
    } catch (error) {
      // Silently fail - GPU info is not critical
    }
  }

  getMetrics(): GPUMetrics | null {
    // Return basic metrics - detailed metrics should come from compute bridge
    return {
      timestamp: Date.now(),
      operation: GPUOperationType.PARTICLE_COMPUTE,
      backend: GPUBackend.AUTO,
      dataSize: 0,
      completionStatus: 1,
      lastOperationTime: 0,
      throughput: 0
    };
  }

  getComputeBuffer(): Float32Array | null {
    // Return null - compute buffer should be accessed through compute bridge
    return null;
  }

  async runPerformanceBenchmark(dataSize: number = 10000): Promise<any> {
    // Delegate to compute bridge if available
    try {
      const { wasmComputeBridge } = await import('./compute');
      return wasmComputeBridge.runBenchmark({ particleCount: dataSize });
    } catch (error) {
      return {
        throughput: 0,
        duration: 0,
        method: 'unavailable',
        threadingUsed: false
      };
    }
  }

  cleanup() {
    this.initialized = false;
    this.initPromise = null;
  }
}

// Singleton instance for communication bridge
export const wasmGPU = new WasmGPUBridge();

// Message handling and event system
type WasmListener = (msg: WasmBridgeMessage) => void;

let wasmListeners: WasmListener[] = [];
let lastCancellationReason: string | null = null;

export function getLastWasmCancellationReason(): string | null {
  return lastCancellationReason;
}

function notifyListeners(msg: any) {
  const wasmMsg = wasmMessageToTypescript(msg);

  // First, route to event store for processing
  try {
    import('../store/stores/eventStore').then(mod => {
      if (mod && mod.useEventStore) {
        const store = mod.useEventStore.getState();
        if (store.handleWasmMessage) {
          store.handleWasmMessage(msg);
        } else {
          console.error('[WASM-Bridge] Event store handleWasmMessage not available');
        }
      } else {
        console.error('[WASM-Bridge] Event store not available');
      }
    });
  } catch (err) {
    console.error('[WASM-Bridge] Failed to route message to event store:', err);
  }

  // Then notify other listeners
  wasmListeners.forEach(listener => {
    try {
      listener(wasmMsg);
    } catch (error) {
      console.error('[WASM-Bridge] Listener error:', error);
    }
  });
}

function wasmMessageToTypescript(wasmMsg: any): WasmBridgeMessage {
  return {
    type: wasmMsg.type || 'unknown',
    payload: wasmMsg.payload || wasmMsg.data || {},
    metadata: wasmMsg.metadata || {}
  };
}

// Media streaming functions
export function subscribeToMediaSignals(cb: (msg: any) => void): () => void {
  const listener = (msg: WasmBridgeMessage) => {
    if (msg.type.startsWith('media:')) {
      cb(msg);
    }
  };

  wasmListeners.push(listener);

  return () => {
    const index = wasmListeners.indexOf(listener);
    if (index > -1) {
      wasmListeners.splice(index, 1);
    }
  };
}

export function sendMediaSignal(msg: any) {
  if (typeof window.sendWasmMessage === 'function') {
    window.sendWasmMessage({
      type: 'media:signal',
      payload: msg,
      metadata: { timestamp: Date.now() }
    });
  }
}

export function subscribeToWasmMessages(cb: WasmListener): () => void {
  wasmListeners.push(cb);

  return () => {
    const index = wasmListeners.indexOf(cb);
    if (index > -1) {
      wasmListeners.splice(index, 1);
    }
  };
}

export function wasmSendMessage(msg: WasmBridgeMessage | EventEnvelope) {
  if (typeof window.sendWasmMessage !== 'function') {
    console.warn('[WASM-Bridge] sendWasmMessage not available');
    return;
  }

  try {
    // Check if this is an EventEnvelope with canonical structure
    if ('correlation_id' in msg && 'version' in msg && 'environment' in msg && 'source' in msg) {
      // This is a canonical EventEnvelope - send it directly
      window.sendWasmMessage(msg);
    } else {
      // Handle legacy WasmBridgeMessage format
      const wasmMsg =
        'type' in msg && 'payload' in msg
          ? (msg as WasmBridgeMessage)
          : {
              type: (msg as EventEnvelope).type || 'unknown',
              payload: (msg as EventEnvelope).payload || {},
              metadata: (msg as EventEnvelope).metadata || {}
            };

      window.sendWasmMessage(wasmMsg);
    }
  } catch (error) {
    console.error('[WASM-Bridge] Error sending message:', error);
  }
}

// Media streaming connection functions
export function connectMediaStreamingToCampaign(
  campaignId: string = '0',
  contextId: string = 'webgpu-particles',
  peerId?: string
): void {
  if (typeof window.mediaStreaming?.connectToCampaign === 'function') {
    window.mediaStreaming.connectToCampaign(campaignId, contextId, peerId || 'default-peer');
  } else {
    console.warn('[WASM-Bridge] Media streaming not available');
  }
}

export function isMediaStreamingConnected(): boolean {
  return typeof window.mediaStreaming?.isConnected === 'function'
    ? window.mediaStreaming.isConnected()
    : false;
}

export function getMediaStreamingURL(): string {
  return typeof window.mediaStreaming?.getURL === 'function' ? window.mediaStreaming.getURL() : '';
}

export function sendMediaStreamingMessage(message: any): void {
  if (typeof window.mediaStreaming?.send === 'function') {
    window.mediaStreaming.send(message);
  } else {
    console.warn('[WASM-Bridge] Media streaming send not available');
  }
}

// Campaign switch handling
export function setupCampaignSwitchHandler(): void {
  if (typeof window === 'undefined') return;

  // Set up the global handler that WASM will call
  (window as any).onCampaignSwitchRequired = (switchEvent: {
    old_campaign_id: string;
    new_campaign_id: string;
    reason: string;
    timestamp?: string;
  }) => {
    // Import the store dynamically to avoid circular dependencies
    import('../store/stores/campaignStore').then(({ useCampaignStore }) => {
      useCampaignStore.getState().handleCampaignSwitchRequired(switchEvent);
    });
  };

  (window as any).onCampaignSwitchCompleted = (switchEvent: {
    old_campaign_id: string;
    new_campaign_id: string;
    reason: string;
    timestamp?: string;
    status: string;
  }) => {
    // Import the store dynamically to avoid circular dependencies
    import('../store/stores/campaignStore').then(({ useCampaignStore }) => {
      useCampaignStore.getState().handleCampaignSwitchCompleted(switchEvent);
    });
  };
}

export function subscribeToMediaStreamingState(callback: (state: string) => void): void {
  if (typeof window.mediaStreaming?.onState === 'function') {
    window.mediaStreaming.onState(callback);
  }
}

export function subscribeToMediaStreamingMessages(callback: (data: any) => void): void {
  if (typeof window.mediaStreaming?.onMessage === 'function') {
    window.mediaStreaming.onMessage(callback);
  }
}

export function disconnectMediaStreaming(): void {
  // Implementation depends on WASM media streaming API
}

// Initialize message handling
if (typeof window !== 'undefined') {
  window.onWasmMessage = notifyListeners;
}
