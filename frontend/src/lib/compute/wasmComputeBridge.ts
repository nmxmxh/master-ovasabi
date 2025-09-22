/**
 * WASM Compute Bridge - Handles WASM-specific compute operations
 * Separated from worker management for better separation of concerns
 */

import { workerManager, type WorkerTask } from './workerManager';

export interface WASMComputeOptions {
  useWorkers?: boolean;
  fallbackToJS?: boolean;
  timeout?: number;
}

export interface WASMParticleData {
  positions: Float32Array;
  velocities?: Float32Array;
  ages?: Float32Array;
  intensities?: Float32Array;
  phases?: Float32Array;
  types?: Float32Array;
  ids?: Float32Array;
}

export class WASMComputeBridge {
  private wasmReady = false;
  private wasmInitializationPromise: Promise<boolean> | null = null;
  private operationCounter = 0;

  constructor() {
    this.initializeWASM();
  }

  private async initializeWASM(): Promise<boolean> {
    if (this.wasmReady) return true;

    if (this.wasmInitializationPromise) {
      return this.wasmInitializationPromise;
    }

    this.wasmInitializationPromise = this.performWASMInitialization();
    return this.wasmInitializationPromise;
  }

  private async performWASMInitialization(): Promise<boolean> {
    try {
      // Wait for WASM functions to be available with timeout
      try {
        await this.waitForWASMFunctions();
      } catch (error) {
        this.wasmReady = false;
        return false;
      }

      // Initialize worker manager with error handling
      try {
        await workerManager.initialize();
      } catch (error) {
        // Continue without workers - we can still use direct WASM calls
      }

      this.wasmReady = true;
      return true;
    } catch (error) {
      console.error('[WASMComputeBridge] Initialization failed:', error);
      this.wasmReady = false;
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
      typeof window.runConcurrentCompute === 'function' &&
      typeof window.submitComputeTask === 'function' &&
      typeof window.runGPUCompute === 'function' &&
      typeof window.runGPUComputeWithOffset === 'function'
    );
  }

  async runParticlePhysics(
    particleData: WASMParticleData,
    deltaTime: number = 0.016667,
    options: WASMComputeOptions = {}
  ): Promise<WASMParticleData> {
    if (!this.wasmReady) {
      await this.initializeWASM();
    }

    const { useWorkers = true, fallbackToJS = true } = options;

    try {
      if (useWorkers && this.shouldUseWorkers(particleData)) {
        return this.runWithWorkers(particleData, deltaTime, 5000);
      } else {
        return this.runWithWASM(particleData, deltaTime, 5000);
      }
    } catch (error) {
      if (fallbackToJS) {
        return this.runWithJavaScript(particleData, deltaTime);
      }

      throw error;
    }
  }

  private shouldUseWorkers(particleData: WASMParticleData): boolean {
    const particleCount = particleData.positions.length / 3;
    return particleCount > 10000; // Use workers for large particle counts
  }

  private async runWithWorkers(
    particleData: WASMParticleData,
    deltaTime: number,
    _timeout: number
  ): Promise<WASMParticleData> {
    const task: WorkerTask = {
      id: `particle-physics-${++this.operationCounter}`,
      type: 'compute',
      data: particleData.positions,
      params: {
        deltaTime,
        animationMode: 1.0
      }
    };

    const result = await workerManager.submitTask(task);

    if (!result.success || !result.data) {
      throw new Error(result.error || 'Worker compute failed');
    }

    return {
      ...particleData,
      positions: result.data
    };
  }

  private async runWithWASM(
    particleData: WASMParticleData,
    deltaTime: number,
    timeout: number
  ): Promise<WASMParticleData> {
    return new Promise((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        reject(new Error('WASM compute timeout'));
      }, timeout);

      try {
        // Use WASM's runConcurrentCompute function
        window.runConcurrentCompute!(
          particleData.positions,
          deltaTime,
          1.0, // animationMode
          (result: Float32Array, _metadata: any) => {
            clearTimeout(timeoutId);
            resolve({
              ...particleData,
              positions: result
            });
          }
        );
      } catch (error) {
        clearTimeout(timeoutId);
        reject(error);
      }
    });
  }

  private runWithJavaScript(particleData: WASMParticleData, deltaTime: number): WASMParticleData {
    const positions = particleData.positions.slice();
    const particleCount = positions.length / 3;

    // Simple JavaScript particle physics simulation
    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;

      // Basic orbital motion
      const x = positions[i3];
      const z = positions[i3 + 2];
      const radius = Math.sqrt(x * x + z * z);

      if (radius > 0.001) {
        const angle = Math.atan2(z, x) + deltaTime * 0.5;
        positions[i3] = radius * Math.cos(angle);
        positions[i3 + 2] = radius * Math.sin(angle);
      }

      // Add some vertical oscillation
      positions[i3 + 1] += Math.sin(Date.now() * 0.001 + i * 0.1) * 0.01;
    }

    return {
      ...particleData,
      positions
    };
  }

  async runGPUCompute(
    inputData: Float32Array,
    operation: number,
    callback?: (result: Float32Array) => void
  ): Promise<Float32Array> {
    if (!this.wasmReady) {
      await this.initializeWASM();
    }

    return new Promise((resolve, reject) => {
      try {
        const success = window.runGPUCompute!(inputData, operation, (result: Float32Array) => {
          if (callback) callback(result);
          resolve(result);
        });

        if (!success) {
          reject(new Error('GPU compute failed to start'));
        }
      } catch (error) {
        reject(error);
      }
    });
  }

  async runGPUComputeWithOffset(
    inputData: Float32Array,
    elapsedTime: number,
    globalParticleOffset: number
  ): Promise<Float32Array> {
    if (!this.wasmReady) {
      await this.initializeWASM();
    }

    return new Promise((resolve, reject) => {
      try {
        // WASM expects Float32Array directly, not Uint8Array
        const success = window.runGPUComputeWithOffset!(
          inputData,
          elapsedTime,
          globalParticleOffset,
          (result: Float32Array) => {
            resolve(result);
          }
        );

        if (!success) {
          reject(new Error('GPU compute with offset failed to start'));
        }
      } catch (error) {
        reject(error);
      }
    });
  }

  async submitComputeTask(
    taskData: Float32Array,
    params: { deltaTime: number; animationMode: number }
  ): Promise<Float32Array> {
    if (!this.wasmReady) {
      await this.initializeWASM();
    }

    return new Promise((resolve, reject) => {
      try {
        window.submitComputeTask(
          'particle_compute', // taskType
          taskData, // data
          (result: Float32Array) => {
            resolve(result);
          },
          params // optional parameters object
        );
      } catch (error) {
        reject(error);
      }
    });
  }

  async runBenchmark(config: { particleCount?: number } = {}): Promise<any> {
    if (!this.wasmReady) {
      await this.initializeWASM();
    }

    return workerManager.runBenchmark(config);
  }

  isInitialized(): boolean {
    return this.wasmReady;
  }

  async waitForInitialization(): Promise<boolean> {
    return this.initializeWASM();
  }

  getMetrics() {
    return workerManager.getMetrics();
  }

  pauseCompute() {
    workerManager.pauseWorkers();
  }

  resumeCompute() {
    workerManager.resumeWorkers();
  }

  cleanup() {
    workerManager.cleanup();
    this.wasmReady = false;
    this.wasmInitializationPromise = null;
  }
}

// Singleton instance
export const wasmComputeBridge = new WASMComputeBridge();
