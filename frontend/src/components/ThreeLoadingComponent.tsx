import { useState, useEffect } from 'react';

interface LoadingComponentProps {
  loadingText?: string;
  progress?: number;
  showProgress?: boolean;
  className?: string;
}

export function ThreeLoadingComponent({
  loadingText = 'Loading 3D Architecture...',
  progress = 0,
  showProgress = true,
  className = ''
}: LoadingComponentProps) {
  const [dots, setDots] = useState('');

  // Animated loading dots
  useEffect(() => {
    const interval = setInterval(() => {
      setDots(prev => (prev.length >= 3 ? '' : prev + '.'));
    }, 500);

    return () => clearInterval(interval);
  }, []);

  return (
    <div
      className={`three-loading ${className}`}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        background: 'linear-gradient(135deg, #0a0a1a 0%, #1a1a2e 50%, #0a0a1a 100%)',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        color: '#00ff88',
        fontFamily: 'monospace',
        zIndex: 1000
      }}
    >
      {/* Main loading spinner */}
      <div
        style={{
          width: '60px',
          height: '60px',
          border: '3px solid rgba(0, 255, 136, 0.3)',
          borderTop: '3px solid #00ff88',
          borderRadius: '50%',
          animation: 'spin 1s linear infinite',
          marginBottom: '20px'
        }}
      />

      {/* Loading text */}
      <div
        style={{
          fontSize: '18px',
          fontWeight: 'bold',
          marginBottom: '10px',
          textAlign: 'center'
        }}
      >
        {loadingText}
        {dots}
      </div>

      {/* Progress bar */}
      {showProgress && (
        <div
          style={{
            width: '300px',
            height: '4px',
            background: 'rgba(0, 255, 136, 0.2)',
            borderRadius: '2px',
            overflow: 'hidden',
            marginBottom: '15px'
          }}
        >
          <div
            style={{
              width: `${progress}%`,
              height: '100%',
              background: 'linear-gradient(90deg, #00ff88, #00aaff)',
              borderRadius: '2px',
              transition: 'width 0.3s ease'
            }}
          />
        </div>
      )}

      {/* Loading steps */}
      <div
        style={{
          fontSize: '12px',
          color: 'rgba(0, 255, 136, 0.7)',
          textAlign: 'center',
          lineHeight: '1.4'
        }}
      >
        <div>🚀 Initializing OVASABI Platform</div>
        <div>📦 Loading Three.js Modules</div>
        <div>🎨 Preparing WebGPU Renderers</div>
        <div>🔗 Connecting to Services Mesh</div>
      </div>

      {/* Particle background effect */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: '100%',
          background: `
            radial-gradient(2px 2px at 20px 30px, rgba(0, 255, 136, 0.3), transparent),
            radial-gradient(2px 2px at 40px 70px, rgba(0, 170, 255, 0.3), transparent),
            radial-gradient(1px 1px at 90px 40px, rgba(255, 107, 107, 0.3), transparent),
            radial-gradient(1px 1px at 130px 80px, rgba(138, 43, 226, 0.3), transparent),
            radial-gradient(2px 2px at 160px 30px, rgba(255, 165, 0, 0.3), transparent)
          `,
          backgroundRepeat: 'repeat',
          backgroundSize: '200px 100px',
          animation: 'float 6s ease-in-out infinite',
          zIndex: -1
        }}
      />

      <style>
        {`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
          
          @keyframes float {
            0%, 100% { transform: translateY(0px); }
            50% { transform: translateY(-10px); }
          }
        `}
      </style>
    </div>
  );
}

// Error boundary component for Three.js loading failures
interface ThreeErrorFallbackProps {
  error: Error;
  retry?: () => void;
}

export function ThreeErrorFallback({ error, retry }: ThreeErrorFallbackProps) {
  return (
    <div
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        background: 'linear-gradient(135deg, #1a0a0a 0%, #2e1a1a 50%, #1a0a0a 100%)',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        color: '#ff6b6b',
        fontFamily: 'monospace',
        zIndex: 1000
      }}
    >
      <div
        style={{
          fontSize: '48px',
          marginBottom: '20px'
        }}
      >
        ⚠️
      </div>

      <div
        style={{
          fontSize: '18px',
          fontWeight: 'bold',
          marginBottom: '10px',
          textAlign: 'center'
        }}
      >
        Failed to Load 3D Architecture
      </div>

      <div
        style={{
          fontSize: '12px',
          color: 'rgba(255, 107, 107, 0.7)',
          textAlign: 'center',
          marginBottom: '20px',
          maxWidth: '400px',
          lineHeight: '1.4'
        }}
      >
        {error.message || 'An error occurred while loading Three.js modules'}
      </div>

      {retry && (
        <button
          onClick={retry}
          style={{
            padding: '10px 20px',
            background: 'rgba(255, 107, 107, 0.2)',
            border: '1px solid #ff6b6b',
            borderRadius: '5px',
            color: '#ff6b6b',
            fontFamily: 'monospace',
            fontSize: '12px',
            cursor: 'pointer'
          }}
        >
          Retry Loading
        </button>
      )}
    </div>
  );
}
