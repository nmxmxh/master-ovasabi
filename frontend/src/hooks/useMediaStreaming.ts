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
  onMessage?: (message: any) => void;
  onState?: (state: string) => void;
}

export function useMediaStreaming({
  campaignId = '0',
  contextId = 'webgpu-particles',
  onMessage,
  onState
}: UseMediaStreamingOptions = {}) {
  const { mediaStreaming, setMediaStreamingState, wasmReady } = useConnectionStore();

  const handleReady = useCallback(() => {
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      if (typeof window.mediaStreaming.onState === 'function') {
        window.mediaStreaming.onState((state: string) => {
          switch (state) {
            case 'connecting':
              setMediaStreamingState({ connecting: true });
              break;
            case 'connected':
              setMediaStreamingState({
                connected: true,
                connecting: false,
                peerId: window.mediaStreaming!.getPeerID(),
                url: window.mediaStreaming!.getURL(),
              });
              break;
            case 'disconnected':
            case 'failed':
              setMediaStreamingState({ connected: false, connecting: false });
              break;
          }
          if (onState) {
            onState(state);
          }
        });
      }
      if (onMessage && typeof window.mediaStreaming.onMessage === 'function') {
        window.mediaStreaming.onMessage(onMessage);
      }
    }
  }, [setMediaStreamingState, onMessage, onState]);

  useEffect(() => {
    if (typeof window !== 'undefined' && window.mediaStreaming) {
      handleReady();
    } else {
      window.addEventListener('mediaStreamingReady', handleReady);
      return () => window.removeEventListener('mediaStreamingReady', handleReady);
    }
  }, [handleReady]);

  const connectToCampaign = useCallback(() => {
    const peerId = typeof window !== 'undefined' && (window as any).userID ? (window as any).userID : undefined;
    if (!peerId) {
        console.error("Peer ID not found on window.userID");
        return;
    }
    if (
      wasmReady &&
      window.mediaStreaming &&
      typeof window.mediaStreaming.connectToCampaign === 'function'
    ) {
      window.mediaStreaming.connectToCampaign(campaignId, contextId, peerId);
    } else {
      console.warn(
        '[Media-Streaming] connectToCampaign: Media streaming not ready, waiting for event...'
      );
    }
  }, [campaignId, contextId, wasmReady]);

  return {
    mediaStreaming,
    connectToCampaign,
    isReady: mediaStreaming.connected,
  };
}