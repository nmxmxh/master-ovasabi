import React, { useRef, useEffect, useState } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { loadAllThreeModules, createWebGPUParticleSystem } from '../lib/three';

interface ParticleRendererProps {
  particles: Float32Array | null;
  onPointerMove?: (position: THREE.Vector3) => void;
}

const Particles: React.FC<ParticleRendererProps> = ({ particles, onPointerMove }) => {
  const meshRef = useRef<THREE.Points>(null!);
  const [three, setThree] = useState<any>(null);
  const [particleSystem, setParticleSystem] = useState<any>(null);

  useEffect(() => {
    loadAllThreeModules().then(setThree);
  }, []);

  useEffect(() => {
    if (three && particles) {
      createWebGPUParticleSystem(particles.length / 3).then(setParticleSystem);
    }
  }, [three, particles]);

  useFrame((state, delta) => {
    if (particleSystem) {
      particleSystem.updateFunction(delta);
      if (meshRef.current) {
        meshRef.current.geometry.attributes.position.needsUpdate = true;
      }
    }

    if (onPointerMove) {
      const { pointer, camera } = state;
      const vector = new THREE.Vector3(pointer.x, pointer.y, 0.5).unproject(camera);
      const dir = vector.sub(camera.position).normalize();
      const distance = -camera.position.z / dir.z;
      const pos = camera.position.clone().add(dir.multiplyScalar(distance));
      onPointerMove(pos);
    }
  });

  if (!particleSystem) return null;

  return <primitive object={particleSystem.mesh} ref={meshRef} />;
};

const ParticleRenderer: React.FC<ParticleRendererProps> = ({ particles, onPointerMove }) => {
  return (
    <Canvas style={{ background: '#000' }} camera={{ position: [0, 0, 25] }}>
      <ambientLight intensity={0.5} />
      <pointLight position={[10, 10, 10]} />
      {particles && <Particles particles={particles} onPointerMove={onPointerMove} />}
    </Canvas>
  );
};

export default ParticleRenderer;
