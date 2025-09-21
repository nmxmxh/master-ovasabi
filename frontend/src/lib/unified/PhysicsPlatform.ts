/**
 * Physics Platform Integration
 *
 * Integrates all physics systems for the distributed physics platform
 * Provides a unified API for physics, environment, and rendering
 */

import { lodManager, type LODEntity } from './LODManager';
import { wasmSendMessage } from '../wasmBridge';
// import * as CANNON from 'cannon-es';

export interface PhysicsPlatformConfig {
  campaignId: string;
  maxEntities: number;
  worldBounds: {
    min: { x: number; y: number; z: number };
    max: { x: number; y: number; z: number };
  };
  physicsRate: number;
  lodConfig: {
    distances: number[];
    polygonCounts: number[];
    textureSizes: number[];
  };
  rendering: {
    antialias: boolean;
    shadows: boolean;
    postProcessing: boolean;
  };
}

export interface PhysicsEntity {
  id: string;
  type: string;
  position: { x: number; y: number; z: number };
  rotation: { x: number; y: number; z: number; w: number };
  scale: { x: number; y: number; z: number };
  velocity: { x: number; y: number; z: number };
  mass: number;
  restitution: number;
  friction: number;
  active: boolean;
  lod: number;
  visible: boolean;
  properties: Record<string, any>;
}

export interface EnvironmentChunk {
  id: string;
  position: { x: number; y: number; z: number };
  bounds: {
    min: { x: number; y: number; z: number };
    max: { x: number; y: number; z: number };
  };
  lod: number;
  data: ArrayBuffer;
  size: number;
  compressed: boolean;
  loaded: boolean;
}

export interface PhysicsPerformance {
  fps: number;
  frameTime: number;
  entityCount: number;
  visibleEntities: number;
  culledEntities: number;
  averageLOD: number;
  memoryUsage: number;
  renderTime: number;
  physicsTime: number;
  lastUpdate: number;
}

export class PhysicsPlatform {
  private config: PhysicsPlatformConfig;
  private entities: Map<string, PhysicsEntity> = new Map();
  private chunks: Map<string, EnvironmentChunk> = new Map();
  private performance: PhysicsPerformance;
  private isInitialized: boolean = false;
  private isRunning: boolean = false;
  private lastUpdate: number = 0;
  private updateInterval: number = 16; // 60 FPS

  // Physics mode
  private physicsMode: 'godot' | 'cannon' = 'godot';
  private godotConnected: boolean = false;

  // Physics event handling
  private eventQueue: any[] = [];

  // Cannon.js fallback
  private world: any = null;
  private bodies: Map<string, any> = new Map();
  private materials: Map<string, any> = new Map();

  // LOD management
  public lodManager = lodManager;

  // Event handling
  private eventListeners: Map<string, Function[]> = new Map();

  constructor(config: PhysicsPlatformConfig) {
    this.config = config;
    this.performance = this.createDefaultPerformance();
    this.initialize();
  }

  private createDefaultPerformance(): PhysicsPerformance {
    return {
      fps: 60,
      frameTime: 16.67,
      entityCount: 0,
      visibleEntities: 0,
      culledEntities: 0,
      averageLOD: 0,
      memoryUsage: 0,
      renderTime: 0,
      physicsTime: 0,
      lastUpdate: Date.now()
    };
  }

  private async initialize(): Promise<void> {
    try {
      // Initialize LOD manager
      lodManager.updateLODConfig({
        distances: this.config.lodConfig.distances,
        polygonCounts: this.config.lodConfig.polygonCounts,
        textureSizes: this.config.lodConfig.textureSizes
      });

      // Set up event listeners
      this.setupEventListeners();

      // Try to initialize Godot physics first
      await this.initializeGodotPhysics();

      // If Godot fails, fallback to Cannon.js
      if (!this.godotConnected) {
        console.warn('[PhysicsPlatform] Godot not available, falling back to Cannon.js');
        this.physicsMode = 'cannon';
        this.initializeCannonPhysics();
      }

      this.isInitialized = true;
      console.log(`[PhysicsPlatform] Initialized successfully - using ${this.physicsMode} physics`);
    } catch (error) {
      console.error('[PhysicsPlatform] Initialization failed:', error);
      throw error;
    }
  }

  private async initializeGodotPhysics(): Promise<void> {
    // Send physics initialization message to WASM/Godot
    const initMessage = {
      type: 'physics:system:start',
      campaignId: this.config.campaignId,
      config: {
        maxEntities: this.config.maxEntities,
        worldBounds: this.config.worldBounds,
        physicsRate: this.config.physicsRate,
        lodConfig: this.config.lodConfig
      }
    };

    wasmSendMessage(initMessage);

    // Set a timeout to check if Godot responds
    return new Promise(resolve => {
      const timeout = setTimeout(() => {
        this.godotConnected = false;
        resolve();
      }, 2000); // 2 second timeout

      // Listen for Godot response
      const checkConnection = () => {
        if (this.godotConnected) {
          clearTimeout(timeout);
          resolve();
        }
      };

      // Check every 100ms
      const interval = setInterval(checkConnection, 100);

      // Clear interval after timeout
      setTimeout(() => {
        clearInterval(interval);
      }, 2000);
    });
  }

  private initializeCannonPhysics(): void {
    // Create physics world
    this.world = new (globalThis as any).CANNON.World();

    // Set gravity
    this.world.gravity.set(0, -9.81, 0);

    // Set broadphase
    this.world.broadphase = new (globalThis as any).CANNON.NaiveBroadphase();

    // Set solver
    (this.world.solver as any).iterations = 10;
    (this.world.solver as any).tolerance = 0.1;

    // Set default material
    const defaultMaterial = new (globalThis as any).CANNON.Material('default');
    this.materials.set('default', defaultMaterial);

    // Set up contact materials
    this.setupContactMaterials();

    // Add world bounds
    this.addWorldBounds();

    // Set up collision detection
    this.setupCollisionDetection();

    console.log('[PhysicsPlatform] Cannon.js physics initialized as fallback');
  }

  private setupContactMaterials(): void {
    if (!this.world) return;

    const defaultMaterial = this.materials.get('default')!;

    // Default contact material
    const defaultContact = new (globalThis as any).CANNON.ContactMaterial(
      defaultMaterial,
      defaultMaterial,
      {
        friction: 0.5,
        restitution: 0.8,
        contactEquationStiffness: 1e8,
        contactEquationRelaxation: 3
      }
    );

    this.world.addContactMaterial(defaultContact);
  }

  private addWorldBounds(): void {
    if (!this.world) return;

    const { min, max } = this.config.worldBounds;
    const thickness = 1;

    // Ground
    const groundShape = new (globalThis as any).CANNON.Plane();
    const groundBody = new (globalThis as any).CANNON.Body({ mass: 0 });
    groundBody.addShape(groundShape);
    groundBody.position.set(0, min.y, 0);
    groundBody.quaternion.setFromAxisAngle(
      new (globalThis as any).CANNON.Vec3(1, 0, 0),
      -Math.PI / 2
    );
    this.world.addBody(groundBody);

    // Walls
    const wallShape = new (globalThis as any).CANNON.Box(
      new (globalThis as any).CANNON.Vec3(thickness, (max.y - min.y) / 2, (max.z - min.z) / 2)
    );

    // Left wall
    const leftWall = new (globalThis as any).CANNON.Body({ mass: 0 });
    leftWall.addShape(wallShape);
    leftWall.position.set(min.x - thickness, (max.y + min.y) / 2, (max.z + min.z) / 2);
    this.world.addBody(leftWall);

    // Right wall
    const rightWall = new (globalThis as any).CANNON.Body({ mass: 0 });
    rightWall.addShape(wallShape);
    rightWall.position.set(max.x + thickness, (max.y + min.y) / 2, (max.z + min.z) / 2);
    this.world.addBody(rightWall);

    // Front wall
    const frontWall = new (globalThis as any).CANNON.Body({ mass: 0 });
    frontWall.addShape(
      new (globalThis as any).CANNON.Box(
        new (globalThis as any).CANNON.Vec3((max.x - min.x) / 2, (max.y - min.y) / 2, thickness)
      )
    );
    frontWall.position.set((max.x + min.x) / 2, (max.y + min.y) / 2, min.z - thickness);
    this.world.addBody(frontWall);

    // Back wall
    const backWall = new (globalThis as any).CANNON.Body({ mass: 0 });
    backWall.addShape(
      new (globalThis as any).CANNON.Box(
        new (globalThis as any).CANNON.Vec3((max.x - min.x) / 2, (max.y - min.y) / 2, thickness)
      )
    );
    backWall.position.set((max.x + min.x) / 2, (max.y + min.y) / 2, max.z + thickness);
    this.world.addBody(backWall);
  }

  private setupCollisionDetection(): void {
    if (!this.world) return;

    this.world.addEventListener('postStep', () => {
      this.handleCollisions();
    });
  }

  private setupEventListeners(): void {
    // Listen for LOD changes
    window.addEventListener('lodChange', (event: Event) => {
      this.handleLODChange((event as CustomEvent).detail);
    });

    // Listen for physics events from Godot via WASM
    window.addEventListener('wasmMessage', (event: Event) => {
      this.handleWASMMessage((event as CustomEvent).detail);
    });

    // Listen for campaign state changes
    window.addEventListener('campaignStateChange', (event: Event) => {
      this.handleCampaignStateChange((event as CustomEvent).detail);
    });
  }

  public start(): void {
    if (!this.isInitialized) {
      throw new Error('PhysicsPlatform not initialized');
    }

    this.isRunning = true;
    lodManager.start();
    this.update();
    console.log('[PhysicsPlatform] Started');
  }

  public stop(): void {
    this.isRunning = false;
    lodManager.stop();
    console.log('[PhysicsPlatform] Stopped');
  }

  private update(): void {
    if (!this.isRunning) return;

    const now = Date.now();
    const deltaTime = now - this.lastUpdate;

    if (deltaTime >= this.updateInterval) {
      this.processPhysics(deltaTime);
      this.updatePerformance(deltaTime);
      this.lastUpdate = now;
    }

    requestAnimationFrame(() => this.update());
  }

  private processPhysics(deltaTime: number): void {
    if (this.physicsMode === 'godot') {
      // Godot handles physics, we just process events
      this.processGodotEvents();
    } else if (this.physicsMode === 'cannon' && this.world) {
      // Step the Cannon.js physics world
      this.world.step(deltaTime / 1000);

      // Update entities from physics bodies
      for (const [id, body] of this.bodies) {
        const entity = this.entities.get(id);
        if (!entity || !entity.active) continue;

        // Update entity from physics body
        entity.position.x = body.position.x;
        entity.position.y = body.position.y;
        entity.position.z = body.position.z;

        entity.rotation.x = body.quaternion.x;
        entity.rotation.y = body.quaternion.y;
        entity.rotation.z = body.quaternion.z;
        entity.rotation.w = body.quaternion.w;

        entity.velocity.x = body.velocity.x;
        entity.velocity.y = body.velocity.y;
        entity.velocity.z = body.velocity.z;

        // Update LOD
        this.updateEntityLOD(entity);
      }
    }

    // Send physics updates to WASM (for Godot mode)
    if (this.physicsMode === 'godot') {
      this.sendPhysicsUpdates();
    }
  }

  private processGodotEvents(): void {
    // Process queued events from Godot
    while (this.eventQueue.length > 0) {
      const event = this.eventQueue.shift();
      this.handleGodotEvent(event);
    }
  }

  private handleGodotEvent(event: any): void {
    switch (event.type) {
      case 'physics:entity:update':
        this.updateEntityFromGodot(event);
        break;
      case 'physics:entity:spawn':
        this.spawnEntityFromGodot(event);
        break;
      case 'physics:entity:destroy':
        this.destroyEntityFromGodot(event);
        break;
      case 'physics:collision':
        this.handleCollisionFromGodot(event);
        break;
      case 'physics:system:ready':
        this.godotConnected = true;
        console.log('[PhysicsPlatform] Godot physics system ready');
        break;
    }
  }

  private updateEntityFromGodot(event: any): void {
    const entity = this.entities.get(event.entity_id);
    if (!entity) return;

    // Update entity from Godot data
    if (event.position) {
      entity.position.x = event.position[0];
      entity.position.y = event.position[1];
      entity.position.z = event.position[2];
    }

    if (event.rotation) {
      entity.rotation.x = event.rotation[0];
      entity.rotation.y = event.rotation[1];
      entity.rotation.z = event.rotation[2];
      entity.rotation.w = event.rotation[3];
    }

    if (event.velocity) {
      entity.velocity.x = event.velocity[0];
      entity.velocity.y = event.velocity[1];
      entity.velocity.z = event.velocity[2];
    }

    // Update LOD
    this.updateEntityLOD(entity);
  }

  private spawnEntityFromGodot(event: any): void {
    const entity: PhysicsEntity = {
      id: event.entity_id,
      type: event.properties?.type || 'default',
      position: {
        x: event.position[0],
        y: event.position[1],
        z: event.position[2]
      },
      rotation: {
        x: event.rotation[0],
        y: event.rotation[1],
        z: event.rotation[2],
        w: event.rotation[3]
      },
      scale: {
        x: event.scale?.[0] || 1,
        y: event.scale?.[1] || 1,
        z: event.scale?.[2] || 1
      },
      velocity: {
        x: event.velocity?.[0] || 0,
        y: event.velocity?.[1] || 0,
        z: event.velocity?.[2] || 0
      },
      mass: event.properties?.mass || 1,
      restitution: event.properties?.restitution || 0.8,
      friction: event.properties?.friction || 0.5,
      active: true,
      lod: 0,
      visible: true,
      properties: event.properties || {}
    };

    this.entities.set(entity.id, entity);
    this.updateEntityLOD(entity);
    console.log('[PhysicsPlatform] Spawned entity from Godot:', entity.id);
  }

  private destroyEntityFromGodot(event: any): void {
    const entityId = event.entity_id;
    this.entities.delete(entityId);
    lodManager.removeEntity(entityId);
    console.log('[PhysicsPlatform] Destroyed entity from Godot:', entityId);
  }

  private handleCollisionFromGodot(event: any): void {
    // Handle collision events from Godot
    console.log('[PhysicsPlatform] Collision from Godot:', event);

    // Emit collision event
    window.dispatchEvent(
      new CustomEvent('physicsCollision', {
        detail: event
      })
    );
  }

  private updateEntityLOD(entity: PhysicsEntity): void {
    const lodEntity: LODEntity = {
      id: entity.id,
      position: entity.position,
      type: entity.type,
      currentLOD: entity.lod,
      targetLOD: entity.lod,
      lastUpdate: Date.now(),
      properties: entity.properties
    };

    lodManager.updateEntity(entity.id, lodEntity);
  }

  private sendPhysicsUpdates(): void {
    const updates = Array.from(this.entities.values()).map(entity => ({
      type: 'physics:entity:update',
      entityId: entity.id,
      position: entity.position,
      rotation: entity.rotation,
      velocity: entity.velocity,
      properties: entity.properties,
      timestamp: Date.now()
    }));

    if (updates.length > 0) {
      wasmSendMessage({
        type: 'physics:batch',
        updates: updates
      } as any);
    }
  }

  private updatePerformance(deltaTime: number): void {
    this.performance.fps = 1000 / deltaTime;
    this.performance.frameTime = deltaTime;
    this.performance.entityCount = this.entities.size;
    this.performance.visibleEntities = lodManager.getVisibleEntityCount();
    this.performance.culledEntities = lodManager.getCulledEntityCount();
    this.performance.averageLOD = lodManager.getAverageLOD();
    this.performance.memoryUsage = lodManager.getMemoryUsage();
    this.performance.lastUpdate = Date.now();
  }

  private handleLODChange(detail: any): void {
    const { entityId, newLOD } = detail;
    const entity = this.entities.get(entityId);

    if (entity) {
      entity.lod = newLOD;
      entity.visible = newLOD < this.config.lodConfig.distances.length;
    }
  }

  private handleWASMMessage(message: any): void {
    if (this.physicsMode === 'godot') {
      // Queue Godot events for processing
      if (message.type === 'physics:batch') {
        // Handle batch of events from Godot
        if (message.events && Array.isArray(message.events)) {
          this.eventQueue.push(...message.events);
        }
      } else if (message.type === 'physics:particle:batch') {
        // Handle particle batch from Godot
        this.handleParticleBatch(message);
      } else if (message.type === 'physics:particle:chunk') {
        // Handle particle chunk from Godot
        this.handleParticleChunk(message);
      } else if (message.type === 'particle:data:update') {
        // Handle processed particle data from WASM
        this.handleProcessedParticleData(message);
      } else if (message.type?.startsWith('physics:')) {
        // Handle individual physics events from Godot
        this.eventQueue.push(message);
      }
    } else {
      // Handle Cannon.js mode messages
      switch (message.type) {
        case 'physics:entity:spawn':
          this.spawnEntity(message);
          break;
        case 'physics:entity:update':
          this.updateEntity(message);
          break;
        case 'physics:entity:destroy':
          this.destroyEntity(message);
          break;
        case 'physics:environment:chunk':
          this.handleEnvironmentChunk(message);
          break;
        case 'physics:collision':
          this.handleCollision(message);
          break;
      }
    }
  }

  private handleCampaignStateChange(detail: any): void {
    // Update physics rules based on campaign state
    if (detail.physicsRules) {
      this.updatePhysicsRules(detail.physicsRules);
    }
  }

  private handleParticleBatch(message: any): void {
    console.log('[PhysicsPlatform] Handling particle batch from Godot:', {
      particleCount: message.payload?.particle_count,
      lodLevel: message.payload?.lod_level,
      compression: message.payload?.compression,
      source: message.payload?.source
    });

    // Store particle batch for processing
    this.eventQueue.push({
      type: 'particle:batch:received',
      payload: message.payload,
      metadata: message.metadata,
      timestamp: Date.now()
    });
  }

  private handleParticleChunk(message: any): void {
    console.log('[PhysicsPlatform] Handling particle chunk from Godot:', {
      chunkIndex: message.payload?.data?.chunk_index,
      totalChunks: message.payload?.data?.total_chunks,
      compressed: message.payload?.data?.compressed,
      originalSize: message.payload?.data?.original_size
    });

    // Store particle chunk for reassembly
    this.eventQueue.push({
      type: 'particle:chunk:received',
      payload: message.payload,
      metadata: message.metadata,
      timestamp: Date.now()
    });
  }

  private handleProcessedParticleData(message: any): void {
    console.log('[PhysicsPlatform] Handling processed particle data from WASM:', {
      count: message.payload?.count,
      lodLevel: message.payload?.lod_level,
      source: message.payload?.source,
      format: message.payload?.format
    });

    // Update LOD manager with new particle data
    if (this.lodManager && message.payload?.particles) {
      // Create LOD entities for particles using 10-value format
      const particleCount = message.payload.count || 0;
      const particles = message.payload.particles || [];
      const lodLevel = message.payload.lod_level || 0;
      const format = message.payload.format || '10_values_per_particle';

      // Update existing particle entities or create new ones
      for (let i = 0; i < particleCount; i++) {
        const entityId = `particle_${i}`;
        const dataIndex = i * 10; // 10 values per particle

        // Extract position from 10-value format (x,y,z,vx,vy,vz,phase,intensity,type,id)
        const position = {
          x: particles[dataIndex] || 0,
          y: particles[dataIndex + 1] || 0,
          z: particles[dataIndex + 2] || 0
        };

        const existingEntity = this.lodManager.getEntity(entityId);
        if (existingEntity) {
          this.lodManager.updateEntity(entityId, {
            position,
            currentLOD: lodLevel,
            lastUpdate: Date.now()
          });
        } else {
          this.lodManager.addEntity({
            id: entityId,
            position,
            type: 'particle',
            currentLOD: lodLevel,
            targetLOD: lodLevel,
            lastUpdate: Date.now(),
            properties: {
              source: message.payload.source || 'godot',
              index: i,
              format: format,
              // Store additional particle data
              velocity: {
                x: particles[dataIndex + 3] || 0,
                y: particles[dataIndex + 4] || 0,
                z: particles[dataIndex + 5] || 0
              },
              phase: particles[dataIndex + 6] || 0,
              intensity: particles[dataIndex + 7] || 0,
              particleType: particles[dataIndex + 8] || 0,
              particleId: particles[dataIndex + 9] || 0
            }
          });
        }
      }
    }

    // Emit particle data update event with 10-value format
    this.emit('particleDataUpdate', {
      particles: message.payload?.particles || [], // Full 10-value particle data
      count: message.payload?.count || 0,
      lodLevel: message.payload?.lod_level || 0,
      source: message.payload?.source || 'unknown',
      format: message.payload?.format || '10_values_per_particle'
    });
  }

  private updatePhysicsRules(rules: any): void {
    // Update physics configuration based on campaign rules
    if (rules.gravity) {
      // Apply gravity changes
    }

    if (rules.physicsRate) {
      this.config.physicsRate = rules.physicsRate;
    }

    if (rules.maxEntities) {
      this.config.maxEntities = rules.maxEntities;
    }
  }

  public spawnEntity(data: any): void {
    if (this.physicsMode === 'godot') {
      // Send spawn request to Godot
      const spawnMessage = {
        type: 'physics:entity:spawn',
        entity_id: data.entityId || data.id,
        position: data.position ? [data.position.x, data.position.y, data.position.z] : [0, 0, 0],
        rotation: data.rotation
          ? [data.rotation.x, data.rotation.y, data.rotation.z, data.rotation.w]
          : [0, 0, 0, 1],
        scale: data.scale ? [data.scale.x, data.scale.y, data.scale.z] : [1, 1, 1],
        velocity: data.velocity ? [data.velocity.x, data.velocity.y, data.velocity.z] : [0, 0, 0],
        properties: {
          type: data.type || 'default',
          mass: data.mass || 1,
          restitution: data.restitution || 0.8,
          friction: data.friction || 0.5,
          ...data.properties
        }
      };

      wasmSendMessage(spawnMessage);
      console.log('[PhysicsPlatform] Sent spawn request to Godot:', spawnMessage.entity_id);
    } else {
      // Create entity locally for Cannon.js
      const entity: PhysicsEntity = {
        id: data.entityId || data.id,
        type: data.type || 'default',
        position: data.position || { x: 0, y: 0, z: 0 },
        rotation: data.rotation || { x: 0, y: 0, z: 0, w: 1 },
        scale: data.scale || { x: 1, y: 1, z: 1 },
        velocity: data.velocity || { x: 0, y: 0, z: 0 },
        mass: data.mass || 1,
        restitution: data.restitution || 0.8,
        friction: data.friction || 0.5,
        active: true,
        lod: 0,
        visible: true,
        properties: data.properties || {}
      };

      this.entities.set(entity.id, entity);

      // Create physics body
      this.createPhysicsBody(entity);

      // Add to LOD manager
      this.updateEntityLOD(entity);

      console.log('[PhysicsPlatform] Spawned entity with Cannon.js:', entity.id);
    }
  }

  private createPhysicsBody(entity: PhysicsEntity): void {
    if (!this.world) return;

    let shape: any;

    // Create shape based on entity type
    switch (entity.type) {
      case 'sphere':
        shape = new (globalThis as any).CANNON.Sphere(entity.scale.x);
        break;
      case 'capsule':
        shape = new (globalThis as any).CANNON.Cylinder(
          entity.scale.x,
          entity.scale.x,
          entity.scale.y
        );
        break;
      case 'box':
      default:
        shape = new (globalThis as any).CANNON.Box(
          new (globalThis as any).CANNON.Vec3(
            entity.scale.x / 2,
            entity.scale.y / 2,
            entity.scale.z / 2
          )
        );
        break;
    }

    // Create material
    const material = this.materials.get('default')!;

    // Create body
    const body = new (globalThis as any).CANNON.Body({
      mass: entity.mass,
      material: material,
      shape: shape,
      position: new (globalThis as any).CANNON.Vec3(
        entity.position.x,
        entity.position.y,
        entity.position.z
      ),
      quaternion: new (globalThis as any).CANNON.Quaternion(
        entity.rotation.x,
        entity.rotation.y,
        entity.rotation.z,
        entity.rotation.w
      ),
      velocity: new (globalThis as any).CANNON.Vec3(
        entity.velocity.x,
        entity.velocity.y,
        entity.velocity.z
      )
    });

    // Set material properties
    body.material.friction = entity.friction;
    body.material.restitution = entity.restitution;

    // Add to world
    this.world.addBody(body);
    this.bodies.set(entity.id, body);
  }

  public updateEntity(data: any): void {
    if (this.physicsMode === 'godot') {
      // Send update request to Godot
      const updateMessage = {
        type: 'physics:entity:update',
        entity_id: data.entityId || data.id,
        position: data.position ? [data.position.x, data.position.y, data.position.z] : undefined,
        rotation: data.rotation
          ? [data.rotation.x, data.rotation.y, data.rotation.z, data.rotation.w]
          : undefined,
        velocity: data.velocity ? [data.velocity.x, data.velocity.y, data.velocity.z] : undefined,
        scale: data.scale ? [data.scale.x, data.scale.y, data.scale.z] : undefined,
        properties: data.properties
      };

      wasmSendMessage(updateMessage);
      console.log('[PhysicsPlatform] Sent update request to Godot:', updateMessage.entity_id);
    } else {
      // Update entity locally for Cannon.js
      const entity = this.entities.get(data.entityId || data.id);
      if (!entity) return;

      // Update entity properties
      if (data.position) entity.position = data.position;
      if (data.rotation) entity.rotation = data.rotation;
      if (data.velocity) entity.velocity = data.velocity;
      if (data.scale) entity.scale = data.scale;
      if (data.properties) entity.properties = { ...entity.properties, ...data.properties };

      // Update physics body
      const body = this.bodies.get(entity.id);
      if (body) {
        if (data.position) {
          body.position.set(entity.position.x, entity.position.y, entity.position.z);
        }
        if (data.rotation) {
          body.quaternion.set(
            entity.rotation.x,
            entity.rotation.y,
            entity.rotation.z,
            entity.rotation.w
          );
        }
        if (data.velocity) {
          body.velocity.set(entity.velocity.x, entity.velocity.y, entity.velocity.z);
        }
      }

      // Update LOD manager
      this.updateEntityLOD(entity);
    }
  }

  public destroyEntity(data: any): void {
    const entityId = data.entityId || data.id;

    if (this.physicsMode === 'godot') {
      // Send destroy request to Godot
      const destroyMessage = {
        type: 'physics:entity:destroy',
        entity_id: entityId
      };

      wasmSendMessage(destroyMessage);
      console.log('[PhysicsPlatform] Sent destroy request to Godot:', entityId);
    } else {
      // Remove from physics world
      const body = this.bodies.get(entityId);
      if (body && this.world) {
        this.world.removeBody(body);
        this.bodies.delete(entityId);
      }

      // Remove from entities
      this.entities.delete(entityId);

      // Remove from LOD manager
      lodManager.removeEntity(entityId);

      console.log('[PhysicsPlatform] Destroyed entity with Cannon.js:', entityId);
    }
  }

  private handleEnvironmentChunk(data: any): void {
    const chunk: EnvironmentChunk = {
      id: data.chunkId || data.id,
      position: data.position || { x: 0, y: 0, z: 0 },
      bounds: data.bounds || {
        min: { x: 0, y: 0, z: 0 },
        max: { x: 0, y: 0, z: 0 }
      },
      lod: data.lod || 0,
      data: data.data || new ArrayBuffer(0),
      size: data.size || 0,
      compressed: data.compressed || false,
      loaded: true
    };

    this.chunks.set(chunk.id, chunk);
    console.log('[PhysicsPlatform] Loaded environment chunk:', chunk.id);
  }

  private handleCollisions(): void {
    // Get all contacts from the world
    const contacts = this.world.contacts;

    for (let i = 0; i < contacts.length; i++) {
      const contact = contacts[i];
      const bodyA = contact.bi;
      const bodyB = contact.bj;

      // Find entities for these bodies
      let entityA: PhysicsEntity | undefined;
      let entityB: PhysicsEntity | undefined;

      for (const [id, body] of this.bodies) {
        if (body === bodyA) {
          entityA = this.entities.get(id);
        }
        if (body === bodyB) {
          entityB = this.entities.get(id);
        }
      }

      if (entityA && entityB) {
        // Create collision event
        const contactPoint = (contact as any).getContactPoint
          ? (contact as any).getContactPoint()
          : { x: 0, y: 0, z: 0 };
        const contactNormal = (contact as any).getContactNormal
          ? (contact as any).getContactNormal()
          : { x: 0, y: 0, z: 0 };
        const impactVelocity = (contact as any).getImpactVelocityAlongNormal
          ? (contact as any).getImpactVelocityAlongNormal()
          : 0;

        const collisionData = {
          type: 'physics:collision',
          entityA: entityA.id,
          entityB: entityB.id,
          position: {
            x: contactPoint.x,
            y: contactPoint.y,
            z: contactPoint.z
          },
          normal: {
            x: contactNormal.x,
            y: contactNormal.y,
            z: contactNormal.z
          },
          force: impactVelocity,
          timestamp: Date.now()
        };

        // Send collision event
        wasmSendMessage(collisionData);

        // Emit collision event
        window.dispatchEvent(
          new CustomEvent('physicsCollision', {
            detail: collisionData
          })
        );

        console.log('[PhysicsPlatform] Collision detected:', entityA.id, 'vs', entityB.id);
      }
    }
  }

  private handleCollision(data: any): void {
    // Handle collision events from WASM
    console.log('[PhysicsPlatform] Collision event received:', data);
  }

  public getEntity(entityId: string): PhysicsEntity | undefined {
    return this.entities.get(entityId);
  }

  public getAllEntities(): PhysicsEntity[] {
    return Array.from(this.entities.values());
  }

  public getVisibleEntities(): PhysicsEntity[] {
    return this.getAllEntities().filter(entity => entity.visible);
  }

  public getChunk(chunkId: string): EnvironmentChunk | undefined {
    return this.chunks.get(chunkId);
  }

  public getAllChunks(): EnvironmentChunk[] {
    return Array.from(this.chunks.values());
  }

  public getPerformance(): PhysicsPerformance {
    return this.performance;
  }

  public getConfig(): PhysicsPlatformConfig {
    return this.config;
  }

  public updateConfig(updates: Partial<PhysicsPlatformConfig>): void {
    this.config = { ...this.config, ...updates };
  }

  public getIsInitialized(): boolean {
    return this.isInitialized;
  }

  public getIsRunning(): boolean {
    return this.isRunning;
  }

  public getEntityCount(): number {
    return this.entities.size;
  }

  public getChunkCount(): number {
    return this.chunks.size;
  }

  public clear(): void {
    if (this.physicsMode === 'godot') {
      // Send clear request to Godot
      const clearMessage = {
        type: 'physics:system:clear',
        campaignId: this.config.campaignId
      };

      wasmSendMessage(clearMessage);
      console.log('[PhysicsPlatform] Sent clear request to Godot');
    } else {
      // Clear physics bodies
      if (this.world) {
        for (const [, body] of this.bodies) {
          this.world.removeBody(body);
        }
      }
      this.bodies.clear();
    }

    // Clear entities
    this.entities.clear();
    this.chunks.clear();

    // Clear LOD manager
    lodManager.clear();

    // Reset performance
    this.performance = this.createDefaultPerformance();

    console.log(`[PhysicsPlatform] Cleared all entities using ${this.physicsMode} physics`);
  }

  public getPhysicsMode(): 'godot' | 'cannon' {
    return this.physicsMode;
  }

  public isGodotConnected(): boolean {
    return this.godotConnected;
  }

  public getOptimizationSuggestions(): string[] {
    return lodManager.getOptimizationSuggestions();
  }

  public optimizeForPerformance(): void {
    lodManager.optimizeForPerformance();
  }

  // Event handling methods
  public emit(event: string, data?: any): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(data);
        } catch (error) {
          console.error(`[PhysicsPlatform] Error in event listener for ${event}:`, error);
        }
      });
    }
  }

  public on(event: string, listener: Function): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event)!.push(listener);
  }

  public off(event: string, listener: Function): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  public removeAllListeners(event?: string): void {
    if (event) {
      this.eventListeners.delete(event);
    } else {
      this.eventListeners.clear();
    }
  }
}

// Export default configuration
export const defaultPhysicsPlatformConfig: PhysicsPlatformConfig = {
  campaignId: 'default',
  maxEntities: 1000,
  worldBounds: {
    min: { x: -100, y: -100, z: -100 },
    max: { x: 100, y: 100, z: 100 }
  },
  physicsRate: 60,
  lodConfig: {
    distances: [50, 100, 200, 500, 1000],
    polygonCounts: [1000, 500, 250, 100, 50],
    textureSizes: [512, 256, 128, 64, 32]
  },
  rendering: {
    antialias: true,
    shadows: true,
    postProcessing: true
  }
};
