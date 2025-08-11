// Enhanced Compute Worker for OVASABI Architecture
// Optimized for performance with memory pooling and efficient processing

class ComputeWorker {
  log(...args) {
    const context = this.getContext();
    console.log(`[COMPUTE-WORKER][${this.workerId}]${context}`, ...args);
  }
  warn(...args) {
    const context = this.getContext();
    console.warn(`[COMPUTE-WORKER][${this.workerId}]${context}`, ...args);
  }
  error(...args) {
    const context = this.getContext();
    console.error(`[COMPUTE-WORKER][${this.workerId}]${context}`, ...args);
  }
  getContext() {
    // Add more context for logs: WASM/WebGPU status, paused, degradation, etc.
    return ` [wasm:${this.capabilities.wasm ? 'on' : 'off'}|webgpu:${this.capabilities.webgpu ? 'on' : 'off'}|paused:${this.isPaused ? 'yes' : 'no'}|degradation:${this.performanceMetrics.degradationLevel}]`;
  }
  constructor() {
    this.capabilities = {
      webgpu: false,
      wasm: false,
      javascript: true
    };
    this.workerId = `worker-${Math.random().toString(36).slice(2, 10)}`;
    this.wasmModule = null;
    this.gpuDevice = null;
    this.isProcessing = false;
    this.isPaused = false;
    this.taskQueue = [];
    this.processingCount = 0;
    this.memoryPools = new Map();
    this.maxPoolSize = 10;
    this.activeTimeouts = new Set();
    this.activeTasks = new Map();
    this.activeResources = new Set();
    this.cleanupHooks = new Set();
    this.shutdownInProgress = false;
    this.performanceMetrics = {
      tasksProcessed: 0,
      totalProcessingTime: 0,
      avgProcessingTime: 0,
      peakThroughput: 0,
      currentThroughput: 0,
      lastOptimizationCheck: Date.now(),
      optimizationInterval: 5000,
      consecutiveSlowTasks: 0,
      performanceThreshold: 1000,
      errorCount: 0,
      consecutiveErrors: 0,
      degradationLevel: 0
    };
    this.optimizationHints = {
      preferredMethod: null,
      lastBenchmarkTime: 0,
      shouldRebenchmark: true,
      fallbacksEnabled: true,
      maxRetries: 3
    };
    this.errorRecovery = {
      retryAttempts: new Map(),
      lastErrorTime: 0,
      cooldownPeriod: 5000,
      maxConsecutiveErrors: 5
    };
    // Persistent WebGPU buffers (10 million particles max)
    this.maxParticles = 10000000;
    this.inputBuffer = null;
    this.outputBufferA = null;
    this.outputBufferB = null;
    this.stagingBuffer = null;
    this.originalPositionsBuffer = null;
    this.pingPongFlag = false;
    this.buffersInitialized = false;
    this.wasmReady = false;
    self.addEventListener('wasmReady', () => {
      this.wasmReady = true;
      this.log('WASM module is ready (event received)');
      this.setupWASMBridgeIntegration(); // Ensure bridge setup after WASM is ready
    });
    this.initialize();
    this.setupErrorHandling();
    this.setupCleanupHandlers();
  }

  // Initialize compute worker with all optimizations
  async initialize() {
    this.log('Initializing with enhanced optimizations...', {
      workerId: this.workerId,
      capabilities: this.capabilities,
      maxPoolSize: this.maxPoolSize,
      maxParticles: this.maxParticles
    });
    try {
      await this.initializeModules();
      // Bridge setup now handled by wasmReady event
      this.log('✅ Initialization complete');
    } catch (error) {
      this.error('Initialization failed:', error);
      this.handleCriticalError(error);
    }
  }

  // Setup comprehensive error handling and recovery
  setupErrorHandling() {
    self.addEventListener('error', event => {
      this.error('Uncaught error:', event);
      this.handleCriticalError(event.error);
    });
    self.addEventListener('unhandledrejection', event => {
      this.error('Unhandled promise rejection:', event);
      this.handleCriticalError(event.reason);
    });
  }

  // Setup cleanup handlers for graceful shutdown
  setupCleanupHandlers() {
    this.cleanupHooks.add(() => {
      this.log('Cleaning up memory pools...');
      for (const [size, pool] of this.memoryPools) {
        pool.length = 0; // Clear all pooled buffers
      }
      this.memoryPools.clear();
    });

    this.cleanupHooks.add(() => {
      if (this.webGPUCache) {
        this.log('Cleaning up WebGPU cache...');
        // Note: WebGPU resources are automatically cleaned up by the browser
        // but we can clear our references
        this.webGPUCache = null;
      }
    });

    this.cleanupHooks.add(() => {
      this.log('Clearing active timeouts...');
      for (const timeoutId of this.activeTimeouts) {
        clearTimeout(timeoutId);
      }
      this.activeTimeouts.clear();
    });

    this.cleanupHooks.add(() => {
      this.log('Cancelling active tasks...');
      for (const [taskId, task] of this.activeTasks) {
        if (task.cancel) {
          task.cancel();
        }
      }
      this.activeTasks.clear();
    });
  }

  // Handle critical errors with graceful degradation
  handleCriticalError(error) {
    this.performanceMetrics.errorCount++;
    this.performanceMetrics.consecutiveErrors++;

    this.error('Critical error detected:', error);

    // Implement degradation levels
    if (this.performanceMetrics.consecutiveErrors >= this.errorRecovery.maxConsecutiveErrors) {
      this.warn('Too many consecutive errors, entering emergency mode');
      this.performanceMetrics.degradationLevel = 3; // Emergency mode
      this.gracefulShutdown('Critical error threshold exceeded');
      return;
    }

    // Gradual degradation based on error frequency
    const now = Date.now();
    if (now - this.errorRecovery.lastErrorTime < this.errorRecovery.cooldownPeriod) {
      this.performanceMetrics.degradationLevel = Math.min(
        this.performanceMetrics.degradationLevel + 1,
        2
      );
    }
    this.errorRecovery.lastErrorTime = now;

    // Apply degradation measures
    this.applyDegradationMeasures();

    // Reset consecutive errors after successful operation
    if (this.performanceMetrics.consecutiveErrors > 0) {
      const resetTimeout = this.safeSetTimeout(() => {
        if (this.performanceMetrics.consecutiveErrors > 0) {
          this.performanceMetrics.consecutiveErrors = Math.max(
            0,
            this.performanceMetrics.consecutiveErrors - 1
          );
          if (this.performanceMetrics.consecutiveErrors === 0) {
            this.log('Error recovery: resetting degradation level');
            this.performanceMetrics.degradationLevel = 0;
          }
        }
      }, this.errorRecovery.cooldownPeriod);
    }

    // Notify main thread of degraded state
    self.postMessage({
      type: 'worker-degraded',
      degradationLevel: this.performanceMetrics.degradationLevel,
      errorCount: this.performanceMetrics.errorCount,
      error: error.message || String(error)
    });
  }

  // Apply measures based on degradation level
  applyDegradationMeasures() {
    const level = this.performanceMetrics.degradationLevel;

    switch (level) {
      case 1: // Reduced performance
        this.log('Applying level 1 degradation: reduced performance');
        this.maxPoolSize = Math.max(3, Math.floor(this.maxPoolSize * 0.7));
        this.optimizationHints.fallbacksEnabled = true;
        break;
      case 2: // Basic functionality only
        this.log('Applying level 2 degradation: basic functionality only');
        this.capabilities.webgpu = false; // Disable WebGPU
        this.maxPoolSize = 3;
        this.optimizationHints.preferredMethod = 'javascript';
        break;
      case 3: // Emergency mode
        this.log('Applying level 3 degradation: emergency mode');
        this.capabilities.webgpu = false;
        this.capabilities.wasm = false;
        this.optimizationHints.preferredMethod = 'javascript';
        this.maxPoolSize = 1;
        break;
    }
  }

  // Safe setTimeout that tracks cleanup
  safeSetTimeout(callback, delay) {
    const timeoutId = setTimeout(() => {
      this.activeTimeouts.delete(timeoutId);
      if (!this.shutdownInProgress) {
        callback();
      }
    }, delay);
    this.activeTimeouts.add(timeoutId);
    return timeoutId;
  }

  // Graceful shutdown with cleanup
  gracefulShutdown(reason = 'Unknown') {
    if (this.shutdownInProgress) {
      return; // Already shutting down
    }

    this.log(`Initiating graceful shutdown: ${reason}`);
    this.shutdownInProgress = true;

    // Execute all cleanup hooks
    for (const cleanupFn of this.cleanupHooks) {
      try {
        cleanupFn();
      } catch (error) {
        this.error('Error during cleanup:', error);
      }
    }

    // Notify main thread of shutdown
    self.postMessage({
      type: 'worker-shutdown',
      reason,
      cleanedUp: true
    });

    // Final cleanup
    this.safeSetTimeout(() => {
      this.log('Shutdown complete');
      // Worker will be terminated by main thread
    }, 100);
  }

  // Setup WASM Bridge integration for advanced features
  setupWASMBridgeIntegration() {
    // Expose all compute methods and fallback logic
    this.wasmBridge = {
      getSharedBuffer: typeof self.getSharedBuffer === 'function' ? self.getSharedBuffer : null,
      getGPUMetricsBuffer:
        typeof self.getGPUMetricsBuffer === 'function' ? self.getGPUMetricsBuffer : null,
      getGPUComputeBuffer:
        typeof self.getGPUComputeBuffer === 'function' ? self.getGPUComputeBuffer : null,
      submitComputeTask:
        typeof self.submitComputeTask === 'function' ? self.submitComputeTask : null,
      benchmarkConcurrentVsGPU:
        typeof self.benchmarkConcurrentVsGPU === 'function' ? self.benchmarkConcurrentVsGPU : null,
      runConcurrentCompute:
        typeof self.runConcurrentCompute === 'function' ? self.runConcurrentCompute : null,
      runGPUCompute: typeof self.runGPUCompute === 'function' ? self.runGPUCompute : null,
      compute: typeof self.compute === 'function' ? self.compute : null
    };
    // Robust WASM function detection
    const required = [
      'runConcurrentCompute',
      'submitComputeTask',
      'runGPUCompute',
      'runGPUComputeWithOffset',
      'jsSendWasmMessage',
      'jsRegisterPendingRequest'
    ];
    const missing = required.filter(fn => typeof self[fn] !== 'function');
    if (missing.length) {
      this.warn('Missing WASM functions:', missing);
      // Optionally retry detection after a short delay
      setTimeout(() => this.setupWASMBridgeIntegration(), 100);
      return;
    }
    this.log(
      '✅ WASM Bridge integration checked. Available methods:',
      Object.keys(this.wasmBridge).filter(k => this.wasmBridge[k])
    );
    // Notify frontend of available compute methods and fallback status
    self.postMessage({
      type: 'worker-methods',
      methods: Object.keys(this.wasmBridge).filter(k => this.wasmBridge[k]),
      capabilities: this.capabilities
    });
    // Process any queued tasks after WASM is ready
    if (this.taskQueue && this.taskQueue.length) {
      this.log('Processing queued tasks after WASM ready');
      while (this.taskQueue.length) {
        this.processComputeTask(this.taskQueue.shift());
      }
    }
  }

  // Memory pool management with adaptive sizing
  getBuffer(size) {
    const pool = this.memoryPools.get(size) || [];
    if (pool.length > 0) {
      return pool.pop();
    }
    return new Float32Array(size);
  }

  returnBuffer(buffer) {
    const size = buffer.length;
    const pool = this.memoryPools.get(size) || [];
    if (pool.length < this.maxPoolSize) {
      // Zero out buffer for security and consistency
      buffer.fill(0);
      pool.push(buffer);
      this.memoryPools.set(size, pool);
    }
  }

  // Adaptive performance monitoring
  checkPerformanceOptimization() {
    const now = Date.now();
    if (
      now - this.performanceMetrics.lastOptimizationCheck <
      this.performanceMetrics.optimizationInterval
    ) {
      return;
    }

    this.performanceMetrics.lastOptimizationCheck = now;

    // If performance is consistently below threshold, suggest rebenchmarking
    if (this.performanceMetrics.currentThroughput < this.performanceMetrics.performanceThreshold) {
      this.performanceMetrics.consecutiveSlowTasks++;
      if (this.performanceMetrics.consecutiveSlowTasks > 3) {
        this.optimizationHints.shouldRebenchmark = true;
        this.performanceMetrics.consecutiveSlowTasks = 0;
      }
    } else {
      this.performanceMetrics.consecutiveSlowTasks = 0;
    }

    // Adaptive memory pool sizing based on usage patterns
    for (const [size, pool] of this.memoryPools) {
      if (pool.length > this.maxPoolSize * 0.8) {
        // Pool is well-used, consider increasing size
        this.maxPoolSize = Math.min(this.maxPoolSize + 2, 20);
      } else if (pool.length === 0 && this.performanceMetrics.tasksProcessed > 10) {
        // Pool is underutilized, consider decreasing size
        this.maxPoolSize = Math.max(this.maxPoolSize - 1, 5);
      }
    }
  }

  async initializeModules() {
    try {
      await this.initializeWasm();
      await this.initializeWebGPU();
      this.initialized = true;

      this.log('Initialized successfully', {
        wasm: this.capabilities.wasm,
        webgpu: this.capabilities.webgpu,
        wasmBridge: !!this.wasmBridge,
        concurrency: navigator.hardwareConcurrency || 4
      });

      // Run performance benchmarks if WASM bridge is available
      if (this.wasmBridge && this.wasmBridge.benchmarkConcurrentVsGPU) {
        this.runPerformanceBenchmarks();
      }

      self.postMessage({
        type: 'worker-ready',
        capabilities: {
          wasm: !!this.wasmModule,
          webgpu: !!this.gpuDevice,
          wasmBridge: !!this.wasmBridge,
          concurrency: navigator.hardwareConcurrency || 4
        }
      });
    } catch (error) {
      this.error('Initialization failed:', error);
      self.postMessage({
        type: 'worker-error',
        error: error.message
      });
    }
  }

  async initializeWasm() {
    // Load WASM module in worker context with proper integration
    try {
      this.log('Loading wasm_exec.js...');
      importScripts('/wasm_exec.js');
      this.log('wasm_exec.js loaded. Initializing Go runtime...');
      const go = new Go();
      let result;
      try {
        this.log('Attempting streaming WASM instantiation...');
        result = await WebAssembly.instantiateStreaming(fetch('/main.wasm'), go.importObject);
        this.log('Streaming instantiation succeeded.');
      } catch (streamError) {
        this.warn('Streaming instantiation failed, trying manual fetch:', streamError);
        let wasmUrl = '/main.threads.wasm';
        let wasmResponse;
        try {
          this.log('Trying to fetch main.threads.wasm...');
          wasmResponse = await fetch(wasmUrl);
          if (!wasmResponse.ok) throw new Error('main.threads.wasm not found');
          this.log('main.threads.wasm fetched.');
        } catch (e) {
          this.warn('main.threads.wasm not available, falling back to main.wasm:', e.message);
          wasmUrl = '/main.wasm';
          wasmResponse = await fetch(wasmUrl);
          this.log('main.wasm fetched.');
        }
        const wasmBytes = await wasmResponse.arrayBuffer();
        const wasmModule = await WebAssembly.compile(wasmBytes);
        result = await WebAssembly.instantiate(wasmModule, go.importObject);
        this.log('Manual WASM instantiation succeeded.');
      }
      this.log('Running Go program...');
      go.run(result.instance);
      this.log('Go program started. Waiting for wasmReady event to complete WASM setup...');
      // WASM bridge setup and capability enabling will be handled by the wasmReady event handler.
    } catch (error) {
      console.warn('[COMPUTE-WORKER] WASM integration failed:', error);
      this.capabilities.wasm = false;
    }
  }

  async initializeWebGPU() {
    if ('gpu' in navigator && navigator.gpu) {
      try {
        this.log('Requesting WebGPU adapter...');
        const adapter = await navigator.gpu.requestAdapter();
        if (!adapter) {
          this.warn('WebGPU adapter not available in worker context');
          self.postMessage({
            type: 'worker-warning',
            message: 'WebGPU adapter not available in worker context'
          });
          this.capabilities.webgpu = false;
          return;
        }
        const limits = adapter.limits || {};
        const maxBufferSize =
          typeof limits.maxBufferSize === 'number' ? limits.maxBufferSize : 268435456;
        const maxStorageBufferBindingSize =
          typeof limits.maxStorageBufferBindingSize === 'number'
            ? limits.maxStorageBufferBindingSize
            : 134217728;
        this.log('Requesting WebGPU device with limits:', {
          maxBufferSize,
          maxStorageBufferBindingSize
        });
        const device = await adapter.requestDevice({
          requiredLimits: {
            maxBufferSize: Math.max(maxBufferSize, 4294967296),
            maxStorageBufferBindingSize: maxStorageBufferBindingSize
          }
        });
        if (!device) {
          this.warn('WebGPU device not available in worker context');
          self.postMessage({
            type: 'worker-warning',
            message: 'WebGPU device not available in worker context'
          });
          this.capabilities.webgpu = false;
          return;
        }
        this.gpuDevice = device;
        this.maxBufferSize = device.limits.maxBufferSize;
        this.log('WebGPU device acquired', {
          maxBufferSize: this.maxBufferSize,
          deviceLimits: device.limits
        });
        this.capabilities.webgpu = true;
      } catch (error) {
        this.warn('WebGPU not available in worker:', error);
        self.postMessage({
          type: 'worker-warning',
          message: 'WebGPU not available in worker: ' + error.message
        });
        this.capabilities.webgpu = false;
      }
    } else {
      this.warn('navigator.gpu not available in worker context');
      self.postMessage({
        type: 'worker-warning',
        message: 'navigator.gpu not available in worker context'
      });
      this.capabilities.webgpu = false;
    }
  }

  async processComputeTask(task) {
    if (this.isPaused) {
      this.warn('Worker is paused, cannot process task.', { taskId: task && task.id });
      throw new Error('Worker is paused');
    }
    if (!this.wasmReady) {
      this.log('WASM not ready, queuing task', { taskId: task && task.id });
      this.taskQueue.push(task);
      return;
    }
    if (!task || typeof task !== 'object') {
      this.error('processComputeTask called with invalid task:', task);
      throw new Error('processComputeTask called with invalid task object');
    }
    this.log('Processing compute task', {
      taskId: task.id,
      dataLength: task.data && task.data.length,
      params: task.params
    });
    const startTime = performance.now();
    let result = null;
    let method = 'unknown';
    if (!task.data || typeof task.data.length === 'undefined') {
      this.error('Invalid task data: missing or invalid data array', {
        taskId: task.id,
        data: task.data
      });
      throw new Error('Invalid task data: missing or invalid data array');
    }
    if (task.data.length === 0) {
      this.warn('Empty data array, returning original', { taskId: task.id });
      return task.data;
    }
    const valuesPerParticle = 10;
    let particleCount = Math.floor(task.data.length / valuesPerParticle);
    const remainder = task.data.length % valuesPerParticle;
    if (remainder !== 0) {
      const message = `Non-divisible data length: ${task.data.length} (represents ${particleCount} complete particles + ${remainder} extra values). Data must be divisible by ${valuesPerParticle} for particle data.`;
      this.warn(message, {
        taskId: task.id,
        sample: Array.from(task.data.slice(0, Math.min(20, task.data.length)))
      });
    }
    this.checkPerformanceOptimization();
    try {
      this.log('Selecting compute method...', {
        preferredMethod: this.optimizationHints.preferredMethod,
        dataLength: task.data.length,
        wasmAvailable: !!this.wasmModule,
        webgpuAvailable: !!this.gpuDevice
      });
      if (this.optimizationHints.preferredMethod === 'webgpu' && this.gpuDevice) {
        this.log('Using WebGPU for compute', { taskId: task.id });
        result = await this.processWithWebGPU(task);
        method = 'webgpu';
      } else if (this.optimizationHints.preferredMethod === 'wasm' && this.wasmModule) {
        this.log('Using WASM for compute', { taskId: task.id });
        result = await this.processWithWasm(task);
        method = 'wasm';
      } else if (task.data.length > 50000 && this.gpuDevice) {
        this.log('Using WebGPU for large data', { taskId: task.id });
        result = await this.processWithWebGPU(task);
        method = 'webgpu';
      } else if (task.data.length > 1000 && this.wasmModule) {
        this.log('Using WASM for medium data', { taskId: task.id });
        result = await this.processWithWasm(task);
        method = 'wasm';
      } else {
        this.log('Using JavaScript for compute', { taskId: task.id });
        result = await this.processWithJavaScript(task);
        method = 'javascript';
      }
      const processingTime = performance.now() - startTime;
      this.log('Task processed', {
        taskId: task.id,
        method,
        processingTime,
        particleCount,
        throughput: particleCount / (processingTime / 1000)
      });
      if (result && typeof result.length !== 'undefined' && result !== task.data) {
        if (result instanceof Float32Array) {
        } else if (result instanceof ArrayBuffer) {
          if (result.byteLength === 0) {
            this.error('Detached ArrayBuffer detected, skipping result construction.', {
              taskId: task.id
            });
            return {
              id: task.id,
              error: 'Detached ArrayBuffer',
              metadata: {
                processingTime: performance.now() - startTime,
                method: 'error'
              }
            };
          }
          result = new Float32Array(result);
        } else {
          result = new Float32Array(result);
        }
        if (this.memoryPools.has(result.length)) {
          this.returnBuffer(result);
        }
      }
      return {
        id: task.id,
        data: result,
        metadata: {
          processingTime,
          method,
          particleCount: task.data.length / 10,
          workerThread: true,
          throughput: task.data.length / 10 / (processingTime / 1000),
          optimizationHints: this.optimizationHints.shouldRebenchmark
            ? { shouldRebenchmark: true }
            : undefined
        }
      };
    } catch (error) {
      this.error('Task processing failed:', error, { taskId: task.id, method });
      return {
        id: task.id,
        error: error.message,
        metadata: {
          processingTime: performance.now() - startTime,
          method: 'error'
        }
      };
    }
  }

  async processWithWebGPU(task) {
    if (!this.gpuDevice) {
      throw new Error('WebGPU device not available');
    }
    // Defensive buffer size check
    const bufferSize = task.data.byteLength;
    if (bufferSize > this.maxBufferSize) {
      throw new Error(
        `Requested buffer size (${bufferSize}) exceeds device maxBufferSize (${this.maxBufferSize})`
      );
    }
    if (this.maxStorageBufferBindingSize && bufferSize > this.maxStorageBufferBindingSize) {
      throw new Error(
        `Requested buffer size (${bufferSize}) exceeds device maxStorageBufferBindingSize (${this.maxStorageBufferBindingSize})`
      );
    }

    const startTime = performance.now();
    const valuesPerParticle = 10; // position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1)
    const particleCount = task.data.length / valuesPerParticle;

    // Cache shader module and pipeline for better performance
    if (!this.webGPUCache) {
      this.webGPUCache = {
        shaderModule: null,
        computePipeline: null,
        bindGroupLayout: null
      };
    }

    // Create compute shader for particle processing (cached)
    if (!this.webGPUCache.shaderModule) {
      const computeShaderCode = `
@group(0) @binding(0) var<storage, read> inputData: array<f32>;
@group(0) @binding(1) var<storage, read_write> outputData: array<f32>;
@group(0) @binding(2) var<storage, read> originalPositions: array<f32>;
@group(0) @binding(3) var<uniform> params: vec4f; // x: time, y: animationMode, z: unused, w: intensity scale
@group(0) @binding(4) var<uniform> uParticleCount: u32;

@compute @workgroup_size(256)
fn shade(@builtin(global_invocation_id) global_id: vec3u) {
  let particleIndex = global_id.x;
  let baseIndex = particleIndex * 10u;
  let origBaseIndex = particleIndex * 3u;

  // Bounds check
  if (baseIndex + 9u >= arrayLength(&inputData)) {
    return;
  }

  // Load particle data
  let pos = vec3f(inputData[baseIndex], inputData[baseIndex+1u], inputData[baseIndex+2u]);
  let vel = vec3f(inputData[baseIndex+3u], inputData[baseIndex+4u], inputData[baseIndex+5u]);
  let phase = inputData[baseIndex+6u];
  let intensity = inputData[baseIndex+7u] * params.w;
  let ptype = inputData[baseIndex+8u]; // renamed from 'type' to 'ptype'
  let pid = inputData[baseIndex+9u];

  // Load original position
  let origPos = vec3f(originalPositions[origBaseIndex], originalPositions[origBaseIndex+1u], originalPositions[origBaseIndex+2u]);

  // Animation parameters
  let globalTime = params.x + f32(particleIndex) * 0.001;
  let animationMode = params.y;

  var newPos = pos;
  var newVel = vel;

  // Galaxy rotation
  if (animationMode == 1.0) {
    let radius = sqrt(origPos.x * origPos.x + origPos.z * origPos.z);
    let angle = atan2(origPos.z, origPos.x) + globalTime * 0.5 + phase * 0.1;
    let spiral = radius * (1.0 + 0.2 * sin(globalTime * 0.3 + phase));
    newPos = vec3f(
      spiral * cos(angle),
      origPos.y + sin(globalTime * 2.0 + phase) * 0.1 * intensity,
      spiral * sin(angle)
    );
    newVel = (newPos - pos) / max(params.x, 0.0001);
  }
  // Yin-Yang flow
  else if (animationMode == 2.0) {
    let t = globalTime + origPos.x * 0.2 + phase * 0.2;
    let yinYang = vec2f(
      cos(t) * 0.5 + sign(origPos.x) * 0.3,
      sin(t) * 0.5
    );
    newPos = origPos + vec3f(yinYang.x, 0.0, yinYang.y) * intensity * (1.0 + ptype * 0.2);
    newVel = (newPos - pos) / max(params.x, 0.0001);
  }
  // Spiral motion
  else if (animationMode == 3.0) {
    let spiralRadius = 0.5 + intensity * 0.3 + ptype * 0.1;
    let angle = globalTime * 2.0 + origPos.y * 0.5 + phase * 0.2;
    newPos = vec3f(
      spiralRadius * cos(angle),
      origPos.y,
      spiralRadius * sin(angle)
    );
    newVel = (newPos - pos) / max(params.x, 0.0001);
  }
  // Wave motion
  else if (animationMode == 4.0) {
    let wave = sin(origPos.x * 2.0 + globalTime * 5.0 + phase) * 0.3;
    newPos = vec3f(origPos.x, origPos.y + wave * intensity * (1.0 + ptype * 0.2), origPos.z);
    newVel = (newPos - pos) / max(params.x, 0.0001);
  }

  // Write results
  outputData[baseIndex] = newPos.x;
  outputData[baseIndex+1u] = newPos.y;
  outputData[baseIndex+2u] = newPos.z;
  outputData[baseIndex+3u] = newVel.x;
  outputData[baseIndex+4u] = newVel.y;
  outputData[baseIndex+5u] = newVel.z;
  outputData[baseIndex+6u] = phase;
  outputData[baseIndex+7u] = intensity;
  outputData[baseIndex+8u] = ptype;
  outputData[baseIndex+9u] = pid;
}
      `;

      this.webGPUCache.shaderModule = this.gpuDevice.createShaderModule({
        code: computeShaderCode
      });
    }

    // Create bind group layout (cached)
    if (!this.webGPUCache.bindGroupLayout) {
      this.webGPUCache.bindGroupLayout = this.gpuDevice.createBindGroupLayout({
        entries: [
          {
            binding: 0,
            visibility: GPUShaderStage.COMPUTE,
            buffer: { type: 'read-only-storage' }
          },
          {
            binding: 1,
            visibility: GPUShaderStage.COMPUTE,
            buffer: { type: 'storage' }
          },
          {
            binding: 2,
            visibility: GPUShaderStage.COMPUTE,
            buffer: { type: 'read-only-storage' }
          },
          {
            binding: 3,
            visibility: GPUShaderStage.COMPUTE,
            buffer: { type: 'uniform' }
          },
          {
            binding: 4,
            visibility: GPUShaderStage.COMPUTE,
            buffer: { type: 'uniform' }
          }
        ]
      });
    }

    // Create compute pipeline (cached)
    if (!this.webGPUCache.computePipeline) {
      this.webGPUCache.computePipeline = this.gpuDevice.createComputePipeline({
        layout: this.gpuDevice.createPipelineLayout({
          bindGroupLayouts: [this.webGPUCache.bindGroupLayout]
        }),
        compute: {
          module: this.webGPUCache.shaderModule,
          entryPoint: 'shade'
        }
      });
    }

    // Initialize persistent buffers if needed
    const requiredSize = task.data.byteLength;
    if (this.maxStorageBufferBindingSize && requiredSize > this.maxStorageBufferBindingSize) {
      throw new Error(
        `Requested buffer size (${requiredSize}) exceeds device maxStorageBufferBindingSize (${this.maxStorageBufferBindingSize})`
      );
    }
    if (!this.buffersInitialized || requiredSize > this.inputBuffer?.size) {
      // (Re)allocate persistent buffers for maxParticles
      const persistentSize = Math.max(requiredSize, this.maxParticles * 8 * 4);
      if (this.maxStorageBufferBindingSize && persistentSize > this.maxStorageBufferBindingSize) {
        throw new Error(
          `Persistent buffer size (${persistentSize}) exceeds device maxStorageBufferBindingSize (${this.maxStorageBufferBindingSize})`
        );
      }
      this.inputBuffer = this.gpuDevice.createBuffer({
        size: persistentSize,
        usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_DST
      });
      this.outputBufferA = this.gpuDevice.createBuffer({
        size: persistentSize,
        usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_SRC
      });
      this.outputBufferB = this.gpuDevice.createBuffer({
        size: persistentSize,
        usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_SRC
      });
      this.stagingBuffer = this.gpuDevice.createBuffer({
        size: persistentSize,
        usage: GPUBufferUsage.COPY_DST | GPUBufferUsage.MAP_READ
      });
      this.originalPositionsBuffer = this.gpuDevice.createBuffer({
        size: persistentSize,
        usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_DST
      });
      this.buffersInitialized = true;
    }

    // Write input data
    this.gpuDevice.queue.writeBuffer(this.inputBuffer, 0, task.data);

    // Write original positions if provided
    if (task.originalPositions && task.originalPositions instanceof Float32Array) {
      this.gpuDevice.queue.writeBuffer(this.originalPositionsBuffer, 0, task.originalPositions);
    }

    // Ping-pong output buffer selection
    const outputBuffer = this.pingPongFlag ? this.outputBufferB : this.outputBufferA;
    this.pingPongFlag = !this.pingPongFlag;

    // Params buffer (per task)
    const paramsBuffer = this.gpuDevice.createBuffer({
      size: 16, // 4 floats
      usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST
    });
    const params = new Float32Array([
      task.params.deltaTime || 0.016667,
      task.params.animationMode || 1.0,
      0, // Reserved
      0 // Reserved
    ]);
    this.gpuDevice.queue.writeBuffer(paramsBuffer, 0, params);

    // Defensive null checks before bind group creation
    if (!this.inputBuffer || !outputBuffer || !this.originalPositionsBuffer || !paramsBuffer) {
      throw new Error('One or more GPU buffers are undefined before createBindGroup');
    }

    // Create bind group
    // Create extra uniform buffer for binding 4
    const extraUniformBuffer = this.gpuDevice.createBuffer({
      size: 16, // Adjust size as needed for your shader
      usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST
    });
    // Fill extraUniformBuffer with appropriate data (all zeros for now)
    this.gpuDevice.queue.writeBuffer(extraUniformBuffer, 0, new Float32Array([0, 0, 0, 0]));

    const bindGroup = this.gpuDevice.createBindGroup({
      layout: this.webGPUCache.bindGroupLayout,
      entries: [
        { binding: 0, resource: { buffer: this.inputBuffer } },
        { binding: 1, resource: { buffer: outputBuffer } },
        { binding: 2, resource: { buffer: this.originalPositionsBuffer } },
        { binding: 3, resource: { buffer: paramsBuffer } },
        { binding: 4, resource: { buffer: extraUniformBuffer } }
      ]
    });

    // Serialize submit and map/unmap operations
    if (!this._gpuOpQueue) {
      this._gpuOpQueue = Promise.resolve();
    }
    const doGpuOp = async () => {
      // Submit compute pass only when buffer is not mapped
      while (this._stagingMapPending) {
        await new Promise(resolve => setTimeout(resolve, 1));
      }
      // Create command encoder and dispatch
      const commandEncoder = this.gpuDevice.createCommandEncoder();
      const computePass = commandEncoder.beginComputePass();
      computePass.setPipeline(this.webGPUCache.computePipeline);
      computePass.setBindGroup(0, bindGroup);
      const workgroupCount = Math.ceil(task.data.length / 256);
      computePass.dispatchWorkgroups(workgroupCount);
      computePass.end();
      // Copy output to persistent staging buffer
      commandEncoder.copyBufferToBuffer(outputBuffer, 0, this.stagingBuffer, 0, requiredSize);
      this.gpuDevice.queue.submit([commandEncoder.finish()]);
      // Map/unmap staging buffer for CPU readback
      this._stagingMapPending = true;
      await this.stagingBuffer.mapAsync(GPUMapMode.READ);
      const resultArrayBuffer = this.stagingBuffer.getMappedRange().slice(0, requiredSize);
      // Defensive: copy buffer before transfer and before unmapping
      const safeCopy = new Float32Array(resultArrayBuffer.byteLength / 4);
      safeCopy.set(new Float32Array(resultArrayBuffer));
      this.stagingBuffer.unmap();
      this._stagingMapPending = false;
      return safeCopy;
    };
    this._gpuOpQueue = this._gpuOpQueue.then(doGpuOp);
    const resultCopy = await this._gpuOpQueue;

    // Destroy params buffer (per-task)
    paramsBuffer.destroy();

    // Update performance metrics
    const processingTime = performance.now() - startTime;
    this.performanceMetrics.tasksProcessed++;
    this.performanceMetrics.totalProcessingTime += processingTime;
    this.performanceMetrics.avgProcessingTime =
      this.performanceMetrics.totalProcessingTime / this.performanceMetrics.tasksProcessed;
    const throughput = particleCount / (processingTime / 1000);
    this.performanceMetrics.currentThroughput = throughput;
    if (throughput > this.performanceMetrics.peakThroughput) {
      this.performanceMetrics.peakThroughput = throughput;
    }

    // Use transferables for large result
    // Defensive: post a copy of the buffer, not the original, to avoid DataCloneError
    const transferBuffer = resultCopy.buffer.slice(0);
    self.postMessage({ type: 'result', data: transferBuffer }, [transferBuffer]);
    return resultCopy;
  }

  async processWithWasm(task) {
    if (!this.wasmModule) {
      throw new Error('WASM module not available');
    }

    const startTime = performance.now();
    const valuesPerParticle = 8; // position(3) + velocity(3) + time(1) + intensity(1)
    const particleCount = task.data.length / valuesPerParticle;

    // Use WASM concurrent processing if available
    if (this.wasmModule.runConcurrentCompute) {
      return new Promise((resolve, reject) => {
        const callback = (result, metadata) => {
          const processingTime = performance.now() - startTime;

          // Update performance metrics
          this.performanceMetrics.tasksProcessed++;
          this.performanceMetrics.totalProcessingTime += processingTime;
          this.performanceMetrics.avgProcessingTime =
            this.performanceMetrics.totalProcessingTime / this.performanceMetrics.tasksProcessed;

          const throughput = particleCount / (processingTime / 1000);
          this.performanceMetrics.currentThroughput = throughput;
          if (throughput > this.performanceMetrics.peakThroughput) {
            this.performanceMetrics.peakThroughput = throughput;
          }

          // Apply any additional metadata if available
          if (metadata && metadata.optimizationHints) {
            this.optimizationHints = { ...this.optimizationHints, ...metadata.optimizationHints };
          }

          resolve(result);
        };

        try {
          const success = this.wasmModule.runConcurrentCompute(
            task.data,
            task.params.deltaTime || 0.016667,
            task.params.animationMode || 1.0,
            callback
          );

          if (!success) {
            reject(new Error('WASM concurrent compute failed'));
          }
        } catch (error) {
          this.warn('WASM concurrent compute error:', error);
          reject(error);
        }
      });
    }

    // Fallback to basic WASM processing with simple compute
    if (this.wasmModule.compute) {
      try {
        const result = this.wasmModule.compute(
          task.data,
          task.params.deltaTime || 0.016667,
          task.params.animationMode || 1.0
        );

        const processingTime = performance.now() - startTime;

        // Update performance metrics
        this.performanceMetrics.tasksProcessed++;
        this.performanceMetrics.totalProcessingTime += processingTime;
        this.performanceMetrics.avgProcessingTime =
          this.performanceMetrics.totalProcessingTime / this.performanceMetrics.tasksProcessed;

        const throughput = particleCount / (processingTime / 1000);
        this.performanceMetrics.currentThroughput = throughput;
        if (throughput > this.performanceMetrics.peakThroughput) {
          this.performanceMetrics.peakThroughput = throughput;
        }

        return result;
      } catch (error) {
        this.warn('WASM basic compute error:', error);
        throw error;
      }
    }

    // Final fallback to JavaScript processing
    this.warn('No WASM compute functions available, falling back to JavaScript');
    return this.processWithJavaScript(task);
  }

  async processWithJavaScript(task) {
    const startTime = performance.now();
    const valuesPerParticle = 10; // position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1)
    const particleCount = Math.floor(task.data.length / valuesPerParticle);

    // Use memory pool for result buffer
    const result = this.getBuffer(particleCount * valuesPerParticle);

    // Copy only complete particle data
    result.set(task.data.slice(0, particleCount * valuesPerParticle));

    const deltaTime = task.params.deltaTime || 0.016667;
    const animationMode = task.params.animationMode || 1.0;

    // Batch processing for better performance
    const batchSize = Math.min(1000, particleCount);

    for (let batch = 0; batch < particleCount; batch += batchSize) {
      const endIndex = Math.min(batch + batchSize, particleCount);

      for (let i = batch; i < endIndex; i++) {
        const i10 = i * 10; // 10 values per particle

        // Extract position, velocity, phase, intensity, type, id
        const x = result[i10];
        const y = result[i10 + 1];
        const z = result[i10 + 2];
        const vx = result[i10 + 3];
        const vy = result[i10 + 4];
        const vz = result[i10 + 5];
        const phase = result[i10 + 6];
        const intensity = result[i10 + 7];
        const type = result[i10 + 8];
        const id = result[i10 + 9];

        if (animationMode >= 1.0 && animationMode < 2.0) {
          // Galaxy rotation - optimized calculations with phase and type
          const radius = Math.sqrt(x * x + z * z);
          if (radius > 0.001) {
            const angle = Math.atan2(z, x) + deltaTime * 0.5 + phase * 0.1;
            const cosAngle = Math.cos(angle);
            const sinAngle = Math.sin(angle);

            result[i10] = radius * cosAngle;
            result[i10 + 1] = y + Math.sin(deltaTime * 2.0 + i * 0.01 + phase) * 0.1;
            result[i10 + 2] = radius * sinAngle;
          }
        } else if (animationMode >= 2.0) {
          // Wave motion with phase and type
          const wavePhase = deltaTime * 5.0 + x * 0.2 + z * 0.2 + phase;
          result[i10] = x;
          result[i10 + 1] = y + Math.sin(wavePhase) * 0.4 * (1.0 + type * 0.2);
          result[i10 + 2] = z;
        }
        // Other modes can be extended similarly
      }

      // Yield control occasionally for responsiveness
      if (batch % 5000 === 0 && batch > 0) {
        await new Promise(resolve => setTimeout(resolve, 0));
      }
    }

    const processingTime = performance.now() - startTime;

    // Update performance metrics
    this.performanceMetrics.tasksProcessed++;
    this.performanceMetrics.totalProcessingTime += processingTime;
    this.performanceMetrics.avgProcessingTime =
      this.performanceMetrics.totalProcessingTime / this.performanceMetrics.tasksProcessed;

    const throughput = particleCount / (processingTime / 1000);
    this.performanceMetrics.currentThroughput = throughput;
    if (throughput > this.performanceMetrics.peakThroughput) {
      this.performanceMetrics.peakThroughput = throughput;
    }

    return result;
  }

  // Run performance benchmarks to determine optimal processing method
  async runPerformanceBenchmarks() {
    if (!this.wasmBridge || !this.wasmBridge.benchmarkConcurrentVsGPU) {
      return;
    }
    this.log('Running performance benchmarks...');
    try {
      // Generate test data with 8 floats per particle
      const testSizes = [1000, 10000, 50000];
      const benchmarkResults = {};
      for (const size of testSizes) {
        const testData = new Float32Array(size * 8);
        for (let i = 0; i < size; i++) {
          // Fill with plausible values: pos(3), vel(3), time, intensity
          const base = i * 8;
          testData[base + 0] = (Math.random() - 0.5) * 10; // x
          testData[base + 1] = (Math.random() - 0.5) * 10; // y
          testData[base + 2] = (Math.random() - 0.5) * 10; // z
          testData[base + 3] = (Math.random() - 0.5) * 2; // vx
          testData[base + 4] = (Math.random() - 0.5) * 2; // vy
          testData[base + 5] = (Math.random() - 0.5) * 2; // vz
          testData[base + 6] = Math.random(); // time
          testData[base + 7] = Math.random(); // intensity
        }
        // Warm-up run (ignore result)
        await this.processWithJavaScript({
          data: testData,
          params: { deltaTime: 0.016667, animationMode: 1.0 }
        });
        // Benchmark JavaScript
        const jsStart = performance.now();
        await this.processWithJavaScript({
          data: testData,
          params: { deltaTime: 0.016667, animationMode: 1.0 }
        });
        benchmarkResults[`javascript_${size}`] = {
          time: performance.now() - jsStart,
          particlesPerMs: size / (performance.now() - jsStart)
        };
        // Benchmark WASM if available
        if (this.wasmModule) {
          await this.processWithWasm({
            data: testData,
            params: { deltaTime: 0.016667, animationMode: 1.0 }
          }); // warm-up
          const wasmStart = performance.now();
          await this.processWithWasm({
            data: testData,
            params: { deltaTime: 0.016667, animationMode: 1.0 }
          });
          benchmarkResults[`wasm_${size}`] = {
            time: performance.now() - wasmStart,
            particlesPerMs: size / (performance.now() - wasmStart)
          };
        }
        // Benchmark WebGPU if available
        if (this.gpuDevice) {
          await this.processWithWebGPU({
            data: testData,
            params: { deltaTime: 0.016667, animationMode: 1.0 }
          }); // warm-up
          const gpuStart = performance.now();
          await this.processWithWebGPU({
            data: testData,
            params: { deltaTime: 0.016667, animationMode: 1.0 }
          });
          benchmarkResults[`webgpu_${size}`] = {
            time: performance.now() - gpuStart,
            particlesPerMs: size / (performance.now() - gpuStart)
          };
        }
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      this.analyzeAndSetOptimalMethod(benchmarkResults);
      this.log('✅ Performance benchmarks complete:', benchmarkResults);
      self.postMessage({
        type: 'benchmark-complete',
        results: benchmarkResults,
        optimalMethod: this.optimizationHints.preferredMethod
      });
    } catch (error) {
      this.error('Benchmark error:', error);
    }
  }

  async runBenchmark(config) {
    const particleCount = config.particleCount || 50000;
    const testData = new Float32Array(particleCount * 8);
    for (let i = 0; i < particleCount; i++) {
      const base = i * 8;
      testData[base + 0] = (i % 100) - 50; // x
      testData[base + 1] = (i % 50) - 25; // y
      testData[base + 2] = (i % 75) - 37; // z
      testData[base + 3] = (Math.random() - 0.5) * 2; // vx
      testData[base + 4] = (Math.random() - 0.5) * 2; // vy
      testData[base + 5] = (Math.random() - 0.5) * 2; // vz
      testData[base + 6] = Math.random(); // time
      testData[base + 7] = Math.random(); // intensity
    }
    // Warm-up run
    await this.processWithJavaScript({
      data: testData,
      params: { deltaTime: 0.016667, animationMode: 1.0 }
    });
    const results = {};
    // Benchmark JavaScript
    const jsStart = performance.now();
    await this.processWithJavaScript({
      data: testData,
      params: { deltaTime: 0.016667, animationMode: 1.0 }
    });
    results.javascript = {
      time: performance.now() - jsStart,
      particlesPerMs: particleCount / (performance.now() - jsStart)
    };
    // Benchmark WASM if available
    if (this.wasmModule) {
      await this.processWithWasm({
        data: testData,
        params: { deltaTime: 0.016667, animationMode: 1.0 }
      }); // warm-up
      const wasmStart = performance.now();
      await this.processWithWasm({
        data: testData,
        params: { deltaTime: 0.016667, animationMode: 1.0 }
      });
      results.wasm = {
        time: performance.now() - wasmStart,
        particlesPerMs: particleCount / (performance.now() - wasmStart)
      };
    }
    // Benchmark WebGPU if available
    if (this.gpuDevice) {
      await this.processWithWebGPU({
        data: testData,
        params: { deltaTime: 0.016667, animationMode: 1.0 }
      }); // warm-up
      const gpuStart = performance.now();
      await this.processWithWebGPU({
        data: testData,
        params: { deltaTime: 0.016667, animationMode: 1.0 }
      });
      results.webgpu = {
        time: performance.now() - gpuStart,
        particlesPerMs: particleCount / (performance.now() - gpuStart)
      };
    }
    self.postMessage({
      type: 'benchmark-result',
      results,
      particleCount
    });
  }

  // Message handler for main thread communication
  async handleMessage(event) {
    const { type, task, config } = event.data || {};
    if (type === 'compute-task' && (!task || typeof task !== 'object')) {
      this.error('Invalid compute-task message: missing or invalid task object', event.data);
      self.postMessage({
        type: 'worker-error',
        error: 'Invalid compute-task message: missing or invalid task object',
        originalMessage: event.data
      });
      return;
    }
    try {
      switch (type) {
        case 'compute-task':
          // Process compute task and post result
          const result = await this.processComputeTask(task);
          self.postMessage({ type: 'compute-result', result });
          break;
        case 'benchmark':
          await this.runBenchmark(config || {});
          break;
        case 'run-performance-benchmarks':
          await this.runPerformanceBenchmarks();
          break;
        case 'pause':
          this.isPaused = true;
          self.postMessage({ type: 'worker-paused' });
          break;
        case 'resume':
          this.isPaused = false;
          self.postMessage({ type: 'worker-resumed' });
          break;
        default:
          this.warn('Unknown message type:', type, event.data);
          self.postMessage({ type: 'worker-warning', message: `Unknown message type: ${type}` });
      }
    } catch (error) {
      this.error('Error handling message:', error);
      self.postMessage({ type: 'worker-error', error: error.message });
    }
  }
}

// Initialize worker
const computeWorker = new ComputeWorker();

// Handle messages from main thread
self.onmessage = event => {
  // Graceful self-termination on shutdown message
  if (event.data && event.data.type === 'shutdown') {
    computeWorker.log('Received shutdown message, terminating worker...');
    // Run cleanup hooks before closing
    if (typeof computeWorker.gracefulShutdown === 'function') {
      computeWorker.gracefulShutdown('Main thread requested shutdown');
    }
    self.close();
    return;
  }
  computeWorker.handleMessage(event);
};

// Error handling
self.onerror = error => {
  computeWorker.error('Error:', error);
  self.postMessage({
    type: 'worker-error',
    error: error.message,
    workerId: computeWorker.workerId
  });
};

computeWorker.log('Worker started');

// Ensure all particle data is processed in 8-value format (position(3), velocity(3), time, intensity)
// Use WASM and WebGPU compute for large-scale updates, fallback to JS if needed
// Benchmark and select optimal compute method dynamically
// Expose all compute APIs to frontend for seamless integration
// Robust error handling, graceful shutdown, and performance monitoring
