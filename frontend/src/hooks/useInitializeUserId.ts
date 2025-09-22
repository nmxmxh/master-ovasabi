import { useEffect, useState } from 'react';
import { useMetadataStore } from '../store/stores/metadataStore';

/**
 * Hook to initialize user ID from WASM when WASM is ready
 */
export const useInitializeUserId = () => {
  const { initializeUserId, userId } = useMetadataStore();
  const [wasmReady, setWasmReady] = useState(false);

  useEffect(() => {
    // Wait for WASM to be ready before initializing
    const handleWasmReady = () => {
      // WASM ready event received
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
    // Effect triggered

    // Only initialize if WASM is ready and userId is still in loading state
    if (wasmReady && userId === 'loading') {
      // Initializing user ID from WASM
      initializeUserId();
    } else if (!wasmReady && userId === 'loading') {
      // WASM not ready, waiting
    } else if (userId !== 'loading') {
      // User ID already initialized
    }
  }, [wasmReady, initializeUserId, userId]);

  return { userId, isInitialized: userId !== 'loading' };
};
