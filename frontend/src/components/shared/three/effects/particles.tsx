import { useRef, useEffect } from 'react';
import * as THREE from 'three';
import { WebGPURenderer } from 'three/webgpu';

export function WebGPUParticles() {
  const mountRef = useRef<HTMLDivElement>(null);
  const runningRef = useRef(true);

  useEffect(() => {
    const width = window.innerWidth;
    const height = window.innerHeight;
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 2000);
    camera.position.set(0, 0, 200);
    camera.lookAt(0, 0, 0);

    let renderer: any;
    let animationFrameId: number;
    let disposed = false;

    // Pointer state (NDC)
    let pointer = { x: 0, y: 0 };

    (async () => {
      if ('gpu' in navigator) {
        try {
          renderer = new WebGPURenderer({ antialias: true });
          await renderer.init();
        } catch (e) {
          return;
        }
      } else {
        return;
      }
      renderer.setClearColor(0x000000, 0); // transparent
      renderer.setSize(width, height, false);
      renderer.domElement.style.width = '100vw';
      renderer.domElement.style.height = '100vh';
      renderer.domElement.style.position = 'absolute';
      renderer.domElement.style.left = '0';
      renderer.domElement.style.top = '0';
      renderer.domElement.style.pointerEvents = 'auto';

      // --- PARTICLE SYSTEM ---
      const PARTICLE_COUNT = 256;
      const positions = new Float32Array(PARTICLE_COUNT * 3);
      const velocities = new Float32Array(PARTICLE_COUNT * 3);
      const colors = new Float32Array(PARTICLE_COUNT * 3);
      for (let i = 0; i < PARTICLE_COUNT; ++i) {
        positions[i * 3 + 0] = (Math.random() - 0.5) * 100;
        positions[i * 3 + 1] = (Math.random() - 0.5) * 100;
        positions[i * 3 + 2] = (Math.random() - 0.5) * 100;
        velocities[i * 3 + 0] = (Math.random() - 0.5) * 0.5;
        velocities[i * 3 + 1] = (Math.random() - 0.5) * 0.5;
        velocities[i * 3 + 2] = (Math.random() - 0.5) * 0.5;
        // Initial color (will animate)
        const hue = i / PARTICLE_COUNT;
        const color = new THREE.Color().setHSL(hue, 1.0, 0.6);
        colors[i * 3 + 0] = color.r;
        colors[i * 3 + 1] = color.g;
        colors[i * 3 + 2] = color.b;
      }
      // Points
      const geometry = new THREE.BufferGeometry();
      geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
      geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
      const pointsMaterial = new THREE.PointsMaterial({
        size: 2.5,
        sizeAttenuation: true,
        transparent: true,
        opacity: 0.8,
        depthWrite: false,
        vertexColors: true
      });
      const points = new THREE.Points(geometry, pointsMaterial);
      scene.add(points);
      // Links
      const maxLinks = PARTICLE_COUNT * 2;
      const linkPositions = new Float32Array(maxLinks * 2 * 3); // 2 points per link
      const linkGeometry = new THREE.BufferGeometry();
      linkGeometry.setAttribute('position', new THREE.BufferAttribute(linkPositions, 3));
      const linkMaterial = new THREE.LineBasicMaterial({
        color: 0x88aaff,
        transparent: true,
        opacity: 0.25
      });
      const links = new THREE.LineSegments(linkGeometry, linkMaterial);
      scene.add(links);

      // Pointer interaction
      renderer.domElement.addEventListener('pointermove', (e: PointerEvent) => {
        pointer.x = (e.clientX / window.innerWidth) * 2 - 1;
        pointer.y = -(e.clientY / window.innerHeight) * 2 + 1;
      });

      runningRef.current = true;
      function animate(time: number) {
        if (!runningRef.current || disposed) return;
        // Animate color
        for (let i = 0; i < PARTICLE_COUNT; ++i) {
          const hue = (i / PARTICLE_COUNT + time * 0.0001) % 1.0;
          const color = new THREE.Color().setHSL(hue, 1.0, 0.6);
          colors[i * 3 + 0] = color.r;
          colors[i * 3 + 1] = color.g;
          colors[i * 3 + 2] = color.b;
        }
        geometry.attributes.color.needsUpdate = true;
        // Move particles
        for (let i = 0; i < PARTICLE_COUNT; ++i) {
          positions[i * 3 + 0] += velocities[i * 3 + 0];
          positions[i * 3 + 1] += velocities[i * 3 + 1];
          positions[i * 3 + 2] += velocities[i * 3 + 2];
          // Bounce off bounds
          for (let j = 0; j < 3; ++j) {
            if (positions[i * 3 + j] > 50 || positions[i * 3 + j] < -50) {
              velocities[i * 3 + j] *= -1;
            }
          }
        }
        geometry.attributes.position.needsUpdate = true;
        // Update links
        let linkIdx = 0;
        for (let i = 0; i < PARTICLE_COUNT; ++i) {
          for (let j = i + 1; j < PARTICLE_COUNT; ++j) {
            const dx = positions[i * 3 + 0] - positions[j * 3 + 0];
            const dy = positions[i * 3 + 1] - positions[j * 3 + 1];
            const dz = positions[i * 3 + 2] - positions[j * 3 + 2];
            const distSq = dx * dx + dy * dy + dz * dz;
            if (distSq < 400) {
              // link if close
              if (linkIdx < maxLinks) {
                linkPositions[linkIdx * 6 + 0] = positions[i * 3 + 0];
                linkPositions[linkIdx * 6 + 1] = positions[i * 3 + 1];
                linkPositions[linkIdx * 6 + 2] = positions[i * 3 + 2];
                linkPositions[linkIdx * 6 + 3] = positions[j * 3 + 0];
                linkPositions[linkIdx * 6 + 4] = positions[j * 3 + 1];
                linkPositions[linkIdx * 6 + 5] = positions[j * 3 + 2];
                linkIdx++;
              }
            }
          }
        }
        for (let i = linkIdx * 6; i < maxLinks * 6; ++i) {
          linkPositions[i] = 0;
        }
        linkGeometry.attributes.position.needsUpdate = true;
        // Camera follows pointer
        camera.position.x += (pointer.x * 100 - camera.position.x) * 0.05;
        camera.position.y += (pointer.y * 100 - camera.position.y) * 0.05;
        camera.lookAt(0, 0, 0);
        renderer.render(scene, camera);
        animationFrameId = requestAnimationFrame(animate);
      }
      animate(0);

      if (mountRef.current && !mountRef.current.contains(renderer.domElement)) {
        mountRef.current.appendChild(renderer.domElement);
      }

      function handleResize() {
        const w = window.innerWidth;
        const h = window.innerHeight;
        renderer.setSize(w, h, false);
        renderer.domElement.style.width = '100vw';
        renderer.domElement.style.height = '100vh';
        camera.aspect = w / h;
        camera.updateProjectionMatrix();
      }
      window.addEventListener('resize', handleResize);

      // Cleanup
      return () => {
        disposed = true;
        runningRef.current = false;
        window.removeEventListener('resize', handleResize);
        if (renderer && renderer.domElement && mountRef.current) {
          mountRef.current.removeChild(renderer.domElement);
        }
        if (animationFrameId) cancelAnimationFrame(animationFrameId);
        if (renderer && typeof renderer.dispose === 'function') renderer.dispose();
        geometry.dispose();
        pointsMaterial.dispose();
        linkGeometry.dispose();
        linkMaterial.dispose();
      };
    })();
    // No cleanup here, handled in async IIFE
  }, []);

  return (
    <div
      ref={mountRef}
      style={{
        position: 'absolute',
        left: 0,
        top: 0,
        width: '100vw',
        height: '100vh',
        zIndex: 1,
        overflow: 'hidden',
        pointerEvents: 'auto'
      }}
    />
  );
}
