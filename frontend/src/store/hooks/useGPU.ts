import { useMemo, useCallback } from 'react';
import { useMetadataStore } from '../stores/metadataStore';

/**
 * Hook for accessing GPU capabilities and WASM GPU bridge information
 */
export function useGPUCapabilities() {
  const deviceMetadata = useMetadataStore(state => state.metadata?.device);

  const gpuCapabilities = useMemo(() => {
    return deviceMetadata?.gpuCapabilities || null;
  }, [deviceMetadata?.gpuCapabilities]);

  const wasmGPUBridge = useMemo(() => {
    return deviceMetadata?.wasmGPUBridge || null;
  }, [deviceMetadata?.wasmGPUBridge]);

  const refreshGPUCapabilities = useCallback(async () => {
    try {
      // Dynamically import WASM GPU bridge to avoid circular dependencies
      const { wasmGPU } = await import('../../lib/wasmBridge.js');
      await wasmGPU.updateMetadataWithGPUInfo();
    } catch (error) {
      console.error('[GPU Capabilities Hook] Failed to refresh GPU capabilities:', error);
    }
  }, []);

  return {
    gpuCapabilities,
    wasmGPUBridge,
    refreshGPUCapabilities,
    hasWebGPU: gpuCapabilities?.webgpu?.available || false,
    hasWebGL: gpuCapabilities?.webgl?.available || false,
    hasWASM: wasmGPUBridge?.initialized || false,
    // Legacy compatibility properties
    isWebGPUAvailable: gpuCapabilities?.webgpu?.available || false,
    isWebGLAvailable: gpuCapabilities?.webgl?.available || false,
    recommendedRenderer: gpuCapabilities?.recommendedRenderer || 'webgl',
    performanceScore: gpuCapabilities?.performanceScore || 0,
    detectedAt: deviceMetadata?.gpuDetectedAt || null
  };
}
