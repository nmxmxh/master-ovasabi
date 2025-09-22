/**
 * Hook for initializing metadata with WASM-generated IDs
 */

import { useEffect, useState } from 'react';
import { useMetadataStore } from '../stores/metadataStore';

/**
 * Hook that initializes metadata with WASM-generated IDs
 * This should be called once when WASM is ready
 */
export function useWasmInitialization() {
  const initializeMetadata = useMetadataStore(state => state.initializeMetadata);
  const initializeUserId = useMetadataStore(state => state.initializeUserId);
  const [wasmReady, setWasmReady] = useState(false);

  useEffect(() => {
    // Wait for WASM to be ready before initializing
    const handleWasmReady = () => {
      console.log('[useWasmInitialization] WASM ready event received');
      setWasmReady(true);
    };

    // Check if WASM is already ready
    if (typeof window !== 'undefined' && (window as any).wasmReady) {
      setWasmReady(true);
    } else {
      // Listen for wasmReady event
      window.addEventListener('wasmReady', handleWasmReady);
    }

    return () => {
      window.removeEventListener('wasmReady', handleWasmReady);
    };
  }, []);

  useEffect(() => {
    // Only initialize when WASM is ready
    if (wasmReady) {
      console.log('[useWasmInitialization] Initializing metadata with WASM IDs');
      initializeMetadata();
      initializeUserId();
    }
  }, [wasmReady, initializeMetadata, initializeUserId]);
}

/**
 * Hook that returns the initialization status
 */
export function useWasmInitializationStatus() {
  const metadata = useMetadataStore(state => state.metadata);
  const userId = useMetadataStore(state => state.userId);

  const isInitialized =
    metadata.user.userId !== 'loading' &&
    metadata.device.deviceId !== 'loading' &&
    metadata.session.sessionId !== 'loading' &&
    userId !== 'loading';

  return {
    isInitialized,
    isLoading: !isInitialized,
    metadata,
    userId
  };
}
