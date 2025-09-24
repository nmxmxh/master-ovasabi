import React, { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import { useInitializeUserId } from './hooks/useInitializeUserId';
import { useCampaignState } from './store/hooks/useCampaign';
import { useCampaignStore } from './store/stores/campaignStore';
import { useMetadata } from './store/hooks/useMetadata';
import { useEventHistory } from './store/hooks/useEvents';
import { useCampaignData } from './providers/CampaignProvider';
import { CampaignProvider } from './providers/CampaignProvider';
import CampaignSwitchingPage from './pages/CampaignSwitchingPage';
import ServiceListPage from './pages/ServiceListPage';
import { setupCampaignSwitchHandler } from './lib/wasmBridge';
import './App.css';
import UserServicePage from './pages/UserServicePage';

// Minimal black and white styles
const minimalStyles = `
  .minimal-app {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    background: #000;
    color: #fff;
    font-size: 12px;
    line-height: 1.4;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  
  .minimal-header {
    border-bottom: 1px solid #333;
    padding: 8px 16px;
    background: #111;
    flex-shrink: 0;
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
    cursor: pointer;
    display: inline-block;
    transition: all 0.2s ease;
  }
  
  .minimal-link:hover {
    background: #333;
    transform: translateY(-1px);
  }
  
  .minimal-link.active {
    background: #fff;
    color: #000;
  }
  
  .minimal-main {
    padding: 16px;
    max-width: 1200px;
    margin: 0 auto;
    flex: 1;
    width: 100%;
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
    transition: all 0.2s ease;
  }
  
  .minimal-button:hover {
    background: #333;
    transform: translateY(-1px);
  }
  
  .minimal-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none;
  }
  
  .minimal-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
  }
  
  .minimal-card {
    border: 1px solid #333;
    padding: 16px;
    background: #000;
    display: flex;
    flex-direction: column;
    min-height: 200px;
    transition: all 0.2s ease;
    position: relative;
  }
  
  .minimal-card:hover {
    border-color: #555;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }
  
  .minimal-card.active {
    border: 2px solid #fff;
    box-shadow: 0 0 8px rgba(255, 255, 255, 0.2);
  }
  
  .minimal-card.active:hover {
    border-color: #fff;
    box-shadow: 0 0 12px rgba(255, 255, 255, 0.3);
  }
  
  .minimal-card-header {
    margin-bottom: 12px;
    flex-shrink: 0;
  }
  
  .minimal-card-title {
    font-size: 13px;
    font-weight: bold;
    color: #fff;
    margin-bottom: 6px;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }
  
  .minimal-card-id {
    font-size: 10px;
    color: #888;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  }
  
  .minimal-card-description {
    font-size: 11px;
    color: #ccc;
    line-height: 1.4;
    margin-bottom: 12px;
    flex: 1;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-features {
    font-size: 10px;
    color: #999;
    margin-bottom: 12px;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-actions {
    margin-top: auto;
    padding-top: 12px;
    border-top: 1px solid #222;
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
    border-radius: 2px;
    font-weight: bold;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  
  .minimal-status.active {
    background: #0f0;
    color: #000;
    border-color: #0f0;
  }
  
  .minimal-status.inactive {
    background: #f00;
    color: #fff;
    border-color: #f00;
  }
  
  .minimal-status.loading {
    background: #ff0;
    color: #000;
    border-color: #ff0;
  }
  
  .minimal-card-button {
    width: 100%;
    background: #000;
    color: #fff;
    border: 1px solid #333;
    padding: 8px 12px;
    font-size: 11px;
    cursor: pointer;
    font-family: inherit;
    transition: all 0.2s ease;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: bold;
  }
  
  .minimal-card-button:hover {
    background: #333;
    border-color: #555;
    transform: translateY(-1px);
  }
  
  .minimal-card-button:active {
    transform: translateY(0);
  }
`;

// Navigation component with active link highlighting
function Navigation() {
  let location;

  try {
    location = useLocation();
  } catch (error) {
    console.error('Navigation: Router context not available:', error);
    return (
      <nav className="minimal-nav">
        <div style={{ color: '#f00' }}>
          Router Error: {error instanceof Error ? error.message : 'Unknown error'}
        </div>
      </nav>
    );
  }

  return (
    <nav className="minimal-nav">
      <Link to="/" className={`minimal-link ${location.pathname === '/' ? 'active' : ''}`}>
        CAMPAIGNS
      </Link>
      <Link
        to="/switching"
        className={`minimal-link ${location.pathname === '/switching' ? 'active' : ''}`}
      >
        SWITCH
      </Link>
      <Link
        to="/services"
        className={`minimal-link ${location.pathname.startsWith('/services') ? 'active' : ''}`}
      >
        SERVICES
      </Link>
    </nav>
  );
}

function App() {
  const { isInitialized } = useInitializeUserId();

  useEffect(() => {
    setupCampaignSwitchHandler();
  }, []);

  // Debug route changes - removed excessive logging

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
    <Router>
      <CampaignProvider>
        <div className="minimal-app">
          <style>{minimalStyles}</style>
          <header className="minimal-header">
            <Navigation />
          </header>

          <main className="minimal-main">
            <Routes>
              <Route path="/" element={<CampaignManagementPage />} />
              <Route path="/switching" element={<CampaignSwitchingPage />} />
              <Route path="/services" element={<ServiceListPage />} />
              <Route
                path="/services/user"
                element={
                  <React.Suspense fallback={<div className="minimal-text">Loading...</div>}>
                    <UserServicePage />
                  </React.Suspense>
                }
              />
            </Routes>
          </main>
        </div>
      </CampaignProvider>
    </Router>
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
            {campaigns.map((campaign, index) => {
              const isCurrentCampaign =
                currentCampaign.campaignId === campaign.id ||
                currentCampaign.campaignId === campaign.campaignId ||
                currentCampaign.id === campaign.id;

              return (
                <div
                  key={campaign.id || index}
                  className={`minimal-card ${isCurrentCampaign ? 'active' : ''}`}
                >
                  <div className="minimal-card-header">
                    <div className="minimal-card-title">
                      {campaign.title || campaign.name || `Campaign ${campaign.id}`}
                    </div>
                    <div className="minimal-card-meta">
                      <span className="minimal-card-id">ID: {campaign.id}</span>
                      <span className={`minimal-status ${campaign.status || 'unknown'}`}>
                        {campaign.status || 'UNKNOWN'}
                      </span>
                      {isCurrentCampaign && (
                        <span
                          className="minimal-status active"
                          style={{
                            background: '#fff',
                            color: '#000',
                            fontWeight: 'bold',
                            border: '1px solid #fff'
                          }}
                        >
                          CURRENT
                        </span>
                      )}
                    </div>
                  </div>

                  {campaign.description && (
                    <div className="minimal-card-description">{campaign.description}</div>
                  )}

                  {campaign.features && campaign.features.length > 0 && (
                    <div className="minimal-card-features">
                      <strong>Features:</strong> {campaign.features.join(', ')}
                    </div>
                  )}

                  <div className="minimal-card-actions">
                    <button
                      onClick={() =>
                        switchCampaignWithData &&
                        switchCampaignWithData(campaign, _response => {
                          // Campaign switch response handler
                        })
                      }
                      className="minimal-card-button"
                      disabled={isCurrentCampaign}
                      style={
                        isCurrentCampaign
                          ? {
                              background: '#333',
                              color: '#666',
                              cursor: 'not-allowed',
                              opacity: 0.6
                            }
                          : {}
                      }
                    >
                      {isCurrentCampaign ? 'CURRENT CAMPAIGN' : 'SWITCH TO CAMPAIGN'}
                    </button>
                  </div>
                </div>
              );
            })}
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
