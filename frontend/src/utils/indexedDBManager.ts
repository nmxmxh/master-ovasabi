/**
 * IndexedDB Manager - Advanced browser database for complex queries and persistence
 * Phase 2 of the multi-layer state management system
 */

export interface IndexedDBConfig {
  name: string;
  version: number;
  stores: StoreConfig[];
}

export interface StoreConfig {
  name: string;
  keyPath: string;
  indexes: IndexConfig[];
}

export interface IndexConfig {
  name: string;
  keyPath: string | string[];
  unique?: boolean;
}

export interface ComputeStateRecord {
  id: string;
  type: string;
  data: Float32Array;
  params: Record<string, number>;
  timestamp: number;
  processingTime: number;
  memoryUsage: number;
  particleCount: number;
  performance: {
    fps: number;
    throughput: number;
    latency: number;
  };
}

export interface UserSessionRecord {
  userId: string;
  sessionId: string;
  deviceId: string;
  timestamp: number;
  sessionType: 'guest' | 'authenticated' | 'system';
  metadata: Record<string, any>;
  computeStats: {
    totalTasks: number;
    avgProcessingTime: number;
    peakThroughput: number;
  };
}

export interface CampaignRecord {
  campaignId: string;
  name: string;
  lastUpdated: number;
  state: Record<string, any>;
  features: string[];
  tags: string[];
  computeHistory: string[]; // Array of compute state IDs
}

export class IndexedDBManager {
  private db: IDBDatabase | null = null;
  private config: IndexedDBConfig;
  private isInitialized = false;
  private initPromise: Promise<void> | null = null;

  constructor() {
    this.config = {
      name: 'OvasabiStateDB',
      version: 1,
      stores: [
        {
          name: 'computeStates',
          keyPath: 'id',
          indexes: [
            { name: 'type', keyPath: 'type' },
            { name: 'timestamp', keyPath: 'timestamp' },
            { name: 'particleCount', keyPath: 'particleCount' },
            { name: 'performance', keyPath: 'performance.fps' }
          ]
        },
        {
          name: 'userSessions',
          keyPath: 'userId',
          indexes: [
            { name: 'sessionId', keyPath: 'sessionId' },
            { name: 'timestamp', keyPath: 'timestamp' },
            { name: 'sessionType', keyPath: 'sessionType' }
          ]
        },
        {
          name: 'campaigns',
          keyPath: 'campaignId',
          indexes: [
            { name: 'lastUpdated', keyPath: 'lastUpdated' },
            { name: 'name', keyPath: 'name' }
          ]
        }
      ]
    };
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;
    if (this.initPromise) return this.initPromise;

    this.initPromise = this.performInitialization();
    return this.initPromise;
  }

  private async performInitialization(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.config.name, this.config.version);

      request.onerror = () => {
        console.error('[IndexedDB] Failed to open database:', request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        this.db = request.result;
        this.isInitialized = true;
        console.log('[IndexedDB] Database initialized successfully');
        resolve();
      };

      request.onupgradeneeded = event => {
        const db = (event.target as IDBOpenDBRequest).result;
        console.log('[IndexedDB] Upgrading database to version', this.config.version);

        // Create object stores
        for (const storeConfig of this.config.stores) {
          if (!db.objectStoreNames.contains(storeConfig.name)) {
            const store = db.createObjectStore(storeConfig.name, { keyPath: storeConfig.keyPath });

            // Create indexes
            for (const indexConfig of storeConfig.indexes) {
              store.createIndex(indexConfig.name, indexConfig.keyPath, {
                unique: indexConfig.unique || false
              });
            }

            console.log(`[IndexedDB] Created store: ${storeConfig.name}`);
          }
        }
      };
    });
  }

  // Compute State Management
  async storeComputeState(state: ComputeStateRecord): Promise<void> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readwrite');
      const store = transaction.objectStore('computeStates');

      const request = store.put(state);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  async getComputeState(id: string): Promise<ComputeStateRecord | null> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readonly');
      const store = transaction.objectStore('computeStates');

      const request = store.get(id);
      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  }

  async getComputeStatesByType(type: string, limit = 100): Promise<ComputeStateRecord[]> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readonly');
      const store = transaction.objectStore('computeStates');
      const index = store.index('type');

      const request = index.getAll(type);
      request.onsuccess = () => {
        const results = request.result || [];
        // Sort by timestamp descending and limit
        results.sort((a, b) => b.timestamp - a.timestamp);
        resolve(results.slice(0, limit));
      };
      request.onerror = () => reject(request.error);
    });
  }

  async getComputeStatesByPerformance(minFps: number, limit = 50): Promise<ComputeStateRecord[]> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readonly');
      const store = transaction.objectStore('computeStates');
      const index = store.index('performance');

      const range = IDBKeyRange.lowerBound(minFps);
      const request = index.getAll(range);
      request.onsuccess = () => {
        const results = request.result || [];
        results.sort((a, b) => b.performance.fps - a.performance.fps);
        resolve(results.slice(0, limit));
      };
      request.onerror = () => reject(request.error);
    });
  }

  // User Session Management
  async storeUserSession(session: UserSessionRecord): Promise<void> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['userSessions'], 'readwrite');
      const store = transaction.objectStore('userSessions');

      const request = store.put(session);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  async getUserSession(userId: string): Promise<UserSessionRecord | null> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['userSessions'], 'readonly');
      const store = transaction.objectStore('userSessions');

      const request = store.get(userId);
      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  }

  async getUserSessionsByType(sessionType: string, limit = 100): Promise<UserSessionRecord[]> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['userSessions'], 'readonly');
      const store = transaction.objectStore('userSessions');
      const index = store.index('sessionType');

      const request = index.getAll(sessionType);
      request.onsuccess = () => {
        const results = request.result || [];
        results.sort((a, b) => b.timestamp - a.timestamp);
        resolve(results.slice(0, limit));
      };
      request.onerror = () => reject(request.error);
    });
  }

  // Campaign Management
  async storeCampaign(campaign: CampaignRecord): Promise<void> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['campaigns'], 'readwrite');
      const store = transaction.objectStore('campaigns');

      const request = store.put(campaign);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  async getCampaign(campaignId: string): Promise<CampaignRecord | null> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['campaigns'], 'readonly');
      const store = transaction.objectStore('campaigns');

      const request = store.get(campaignId);
      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  }

  async getAllCampaigns(limit = 50): Promise<CampaignRecord[]> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['campaigns'], 'readonly');
      const store = transaction.objectStore('campaigns');

      const request = store.getAll();
      request.onsuccess = () => {
        const results = request.result || [];
        results.sort((a, b) => b.lastUpdated - a.lastUpdated);
        resolve(results.slice(0, limit));
      };
      request.onerror = () => reject(request.error);
    });
  }

  // Analytics and Queries
  async getPerformanceAnalytics(timeRange: { start: number; end: number }): Promise<{
    avgFps: number;
    avgThroughput: number;
    avgLatency: number;
    totalTasks: number;
    peakPerformance: ComputeStateRecord | null;
  }> {
    await this.ensureInitialized();

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readonly');
      const store = transaction.objectStore('computeStates');
      const index = store.index('timestamp');

      const range = IDBKeyRange.bound(timeRange.start, timeRange.end);
      const request = index.getAll(range);

      request.onsuccess = () => {
        const results = request.result || [];

        if (results.length === 0) {
          resolve({
            avgFps: 0,
            avgThroughput: 0,
            avgLatency: 0,
            totalTasks: 0,
            peakPerformance: null
          });
          return;
        }

        const totalFps = results.reduce((sum, r) => sum + r.performance.fps, 0);
        const totalThroughput = results.reduce((sum, r) => sum + r.performance.throughput, 0);
        const totalLatency = results.reduce((sum, r) => sum + r.performance.latency, 0);

        const peakPerformance = results.reduce((peak, current) =>
          current.performance.fps > peak.performance.fps ? current : peak
        );

        resolve({
          avgFps: totalFps / results.length,
          avgThroughput: totalThroughput / results.length,
          avgLatency: totalLatency / results.length,
          totalTasks: results.length,
          peakPerformance
        });
      };

      request.onerror = () => reject(request.error);
    });
  }

  // Cleanup and Maintenance
  async cleanupOldData(maxAge: number): Promise<number> {
    await this.ensureInitialized();

    const cutoffTime = Date.now() - maxAge;
    let deletedCount = 0;

    // Clean up old compute states
    const computeStates = await this.getComputeStatesByType('any', 10000);
    const oldComputeStates = computeStates.filter(s => s.timestamp < cutoffTime);

    for (const state of oldComputeStates) {
      await this.deleteComputeState(state.id);
      deletedCount++;
    }

    return deletedCount;
  }

  private async deleteComputeState(id: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction(['computeStates'], 'readwrite');
      const store = transaction.objectStore('computeStates');

      const request = store.delete(id);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  // Utility methods
  private async ensureInitialized(): Promise<void> {
    if (!this.isInitialized) {
      await this.initialize();
    }
  }

  async getDatabaseStats(): Promise<{
    computeStates: number;
    userSessions: number;
    campaigns: number;
    totalSize: number;
  }> {
    await this.ensureInitialized();

    const [computeStates, userSessions, campaigns] = await Promise.all([
      this.countRecords('computeStates'),
      this.countRecords('userSessions'),
      this.countRecords('campaigns')
    ]);

    return {
      computeStates,
      userSessions,
      campaigns,
      totalSize: computeStates + userSessions + campaigns
    };
  }

  private async countRecords(storeName: string): Promise<number> {
    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([storeName], 'readonly');
      const store = transaction.objectStore(storeName);

      const request = store.count();
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  async close(): Promise<void> {
    if (this.db) {
      this.db.close();
      this.db = null;
      this.isInitialized = false;
    }
  }
}

// Singleton instance
export const indexedDBManager = new IndexedDBManager();
