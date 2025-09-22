// Performance optimization utilities for stores
import { useCallback, useMemo } from 'react';

// Debounce utility
export const debounce = <T extends (...args: any[]) => any>(
  func: T,
  wait: number
): ((...args: Parameters<T>) => void) => {
  let timeout: NodeJS.Timeout;

  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
};

// Throttle utility
export const throttle = <T extends (...args: any[]) => any>(
  func: T,
  limit: number
): ((...args: Parameters<T>) => void) => {
  let inThrottle: boolean;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
};

// Memoization utility
export const createMemoizedSelector = <T, R>(
  selector: (state: T) => R,
  equalityFn?: (a: R, b: R) => boolean
) => {
  let lastResult: R;
  let lastState: T;

  return (state: T): R => {
    if (state !== lastState) {
      lastState = state;
      lastResult = selector(state);
    } else if (equalityFn && !equalityFn(lastResult, lastResult)) {
      lastResult = selector(state);
    }

    return lastResult;
  };
};

// Store subscription optimization
export class StoreSubscriptionManager {
  private subscriptions = new Map<string, Set<() => void>>();
  private batchUpdates = new Set<string>();
  private batchTimeout: NodeJS.Timeout | null = null;

  subscribe(storeName: string, callback: () => void) {
    if (!this.subscriptions.has(storeName)) {
      this.subscriptions.set(storeName, new Set());
    }
    this.subscriptions.get(storeName)!.add(callback);

    return () => {
      this.subscriptions.get(storeName)?.delete(callback);
    };
  }

  notify(storeName: string) {
    this.batchUpdates.add(storeName);

    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
    }

    this.batchTimeout = setTimeout(() => {
      this.flushBatchUpdates();
    }, 0);
  }

  private flushBatchUpdates() {
    this.batchUpdates.forEach(storeName => {
      const callbacks = this.subscriptions.get(storeName);
      if (callbacks) {
        callbacks.forEach(callback => {
          try {
            callback();
          } catch (error) {
            console.error(`[StoreSubscriptionManager] Error in ${storeName} callback:`, error);
          }
        });
      }
    });

    this.batchUpdates.clear();
    this.batchTimeout = null;
  }
}

// Performance monitoring
export class StorePerformanceMonitor {
  private metrics = new Map<
    string,
    {
      operationCount: number;
      totalTime: number;
      averageTime: number;
      lastOperation: number;
    }
  >();

  startTiming(operation: string): () => void {
    const startTime = performance.now();

    return () => {
      const endTime = performance.now();
      const duration = endTime - startTime;

      this.recordMetric(operation, duration);
    };
  }

  private recordMetric(operation: string, duration: number) {
    const existing = this.metrics.get(operation) || {
      operationCount: 0,
      totalTime: 0,
      averageTime: 0,
      lastOperation: 0
    };

    existing.operationCount++;
    existing.totalTime += duration;
    existing.averageTime = existing.totalTime / existing.operationCount;
    existing.lastOperation = duration;

    this.metrics.set(operation, existing);
  }

  getMetrics(operation?: string) {
    if (operation) {
      return this.metrics.get(operation);
    }

    return Object.fromEntries(this.metrics);
  }

  getSlowOperations(threshold: number = 10) {
    const slowOps: Array<{ operation: string; metrics: any }> = [];

    this.metrics.forEach((metrics, operation) => {
      if (metrics.averageTime > threshold) {
        slowOps.push({ operation, metrics });
      }
    });

    return slowOps.sort((a, b) => b.metrics.averageTime - a.metrics.averageTime);
  }

  reset() {
    this.metrics.clear();
  }
}

// Export singleton instances
export const storeSubscriptionManager = new StoreSubscriptionManager();
export const storePerformanceMonitor = new StorePerformanceMonitor();

// React hooks for performance optimization
export const useOptimizedCallback = <T extends (...args: any[]) => any>(
  callback: T,
  deps: React.DependencyList,
  options: { debounce?: number; throttle?: number } = {}
): T => {
  const { debounce: debounceMs, throttle: throttleMs } = options;

  const optimizedCallback = useCallback(callback, deps);

  if (debounceMs) {
    return useMemo(
      () => debounce(optimizedCallback, debounceMs),
      [optimizedCallback, debounceMs]
    ) as T;
  }

  if (throttleMs) {
    return useMemo(
      () => throttle(optimizedCallback, throttleMs),
      [optimizedCallback, throttleMs]
    ) as T;
  }

  return optimizedCallback;
};

export const useOptimizedMemo = <T>(
  factory: () => T,
  deps: React.DependencyList,
  equalityFn?: (a: T, b: T) => boolean
): T => {
  const memoizedFactory = useMemo(() => factory, deps);

  return useMemo(() => {
    const endTiming = storePerformanceMonitor.startTiming('useOptimizedMemo');
    const result = memoizedFactory();
    endTiming();
    return result;
  }, [memoizedFactory, equalityFn]);
};

// Store state optimization
export const optimizeStoreState = <T extends Record<string, any>>(
  state: T,
  options: {
    deepEqual?: boolean;
    excludeKeys?: string[];
    includeKeys?: string[];
  } = {}
): T => {
  const { excludeKeys = [], includeKeys = [] } = options;

  if (includeKeys.length > 0) {
    const filtered: Record<string, any> = {};
    includeKeys.forEach(key => {
      if (key in state) {
        filtered[key] = state[key];
      }
    });
    return filtered as T;
  }

  if (excludeKeys.length > 0) {
    const filtered = { ...state };
    excludeKeys.forEach(key => {
      delete filtered[key];
    });
    return filtered;
  }

  return state;
};

// Batch store updates
export class StoreBatchManager {
  private pendingUpdates = new Map<string, () => void>();
  private batchTimeout: NodeJS.Timeout | null = null;
  private batchDelay = 0; // Immediate execution by default

  setBatchDelay(delay: number) {
    this.batchDelay = delay;
  }

  addUpdate(storeName: string, update: () => void) {
    this.pendingUpdates.set(storeName, update);

    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
    }

    this.batchTimeout = setTimeout(() => {
      this.flushUpdates();
    }, this.batchDelay);
  }

  private flushUpdates() {
    const endTiming = storePerformanceMonitor.startTiming('batchUpdate');

    this.pendingUpdates.forEach((update, storeName) => {
      try {
        update();
      } catch (error) {
        console.error(`[StoreBatchManager] Error updating ${storeName}:`, error);
      }
    });

    this.pendingUpdates.clear();
    this.batchTimeout = null;

    endTiming();
  }

  forceFlush() {
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
      this.flushUpdates();
    }
  }
}

export const storeBatchManager = new StoreBatchManager();
