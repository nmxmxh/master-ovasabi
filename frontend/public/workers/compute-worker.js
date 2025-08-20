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

  // Refactored: Only delegate compute to WASM via bridge, never process/shade/animate in JS
  async processComputeTask(task) {
    if (this.isPaused) {
      this.warn('Worker is paused, cannot process task.', { taskId: task && task.id });
      return {
        id: task && task.id ? task.id : null,
        error: 'Worker is paused',
        metadata: { method: 'error' }
      };
    }
    if (!this.wasmReady) {
      this.log('WASM not ready, queuing task', { taskId: task && task.id });
      this.taskQueue.push(task);
      return;
    }
    if (
      !task ||
      typeof task !== 'object' ||
      !task.data ||
      typeof task.data.length !== 'number' ||
      !task.params
    ) {
      this.error('processComputeTask called with invalid task:', task);
      self.postMessage({
        type: 'worker-error',
        error: 'processComputeTask called with invalid task object',
        originalMessage: task
      });
      return {
        id: null,
        error: 'Invalid task object',
        metadata: { method: 'error' }
      };
    }
    // Documented format: { id, data: Float32Array, params: { deltaTime, animationMode } }
    this.log('Forwarding compute task to WASM', {
      taskId: task.id || null,
      dataLength: task.data.length,
      params: task.params
    });
    // Always delegate to WASM via bridge
    try {
      if (typeof self.runConcurrentCompute === 'function') {
        // Use runConcurrentCompute for all tasks
        return await new Promise((resolve, reject) => {
          self.runConcurrentCompute(
            task.data,
            task.params.deltaTime || 0.016667,
            task.params.animationMode || 1.0,
            function (result, metadata) {
              resolve({
                id: task.id,
                data: result,
                metadata: metadata || {}
              });
            }
          );
        });
      } else {
        this.error('WASM bridge runConcurrentCompute not available');
        return {
          id: task.id,
          error: 'WASM bridge runConcurrentCompute not available',
          metadata: { method: 'error' }
        };
      }
    } catch (error) {
      this.error('Task processing failed (WASM bridge):', error, { taskId: task.id });
      return {
        id: task.id,
        error: error.message,
        metadata: { method: 'error' }
      };
    }
  }

  // Removed: All WebGPU compute logic. Only WASM should handle compute/shading/animation.
  async processWithWebGPU(task) {
    this.error('processWithWebGPU should not be called. All compute is delegated to WASM.');
    return {
      id: task.id,
      error: 'processWithWebGPU not supported. Use WASM bridge.',
      metadata: { method: 'error' }
    };
  }

  // Removed: All JS compute logic. Only WASM should handle compute/shading/animation.
  async processWithJavaScript(task) {
    this.error('processWithJavaScript should not be called. All compute is delegated to WASM.');
    return {
      id: task.id,
      error: 'processWithJavaScript not supported. Use WASM bridge.',
      metadata: { method: 'error' }
    };
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
    // Defensive: log full event for traceability
    if (type === 'compute-task' && (!task || typeof task !== 'object')) {
      this.error('Invalid compute-task message: missing or invalid task object', event.data);
      self.postMessage({
        type: 'worker-error',
        error: 'Invalid compute-task message: missing or invalid task object',
        originalMessage: event.data
      });
      // Return a result with id: null for main thread handling
      self.postMessage({
        type: 'compute-result',
        result: {
          id: null,
          error: 'Invalid compute-task message',
          metadata: { method: 'error' }
        }
      });
      return;
    }
    try {
      switch (type) {
        case 'compute-task': {
          // Process compute task and post result
          const result = await this.processComputeTask(task);
          self.postMessage({ type: 'compute-result', result });
          break;
        }
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
