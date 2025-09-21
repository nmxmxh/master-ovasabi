/**
 * LOD (Level of Detail) Manager
 *
 * Manages LOD levels for physics entities and environment chunks
 * Optimizes rendering performance for intense scenes
 */

import type { Vector3 } from './IDManager';

export interface LODConfig {
  levels: LODLevel[];
  distances: number[];
  polygonCounts: number[];
  textureSizes: number[];
  compressionLevels: number[];
}

export interface LODLevel {
  level: number;
  distance: number;
  polygonCount: number;
  textureSize: number;
  compressed: boolean;
  properties: Record<string, any>;
}

export interface LODEntity {
  id: string;
  position: Vector3;
  type: string;
  currentLOD: number;
  targetLOD: number;
  lastUpdate: number;
  properties: Record<string, any>;
}

export interface LODPerformance {
  totalEntities: number;
  visibleEntities: number;
  culledEntities: number;
  averageLOD: number;
  memoryUsage: number;
  renderTime: number;
  lastUpdate: number;
}

export class LODManager {
  private entities: Map<string, LODEntity> = new Map();
  private config: LODConfig;
  private performance: LODPerformance;
  private cameraPosition: Vector3 = { x: 0, y: 0, z: 0 };
  // private cameraDirection: Vector3 = { x: 0, y: 0, z: -1 }; // Currently unused
  private updateInterval: number = 100; // Update every 100ms
  private lastUpdate: number = 0;
  private isActive: boolean = false;

  constructor(config?: Partial<LODConfig>) {
    this.config = this.createDefaultConfig(config);
    this.performance = this.createDefaultPerformance();
  }

  private createDefaultConfig(overrides?: Partial<LODConfig>): LODConfig {
    const defaultConfig: LODConfig = {
      levels: [
        {
          level: 0,
          distance: 50,
          polygonCount: 1000,
          textureSize: 512,
          compressed: false,
          properties: {}
        },
        {
          level: 1,
          distance: 100,
          polygonCount: 500,
          textureSize: 256,
          compressed: false,
          properties: {}
        },
        {
          level: 2,
          distance: 200,
          polygonCount: 250,
          textureSize: 128,
          compressed: true,
          properties: {}
        },
        {
          level: 3,
          distance: 500,
          polygonCount: 100,
          textureSize: 64,
          compressed: true,
          properties: {}
        },
        {
          level: 4,
          distance: 1000,
          polygonCount: 50,
          textureSize: 32,
          compressed: true,
          properties: {}
        }
      ],
      distances: [50, 100, 200, 500, 1000],
      polygonCounts: [1000, 500, 250, 100, 50],
      textureSizes: [512, 256, 128, 64, 32],
      compressionLevels: [0, 0, 1, 1, 2]
    };

    return { ...defaultConfig, ...overrides };
  }

  private createDefaultPerformance(): LODPerformance {
    return {
      totalEntities: 0,
      visibleEntities: 0,
      culledEntities: 0,
      averageLOD: 0,
      memoryUsage: 0,
      renderTime: 0,
      lastUpdate: Date.now()
    };
  }

  public start(): void {
    this.isActive = true;
    this.update();
  }

  public stop(): void {
    this.isActive = false;
  }

  public update(): void {
    if (!this.isActive) return;

    const now = Date.now();
    if (now - this.lastUpdate < this.updateInterval) {
      requestAnimationFrame(() => this.update());
      return;
    }

    this.lastUpdate = now;
    this.processLODUpdates();
    this.updatePerformance();
    requestAnimationFrame(() => this.update());
  }

  private processLODUpdates(): void {
    let visibleCount = 0;
    let culledCount = 0;
    let totalLOD = 0;

    for (const [id, entity] of this.entities) {
      const distance = this.calculateDistance(entity.position, this.cameraPosition);
      const newLOD = this.calculateLOD(distance, entity.type);

      // Update entity LOD
      entity.currentLOD = newLOD;
      entity.targetLOD = newLOD;
      entity.lastUpdate = Date.now();

      if (newLOD >= this.config.levels.length) {
        culledCount++;
      } else {
        visibleCount++;
        totalLOD += newLOD;
      }

      // Trigger LOD change event
      this.onLODChange(id, entity, newLOD);
    }

    this.performance.visibleEntities = visibleCount;
    this.performance.culledEntities = culledCount;
    this.performance.averageLOD = visibleCount > 0 ? totalLOD / visibleCount : 0;
  }

  private calculateLOD(distance: number, _entityType: string): number {
    // Find the appropriate LOD level based on distance
    for (let i = 0; i < this.config.distances.length; i++) {
      if (distance <= this.config.distances[i]) {
        return i;
      }
    }

    // If beyond all distances, cull the entity
    return this.config.distances.length;
  }

  private calculateDistance(pos1: Vector3, pos2: Vector3): number {
    const dx = pos1.x - pos2.x;
    const dy = pos1.y - pos2.y;
    const dz = pos1.z - pos2.z;
    return Math.sqrt(dx * dx + dy * dy + dz * dz);
  }

  private onLODChange(entityId: string, entity: LODEntity, newLOD: number): void {
    // Emit LOD change event
    const event = new CustomEvent('lodChange', {
      detail: {
        entityId,
        entity,
        newLOD,
        oldLOD: entity.currentLOD,
        timestamp: Date.now()
      }
    });

    window.dispatchEvent(event);
  }

  private updatePerformance(): void {
    this.performance.totalEntities = this.entities.size;
    this.performance.lastUpdate = Date.now();

    // Calculate memory usage (simplified)
    this.performance.memoryUsage = this.entities.size * 1024; // 1KB per entity estimate
  }

  public addEntity(entity: LODEntity): void {
    this.entities.set(entity.id, entity);
  }

  public removeEntity(entityId: string): void {
    this.entities.delete(entityId);
  }

  public updateEntity(entityId: string, updates: Partial<LODEntity>): void {
    const entity = this.entities.get(entityId);
    if (entity) {
      Object.assign(entity, updates);
    }
  }

  public getEntity(entityId: string): LODEntity | undefined {
    return this.entities.get(entityId);
  }

  public getAllEntities(): LODEntity[] {
    return Array.from(this.entities.values());
  }

  public getVisibleEntities(): LODEntity[] {
    return this.getAllEntities().filter(entity => entity.currentLOD < this.config.levels.length);
  }

  public getCulledEntities(): LODEntity[] {
    return this.getAllEntities().filter(entity => entity.currentLOD >= this.config.levels.length);
  }

  public setCameraPosition(position: Vector3): void {
    this.cameraPosition = position;
  }

  public setCameraDirection(direction: Vector3): void {
    // this.cameraDirection = direction; // Currently unused
    console.log('[LODManager] Camera direction updated:', direction);
  }

  public getLODConfig(): LODConfig {
    return this.config;
  }

  public updateLODConfig(config: Partial<LODConfig>): void {
    this.config = { ...this.config, ...config };
  }

  public getPerformance(): LODPerformance {
    return this.performance;
  }

  public getLODLevel(level: number): LODLevel | undefined {
    return this.config.levels[level];
  }

  public getLODForDistance(distance: number): number {
    return this.calculateLOD(distance, 'default');
  }

  public getLODForEntity(entityId: string): number {
    const entity = this.entities.get(entityId);
    return entity ? entity.currentLOD : -1;
  }

  public isEntityVisible(entityId: string): boolean {
    const entity = this.entities.get(entityId);
    return entity ? entity.currentLOD < this.config.levels.length : false;
  }

  public getEntityCount(): number {
    return this.entities.size;
  }

  public getVisibleEntityCount(): number {
    return this.performance.visibleEntities;
  }

  public getCulledEntityCount(): number {
    return this.performance.culledEntities;
  }

  public clear(): void {
    this.entities.clear();
    this.performance = this.createDefaultPerformance();
  }

  public getMemoryUsage(): number {
    return this.performance.memoryUsage;
  }

  public getAverageLOD(): number {
    return this.performance.averageLOD;
  }

  public getRenderTime(): number {
    return this.performance.renderTime;
  }

  public setRenderTime(time: number): void {
    this.performance.renderTime = time;
  }

  public setUpdateInterval(interval: number): void {
    this.updateInterval = interval;
  }

  public getUpdateInterval(): number {
    return this.updateInterval;
  }

  public isRunning(): boolean {
    return this.isActive;
  }

  public getEntityTypes(): string[] {
    const types = new Set<string>();
    for (const entity of this.entities.values()) {
      types.add(entity.type);
    }
    return Array.from(types);
  }

  public getEntitiesByType(type: string): LODEntity[] {
    return this.getAllEntities().filter(entity => entity.type === type);
  }

  public getEntitiesByLOD(level: number): LODEntity[] {
    return this.getAllEntities().filter(entity => entity.currentLOD === level);
  }

  public getLODDistribution(): Record<number, number> {
    const distribution: Record<number, number> = {};

    for (const entity of this.entities.values()) {
      const level = entity.currentLOD;
      distribution[level] = (distribution[level] || 0) + 1;
    }

    return distribution;
  }

  public optimizeForPerformance(): void {
    // Reduce update frequency if performance is poor
    if (this.performance.renderTime > 16.67) {
      // 60 FPS threshold
      this.updateInterval = Math.min(this.updateInterval * 1.5, 500);
    } else if (this.performance.renderTime < 8.33) {
      // 120 FPS threshold
      this.updateInterval = Math.max(this.updateInterval * 0.8, 50);
    }
  }

  public getOptimizationSuggestions(): string[] {
    const suggestions: string[] = [];

    if (this.performance.renderTime > 16.67) {
      suggestions.push('Consider reducing LOD update frequency');
      suggestions.push('Consider culling more distant entities');
    }

    if (this.performance.memoryUsage > 100 * 1024 * 1024) {
      // 100MB
      suggestions.push('Consider reducing entity count');
      suggestions.push('Consider using more aggressive LOD levels');
    }

    if (this.performance.averageLOD < 1) {
      suggestions.push('Consider increasing LOD distances for better performance');
    }

    return suggestions;
  }
}

// Export singleton instance
export const lodManager = new LODManager();

// Export default configuration
export const defaultLODConfig: LODConfig = {
  levels: [
    {
      level: 0,
      distance: 50,
      polygonCount: 1000,
      textureSize: 512,
      compressed: false,
      properties: {}
    },
    {
      level: 1,
      distance: 100,
      polygonCount: 500,
      textureSize: 256,
      compressed: false,
      properties: {}
    },
    {
      level: 2,
      distance: 200,
      polygonCount: 250,
      textureSize: 128,
      compressed: true,
      properties: {}
    },
    {
      level: 3,
      distance: 500,
      polygonCount: 100,
      textureSize: 64,
      compressed: true,
      properties: {}
    },
    {
      level: 4,
      distance: 1000,
      polygonCount: 50,
      textureSize: 32,
      compressed: true,
      properties: {}
    }
  ],
  distances: [50, 100, 200, 500, 1000],
  polygonCounts: [1000, 500, 250, 100, 50],
  textureSizes: [512, 256, 128, 64, 32],
  compressionLevels: [0, 0, 1, 1, 2]
};
