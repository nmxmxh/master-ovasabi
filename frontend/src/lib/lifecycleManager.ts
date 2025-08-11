// Frontend Lifecycle Manager - Integrates with backend SimpleLifecycleManager
// Coordinates React component cleanup, WASM resource management, and WebSocket connections

import { wasmGPU } from './wasmBridge';

export interface LifecycleHook {
  name: string;
  cleanup: () => void | Promise<void>;
  priority: number; // Higher number = executed first during cleanup
}

export class FrontendLifecycleManager {
  private hooks: Map<string, LifecycleHook> = new Map();
  private isShuttingDown = false;
  private shutdownPromise: Promise<void> | null = null;
  private hiddenTimeout: NodeJS.Timeout | null = null;

  constructor() {
    this.setupGlobalHandlers();
  }
  /**
   * Shutdown and reload the page after cleanup completes
   */
  forceReloadAfterShutdown(): void {
    this.shutdown().then(() => {
      if (typeof window !== 'undefined') {
        window.location.reload();
      }
    });
  }

  /**
   * Register a cleanup hook with the lifecycle manager
   */
  registerCleanup(
    name: string,
    cleanup: () => void | Promise<void>,
    priority: number = 0
  ): () => void {
    console.log(`[Frontend Lifecycle] Registering cleanup hook: ${name} (priority: ${priority})`);

    this.hooks.set(name, { name, cleanup, priority });

    // Return unregister function
    return () => {
      console.log(`[Frontend Lifecycle] Unregistering cleanup hook: ${name}`);
      this.hooks.delete(name);
    };
  }

  /**
   * Execute all cleanup hooks in priority order
   */
  async shutdown(): Promise<void> {
    if (this.isShuttingDown) {
      console.log('[Frontend Lifecycle] Shutdown already in progress, waiting for completion...');
      return this.shutdownPromise || Promise.resolve();
    }

    this.isShuttingDown = true;
    console.log('[Frontend Lifecycle] Starting frontend shutdown sequence...');

    this.shutdownPromise = this.executeShutdown();
    return this.shutdownPromise;
  }

  private async executeShutdown(): Promise<void> {
    // Clear any pending hidden timeout since we're shutting down
    if (this.hiddenTimeout) {
      clearTimeout(this.hiddenTimeout);
      this.hiddenTimeout = null;
    }

    // Sort hooks by priority (highest first)
    const sortedHooks = Array.from(this.hooks.values()).sort((a, b) => b.priority - a.priority);

    console.log(
      `[Frontend Lifecycle] Executing ${sortedHooks.length} cleanup hooks in priority order:`
    );
    sortedHooks.forEach(hook => {
      console.log(`  - ${hook.name} (priority: ${hook.priority})`);
    });

    const results: Array<{ name: string; success: boolean; error?: Error }> = [];
    const timeoutMs = 2000;
    let timedOut = false;
    const timeoutPromise = new Promise<void>(resolve => {
      setTimeout(() => {
        timedOut = true;
        console.warn(
          '[Frontend Lifecycle] Global shutdown timeout reached (2s), forcing completion.'
        );
        resolve();
      }, timeoutMs);
    });

    // Run all hooks with timeout
    await Promise.race([
      (async () => {
        for (const hook of sortedHooks) {
          if (timedOut) break;
          try {
            console.log(`[Frontend Lifecycle] Executing cleanup: ${hook.name}`);
            const result = hook.cleanup();
            if (result instanceof Promise) {
              await Promise.race([result, timeoutPromise]);
            }
            results.push({ name: hook.name, success: true });
            console.log(`[Frontend Lifecycle] ✅ Cleanup completed: ${hook.name}`);
          } catch (error) {
            const err = error instanceof Error ? error : new Error(String(error));
            results.push({ name: hook.name, success: false, error: err });
            console.error(`[Frontend Lifecycle] ❌ Cleanup failed: ${hook.name}`, err);
          }
        }
      })(),
      timeoutPromise
    ]);

    // --- Synchronous WASM cleanup after all JS hooks ---
    if (
      !timedOut &&
      typeof window !== 'undefined' &&
      typeof (window as any).go_syncCleanup === 'function'
    ) {
      try {
        console.log('[Frontend Lifecycle] Calling synchronous WASM cleanup (go_syncCleanup)...');
        (window as any).go_syncCleanup();
        console.log('[Frontend Lifecycle] Synchronous WASM cleanup complete.');
      } catch (err) {
        console.warn('[Frontend Lifecycle] Synchronous WASM cleanup failed:', err);
      }
    }

    // Log summary
    const successful = results.filter(r => r.success).length;
    const failed = results.filter(r => !r.success).length;

    console.log(
      `[Frontend Lifecycle] Shutdown complete: ${successful} successful, ${failed} failed`
    );

    if (failed > 0) {
      console.error(
        '[Frontend Lifecycle] Failed cleanups:',
        results.filter(r => !r.success)
      );
    }

    this.hooks.clear();
  }

  /**
   * Setup global event handlers for cleanup
   */
  private setupGlobalHandlers(): void {
    // Page unload cleanup
    const handleBeforeUnload = async (_event: BeforeUnloadEvent) => {
      if (this.hooks.size > 0) {
        if (typeof window !== 'undefined') {
          (window as any).isPageUnloading = true;
        }
        console.log('[Frontend Lifecycle] Page unload detected, initiating cleanup...');

        // For critical cleanups, we can try to execute them synchronously
        // But most cleanup should happen in the background
        this.shutdown();

        // Note: Modern browsers severely limit what you can do in beforeunload
        // so we mainly rely on visibilitychange for cleanup
      }
    };

    // Page visibility change cleanup (more reliable than beforeunload)
    const handleVisibilityChange = () => {
      if (document.hidden && this.hooks.size > 0) {
        console.log('[Frontend Lifecycle] Page hidden, pausing operations (not shutting down)...');
        // Don't shutdown completely on visibility change - just pause operations
        // Full shutdown should only happen on actual page unload
        this.pauseOperations();

        // Set a timeout to shutdown if hidden for too long (indicates page close, not tab switch)
        this.hiddenTimeout = setTimeout(() => {
          console.log(
            '[Frontend Lifecycle] Page hidden for extended period, initiating shutdown...'
          );
          this.shutdown();
        }, 30000); // 30 seconds timeout
      } else if (!document.hidden) {
        console.log('[Frontend Lifecycle] Page visible, resuming operations...');

        // Clear the shutdown timeout since page is visible again
        if (this.hiddenTimeout) {
          clearTimeout(this.hiddenTimeout);
          this.hiddenTimeout = null;
        }

        this.resumeOperations();
      }
    };

    // Pagehide event (covers more cases than beforeunload)
    const handlePageHide = (_event: PageTransitionEvent) => {
      console.log('[Frontend Lifecycle] Page hide detected, initiating cleanup...');
      this.shutdown();
    };

    // Register event listeners
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', handleBeforeUnload);
      window.addEventListener('pagehide', handlePageHide);

      if (typeof document !== 'undefined') {
        document.addEventListener('visibilitychange', handleVisibilityChange);
      }
    }

    // Store cleanup functions for potential later removal
    this.registerCleanup(
      'global-event-handlers',
      () => {
        if (typeof window !== 'undefined') {
          window.removeEventListener('beforeunload', handleBeforeUnload);
          window.removeEventListener('pagehide', handlePageHide);
        }

        if (typeof document !== 'undefined') {
          document.removeEventListener('visibilitychange', handleVisibilityChange);
        }
      },
      1000
    ); // High priority to remove handlers last
  }

  /**
   * Pause operations without shutting down (for visibility changes)
   */
  pauseOperations(): void {
    console.log('[Frontend Lifecycle] Pausing operations...');

    // Pause WASM workers without terminating them
    if (typeof window !== 'undefined' && (window as any).wasmGPU) {
      try {
        (window as any).wasmGPU.pauseWorkers();
      } catch (error) {
        console.warn('[Frontend Lifecycle] Failed to pause WASM workers:', error);
      }
    }
  }

  /**
   * Resume operations after pause
   */
  resumeOperations(): void {
    console.log('[Frontend Lifecycle] Resuming operations...');

    // Resume WASM workers
    if (typeof window !== 'undefined' && (window as any).wasmGPU) {
      try {
        (window as any).wasmGPU.resumeWorkers();
      } catch (error) {
        console.warn('[Frontend Lifecycle] Failed to resume WASM workers:', error);
      }
    }

    // --- Robustly re-attach WASM event listeners ---
    // Use correct global types for custom properties
    // Re-attach WASM message listener
    if (typeof (window as any).subscribeToWasmMessages === 'function') {
      if (!(window as any).__wasmListenerAttached) {
        (window as any).__wasmListenerAttached = true;
        (window as any).subscribeToWasmMessages((msg: any) => {
          if (typeof (window as any).useGlobalStore === 'function') {
            try {
              (window as any).useGlobalStore.getState().handleWasmMessage?.(msg);
            } catch (err) {
              console.warn('[Frontend Lifecycle] Error forwarding WASM message to store:', err);
            }
          }
        });
        console.log('[Frontend Lifecycle] WASM event listener re-attached');
      }
    }

    // Re-attach media streaming state listener
    if (typeof (window as any).subscribeToMediaStreamingState === 'function') {
      if (!(window as any).__mediaStreamingListenerAttached) {
        (window as any).__mediaStreamingListenerAttached = true;
        (window as any).subscribeToMediaStreamingState((state: string) => {
          if (typeof (window as any).useGlobalStore === 'function') {
            try {
              (window as any).useGlobalStore
                .getState()
                .setMediaStreamingState?.({ connected: state === 'connected' });
            } catch (err) {
              console.warn(
                '[Frontend Lifecycle] Error forwarding media streaming state to store:',
                err
              );
            }
          }
        });
        console.log('[Frontend Lifecycle] Media streaming state listener re-attached');
      }
    }

    // Update store state after resume
    if (typeof (window as any).useGlobalStore === 'function') {
      try {
        (window as any).useGlobalStore.getState().setConnectionState?.({ wasmReady: true });
      } catch (err) {
        console.warn(
          '[Frontend Lifecycle] Error updating store connection state after resume:',
          err
        );
      }
    }
  }

  /**
   * Force shutdown immediately (for emergency situations)
   */
  forceShutdown(): void {
    console.warn('[Frontend Lifecycle] Force shutdown initiated...');

    // Clear any timeouts
    if (this.hiddenTimeout) {
      clearTimeout(this.hiddenTimeout);
      this.hiddenTimeout = null;
    }

    // Force immediate cleanup without waiting
    this.isShuttingDown = true;
    this.executeShutdown().catch(error => {
      console.error('[Frontend Lifecycle] Force shutdown failed:', error);
    });
  }

  /**
   * Register common React component cleanup patterns
   */
  registerReactComponent(
    componentName: string,
    cleanupFunctions: Array<() => void | Promise<void>>,
    priority: number = 100
  ): () => void {
    const cleanup = async () => {
      console.log(`[Frontend Lifecycle] Cleaning up React component: ${componentName}`);

      for (const fn of cleanupFunctions) {
        try {
          const result = fn();
          if (result instanceof Promise) {
            await result;
          }
        } catch (error) {
          console.error(`[Frontend Lifecycle] Error in ${componentName} cleanup:`, error);
        }
      }
    };

    return this.registerCleanup(`react-component-${componentName}`, cleanup, priority);
  }

  /**
   * Get current status of lifecycle manager
   */
  getStatus(): {
    registeredHooks: number;
    isShuttingDown: boolean;
    hookNames: string[];
  } {
    return {
      registeredHooks: this.hooks.size,
      isShuttingDown: this.isShuttingDown,
      hookNames: Array.from(this.hooks.keys())
    };
  }
}

// Singleton instance for global use

export const frontendLifecycleManager = new FrontendLifecycleManager();

// --- Global WebSocket tracking ---
if (typeof window !== 'undefined') {
  if (!Array.isArray((window as any).allWebSockets)) {
    (window as any).allWebSockets = [];
  }

  // Expose forceReloadAfterShutdown globally
  (window as any).forceReloadAfterShutdown = () => {
    frontendLifecycleManager.forceReloadAfterShutdown();
  };
}

// Add high-priority synchronous cleanup for workers, WASM, and all WebSockets
frontendLifecycleManager.registerCleanup(
  'critical-sync-cleanup',
  () => {
    console.log('[Critical Cleanup] Running on unload/pagehide');
    try {
      if (typeof window !== 'undefined') {
        // Set shutdown flag to prevent reconnect attempts
        (window as any).isShuttingDown = true;

        // Terminate compute worker(s)
        if (
          (window as any).computeWorker &&
          typeof (window as any).computeWorker.terminate === 'function'
        ) {
          (window as any).computeWorker.terminate();
        }
        if (Array.isArray((window as any).computeWorkers)) {
          (window as any).computeWorkers.forEach((w: Worker) => {
            if (w && typeof w.terminate === 'function') w.terminate();
          });
        }

        // WASM cleanup
        if ((window as any).wasmGPU && typeof (window as any).wasmGPU.cleanup === 'function') {
          (window as any).wasmGPU.cleanup();
        }

        // --- Close all tracked WebSockets ---
        if (Array.isArray((window as any).allWebSockets)) {
          (window as any).allWebSockets.forEach((ws: WebSocket, idx: number) => {
            if (
              ws &&
              (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)
            ) {
              console.log(`[Critical Cleanup] Closing WebSocket #${idx}`);
              ws.close(1000, 'Page unload');
            }
          });
        }

        // --- Media streaming client cleanup ---
        // If managed in WASM, trigger shutdown via JS API
        if (
          typeof (window as any).mediaStreaming !== 'undefined' &&
          typeof (window as any).mediaStreaming.shutdown === 'function'
        ) {
          console.log('[Critical Cleanup] Shutting down WASM media streaming client via JS API');
          (window as any).mediaStreaming.shutdown();
        }

        // If media streaming clients are JS objects, push them to allWebSockets or add a similar array (e.g., window.allMediaClients)
        // If managed in WASM, ensure a JS-accessible cleanup method is called here
        // Example:
        // if (Array.isArray((window as any).allMediaClients)) {
        //   (window as any).allMediaClients.forEach((client: any) => {
        //     if (typeof client.cleanup === 'function') client.cleanup();
        //   });
        // }
      }
    } catch (err) {
      console.warn('Critical sync cleanup error:', err);
    }
  },
  9999 // Highest priority
);

// Expose globally for coordination with other systems
if (typeof window !== 'undefined') {
  (window as any).frontendLifecycleManager = frontendLifecycleManager;
}

/**
 * React hook for component lifecycle management
 */
export function useLifecycleCleanup(
  componentName: string,
  cleanupFunctions: Array<() => void | Promise<void>>,
  priority: number = 100
): void {
  // Use React's useEffect to register/unregister cleanup
  React.useEffect(() => {
    const unregister = frontendLifecycleManager.registerReactComponent(
      componentName,
      cleanupFunctions,
      priority
    );

    // Return cleanup function that unregisters the hook
    return unregister;
  }, [componentName, priority]); // Note: cleanupFunctions intentionally not in deps to avoid re-registration
}

/**
 * High-level cleanup registration for common patterns
 */
export const LifecycleUtils = {
  /**
   * Register WASM resource cleanup
   */
  registerWasmCleanup: (priority: number = 900) => {
    return frontendLifecycleManager.registerCleanup(
      'wasm-gpu-bridge',
      () => {
        console.log('[Frontend Lifecycle] Cleaning up WASM GPU Bridge...');
        wasmGPU.cleanup();
      },
      priority
    );
  },

  /**
   * Register WebSocket cleanup
   */
  registerWebSocketCleanup: (ws: WebSocket, name: string = 'websocket', priority: number = 800) => {
    return frontendLifecycleManager.registerCleanup(
      `websocket-${name}`,
      () => {
        if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
          console.log(`[Frontend Lifecycle] Closing WebSocket: ${name}`);
          ws.close(1000, 'Frontend cleanup');
        }
      },
      priority
    );
  },

  /**
   * Register interval/timeout cleanup
   */
  registerTimerCleanup: (timerId: number, type: 'interval' | 'timeout', name: string = 'timer') => {
    return frontendLifecycleManager.registerCleanup(
      `timer-${name}`,
      () => {
        console.log(`[Frontend Lifecycle] Clearing ${type}: ${name}`);
        if (type === 'interval') {
          clearInterval(timerId);
        } else {
          clearTimeout(timerId);
        }
      },
      300
    );
  },

  /**
   * Register Three.js resource cleanup
   */
  registerThreeJSCleanup: (
    renderer: any,
    scene: any,
    _camera: any,
    name: string = 'threejs',
    priority: number = 700
  ) => {
    return frontendLifecycleManager.registerCleanup(
      `threejs-${name}`,
      () => {
        console.log(`[Frontend Lifecycle] Cleaning up Three.js resources: ${name}`);

        // Dispose of geometries and materials in scene
        if (scene) {
          scene.traverse((object: any) => {
            if (object.geometry) {
              object.geometry.dispose();
            }
            if (object.material) {
              if (Array.isArray(object.material)) {
                object.material.forEach((material: any) => material.dispose());
              } else {
                object.material.dispose();
              }
            }
          });
        }

        // Dispose of renderer
        if (renderer && typeof renderer.dispose === 'function') {
          renderer.dispose();
        }

        // Clear render targets
        if (renderer && renderer.getRenderTarget) {
          const renderTarget = renderer.getRenderTarget();
          if (renderTarget) {
            renderTarget.dispose();
          }
        }
      },
      priority
    );
  },

  /**
   * Register event listener cleanup
   */
  registerEventListenerCleanup: (
    element: EventTarget,
    eventType: string,
    listener: EventListener,
    name: string = 'event-listener'
  ) => {
    return frontendLifecycleManager.registerCleanup(
      `event-listener-${name}`,
      () => {
        console.log(`[Frontend Lifecycle] Removing event listener: ${eventType} on ${name}`);
        element.removeEventListener(eventType, listener);
      },
      200
    );
  }
};

/**
 * Coordinate with backend lifecycle system
 */
export const BackendCoordination = {
  /**
   * Notify backend about frontend shutdown
   */
  notifyBackendShutdown: async () => {
    try {
      // If WebSocket is available, send shutdown notification
      if (typeof window !== 'undefined' && (window as any).sendWasmMessage) {
        console.log('[Frontend Lifecycle] Notifying backend of frontend shutdown...');

        (window as any).sendWasmMessage({
          type: 'frontend:lifecycle:v1:shutdown',
          payload: {
            timestamp: new Date().toISOString(),
            reason: 'frontend_cleanup'
          },
          metadata: {
            source: 'frontend-lifecycle-manager',
            urgency: 'medium'
          }
        });
      }
    } catch (error) {
      console.warn('[Frontend Lifecycle] Failed to request backend shutdown:', error);
    }
  }
};

// Register backend coordination as high-priority cleanup
frontendLifecycleManager.registerCleanup(
  'backend-coordination',
  async () => {
    await BackendCoordination.notifyBackendShutdown();
  },
  950
);

// Automatically register WASM cleanup
LifecycleUtils.registerWasmCleanup(980);

console.log('[Frontend Lifecycle] Frontend Lifecycle Manager initialized');

// Export React import for the hook
import React from 'react';

// Expose force shutdown globally for emergency use
if (typeof window !== 'undefined') {
  (window as any).forceShutdown = () => {
    console.warn('[Frontend Lifecycle] Emergency shutdown triggered by user');
    frontendLifecycleManager.forceShutdown();
  };
}
