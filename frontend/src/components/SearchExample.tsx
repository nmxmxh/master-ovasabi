import { useState, useEffect, useRef, useMemo } from 'react';
import { useWasmBridge, buildCanonicalMetadata } from '../lib/hooks/useWasmBridge';
import { useGlobalStore } from '../store/global';
import { useDate } from '../lib/hooks/useDate';

// Utility to deeply sanitize payload: remove functions, undefined, and ensure only primitives/objects/arrays
function sanitizePayload(obj: any): any {
  if (obj === null || typeof obj === 'undefined') return null;
  if (typeof obj === 'function') return undefined;
  if (typeof obj !== 'object') return obj;
  if (Array.isArray(obj)) return obj.map(sanitizePayload);
  const result: any = {};
  for (const key in obj) {
    if (!Object.prototype.hasOwnProperty.call(obj, key)) continue;
    const value = sanitizePayload(obj[key]);
    if (typeof value !== 'undefined') result[key] = value;
  }
  return result;
}

function logStatus(label: string, value: any) {
  // Enhanced logging with timestamp and grouping
  const time = new Date().toISOString();
  if (typeof value === 'object') {
    console.groupCollapsed(`[SearchExample][${time}] ${label}`);
    console.dir(value);
    console.groupEnd();
  } else {
    console.log(`[SearchExample][${time}] ${label}:`, value);
  }
}

export function SearchExample() {
  // Use the useDate hook, default to Africa/Lagos (GMT+1) for demo
  const date = useDate('Africa/Lagos');

  // Local UI state for the search form
  const [state, setState] = useState<{
    query: string;
    selectedTypes: string[];
    results: any[];
    loading: boolean;
    error: string | null;
    currentQuery: string | null;
    pageNumber: number;
    total: number;
    pageSize: number;
  }>({
    query: '',
    selectedTypes: [],
    results: [],
    loading: false,
    error: null,
    currentQuery: null,
    pageNumber: 1,
    total: 0,
    pageSize: 10
  });

  // Helper: display current time in GMT+1 and user's local time
  const nowLagos = date.now();
  const nowLocal = date.now().setZone(Intl.DateTimeFormat().resolvedOptions().timeZone);

  // Zustand global state and events
  const globalState = useGlobalStore();
  const eventTypes = useGlobalStore(state => state.eventTypes);
  const prevGlobalState = useRef(globalState);

  // WASM bridge
  const { connected, ready, sendNexusEvent } = useWasmBridge({
    onMessage: msg => {
      logStatus('Received message from backend', msg);
      // Canonical event handling: check for canonical event types
      if (msg.type && typeof msg.type === 'string') {
        if (msg.type === 'search:search:v1:completed') {
          logStatus('Search completed payload', msg.payload);
          setState(prev => ({
            ...prev,
            loading: false,
            results: msg.payload?.results || [],
            total: msg.payload?.total || 0,
            currentQuery: msg.payload?.query || '',
            error: null
          }));
        } else if (msg.type === 'search:search:v1:failed') {
          logStatus('Search failed payload', msg.payload);
          setState(prev => ({
            ...prev,
            loading: false,
            error: msg.payload?.error || 'Search failed'
          }));
        } else {
          logStatus('Other canonical event type', msg);
        }
      } else {
        logStatus('Non-canonical or unknown message type', msg);
      }
    }
  });

  // Canonical event types for search (fallback to defaults if not loaded)
  const canonicalTypes = useMemo(() => {
    if (eventTypes && eventTypes.length > 0) {
      // Only search-related canonical event types ending in :requested
      return eventTypes.filter(t => t.startsWith('search:') && t.endsWith(':requested'));
    }
    // Fallback to hardcoded canonical types if registry not loaded
    return ['search:search:v1:requested', 'search:suggest:v1:requested'];
  }, [eventTypes]);

  // Log global state changes
  useEffect(() => {
    if (prevGlobalState.current !== globalState) {
      logStatus('Global state changed', globalState);
      prevGlobalState.current = globalState;
    }
  }, [globalState]);

  useEffect(() => {
    if (!state.query) return;
    const handler = setTimeout(() => {
      handleSearch();
    }, 400);
    return () => clearTimeout(handler);
  }, [state.query, canonicalTypes]);

  // Main federated search handler with event type validation and canonical metadata
  const handleSearch = () => {
    if (!state.query.trim()) return;
    // Validate selected types against canonical event types
    const invalidTypes = state.selectedTypes.filter(t => !canonicalTypes.includes(t));
    if (invalidTypes.length > 0) {
      setState(prev => ({ ...prev, error: `Invalid type(s): ${invalidTypes.join(', ')}` }));
      logStatus('Invalid event types selected', invalidTypes);
      return;
    }
    setState(prev => ({
      ...prev,
      loading: true,
      results: [],
      error: null,
      currentQuery: prev.query.trim(),
      pageNumber: 1
    }));
    // Use canonical metadata builder from bridge
    const globalMetadata = useGlobalStore.getState().metadata;
    // For demo, send one event per selected canonical type
    state.selectedTypes.forEach(eventType => {
      const rawPayload = {
        query: state.query.trim(),
        type: eventType,
        pageSize: state.pageSize,
        pageNumber: 1
      };
      // Always include a correlation_id in metadata
      const correlationId = generateUUID();
      const outgoing = {
        eventId: correlationId,
        eventType,
        entityId: 'frontend',
        campaignId: 'ovasabi_website',
        payload: sanitizePayload(rawPayload),
        metadata: {
          ...buildCanonicalMetadata(globalMetadata),
          correlation_id: correlationId
        }
      };
      logStatus('Sending federated search nexus_event', outgoing);
      if (useGlobalStore.getState().emitEvent) {
        useGlobalStore.getState().emitEvent({
          type: outgoing.eventType,
          payload: outgoing.payload,
          metadata: outgoing.metadata
        });
      }
      sendNexusEvent(outgoing);
    });
  };

  // Quick search (minimal payload) with event type validation and canonical metadata
  const handleQuickSearch = () => {
    if (!state.query.trim()) return;
    const invalidTypes = state.selectedTypes.filter(t => !canonicalTypes.includes(t));
    if (invalidTypes.length > 0) {
      setState(prev => ({ ...prev, error: `Invalid type(s): ${invalidTypes.join(', ')}` }));
      logStatus('Invalid event types selected', invalidTypes);
      return;
    }
    setState(prev => ({
      ...prev,
      loading: true,
      results: [],
      error: null,
      currentQuery: prev.query.trim(),
      pageNumber: 1
    }));
    const globalMetadata = useGlobalStore.getState().metadata;
    // @ts-ignore: access internal helper
    // Use canonical metadata builder from bridge (direct import)
    state.selectedTypes.forEach(eventType => {
      const rawPayload = {
        query: state.query.trim(),
        type: eventType
      };
      const correlationId = generateUUID();
      const outgoing = {
        eventId: correlationId,
        eventType,
        entityId: 'frontend',
        campaignId: 'ovasabi_website',
        payload: sanitizePayload(rawPayload),
        metadata: {
          ...buildCanonicalMetadata(globalMetadata),
          correlation_id: correlationId
        }
      };
      logStatus('Sending quick search nexus_event', outgoing);
      if (useGlobalStore.getState().emitEvent) {
        useGlobalStore.getState().emitEvent({
          type: outgoing.eventType,
          payload: outgoing.payload,
          metadata: outgoing.metadata
        });
      }
      sendNexusEvent(outgoing);
    });
  };

  return (
    <div className="search-container" style={{ maxWidth: 800, margin: '0 auto', padding: 24 }}>
      <div
        className="search-header"
        style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
      >
        <h2>Federated Search Example</h2>
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'flex-end',
            fontWeight: 500
          }}
        >
          <div>
            {' '}
            Status: {connected ? '🟢 Connected' : '🔴 Disconnected'} | WASM:{' '}
            {ready ? 'Ready' : 'Not Ready'}
          </div>
          <div style={{ fontSize: 13, color: '#666', marginTop: 2 }}>
            <span>
              Time (GMT+1):{' '}
              {date.format(nowLagos, { format: 'yyyy LLL dd, HH:mm ZZZZ', zone: 'Africa/Lagos' })}
            </span>
            <br />
            <span>
              Your Timezone: {nowLocal.zoneName ?? ''} (
              {date.format(nowLocal, {
                format: 'yyyy LLL dd, HH:mm ZZZZ',
                zone: nowLocal.zoneName ?? undefined
              })}
              )
            </span>
          </div>
        </div>
      </div>

      <div
        className="search-form"
        style={{ margin: '24px 0', display: 'flex', flexDirection: 'column', gap: 16 }}
      >
        <input
          type="text"
          value={state.query}
          onChange={e => {
            const value = e.target.value;
            setState(prev => ({ ...prev, query: value }));
          }}
          placeholder="Enter search query..."
          autoComplete="off"
          style={{
            padding: 10,
            fontSize: 16,
            borderRadius: 4,
            border: '1px solid #bbb',
            width: '100%'
          }}
        />

        <div className="search-types" style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <span>Types:</span>
          {canonicalTypes.length > 0 ? (
            canonicalTypes.map(type => {
              // Show a user-friendly label (action segment)
              const label = type.split(':')[1];
              return (
                <label key={type} style={{ marginRight: 8 }}>
                  <input
                    type="checkbox"
                    checked={state.selectedTypes.includes(type)}
                    onChange={e => {
                      setState(prev => ({
                        ...prev,
                        selectedTypes: e.target.checked
                          ? [...prev.selectedTypes, type]
                          : prev.selectedTypes.filter(t => t !== type)
                      }));
                    }}
                  />
                  {label}
                </label>
              );
            })
          ) : (
            <span style={{ color: '#888' }}>No event types loaded</span>
          )}
        </div>

        <div className="search-actions" style={{ display: 'flex', gap: 12 }}>
          <button
            onClick={handleSearch}
            disabled={!connected || !ready || !state.query.trim()}
            style={{
              padding: '8px 18px',
              borderRadius: 4,
              border: '1px solid #bbb',
              background: '#f7f7f7',
              cursor: 'pointer'
            }}
          >
            Search
          </button>
          <button
            onClick={handleQuickSearch}
            disabled={!connected || !ready || !state.query.trim()}
            style={{
              padding: '8px 18px',
              borderRadius: 4,
              border: '1px solid #bbb',
              background: '#f7f7f7',
              cursor: 'pointer'
            }}
          >
            Quick Search
          </button>
        </div>
      </div>

      {state.loading && (
        <div className="loading" style={{ color: '#007bff', fontWeight: 500 }}>
          Searching...
        </div>
      )}
      {state.error && (
        <div className="error" style={{ color: 'red', fontWeight: 500 }}>
          Error: {state.error}
        </div>
      )}
      {state.currentQuery && (
        <div className="current-query" style={{ color: '#444', margin: '12px 0' }}>
          Showing results for: "{state.currentQuery}"
        </div>
      )}

      {/* Results Section - improved UI */}
      <div className="search-results-list" style={{ marginTop: 24 }}>
        {state.results && state.results.length > 0 ? (
          <div className="results-list-ui">
            <div
              className="results-header"
              style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
            >
              <h3 style={{ margin: 0 }}>
                Results <span style={{ color: '#888' }}>({state.results.length})</span>
              </h3>
              {state.total > 0 && <span style={{ color: '#888' }}>Total: {state.total}</span>}
            </div>
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {state.results.map((result, index) => (
                <li
                  key={result.id || index}
                  style={{
                    border: '1px solid #eee',
                    borderRadius: 6,
                    margin: '12px 0',
                    padding: 16,
                    background: '#fafbfc'
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between'
                    }}
                  >
                    <span style={{ fontWeight: 600, color: '#2d3748' }}>
                      {result.entityType || 'Result'}
                    </span>
                    <span style={{ color: '#888', fontSize: 13 }}>Score: {result.score}</span>
                  </div>
                  <div style={{ marginTop: 8 }}>
                    {result.fields && (
                      <table
                        style={{
                          width: '100%',
                          fontSize: 14,
                          background: 'white',
                          borderCollapse: 'collapse'
                        }}
                      >
                        <tbody>
                          {Object.entries(result.fields).map(([key, value]) => (
                            <tr key={key}>
                              <td
                                style={{
                                  color: '#666',
                                  padding: '2px 8px',
                                  width: 120,
                                  fontWeight: 500
                                }}
                              >
                                {key}
                              </td>
                              <td style={{ color: '#222', padding: '2px 8px' }}>{String(value)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </div>
                </li>
              ))}
            </ul>
            {/* Show pagination if there are more results */}
            {state.total > state.pageNumber * state.pageSize && (
              <div
                className="pagination"
                style={{ marginTop: 16, display: 'flex', alignItems: 'center', gap: 16 }}
              >
                <button
                  style={{
                    padding: '6px 18px',
                    borderRadius: 4,
                    border: '1px solid #bbb',
                    background: '#f7f7f7',
                    cursor: 'pointer'
                  }}
                  onClick={() => {
                    setState(prev => ({ ...prev, loading: true, pageNumber: prev.pageNumber + 1 }));
                    const outgoing = {
                      eventType: 'search.requested',
                      payload: {
                        data: {
                          eventType: 'search.requested',
                          query: state.currentQuery,
                          types: state.selectedTypes,
                          pageSize: state.pageSize,
                          pageNumber: state.pageNumber + 1
                        }
                      }
                    };
                    logStatus('Sending canonical pagination search.requested event', outgoing);
                    if (!ready) {
                      logStatus('WASM Bridge not ready, cannot send event', outgoing);
                      return;
                    }
                    sendNexusEvent(outgoing);
                  }}
                >
                  Load More (Page {state.pageNumber + 1})
                </button>
                <span className="pagination-info" style={{ color: '#888' }}>
                  Showing {state.results.length} of {state.total} results
                </span>
              </div>
            )}
          </div>
        ) : (
          <div style={{ color: '#888', textAlign: 'center', marginTop: 32 }}>
            {state.loading ? 'Searching...' : 'No results yet.'}
          </div>
        )}
      </div>

      {/* Debug Panel: Show global state, events, and search state */}
      <div
        className="debug-panel"
        style={{ marginTop: 40, background: '#f5f5f5', borderRadius: 8, padding: 16 }}
      >
        <h4 style={{ margin: '0 0 8px 0', color: '#333' }}>Debug Panel</h4>
        <div style={{ fontSize: 13, color: '#444', marginBottom: 8 }}>
          <b>Global State:</b>
          <pre
            style={{
              background: '#fff',
              borderRadius: 4,
              padding: 8,
              maxHeight: 200,
              overflow: 'auto'
            }}
          >
            {JSON.stringify(globalState, null, 2)}
          </pre>
        </div>
        <div style={{ fontSize: 13, color: '#444', marginBottom: 8 }}>
          <b>Search State:</b>
          <pre
            style={{
              background: '#fff',
              borderRadius: 4,
              padding: 8,
              maxHeight: 120,
              overflow: 'auto'
            }}
          >
            {JSON.stringify(state, null, 2)}
          </pre>
        </div>
        <div style={{ fontSize: 13, color: '#444', marginBottom: 8 }}>
          <b>Recent Events:</b>
          <pre
            style={{
              background: '#fff',
              borderRadius: 4,
              padding: 8,
              maxHeight: 120,
              overflow: 'auto'
            }}
          >
            {JSON.stringify(globalState.events.slice(-10), null, 2)}
          </pre>
        </div>
        <div style={{ fontSize: 13, color: '#444' }}>
          <b>Time (GMT+1):</b>{' '}
          {date.format(nowLagos, { format: 'yyyy LLL dd, HH:mm ZZZZ', zone: 'Africa/Lagos' })}
          <br />
          <b>Your Timezone:</b> {nowLocal.zoneName ?? ''} (
          {date.format(nowLocal, {
            format: 'yyyy LLL dd, HH:mm ZZZZ',
            zone: nowLocal.zoneName ?? undefined
          })}
          )
        </div>
      </div>
    </div>
  );
}

// Utility to generate a UUID (RFC4122 v4)
function generateUUID(): string {
  // Simple browser-safe UUID generator
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = crypto.getRandomValues(new Uint8Array(1))[0] % 16;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
