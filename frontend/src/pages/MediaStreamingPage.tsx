import React, { useEffect, useCallback, useState } from 'react';
import { useWebRTC } from '../hooks/useWebRTC';
import ParticleRenderer from '../components/ParticleRenderer';
import * as THREE from 'three';
import { useConnectionStore } from '../store';

const MediaStreamingPage: React.FC = () => {
  const [particles, setParticles] = useState<Float32Array | null>(null);
  const { peerId, url } = useConnectionStore(state => state.mediaStreaming);

  const handleDataMessage = useCallback((data: string) => {
    try {
      const message = JSON.parse(data);
      if (message.Type === 'particle_data' && message.Data.particles) {
        const particleData = new Float32Array(message.Data.particles);
        setParticles(particleData);
      }
    } catch (error) {
      console.error('Failed to parse message:', error);
    }
  }, []);

  const { connected, connecting, start, stop, sendData } = useWebRTC(
    'default-room',
    'default-user',
    handleDataMessage
  );

  const sendPointerPosition = useCallback(
    (position: THREE.Vector3) => {
      if (connected && sendData) {
        const message = {
          Type: 'pointer_move',
          Data: {
            x: position.x,
            y: position.y,
            z: position.z
          }
        };
        sendData(JSON.stringify(message));
      }
    },
    [connected, sendData]
  );

  useEffect(() => {
    console.log('[MediaStreamingPage] Component mounted');
    return () => console.log('[MediaStreamingPage] Component unmounted');
  }, []);

  const handleConnect = () => {
    console.log('[MediaStreamingPage] Attempting to connect...');
    start();
  };

  const handleDisconnect = () => {
    console.log('[MediaStreamingPage] Attempting to disconnect...');
    stop();
  };

  return (
    <div className="minimal-section">
      <h1 className="minimal-title">Media Streaming Demo</h1>
      <div className="minimal-text">
        <p>Status: {connected ? 'Connected' : connecting ? 'Connecting...' : 'Disconnected'}</p>
        <p>Peer ID: {peerId || 'N/A'}</p>
        <p>URL: {url || 'N/A'}</p>
      </div>
      <button onClick={handleConnect} className="minimal-button" disabled={connected || connecting}>
        Connect
      </button>
      <button onClick={handleDisconnect} className="minimal-button" disabled={!connected}>
        Disconnect
      </button>

      <div className="minimal-section" style={{ marginTop: '2rem', height: '500px' }}>
        <h2 className="minimal-title">Live Stream (3D)</h2>
        {connected ? (
          <ParticleRenderer particles={particles} onPointerMove={sendPointerPosition} />
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
    </div>
  );
};

export default MediaStreamingPage;
