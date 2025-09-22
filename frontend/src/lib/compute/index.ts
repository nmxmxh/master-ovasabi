/**
 * Compute Module - Unified interface for all compute operations
 * Provides clean separation between worker management and WASM bridge
 */

export {
  workerManager,
  type WorkerTask,
  type WorkerResult,
  type WorkerCapabilities,
  type WorkerMetrics
} from './workerManager';
export {
  wasmComputeBridge,
  type WASMComputeOptions,
  type WASMParticleData
} from './wasmComputeBridge';

// Re-export for backward compatibility
export { wasmComputeBridge as wasmGPU } from './wasmComputeBridge';
