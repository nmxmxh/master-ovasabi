/**
 * Unified State Management System
 *
 * Consolidates all state management patterns for 2D → 3D transition
 * Supports both traditional web state and 3D physics state
 */

import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { idManager, type Vector3 } from './IDManager';

// Core state interfaces
export interface CoreState {
  // 2D/Web state
  user: UserState;
  session: SessionState;
  campaign: CampaignState;
  connection: ConnectionState;

  // 3D/Physics state
  physics: PhysicsState;
  environment: EnvironmentState;
  rendering: RenderingState;
}

export interface UserState {
  id: string;
  username: string;
  position: Vector3; // 3D position
  rotation: Vector3; // 3D rotation
  metadata: Record<string, any>;
}

export interface SessionState {
  id: string;
  startTime: number;
  lastActivity: number;
  isActive: boolean;
  physicsEnabled: boolean;
}

export interface CampaignState {
  id: string;
  name: string;
  type: '2d' | '3d' | 'mixed';
  physics: {
    enabled: boolean;
    gravity: Vector3;
    worldBounds: {
      min: Vector3;
      max: Vector3;
    };
  };
  environment: {
    sceneId: string;
    lighting: LightingConfig;
    fog: FogConfig;
  };
  features: string[];
}

export interface ConnectionState {
  isConnected: boolean;
  wasmReady: boolean;
  webgpuReady: boolean;
  physicsReady: boolean;
  lastPing: number;
}

export interface PhysicsState {
  enabled: boolean;
  entities: Map<string, PhysicsEntity>;
  world: PhysicsWorld;
  performance: PhysicsPerformance;
}

export interface PhysicsEntity {
  id: string;
  type: string;
  position: Vector3;
  rotation: { x: number; y: number; z: number; w: number };
  scale: Vector3;
  velocity: Vector3;
  mass: number;
  restitution: number;
  friction: number;
  active: boolean;
  lod: number;
  visible: boolean;
  properties: Record<string, any>;
}

export interface PhysicsWorld {
  gravity: Vector3;
  bounds: {
    min: Vector3;
    max: Vector3;
  };
  timeStep: number;
  iterations: number;
}

export interface PhysicsPerformance {
  fps: number;
  frameTime: number;
  entityCount: number;
  collisionCount: number;
  gpuUtilization: number;
}

export interface EnvironmentState {
  sceneId: string;
  lighting: LightingConfig;
  fog: FogConfig;
  skybox: SkyboxConfig;
  postProcessing: PostProcessingConfig;
}

export interface LightingConfig {
  ambient: {
    color: string;
    intensity: number;
  };
  directional: {
    color: string;
    intensity: number;
    position: Vector3;
    target: Vector3;
  };
  shadows: {
    enabled: boolean;
    resolution: number;
    bias: number;
  };
}

export interface FogConfig {
  enabled: boolean;
  color: string;
  near: number;
  far: number;
  density: number;
}

export interface SkyboxConfig {
  type: 'color' | 'texture' | 'hdri';
  value: string;
  rotation: Vector3;
}

export interface PostProcessingConfig {
  bloom: {
    enabled: boolean;
    threshold: number;
    strength: number;
  };
  ssao: {
    enabled: boolean;
    radius: number;
    intensity: number;
  };
  toneMapping: {
    enabled: boolean;
    exposure: number;
  };
}

export interface RenderingState {
  renderer: 'webgl' | 'webgpu';
  antialias: boolean;
  shadowMap: boolean;
  postProcessing: boolean;
  performance: RenderingPerformance;
}

export interface RenderingPerformance {
  fps: number;
  frameTime: number;
  drawCalls: number;
  triangles: number;
  memoryUsage: number;
}

// Unified State Manager
interface UnifiedStateManager extends CoreState {
  // Actions
  initializePhysics: (config: PhysicsWorld) => void;
  addPhysicsEntity: (entity: Omit<PhysicsEntity, 'id'>) => string;
  removePhysicsEntity: (id: string) => void;
  updatePhysicsEntity: (id: string, updates: Partial<PhysicsEntity>) => void;

  setEnvironment: (environment: Partial<EnvironmentState>) => void;
  updateLighting: (lighting: Partial<LightingConfig>) => void;
  updateFog: (fog: Partial<FogConfig>) => void;

  setRenderingMode: (mode: '2d' | '3d' | 'mixed') => void;
  updateRenderingConfig: (config: Partial<RenderingState>) => void;

  // Campaign management
  switchCampaign: (campaignId: string) => void;
  updateCampaignPhysics: (physics: Partial<CampaignState['physics']>) => void;

  // Performance monitoring
  updatePerformance: (type: 'physics' | 'rendering', metrics: any) => void;

  // Event handling
  emitStateChange: (type: string, data: any) => void;
  subscribeToStateChanges: (callback: (type: string, data: any) => void) => () => void;
}

export const useUnifiedState = create<UnifiedStateManager>()(
  devtools(
    set => ({
      // Initial state
      user: {
        id: '',
        username: 'Guest',
        position: { x: 0, y: 0, z: 0 },
        rotation: { x: 0, y: 0, z: 0 },
        metadata: {}
      },
      session: {
        id: '',
        startTime: Date.now(),
        lastActivity: Date.now(),
        isActive: true,
        physicsEnabled: false
      },
      campaign: {
        id: '0',
        name: 'Default Campaign',
        type: '2d',
        physics: {
          enabled: false,
          gravity: { x: 0, y: -9.81, z: 0 },
          worldBounds: {
            min: { x: -100, y: -100, z: -100 },
            max: { x: 100, y: 100, z: 100 }
          }
        },
        environment: {
          sceneId: 'default',
          lighting: {
            ambient: { color: '#404040', intensity: 0.6 },
            directional: {
              color: '#ffffff',
              intensity: 0.8,
              position: { x: 10, y: 10, z: 5 },
              target: { x: 0, y: 0, z: 0 }
            },
            shadows: { enabled: true, resolution: 2048, bias: 0.0001 }
          },
          fog: { enabled: false, color: '#000000', near: 1, far: 100, density: 0.01 },
          skybox: { type: 'color', value: '#0a0a0a', rotation: { x: 0, y: 0, z: 0 } },
          postProcessing: {
            bloom: { enabled: false, threshold: 0.8, strength: 0.5 },
            ssao: { enabled: false, radius: 0.5, intensity: 1.0 },
            toneMapping: { enabled: true, exposure: 1.0 }
          }
        },
        features: []
      },
      connection: {
        isConnected: false,
        wasmReady: false,
        webgpuReady: false,
        physicsReady: false,
        lastPing: 0
      },
      physics: {
        enabled: false,
        entities: new Map(),
        world: {
          gravity: { x: 0, y: -9.81, z: 0 },
          bounds: {
            min: { x: -100, y: -100, z: -100 },
            max: { x: 100, y: 100, z: 100 }
          },
          timeStep: 1 / 60,
          iterations: 10
        },
        performance: {
          fps: 60,
          frameTime: 16.67,
          entityCount: 0,
          collisionCount: 0,
          gpuUtilization: 0
        }
      },
      environment: {
        sceneId: 'default',
        lighting: {
          ambient: { color: '#404040', intensity: 0.6 },
          directional: {
            color: '#ffffff',
            intensity: 0.8,
            position: { x: 10, y: 10, z: 5 },
            target: { x: 0, y: 0, z: 0 }
          },
          shadows: { enabled: true, resolution: 2048, bias: 0.0001 }
        },
        fog: { enabled: false, color: '#000000', near: 1, far: 100, density: 0.01 },
        skybox: { type: 'color', value: '#0a0a0a', rotation: { x: 0, y: 0, z: 0 } },
        postProcessing: {
          bloom: { enabled: false, threshold: 0.8, strength: 0.5 },
          ssao: { enabled: false, radius: 0.5, intensity: 1.0 },
          toneMapping: { enabled: true, exposure: 1.0 }
        }
      },
      rendering: {
        renderer: 'webgl',
        antialias: true,
        shadowMap: true,
        postProcessing: false,
        performance: {
          fps: 60,
          frameTime: 16.67,
          drawCalls: 0,
          triangles: 0,
          memoryUsage: 0
        }
      },

      // Actions
      initializePhysics: (config: PhysicsWorld) => {
        set(state => ({
          physics: {
            ...state.physics,
            enabled: true,
            world: config
          },
          session: {
            ...state.session,
            physicsEnabled: true
          }
        }));
      },

      addPhysicsEntity: entityData => {
        const entityId = idManager.generatePhysicsID(entityData.position, entityData.velocity).id;
        const entity: PhysicsEntity = {
          id: entityId,
          type: entityData.type || 'default',
          position: entityData.position || { x: 0, y: 0, z: 0 },
          rotation: entityData.rotation || { x: 0, y: 0, z: 0, w: 1 },
          scale: entityData.scale || { x: 1, y: 1, z: 1 },
          velocity: entityData.velocity || { x: 0, y: 0, z: 0 },
          mass: entityData.mass || 1,
          restitution: entityData.restitution || 0.8,
          friction: entityData.friction || 0.5,
          active: entityData.active !== undefined ? entityData.active : true,
          lod: entityData.lod || 0,
          visible: entityData.visible !== undefined ? entityData.visible : true,
          properties: entityData.properties || {}
        };

        set(state => {
          const newEntities = new Map(state.physics.entities);
          newEntities.set(entityId, entity);
          return {
            physics: {
              ...state.physics,
              entities: newEntities,
              performance: {
                ...state.physics.performance,
                entityCount: newEntities.size
              }
            }
          };
        });

        return entityId;
      },

      removePhysicsEntity: id => {
        set(state => {
          const newEntities = new Map(state.physics.entities);
          newEntities.delete(id);
          return {
            physics: {
              ...state.physics,
              entities: newEntities,
              performance: {
                ...state.physics.performance,
                entityCount: newEntities.size
              }
            }
          };
        });
      },

      updatePhysicsEntity: (id, updates) => {
        set(state => {
          const newEntities = new Map(state.physics.entities);
          const entity = newEntities.get(id);
          if (entity) {
            newEntities.set(id, { ...entity, ...updates });
          }
          return {
            physics: {
              ...state.physics,
              entities: newEntities
            }
          };
        });
      },

      setEnvironment: environment => {
        set(state => ({
          environment: { ...state.environment, ...environment }
        }));
      },

      updateLighting: lighting => {
        set(state => ({
          environment: {
            ...state.environment,
            lighting: { ...state.environment.lighting, ...lighting }
          }
        }));
      },

      updateFog: fog => {
        set(state => ({
          environment: {
            ...state.environment,
            fog: { ...state.environment.fog, ...fog }
          }
        }));
      },

      setRenderingMode: mode => {
        set(state => ({
          campaign: {
            ...state.campaign,
            type: mode
          }
        }));
      },

      updateRenderingConfig: config => {
        set(state => ({
          rendering: { ...state.rendering, ...config }
        }));
      },

      switchCampaign: campaignId => {
        set(state => ({
          campaign: { ...state.campaign, id: campaignId }
        }));
      },

      updateCampaignPhysics: physics => {
        set(state => ({
          campaign: {
            ...state.campaign,
            physics: { ...state.campaign.physics, ...physics }
          }
        }));
      },

      updatePerformance: (type, metrics) => {
        set(state => {
          if (type === 'physics') {
            return {
              physics: {
                ...state.physics,
                performance: { ...state.physics.performance, ...metrics }
              }
            };
          } else if (type === 'rendering') {
            return {
              rendering: {
                ...state.rendering,
                performance: { ...state.rendering.performance, ...metrics }
              }
            };
          }
          return state;
        });
      },

      emitStateChange: (type, data) => {
        // Emit state change events for real-time synchronization
        console.log(`[UnifiedState] State change: ${type}`, data);
      },

      subscribeToStateChanges: callback => {
        // Return unsubscribe function
        // TODO: Implement actual subscription mechanism
        console.log('[UnifiedState] Subscribed to state changes', callback);
        return () => {
          console.log('[UnifiedState] Unsubscribed from state changes');
        };
      }
    }),
    {
      name: 'unified-state-manager'
    }
  )
);

// Export hooks for specific state slices
export const usePhysicsState = () => useUnifiedState(state => state.physics);
export const useEnvironmentState = () => useUnifiedState(state => state.environment);
export const useRenderingState = () => useUnifiedState(state => state.rendering);
export const useCampaignState = () => useUnifiedState(state => state.campaign);
export const useConnectionState = () => useUnifiedState(state => state.connection);
