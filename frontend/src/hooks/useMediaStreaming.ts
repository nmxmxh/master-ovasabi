import { useEffect, useCallback } from 'react';
import { useConnectionStore } from '../store/stores/connectionStore';

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
  const { mediaStreaming, setMediaStreamingState } = useConnectionStore();

  // Handler to set API ref when ready
  const handleReady = useCallback(() => {
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      setMediaStreamingState({ connected: true, peerId: window.mediaStreaming.peerId, url: window.mediaStreaming.getURL() });
    }
  }, [setMediaStreamingState]);

  useEffect(() => {
    // If already ready, set immediately
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      handleReady();
    } else {
      window.addEventListener('mediaStreamingReady', handleReady);
      return () => window.removeEventListener('mediaStreamingReady', handleReady);
    }
  }, [handleReady]);

  // Connect to campaign only when ready
  const connectToCampaign = useCallback(() => {
    const peerId = typeof window !== 'undefined' && (window as any).userID ? (window as any).userID : undefined;
    if (
      mediaStreaming.connected &&
      window.mediaStreaming &&
      typeof window.mediaStreaming.connectToCampaign === 'function'
    ) {
      window.mediaStreaming.connectToCampaign(campaignId, contextId, peerId);
    } else {
      console.warn(
        '[Media-Streaming] connectToCampaign: Media streaming not ready, waiting for event...'
      );
    }
  }, [campaignId, contextId, mediaStreaming.connected]);

  // Optionally expose other API methods (send, onMessage, etc.)
  return {
    mediaStreaming,
    connectToCampaign,
    isReady: mediaStreaming.connected
  };
}