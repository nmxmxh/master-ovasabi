import React, { useEffect, useRef } from 'react';
import { loadThreeCore } from '../lib/three/index';

interface PhysicsEnvironment3DProps {
  children?: React.ReactNode;
  gravity?: [number, number, number];
  temperature?: number;
  airDensity?: number;
}

export const PhysicsEnvironment3D: React.FC<PhysicsEnvironment3DProps> = ({
  children,
  gravity = [0, -9.8, 0],
  temperature = 20,
  airDensity = 1.2
}) => {
  const envRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let scene: any, camera: any, renderer: any, animationId: number;

    async function initEnv() {
      const core = await loadThreeCore();
      scene = new core.Scene();
      camera = new core.PerspectiveCamera(
        75,
        envRef.current!.clientWidth / envRef.current!.clientHeight,
        0.1,
        1000
      );
      camera.position.set(0, 2, 8);

      renderer = new core.WebGLRenderer({ antialias: true });
      renderer.setSize(envRef.current!.clientWidth, envRef.current!.clientHeight);
      envRef.current!.appendChild(renderer.domElement);

      // Example: Ground plane (manual BufferGeometry for compatibility)
      const groundGeometry = new core.BufferGeometry();
      const vertices = new Float32Array([-5, 0, -5, 5, 0, -5, 5, 0, 5, -5, 0, 5]);
      groundGeometry.setAttribute('position', new core.BufferAttribute(vertices, 3));
      groundGeometry.setIndex([0, 1, 2, 0, 2, 3]);
      const groundMaterial = new core.MeshBasicMaterial({ color: 0x444444 });
      const ground = new core.Mesh(groundGeometry, groundMaterial);
      ground.position.y = -0.5;
      scene.add(ground);

      // Lighting
      const ambientLight = new core.AmbientLight(0xffffff, 0.5);
      scene.add(ambientLight);
      const sun = new core.DirectionalLight(0xfff7e0, 1.0);
      sun.position.set(10, 20, 10);
      scene.add(sun);

      // Animation loop
      function animate() {
        animationId = requestAnimationFrame(animate);
        renderer.render(scene, camera);
      }
      animate();
    }

    initEnv();

    return () => {
      if (animationId) cancelAnimationFrame(animationId);
      if (renderer) {
        renderer.dispose();
        envRef.current?.removeChild(renderer.domElement);
      }
    };
  }, [gravity, temperature, airDensity]);

  return (
    <div
      ref={envRef}
      style={{ width: '100%', height: '500px', background: '#222', position: 'relative' }}
    >
      {children}
    </div>
  );
};
