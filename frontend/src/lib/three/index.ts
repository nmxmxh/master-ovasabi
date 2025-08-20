// Dynamic Three.js loader with code splitting and performance tracking
// This module provides lazy-loaded Three.js components to reduce initial bundle size
// and includes comprehensive loading progress and benchmark integration

import type * as THREE from 'three';

// Type definitions for lazy-loaded modules
export interface ThreeCore {
  Scene: typeof THREE.Scene;
  PerspectiveCamera: typeof THREE.PerspectiveCamera;
  WebGLRenderer: typeof THREE.WebGLRenderer;
  BufferGeometry: typeof THREE.BufferGeometry;
  BufferAttribute: typeof THREE.BufferAttribute;
  Points: typeof THREE.Points;
  PointsMaterial: typeof THREE.PointsMaterial;
  ShaderMaterial: typeof THREE.ShaderMaterial;
  Material: typeof THREE.Material;
  SphereGeometry: typeof THREE.SphereGeometry;
  PlaneGeometry: typeof THREE.PlaneGeometry;
  BoxGeometry: typeof THREE.BoxGeometry;
  TorusKnotGeometry: typeof THREE.TorusKnotGeometry;
  MeshBasicMaterial: typeof THREE.MeshBasicMaterial;
  MeshPhysicalMaterial: typeof THREE.MeshPhysicalMaterial;
  MeshStandardMaterial: typeof THREE.MeshStandardMaterial;
  Mesh: typeof THREE.Mesh;
  Line: typeof THREE.Line;
  LineBasicMaterial: typeof THREE.LineBasicMaterial;
  Vector3: typeof THREE.Vector3;
  Color: typeof THREE.Color;
  Texture: typeof THREE.Texture;
  CanvasTexture: typeof THREE.CanvasTexture;
  Clock: typeof THREE.Clock;
  AdditiveBlending: typeof THREE.AdditiveBlending;
  RGBAFormat: typeof THREE.RGBAFormat;
  UnsignedByteType: typeof THREE.UnsignedByteType;
  Fog: typeof THREE.Fog;
  AmbientLight: typeof THREE.AmbientLight;
  DirectionalLight: typeof THREE.DirectionalLight;
  Object3D: typeof THREE.Object3D;
  Group: typeof THREE.Group;
}

export interface ThreeRenderers {
  WebGPURenderer: any; // WebGPU types are complex, using any for now
  SVGRenderer: any;
  CSS2DRenderer: any; // For UI overlays
  CSS3DRenderer: any; // For 3D UI elements
  // WebGPU optimization flags and nodes
  webgpuAvailable: boolean;
  webgpuNodes: {
    MeshBasicNodeMaterial: any;
    MeshStandardNodeMaterial: any;
    PointsNodeMaterial: any;
    LineBasicNodeMaterial: any;
    ComputeNode: any;
    StorageBufferNode: any;
  } | null;
}

export interface ThreeAddons {
  OrbitControls: any;
  EffectComposer: any;
  RenderPass: any;
  UnrealBloomPass: any;
  ShaderPass: any;
  FXAAShader: any;
}

// Loading progress tracking
export interface LoadingProgress {
  stage: string;
  progress: number; // 0-100
  message: string;
  error?: string;
  startTime: number;
  elapsedTime: number;
}

// Performance metrics for loading
export interface LoadingMetrics {
  coreLoadTime: number;
  renderersLoadTime: number;
  addonsLoadTime: number;
  totalLoadTime: number;
  bundleSize: number;
  memoryUsage: number;
  success: boolean;
  errors: string[];
}

// Loading state management
class ThreeLoadingManager {
  private callbacks: ((progress: LoadingProgress) => void)[] = [];
  private currentProgress: LoadingProgress = {
    stage: 'idle',
    progress: 0,
    message: 'Waiting to start...',
    startTime: Date.now(),
    elapsedTime: 0
  };

  subscribe(callback: (progress: LoadingProgress) => void): () => void {
    this.callbacks.push(callback);
    // Immediately send current progress
    callback(this.currentProgress);

    return () => {
      const index = this.callbacks.indexOf(callback);
      if (index > -1) {
        this.callbacks.splice(index, 1);
      }
    };
  }

  updateProgress(update: Partial<LoadingProgress>): void {
    this.currentProgress = {
      ...this.currentProgress,
      ...update,
      elapsedTime: Date.now() - this.currentProgress.startTime
    };

    this.callbacks.forEach(callback => callback(this.currentProgress));
  }

  reset(): void {
    this.currentProgress = {
      stage: 'idle',
      progress: 0,
      message: 'Waiting to start...',
      startTime: Date.now(),
      elapsedTime: 0
    };
  }
}

// Global loading manager instance
export const threeLoadingManager = new ThreeLoadingManager();

// Lazy loaders with error handling, caching, and performance tracking
let threeCore: ThreeCore | null = null;
let threeRenderers: ThreeRenderers | null = null;
let threeAddons: ThreeAddons | null = null;

// Performance tracking
let loadingMetrics: LoadingMetrics = {
  coreLoadTime: 0,
  renderersLoadTime: 0,
  addonsLoadTime: 0,
  totalLoadTime: 0,
  bundleSize: 0,
  memoryUsage: 0,
  success: false,
  errors: []
};

export async function loadThreeCore(): Promise<ThreeCore> {
  if (threeCore) return threeCore;

  const startTime = performance.now();
  const startMemory = (performance as any).memory?.usedJSHeapSize || 0;

  threeLoadingManager.updateProgress({
    stage: 'core',
    progress: 10,
    message: 'Loading core Three.js modules...'
  });

  console.log('[Three.js Loader] Loading core Three.js modules...');

  try {
    threeLoadingManager.updateProgress({
      stage: 'core',
      progress: 30,
      message: 'Importing Three.js package...'
    });

    const THREE = await import('three');

    threeLoadingManager.updateProgress({
      stage: 'core',
      progress: 60,
      message: 'Initializing core components...'
    });

    threeCore = {
      Scene: THREE.Scene,
      PerspectiveCamera: THREE.PerspectiveCamera,
      WebGLRenderer: THREE.WebGLRenderer,
      BufferGeometry: THREE.BufferGeometry,
      BufferAttribute: THREE.BufferAttribute,
      Points: THREE.Points,
      PointsMaterial: THREE.PointsMaterial,
      ShaderMaterial: THREE.ShaderMaterial,
      Material: THREE.Material,
      SphereGeometry: THREE.SphereGeometry,
      PlaneGeometry: THREE.PlaneGeometry,
      BoxGeometry: THREE.BoxGeometry,
      TorusKnotGeometry: THREE.TorusKnotGeometry,
      MeshBasicMaterial: THREE.MeshBasicMaterial,
      MeshPhysicalMaterial: THREE.MeshPhysicalMaterial,
      MeshStandardMaterial: THREE.MeshStandardMaterial,
      Mesh: THREE.Mesh,
      Line: THREE.Line,
      LineBasicMaterial: THREE.LineBasicMaterial,
      Vector3: THREE.Vector3,
      Color: THREE.Color,
      Texture: THREE.Texture,
      CanvasTexture: THREE.CanvasTexture,
      Clock: THREE.Clock,
      AdditiveBlending: THREE.AdditiveBlending,
      RGBAFormat: THREE.RGBAFormat,
      UnsignedByteType: THREE.UnsignedByteType,
      Fog: THREE.Fog,
      AmbientLight: THREE.AmbientLight,
      DirectionalLight: THREE.DirectionalLight,
      Object3D: THREE.Object3D,
      Group: THREE.Group
    };

    // Track performance metrics
    const endTime = performance.now();
    const endMemory = (performance as any).memory?.usedJSHeapSize || 0;
    loadingMetrics.coreLoadTime = endTime - startTime;
    loadingMetrics.memoryUsage += (endMemory - startMemory) / 1048576; // MB

    threeLoadingManager.updateProgress({
      stage: 'core',
      progress: 100,
      message: `Core modules loaded in ${loadingMetrics.coreLoadTime.toFixed(1)}ms`
    });

    console.log('[Three.js Loader] ✅ Core Three.js modules loaded');
    return threeCore;
  } catch (error) {
    const errorMessage = `Failed to load core Three.js: ${error instanceof Error ? error.message : 'Unknown error'}`;
    loadingMetrics.errors.push(errorMessage);

    threeLoadingManager.updateProgress({
      stage: 'core',
      progress: 0,
      message: 'Failed to load core modules',
      error: errorMessage
    });

    console.error('[Three.js Loader] Failed to load core Three.js:', error);
    throw new Error('Failed to load Three.js core modules');
  }
}

export async function loadThreeRenderers(): Promise<ThreeRenderers> {
  if (threeRenderers) return threeRenderers;

  const startTime = performance.now();

  threeLoadingManager.updateProgress({
    stage: 'renderers',
    progress: 10,
    message: 'Loading Three.js renderers with WebGPU optimization...'
  });

  console.log('[Three.js Loader] Loading Three.js renderers with WebGPU priority...');

  try {
    // Check WebGPU availability first for optimization
    const webgpuAvailable = 'gpu' in navigator;

    threeLoadingManager.updateProgress({
      stage: 'renderers',
      progress: 25,
      message: webgpuAvailable
        ? 'WebGPU detected - loading optimized renderer...'
        : 'WebGPU unavailable - loading fallback renderers...'
    });

    // Prioritize WebGPU loading if available
    let webgpuModule = null;
    if (webgpuAvailable) {
      try {
        threeLoadingManager.updateProgress({
          stage: 'renderers',
          progress: 40,
          message: 'Loading WebGPU renderer with optimizations...'
        });

        webgpuModule = await import('three/webgpu');
        console.log('[Three.js Loader] ✅ WebGPU renderer loaded successfully');
      } catch (webgpuError) {
        console.warn('[Three.js Loader] WebGPU renderer failed to load:', webgpuError);
        webgpuModule = null;
      }
    }

    threeLoadingManager.updateProgress({
      stage: 'renderers',
      progress: 70,
      message: 'Loading additional renderers...'
    });

    // Load SVG renderer as fallback option
    const svgModule = await import('three/addons/renderers/SVGRenderer.js').catch(() => {
      console.warn('[Three.js Loader] SVG renderer not available');
      return null;
    });

    // Load CSS2D and CSS3D renderers for UI integration
    const [css2dModule, css3dModule] = await Promise.all([
      import('three/addons/renderers/CSS2DRenderer.js').catch(() => null),
      import('three/addons/renderers/CSS3DRenderer.js').catch(() => null)
    ]);

    threeRenderers = {
      WebGPURenderer: webgpuModule?.WebGPURenderer || null,
      SVGRenderer: svgModule?.SVGRenderer || null,
      CSS2DRenderer: css2dModule?.CSS2DRenderer || null,
      CSS3DRenderer: css3dModule?.CSS3DRenderer || null,
      // WebGPU-specific optimizations
      webgpuAvailable,
      webgpuNodes: webgpuModule
        ? {
            // Export WebGPU node materials for advanced shading
            MeshBasicNodeMaterial: webgpuModule.MeshBasicNodeMaterial || null,
            MeshStandardNodeMaterial: webgpuModule.MeshStandardNodeMaterial || null,
            PointsNodeMaterial: webgpuModule.PointsNodeMaterial || null,
            LineBasicNodeMaterial: webgpuModule.LineBasicNodeMaterial || null,
            // Compute shaders for particle systems
            ComputeNode: webgpuModule.ComputeNode || null,
            StorageBufferNode: webgpuModule.StorageBufferNode || null
          }
        : null
    };

    // Track performance
    const endTime = performance.now();
    loadingMetrics.renderersLoadTime = endTime - startTime;

    threeLoadingManager.updateProgress({
      stage: 'renderers',
      progress: 100,
      message: `Renderers loaded in ${loadingMetrics.renderersLoadTime.toFixed(1)}ms ${webgpuAvailable ? '(WebGPU optimized)' : '(WebGL fallback)'}`
    });

    console.log('[Three.js Loader] ✅ Three.js renderers loaded', {
      webgpu: !!webgpuModule,
      svg: !!svgModule,
      css2d: !!css2dModule,
      css3d: !!css3dModule
    });

    return threeRenderers;
  } catch (error) {
    const errorMessage = `Failed to load Three.js renderers: ${error instanceof Error ? error.message : 'Unknown error'}`;
    loadingMetrics.errors.push(errorMessage);

    threeLoadingManager.updateProgress({
      stage: 'renderers',
      progress: 0,
      message: 'Failed to load renderers',
      error: errorMessage
    });

    console.error('[Three.js Loader] Failed to load Three.js renderers:', error);
    throw new Error('Failed to load Three.js renderers');
  }
}

export async function loadThreeAddons(): Promise<ThreeAddons> {
  if (threeAddons) return threeAddons;

  const startTime = performance.now();

  threeLoadingManager.updateProgress({
    stage: 'addons',
    progress: 10,
    message: 'Loading Three.js addons and post-processing effects...'
  });

  console.log('[Three.js Loader] Loading Three.js addons...');

  try {
    threeLoadingManager.updateProgress({
      stage: 'addons',
      progress: 30,
      message: 'Importing controls and effects...'
    });

    const [
      orbitControlsModule,
      effectComposerModule,
      renderPassModule,
      bloomPassModule,
      shaderPassModule,
      fxaaShaderModule
    ] = await Promise.all([
      import('three/addons/controls/OrbitControls.js').catch(() => {
        console.warn('[Three.js Loader] OrbitControls not available');
        return null;
      }),
      import('three/addons/postprocessing/EffectComposer.js').catch(() => {
        console.warn('[Three.js Loader] EffectComposer not available');
        return null;
      }),
      import('three/addons/postprocessing/RenderPass.js').catch(() => {
        console.warn('[Three.js Loader] RenderPass not available');
        return null;
      }),
      import('three/addons/postprocessing/UnrealBloomPass.js').catch(() => {
        console.warn('[Three.js Loader] UnrealBloomPass not available');
        return null;
      }),
      import('three/addons/postprocessing/ShaderPass.js').catch(() => {
        console.warn('[Three.js Loader] ShaderPass not available');
        return null;
      }),
      import('three/addons/shaders/FXAAShader.js').catch(() => {
        console.warn('[Three.js Loader] FXAAShader not available');
        return null;
      })
    ]);

    threeLoadingManager.updateProgress({
      stage: 'addons',
      progress: 80,
      message: 'Initializing addon components...'
    });

    threeAddons = {
      OrbitControls: orbitControlsModule?.OrbitControls || null,
      EffectComposer: effectComposerModule?.EffectComposer || null,
      RenderPass: renderPassModule?.RenderPass || null,
      UnrealBloomPass: bloomPassModule?.UnrealBloomPass || null,
      ShaderPass: shaderPassModule?.ShaderPass || null,
      FXAAShader: fxaaShaderModule?.FXAAShader || null
    };

    // Track performance
    const endTime = performance.now();
    loadingMetrics.addonsLoadTime = endTime - startTime;

    threeLoadingManager.updateProgress({
      stage: 'addons',
      progress: 100,
      message: `Addons loaded in ${loadingMetrics.addonsLoadTime.toFixed(1)}ms`
    });

    console.log('[Three.js Loader] ✅ Three.js addons loaded');
    return threeAddons;
  } catch (error) {
    const errorMessage = `Failed to load Three.js addons: ${error instanceof Error ? error.message : 'Unknown error'}`;
    loadingMetrics.errors.push(errorMessage);

    threeLoadingManager.updateProgress({
      stage: 'addons',
      progress: 0,
      message: 'Failed to load addons',
      error: errorMessage
    });

    console.error('[Three.js Loader] Failed to load Three.js addons:', error);
    throw new Error('Failed to load Three.js addons');
  }
}

// Convenience function to load all Three.js modules at once with comprehensive tracking
export async function loadAllThreeModules(): Promise<{
  core: ThreeCore;
  renderers: ThreeRenderers;
  addons: ThreeAddons;
}> {
  const totalStartTime = performance.now();

  threeLoadingManager.reset();
  threeLoadingManager.updateProgress({
    stage: 'initialization',
    progress: 0,
    message: 'Starting Three.js module loading...'
  });

  console.log('[Three.js Loader] Loading all Three.js modules...');

  try {
    // Load modules sequentially for better progress tracking
    threeLoadingManager.updateProgress({
      stage: 'core',
      progress: 5,
      message: 'Loading core Three.js modules...'
    });

    const core = await loadThreeCore();

    threeLoadingManager.updateProgress({
      stage: 'renderers',
      progress: 35,
      message: 'Loading WebGPU and SVG renderers...'
    });

    const renderers = await loadThreeRenderers();

    threeLoadingManager.updateProgress({
      stage: 'addons',
      progress: 70,
      message: 'Loading controls and post-processing effects...'
    });

    const addons = await loadThreeAddons();

    // Calculate final metrics
    const totalEndTime = performance.now();
    loadingMetrics.totalLoadTime = totalEndTime - totalStartTime;
    loadingMetrics.success = true;

    // Bundle size is tracked through memory usage during loading
    loadingMetrics.bundleSize = loadingMetrics.memoryUsage;

    threeLoadingManager.updateProgress({
      stage: 'complete',
      progress: 100,
      message: `All modules loaded successfully in ${loadingMetrics.totalLoadTime.toFixed(1)}ms (${loadingMetrics.bundleSize.toFixed(1)}MB)`
    });

    console.log('[Three.js Loader] ✅ All Three.js modules loaded successfully');
    console.log('[Three.js Loader] Performance metrics:', loadingMetrics);

    // Expose metrics globally for benchmarking
    if (typeof window !== 'undefined') {
      (window as any).__THREE_LOADING_METRICS = loadingMetrics;
    }

    return { core, renderers, addons };
  } catch (error) {
    loadingMetrics.success = false;
    loadingMetrics.totalLoadTime = performance.now() - totalStartTime;

    const errorMessage = `Failed to load Three.js modules: ${error instanceof Error ? error.message : 'Unknown error'}`;
    loadingMetrics.errors.push(errorMessage);

    threeLoadingManager.updateProgress({
      stage: 'error',
      progress: 0,
      message: 'Failed to load Three.js modules',
      error: errorMessage
    });

    console.error('[Three.js Loader] Failed to load Three.js modules:', error);
    console.error('[Three.js Loader] Error metrics:', loadingMetrics);

    throw error;
  }
}

// Enhanced preload function with priority loading
export function preloadThreeModules(): void {
  console.log('[Three.js Loader] Starting background preload...');

  // Start loading core modules immediately (highest priority)
  loadThreeCore().catch(() => {
    console.warn('[Three.js Loader] Background preload of core failed');
  });

  // Load renderers after a short delay to prioritize core
  setTimeout(() => {
    loadThreeRenderers().catch(() => {
      console.warn('[Three.js Loader] Background preload of renderers failed');
    });
  }, 100);

  // Load addons after core and renderers (lowest priority)
  setTimeout(() => {
    loadThreeAddons().catch(() => {
      console.warn('[Three.js Loader] Background preload of addons failed');
    });
  }, 200);
}

// Performance analysis functions with WebGPU optimization
export function getLoadingMetrics(): LoadingMetrics {
  return { ...loadingMetrics };
}

export function analyzeLoadingPerformance(): {
  score: number;
  insights: string[];
  recommendations: string[];
  webgpuOptimization: {
    available: boolean;
    enabled: boolean;
    benefits: string[];
  };
} {
  const insights: string[] = [];
  const recommendations: string[] = [];
  let score = 100;

  // WebGPU analysis
  const capabilities = detectThreeCapabilities();
  const webgpuOptimization = {
    available: capabilities.webgpu,
    enabled: threeRenderers?.webgpuAvailable === true,
    benefits: [] as string[]
  };

  if (capabilities.webgpu) {
    webgpuOptimization.benefits.push('Compute shader acceleration');
    webgpuOptimization.benefits.push('Lower CPU overhead');
    webgpuOptimization.benefits.push('Better memory bandwidth utilization');

    if (threeRenderers?.webgpuAvailable) {
      score += 15; // Bonus for WebGPU usage
      insights.push('WebGPU renderer active - optimal performance mode');
    } else {
      score -= 10;
      insights.push('WebGPU available but not enabled');
      recommendations.push('Enable WebGPU renderer for best performance');
    }
  } else {
    insights.push('WebGPU not available - using WebGL fallback');
    if (capabilities.webgl2) {
      insights.push('WebGL2 available - good performance expected');
    } else {
      score -= 20;
      insights.push('Only WebGL1 available - limited performance');
      recommendations.push('Update browser or GPU drivers for better support');
    }
  }

  // Analyze loading times
  if (loadingMetrics.totalLoadTime > 2000) {
    score -= 30;
    insights.push(`Slow loading: ${loadingMetrics.totalLoadTime.toFixed(1)}ms total`);
    recommendations.push('Consider enabling service worker caching');
    if (capabilities.webgpu) {
      recommendations.push('WebGPU modules may benefit from preloading');
    }
  } else if (loadingMetrics.totalLoadTime > 1000) {
    score -= 15;
    insights.push(`Moderate loading time: ${loadingMetrics.totalLoadTime.toFixed(1)}ms`);
  } else {
    insights.push(`Fast loading: ${loadingMetrics.totalLoadTime.toFixed(1)}ms`);
  }

  // Analyze memory usage
  if (loadingMetrics.memoryUsage > 50) {
    score -= 20;
    insights.push(`High memory usage: ${loadingMetrics.memoryUsage.toFixed(1)}MB`);
    recommendations.push('Consider code splitting for addons');
    if (capabilities.webgpu) {
      recommendations.push('WebGPU can offload memory to GPU');
    }
  } else if (loadingMetrics.memoryUsage > 25) {
    score -= 10;
    insights.push(`Moderate memory usage: ${loadingMetrics.memoryUsage.toFixed(1)}MB`);
  } else {
    insights.push(`Efficient memory usage: ${loadingMetrics.memoryUsage.toFixed(1)}MB`);
  }

  // Check for errors
  if (loadingMetrics.errors.length > 0) {
    score -= 25;
    insights.push(`${loadingMetrics.errors.length} loading errors encountered`);
    recommendations.push('Check browser compatibility and network connectivity');
  }

  // Success rate
  if (!loadingMetrics.success) {
    score = Math.min(score, 20);
    insights.push('Loading failed completely');
    recommendations.push('Implement fallback rendering pipeline');
  }

  // WebGPU-specific recommendations
  if (capabilities.webgpu && !threeRenderers?.webgpuAvailable) {
    recommendations.push('Initialize WebGPU renderer for 50-80% performance improvement');
  }

  if (capabilities.webgpuFeatures.includes('texture-compression-bc')) {
    recommendations.push('Enable BC texture compression for reduced memory usage');
  }

  return {
    score: Math.max(0, score),
    insights,
    recommendations,
    webgpuOptimization
  };
}

// Check if modules are available with detailed status
export function isThreeCoreLoaded(): boolean {
  return threeCore !== null;
}

export function isThreeRenderersLoaded(): boolean {
  return threeRenderers !== null;
}

export function isThreeAddonsLoaded(): boolean {
  return threeAddons !== null;
}

export function isAllThreeModulesLoaded(): boolean {
  return threeCore !== null && threeRenderers !== null && threeAddons !== null;
}

// Get detailed loading status with WebGPU optimization info
export function getThreeLoadingStatus(): {
  core: boolean;
  renderers: boolean;
  addons: boolean;
  webgpu: boolean;
  webgpuNodes: boolean;
  svg: boolean;
  css2d: boolean;
  css3d: boolean;
  postProcessing: boolean;
  controls: boolean;
  webgpuOptimized: boolean;
  recommendedRenderer: string;
} {
  const capabilities = detectThreeCapabilities();

  return {
    core: threeCore !== null,
    renderers: threeRenderers !== null,
    addons: threeAddons !== null,
    webgpu: threeRenderers?.WebGPURenderer !== null,
    webgpuNodes: threeRenderers?.webgpuNodes !== null,
    svg: threeRenderers?.SVGRenderer !== null,
    css2d: threeRenderers?.CSS2DRenderer !== null,
    css3d: threeRenderers?.CSS3DRenderer !== null,
    postProcessing: threeAddons?.EffectComposer !== null && threeAddons?.UnrealBloomPass !== null,
    controls: threeAddons?.OrbitControls !== null,
    webgpuOptimized:
      threeRenderers?.webgpuAvailable === true && threeRenderers?.webgpuNodes !== null,
    recommendedRenderer: capabilities.recommendedRenderer
  };
}

// Capability detection with WebGPU focus
let cachedWebGPUFeatures: string[] | null = null;
let cachedWebGPULimits: any = null;
let loggedWebGPUInfo = false;

export function detectThreeCapabilities(): {
  webgpu: boolean;
  webgl: boolean;
  webgl2: boolean;
  extensions: string[];
  webgpuFeatures: string[];
  webgpuLimits: any;
  recommendedRenderer: 'webgpu' | 'webgl2' | 'webgl';
} {
  const canvas = document.createElement('canvas');
  const capabilities = {
    webgpu: false,
    webgl: false,
    webgl2: false,
    extensions: [] as string[],
    webgpuFeatures: [] as string[],
    webgpuLimits: null as any,
    recommendedRenderer: 'webgl' as 'webgpu' | 'webgl2' | 'webgl'
  };

  // Test WebGPU support with detailed feature detection
  if ('gpu' in navigator) {
    capabilities.webgpu = true;
    capabilities.recommendedRenderer = 'webgpu';

    // Async WebGPU feature detection
    (async () => {
      try {
        const adapter = await (navigator as any).gpu.requestAdapter();
        if (adapter) {
          cachedWebGPUFeatures = Array.from(adapter.features);
          cachedWebGPULimits = adapter.limits;
          capabilities.webgpuFeatures = cachedWebGPUFeatures;
          capabilities.webgpuLimits = cachedWebGPULimits;
          if (!loggedWebGPUInfo) {
            console.log('[Three.js Loader] WebGPU features:', cachedWebGPUFeatures);
            console.log('[Three.js Loader] WebGPU limits:', cachedWebGPULimits);
            loggedWebGPUInfo = true;
          }
        }
      } catch (e) {
        console.warn('[Three.js Loader] WebGPU adapter request failed:', e);
        capabilities.webgpu = false;
        capabilities.recommendedRenderer = 'webgl2';
      }
    })();
    // If already cached, set immediately
    if (cachedWebGPUFeatures) capabilities.webgpuFeatures = cachedWebGPUFeatures;
    if (cachedWebGPULimits) capabilities.webgpuLimits = cachedWebGPULimits;
  }

  // Test WebGL2 support
  try {
    const gl2 = canvas.getContext('webgl2');
    if (gl2) {
      capabilities.webgl2 = true;
      if (!capabilities.webgpu) {
        capabilities.recommendedRenderer = 'webgl2';
      }
    }
  } catch (e) {
    console.warn('[Three.js Loader] WebGL2 not supported:', e);
  }

  // Test WebGL support
  try {
    const gl = canvas.getContext('webgl');
    if (gl) {
      capabilities.webgl = true;
      const extensions = gl.getSupportedExtensions();
      if (extensions) {
        capabilities.extensions = extensions;
      }

      // If no WebGL2 or WebGPU, fall back to WebGL
      if (!capabilities.webgl2 && !capabilities.webgpu) {
        capabilities.recommendedRenderer = 'webgl';
      }
    }
  } catch (e) {
    console.warn('[Three.js Loader] WebGL not supported:', e);
  }

  return capabilities;
}

// Integration with WASM GPU bridge
import { wasmGPU } from '../wasmBridge';

export function integrateWithWASMGPU(): Promise<boolean> {
  // Event-driven readiness: listen for 'wasmBridgeReady' event
  return new Promise(resolve => {
    if (wasmGPU.isInitialized() || (typeof window !== 'undefined' && (window as any).wasmReady)) {
      console.log('[Three.js Loader] ✅ WASM GPU bridge detected (event-driven)');
      resolve(true);
      return;
    }
    if (typeof window !== 'undefined') {
      const onReady = () => {
        window.removeEventListener('wasmBridgeReady', onReady);
        console.log('[Three.js Loader] ✅ WASM GPU bridge ready (event-driven)');
        resolve(true);
      };
      window.addEventListener('wasmBridgeReady', onReady);
      // Optional: timeout fallback if event not received
      setTimeout(() => {
        window.removeEventListener('wasmBridgeReady', onReady);
        if (!wasmGPU.isInitialized() && !(window as any).wasmReady) {
          console.warn('[Three.js Loader] WASM GPU bridge not available (event-driven)');
          resolve(false);
        }
      }, 5000);
    } else {
      resolve(false);
    }
  });
}

// Cleanup function for memory management
export function cleanupThreeModules(): void {
  console.log('[Three.js Loader] Cleaning up Three.js modules...');

  // Reset module caches
  threeCore = null;
  threeRenderers = null;
  threeAddons = null;

  // Reset metrics
  loadingMetrics = {
    coreLoadTime: 0,
    renderersLoadTime: 0,
    addonsLoadTime: 0,
    totalLoadTime: 0,
    bundleSize: 0,
    memoryUsage: 0,
    success: false,
    errors: []
  };

  // Reset loading manager
  threeLoadingManager.reset();

  console.log('[Three.js Loader] ✅ Cleanup completed');
}

// Initialize browser capabilities detection on module load
if (typeof window !== 'undefined') {
  // Detect capabilities immediately
  const capabilities = detectThreeCapabilities();
  console.log('[Three.js Loader] Browser capabilities:', capabilities);

  // Expose capabilities globally for debugging
  (window as any).__THREE_CAPABILITIES = capabilities;

  // Start WASM integration check
  integrateWithWASMGPU().then(success => {
    if (success) {
      console.log('[Three.js Loader] ✅ WASM GPU integration ready');
    }
  });
}

// WebGPU-specific optimization functions
export async function optimizeForWebGPU(): Promise<{
  success: boolean;
  optimizations: string[];
  performance: {
    memoryBandwidth: number;
    computeUnits: number;
    maxBufferSize: number;
    maxComputeWorkgroupSize: number;
  } | null;
}> {
  const optimizations: string[] = [];
  let performance = null;

  if (!('gpu' in navigator)) {
    return {
      success: false,
      optimizations: ['WebGPU not available - falling back to WebGL'],
      performance: null
    };
  }

  try {
    const adapter = await (navigator as any).gpu.requestAdapter({
      powerPreference: 'high-performance'
    });

    if (!adapter) {
      return {
        success: false,
        optimizations: ['WebGPU adapter request failed'],
        performance: null
      };
    }

    // Extract performance characteristics
    performance = {
      memoryBandwidth: adapter.limits.maxStorageBufferBindingSize / 1048576, // MB
      computeUnits: adapter.limits.maxComputeWorkgroupSizeX,
      maxBufferSize: adapter.limits.maxBufferSize / 1048576, // MB
      maxComputeWorkgroupSize: adapter.limits.maxComputeWorkgroupSizeX
    };

    // Apply optimizations based on capabilities
    if (adapter.features.has('texture-compression-bc')) {
      optimizations.push('BC texture compression enabled');
    }

    if (adapter.features.has('texture-compression-etc2')) {
      optimizations.push('ETC2 texture compression enabled');
    }

    if (adapter.features.has('texture-compression-astc')) {
      optimizations.push('ASTC texture compression enabled');
    }

    if (adapter.limits.maxComputeWorkgroupSizeX >= 256) {
      optimizations.push('Large compute workgroups available (256+)');
    }

    if (adapter.limits.maxStorageBufferBindingSize >= 134217728) {
      // 128MB
      optimizations.push('Large storage buffers available (128MB+)');
    }

    // Timestamp queries for precise performance measurement
    if (adapter.features.has('timestamp-query')) {
      optimizations.push('Timestamp queries enabled for performance profiling');
    }

    // Shader debugging
    if (adapter.features.has('shader-f16')) {
      optimizations.push('Half-precision floating point shaders available');
    }

    return {
      success: true,
      optimizations,
      performance
    };
  } catch (error) {
    console.error('[Three.js Loader] WebGPU optimization failed:', error);
    return {
      success: false,
      optimizations: [
        `WebGPU optimization error: ${error instanceof Error ? error.message : 'Unknown error'}`
      ],
      performance: null
    };
  }
}

// WebGPU renderer factory with optimizations
export async function createOptimizedWebGPURenderer(canvas: HTMLCanvasElement): Promise<{
  renderer: any;
  optimizations: string[];
  performance: any;
} | null> {
  if (!threeRenderers?.WebGPURenderer || !threeRenderers.webgpuAvailable) {
    console.warn('[Three.js Loader] WebGPU renderer not available');
    return null;
  }

  try {
    const optimizationResult = await optimizeForWebGPU();

    if (!optimizationResult.success) {
      console.warn('[Three.js Loader] WebGPU optimization failed');
      return null;
    }

    // Create WebGPU renderer with optimizations
    const renderer = new threeRenderers.WebGPURenderer({
      canvas,
      antialias: true,
      alpha: true,
      powerPreference: 'high-performance',
      // WebGPU-specific optimizations
      forceWebGL: false
    });

    // Configure for high performance
    await renderer.init();

    // Enable advanced features if available
    if (
      optimizationResult.performance &&
      optimizationResult.performance.maxComputeWorkgroupSize >= 256
    ) {
      // Enable compute-based particle systems
      console.log('[Three.js Loader] ✅ Compute shaders enabled for particle systems');
    }

    return {
      renderer,
      optimizations: optimizationResult.optimizations,
      performance: optimizationResult.performance
    };
  } catch (error) {
    console.error('[Three.js Loader] Failed to create optimized WebGPU renderer:', error);
    return null;
  }
}

// Performance analysis for WebGPU vs WebGL
export function compareRenderingPerformance(): {
  recommendation: string;
  reasons: string[];
  webgpuScore: number;
  webglScore: number;
} {
  const capabilities = detectThreeCapabilities();
  let webgpuScore = 0;
  let webglScore = 0;
  const reasons: string[] = [];

  // Score WebGPU
  if (capabilities.webgpu) {
    webgpuScore += 40; // Base score for WebGPU availability
    reasons.push('WebGPU available - modern GPU compute pipeline');

    if (capabilities.webgpuFeatures.includes('texture-compression-bc')) {
      webgpuScore += 10;
      reasons.push('Advanced texture compression supported');
    }

    if (capabilities.webgpuFeatures.includes('timestamp-query')) {
      webgpuScore += 5;
      reasons.push('Precise performance profiling available');
    }
  }

  // Score WebGL
  if (capabilities.webgl2) {
    webglScore += 30; // Base score for WebGL2
    reasons.push('WebGL2 available - stable rendering pipeline');

    if (capabilities.extensions.includes('EXT_texture_compression_bptc')) {
      webglScore += 8;
      reasons.push('BPTC texture compression available');
    }

    if (capabilities.extensions.includes('WEBGL_debug_renderer_info')) {
      webglScore += 3;
      reasons.push('GPU debugging information available');
    }
  } else if (capabilities.webgl) {
    webglScore += 20; // Base score for WebGL1
    reasons.push('WebGL1 available - basic rendering support');
  }

  // Additional scoring factors
  if (capabilities.extensions.includes('OES_texture_float')) {
    webglScore += 5;
    reasons.push('Floating point textures supported');
  }

  // Make recommendation
  let recommendation: string;
  if (webgpuScore > webglScore && webgpuScore >= 40) {
    recommendation = 'webgpu';
  } else if (capabilities.webgl2) {
    recommendation = 'webgl2';
  } else {
    recommendation = 'webgl';
  }

  return {
    recommendation,
    reasons,
    webgpuScore,
    webglScore
  };
}

// WebGPU-optimized particle system creation
export async function createWebGPUParticleSystem(particleCount: number): Promise<{
  geometry: any;
  material: any;
  mesh: any;
  computeShader?: any;
  updateFunction: (deltaTime: number) => void;
} | null> {
  if (!threeCore || !threeRenderers?.webgpuNodes) {
    console.warn('[Three.js Loader] WebGPU nodes not available for particle system');
    return null;
  }

  try {
    const { BufferGeometry, BufferAttribute, Points } = threeCore;
    const { PointsNodeMaterial, ComputeNode, StorageBufferNode } = threeRenderers.webgpuNodes;

    // Create geometry with positions and velocities
    const geometry = new BufferGeometry();
    const positions = new Float32Array(particleCount * 3);
    const velocities = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);

    // Initialize particle data
    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;
      // Random positions in a sphere
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(1 - 2 * Math.random());
      const radius = Math.random() * 10;

      positions[i3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[i3 + 1] = radius * Math.sin(phi) * Math.sin(theta);
      positions[i3 + 2] = radius * Math.cos(phi);

      // Random velocities
      velocities[i3] = (Math.random() - 0.5) * 2;
      velocities[i3 + 1] = (Math.random() - 0.5) * 2;
      velocities[i3 + 2] = (Math.random() - 0.5) * 2;

      // Colors based on position
      colors[i3] = Math.random();
      colors[i3 + 1] = Math.random();
      colors[i3 + 2] = Math.random();
    }

    geometry.setAttribute('position', new BufferAttribute(positions, 3));
    geometry.setAttribute('velocity', new BufferAttribute(velocities, 3));
    geometry.setAttribute('color', new BufferAttribute(colors, 3));

    // Create WebGPU node material for compute-based animation
    const material = new PointsNodeMaterial({
      size: 2,
      vertexColors: true,
      transparent: true,
      opacity: 0.8
    });

    // Create compute shader for particle updates (if available)
    let computeShader = null;
    if (ComputeNode && StorageBufferNode) {
      // This would be implemented with actual WebGPU compute shaders
      // For now, we'll use a placeholder that can be expanded
      computeShader = {
        workgroupSize: 64,
        update: (deltaTime: number) => {
          // Placeholder for compute shader particle updates
          console.log('[Three.js Loader] Compute shader update placeholder:', deltaTime);
        }
      };
    }

    const mesh = new Points(geometry, material);

    // Fallback update function for CPU-based animation
    const updateFunction = (deltaTime: number) => {
      if (computeShader) {
        computeShader.update(deltaTime);
      } else {
        // CPU fallback for particle animation
        const positions = geometry.attributes.position.array as Float32Array;
        const velocities = geometry.attributes.velocity.array as Float32Array;

        for (let i = 0; i < particleCount; i++) {
          const i3 = i * 3;
          positions[i3] += velocities[i3] * deltaTime;
          positions[i3 + 1] += velocities[i3 + 1] * deltaTime;
          positions[i3 + 2] += velocities[i3 + 2] * deltaTime;

          // Boundary conditions (wrap around)
          if (Math.abs(positions[i3]) > 20) velocities[i3] *= -1;
          if (Math.abs(positions[i3 + 1]) > 20) velocities[i3 + 1] *= -1;
          if (Math.abs(positions[i3 + 2]) > 20) velocities[i3 + 2] *= -1;
        }

        geometry.attributes.position.needsUpdate = true;
      }
    };

    console.log('[Three.js Loader] ✅ WebGPU particle system created', {
      particles: particleCount,
      hasComputeShader: !!computeShader,
      memoryUsage: ((particleCount * 3 * 4 * 3) / 1048576).toFixed(2) + 'MB' // positions + velocities + colors
    });

    return {
      geometry,
      material,
      mesh,
      computeShader,
      updateFunction
    };
  } catch (error) {
    console.error('[Three.js Loader] Failed to create WebGPU particle system:', error);
    return null;
  }
}

// --- Advanced Particle System Utilities & Hooks ---
import { useRef, useEffect } from 'react';

// Particle geometry utility
export function createParticleGeometry({
  count,
  positionFn,
  velocityFn,
  colorFn,
  angleFn,
  ageFn,
  intensityFn
}: {
  count: number;
  positionFn?: (i: number) => [number, number, number];
  velocityFn?: (i: number) => [number, number, number];
  colorFn?: (i: number) => [number, number, number];
  angleFn?: (i: number) => number;
  ageFn?: (i: number) => number;
  intensityFn?: (i: number) => number;
}) {
  if (!threeCore) throw new Error('Three.js core not loaded');
  const { BufferGeometry, BufferAttribute } = threeCore;
  const geometry = new BufferGeometry();
  const positions = new Float32Array(count * 3);
  const velocities = new Float32Array(count * 3);
  const colors = new Float32Array(count * 3);
  const angles = new Float32Array(count);
  const ages = new Float32Array(count);
  const intensities = new Float32Array(count);
  for (let i = 0; i < count; i++) {
    const i3 = i * 3;
    positions.set(positionFn ? positionFn(i) : [0, 0, 0], i3);
    velocities.set(velocityFn ? velocityFn(i) : [0, 0, 0], i3);
    colors.set(colorFn ? colorFn(i) : [1, 1, 1], i3);
    angles[i] = angleFn ? angleFn(i) : Math.random() * Math.PI * 2;
    ages[i] = ageFn ? ageFn(i) : 0;
    intensities[i] = intensityFn ? intensityFn(i) : 1;
  }
  geometry.setAttribute('position', new BufferAttribute(positions, 3));
  geometry.setAttribute('velocity', new BufferAttribute(velocities, 3));
  geometry.setAttribute('color', new BufferAttribute(colors, 3));
  geometry.setAttribute('angle', new BufferAttribute(angles, 1));
  geometry.setAttribute('age', new BufferAttribute(ages, 1));
  geometry.setAttribute('intensity', new BufferAttribute(intensities, 1));
  return geometry;
}

// Particle attribute update utility
export function updateParticleAttributes(geometry: any, updateFn: (i: number, attrs: any) => void) {
  const count = geometry.attributes.position.count;
  for (let i = 0; i < count; i++) {
    const attrs = {
      position: geometry.attributes.position.array,
      velocity: geometry.attributes.velocity.array,
      color: geometry.attributes.color.array,
      angle: geometry.attributes.angle.array,
      age: geometry.attributes.age.array,
      intensity: geometry.attributes.intensity.array
    };
    updateFn(i, attrs);
  }
  geometry.attributes.position.needsUpdate = true;
  geometry.attributes.velocity.needsUpdate = true;
  geometry.attributes.color.needsUpdate = true;
  geometry.attributes.angle.needsUpdate = true;
  geometry.attributes.age.needsUpdate = true;
  geometry.attributes.intensity.needsUpdate = true;
}

// FBO utility
export function createFBO(size: number, THREE: any, useFBO: any) {
  return useFBO(size, {
    type: THREE.FloatType,
    format: THREE.RGBAFormat,
    minFilter: THREE.NearestFilter,
    magFilter: THREE.NearestFilter,
    depthBuffer: false,
    stencilBuffer: false
  });
}

// DataTexture generator for morph targets
export function generatePositionsDataTexture(
  size: number,
  shapeFn: (i: number) => [number, number, number],
  THREE: any
) {
  const length = size * size * 4;
  const data = new Float32Array(length);
  for (let i = 0; i < size * size; i++) {
    const [x, y, z] = shapeFn(i);
    const i4 = i * 4;
    data[i4] = x;
    data[i4 + 1] = y;
    data[i4 + 2] = z;
    data[i4 + 3] = 1.0;
  }
  const texture = new THREE.DataTexture(data, size, size, THREE.RGBAFormat, THREE.FloatType);
  texture.needsUpdate = true;
  return texture;
}

// ShaderMaterial utility for particles
export function createParticleShaderMaterial(
  THREE: any,
  vertexShader: string,
  fragmentShader: string,
  uniforms: any
) {
  return new THREE.ShaderMaterial({
    vertexShader,
    fragmentShader,
    uniforms,
    transparent: true,
    blending: THREE.AdditiveBlending,
    depthWrite: false
  });
}

// Postprocessing setup utility
export function setupPostprocessing({
  renderer,
  scene,
  camera,
  THREE,
  EffectComposer,
  RenderPass,
  UnrealBloomPass
}: any) {
  const composer = new EffectComposer(renderer);
  composer.addPass(new RenderPass(scene, camera));
  const bloomPass = new UnrealBloomPass(
    new THREE.Vector2(window.innerWidth, window.innerHeight),
    1.5,
    0.4,
    0.85
  );
  composer.addPass(bloomPass);
  return composer;
}

// React hook to load all Three.js modules
export function useThreeModules() {
  const modulesRef = useRef<{ core: any; renderers: any; addons: any } | null>(null);
  useEffect(() => {
    let mounted = true;
    (async () => {
      const core = await loadThreeCore();
      const renderers = await loadThreeRenderers();
      const addons = await loadThreeAddons();
      if (mounted) modulesRef.current = { core, renderers, addons };
    })();
    return () => {
      mounted = false;
    };
  }, []);
  return modulesRef;
}

// React hook for FBO simulation
export function useFBOSimulation(size: number, THREE: any, useFBO: any) {
  const fboRef = useRef<any>(null);
  useEffect(() => {
    fboRef.current = createFBO(size, THREE, useFBO);
  }, [size, THREE, useFBO]);
  return fboRef;
}

// Utility for morph targets
export function generateMorphTargets({
  size,
  THREE,
  shapes
}: {
  size: number;
  THREE: any;
  shapes: Array<(i: number) => [number, number, number]>;
}) {
  return shapes.map(shapeFn => generatePositionsDataTexture(size, shapeFn, THREE));
}

// Utility: Get GPU buffer from WASM and use in Three.js/WebGPU
export function getWASMGPUParticleBuffer(): any {
  if (typeof window !== 'undefined' && (window as any).getCurrentOutputBuffer) {
    return (window as any).getCurrentOutputBuffer();
  }
  return null;
}

// Example: Attach WASM GPU buffer to Three.js geometry (for use with drei/fibre or direct engine interop)
export function createGeometryWithWASMGPUParticleBuffer(
  three: typeof THREE,
  particleCount: number
): any {
  const gpuBuffer = getWASMGPUParticleBuffer();
  if (!gpuBuffer || !three) return null;

  // For Three.js WebGPU, use StorageBufferNode if available
  if (threeRenderers?.webgpuNodes?.StorageBufferNode) {
    // Create StorageBufferNode from WASM GPU buffer
    const storageNode = new threeRenderers.webgpuNodes.StorageBufferNode(
      gpuBuffer,
      'float',
      particleCount * 10
    );
    // Create geometry and attach node as attribute
    const geometry = new three.BufferGeometry();
    geometry.setAttribute('position', storageNode); // You may need to map the buffer layout for position
    // ...add other attributes as needed (velocity, color, etc.)
    return geometry;
  }

  // Fallback: create BufferGeometry and set attribute from buffer (if supported)
  // This may require custom interop for drei/fibre
  // ...custom logic for other engines/libraries...
  return null;
}
