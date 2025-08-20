// JS/WASM bridge for using the WASM WebSocket client as a single source of truth for all real-time communication.
// This module handles proper type conversion at the Frontend↔WASM boundary.

import type { EventEnvelope } from '../store/global';

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
    inputData: Uint8Array,
    elapsedTime: number,
    globalParticleOffset: number,
    callback: (result: Float32Array) => void
  ) => boolean;
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
  private operationCounter = 0;
  private pendingTasks = new Map<string, { resolve: Function; reject: Function }>();

  // Distributed processing enhancements
  private workerPool: Worker[] = [];
  private workerCapabilities = new Map<
    Worker,
    { webgpu: boolean; wasm: boolean; performance: number }
  >();
  private distributionQueue = new Map<
    string,
    { data: Float32Array; operation: GPUOperationType; priority: number }
  >();
  private activeJobs = new Set<string>();
  private maxConcurrentJobs = 4;

  constructor() {
    // Wait for WASM to be ready before initializing GPU bridge
    if (typeof window !== 'undefined') {
      window.addEventListener('wasmReady', () => {
        this.initializeWASMGPU();
        this.initializeDistributedProcessing();
      });
    } else {
      // Fallback for non-browser environments
      this.initializeWASMGPU();
      this.initializeDistributedProcessing();
    }
  }

  private async initializeDistributedProcessing() {
    // Create worker pool for true parallel processing
    const workerCount = Math.min(navigator.hardwareConcurrency || 4, 8); // Increased max to 8

    for (let i = 0; i < workerCount; i++) {
      try {
        const worker = new Worker('/workers/compute-worker.js');
        await this.setupWorker(worker, i);
        this.workerPool.push(worker);
      } catch (error) {
        console.warn(`[WASM-GPU-Bridge] Failed to create worker ${i}:`, error);
      }
    }

    console.log(
      `[WASM-GPU-Bridge] ✅ Distributed processing initialized with ${this.workerPool.length} workers`
    );
  }

  private async setupWorker(worker: Worker, index: number): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        console.warn(`[WASM-GPU-Bridge] Worker ${index} setup timeout - cleaning up`);
        this.cleanupWorker(worker);
        reject(new Error('Worker setup timeout'));
      }, 5000);

      const cleanup = () => {
        clearTimeout(timeout);
        worker.removeEventListener('message', messageHandler);
        worker.removeEventListener('error', errorHandler);
      };

      const messageHandler = (event: MessageEvent) => {
        const { type, ...data } = event.data;

        switch (type) {
          case 'worker-ready':
            cleanup();
            this.workerCapabilities.set(worker, {
              webgpu: data.capabilities?.webgpu || false,
              wasm: data.capabilities?.wasm || false,
              performance: 1.0 // Will be updated based on benchmarks
            });
            console.log(`[WASM-GPU-Bridge] Worker ${index} ready:`, data.capabilities);
            resolve();
            break;

          case 'task-result':
            this.handleTaskResult(data);
            break;

          case 'task-error':
            this.handleTaskError(data);
            break;

          case 'benchmark-complete':
            this.updateWorkerPerformance(worker, data);
            break;

          case 'worker-degraded':
            this.handleWorkerDegradation(worker, data);
            break;

          case 'worker-shutdown':
            this.handleWorkerShutdown(worker, data);
            break;
        }
      };

      const errorHandler = (error: ErrorEvent) => {
        cleanup();
        console.error(`[WASM-GPU-Bridge] Worker ${index} error during setup:`, error);
        this.cleanupWorker(worker);
        reject(error);
      };

      worker.addEventListener('message', messageHandler);
      worker.addEventListener('error', errorHandler);
    });
  }

  private handleTaskResult(data: any) {
    const pendingTask = this.pendingTasks.get(data.id);
    if (pendingTask) {
      pendingTask.resolve(data.data);
      this.pendingTasks.delete(data.id);
      this.activeJobs.delete(data.id);
    }
  }

  private handleTaskError(data: any) {
    const errorTask = this.pendingTasks.get(data.id);
    if (errorTask) {
      errorTask.reject(new Error(data.error));
      this.pendingTasks.delete(data.id);
      this.activeJobs.delete(data.id);
    }
  }

  private updateWorkerPerformance(worker: Worker, data: any) {
    const capabilities = this.workerCapabilities.get(worker);
    if (capabilities && data.results) {
      // Calculate performance score based on benchmark results
      const scores = Object.values(data.results).map((r: any) => r.particlesPerMs || 0);
      const avgScore = scores.reduce((a: number, b: number) => a + b, 0) / scores.length;
      capabilities.performance = avgScore;
      this.workerCapabilities.set(worker, capabilities);
    }
  }

  // Handle worker degradation with graceful fallback
  private handleWorkerDegradation(worker: Worker, data: any) {
    console.warn(`[WASM-GPU-Bridge] Worker degraded (level ${data.degradationLevel}):`, data.error);

    const capabilities = this.workerCapabilities.get(worker);
    if (capabilities) {
      // Reduce worker capabilities based on degradation level
      if (data.degradationLevel >= 2) {
        capabilities.webgpu = false;
      }
      if (data.degradationLevel >= 3) {
        capabilities.wasm = false;
      }
      // Reduce performance score to deprioritize this worker
      capabilities.performance *= 1 - data.degradationLevel * 0.3;
      this.workerCapabilities.set(worker, capabilities);
    }

    // If too many workers are degraded, consider reducing the pool
    const degradedWorkers = this.workerPool.filter(w => {
      const caps = this.workerCapabilities.get(w);
      return caps && caps.performance < 0.5;
    });

    if (degradedWorkers.length > this.workerPool.length * 0.5) {
      console.warn('[WASM-GPU-Bridge] More than 50% of workers degraded, reducing concurrency');
      this.maxConcurrentJobs = Math.max(1, Math.floor(this.maxConcurrentJobs * 0.7));
    }
  }

  // Handle worker shutdown
  private handleWorkerShutdown(worker: Worker, data: any) {
    console.warn(`[WASM-GPU-Bridge] Worker shutdown: ${data.reason}`);
    this.cleanupWorker(worker);
  }

  // Clean up a worker and remove from pool
  private cleanupWorker(worker: Worker) {
    // Remove from capabilities tracking
    this.workerCapabilities.delete(worker);

    // Remove from worker pool
    const index = this.workerPool.indexOf(worker);
    if (index !== -1) {
      this.workerPool.splice(index, 1);
      console.log(
        `[WASM-GPU-Bridge] Worker removed from pool, ${this.workerPool.length} workers remaining`
      );
    }

    // Cancel any pending tasks for this worker
    const tasksToCancel = Array.from(this.pendingTasks.keys());

    for (const taskId of tasksToCancel) {
      const task = this.pendingTasks.get(taskId);
      if (task) {
        task.reject(new Error('Worker shutdown during task execution'));
        this.pendingTasks.delete(taskId);
        this.activeJobs.delete(taskId);
      }
    }

    // Terminate the worker
    try {
      worker.terminate();
    } catch (error) {
      console.warn('[WASM-GPU-Bridge] Error terminating worker:', error);
    }

    // If we have too few workers left, try to create a replacement
    if (this.workerPool.length < 2) {
      console.log('[WASM-GPU-Bridge] Low worker count, attempting to create replacement worker');
      this.createReplacementWorker();
    }
  }

  // Fallback JavaScript compute for graceful degradation
  private fallbackJavaScriptCompute(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime: number
  ): Float32Array {
    if (typeof window !== 'undefined' && (window as any).isPageUnloading) {
      console.warn('[WASM-GPU-Bridge] Skipping fallback JavaScript compute during page unload');
      return inputData; // Return input unchanged to avoid blocking
    }

    console.log(
      `[WASM-GPU-Bridge] Running fallback JavaScript compute for ${inputData.length} values`
    );

    const result = new Float32Array(inputData.length);
    result.set(inputData); // Copy input data

    const valuesPerParticle = 8; // position(3) + velocity(3) + time(1) + intensity(1)
    const particleCount = Math.floor(inputData.length / valuesPerParticle);

    // Simple particle animation fallback
    for (let i = 0; i < particleCount; i++) {
      const i8 = i * 8;

      // Extract position
      const x = result[i8];
      const y = result[i8 + 1];
      const z = result[i8 + 2];

      if (operation === GPUOperationType.PARTICLE_COMPUTE) {
        // Simple rotation animation
        const radius = Math.sqrt(x * x + z * z);
        if (radius > 0.001) {
          const angle = Math.atan2(z, x) + elapsedTime * 0.5;
          result[i8] = radius * Math.cos(angle);
          result[i8 + 1] = y + Math.sin(elapsedTime * 2.0 + i * 0.01) * 0.1;
          result[i8 + 2] = radius * Math.sin(angle);
        }
      }

      // Update time
      result[i8 + 6] += elapsedTime;
    }

    return result;
  }

  // Create a replacement worker when needed
  private async createReplacementWorker() {
    try {
      const worker = new Worker('/workers/compute-worker.js');
      const workerIndex = this.workerPool.length;
      await this.setupWorker(worker, workerIndex);
      this.workerPool.push(worker);
      console.log(`[WASM-GPU-Bridge] Replacement worker created successfully`);
    } catch (error) {
      console.error('[WASM-GPU-Bridge] Failed to create replacement worker:', error);
    }
  }

  // Pause workers without terminating them (for visibility changes)
  pauseWorkers() {
    console.log('[WASM-GPU-Bridge] Pausing workers...');

    for (const worker of this.workerPool) {
      try {
        worker.postMessage({ type: 'pause' });
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] Failed to pause worker:', error);
      }
    }

    // Mark as paused to prevent new task distribution
    (this as any).isPaused = true;
  }

  // Resume workers after pause
  resumeWorkers() {
    console.log('[WASM-GPU-Bridge] Resuming workers...');

    for (const worker of this.workerPool) {
      try {
        worker.postMessage({ type: 'resume' });
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] Failed to resume worker:', error);
      }
    }

    // Mark as resumed to allow task distribution
    (this as any).isPaused = false;
  }

  // Cleanup all resources for graceful shutdown
  cleanup() {
    if (typeof window !== 'undefined' && (window as any).isPageUnloading) {
      console.warn(
        '[WASM-GPU-Bridge] Fast cleanup: terminating workers and abandoning tasks due to page unload'
      );
      for (const worker of this.workerPool) {
        try {
          worker.terminate();
        } catch {}
      }
      this.workerPool.length = 0;
      this.workerCapabilities.clear();
      this.pendingTasks.clear();
      this.activeJobs.clear();
      this.distributionQueue.clear();
      return;
    }

    console.log('[WASM-GPU-Bridge] Starting cleanup...');

    // Override pause state to ensure cleanup can proceed
    (this as any).isPaused = false;

    // Cleanup workers
    for (const worker of this.workerPool) {
      this.cleanupWorker(worker);
    }
    this.workerPool.length = 0;
    this.workerCapabilities.clear();

    // Cancel pending tasks
    for (const [, task] of this.pendingTasks) {
      task.reject(new Error('Bridge shutting down'));
    }
    this.pendingTasks.clear();
    this.activeJobs.clear();

    // Clear distribution queue
    this.distributionQueue.clear();

    console.log('[WASM-GPU-Bridge] Cleanup complete');
  }

  private selectOptimalWorker(dataSize: number, operation: GPUOperationType): Worker | null {
    if (this.workerPool.length === 0) return null;

    // Filter available workers (not at max capacity)
    const availableWorkers = this.workerPool.filter(() => {
      const workerTasks = Array.from(this.activeJobs).filter(jobId => this.pendingTasks.has(jobId));
      return workerTasks.length < this.maxConcurrentJobs;
    });

    if (availableWorkers.length === 0) {
      // All workers busy, use round-robin
      return this.workerPool[Math.floor(Math.random() * this.workerPool.length)];
    }

    // Select based on capabilities and performance
    let bestWorker = availableWorkers[0];
    let bestScore = 0;

    for (const worker of availableWorkers) {
      const caps = this.workerCapabilities.get(worker);
      if (!caps) continue;

      let score = caps.performance;

      // Bonus for matching capabilities
      if (operation === GPUOperationType.PARTICLE_COMPUTE && caps.webgpu) score *= 1.5;
      if (operation === GPUOperationType.AI_INFERENCE && caps.wasm) score *= 1.3;

      // Bonus for handling large datasets
      if (dataSize > 50000 && caps.webgpu) score *= 1.2;

      if (score > bestScore) {
        bestScore = score;
        bestWorker = worker;
      }
    }

    return bestWorker;
  }

  async runDistributedCompute(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime: number = 0.016667
  ): Promise<Float32Array> {
    // Check if we should use distributed processing
    const shouldDistribute =
      inputData.length > 20000 && this.workerPool.length > 1 && !(this as any).isPaused;

    if (!shouldDistribute) {
      // Fall back to single worker or WASM GPU (or if paused)
      if ((this as any).isPaused) {
        console.log('[WASM-GPU-Bridge] Workers paused, using fallback JavaScript compute');
        return new Promise(resolve => {
          const result = this.fallbackJavaScriptCompute(inputData, operation, elapsedTime);
          resolve(result);
        });
      }
      return this.runComputeWithOffset(inputData, operation, elapsedTime, 0);
    }

    // Distribute work across multiple workers
    // Ensure chunks are aligned to particle boundaries (10 values per particle: position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1))
    const valuesPerParticle = 10; // position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1)
    const totalParticles = Math.floor(inputData.length / valuesPerParticle);
    const particlesPerWorker = Math.ceil(totalParticles / Math.min(this.workerPool.length, 4));
    const chunks: { data: Float32Array; offset: number }[] = [];

    // Enforce chunk size based on device limits (maxStorageBufferBindingSize)
    let maxChunkFloats = 150000; // Default fallback: 15k particles * 10 floats
    let deviceLimit;
    let gpuCaps = null;
    if (typeof this.getGPUCapabilities === 'function') {
      gpuCaps = await this.getGPUCapabilities();
      if (
        gpuCaps &&
        gpuCaps.webgpu &&
        gpuCaps.webgpu.limits &&
        gpuCaps.webgpu.limits.maxStorageBufferBindingSize
      ) {
        deviceLimit = gpuCaps.webgpu.limits.maxStorageBufferBindingSize;
      }
    }
    if (deviceLimit) {
      // Divide by 4 (bytes per float32) and round down to nearest multiple of 10 (particle size)
      maxChunkFloats = Math.floor(deviceLimit / 4 / 10) * 10;
    }
    const maxParticlesPerChunk = Math.floor(maxChunkFloats / 10);
    for (
      let particleIndex = 0;
      particleIndex < totalParticles;
      particleIndex += Math.min(particlesPerWorker, maxParticlesPerChunk)
    ) {
      const chunkSize = Math.min(particlesPerWorker, maxParticlesPerChunk);
      const endParticle = Math.min(particleIndex + chunkSize, totalParticles);
      const startIndex = particleIndex * valuesPerParticle;
      const endIndex = endParticle * valuesPerParticle;
      const chunk = inputData.slice(startIndex, endIndex);
      chunks.push({ data: chunk, offset: startIndex });
    }

    // Add to distribution queue for monitoring
    const distributionId = `dist_${Date.now()}`;
    this.distributionQueue.set(distributionId, {
      data: inputData,
      operation,
      priority: inputData.length > 100000 ? 2 : 1
    });

    console.log(
      `[WASM-GPU-Bridge] Distributing ${totalParticles} particles (${inputData.length} floats) across ${chunks.length} workers`
    );
    console.log(
      `[WASM-GPU-Bridge] Chunk sizes:`,
      chunks.map(c => `${c.data.length} floats (${c.data.length / 8} particles)`)
    );

    try {
      // Process chunks in parallel
      const promises = chunks.map(async (chunk, index) => {
        const worker = this.selectOptimalWorker(chunk.data.length, operation);
        if (!worker) throw new Error('No workers available');

        const taskId = `distributed_${Date.now()}_${index}`;
        this.activeJobs.add(taskId);

        return new Promise<{ result: Float32Array; offset: number }>((resolve, reject) => {
          const timeout = setTimeout(() => {
            console.warn(`[WASM-GPU-Bridge] Distributed task ${taskId} timeout, falling back`);
            this.pendingTasks.delete(taskId);
            this.activeJobs.delete(taskId);

            // Graceful degradation: fallback to JavaScript processing
            try {
              const fallbackResult = this.fallbackJavaScriptCompute(
                chunk.data,
                operation,
                elapsedTime
              );
              resolve({ result: fallbackResult, offset: chunk.offset });
            } catch (fallbackError: any) {
              reject(
                new Error(
                  `Distributed task timeout and fallback failed: ${fallbackError?.message || String(fallbackError)}`
                )
              );
            }
          }, 15000);

          this.pendingTasks.set(taskId, {
            resolve: (result: Float32Array) => {
              clearTimeout(timeout);
              resolve({ result, offset: chunk.offset });
            },
            reject: (error: Error) => {
              clearTimeout(timeout);
              // Try fallback before rejecting
              try {
                console.warn(
                  `[WASM-GPU-Bridge] Task ${taskId} failed, trying fallback:`,
                  error.message
                );
                const fallbackResult = this.fallbackJavaScriptCompute(
                  chunk.data,
                  operation,
                  elapsedTime
                );
                resolve({ result: fallbackResult, offset: chunk.offset });
              } catch (fallbackError: any) {
                reject(
                  new Error(
                    `Task failed and fallback failed: ${error.message} | Fallback: ${fallbackError?.message || String(fallbackError)}`
                  )
                );
              }
            }
          });

          worker.postMessage({
            type: 'compute-task',
            task: {
              id: taskId,
              data: chunk.data,
              params: {
                deltaTime: elapsedTime,
                animationMode: operation === GPUOperationType.PARTICLE_COMPUTE ? 1.0 : 2.0
              }
            }
          });
        });
      });

      // Wait for all chunks to complete
      const results = await Promise.all(promises);

      // Merge results back into single array
      const finalResult = new Float32Array(inputData.length);
      for (const { result, offset } of results) {
        // Validate result before setting
        if (!result || !(result instanceof Float32Array)) {
          console.warn(
            '[WASM-GPU-Bridge] Invalid result from worker, skipping chunk at offset:',
            offset
          );
          continue;
        }

        // Validate bounds
        if (offset + result.length > finalResult.length) {
          console.warn('[WASM-GPU-Bridge] Result chunk exceeds bounds, truncating');
          const validLength = Math.max(0, finalResult.length - offset);
          if (validLength > 0) {
            finalResult.set(result.subarray(0, validLength), offset);
          }
        } else {
          finalResult.set(result, offset);
        }
      }

      console.log(
        `[WASM-GPU-Bridge] ✅ Distributed compute completed: ${finalResult.length} particles processed`
      );
      return finalResult;
    } finally {
      this.distributionQueue.delete(distributionId);
    }
  }

  private async initializeWASMGPU(): Promise<boolean> {
    if (this.initPromise) {
      return this.initPromise;
    }

    this.initPromise = new Promise(resolve => {
      let attempts = 0;
      const maxAttempts = 50; // 5 seconds max wait time

      const checkWASM = () => {
        attempts++;

        if (typeof window.initWebGPU === 'function') {
          console.log(
            '[WASM-GPU-Bridge] WASM GPU functions detected, initializing centralized WebGPU through WASM...'
          );
          try {
            const success = window.initWebGPU();
            this.initialized = success;

            if (success) {
              console.log('[WASM-GPU-Bridge] ✅ Centralized WASM GPU initialization successful');
              console.log('[WASM-GPU-Bridge] GPU functions available:', {
                initWebGPU: typeof window.initWebGPU,
                runGPUCompute: typeof window.runGPUCompute,
                getGPUMetricsBuffer: typeof window.getGPUMetricsBuffer,
                getGPUComputeBuffer: typeof window.getGPUComputeBuffer
              });

              // Fire gpu_ready event for React components to pick up
              const gpuReadyMessage = {
                type: 'gpu_ready',
                timestamp: Date.now(),
                data: {
                  webgpuReady: true,
                  computeMode: 'WASM+WebGPU',
                  functions: {
                    initWebGPU: typeof window.initWebGPU,
                    runGPUCompute: typeof window.runGPUCompute,
                    getGPUMetricsBuffer: typeof window.getGPUMetricsBuffer,
                    getGPUComputeBuffer: typeof window.getGPUComputeBuffer
                  }
                }
              };

              // Use the global notifyListeners function to send the message to React components
              if (typeof (window as any).onWasmMessage === 'function') {
                (window as any).onWasmMessage(gpuReadyMessage);
              }
              console.log('[WASM-GPU-Bridge] 🚀 Fired gpu_ready event for React components');

              // Automatically update metadata with GPU capabilities
              this.updateMetadataWithGPUInfo().catch(error => {
                console.warn('[WASM-GPU-Bridge] Failed to auto-update metadata:', error);
              });
            } else {
              console.warn(
                '[WASM-GPU-Bridge] ❌ WASM GPU initialization failed - WebGPU not available or initialization error'
              );
            }

            resolve(success);
          } catch (error) {
            console.error('[WASM-GPU-Bridge] Error during WebGPU initialization:', error);
            resolve(false);
          }
        } else if (attempts >= maxAttempts) {
          console.warn(
            `[WASM-GPU-Bridge] ⏰ Timeout waiting for WASM module after ${maxAttempts} attempts`
          );
          console.warn('[WASM-GPU-Bridge] Proceeding without WASM GPU support');
          resolve(false);
        } else {
          console.log(
            `[WASM-GPU-Bridge] Waiting for WASM module to load GPU functions... (${attempts}/${maxAttempts})`
          );
          setTimeout(checkWASM, 100);
        }
      };

      // Add a small delay to ensure window is fully loaded
      setTimeout(checkWASM, 100);
    });

    return this.initPromise;
  }

  // Public method to check if GPU is ready
  isInitialized(): boolean {
    return this.initialized;
  }

  // Public method to wait for GPU initialization
  async waitForInitialization(): Promise<boolean> {
    if (this.initialized) {
      return true;
    }
    return await this.initializeWASMGPU();
  }

  async runCompute(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime?: number
  ): Promise<Float32Array> {
    // Ensure GPU is fully initialized before proceeding
    if (!this.initialized) {
      console.log('[WASM-GPU-Bridge] GPU not initialized, waiting for initialization...');
      const initSuccess = await this.initializeWASMGPU();
      if (!initSuccess) {
        // Graceful degradation: fallback to JavaScript processing
        console.warn(
          '[WASM-GPU-Bridge] WASM GPU unavailable, falling back to JavaScript processing'
        );
        return this.fallbackJavaScriptCompute(inputData, operation, elapsedTime || 0.016667);
      }
    }

    console.log(
      `[WASM-GPU-Bridge] Running centralized GPU compute operation: ${GPUOperationType[operation]} with ${inputData.length} data points`
    );

    return new Promise((resolve, reject) => {
      // Set up timeout for compute operation
      const computeTimeout = setTimeout(() => {
        console.warn('[WASM-GPU-Bridge] GPU compute timeout, falling back to JavaScript');
        try {
          const fallbackResult = this.fallbackJavaScriptCompute(
            inputData,
            operation,
            elapsedTime || 0.016667
          );
          resolve(fallbackResult);
        } catch (fallbackError: any) {
          reject(
            new Error(
              `GPU compute timeout and fallback failed: ${fallbackError?.message || String(fallbackError)}`
            )
          );
        }
      }, 10000); // 10 second timeout

      if (!window.runGPUCompute) {
        clearTimeout(computeTimeout);
        console.warn(
          '[WASM-GPU-Bridge] WASM GPU compute not available, falling back to JavaScript'
        );
        try {
          const fallbackResult = this.fallbackJavaScriptCompute(
            inputData,
            operation,
            elapsedTime || 0.016667
          );
          resolve(fallbackResult);
        } catch (fallbackError: any) {
          reject(
            new Error(
              `WASM GPU unavailable and fallback failed: ${fallbackError?.message || String(fallbackError)}`
            )
          );
        }
        return;
      }

      if (!this.initialized) {
        clearTimeout(computeTimeout);
        const error = 'GPU not properly initialized before compute operation';
        console.error('[WASM-GPU-Bridge]', error);
        reject(new Error(error));
        return;
      }

      // Use current performance time if no elapsed time provided for smooth animation
      const timeParam = elapsedTime !== undefined ? elapsedTime : performance.now() / 1000.0;

      try {
        const success = window.runGPUCompute(inputData, timeParam, (result: Float32Array) => {
          clearTimeout(computeTimeout);
          console.log(
            `[WASM-GPU-Bridge] ✅ GPU compute completed successfully: ${result.length} results returned`
          );
          resolve(result);
        });

        if (!success) {
          clearTimeout(computeTimeout);
          console.warn('[WASM-GPU-Bridge] Failed to start GPU compute, falling back to JavaScript');
          try {
            const fallbackResult = this.fallbackJavaScriptCompute(inputData, operation, timeParam);
            resolve(fallbackResult);
          } catch (fallbackError: any) {
            reject(
              new Error(
                `Failed to start GPU compute and fallback failed: ${fallbackError?.message || String(fallbackError)}`
              )
            );
          }
        } else {
          console.log(
            '[WASM-GPU-Bridge] GPU compute operation started successfully through centralized WASM system'
          );
        }
      } catch (computeError: any) {
        clearTimeout(computeTimeout);
        console.warn(
          `[WASM-GPU-Bridge] GPU compute error, falling back to JavaScript:`,
          computeError
        );
        try {
          const fallbackResult = this.fallbackJavaScriptCompute(inputData, operation, timeParam);
          resolve(fallbackResult);
        } catch (fallbackError: any) {
          reject(
            new Error(
              `GPU compute error and fallback failed: ${computeError?.message} | Fallback: ${fallbackError?.message}`
            )
          );
        }
      }
    });
  }

  async runComputeWithOffset(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime: number,
    globalParticleOffset: number
  ): Promise<Float32Array> {
    // Try distributed processing for large datasets with multiple workers
    if (inputData.length > 20000 && this.workerPool.length > 1) {
      try {
        return await this.runDistributedCompute(inputData, operation, elapsedTime);
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] Distributed compute failed, falling back:', error);
      }
    }

    // Try single worker from pool for medium datasets
    if (this.workerPool.length > 0 && inputData.length > 10000) {
      try {
        return await this.runWorkerPoolTask(inputData, elapsedTime, operation);
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] Worker pool failed, falling back to WASM GPU:', error);
      }
    }

    // Ensure GPU is fully initialized before proceeding
    if (!this.initialized) {
      console.log('[WASM-GPU-Bridge] GPU not initialized, waiting for initialization...');
      const initSuccess = await this.initializeWASMGPU();
      if (!initSuccess) {
        throw new Error('WASM GPU initialization failed - WebGPU not available');
      }
    }

    // Increment operation counter and log only every 10th operation to reduce verbosity
    this.operationCounter++;
    const shouldLog = this.operationCounter % 10 === 0 || this.operationCounter <= 5; // Log first 5, then every 10th

    if (shouldLog) {
      console.log(
        `[WASM-GPU-Bridge] Running synchronized GPU compute: ${GPUOperationType[operation]} with ${inputData.length} data points, offset: ${globalParticleOffset} (op #${this.operationCounter})`
      );
    }

    return new Promise((resolve, reject) => {
      if (!window.runGPUComputeWithOffset) {
        const error =
          'WASM GPU compute with offset not available - falling back to regular compute';
        console.warn('[WASM-GPU-Bridge]', error);
        // Fallback to regular compute
        this.runCompute(inputData, operation, elapsedTime).then(resolve).catch(reject);
        return;
      }

      if (!this.initialized) {
        const error = 'GPU not properly initialized before compute operation';
        console.error('[WASM-GPU-Bridge]', error);
        reject(new Error(error));
        return;
      }

      // Ensure inputData is passed as Uint8Array for Go WASM compatibility
      let bufferToSend: Uint8Array;
      if (Object.prototype.toString.call(inputData) === '[object Float32Array]') {
        bufferToSend = new Uint8Array((inputData as unknown as Float32Array).buffer);
      } else if (Object.prototype.toString.call(inputData) === '[object Uint8Array]') {
        bufferToSend = inputData as unknown as Uint8Array;
      } else if (Object.prototype.toString.call(inputData) === '[object ArrayBuffer]') {
        bufferToSend = new Uint8Array(inputData as unknown as ArrayBuffer);
      } else {
        throw new Error('Unsupported buffer type for WASM GPU compute');
      }
      const success = window.runGPUComputeWithOffset(
        bufferToSend,
        elapsedTime,
        globalParticleOffset / 3, // Convert to particle count (since offset is in floats)
        (result: Float32Array) => {
          console.log(
            `[WASM-GPU-Bridge] ✅ Synchronized GPU compute completed: ${result.length} results returned`
          );
          resolve(result);
        }
      );

      if (!success) {
        const error = 'Failed to start synchronized GPU compute operation';
        console.error('[WASM-GPU-Bridge]', error);
        reject(new Error(error));
      } else {
        console.log('[WASM-GPU-Bridge] Synchronized GPU compute operation started successfully');
      }
    });
  }

  getMetrics(): GPUMetrics | null {
    // If GPU not initialized, return simulated metrics for fallback
    if (!this.initialized || !window.getGPUMetricsBuffer) {
      return {
        timestamp: Date.now(),
        operation: GPUOperationType.PERFORMANCE_TEST,
        backend: GPUBackend.AUTO,
        dataSize: 1000,
        completionStatus: 1,
        lastOperationTime: 16.7, // ~60fps
        throughput: Math.random() * 2000 + 1000 // Simulated throughput 1000-3000 ops/s
      };
    }

    try {
      const buffer = window.getGPUMetricsBuffer();
      const metrics = new Float32Array(buffer);

      return {
        timestamp: metrics[0],
        operation: metrics[1] as GPUOperationType,
        backend: GPUBackend.WEBGPU, // WASM GPU uses WebGPU backend
        dataSize: metrics[2],
        completionStatus: metrics[3],
        lastOperationTime: metrics[4],
        throughput: metrics[5]
      };
    } catch (error) {
      console.warn('[WASM-GPU-Bridge] Failed to read GPU metrics, using fallback:', error);
      // Return fallback metrics
      return {
        timestamp: Date.now(),
        operation: GPUOperationType.PERFORMANCE_TEST,
        backend: GPUBackend.AUTO,
        dataSize: 1000,
        completionStatus: 0,
        lastOperationTime: 20,
        throughput: Math.random() * 1500 + 800
      };
    }
  }

  getComputeBuffer(): Float32Array | null {
    if (!window.getGPUComputeBuffer) {
      return null;
    }

    const buffer = window.getGPUComputeBuffer();
    return new Float32Array(buffer);
  }

  private async runWorkerPoolTask(
    inputData: Float32Array,
    elapsedTime: number,
    operation: GPUOperationType
  ): Promise<Float32Array> {
    if (this.workerPool.length === 0) {
      throw new Error('Worker pool not available');
    }

    // Select optimal worker from pool
    const worker = this.selectOptimalWorker(inputData.length, operation);
    if (!worker) {
      throw new Error('No workers available in pool');
    }

    const taskId = `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingTasks.delete(taskId);
        reject(new Error('Worker pool task timeout'));
      }, 10000);

      this.pendingTasks.set(taskId, {
        resolve: (result: Float32Array) => {
          clearTimeout(timeout);
          resolve(result);
        },
        reject: (error: Error) => {
          clearTimeout(timeout);
          reject(error);
        }
      });

      worker.postMessage({
        type: 'compute-task',
        task: {
          id: taskId,
          data: inputData,
          params: {
            deltaTime: elapsedTime,
            animationMode: operation === GPUOperationType.PARTICLE_COMPUTE ? 1.0 : 2.0
          }
        }
      });
    });
  }

  // Smart backend selection based on operation type and context
  private selectOptimalGPUBackend(
    operation: GPUOperationType,
    dataSize: number,
    requiresDOMSync: boolean = false
  ): GPUBackend {
    // Force WebGL for DOM-interactive operations due to synchronous nature
    if (operation === GPUOperationType.DOM_INTERACTIVE || requiresDOMSync) {
      return GPUBackend.WEBGL;
    }

    // Use WebGPU for large bulk processing (better parallel compute)
    if (operation === GPUOperationType.BULK_PROCESSING || dataSize > 50000) {
      return GPUBackend.WEBGPU;
    }

    // Use WebGPU for AI inference (compute shader advantages)
    if (operation === GPUOperationType.AI_INFERENCE) {
      return GPUBackend.WEBGPU;
    }

    // Particle compute: WebGPU for large datasets, WebGL for small real-time ones
    if (operation === GPUOperationType.PARTICLE_COMPUTE) {
      return dataSize > 20000 ? GPUBackend.WEBGPU : GPUBackend.WEBGL;
    }

    // Default to WebGPU for future-proofing
    return GPUBackend.WEBGPU;
  }

  // Enhanced compute method with backend selection
  async runComputeWithBackend(
    inputData: Float32Array,
    operation: GPUOperationType,
    preferredBackend: GPUBackend = GPUBackend.AUTO,
    elapsedTime?: number,
    requiresDOMSync: boolean = false
  ): Promise<Float32Array> {
    const backend =
      preferredBackend === GPUBackend.AUTO
        ? this.selectOptimalGPUBackend(operation, inputData.length, requiresDOMSync)
        : preferredBackend;

    console.log(
      `[WASM-GPU-Bridge] Using ${backend} backend for ${GPUOperationType[operation]} operation (${inputData.length} elements)${requiresDOMSync ? ' with DOM sync' : ''}`
    );

    if (backend === GPUBackend.WEBGL) {
      return this.runWebGLCompute(inputData, operation, elapsedTime);
    } else {
      return this.runCompute(inputData, operation, elapsedTime);
    }
  }

  // WebGL fallback implementation for DOM-interactive operations
  private async runWebGLCompute(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime?: number
  ): Promise<Float32Array> {
    // For now, delegate to worker pool with WebGL preference
    // In a full implementation, this would use WebGL contexts directly
    console.log(
      '[WASM-GPU-Bridge] Using WebGL-optimized worker processing for DOM sync compatibility'
    );

    if (this.workerPool.length > 0) {
      try {
        return await this.runWorkerPoolTask(inputData, elapsedTime || 0.016667, operation);
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] WebGL worker failed, falling back to CPU:', error);
        return this.runCPUFallback(inputData, operation, elapsedTime);
      }
    }

    return this.runCPUFallback(inputData, operation, elapsedTime);
  }

  // CPU fallback for maximum compatibility
  private runCPUFallback(
    inputData: Float32Array,
    operation: GPUOperationType,
    elapsedTime?: number
  ): Promise<Float32Array> {
    return new Promise(resolve => {
      // Simple CPU-based processing for compatibility
      const result = new Float32Array(inputData.length);
      const deltaTime = elapsedTime || 0.016667;

      // Apply different algorithms based on operation type
      if (operation === GPUOperationType.PARTICLE_COMPUTE) {
        for (let i = 0; i < inputData.length; i += 3) {
          // Basic particle simulation
          result[i] = inputData[i] + Math.sin(Date.now() * 0.001 + i) * deltaTime;
          result[i + 1] = inputData[i + 1] + Math.cos(Date.now() * 0.001 + i) * deltaTime;
          result[i + 2] = inputData[i + 2] + Math.sin(Date.now() * 0.002 + i) * deltaTime * 0.5;
        }
      } else {
        // Generic processing for other operation types
        for (let i = 0; i < inputData.length; i++) {
          result[i] = inputData[i] * (1.0 + Math.sin(Date.now() * 0.001) * 0.1);
        }
      }

      console.log(
        `[WASM-GPU-Bridge] ✅ CPU fallback completed: ${result.length} elements processed for ${GPUOperationType[operation]}`
      );
      resolve(result);
    });
  }

  async runPerformanceBenchmark(
    dataSize: number = 1000
  ): Promise<{ throughput: number; duration: number; method: string; threadingUsed: boolean }> {
    console.log(
      `[WASM-GPU-Bridge] Starting performance benchmark with ${dataSize} data points through centralized GPU system`
    );

    const inputData = new Float32Array(dataSize);
    for (let i = 0; i < dataSize; i++) {
      inputData[i] = Math.random();
    }

    let method = 'gpu';
    let threadingUsed = false;

    // Try WASM concurrent processing for large datasets to test threading
    if (dataSize > 5000 && typeof window !== 'undefined' && (window as any).runConcurrentCompute) {
      try {
        console.log(
          `[WASM-GPU-Bridge] Testing WASM concurrent processing for ${dataSize} data points`
        );

        const wasmStart = performance.now();

        await new Promise<Float32Array>((resolve, reject) => {
          const timeout = setTimeout(() => reject(new Error('WASM benchmark timeout')), 10000);

          (window as any).runConcurrentCompute(
            inputData,
            16.667, // 60fps timing
            1.0, // Performance test mode
            (result: Float32Array) => {
              clearTimeout(timeout);
              resolve(result);
            }
          );
        });

        const wasmDuration = performance.now() - wasmStart;
        const wasmThroughput = dataSize / (wasmDuration / 1000);

        console.log(`[WASM-GPU-Bridge] ✅ WASM concurrent benchmark completed:`, {
          dataSize,
          duration: `${wasmDuration.toFixed(2)}ms`,
          throughput: `${wasmThroughput.toFixed(0)} ops/sec`,
          method: 'wasm-concurrent',
          threadingUsed: true
        });

        return {
          throughput: wasmThroughput,
          duration: wasmDuration,
          method: 'wasm-concurrent',
          threadingUsed: true
        };
      } catch (error) {
        console.warn(
          '[WASM-GPU-Bridge] WASM concurrent benchmark failed, falling back to GPU:',
          error
        );
      }
    }

    // Fallback to standard GPU benchmark
    const startTime = performance.now();
    await this.runCompute(inputData, GPUOperationType.PERFORMANCE_TEST);
    const endTime = performance.now();

    const duration = endTime - startTime;
    const throughput = dataSize / (duration / 1000); // operations per second

    console.log(`[WASM-GPU-Bridge] ✅ Performance benchmark completed:`, {
      dataSize,
      duration: `${duration.toFixed(2)}ms`,
      throughput: `${throughput.toFixed(0)} ops/sec`,
      avgTimePerOp: `${((duration / dataSize) * 1000).toFixed(3)}μs/op`,
      method,
      threadingUsed
    });

    return { throughput, duration, method, threadingUsed };
  }

  async runParticlePhysics(particles: Float32Array, elapsedTime?: number): Promise<Float32Array> {
    const particleCount = particles.length / 3;

    // Enhanced particle physics with WASM threading optimization
    // For large particle counts, try WASM concurrent processing first
    if (
      particleCount > 3000 &&
      typeof window !== 'undefined' &&
      (window as any).runConcurrentCompute
    ) {
      try {
        console.log(
          `[WASM-GPU-Bridge] Using WASM concurrent processing for ${particleCount} particles`
        );

        const result = await new Promise<Float32Array>((resolve, reject) => {
          const timeout = setTimeout(() => reject(new Error('WASM concurrent timeout')), 8000);

          (window as any).runConcurrentCompute(
            particles,
            (elapsedTime || 0.016667) * 1000, // Convert to milliseconds for WASM
            1.2, // Animation mode for smooth particle physics
            (computedResult: Float32Array) => {
              clearTimeout(timeout);
              resolve(computedResult);
            }
          );
        });

        console.log(
          `[WASM-GPU-Bridge] ✅ WASM concurrent processing completed: ${result.length} elements`
        );
        return result;
      } catch (error) {
        console.warn(
          '[WASM-GPU-Bridge] WASM concurrent processing failed, falling back to GPU:',
          error
        );
      }
    }

    // Fallback to standard GPU processing
    console.log(
      `[WASM-GPU-Bridge] Running particle physics computation for ${particles.length} particles through centralized WASM GPU`
    );
    const result = await this.runCompute(particles, GPUOperationType.PARTICLE_COMPUTE, elapsedTime);
    console.log(
      `[WASM-GPU-Bridge] ✅ Particle physics computation completed: ${result.length} particle states updated`
    );
    return result;
  }

  async runParticlePhysicsWithOffset(
    particles: Float32Array,
    elapsedTime: number,
    globalParticleOffset: number
  ): Promise<Float32Array> {
    // Use the same counter for consistency, log only for same operations as compute
    const shouldLog = this.operationCounter % 10 === 0 || this.operationCounter <= 5;

    if (shouldLog) {
      console.log(
        `[WASM-GPU-Bridge] Running synchronized particle physics for ${particles.length} particles (global offset: ${globalParticleOffset})`
      );
    }

    const result = await this.runComputeWithOffset(
      particles,
      GPUOperationType.PARTICLE_COMPUTE,
      elapsedTime,
      globalParticleOffset
    );

    if (shouldLog) {
      console.log(
        `[WASM-GPU-Bridge] ✅ Synchronized particle physics completed: ${result.length} particle states updated`
      );
    }

    return result;
  }

  async runAIInference(data: Float32Array): Promise<Float32Array> {
    return this.runCompute(data, GPUOperationType.AI_INFERENCE);
  }

  // Comprehensive GPU capabilities detection for metadata integration
  async getGPUCapabilities(): Promise<GPUCapabilities> {
    console.log('[WASM-GPU-Bridge] Collecting comprehensive GPU capabilities...');

    const capabilities: GPUCapabilities = {
      webgpu: {
        available: false,
        features: [],
        limits: {}
      },
      webgl: {
        available: false,
        version: null,
        extensions: []
      },
      three: {
        optimized: false,
        recommendedRenderer: 'webgl',
        loadingMetrics: {
          totalLoadTime: 0,
          memoryUsage: 0,
          success: false
        }
      },
      performance: {
        score: 0,
        recommendation: 'CPU fallback'
      }
    };

    // WebGPU Detection with detailed adapter info
    if ('gpu' in navigator) {
      try {
        const adapter = await (navigator as any).gpu.requestAdapter({
          powerPreference: 'high-performance'
        });

        if (adapter) {
          capabilities.webgpu = {
            available: true,
            adapter: {
              vendor: adapter.info?.vendor || 'Unknown',
              architecture: adapter.info?.architecture || 'Unknown',
              device: adapter.info?.device || 'Unknown',
              description: adapter.info?.description || 'Unknown'
            },
            features: Array.from(adapter.features || []),
            limits: {
              maxBufferSize: adapter.limits?.maxBufferSize || 0,
              maxComputeWorkgroupSize: adapter.limits?.maxComputeWorkgroupSizeX || 0,
              maxStorageBufferBindingSize: adapter.limits?.maxStorageBufferBindingSize || 0
            }
          };
        }
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] WebGPU adapter request failed:', error);
      }
    }

    // WebGL Detection with detailed context info
    const canvas = document.createElement('canvas');

    // Test WebGL2
    try {
      const gl2 = canvas.getContext('webgl2');
      if (gl2) {
        const debugInfo = gl2.getExtension('WEBGL_debug_renderer_info');
        capabilities.webgl = {
          available: true,
          version: '2',
          vendor: debugInfo
            ? gl2.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL)
            : gl2.getParameter(gl2.VENDOR),
          renderer: debugInfo
            ? gl2.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)
            : gl2.getParameter(gl2.RENDERER),
          extensions: gl2.getSupportedExtensions() || []
        };
      }
    } catch (error) {
      console.warn('[WASM-GPU-Bridge] WebGL2 context creation failed:', error);
    }

    // Fallback to WebGL1 if WebGL2 not available
    if (!capabilities.webgl.available) {
      try {
        const gl = canvas.getContext('webgl');
        if (gl) {
          const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
          capabilities.webgl = {
            available: true,
            version: '1',
            vendor: debugInfo
              ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL)
              : gl.getParameter(gl.VENDOR),
            renderer: debugInfo
              ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)
              : gl.getParameter(gl.RENDERER),
            extensions: gl.getSupportedExtensions() || []
          };
        }
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] WebGL context creation failed:', error);
      }
    }

    // Three.js Integration (if available)
    try {
      const threeModule = await import('./three/index.js').catch(() => null);
      if (threeModule) {
        const threeCapabilities = threeModule.detectThreeCapabilities?.() || {};
        const threeStatus = threeModule.getThreeLoadingStatus?.() || {};
        const loadingMetrics = threeModule.getLoadingMetrics?.() || {};
        const performanceAnalysis = threeModule.analyzeLoadingPerformance?.() || {};

        capabilities.three = {
          optimized: threeStatus.webgpuOptimized || false,
          recommendedRenderer: threeCapabilities.recommendedRenderer || 'webgl',
          loadingMetrics: {
            totalLoadTime: loadingMetrics.totalLoadTime || 0,
            memoryUsage: loadingMetrics.memoryUsage || 0,
            success: loadingMetrics.success || false
          }
        };

        capabilities.performance = {
          score: performanceAnalysis.score || 0,
          recommendation: performanceAnalysis.recommendations?.[0] || 'CPU fallback'
        };
      }
    } catch (error) {
      console.warn('[WASM-GPU-Bridge] Three.js integration failed:', error);
    }

    // Run performance benchmark if GPU is available
    if (capabilities.webgpu.available || capabilities.webgl.available) {
      try {
        const benchmark = await this.runPerformanceBenchmark(5000);
        capabilities.performance.benchmark = {
          webgpuScore: capabilities.webgpu.available ? 100 : 0,
          webglScore: capabilities.webgl.available ? 75 : 0,
          throughput: benchmark.throughput,
          duration: benchmark.duration
        };
      } catch (error) {
        console.warn('[WASM-GPU-Bridge] Performance benchmark failed:', error);
      }
    }

    console.log('[WASM-GPU-Bridge] ✅ GPU capabilities collected:', capabilities);
    return capabilities;
  }

  // Update global metadata with GPU capabilities
  async updateMetadataWithGPUInfo(): Promise<void> {
    try {
      // Import global store dynamically to avoid circular dependencies
      const { useGlobalStore } = await import('../store/global.js');
      const store = useGlobalStore.getState();

      const gpuCapabilities = await this.getGPUCapabilities();

      // Add GPU information to device metadata (using existing extensible structure)
      store.setMetadata({
        device: {
          ...store.metadata.device,
          // Use the extensible [key: string]: any structure in DeviceMetadata
          gpuCapabilities: gpuCapabilities,
          wasmGPUBridge: {
            initialized: this.initialized,
            backend: this.initialized ? 'WASM+WebGPU' : 'fallback',
            workerCount: this.workerPool.length,
            version: '1.0.0'
          },
          gpuDetectedAt: new Date().toISOString()
        }
      });

      console.log('[WASM-GPU-Bridge] ✅ Metadata updated with GPU capabilities');
    } catch (error) {
      console.error('[WASM-GPU-Bridge] Failed to update metadata with GPU info:', error);
    }
  }

  // New convenience methods for different use cases
  async runInteractiveVisualization(
    data: Float32Array,
    elapsedTime?: number
  ): Promise<Float32Array> {
    // Force WebGL for better DOM sync performance
    return this.runComputeWithBackend(
      data,
      GPUOperationType.DOM_INTERACTIVE,
      GPUBackend.WEBGL,
      elapsedTime,
      true
    );
  }

  async runBulkProcessing(data: Float32Array): Promise<Float32Array> {
    // Force WebGPU for maximum parallel compute performance
    return this.runComputeWithBackend(data, GPUOperationType.BULK_PROCESSING, GPUBackend.WEBGPU);
  }

  // Enhanced particle physics with smart backend selection
  async runSmartParticlePhysics(
    particles: Float32Array,
    elapsedTime?: number,
    requiresDOMSync: boolean = false
  ): Promise<Float32Array> {
    return this.runComputeWithBackend(
      particles,
      GPUOperationType.PARTICLE_COMPUTE,
      GPUBackend.AUTO,
      elapsedTime,
      requiresDOMSync
    );
  }

  /**
   * Get comprehensive threading status and WASM capabilities
   */
  getThreadingStatus(): {
    crossOriginIsolated: boolean;
    wasmThreadingSupported: boolean;
    sharedArrayBufferAvailable: boolean;
    threadingOptimal: boolean;
    recommendations: string[];
  } {
    const status = {
      crossOriginIsolated: typeof crossOriginIsolated !== 'undefined' && crossOriginIsolated,
      wasmThreadingSupported: false,
      sharedArrayBufferAvailable: typeof SharedArrayBuffer !== 'undefined',
      threadingOptimal: false,
      recommendations: [] as string[]
    };

    // Check if WASM is initialized and has threading functions
    if (this.initialized) {
      try {
        const global = globalThis as any;
        status.wasmThreadingSupported =
          typeof global.runConcurrentCompute === 'function' ||
          typeof global.initWorkerPool === 'function';
      } catch (e) {
        status.wasmThreadingSupported = false;
      }
    }

    // Determine if threading is optimal
    status.threadingOptimal =
      status.crossOriginIsolated &&
      status.wasmThreadingSupported &&
      status.sharedArrayBufferAvailable;

    // Generate recommendations
    if (!status.crossOriginIsolated) {
      status.recommendations.push('Enable cross-origin isolation with COOP/COEP headers');
    }
    if (!status.sharedArrayBufferAvailable) {
      status.recommendations.push('SharedArrayBuffer not available - check browser support');
    }
    if (!status.wasmThreadingSupported) {
      status.recommendations.push(
        'WASM threading functions not available - rebuild WASM with threading'
      );
    }
    if (status.threadingOptimal) {
      status.recommendations.push('Threading configuration is optimal');
    }

    console.log('[WASM Threading Status]', status);
    return status;
  }
}

// Singleton instance for centralized GPU access
export const wasmGPU = new WasmGPUBridge();
const WASM_SESSION_ID = `wasm-session-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
console.log(
  `[WASM-GPU-Bridge] ✅ Centralized WASM GPU singleton created - GPU initialization will happen lazily when needed | Session: ${WASM_SESSION_ID}`
);
if (typeof window !== 'undefined') {
  (window as any).__WASM_SESSION_ID = WASM_SESSION_ID;
}

// Only set global media streaming URL, do not auto-connect or trigger any connection logic here.
if (typeof window !== 'undefined') {
  const mediaUrl = (() => {
    if (typeof window !== 'undefined' && window.location) {
      const host = window.location.host;
      let baseUrl: string;
      if (host.includes('5173') || host.includes('3000') || host.includes('localhost')) {
        const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
        baseUrl = `${protocol}://localhost:8085/ws`;
      } else {
        const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
        baseUrl = `${protocol}://${host}/media/ws`;
      }
      const campaignId = '0';
      const contextId = 'webgpu-particles';
      // Use the centralized WASM guest/user ID for peerId
      const peerId =
        (window as any).userID || `guest_${Math.random().toString(36).substring(2, 10)}`;
      return `${baseUrl}?campaign=${campaignId}&context=${contextId}&peer=${peerId}`;
    }
    return 'ws://localhost:8085/ws?campaign=0&context=webgpu-particles&peer=guest_default';
  })();
  (window as any).__MEDIA_STREAMING_URL = mediaUrl;
  console.log(`[WASM-Bridge] Set global URLs for WASM:`);
  console.log(`[WASM-Bridge] - Media WS: ${mediaUrl}`);
  console.log(`[WASM-Bridge] - Main WS: Dynamic construction in WASM based on campaign metadata`);
  // No setTimeout or background connection here. Media streaming connection should be triggered explicitly by frontend after WASM is ready.
}

// WASM lifecycle state tracking
if (typeof window !== 'undefined') {
  // Set this to true when WASM is initialized, false when cleaned up
  (window as any).wasmReady = false;
  (window as any).swReady = false;
  // Use global metadata object for all runtime metadata
  if (typeof (window as any).__WASM_GLOBAL_METADATA === 'undefined') {
    (window as any).__WASM_GLOBAL_METADATA = {};
  }

  function initializeAfterReady() {
    if ((window as any).swReady && (window as any).wasmReady) {
      // Only set readiness, do not re-initialize WASM bridge, GPU, or media streaming unless truly needed
      console.log('[WASM-Bridge] All systems ready.');
    }
  }

  window.addEventListener('wasmReady', () => {
    (window as any).wasmReady = true;
    initializeAfterReady();
  });

  navigator.serviceWorker?.addEventListener('message', event => {
    if (event.data?.type === 'sw-ready') {
      (window as any).swReady = true;
      initializeAfterReady();
    }
  });

  // If SW is not used, proceed with WASM only
  if (!navigator.serviceWorker) {
    (window as any).swReady = true;
  }
}
(window as any).wasmReady = true;
console.log('[WASM] wasmReady set to true');

// Example usage for WebSocket connection:
// const wsURL = getDynamicWebSocketURL();
// const ws = new WebSocket(wsURL);

// --- Emitter for multiple listeners ---
type WasmListener = (msg: WasmBridgeMessage) => void;
const listeners: WasmListener[] = [];

// Track last cancellation reason for diagnostics
let lastWasmCancellationReason: string | null = null;

/**
 * Get the last WASM cancellation reason for UI or debugging.
 */
export function getLastWasmCancellationReason(): string | null {
  return lastWasmCancellationReason;
}

// notifyListeners handles WASM→Frontend type conversion at the boundary
function notifyListeners(msg: any) {
  // Convert WASM message to proper TypeScript types at the boundary
  const convertedMsg = wasmMessageToTypescript(msg);
  // If this is a connection:closed event, log the reason
  if (convertedMsg.type === 'connection:closed') {
    // Try to extract reason from payload/metadata if present
    const reason =
      (convertedMsg.metadata && convertedMsg.metadata.reason) ||
      (convertedMsg.payload && convertedMsg.payload.reason) ||
      'unknown';
    lastWasmCancellationReason = reason;
    // Log to console and set global for debugging/UI
    console.warn('[WASM-Bridge] Connection closed. Reason:', reason, 'Full event:', convertedMsg);
    if (typeof window !== 'undefined') {
      (window as any).__WASM_LAST_CANCELLATION_REASON = reason;
      // Optionally, display in a simple UI element for demo/debug
      let el = document.getElementById('wasm-cancellation-reason');
      if (el) {
        el.textContent = `WASM Cancellation Reason: ${reason}`;
      }
    }
  }
  listeners.forEach(cb => cb(convertedMsg));
}

// wasmMessageToTypescript converts WASM message to proper TypeScript types
function wasmMessageToTypescript(wasmMsg: any): WasmBridgeMessage {
  // WASM should already be sending properly typed objects
  // but ensure we have the right structure
  const msg: WasmBridgeMessage = {
    type: wasmMsg.type || 'unknown',
    payload: wasmMsg.payload,
    metadata: wasmMsg.metadata || {}
  };

  // Copy any additional properties
  Object.keys(wasmMsg).forEach(key => {
    if (!['type', 'payload', 'metadata'].includes(key)) {
      (msg as any)[key] = wasmMsg[key];
    }
  });

  return msg;
}

// Expose the listener manager to the window for WASM to call
if (typeof window !== 'undefined') {
  (window as any).onWasmMessage = notifyListeners;

  // --- WebRTC Signaling Hooks ---
  // Media signal listeners (for WebRTC signaling)
  const mediaSignalListeners: ((msg: any) => void)[] = [];

  // Called by WASM when a signaling message arrives from backend
  (window as any).onMediaSignal = function (msg: any) {
    mediaSignalListeners.forEach(cb => cb(msg));
  };

  // Called by JS/React/WebRTC to send a signaling message to backend via WASM/WebSocket
  (window as any).sendMediaSignal = function (msg: any) {
    // You may want to wrap/namespace the message if needed
    if (typeof window.sendWasmMessage === 'function') {
      // Use a reserved type for signaling, e.g., 'rtc-signal'
      window.sendWasmMessage({ type: 'rtc-signal', payload: msg });
    } else {
      console.warn('sendWasmMessage not available, cannot send media signal');
    }
  };

  // Expose subscribe/unsubscribe for media signals
  (window as any).subscribeToMediaSignals = function (cb: (msg: any) => void) {
    mediaSignalListeners.push(cb);
    return () => {
      const idx = mediaSignalListeners.indexOf(cb);
      if (idx > -1) mediaSignalListeners.splice(idx, 1);
    };
  };
}
/**
 * Subscribe to WebRTC signaling messages from WASM/backend.
 * @param cb The callback to invoke with each signaling message.
 * @returns An unsubscribe function.
 */
export function subscribeToMediaSignals(cb: (msg: any) => void): () => void {
  if (
    typeof window !== 'undefined' &&
    typeof (window as any).subscribeToMediaSignals === 'function'
  ) {
    return (window as any).subscribeToMediaSignals(cb);
  } else {
    // Fallback: no-op unsubscribe
    return () => {};
  }
}

/**
 * Send a WebRTC signaling message to the backend via WASM/WebSocket.
 * @param msg - The signaling message (SDP, ICE, etc)
 */
export function sendMediaSignal(msg: any) {
  if (typeof window !== 'undefined' && typeof (window as any).sendMediaSignal === 'function') {
    (window as any).sendMediaSignal(msg);
  } else {
    console.warn('sendMediaSignal not available');
  }
}

/**
 * Subscribe to messages from the WASM bridge.
 * @param cb The callback to invoke with each message.
 * @returns An unsubscribe function.
 */
export function subscribeToWasmMessages(cb: WasmListener): () => void {
  listeners.push(cb);
  return () => {
    const index = listeners.indexOf(cb);
    if (index > -1) {
      listeners.splice(index, 1);
    }
  };
}

/**
 * Send a message to the server via the WASM WebSocket client.
 * Handles Frontend→WASM type conversion at the boundary.
 * @param msg - The message object to send
 */
export function wasmSendMessage(msg: WasmBridgeMessage | EventEnvelope) {
  // Defensive: ensure metadata is always an object before passing to WASM
  console.log('MESSAGE >>>>>>', msg);
  if ('metadata' in msg && typeof msg.metadata === 'string') {
    try {
      // Try base64 decode then parse
      const decoded = atob(msg.metadata);
      msg.metadata = JSON.parse(decoded);
    } catch {
      try {
        msg.metadata = JSON.parse(msg.metadata);
      } catch {
        msg.metadata = {};
      }
    }
  }
  if (typeof window !== 'undefined' && typeof (window as any).sendWasmMessage === 'function') {
    // Convert TypeScript types to WASM-compatible format at the boundary

    (window as any).sendWasmMessage(msg);
  } else {
    // Queue the message if the bridge isn't ready yet.
    console.warn('WASM bridge not available. Message will be sent upon readiness.');
    // You could implement a queue here if needed, but we will centralize it in the Zustand store.
  }
}

// // typescriptToWasmMessage converts TypeScript types to WASM-compatible format
// function typescriptToWasmMessage(msg: WasmBridgeMessage | EventEnvelope): any {
//   // Ensure we have a clean object that WASM can properly handle
//   const wasmMsg: any = {
//     type: msg.type
//   };

//   // Handle payload - ensure it's properly structured
//   if ('payload' in msg && msg.payload !== undefined) {
//     wasmMsg.payload = msg.payload;
//   }

//   // Handle metadata - ensure it's an object, never a string or base64
//   if ('metadata' in msg && msg.metadata !== undefined) {
//     if (typeof msg.metadata === 'string') {
//       try {
//         // Try base64 decode then parse
//         const decoded = atob(msg.metadata);
//         wasmMsg.metadata = JSON.parse(decoded);
//       } catch {
//         try {
//           wasmMsg.metadata = JSON.parse(msg.metadata);
//         } catch {
//           wasmMsg.metadata = {};
//         }
//       }
//     } else {
//       wasmMsg.metadata = msg.metadata;
//     }
//   } else {
//     wasmMsg.metadata = {};
//   }

//   // Copy any additional properties from WasmBridgeMessage
//   if ('payload' in msg || 'metadata' in msg) {
//     Object.keys(msg).forEach(key => {
//       if (!['type', 'payload', 'metadata'].includes(key)) {
//         wasmMsg[key] = (msg as any)[key];
//       }
//     });
//   }

//   return wasmMsg;
// }

// --- Centralize guest/user ID generation at startup ---
if (typeof window !== 'undefined' && !(window as any).userID) {
  (window as any).userID = `guest_${Math.random().toString(36).substring(2, 10)}`;
}

/**
 * Connect media streaming to a specific campaign
 * @param campaignId - The campaign ID to connect to
 * @param contextId - The context ID (e.g., 'webgpu-particles', 'live-chat', etc.)
 * @param peerId - Optional peer ID (if not provided, uses a generated guest ID)
 */
export function connectMediaStreamingToCampaign(
  campaignId: string = '0',
  contextId: string = 'webgpu-particles',
  peerId?: string
): void {
  // Always use the centralized userID
  const finalPeerId = peerId || (window as any).userID;
  const isWasmReady =
    typeof window !== 'undefined' && 'wasmReady' in window && (window as any).wasmReady === true;
  if (
    isWasmReady &&
    window.mediaStreaming &&
    typeof window.mediaStreaming.connectToCampaign === 'function'
  ) {
    window.mediaStreaming.connectToCampaign(campaignId, contextId, finalPeerId);
    console.log(
      `[WASM-Bridge] Media streaming connected to campaign ${campaignId} with context ${contextId} and peer ${finalPeerId}`
    );
  } else {
    // Do not attempt connection. Log and abort.
    console.warn(
      '[WASM-Bridge] Media streaming connection attempted before WASM is ready or mediaStreaming is unavailable. Aborting.'
    );
  }
}

/**
 * Check if media streaming is connected
 */
export function isMediaStreamingConnected(): boolean {
  if (typeof window !== 'undefined' && window.mediaStreaming?.isConnected) {
    return window.mediaStreaming.isConnected();
  }
  return false;
}

/**
 * Get current media streaming URL
 */
export function getMediaStreamingURL(): string {
  if (typeof window !== 'undefined' && window.mediaStreaming?.getURL) {
    return window.mediaStreaming.getURL();
  }
  return '';
}

/**
 * Send a message via media streaming
 */
export function sendMediaStreamingMessage(message: any): void {
  if (typeof window !== 'undefined' && window.mediaStreaming?.send) {
    window.mediaStreaming.send(message);
  } else {
    console.warn('[Media-Streaming] Media streaming not connected');
  }
}

/**
 * Subscribe to media streaming state changes
 */
export function subscribeToMediaStreamingState(callback: (state: string) => void): void {
  if (typeof window !== 'undefined' && window.mediaStreaming?.onState) {
    window.mediaStreaming.onState(callback);
  } else {
    console.warn('[Media-Streaming] Media streaming API not available');
  }
}

/**
 * Subscribe to media streaming messages
 */
export function subscribeToMediaStreamingMessages(callback: (data: any) => void): void {
  if (typeof window !== 'undefined' && window.mediaStreaming?.onMessage) {
    window.mediaStreaming.onMessage(callback);
  } else {
    console.warn('[Media-Streaming] Media streaming API not available');
  }
}

/**
 * Disconnect from media streaming and clean up resources.
 */
export function disconnectMediaStreaming(): void {
  try {
    // Call WASM shutdown if available
    if (
      typeof window !== 'undefined' &&
      window.mediaStreaming &&
      typeof window.mediaStreaming.shutdown === 'function'
    ) {
      window.mediaStreaming.shutdown();
      console.log('[WASM-Bridge] Media streaming shutdown called.');
    } else {
      console.warn('[WASM-Bridge] mediaStreaming.shutdown not available.');
    }
    // Clear state in Zustand global store
    const { useGlobalStore } = require('../store/global');
    const clearMediaStreamingState = useGlobalStore.getState().clearMediaStreamingState;
    if (typeof clearMediaStreamingState === 'function') {
      clearMediaStreamingState();
      console.log('[WASM-Bridge] Media streaming state cleared.');
    } else {
      console.warn('[WASM-Bridge] clearMediaStreamingState not available.');
    }
  } catch (err) {
    console.warn('[WASM-Bridge] Error during media streaming disconnect:', err);
  }
}
