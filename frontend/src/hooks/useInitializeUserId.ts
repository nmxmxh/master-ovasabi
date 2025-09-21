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
      console.log('[useInitializeUserId] WASM ready event received');
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
    console.log('[useInitializeUserId] Effect triggered:', {
      wasmReady,
      userId,
      isInitialized: userId !== 'loading'
    });

    // Only initialize if WASM is ready and userId is still in loading state
    if (wasmReady && userId === 'loading') {
      console.log('[useInitializeUserId] Initializing user ID from WASM');
      initializeUserId();
    } else if (!wasmReady && userId === 'loading') {
      console.log('[useInitializeUserId] WASM not ready, waiting...');
    } else if (userId !== 'loading') {
      console.log('[useInitializeUserId] User ID already initialized:', userId);
    }
  }, [wasmReady, initializeUserId, userId]);

  return { userId, isInitialized: userId !== 'loading' };
};
