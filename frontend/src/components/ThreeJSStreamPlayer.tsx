import React, { useRef, useEffect, useState } from 'react';
import { Canvas } from '@react-three/fiber';
import * as THREE from 'three';

interface ThreeJSStreamPlayerProps {
  stream: MediaStream | null;
}

const VideoPlane: React.FC<ThreeJSStreamPlayerProps> = ({ stream }) => {
  const videoRef = useRef<HTMLVideoElement>(document.createElement('video'));
  const [texture, setTexture] = useState<THREE.VideoTexture | null>(null);

  useEffect(() => {
    if (stream) {
      const video = videoRef.current;
      video.srcObject = stream;
      video.muted = true;
      video.play().catch(err => console.error('Video play failed:', err));
      const videoTexture = new THREE.VideoTexture(video);
      setTexture(videoTexture);
    }
  }, [stream]);

  return (
    <mesh>
      <planeGeometry args={[16, 9]} />
      <meshBasicMaterial map={texture} />
    </mesh>
  );
};

const ThreeJSStreamPlayer: React.FC<ThreeJSStreamPlayerProps> = ({ stream }) => {
  return (
    <Canvas style={{ background: '#111' }}>
      <ambientLight intensity={0.5} />
      <pointLight position={[10, 10, 10]} />
      {stream && <VideoPlane stream={stream} />}
    </Canvas>
  );
};

export default ThreeJSStreamPlayer;