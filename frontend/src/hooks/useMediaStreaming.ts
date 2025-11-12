import { useEffect, useCallback, useRef } from 'react';
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
  const {
    mediaStreaming,
    setMediaStreamingState,
    clearMediaStreamingState,
    connectMediaStreaming,
    disconnectMediaStreaming,
    wasmReady
  } = useConnectionStore();
  const lastCampaignKeyRef = useRef<string | null>(null);

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
                url: window.mediaStreaming!.getURL()
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

  const connect = useCallback(() => {
    connectMediaStreaming(campaignId, contextId);
  }, [campaignId, contextId, connectMediaStreaming]);

  const disconnect = useCallback(() => {
    disconnectMediaStreaming();
    clearMediaStreamingState();
  }, [clearMediaStreamingState, disconnectMediaStreaming]);

  const connectToCampaign = useCallback(() => {
    const peerId =
      typeof window !== 'undefined' && (window as any).userID ? (window as any).userID : undefined;
    if (!peerId) {
      console.error('Peer ID not found on window.userID');
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

  useEffect(() => {
    if (!wasmReady) {
      return;
    }
    const mediaAPI =
      typeof window !== 'undefined' ? (window as any).mediaStreaming : undefined;
    if (!mediaAPI || typeof mediaAPI.connect !== 'function') {
      return;
    }
    if (!mediaStreaming.connected && !mediaStreaming.connecting) {
      connect();
    }
  }, [connect, mediaStreaming.connected, mediaStreaming.connecting, wasmReady]);

  useEffect(() => {
    if (!wasmReady || !mediaStreaming.connected) {
      return;
    }
    const mediaAPI =
      typeof window !== 'undefined' ? (window as any).mediaStreaming : undefined;
    if (!mediaAPI || typeof mediaAPI.connectToCampaign !== 'function') {
      return;
    }
    const campaignKey = `${campaignId}:${contextId}`;
    if (lastCampaignKeyRef.current === campaignKey) {
      return;
    }
    connectToCampaign();
    lastCampaignKeyRef.current = campaignKey;
  }, [campaignId, contextId, connectToCampaign, mediaStreaming.connected, wasmReady]);

  return {
    mediaStreaming,
    connect,
    disconnect,
    connectToCampaign,
    isReady: mediaStreaming.connected && wasmReady
  };
}