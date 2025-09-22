import React, { useEffect, useState, useCallback } from 'react';
import AnalyticsDemo from '../components/AnalyticsDemo';
import { stateSyncManager, type SyncStatus, type SyncConflict } from '../utils/stateSyncManager';
import { indexedDBManager } from '../utils/indexedDBManager';
// import { stateManager } from '../utils/stateManager'; // Unused import

interface StateManagementDemoProps {
  className?: string;
}

export const StateManagementDemo: React.FC<StateManagementDemoProps> = ({ className }) => {
  const [syncStatus, setSyncStatus] = useState<SyncStatus>({
    wasm: false,
    indexedDB: false,
    serviceWorker: false,
    localStorage: false,
    sessionStorage: false,
    lastSync: 0,
    conflicts: 0
  });

  const [conflicts, setConflicts] = useState<SyncConflict[]>([]);
  const [isInitialized, setIsInitialized] = useState(false);
  const [currentTab, setCurrentTab] = useState<'analytics' | 'sync' | 'storage' | 'performance'>(
    'analytics'
  );

  // Initialize state management system
  useEffect(() => {
    const initialize = async () => {
      try {
        await stateSyncManager.initialize();
        setIsInitialized(true);
        updateSyncStatus();
      } catch (error) {
        console.error('Failed to initialize state management:', error);
      }
    };

    initialize();
  }, []);

  // Update sync status periodically
  useEffect(() => {
    if (!isInitialized) return;

    const interval = setInterval(async () => {
      await updateSyncStatus();
    }, 5000);

    return () => clearInterval(interval);
  }, [isInitialized]);

  const updateSyncStatus = useCallback(async () => {
    try {
      const status = await stateSyncManager.getSyncStatus();
      const conflicts = await stateSyncManager.getConflicts();
      setSyncStatus(status);
      setConflicts(conflicts);
    } catch (error) {
      console.error('Failed to update sync status:', error);
    }
  }, []);

  const handleClearConflicts = useCallback(async () => {
    try {
      await stateSyncManager.clearConflicts();
      await updateSyncStatus();
    } catch (error) {
      console.error('Failed to clear conflicts:', error);
    }
  }, [updateSyncStatus]);

  const handleForceSync = useCallback(async () => {
    try {
      await stateSyncManager.initialize();
      await updateSyncStatus();
    } catch (error) {
      console.error('Failed to force sync:', error);
    }
  }, [updateSyncStatus]);

  const handleClearStorage = useCallback(async () => {
    try {
      // Clear all storage layers
      if (typeof window !== 'undefined' && (window as any).clearAllStorage) {
        await (window as any).clearAllStorage();
      }

      // Clear IndexedDB
      await indexedDBManager.cleanupOldData(0); // Clear all data

      // Clear browser storage
      localStorage.clear();
      sessionStorage.clear();

      await updateSyncStatus();
    } catch (error) {
      console.error('Failed to clear storage:', error);
    }
  }, [updateSyncStatus]);

  if (!isInitialized) {
    return (
      <div className="state-management-demo loading">
        <div className="loading-spinner">
          <div className="spinner"></div>
          <p>Initializing Multi-Layer State Management System...</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`state-management-demo ${className || ''}`}>
      <div className="demo-header">
        <h1>Multi-Layer State Management Demo</h1>
        <p>Real-time analytics with WASM, IndexedDB, Service Worker, and Browser Storage</p>
      </div>

      <div className="demo-tabs">
        <button
          className={`tab ${currentTab === 'analytics' ? 'active' : ''}`}
          onClick={() => setCurrentTab('analytics')}
        >
          Real-time Analytics
        </button>
        <button
          className={`tab ${currentTab === 'sync' ? 'active' : ''}`}
          onClick={() => setCurrentTab('sync')}
        >
          Sync Status
        </button>
        <button
          className={`tab ${currentTab === 'storage' ? 'active' : ''}`}
          onClick={() => setCurrentTab('storage')}
        >
          Storage Layers
        </button>
        <button
          className={`tab ${currentTab === 'performance' ? 'active' : ''}`}
          onClick={() => setCurrentTab('performance')}
        >
          Performance
        </button>
      </div>

      <div className="demo-content">
        {currentTab === 'analytics' && <AnalyticsDemo />}

        {currentTab === 'sync' && (
          <div className="sync-panel">
            <div className="sync-header">
              <h2>Synchronization Status</h2>
              <div className="sync-actions">
                <button onClick={handleForceSync} className="btn btn-primary">
                  Force Sync
                </button>
                <button onClick={handleClearConflicts} className="btn btn-secondary">
                  Clear Conflicts
                </button>
              </div>
            </div>

            <div className="sync-status-grid">
              <div className={`status-card ${syncStatus.wasm ? 'connected' : 'disconnected'}`}>
                <h3>WASM Memory</h3>
                <div className="status-indicator">
                  <div className={`indicator ${syncStatus.wasm ? 'green' : 'red'}`}></div>
                  <span>{syncStatus.wasm ? 'Connected' : 'Disconnected'}</span>
                </div>
              </div>

              <div className={`status-card ${syncStatus.indexedDB ? 'connected' : 'disconnected'}`}>
                <h3>IndexedDB</h3>
                <div className="status-indicator">
                  <div className={`indicator ${syncStatus.indexedDB ? 'green' : 'red'}`}></div>
                  <span>{syncStatus.indexedDB ? 'Connected' : 'Disconnected'}</span>
                </div>
              </div>

              <div
                className={`status-card ${syncStatus.serviceWorker ? 'connected' : 'disconnected'}`}
              >
                <h3>Service Worker</h3>
                <div className="status-indicator">
                  <div className={`indicator ${syncStatus.serviceWorker ? 'green' : 'red'}`}></div>
                  <span>{syncStatus.serviceWorker ? 'Active' : 'Inactive'}</span>
                </div>
              </div>

              <div
                className={`status-card ${syncStatus.localStorage ? 'connected' : 'disconnected'}`}
              >
                <h3>Local Storage</h3>
                <div className="status-indicator">
                  <div className={`indicator ${syncStatus.localStorage ? 'green' : 'red'}`}></div>
                  <span>{syncStatus.localStorage ? 'Available' : 'Unavailable'}</span>
                </div>
              </div>

              <div
                className={`status-card ${syncStatus.sessionStorage ? 'connected' : 'disconnected'}`}
              >
                <h3>Session Storage</h3>
                <div className="status-indicator">
                  <div className={`indicator ${syncStatus.sessionStorage ? 'green' : 'red'}`}></div>
                  <span>{syncStatus.sessionStorage ? 'Available' : 'Unavailable'}</span>
                </div>
              </div>

              <div className="status-card">
                <h3>Last Sync</h3>
                <div className="status-value">
                  {syncStatus.lastSync
                    ? new Date(syncStatus.lastSync).toLocaleTimeString()
                    : 'Never'}
                </div>
              </div>
            </div>

            {conflicts.length > 0 && (
              <div className="conflicts-panel">
                <h3>Sync Conflicts ({conflicts.length})</h3>
                <div className="conflicts-list">
                  {conflicts.map((conflict, index) => (
                    <div key={index} className="conflict-item">
                      <div className="conflict-layer">{conflict.layer}</div>
                      <div className="conflict-key">{conflict.key}</div>
                      <div className="conflict-values">
                        <div className="conflict-local">
                          <strong>Local:</strong> {conflict.localValue}
                        </div>
                        <div className="conflict-remote">
                          <strong>Remote:</strong> {conflict.remoteValue}
                        </div>
                      </div>
                      <div className="conflict-resolution">Resolution: {conflict.resolution}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {currentTab === 'storage' && (
          <div className="storage-panel">
            <div className="storage-header">
              <h2>Storage Layers</h2>
              <button onClick={handleClearStorage} className="btn btn-danger">
                Clear All Storage
              </button>
            </div>

            <div className="storage-layers">
              <div className="storage-layer">
                <h3>WASM Memory Pools</h3>
                <p>High-performance in-memory state with memory pool optimization</p>
                <div className="layer-features">
                  <span className="feature">Memory Pools</span>
                  <span className="feature">Zero-copy Operations</span>
                  <span className="feature">Concurrent Access</span>
                </div>
              </div>

              <div className="storage-layer">
                <h3>IndexedDB</h3>
                <p>Advanced browser database for complex queries and persistence</p>
                <div className="layer-features">
                  <span className="feature">Complex Queries</span>
                  <span className="feature">Indexed Search</span>
                  <span className="feature">Large Data Sets</span>
                </div>
              </div>

              <div className="storage-layer">
                <h3>Service Worker Cache</h3>
                <p>Offline capabilities and background synchronization</p>
                <div className="layer-features">
                  <span className="feature">Offline Support</span>
                  <span className="feature">Background Sync</span>
                  <span className="feature">Push Notifications</span>
                </div>
              </div>

              <div className="storage-layer">
                <h3>Browser Storage</h3>
                <p>Session and persistent storage for user state</p>
                <div className="layer-features">
                  <span className="feature">Session Persistence</span>
                  <span className="feature">User Preferences</span>
                  <span className="feature">Quick Access</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {currentTab === 'performance' && (
          <div className="performance-panel">
            <h2>Performance Metrics</h2>
            <div className="performance-grid">
              <div className="metric-card">
                <h3>Memory Usage</h3>
                <div className="metric-value">
                  {((performance as any).memory?.usedJSHeapSize / 1024 / 1024).toFixed(1)} MB
                </div>
                <div className="metric-label">JavaScript Heap</div>
              </div>

              <div className="metric-card">
                <h3>Storage Quota</h3>
                <div className="metric-value">
                  {navigator.storage && 'estimate' in navigator.storage ? 'Available' : 'Unknown'}
                </div>
                <div className="metric-label">Browser Storage</div>
              </div>

              <div className="metric-card">
                <h3>WebAssembly</h3>
                <div className="metric-value">
                  {typeof WebAssembly !== 'undefined' ? 'Supported' : 'Not Supported'}
                </div>
                <div className="metric-label">WASM Support</div>
              </div>

              <div className="metric-card">
                <h3>Service Worker</h3>
                <div className="metric-value">
                  {'serviceWorker' in navigator ? 'Supported' : 'Not Supported'}
                </div>
                <div className="metric-label">SW Support</div>
              </div>
            </div>
          </div>
        )}
      </div>

      <style>{`
        .state-management-demo {
          min-height: 100vh;
          background: #0a0a0a;
          color: #ffffff;
        }

        .loading {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100vh;
        }

        .loading-spinner {
          text-align: center;
        }

        .spinner {
          width: 40px;
          height: 40px;
          border: 4px solid #333;
          border-top: 4px solid #007bff;
          border-radius: 50%;
          animation: spin 1s linear infinite;
          margin: 0 auto 1rem;
        }

        @keyframes spin {
          0% {
            transform: rotate(0deg);
          }
          100% {
            transform: rotate(360deg);
          }
        }

        .demo-header {
          padding: 2rem;
          text-align: center;
          background: #1a1a1a;
          border-bottom: 1px solid #333;
        }

        .demo-header h1 {
          margin: 0 0 0.5rem 0;
          font-size: 2.5rem;
          background: linear-gradient(45deg, #007bff, #00ff88);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
        }

        .demo-header p {
          margin: 0;
          color: #aaa;
          font-size: 1.1rem;
        }

        .demo-tabs {
          display: flex;
          background: #1a1a1a;
          border-bottom: 1px solid #333;
        }

        .tab {
          flex: 1;
          padding: 1rem 2rem;
          background: none;
          border: none;
          color: #aaa;
          cursor: pointer;
          transition: all 0.2s;
          border-bottom: 3px solid transparent;
        }

        .tab:hover {
          color: #ffffff;
          background: #2a2a2a;
        }

        .tab.active {
          color: #007bff;
          border-bottom-color: #007bff;
          background: #2a2a2a;
        }

        .demo-content {
          flex: 1;
          overflow: hidden;
        }

        .sync-panel,
        .storage-panel,
        .performance-panel {
          padding: 2rem;
        }

        .sync-header,
        .storage-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 2rem;
        }

        .sync-header h2,
        .storage-header h2 {
          margin: 0;
        }

        .sync-actions {
          display: flex;
          gap: 1rem;
        }

        .btn {
          padding: 0.5rem 1rem;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-weight: 500;
          transition: all 0.2s;
        }

        .btn-primary {
          background: #007bff;
          color: white;
        }

        .btn-primary:hover {
          background: #0056b3;
        }

        .btn-secondary {
          background: #6c757d;
          color: white;
        }

        .btn-secondary:hover {
          background: #545b62;
        }

        .btn-danger {
          background: #dc3545;
          color: white;
        }

        .btn-danger:hover {
          background: #c82333;
        }

        .sync-status-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
          gap: 1rem;
          margin-bottom: 2rem;
        }

        .status-card {
          background: #2a2a2a;
          padding: 1.5rem;
          border-radius: 8px;
          border: 1px solid #333;
        }

        .status-card.connected {
          border-color: #00ff88;
        }

        .status-card.disconnected {
          border-color: #dc3545;
        }

        .status-card h3 {
          margin: 0 0 1rem 0;
          color: #ffffff;
        }

        .status-indicator {
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }

        .indicator {
          width: 12px;
          height: 12px;
          border-radius: 50%;
        }

        .indicator.green {
          background: #00ff88;
        }

        .indicator.red {
          background: #dc3545;
        }

        .status-value {
          font-size: 1.2rem;
          color: #00ff88;
          font-weight: bold;
        }

        .conflicts-panel {
          background: #2a2a2a;
          padding: 1.5rem;
          border-radius: 8px;
          border: 1px solid #dc3545;
        }

        .conflicts-panel h3 {
          margin: 0 0 1rem 0;
          color: #dc3545;
        }

        .conflicts-list {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }

        .conflict-item {
          background: #1a1a1a;
          padding: 1rem;
          border-radius: 4px;
          border-left: 4px solid #dc3545;
        }

        .conflict-layer {
          font-weight: bold;
          color: #dc3545;
          margin-bottom: 0.5rem;
        }

        .conflict-key {
          color: #aaa;
          margin-bottom: 0.5rem;
        }

        .conflict-values {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 1rem;
          margin-bottom: 0.5rem;
        }

        .conflict-local,
        .conflict-remote {
          font-size: 0.9rem;
        }

        .conflict-resolution {
          color: #007bff;
          font-size: 0.9rem;
        }

        .storage-layers {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 1.5rem;
        }

        .storage-layer {
          background: #2a2a2a;
          padding: 1.5rem;
          border-radius: 8px;
          border: 1px solid #333;
        }

        .storage-layer h3 {
          margin: 0 0 1rem 0;
          color: #007bff;
        }

        .storage-layer p {
          margin: 0 0 1rem 0;
          color: #aaa;
          line-height: 1.5;
        }

        .layer-features {
          display: flex;
          flex-wrap: wrap;
          gap: 0.5rem;
        }

        .feature {
          background: #1a1a1a;
          padding: 0.25rem 0.75rem;
          border-radius: 12px;
          font-size: 0.8rem;
          color: #00ff88;
        }

        .performance-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 1.5rem;
        }

        .metric-card {
          background: #2a2a2a;
          padding: 1.5rem;
          border-radius: 8px;
          text-align: center;
          border: 1px solid #333;
        }

        .metric-card h3 {
          margin: 0 0 1rem 0;
          color: #ffffff;
        }

        .metric-value {
          font-size: 2rem;
          font-weight: bold;
          color: #00ff88;
          margin-bottom: 0.5rem;
        }

        .metric-label {
          color: #aaa;
          font-size: 0.9rem;
        }
      `}</style>
    </div>
  );
};

export default StateManagementDemo;
