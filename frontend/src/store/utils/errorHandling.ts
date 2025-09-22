// Error handling utilities for stores
export interface StoreError {
  id: string;
  storeName: string;
  error: Error;
  timestamp: string;
  context?: any;
  retryCount: number;
  resolved: boolean;
}

export interface ErrorHandler {
  handleError: (error: StoreError) => void;
  handleRecovery: (storeName: string) => void;
  clearErrors: (storeName?: string) => void;
  getErrors: (storeName?: string) => StoreError[];
}

class StoreErrorManager {
  private errors: Map<string, StoreError[]> = new Map();
  private maxErrorsPerStore = 10;
  private retryDelays = [1000, 2000, 5000]; // Exponential backoff

  handleError(storeName: string, error: Error, context?: any): StoreError {
    const errorId = `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

    const storeError: StoreError = {
      id: errorId,
      storeName,
      error,
      timestamp: new Date().toISOString(),
      context,
      retryCount: 0,
      resolved: false
    };

    // Store the error
    if (!this.errors.has(storeName)) {
      this.errors.set(storeName, []);
    }

    const storeErrors = this.errors.get(storeName)!;
    storeErrors.push(storeError);

    // Keep only the most recent errors
    if (storeErrors.length > this.maxErrorsPerStore) {
      storeErrors.splice(0, storeErrors.length - this.maxErrorsPerStore);
    }

    console.error(`[StoreErrorManager] Error in ${storeName}:`, {
      errorId,
      message: error.message,
      stack: error.stack,
      context
    });

    return storeError;
  }

  async attemptRecovery(storeName: string, errorId: string): Promise<boolean> {
    const storeErrors = this.errors.get(storeName);
    if (!storeErrors) return false;

    const storeError = storeErrors.find(e => e.id === errorId);
    if (!storeError || storeError.resolved) return false;

    const retryDelay =
      this.retryDelays[Math.min(storeError.retryCount, this.retryDelays.length - 1)];

    console.log(
      `[StoreErrorManager] Attempting recovery for ${storeName} (attempt ${storeError.retryCount + 1})`
    );

    return new Promise(resolve => {
      setTimeout(() => {
        try {
          // Attempt store-specific recovery
          this.performStoreRecovery(storeName);

          // Mark as resolved
          storeError.resolved = true;
          storeError.retryCount++;

          console.log(`[StoreErrorManager] Recovery successful for ${storeName}`);
          resolve(true);
        } catch (recoveryError) {
          storeError.retryCount++;
          console.error(`[StoreErrorManager] Recovery failed for ${storeName}:`, recoveryError);
          resolve(false);
        }
      }, retryDelay);
    });
  }

  private performStoreRecovery(storeName: string) {
    // Store-specific recovery logic
    switch (storeName) {
      case 'event':
        this.recoverEventStore();
        break;
      case 'connection':
        this.recoverConnectionStore();
        break;
      case 'campaign':
        this.recoverCampaignStore();
        break;
      case 'metadata':
        this.recoverMetadataStore();
        break;
      default:
        console.warn(`[StoreErrorManager] Unknown store: ${storeName}`);
    }
  }

  private recoverEventStore() {
    // Clear queued events and reset state
    console.log('[StoreErrorManager] Recovering event store');
    // Implementation would depend on the specific store structure
  }

  private recoverConnectionStore() {
    // Reset connection state and attempt reconnection
    console.log('[StoreErrorManager] Recovering connection store');
    // Implementation would depend on the specific store structure
  }

  private recoverCampaignStore() {
    // Clear campaign cache and reload from server
    console.log('[StoreErrorManager] Recovering campaign store');
    // Implementation would depend on the specific store structure
  }

  private recoverMetadataStore() {
    // Reinitialize metadata from WASM
    console.log('[StoreErrorManager] Recovering metadata store');
    // Implementation would depend on the specific store structure
  }

  clearErrors(storeName?: string) {
    if (storeName) {
      this.errors.delete(storeName);
    } else {
      this.errors.clear();
    }
  }

  getErrors(storeName?: string): StoreError[] {
    if (storeName) {
      return this.errors.get(storeName) || [];
    }

    const allErrors: StoreError[] = [];
    this.errors.forEach(storeErrors => {
      allErrors.push(...storeErrors);
    });

    return allErrors.sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );
  }

  getErrorStats() {
    const stats: Record<string, { total: number; unresolved: number; resolved: number }> = {};

    this.errors.forEach((storeErrors, storeName) => {
      const unresolved = storeErrors.filter(e => !e.resolved).length;
      const resolved = storeErrors.filter(e => e.resolved).length;

      stats[storeName] = {
        total: storeErrors.length,
        unresolved,
        resolved
      };
    });

    return stats;
  }
}

// Export singleton instance
export const storeErrorManager = new StoreErrorManager();

// Error boundary utilities
export const withErrorHandling = <T extends any>(
  storeName: string,
  operation: () => T,
  context?: any
): T | null => {
  try {
    return operation();
  } catch (error) {
    if (error instanceof Error) {
      storeErrorManager.handleError(storeName, error, context);
    }
    return null;
  }
};

export const withAsyncErrorHandling = async <T extends any>(
  storeName: string,
  operation: () => Promise<T>,
  context?: any
): Promise<T | null> => {
  try {
    return await operation();
  } catch (error) {
    if (error instanceof Error) {
      storeErrorManager.handleError(storeName, error, context);
    }
    return null;
  }
};

// Store health monitoring
export const createStoreHealthMonitor = (storeName: string) => {
  let lastHealthCheck = Date.now();
  const healthCheckInterval = 30000; // 30 seconds

  return {
    checkHealth: () => {
      const now = Date.now();
      if (now - lastHealthCheck > healthCheckInterval) {
        lastHealthCheck = now;

        const errors = storeErrorManager.getErrors(storeName);
        const unresolvedErrors = errors.filter(e => !e.resolved);

        if (unresolvedErrors.length > 0) {
          console.warn(
            `[StoreHealthMonitor] ${storeName} has ${unresolvedErrors.length} unresolved errors`
          );
        }

        return {
          healthy: unresolvedErrors.length === 0,
          errorCount: unresolvedErrors.length,
          lastCheck: new Date(lastHealthCheck).toISOString()
        };
      }

      return { healthy: true, errorCount: 0, lastCheck: new Date(lastHealthCheck).toISOString() };
    }
  };
};
