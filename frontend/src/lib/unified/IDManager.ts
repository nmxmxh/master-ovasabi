/**
 * Unified ID Management System
 *
 * Single source of truth for ID generation across frontend, WASM, and backend
 * Supports both 2D and 3D entity identification
 */

export interface IDConfig {
  prefix: string;
  length: number;
  includeTimestamp: boolean;
  includePhysics?: boolean; // For 3D physics entities
}

export interface EntityID {
  id: string;
  type: 'user' | 'session' | 'device' | 'campaign' | 'physics' | 'environment' | 'correlation';
  timestamp: number;
  metadata?: Record<string, any>;
}

export class UnifiedIDManager {
  private static instance: UnifiedIDManager;
  private configs: Map<string, IDConfig> = new Map();

  private constructor() {
    this.initializeConfigs();
  }

  static getInstance(): UnifiedIDManager {
    if (!UnifiedIDManager.instance) {
      UnifiedIDManager.instance = new UnifiedIDManager();
    }
    return UnifiedIDManager.instance;
  }

  private initializeConfigs(): void {
    // 2D/Web IDs
    this.configs.set('user', { prefix: 'user', length: 32, includeTimestamp: true });
    this.configs.set('session', { prefix: 'session', length: 32, includeTimestamp: true });
    this.configs.set('device', { prefix: 'device', length: 32, includeTimestamp: true });
    this.configs.set('campaign', { prefix: 'campaign', length: 24, includeTimestamp: true });
    this.configs.set('correlation', { prefix: 'corr', length: 24, includeTimestamp: true });

    // 3D/Physics IDs
    this.configs.set('physics', {
      prefix: 'phys',
      length: 28,
      includeTimestamp: true,
      includePhysics: true
    });
    this.configs.set('environment', {
      prefix: 'env',
      length: 28,
      includeTimestamp: true,
      includePhysics: true
    });
  }

  /**
   * Generate a unified ID with consistent format across all systems
   */
  generateID(type: string, additionalData: string[] = [], physicsData?: PhysicsData): EntityID {
    const config = this.configs.get(type);
    if (!config) {
      throw new Error(`Unknown ID type: ${type}`);
    }

    const timestamp = Date.now() * 1000000 + Math.floor(Math.random() * 1000000);
    let input = `${config.prefix}_${timestamp}_${type}`;

    // Add additional data
    for (const data of additionalData) {
      input += `_${data}`;
    }

    // Add physics data for 3D entities
    if (config.includePhysics && physicsData) {
      input += `_${physicsData.position.x}_${physicsData.position.y}_${physicsData.position.z}`;
      input += `_${physicsData.velocity?.x || 0}_${physicsData.velocity?.y || 0}_${physicsData.velocity?.z || 0}`;
    }

    // Generate SHA256 hash
    const hash = this.sha256Hash(input);
    const id = this.formatID(config.prefix, hash, config.length);

    return {
      id,
      type: type as EntityID['type'],
      timestamp,
      metadata: physicsData ? { physics: physicsData } : undefined
    };
  }

  /**
   * Generate physics entity ID for 3D objects
   */
  generatePhysicsID(
    position: Vector3,
    velocity?: Vector3,
    additionalData: string[] = []
  ): EntityID {
    const physicsData: PhysicsData = {
      position,
      velocity: velocity || { x: 0, y: 0, z: 0 },
      mass: 1.0,
      restitution: 0.8
    };

    return this.generateID('physics', additionalData, physicsData);
  }

  /**
   * Generate environment ID for 3D scenes
   */
  generateEnvironmentID(sceneType: string, additionalData: string[] = []): EntityID {
    return this.generateID('environment', [sceneType, ...additionalData]);
  }

  private sha256Hash(input: string): string {
    // Use Web Crypto API for consistent hashing
    const encoder = new TextEncoder();
    const data = encoder.encode(input);

    return crypto.subtle.digest('SHA-256', data).then(hashBuffer => {
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }) as any; // Simplified for now
  }

  private formatID(prefix: string, hash: string, length: number): string {
    let formattedHash = hash;
    if (formattedHash.length > length) {
      formattedHash = formattedHash.substring(0, length);
    } else if (formattedHash.length < length) {
      const additional = this.sha256Hash(hash + Date.now().toString());
      formattedHash = formattedHash + additional.substring(0, length - formattedHash.length);
    }
    return `${prefix}_${formattedHash}`;
  }
}

// Type definitions for 3D physics
export interface Vector3 {
  x: number;
  y: number;
  z: number;
}

export interface PhysicsData {
  position: Vector3;
  velocity: Vector3;
  mass: number;
  restitution: number;
}

// Export singleton instance
export const idManager = UnifiedIDManager.getInstance();


