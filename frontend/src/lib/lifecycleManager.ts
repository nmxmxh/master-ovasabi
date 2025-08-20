import React from 'react';

type CleanupHook = () => void | Promise<void>;

class FrontendLifecycleManager {
  private hooks: CleanupHook[] = [];
  private isShuttingDown = false;
  private shutdownPromise: Promise<void> | null = null;
  private hiddenTimeout: NodeJS.Timeout | null = null;

  constructor() {
    this.setupGlobalHandlers();
  }

  registerCleanup(cleanup: CleanupHook): void {
    this.hooks.push(cleanup);
  }

  async shutdown(forceFull: boolean = true): Promise<void> {
    if (this.isShuttingDown) return this.shutdownPromise || Promise.resolve();
    this.isShuttingDown = true;
    this.shutdownPromise = (async () => {
      if (this.hiddenTimeout) {
        clearTimeout(this.hiddenTimeout);
        this.hiddenTimeout = null;
      }
      for (const hook of this.hooks) {
        try {
          const result = hook();
          if (result instanceof Promise) await result;
        } catch (err) {
          // Log error if needed
        }
      }
      // Only perform WASM sync cleanup and clear hooks for reload/shutdown events
      if (forceFull) {
        if (typeof window !== 'undefined' && typeof (window as any).go_syncCleanup === 'function') {
          try {
            (window as any).go_syncCleanup();
          } catch {}
        }
        this.hooks = [];
      }
    })();
    return this.shutdownPromise;
  }

  forceReloadAfterShutdown(): void {
    this.shutdown().then(() => {
      if (typeof window !== 'undefined') window.location.reload();
    });
  }

  private setupGlobalHandlers(): void {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => this.shutdown(true));
      window.addEventListener('pagehide', () => this.shutdown(true));
      if (typeof document !== 'undefined') {
        document.addEventListener('visibilitychange', () => {
          if (document.hidden) {
            this.hiddenTimeout = setTimeout(() => this.shutdown(false), 30000);
          } else if (this.hiddenTimeout) {
            clearTimeout(this.hiddenTimeout);
            this.hiddenTimeout = null;
          }
        });
      }
    }
  }

  forceShutdown(): void {
    this.isShuttingDown = true;
    this.shutdown();
  }
}

export const frontendLifecycleManager = new FrontendLifecycleManager();

if (typeof window !== 'undefined') {
  (window as any).frontendLifecycleManager = frontendLifecycleManager;
  (window as any).forceReloadAfterShutdown = () =>
    frontendLifecycleManager.forceReloadAfterShutdown();
  (window as any).forceShutdown = () => frontendLifecycleManager.forceShutdown();
}

export function useLifecycleCleanup(
  componentName: string,
  cleanupFunctions: Array<CleanupHook>
): void {
  React.useEffect(() => {
    cleanupFunctions.forEach(fn => frontendLifecycleManager.registerCleanup(fn));
    return () => {};
  }, [componentName]);
}

// Example: register WASM cleanup
frontendLifecycleManager.registerCleanup(() => {
  if (
    typeof window !== 'undefined' &&
    (window as any).wasmGPU &&
    typeof (window as any).wasmGPU.cleanup === 'function'
  ) {
    (window as any).wasmGPU.cleanup();
  }
});

// Example: notify backend on shutdown
frontendLifecycleManager.registerCleanup(() => {
  if (typeof window !== 'undefined' && (window as any).sendWasmMessage) {
    (window as any).sendWasmMessage({
      type: 'frontend:lifecycle:v1:shutdown',
      payload: { timestamp: new Date().toISOString(), reason: 'frontend_cleanup' },
      metadata: { source: 'frontend-lifecycle-manager', urgency: 'medium' }
    });
  }
});
