/**
 * Real-time compute streaming hook that bridges WebGPU → Backend
 * Enables backend services to access WASM compute model, WebGPU, and Three.js renderer
 */

import { useEffect, useRef, useState } from 'react';
import { wasmGPU, wasmSendMessage } from '../lib/wasmBridge';
import { useConnectionStatus, useEmitEvent, useEventHistory } from '../store';

interface ComputeStreamConfig {
  streamToBackend: boolean;
  updateFrequency: number; // Hz
  compressionLevel: 'none' | 'low' | 'high';
  includeMetrics: boolean;
  includePositions: boolean;
}

interface ComputeStreamState {
  streaming: boolean;
  lastUpdate: number;
  packetsPerSecond: number;
  bytesPerSecond: number;
  backendConnected: boolean;
}

export function useComputeStreaming(config: ComputeStreamConfig) {
  const [streamState, setStreamState] = useState<ComputeStreamState>({
    streaming: false,
    lastUpdate: 0,
    packetsPerSecond: 0,
    bytesPerSecond: 0,
    backendConnected: false
  });

  const { isConnected } = useConnectionStatus();
  const emitEvent = useEmitEvent();
  const intervalRef = useRef<number | undefined>(undefined);
  const metricsRef = useRef({ packets: 0, bytes: 0, startTime: Date.now() });

  // Stream compute results to backend
  const streamComputeData = async () => {
    try {
      const payload: any = {};

      // Include GPU metrics if requested
      if (config.includeMetrics) {
        const metrics = wasmGPU.getMetrics();
        if (metrics) {
          payload.gpuMetrics = {
            timestamp: metrics.timestamp,
            operation: metrics.operation,
            throughput: metrics.throughput,
            lastOperationTime: metrics.lastOperationTime,
            completionStatus: metrics.completionStatus
          };
        }
      }

      // Include particle positions if requested
      if (config.includePositions) {
        const computeBuffer = wasmGPU.getComputeBuffer();
        if (computeBuffer && computeBuffer.length > 0) {
          // Sample positions for backend analysis (reduce bandwidth)
          const sampleRate =
            config.compressionLevel === 'high' ? 10 : config.compressionLevel === 'low' ? 5 : 1;

          const sampledPositions = [];
          for (let i = 0; i < computeBuffer.length; i += sampleRate * 3) {
            sampledPositions.push(
              computeBuffer[i], // x
              computeBuffer[i + 1], // y
              computeBuffer[i + 2] // z
            );
          }

          payload.particlePositions = {
            data: sampledPositions,
            sampleRate,
            totalParticles: computeBuffer.length / 3,
            timestamp: Date.now()
          };
        }
      }

      // Send to backend via WASM WebSocket
      emitEvent({
        type: 'compute:stream:v1:data',
        payload,
        metadata: {
          campaign: { campaignId: 0, features: [] },
          user: {},
          device: { deviceId: 'demo', consentGiven: true },
          session: { sessionId: 'demo' },
          streamConfig: config,
          renderState: {
            fps: 60, // Could get from Three.js renderer
            frameTime: performance.now(),
            memoryUsage: (performance as any).memory?.usedJSHeapSize || 0
          }
        } as any
      });

      // Update metrics
      const dataSize = JSON.stringify(payload).length;
      metricsRef.current.packets++;
      metricsRef.current.bytes += dataSize;

      const now = Date.now();
      const elapsed = (now - metricsRef.current.startTime) / 1000;

      setStreamState(prev => ({
        ...prev,
        lastUpdate: now,
        packetsPerSecond: metricsRef.current.packets / elapsed,
        bytesPerSecond: metricsRef.current.bytes / elapsed
      }));
    } catch (error) {
      console.error('[ComputeStreaming] Error streaming data:', error);
    }
  };

  // Start streaming
  const startStreaming = () => {
    if (!config.streamToBackend || !isConnected) return;

    const intervalMs = 1000 / config.updateFrequency;
    intervalRef.current = window.setInterval(streamComputeData, intervalMs);

    setStreamState(prev => ({ ...prev, streaming: true }));
    metricsRef.current = { packets: 0, bytes: 0, startTime: Date.now() };

    console.log(`[ComputeStreaming] Started streaming at ${config.updateFrequency}Hz`);
  };

  // Stop streaming
  const stopStreaming = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = undefined;
    }

    setStreamState(prev => ({ ...prev, streaming: false }));
    console.log('[ComputeStreaming] Stopped streaming');
  };

  // Effect to manage streaming based on connection state
  useEffect(() => {
    const connected = isConnected;
    setStreamState(prev => ({ ...prev, backendConnected: connected }));

    if (connected && config.streamToBackend) {
      startStreaming();
    } else {
      stopStreaming();
    }

    return () => stopStreaming();
  }, [isConnected, config.streamToBackend, config.updateFrequency]);

  return {
    streamState,
    startStreaming,
    stopStreaming,
    isStreaming: streamState.streaming
  };
}

/**
 * Backend Compute Control Hook
 * Allows backend to control WASM compute model and WebGPU operations
 */
export function useBackendComputeControl() {
  const events = useEventHistory();

  useEffect(() => {
    // Process compute events
    const computeEvents = events.filter(
      e =>
        e.type.startsWith('compute:control:') ||
        e.type.startsWith('gpu:control:') ||
        e.type.startsWith('render:control:')
    );

    computeEvents.forEach(event => {
      handleBackendComputeCommand(event);
    });
  }, [events]);

  const handleBackendComputeCommand = (event: any) => {
    const { type, payload } = event;

    switch (type) {
      case 'compute:control:v1:setParams':
        // Backend controls animation parameters
        if (payload.animationMode !== undefined) {
          console.log(`[BackendControl] Setting animation mode: ${payload.animationMode}`);
          // Could modify WASM parameters directly
        }
        break;

      case 'gpu:control:v1:benchmark':
        // Backend requests performance benchmark
        console.log('[BackendControl] Running GPU benchmark...');
        wasmGPU.runPerformanceBenchmark(payload.dataSize || 10000).then(results => {
          wasmSendMessage({
            type: 'gpu:benchmark:v1:completed',
            payload: results,
            metadata: { requestId: event.correlationId }
          });
        });
        break;

      case 'render:control:v1:capture':
        // Backend requests render capture
        console.log('[BackendControl] Capturing render state...');
        // Could capture Three.js canvas and send to backend
        break;

      default:
        console.log(`[BackendControl] Unknown command: ${type}`);
    }
  };
}
