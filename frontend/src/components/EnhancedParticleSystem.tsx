/**
 * Particle Buffer Data Format (10 floats per particle):
 * [0]  position.x
 * [1]  position.y
 * [2]  position.z
 * [3]  velocity.x
 * [4]  velocity.y
 * [5]  velocity.z
 * [6]  phase
 * [7]  intensity
 * [8]  type
 * [9]  id
 *
 * All particle attributes are packed in a Float32Array for efficient GPU upload and WASM interop.
 * Update attribute pointers in Three.js/engine to match this format for custom shaders or WebGPU.
 */
import React, { useRef, useEffect, useState, useCallback, useMemo } from 'react';
import {
  loadAllThreeModules,
  type ThreeCore,
  type ThreeRenderers,
  type ThreeAddons
} from '../lib/three';
import { useGPUCapabilities, useEmitEvent } from '../store/global';
import { connectMediaStreamingToCampaign, wasmGPU } from '../lib/wasmBridge';
import { useConnectionStatus, useMediaStreamingState } from '../store/global';

// Logger utility for consistent logging
const logger = {
  info: (msg: string, data?: any) => {
    const timestamp = new Date().toISOString();
    console.log(`[Enhanced-Particles][${timestamp}] ${msg}`, data || '');
  },
  error: (msg: string, error?: any) => {
    const timestamp = new Date().toISOString();
    console.error(`[Enhanced-Particles][${timestamp}] ${msg}`, error || '');
  },
  warn: (msg: string, data?: any) => {
    const timestamp = new Date().toISOString();
    console.warn(`[Enhanced-Particles][${timestamp}] ${msg}`, data || '');
  }
};

interface ParticleMetrics {
  particleCount: number;
  fps: number;
  gpuUtilization: number;
  frameTime: number;
  computeMode: 'JS' | 'WASM' | 'WASM+WebGPU';
  renderMode: 'WebGL' | 'WebGPU' | 'WebGPU+WASM';
  animationMode: 'galaxy' | 'yin-yang' | 'wave' | 'spiral';
  connectionStrength: number;
  wasmReady: boolean;
  wasmMetrics: any;
  lastUpdate: number;
  webgpuReady: boolean;
  workerMetrics?: {
    activeWorkers: number;
    totalWorkers: number;
    queueDepth: number;
    throughput: number;
    avgLatency: number;
    peakThroughput: number;
    tasksProcessed: number;
  };
}

const EnhancedParticleSystem: React.FC = () => {
  // State for named particles
  const [namedParticles, setNamedParticles] = useState<
    Array<{ name: string; priority: number; color: string; scale: number }>
  >([
    { name: 'Ghost', priority: 1, color: '#e0e0e0', scale: 2.0 }, // ghostly white-grey
    { name: 'Shadow', priority: 2, color: '#222222', scale: 1.8 }, // deep black
    { name: 'Mist', priority: 3, color: '#b0b0b0', scale: 1.6 }, // soft grey
    { name: 'Specter', priority: 4, color: '#ffffff', scale: 1.7 } // pure white
  ]);

  const canvasRef = useRef<HTMLCanvasElement>(null);

  // GPU capabilities from global state
  const { isWebGPUAvailable, recommendedRenderer, gpuCapabilities } = useGPUCapabilities();

  // Event emission for state tracking
  const emitEvent = useEmitEvent();

  // Three.js loading state
  const [threeLoaded, setThreeLoaded] = useState(false);
  const [threeModules, setThreeModules] = useState<{
    core: ThreeCore;
    renderers: ThreeRenderers;
    addons: ThreeAddons;
  } | null>(null);

  // Store THREE modules for use throughout component
  const threeRef = useRef<any>(null);

  // Three.js refs with any types until modules load
  const sceneRef = useRef<any>(null);
  const rendererRef = useRef<any>(null);
  const cameraRef = useRef<any>(null);
  const controlsRef = useRef<any>(null);
  const particlesRef = useRef<any>(null);
  const animationIdRef = useRef<number | null>(null);
  const rendererInitializedRef = useRef<boolean>(false);
  const clockRef = useRef<any>(null);
  const gpuProcessingRef = useRef<boolean>(false); // Prevent concurrent GPU operations
  const gpuStartTimeRef = useRef<number>(Date.now() + 3000); // Delay GPU processing by 3 seconds
  const framePerformanceRef = useRef<{
    lastFrameTime: number;
    gpuBudget: number;
    targetFPS: number;
  }>({
    lastFrameTime: 0,
    gpuBudget: 8.0, // 8ms GPU budget for 120fps (13.3ms frame budget - 5ms buffer)
    targetFPS: 120
  });
  const chunkIndexRef = useRef<number>(0); // Track which chunk to process next

  // Animation parameters - dynamically adjusted based on WebGPU capabilities
  const getParameters = useCallback((renderMode: string) => {
    // Multiplier for architectural demo scale
    const multiplier = 25;
    // Target 1,000,000 particles for main pattern
    const baseParams = {
      galaxy: {
        count: 40000 * multiplier, // 1,000,000
        size: 0.08,
        radius: 15,
        branches: 6,
        spin: 2.0,
        randomness: 0.6,
        randomnessPower: 2,
        innerColor: '#e0e0e0',
        outsideColor: '#222222'
      },
      'yin-yang': {
        count: 30000 * multiplier, // 750,000
        size: 0.1,
        radius: 12,
        flowSpeed: 0.6,
        yinColor: '#b0b0b0',
        yangColor: '#ffffff'
      },
      wave: {
        count: 35000 * multiplier, // 875,000
        size: 0.06,
        amplitude: 8,
        frequency: 0.015,
        speed: 3.0,
        baseColor: '#e0e0e0'
      },
      spiral: {
        count: 25000 * multiplier, // 625,000
        size: 0.09,
        radius: 14,
        spiralArms: 4,
        tightness: 3,
        height: 10,
        rotationSpeed: 1.5,
        baseColor: '#b0b0b0'
      }
    };

    // Enhanced parameters for WebGPU (optimized particle counts)
    if (renderMode.includes('WebGPU')) {
      baseParams.galaxy.count = 1000000;
      baseParams['yin-yang'].count = 1000000;
      baseParams.wave.count = 1000000;
      baseParams.spiral.count = 1000000;
    }

    return baseParams;
  }, []);

  const [currentRenderMode, setCurrentRenderMode] = useState('WebGL');
  const parameters = useMemo(
    () => getParameters(currentRenderMode),
    [getParameters, currentRenderMode]
  );

  const [metrics, setMetrics] = useState<ParticleMetrics>({
    particleCount: parameters.galaxy.count,
    fps: 30, // Architectural demo inspiration: lower FPS for stability
    gpuUtilization: 0,
    frameTime: 33.33, // 30 FPS target
    connectionStrength: 0,
    wasmReady: false,
    wasmMetrics: null,
    lastUpdate: Date.now(),
    webgpuReady: !!navigator.gpu,
    computeMode: 'WASM+WebGPU',
    renderMode: navigator.gpu ? 'WebGPU' : 'WebGL', // Auto-detect WebGPU support
    animationMode: 'galaxy',
    workerMetrics: {
      activeWorkers: 4,
      totalWorkers: 4,
      queueDepth: 0,
      throughput: 0,
      avgLatency: 0,
      peakThroughput: 0,
      tasksProcessed: 0
    }
  });

  // Get connection status from global store
  const { wasmReady } = useConnectionStatus();

  // Load Three.js modules on mount
  useEffect(() => {
    let mounted = true;

    const loadThreeJs = async () => {
      try {
        const modules = await loadAllThreeModules();
        if (mounted) {
          setThreeModules(modules);
          setThreeLoaded(true);
          threeRef.current = modules.core;
          // Initialize clock after modules load
          clockRef.current = new modules.core.Clock();
        }
      } catch (error) {
        console.error('Failed to load Three.js modules:', error);
      }
    };

    loadThreeJs();
    return () => {
      mounted = false;
    };
  }, []);

  // Dynamic FPS target detection based on device capabilities
  const detectOptimalFPS = useCallback(() => {
    // Force 30fps for all devices to reduce heat and resource usage
    return 120;
  }, []);

  // Automatic WebGPU detection and render mode switching
  useEffect(() => {
    const detectWebGPU = async () => {
      try {
        const webgpuSupported = typeof navigator !== 'undefined' && 'gpu' in navigator;

        // Use centralized WASM GPU bridge instead of direct window functions
        let wasmWebgpuReady = false;
        try {
          wasmWebgpuReady = await wasmGPU.waitForInitialization();
          logger.info(`Centralized WASM GPU Bridge initialized: ${wasmWebgpuReady}`);
        } catch (error) {
          logger.warn(`WASM GPU Bridge initialization failed:`, error);
          // Fallback to checking window functions
          wasmWebgpuReady =
            typeof window.initWebGPU === 'function' && typeof window.runGPUCompute === 'function';
        }

        logger.info(
          `WebGPU Detection - Browser: ${webgpuSupported}, WASM: ${wasmWebgpuReady}, Global WASM Ready: ${wasmReady}`
        );

        let newRenderMode = 'WebGL';
        let newComputeMode: 'JS' | 'WASM' | 'WASM+WebGPU' = 'JS';

        if (webgpuSupported && wasmWebgpuReady) {
          newRenderMode = 'WebGPU+WASM';
          newComputeMode = 'WASM+WebGPU';
          logger.info('🚀 Using enhanced WebGPU+WASM mode for maximum performance');
        } else if (webgpuSupported) {
          newRenderMode = 'WebGPU';
          newComputeMode = 'JS';
          logger.info('⚡ Using WebGPU rendering with JavaScript compute');
        } else if (wasmWebgpuReady) {
          newRenderMode = 'WebGL';
          newComputeMode = 'WASM';
          logger.info('🔧 Using WebGL rendering with WASM compute');
        } else {
          newRenderMode = 'WebGL';
          newComputeMode = 'JS';
          logger.info('📱 Using WebGL rendering with JavaScript compute (fallback)');
        }

        setCurrentRenderMode(newRenderMode);
        setMetrics(prev => ({
          ...prev,
          renderMode: newRenderMode as 'WebGL' | 'WebGPU' | 'WebGPU+WASM',
          computeMode: newComputeMode,
          webgpuReady: webgpuSupported,
          wasmReady: wasmWebgpuReady,
          particleCount: getParameters(newRenderMode).galaxy.count
        }));
      } catch (error) {
        logger.error('WebGPU detection failed:', error);
        setCurrentRenderMode('WebGL');
        setMetrics(prev => ({
          ...prev,
          renderMode: 'WebGL',
          computeMode: 'JS',
          webgpuReady: false,
          wasmReady: false
        }));
      }
    };

    detectWebGPU();
  }, [getParameters]);

  // Monitor WASM GPU bridge status and particle animation connectivity
  useEffect(() => {
    const monitorWASMStatus = async () => {
      if (metrics.computeMode.includes('WASM')) {
        try {
          const isInitialized = wasmGPU.isInitialized();
          const bridgeMetrics = wasmGPU.getMetrics();

          logger.info(
            `[WASM-Monitor] Bridge initialized: ${isInitialized}, Metrics:`,
            bridgeMetrics
          );

          if (!isInitialized) {
            logger.info(`[WASM-Monitor] Attempting to initialize WASM GPU bridge...`);
            const success = await wasmGPU.waitForInitialization();
            logger.info(`[WASM-Monitor] Bridge initialization result: ${success}`);

            if (success) {
              setMetrics(prev => ({ ...prev, wasmReady: true }));
            }
          }
        } catch (error) {
          logger.error(`[WASM-Monitor] Bridge monitoring failed:`, error);
        }
      }
    };

    // Monitor immediately and then every 5 seconds
    monitorWASMStatus();
    const interval = setInterval(monitorWASMStatus, 5000);
    return () => clearInterval(interval);
  }, [metrics.computeMode]);

  // Media streaming state
  // Use global store for media streaming state
  const { mediaStreaming } = useMediaStreamingState();

  // Create particle texture - optimized for WebGL/WebGPU compatibility
  const createParticleTexture = useCallback(() => {
    const THREE = threeRef.current;
    if (!THREE) return undefined;

    const canvas = document.createElement('canvas');
    canvas.width = 256; // Reasonable resolution
    canvas.height = 256;
    const context = canvas.getContext('2d');

    if (context) {
      // Create clean radial gradient for particle effect
      const gradient = context.createRadialGradient(128, 128, 0, 128, 128, 128);
      gradient.addColorStop(0, 'rgba(255, 255, 255, 1)');
      gradient.addColorStop(0.3, 'rgba(255, 255, 255, 0.8)');
      gradient.addColorStop(0.6, 'rgba(255, 255, 255, 0.4)');
      gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');

      context.fillStyle = gradient;
      context.fillRect(0, 0, 256, 256);
    }

    if (!THREE) return undefined;
    const texture = new THREE.CanvasTexture(canvas);
    // Explicitly set texture properties for WebGL/WebGPU compatibility
    texture.generateMipmaps = false;
    texture.flipY = false; // Disable FLIP_Y to avoid WebGL errors
    texture.premultiplyAlpha = false; // Disable premultiplyAlpha to avoid WebGL errors
    texture.format = THREE.RGBAFormat;
    texture.type = THREE.UnsignedByteType;

    return texture;
  }, []);

  // Create enhanced particle material with advanced shader effects
  const createEnhancedParticleMaterial = useCallback(() => {
    const THREE = threeRef.current;
    if (!THREE) return undefined;

    const texture = createParticleTexture();
    if (!texture) return undefined;

    // CRITICAL: Early WebGPU detection to prevent ShaderMaterial incompatibility
    const isWebGPURenderer = rendererRef.current?.constructor.name === 'WebGPURenderer';
    const webGPURecommended = recommendedRenderer === 'webgpu';
    const webGPUAvailable = isWebGPUAvailable;

    if (isWebGPURenderer || webGPURecommended || webGPUAvailable) {
      console.log(
        '[EnhancedParticleSystem] WebGPU detected - using safe PointsMaterial to avoid ShaderMaterial incompatibility'
      );

      // Use basic PointsMaterial for WebGPU compatibility
      const safeMaterial = new THREE.PointsMaterial({
        size: 8.0,
        map: texture,
        transparent: true,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
        vertexColors: true,
        sizeAttenuation: true,
        opacity: 0.8
      });

      return safeMaterial;
    }

    // Enhanced WebGPU detection using multiple methods
    const isWebGPU =
      rendererRef.current &&
      (rendererRef.current.constructor.name === 'WebGPURenderer' ||
        rendererRef.current.isWebGPURenderer === true ||
        recommendedRenderer === 'webgpu');

    // Additional check for WebGPU node materials availability
    const hasNodeMaterials = threeModules?.renderers?.webgpuNodes?.PointsNodeMaterial;

    console.log('[EnhancedParticleSystem] Material creation debug:', {
      isWebGPU,
      hasNodeMaterials,
      rendererType: rendererRef.current?.constructor.name,
      recommendedRenderer,
      isWebGPUAvailable
    });

    if (isWebGPU && hasNodeMaterials) {
      try {
        // Use WebGPU node material for compatibility
        console.log('[EnhancedParticleSystem] Creating WebGPU-compatible node material');

        const webgpuNodes = threeModules.renderers.webgpuNodes;
        if (!webgpuNodes?.PointsNodeMaterial) {
          throw new Error('PointsNodeMaterial not available in webgpuNodes');
        }

        const { PointsNodeMaterial } = webgpuNodes;

        const material = new PointsNodeMaterial({
          transparent: true,
          blending: THREE.AdditiveBlending,
          depthWrite: false,
          vertexColors: true,
          size: 8.0,
          sizeAttenuation: true,
          opacity: 0.8
        });

        // Add texture if available
        if (texture) {
          material.map = texture;
        }

        console.log('[EnhancedParticleSystem] WebGPU node material created successfully');
        return material;
      } catch (error) {
        console.warn(
          '[EnhancedParticleSystem] Failed to create WebGPU node material, falling back to shader material:',
          error
        );
      }
    }

    // Fallback to traditional shader material for WebGL or if WebGPU node material fails
    console.log('[EnhancedParticleSystem] Creating WebGL shader material');

    if (isWebGPU && !hasNodeMaterials) {
      console.warn(
        '[EnhancedParticleSystem] WebGPU renderer detected but no node materials available - using PointsMaterial instead of ShaderMaterial to avoid compatibility issues'
      );

      // Use PointsMaterial for WebGPU compatibility when node materials aren't available
      const material = new THREE.PointsMaterial({
        size: 8.0,
        map: texture,
        transparent: true,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
        vertexColors: true,
        sizeAttenuation: true,
        opacity: 0.8
      });

      return material;
    }

    // Enhanced vertex shader with velocity orientation and age-based scaling
    const vertexShader = `
      attribute float size;
      attribute float scale;
      attribute vec3 velocity;
      attribute float age;
      attribute float intensity;
      attribute float phase;
      attribute float type;
      attribute float id;

      varying vec3 vColor;
      varying vec2 vUv;
      varying float vIntensity;
      varying vec3 vVelocity;
      varying float vAge;
      varying float vPhase;
      varying float vType;
      varying float vId;

      uniform float uTime;
      uniform float uPixelRatio;

      void main() {
        vColor = color;
        vUv = uv;
        vIntensity = intensity;
        vVelocity = velocity;
        vAge = age;
        vPhase = phase;
        vType = type;
        vId = id;

        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);

        // Age-based size scaling with pulsing effect
        float ageScale = 1.0 - smoothstep(0.0, 10.0, age);
        float pulseScale = 1.0 + sin(uTime * 5.0 + age + phase) * 0.1;

        // Velocity-based orientation scaling
        float velocityMagnitude = length(velocity);
        float velocityScale = 1.0 + velocityMagnitude * 2.0;

        // Intensity-based scaling with dynamic range
        float intensityScale = mix(0.5, 2.0, intensity);

        // Named/special particle scaling
        float typeScale = (type == 1.0) ? 1.5 : ((type == 2.0) ? 2.0 : 1.0);

        // Combined dynamic size calculation
        float finalScale = scale * ageScale * pulseScale * velocityScale * intensityScale * typeScale;

        gl_PointSize = size * finalScale * uPixelRatio;
        gl_Position = projectionMatrix * mvPosition;
      }
    `;

    // Enhanced fragment shader with motion blur and advanced effects
    const fragmentShader = `
      varying vec3 vColor;
      varying vec2 vUv;
      varying float vIntensity;
      varying vec3 vVelocity;
      varying float vAge;
      varying float vPhase;
      varying float vType;
      varying float vId;

      uniform sampler2D uTexture;
      uniform float uTime;
      uniform float uOpacity;

      void main() {
        vec2 cxy = 2.0 * gl_PointCoord - 1.0;
        float r = dot(cxy, cxy);

        // Discard fragments outside circle
        if (r > 1.0) discard;

        // Motion blur effect based on velocity
        float velocityLength = length(vVelocity);
        vec2 motionBlurOffset = normalize(vVelocity.xy) * velocityLength * 0.3;

        // Sample texture with motion blur
        vec4 texColor1 = texture2D(uTexture, vUv);
        vec4 texColor2 = texture2D(uTexture, vUv + motionBlurOffset * 0.5);
        vec4 texColor3 = texture2D(uTexture, vUv - motionBlurOffset * 0.5);

        // Blend motion blur samples
        vec4 blurredTexture = (texColor1 * 0.5 + texColor2 * 0.25 + texColor3 * 0.25);

        // Age-based color evolution
        vec3 youngColor = vColor;
        vec3 oldColor = vColor * 0.3 + vec3(0.2, 0.1, 0.4); // Purple tint for older particles
        vec3 ageColor = mix(youngColor, oldColor, smoothstep(0.0, 8.0, vAge));

        // Velocity-based color intensity
        float velocityIntensity = smoothstep(0.0, 2.0, velocityLength);
        vec3 velocityColor = mix(ageColor, ageColor * 1.5, velocityIntensity);

        // Intensity-based glow effect
        float glowFactor = pow(vIntensity, 2.0);
        vec3 glowColor = velocityColor + vec3(0.2, 0.4, 0.8) * glowFactor;

        // Named/special particle color boost
        if (vType == 1.0) {
          glowColor += vec3(1.0, 1.0, 0.2) * 0.5; // Named: golden glow
        } else if (vType == 2.0) {
          glowColor += vec3(0.2, 1.0, 1.0) * 0.5; // Special: cyan glow
        }

        // Phase-based pulsing effect
        float phasePulse = 0.8 + 0.2 * sin(vPhase + uTime * 2.0);
        glowColor *= phasePulse;

        // Radial fade with enhanced falloff
        float alpha = 1.0 - smoothstep(0.0, 1.0, r);
        alpha = pow(alpha, 1.5);

        // Age-based alpha modulation
        float ageAlpha = 1.0 - smoothstep(5.0, 12.0, vAge);

        // Final color composition
        vec3 finalColor = glowColor * blurredTexture.rgb;
        float finalAlpha = alpha * ageAlpha * vIntensity * uOpacity;

        // Add sparkle effect for high-intensity or named particles
        if (vIntensity > 0.8 || vType == 1.0) {
          float sparkle = sin(uTime * 10.0 + vAge * 5.0 + vId) * 0.5 + 0.5;
          finalColor += vec3(1.0, 1.0, 1.0) * sparkle * 0.3;
        }

        gl_FragColor = vec4(finalColor, finalAlpha);
      }
    `;

    // Create shader material with enhanced uniforms (only for WebGL)
    if (isWebGPU) {
      console.warn(
        '[EnhancedParticleSystem] Attempted to create ShaderMaterial with WebGPU - using PointsMaterial fallback'
      );

      // Final fallback: use basic PointsMaterial for WebGPU
      const material = new THREE.PointsMaterial({
        size: 8.0,
        map: texture,
        transparent: true,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
        vertexColors: true,
        sizeAttenuation: true,
        opacity: 0.8
      });

      return material;
    }

    const material = new THREE.ShaderMaterial({
      uniforms: {
        uTexture: { value: texture },
        uTime: { value: 0.0 },
        uOpacity: { value: 0.8 },
        uPixelRatio: { value: Math.min(window.devicePixelRatio, 2) }
      },
      vertexShader,
      fragmentShader,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      vertexColors: true
    });

    return material;
  }, [createParticleTexture, threeModules, recommendedRenderer, isWebGPUAvailable]);

  // Utility function to validate and sanitize position values
  const validatePosition = useCallback(
    (x: number, y: number, z: number): [number, number, number] => {
      const safeX = isFinite(x) ? x : (Math.random() - 0.5) * 10;
      const safeY = isFinite(y) ? y : (Math.random() - 0.5) * 10;
      const safeZ = isFinite(z) ? z : (Math.random() - 0.5) * 10;
      return [safeX, safeY, safeZ];
    },
    []
  );

  // Update: Attribute buffers for 10-float format (position(3), velocity(3), phase(1), intensity(1), type(1), id(1))
  // Example for galaxy pattern, repeat for other patterns as needed
  const generateGalaxyPattern = useCallback(
    (params: typeof parameters.galaxy) => {
      const THREE = threeRef.current;
      if (!THREE)
        return {
          particleData: new Float32Array(0),
          positions: new Float32Array(0),
          colors: new Float32Array(0),
          scales: new Float32Array(0),
          velocities: new Float32Array(0),
          ages: new Float32Array(0),
          intensities: new Float32Array(0),
          phases: new Float32Array(0),
          types: new Float32Array(0),
          ids: new Float32Array(0)
        };

      const particleData = new Float32Array(params.count * 10);
      const positions = new Float32Array(params.count * 3);
      const colors = new Float32Array(params.count * 3);
      const scales = new Float32Array(params.count);
      const velocities = new Float32Array(params.count * 3);
      const ages = new Float32Array(params.count);
      const intensities = new Float32Array(params.count);
      const phases = new Float32Array(params.count);
      const types = new Float32Array(params.count);
      const ids = new Float32Array(params.count);

      const colorInside = new THREE.Color(params.innerColor);
      const colorOutside = new THREE.Color(params.outsideColor);

      let nanCount = 0;

      for (let i = 0; i < params.count; i++) {
        const i3 = i * 3;
        const i10 = i * 10;

        // Position
        const radius = Math.random() * params.radius;
        const spinAngle = radius * params.spin;
        const branchAngle = ((i % params.branches) / params.branches) * Math.PI * 2;

        const randomX =
          Math.pow(Math.random(), params.randomnessPower) *
          (Math.random() < 0.5 ? 1 : -1) *
          params.randomness *
          radius;
        const randomY =
          Math.pow(Math.random(), params.randomnessPower) *
          (Math.random() < 0.5 ? 1 : -1) *
          params.randomness *
          radius;
        const randomZ =
          Math.pow(Math.random(), params.randomnessPower) *
          (Math.random() < 0.5 ? 1 : -1) *
          params.randomness *
          radius;

        const x = Math.cos(branchAngle + spinAngle) * radius + randomX;
        const y = randomY;
        const z = Math.sin(branchAngle + spinAngle) * radius + randomZ;

        // Validate and assign positions
        const [safeX, safeY, safeZ] = validatePosition(x, y, z);
        if (x !== safeX || y !== safeY || z !== safeZ) nanCount++;

        // Store in 10-float format
        particleData[i10] = safeX; // position.x
        particleData[i10 + 1] = safeY; // position.y
        particleData[i10 + 2] = safeZ; // position.z
        particleData[i10 + 3] = (Math.random() - 0.5) * 0.1; // velocity.x
        particleData[i10 + 4] = (Math.random() - 0.5) * 0.1; // velocity.y
        particleData[i10 + 5] = (Math.random() - 0.5) * 0.1; // velocity.z
        particleData[i10 + 6] = Math.random() * Math.PI * 2; // phase
        particleData[i10 + 7] = 0.5 + Math.random() * 0.5; // intensity
        particleData[i10 + 8] = 0; // ptype (regular)
        particleData[i10 + 9] = i; // id

        // Store for Three.js rendering
        positions[i3] = safeX;
        positions[i3 + 1] = safeY;
        positions[i3 + 2] = safeZ;
        velocities[i3] = particleData[i10 + 3];
        velocities[i3 + 1] = particleData[i10 + 4];
        velocities[i3 + 2] = particleData[i10 + 5];
        phases[i] = particleData[i10 + 6];
        intensities[i] = particleData[i10 + 7];
        // ptype for rendering
        types[i] = particleData[i10 + 8];
        ids[i] = particleData[i10 + 9];
        ages[i] = Math.random() * 5.0;

        // Color
        const mixedColor = colorInside.clone();
        mixedColor.lerp(colorOutside, radius / params.radius);
        colors[i3] = mixedColor.r;
        colors[i3 + 1] = mixedColor.g;
        colors[i3 + 2] = mixedColor.b;

        scales[i] = Math.random() * 2.0 + 0.5;
      }

      if (nanCount > 0) {
        logger.warn(`Galaxy pattern generated ${nanCount} NaN values, corrected to safe positions`);
      }

      return {
        particleData,
        positions,
        colors,
        scales,
        velocities,
        ages,
        intensities,
        phases,
        types,
        ids
      };
    },
    [validatePosition]
  );

  // Generate yin-yang pattern
  const generateYinYangPattern = useCallback(
    (params: (typeof parameters)['yin-yang']) => {
      const THREE = threeRef.current;
      if (!THREE)
        return {
          particleData: new Float32Array(0),
          positions: new Float32Array(0),
          colors: new Float32Array(0),
          scales: new Float32Array(0),
          velocities: new Float32Array(0),
          ages: new Float32Array(0),
          intensities: new Float32Array(0),
          phases: new Float32Array(0),
          types: new Float32Array(0),
          ids: new Float32Array(0)
        };

      const particleData = new Float32Array(params.count * 10);
      const positions = new Float32Array(params.count * 3);
      const colors = new Float32Array(params.count * 3);
      const scales = new Float32Array(params.count);
      const velocities = new Float32Array(params.count * 3);
      const ages = new Float32Array(params.count);
      const intensities = new Float32Array(params.count);
      const phases = new Float32Array(params.count);
      const types = new Float32Array(params.count);
      const ids = new Float32Array(params.count);

      const yinColor = new THREE.Color(params.yinColor);
      const yangColor = new THREE.Color(params.yangColor);

      let nanCount = 0;

      for (let i = 0; i < params.count; i++) {
        const i3 = i * 3;
        const i10 = i * 10;

        const angle = (i / params.count) * Math.PI * 4;
        const radius = 2 + (i % 200) * 0.03;
        const spiral = (i / params.count) * Math.PI * 6;

        const isYin = Math.sin(angle * 2) > 0;
        const flowOffset = isYin ? 0 : Math.PI;

        const x = Math.cos(angle + flowOffset) * radius + Math.sin(spiral) * 1.5;
        const y = Math.sin(angle + flowOffset) * radius + Math.cos(spiral) * 1.5;
        const z = Math.sin(angle * 3) * 2 + Math.cos(spiral * 0.5) * 1;

        const randomX = (Math.random() - 0.5) * 1;
        const randomY = (Math.random() - 0.5) * 1;
        const randomZ = (Math.random() - 0.5) * 1;

        // Validate and assign positions
        const [safeX, safeY, safeZ] = validatePosition(x + randomX, y + randomY, z + randomZ);
        if (x + randomX !== safeX || y + randomY !== safeY || z + randomZ !== safeZ) nanCount++;

        // Store in 10-float format
        particleData[i10] = safeX;
        particleData[i10 + 1] = safeY;
        particleData[i10 + 2] = safeZ;
        particleData[i10 + 3] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 4] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 5] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 6] = Math.random() * Math.PI * 2;
        particleData[i10 + 7] = 0.5 + Math.random() * 0.5;
        particleData[i10 + 8] = isYin ? 1 : 2; // ptype: 1=yin, 2=yang
        particleData[i10 + 9] = i;

        positions[i3] = safeX;
        positions[i3 + 1] = safeY;
        positions[i3 + 2] = safeZ;
        velocities[i3] = particleData[i10 + 3];
        velocities[i3 + 1] = particleData[i10 + 4];
        velocities[i3 + 2] = particleData[i10 + 5];
        phases[i] = particleData[i10 + 6];
        intensities[i] = particleData[i10 + 7];
        types[i] = particleData[i10 + 8];
        ids[i] = particleData[i10 + 9];
        ages[i] = Math.random() * 5.0;

        // Color assignment
        const color = isYin ? yinColor : yangColor;
        colors[i3] = color.r;
        colors[i3 + 1] = color.g;
        colors[i3 + 2] = color.b;

        scales[i] = Math.random() * 1.8 + 0.6;
      }

      if (nanCount > 0) {
        logger.warn(
          `Yin-Yang pattern generated ${nanCount} NaN values, corrected to safe positions`
        );
      }

      return {
        particleData,
        positions,
        colors,
        scales,
        velocities,
        ages,
        intensities,
        phases,
        types,
        ids
      };
    },
    [validatePosition]
  );

  // Generate wave pattern
  const generateWavePattern = useCallback(
    (params: typeof parameters.wave) => {
      const THREE = threeRef.current;
      if (!THREE)
        return {
          particleData: new Float32Array(0),
          positions: new Float32Array(0),
          colors: new Float32Array(0),
          scales: new Float32Array(0),
          velocities: new Float32Array(0),
          ages: new Float32Array(0),
          intensities: new Float32Array(0),
          phases: new Float32Array(0),
          types: new Float32Array(0),
          ids: new Float32Array(0)
        };

      const particleData = new Float32Array(params.count * 10);
      const positions = new Float32Array(params.count * 3);
      const colors = new Float32Array(params.count * 3);
      const scales = new Float32Array(params.count);
      const velocities = new Float32Array(params.count * 3);
      const ages = new Float32Array(params.count);
      const intensities = new Float32Array(params.count);
      const phases = new Float32Array(params.count);
      const types = new Float32Array(params.count);
      const ids = new Float32Array(params.count);

      const baseColor = new THREE.Color(params.baseColor);

      let nanCount = 0;

      for (let i = 0; i < params.count; i++) {
        const i3 = i * 3;
        const i10 = i * 10;

        const x = (Math.random() - 0.5) * 20;
        const z = (Math.random() - 0.5) * 20;
        const y = 0;

        // Validate and assign positions
        const [safeX, safeY, safeZ] = validatePosition(x, y, z);
        if (x !== safeX || y !== safeY || z !== safeZ) nanCount++;

        // Store in 10-float format
        particleData[i10] = safeX;
        particleData[i10 + 1] = safeY;
        particleData[i10 + 2] = safeZ;
        particleData[i10 + 3] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 4] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 5] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 6] = Math.random() * Math.PI * 2;
        particleData[i10 + 7] = 0.5 + Math.random() * 0.5;
        particleData[i10 + 8] = 0; // ptype: regular
        particleData[i10 + 9] = i;

        positions[i3] = safeX;
        positions[i3 + 1] = safeY;
        positions[i3 + 2] = safeZ;
        velocities[i3] = particleData[i10 + 3];
        velocities[i3 + 1] = particleData[i10 + 4];
        velocities[i3 + 2] = particleData[i10 + 5];
        phases[i] = particleData[i10 + 6];
        intensities[i] = particleData[i10 + 7];
        types[i] = particleData[i10 + 8];
        ids[i] = particleData[i10 + 9];
        ages[i] = Math.random() * 5.0;

        // Color variation
        const colorVariation = Math.random() * 0.5 + 0.5;
        colors[i3] = baseColor.r * colorVariation;
        colors[i3 + 1] = baseColor.g * colorVariation;
        colors[i3 + 2] = baseColor.b * colorVariation;

        scales[i] = Math.random() * 1.5 + 0.7;
      }

      if (nanCount > 0) {
        logger.warn(`Wave pattern generated ${nanCount} NaN values, corrected to safe positions`);
      }

      return {
        particleData,
        positions,
        colors,
        scales,
        velocities,
        ages,
        intensities,
        phases,
        types,
        ids
      };
    },
    [validatePosition]
  );

  // Generate spiral pattern
  const generateSpiralPattern = useCallback(
    (params: typeof parameters.spiral) => {
      const THREE = threeRef.current;
      if (!THREE)
        return {
          particleData: new Float32Array(0),
          positions: new Float32Array(0),
          colors: new Float32Array(0),
          scales: new Float32Array(0),
          velocities: new Float32Array(0),
          ages: new Float32Array(0),
          intensities: new Float32Array(0),
          phases: new Float32Array(0),
          types: new Float32Array(0),
          ids: new Float32Array(0)
        };

      const particleData = new Float32Array(params.count * 10);
      const positions = new Float32Array(params.count * 3);
      const colors = new Float32Array(params.count * 3);
      const scales = new Float32Array(params.count);
      const velocities = new Float32Array(params.count * 3);
      const ages = new Float32Array(params.count);
      const intensities = new Float32Array(params.count);
      const phases = new Float32Array(params.count);
      const types = new Float32Array(params.count);
      const ids = new Float32Array(params.count);

      const baseColor = new THREE.Color(params.baseColor);

      let nanCount = 0;

      for (let i = 0; i < params.count; i++) {
        const i3 = i * 3;
        const i10 = i * 10;

        const t = i / params.count;
        const angle = t * Math.PI * params.tightness * params.spiralArms;
        const radius = t * params.radius;
        const height = (t - 0.5) * params.height;

        const armOffset = ((i % params.spiralArms) / params.spiralArms) * Math.PI * 2;

        const x = Math.cos(angle + armOffset) * radius + (Math.random() - 0.5) * 0.5;
        const y = height + (Math.random() - 0.5) * 0.3;
        const z = Math.sin(angle + armOffset) * radius + (Math.random() - 0.5) * 0.5;

        // Validate and assign positions
        const [safeX, safeY, safeZ] = validatePosition(x, y, z);
        if (x !== safeX || y !== safeY || z !== safeZ) nanCount++;

        // Store in 10-float format
        particleData[i10] = safeX;
        particleData[i10 + 1] = safeY;
        particleData[i10 + 2] = safeZ;
        particleData[i10 + 3] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 4] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 5] = (Math.random() - 0.5) * 0.1;
        particleData[i10 + 6] = Math.random() * Math.PI * 2;
        particleData[i10 + 7] = 0.5 + Math.random() * 0.5; // intensity
        particleData[i10 + 8] = 0; // ptype: regular
        particleData[i10 + 9] = i;

        positions[i3] = safeX;
        positions[i3 + 1] = safeY;
        positions[i3 + 2] = safeZ;
        velocities[i3] = particleData[i10 + 3];
        velocities[i3 + 1] = particleData[i10 + 4];
        velocities[i3 + 2] = particleData[i10 + 5];
        phases[i] = particleData[i10 + 6];
        intensities[i] = particleData[i10 + 7];
        types[i] = particleData[i10 + 8];
        ids[i] = particleData[i10 + 9];
        ages[i] = Math.random() * 5.0;

        // Color variation based on height and distance
        const colorFactor = (Math.abs(height) / (params.height / 2)) * 0.5 + 0.5;
        colors[i3] = baseColor.r * colorFactor;
        colors[i3 + 1] = baseColor.g * colorFactor;
        colors[i3 + 2] = baseColor.b * colorFactor;

        scales[i] = Math.random() * 1.6 + 0.8;
      }

      if (nanCount > 0) {
        logger.warn(`Spiral pattern generated ${nanCount} NaN values, corrected to safe positions`);
      }

      return {
        particleData,
        positions,
        colors,
        scales,
        velocities,
        ages,
        intensities,
        phases,
        types,
        ids
      };
    },
    [validatePosition]
  );

  // Generate pattern based on current animation mode
  // Utility: inject named particles into a pattern
  const injectNamedParticles = (
    pattern: any,
    namedParticles: Array<{ name: string; priority: number; color: string; scale: number }>
  ) => {
    // Accept any Float32Array regardless of ArrayBuffer type
    const { positions, colors, scales, types, particleData } = pattern;
    for (let i = 0; i < namedParticles.length && i < positions.length / 3; i++) {
      const { color, scale } = namedParticles[i];
      const i3 = i * 3;
      const i10 = i * 10;
      // Set color
      const c = new threeRef.current.Color(color);
      colors[i3] = c.r;
      colors[i3 + 1] = c.g;
      colors[i3 + 2] = c.b;
      // Set scale
      scales[i] = scale;
      // Set ptype to 1 (named)
      types[i] = 1;
      if (particleData) {
        particleData[i10 + 8] = 1;
      }
    }
    return pattern;
  };
  const generateCurrentPattern = useCallback(() => {
    switch (metrics.animationMode) {
      case 'galaxy':
        return generateGalaxyPattern(parameters.galaxy);
      case 'yin-yang':
        return generateYinYangPattern(parameters['yin-yang']);
      case 'wave':
        return generateWavePattern(parameters.wave);
      case 'spiral':
        return generateSpiralPattern(parameters.spiral);
      default:
        return generateGalaxyPattern(parameters.galaxy);
    }
  }, [
    metrics.animationMode,
    parameters,
    generateGalaxyPattern,
    generateYinYangPattern,
    generateWavePattern,
    generateSpiralPattern
  ]);

  // Update particles animation with optimized frame-based GPU processing
  const updateParticleAnimation = useCallback(
    async (positions: Float32Array, elapsedTime: number) => {
      // For WebGPU, use GPU buffer directly from WASM
      if (metrics.renderMode === 'WebGPU' && (window as any).getCurrentOutputBuffer) {
        const gpuBuffer = (window as any).getCurrentOutputBuffer();
        // If using Three.js WebGPURenderer, set vertex buffer directly
        if (rendererRef.current && rendererRef.current.constructor.name === 'WebGPURenderer') {
          if (
            rendererRef.current.renderPass &&
            typeof rendererRef.current.renderPass.setVertexBuffer === 'function'
          ) {
            rendererRef.current.renderPass.setVertexBuffer(0, gpuBuffer);
          }
        }
        // If using Three.js geometry, wrap GPU buffer as needed
        // ...additional Three.js WebGPU interop logic here...
        return;
      }

      const particleCount = positions.length / 3;
      const frameStart = performance.now();

      // Dynamic FPS target and GPU budget calculation
      const targetFPS = detectOptimalFPS();
      const frameBudget = 1000 / targetFPS; // ms per frame (8.33ms for 120fps, 16.67ms for 60fps, 33.33ms for 30fps)
      const gpuBudget = frameBudget * 0.6; // 60% of frame budget for GPU (5ms for 120fps, 10ms for 60fps, 20ms for 30fps)

      framePerformanceRef.current.targetFPS = targetFPS;
      framePerformanceRef.current.gpuBudget = gpuBudget;

      // Check if GPU delay has passed
      const gpuDelayPassed = Date.now() > gpuStartTimeRef.current;

      // Enhanced chunk-based processing with overlap to reduce visual boundaries
      // Dynamically set chunk size based on GPU device limits
      let maxChunkFloats = 50000; // Lower from 150000 to 50000 for smaller chunks in fallback
      if (gpuCapabilities && gpuCapabilities.maxStorageBufferBindingSize) {
        // Divide by 4 (bytes per float32) and round down to nearest multiple of 3
        maxChunkFloats = Math.floor(gpuCapabilities.maxStorageBufferBindingSize / 4 / 3) * 3;
      }
      const PARTICLES_PER_CHUNK = Math.floor(maxChunkFloats / 3);
      const CHUNK_OVERLAP = 1998; // Overlap between chunks for smoother transitions (divisible by 3)
      const CHUNK_SIZE = PARTICLES_PER_CHUNK * 3; // 3 floats per particle (x, y, z)
      const totalChunks = Math.ceil(positions.length / CHUNK_SIZE);

      // Rotate through chunks each frame for continuous processing
      const frameNumber = Math.floor(elapsedTime * targetFPS);
      const currentChunkIndex = frameNumber % totalChunks;
      chunkIndexRef.current = currentChunkIndex;

      // Process chunk if conditions are met
      const shouldUseGPU =
        particleCount > 50000 &&
        gpuDelayPassed &&
        framePerformanceRef.current.lastFrameTime < gpuBudget;

      // Debug: Log performance metrics occasionally
      if (Date.now() % 3000 < 100) {
        logger.info(
          `Enhanced Frame Performance: Target ${targetFPS}fps, Budget ${frameBudget.toFixed(1)}ms, GPU Budget ${gpuBudget.toFixed(1)}ms, Last Frame ${framePerformanceRef.current.lastFrameTime.toFixed(1)}ms, Chunk ${currentChunkIndex}/${totalChunks} (50k+overlap), useGPU: ${shouldUseGPU}`
        );
      }

      // Use centralized WASM GPU bridge for chunk processing
      if (
        shouldUseGPU &&
        metrics.computeMode.includes('WASM') &&
        wasmGPU.isInitialized() &&
        metrics.wasmReady &&
        !gpuProcessingRef.current
      ) {
        gpuProcessingRef.current = true;

        try {
          // Calculate chunk boundaries with overlap for smoother transitions
          const startOffset = Math.max(0, currentChunkIndex * CHUNK_SIZE - CHUNK_OVERLAP);
          const endOffset = Math.min(
            positions.length,
            (currentChunkIndex + 1) * CHUNK_SIZE + CHUNK_OVERLAP
          );
          let chunkLength = endOffset - startOffset;

          // Ensure chunk length is divisible by 3 (for x, y, z coordinates)
          chunkLength = Math.floor(chunkLength / 3) * 3;
          const adjustedEndOffset = startOffset + chunkLength;

          // Create 3-float position chunk
          const positionChunk = positions.slice(startOffset, adjustedEndOffset);

          // Skip empty or invalid chunks
          if (positionChunk.length === 0 || positionChunk.length % 3 !== 0) {
            logger.warn(
              `[WASM-GPU-Bridge] Skipping invalid chunk ${currentChunkIndex}: length=${positionChunk.length}`
            );
            return;
          }

          // Extract chunk from unified particle data (8-float format)
          const particleCount = Math.floor(chunkLength / 3); // Based on 3-float position chunk

          let wasmChunk: Float32Array;

          // Only log processing for every 4th chunk or first/last chunk to reduce verbosity
          const shouldLog =
            currentChunkIndex === 0 ||
            currentChunkIndex === totalChunks - 1 ||
            currentChunkIndex % 4 === 0;

          if (shouldLog) {
            logger.info(
              `[WASM-GPU-Bridge] Processing SYNCHRONIZED chunk ${currentChunkIndex}/${totalChunks}: ${particleCount} particles (8-float format, global offset ${startOffset / 3})`
            );
          }

          // Create 8-float chunk from 3-float positions
          wasmChunk = new Float32Array(particleCount * 8);
          for (let i = 0; i < particleCount; i++) {
            const posBase = i * 3;
            const wasmBase = i * 8;
            // Position (3 floats)
            wasmChunk[wasmBase] = positionChunk[posBase];
            wasmChunk[wasmBase + 1] = positionChunk[posBase + 1];
            wasmChunk[wasmBase + 2] = positionChunk[posBase + 2];
            // Velocity (3 floats) - initialize with small random values
            wasmChunk[wasmBase + 3] = (Math.random() - 0.5) * 0.1;
            wasmChunk[wasmBase + 4] = (Math.random() - 0.5) * 0.1;
            wasmChunk[wasmBase + 5] = (Math.random() - 0.5) * 0.1;
            // Time and intensity (2 floats)
            wasmChunk[wasmBase + 6] = elapsedTime;
            wasmChunk[wasmBase + 7] = 1.0; // intensity
          }

          // Use synchronized processing with global particle offset for coherent animation
          const result = await wasmGPU.runParticlePhysicsWithOffset(
            wasmChunk,
            elapsedTime,
            startOffset / 3 // Convert to particle count for global indexing
          );

          if (result && result.length >= particleCount * 8) {
            // WASM returns 8-float format, extract positions for Three.js
            let changedCount = 0;
            let nanCount = 0;

            for (let i = 0; i < particleCount; i++) {
              const resultBase = i * 8;
              const positionBase = (startOffset / 3 + i) * 3;

              // Extract position from 8-float WASM result
              const newX = result[resultBase];
              const newY = result[resultBase + 1];
              const newZ = result[resultBase + 2];

              // Validate WASM result before using it
              if (isFinite(newX) && isFinite(newY) && isFinite(newZ)) {
                const diffX = Math.abs(positions[positionBase] - newX);
                const diffY = Math.abs(positions[positionBase + 1] - newY);
                const diffZ = Math.abs(positions[positionBase + 2] - newZ);

                if (diffX > 0.001 || diffY > 0.001 || diffZ > 0.001) {
                  positions[positionBase] = newX;
                  positions[positionBase + 1] = newY;
                  positions[positionBase + 2] = newZ;
                  changedCount++;
                }
              } else {
                nanCount++;
              }
            }

            if (shouldLog && (changedCount > 0 || nanCount > 0)) {
              logger.info(
                `[WASM-GPU-Bridge] ✅ Chunk ${currentChunkIndex} processed: ${changedCount} particles updated, ${nanCount} NaN values filtered (8-float format)`
              );
            }

            // Mark positions as needing update for Three.js
            if (particlesRef.current && particlesRef.current.geometry) {
              particlesRef.current.geometry.attributes.position.needsUpdate = true;
            }
          }
        } catch (error) {
          logger.error(`[WASM-GPU-Bridge] Chunk ${currentChunkIndex} processing failed:`, error);
        } finally {
          gpuProcessingRef.current = false;
        }
      } else {
        // CPU fallback for current chunk with overlap
        const startOffset = Math.max(0, currentChunkIndex * CHUNK_SIZE - CHUNK_OVERLAP);
        let endOffset = Math.min(
          positions.length,
          (currentChunkIndex + 1) * CHUNK_SIZE + CHUNK_OVERLAP
        );

        // Ensure chunk length is divisible by 3 for CPU fallback as well
        const chunkLength = endOffset - startOffset;
        const adjustedChunkLength = Math.floor(chunkLength / 3) * 3;
        endOffset = startOffset + adjustedChunkLength;

        if (adjustedChunkLength > 0) {
          performCPUAnimation(startOffset, endOffset);
        }
      }

      // Record frame performance
      const frameEnd = performance.now();
      framePerformanceRef.current.lastFrameTime = frameEnd - frameStart;

      function performCPUAnimation(startOffset: number, endOffset: number) {
        let animatedCount = 0;
        for (let offset = startOffset; offset < endOffset; offset += 3) {
          const i = offset / 3;

          // Safety check: validate current positions before animation
          if (
            !isFinite(positions[offset]) ||
            !isFinite(positions[offset + 1]) ||
            !isFinite(positions[offset + 2])
          ) {
            // Reset to safe default if position is invalid
            positions[offset] = (Math.random() - 0.5) * 10;
            positions[offset + 1] = (Math.random() - 0.5) * 10;
            positions[offset + 2] = (Math.random() - 0.5) * 10;
          }

          switch (metrics.animationMode) {
            case 'galaxy':
              const radius = Math.sqrt(positions[offset] ** 2 + positions[offset + 2] ** 2);
              // Prevent division by zero and NaN in atan2
              const angle =
                radius > 0.001
                  ? Math.atan2(positions[offset + 2], positions[offset]) +
                    elapsedTime * 0.8 * (1 + radius * 0.05)
                  : elapsedTime * 0.8;

              const newX = radius * Math.cos(angle);
              const newZ = radius * Math.sin(angle);
              const newY = positions[offset + 1] + Math.sin(elapsedTime * 1.5 + i * 0.01) * 0.05;

              // Validate before assignment
              if (isFinite(newX) && isFinite(newZ) && isFinite(newY)) {
                positions[offset] = newX;
                positions[offset + 2] = newZ;
                positions[offset + 1] = newY;
              }
              break;

            case 'yin-yang':
              const centerX = positions[offset];
              const centerZ = positions[offset + 2];
              const dist = Math.sqrt(centerX ** 2 + centerZ ** 2);
              // Prevent NaN in atan2 when both arguments are 0
              const flowAngle =
                dist > 0.001 ? Math.atan2(centerZ, centerX) + elapsedTime * 2.0 : elapsedTime * 2.0;

              const yinX = dist * Math.cos(flowAngle);
              const yinZ = dist * Math.sin(flowAngle);
              const yinY = positions[offset + 1] + Math.sin(elapsedTime * 4 + i * 0.1) * 0.1;

              // Validate before assignment
              if (isFinite(yinX) && isFinite(yinZ) && isFinite(yinY)) {
                positions[offset] = yinX;
                positions[offset + 2] = yinZ;
                positions[offset + 1] = yinY;
              }
              break;

            case 'wave':
              // Validate input positions before calculations
              if (isFinite(positions[offset]) && isFinite(positions[offset + 2])) {
                const waveY =
                  Math.sin(positions[offset] * 0.5 + elapsedTime * 6) *
                  Math.cos(positions[offset + 2] * 0.5 + elapsedTime * 4) *
                  5;

                // Validate result before assignment
                if (isFinite(waveY)) {
                  positions[offset + 1] = waveY;
                }
              }
              break;

            case 'spiral':
              const spiralRadius = Math.sqrt(positions[offset] ** 2 + positions[offset + 2] ** 2);
              // Prevent NaN in atan2 when radius is 0
              const spiralAngle =
                spiralRadius > 0.001
                  ? Math.atan2(positions[offset + 2], positions[offset]) + elapsedTime * 1.5
                  : elapsedTime * 1.5;

              const spiralX = spiralRadius * Math.cos(spiralAngle);
              const spiralZ = spiralRadius * Math.sin(spiralAngle);
              const spiralY =
                positions[offset + 1] + Math.sin(elapsedTime * 3 + spiralRadius * 0.2) * 0.15;

              // Validate before assignment
              if (isFinite(spiralX) && isFinite(spiralZ) && isFinite(spiralY)) {
                positions[offset] = spiralX;
                positions[offset + 2] = spiralZ;
                positions[offset + 1] = spiralY;
              }
              break;
          }
          animatedCount++;
        }

        if (particlesRef.current && particlesRef.current.geometry) {
          particlesRef.current.geometry.attributes.position.needsUpdate = true;
        }
      }
    },
    [
      metrics.computeMode,
      metrics.wasmReady,
      metrics.animationMode,
      detectOptimalFPS,
      metrics.renderMode
    ]
  );

  // Initialize scene
  useEffect(() => {
    if (!canvasRef.current || !threeLoaded || !threeModules) return;

    logger.info(`Initializing Enhanced Particle System with ${metrics.animationMode} animation...`);

    const { core: THREE, renderers, addons } = threeModules;
    const { WebGPURenderer } = renderers;
    const { OrbitControls } = addons;
    const canvas = canvasRef.current;

    // Wait for canvas to be properly sized
    const waitForCanvas = () => {
      return new Promise<void>(resolve => {
        const checkSize = () => {
          if (canvas.clientWidth > 0 && canvas.clientHeight > 0) {
            resolve();
          } else {
            setTimeout(checkSize, 10);
          }
        };
        checkSize();
      });
    };

    // Renderer setup with enhanced WebGPU automatic detection using global state
    const initializeRenderer = async () => {
      let renderer = null;

      // Use GPU capabilities from global state for better detection
      const useWebGPU = isWebGPUAvailable && recommendedRenderer === 'webgpu';
      const wasmWebgpuReady =
        typeof window.initWebGPU === 'function' && typeof window.runGPUCompute === 'function';

      logger.info(
        `WebGPU Support - Available: ${isWebGPUAvailable}, Recommended: ${recommendedRenderer}, WASM: ${wasmWebgpuReady}`
      );
      logger.info('GPU Capabilities:', gpuCapabilities);

      // Emit particle system initialization event
      emitEvent({
        type: 'particle-system:init:v1:started',
        payload: {
          webgpuAvailable: isWebGPUAvailable,
          recommendedRenderer,
          wasmWebgpuReady,
          gpuCapabilities
        },
        metadata: {} as any
      });

      // Try WebGPU first if it's recommended and available
      if (useWebGPU) {
        try {
          logger.info('🚀 Attempting WebGPU renderer initialization (from global state)...');

          // Ensure canvas has proper dimensions before creating renderer
          const canvasWidth = canvas.clientWidth || 800;
          const canvasHeight = canvas.clientHeight || 600;

          if (canvasWidth === 0 || canvasHeight === 0) {
            throw new Error(`Invalid canvas dimensions: ${canvasWidth}x${canvasHeight}`);
          }

          logger.info(`Canvas dimensions: ${canvasWidth}x${canvasHeight}`);

          const webgpuRenderer = new WebGPURenderer({
            canvas,
            antialias: false, // Disable for compatibility
            alpha: true,
            powerPreference: 'high-performance'
          });

          // Add timeout to WebGPU initialization
          const initPromise = webgpuRenderer.init();
          const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => reject(new Error('WebGPU init timeout')), 5000)
          );

          await Promise.race([initPromise, timeoutPromise]);

          // Set size with validated dimensions
          webgpuRenderer.setSize(canvasWidth, canvasHeight);
          webgpuRenderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

          renderer = webgpuRenderer;
          rendererInitializedRef.current = true;

          // Update metrics to reflect successful WebGPU initialization
          const newRenderMode = wasmWebgpuReady ? 'WebGPU+WASM' : 'WebGPU';
          const newComputeMode = wasmWebgpuReady ? 'WASM+WebGPU' : 'JS';

          setMetrics(prev => ({
            ...prev,
            renderMode: newRenderMode,
            computeMode: newComputeMode,
            webgpuReady: true,
            wasmReady: wasmWebgpuReady,
            connectionStrength: Math.max(prev.connectionStrength, wasmWebgpuReady ? 1.0 : 0.8)
          }));

          // Update current render mode to trigger parameter recalculation
          setCurrentRenderMode(newRenderMode);

          logger.info(
            `✅ WebGPU renderer initialized successfully${wasmWebgpuReady ? ' with WASM support' : ''}`
          );

          // Emit successful WebGPU renderer initialization event
          emitEvent({
            type: 'particle-system:renderer:v1:webgpu-success',
            payload: {
              rendererType: 'webgpu',
              wasmSupport: wasmWebgpuReady,
              canvasSize: { width: canvasWidth, height: canvasHeight }
            },
            metadata: {} as any
          });

          return renderer;
        } catch (error) {
          logger.warn('WebGPU initialization failed, falling back to WebGL:', error);
          renderer = null;
        }
      } else {
        logger.info('WebGPU not supported by browser, using WebGL');
      }

      // WebGL fallback with optimized settings
      if (!renderer) {
        const webglRenderer = new THREE.WebGLRenderer({
          canvas,
          antialias: true,
          alpha: true,
          powerPreference: 'high-performance',
          preserveDrawingBuffer: false,
          stencil: false
        });

        webglRenderer.setSize(canvas.clientWidth, canvas.clientHeight);
        webglRenderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

        renderer = webglRenderer;
        rendererInitializedRef.current = true;
        logger.info('✅ WebGL renderer initialized');

        // Emit WebGL renderer initialization event
        emitEvent({
          type: 'particle-system:renderer:v1:webgl-success',
          payload: {
            rendererType: 'webgl',
            fallbackReason: useWebGPU ? 'webgpu-failed' : 'webgl-preferred',
            canvasSize: { width: canvas.clientWidth, height: canvas.clientHeight }
          },
          metadata: {} as any
        });
      }

      return renderer;
    };

    const setupScene = async () => {
      // Wait for canvas to be properly sized before initializing
      await waitForCanvas();

      const scene = new THREE.Scene();
      scene.background = new THREE.Color(0x000000); // Pure black background for maximum darkness

      // Camera setup - optimized for new scale
      const camera = new THREE.PerspectiveCamera(
        75,
        canvas.clientWidth / canvas.clientHeight,
        0.1,
        200
      );
      // Cinematic camera: start at a dramatic angle and distance
      camera.position.set(16, 10, 18);
      camera.lookAt(0, 0, 0);
      camera.fov = 65;
      camera.updateProjectionMatrix();

      // Add subtle fog for depth (atmospheric effect)
      scene.fog = new threeRef.current.Fog(0x000000, 40, 120);

      // Store camera for animation
      camera.userData.cinematic = {
        orbitRadius: 18,
        orbitHeight: 8,
        orbitSpeed: 0.18,
        fovBase: 65,
        fovRange: 10
      };

      logger.info(
        `Camera positioned at: ${camera.position.x}, ${camera.position.y}, ${camera.position.z}`
      );

      const renderer = await initializeRenderer();

      // Generate initial pattern
      const particleCount = parameters[metrics.animationMode].count;
      logger.info(
        `🚀 Generating ${metrics.animationMode} pattern with ${particleCount.toLocaleString()} particles for ${currentRenderMode} rendering...`
      );

      if (currentRenderMode.includes('WebGPU')) {
        const performanceLevel =
          currentRenderMode.includes('WASM') && particleCount >= 60000
            ? 'ULTRA'
            : particleCount >= 40000
              ? 'HIGH'
              : 'ENHANCED';
        logger.info(
          `💪 WebGPU Performance Mode: ${performanceLevel} particle density! (${particleCount.toLocaleString()} particles)`
        );
      }

      // Generate pattern and inject named particles
      let pattern = generateCurrentPattern();
      if (namedParticles && namedParticles.length > 0) {
        pattern = injectNamedParticles(pattern as any, namedParticles);
        logger.info(`Injected ${namedParticles.length} named particles into pattern.`);
      }
      const { positions, colors, scales, velocities, ages, intensities, phases, types, ids } =
        pattern;

      logger.info(
        `Generated pattern: positions=${positions.length}, colors=${colors.length}, scales=${scales.length}, velocities=${velocities?.length || 0}, ages=${ages?.length || 0}, intensities=${intensities?.length || 0}`
      );

      // Debug: Check position bounds
      let minX = Infinity,
        maxX = -Infinity,
        minY = Infinity,
        maxY = -Infinity,
        minZ = Infinity,
        maxZ = -Infinity;
      for (let i = 0; i < positions.length; i += 3) {
        minX = Math.min(minX, positions[i]);
        maxX = Math.max(maxX, positions[i]);
        minY = Math.min(minY, positions[i + 1]);
        maxY = Math.max(maxY, positions[i + 1]);
        minZ = Math.min(minZ, positions[i + 2]);
        maxZ = Math.max(maxZ, positions[i + 2]);
      }
      logger.info(
        `Position bounds: X[${minX.toFixed(2)}, ${maxX.toFixed(2)}], Y[${minY.toFixed(2)}, ${maxY.toFixed(2)}], Z[${minZ.toFixed(2)}, ${maxZ.toFixed(2)}]`
      );

      // Final validation: sanitize any remaining NaN values before creating geometry
      let finalNanCount = 0;
      for (let i = 0; i < positions.length; i += 3) {
        if (!isFinite(positions[i]) || !isFinite(positions[i + 1]) || !isFinite(positions[i + 2])) {
          positions[i] = (Math.random() - 0.5) * 10;
          positions[i + 1] = (Math.random() - 0.5) * 10;
          positions[i + 2] = (Math.random() - 0.5) * 10;
          finalNanCount++;
        }
      }

      if (finalNanCount > 0) {
        logger.warn(
          `🔧 Final sanitization: corrected ${finalNanCount} NaN positions before geometry creation`
        );
      }

      // Create geometry with enhanced attributes for advanced shader effects
      // Attribute pointers must match the buffer format:
      // position: [0,1,2], velocity: [3,4,5], phase: [6], intensity: [7], type: [8], id: [9]
      const geometry = new THREE.BufferGeometry();
      geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
      geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
      geometry.setAttribute('scale', new THREE.BufferAttribute(scales, 1));

      // Enhanced attributes for advanced particle effects
      if (velocities && ages && intensities) {
        geometry.setAttribute('velocity', new THREE.BufferAttribute(velocities, 3)); // [3,4,5]
        geometry.setAttribute('age', new THREE.BufferAttribute(ages, 1)); // custom
        geometry.setAttribute('intensity', new THREE.BufferAttribute(intensities, 1)); // [7]
        // Wire new attributes for advanced shader support
        if (phases) geometry.setAttribute('phase', new THREE.BufferAttribute(phases, 1)); // [6]
        if (types) geometry.setAttribute('type', new THREE.BufferAttribute(types, 1)); // [8]
        if (ids) geometry.setAttribute('id', new THREE.BufferAttribute(ids, 1)); // [9]
      }

      // Add UV coordinates for texture compatibility (required for WebGPU)
      const uvs = new Float32Array((positions.length / 3) * 2);
      for (let i = 0; i < uvs.length; i += 2) {
        uvs[i] = 0.5; // u coordinate
        uvs[i + 1] = 0.5; // v coordinate
      }
      geometry.setAttribute('uv', new THREE.BufferAttribute(uvs, 2));

      logger.info(
        `Created enhanced geometry with ${positions.length / 3} particles and ${velocities ? 'advanced' : 'basic'} shader attributes`
      );

      // Create enhanced material with advanced shader effects
      let material;
      const hasEnhancedAttributes = velocities && ages && intensities;

      if (hasEnhancedAttributes) {
        material = createEnhancedParticleMaterial();
        logger.info(
          'Using enhanced shader material with motion blur, age-based evolution, and velocity orientation'
        );
      }

      // Fallback to standard material if enhanced material creation fails
      if (!material) {
        const texture = createParticleTexture();
        material = new THREE.PointsMaterial({
          size: parameters[metrics.animationMode].size * 5, // 5x size for good visibility
          map: texture,
          transparent: true,
          blending: THREE.AdditiveBlending,
          depthWrite: false,
          vertexColors: true,
          sizeAttenuation: true,
          opacity: 0.8 // Slightly reduced for better blending
        });

        // Override vertex shader to use scale attribute
        material.onBeforeCompile = shader => {
          shader.vertexShader = shader.vertexShader.replace(
            'attribute float size;',
            'attribute float size;\nattribute float scale;'
          );
          shader.vertexShader = shader.vertexShader.replace(
            'gl_PointSize = size;',
            'gl_PointSize = size * scale;'
          );
        };

        logger.info('Using fallback standard material with scale attribute');
      }

      logger.info(
        `Created material with base size: ${parameters[metrics.animationMode].size * 5} for ${metrics.animationMode} mode`
      );

      // Create points
      const particles = new THREE.Points(geometry, material);
      scene.add(particles);

      logger.info(`Added particles to scene, scene children: ${scene.children.length}`);

      // Setup controls - optimized for smaller scale
      const controls = new OrbitControls(camera, canvas);
      controls.enabled = true; // Explicitly enable controls
      controls.enableDamping = true;
      controls.dampingFactor = 0.06; // Smooth movement
      controls.enableZoom = true;
      controls.enableRotate = true; // Explicitly enable rotation
      controls.enablePan = true;
      controls.autoRotate = false; // Disable auto-rotate for manual control
      controls.autoRotateSpeed = 0; // Disabled
      controls.zoomSpeed = 1.0; // Standard zoom speed
      controls.rotateSpeed = 0.7; // Smooth rotation
      controls.panSpeed = 0.8; // Smooth panning
      controls.minDistance = 0.6; // Reduced from 3 to 0.6 for closer zoom capability (5x closer)
      controls.maxDistance = 20; // Reduced from 100 to 20 for closer view range (5x closer)
      controls.minPolarAngle = 0; // Allow full vertical rotation
      controls.maxPolarAngle = Math.PI; // Allow full vertical rotation
      controls.target.set(0, 0, 0); // Always look at center

      // Ensure controls listen to the correct DOM element
      controls.domElement = canvas;
      controls.update(); // Force initial update

      // Ensure canvas can receive focus for keyboard controls
      canvas.tabIndex = 0;
      canvas.focus();

      // Store references
      sceneRef.current = scene;
      rendererRef.current = renderer;
      cameraRef.current = camera;
      controlsRef.current = controls;
      particlesRef.current = particles;

      // Log controls setup for debugging
      logger.info('✅ Enhanced particle system initialized with OrbitControls enabled');
      // Add subtle ambient and directional lighting for depth
      const ambientLight = new threeRef.current.AmbientLight(0xffffff, 0.12);
      scene.add(ambientLight);
      const directionalLight = new threeRef.current.DirectionalLight(0x8888ff, 0.18);
      directionalLight.position.set(20, 40, 20);
      scene.add(directionalLight);
      logger.info(
        `Controls enabled: ${controls.enabled}, damping: ${controls.enableDamping}, zoom: ${controls.enableZoom}, rotate: ${controls.enableRotate}, pan: ${controls.enablePan}`
      );

      // Auto-connect to campaign
      try {
        await connectMediaStreamingToCampaign('0', 'enhanced-particles');
        logger.info('✅ Connected to campaign');
      } catch (error) {
        logger.warn('Failed to connect to campaign:', error);
      }
    };

    setupScene();

    return () => {
      // Stop animation first
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
        animationIdRef.current = null;
      }

      // Cleanup in proper order to avoid disposal errors
      if (controlsRef.current) {
        controlsRef.current.dispose();
        controlsRef.current = null;
      }

      if (particlesRef.current) {
        // Remove from scene first
        if (sceneRef.current) {
          sceneRef.current.remove(particlesRef.current);
        }

        // Dispose geometry
        if (particlesRef.current.geometry) {
          particlesRef.current.geometry.dispose();
        }

        // Dispose material carefully
        if (particlesRef.current.material) {
          const material = particlesRef.current.material;
          if (material instanceof THREE.Material) {
            // Dispose texture first if it exists (cast to PointsMaterial for map access)
            const pointsMaterial = material as any;
            if (pointsMaterial.map) {
              pointsMaterial.map.dispose();
            }
            material.dispose();
          }
        }
        particlesRef.current = null;
      }

      if (rendererRef.current) {
        // Clear the renderer first
        rendererRef.current.clear();
        rendererRef.current.dispose();
        rendererRef.current = null;
      }

      // Clear scene
      if (sceneRef.current) {
        sceneRef.current.clear();
        sceneRef.current = null;
      }

      // Clear other refs
      cameraRef.current = null;
      rendererInitializedRef.current = false;
    };
  }, [
    metrics.animationMode,
    metrics.renderMode,
    generateCurrentPattern,
    createParticleTexture,
    parameters,
    threeLoaded,
    threeModules
  ]);

  // Media streaming monitoring
  // No need for local subscription, state is managed by global store

  // Animation loop with enhanced frame timing
  useEffect(() => {
    let frameCount = 0;
    let lastTime = 0;
    let fpsHistory: number[] = []; // Track FPS history for adaptive targeting

    const animate = (currentTime: number) => {
      if (
        !sceneRef.current ||
        !rendererRef.current ||
        !cameraRef.current ||
        !particlesRef.current ||
        !rendererInitializedRef.current
      ) {
        animationIdRef.current = requestAnimationFrame(animate);
        return;
      }

      frameCount++;
      const deltaTime = currentTime - lastTime;
      const elapsedTime = clockRef.current.getElapsedTime();

      // Update FPS and GPU metrics
      if (deltaTime >= 1000) {
        const fps = Math.round((frameCount * 1000) / deltaTime);

        // Track FPS history for adaptive performance
        fpsHistory.push(fps);
        if (fpsHistory.length > 10) fpsHistory.shift(); // Keep last 10 seconds
        const avgFPS = fpsHistory.reduce((a, b) => a + b, 0) / fpsHistory.length;

        // Adaptive GPU budget based on actual performance
        const targetFPS = detectOptimalFPS();
        const isPerformingWell = avgFPS >= targetFPS * 0.9; // 90% of target

        if (!isPerformingWell && framePerformanceRef.current.gpuBudget > 2) {
          framePerformanceRef.current.gpuBudget *= 0.9; // Reduce GPU budget if struggling
        } else if (isPerformingWell && framePerformanceRef.current.gpuBudget < 15) {
          framePerformanceRef.current.gpuBudget *= 1.05; // Increase GPU budget if performing well
        }

        // Enhanced GPU metrics collection with performance data
        let newMetrics: Partial<ParticleMetrics> = {
          fps,
          frameTime: deltaTime / frameCount
        };

        if (metrics.wasmReady && window.getGPUMetricsBuffer) {
          try {
            const gpuBuffer = new Float32Array(window.getGPUMetricsBuffer());

            newMetrics = {
              ...newMetrics,
              gpuUtilization: Math.min(100, (gpuBuffer[5] || 0) * 10),
              frameTime: 1000 / fps,
              workerMetrics: {
                activeWorkers: gpuBuffer[13] || 0,
                totalWorkers: gpuBuffer[14] || 8,
                queueDepth: gpuBuffer[15] || 0,
                throughput: gpuBuffer[6] || 0,
                avgLatency: gpuBuffer[7] || 0,
                peakThroughput: gpuBuffer[12] || 0,
                tasksProcessed: gpuBuffer[10] || 0
              }
            };
          } catch (error) {
            console.warn('Error reading GPU metrics buffer:', error);
          }
        }

        setMetrics(prev => ({ ...prev, ...newMetrics }));

        // Debug log performance every 5 seconds
        if (frameCount % (targetFPS * 5) === 0 && particlesRef.current) {
          const particleCount = particlesRef.current.geometry.attributes.position.count;
          logger.info(
            `Performance: ${fps}fps (avg: ${avgFPS.toFixed(1)}, target: ${targetFPS}), ${particleCount} particles, GPU budget: ${framePerformanceRef.current.gpuBudget.toFixed(1)}ms, chunk: ${chunkIndexRef.current}`
          );
        }

        frameCount = 0;
        lastTime = currentTime;
      }

      // Update controls (essential for OrbitControls to work)
      if (controlsRef.current) {
        controlsRef.current.update();
      }

      // Update particle animation with RAF time for precise timing
      const geometry = particlesRef.current.geometry;
      const positions = geometry.attributes.position.array as Float32Array;

      // Update enhanced shader uniforms if using enhanced material
      const material = particlesRef.current.material as any;
      if (material.uniforms && material.uniforms.uTime) {
        material.uniforms.uTime.value = elapsedTime;
      }

      // Handle async particle animation with RAF timing
      updateParticleAnimation(positions, elapsedTime).catch(error => {
        logger.error('Particle animation failed:', error);
      });

      geometry.attributes.position.needsUpdate = true;
      geometry.attributes.color.needsUpdate = true;

      // Update enhanced attributes if they exist
      if (geometry.attributes.velocity) {
        geometry.attributes.velocity.needsUpdate = true;
      }
      if (geometry.attributes.age) {
        // Gradually increase age over time
        const ages = geometry.attributes.age.array as Float32Array;
        for (let i = 0; i < ages.length; i++) {
          ages[i] += deltaTime / 1000; // Convert to seconds
        }
        geometry.attributes.age.needsUpdate = true;
      }
      if (geometry.attributes.intensity) {
        geometry.attributes.intensity.needsUpdate = true;
      }

      // Enhanced rendering with better WebGPU/WebGL compatibility
      try {
        if (rendererRef.current && sceneRef.current && cameraRef.current) {
          if (rendererRef.current && rendererRef.current.constructor.name === 'WebGPURenderer') {
            // WebGPU renderer - check backend status
            const backend = (rendererRef.current as any).backend;
            if (backend && backend.device) {
              // True WebGPU backend - use renderAsync
              rendererRef.current.renderAsync(sceneRef.current, cameraRef.current).catch(() => {
                // Fallback to sync render on async failure
                if (rendererRef.current && sceneRef.current && cameraRef.current) {
                  rendererRef.current.render(sceneRef.current, cameraRef.current);
                }
              });
            } else {
              // WebGPU running on WebGL2 backend
              rendererRef.current.render(sceneRef.current, cameraRef.current);
            }
          } else {
            // Standard WebGL renderer
            rendererRef.current.render(sceneRef.current, cameraRef.current);
          }
        }
      } catch (error) {
        logger.warn('Render error, continuing animation:', error);
        // Don't stop animation on render errors
      }

      animationIdRef.current = requestAnimationFrame(animate);
    };

    animationIdRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
      }
    };
  }, [updateParticleAnimation, detectOptimalFPS]);

  // Handle resize
  useEffect(() => {
    const handleResize = () => {
      if (!canvasRef.current || !rendererRef.current || !cameraRef.current) return;

      const canvas = canvasRef.current;
      cameraRef.current.aspect = canvas.clientWidth / canvas.clientHeight;
      cameraRef.current.updateProjectionMatrix();
      rendererRef.current.setSize(canvas.clientWidth, canvas.clientHeight);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Add debug functionality for particle analysis
  useEffect(() => {
    (window as any).debugParticles = () => {
      console.log('Current Animation Mode:', metrics.animationMode);
      console.log('Active Chunk:', chunkIndexRef.current);
      console.log('GPU Processing:', gpuProcessingRef.current);
      console.log('Target FPS:', framePerformanceRef.current.targetFPS);
      console.log('GPU Budget:', framePerformanceRef.current.gpuBudget.toFixed(1) + 'ms');
      console.log('Particle Count:', metrics.particleCount);
      console.log('Render Mode:', metrics.renderMode);
      console.log('Compute Mode:', metrics.computeMode);
    };

    (window as any).resetParticles = () => {
      const { positions } = generateCurrentPattern();
      if (particlesRef.current) {
        const positionAttribute = particlesRef.current.geometry.attributes.position;
        // Copy new positions to the existing buffer
        (positionAttribute.array as Float32Array).set(positions);
        positionAttribute.needsUpdate = true;
        logger.info('Particles reset to initial pattern');
      }
    };

    return () => {
      delete (window as any).debugParticles;
      delete (window as any).resetParticles;
    };
  }, [metrics, generateCurrentPattern]);

  const switchAnimationMode = (mode: ParticleMetrics['animationMode']) => {
    setMetrics(prev => ({
      ...prev,
      animationMode: mode,
      particleCount: parameters[mode].count
    }));
  };

  // Move this conditional below all hooks!
  if (
    typeof window !== 'undefined' &&
    ((window as any).isPageUnloading || (window as any).isShuttingDown)
  ) {
    return (
      <div style={{ textAlign: 'center', padding: '2em', color: '#888' }}>
        Particle system is shutting down...
      </div>
    );
  }

  return (
    <div style={{ position: 'relative', width: '100%', height: '100vh' }}>
      {/* Named Particles UI */}
      <div
        style={{
          position: 'absolute',
          top: 20,
          right: 20,
          background: 'rgba(0,0,0,0.8)',
          color: '#fff',
          padding: 12,
          borderRadius: 8,
          zIndex: 1001,
          minWidth: 220
        }}
      >
        <div style={{ fontWeight: 'bold', marginBottom: 8 }}>Named Particles</div>
        {namedParticles.map(p => (
          <div
            key={p.name}
            style={{ marginBottom: 6, display: 'flex', alignItems: 'center', gap: 8 }}
          >
            <span
              style={{
                width: 16,
                height: 16,
                background: p.color,
                borderRadius: '50%',
                display: 'inline-block',
                border: '1px solid #333'
              }}
            ></span>
            <span style={{ fontWeight: 'bold', color: p.color }}>{p.name}</span>
            <span style={{ fontSize: 12, color: '#aaa' }}>Scale: {p.scale}</span>
          </div>
        ))}
        <button
          style={{
            marginTop: 8,
            padding: '4px 10px',
            borderRadius: 4,
            background: '#222',
            color: '#fff',
            border: '1px solid #444',
            cursor: 'pointer',
            fontSize: 13
          }}
          onClick={() =>
            setNamedParticles(
              namedParticles.length
                ? []
                : [
                    { name: 'Ghost', priority: 1, color: '#e0e0e0', scale: 2.0 },
                    { name: 'Shadow', priority: 2, color: '#222222', scale: 1.8 },
                    { name: 'Mist', priority: 3, color: '#b0b0b0', scale: 1.6 },
                    { name: 'Specter', priority: 4, color: '#ffffff', scale: 1.7 }
                  ]
            )
          }
        >
          {namedParticles.length ? 'Hide Named Particles' : 'Show Named Particles'}
        </button>
      </div>
      {/* Loading state for Three.js modules */}
      {!threeLoaded && (
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: 1000,
            color: '#00ff88',
            fontFamily: 'monospace',
            textAlign: 'center',
            fontSize: '18px'
          }}
        >
          <div style={{ marginBottom: '10px' }}>Loading Enhanced Particle System...</div>
          <div style={{ fontSize: '12px', opacity: 0.7 }}>
            Preparing Three.js, WebGPU & WASM modules
          </div>
        </div>
      )}

      <canvas
        ref={canvasRef}
        style={{
          width: '100%',
          height: '100%',
          display: 'block',
          background: '#000',
          cursor: 'grab', // Visual feedback for interactions
          touchAction: 'none', // Prevent default touch behaviors that might interfere
          opacity: threeLoaded ? 1 : 0.3
        }}
        onPointerDown={() => {
          // Visual feedback when interacting
          if (canvasRef.current) {
            canvasRef.current.style.cursor = 'grabbing';
          }
        }}
        onPointerUp={() => {
          // Reset cursor when done interacting
          if (canvasRef.current) {
            canvasRef.current.style.cursor = 'grab';
          }
        }}
      />

      {/* Enhanced UI Controls */}
      <div
        style={{
          position: 'absolute',
          top: '20px',
          left: '20px',
          background: 'rgba(0, 0, 0, 0.9)',
          color: 'white',
          padding: '20px',
          borderRadius: '10px',
          fontFamily: 'monospace',
          fontSize: '14px',
          minWidth: '320px',
          backdropFilter: 'blur(10px)',
          border: '1px solid rgba(255, 255, 255, 0.1)'
        }}
      >
        <h3 style={{ margin: '0 0 15px 0', color: '#64ffda' }}>🌌 Enhanced Particle System</h3>

        <div style={{ marginBottom: '15px' }}>
          <strong>Animation Mode:</strong>
          <div style={{ display: 'flex', gap: '8px', marginTop: '8px', flexWrap: 'wrap' }}>
            {(['galaxy', 'yin-yang', 'wave', 'spiral'] as const).map(mode => (
              <button
                key={mode}
                onClick={() => switchAnimationMode(mode)}
                style={{
                  background:
                    metrics.animationMode === mode
                      ? 'rgba(100, 255, 218, 0.2)'
                      : 'rgba(255, 255, 255, 0.1)',
                  border: `1px solid ${metrics.animationMode === mode ? '#64ffda' : 'rgba(255, 255, 255, 0.3)'}`,
                  color: metrics.animationMode === mode ? '#64ffda' : '#fff',
                  padding: '6px 12px',
                  borderRadius: '6px',
                  fontSize: '12px',
                  cursor: 'pointer',
                  textTransform: 'capitalize'
                }}
              >
                {mode}
              </button>
            ))}
          </div>
        </div>

        <div
          style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', fontSize: '13px' }}
        >
          <div>
            FPS: <span style={{ color: '#4fc3f7' }}>{metrics.fps}</span>
            <span style={{ fontSize: '11px', color: '#888' }}>
              {' '}
              (Target: {framePerformanceRef.current.targetFPS})
            </span>
          </div>
          <div>
            Frame Time: <span style={{ color: '#81c784' }}>{metrics.frameTime.toFixed(1)}ms</span>
            <span style={{ fontSize: '11px', color: '#888' }}>
              {' '}
              (Budget: {framePerformanceRef.current.gpuBudget.toFixed(1)}ms)
            </span>
          </div>
          <div>
            Particles:{' '}
            <span
              style={{
                color:
                  metrics.particleCount >= 120000
                    ? '#ff0080' // Magenta for 120k+ particles (HIGH mode)
                    : metrics.particleCount >= 80000
                      ? '#ff1493' // Hot pink for 80k+ particles (enhanced mode)
                      : metrics.particleCount >= 60000
                        ? '#00e676' // Ultra green for 60k+
                        : metrics.particleCount >= 40000
                          ? '#4caf50' // High green for 40k+
                          : metrics.particleCount >= 20000
                            ? '#81c784' // Enhanced green for 20k+
                            : '#ffb74d', // Standard orange for <20k
                fontWeight: metrics.particleCount >= 80000 ? 'bold' : 'normal',
                textShadow: metrics.particleCount >= 120000 ? '0 0 10px #ff0080' : 'none'
              }}
            >
              {metrics.particleCount.toLocaleString()}
              {metrics.particleCount >= 120000
                ? ' 🔥💀⚡🚀🚀' // HIGH mode indicators
                : metrics.particleCount >= 80000
                  ? ' 🔥⚡🚀🚀' // Enhanced mode indicators
                  : metrics.particleCount >= 60000
                    ? ' ⚡🚀🚀'
                    : metrics.particleCount >= 40000
                      ? ' ⚡🚀'
                      : metrics.particleCount >= 20000
                        ? ' 🚀'
                        : ''}
            </span>
          </div>
          <div>
            Chunk: <span style={{ color: '#ff9800' }}>{chunkIndexRef.current}</span>
            <span style={{ fontSize: '11px', color: '#888' }}> (50k particles/chunk)</span>
          </div>
          <div>
            Render: <span style={{ color: '#ffb74d' }}>{metrics.renderMode}</span>
          </div>
          <div>
            Compute: <span style={{ color: '#e57373' }}>{metrics.computeMode}</span>
          </div>
          {metrics.wasmReady && metrics.workerMetrics && (
            <>
              <div>
                Workers:{' '}
                <span style={{ color: '#ab47bc' }}>
                  {metrics.workerMetrics.activeWorkers}/{metrics.workerMetrics.totalWorkers}
                </span>
              </div>
              <div>
                Queue: <span style={{ color: '#66bb6a' }}>{metrics.workerMetrics.queueDepth}</span>
              </div>
              <div>
                Throughput:{' '}
                <span style={{ color: '#42a5f5' }}>
                  {metrics.workerMetrics.throughput.toFixed(1)} tasks/s
                </span>
              </div>
              <div>
                Latency:{' '}
                <span style={{ color: '#ffa726' }}>
                  {metrics.workerMetrics.avgLatency.toFixed(1)}ms
                </span>
              </div>
            </>
          )}
        </div>

        <div
          style={{
            marginTop: '15px',
            paddingTop: '15px',
            borderTop: '1px solid rgba(255, 255, 255, 0.2)'
          }}
        >
          <div style={{ fontSize: '13px', marginBottom: '8px' }}>
            Media Streaming:
            <span
              style={{
                color: mediaStreaming?.connected ? '#4caf50' : '#ff9800',
                fontWeight: 'bold',
                marginLeft: '8px'
              }}
            >
              {mediaStreaming?.connected ? '● CONNECTED' : '○ DISCONNECTED'}
            </span>
          </div>
          <div style={{ fontSize: '11px', color: '#888' }}>
            Peer: {mediaStreaming?.peerId || 'N/A'} |{' '}
            {mediaStreaming?.error ? `Error: ${mediaStreaming.error}` : 'OK'}
          </div>
        </div>

        <div
          style={{
            marginTop: '15px',
            fontSize: '11px',
            color: '#666',
            fontStyle: 'italic'
          }}
        >
          🖱️ Left-click + drag to rotate • Right-click + drag to pan • Scroll to zoom
          <br />
          Press and hold to see cursor change • OrbitControls active
        </div>
      </div>
    </div>
  );
};

export default EnhancedParticleSystem;
