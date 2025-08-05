import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  initializeGlobalState,
  useEventHistory,
  useConnectionStatus,
  useMetadata,
  useGlobalStore
} from './store/global';
import { LeadFormWithStream } from './components/LeadFormWithStream';

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

function App() {
  // Initialize the global state and WASM bridge connection
  useEffect(() => {
    const cleanup = initializeGlobalState();
    return cleanup;
  }, []);

  // Fetch campaigns from default_campaign.json and optionally from backend API (seeded DB)
  const [campaigns, setCampaigns] = useState([
    { id: 0, slug: 'ovasabi_website', name: 'Ovasabi Website' }
  ]);
  useEffect(() => {
    fetch('/api/campaigns')
      .then(res => res.json())
      .then(data => {
        if (Array.isArray(data) && data.length > 0) {
          // Defensive: ensure id, slug, name are present and fallback if missing
          setCampaigns(
            data
              .map((c, idx) => ({
                id: typeof c.id === 'number' ? c.id : idx,
                slug: c.slug || `campaign_${idx}`,
                name: c.title || c.name || c.slug || `Campaign ${idx + 1}`
              }))
              .sort((a, b) => a.id - b.id)
          );
        }
      })
      .catch(() => {
        setCampaigns([{ id: 0, slug: 'ovasabi_website', name: 'Ovasabi Website' }]);
      });
  }, []);

  const { metadata } = useMetadata();
  // Defensive: only use switchCampaign if present in store
  const switchCampaign = useGlobalStore(state =>
    typeof state.switchCampaign === 'function' ? state.switchCampaign : undefined
  );

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>OVASABI Search Platform</h1>
      <ConnectionStatus />
      <SearchInterface />
      <EventHistory />
      <MetadataDisplay />
      <LiveWasmMetadataDisplay />
      <LeadFormWithStream />
      {/* Active Campaign List */}
      <div style={{ marginTop: '40px', borderTop: '1px solid #eee', paddingTop: '20px' }}>
        <h3>Active Campaigns</h3>
        <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
          {campaigns.map(c => (
            <button
              key={c.id}
              onClick={() => switchCampaign && switchCampaign(c.id, c.slug)}
              style={{
                padding: '10px 18px',
                borderRadius: '6px',
                border:
                  metadata.campaign?.campaignId === c.id ? '2px solid #007bff' : '1px solid #ccc',
                background: metadata.campaign?.campaignId === c.id ? '#e3f2fd' : '#fff',
                color: '#333',
                fontWeight: metadata.campaign?.campaignId === c.id ? 'bold' : 'normal',
                cursor: switchCampaign ? 'pointer' : 'not-allowed',
                marginBottom: '8px',
                opacity: switchCampaign ? 1 : 0.5,
                minWidth: '180px',
                textAlign: 'left',
                boxShadow: metadata.campaign?.campaignId === c.id ? '0 2px 8px #007bff22' : 'none',
                transition: 'border 0.2s, background 0.2s'
              }}
              disabled={!switchCampaign}
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
            </button>
          ))}
        </div>
        {!switchCampaign && (
          <div style={{ color: '#dc3545', marginTop: '12px' }}>
            Campaign switching is not available (store missing switchCampaign).
          </div>
        )}
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
  const { connected, connecting, wasmReady, reconnectAttempts, isConnected } =
    useConnectionStatus();

  // Window state tracking
  const [documentHidden, setDocumentHidden] = useState(
    typeof document !== 'undefined' ? document.hidden : false
  );
  const [windowFocused, setWindowFocused] = useState(
    typeof document !== 'undefined' ? document.hasFocus() : true
  );

  const statusColor = useMemo(() => {
    if (isConnected) return '#4CAF50'; // Green
    if (connecting) return '#FF9800'; // Orange
    return '#F44336'; // Red
  }, [isConnected, connecting]);

  const statusText = useMemo(() => {
    if (isConnected) return 'Connected';
    if (connecting) return 'Connecting...';
    return `Disconnected (${reconnectAttempts} attempts)`;
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
        border: `2px solid ${statusColor}`
      }}
    >
      <h3 style={{ margin: '0 0 10px 0' }}>System Status</h3>
      <div style={{ display: 'flex', gap: '20px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ color: statusColor, fontWeight: 'bold', fontSize: '16px' }}>
          Status: {statusText}
        </div>
        <div>WASM: {wasmReady ? '✅ Ready' : '❌ Not Ready'}</div>
        <div>WebSocket: {connected ? '✅ Connected' : '❌ Disconnected'}</div>
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
