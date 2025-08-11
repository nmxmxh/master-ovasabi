import { useEffect } from 'react';
import styled, { keyframes } from 'styled-components';
import { useGPUCapabilities } from '../store/global';

// Animations
const slideIn = keyframes`
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`;

const pulse = keyframes`
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
`;

// Styled Components following project patterns
const Style = {
  Container: styled.div<{ className?: string }>`
    background: rgba(255, 255, 255, 0.98);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
    backdrop-filter: blur(15px);
    -webkit-backdrop-filter: blur(15px);
    animation: ${slideIn} 0.6s cubic-bezier(0.4, 0, 0.2, 1);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

    &:hover {
      box-shadow: 0 12px 48px rgba(0, 0, 0, 0.15);
      transform: translateY(-2px);
    }

    @media (max-width: 768px) {
      padding: 16px;
      margin: 8px;
    }
  `,

  LoadingContainer: styled.div`
    background: rgba(243, 244, 246, 0.8);
    border-radius: 8px;
    padding: 32px;
    text-align: center;
    color: #6b7280;
    animation: ${slideIn} 0.4s ease-out;
  `,

  LoadingSpinner: styled.div`
    width: 24px;
    height: 24px;
    border: 2px solid #e5e7eb;
    border-top: 2px solid #3b82f6;
    border-radius: 50%;
    margin: 0 auto 16px;
    animation: spin 1s linear infinite;

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }
  `,

  Header: styled.div`
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  `,

  Title: styled.h3`
    font-size: 20px;
    font-weight: 600;
    color: #1f2937;
    margin: 0;
  `,

  RefreshButton: styled.button`
    padding: 8px 16px;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

    &:hover {
      background: #2563eb;
      transform: translateY(-1px);
      box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
    }

    &:active {
      transform: translateY(0);
    }
  `,

  PerformanceSection: styled.div`
    margin-bottom: 32px;
  `,

  PerformanceHeader: styled.div`
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  `,

  PerformanceLabel: styled.span`
    font-size: 14px;
    font-weight: 500;
    color: #374151;
  `,

  PerformanceBadge: styled.span<{ $color: string }>`
    padding: 4px 12px;
    border-radius: 16px;
    color: white;
    font-size: 12px;
    font-weight: 600;
    background: ${props => props.$color};
    animation: ${pulse} 2s infinite;
  `,

  ProgressBar: styled.div`
    width: 100%;
    height: 8px;
    background: #e5e7eb;
    border-radius: 4px;
    overflow: hidden;
    position: relative;
  `,

  ProgressFill: styled.div<{ $width: number }>`
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #06b6d4);
    border-radius: 4px;
    transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
    width: ${props => props.$width}%;
    position: relative;

    &::after {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
      animation: shimmer 2s infinite;
    }

    @keyframes shimmer {
      0% {
        transform: translateX(-100%);
      }
      100% {
        transform: translateX(100%);
      }
    }
  `,

  StatusGrid: styled.div`
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 16px;
    margin-bottom: 32px;

    @media (max-width: 768px) {
      grid-template-columns: 1fr;
      gap: 12px;
    }
  `,

  StatusCard: styled.div`
    background: rgba(249, 250, 251, 0.8);
    border: 1px solid rgba(0, 0, 0, 0.05);
    border-radius: 8px;
    padding: 16px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

    &:hover {
      background: rgba(249, 250, 251, 1);
      border-color: rgba(59, 130, 246, 0.2);
      transform: translateY(-1px);
    }
  `,

  StatusHeader: styled.div`
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  `,

  StatusIndicator: styled.div<{ $available: boolean }>`
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: ${props => (props.$available ? '#10b981' : '#ef4444')};
    box-shadow: 0 0 0 2px
      ${props => (props.$available ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)')};
    animation: ${props => (props.$available ? pulse : 'none')} 2s infinite;
  `,

  StatusTitle: styled.span`
    font-weight: 600;
    font-size: 14px;
    color: #1f2937;
  `,

  StatusText: styled.p`
    font-size: 12px;
    color: #6b7280;
    margin: 0 0 4px 0;
  `,

  StatusDetail: styled.p`
    font-size: 10px;
    color: #9ca3af;
    margin: 0;
  `,

  RecommendationSection: styled.div`
    margin-bottom: 32px;
  `,

  SectionTitle: styled.h4`
    font-weight: 500;
    font-size: 14px;
    color: #374151;
    margin-bottom: 16px;
  `,

  RecommendationContainer: styled.div`
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  `,

  RendererBadge: styled.span<{ $type: string }>`
    padding: 6px 16px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 600;
    background: ${props =>
      props.$type === 'webgpu' ? 'rgba(147, 51, 234, 0.1)' : 'rgba(59, 130, 246, 0.1)'};
    color: ${props => (props.$type === 'webgpu' ? '#7c3aed' : '#2563eb')};
    border: 1px solid
      ${props => (props.$type === 'webgpu' ? 'rgba(147, 51, 234, 0.2)' : 'rgba(59, 130, 246, 0.2)')};
  `,

  RecommendationReason: styled.span`
    font-size: 12px;
    color: #6b7280;
    font-style: italic;
  `,

  InfoSection: styled.div`
    margin-bottom: 24px;
  `,

  InfoCard: styled.div`
    background: rgba(249, 250, 251, 0.8);
    border: 1px solid rgba(0, 0, 0, 0.05);
    border-radius: 8px;
    padding: 16px;
  `,

  InfoGrid: styled.div`
    font-size: 12px;
    line-height: 1.6;
    color: #374151;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;

    > div {
      margin-bottom: 4px;
    }

    span {
      font-weight: 600;
    }
  `,

  Timestamp: styled.div`
    font-size: 12px;
    color: #9ca3af;
    text-align: center;
    margin-top: 16px;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  `
};

interface GPUDashboardProps {
  className?: string;
}

// Utility function to safely render any value as a string
const safeRender = (value: any): string => {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  if (typeof value === 'object') {
    try {
      // For objects, try to extract meaningful information
      if (value.vendor || value.device || value.description) {
        const parts = [];
        if (value.vendor) parts.push(value.vendor);
        if (value.device) parts.push(value.device);
        if (value.description && value.description !== value.device) parts.push(value.description);
        return parts.join(' - ');
      }
      return JSON.stringify(value);
    } catch {
      return '[Object]';
    }
  }
  return String(value);
};

export function GPUDashboard({ className = '' }: GPUDashboardProps) {
  const {
    gpuCapabilities,
    wasmGPUBridge,
    refreshGPUCapabilities,
    isWebGPUAvailable,
    isWebGLAvailable,
    recommendedRenderer,
    performanceScore,
    detectedAt
  } = useGPUCapabilities();

  // Auto-refresh GPU capabilities when component mounts
  useEffect(() => {
    if (!gpuCapabilities) {
      refreshGPUCapabilities();
    }
  }, [gpuCapabilities, refreshGPUCapabilities]);

  if (!gpuCapabilities) {
    return (
      <Style.Container className={className}>
        <Style.LoadingContainer>
          <Style.LoadingSpinner />
          <p>Detecting GPU capabilities...</p>
          <Style.RefreshButton onClick={refreshGPUCapabilities}>Refresh</Style.RefreshButton>
        </Style.LoadingContainer>
      </Style.Container>
    );
  }

  const getPerformanceBadge = (score: number) => {
    if (score >= 80) return { text: 'Excellent', color: '#10b981' };
    if (score >= 60) return { text: 'Good', color: '#f59e0b' };
    if (score >= 40) return { text: 'Fair', color: '#f97316' };
    return { text: 'Poor', color: '#ef4444' };
  };

  const performanceBadge = getPerformanceBadge(performanceScore);

  return (
    <Style.Container className={className}>
      <Style.Header>
        <Style.Title>GPU Capabilities</Style.Title>
        <Style.RefreshButton onClick={refreshGPUCapabilities}>Refresh</Style.RefreshButton>
      </Style.Header>

      {/* Performance Score */}
      <Style.PerformanceSection>
        <Style.PerformanceHeader>
          <Style.PerformanceLabel>Performance Score:</Style.PerformanceLabel>
          <Style.PerformanceBadge $color={performanceBadge.color}>
            {performanceBadge.text} ({performanceScore}/100)
          </Style.PerformanceBadge>
        </Style.PerformanceHeader>
        <Style.ProgressBar>
          <Style.ProgressFill $width={performanceScore} />
        </Style.ProgressBar>
      </Style.PerformanceSection>

      {/* Backend Availability */}
      <Style.StatusGrid>
        <Style.StatusCard>
          <Style.StatusHeader>
            <Style.StatusIndicator $available={isWebGPUAvailable} />
            <Style.StatusTitle>WebGPU</Style.StatusTitle>
          </Style.StatusHeader>
          <Style.StatusText>{isWebGPUAvailable ? 'Available' : 'Not Available'}</Style.StatusText>
          {gpuCapabilities.webgpu?.adapter && (
            <Style.StatusDetail>{safeRender(gpuCapabilities.webgpu.adapter)}</Style.StatusDetail>
          )}
        </Style.StatusCard>

        <Style.StatusCard>
          <Style.StatusHeader>
            <Style.StatusIndicator $available={isWebGLAvailable} />
            <Style.StatusTitle>WebGL</Style.StatusTitle>
          </Style.StatusHeader>
          <Style.StatusText>{isWebGLAvailable ? 'Available' : 'Not Available'}</Style.StatusText>
          {gpuCapabilities.webgl?.version && (
            <Style.StatusDetail>{safeRender(gpuCapabilities.webgl.version)}</Style.StatusDetail>
          )}
        </Style.StatusCard>
      </Style.StatusGrid>

      {/* Recommended Renderer */}
      <Style.RecommendationSection>
        <Style.SectionTitle>Recommended Renderer</Style.SectionTitle>
        <Style.RecommendationContainer>
          <Style.RendererBadge $type={recommendedRenderer}>
            {recommendedRenderer.toUpperCase()}
          </Style.RendererBadge>
          <Style.RecommendationReason>
            {gpuCapabilities.three?.reasonForRecommendation ||
              'Auto-detected based on capabilities'}
          </Style.RecommendationReason>
        </Style.RecommendationContainer>
      </Style.RecommendationSection>

      {/* GPU Information */}
      {gpuCapabilities.webgpu?.available && (
        <Style.InfoSection>
          <Style.SectionTitle>GPU Information</Style.SectionTitle>
          <Style.InfoCard>
            <Style.InfoGrid>
              {gpuCapabilities.webgpu.vendor && (
                <div>
                  <span>Vendor:</span> {safeRender(gpuCapabilities.webgpu.vendor)}
                </div>
              )}
              {gpuCapabilities.webgpu.architecture && (
                <div>
                  <span>Architecture:</span> {safeRender(gpuCapabilities.webgpu.architecture)}
                </div>
              )}
              {gpuCapabilities.webgpu.device && (
                <div>
                  <span>Device:</span> {safeRender(gpuCapabilities.webgpu.device)}
                </div>
              )}
            </Style.InfoGrid>
          </Style.InfoCard>
        </Style.InfoSection>
      )}

      {/* WASM Bridge Status */}
      {wasmGPUBridge && (
        <Style.InfoSection>
          <Style.SectionTitle>WASM Integration</Style.SectionTitle>
          <Style.InfoCard>
            <Style.InfoGrid>
              <div>
                <span>Backend:</span> {wasmGPUBridge.backend}
              </div>
              <div>
                <span>Status:</span> {wasmGPUBridge.initialized ? 'Initialized' : 'Not Initialized'}
              </div>
              <div>
                <span>Workers:</span> {wasmGPUBridge.workerCount}
              </div>
              <div>
                <span>Version:</span> {wasmGPUBridge.version}
              </div>
            </Style.InfoGrid>
          </Style.InfoCard>
        </Style.InfoSection>
      )}

      {/* Detection Timestamp */}
      {detectedAt && (
        <Style.Timestamp>Detected at: {new Date(detectedAt).toLocaleString()}</Style.Timestamp>
      )}
    </Style.Container>
  );
}

export default GPUDashboard;
