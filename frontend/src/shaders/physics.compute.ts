/**
 * WebGPU Compute Shaders for Physics Rendering
 *
 * These shaders handle the transformation of physics data to render data
 * for efficient 3D rendering in the distributed physics platform
 */

// Physics entity data structure
export interface PhysicsEntityData {
  position: [number, number, number];
  rotation: [number, number, number, number]; // quaternion
  velocity: [number, number, number];
  scale: [number, number, number];
  mass: number;
  restitution: number;
  friction: number;
  entityType: number;
  active: number;
  lod: number;
}

// Render entity data structure
export interface RenderEntityData {
  position: [number, number, number];
  rotation: [number, number, number, number];
  scale: [number, number, number];
  material: number;
  lod: number;
  visible: number;
  culled: number;
}

// Physics compute shader for entity updates
export const physicsComputeShader = `
@group(0) @binding(0) var<storage, read> physicsData: array<PhysicsEntityData>;
@group(0) @binding(1) var<storage, read_write> renderData: array<RenderEntityData>;
@group(0) @binding(2) var<uniform> camera: CameraUniform;
@group(0) @binding(3) var<uniform> time: TimeUniform;

struct CameraUniform {
  position: vec3<f32>,
  direction: vec3<f32>,
  up: vec3<f32>,
  fov: f32,
  aspect: f32,
  near: f32,
  far: f32,
}

struct TimeUniform {
  deltaTime: f32,
  totalTime: f32,
  frameCount: u32,
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
  let index = global_id.x;
  if (index >= arrayLength(&physicsData)) {
    return;
  }
  
  let physics = physicsData[index];
  let render = &renderData[index];
  
  // Skip inactive entities
  if (physics.active == 0) {
    render.visible = 0;
    render.culled = 1;
    return;
  }
  
  // Update position based on velocity
  let newPosition = vec3<f32>(
    physics.position[0] + physics.velocity[0] * time.deltaTime,
    physics.position[1] + physics.velocity[1] * time.deltaTime,
    physics.position[2] + physics.velocity[2] * time.deltaTime
  );
  
  // Apply physics constraints (simplified)
  let constrainedPosition = applyPhysicsConstraints(newPosition, physics);
  
  // Update render data
  render.position = [constrainedPosition.x, constrainedPosition.y, constrainedPosition.z];
  render.rotation = physics.rotation;
  render.scale = physics.scale;
  render.material = getMaterialForEntityType(physics.entityType);
  render.lod = physics.lod;
  
  // Frustum culling
  let culled = isFrustumCulled(constrainedPosition, physics.scale, camera);
  render.culled = select(0, 1, culled);
  
  // LOD calculation based on distance
  let distance = length(constrainedPosition - camera.position);
  let lod = calculateLOD(distance, physics.entityType);
  render.lod = lod;
  
  // Visibility determination
  render.visible = select(1, 0, culled || lod > 3);
}

fn applyPhysicsConstraints(position: vec3<f32>, physics: PhysicsEntityData) -> vec3<f32> {
  // Simple gravity and bounds checking
  let gravity = vec3<f32>(0.0, -9.81, 0.0);
  let worldBounds = vec3<f32>(100.0, 100.0, 100.0);
  
  var constrainedPos = position;
  
  // Apply gravity
  constrainedPos += gravity * time.deltaTime * time.deltaTime * 0.5;
  
  // World bounds constraint
  constrainedPos = clamp(constrainedPos, -worldBounds, worldBounds);
  
  return constrainedPos;
}

fn getMaterialForEntityType(entityType: u32) -> u32 {
  switch (entityType) {
    case 0u: return 0u; // Default material
    case 1u: return 1u; // Metal material
    case 2u: return 2u; // Wood material
    case 3u: return 3u; // Stone material
    case 4u: return 4u; // Glass material
    default: return 0u;
  }
}

fn isFrustumCulled(position: vec3<f32>, scale: vec3<f32>, camera: CameraUniform) -> bool {
  // Simple frustum culling
  let distance = length(position - camera.position);
  let maxDistance = camera.far * 0.8; // Cull at 80% of far plane
  
  return distance > maxDistance;
}

fn calculateLOD(distance: f32, entityType: u32) -> u32 {
  // LOD distances based on entity type
  let lodDistances = array<f32, 4>(
    50.0,   // LOD 0: 0-50m
    100.0,  // LOD 1: 50-100m
    200.0,  // LOD 2: 100-200m
    500.0   // LOD 3: 200-500m
  );
  
  for (var i = 0u; i < 4u; i++) {
    if (distance <= lodDistances[i]) {
      return i;
    }
  }
  
  return 4u; // Culled
}
`;

// Particle system compute shader
export const particleComputeShader = `
@group(0) @binding(0) var<storage, read> particleData: array<ParticleData>;
@group(0) @binding(1) var<storage, read_write> renderParticles: array<RenderParticleData>;
@group(0) @binding(2) var<uniform> time: TimeUniform;
@group(0) @binding(3) var<uniform> physics: PhysicsUniform;

struct ParticleData {
  position: vec3<f32>,
  velocity: vec3<f32>,
  life: f32,
  size: f32,
  color: vec4<f32>,
  type: u32,
}

struct RenderParticleData {
  position: vec3<f32>,
  size: f32,
  color: vec4<f32>,
  alpha: f32,
  visible: u32,
}

struct PhysicsUniform {
  gravity: vec3<f32>,
  wind: vec3<f32>,
  damping: f32,
  worldBounds: vec3<f32>,
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
  let index = global_id.x;
  if (index >= arrayLength(&particleData)) {
    return;
  }
  
  let particle = particleData[index];
  let render = &renderParticles[index];
  
  // Skip dead particles
  if (particle.life <= 0.0) {
    render.visible = 0;
    return;
  }
  
  // Update particle physics
  var newPosition = particle.position;
  var newVelocity = particle.velocity;
  
  // Apply gravity
  newVelocity += physics.gravity * time.deltaTime;
  
  // Apply wind
  newVelocity += physics.wind * time.deltaTime;
  
  // Apply damping
  newVelocity *= physics.damping;
  
  // Update position
  newPosition += newVelocity * time.deltaTime;
  
  // Apply world bounds
  newPosition = clamp(newPosition, -physics.worldBounds, physics.worldBounds);
  
  // Update life
  let newLife = particle.life - time.deltaTime;
  
  // Update render data
  render.position = newPosition;
  render.size = particle.size * (newLife / particle.life); // Shrink as life decreases
  render.color = particle.color;
  render.alpha = newLife / particle.life;
  render.visible = select(0, 1, newLife > 0.0);
}
`;

// Collision detection compute shader
export const collisionComputeShader = `
@group(0) @binding(0) var<storage, read> entities: array<PhysicsEntityData>;
@group(0) @binding(1) var<storage, read_write> collisions: array<CollisionData>;
@group(0) @binding(2) var<uniform> time: TimeUniform;

struct CollisionData {
  entityA: u32,
  entityB: u32,
  position: vec3<f32>,
  normal: vec3<f32>,
  force: f32,
  valid: u32,
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
  let index = global_id.x;
  if (index >= arrayLength(&entities)) {
    return;
  }
  
  let entityA = entities[index];
  if (entityA.active == 0) {
    return;
  }
  
  // Check collisions with other entities
  for (var i = index + 1u; i < arrayLength(&entities); i++) {
    let entityB = entities[i];
    if (entityB.active == 0) {
      continue;
    }
    
    // Calculate distance between entities
    let distance = length(
      vec3<f32>(entityA.position[0], entityA.position[1], entityA.position[2]) -
      vec3<f32>(entityB.position[0], entityB.position[1], entityB.position[2])
    );
    
    // Check if collision occurred
    let collisionRadius = (entityA.scale[0] + entityB.scale[0]) * 0.5;
    if (distance < collisionRadius) {
      // Calculate collision data
      let collisionPos = (vec3<f32>(entityA.position[0], entityA.position[1], entityA.position[2]) +
                         vec3<f32>(entityB.position[0], entityB.position[1], entityB.position[2])) * 0.5;
      
      let normal = normalize(
        vec3<f32>(entityA.position[0], entityA.position[1], entityA.position[2]) -
        vec3<f32>(entityB.position[0], entityB.position[1], entityB.position[2])
      );
      
      let force = length(
        vec3<f32>(entityA.velocity[0], entityA.velocity[1], entityA.velocity[2]) -
        vec3<f32>(entityB.velocity[0], entityB.velocity[1], entityB.velocity[2])
      );
      
      // Store collision data
      let collisionIndex = index * arrayLength(&entities) + i;
      if (collisionIndex < arrayLength(&collisions)) {
        collisions[collisionIndex] = CollisionData(
          index,
          i,
          collisionPos,
          normal,
          force,
          1
        );
      }
    }
  }
}
`;

// LOD management compute shader
export const lodComputeShader = `
@group(0) @binding(0) var<storage, read> entities: array<PhysicsEntityData>;
@group(0) @binding(1) var<storage, read_write> renderData: array<RenderEntityData>;
@group(0) @binding(2) var<uniform> camera: CameraUniform;
@group(0) @binding(3) var<uniform> lodSettings: LODSettings;

struct LODSettings {
  distances: array<f32, 4>,
  polygonCounts: array<u32, 4>,
  textureSizes: array<u32, 4>,
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
  let index = global_id.x;
  if (index >= arrayLength(&entities)) {
    return;
  }
  
  let entity = entities[index];
  let render = &renderData[index];
  
  if (entity.active == 0) {
    render.visible = 0;
    return;
  }
  
  // Calculate distance from camera
  let entityPos = vec3<f32>(entity.position[0], entity.position[1], entity.position[2]);
  let distance = length(entityPos - camera.position);
  
  // Determine LOD level
  var lodLevel = 0u;
  for (var i = 0u; i < 4u; i++) {
    if (distance <= lodSettings.distances[i]) {
      lodLevel = i;
      break;
    }
  }
  
  // Update render data based on LOD
  render.lod = lodLevel;
  render.visible = select(0, 1, lodLevel < 4u);
  
  // Adjust scale based on LOD (simplified)
  let lodScale = 1.0 - f32(lodLevel) * 0.2;
  render.scale = [
    entity.scale[0] * lodScale,
    entity.scale[1] * lodScale,
    entity.scale[2] * lodScale
  ];
}
`;

// Occlusion culling compute shader
export const occlusionCullingShader = `
@group(0) @binding(0) var<storage, read> entities: array<PhysicsEntityData>;
@group(0) @binding(1) var<storage, read_write> renderData: array<RenderEntityData>;
@group(0) @binding(2) var<uniform> camera: CameraUniform;
@group(0) @binding(3) var<texture_2d> depthTexture: texture_depth_2d;
@group(0) @binding(4) var<sampler> depthSampler: sampler;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
  let index = global_id.x;
  if (index >= arrayLength(&entities)) {
    return;
  }
  
  let entity = entities[index];
  let render = &renderData[index];
  
  if (entity.active == 0) {
    render.visible = 0;
    return;
  }
  
  // Project entity position to screen space
  let entityPos = vec3<f32>(entity.position[0], entity.position[1], entity.position[2]);
  let screenPos = projectToScreenSpace(entityPos, camera);
  
  // Check if entity is on screen
  if (screenPos.x < 0.0 || screenPos.x > 1.0 || screenPos.y < 0.0 || screenPos.y > 1.0) {
    render.visible = 0;
    return;
  }
  
  // Sample depth texture at entity position
  let depth = textureSample(depthTexture, depthSampler, screenPos.xy).r;
  
  // Check if entity is occluded
  let entityDepth = screenPos.z;
  let occluded = entityDepth > depth + 0.01; // Small bias to prevent z-fighting
  
  render.visible = select(1, 0, occluded);
}

fn projectToScreenSpace(worldPos: vec3<f32>, camera: CameraUniform) -> vec4<f32> {
  // Convert world position to clip space
  let viewPos = worldPos - camera.position;
  let clipPos = vec4<f32>(viewPos, 1.0);
  
  // Apply perspective projection
  let fov = camera.fov;
  let aspect = camera.aspect;
  let near = camera.near;
  let far = camera.far;
  
  let f = 1.0 / tan(fov * 0.5);
  let zRange = far - near;
  
  let proj = mat4x4<f32>(
    vec4<f32>(f / aspect, 0.0, 0.0, 0.0),
    vec4<f32>(0.0, f, 0.0, 0.0),
    vec4<f32>(0.0, 0.0, -(far + near) / zRange, -1.0),
    vec4<f32>(0.0, 0.0, -(2.0 * far * near) / zRange, 0.0)
  );
  
  let clip = proj * clipPos;
  
  // Convert to screen space
  let screenPos = vec4<f32>(
    (clip.x / clip.w + 1.0) * 0.5,
    (clip.y / clip.w + 1.0) * 0.5,
    clip.z / clip.w,
    clip.w
  );
  
  return screenPos;
}
`;

// Export all shaders
export const physicsShaders = {
  physics: physicsComputeShader,
  particles: particleComputeShader,
  collision: collisionComputeShader,
  lod: lodComputeShader,
  occlusion: occlusionCullingShader
};


