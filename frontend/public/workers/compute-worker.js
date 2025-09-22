// Enhanced Compute Worker for OVASABI Architecture
// Focused on compute operations with proper separation from WASM bridge

class ComputeWorker {
  log(...args) {
    // Reduced logging - only log important events
    if (
      args[0] &&
      typeof args[0] === 'string' &&
      (args[0].includes('✅') ||
        args[0].includes('❌') ||
        args[0].includes('ERROR') ||
        args[0].includes('Initialization'))
    ) {
      const context = this.getContext();
      console.log(`[COMPUTE-WORKER][${this.workerId}]${context}`, ...args);
    }
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
    // Simplified context for compute worker
    return ` [wasm:${this.capabilities.wasm ? 'on' : 'off'}|webgpu:${this.capabilities.webgpu ? 'on' : 'off'}|paused:${this.isPaused ? 'yes' : 'no'}]`;
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
      this.setupWASMIntegration(); // Setup WASM integration after WASM is ready
    });
    this.initialize();
    this.setupErrorHandling();
    this.setupCleanupHandlers();
  }

  // Initialize compute worker with all optimizations
  async initialize() {
    // Initializing with enhanced optimizations
    try {
      await this.initializeModules();
      // WASM integration setup handled by wasmReady event
      // Initialization complete
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

  // Setup WASM integration for compute operations
  setupWASMIntegration() {
    // Workers don't have direct access to WASM functions from main thread
    // Instead, we'll communicate with the main thread for WASM operations
    // Setting up WASM integration via main thread communication

    // WASM is available via main thread communication
    this.capabilities.wasm = true;
    this.capabilities.wasmViaMainThread = true;

    // WASM integration set up via main thread communication

    // Notify main thread of capabilities
    self.postMessage({
      type: 'worker-capabilities',
      capabilities: this.capabilities,
      wasmViaMainThread: true
    });

    // Process any queued tasks after WASM is ready
    if (this.taskQueue && this.taskQueue.length) {
      this.log('Processing queued tasks after WASM ready');
      while (this.taskQueue.length) {
        this.processComputeTask(this.taskQueue.shift());
      }
    }
  }

  // Call WASM function via main thread communication
  async callWASMFunction(functionName, args) {
    return new Promise((resolve, reject) => {
      const callId = `wasm_call_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

      // Store the callback for when we get the response
      this.wasmCallbacks = this.wasmCallbacks || new Map();
      this.wasmCallbacks.set(callId, { resolve, reject });

      // Send request to main thread
      self.postMessage({
        type: 'wasm-call-request',
        callId,
        functionName,
        args: args
      });

      // Set timeout to avoid hanging
      setTimeout(() => {
        if (this.wasmCallbacks.has(callId)) {
          this.wasmCallbacks.delete(callId);
          reject(new Error(`WASM call timeout: ${functionName}`));
        }
      }, 10000); // 10 second timeout
    });
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

      // Initialized successfully

      // Run performance benchmarks if WASM functions are available
      if (this.capabilities.wasm && typeof self.runConcurrentCompute === 'function') {
        this.runPerformanceBenchmarks();
      }

      self.postMessage({
        type: 'worker-ready',
        capabilities: {
          wasm: this.capabilities.wasm,
          webgpu: this.capabilities.webgpu,
          javascript: this.capabilities.javascript,
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
      // Loading wasm_exec.js

      // Try to load wasm_exec.js with error handling
      try {
        importScripts('/wasm_exec.js');
        // wasm_exec.js loaded
      } catch (importError) {
        this.warn('Failed to import wasm_exec.js:', importError);
        this.capabilities.wasm = false;
        return;
      }

      const go = new Go();
      let result;

      try {
        // Attempting streaming WASM instantiation
        result = await WebAssembly.instantiateStreaming(
          fetch(`/main.wasm?v=${Date.now()}`),
          go.importObject
        );
        // Streaming instantiation succeeded
      } catch (streamError) {
        this.warn('Streaming instantiation failed, trying manual fetch:', streamError);

        let wasmUrl = '/main.threads.wasm';
        let wasmResponse;

        try {
          // Trying to fetch main.threads.wasm
          wasmResponse = await fetch(`${wasmUrl}?v=${Date.now()}`);
          if (!wasmResponse.ok) throw new Error('main.threads.wasm not found');
          // main.threads.wasm fetched
        } catch (e) {
          this.warn('main.threads.wasm not available, falling back to main.wasm:', e.message);
          wasmUrl = '/main.wasm';
          wasmResponse = await fetch(`${wasmUrl}?v=${Date.now()}`);
          // main.wasm fetched
        }

        const wasmBytes = await wasmResponse.arrayBuffer();
        const wasmModule = await WebAssembly.compile(wasmBytes);
        result = await WebAssembly.instantiate(wasmModule, go.importObject);
        // Manual WASM instantiation succeeded
      }

      // Running Go program
      go.run(result.instance);
      // Go program started

      // Workers don't have direct access to WASM functions - they communicate with main thread
      // Worker WASM initialization complete
      this.wasmReady = true;
      this.setupWASMIntegration();
    } catch (error) {
      console.warn('[COMPUTE-WORKER] WASM integration failed:', error);
      this.capabilities.wasm = false;
      this.setupWASMIntegration();
    }
  }

  async initializeWebGPU() {
    // Skip WebGPU initialization in workers to prevent "external Instance reference" errors
    // WebGPU should only be initialized in the main thread to avoid conflicts
    // Skipping WebGPU initialization in worker
    this.capabilities.webgpu = false;

    // Notify main thread that worker is ready without WebGPU
    self.postMessage({
      type: 'worker-ready',
      capabilities: this.capabilities,
      message: 'Worker ready without WebGPU (centralized in main thread)'
    });
    return;
  }

  // Process compute tasks with WASM or JavaScript fallback
  async processComputeTask(task) {
    if (this.isPaused) {
      this.warn('Worker is paused, cannot process task.', { taskId: task && task.id });
      return {
        id: task && task.id ? task.id : null,
        error: 'Worker is paused',
        metadata: { method: 'error' }
      };
    }

    if (!task || typeof task !== 'object' || !task.data || typeof task.data.length !== 'number') {
      this.error('processComputeTask called with invalid task:', task);
      return {
        id: task?.id || null,
        error: 'Invalid task object',
        metadata: { method: 'error' }
      };
    }

    const taskId = task.id || `task-${Date.now()}`;
    const dataLength = task.data.length;
    const params = task.params || { deltaTime: 0.016667, animationMode: 1.0 };

    this.log('Processing compute task', {
      taskId,
      dataLength,
      params,
      wasmReady: this.wasmReady
    });

    try {
      // Try WASM first if available (via main thread communication)
      if (this.capabilities.wasm && this.capabilities.wasmViaMainThread) {
        return await this.processWithWASMViaMainThread(taskId, task.data, params);
      } else {
        // Fallback to JavaScript
        return await this.processWithJavaScript(taskId, task.data, params);
      }
    } catch (error) {
      this.error('Task processing failed:', error, { taskId });
      return {
        id: taskId,
        error: error.message,
        metadata: { method: 'error' }
      };
    }
  }

  async processWithWASM(taskId, data, params) {
    return new Promise((resolve, reject) => {
      try {
        self.runConcurrentCompute(
          data,
          params.deltaTime,
          params.animationMode,
          (result, metadata) => {
            resolve({
              id: taskId,
              data: result,
              metadata: { ...metadata, method: 'wasm' }
            });
          }
        );
      } catch (error) {
        reject(error);
      }
    });
  }

  async processWithWASMViaMainThread(taskId, data, params) {
    try {
      const result = await this.callWASMFunction('runConcurrentCompute', [
        data,
        params.deltaTime,
        params.animationMode
      ]);

      return {
        id: taskId,
        data: result.result || result,
        metadata: {
          ...(result.metadata || {}),
          method: 'wasm-via-main-thread'
        }
      };
    } catch (error) {
      this.warn('WASM via main thread failed, falling back to JavaScript:', error.message);
      return await this.processWithJavaScript(taskId, data, params);
    }
  }

  async processWithJavaScript(taskId, data, params) {
    const result = data.slice();
    const particleCount = result.length / 3;
    const deltaTime = params.deltaTime || 0.016667;

    // Simple JavaScript particle physics
    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;

      // Basic orbital motion
      const x = result[i3];
      const z = result[i3 + 2];
      const radius = Math.sqrt(x * x + z * z);

      if (radius > 0.001) {
        const angle = Math.atan2(z, x) + deltaTime * 0.5;
        result[i3] = radius * Math.cos(angle);
        result[i3 + 2] = radius * Math.sin(angle);
      }

      // Add vertical oscillation
      result[i3 + 1] += Math.sin(Date.now() * 0.001 + i * 0.1) * 0.01;
    }

    return {
      id: taskId,
      data: result,
      metadata: { method: 'javascript', particleCount }
    };
  }

  // Run performance benchmarks to determine optimal processing method
  async runPerformanceBenchmarks() {
    if (!this.capabilities.wasm || typeof self.runConcurrentCompute !== 'function') {
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
        case 'status':
          // Respond to status request
          self.postMessage({
            type: 'worker-status',
            capabilities: this.capabilities,
            isPaused: this.isPaused,
            isProcessing: this.isProcessing,
            taskQueueLength: this.taskQueue.length,
            activeTasks: this.activeTasks.size,
            performanceMetrics: this.performanceMetrics
          });
          break;
        case 'wasm-call-response':
          // Handle WASM function call response from main thread
          if (this.wasmCallbacks && this.wasmCallbacks.has(event.data.callId)) {
            const { resolve, reject } = this.wasmCallbacks.get(event.data.callId);
            this.wasmCallbacks.delete(event.data.callId);

            if (event.data.error) {
              reject(new Error(event.data.error));
            } else {
              resolve(event.data.result);
            }
          }
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
