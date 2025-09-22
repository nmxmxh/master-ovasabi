/**
 * Centralized WebGPU Manager
 * Prevents multiple WebGPU initialization attempts and manages device lifecycle
 */

interface WebGPUStatus {
  initialized: boolean;
  device: GPUDevice | null;
  adapter: GPUAdapter | null;
  capabilities: {
    webgpu: boolean;
    compute: boolean;
    storage: boolean;
  };
  error: string | null;
}

class WebGPUManager {
  private static instance: WebGPUManager;
  private status: WebGPUStatus = {
    initialized: false,
    device: null,
    adapter: null,
    capabilities: {
      webgpu: false,
      compute: false,
      storage: false
    },
    error: null
  };
  private initializationPromise: Promise<boolean> | null = null;
  private listeners: Set<(status: WebGPUStatus) => void> = new Set();

  private constructor() {}

  static getInstance(): WebGPUManager {
    if (!WebGPUManager.instance) {
      WebGPUManager.instance = new WebGPUManager();
    }
    return WebGPUManager.instance;
  }

  /**
   * Initialize WebGPU with proper error handling and singleton pattern
   */
  async initialize(): Promise<boolean> {
    // If already initialized, return current status
    if (this.status.initialized) {
      return true;
    }

    // If initialization is in progress, wait for it
    if (this.initializationPromise) {
      return this.initializationPromise;
    }

    // Start new initialization
    this.initializationPromise = this.performInitialization();
    const result = await this.initializationPromise;
    this.initializationPromise = null;

    return result;
  }

  private async performInitialization(): Promise<boolean> {
    try {
      console.log('[WebGPU-Manager] Starting centralized WebGPU initialization...');

      // Check if WebGPU is supported
      if (!('gpu' in navigator)) {
        throw new Error('WebGPU not supported in this browser');
      }

      // WebGPU works fine in workers - this check was incorrect
      // Workers can initialize WebGPU and it's actually preferred for performance

      // Request adapter
      const adapter = await navigator.gpu.requestAdapter({
        powerPreference: 'high-performance'
      });

      if (!adapter) {
        throw new Error('No WebGPU adapter available');
      }

      // Request device
      const device = await adapter.requestDevice({
        requiredLimits: {
          maxBufferSize: 4294967296 // 4GB
        }
      });

      if (!device) {
        throw new Error('Failed to get WebGPU device');
      }

      // Set up device lost handling
      device.addEventListener('uncapturederror', event => {
        console.warn('[WebGPU-Manager] Device error detected:', event);
        this.handleDeviceLost();
      });

      device.lost.then(info => {
        console.warn('[WebGPU-Manager] Device lost:', info);
        this.handleDeviceLost();
      });

      // Update status
      this.status = {
        initialized: true,
        device,
        adapter,
        capabilities: {
          webgpu: true,
          compute: true,
          storage: true
        },
        error: null
      };

      console.log('[WebGPU-Manager] WebGPU initialized successfully');
      this.notifyListeners();
      return true;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      console.error('[WebGPU-Manager] WebGPU initialization failed:', errorMessage);

      this.status = {
        initialized: false,
        device: null,
        adapter: null,
        capabilities: {
          webgpu: false,
          compute: false,
          storage: false
        },
        error: errorMessage
      };

      this.notifyListeners();
      return false;
    }
  }

  private handleDeviceLost(): void {
    console.warn('[WebGPU-Manager] Handling device lost event...');

    this.status = {
      initialized: false,
      device: null,
      adapter: null,
      capabilities: {
        webgpu: false,
        compute: false,
        storage: false
      },
      error: 'Device lost'
    };

    this.notifyListeners();
  }

  /**
   * Get current WebGPU status
   */
  getStatus(): WebGPUStatus {
    return { ...this.status };
  }

  /**
   * Get WebGPU device (only if initialized)
   */
  getDevice(): GPUDevice | null {
    return this.status.device;
  }

  /**
   * Get WebGPU adapter (only if initialized)
   */
  getAdapter(): GPUAdapter | null {
    return this.status.adapter;
  }

  /**
   * Check if WebGPU is available and initialized
   */
  isAvailable(): boolean {
    return this.status.initialized && this.status.device !== null;
  }

  /**
   * Subscribe to status changes
   */
  subscribe(listener: (status: WebGPUStatus) => void): () => void {
    this.listeners.add(listener);

    // Return unsubscribe function
    return () => {
      this.listeners.delete(listener);
    };
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => {
      try {
        listener(this.getStatus());
      } catch (error) {
        console.error('[WebGPU-Manager] Error in status listener:', error);
      }
    });
  }

  /**
   * Cleanup resources
   */
  cleanup(): void {
    if (this.status.device) {
      this.status.device.destroy();
    }

    this.status = {
      initialized: false,
      device: null,
      adapter: null,
      capabilities: {
        webgpu: false,
        compute: false,
        storage: false
      },
      error: null
    };

    this.listeners.clear();
    this.initializationPromise = null;
  }
}

export const webGPUManager = WebGPUManager.getInstance();
export default webGPUManager;
