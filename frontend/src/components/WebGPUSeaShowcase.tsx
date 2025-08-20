import { useEffect, useRef, useState } from 'react';
import { loadAllThreeModules, threeLoadingManager } from '../lib/three';
import { ThreeLoadingComponent, ThreeErrorFallback } from './ThreeLoadingComponent';

// GLSL shaders (adapted for Three.js)
const vertexShader = `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const fragmentShader = `
  uniform float iGlobalTime;
  uniform vec2 iResolution;
  varying vec2 vUv;

  const int NUM_STEPS = 8;
  const float PI     = 3.1415;
  const float EPSILON  = 1e-3;
  float EPSILON_NRM  = 0.1 / iResolution.x;

  // sea variables
  const int ITER_GEOMETRY = 3;
  const int ITER_FRAGMENT = 5;
  const float SEA_HEIGHT = 0.6;
  const float SEA_CHOPPY = 1.0;
  const float SEA_SPEED = 1.0;
  const float SEA_FREQ = 0.16;
  const vec3 SEA_BASE = vec3(0.1,0.19,0.22);
  const vec3 SEA_WATER_COLOR = vec3(0.8,0.9,0.6);
  float SEA_TIME = iGlobalTime * SEA_SPEED;
  mat2 octave_m = mat2(1.6,1.2,-1.2,1.6);

  mat3 fromEuler(vec3 ang) {
    vec2 a1 = vec2(sin(ang.x),cos(ang.x));
    vec2 a2 = vec2(sin(ang.y),cos(ang.y));
    vec2 a3 = vec2(sin(ang.z),cos(ang.z));
    mat3 m;
    m[0] = vec3(
      a1.y*a3.y+a1.x*a2.x*a3.x,
      a1.y*a2.x*a3.x+a3.y*a1.x,
      -a2.y*a3.x
    );
    m[1] = vec3(-a2.y*a1.x,a1.y*a2.y,a2.x);
    m[2] = vec3(
      a3.y*a1.x*a2.x+a1.y*a3.x,
      a1.x*a3.x-a1.y*a3.y*a2.x,
      a2.y*a3.y
    );
    return m;
  }

  float hash( vec2 p ) {
    float h = dot(p,vec2(127.1,311.7));  
    return fract(sin(h)*43758.5453123);
  }

  float noise( in vec2 p ) {
    vec2 i = floor(p);
    vec2 f = fract(p);  
    vec2 u = f * f * (3.0 - 2.0 * f);
    return -1.0 + 2.0 * mix(
      mix(
        hash(i + vec2(0.0,0.0)
      ), 
      hash(i + vec2(1.0,0.0)), u.x),
      mix(hash(i + vec2(0.0,1.0) ), 
      hash(i + vec2(1.0,1.0) ), u.x), 
      u.y
    );
  }

  float diffuse(vec3 n,vec3 l,float p) {
    return pow(dot(n,l) * 0.4 + 0.6,p);
  }

  float specular(vec3 n,vec3 l,vec3 e,float s) {    
    float nrm = (s + 8.0) / (3.1415 * 8.0);
    return pow(max(dot(reflect(e,n),l),0.0),s) * nrm;
  }

  vec3 getSkyColor(vec3 e) {
    e.y = max(e.y, 0.0);
    vec3 ret;
    ret.x = pow(1.0 - e.y, 2.0);
    ret.y = 1.0 - e.y;
    ret.z = 0.6+(1.0 - e.y) * 0.4;
    return ret;
  }

  float sea_octave(vec2 uv, float choppy) {
    uv += noise(uv);         
    vec2 wv = 1.0 - abs(sin(uv));
    vec2 swv = abs(cos(uv));    
    wv = mix(wv, swv, wv);
    return pow(1.0 - pow(wv.x * wv.y, 0.65), choppy);
  }

  float map(vec3 p) {
    float freq = SEA_FREQ;
    float amp = SEA_HEIGHT;
    float choppy = SEA_CHOPPY;
    vec2 uv = p.xz; 
    uv.x *= 0.75;

    float d, h = 0.0;    
    for(int i = 0; i < ITER_GEOMETRY; i++) {        
      d = sea_octave((uv + SEA_TIME) * freq, choppy);
      d += sea_octave((uv - SEA_TIME) * freq, choppy);
      h += d * amp;        
      uv *= octave_m;
      freq *= 1.9; 
      amp *= 0.22;
      choppy = mix(choppy, 1.0, 0.2);
    }
    return p.y - h;
  }

  float map_detailed(vec3 p) {
      float freq = SEA_FREQ;
      float amp = SEA_HEIGHT;
      float choppy = SEA_CHOPPY;
      vec2 uv = p.xz;
      uv.x *= 0.75;

      float d, h = 0.0;    
      for(int i = 0; i < ITER_FRAGMENT; i++) {        
        d = sea_octave((uv+SEA_TIME) * freq, choppy);
        d += sea_octave((uv-SEA_TIME) * freq, choppy);
        h += d * amp;        
        uv *= octave_m;
        freq *= 1.9; 
        amp *= 0.22;
        choppy = mix(choppy,1.0,0.2);
      }
      return p.y - h;
  }

  vec3 getSeaColor(
    vec3 p,
    vec3 n, 
    vec3 l, 
    vec3 eye, 
    vec3 dist
  ) {  
    float fresnel = 1.0 - max(dot(n,-eye),0.0);
    fresnel = pow(fresnel,3.0) * 0.65;

    vec3 reflected = getSkyColor(reflect(eye,n));    
    vec3 refracted = SEA_BASE + diffuse(n,l,80.0) * SEA_WATER_COLOR * 0.12; 

    vec3 color = mix(refracted,reflected,fresnel);

    float atten = max(1.0 - dot(dist,dist) * 0.001, 0.0);
    color += SEA_WATER_COLOR * (p.y - SEA_HEIGHT) * 0.18 * atten;

    color += vec3(specular(n,l,eye,60.0));

    return color;
  }

  // tracing
  vec3 getNormal(vec3 p, float eps) {
    vec3 n;
    n.y = map_detailed(p);    
    n.x = map_detailed(vec3(p.x+eps,p.y,p.z)) - n.y;
    n.z = map_detailed(vec3(p.x,p.y,p.z+eps)) - n.y;
    n.y = eps;
    return normalize(n);
  }

  float heightMapTracing(vec3 ori, vec3 dir, out vec3 p) {  
    float tm = 0.0;
    float tx = 1000.0;    
    float hx = map(ori + dir * tx);

    if(hx > 0.0) {
      return tx;   
    }

    float hm = map(ori + dir * tm);    
    float tmid = 0.0;
    for(int i = 0; i < NUM_STEPS; i++) {
      tmid = mix(tm,tx, hm/(hm-hx));                   
      p = ori + dir * tmid;                   
      float hmid = map(p);
      if(hmid < 0.0) {
        tx = tmid;
        hx = hmid;
      } else {
        tm = tmid;
        hm = hmid;
       }
    }
    return tmid;
  }

  void main() {
    vec2 uv = vUv;
    vec2 fragCoord = uv * iResolution;
    uv = uv * 2.0 - 1.0;
    uv.x *= iResolution.x / iResolution.y;    
    float time = iGlobalTime * 0.3;

    // ray
    vec3 ang = vec3(
      sin(time*3.0)*0.1,sin(time)*0.2+0.3,time
    );    
    vec3 ori = vec3(0.0,3.5,time*5.0);
    vec3 dir = normalize(
      vec3(uv.xy,-2.0)
    );
    dir.z += length(uv) * 0.15;
    dir = normalize(dir);

    // tracing
    vec3 p;
    heightMapTracing(ori,dir,p);
    vec3 dist = p - ori;
    vec3 n = getNormal(
      p,
      dot(dist,dist) * EPSILON_NRM
    );
    vec3 light = normalize(vec3(0.0,1.0,0.8)); 

    // color
    vec3 color = mix(
      getSkyColor(dir),
      getSeaColor(p,n,light,dir,dist),
      pow(smoothstep(0.0,-0.05,dir.y),0.3)
    );

    // post
    gl_FragColor = vec4(pow(color,vec3(0.75)), 1.0);
  }
`;

export default function WebGPUSeaShowcase() {
  const mountRef = useRef<HTMLDivElement | null>(null);
  const [loading, setLoading] = useState(true);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<Error | null>(null);
  const [modules, setModules] = useState<any>(null);

  useEffect(() => {
    const unsubscribe = threeLoadingManager.subscribe(p => {
      setProgress(p.progress);
      setLoading(p.stage !== 'complete');
      if (p.error) setError(new Error(p.error));
    });
    loadAllThreeModules().then(setModules).catch(setError);
    return () => unsubscribe();
  }, []);

  useEffect(() => {
    if (!modules || error || !mountRef.current) return;
    const { core: THREE, renderers, addons } = modules;
    const scene = new THREE.Scene();
    scene.fog = new THREE.Fog(0x222244, 100, 400);
    const camera = new THREE.PerspectiveCamera(45, window.innerWidth / window.innerHeight, 1, 2000);
    camera.position.set(0, 40, 120);
    scene.add(camera);

    // Sea shader material
    const uniforms = {
      iGlobalTime: { value: 0.0 },
      iResolution: { value: { x: window.innerWidth, y: window.innerHeight } }
    };
    let renderer;
    if (renderers.WebGPURenderer && renderers.webgpuAvailable) {
      renderer = new renderers.WebGPURenderer({ antialias: true });
    } else {
      renderer = new THREE.WebGLRenderer({ antialias: true });
    }
    renderer.setClearColor(0x222244, 1);
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.shadowMap.enabled = true;
    mountRef.current.appendChild(renderer.domElement);

    let seaMaterial;
    let seaMesh;
    const seaGeometry = new THREE.PlaneGeometry(200, 200, 128, 128);
    // If using WebGPURenderer, fallback to MeshStandardMaterial for compatibility
    if (renderers.WebGPURenderer && renderer instanceof renderers.WebGPURenderer) {
      seaMaterial = new THREE.MeshStandardMaterial({
        color: 0x1a3340,
        metalness: 0.7,
        roughness: 0.3,
        transparent: true,
        opacity: 0.95,
        emissive: 0x00aaff,
        emissiveIntensity: 0.2
      });
      seaMesh = new THREE.Mesh(seaGeometry, seaMaterial);
      seaMesh.rotation.x = -Math.PI / 2;
      seaMesh.position.y = 0;
      scene.add(seaMesh);
    } else {
      seaMaterial = new THREE.ShaderMaterial({
        uniforms,
        vertexShader,
        fragmentShader,
        side:
          THREE.DoubleSide !== undefined
            ? THREE.DoubleSide
            : THREE.Side
              ? THREE.Side.DoubleSide
              : 2,
        transparent: true
      });
      seaMesh = new THREE.Mesh(seaGeometry, seaMaterial);
      seaMesh.rotation.x = -Math.PI / 2;
      seaMesh.position.y = 0;
      scene.add(seaMesh);
    }

    // Add architectural elements above the sea
    for (let i = 0; i < 5; i++) {
      const buildingGeo = new THREE.BoxGeometry(10, 30, 10);
      const buildingMat = new THREE.MeshPhysicalMaterial({
        color: 0x222244,
        metalness: 0.7,
        roughness: 0.2,
        clearcoat: 0.5,
        transparent: true,
        opacity: 0.95,
        emissive: 0x00aaff,
        emissiveIntensity: 0.3
      });
      const building = new THREE.Mesh(buildingGeo, buildingMat);
      building.position.set(
        Math.random() * 120 - 60,
        20 + Math.random() * 10,
        Math.random() * 120 - 60
      );
      scene.add(building);
    }
    // Neon bridge
    const bridgeGeo = new THREE.BoxGeometry(60, 2, 8);
    const bridgeMat = new THREE.MeshStandardMaterial({
      color: 0x888888,
      metalness: 0.5,
      roughness: 0.6,
      emissive: 0x00ff88,
      emissiveIntensity: 0.2
    });
    const bridge = new THREE.Mesh(bridgeGeo, bridgeMat);
    bridge.position.set(0, 10, 0);
    scene.add(bridge);

    // Neon ground grid
    const gridGeo = new THREE.PlaneGeometry(200, 200, 32, 32);
    const gridMat = new THREE.MeshBasicMaterial({
      color: 0x00ff88,
      wireframe: true,
      transparent: true,
      opacity: 0.15
    });
    const grid = new THREE.Mesh(gridGeo, gridMat);
    grid.rotation.x = -Math.PI / 2;
    grid.position.y = 0.1;
    scene.add(grid);

    // Lighting
    const ambient = new THREE.AmbientLight(0x222244, 1.2);
    scene.add(ambient);
    const directional = new THREE.DirectionalLight(0xffffff, 1.5);
    directional.position.set(60, 100, 40);
    directional.castShadow = true;
    scene.add(directional);

    if (renderers.WebGPURenderer && renderers.webgpuAvailable) {
      renderer = new renderers.WebGPURenderer({ antialias: true });
    } else {
      renderer = new THREE.WebGLRenderer({ antialias: true });
    }
    renderer.setClearColor(0x222244, 1);
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.shadowMap.enabled = true;
    mountRef.current.appendChild(renderer.domElement);

    // Controls
    if (addons.OrbitControls) {
      const controls = new addons.OrbitControls(camera, renderer.domElement);
      controls.minDistance = 50;
      controls.maxDistance = 400;
    }

    // Animate
    const clock = new THREE.Clock();
    const animate = () => {
      uniforms.iGlobalTime.value += clock.getDelta();
      if (
        renderers.WebGPURenderer &&
        renderer instanceof renderers.WebGPURenderer &&
        typeof renderer.renderAsync === 'function'
      ) {
        renderer.renderAsync(scene, camera);
      } else {
        renderer.render(scene, camera);
      }
      requestAnimationFrame(animate);
    };
    animate();

    // Responsive resize
    const handleResize = () => {
      camera.aspect = window.innerWidth / window.innerHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(window.innerWidth, window.innerHeight);
      uniforms.iResolution.value.x = window.innerWidth;
      uniforms.iResolution.value.y = window.innerHeight;
    };
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      if (renderer) {
        renderer.dispose();
        if (renderer.domElement && renderer.domElement.parentNode)
          renderer.domElement.parentNode.removeChild(renderer.domElement);
      }
    };
  }, [modules, error]);

  if (error) return <ThreeErrorFallback error={error} retry={() => window.location.reload()} />;
  if (loading)
    return (
      <ThreeLoadingComponent loadingText="Loading WebGPU Sea Showcase..." progress={progress} />
    );
  return (
    <div
      ref={mountRef}
      style={{ width: '100vw', height: '100vh', position: 'relative', zIndex: 1 }}
    />
  );
}
