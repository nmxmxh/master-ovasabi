import { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import CampaignSwitchingPage from './pages/CampaignSwitchingPage';
import { useInitializeUserId } from './hooks/useInitializeUserId';
import { useCampaignState } from './store/hooks/useCampaign';
import { useCampaignStore } from './store/stores/campaignStore';
import { useMetadata } from './store/hooks/useMetadata';
import { useEventHistory } from './store/hooks/useEvents';
import { useCampaignData } from './providers/CampaignProvider';
import { CampaignProvider } from './providers/CampaignProvider';
import { setupCampaignSwitchHandler } from './lib/wasmBridge';
import './App.css';

// Minimal black and white styles
const minimalStyles = `
  .minimal-app {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    background: #000;
    color: #fff;
    font-size: 12px;
    line-height: 1.4;
  }
  
  .minimal-header {
    border-bottom: 1px solid #333;
    padding: 8px 16px;
    background: #111;
  }
  
  .minimal-nav {
    display: flex;
    gap: 16px;
  }
  
  .minimal-link {
    color: #fff;
    text-decoration: none;
    padding: 4px 8px;
    border: 1px solid #333;
    background: #000;
    font-size: 11px;
  }
  
  .minimal-link:hover {
    background: #333;
  }
  
  .minimal-link.active {
    background: #fff;
    color: #000;
  }
  
  .minimal-main {
    padding: 16px;
    max-width: 1200px;
    margin: 0 auto;
  }
  
  .minimal-section {
    margin-bottom: 24px;
    border: 1px solid #333;
    padding: 12px;
    background: #111;
  }
  
  .minimal-title {
    font-size: 14px;
    font-weight: bold;
    margin-bottom: 8px;
    color: #fff;
  }
  
  .minimal-text {
    font-size: 11px;
    color: #ccc;
    margin-bottom: 4px;
  }
  
  .minimal-button {
    background: #000;
    color: #fff;
    border: 1px solid #333;
    padding: 4px 8px;
    font-size: 11px;
    cursor: pointer;
    font-family: inherit;
  }
  
  .minimal-button:hover {
    background: #333;
  }
  
  .minimal-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .minimal-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;
  }
  
  .minimal-card {
    border: 1px solid #333;
    padding: 8px;
    background: #000;
  }
  
  .minimal-code {
    background: #000;
    color: #0f0;
    padding: 8px;
    border: 1px solid #333;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 10px;
    overflow-x: auto;
    white-space: pre-wrap;
  }
  
  .minimal-status {
    display: inline-block;
    padding: 2px 6px;
    font-size: 10px;
    border: 1px solid #333;
  }
  
  .minimal-status.active {
    background: #0f0;
    color: #000;
  }
  
  .minimal-status.inactive {
    background: #f00;
    color: #fff;
  }
  
  .minimal-status.loading {
    background: #ff0;
    color: #000;
  }
`;

function App() {
  const { isInitialized } = useInitializeUserId();

  useEffect(() => {
    setupCampaignSwitchHandler();
  }, []);

  if (!isInitialized) {
    return (
      <div className="minimal-app">
        <style>{minimalStyles}</style>
        <div className="minimal-section">
          <div className="minimal-text">Initializing...</div>
        </div>
      </div>
    );
  }

  return (
    <CampaignProvider>
      <Router>
        <div className="minimal-app">
          <style>{minimalStyles}</style>
          <header className="minimal-header">
            <nav className="minimal-nav">
              <Link to="/" className="minimal-link">
                CAMPAIGNS
              </Link>
              <Link to="/switching" className="minimal-link">
                SWITCH
              </Link>
            </nav>
          </header>

          <main className="minimal-main">
            <Routes>
              <Route path="/" element={<CampaignManagementPage />} />
              <Route path="/switching" element={<CampaignSwitchingPage />} />
            </Routes>
          </main>
        </div>
      </Router>
    </CampaignProvider>
  );
}

// Main Campaign Management Page
function CampaignManagementPage() {
  const { switchCampaignWithData } = useCampaignStore();
  const {
    campaigns,
    loading: campaignsLoading,
    error: campaignsError,
    refresh: refreshCampaigns
  } = useCampaignData();
  const { metadata } = useMetadata();
  const campaignState = useCampaignState();
  const events = useEventHistory(undefined, 20);

  const currentCampaign = campaignState.state || {};

  return (
    <div>
      {/* System Status */}
      <div className="minimal-section">
        <div className="minimal-title">SYSTEM STATUS</div>
        <div className="minimal-text">
          Events: {events.length} | Campaigns: {campaigns.length} | Status:{' '}
          {campaignsLoading ? 'LOADING' : 'READY'}
        </div>
        <button onClick={refreshCampaigns} className="minimal-button" disabled={campaignsLoading}>
          {campaignsLoading ? 'LOADING...' : 'REFRESH'}
        </button>
      </div>

      {/* Current Campaign */}
      <div className="minimal-section">
        <div className="minimal-title">CURRENT CAMPAIGN</div>
        <div className="minimal-text">
          ID: {currentCampaign.campaignId || 'N/A'} | Status: {currentCampaign.status || 'UNKNOWN'}
        </div>
        {currentCampaign.title && (
          <div className="minimal-text">Title: {currentCampaign.title}</div>
        )}
      </div>

      {/* Campaign List */}
      <div className="minimal-section">
        <div className="minimal-title">CAMPAIGNS</div>
        {campaignsLoading ? (
          <div className="minimal-text">Loading campaigns...</div>
        ) : campaignsError ? (
          <div className="minimal-text">Error: {campaignsError}</div>
        ) : campaigns.length === 0 ? (
          <div className="minimal-text">No campaigns available</div>
        ) : (
          <div className="minimal-grid">
            {campaigns.map((campaign, index) => (
              <div key={campaign.id || index} className="minimal-card">
                <div className="minimal-text">
                  <strong>{campaign.title || campaign.name || `Campaign ${campaign.id}`}</strong>
                </div>
                <div className="minimal-text">
                  ID: {campaign.id} |
                  <span className={`minimal-status ${campaign.status || 'unknown'}`}>
                    {campaign.status || 'UNKNOWN'}
                  </span>
                </div>
                {campaign.description && <div className="minimal-text">{campaign.description}</div>}
                {campaign.features && campaign.features.length > 0 && (
                  <div className="minimal-text">Features: {campaign.features.join(', ')}</div>
                )}
                <button
                  onClick={() =>
                    switchCampaignWithData &&
                    switchCampaignWithData(campaign, response => {
                      console.log('Campaign switch response:', response);
                    })
                  }
                  className="minimal-button"
                  style={{ marginTop: '8px' }}
                >
                  SWITCH
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Debug Info */}
      <div className="minimal-section">
        <div className="minimal-title">DEBUG INFO</div>
        <div className="minimal-code">
          {JSON.stringify(
            {
              campaigns: campaigns.length,
              currentCampaign: currentCampaign.campaignId,
              events: events.length,
              metadata: metadata?.campaign
            },
            null,
            2
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
