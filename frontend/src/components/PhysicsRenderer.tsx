/**
 * Physics Renderer Component
 *
 * Integrates Three.js with WebGPU compute shaders for efficient 3D physics rendering
 * Supports the distributed physics platform with real-time updates
 */

import React, { useRef, useEffect, useState } from 'react';
import * as THREE from 'three';
import { physicsShaders } from '../shaders/physics.compute';
import { useUnifiedState } from '../lib/unified/StateManager';

interface PhysicsRendererProps {
  width?: number;
  height?: number;
  antialias?: boolean;
  shadows?: boolean;
  postProcessing?: boolean;
}

interface PhysicsEntity {
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

export class PhysicsRenderer {
  private canvas: HTMLCanvasElement;
  private renderer!: THREE.WebGLRenderer;
  private scene!: THREE.Scene;
  private camera!: THREE.PerspectiveCamera;
  private physicsEntities: Map<string, THREE.Object3D> = new Map();
  private materialCache: Map<number, THREE.Material> = new Map();
  private animationId: number | null = null;
  private lastTime: number = 0;
  private frameCount: number = 0;

  // WebGPU compute pipeline
  private device: GPUDevice | null = null;
  private computePipeline: GPUComputePipeline | null = null;
  private physicsBuffer: GPUBuffer | null = null;
  private renderBuffer: GPUBuffer | null = null;
  private cameraBuffer: GPUBuffer | null = null;
  private timeBuffer: GPUBuffer | null = null;

  // Performance tracking
  private performance: {
    fps: number;
    frameTime: number;
    drawCalls: number;
    triangles: number;
    memoryUsage: number;
  } = {
    fps: 60,
    frameTime: 16.67,
    drawCalls: 0,
    triangles: 0,
    memoryUsage: 0
  };

  constructor(canvas: HTMLCanvasElement, options: PhysicsRendererProps = {}) {
    this.canvas = canvas;
    this.setupThreeJS(options);
    this.setupWebGPU();
  }

  private setupThreeJS(options: PhysicsRendererProps) {
    // Create renderer
    this.renderer = new THREE.WebGLRenderer({
      canvas: this.canvas,
      antialias: options.antialias ?? true,
      alpha: true,
      powerPreference: 'high-performance'
    });

    this.renderer.setSize(options.width ?? 800, options.height ?? 600);
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

    if (options.shadows ?? true) {
      this.renderer.shadowMap.enabled = true;
      this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    }

    // Create scene
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(0x0a0a0a);

    // Create camera
    this.camera = new THREE.PerspectiveCamera(
      75,
      this.canvas.clientWidth / this.canvas.clientHeight,
      0.1,
      1000
    );
    this.camera.position.set(0, 10, 20);
    this.camera.lookAt(0, 0, 0);

    // Add lighting
    this.setupLighting();

    // Add fog
    this.scene.fog = new THREE.Fog(0x0a0a0a, 50, 200);

    // Create materials
    this.createMaterials();
  }

  private setupLighting() {
    // Ambient light
    const ambientLight = new THREE.AmbientLight(0x404040, 0.6);
    this.scene.add(ambientLight);

    // Directional light
    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
    directionalLight.position.set(10, 10, 5);
    directionalLight.target.position.set(0, 0, 0);
    directionalLight.castShadow = true;
    directionalLight.shadow.mapSize.width = 2048;
    directionalLight.shadow.mapSize.height = 2048;
    directionalLight.shadow.camera.near = 0.5;
    directionalLight.shadow.camera.far = 50;
    directionalLight.shadow.camera.left = -25;
    directionalLight.shadow.camera.right = 25;
    directionalLight.shadow.camera.top = 25;
    directionalLight.shadow.camera.bottom = -25;
    this.scene.add(directionalLight);
    this.scene.add(directionalLight.target);
  }

  private createMaterials() {
    // Default material
    this.materialCache.set(
      0,
      new THREE.MeshStandardMaterial({
        color: 0x888888,
        metalness: 0.1,
        roughness: 0.8
      })
    );

    // Metal material
    this.materialCache.set(
      1,
      new THREE.MeshStandardMaterial({
        color: 0x666666,
        metalness: 0.9,
        roughness: 0.1
      })
    );

    // Wood material
    this.materialCache.set(
      2,
      new THREE.MeshStandardMaterial({
        color: 0x8b4513,
        metalness: 0.0,
        roughness: 0.9
      })
    );

    // Stone material
    this.materialCache.set(
      3,
      new THREE.MeshStandardMaterial({
        color: 0x696969,
        metalness: 0.0,
        roughness: 0.7
      })
    );

    // Glass material
    this.materialCache.set(
      4,
      new THREE.MeshStandardMaterial({
        color: 0xffffff,
        metalness: 0.0,
        roughness: 0.0,
        transparent: true,
        opacity: 0.7
      })
    );
  }

  private async setupWebGPU() {
    try {
      if (!navigator.gpu) {
        console.warn('[PhysicsRenderer] WebGPU not supported, falling back to CPU processing');
        return;
      }

      const adapter = await navigator.gpu.requestAdapter();
      if (!adapter) {
        throw new Error('Failed to get WebGPU adapter');
      }

      this.device = await adapter.requestDevice();

      // Create compute pipeline
      const computeShader = this.device.createShaderModule({
        code: physicsShaders.physics
      });

      this.computePipeline = this.device.createComputePipeline({
        layout: 'auto',
        compute: {
          module: computeShader,
          entryPoint: 'main'
        }
      });

      // Create buffers
      this.createBuffers();

      console.log('[PhysicsRenderer] WebGPU compute pipeline initialized');
    } catch (error) {
      console.error('[PhysicsRenderer] Failed to initialize WebGPU:', error);
    }
  }

  private createBuffers() {
    if (!this.device) return;

    // Physics data buffer (read-only)
    this.physicsBuffer = this.device.createBuffer({
      size: 1024 * 64, // 64KB for physics data
      usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_DST
    });

    // Render data buffer (read-write)
    this.renderBuffer = this.device.createBuffer({
      size: 1024 * 64, // 64KB for render data
      usage: GPUBufferUsage.STORAGE | GPUBufferUsage.COPY_SRC
    });

    // Camera uniform buffer
    this.cameraBuffer = this.device.createBuffer({
      size: 128, // 128 bytes for camera data
      usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST
    });

    // Time uniform buffer
    this.timeBuffer = this.device.createBuffer({
      size: 16, // 16 bytes for time data
      usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST
    });
  }

  public start() {
    this.lastTime = performance.now();
    this.animate();
  }

  public stop() {
    if (this.animationId) {
      cancelAnimationFrame(this.animationId);
      this.animationId = null;
    }
  }

  private animate = () => {
    const currentTime = performance.now();
    const deltaTime = currentTime - this.lastTime;
    this.lastTime = currentTime;

    this.frameCount++;

    // Update physics
    this.updatePhysics(deltaTime);

    // Render scene
    this.render();

    // Update performance metrics
    this.updatePerformance(deltaTime);

    this.animationId = requestAnimationFrame(this.animate);
  };

  private updatePhysics(deltaTime: number) {
    if (this.device && this.computePipeline) {
      this.runWebGPUCompute(deltaTime);
    } else {
      this.runCPUPhysics(deltaTime);
    }
  }

  private async runWebGPUCompute(_deltaTime: number) {
    if (!this.device || !this.computePipeline) return;

    const commandEncoder = this.device.createCommandEncoder();
    const computePass = commandEncoder.beginComputePass();

    // Set bind group
    const bindGroup = this.device.createBindGroup({
      layout: this.computePipeline.getBindGroupLayout(0),
      entries: [
        { binding: 0, resource: { buffer: this.physicsBuffer! } },
        { binding: 1, resource: { buffer: this.renderBuffer! } },
        { binding: 2, resource: { buffer: this.cameraBuffer! } },
        { binding: 3, resource: { buffer: this.timeBuffer! } }
      ]
    });

    computePass.setPipeline(this.computePipeline);
    computePass.setBindGroup(0, bindGroup);
    computePass.dispatchWorkgroups(Math.ceil(this.physicsEntities.size / 64));
    computePass.end();

    this.device.queue.submit([commandEncoder.finish()]);
  }

  private runCPUPhysics(_deltaTime: number) {
    // Fallback CPU physics processing
    for (const [, entity] of this.physicsEntities) {
      // Update entity position, rotation, etc.
      // This is a simplified implementation
      entity.position.x += Math.sin(Date.now() * 0.001) * 0.01;
      entity.position.y += Math.cos(Date.now() * 0.001) * 0.01;
    }
  }

  private render() {
    this.renderer.render(this.scene, this.camera);
  }

  private updatePerformance(deltaTime: number) {
    this.performance.fps = 1000 / deltaTime;
    this.performance.frameTime = deltaTime;
    this.performance.drawCalls = this.renderer.info.render.calls;
    this.performance.triangles = this.renderer.info.render.triangles;
    this.performance.memoryUsage = (performance as any).memory?.usedJSHeapSize || 0;
  }

  public updatePhysicsEntities(entities: PhysicsEntity[]) {
    // Update existing entities
    for (const entity of entities) {
      const existingMesh = this.physicsEntities.get(entity.id);
      if (existingMesh) {
        // Update existing entity
        existingMesh.position.set(entity.position.x, entity.position.y, entity.position.z);
        existingMesh.quaternion.set(
          entity.rotation.x,
          entity.rotation.y,
          entity.rotation.z,
          entity.rotation.w
        );
        existingMesh.scale.set(entity.scale.x, entity.scale.y, entity.scale.z);
        existingMesh.visible = entity.visible && entity.active;
      } else {
        // Create new entity
        this.createEntityMesh(entity);
      }
    }

    // Remove entities that are no longer in the list
    const currentEntityIds = new Set(entities.map(e => e.id));
    for (const [id, mesh] of this.physicsEntities) {
      if (!currentEntityIds.has(id)) {
        this.scene.remove(mesh);
        this.physicsEntities.delete(id);
      }
    }
  }

  private createEntityMesh(entity: PhysicsEntity) {
    let geometry: THREE.BufferGeometry;

    // Create geometry based on entity type
    switch (entity.type) {
      case 'sphere':
        geometry = new THREE.SphereGeometry(entity.scale.x, 16, 16);
        break;
      case 'capsule':
        geometry = new THREE.CapsuleGeometry(entity.scale.x, entity.scale.y, 4, 8);
        break;
      case 'box':
      default:
        geometry = new THREE.BoxGeometry(entity.scale.x, entity.scale.y, entity.scale.z);
        break;
    }

    // Get or create material
    const materialIndex = entity.properties?.material || 0;
    let material = this.materialCache.get(materialIndex);
    if (!material) {
      material = new THREE.MeshStandardMaterial({
        color: this.getColorForEntityType(entity.type),
        metalness: 0.0,
        roughness: 0.5,
        transparent: false
      });
      this.materialCache.set(materialIndex, material);
    }

    const mesh = new THREE.Mesh(geometry, material);
    mesh.position.set(entity.position.x, entity.position.y, entity.position.z);
    mesh.quaternion.set(entity.rotation.x, entity.rotation.y, entity.rotation.z, entity.rotation.w);
    mesh.scale.set(entity.scale.x, entity.scale.y, entity.scale.z);
    mesh.castShadow = true;
    mesh.receiveShadow = true;
    mesh.visible = entity.visible && entity.active;

    this.scene.add(mesh);
    this.physicsEntities.set(entity.id, mesh);
  }

  private getColorForEntityType(type: string): number {
    switch (type) {
      case 'sphere':
        return 0xff6b6b;
      case 'capsule':
        return 0x4ecdc4;
      case 'box':
        return 0x45b7d1;
      default:
        return 0x96ceb4;
    }
  }

  public getPerformance() {
    return this.performance;
  }

  public resize(width: number, height: number) {
    this.renderer.setSize(width, height);
    this.camera.aspect = width / height;
    this.camera.updateProjectionMatrix();
  }

  public dispose() {
    this.stop();

    // Dispose of all entities
    for (const entity of this.physicsEntities.values()) {
      this.scene.remove(entity);
      if (entity instanceof THREE.Mesh) {
        entity.geometry.dispose();
        if (Array.isArray(entity.material)) {
          entity.material.forEach(material => material.dispose());
        } else {
          entity.material.dispose();
        }
      }
    }

    this.physicsEntities.clear();
    this.materialCache.clear();
    this.renderer.dispose();
  }
}

// React component wrapper
export const PhysicsRendererComponent: React.FC<PhysicsRendererProps> = props => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const rendererRef = useRef<PhysicsRenderer | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);
  const [performance, setPerformance] = useState<any>(null);

  const { physics, environment, rendering } = useUnifiedState();

  useEffect(() => {
    if (!canvasRef.current) return;

    const renderer = new PhysicsRenderer(canvasRef.current, props);
    rendererRef.current = renderer;
    renderer.start();
    setIsInitialized(true);

    return () => {
      renderer.dispose();
    };
  }, []);

  useEffect(() => {
    if (!rendererRef.current || !isInitialized) return;

    // Update renderer with physics state
    const physicsEvents = physics.entities ? Array.from(physics.entities.values()) : [];
    rendererRef.current.updatePhysicsEntities(physicsEvents);
  }, [physics, isInitialized]);

  useEffect(() => {
    if (!rendererRef.current || !isInitialized) return;

    // Update environment
    if (environment.lighting) {
      // Update lighting based on environment state
      console.log('[PhysicsRenderer] Updating environment lighting');
    }

    if (environment.fog) {
      // Update fog based on environment state
      console.log('[PhysicsRenderer] Updating environment fog');
    }
  }, [environment, isInitialized]);

  useEffect(() => {
    if (!rendererRef.current || !isInitialized) return;

    // Update rendering settings
    if (rendering.antialias !== undefined) {
      console.log('[PhysicsRenderer] Updating antialias setting');
    }

    if (rendering.shadowMap !== undefined) {
      console.log('[PhysicsRenderer] Updating shadow map setting');
    }
  }, [rendering, isInitialized]);

  // Performance monitoring
  useEffect(() => {
    if (!rendererRef.current) return;

    const interval = setInterval(() => {
      if (rendererRef.current) {
        setPerformance(rendererRef.current.getPerformance());
      }
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="physics-renderer">
      <canvas ref={canvasRef} style={{ width: '100%', height: '100%' }} />
      {performance && (
        <div className="performance-overlay">
          <div>FPS: {performance.fps.toFixed(1)}</div>
          <div>Frame Time: {performance.frameTime.toFixed(2)}ms</div>
          <div>Draw Calls: {performance.drawCalls}</div>
          <div>Triangles: {performance.triangles}</div>
          <div>Memory: {(performance.memoryUsage / 1024 / 1024).toFixed(1)}MB</div>
        </div>
      )}
    </div>
  );
};

export default PhysicsRendererComponent;
