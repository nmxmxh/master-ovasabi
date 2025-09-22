/**
 * Enhanced Particle System - Cleaned and Refactored
 *
 * Key improvements:
 * - Simplified renderer initialization
 * - Fixed WebGL/WebGPU material compatibility
 * - Better error handling and fallbacks
 * - Cleaner component structure
 * - Proper resource cleanup
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';
import styled from 'styled-components';
import {
  loadAllThreeModules,
  type ThreeCore,
  type ThreeRenderers,
  type ThreeAddons
} from '../lib/three';
import {
  // useGPUCapabilities,
  // useEmitEvent,
  useConnectionStatus,
  useMediaStreamingState
} from '../store';
import { EnhancedComputeManager } from '../lib/compute/EnhancedComputeManager';
import { connectMediaStreamingToCampaign } from '../lib/wasmBridge';

// Simplified logger - use direct console calls to avoid React dev mode interference
const logger = {
  info: (msg: string, data?: any) => {
    if (data) {
      console.log(`[Particles] ${msg}`, data);
    } else {
      console.log(`[Particles] ${msg}`);
    }
  },
  error: (msg: string, error?: any) => {
    if (error) {
      console.error(`[Particles] ${msg}`, error);
    } else {
      console.error(`[Particles] ${msg}`);
    }
  },
  warn: (msg: string, data?: any) => {
    if (data) {
      console.log(`[Particles] WARN: ${msg}`, data);
    } else {
      console.log(`[Particles] WARN: ${msg}`);
    }
  }
};

interface ParticleMetrics {
  particleCount: number;
  fps: number;
  gpuUtilization: number;
  frameTime: number;
  computeMode: 'JS' | 'WASM' | 'WASM+WebGPU';
  renderMode: 'WebGL' | 'WebGPU';
  animationMode: 'galaxy' | 'yin-yang' | 'wave' | 'spiral';
  connectionStrength: number;
  wasmReady: boolean;
  webgpuReady: boolean;
}

// Renderer configuration
interface RendererConfig {
  type: 'webgl' | 'webgpu';
  antialias: boolean;
  alpha: boolean;
  powerPreference: 'default' | 'high-performance' | 'low-power';
}

// Theme configuration
const theme = {
  colors: {
    primary: '#64ffda',
    secondary: '#4fc3f7',
    success: '#4caf50',
    warning: '#ff9800',
    error: '#f44336',
    info: '#81c784',
    background: 'rgba(0, 0, 0, 0.9)',
    text: '#ffffff',
    textSecondary: '#e0e0e0',
    textMuted: '#666666',
    border: 'rgba(255, 255, 255, 0.1)',
    borderActive: 'rgba(255, 255, 255, 0.3)'
  },
  spacing: {
    xs: '4px',
    sm: '8px',
    md: '12px',
    lg: '15px',
    xl: '20px'
  },
  borderRadius: {
    sm: '4px',
    md: '6px',
    lg: '10px'
  },
  fontSizes: {
    xs: '11px',
    sm: '12px',
    md: '13px',
    lg: '14px',
    xl: '16px',
    xxl: '18px'
  },
  breakpoints: {
    mobile: '480px',
    tablet: '768px'
  }
};

// Styled Components
const ParticleSystemContainer = styled.div`
  position: relative;
  width: 100%;
  height: 100vh;
`;

const ParticleCanvas = styled.canvas`
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
  cursor: grab;
  transition: filter 0.3s ease;

  &:active {
    cursor: grabbing;
  }

  &:hover {
    filter: brightness(1.05);
  }
`;

const LoadingContainer = styled.div`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #00ff88;
  font-family: monospace;
  text-align: center;
  z-index: 1000;
  animation: pulse 2s ease-in-out infinite;

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.7;
    }
  }
`;

const LoadingTitle = styled.div`
  font-size: 18px;
  margin-bottom: 10px;
`;

const LoadingSubtitle = styled.div`
  font-size: 12px;
  opacity: 0.7;
`;

const ControlPanel = styled.div`
  position: absolute;
  top: ${theme.spacing.xl};
  left: ${theme.spacing.xl};
  background: ${theme.colors.background};
  color: ${theme.colors.text};
  padding: ${theme.spacing.xl};
  border-radius: ${theme.borderRadius.lg};
  font-family: monospace;
  font-size: ${theme.fontSizes.lg};
  min-width: 300px;
  backdrop-filter: blur(10px);
  border: 1px solid ${theme.colors.border};
  z-index: 1001;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;

  @media (max-width: ${theme.breakpoints.tablet}) {
    top: ${theme.spacing.md};
    left: ${theme.spacing.md};
    right: ${theme.spacing.md};
    min-width: auto;
    padding: ${theme.spacing.lg};
    font-size: ${theme.fontSizes.sm};
  }

  @media (max-width: ${theme.breakpoints.mobile}) {
    padding: ${theme.spacing.md};
    font-size: ${theme.fontSizes.xs};
  }
`;

const ControlTitle = styled.h3`
  margin: 0 0 ${theme.spacing.lg} 0;
  color: ${theme.colors.primary};
  font-size: ${theme.fontSizes.xl};
  font-weight: 600;
`;

const AnimationModeSection = styled.div`
  margin-bottom: ${theme.spacing.lg};
`;

const AnimationModeLabel = styled.strong`
  display: block;
  margin-bottom: ${theme.spacing.sm};
  color: ${theme.colors.textSecondary};
`;

const AnimationModeButtons = styled.div`
  display: flex;
  gap: ${theme.spacing.sm};
  margin-top: ${theme.spacing.sm};
  flex-wrap: wrap;
`;

const AnimationModeButton = styled.button<{ $active: boolean }>`
  background: ${props => (props.$active ? 'rgba(100, 255, 218, 0.2)' : 'rgba(255, 255, 255, 0.1)')};
  border: 1px solid ${props => (props.$active ? theme.colors.primary : theme.colors.borderActive)};
  color: ${props => (props.$active ? theme.colors.primary : theme.colors.text)};
  padding: ${theme.spacing.xs} ${theme.spacing.md};
  border-radius: ${theme.borderRadius.md};
  font-size: ${theme.fontSizes.sm};
  cursor: pointer;
  text-transform: capitalize;
  transition: all 0.2s ease;

  &:hover {
    background: ${props =>
      props.$active ? 'rgba(100, 255, 218, 0.3)' : 'rgba(255, 255, 255, 0.2)'};
    transform: translateY(-1px);
  }

  &:active {
    transform: translateY(0);
  }
`;

const MetricsGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: ${theme.spacing.md};
  font-size: ${theme.fontSizes.md};

  @media (max-width: ${theme.breakpoints.mobile}) {
    grid-template-columns: 1fr;
    gap: ${theme.spacing.sm};
    font-size: ${theme.fontSizes.sm};
  }
`;

const MetricItem = styled.div`
  display: flex;
  align-items: center;
  gap: ${theme.spacing.xs};
`;

const MetricValue = styled.span<{ $color: string }>`
  color: ${props => props.$color};
  font-weight: 500;
`;

const StatusSection = styled.div`
  margin-top: ${theme.spacing.lg};
  padding-top: ${theme.spacing.lg};
  border-top: 1px solid rgba(255, 255, 255, 0.2);
`;

const StatusItem = styled.div`
  font-size: ${theme.fontSizes.md};
  margin-bottom: ${theme.spacing.sm};
  display: flex;
  align-items: center;
  gap: ${theme.spacing.sm};
`;

const StatusIndicator = styled.span<{ $color: string }>`
  color: ${props => props.$color};
  font-weight: bold;
`;

const Instructions = styled.div`
  margin-top: ${theme.spacing.lg};
  font-size: ${theme.fontSizes.xs};
  color: ${theme.colors.textMuted};
  font-style: italic;
  line-height: 1.4;
`;

const EnhancedParticleSystem: React.FC = () => {
  // Core refs
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const sceneRef = useRef<any>(null);
  const rendererRef = useRef<any>(null);
  const cameraRef = useRef<any>(null);
  const controlsRef = useRef<any>(null);
  const particlesRef = useRef<any>(null);
  const animationIdRef = useRef<number | null>(null);
  const clockRef = useRef<any>(null);

  // State to track particle system readiness
  const [particlesReady, setParticlesReady] = useState(false);
  const computeManagerRef = useRef<EnhancedComputeManager | null>(null);

  // State
  const [threeLoaded, setThreeLoaded] = useState(false);
  const [threeModules, setThreeModules] = useState<{
    core: ThreeCore;
    renderers: ThreeRenderers;
    addons: ThreeAddons;
  } | null>(null);
  const [animationMode, setAnimationMode] = useState<'galaxy' | 'yin-yang' | 'wave' | 'spiral'>(
    'galaxy'
  );
  const [metrics, setMetrics] = useState<ParticleMetrics>({
    particleCount: 1000000, // 1 million particles for WebGPU
    fps: 60,
    gpuUtilization: 0,
    frameTime: 16.67,
    computeMode: 'JS',
    renderMode: 'WebGL',
    animationMode: 'galaxy',
    connectionStrength: 0,
    wasmReady: false,
    webgpuReady: false
  });

  // Global state hooks
  // const { isWebGPUAvailable, recommendedRenderer } = useGPUCapabilities();
  // const emitEvent = useEmitEvent();
  const { wasmReady } = useConnectionStatus();
  const { mediaStreaming } = useMediaStreamingState();

  // Check browser WebGPU flags and provide guidance
  const checkWebGPUBrowserFlags = useCallback(async () => {
    const userAgent = navigator.userAgent.toLowerCase();
    const isChrome = userAgent.includes('chrome') && !userAgent.includes('edge');
    const isEdge = userAgent.includes('edge');

    if (isChrome || isEdge) {
      logger.info('WebGPU Browser Flag Check:');
      logger.info('  Browser: ' + (isChrome ? 'Chrome' : 'Edge'));
      logger.info('  WebGPU Flag: chrome://flags/#enable-unsafe-webgpu');
      logger.info('  Status: ' + ('gpu' in navigator ? 'Available' : 'Not Available'));

      if ('gpu' in navigator) {
        // Test if WebGPU is actually functional
        try {
          const adapter = await (navigator as any).gpu.requestAdapter();
          if (adapter) {
            logger.info('  ✅ WebGPU adapter is functional');
            const features = Array.from(adapter.features);
            logger.info('  WebGPU features:', features.slice(0, 5)); // Show first 5 features
          } else {
            logger.warn('  ⚠️ WebGPU adapter not available - flag may not be properly enabled');
            logger.info('  Action Required: Enable WebGPU flag and restart browser');
            logger.info('  URL: chrome://flags/#enable-unsafe-webgpu');
          }
        } catch (error) {
          logger.warn('  ⚠️ WebGPU test failed:', error);
          logger.info('  Action Required: Enable WebGPU flag and restart browser');
          logger.info('  URL: chrome://flags/#enable-unsafe-webgpu');
        }
      } else {
        logger.info('  Action Required: Enable WebGPU flag and restart browser');
        logger.info('  URL: chrome://flags/#enable-unsafe-webgpu');
      }
    } else {
      logger.info('WebGPU Browser Flag Check:');
      logger.info('  Browser: ' + userAgent.split(' ')[0]);
      logger.info('  WebGPU Support: Limited or not available');
      logger.info('  Recommendation: Use Chrome or Edge with WebGPU enabled');
    }
  }, []);

  // Simplified WebGPU availability check
  const checkWebGPUAvailability = useCallback((): boolean => {
    const hasWebGPU = 'gpu' in navigator;
    const hasWebGPURenderer = !!threeModules?.renderers?.WebGPURenderer;

    logger.info('WebGPU availability check:', {
      browserWebGPU: hasWebGPU,
      threeWebGPURenderer: hasWebGPURenderer,
      available: hasWebGPU && hasWebGPURenderer
    });

    return hasWebGPU && hasWebGPURenderer;
  }, [threeModules]);

  // Check WASM WebGPU availability
  const checkWasmWebGPUAvailability = useCallback(async (): Promise<boolean> => {
    if (typeof (window as any).checkWebGPUAvailability !== 'function') {
      logger.warn('WASM checkWebGPUAvailability function not available');
      return false;
    }

    try {
      const status = (window as any).checkWebGPUAvailability();
      logger.info('WASM WebGPU status:', status);

      // Also check device validity if available
      if (
        status?.initialized === true &&
        typeof (window as any).checkWebGPUDeviceValidity === 'function'
      ) {
        const isValid = (window as any).checkWebGPUDeviceValidity();
        if (!isValid) {
          logger.warn('WASM WebGPU device is invalid');
          return false;
        }
      }

      return status?.initialized === true;
    } catch (error) {
      logger.error('Error checking WASM WebGPU availability:', error);
      return false;
    }
  }, []);

  // Three.js loading and compute manager initialization
  useEffect(() => {
    let mounted = true;

    const loadThreeJs = async () => {
      try {
        const modules = await loadAllThreeModules();
        if (mounted) {
          setThreeModules(modules);
          setThreeLoaded(true);
          clockRef.current = new modules.core.Clock();

          // Note: Compute manager initialization moved to separate useEffect
          // to ensure it happens after WASM is ready
        }
      } catch (error) {
        logger.error('Failed to load Three.js modules:', error);
      }
    };

    loadThreeJs();
    return () => {
      mounted = false;
      // Cleanup compute manager
      if (computeManagerRef.current) {
        computeManagerRef.current.destroy();
        computeManagerRef.current = null;
      }
    };
  }, []);

  // Initialize compute manager after WASM is ready
  useEffect(() => {
    if (!threeLoaded || computeManagerRef.current) {
      return; // Don't initialize if Three.js isn't loaded or compute manager already exists
    }

    logger.info('Starting compute manager initialization...');

    const initializeComputeManager = async () => {
      logger.info('Waiting for WASM functions to be available...');

      // Wait for WASM to be ready with longer timeout for better reliability
      await new Promise<void>(resolve => {
        let attempts = 0;
        const maxAttempts = 100; // 10 seconds instead of 5
        let isResolved = false; // Guard to prevent multiple resolutions

        const checkWasm = () => {
          if (isResolved) return; // Prevent multiple calls

          attempts++;
          const wasmStatus = {
            runConcurrentCompute: typeof (window as any).runConcurrentCompute === 'function',
            runGPUCompute: typeof (window as any).runGPUCompute === 'function',
            getGPUMetricsBuffer: typeof (window as any).getGPUMetricsBuffer === 'function'
          };

          // Log every 20 attempts to reduce spam and prevent stack overflow
          if (attempts % 20 === 0 || attempts <= 3) {
            console.log(
              `[Particles] Checking WASM functions (attempt ${attempts}/${maxAttempts}):`,
              wasmStatus
            );
          }

          if (
            wasmStatus.runConcurrentCompute &&
            wasmStatus.runGPUCompute &&
            wasmStatus.getGPUMetricsBuffer
          ) {
            console.log(
              '[Particles] ✅ WASM functions detected, proceeding with compute manager initialization'
            );
            isResolved = true;
            resolve();
          } else if (attempts >= maxAttempts) {
            console.warn(
              '[Particles] ⚠️ WASM functions not available after maximum attempts, initializing compute manager anyway'
            );
            isResolved = true;
            resolve();
          } else {
            // Use setTimeout to prevent stack overflow
            setTimeout(() => {
              if (!isResolved) {
                checkWasm();
              }
            }, 100);
          }
        };

        // Start the check
        checkWasm();
      });

      // Initialize centralized WebGPU system
      try {
        logger.info('🔄 Starting centralized WebGPU initialization...');
        const { webGPUManager } = await import('../lib/gpu/WebGPUManager');
        const gpuInitialized = await webGPUManager.initialize();
        logger.info('✅ Centralized WebGPU initialization result:', gpuInitialized);

        if (gpuInitialized) {
          // Subscribe to WebGPU status changes
          webGPUManager.subscribe(status => {
            if (!status.initialized && status.error) {
              logger.warn('WebGPU device lost or error:', status.error);
              handleContextLoss();
            }
          });
        }
      } catch (error) {
        logger.error('❌ Centralized WebGPU initialization failed:', error);
      }

      // Now initialize the compute manager
      logger.info('Creating EnhancedComputeManager instance...');
      computeManagerRef.current = new EnhancedComputeManager();
      logger.info('✅ Enhanced compute manager initialized after WASM ready');

      // Update metrics to reflect WASM readiness
      setMetrics(prev => ({ ...prev, wasmReady: true, computeMode: 'WASM' }));
    };

    initializeComputeManager().catch(error => {
      logger.error('Failed to initialize compute manager:', error);
    });
  }, [threeLoaded]);

  // Determine optimal renderer configuration - WebGPU preferred when available
  const determineRendererConfig = useCallback(async (): Promise<RendererConfig> => {
    // Check if WebGPU is available for Three.js rendering
    const webgpuAvailable = 'gpu' in navigator;
    const wasmReady =
      typeof (window as any).runConcurrentCompute === 'function' &&
      typeof (window as any).runGPUCompute === 'function' &&
      typeof (window as any).getGPUMetricsBuffer === 'function';

    // Check browser WebGPU flags first
    await checkWebGPUBrowserFlags();

    // Check if canvas already has a WebGL context (prevents WebGPU from working)
    const canvas = canvasRef.current;
    const hasExistingWebGLContext =
      canvas && (canvas.getContext('webgl2') || canvas.getContext('webgl'));

    // Prefer WebGL for main thread due to Three.js WebGPU compatibility issues
    // WebGPU compute is still available via WASM workers
    let useWebGPU = false;

    // Check if WASM WebGPU is available for compute operations
    const wasmWebGPUAvailable = await checkWasmWebGPUAvailability();
    logger.info('WASM WebGPU availability:', wasmWebGPUAvailable);

    // For now, disable WebGPU renderer in main thread due to Three.js compatibility issues
    // The compute worker can still use WebGPU for compute operations
    if (false && checkWebGPUAvailability()) {
      if (hasExistingWebGLContext) {
        logger.info('Canvas already has WebGL context, creating new canvas for WebGPU...');

        // Create a new canvas for WebGPU
        const newCanvas = document.createElement('canvas');

        // Copy all styling and dimensions from the old canvas
        newCanvas.style.width = canvas?.style.width || '100%';
        newCanvas.style.height = canvas?.style.height || '100%';
        newCanvas.style.display = canvas?.style.display || 'block';
        newCanvas.style.background = canvas?.style.background || '#000';
        newCanvas.style.cursor = canvas?.style.cursor || 'grab';

        // Set canvas dimensions to match the old canvas
        newCanvas.width = canvas?.width || canvas?.clientWidth || 800;
        newCanvas.height = canvas?.height || canvas?.clientHeight || 600;

        // Replace the old canvas with the new one
        const parent = canvas?.parentNode;
        if (parent && canvas) {
          parent!.replaceChild(newCanvas, canvas as Node);
          canvasRef.current = newCanvas;

          // Ensure the new canvas is properly sized
          newCanvas.width = newCanvas.clientWidth;
          newCanvas.height = newCanvas.clientHeight;

          // Small delay to ensure DOM has updated
          await new Promise(resolve => setTimeout(resolve, 10));

          logger.info('✅ New canvas created for WebGPU renderer', {
            width: newCanvas.width,
            height: newCanvas.height,
            clientWidth: newCanvas.clientWidth,
            clientHeight: newCanvas.clientHeight
          });
          useWebGPU = true;
        } else {
          logger.warn('Cannot replace canvas - no parent node found, using WebGL fallback');
          useWebGPU = false;
        }
      } else {
        // WebGPU is available and canvas is clean - use it!
        logger.info('✅ WebGPU available and canvas is clean - using WebGPU renderer');
        useWebGPU = true;
      }
    } else {
      logger.info(
        'WebGPU renderer disabled in main thread - using WebGL (WebGPU compute still available via WASM workers)'
      );
      useWebGPU = false;
    }

    logger.info('Renderer config decision:', {
      webgpuAvailable,
      wasmWebGPUAvailable,
      wasmReady,
      webgpuRendererAvailable: !!threeModules?.renderers?.WebGPURenderer,
      hasExistingWebGLContext,
      selectedRenderer: useWebGPU ? 'webgpu' : 'webgl',
      reason: useWebGPU
        ? 'WebGPU available and canvas clean - using WebGPU for rendering'
        : 'WebGPU renderer disabled in main thread - using WebGL (WebGPU compute available via WASM workers)',
      note: 'WASM WebGPU will be used separately for compute operations'
    });

    return {
      type: useWebGPU ? 'webgpu' : 'webgl',
      antialias: false,
      alpha: true,
      powerPreference: useWebGPU ? 'high-performance' : 'default'
    };
  }, [threeModules, checkWebGPUAvailability]);

  // Create renderer with proper error handling and WebGPU optimization
  const createRenderer = useCallback(
    async (canvas: HTMLCanvasElement, config: RendererConfig) => {
      const { core: THREE } = threeModules!;

      // Validate config
      if (!config) {
        logger.error('Config is null or undefined');
        throw new Error('Renderer config is required');
      }

      // Check if canvas is valid and has proper dimensions
      if (!canvas) {
        throw new Error('Canvas is null or undefined');
      }

      // For new canvases, ensure they have proper dimensions
      if (canvas.clientWidth === 0 || canvas.clientHeight === 0) {
        logger.warn('Canvas has zero dimensions, setting default size');

        // Try to get dimensions from parent container
        const parent = canvas.parentElement;
        const parentWidth = parent?.clientWidth || 800;
        const parentHeight = parent?.clientHeight || 600;

        // Set canvas dimensions
        canvas.width = parentWidth;
        canvas.height = parentHeight;
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.display = 'block';

        logger.info('Canvas dimensions set:', {
          width: canvas.width,
          height: canvas.height,
          clientWidth: canvas.clientWidth,
          clientHeight: canvas.clientHeight,
          parentWidth,
          parentHeight
        });

        // If still zero, wait a moment for the canvas to be properly sized
        if (canvas.clientWidth === 0 || canvas.clientHeight === 0) {
          logger.warn('Canvas still has zero dimensions after setting defaults');
          // Don't throw error, let the renderer creation attempt to handle it
        }
      }

      // Canvas replacement is now handled in determineRendererConfig
      // This ensures we have a clean canvas for WebGPU before reaching this point

      // Handle WebGL context checks for both WebGL and fallback cases
      if (config.type === 'webgl') {
        // For WebGL, check for existing context and handle context loss properly
        const existingContext = canvas.getContext('webgl2') || canvas.getContext('webgl');
        if (existingContext) {
          if (existingContext.isContextLost && existingContext.isContextLost()) {
            logger.warn('Existing WebGL context is lost, resetting canvas');
            // Reset the canvas to clear the lost context
            const parent = canvas.parentNode;
            if (parent) {
              const newCanvas = canvas.cloneNode(false) as HTMLCanvasElement;
              newCanvas.width = canvas.width;
              newCanvas.height = canvas.height;
              parent.replaceChild(newCanvas, canvas);
              // Update the canvas reference
              canvasRef.current = newCanvas;
              return createRenderer(newCanvas, config);
            }
          } else {
            logger.info('Existing WebGL context found, proceeding with renderer creation');
          }
        }
      }

      try {
        // Try WebGPU renderer if requested - with better error handling
        if (config.type === 'webgpu' && threeModules?.renderers?.WebGPURenderer) {
          logger.info('Creating WebGPU renderer...');
          try {
            // Test WebGPU availability first
            if (!('gpu' in navigator)) {
              throw new Error('WebGPU not available in navigator');
            }

            // Test WebGPU adapter
            const adapter = await (navigator as any).gpu.requestAdapter({
              powerPreference: 'high-performance'
            });
            if (!adapter) {
              throw new Error('No WebGPU adapter available');
            }

            // Check if we're in a worker context (where WebGPU works better)
            const isWorkerContext = typeof (globalThis as any).importScripts === 'function';
            if (isWorkerContext) {
              logger.info('WebGPU adapter available in worker context, creating renderer...');
            } else {
              logger.warn(
                'WebGPU adapter available in main thread, but Three.js WebGPU renderer is experimental'
              );
              // For now, prefer WebGL in main thread due to Three.js WebGPU compatibility issues
              throw new Error('Three.js WebGPU renderer has compatibility issues in main thread');
            }

            // Simple WebGPU renderer configuration
            const webgpuRenderer = new threeModules.renderers.WebGPURenderer({
              canvas,
              antialias: false,
              alpha: true,
              powerPreference: 'high-performance'
            });

            // Initialize WebGPU renderer
            await webgpuRenderer.init();

            // Check if the renderer actually fell back to WebGL
            const context = webgpuRenderer.getContext && webgpuRenderer.getContext();
            const contextType = context?.constructor?.name;

            logger.info('🔍 WebGPU renderer context check:', {
              contextType,
              isWebGL: contextType?.includes('WebGL'),
              isWebGPU: contextType?.includes('GPU'),
              hasContext: !!context
            });

            if (context && context.constructor.name.includes('WebGL')) {
              logger.warn('❌ WebGPU renderer fell back to WebGL:', {
                contextType,
                reason:
                  'Three.js WebGPU renderer fell back to WebGL2 - this is a known issue with experimental WebGPU support'
              });
              throw new Error(`WebGPU renderer fell back to ${context.constructor.name}`);
            }

            logger.info(
              '✅ WebGPU renderer created and initialized successfully with context:',
              contextType
            );
            return webgpuRenderer;
          } catch (webgpuError: any) {
            logger.warn('❌ WebGPU renderer creation failed, falling back to WebGL:', {
              error: webgpuError?.message || webgpuError,
              errorType: webgpuError?.constructor?.name || 'Unknown',
              reason: webgpuError?.message?.includes('WebGPU is not available')
                ? 'WebGPU is disabled in browser flags - enable chrome://flags/#enable-unsafe-webgpu'
                : webgpuError?.message?.includes('fell back to WebGL')
                  ? 'Three.js WebGPU renderer fell back to WebGL2 - this is a known issue with experimental WebGPU support'
                  : 'Three.js WebGPU renderer has compatibility issues'
            });

            // Update config to use WebGL fallback
            config.type = 'webgl';

            // Update metrics to reflect WebGL fallback
            setMetrics(prev => ({
              ...prev,
              renderMode: 'WebGL',
              webgpuReady: false,
              particleCount: Math.min(prev.particleCount, 100000) // Ensure WebGL particle limit
            }));
          }
        }

        // Fallback to WebGL
        logger.info('Creating WebGL renderer...');

        // Test WebGL context creation first with proper error handling
        const testCanvas = document.createElement('canvas');
        let testContext = null;

        try {
          testContext = testCanvas.getContext('webgl2') || testCanvas.getContext('webgl');
        } catch (error) {
          logger.warn('WebGL context test failed:', error);
        }

        if (!testContext) {
          throw new Error('WebGL is not supported in this browser');
        }

        // Skip precision test as it's causing issues with context loss
        logger.info('WebGL context test passed, skipping precision test to avoid context loss');

        // Set appropriate particle count based on renderer type
        if (config.type === 'webgl') {
          // WebGL can handle up to 50,000 particles efficiently
          const webglParticleCount = Math.min(metrics.particleCount, 50000);
          if (webglParticleCount !== metrics.particleCount) {
            logger.info(
              `Setting particle count to ${webglParticleCount} for WebGL compatibility (requested: ${metrics.particleCount})`
            );
            setMetrics(prev => ({ ...prev, particleCount: webglParticleCount }));
          }
        } else if (config.type === 'webgpu') {
          // WebGPU can handle up to 1,000,000 particles efficiently
          const webgpuParticleCount = Math.min(metrics.particleCount, 1000000);
          if (webgpuParticleCount !== metrics.particleCount) {
            logger.info(
              `Setting particle count to ${webgpuParticleCount} for WebGPU (requested: ${metrics.particleCount})`
            );
            setMetrics(prev => ({ ...prev, particleCount: webgpuParticleCount }));
          } else {
            logger.info(`Using full particle count ${metrics.particleCount} for WebGPU`);
          }
        }

        // Create WebGL renderer with minimal options to avoid context issues
        const rendererOptions: any = {
          canvas,
          antialias: false, // Disable for better performance
          alpha: true, // Always use alpha for particles
          powerPreference: 'default',
          failIfMajorPerformanceCaveat: false,
          preserveDrawingBuffer: false,
          premultipliedAlpha: false,
          stencil: false,
          depth: true
        };

        // Don't pass existing context to avoid precision errors
        // Let Three.js create a fresh context
        const renderer = new THREE.WebGLRenderer(rendererOptions);

        // Verify renderer was created successfully
        if (!renderer.domElement) {
          throw new Error('Failed to create WebGL renderer');
        }

        return renderer;
      } catch (error) {
        logger.error('Renderer creation failed:', error);

        // Final fallback to basic WebGL with minimal options
        try {
          const fallbackOptions: any = {
            canvas,
            antialias: false,
            alpha: true,
            powerPreference: 'default',
            failIfMajorPerformanceCaveat: false,
            preserveDrawingBuffer: false,
            premultipliedAlpha: false,
            stencil: false,
            depth: true
          };

          // Don't pass context to fallback to avoid conflicts
          return new THREE.WebGLRenderer(fallbackOptions);
        } catch (fallbackError) {
          logger.error('Final WebGL fallback failed:', fallbackError);

          // Last resort: try with a completely new canvas
          try {
            logger.warn('Attempting to create renderer with new canvas...');
            const newCanvas = document.createElement('canvas');
            newCanvas.width = canvas.width;
            newCanvas.height = canvas.height;
            newCanvas.style.width = '100%';
            newCanvas.style.height = '100%';
            newCanvas.style.display = 'block';
            newCanvas.style.background = '#000';

            const lastResortRenderer = new THREE.WebGLRenderer({
              canvas: newCanvas,
              antialias: false,
              alpha: true,
              powerPreference: 'default',
              failIfMajorPerformanceCaveat: false,
              preserveDrawingBuffer: false,
              premultipliedAlpha: false,
              stencil: false,
              depth: true
            });

            // Replace the old canvas with the new one
            const parent = canvas.parentNode;
            if (parent) {
              parent.replaceChild(newCanvas, canvas);
              canvasRef.current = newCanvas;
            }

            logger.info('Successfully created renderer with new canvas');
            return lastResortRenderer;
          } catch (lastResortError) {
            logger.error('Last resort renderer creation failed:', lastResortError);
            throw new Error('Unable to create any renderer');
          }
        }
      }
    },
    [threeModules]
  );

  // Create particle material with proper WebGL/WebGPU compatibility
  const createParticleMaterial = useCallback((THREE: any, rendererType: string = 'webgl') => {
    // Create simple particle texture
    const canvas = document.createElement('canvas');
    canvas.width = 64;
    canvas.height = 64;
    const context = canvas.getContext('2d');

    if (context) {
      const gradient = context.createRadialGradient(32, 32, 0, 32, 32, 32);
      gradient.addColorStop(0, 'rgba(255, 255, 255, 1)');
      gradient.addColorStop(0.5, 'rgba(255, 255, 255, 0.5)');
      gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
      context.fillStyle = gradient;
      context.fillRect(0, 0, 64, 64);
    }

    const texture = new THREE.CanvasTexture(canvas);
    texture.generateMipmaps = false;
    texture.flipY = false;

    // Use PointsMaterial optimized for particles with WebGL/WebGPU compatibility
    const material = new THREE.PointsMaterial({
      size: 10.0, // Much larger size for debugging visibility
      map: texture,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      vertexColors: true,
      sizeAttenuation: true, // Enable for better depth perception
      opacity: 1.0, // Fully opaque for maximum visibility
      // Additional optimizations
      fog: false
      // Note: 'lights' is not a valid property for PointsMaterial
    });

    // Log material creation for debugging
    logger.info('✅ Particle material created:', {
      rendererType,
      materialType: material.type,
      size: material.size,
      transparent: material.transparent,
      blending: material.blending,
      hasTexture: !!material.map
    });

    return material;
  }, []);

  // Generate particle pattern optimized for 1M particles with memory management
  const generateParticlePattern = useCallback(async (mode: string, count: number) => {
    logger.info(`🎨 Starting particle generation: mode=${mode}, count=${count}`);

    // Check available memory and limit particle count if necessary
    const maxParticles = Math.min(count, 500000); // Limit to 500K particles for memory safety
    if (maxParticles !== count) {
      logger.warn(`⚠️ Limiting particle count from ${count} to ${maxParticles} for memory safety`);
    }

    // Use chunked allocation to avoid memory allocation failures
    let positions: Float32Array;
    let colors: Float32Array;

    try {
      positions = new Float32Array(maxParticles * 3);
      colors = new Float32Array(maxParticles * 3);
    } catch (error) {
      logger.error('❌ Particle generation failed:', error);
      // Fallback to much smaller count
      const fallbackCount = 10000;
      logger.warn(`🔄 Falling back to ${fallbackCount} particles due to memory constraints`);
      positions = new Float32Array(fallbackCount * 3);
      colors = new Float32Array(fallbackCount * 3);
      return { positions, colors, count: fallbackCount };
    }

    // Use batch processing for better performance with large particle counts
    const batchSize = 10000;
    const batches = Math.ceil(maxParticles / batchSize);

    logger.info(`📊 Processing ${batches} batches of ${batchSize} particles each`);

    for (let batch = 0; batch < batches; batch++) {
      const startIndex = batch * batchSize;
      const endIndex = Math.min(startIndex + batchSize, maxParticles);

      // Log progress every 10 batches
      if (batch % 10 === 0) {
        logger.info(`🔄 Processing batch ${batch + 1}/${batches} (${startIndex}-${endIndex})`);
      }

      for (let i = startIndex; i < endIndex; i++) {
        const i3 = i * 3;
        let x = 0,
          y = 0,
          z = 0;

        switch (mode) {
          case 'galaxy':
            const radius = Math.random() * 15; // Larger galaxy for 1M particles
            const angle = Math.random() * Math.PI * 2;
            x = Math.cos(angle) * radius;
            y = (Math.random() - 0.5) * 3;
            z = Math.sin(angle) * radius;
            break;
          case 'yin-yang':
            const t = i / count;
            const spiral = t * Math.PI * 6; // More complex spiral
            x = Math.cos(spiral) * (3 + t * 5);
            y = Math.sin(spiral * 3) * 3;
            z = Math.sin(spiral) * (3 + t * 5);
            break;
          case 'wave':
            x = (Math.random() - 0.5) * 30; // Larger wave field
            y = Math.sin(x * 0.3) * 8;
            z = (Math.random() - 0.5) * 30;
            break;
          case 'spiral':
            const spiralT = i / count;
            const spiralAngle = spiralT * Math.PI * 8; // More complex spiral
            const spiralRadius = spiralT * 12;
            x = Math.cos(spiralAngle) * spiralRadius;
            y = (spiralT - 0.5) * 15;
            z = Math.sin(spiralAngle) * spiralRadius;
            break;
        }

        positions[i3] = x;
        positions[i3 + 1] = y;
        positions[i3 + 2] = z;

        // Optimized color generation
        const normalizedX = (x + 15) / 30; // Normalize to 0-1
        const normalizedY = (y + 8) / 16;
        const normalizedZ = (z + 15) / 30;

        colors[i3] = 0.3 + normalizedX * 0.7;
        colors[i3 + 1] = 0.3 + normalizedY * 0.7;
        colors[i3 + 2] = 0.3 + normalizedZ * 0.7;
      }

      // Yield control to prevent blocking the main thread
      if (batch % 10 === 0) {
        // Allow other tasks to run
        await new Promise(resolve => setTimeout(resolve, 0));
      }
    }

    logger.info(`✅ Particle generation complete: ${maxParticles} particles generated`);
    return { positions, colors, count: maxParticles };
  }, []);

  // Debounced context loss recovery to prevent rapid reinitializations
  const contextLossTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const handleContextLoss = useCallback(() => {
    // Debounce context loss handling to prevent rapid reinitializations
    if (contextLossTimeoutRef.current) {
      clearTimeout(contextLossTimeoutRef.current);
    }

    contextLossTimeoutRef.current = setTimeout(() => {
      logger.warn('WebGL context lost, attempting recovery...');

      // Stop current animation
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
        animationIdRef.current = null;
      }

      // Properly dispose of renderer resources
      if (rendererRef.current) {
        try {
          rendererRef.current.dispose();
          rendererRef.current.forceContextLoss();
        } catch (error) {
          logger.error('Error disposing renderer:', error);
        }
        rendererRef.current = null;
      }

      // Clear all references
      if (particlesRef.current) {
        particlesRef.current = null;
      }
      if (controlsRef.current) {
        controlsRef.current = null;
      }

      // Reset particles ready state
      setParticlesReady(false);

      // Reset the canvas completely
      if (canvasRef.current) {
        const canvas = canvasRef.current;
        const parent = canvas.parentNode;
        if (parent) {
          // Create a completely new canvas
          const newCanvas = document.createElement('canvas');
          newCanvas.width = canvas.width;
          newCanvas.height = canvas.height;
          newCanvas.style.width = canvas.style.width;
          newCanvas.style.height = canvas.style.height;
          newCanvas.style.display = canvas.style.display;
          newCanvas.style.background = canvas.style.background;
          newCanvas.style.cursor = canvas.style.cursor;

          // Replace the old canvas
          parent.replaceChild(newCanvas, canvas);
          canvasRef.current = newCanvas;

          logger.info('Canvas reset complete, will reinitialize...');
        }
      }

      // Attempt to reinitialize after a delay
      setTimeout(() => {
        if (canvasRef.current && threeLoaded && threeModules) {
          logger.info('Attempting to reinitialize after context loss...');
          // Trigger re-initialization by clearing the scene reference
          sceneRef.current = null;
          // The useEffect will detect this and reinitialize
        }
      }, 3000); // Increased delay to prevent rapid reinitialization
    }, 1000); // Debounce delay
  }, [threeLoaded, threeModules]);

  // Initialize scene - prevent multiple initializations
  useEffect(() => {
    if (!canvasRef.current || !threeLoaded || !threeModules) return;

    // Prevent multiple initializations
    if (sceneRef.current || rendererRef.current) {
      logger.warn('Scene already initialized, skipping...');
      return;
    }

    // Check if canvas is properly sized
    if (canvasRef.current.clientWidth === 0 || canvasRef.current.clientHeight === 0) {
      logger.warn('Canvas not ready, waiting for proper dimensions...');
      return;
    }

    // Validate canvas state
    const canvasElement = canvasRef.current;
    const existingContext = canvasElement.getContext('webgl2') || canvasElement.getContext('webgl');
    if (existingContext && existingContext.isContextLost && existingContext.isContextLost()) {
      logger.warn('Canvas has lost context, resetting...');
      handleContextLoss();
      return;
    }

    const { core: THREE, addons } = threeModules;

    const initializeScene = async () => {
      try {
        logger.info('🎬 Starting scene initialization...');

        // Wait for canvas to be ready and ensure it's properly set up
        await new Promise<void>(resolve => {
          let attempts = 0;
          const maxAttempts = 50; // 5 seconds max

          const checkSize = () => {
            attempts++;
            logger.info(`Checking canvas dimensions (attempt ${attempts}/${maxAttempts}):`, {
              clientWidth: canvasElement.clientWidth,
              clientHeight: canvasElement.clientHeight,
              width: canvasElement.width,
              height: canvasElement.height,
              offsetWidth: canvasElement.offsetWidth,
              offsetHeight: canvasElement.offsetHeight,
              parentWidth: canvasElement.parentElement?.clientWidth,
              parentHeight: canvasElement.parentElement?.clientHeight
            });

            // Force canvas sizing if it's zero
            if (canvasElement.clientWidth === 0 || canvasElement.clientHeight === 0) {
              // Set default dimensions if canvas is not sized
              const defaultWidth = 800;
              const defaultHeight = 600;

              canvasElement.width = defaultWidth;
              canvasElement.height = defaultHeight;
              canvasElement.style.width = '100%';
              canvasElement.style.height = '100%';
              canvasElement.style.display = 'block';
              canvasElement.style.background = '#000';
              canvasElement.style.cursor = 'grab';

              logger.warn(
                `Canvas had zero dimensions, set to defaults: ${defaultWidth}x${defaultHeight}`
              );
            }

            if (canvasElement.width > 0 && canvasElement.height > 0) {
              logger.info('✅ Canvas ready for rendering:', {
                finalWidth: canvasElement.width,
                finalHeight: canvasElement.height,
                finalClientWidth: canvasElement.clientWidth,
                finalClientHeight: canvasElement.clientHeight
              });

              resolve();
            } else if (attempts >= maxAttempts) {
              logger.error(
                '❌ Canvas sizing failed after maximum attempts, using fallback dimensions'
              );
              // Force fallback dimensions
              canvasElement.width = 800;
              canvasElement.height = 600;
              resolve();
            } else {
              logger.warn('Canvas not ready, retrying in 100ms...');
              setTimeout(checkSize, 100);
            }
          };
          checkSize();
        });

        // Create scene
        logger.info('Creating Three.js scene...');
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x000000);
        logger.info('✅ Scene created');

        // Create camera
        logger.info('Creating camera...');
        const camera = new THREE.PerspectiveCamera(
          75,
          canvasElement.clientWidth / canvasElement.clientHeight,
          0.1,
          1000
        );
        camera.position.set(0, 0, 30); // Move camera back to see particles better
        camera.lookAt(0, 0, 0); // Ensure camera is looking at origin where particles are

        // Test camera setup
        logger.info('✅ Camera created and positioned at (0, 0, 30) looking at (0, 0, 0)');
        logger.info('🔍 Camera setup test:', {
          position: [camera.position.x, camera.position.y, camera.position.z],
          target: [0, 0, 0],
          fov: camera.fov,
          aspect: camera.aspect,
          near: camera.near,
          far: camera.far
        });

        // Create renderer
        logger.info('Creating renderer...');
        let renderer, config;
        try {
          config = await determineRendererConfig();
          logger.info('Renderer config:', config);
          renderer = await createRenderer(canvasElement, config);
          renderer.setSize(canvasElement.clientWidth, canvasElement.clientHeight);
          renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
          logger.info('✅ Renderer created and configured');
        } catch (rendererError) {
          logger.error('❌ Renderer creation failed:', rendererError);
          throw rendererError;
        }

        // Create controls
        logger.info('Creating orbit controls...');
        const controls = new addons.OrbitControls(camera, canvasElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.05;
        controls.enableZoom = true;
        controls.enableRotate = true;
        controls.enablePan = true;
        logger.info('✅ Controls created');

        // Generate particles
        logger.info('Generating particle pattern...');
        let positions, colors;
        try {
          // Add timeout to prevent hanging
          const result = (await Promise.race([
            generateParticlePattern(animationMode, metrics.particleCount),
            new Promise((_, reject) =>
              setTimeout(
                () => reject(new Error('Particle generation timeout after 10 seconds')),
                10000
              )
            )
          ])) as { positions: Float32Array; colors: Float32Array; count: number };
          positions = result.positions;
          colors = result.colors;

          // Update metrics with actual particle count if different
          if (result.count !== metrics.particleCount) {
            setMetrics(prev => ({ ...prev, particleCount: result.count }));
          }

          logger.info('✅ Particle pattern generated');
        } catch (particleError) {
          logger.error('❌ Particle generation failed:', particleError);

          // Fallback: create simple test particles
          logger.info('🔄 Creating fallback test particles...');
          const fallbackCount = 1000; // Much smaller for testing
          positions = new Float32Array(fallbackCount * 3);
          colors = new Float32Array(fallbackCount * 3);

          for (let i = 0; i < fallbackCount; i++) {
            const i3 = i * 3;
            positions[i3] = (Math.random() - 0.5) * 20; // x
            positions[i3 + 1] = (Math.random() - 0.5) * 20; // y
            positions[i3 + 2] = (Math.random() - 0.5) * 20; // z

            colors[i3] = Math.random(); // r
            colors[i3 + 1] = Math.random(); // g
            colors[i3 + 2] = Math.random(); // b
          }

          logger.info(`✅ Fallback particles created: ${fallbackCount} particles`);
        }

        // Debug: Add a test particle at origin for visibility testing
        if (positions.length >= 3) {
          positions[0] = 0; // x
          positions[1] = 0; // y
          positions[2] = 0; // z
          colors[0] = 1.0; // r
          colors[1] = 0.0; // g
          colors[2] = 0.0; // b

          // Add a few more test particles in different positions
          if (positions.length >= 12) {
            // Particle 1: Red at origin
            positions[0] = 0;
            positions[1] = 0;
            positions[2] = 0;
            colors[0] = 1.0;
            colors[1] = 0.0;
            colors[2] = 0.0;

            // Particle 2: Green at (5, 0, 0)
            positions[3] = 5;
            positions[4] = 0;
            positions[5] = 0;
            colors[3] = 0.0;
            colors[4] = 1.0;
            colors[5] = 0.0;

            // Particle 3: Blue at (0, 5, 0)
            positions[6] = 0;
            positions[7] = 5;
            positions[8] = 0;
            colors[6] = 0.0;
            colors[7] = 0.0;
            colors[8] = 1.0;

            // Particle 4: White at (0, 0, 5)
            positions[9] = 0;
            positions[10] = 0;
            positions[11] = 5;
            colors[9] = 1.0;
            colors[10] = 1.0;
            colors[11] = 1.0;
          }
        }

        // Create geometry
        logger.info('Creating particle geometry...');
        let geometry;
        try {
          geometry = new THREE.BufferGeometry();
          geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
          geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
          logger.info('✅ Geometry created');
        } catch (geometryError) {
          logger.error('❌ Geometry creation failed:', geometryError);
          throw geometryError;
        }

        // Create material
        logger.info('Creating particle material...');
        let material;
        try {
          // Pass renderer type for compatibility
          const rendererType = rendererRef.current?.constructor?.name?.toLowerCase() || 'webgl';
          material = createParticleMaterial(THREE, rendererType);
          logger.info('✅ Material created');
        } catch (materialError) {
          logger.error('❌ Material creation failed:', materialError);
          throw materialError;
        }

        // Create particles
        logger.info('Creating particle system...');
        let particles;
        try {
          particles = new THREE.Points(geometry, material);
          scene.add(particles);
          logger.info('✅ Particles added to scene');

          // Mark particles as ready for updates
          setParticlesReady(true);
        } catch (particleError) {
          logger.error('❌ Particle system creation failed:', particleError);
          throw particleError;
        }

        // Debug: Log particle creation
        logger.info('Particles created:', {
          particleCount: positions.length / 3,
          geometryVertices: geometry.attributes.position.count,
          materialSize: material.size,
          sceneChildren: scene.children.length,
          firstParticlePosition: [positions[0], positions[1], positions[2]],
          cameraPosition: [camera.position.x, camera.position.y, camera.position.z],
          rendererSize: [renderer.domElement.width, renderer.domElement.height],
          testParticles: {
            red: [positions[0], positions[1], positions[2]],
            green: [positions[3], positions[4], positions[5]],
            blue: [positions[6], positions[7], positions[8]],
            white: [positions[9], positions[10], positions[11]]
          },
          rendererType: renderer.constructor.name,
          rendererContext: renderer.getContext?.()?.constructor?.name || 'unknown'
        });

        // Store references
        logger.info('Storing scene references...');
        sceneRef.current = scene;
        rendererRef.current = renderer;
        cameraRef.current = camera;
        controlsRef.current = controls;
        particlesRef.current = particles;
        logger.info('✅ All references stored successfully');

        // Add context loss event listeners
        canvasElement.addEventListener('webglcontextlost', event => {
          logger.warn('WebGL context lost event received');
          event.preventDefault();
          handleContextLoss();
        });

        canvasElement.addEventListener('webglcontextrestored', () => {
          logger.info('WebGL context restored event received');
          // Context will be restored automatically
        });

        // Update metrics
        logger.info('Updating metrics...');
        setMetrics(prev => ({
          ...prev,
          renderMode: config.type === 'webgpu' ? 'WebGPU' : 'WebGL',
          webgpuReady: config.type === 'webgpu',
          wasmReady: wasmReady,
          computeMode: wasmReady ? (config.type === 'webgpu' ? 'WASM+WebGPU' : 'WASM') : 'JS'
        }));
        logger.info('✅ Metrics updated');

        logger.info('🎉 Scene initialized successfully - all refs should now be available!');

        // Test particle visibility immediately
        setTimeout(() => {
          if (particlesRef.current && particlesRef.current.geometry && sceneRef.current) {
            const particleCount = particlesRef.current.geometry.attributes.position.count;
            const firstPosition = particlesRef.current.geometry.attributes.position.array.slice(
              0,
              3
            );
            const materialSize = particlesRef.current.material.size;

            logger.info('🔍 Particle visibility test:', {
              particleCount,
              firstPosition,
              materialSize,
              sceneChildren: sceneRef.current.children.length,
              cameraPosition: [camera.position.x, camera.position.y, camera.position.z],
              cameraTarget: [0, 0, 0],
              rendererSize: [renderer.domElement.width, renderer.domElement.height]
            });

            // Test if particles are in camera view
            const distance = Math.sqrt(
              firstPosition[0] ** 2 + firstPosition[1] ** 2 + firstPosition[2] ** 2
            );
            const cameraDistance = Math.sqrt(
              camera.position.x ** 2 + camera.position.y ** 2 + camera.position.z ** 2
            );

            logger.info('🔍 Visibility analysis:', {
              particleDistance: distance,
              cameraDistance: cameraDistance,
              particlesInView: distance < cameraDistance + 20,
              materialVisible: materialSize > 0
            });

            // Force a test render to verify particles are visible
            if (rendererRef.current && sceneRef.current && cameraRef.current) {
              try {
                rendererRef.current.render(sceneRef.current, cameraRef.current);
                logger.info('🎨 Test render completed successfully');
              } catch (renderError) {
                logger.error('❌ Test render failed:', renderError);
              }
            }
          }
        }, 200);

        // Connect to campaign
        try {
          await connectMediaStreamingToCampaign('0', 'enhanced-particles');
          logger.info('Connected to campaign');
        } catch (error) {
          logger.warn('Failed to connect to campaign:', error);
        }
      } catch (error) {
        logger.error('Scene initialization failed:', error);
      }
    };

    initializeScene();

    return () => {
      // Cleanup with proper order to prevent context loss
      try {
        // Stop animation first
        if (animationIdRef.current) {
          cancelAnimationFrame(animationIdRef.current);
          animationIdRef.current = null;
        }

        // Dispose controls
        if (controlsRef.current) {
          try {
            controlsRef.current.dispose();
          } catch (error) {
            logger.warn('Error disposing controls:', error);
          }
          controlsRef.current = null;
        }

        // Dispose particles and geometry
        if (particlesRef.current) {
          try {
            if (sceneRef.current) {
              sceneRef.current.remove(particlesRef.current);
            }
            if (particlesRef.current.geometry) {
              particlesRef.current.geometry.dispose();
            }
            if (particlesRef.current.material) {
              particlesRef.current.material.dispose();
            }
          } catch (error) {
            logger.warn('Error disposing particles:', error);
          }
          particlesRef.current = null;
        }

        // Dispose renderer last to prevent context loss
        if (rendererRef.current) {
          try {
            // Clear the renderer before disposing
            rendererRef.current.clear();
            rendererRef.current.dispose();
          } catch (error) {
            logger.warn('Error disposing renderer:', error);
          }
          rendererRef.current = null;
        }

        // Clear other references
        if (sceneRef.current) {
          sceneRef.current = null;
        }
        if (cameraRef.current) {
          cameraRef.current = null;
        }
      } catch (error) {
        logger.error('Error during cleanup:', error);
      }
    };
  }, [
    threeLoaded,
    threeModules,
    // Remove dependencies that cause re-initialization
    // animationMode,
    // metrics.particleCount,
    // determineRendererConfig, // Removed to prevent infinite loops
    createRenderer,
    createParticleMaterial,
    generateParticlePattern,
    wasmReady,
    handleContextLoss
  ]);

  // Handle animation mode changes without re-initializing scene
  useEffect(() => {
    if (!particlesRef.current || !threeModules || !particlesReady) return;

    // Regenerate particle pattern for new animation mode
    const updateParticles = async () => {
      try {
        // Validate that particles and geometry exist before updating
        if (!particlesRef.current || !particlesRef.current.geometry) {
          logger.warn('Particles or geometry not ready, skipping update');
          return;
        }

        const {
          positions,
          colors,
          count: actualCount
        } = await generateParticlePattern(animationMode, metrics.particleCount);

        // Update geometry attributes with additional validation
        const geometry = particlesRef.current.geometry;
        if (!geometry) {
          logger.warn('Geometry is null, cannot update particle pattern');
          return;
        }

        const { core: THREE } = threeModules;
        geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
        geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

        // Update metrics with actual particle count
        if (actualCount !== metrics.particleCount) {
          setMetrics(prev => ({ ...prev, particleCount: actualCount }));
        }

        logger.info(`Animation mode changed to: ${animationMode}`);
      } catch (error) {
        logger.error('Failed to update particle pattern:', error);
      }
    };

    updateParticles();
  }, [animationMode, generateParticlePattern, metrics.particleCount, threeModules, particlesReady]);

  // CPU fallback particle animation
  const updateParticlesCPU = useCallback(
    (positions: Float32Array, elapsedTime: number, mode: string) => {
      // Debug: Log CPU fallback usage (log every 60 frames to reduce spam)
      if (Math.random() < 0.016) {
        // ~1% chance = ~1 log per second at 60fps
        logger.info('Using CPU fallback for particle animation:', {
          particleCount: positions.length / 3,
          elapsedTime,
          mode,
          firstParticlePosition: [positions[0], positions[1], positions[2]]
        });
      }

      for (let i = 0; i < positions.length; i += 3) {
        switch (mode) {
          case 'galaxy':
            const radius = Math.sqrt(positions[i] ** 2 + positions[i + 2] ** 2);
            if (radius > 0.001) {
              const angle = Math.atan2(positions[i + 2], positions[i]) + elapsedTime * 0.5;
              positions[i] = radius * Math.cos(angle);
              positions[i + 2] = radius * Math.sin(angle);
            }
            break;
          case 'yin-yang':
            const dist = Math.sqrt(positions[i] ** 2 + positions[i + 2] ** 2);
            if (dist > 0.001) {
              const flowAngle = Math.atan2(positions[i + 2], positions[i]) + elapsedTime * 2.0;
              positions[i] = dist * Math.cos(flowAngle);
              positions[i + 2] = dist * Math.sin(flowAngle);
            }
            break;
          case 'wave':
            positions[i + 1] = Math.sin(positions[i] * 0.5 + elapsedTime * 3) * 3;
            break;
          case 'spiral':
            const spiralRadius = Math.sqrt(positions[i] ** 2 + positions[i + 2] ** 2);
            if (spiralRadius > 0.001) {
              const spiralAngle = Math.atan2(positions[i + 2], positions[i]) + elapsedTime * 1.5;
              positions[i] = spiralRadius * Math.cos(spiralAngle);
              positions[i + 2] = spiralRadius * Math.sin(spiralAngle);
            }
            break;
        }
      }

      if (particlesRef.current && particlesRef.current.geometry) {
        particlesRef.current.geometry.attributes.position.needsUpdate = true;
      }
    },
    []
  );

  // Animation loop
  useEffect(() => {
    // Add debug log to show when this useEffect is triggered
    logger.info('🎬 Animation loop useEffect triggered with refs:', {
      scene: !!sceneRef.current,
      renderer: !!rendererRef.current,
      camera: !!cameraRef.current,
      particles: !!particlesRef.current
    });

    if (
      !sceneRef.current ||
      !rendererRef.current ||
      !cameraRef.current ||
      !particlesRef.current ||
      !particlesReady
    ) {
      logger.warn('Animation loop not started - missing required refs:', {
        scene: !!sceneRef.current,
        renderer: !!rendererRef.current,
        camera: !!cameraRef.current,
        particles: !!particlesRef.current,
        particlesReady
      });
      return;
    }

    // Add immediate debug log to confirm animation loop useEffect is running
    logger.info('🎬 Animation loop useEffect triggered - starting animation loop');

    logger.info('🎬 Starting animation loop with particles:', {
      particleCount: particlesRef.current?.geometry?.attributes?.position?.count || 0,
      materialSize: particlesRef.current?.material?.size || 0,
      cameraPosition: [
        cameraRef.current.position.x,
        cameraRef.current.position.y,
        cameraRef.current.position.z
      ]
    });

    // Add immediate debug log to confirm animation loop started
    logger.info('🎬 Animation loop started - first frame should render soon');

    // Force immediate first frame to test
    setTimeout(() => {
      if (sceneRef.current && rendererRef.current && cameraRef.current && particlesRef.current) {
        logger.info('🎬 Testing immediate render - all refs available');
        try {
          rendererRef.current.render(sceneRef.current, cameraRef.current);
          logger.info('🎬 Immediate render test successful');
        } catch (error) {
          logger.error('🎬 Immediate render test failed:', error);
        }
      } else {
        logger.warn('🎬 Immediate render test skipped - missing refs:', {
          scene: !!sceneRef.current,
          renderer: !!rendererRef.current,
          camera: !!cameraRef.current,
          particles: !!particlesRef.current
        });
      }
    }, 100);

    let frameCount = 0;
    let lastTime = 0;

    const animate = (currentTime: number) => {
      if (
        !sceneRef.current ||
        !rendererRef.current ||
        !cameraRef.current ||
        !particlesRef.current
      ) {
        return;
      }

      // Debug: Log every frame for first 10 frames to confirm animation is running
      if (frameCount < 10) {
        logger.info(`🎬 Animation frame ${frameCount + 1} - animation loop is working!`);
      }

      // Check for context loss
      try {
        const gl = rendererRef.current.getContext();
        if (gl && gl.isContextLost()) {
          handleContextLoss();
          return;
        }
      } catch (error) {
        logger.warn('Context check failed:', error);
        handleContextLoss();
        return;
      }

      frameCount++;
      const deltaTime = currentTime - lastTime;

      // Debug: Log first few frames to confirm animation loop is running
      if (frameCount <= 5) {
        logger.info(`🎬 Frame ${frameCount} rendering:`, {
          currentTime,
          deltaTime,
          particlesVisible: !!particlesRef.current,
          sceneChildren: sceneRef.current?.children?.length || 0
        });
      }

      // Update FPS
      if (deltaTime >= 1000) {
        const fps = Math.round((frameCount * 1000) / deltaTime);
        setMetrics(prev => ({ ...prev, fps, frameTime: 1000 / fps }));
        frameCount = 0;
        lastTime = currentTime;
      }

      // Update controls
      if (controlsRef.current) {
        controlsRef.current.update();
      }

      // Enhanced particle animation using compute manager
      if (!particlesRef.current?.geometry?.attributes?.position) {
        logger.warn('Particle geometry not ready, skipping animation frame');
        return;
      }

      const positions = particlesRef.current.geometry.attributes.position.array as Float32Array;
      const elapsedTime = clockRef.current?.getElapsedTime() || 0;

      // Use compute manager for enhanced processing if available
      if (computeManagerRef.current && wasmReady) {
        // Debug: Log WASM compute attempt (less frequently)
        if (frameCount % 300 === 0) {
          // Every 5 seconds instead of every second
          logger.info('Attempting WASM compute:', {
            particleCount: positions.length / 3,
            elapsedTime,
            animationMode,
            wasmReady,
            computeManagerReady: !!computeManagerRef.current,
            runConcurrentComputeAvailable:
              typeof (window as any).runConcurrentCompute === 'function',
            runGPUComputeAvailable: typeof (window as any).runGPUCompute === 'function'
          });
        }

        // Prepare particle data (10 values per particle: x, y, z, vx, vy, vz, phase, intensity, type, id)
        const particleData = new Float32Array((positions.length / 3) * 10);
        for (let i = 0; i < positions.length; i += 3) {
          const particleIndex = i / 3;
          const dataIndex = particleIndex * 10;

          // Position
          particleData[dataIndex] = positions[i];
          particleData[dataIndex + 1] = positions[i + 1];
          particleData[dataIndex + 2] = positions[i + 2];

          // Velocity (derived from previous frame)
          particleData[dataIndex + 3] = 0;
          particleData[dataIndex + 4] = 0;
          particleData[dataIndex + 5] = 0;

          // Phase, intensity, type, id
          particleData[dataIndex + 6] = particleIndex * 0.1;
          particleData[dataIndex + 7] = 1.0;
          particleData[dataIndex + 8] =
            animationMode === 'galaxy' ? 1 : animationMode === 'wave' ? 2 : 3;
          particleData[dataIndex + 9] = particleIndex;
        }

        // Process with compute manager
        computeManagerRef.current
          .processParticles(
            particleData,
            elapsedTime,
            animationMode === 'galaxy'
              ? 1
              : animationMode === 'wave'
                ? 2
                : animationMode === 'spiral'
                  ? 3
                  : 0,
            'high'
          )
          .then(result => {
            // Debug: Log WASM compute result (less frequently)
            if (frameCount % 300 === 0) {
              // Every 5 seconds instead of every second
              logger.info('WASM compute result:', {
                resultType: typeof result,
                isArray: Array.isArray(result),
                hasResult: result && typeof result === 'object' && 'result' in result,
                hasMetadata: result && typeof result === 'object' && 'metadata' in result,
                resultDataLength:
                  (result as any)?.result?.length || (result as any)?.length || 'no result',
                firstResultPosition: (result as any)?.result
                  ? [
                      (result as any).result[0],
                      (result as any).result[1],
                      (result as any).result[2]
                    ]
                  : (result as any)?.[0]
                    ? [(result as any)[0], (result as any)[1], (result as any)[2]]
                    : 'no result'
              });
            }

            // Extract the actual result data from the wrapper object or use directly
            const resultData = (result as any)?.result || result;

            // Update positions from compute result
            for (let i = 0; i < positions.length; i += 3) {
              const particleIndex = i / 3;
              const dataIndex = particleIndex * 10;

              positions[i] = resultData[dataIndex];
              positions[i + 1] = resultData[dataIndex + 1];
              positions[i + 2] = resultData[dataIndex + 2];
            }

            if (particlesRef.current?.geometry?.attributes?.position) {
              particlesRef.current.geometry.attributes.position.needsUpdate = true;
            }
          })
          .catch(error => {
            logger.warn('Compute processing failed, falling back to CPU:', error);
            // Fallback to CPU processing
            updateParticlesCPU(positions, elapsedTime, animationMode);
          });
      } else {
        // Debug: Log why WASM compute is not being used (less frequently)
        if (frameCount % 300 === 0) {
          // Every 5 seconds instead of every second
          logger.info('WASM compute not available, using CPU fallback:', {
            computeManagerReady: !!computeManagerRef.current,
            wasmReady,
            runConcurrentComputeAvailable:
              typeof (window as any).runConcurrentCompute === 'function'
          });
        }
        // Fallback to CPU processing
        updateParticlesCPU(positions, elapsedTime, animationMode);
      }

      // Render
      try {
        rendererRef.current.render(sceneRef.current, cameraRef.current);

        // Debug: Log rendering info occasionally
        if (frameCount % 300 === 0) {
          // Every 5 seconds
          const particles = particlesRef.current;
          logger.info('Rendering frame:', {
            frameCount,
            particlesVisible: !!particles,
            sceneChildren: sceneRef.current?.children?.length || 0,
            cameraPosition: [
              cameraRef.current?.position?.x,
              cameraRef.current?.position?.y,
              cameraRef.current?.position?.z
            ],
            rendererType: rendererRef.current?.constructor?.name,
            particleCount: particles?.geometry?.attributes?.position?.count || 0,
            particleSize: particles?.material?.size || 0,
            firstParticlePosition: particles?.geometry?.attributes?.position?.array?.slice(
              0,
              3
            ) || [0, 0, 0],
            rendererSize: [
              rendererRef.current?.domElement?.width,
              rendererRef.current?.domElement?.height
            ]
          });
        }
      } catch (error) {
        logger.warn('Render error:', error);
      }

      animationIdRef.current = requestAnimationFrame(animate);
    };

    animationIdRef.current = requestAnimationFrame(animate);

    // Add immediate debug log to confirm requestAnimationFrame was called
    logger.info('🎬 requestAnimationFrame called - animation loop should start soon');

    return () => {
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
      }
    };
  }, [
    animationMode,
    wasmReady,
    sceneRef.current,
    rendererRef.current,
    cameraRef.current,
    particlesRef.current
  ]);

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

  // Loading state
  if (!threeLoaded) {
    return (
      <LoadingContainer>
        <LoadingTitle>Loading Enhanced Particle System...</LoadingTitle>
        <LoadingSubtitle>Preparing Three.js modules</LoadingSubtitle>
      </LoadingContainer>
    );
  }

  return (
    <ParticleSystemContainer>
      <ParticleCanvas ref={canvasRef} />

      <ControlPanel>
        <ControlTitle>🌌 Enhanced Particle System</ControlTitle>

        <AnimationModeSection>
          <AnimationModeLabel>Animation Mode:</AnimationModeLabel>
          <AnimationModeButtons>
            {(['galaxy', 'yin-yang', 'wave', 'spiral'] as const).map(mode => (
              <AnimationModeButton
                key={mode}
                onClick={() => setAnimationMode(mode)}
                $active={animationMode === mode}
              >
                {mode}
              </AnimationModeButton>
            ))}
          </AnimationModeButtons>
        </AnimationModeSection>

        <MetricsGrid>
          <MetricItem>
            FPS: <MetricValue $color="#4fc3f7">{metrics.fps}</MetricValue>
          </MetricItem>
          <MetricItem>
            Frame Time: <MetricValue $color="#81c784">{metrics.frameTime.toFixed(1)}ms</MetricValue>
          </MetricItem>
          <MetricItem>
            Particles:{' '}
            <MetricValue $color="#ff9800">{metrics.particleCount.toLocaleString()}</MetricValue>
          </MetricItem>
          <MetricItem>
            Render: <MetricValue $color="#ffb74d">{metrics.renderMode}</MetricValue>
          </MetricItem>
          <MetricItem>
            Compute: <MetricValue $color="#e57373">{metrics.computeMode}</MetricValue>
          </MetricItem>
          <MetricItem>
            WASM:{' '}
            <MetricValue $color={metrics.wasmReady ? '#4caf50' : '#f44336'}>
              {metrics.wasmReady ? 'Ready' : 'Not Ready'}
            </MetricValue>
          </MetricItem>
        </MetricsGrid>

        <StatusSection>
          <StatusItem>
            Media Streaming:
            <StatusIndicator $color={mediaStreaming?.connected ? '#4caf50' : '#ff9800'}>
              {mediaStreaming?.connected ? '● CONNECTED' : '○ DISCONNECTED'}
            </StatusIndicator>
          </StatusItem>
          {metrics.renderMode === 'WebGL' && metrics.webgpuReady === false && (
            <StatusItem>
              WebGPU Status:
              <StatusIndicator $color="#ff9800">
                ○ DISABLED - Enable chrome://flags/#enable-unsafe-webgpu
              </StatusIndicator>
            </StatusItem>
          )}
        </StatusSection>

        <Instructions>
          🖱️ Left-click + drag to rotate • Right-click + drag to pan • Scroll to zoom
        </Instructions>
      </ControlPanel>
    </ParticleSystemContainer>
  );
};

export default EnhancedParticleSystem;
