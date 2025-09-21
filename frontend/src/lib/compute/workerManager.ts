/**
 * Worker Manager - Handles compute worker lifecycle and task distribution
 * Separated from WASM bridge for better maintainability
 */

export interface WorkerTask {
  id: string;
  type: 'compute' | 'benchmark' | 'shutdown';
  data?: Float32Array;
  params?: {
    deltaTime?: number;
    animationMode?: number;
    particleCount?: number;
  };
  priority?: number;
  timestamp?: number;
}

export interface WorkerResult {
  id: string;
  success: boolean;
  data?: Float32Array;
  error?: string;
  metadata?: {
    processingTime: number;
    method: string;
    workerId: string;
  };
}

export interface WorkerCapabilities {
  wasm: boolean;
  webgpu: boolean;
  concurrency: number;
  maxParticles: number;
}

export interface WorkerMetrics {
  activeWorkers: number;
  totalWorkers: number;
  queueDepth: number;
  throughput: number;
  avgLatency: number;
  peakThroughput: number;
  tasksProcessed: number;
  errorCount: number;
}

export class WorkerManager {
  private workers: Worker[] = [];
  private workerCapabilities = new Map<Worker, WorkerCapabilities>();
  private workerMetrics = new Map<Worker, WorkerMetrics>();
  private taskQueue: WorkerTask[] = [];
  private pendingTasks = new Map<string, { resolve: Function; reject: Function }>();
  private maxWorkers = 4;
  private isInitialized = false;
  private initializationPromise: Promise<void> | null = null;

  constructor() {
    this.setupErrorHandling();
  }

  private setupErrorHandling() {
    // Global error handling for worker failures
    window.addEventListener('error', event => {
      if (event.filename?.includes('compute-worker')) {
        console.error('[WorkerManager] Worker error detected:', event);
        this.handleWorkerError(event);
      }
    });
  }

  private handleWorkerError(_error: any) {
    console.warn('[WorkerManager] Handling worker error, attempting recovery...');
    // Implement worker recovery logic
    this.recoverFromWorkerFailure();
  }

  private async recoverFromWorkerFailure() {
    // Remove failed workers and create replacements
    const failedWorkers = this.workers.filter(worker => {
      try {
        worker.postMessage({ type: 'ping' });
        return false;
      } catch {
        return true;
      }
    });

    for (const worker of failedWorkers) {
      this.cleanupWorker(worker);
    }

    // Create replacement workers
    const neededWorkers = this.maxWorkers - this.workers.length;
    for (let i = 0; i < neededWorkers; i++) {
      await this.createWorker();
    }
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    if (this.initializationPromise) {
      return this.initializationPromise;
    }

    this.initializationPromise = this.performInitialization();
    return this.initializationPromise;
  }

  private async performInitialization(): Promise<void> {
    try {
      console.log('[WorkerManager] Initializing worker pool...');

      // Create initial workers
      const workerPromises = Array.from({ length: this.maxWorkers }, () => this.createWorker());
      await Promise.all(workerPromises);

      this.isInitialized = true;
      console.log(`[WorkerManager] ✅ Initialized ${this.workers.length} workers`);
    } catch (error) {
      console.error('[WorkerManager] Initialization failed:', error);
      throw error;
    }
  }

  private async createWorker(): Promise<Worker> {
    return new Promise((resolve, reject) => {
      try {
        const worker = new Worker('/workers/compute-worker.js');

        const timeout = setTimeout(() => {
          worker.terminate();
          reject(new Error('Worker setup timeout - WASM may not be available'));
        }, 20000); // Increased timeout to 20 seconds for WASM loading

        const messageHandler = (event: MessageEvent) => {
          const { type, capabilities, error } = event.data;

          if (type === 'worker-ready') {
            clearTimeout(timeout);
            worker.removeEventListener('message', messageHandler);

            this.workers.push(worker);
            this.workerCapabilities.set(worker, capabilities);
            this.workerMetrics.set(worker, {
              activeWorkers: 1,
              totalWorkers: this.workers.length,
              queueDepth: 0,
              throughput: 0,
              avgLatency: 0,
              peakThroughput: 0,
              tasksProcessed: 0,
              errorCount: 0
            });

            this.setupWorkerEventHandlers(worker);
            resolve(worker);
          } else if (type === 'worker-error') {
            clearTimeout(timeout);
            worker.removeEventListener('message', messageHandler);
            worker.terminate();
            reject(new Error(`Worker initialization failed: ${error}`));
          }
        };

        worker.addEventListener('message', messageHandler);
        worker.addEventListener('error', error => {
          clearTimeout(timeout);
          worker.removeEventListener('message', messageHandler);
          reject(error);
        });
      } catch (error) {
        reject(error);
      }
    });
  }

  private setupWorkerEventHandlers(worker: Worker) {
    worker.addEventListener('message', event => {
      this.handleWorkerMessage(worker, event.data);
    });

    worker.addEventListener('error', error => {
      console.error('[WorkerManager] Worker error:', error);
      this.handleWorkerError(error);
    });
  }

  private handleWorkerMessage(worker: Worker, data: any) {
    const { type, result, error, taskId, metrics } = data;

    switch (type) {
      case 'compute-result':
        this.handleTaskResult(taskId, result, error);
        break;
      case 'worker-metrics':
        this.updateWorkerMetrics(worker, metrics);
        break;
      case 'worker-degraded':
        this.handleWorkerDegradation(worker, data);
        break;
      case 'worker-shutdown':
        this.handleWorkerShutdown(worker);
        break;
      case 'worker-capabilities':
        this.handleWorkerCapabilities(worker, data);
        break;
      default:
        console.warn('[WorkerManager] Unknown worker message type:', type);
    }
  }

  private handleTaskResult(taskId: string, result: any, error?: string) {
    const pendingTask = this.pendingTasks.get(taskId);
    if (pendingTask) {
      this.pendingTasks.delete(taskId);
      if (error) {
        pendingTask.reject(new Error(error));
      } else {
        pendingTask.resolve(result);
      }
    }
  }

  private updateWorkerMetrics(worker: Worker, metrics: any) {
    const currentMetrics = this.workerMetrics.get(worker);
    if (currentMetrics) {
      this.workerMetrics.set(worker, {
        ...currentMetrics,
        ...metrics
      });
    }
  }

  private handleWorkerDegradation(_worker: Worker, data: any) {
    console.warn(`[WorkerManager] Worker degraded: ${data.degradationLevel}`);
    // Implement degradation handling
  }

  private handleWorkerShutdown(worker: Worker) {
    console.log('[WorkerManager] Worker shutdown detected');
    this.cleanupWorker(worker);
  }

  private cleanupWorker(worker: Worker) {
    const index = this.workers.indexOf(worker);
    if (index > -1) {
      this.workers.splice(index, 1);
    }

    this.workerCapabilities.delete(worker);
    this.workerMetrics.delete(worker);

    try {
      worker.terminate();
    } catch (error) {
      console.warn('[WorkerManager] Error terminating worker:', error);
    }
  }

  private handleWorkerCapabilities(worker: Worker, data: any) {
    console.log('[WorkerManager] Worker capabilities:', data);
    // Store worker capabilities for future reference
    if (data.capabilities) {
      this.workerCapabilities.set(worker, data.capabilities);
    }
  }

  async submitTask(task: WorkerTask): Promise<WorkerResult> {
    if (!this.isInitialized) {
      await this.initialize();
    }

    return new Promise((resolve, reject) => {
      const taskId = task.id || `task-${Date.now()}-${Math.random().toString(36).slice(2)}`;
      const taskWithId = { ...task, id: taskId };

      this.pendingTasks.set(taskId, { resolve, reject });

      // Find available worker
      const availableWorker = this.findAvailableWorker();
      if (availableWorker) {
        availableWorker.postMessage({
          type: 'compute-task',
          task: taskWithId
        });
      } else {
        // Queue task if no workers available
        this.taskQueue.push(taskWithId);
      }
    });
  }

  private findAvailableWorker(): Worker | null {
    // Simple round-robin selection
    return this.workers.length > 0 ? this.workers[0] : null;
  }

  async runBenchmark(config: { particleCount?: number } = {}): Promise<any> {
    const task: WorkerTask = {
      id: `benchmark-${Date.now()}`,
      type: 'benchmark',
      params: config
    };

    return this.submitTask(task);
  }

  getMetrics(): WorkerMetrics {
    const allMetrics = Array.from(this.workerMetrics.values());

    return {
      activeWorkers: this.workers.length,
      totalWorkers: this.maxWorkers,
      queueDepth: this.taskQueue.length,
      throughput: allMetrics.reduce((sum, m) => sum + m.throughput, 0),
      avgLatency: allMetrics.reduce((sum, m) => sum + m.avgLatency, 0) / allMetrics.length || 0,
      peakThroughput: Math.max(...allMetrics.map(m => m.peakThroughput), 0),
      tasksProcessed: allMetrics.reduce((sum, m) => sum + m.tasksProcessed, 0),
      errorCount: allMetrics.reduce((sum, m) => sum + m.errorCount, 0)
    };
  }

  pauseWorkers() {
    this.workers.forEach(worker => {
      worker.postMessage({ type: 'pause' });
    });
  }

  resumeWorkers() {
    this.workers.forEach(worker => {
      worker.postMessage({ type: 'resume' });
    });
  }

  cleanup() {
    this.workers.forEach(worker => {
      worker.postMessage({ type: 'shutdown' });
      this.cleanupWorker(worker);
    });

    this.workers = [];
    this.workerCapabilities.clear();
    this.workerMetrics.clear();
    this.pendingTasks.clear();
    this.taskQueue = [];
    this.isInitialized = false;
  }
}

// Singleton instance
export const workerManager = new WorkerManager();
