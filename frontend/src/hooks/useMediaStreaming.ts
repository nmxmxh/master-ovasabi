import { useEffect, useRef, useCallback } from 'react';

/**
 * useMediaStreaming - React hook for event-driven media streaming integration.
 * Waits for 'mediaStreamingReady' event before allowing connectToCampaign and other API calls.
 * Returns a ref to the mediaStreaming API and a connectToCampaign function.
 */
export interface UseMediaStreamingOptions {
  campaignId?: string;
  contextId?: string;
}

export function useMediaStreaming({
  campaignId = '0',
  contextId = 'webgpu-particles'
}: UseMediaStreamingOptions = {}) {
  const apiRef = useRef<any>(null);
  const readyRef = useRef(false);

  // Handler to set API ref when ready
  const handleReady = useCallback(() => {
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      apiRef.current = window.mediaStreaming;
      readyRef.current = true;
    }
  }, []);

  useEffect(() => {
    // If already ready, set immediately
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      apiRef.current = window.mediaStreaming;
      readyRef.current = true;
    } else {
      window.addEventListener('mediaStreamingReady', handleReady);
      return () => window.removeEventListener('mediaStreamingReady', handleReady);
    }
  }, [handleReady]);

  // Connect to campaign only when ready
  const connectToCampaign = useCallback(() => {
    const peerId = typeof window !== 'undefined' && window.userID ? window.userID : undefined;
    if (
      readyRef.current &&
      apiRef.current &&
      typeof apiRef.current.connectToCampaign === 'function'
    ) {
      apiRef.current.connectToCampaign(campaignId, contextId, peerId);
    } else {
      console.warn(
        '[Media-Streaming] connectToCampaign: Media streaming not ready, waiting for event...'
      );
    }
  }, [campaignId, contextId]);

  // Optionally expose other API methods (send, onMessage, etc.)
  return {
    mediaStreaming: apiRef,
    connectToCampaign,
    isReady: readyRef.current
  };
}
