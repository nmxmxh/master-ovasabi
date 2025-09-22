/**
 * State Synchronization Manager - Phase 4
 * Coordinates state across all layers: WASM Memory, IndexedDB, Service Worker, and Browser Storage
 */

import {
  indexedDBManager,
  type ComputeStateRecord,
  type UserSessionRecord
} from './indexedDBManager';
// import { stateManager, type UserState } from './stateManager'; // Unused import
import type { UserState } from './stateManager';

export interface SyncStatus {
  wasm: boolean;
  indexedDB: boolean;
  serviceWorker: boolean;
  localStorage: boolean;
  sessionStorage: boolean;
  lastSync: number;
  conflicts: number;
}

export interface SyncConflict {
  layer: string;
  key: string;
  localValue: any;
  remoteValue: any;
  timestamp: number;
  resolution: 'local' | 'remote' | 'merge' | 'pending';
}

export class StateSyncManager {
  private syncStatus: SyncStatus = {
    wasm: false,
    indexedDB: false,
    serviceWorker: false,
    localStorage: false,
    sessionStorage: false,
    lastSync: 0,
    conflicts: 0
  };

  private conflicts: SyncConflict[] = [];
  private syncInterval: number | null = null;
  private isInitialized = false;

  constructor() {
    this.setupEventListeners();
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      // Initialize all storage layers
      await Promise.all([
        this.initializeWASM(),
        this.initializeIndexedDB(),
        this.initializeServiceWorker(),
        this.initializeBrowserStorage()
      ]);

      // Perform initial sync
      await this.performFullSync();

      // Start periodic sync
      this.startPeriodicSync();

      this.isInitialized = true;
      console.log('[StateSyncManager] Initialized successfully');
    } catch (error) {
      console.error('[StateSyncManager] Initialization failed:', error);
      throw error;
    }
  }

  private async initializeWASM(): Promise<void> {
    try {
      if (typeof window !== 'undefined' && (window as any).initializeState) {
        await (window as any).initializeState();
        this.syncStatus.wasm = true;
        console.log('[StateSyncManager] WASM initialized');
      }
    } catch (error) {
      console.warn('[StateSyncManager] WASM initialization failed:', error);
    }
  }

  private async initializeIndexedDB(): Promise<void> {
    try {
      await indexedDBManager.initialize();
      this.syncStatus.indexedDB = true;
      console.log('[StateSyncManager] IndexedDB initialized');
    } catch (error) {
      console.warn('[StateSyncManager] IndexedDB initialization failed:', error);
    }
  }

  private async initializeServiceWorker(): Promise<void> {
    try {
      if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
        this.syncStatus.serviceWorker = true;
        console.log('[StateSyncManager] Service Worker available');
      }
    } catch (error) {
      console.warn('[StateSyncManager] Service Worker check failed:', error);
    }
  }

  private async initializeBrowserStorage(): Promise<void> {
    try {
      this.syncStatus.localStorage = typeof Storage !== 'undefined' && !!localStorage;
      this.syncStatus.sessionStorage = typeof Storage !== 'undefined' && !!sessionStorage;
      console.log('[StateSyncManager] Browser storage initialized');
    } catch (error) {
      console.warn('[StateSyncManager] Browser storage initialization failed:', error);
    }
  }

  private setupEventListeners(): void {
    // Listen for WASM state changes
    if (typeof window !== 'undefined') {
      window.addEventListener('wasmReady', () => {
        this.syncStatus.wasm = true;
        this.performFullSync();
      });

      // Listen for user ID changes
      (window as any).onUserIDChanged = (newUserId: string) => {
        this.handleUserIDChange(newUserId);
      };
    }

    // Listen for storage changes
    window.addEventListener('storage', event => {
      if (event.key?.includes('user_state') || event.key?.includes('temp_user_state')) {
        this.handleStorageChange(event);
      }
    });

    // Listen for online/offline changes
    window.addEventListener('online', () => {
      this.performFullSync();
    });

    window.addEventListener('offline', () => {
      this.handleOfflineMode();
    });
  }

  private async performFullSync(): Promise<void> {
    try {
      console.log('[StateSyncManager] Performing full sync...');

      // Get current state from all layers
      const [wasmState, indexedDBState, localStorageState, sessionStorageState] = await Promise.all(
        [
          this.getWASMState(),
          this.getIndexedDBState(),
          this.getLocalStorageState(),
          this.getSessionStorageState()
        ]
      );

      // Detect conflicts
      const conflicts = this.detectConflicts({
        wasm: wasmState,
        indexedDB: indexedDBState,
        localStorage: localStorageState,
        sessionStorage: sessionStorageState
      });

      // Resolve conflicts
      if (conflicts.length > 0) {
        await this.resolveConflicts(conflicts);
      }

      // Sync to all layers
      await this.syncToAllLayers(wasmState);

      this.syncStatus.lastSync = Date.now();
      this.syncStatus.conflicts = conflicts.length;

      console.log('[StateSyncManager] Full sync completed');
    } catch (error) {
      console.error('[StateSyncManager] Full sync failed:', error);
    }
  }

  private async getWASMState(): Promise<UserState | null> {
    try {
      if (typeof window !== 'undefined' && (window as any).getState) {
        const state = await (window as any).getState();
        return state as UserState;
      }
    } catch (error) {
      console.warn('[StateSyncManager] Failed to get WASM state:', error);
    }
    return null;
  }

  private async getIndexedDBState(): Promise<UserSessionRecord | null> {
    try {
      if (this.syncStatus.indexedDB) {
        // Get the most recent user session
        const sessions = await indexedDBManager.getUserSessionsByType('any', 1);
        return sessions[0] || null;
      }
    } catch (error) {
      console.warn('[StateSyncManager] Failed to get IndexedDB state:', error);
    }
    return null;
  }

  private async getLocalStorageState(): Promise<UserState | null> {
    try {
      if (this.syncStatus.localStorage) {
        const state = localStorage.getItem('persistent_user_state');
        return state ? JSON.parse(state) : null;
      }
    } catch (error) {
      console.warn('[StateSyncManager] Failed to get localStorage state:', error);
    }
    return null;
  }

  private async getSessionStorageState(): Promise<UserState | null> {
    try {
      if (this.syncStatus.sessionStorage) {
        const state = sessionStorage.getItem('temp_user_state');
        return state ? JSON.parse(state) : null;
      }
    } catch (error) {
      console.warn('[StateSyncManager] Failed to get sessionStorage state:', error);
    }
    return null;
  }

  private detectConflicts(states: {
    wasm: UserState | null;
    indexedDB: UserSessionRecord | null;
    localStorage: UserState | null;
    sessionStorage: UserState | null;
  }): SyncConflict[] {
    const conflicts: SyncConflict[] = [];
    const layers = Object.entries(states).filter(([_, state]) => state !== null);

    for (let i = 0; i < layers.length; i++) {
      for (let j = i + 1; j < layers.length; j++) {
        const [layer1, state1] = layers[i];
        const [layer2, state2] = layers[j];

        // Compare user IDs
        if (
          state1 &&
          state2 &&
          'userId' in state1 &&
          'userId' in state2 &&
          state1.userId !== state2.userId
        ) {
          conflicts.push({
            layer: `${layer1} vs ${layer2}`,
            key: 'userId',
            localValue: state1.userId,
            remoteValue: state2.userId,
            timestamp: Date.now(),
            resolution: 'pending'
          });
        }

        // Compare session IDs
        if (
          state1 &&
          state2 &&
          'sessionId' in state1 &&
          'sessionId' in state2 &&
          state1.sessionId !== state2.sessionId
        ) {
          conflicts.push({
            layer: `${layer1} vs ${layer2}`,
            key: 'sessionId',
            localValue: state1.sessionId,
            remoteValue: state2.sessionId,
            timestamp: Date.now(),
            resolution: 'pending'
          });
        }
      }
    }

    return conflicts;
  }

  private async resolveConflicts(conflicts: SyncConflict[]): Promise<void> {
    console.log(`[StateSyncManager] Resolving ${conflicts.length} conflicts...`);

    for (const conflict of conflicts) {
      // Simple resolution strategy: prefer WASM > IndexedDB > localStorage > sessionStorage
      const priority = ['wasm', 'indexedDB', 'localStorage', 'sessionStorage'];
      const layer1Priority = priority.indexOf(conflict.layer.split(' vs ')[0]);
      const layer2Priority = priority.indexOf(conflict.layer.split(' vs ')[1]);

      if (layer1Priority < layer2Priority) {
        conflict.resolution = 'local';
      } else {
        conflict.resolution = 'remote';
      }

      this.conflicts.push(conflict);
    }
  }

  private async syncToAllLayers(primaryState: UserState | null): Promise<void> {
    if (!primaryState) return;

    const syncPromises: Promise<void>[] = [];

    // Sync to WASM
    if (this.syncStatus.wasm && typeof window !== 'undefined' && (window as any).updateState) {
      syncPromises.push((window as any).updateState(JSON.stringify(primaryState)));
    }

    // Sync to IndexedDB
    if (this.syncStatus.indexedDB) {
      const userSession: UserSessionRecord = {
        userId: primaryState.userId,
        sessionId: primaryState.sessionId,
        deviceId: primaryState.deviceId,
        timestamp: primaryState.timestamp,
        sessionType: primaryState.isTemporary ? 'guest' : 'authenticated',
        metadata: {},
        computeStats: {
          totalTasks: 0,
          avgProcessingTime: 0,
          peakThroughput: 0
        }
      };
      syncPromises.push(indexedDBManager.storeUserSession(userSession));
    }

    // Sync to localStorage
    if (this.syncStatus.localStorage) {
      syncPromises.push(
        new Promise<void>(resolve => {
          localStorage.setItem('persistent_user_state', JSON.stringify(primaryState));
          resolve();
        })
      );
    }

    // Sync to sessionStorage
    if (this.syncStatus.sessionStorage) {
      syncPromises.push(
        new Promise<void>(resolve => {
          sessionStorage.setItem('temp_user_state', JSON.stringify(primaryState));
          resolve();
        })
      );
    }

    await Promise.allSettled(syncPromises);
  }

  private async handleUserIDChange(newUserId: string): Promise<void> {
    console.log('[StateSyncManager] User ID changed:', newUserId);

    // Update all layers with new user ID
    const currentState = await this.getWASMState();
    if (currentState) {
      currentState.userId = newUserId;
      currentState.timestamp = Date.now();
      await this.syncToAllLayers(currentState);
    }
  }

  private async handleStorageChange(event: StorageEvent): Promise<void> {
    console.log('[StateSyncManager] Storage change detected:', event.key);

    // Trigger sync when storage changes
    setTimeout(() => {
      this.performFullSync();
    }, 100);
  }

  private async handleOfflineMode(): Promise<void> {
    console.log('[StateSyncManager] Entering offline mode');
    // In offline mode, we rely on local storage and WASM memory
    // Service worker will handle background sync when online
  }

  private startPeriodicSync(): void {
    // Sync every 30 seconds
    this.syncInterval = window.setInterval(() => {
      this.performFullSync();
    }, 30000);
  }

  // Public API methods
  async syncComputeState(computeState: ComputeStateRecord): Promise<void> {
    try {
      // Store in IndexedDB
      if (this.syncStatus.indexedDB) {
        await indexedDBManager.storeComputeState(computeState);
      }

      // Store in WASM if available
      if (
        this.syncStatus.wasm &&
        typeof window !== 'undefined' &&
        (window as any).storeComputeState
      ) {
        await (window as any).storeComputeState(JSON.stringify(computeState));
      }

      // Notify service worker for background sync
      if (this.syncStatus.serviceWorker && navigator.serviceWorker.controller) {
        navigator.serviceWorker.controller.postMessage({
          type: 'sync-compute-state',
          data: computeState
        });
      }
    } catch (error) {
      console.error('[StateSyncManager] Failed to sync compute state:', error);
    }
  }

  async getSyncStatus(): Promise<SyncStatus> {
    return { ...this.syncStatus };
  }

  async getConflicts(): Promise<SyncConflict[]> {
    return [...this.conflicts];
  }

  async clearConflicts(): Promise<void> {
    this.conflicts = [];
    this.syncStatus.conflicts = 0;
  }

  async cleanup(): Promise<void> {
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }

    if (this.syncStatus.indexedDB) {
      await indexedDBManager.close();
    }
  }
}

// Singleton instance
export const stateSyncManager = new StateSyncManager();
