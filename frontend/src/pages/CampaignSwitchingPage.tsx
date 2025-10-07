import React, { useState, useEffect, useMemo } from 'react';
import toast, { Toaster } from 'react-hot-toast';
import { useCampaignStore } from '../store/stores/campaignStore';
import { useCampaignData } from '../providers/CampaignProvider';
import { useMetadata } from '../store/hooks/useMetadata';
import { useEventHistory } from '../store/hooks/useEvents';
import SimpleCampaignUIRenderer from '../components/SimpleCampaignUIRenderer';

interface CampaignSwitchEvent {
  old_campaign_id: string;
  new_campaign_id: string;
  reason: string;
  timestamp: string;
  status?: string;
}

// Minimal black and white styles for switching page
const switchingStyles = `
  .switching-app {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    background: #000;
    color: #fff;
    font-size: 12px;
    line-height: 1.4;
  }
  
  .switching-header {
    border-bottom: 1px solid #333;
    padding: 8px 16px;
    background: #111;
  }
  
  .switching-main {
    padding: 16px;
    max-width: 1200px;
    margin: 0 auto;
  }
  
  .switching-section {
    margin-bottom: 24px;
    border: 1px solid #333;
    padding: 12px;
    background: #111;
  }
  
  .switching-title {
    font-size: 14px;
    font-weight: bold;
    margin-bottom: 8px;
    color: #fff;
  }
  
  .switching-text {
    font-size: 11px;
    color: #ccc;
    margin-bottom: 4px;
  }
  
  .switching-button {
    background: #000;
    color: #fff;
    border: 1px solid #333;
    padding: 6px 12px;
    font-size: 10px;
    font-weight: bold;
    cursor: pointer;
    font-family: inherit;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    transition: all 0.2s ease;
  }
  
  .switching-button:hover {
    background: #333;
    border-color: #555;
    transform: translateY(-1px);
  }
  
  .switching-button:disabled {
    opacity: 0.3;
    cursor: not-allowed;
    background: #111;
    border-color: #222;
  }
  
  .switching-button:not(:disabled):active {
    transform: translateY(0);
    background: #222;
  }
  
  .switching-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;
  }
  
  .switching-card {
    border: 1px solid #333;
    padding: 12px;
    background: #000;
    transition: all 0.2s ease;
    cursor: pointer;
  }
  
  .switching-card:hover {
    background: #111;
    border-color: #555;
    transform: translateY(-1px);
  }
  
  .switching-card.active {
    border-color: #0f0;
    background: #001100;
    box-shadow: 0 0 8px rgba(0, 255, 0, 0.2);
  }
  
  .switching-status {
    display: inline-block;
    padding: 3px 8px;
    font-size: 9px;
    font-weight: bold;
    border: 1px solid #333;
    border-radius: 3px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  
  .switching-status.active {
    background: #0f0;
    color: #000;
    border-color: #0f0;
    box-shadow: 0 0 4px rgba(0, 255, 0, 0.3);
  }
  
  .switching-status.inactive {
    background: #f00;
    color: #fff;
    border-color: #f00;
  }
  
  .switching-status.loading {
    background: #ff0;
    color: #000;
    border-color: #ff0;
    animation: pulse 1.5s infinite;
  }
  
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }
  
  .switching-tabs {
    display: flex;
    border-bottom: 1px solid #333;
    margin-bottom: 16px;
  }
  
  .switching-tab {
    background: #000;
    color: #fff;
    border: 1px solid #333;
    border-bottom: none;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 11px;
  }
  
  .switching-tab:hover {
    background: #333;
  }
  
  .switching-tab.active {
    background: #fff;
    color: #000;
  }
  
  .switching-code {
    background: #000;
    color: #0f0;
    padding: 8px;
    border: 1px solid #333;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 10px;
    overflow-x: auto;
    white-space: pre-wrap;
  }
`;

const CampaignSwitchingPage: React.FC = () => {
  // CampaignSwitchingPage rendered

  const { switchCampaignWithData, currentCampaign } = useCampaignStore();
  const {
    campaigns,
    loading: campaignsLoading,
    error: campaignsError,
    refresh: refreshCampaigns
  } = useCampaignData();
  const { metadata } = useMetadata();
  const events = useEventHistory(undefined, 50);

  // Toast notification helper
  const showToast = (
    message: string,
    type: 'success' | 'error' | 'loading' = 'success'
  ): string | undefined => {
    switch (type) {
      case 'success':
        return toast.success(message, {
          duration: 4000,
          style: {
            background: '#000',
            color: '#fff',
            border: '1px solid #333',
            fontFamily: 'Monaco, Menlo, Consolas, monospace',
            fontSize: '12px'
          }
        });
      case 'error':
        return toast.error(message, {
          duration: 6000,
          style: {
            background: '#000',
            color: '#f00',
            border: '1px solid #f00',
            fontFamily: 'Monaco, Menlo, Consolas, monospace',
            fontSize: '12px'
          }
        });
      case 'loading':
        return toast.loading(message, {
          style: {
            background: '#000',
            color: '#fff',
            border: '1px solid #333',
            fontFamily: 'Monaco, Menlo, Consolas, monospace',
            fontSize: '12px'
          }
        });
      default:
        return undefined;
    }
  };

  const [switchHistory, setSwitchHistory] = useState<CampaignSwitchEvent[]>([]);
  const [isSwitching, setIsSwitching] = useState(false);
  const [selectedCampaign, setSelectedCampaign] = useState<any>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  // Filter campaign switch events - memoized to prevent infinite re-renders
  const switchEvents = useMemo(() => {
    return events.filter(
      e =>
        e.type?.includes('campaign:switch') ||
        e.type?.includes('campaign:switch:required') ||
        e.type?.includes('campaign:switch:completed')
    );
  }, [events]);

  // Enhanced campaign information with detailed logging
  const getCampaignInfo = (campaign: any) => {
    const info = {
      id: campaign.id || 'Unknown',
      title: campaign.title || campaign.name || 'Untitled Campaign',
      slug: campaign.slug || 'unknown',
      description: campaign.description || 'No description available',
      status: campaign.status || 'unknown',
      features: campaign.features || [],
      metadata: campaign.metadata || {},
      ui_content: campaign.ui_content || {},
      service_configs: campaign.service_configs || {},
      theme: campaign.theme || {},
      ranking_formula: campaign.ranking_formula || 'N/A',
      owner_id: campaign.owner_id || 'Unknown',
      start_date: campaign.start_date || 'N/A',
      end_date: campaign.end_date || 'N/A'
    };

    console.log('[CampaignSwitchingPage] Enhanced campaign info:', {
      original: campaign,
      enhanced: info,
      hasUI: !!info.ui_content,
      hasTheme: !!info.theme,
      hasConfigs: !!info.service_configs
    });

    return info;
  };

  // Update switch history when events change
  useEffect(() => {
    const newHistory: CampaignSwitchEvent[] = [];

    switchEvents.forEach(event => {
      console.log('[CampaignSwitchingPage] Processing switch event:', {
        type: event.type,
        payload: event.payload,
        timestamp: event.timestamp
      });

      if (
        event.type === 'campaign:switch:required' ||
        event.type === 'campaign:switch:completed' ||
        event.type === 'campaign:switch:v1:success'
      ) {
        const payload = event.payload as any;
        if (payload) {
          newHistory.push({
            old_campaign_id: payload.old_campaign_id || payload.campaign_id || 'Unknown',
            new_campaign_id: payload.new_campaign_id || payload.campaign_id || 'Unknown',
            reason: payload.reason || payload.switch_reason || 'Unknown',
            timestamp: event.timestamp || new Date().toISOString(),
            status:
              event.type.includes('completed') || event.type.includes('success')
                ? 'completed'
                : 'in_progress'
          });
        }
      }
    });

    setSwitchHistory(
      newHistory.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    );
  }, [switchEvents]);

  const handleCampaignSwitch = async (campaign: any) => {
    const campaignInfo = getCampaignInfo(campaign);
    const isCurrent =
      currentCampaign?.id === campaign.id;

    console.log('[CampaignSwitchingPage] Switch attempt:', {
      campaign: campaignInfo,
      currentCampaign: currentCampaign,
      isCurrent,
      isSwitching,
      switchFunction: !!switchCampaignWithData
    });

    if (isSwitching || isCurrent) {
      console.log('[CampaignSwitchingPage] Switch blocked:', { isSwitching, isCurrent });
      return;
    }

    setSelectedCampaign(campaign);
    setIsSwitching(true);

    const loadingToast = showToast(`Switching to ${campaignInfo.title}...`, 'loading');

    try {
      if (switchCampaignWithData) {
        console.log('[CampaignSwitchingPage] Calling switchCampaignWithData...');

        // Use a fast timeout for responsive campaign switching
        await new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => {
            console.log('[CampaignSwitchingPage] Switch timeout after 1 second');
            toast.dismiss(loadingToast);
            showToast('Campaign switch timed out', 'error');
            reject(new Error('Switch timeout'));
          }, 1000); // Fast 1 second timeout for responsive switching

          switchCampaignWithData(campaign, response => {
            clearTimeout(timeout);
            console.log('[CampaignSwitchingPage] Switch response:', response);

            if (response.type?.includes('success')) {
              // Campaign switched successfully
              console.log('[CampaignSwitchingPage] Switch successful');
              toast.dismiss(loadingToast);
              showToast(`Successfully switched to ${campaignInfo.title}`, 'success');
              resolve();
            } else if (response.type?.includes('failed')) {
              // Campaign switch failed
              console.log('[CampaignSwitchingPage] Switch failed:', response.payload?.error);
              toast.dismiss(loadingToast);
              showToast(response.payload?.error || 'Failed to switch campaign', 'error');
              reject(new Error(response.payload?.error || 'Switch failed'));
            } else {
              console.log('[CampaignSwitchingPage] Switch response not recognized:', response.type);
              // Don't resolve immediately for unrecognized responses
              // Let the timeout handle it
            }
          });
        });
      } else {
        console.log('[CampaignSwitchingPage] switchCampaignWithData not available');
        throw new Error('Switch function not available');
      }
    } catch (error) {
      // Campaign switch error
      console.error('[CampaignSwitchingPage] Switch error:', error);
      toast.dismiss(loadingToast);
      showToast(error instanceof Error ? error.message : 'Unknown error occurred', 'error');
    } finally {
      setIsSwitching(false);
      setSelectedCampaign(null);
    }
  };

  return (
    <div className="switching-app">
      <style>{switchingStyles}</style>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#000',
            color: '#fff',
            border: '1px solid #333',
            fontFamily: 'Monaco, Menlo, Consolas, monospace',
            fontSize: '12px',
            padding: '12px 16px',
            borderRadius: '4px'
          }
        }}
      />

      {/* Header */}
      <div className="switching-header">
        <div className="switching-title">CAMPAIGN SWITCHING INTERFACE</div>
        <div className="switching-text">
          Real-time campaign management with seamless switching capabilities
        </div>
      </div>

      <main className="switching-main">
        {/* Tab Navigation */}
        <div className="switching-tabs">
          <button
            className={`switching-tab ${activeTab === 0 ? 'active' : ''}`}
            onClick={() => setActiveTab(0)}
          >
            CAMPAIGN INTERFACE
          </button>
          <button
            className={`switching-tab ${activeTab === 1 ? 'active' : ''}`}
            onClick={() => setActiveTab(1)}
          >
            MANAGEMENT
          </button>
          <button
            className={`switching-tab ${activeTab === 2 ? 'active' : ''}`}
            onClick={() => setActiveTab(2)}
          >
            HISTORY & EVENTS
          </button>
        </div>

        {/* Tab Content */}
        {activeTab === 0 && (
          <div className="switching-section" style={{ padding: '0', background: '#000' }}>
            <div
              style={{
                padding: '16px',
                borderBottom: '1px solid #333',
                background: '#111'
              }}
            >
              <div className="switching-title">CAMPAIGN INTERFACE</div>
              <div className="switching-text">
                {currentCampaign
                  ? `Rendering interface for: ${currentCampaign.title || 'Current Campaign'}`
                  : 'No campaign selected - choose a campaign to view its interface'}
              </div>
            </div>
            <div style={{ minHeight: '500px' }}>
              <SimpleCampaignUIRenderer
                campaign={currentCampaign}
                isLoading={isSwitching && selectedCampaign !== null}
              />
            </div>
          </div>
        )}

        {activeTab === 1 && (
          <div>
            {/* Current Campaign Status */}
            <div className="switching-section">
              <div className="switching-title">CURRENT CAMPAIGN STATUS</div>
              {currentCampaign ? (
                <div>
                  {(() => {
                    const campaignInfo = getCampaignInfo(currentCampaign);
                    return (
                      <>
                        <div className="switching-text">
                          <strong>{campaignInfo.title}</strong>
                        </div>
                        <div className="switching-text">
                          ID: {campaignInfo.id} | Slug: {campaignInfo.slug} |
                          <span className={`switching-status ${campaignInfo.status}`}>
                            {campaignInfo.status.toUpperCase()}
                          </span>
                        </div>
                        <div className="switching-text">
                          Description: {campaignInfo.description}
                        </div>
                        {campaignInfo.features.length > 0 && (
                          <div className="switching-text">
                            Features: {campaignInfo.features.join(', ')}
                          </div>
                        )}
                        {campaignInfo.theme && Object.keys(campaignInfo.theme).length > 0 && (
                          <div className="switching-text">
                            Theme:{' '}
                            {campaignInfo.theme.primary_color
                              ? `Primary: ${campaignInfo.theme.primary_color}`
                              : 'Custom theme'}
                          </div>
                        )}
                        {campaignInfo.ui_content &&
                          Object.keys(campaignInfo.ui_content).length > 0 && (
                            <div className="switching-text">
                              UI Components: {Object.keys(campaignInfo.ui_content).length}{' '}
                              configured
                            </div>
                          )}
                        <div className="switching-text">
                          Owner: {campaignInfo.owner_id} | Start: {campaignInfo.start_date} | End:{' '}
                          {campaignInfo.end_date}
                        </div>
                      </>
                    );
                  })()}
                </div>
              ) : (
                <div className="switching-text">
                  No Active Campaign - Select a campaign to get started.
                </div>
              )}
            </div>

            {/* System Stats */}
            <div className="switching-section">
              <div className="switching-title">SYSTEM STATISTICS</div>
              <div className="switching-grid">
                <div className="switching-card">
                  <div className="switching-text">
                    <strong>{campaigns.length}</strong>
                    <br />
                    Total Campaigns
                  </div>
                </div>
                <div className="switching-card">
                  <div className="switching-text">
                    <strong>{campaigns.filter(c => c.status === 'active').length}</strong>
                    <br />
                    Active Campaigns
                  </div>
                </div>
                <div className="switching-card">
                  <div className="switching-text">
                    <strong>{switchHistory.length}</strong>
                    <br />
                    Switch Events
                  </div>
                </div>
                <div className="switching-card">
                  <div className="switching-text">
                    <strong>{events.length}</strong>
                    <br />
                    System Events
                  </div>
                </div>
              </div>
            </div>

            {/* Available Campaigns */}
            <div className="switching-section">
              <div className="switching-title">AVAILABLE CAMPAIGNS</div>
              <button
                onClick={refreshCampaigns}
                className="switching-button"
                disabled={campaignsLoading}
                style={{ marginBottom: '12px' }}
              >
                {campaignsLoading ? 'LOADING...' : 'REFRESH'}
              </button>

              {campaignsLoading ? (
                <div className="switching-text">Loading campaigns...</div>
              ) : campaignsError ? (
                <div className="switching-text">Error: {campaignsError}</div>
              ) : (
                <div className="switching-grid">
                  {campaigns.map(campaign => {
                    const campaignInfo = getCampaignInfo(campaign);
                    const isCurrent =
                      currentCampaign?.id === campaign.id;

                    return (
                      <div
                        key={campaign.id}
                        className={`switching-card ${isCurrent ? 'active' : ''}`}
                        onClick={() => handleCampaignSwitch(campaign)}
                      >
                        <div
                          className="switching-text"
                          style={{ fontSize: '13px', marginBottom: '8px' }}
                        >
                          <strong>{campaignInfo.title}</strong>
                        </div>
                        <div
                          className="switching-text"
                          style={{ fontSize: '10px', color: '#888', marginBottom: '6px' }}
                        >
                          ID: {campaignInfo.id} | Slug: {campaignInfo.slug}
                        </div>
                        <div className="switching-text" style={{ marginBottom: '8px' }}>
                          <span className={`switching-status ${campaignInfo.status}`}>
                            {campaignInfo.status.toUpperCase()}
                          </span>
                        </div>
                        <div
                          className="switching-text"
                          style={{ marginBottom: '8px', lineHeight: '1.3' }}
                        >
                          {campaignInfo.description}
                        </div>
                        {campaignInfo.features.length > 0 && (
                          <div className="switching-text" style={{ marginBottom: '6px' }}>
                            <span style={{ color: '#0f0', fontWeight: 'bold' }}>FEATURES:</span>{' '}
                            {campaignInfo.features.slice(0, 3).join(', ')}
                            {campaignInfo.features.length > 3 &&
                              ` +${campaignInfo.features.length - 3} more`}
                          </div>
                        )}
                        {campaignInfo.theme && campaignInfo.theme.primary_color && (
                          <div className="switching-text" style={{ marginBottom: '6px' }}>
                            <span style={{ color: '#0f0', fontWeight: 'bold' }}>THEME:</span>
                            <span
                              style={{
                                display: 'inline-block',
                                width: '12px',
                                height: '8px',
                                backgroundColor: campaignInfo.theme.primary_color,
                                marginLeft: '6px',
                                border: '1px solid #333'
                              }}
                            ></span>
                            {campaignInfo.theme.primary_color}
                          </div>
                        )}
                        {campaignInfo.ui_content &&
                          Object.keys(campaignInfo.ui_content).length > 0 && (
                            <div className="switching-text" style={{ marginBottom: '6px' }}>
                              <span style={{ color: '#0f0', fontWeight: 'bold' }}>UI:</span>{' '}
                              {Object.keys(campaignInfo.ui_content).length} components
                            </div>
                          )}
                        <button
                          className="switching-button"
                          onClick={e => {
                            e.stopPropagation();
                            handleCampaignSwitch(campaign);
                          }}
                          disabled={isSwitching || isCurrent}
                          style={{ marginTop: '8px' }}
                        >
                          {isCurrent ? 'CURRENT' : 'SWITCH TO'}
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Switch History */}
            <div className="switching-section">
              <div className="switching-title">SWITCH HISTORY</div>
              <button
                onClick={() => setShowDetails(!showDetails)}
                className="switching-button"
                style={{ marginBottom: '12px' }}
              >
                {showDetails ? 'HIDE DETAILS' : 'VIEW DETAILS'}
              </button>

              {switchHistory.length === 0 ? (
                <div className="switching-text">
                  No Switch History - Campaign switches will appear here.
                </div>
              ) : (
                <div>
                  {switchHistory
                    .slice(0, showDetails ? switchHistory.length : 5)
                    .map((switchEvent, index) => (
                      <div key={index} className="switching-card" style={{ marginBottom: '8px' }}>
                        <div className="switching-text">
                          <strong>
                            {switchEvent.old_campaign_id} → {switchEvent.new_campaign_id}
                          </strong>
                          <span className={`switching-status ${switchEvent.status || 'completed'}`}>
                            {switchEvent.status || 'completed'}
                          </span>
                        </div>
                        <div className="switching-text">
                          Reason: {switchEvent.reason} |
                          {new Date(switchEvent.timestamp).toLocaleString()}
                        </div>
                      </div>
                    ))}
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 2 && (
          <div>
            {/* Switch History */}
            <div className="switching-section">
              <div className="switching-title">SWITCH HISTORY</div>
              <button
                onClick={() => setShowDetails(!showDetails)}
                className="switching-button"
                style={{ marginBottom: '12px' }}
              >
                {showDetails ? 'HIDE DETAILS' : 'VIEW DETAILS'}
              </button>

              {switchHistory.length === 0 ? (
                <div className="switching-text">
                  No Switch History - Campaign switches will appear here.
                </div>
              ) : (
                <div>
                  {switchHistory
                    .slice(0, showDetails ? switchHistory.length : 5)
                    .map((switchEvent, index) => (
                      <div key={index} className="switching-card" style={{ marginBottom: '8px' }}>
                        <div className="switching-text">
                          <strong>
                            {switchEvent.old_campaign_id} → {switchEvent.new_campaign_id}
                          </strong>
                          <span className={`switching-status ${switchEvent.status || 'completed'}`}>
                            {switchEvent.status || 'completed'}
                          </span>
                        </div>
                        <div className="switching-text">
                          Reason: {switchEvent.reason} |
                          {new Date(switchEvent.timestamp).toLocaleString()}
                        </div>
                      </div>
                    ))}
                </div>
              )}
            </div>

            {/* System Information */}
            <div className="switching-section">
              <div className="switching-title">SYSTEM INFORMATION</div>
              <div className="switching-text">
                <strong>Metadata:</strong>
              </div>
              <div className="switching-code">{JSON.stringify(metadata, null, 2)}</div>

              <div className="switching-text">
                <strong>Recent Events ({events.length}):</strong>
              </div>
              <div className="switching-code" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                {events.slice(0, 10).map((event, index) => (
                  <div key={index}>
                    {event.type} | {new Date(event.timestamp).toLocaleTimeString()} |
                    {event.type?.includes('success') ? 'SUCCESS' : 'INFO'}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

export default CampaignSwitchingPage;
