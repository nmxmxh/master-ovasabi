import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import styled from 'styled-components';
import {
  useEventHistory,
  useConnectionStatus,
  useMetadata,
  useGlobalStore,
  useCampaignState,
  useCampaignUpdates
} from './store/global';
import EnhancedParticleSystem from './components/EnhancedParticleSystem';

// Hook to poll window.__WASM_GLOBAL_METADATA
function useWasmGlobalMetadata(pollInterval = 1000) {
  const [wasmMetadata, setWasmMetadata] = useState<any>(null);
  useEffect(() => {
    function poll() {
      if (typeof window !== 'undefined' && window.__WASM_GLOBAL_METADATA) {
        setWasmMetadata(window.__WASM_GLOBAL_METADATA);
      }
    }
    poll();
    const interval = setInterval(poll, pollInterval);
    return () => {
      clearInterval(interval);
    };
  }, [pollInterval]);
  return wasmMetadata;
}

// Utility to generate a UUID (RFC4122 v4)
function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = crypto.getRandomValues(new Uint8Array(1))[0] % 16;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Enhanced logging function
function logStatus(label: string, value: any) {
  const time = new Date().toISOString();
  if (typeof value === 'object') {
    console.groupCollapsed(`[App][${time}] ${label}`);
    console.dir(value);
    console.groupEnd();
  } else {
    console.log(`[App][${time}] ${label}:`, value);
  }
}

// Styled Components following project patterns
const AppStyle = {
  Container: styled.div`
    padding: 20px;
    background: #f8fafc;
    min-height: 100vh;
    color: #1f2937;
  `,

  Title: styled.h1`
    color: #1f2937;
    font-size: 28px;
    font-weight: 600;
    margin-bottom: 24px;
    text-align: center;
  `,

  Section: styled.div<{ $marginTop?: string }>`
    margin-top: ${props => props.$marginTop || '40px'};
    border-top: 1px solid #e5e7eb;
    padding-top: 20px;
  `,

  SectionHeader: styled.div`
    display: flex;
    align-items: center;
    gap: 20px;
    margin-bottom: 20px;
  `,

  SectionTitle: styled.h3`
    margin: 0;
    color: #374151;
    font-size: 20px;
    font-weight: 500;
  `,

  ButtonGroup: styled.div`
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  `,

  Button: styled.button<{ $active?: boolean; $color?: string }>`
    padding: 8px 16px;
    background-color: ${props => (props.$active ? props.$color || '#3b82f6' : '#ffffff')};
    color: ${props => (props.$active ? '#ffffff' : props.$color || '#3b82f6')};
    border: 2px solid ${props => props.$color || '#3b82f6'};
    border-radius: 6px;
    cursor: pointer;
    font-weight: 600;
    font-size: 14px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

    &:hover {
      transform: translateY(-1px);
      box-shadow: 0 4px 12px ${props => `${props.$color || '#3b82f6'}40`};
    }

    &:active {
      transform: translateY(0);
    }
  `,

  CampaignGrid: styled.div`
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
    margin-top: 16px;
  `,

  CampaignCard: styled.div`
    background: rgba(255, 255, 255, 0.9);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: 8px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    min-width: 200px;

    &:hover {
      background: rgba(255, 255, 255, 1);
      border-color: rgba(59, 130, 246, 0.3);
      transform: translateY(-2px);
      box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
    }
  `,

  WarningText: styled.div`
    color: #dc2626;
    margin-top: 12px;
    font-size: 14px;
    font-weight: 500;
  `,

  PWAContainer: styled.div`
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 8px;
  `,

  PWAButton: styled.button<{ $type?: 'install' | 'update' }>`
    padding: 12px 16px;
    background: ${props => (props.$type === 'update' ? '#f59e0b' : '#6366f1')};
    color: white;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    font-size: 14px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 8px 20px rgba(0, 0, 0, 0.2);
      background: ${props => (props.$type === 'update' ? '#d97706' : '#4f46e5')};
    }

    &:active {
      transform: translateY(0);
    }
  `,

  PWAStatus: styled.div<{ $status: string }>`
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 999;
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    background: ${props => {
      switch (props.$status) {
        case 'registered':
          return '#10b981';
        case 'error':
          return '#ef4444';
        case 'updated':
          return '#f59e0b';
        default:
          return '#6b7280';
      }
    }};
    color: white;
    opacity: 0.9;
  `
};

function App() {
  // Zustand WASM/GPU function state
  console.log('[App] Component mounting/re-rendering at', new Date().toISOString());

  // State for particle system selection with automatic WebGPU detection
  const [particleSystem, setParticleSystem] = useState<'ghost' | 'enhanced' | 'architecture'>(
    'enhanced'
  );
  const [wasmWebgpuReady, setWasmWebgpuReady] = useState<boolean>(false);
  const initialSetupDone = useRef(false);

  // Detect WebGPU availability and WASM readiness
  useEffect(() => {
    // Check browser WebGPU support
    const checkWebGPU = () => {
      const hasWebGPU = typeof navigator !== 'undefined' && 'gpu' in navigator;
      console.log('[App] Browser WebGPU support detected:', hasWebGPU);
      return hasWebGPU;
    };

    // Check WASM WebGPU readiness
    const checkWasmWebGPU = () => {
      const wasmFunctions = {
        initWebGPU: typeof window.initWebGPU === 'function',
        runGPUCompute: typeof window.runGPUCompute === 'function',
        getGPUMetricsBuffer: typeof window.getGPUMetricsBuffer === 'function'
      };
      const wasmReady = Object.values(wasmFunctions).every(available => available);
      setWasmWebgpuReady(wasmReady);
      console.log('[App] WASM WebGPU functions available:', wasmReady, wasmFunctions);
      return wasmReady;
    };

    const browserWebGPU = checkWebGPU();
    const wasmWebGPU = checkWasmWebGPU();

    // Auto-select the best particle system based on capabilities
    if (!initialSetupDone.current) {
      if (browserWebGPU && wasmWebGPU) {
        console.log('[App] ✅ Full WebGPU + WASM support detected - using Enhanced system');
        if (particleSystem !== 'enhanced') setParticleSystem('enhanced');
      } else if (browserWebGPU) {
        console.log(
          '[App] ⚡ Browser WebGPU detected - using Enhanced system with WebGPU rendering'
        );
        if (particleSystem !== 'enhanced') setParticleSystem('enhanced');
      } else {
        console.log('[App] 🔄 WebGL fallback - using Enhanced system with WebGL');
        if (particleSystem !== 'enhanced') setParticleSystem('enhanced');
      }
      initialSetupDone.current = true;
    }

    // Recheck WASM functions periodically until they're available
    const recheckInterval = setInterval(() => {
      if (!wasmWebgpuReady) {
        const newWasmReady = checkWasmWebGPU();
        if (newWasmReady && browserWebGPU) {
          console.log(
            '[App] 🚀 WASM WebGPU now ready - Enhanced system upgraded to full WebGPU+WASM'
          );
          // Only trigger if not already enhanced to prevent unnecessary remounts
          if (particleSystem !== 'enhanced') setParticleSystem('enhanced');
          clearInterval(recheckInterval);
        }
      } else {
        clearInterval(recheckInterval);
      }
    }, 2000);

    return () => clearInterval(recheckInterval);
  }, []); // Run only once on mount

  // Fetch campaigns using WebSocket event pattern for real-time consistency
  const [campaigns, setCampaigns] = useState([
    { id: 0, slug: 'ovasabi_website', name: 'Ovasabi Website' }
  ]);
  const { emitEvent } = useGlobalStore();
  const { isConnected } = useConnectionStatus();
  const { metadata } = useMetadata();

  useEffect(() => {
    // Wait for WebSocket connection before requesting campaigns
    if (!isConnected) {
      return;
    }

    // Request campaign list through WebSocket event system
    emitEvent(
      {
        type: 'campaign:list:v1:requested',
        payload: {
          limit: 50,
          offset: 0
        },
        metadata: metadata
      },
      (response: any) => {
        logStatus('Campaign list response received', response);

        if (response.type === 'campaign:list:v1:success' && response.payload) {
          const campaignData = response.payload.campaigns || response.payload;

          if (Array.isArray(campaignData) && campaignData.length > 0) {
            // Defensive: ensure id, slug, name are present and fallback if missing
            setCampaigns(
              campaignData
                .map((c: any, idx: number) => ({
                  id: typeof c.id === 'number' ? c.id : idx,
                  slug: c.slug || `campaign_${idx}`,
                  name: c.title || c.name || c.slug || `Campaign ${idx + 1}`
                }))
                .sort((a, b) => a.id - b.id)
            );
          }
        } else if (response.type === 'campaign:list:v1:failed') {
          logStatus('Campaign list request failed, using fallback', response);
          setCampaigns([{ id: 0, slug: 'ovasabi_website', name: 'Ovasabi Website' }]);
        }
      }
    );

    // Fallback: if no response within 5 seconds, use default
    const fallbackTimer = setTimeout(() => {
      logStatus('Campaign list request timeout, using fallback', null);
      setCampaigns([{ id: 0, slug: 'ovasabi_website', name: 'Ovasabi Website' }]);
    }, 5000);

    return () => clearTimeout(fallbackTimer);
  }, [isConnected, emitEvent]);

  // Defensive: only use switchCampaign if present in store
  const switchCampaign = useGlobalStore(state =>
    typeof state.switchCampaign === 'function' ? state.switchCampaign : undefined
  );

  return (
    <AppStyle.Container>
      <EnhancedParticleSystem />
      <AppStyle.Title>OVASABI Campaign Management Demo</AppStyle.Title>
      <ConnectionStatus />
      <CampaignOperations />
      <SearchInterface />
      <EventHistory />
      <MetadataDisplay />
      <LiveWasmMetadataDisplay />

      {/* Active Campaign List */}
      <AppStyle.Section>
        <AppStyle.SectionTitle>Active Campaigns</AppStyle.SectionTitle>
        <AppStyle.CampaignGrid>
          {campaigns.map(c => (
            <AppStyle.CampaignCard
              key={c.id}
              onClick={() => switchCampaign && switchCampaign(c.id, c.slug)}
              style={{
                border: metadata.campaign?.campaignId === c.id ? '2px solid #007bff' : undefined,
                background: metadata.campaign?.campaignId === c.id ? '#e3f2fd' : undefined,
                fontWeight: metadata.campaign?.campaignId === c.id ? 'bold' : 'normal',
                cursor: switchCampaign ? 'pointer' : 'not-allowed',
                opacity: switchCampaign ? 1 : 0.5,
                boxShadow:
                  metadata.campaign?.campaignId === c.id ? '0 2px 8px #007bff22' : undefined
              }}
            >
              <span
                style={{
                  fontSize: '16px',
                  fontWeight: metadata.campaign?.campaignId === c.id ? 'bold' : 'normal'
                }}
              >
                {c.name}
              </span>
              {metadata.campaign?.campaignId === c.id && (
                <span style={{ color: '#007bff', marginLeft: '8px' }}>(Active)</span>
              )}
              <br />
              <span style={{ fontSize: '12px', color: '#888' }}>{c.slug}</span>
            </AppStyle.CampaignCard>
          ))}
        </AppStyle.CampaignGrid>
        {!switchCampaign && (
          <AppStyle.WarningText>
            Campaign switching is not available (store missing switchCampaign).
          </AppStyle.WarningText>
        )}
      </AppStyle.Section>
    </AppStyle.Container>
  );
}

// Campaign Operations Demo Component
function CampaignOperations() {
  const campaignState = useCampaignState();
  const { updateCampaign, updateCampaignFeatures, updateCampaignConfig } = useCampaignUpdates();
  const { metadata } = useMetadata();
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    feature: '',
    bannerText: '',
    customJson: ''
  });

  const currentCampaign = campaignState.state || {};
  const currentFeatures = (currentCampaign as any).features || metadata.campaign?.features || [];

  return (
    <div style={{ marginBottom: '30px' }}>
      <h3>🎯 Campaign Operations (Direct State Management)</h3>

      {/* Current Campaign State Display */}
      <div
        style={{
          background: '#f8f9fa',
          border: '1px solid #dee2e6',
          borderRadius: '8px',
          padding: '15px',
          marginBottom: '20px'
        }}
      >
        <h4 style={{ margin: '0 0 10px 0', color: '#495057' }}>Current Campaign State</h4>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '10px'
          }}
        >
          <div>
            <strong>Campaign ID:</strong> {metadata.campaign?.campaignId || 'N/A'}
            <br />
            <strong>Slug:</strong> {metadata.campaign?.slug || 'N/A'}
          </div>
          <div>
            <strong>Title:</strong> {(currentCampaign as any).title || 'N/A'}
            <br />
            <strong>Status:</strong> {(currentCampaign as any).status || 'N/A'}
          </div>
          <div>
            <strong>Features:</strong>{' '}
            {currentFeatures.length > 0 ? currentFeatures.join(', ') : 'None'}
          </div>
        </div>
        {(currentCampaign as any).ui_content && (
          <div style={{ marginTop: '10px' }}>
            <strong>UI Banner:</strong> {(currentCampaign as any).ui_content.banner || 'N/A'}
          </div>
        )}
      </div>

      {/* Campaign Update Operations */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '20px',
          marginBottom: '20px'
        }}
      >
        {/* Basic Campaign Update */}
        <div style={{ border: '1px solid #ddd', borderRadius: '8px', padding: '15px' }}>
          <h4 style={{ margin: '0 0 15px 0', color: '#007bff' }}>📝 Update Campaign Info</h4>
          <div style={{ marginBottom: '10px' }}>
            <input
              type="text"
              placeholder="Campaign title"
              value={formData.title}
              onChange={e => setFormData(prev => ({ ...prev, title: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                marginBottom: '8px'
              }}
            />
            <input
              type="text"
              placeholder="Campaign description"
              value={formData.description}
              onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px'
              }}
            />
          </div>
          <button
            onClick={() => {
              if (formData.title || formData.description) {
                const updates: any = {};
                if (formData.title) updates.title = formData.title;
                if (formData.description) updates.description = formData.description;
                updates.status = 'active';
                updates.last_updated = new Date().toISOString();

                updateCampaign(updates, response => {
                  console.log('Campaign update response:', response);
                });
                setFormData(prev => ({ ...prev, title: '', description: '' }));
              }
            }}
            disabled={!formData.title && !formData.description}
            style={{
              padding: '10px 15px',
              backgroundColor: '#007bff',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              width: '100%'
            }}
          >
            Update Campaign
          </button>
        </div>

        {/* Feature Management */}
        <div style={{ border: '1px solid #ddd', borderRadius: '8px', padding: '15px' }}>
          <h4 style={{ margin: '0 0 15px 0', color: '#28a745' }}>🎛️ Manage Features</h4>
          <div style={{ marginBottom: '10px' }}>
            <input
              type="text"
              placeholder="Feature name (e.g., analytics, notifications)"
              value={formData.feature}
              onChange={e => setFormData(prev => ({ ...prev, feature: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                marginBottom: '8px'
              }}
            />
          </div>
          <div style={{ display: 'flex', gap: '5px', marginBottom: '10px' }}>
            <button
              onClick={() => {
                if (formData.feature) {
                  updateCampaignFeatures([formData.feature], 'add');
                  setFormData(prev => ({ ...prev, feature: '' }));
                }
              }}
              disabled={!formData.feature}
              style={{
                flex: 1,
                padding: '8px',
                backgroundColor: '#28a745',
                color: 'white',
                border: 'none',
                borderRadius: '4px'
              }}
            >
              Add
            </button>
            <button
              onClick={() => {
                if (formData.feature) {
                  updateCampaignFeatures([formData.feature], 'remove');
                  setFormData(prev => ({ ...prev, feature: '' }));
                }
              }}
              disabled={!formData.feature}
              style={{
                flex: 1,
                padding: '8px',
                backgroundColor: '#dc3545',
                color: 'white',
                border: 'none',
                borderRadius: '4px'
              }}
            >
              Remove
            </button>
          </div>
          <div style={{ fontSize: '12px', color: '#666' }}>
            Current features: {currentFeatures.length > 0 ? currentFeatures.join(', ') : 'None'}
          </div>
        </div>
      </div>

      {/* UI Configuration */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
        {/* UI Content Update */}
        <div style={{ border: '1px solid #ddd', borderRadius: '8px', padding: '15px' }}>
          <h4 style={{ margin: '0 0 15px 0', color: '#ff6b35' }}>🎨 Update UI Content</h4>
          <div style={{ marginBottom: '10px' }}>
            <input
              type="text"
              placeholder="Banner text"
              value={formData.bannerText}
              onChange={e => setFormData(prev => ({ ...prev, bannerText: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px'
              }}
            />
          </div>
          <button
            onClick={() => {
              if (formData.bannerText) {
                updateCampaignConfig('ui_content', {
                  banner: formData.bannerText,
                  cta: 'Join the updated campaign!',
                  theme: 'modern',
                  updated_at: new Date().toISOString()
                });
                setFormData(prev => ({ ...prev, bannerText: '' }));
              }
            }}
            disabled={!formData.bannerText}
            style={{
              padding: '10px 15px',
              backgroundColor: '#ff6b35',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              width: '100%'
            }}
          >
            Update UI Content
          </button>
        </div>

        {/* Custom JSON Update */}
        <div style={{ border: '1px solid #ddd', borderRadius: '8px', padding: '15px' }}>
          <h4 style={{ margin: '0 0 15px 0', color: '#6f42c1' }}>⚙️ Custom Update</h4>
          <div style={{ marginBottom: '10px' }}>
            <textarea
              placeholder='{"custom_field": "value", "settings": {"enabled": true}}'
              value={formData.customJson}
              onChange={e => setFormData(prev => ({ ...prev, customJson: e.target.value }))}
              style={{
                width: '100%',
                padding: '8px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                minHeight: '60px',
                fontFamily: 'monospace',
                fontSize: '12px'
              }}
            />
          </div>
          <button
            onClick={() => {
              if (formData.customJson) {
                try {
                  const customData = JSON.parse(formData.customJson);
                  updateCampaign(customData);
                  setFormData(prev => ({ ...prev, customJson: '' }));
                } catch (e) {
                  alert('Invalid JSON format');
                }
              }
            }}
            disabled={!formData.customJson}
            style={{
              padding: '10px 15px',
              backgroundColor: '#6f42c1',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              width: '100%'
            }}
          >
            Send Custom Update
          </button>
        </div>
      </div>

      {/* Quick Actions */}
      <div
        style={{ marginTop: '20px', padding: '15px', background: '#e9ecef', borderRadius: '8px' }}
      >
        <h4 style={{ margin: '0 0 10px 0', color: '#495057' }}>⚡ Quick Actions</h4>
        <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
          <button
            onClick={() => updateCampaignFeatures(['waitlist', 'referral', 'leaderboard'], 'set')}
            style={{
              padding: '8px 12px',
              backgroundColor: '#17a2b8',
              color: 'white',
              border: 'none',
              borderRadius: '4px'
            }}
          >
            Set Default Features
          </button>
          <button
            onClick={() =>
              updateCampaignConfig('communication', {
                broadcast_enabled: true,
                ws_frequency: '5s',
                channels: ['main', 'updates']
              })
            }
            style={{
              padding: '8px 12px',
              backgroundColor: '#ffc107',
              color: '#212529',
              border: 'none',
              borderRadius: '4px'
            }}
          >
            Enable Broadcasting
          </button>
          <button
            onClick={() =>
              updateCampaign({
                status: 'active',
                priority: 'high',
                version: '2.0.0',
                last_updated: new Date().toISOString()
              })
            }
            style={{
              padding: '8px 12px',
              backgroundColor: '#28a745',
              color: 'white',
              border: 'none',
              borderRadius: '4px'
            }}
          >
            Activate Campaign
          </button>
        </div>
      </div>
    </div>
  );
}

// Component to display live WASM global metadata
function LiveWasmMetadataDisplay() {
  const wasmMetadata = useWasmGlobalMetadata(1000);
  useEffect(() => {
    if (typeof window !== 'undefined') {
      console.log(
        '[LiveWasmMetadataDisplay] window.__WASM_GLOBAL_METADATA:',
        window.__WASM_GLOBAL_METADATA
      );
    }
  }, [wasmMetadata]);
  return (
    <div style={{ marginTop: '30px' }}>
      <h3>Live WASM Metadata (window.__WASM_GLOBAL_METADATA)</h3>
      <details
        style={{
          border: '1px solid #ddd',
          borderRadius: '6px',
          padding: '10px',
          backgroundColor: '#fafafa'
        }}
      >
        <summary
          style={{
            cursor: 'pointer',
            fontWeight: 'bold',
            padding: '5px 0'
          }}
        >
          Click to view live WASM metadata
        </summary>
        <pre
          style={{
            marginTop: '10px',
            backgroundColor: '#fff',
            padding: '15px',
            borderRadius: '4px',
            fontSize: '12px',
            overflow: 'auto',
            maxHeight: '300px',
            border: '1px solid #eee'
          }}
        >
          {wasmMetadata ? JSON.stringify(wasmMetadata, null, 2) : 'No WASM metadata available'}
        </pre>
      </details>
    </div>
  );
}

// Component to show connection status with window state indicators
function ConnectionStatus() {
  const { connected, connecting, reconnectAttempts, isConnected, wasmReady } =
    useConnectionStatus();

  // Window state tracking
  const [documentHidden, setDocumentHidden] = useState(
    typeof document !== 'undefined' ? document.hidden : false
  );
  const [windowFocused, setWindowFocused] = useState(
    typeof document !== 'undefined' ? document.hasFocus() : true
  );

  // Raw WebSocket status color
  const wsStatusColor = useMemo(() => {
    if (connected) return '#4CAF50'; // Green
    if (connecting) return '#FF9800'; // Orange
    return '#F44336'; // Red
  }, [connected, connecting]);

  // Raw WebSocket status text
  const wsStatusText = useMemo(() => {
    if (connected) return 'WebSocket Connected';
    if (connecting) return 'WebSocket Connecting...';
    return `WebSocket Disconnected (${reconnectAttempts} attempts)`;
  }, [connected, connecting, reconnectAttempts]);

  // Combined status color (WASM + WebSocket)
  const combinedStatusColor = useMemo(() => {
    if (isConnected) return '#4CAF50'; // Green
    if (connecting) return '#FF9800'; // Orange
    return '#F44336'; // Red
  }, [isConnected, connecting]);

  // Combined status text
  const combinedStatusText = useMemo(() => {
    if (isConnected) return 'System Connected (WASM + WebSocket)';
    if (connecting) return 'System Connecting...';
    return `System Disconnected (${reconnectAttempts} attempts)`;
  }, [isConnected, connecting, reconnectAttempts]);

  useEffect(() => {
    const handleVisibilityChange = () => setDocumentHidden(document.hidden);
    const handleFocus = () => setWindowFocused(true);
    const handleBlur = () => setWindowFocused(false);

    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', handleVisibilityChange);
    }
    if (typeof window !== 'undefined') {
      window.addEventListener('focus', handleFocus);
      window.addEventListener('blur', handleBlur);
    }

    return () => {
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', handleVisibilityChange);
      }
      if (typeof window !== 'undefined') {
        window.removeEventListener('focus', handleFocus);
        window.removeEventListener('blur', handleBlur);
      }
    };
  }, []);

  return (
    <div
      style={{
        padding: '15px',
        backgroundColor: '#f5f5f5',
        borderRadius: '8px',
        marginBottom: '20px',
        border: `2px solid ${combinedStatusColor}`
      }}
    >
      <h3 style={{ margin: '0 0 10px 0' }}>System Status</h3>
      <div style={{ display: 'flex', gap: '20px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ color: combinedStatusColor, fontWeight: 'bold', fontSize: '16px' }}>
          {combinedStatusText}
        </div>
        <div style={{ color: wsStatusColor, fontWeight: 'bold', fontSize: '15px' }}>
          {wsStatusText}
        </div>
        <div>WASM: {wasmReady ? '✅ Ready' : '❌ Not Ready'}</div>
        <div>Window: {windowFocused ? '👁️ Focused' : '😴 Unfocused'}</div>
        <div>Tab: {documentHidden ? '🙈 Hidden' : '👀 Visible'}</div>
        <div>
          Network:{' '}
          {typeof navigator !== 'undefined' && navigator.onLine ? '🌐 Online' : '📴 Offline'}
        </div>
      </div>
      {reconnectAttempts > 0 && (
        <div style={{ marginTop: '8px', fontSize: '14px', color: '#666' }}>
          Auto-reconnection will trigger when window gains focus, becomes visible, or network comes
          back online.
        </div>
      )}
    </div>
  );
}

// Main search interface component
function SearchInterface() {
  const [searchState, setSearchState] = useState({
    query: '',
    loading: false,
    results: [],
    error: null,
    currentQuery: '',
    stopped: false
  });

  const { metadata } = useMetadata();
  const globalStore = useGlobalStore();

  // Listen for search responses
  const searchEvents = useEventHistory('search:search:v1:success', 10);
  const searchFailedEvents = useEventHistory('search:search:v1:failed', 5);

  // Handle search responses
  useEffect(() => {
    const latestCompleted = searchEvents[searchEvents.length - 1];
    const latestFailed = searchFailedEvents[searchFailedEvents.length - 1];

    if (latestCompleted && latestCompleted.timestamp > (latestFailed?.timestamp || 0)) {
      logStatus('Search completed successfully', latestCompleted.payload);
      setSearchState(prev => ({
        ...prev,
        loading: false,
        results: (latestCompleted.payload?.results || []).sort((a: any, b: any) => {
          // Sort by most recent (assume result.timestamp or fallback to index)
          const ta = new Date(a.timestamp || 0).getTime();
          const tb = new Date(b.timestamp || 0).getTime();
          return tb - ta;
        }),
        error: null,
        currentQuery: latestCompleted.payload?.query || prev.currentQuery,
        query: '', // Clear input
        stopped: false
      }));
      // Actually clear the input field by resetting value in the DOM
      setTimeout(() => {
        const inputEl = document.querySelector('input[type="text"]');
        if (inputEl) (inputEl as HTMLInputElement).value = '';
      }, 0);
    } else if (latestFailed) {
      logStatus('Search failed', latestFailed.payload);
      setSearchState(prev => ({
        ...prev,
        loading: false,
        error: latestFailed.payload?.error || 'Search failed',
        results: [],
        stopped: false
      }));
    }
  }, [searchEvents, searchFailedEvents]);

  // Handle search submission
  const handleSearch = useCallback(
    (query: string) => {
      if (!query.trim()) return;

      logStatus('Initiating search', { query: query.trim() });

      setSearchState(prev => ({
        ...prev,
        loading: true,
        error: null,
        results: [],
        currentQuery: query.trim(),
        stopped: false
      }));

      const correlationId = generateUUID();
      const searchEvent = {
        type: 'search:search:v1:requested',
        payload: {
          query: query.trim(),
          types: [], // Empty array for all types
          page_size: 20,
          page_number: 1,
          campaign_id: metadata.campaign?.campaignId || 0
        },
        metadata: {
          ...metadata,
          correlation_id: correlationId,
          timestamp: new Date().toISOString()
        }
      };

      logStatus('Emitting search event', searchEvent);
      logStatus('Search payload structure', {
        query: searchEvent.payload.query,
        types: searchEvent.payload.types,
        page_size: searchEvent.payload.page_size,
        page_number: searchEvent.payload.page_number,
        campaign_id: searchEvent.payload.campaign_id
      });
      globalStore.emitEvent(searchEvent);
    },
    [metadata, globalStore]
  );

  // Stop search handler
  const handleStopSearch = useCallback(() => {
    setSearchState(prev => ({ ...prev, loading: false, stopped: true }));
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      handleSearch(searchState.query);
    },
    [handleSearch, searchState.query]
  );

  const handleInputChange = useCallback((value: string) => {
    setSearchState(prev => ({ ...prev, query: value }));
  }, []);

  return (
    <div style={{ marginBottom: '30px' }}>
      <h3>Search</h3>

      <form onSubmit={handleSubmit} style={{ marginBottom: '20px' }}>
        <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
          <input
            type="text"
            value={searchState.query}
            onChange={e => handleInputChange(e.target.value)}
            placeholder="Enter your search query..."
            style={{
              padding: '12px',
              fontSize: '16px',
              borderRadius: '6px',
              border: '2px solid #ddd',
              flex: '1',
              outline: 'none'
            }}
            disabled={searchState.loading}
          />
          <button
            type="submit"
            disabled={searchState.loading || !searchState.query.trim()}
            style={{
              padding: '12px 24px',
              fontSize: '16px',
              borderRadius: '6px',
              border: 'none',
              backgroundColor: searchState.loading ? '#ccc' : '#007bff',
              color: 'white',
              cursor: searchState.loading ? 'not-allowed' : 'pointer',
              fontWeight: 'bold'
            }}
          >
            {searchState.loading ? 'Searching...' : 'Search'}
          </button>
          {searchState.loading && (
            <button
              type="button"
              onClick={handleStopSearch}
              style={{
                padding: '12px 18px',
                fontSize: '16px',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: '#dc3545',
                color: 'white',
                cursor: 'pointer',
                fontWeight: 'bold'
              }}
            >
              Stop
            </button>
          )}
        </div>
      </form>

      {/* Status and Results */}
      {searchState.loading && !searchState.stopped && (
        <div style={{ color: '#007bff', fontWeight: 'bold', marginBottom: '15px' }}>
          🔍 Searching for "{searchState.currentQuery}"...
        </div>
      )}
      {searchState.stopped && (
        <div style={{ color: '#dc3545', fontWeight: 'bold', marginBottom: '15px' }}>
          ⏹️ Search stopped.
        </div>
      )}

      {searchState.error && (
        <div
          style={{
            color: '#dc3545',
            backgroundColor: '#f8d7da',
            padding: '10px',
            borderRadius: '4px',
            marginBottom: '15px',
            border: '1px solid #f5c6cb'
          }}
        >
          ❌ Error: {searchState.error}
        </div>
      )}

      {searchState.currentQuery && !searchState.loading && !searchState.stopped && (
        <SearchResults query={searchState.currentQuery} results={searchState.results} />
      )}
    </div>
  );
}

// Search results display component
function SearchResults({ query, results }: { query: string; results: any[] }) {
  return (
    <div>
      <h4>Results for "{query}"</h4>
      {results.length > 0 ? (
        <div
          style={{
            border: '1px solid #ddd',
            borderRadius: '8px',
            padding: '15px',
            backgroundColor: '#fafafa'
          }}
        >
          {results.map((result: any, index: number) => (
            <div
              key={result.id || index}
              style={{
                padding: '15px',
                border: '1px solid #eee',
                borderRadius: '6px',
                marginBottom: '10px',
                backgroundColor: 'white'
              }}
            >
              <div style={{ fontWeight: 'bold', marginBottom: '5px', fontSize: '18px' }}>
                {result.title || `Result ${index + 1}`}
              </div>
              <div style={{ color: '#666', fontSize: '14px', marginBottom: '8px' }}>
                {result.description || result.content || 'No description available'}
              </div>
              {/* Detailed fields display */}
              {result.fields && (
                <details style={{ marginBottom: '8px' }}>
                  <summary style={{ fontWeight: 'bold', cursor: 'pointer' }}>Details</summary>
                  <div style={{ fontSize: '13px', color: '#444', marginTop: '6px' }}>
                    {Object.entries(result.fields).map(([key, value]) => (
                      <div key={key} style={{ marginBottom: '4px' }}>
                        <span style={{ fontWeight: 'bold' }}>{key}:</span>{' '}
                        {Array.isArray(value)
                          ? value.join(', ')
                          : typeof value === 'object'
                            ? JSON.stringify(value)
                            : String(value)}
                      </div>
                    ))}
                  </div>
                </details>
              )}
              {result.score && (
                <div style={{ fontSize: '12px', color: '#888', marginTop: '5px' }}>
                  Relevance Score: {result.score.toFixed(3)}
                </div>
              )}
              {result.timestamp && (
                <div style={{ fontSize: '12px', color: '#888', marginTop: '5px' }}>
                  Timestamp: {new Date(result.timestamp).toLocaleString()}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div
          style={{
            textAlign: 'center',
            color: '#666',
            fontStyle: 'italic',
            padding: '20px'
          }}
        >
          No results found. Try a different search term.
        </div>
      )}
    </div>
  );
}

// Component to display recent events
function EventHistory() {
  const events = useEventHistory(undefined, 8);
  // Sort events by latest first
  const sortedEvents = [...events].sort((a: any, b: any) => {
    const ta = new Date(a.timestamp || 0).getTime();
    const tb = new Date(b.timestamp || 0).getTime();
    return tb - ta;
  });

  return (
    <div style={{ marginBottom: '30px' }}>
      <h3>Recent System Events</h3>
      <div
        style={{
          maxHeight: '300px',
          overflowY: 'auto',
          border: '1px solid #ddd',
          borderRadius: '6px',
          backgroundColor: '#fafafa'
        }}
      >
        {sortedEvents.length > 0 ? (
          sortedEvents.map((event, index) => (
            <div
              key={index}
              style={{
                padding: '12px',
                borderBottom: index < sortedEvents.length - 1 ? '1px solid #eee' : 'none'
              }}
            >
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: '5px'
                }}
              >
                <span
                  style={{
                    fontWeight: 'bold',
                    color: event.type.includes('failed') ? '#dc3545' : '#007bff',
                    fontSize: '14px'
                  }}
                >
                  {event.type}
                </span>
                <span style={{ fontSize: '12px', color: '#666' }}>
                  {new Date(event.timestamp).toLocaleTimeString()}
                </span>
              </div>
              {event.payload && (
                <div
                  style={{
                    fontSize: '12px',
                    color: '#555',
                    fontFamily: 'monospace',
                    backgroundColor: '#fff',
                    padding: '5px',
                    borderRadius: '3px',
                    maxHeight: '60px',
                    overflow: 'hidden'
                  }}
                >
                  {JSON.stringify(event.payload, null, 1).substring(0, 150)}
                  {JSON.stringify(event.payload).length > 150 && '...'}
                </div>
              )}
            </div>
          ))
        ) : (
          <div
            style={{
              padding: '20px',
              textAlign: 'center',
              color: '#666',
              fontStyle: 'italic'
            }}
          >
            No events yet
          </div>
        )}
      </div>
    </div>
  );
}

// Component to display current metadata
function MetadataDisplay() {
  const { metadata } = useMetadata();

  return (
    <div>
      <h3>System Metadata</h3>
      <details
        style={{
          border: '1px solid #ddd',
          borderRadius: '6px',
          padding: '10px',
          backgroundColor: '#fafafa'
        }}
      >
        <summary
          style={{
            cursor: 'pointer',
            fontWeight: 'bold',
            padding: '5px 0'
          }}
        >
          Click to view current metadata
        </summary>
        <pre
          style={{
            marginTop: '10px',
            backgroundColor: '#fff',
            padding: '15px',
            borderRadius: '4px',
            fontSize: '12px',
            overflow: 'auto',
            maxHeight: '300px',
            border: '1px solid #eee'
          }}
        >
          {JSON.stringify(metadata, null, 2)}
        </pre>
      </details>
    </div>
  );
}

export default App;
