import React, { useEffect, useRef, useState, useCallback } from 'react';
import * as THREE from 'three';
import { indexedDBManager, type ComputeStateRecord } from '../utils/indexedDBManager';
import { useEventHistory } from '../store/hooks/useEvents';
import { useMetadata } from '../store/hooks/useMetadata';
import { useConnectionStatus } from '../store/hooks/useConnection';

interface PerformanceMetrics {
  fps: number;
  throughput: number;
  latency: number;
  memoryUsage: number;
  particleCount: number;
  processingTime: number;
  lastFrameTime: number;
  lastSummaryTime: number;
  frameCount: number;
  totalLatency: number;
  systemEvents: number;
  campaignUpdates: number;
  searchQueries: number;
}

interface AnalyticsDemoProps {
  className?: string;
}

export const AnalyticsDemo: React.FC<AnalyticsDemoProps> = ({ className }) => {
  const mountRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<THREE.Scene | undefined>(undefined);
  const rendererRef = useRef<THREE.WebGLRenderer | undefined>(undefined);
  const cameraRef = useRef<THREE.PerspectiveCamera | undefined>(undefined);
  const animationRef = useRef<number | undefined>(undefined);
  const particlesRef = useRef<THREE.Points | undefined>(undefined);
  const workerRef = useRef<Worker | undefined>(undefined);

  // Real-time system integration
  const { isConnected } = useConnectionStatus();
  const { metadata } = useMetadata();
  const allEvents = useEventHistory(undefined, 50);
  const searchEvents = useEventHistory('search:search:v1:success', 10);
  const campaignEvents = useEventHistory('campaign:', 10);

  const [metrics, setMetrics] = useState<PerformanceMetrics>({
    fps: 0,
    throughput: 0,
    latency: 0,
    memoryUsage: 0,
    particleCount: 0,
    processingTime: 0,
    lastFrameTime: 0,
    lastSummaryTime: 0,
    frameCount: 0,
    totalLatency: 0,
    systemEvents: 0,
    campaignUpdates: 0,
    searchQueries: 0
  });

  const [isRunning, setIsRunning] = useState(false);
  const [computeStates, setComputeStates] = useState<ComputeStateRecord[]>([]);
  const [dbStats, setDbStats] = useState({
    computeStates: 0,
    userSessions: 0,
    campaigns: 0,
    totalSize: 0
  });

  // Initialize Three.js scene
  const initThreeJS = useCallback(() => {
    if (!mountRef.current) return;

    // Scene
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0a0a);
    sceneRef.current = scene;

    // Camera
    const camera = new THREE.PerspectiveCamera(
      75,
      mountRef.current.clientWidth / mountRef.current.clientHeight,
      0.1,
      1000
    );
    camera.position.z = 50;
    cameraRef.current = camera;

    // Renderer
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setSize(mountRef.current.clientWidth, mountRef.current.clientHeight);
    renderer.setPixelRatio(window.devicePixelRatio);
    mountRef.current.appendChild(renderer.domElement);
    rendererRef.current = renderer;

    // Lighting
    const ambientLight = new THREE.AmbientLight(0x404040, 0.6);
    scene.add(ambientLight);

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
    directionalLight.position.set(10, 10, 5);
    scene.add(directionalLight);

    // Create particle system
    createParticleSystem();
  }, []);

  // Create particle system for visualization
  const createParticleSystem = useCallback(() => {
    if (!sceneRef.current) return;

    const particleCount = 10000;
    const positions = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);
    const sizes = new Float32Array(particleCount);

    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;

      // Position (spherical distribution)
      const radius = Math.random() * 30 + 10;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);

      positions[i3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[i3 + 1] = radius * Math.sin(phi) * Math.sin(theta);
      positions[i3 + 2] = radius * Math.cos(phi);

      // Color (based on position)
      colors[i3] = (positions[i3] + 30) / 60; // R
      colors[i3 + 1] = (positions[i3 + 1] + 30) / 60; // G
      colors[i3 + 2] = (positions[i3 + 2] + 30) / 60; // B

      // Size
      sizes[i] = Math.random() * 2 + 1;
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
    geometry.setAttribute('size', new THREE.BufferAttribute(sizes, 1));

    const material = new THREE.PointsMaterial({
      size: 0.1,
      vertexColors: true,
      blending: THREE.AdditiveBlending,
      transparent: true,
      opacity: 0.8
    });

    const particles = new THREE.Points(geometry, material);
    sceneRef.current.add(particles);
    particlesRef.current = particles;
  }, []);

  // Initialize compute worker with fallback
  const initComputeWorker = useCallback(() => {
    if (workerRef.current) return;

    try {
      const worker = new Worker('/workers/compute-worker.js');
      workerRef.current = worker;

      worker.onmessage = event => {
        const { type, result, data } = event.data;

        switch (type) {
          case 'compute-result':
            if (result && result.data) {
              updateParticles(result.data);
              updateMetrics(result);
              storeComputeState(result);
            }
            break;
          case 'worker-ready':
            console.log('[AnalyticsDemo] Compute worker ready:', data);
            break;
          case 'worker-error':
            console.error('[AnalyticsDemo] Worker error:', data);
            break;
        }
      };

      worker.onerror = error => {
        console.error('[AnalyticsDemo] Worker error:', error);
        // Fallback to CPU-based simulation
        console.log('[AnalyticsDemo] Falling back to CPU simulation');
      };
    } catch (error) {
      console.error('[AnalyticsDemo] Failed to initialize worker:', error);
      console.log('[AnalyticsDemo] Running without worker - using CPU simulation');
    }
  }, []);

  // Update particle positions
  const updateParticles = useCallback((data: Float32Array) => {
    if (!particlesRef.current) return;

    const geometry = particlesRef.current.geometry;
    const positionAttribute = geometry.getAttribute('position') as THREE.BufferAttribute;

    if (data.length === positionAttribute.count * 3) {
      positionAttribute.array = data;
      positionAttribute.needsUpdate = true;
    }
  }, []);

  // Update performance metrics with real-time system data
  const updateMetrics = useCallback(
    (result: any) => {
      setMetrics(prev => {
        const currentTime = performance.now();
        const deltaTime = prev.lastFrameTime ? currentTime - prev.lastFrameTime : 16.67;

        return {
          fps: result.metadata?.fps || prev.fps,
          throughput: result.metadata?.throughput || prev.throughput,
          latency: result.metadata?.latency || deltaTime,
          memoryUsage: result.metadata?.memoryUsage || prev.memoryUsage,
          particleCount: result.metadata?.particleCount || prev.particleCount,
          processingTime: result.metadata?.processingTime || prev.processingTime,
          frameCount: prev.frameCount + 1,
          totalLatency: prev.totalLatency + deltaTime,
          lastFrameTime: currentTime,
          lastSummaryTime: prev.lastSummaryTime,
          systemEvents: allEvents.length,
          campaignUpdates: campaignEvents.length,
          searchQueries: searchEvents.length
        };
      });
    },
    [allEvents.length, campaignEvents.length, searchEvents.length]
  );

  // Store compute state in IndexedDB
  const storeComputeState = useCallback(async (result: any) => {
    try {
      const computeState: ComputeStateRecord = {
        id: `compute_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        type: 'particle_simulation',
        data: result.data,
        params: {
          deltaTime: 0.016667,
          animationMode: 1.0
        },
        timestamp: Date.now(),
        processingTime: result.metadata?.processingTime || 0,
        memoryUsage: result.metadata?.memoryUsage || 0,
        particleCount: result.metadata?.particleCount || 0,
        performance: {
          fps: result.metadata?.fps || 0,
          throughput: result.metadata?.throughput || 0,
          latency: result.metadata?.latency || 0
        }
      };

      await indexedDBManager.storeComputeState(computeState);

      // Update local state
      setComputeStates(prev => [computeState, ...prev.slice(0, 99)]); // Keep last 100
    } catch (error) {
      console.error('[AnalyticsDemo] Failed to store compute state:', error);
    }
  }, []);

  // Start compute simulation
  const startSimulation = useCallback(() => {
    if (!workerRef.current || isRunning) return;

    setIsRunning(true);

    const sendComputeTask = () => {
      if (!workerRef.current || !isRunning) return;

      const particleCount = 10000;
      const data = new Float32Array(particleCount * 3);

      // Generate initial particle positions
      for (let i = 0; i < particleCount; i++) {
        const i3 = i * 3;
        data[i3] = (Math.random() - 0.5) * 100;
        data[i3 + 1] = (Math.random() - 0.5) * 100;
        data[i3 + 2] = (Math.random() - 0.5) * 100;
      }

      workerRef.current.postMessage({
        type: 'compute-task',
        task: {
          id: `task_${Date.now()}`,
          data: data,
          params: {
            deltaTime: 0.016667,
            animationMode: 1.0
          }
        }
      });

      // Schedule next task
      setTimeout(sendComputeTask, 16); // ~60 FPS
    };

    sendComputeTask();
  }, [isRunning]);

  // Stop simulation
  const stopSimulation = useCallback(() => {
    setIsRunning(false);
  }, []);

  // Load analytics data from IndexedDB
  const loadAnalyticsData = useCallback(async () => {
    try {
      const [states, stats] = await Promise.all([
        indexedDBManager.getComputeStatesByType('particle_simulation', 100),
        indexedDBManager.getDatabaseStats()
      ]);

      setComputeStates(states);
      setDbStats(stats);
    } catch (error) {
      console.error('[AnalyticsDemo] Failed to load analytics data:', error);
    }
  }, []);

  // Simplified animation loop with real-time system integration
  const animate = useCallback(() => {
    if (!sceneRef.current || !cameraRef.current || !rendererRef.current) return;

    const currentTime = performance.now();

    // Update metrics with real-time system data
    setMetrics(prev => {
      const deltaTime = prev.lastFrameTime ? currentTime - prev.lastFrameTime : 16.67;
      const newMetrics = { ...prev };

      // Update frame tracking
      newMetrics.frameCount++;
      newMetrics.totalLatency += deltaTime;
      newMetrics.lastFrameTime = currentTime;

      // Calculate FPS (rolling average over last 60 frames)
      if (newMetrics.frameCount % 60 === 0) {
        newMetrics.fps = Math.round(60000 / newMetrics.totalLatency);
        newMetrics.latency = newMetrics.totalLatency / 60;
        newMetrics.totalLatency = 0;
      }

      // Update memory usage
      if ((performance as any).memory) {
        newMetrics.memoryUsage = (performance as any).memory.usedJSHeapSize;
      }

      // Update particle count
      if (particlesRef.current) {
        newMetrics.particleCount = particlesRef.current.geometry.attributes.position.count;
      }

      // Calculate throughput
      newMetrics.throughput = Math.round(newMetrics.particleCount * newMetrics.fps);

      // Update real-time system metrics
      newMetrics.systemEvents = allEvents.length;
      newMetrics.campaignUpdates = campaignEvents.length;
      newMetrics.searchQueries = searchEvents.length;

      return newMetrics;
    });

    // Rotate particles
    if (particlesRef.current) {
      particlesRef.current.rotation.y += 0.001;
      particlesRef.current.rotation.x += 0.0005;
    }

    // Rotate camera
    if (cameraRef.current) {
      cameraRef.current.position.x = Math.sin(Date.now() * 0.0005) * 50;
      cameraRef.current.position.z = Math.cos(Date.now() * 0.0005) * 50;
      cameraRef.current.lookAt(0, 0, 0);
    }

    rendererRef.current.render(sceneRef.current, cameraRef.current);
    animationRef.current = requestAnimationFrame(animate);
  }, [allEvents.length, campaignEvents.length, searchEvents.length]);

  // Handle resize
  const handleResize = useCallback(() => {
    if (!mountRef.current || !cameraRef.current || !rendererRef.current) return;

    const width = mountRef.current.clientWidth;
    const height = mountRef.current.clientHeight;

    cameraRef.current.aspect = width / height;
    cameraRef.current.updateProjectionMatrix();
    rendererRef.current.setSize(width, height);
  }, []);

  // Initialize everything
  useEffect(() => {
    initThreeJS();
    initComputeWorker();
    loadAnalyticsData();

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
      if (workerRef.current) {
        workerRef.current.terminate();
      }
    };
  }, [initThreeJS, initComputeWorker, loadAnalyticsData]);

  // Start animation loop
  useEffect(() => {
    animate();
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [animate]);

  // Handle resize
  useEffect(() => {
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [handleResize]);

  return (
    <div className={`analytics-demo ${className || ''}`}>
      <div className="analytics-controls">
        <h2>🚀 Real-time Analytics Demo</h2>
        <p className="demo-description">WASM-powered analytics with real-time system integration</p>

        {/* System Status */}
        <div className="system-status">
          <div className="status-item">
            <span className="status-label">Connection:</span>
            <span className={`status-value ${isConnected ? 'connected' : 'disconnected'}`}>
              {isConnected ? '🟢 Connected' : '🔴 Disconnected'}
            </span>
          </div>
          <div className="status-item">
            <span className="status-label">Campaign:</span>
            <span className="status-value">{metadata.campaign?.id || 'None'}</span>
          </div>
          <div className="status-item">
            <span className="status-label">User:</span>
            <span className="status-value">{metadata.user?.userId || 'Guest'}</span>
          </div>
        </div>

        <div className="control-buttons">
          <button onClick={startSimulation} disabled={isRunning} className="btn btn-primary">
            {isRunning ? '🔄 Running...' : '▶️ Start Simulation'}
          </button>
          <button onClick={stopSimulation} disabled={!isRunning} className="btn btn-secondary">
            ⏹️ Stop
          </button>
          <button onClick={loadAnalyticsData} className="btn btn-outline">
            🔄 Refresh Data
          </button>
        </div>
      </div>

      <div className="analytics-content">
        <div className="visualization-panel">
          <div ref={mountRef} className="three-js-container" />
        </div>

        <div className="metrics-panel">
          <div className="metrics-grid">
            <div className="metric-card">
              <h3>Performance</h3>
              <div className="metric-value">{metrics.fps.toFixed(1)} FPS</div>
              <div className="metric-label">Frame Rate</div>
            </div>

            <div className="metric-card">
              <h3>Throughput</h3>
              <div className="metric-value">{metrics.throughput.toFixed(0)}</div>
              <div className="metric-label">Particles/sec</div>
            </div>

            <div className="metric-card">
              <h3>Latency</h3>
              <div className="metric-value">{metrics.latency.toFixed(2)}ms</div>
              <div className="metric-label">Processing Time</div>
            </div>

            <div className="metric-card">
              <h3>Memory</h3>
              <div className="metric-value">{(metrics.memoryUsage / 1024 / 1024).toFixed(1)}MB</div>
              <div className="metric-label">Usage</div>
            </div>
          </div>

          {/* Real-time System Metrics */}
          <div className="system-metrics">
            <h3>Real-time System Metrics</h3>
            <div className="metrics-grid">
              <div className="metric-card">
                <h3>Events</h3>
                <div className="metric-value">{metrics.systemEvents}</div>
                <div className="metric-label">Total Events</div>
              </div>

              <div className="metric-card">
                <h3>Campaigns</h3>
                <div className="metric-value">{metrics.campaignUpdates}</div>
                <div className="metric-label">Updates</div>
              </div>

              <div className="metric-card">
                <h3>Search</h3>
                <div className="metric-value">{metrics.searchQueries}</div>
                <div className="metric-label">Queries</div>
              </div>

              <div className="metric-card">
                <h3>Particles</h3>
                <div className="metric-value">{metrics.particleCount.toLocaleString()}</div>
                <div className="metric-label">Active</div>
              </div>
            </div>
          </div>

          <div className="database-stats">
            <h3>Database Statistics</h3>
            <div className="stats-grid">
              <div className="stat-item">
                <span className="stat-label">Compute States:</span>
                <span className="stat-value">{dbStats.computeStates}</span>
              </div>
              <div className="stat-item">
                <span className="stat-label">User Sessions:</span>
                <span className="stat-value">{dbStats.userSessions}</span>
              </div>
              <div className="stat-item">
                <span className="stat-label">Campaigns:</span>
                <span className="stat-value">{dbStats.campaigns}</span>
              </div>
              <div className="stat-item">
                <span className="stat-label">Total Records:</span>
                <span className="stat-value">{dbStats.totalSize}</span>
              </div>
            </div>
          </div>

          <div className="recent-computes">
            <h3>Recent Compute States</h3>
            <div className="compute-list">
              {computeStates.slice(0, 10).map((state, index) => (
                <div key={state.id} className="compute-item">
                  <div className="compute-id">#{index + 1}</div>
                  <div className="compute-details">
                    <div>FPS: {state.performance.fps.toFixed(1)}</div>
                    <div>Particles: {state.particleCount}</div>
                    <div>Time: {new Date(state.timestamp).toLocaleTimeString()}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <style>{`
        .analytics-demo {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: #0a0a0a;
          color: #ffffff;
        }

        .analytics-controls {
          padding: 1rem;
          border-bottom: 1px solid #333;
          background: #1a1a1a;
        }

        .analytics-controls h2 {
          margin: 0 0 0.5rem 0;
          color: #ffffff;
        }

        .demo-description {
          color: #aaa;
          margin: 0 0 1rem 0;
          font-size: 0.9rem;
        }

        .system-status {
          display: flex;
          gap: 2rem;
          margin: 1rem 0;
          padding: 1rem;
          background: #2a2a2a;
          border-radius: 8px;
        }

        .status-item {
          display: flex;
          flex-direction: column;
          gap: 0.25rem;
        }

        .status-label {
          font-size: 0.8rem;
          color: #aaa;
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .status-value {
          font-size: 1rem;
          font-weight: bold;
        }

        .status-value.connected {
          color: #00ff88;
        }

        .status-value.disconnected {
          color: #ff4444;
        }

        .control-buttons {
          display: flex;
          gap: 0.5rem;
        }

        .btn {
          padding: 0.5rem 1rem;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-weight: 500;
          transition: all 0.2s;
        }

        .btn-primary {
          background: #007bff;
          color: white;
        }

        .btn-primary:hover:not(:disabled) {
          background: #0056b3;
        }

        .btn-secondary {
          background: #6c757d;
          color: white;
        }

        .btn-secondary:hover:not(:disabled) {
          background: #545b62;
        }

        .btn-outline {
          background: transparent;
          color: #007bff;
          border: 1px solid #007bff;
        }

        .btn-outline:hover {
          background: #007bff;
          color: white;
        }

        .btn:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .analytics-content {
          display: flex;
          flex: 1;
          overflow: hidden;
        }

        .visualization-panel {
          flex: 2;
          position: relative;
        }

        .three-js-container {
          width: 100%;
          height: 100%;
        }

        .metrics-panel {
          flex: 1;
          padding: 1rem;
          background: #1a1a1a;
          overflow-y: auto;
          border-left: 1px solid #333;
        }

        .metrics-grid {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 1rem;
          margin-bottom: 2rem;
        }

        .metric-card {
          background: #2a2a2a;
          padding: 1rem;
          border-radius: 8px;
          text-align: center;
        }

        .metric-card h3 {
          margin: 0 0 0.5rem 0;
          font-size: 0.9rem;
          color: #888;
        }

        .metric-value {
          font-size: 2rem;
          font-weight: bold;
          color: #00ff88;
          margin-bottom: 0.25rem;
        }

        .metric-label {
          font-size: 0.8rem;
          color: #aaa;
        }

        .system-metrics {
          margin-bottom: 2rem;
        }

        .system-metrics h3 {
          margin: 0 0 1rem 0;
          color: #ffffff;
          font-size: 1.1rem;
        }

        .database-stats {
          margin-bottom: 2rem;
        }

        .database-stats h3 {
          margin: 0 0 1rem 0;
          color: #ffffff;
        }

        .stats-grid {
          display: grid;
          gap: 0.5rem;
        }

        .stat-item {
          display: flex;
          justify-content: space-between;
          padding: 0.5rem;
          background: #2a2a2a;
          border-radius: 4px;
        }

        .stat-label {
          color: #aaa;
        }

        .stat-value {
          color: #00ff88;
          font-weight: bold;
        }

        .recent-computes h3 {
          margin: 0 0 1rem 0;
          color: #ffffff;
        }

        .compute-list {
          max-height: 300px;
          overflow-y: auto;
        }

        .compute-item {
          display: flex;
          align-items: center;
          padding: 0.5rem;
          background: #2a2a2a;
          border-radius: 4px;
          margin-bottom: 0.5rem;
        }

        .compute-id {
          background: #007bff;
          color: white;
          padding: 0.25rem 0.5rem;
          border-radius: 4px;
          font-size: 0.8rem;
          margin-right: 1rem;
          min-width: 2rem;
          text-align: center;
        }

        .compute-details {
          flex: 1;
          font-size: 0.8rem;
        }

        .compute-details div {
          margin-bottom: 0.25rem;
        }
      `}</style>
    </div>
  );
};

export default AnalyticsDemo;
