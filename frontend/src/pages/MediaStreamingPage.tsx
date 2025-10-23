import React, { useEffect, useCallback, useRef } from 'react';
import { useMediaStreaming } from '../hooks/useMediaStreaming';
import { useConnectionStore } from '../store';
import { useWebRTC } from '../hooks/useWebRTC';
import { useWebRTCStats } from '../hooks/useWebRTCStats';
import ThreeJSStreamPlayer from '../components/ThreeJSStreamPlayer';

const WebRTCStats: React.FC<{ stats: RTCStatsReport | null }> = ({ stats }) => {
  if (!stats) {
    return <p className="minimal-text">No stats available</p>;
  }

  const statsArray = Array.from(stats.values());

  return (
    <div className="minimal-code">
      {statsArray.map(stat => (
        <div key={stat.id}>
          <h3>{stat.type}</h3>
          <pre>{JSON.stringify(stat, null, 2)}</pre>
        </div>
      ))}
    </div>
  );
};

const MediaStreamingPage: React.FC = () => {
  const { mediaStreaming, setMediaStreamingState } = useConnectionStore();
  const { connectToCampaign, isReady } = useMediaStreaming();
  const { peerConnection, remoteStream, start, stop } = useWebRTC('default-room', 'default-user');
  const stats = useWebRTCStats(peerConnection);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    console.log('[MediaStreamingPage] Component mounted');
    return () => console.log('[MediaStreamingPage] Component unmounted');
  }, []);

  const handleStreamEvent = useCallback(
    (e: CustomEvent) => {
      console.log('[MediaStreamingPage] Received mediaStream event:', e.detail);
      if (e.detail.stream) {
        setMediaStreamingState({ remoteStream: e.detail.stream });
        if (videoRef.current) {
          videoRef.current.srcObject = e.detail.stream;
        }
      }
    },
    [setMediaStreamingState]
  );

  const handleDisconnectEvent = useCallback(() => {
    console.log('[MediaStreamingPage] Received mediaStreamDisconnect event');
    setMediaStreamingState({ remoteStream: null });
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
  }, [setMediaStreamingState]);

  useEffect(() => {
    if (isReady) {
      console.log('[MediaStreamingPage] Media streaming is ready, attaching event listeners');
      window.addEventListener('mediaStream', handleStreamEvent as EventListener);
      window.addEventListener('mediaStreamDisconnect', handleDisconnectEvent);

      return () => {
        console.log('[MediaStreamingPage] Cleaning up media streaming event listeners');
        window.removeEventListener('mediaStream', handleStreamEvent as EventListener);
        window.removeEventListener('mediaStreamDisconnect', handleDisconnectEvent);
      };
    }
  }, [isReady, handleStreamEvent, handleDisconnectEvent]);

  const handleConnect = () => {
    console.log('[MediaStreamingPage] Attempting to connect...');
    start();
    connectToCampaign();
  };

  const handleDisconnect = () => {
    console.log('[MediaStreamingPage] Attempting to disconnect...');
    stop();
    if (window.mediaStreaming && typeof window.mediaStreaming.disconnect === 'function') {
      window.mediaStreaming.disconnect();
    }
  };

  return (
    <div className="minimal-section">
      <h1 className="minimal-title">Media Streaming Demo</h1>
      <div className="minimal-text">
        <p>Status: {mediaStreaming.connected ? 'Connected' : 'Disconnected'}</p>
        <p>Peer ID: {mediaStreaming.peerId || 'N/A'}</p>
        <p>URL: {mediaStreaming.url || 'N/A'}</p>
      </div>
      <button
        onClick={handleConnect}
        className="minimal-button"
        disabled={mediaStreaming.connected || !isReady}
      >
        Connect
      </button>
      <button
        onClick={handleDisconnect}
        className="minimal-button"
        disabled={!mediaStreaming.connected}
      >
        Disconnect
      </button>

      <div className="minimal-section" style={{ marginTop: '2rem', height: '500px' }}>
        <h2 className="minimal-title">Live Stream (3D)</h2>
        {remoteStream ? (
          <ThreeJSStreamPlayer stream={remoteStream} />
        ) : (
          <div
            style={{
              border: '1px dashed #333',
              padding: '2rem',
              textAlign: 'center',
              height: '100%'
            }}
          >
            <p className="minimal-text">No stream available</p>
          </div>
        )}
      </div>

      <div className="minimal-section" style={{ marginTop: '2rem' }}>
        <h2 className="minimal-title">WebRTC Stats</h2>
        <WebRTCStats stats={stats} />
      </div>
    </div>
  );
};

export default MediaStreamingPage;
