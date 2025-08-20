import React, { useEffect, useRef } from 'react';
import { loadThreeCore, loadThreeRenderers } from '../lib/three/index';
import { wasmSendMessage } from '../lib/wasmBridge';
import { useGlobalStore } from '../store/global';

interface PhysicsEntity3DProps {
  entityId: string;
  metadata: any;
}

export const PhysicsEntity3D: React.FC<PhysicsEntity3DProps> = ({ entityId, metadata }) => {
  const threeRef = useRef<HTMLDivElement>(null);
  const isConnected = useGlobalStore(state => state.isConnected());

  useEffect(() => {
    let scene: any, camera: any, renderer: any, mesh: any, animationId: number;

    async function initThree() {
      const core = await loadThreeCore();
      await loadThreeRenderers();

      scene = new core.Scene();
      camera = new core.PerspectiveCamera(
        75,
        threeRef.current!.clientWidth / threeRef.current!.clientHeight,
        0.1,
        1000
      );
      camera.position.z = 5;

      renderer = new core.WebGLRenderer({ antialias: true });
      renderer.setSize(threeRef.current!.clientWidth, threeRef.current!.clientHeight);
      threeRef.current!.appendChild(renderer.domElement);

      // Example: Sphere entity
      const geometry = new core.SphereGeometry(metadata.radius || 1, 32, 32);
      // Use MeshBasicMaterial for compatibility with ThreeCore
      const material = new core.MeshBasicMaterial({ color: metadata.color || 0x2194f3 });
      mesh = new core.Mesh(geometry, material);
      scene.add(mesh);

      // Lighting
      const ambientLight = new core.AmbientLight(0xffffff, 0.6);
      scene.add(ambientLight);
      const directionalLight = new core.DirectionalLight(0xffffff, 0.8);
      directionalLight.position.set(5, 10, 7.5);
      scene.add(directionalLight);

      // Animation loop
      function animate() {
        animationId = requestAnimationFrame(animate);
        mesh.rotation.y += 0.01;
        renderer.render(scene, camera);
      }
      animate();
    }

    initThree();

    // Register entity with Godot via ws-gateway
    if (isConnected) {
      wasmSendMessage({
        type: 'register_entity',
        payload: {
          entity_id: entityId,
          metadata
        }
      });
    }

    return () => {
      if (animationId) cancelAnimationFrame(animationId);
      if (renderer) {
        renderer.dispose();
        threeRef.current?.removeChild(renderer.domElement);
      }
    };
  }, [entityId, metadata, isConnected]);

  return <div ref={threeRef} style={{ width: '100%', height: '400px', background: '#111' }} />;
};
