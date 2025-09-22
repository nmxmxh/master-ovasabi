/**
 * Distributed Physics Platform Component
 *
 * Main component that integrates all physics systems for the distributed platform
 * Provides a unified interface for physics, environment, and rendering
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';
import {
  PhysicsPlatform,
  type PhysicsPlatformConfig,
  defaultPhysicsPlatformConfig
} from '../lib/unified/PhysicsPlatform';
import { PhysicsRendererComponent } from './PhysicsRenderer';
import { useUnifiedState } from '../lib/unified/StateManager';

interface DistributedPhysicsPlatformProps {
  campaignId: string;
  config?: Partial<PhysicsPlatformConfig>;
  onPhysicsUpdate?: (entities: any[]) => void;
  onPerformanceUpdate?: (performance: any) => void;
  onLODChange?: (entityId: string, newLOD: number) => void;
}

export const DistributedPhysicsPlatform: React.FC<DistributedPhysicsPlatformProps> = ({
  campaignId,
  config = {},
  onPhysicsUpdate,
  onPerformanceUpdate,
  onLODChange
}) => {
  const platformRef = useRef<PhysicsPlatform | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [performance, setPerformance] = useState<any>(null);
  const [entities, setEntities] = useState<any[]>([]);
  const [chunks, setChunks] = useState<any[]>([]);
  const [optimizationSuggestions, setOptimizationSuggestions] = useState<string[]>([]);

  const { physics, rendering, campaign } = useUnifiedState();

  // Initialize physics platform
  useEffect(() => {
    const initializePlatform = async () => {
      try {
        const platformConfig: PhysicsPlatformConfig = {
          ...defaultPhysicsPlatformConfig,
          ...config,
          campaignId
        };

        const platform = new PhysicsPlatform(platformConfig);
        platformRef.current = platform;

        // Set up event listeners
        setupEventListeners(platform);

        setIsInitialized(true);
        console.log('[DistributedPhysicsPlatform] Initialized successfully');
      } catch (error) {
        console.error('[DistributedPhysicsPlatform] Initialization failed:', error);
      }
    };

    initializePlatform();

    return () => {
      if (platformRef.current) {
        platformRef.current.stop();
        platformRef.current.clear();
      }
    };
  }, [campaignId, config]);

  // Start/stop platform based on campaign state
  useEffect(() => {
    if (!platformRef.current || !isInitialized) return;

    if (campaign.id && campaign.id !== 'default') {
      startPlatform();
    } else {
      stopPlatform();
    }
  }, [campaign.id, isInitialized]);

  // Update platform configuration when campaign changes
  useEffect(() => {
    if (!platformRef.current || !isInitialized) return;

    const newConfig: Partial<PhysicsPlatformConfig> = {
      campaignId: campaign.id || 'default',
      maxEntities: 1000, // Default value
      worldBounds: physics.world?.bounds || {
        min: { x: -100, y: -100, z: -100 },
        max: { x: 100, y: 100, z: 100 }
      },
      physicsRate: 60, // Default value
      lodConfig: {
        distances: [50, 100, 200, 500, 1000], // Default values
        polygonCounts: [1000, 500, 250, 100, 50],
        textureSizes: [512, 256, 128, 64, 32]
      },
      rendering: {
        antialias: rendering.antialias ?? true,
        shadows: rendering.shadowMap ?? true,
        postProcessing: rendering.postProcessing ?? true
      }
    };

    platformRef.current.updateConfig(newConfig);
  }, [campaign.id, physics, rendering, isInitialized]);

  // Update entities when physics state changes
  useEffect(() => {
    if (!platformRef.current || !isInitialized) return;

    // Update entities from physics state
    if (physics.entities) {
      const entityArray = Array.from(physics.entities.values());
      setEntities(entityArray);

      // Sync entities with physics platform
      const platformEntities = platformRef.current.getAllEntities();
      const platformEntityIds = new Set(platformEntities.map(e => e.id));

      // Add new entities to platform
      entityArray.forEach(entity => {
        if (!platformEntityIds.has(entity.id)) {
          platformRef.current?.spawnEntity(entity);
        } else {
          platformRef.current?.updateEntity(entity);
        }
      });

      // Remove entities that are no longer in state
      platformEntities.forEach(platformEntity => {
        if (!physics.entities.has(platformEntity.id)) {
          platformRef.current?.destroyEntity({ id: platformEntity.id });
        }
      });
    }
  }, [physics.entities, isInitialized]);

  // Update environment chunks
  useEffect(() => {
    if (!platformRef.current || !isInitialized) return;

    // Update chunks from platform
    const chunkArray = platformRef.current.getAllChunks();
    setChunks(chunkArray);
  }, [isInitialized]);

  // Performance monitoring
  useEffect(() => {
    if (!platformRef.current || !isRunning) return;

    const interval = setInterval(() => {
      if (platformRef.current) {
        const perf = platformRef.current.getPerformance();
        setPerformance(perf);
        setOptimizationSuggestions(platformRef.current.getOptimizationSuggestions());

        if (onPerformanceUpdate) {
          onPerformanceUpdate(perf);
        }
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isRunning, onPerformanceUpdate]);

  const setupEventListeners = (_platform: PhysicsPlatform) => {
    // Listen for LOD changes
    window.addEventListener('lodChange', (event: Event) => {
      const customEvent = event as CustomEvent;
      const { entityId, newLOD } = customEvent.detail;
      if (onLODChange) {
        onLODChange(entityId, newLOD);
      }
    });

    // Listen for physics updates
    window.addEventListener('physicsUpdate', (event: Event) => {
      const customEvent = event as CustomEvent;
      const { entities } = customEvent.detail;
      setEntities(entities);
      if (onPhysicsUpdate) {
        onPhysicsUpdate(entities);
      }
    });
  };

  const startPlatform = useCallback(() => {
    if (platformRef.current && !isRunning) {
      platformRef.current.start();
      setIsRunning(true);
      console.log('[DistributedPhysicsPlatform] Started');
    }
  }, [isRunning]);

  const stopPlatform = useCallback(() => {
    if (platformRef.current && isRunning) {
      platformRef.current.stop();
      setIsRunning(false);
      console.log('[DistributedPhysicsPlatform] Stopped');
    }
  }, [isRunning]);

  const spawnEntity = useCallback((entityData: any) => {
    // Add to state manager first
    const { addPhysicsEntity } = useUnifiedState.getState();
    const entityId = addPhysicsEntity(entityData);

    // The platform will be updated via the useEffect that watches physics.entities
    console.log('[DistributedPhysicsPlatform] Spawned entity:', entityId);
  }, []);

  const destroyEntity = useCallback((entityId: string) => {
    // Remove from state manager first
    const { removePhysicsEntity } = useUnifiedState.getState();
    removePhysicsEntity(entityId);

    // The platform will be updated via the useEffect that watches physics.entities
  }, []);

  const optimizePerformance = useCallback(() => {
    if (platformRef.current) {
      platformRef.current.optimizeForPerformance();
    }
  }, []);

  if (!isInitialized) {
    return (
      <div className="physics-platform-loading">
        <div className="loading-spinner">
          <div className="spinner"></div>
          <p>Initializing Physics Platform...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="distributed-physics-platform">
      {/* Physics Renderer */}
      <div className="physics-renderer-container">
        <PhysicsRendererComponent
          width={800}
          height={600}
          antialias={rendering.antialias}
          shadows={rendering.shadowMap}
          postProcessing={rendering.postProcessing}
        />
      </div>

      {/* Performance Overlay */}
      {performance && (
        <div className="performance-overlay">
          <h3>Physics Performance</h3>
          <div className="performance-metrics">
            <div className="metric">
              <span className="label">FPS:</span>
              <span className="value">{performance.fps.toFixed(1)}</span>
            </div>
            <div className="metric">
              <span className="label">Frame Time:</span>
              <span className="value">{performance.frameTime.toFixed(2)}ms</span>
            </div>
            <div className="metric">
              <span className="label">Entities:</span>
              <span className="value">{performance.entityCount}</span>
            </div>
            <div className="metric">
              <span className="label">Visible:</span>
              <span className="value">{performance.visibleEntities}</span>
            </div>
            <div className="metric">
              <span className="label">Culled:</span>
              <span className="value">{performance.culledEntities}</span>
            </div>
            <div className="metric">
              <span className="label">Avg LOD:</span>
              <span className="value">{performance.averageLOD.toFixed(1)}</span>
            </div>
            <div className="metric">
              <span className="label">Memory:</span>
              <span className="value">{(performance.memoryUsage / 1024 / 1024).toFixed(1)}MB</span>
            </div>
          </div>
        </div>
      )}

      {/* Optimization Suggestions */}
      {optimizationSuggestions.length > 0 && (
        <div className="optimization-suggestions">
          <h3>Performance Suggestions</h3>
          <ul>
            {optimizationSuggestions.map((suggestion, index) => (
              <li key={index}>{suggestion}</li>
            ))}
          </ul>
          <button onClick={optimizePerformance} className="optimize-button">
            Apply Optimizations
          </button>
        </div>
      )}

      {/* Entity Management */}
      <div className="entity-management">
        <h3>Entity Management</h3>
        <div className="entity-controls">
          <button onClick={() => spawnEntity({ type: 'cube', position: { x: 0, y: 0, z: 0 } })}>
            Spawn Cube
          </button>
          <button onClick={() => spawnEntity({ type: 'sphere', position: { x: 5, y: 0, z: 0 } })}>
            Spawn Sphere
          </button>
          <button onClick={() => spawnEntity({ type: 'capsule', position: { x: -5, y: 0, z: 0 } })}>
            Spawn Capsule
          </button>
        </div>
        <div className="entity-list">
          <h4>Active Entities ({entities.length})</h4>
          {entities.map(entity => (
            <div key={entity.id} className="entity-item">
              <span className="entity-id">{entity.id}</span>
              <span className="entity-type">{entity.type}</span>
              <span className="entity-lod">LOD: {entity.lod}</span>
              <span className="entity-visible">{entity.visible ? 'Visible' : 'Culled'}</span>
              <button onClick={() => destroyEntity(entity.id)} className="destroy-button">
                Destroy
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Environment Chunks */}
      <div className="environment-chunks">
        <h3>Environment Chunks</h3>
        <div className="chunk-list">
          <h4>Loaded Chunks ({chunks.length})</h4>
          {chunks.map(chunk => (
            <div key={chunk.id} className="chunk-item">
              <span className="chunk-id">{chunk.id}</span>
              <span className="chunk-lod">LOD: {chunk.lod}</span>
              <span className="chunk-size">{(chunk.size / 1024).toFixed(1)}KB</span>
              <span className="chunk-compressed">{chunk.compressed ? 'Compressed' : 'Raw'}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Platform Controls */}
      <div className="platform-controls">
        <h3>Platform Controls</h3>
        <div className="control-buttons">
          <button
            onClick={isRunning ? stopPlatform : startPlatform}
            className={isRunning ? 'stop-button' : 'start-button'}
          >
            {isRunning ? 'Stop Platform' : 'Start Platform'}
          </button>
          <button onClick={() => platformRef.current?.clear()} className="clear-button">
            Clear All
          </button>
          <button onClick={optimizePerformance} className="optimize-button">
            Optimize Performance
          </button>
        </div>
      </div>

      {/* Campaign Info */}
      <div className="campaign-info">
        <h3>Campaign Information</h3>
        <div className="campaign-details">
          <div className="detail">
            <span className="label">Campaign ID:</span>
            <span className="value">{campaign.id || 'None'}</span>
          </div>
          <div className="detail">
            <span className="label">Physics Rate:</span>
            <span className="value">60 Hz</span>
          </div>
          <div className="detail">
            <span className="label">Max Entities:</span>
            <span className="value">1000</span>
          </div>
          <div className="detail">
            <span className="label">World Bounds:</span>
            <span className="value">
              {physics.world?.bounds
                ? `${physics.world.bounds.min.x} to ${physics.world.bounds.max.x}`
                : 'Default'}
            </span>
          </div>
          <div className="detail">
            <span className="label">Physics Mode:</span>
            <span className="value">
              {platformRef.current?.getPhysicsMode() === 'godot'
                ? `Godot ${platformRef.current?.isGodotConnected() ? '(Connected)' : '(Disconnected)'}`
                : 'Cannon.js (Fallback)'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DistributedPhysicsPlatform;
