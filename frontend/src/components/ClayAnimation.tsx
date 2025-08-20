import React, { useRef, useEffect, useState } from 'react';
import styled from 'styled-components';

const ClayContainer = styled.div`
  position: relative;
  width: 100vw;
  height: 100vh;
  background: #1a1a2e;
  overflow: hidden;
`;

const Overlay = styled.div`
  position: absolute;
  top: 20px;
  left: 0;
  width: 100%;
  text-align: center;
  color: #e0e0ff;
  z-index: 10;
  pointer-events: none;
`;

const FPSCounter = styled.div`
  position: absolute;
  top: 20px;
  right: 20px;
  background: rgba(0, 0, 0, 0.3);
  color: #a0ffc0;
  padding: 8px 15px;
  border-radius: 20px;
  font-family: monospace;
  font-size: 1.1rem;
  border: 1px solid rgba(100, 255, 150, 0.2);
  z-index: 10;
`;

const Controls = styled.div`
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.5);
  color: white;
  padding: 15px;
  border-radius: 15px;
  display: flex;
  gap: 15px;
  z-index: 10;
  backdrop-filter: blur(5px);
  border: 1px solid rgba(255, 255, 255, 0.1);
`;

const ControlGroup = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

const ClayAnimation: React.FC = () => {
  const mountRef = useRef<HTMLDivElement>(null);
  const [fps, setFps] = useState(60);
  const [deformIntensity, setDeformIntensity] = useState(0.5);
  const [animationSpeed, setAnimationSpeed] = useState(1.0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let THREE: any;
    let renderer: any;
    let scene: any;
    let camera: any;
    let controls: any;
    let clayMesh: any;
    let geometry: any;
    let particles: any;
    let clock: any;
    let time = 0;
    let frameCount = 0;
    let lastFpsUpdate = 0;
    let originalVertices: Float32Array;
    let animationFrameId: number;
    let isMounted = true;

    (async () => {
      // Use the dynamic loader from src/lib/three/index.ts
      const { loadThreeCore, loadThreeRenderers } = await import('../lib/three');
      THREE = await loadThreeCore();
      const renderers = await loadThreeRenderers();
      let WebGPURenderer = renderers.WebGPURenderer;
      // Setup renderer
      if (WebGPURenderer && renderers.webgpuAvailable) {
        renderer = new WebGPURenderer({ antialias: true, powerPreference: 'high-performance' });
      } else {
        renderer = new THREE.WebGLRenderer({ antialias: true });
      }
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      renderer.setSize(window.innerWidth, window.innerHeight);
      mountRef.current?.appendChild(renderer.domElement);

      if (!isMounted || !mountRef.current) return;
      scene = new THREE.Scene();
      scene.background = new THREE.Color(0x0a0a1a);
      scene.fog = new THREE.Fog(0x0a0a1a, 15, 30);

      // Camera
      camera = new THREE.PerspectiveCamera(45, window.innerWidth / window.innerHeight, 0.1, 1000);
      camera.position.set(0, 2, 8);
      camera.lookAt(0, 0, 0);

      // Controls
      const { OrbitControls } = await import('three/examples/jsm/controls/OrbitControls.js');
      controls = new OrbitControls(camera, renderer.domElement);
      controls.enableDamping = true;
      controls.dampingFactor = 0.05;
      controls.screenSpacePanning = false;
      controls.minDistance = 5;
      controls.maxDistance = 20;

      // Lighting
      scene.add(new THREE.AmbientLight(0x404040, 2));
      const directionalLight = new THREE.DirectionalLight(0xffeedd, 1.5);
      directionalLight.position.set(5, 10, 7);
      directionalLight.castShadow = true;
      scene.add(directionalLight);
      const backLight = new THREE.DirectionalLight(0x4466cc, 1.2);
      backLight.position.set(-5, 5, -5);
      scene.add(backLight);

      // Clay material
      const clayMaterial = new THREE.MeshStandardMaterial({
        color: 0xdd8866,
        roughness: 0.9,
        metalness: 0.1,
        flatShading: true,
        emissive: 0x221100,
        emissiveIntensity: 0.1
      });
      geometry = new THREE.TorusKnotGeometry(1.5, 0.5, 256, 64);
      clayMesh = new THREE.Mesh(geometry, clayMaterial);
      clayMesh.castShadow = true;
      clayMesh.receiveShadow = true;
      scene.add(clayMesh);
      originalVertices = new Float32Array(geometry.attributes.position.array);

      // Ground
      const groundGeometry = new THREE.PlaneGeometry(30, 30, 1, 1);
      const groundMaterial = new THREE.MeshStandardMaterial({
        color: 0x1a1a2e,
        roughness: 0.8,
        metalness: 0.2
      });
      const ground = new THREE.Mesh(groundGeometry, groundMaterial);
      ground.rotation.x = -Math.PI / 2;
      ground.position.y = -3;
      ground.receiveShadow = true;
      scene.add(ground);

      // Particles
      const particleCount = 1000;
      const particlesGeometry = new THREE.BufferGeometry();
      const positions = new Float32Array(particleCount * 3);
      const colors = new Float32Array(particleCount * 3);
      for (let i = 0; i < particleCount; i++) {
        const i3 = i * 3;
        positions[i3] = (Math.random() - 0.5) * 40;
        positions[i3 + 1] = (Math.random() - 0.5) * 20;
        positions[i3 + 2] = (Math.random() - 0.5) * 40;
        colors[i3] = 0.8 + Math.random() * 0.2;
        colors[i3 + 1] = 0.5 + Math.random() * 0.3;
        colors[i3 + 2] = 0.4 + Math.random() * 0.2;
      }
      particlesGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
      particlesGeometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
      const particlesMaterial = new THREE.PointsMaterial({
        size: 0.1,
        vertexColors: true,
        transparent: true,
        opacity: 0.7
      });
      particles = new THREE.Points(particlesGeometry, particlesMaterial);
      scene.add(particles);

      // Animation
      clock = new THREE.Clock();
      time = 0;
      frameCount = 0;
      lastFpsUpdate = 0;
      setLoading(false);

      function animate() {
        const delta = clock.getDelta();
        time += delta * animationSpeed;
        frameCount++;
        if (time - lastFpsUpdate >= 1) {
          setFps(Math.round(frameCount / (time - lastFpsUpdate)));
          frameCount = 0;
          lastFpsUpdate = time;
        }
        // Clay deformation
        const vertices = geometry.attributes.position.array;
        const count = vertices.length / 3;
        for (let i = 0; i < count; i++) {
          const i3 = i * 3;
          const x = originalVertices[i3];
          const y = originalVertices[i3 + 1];
          const z = originalVertices[i3 + 2];
          const wave1 = deformIntensity * 0.3 * Math.sin(time * 0.5 + x * 1.5 + y * 0.7 + z * 1.2);
          const wave2 = deformIntensity * 0.2 * Math.sin(time * 0.7 + x * 0.8 + y * 1.3 + z * 0.9);
          const wave3 = deformIntensity * 0.1 * Math.sin(time * 1.2 + x * 1.1 + y * 0.5 + z * 1.4);
          const scale = 1.0 + (wave1 + wave2 + wave3) * 0.15;
          vertices[i3] = x * scale;
          vertices[i3 + 1] = y * scale;
          vertices[i3 + 2] = z * scale;
        }
        geometry.attributes.position.needsUpdate = true;
        geometry.computeVertexNormals();
        clayMesh.rotation.x = time * 0.1;
        clayMesh.rotation.y = time * 0.2;
        particles.rotation.y = time * 0.01;
        controls.update();
        if (renderer && renderer.renderAsync) {
          // Use renderAsync for WebGPU
          renderer.renderAsync(scene, camera);
        } else {
          // Fallback to standard render for WebGL
          renderer.render(scene, camera);
        }
        animationFrameId = requestAnimationFrame(animate);
      }
      animate();

      // Resize
      function onResize() {
        camera.aspect = window.innerWidth / window.innerHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(window.innerWidth, window.innerHeight);
      }
      window.addEventListener('resize', onResize);

      // Cleanup
      return () => {
        isMounted = false;
        cancelAnimationFrame(animationFrameId);
        window.removeEventListener('resize', onResize);
        if (renderer && renderer.domElement && mountRef.current) {
          mountRef.current.removeChild(renderer.domElement);
        }
      };
    })();

    return () => {
      isMounted = false;
    };
  }, [deformIntensity, animationSpeed]);

  // Controls
  return (
    <ClayContainer ref={mountRef}>
      <Overlay>
        <h1>Realistic Clay Animation</h1>
        <p>WebGPU + Three.js simulation of organic clay material with dynamic deformation</p>
      </Overlay>
      <FPSCounter>{fps} FPS</FPSCounter>
      <Controls>
        <ControlGroup>
          <label htmlFor="deformIntensity">Deformation Intensity</label>
          <input
            type="range"
            id="deformIntensity"
            min={0.1}
            max={1.0}
            step={0.1}
            value={deformIntensity}
            onChange={e => setDeformIntensity(Number(e.target.value))}
          />
        </ControlGroup>
        <ControlGroup>
          <label htmlFor="animationSpeed">Animation Speed</label>
          <input
            type="range"
            id="animationSpeed"
            min={0.2}
            max={2.0}
            step={0.1}
            value={animationSpeed}
            onChange={e => setAnimationSpeed(Number(e.target.value))}
          />
        </ControlGroup>
        <button
          className="btn"
          style={{
            background: '#dd8866',
            color: 'white',
            borderRadius: 20,
            padding: '8px 15px',
            fontWeight: 600
          }}
          onClick={() => setDeformIntensity(0.5)}
        >
          Reset Clay
        </button>
      </Controls>
      {loading && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            height: '100%',
            background: '#0a0a1a',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            flexDirection: 'column',
            zIndex: 1000,
            transition: 'opacity 1s ease'
          }}
        >
          <div
            className="spinner"
            style={{
              width: 50,
              height: 50,
              border: '5px solid rgba(255,255,255,0.1)',
              borderRadius: '50%',
              borderTop: '5px solid #ff9966',
              animation: 'spin 1s linear infinite',
              marginBottom: 20
            }}
          />
          <p style={{ color: '#e0e0ff', fontSize: '1.2rem', maxWidth: 300, textAlign: 'center' }}>
            Initializing WebGPU renderer and clay simulation...
          </p>
        </div>
      )}
    </ClayContainer>
  );
};

export default ClayAnimation;
