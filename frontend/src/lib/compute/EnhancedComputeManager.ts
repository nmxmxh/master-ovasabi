// Enhanced Compute Manager for OVASABI Architecture
// Coordinates between main thread, Web Workers, WASM concurrency, and WebGPU

import { generateTaskId } from '../../utils/cryptoIds';

export interface ComputeTask {
  id: string;
  data: Float32Array;
  params: {
    deltaTime?: number;
    animationMode?: number;
    priority?: 'high' | 'normal' | 'low';
  };
  callback?: (result: Float32Array, metadata: any) => void;
}

export interface ComputeCapabilities {
  webgpu: boolean;
  wasm: boolean;
  webWorkers: boolean;
  concurrentWorkers: number;
  wasmWorkers: number;
}

export interface PerformanceMetrics {
  avgProcessingTime: number;
  particlesPerSecond: number;
  memoryUsage: number;
  activeWorkers: number;
  queueDepth: number;
  method: string;
}

// Optimized circular buffer for performance metrics
class CircularBuffer<T> {
  private buffer: T[] = [];
  private head: number = 0;
  private size: number = 0;
  private capacity: number;

  constructor(capacity: number) {
    this.capacity = capacity;
  }

  push(item: T): void {
    if (this.size < this.capacity) {
      this.buffer.push(item);
      this.size++;
    } else {
      this.buffer[this.head] = item;
      this.head = (this.head + 1) % this.capacity;
    }
  }

  toArray(): T[] {
    if (this.size < this.capacity) {
      return this.buffer.slice();
    }
    return [...this.buffer.slice(this.head), ...this.buffer.slice(0, this.head)];
  }

  get length(): number {
    return this.size;
  }

  getLatest(): T | undefined {
    if (this.size === 0) return undefined;
    const latestIndex =
      this.size < this.capacity ? this.size - 1 : (this.head - 1 + this.capacity) % this.capacity;
    return this.buffer[latestIndex];
  }
}

class EnhancedComputeManager {
  private workers: Worker[] = [];
  private workerCount: number;
  private taskQueue: ComputeTask[] = [];
  private pendingTasks: Map<string, ComputeTask> = new Map();
  private performance: PerformanceMetrics[] = [];
  private capabilities: ComputeCapabilities;
  private wasmModule: any = null;
  private adaptiveQuality: boolean = true;
  private targetFPS: number = 60;
  private lastOptimization: number = 0;
  // Note: taskIdCounter removed as we now use secure crypto-based ID generation
  private performanceBuffer: CircularBuffer<PerformanceMetrics>;
  private workerCapabilities: Map<number, any> = new Map();

  constructor(workerCount?: number) {
    this.workerCount = workerCount || Math.min(navigator.hardwareConcurrency || 2, 2);
    this.capabilities = {
      webgpu: 'gpu' in navigator,
      wasm: false,
      webWorkers: typeof Worker !== 'undefined',
      concurrentWorkers: this.workerCount,
      wasmWorkers: 0
    };

    // Use circular buffer for performance metrics (memory efficient)
    this.performanceBuffer = new CircularBuffer<PerformanceMetrics>(100);

    this.initialize();
  }

  private async initialize(): Promise<void> {
    console.log('[COMPUTE-MANAGER] Initializing enhanced compute system...');

    // Initialize WASM module
    await this.initializeWasm();

    // Initialize Web Workers
    if (this.capabilities.webWorkers) {
      await this.initializeWorkers();
    }

    // Start performance monitoring
    this.startPerformanceMonitoring();

    console.log('[COMPUTE-MANAGER] Initialization complete:', this.capabilities);
  }

  private async initializeWasm(): Promise<void> {
    try {
      // Wait for WASM to be ready via event or polling fallback
      const wasmReady = await this.waitForWasmReady();

      if (wasmReady) {
        this.wasmModule = window;
        this.capabilities.wasm = true;

        // Get WASM worker pool status
        const status = (window as any).getWorkerPoolStatus?.();
        if (status && status.workers) {
          this.capabilities.wasmWorkers = status.workers;
        }

        // Check for enhanced worker pool
        const enhancedStatus = (window as any).getEnhancedWorkerPoolStatus?.();
        if (enhancedStatus && enhancedStatus.initialized) {
          this.capabilities.wasmWorkers = enhancedStatus.workers || this.capabilities.wasmWorkers;
        }

        // Initialize WebGPU in workers for better performance
        console.log('[COMPUTE-MANAGER] Initializing WebGPU in worker context...');
        try {
          const webgpuManager = await import('../gpu/WebGPUManager');
          const manager = webgpuManager.webGPUManager;
          const webgpuReady = await manager.initialize();
          if (webgpuReady) {
            this.capabilities.webgpu = true;
            console.log('[COMPUTE-MANAGER] WebGPU initialized successfully in worker');
          }
        } catch (error) {
          console.warn('[COMPUTE-MANAGER] WebGPU initialization failed in worker:', error);
        }

        console.log('[COMPUTE-MANAGER] WASM module connected with enhanced features');
      } else {
        console.warn('[COMPUTE-MANAGER] WASM not available after waiting');
      }
    } catch (error) {
      console.warn('[COMPUTE-MANAGER] WASM initialization failed:', error);
    }
  }

  private async waitForWasmReady(): Promise<boolean> {
    return new Promise(resolve => {
      let resolved = false;

      // First, check if WASM is already ready
      const hasWasmFunctions =
        typeof window !== 'undefined' &&
        typeof (window as any).runConcurrentCompute === 'function' &&
        typeof (window as any).runGPUCompute === 'function' &&
        typeof (window as any).getGPUMetricsBuffer === 'function';

      if (hasWasmFunctions) {
        console.log('[COMPUTE-MANAGER] WASM functions already available');
        resolve(true);
        return;
      }

      // Listen for wasmReady event (preferred method)
      const onWasmReady = () => {
        if (resolved) return;
        resolved = true;
        window.removeEventListener('wasmReady', onWasmReady);
        console.log('[COMPUTE-MANAGER] WASM ready via event');
        resolve(true);
      };

      window.addEventListener('wasmReady', onWasmReady);

      // Fallback: polling if event doesn't come within reasonable time
      const maxAttempts = 30; // 3 seconds total
      let attempts = 0;

      const checkWasm = () => {
        if (resolved) return;

        attempts++;

        // Check if WASM functions are available
        const hasWasmFunctions =
          typeof window !== 'undefined' &&
          typeof (window as any).runConcurrentCompute === 'function' &&
          typeof (window as any).runGPUCompute === 'function' &&
          typeof (window as any).getGPUMetricsBuffer === 'function';

        if (hasWasmFunctions) {
          if (!resolved) {
            resolved = true;
            window.removeEventListener('wasmReady', onWasmReady);
            console.log(
              '[COMPUTE-MANAGER] WASM functions detected via polling after',
              attempts,
              'attempts'
            );
            resolve(true);
          }
          return;
        }

        if (attempts >= maxAttempts) {
          if (!resolved) {
            resolved = true;
            window.removeEventListener('wasmReady', onWasmReady);
            console.warn(
              '[COMPUTE-MANAGER] WASM functions not available after',
              maxAttempts,
              'attempts'
            );
            resolve(false);
          }
          return;
        }

        // Check again in 100ms
        setTimeout(checkWasm, 100);
      };

      // Start polling after a short delay to give the event a chance
      setTimeout(checkWasm, 100);
    });
  }

  private async initializeWorkers(): Promise<void> {
    for (let i = 0; i < this.workerCount; i++) {
      try {
        const worker = new Worker('/workers/compute-worker.js');

        worker.onmessage = event => this.handleWorkerMessage(event, i);
        worker.onerror = error => this.handleWorkerError(error, i);

        this.workers.push(worker);

        // Request worker status
        worker.postMessage({ type: 'status' });
      } catch (error) {
        console.error(`[COMPUTE-MANAGER] Failed to create worker ${i}:`, error);
      }
    }

    console.log(`[COMPUTE-MANAGER] Initialized ${this.workers.length} workers`);
  }

  private handleWorkerMessage(event: MessageEvent, workerIndex: number): void {
    const { type, ...data } = event.data;

    switch (type) {
      case 'worker-ready':
        console.log(`[COMPUTE-MANAGER] Worker ${workerIndex} ready:`, data.capabilities);
        break;

      case 'task-result':
        this.handleTaskResult(data);
        break;

      case 'task-error':
        this.handleTaskError(data);
        break;

      case 'benchmark-result':
        console.log(`[COMPUTE-MANAGER] Benchmark from worker ${workerIndex}:`, data);
        break;

      case 'worker-status':
        console.log(`[COMPUTE-MANAGER] Worker ${workerIndex} status:`, data);
        break;

      case 'status':
        // Handle status messages from workers
        console.log(`[COMPUTE-MANAGER] Worker ${workerIndex} status update:`, data);
        break;

      case 'worker-capabilities':
        // Handle worker capabilities messages
        console.log(`[COMPUTE-MANAGER] Worker ${workerIndex} capabilities:`, data);
        // Update worker capabilities in our tracking
        if (data.capabilities) {
          this.workerCapabilities.set(workerIndex, data.capabilities);
          console.log(
            `[COMPUTE-MANAGER] Updated worker ${workerIndex} capabilities:`,
            data.capabilities
          );
        }
        break;

      case 'wasm-call-request':
        // Handle WASM function call request from worker
        this.handleWASMCallRequest(workerIndex, event.data);
        break;

      default:
        console.warn('[COMPUTE-MANAGER] Unknown worker message:', type);
    }
  }

  private handleWorkerError(error: ErrorEvent, workerIndex: number): void {
    console.error(`[COMPUTE-MANAGER] Worker ${workerIndex} error:`, error);
  }

  private async handleWASMCallRequest(workerIndex: number, data: any): Promise<void> {
    const { callId, functionName, args } = data;
    const worker = this.workers[workerIndex];

    if (!worker) {
      console.error(`[COMPUTE-MANAGER] Worker ${workerIndex} not found for WASM call`);
      return;
    }

    try {
      // Call the WASM function on the main thread
      let result;

      switch (functionName) {
        case 'runConcurrentCompute':
          if (typeof window.runConcurrentCompute === 'function') {
            result = await new Promise(resolve => {
              window.runConcurrentCompute!(
                args[0],
                args[1],
                args[2],
                (resultData: any, metadata: any) => {
                  resolve({ result: resultData, metadata });
                }
              );
            });
          } else {
            throw new Error('runConcurrentCompute not available');
          }
          break;

        case 'runGPUCompute':
          if (typeof window.runGPUCompute === 'function') {
            result = await new Promise((resolve, reject) => {
              const success = window.runGPUCompute!(args[0], args[1], (resultData: any) => {
                resolve(resultData);
              });
              if (!success) {
                reject(new Error('runGPUCompute failed to start'));
              }
            });
          } else {
            throw new Error('runGPUCompute not available');
          }
          break;

        case 'submitComputeTask':
          if (typeof window.submitComputeTask === 'function') {
            result = window.submitComputeTask(args[0], args[1], args[2], args[3]);
          } else {
            throw new Error('submitComputeTask not available');
          }
          break;

        default:
          throw new Error(`Unknown WASM function: ${functionName}`);
      }

      // Send response back to worker
      worker.postMessage({
        type: 'wasm-call-response',
        callId,
        result
      });
    } catch (error) {
      // Send error response back to worker
      worker.postMessage({
        type: 'wasm-call-response',
        callId,
        error: error instanceof Error ? error.message : 'Unknown error'
      });
    }
  }

  private handleTaskResult(result: any): void {
    const task = this.pendingTasks.get(result.id);
    if (task && task.callback) {
      task.callback(result.data, result.metadata);
      this.pendingTasks.delete(result.id);

      // Update performance metrics
      this.updatePerformanceMetrics(result.metadata);
    }
  }

  private handleTaskError(error: any): void {
    const task = this.pendingTasks.get(error.id);
    if (task) {
      console.error('[COMPUTE-MANAGER] Task failed:', error);
      this.pendingTasks.delete(error.id);
    }
  }

  private updatePerformanceMetrics(metadata: any): void {
    const metrics: PerformanceMetrics = {
      avgProcessingTime: metadata.processingTime,
      particlesPerSecond: (metadata.particleCount * 1000) / metadata.processingTime,
      memoryUsage: 0, // Could be tracked
      activeWorkers: metadata.workerCount || 1,
      queueDepth: this.taskQueue.length,
      method: metadata.method
    };

    this.performanceBuffer.push(metrics);
  }

  private generateTaskId(): string {
    // Use secure crypto-based ID generation for better security
    return generateTaskId();
  }

  private startPerformanceMonitoring(): void {
    setInterval(() => {
      const now = Date.now();
      if (
        this.adaptiveQuality &&
        this.performanceBuffer.length > 10 &&
        now - this.lastOptimization > 1000
      ) {
        this.adjustQualityBasedOnPerformance();
        this.lastOptimization = now;
      }
    }, 1000);
  }

  private adjustQualityBasedOnPerformance(): void {
    const recentMetrics = this.performanceBuffer.toArray().slice(-10);
    if (recentMetrics.length === 0) return;

    const avgFPS =
      1000 /
      (recentMetrics.reduce((sum, m) => sum + m.avgProcessingTime, 0) / recentMetrics.length);

    if (avgFPS < this.targetFPS * 0.8) {
      console.log('[COMPUTE-MANAGER] Performance below target, suggesting quality reduction');
      // Could emit events for quality adjustment
    } else if (avgFPS > this.targetFPS * 1.2) {
      console.log('[COMPUTE-MANAGER] Performance above target, suggesting quality increase');
    }
  }

  public async processParticles(
    data: Float32Array,
    deltaTime: number = 0.016667,
    animationMode: number = 1.0,
    priority: 'high' | 'normal' | 'low' = 'normal'
  ): Promise<Float32Array> {
    return new Promise(resolve => {
      const task: ComputeTask = {
        id: this.generateTaskId(),
        data,
        params: { deltaTime, animationMode, priority },
        callback: result => {
          resolve(result);
        }
      };

      this.submitTask(task);
    });
  }

  private submitTask(task: ComputeTask): void {
    this.pendingTasks.set(task.id, task);

    // Choose optimal processing method
    const method = this.selectOptimalMethod(task.data.length, task.params.priority);

    switch (method) {
      case 'wasm-concurrent':
        this.processWithWasmConcurrent(task);
        break;

      case 'webgpu':
        this.processWithWebGPU(task);
        break;

      case 'worker':
        this.processWithWorker(task);
        break;

      case 'main-thread':
      default:
        this.processWithMainThread(task);
        break;
    }
  }

  private selectOptimalMethod(dataSize: number, priority?: string): string {
    // High priority tasks use fastest available method
    if (priority === 'high') {
      if (this.capabilities.webgpu && dataSize > 10000) {
        return 'webgpu';
      }
      if (this.capabilities.wasm && dataSize > 5000) {
        return 'wasm-concurrent';
      }
    }

    // Large datasets benefit from GPU processing
    if (dataSize > 50000 && this.capabilities.webgpu) {
      return 'webgpu';
    }

    // Medium datasets benefit from WASM concurrency
    if (dataSize > 10000 && this.capabilities.wasm) {
      return 'wasm-concurrent';
    }

    // Small to medium datasets use workers to avoid blocking main thread
    if (dataSize > 1000 && this.workers.length > 0) {
      return 'worker';
    }

    // Small datasets can be processed on main thread
    return 'main-thread';
  }

  private processWithWasmConcurrent(task: ComputeTask): void {
    if (!this.wasmModule?.runConcurrentCompute) {
      this.processWithWorker(task);
      return;
    }

    const startTime = performance.now();

    // Convert Float32Array to JS-compatible format
    const jsData = new Float32Array(task.data);

    const success = this.wasmModule.runConcurrentCompute(
      jsData,
      task.params.deltaTime || 0.016667,
      task.params.animationMode || 1.0,
      (result: Float32Array, metadata: any) => {
        const processingTime = performance.now() - startTime;
        this.handleTaskResult({
          id: task.id,
          data: result,
          metadata: {
            ...metadata,
            processingTime,
            method: 'wasm-concurrent'
          }
        });
      }
    );

    if (!success) {
      console.warn('[COMPUTE-MANAGER] WASM concurrent processing failed, falling back to worker');
      this.processWithWorker(task);
    }
  }

  private processWithWebGPU(task: ComputeTask): void {
    if (!this.wasmModule?.runGPUCompute) {
      this.processWithWorker(task);
      return;
    }

    const startTime = performance.now();

    // Convert Float32Array to JS-compatible format
    const jsData = new Float32Array(task.data);

    const success = this.wasmModule.runGPUCompute(
      jsData,
      task.params.deltaTime || 0.016667,
      (result: Float32Array) => {
        const processingTime = performance.now() - startTime;
        this.handleTaskResult({
          id: task.id,
          data: result,
          metadata: {
            processingTime,
            method: 'webgpu',
            particleCount: task.data.length / 3
          }
        });
      }
    );

    if (!success) {
      console.warn('[COMPUTE-MANAGER] WebGPU processing failed, falling back to worker');
      this.processWithWorker(task);
    }
  }

  private processWithWorker(task: ComputeTask): void {
    if (this.workers.length === 0) {
      this.processWithMainThread(task);
      return;
    }

    // Select worker with round-robin
    const workerIndex = this.pendingTasks.size % this.workers.length;
    const worker = this.workers[workerIndex];

    worker.postMessage({
      type: 'compute-task',
      payload: {
        id: task.id,
        data: task.data,
        params: task.params
      }
    });
  }

  private processWithMainThread(task: ComputeTask): void {
    // Fallback CPU processing on main thread
    setTimeout(() => {
      const startTime = performance.now();
      const result = this.processParticlesCPU(task.data, task.params);
      const processingTime = performance.now() - startTime;

      this.handleTaskResult({
        id: task.id,
        data: result,
        metadata: {
          processingTime,
          method: 'main-thread',
          particleCount: task.data.length / 3
        }
      });
    }, 0);
  }

  private processParticlesCPU(data: Float32Array, params: any): Float32Array {
    const result = new Float32Array(data.length);
    const deltaTime = params.deltaTime || 0.016667;
    const animationMode = params.animationMode || 1.0;

    for (let i = 0; i < data.length; i += 3) {
      const x = data[i];
      const y = data[i + 1];
      const z = data[i + 2];
      const particleIndex = i / 3;

      if (animationMode >= 1.0 && animationMode < 2.0) {
        // Galaxy rotation
        const radius = Math.sqrt(x * x + z * z);
        if (radius > 0.001) {
          const angle = Math.atan2(z, x) + deltaTime * 0.5;
          result[i] = radius * Math.cos(angle);
          result[i + 1] = y + Math.sin(deltaTime * 2.0 + particleIndex * 0.01) * 0.1;
          result[i + 2] = radius * Math.sin(angle);
        } else {
          result[i] = x;
          result[i + 1] = y;
          result[i + 2] = z;
        }
      } else {
        // Copy original positions
        result[i] = x;
        result[i + 1] = y;
        result[i + 2] = z;
      }
    }

    return result;
  }

  public async benchmark(particleCount: number = 50000): Promise<any> {
    const testData = new Float32Array(particleCount * 3);

    // Generate test data
    for (let i = 0; i < testData.length; i += 3) {
      testData[i] = ((i / 3) % 100) - 50;
      testData[i + 1] = ((i / 3) % 50) - 25;
      testData[i + 2] = ((i / 3) % 75) - 37;
    }

    const results: any = {
      particleCount,
      methods: {}
    };

    // Benchmark main thread
    const mainStart = performance.now();
    await this.processParticles(testData, 0.016667, 1.0, 'high');
    results.methods.mainThread = {
      time: performance.now() - mainStart,
      particlesPerSecond: (particleCount * 1000) / (performance.now() - mainStart)
    };

    // Benchmark WASM if available
    if (this.capabilities.wasm && this.wasmModule?.benchmarkConcurrentVsGPU) {
      const wasmBenchmark = this.wasmModule.benchmarkConcurrentVsGPU(particleCount);
      results.methods.wasm = wasmBenchmark.results;
    }

    // Request worker benchmarks
    if (this.workers.length > 0) {
      const worker = this.workers[0];
      worker.postMessage({
        type: 'benchmark',
        particleCount
      });
    }

    return results;
  }

  public getCapabilities(): ComputeCapabilities {
    return { ...this.capabilities };
  }

  public getPerformanceMetrics(): PerformanceMetrics[] {
    return this.performanceBuffer.toArray();
  }

  public setAdaptiveQuality(enabled: boolean): void {
    this.adaptiveQuality = enabled;
  }

  public setTargetFPS(fps: number): void {
    this.targetFPS = fps;
  }

  public getStatus(): any {
    return {
      capabilities: this.capabilities,
      pendingTasks: this.pendingTasks.size,
      queueDepth: this.taskQueue.length,
      workers: this.workers.length,
      performance: this.performance.slice(-10),
      wasmStatus: this.wasmModule?.getWorkerPoolStatus?.()
    };
  }

  public destroy(): void {
    // Cleanup workers
    this.workers.forEach(worker => worker.terminate());
    this.workers = [];

    // Clear pending tasks
    this.pendingTasks.clear();
    this.taskQueue = [];

    console.log('[COMPUTE-MANAGER] Destroyed');
  }
}

export { EnhancedComputeManager };
