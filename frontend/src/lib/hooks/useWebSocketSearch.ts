import { useEffect, useRef, useState, useCallback } from 'react';

export interface SearchRequest {
  query: string;
  types?: string[];
  page_size?: number;
  page_number?: number;
  metadata?: Record<string, any>;
}

export interface SearchResult {
  results: any[];
  total: number;
  page_number: number;
  page_size: number;
  metadata?: Record<string, any>;
}

export function useWebSocketSearch() {
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  const [results, setResults] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Connect on mount
  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8090/ws/ovasabi_website/guest_kr5zmyvq');
    wsRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onmessage = event => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'search.completed' || data.type === 'search.result') {
          setResults(data.payload || data);
          setLoading(false);
        }
      } catch (err) {
        setError('Failed to parse message');
        setLoading(false);
      }
    };
    return () => {
      ws.close();
    };
  }, []);

  // Send a search request
  const search = useCallback((params: SearchRequest) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setError('WebSocket not connected');
      return;
    }
    setLoading(true);
    setError(null);
    const message = {
      type: 'search',
      payload: {
        query: params.query,
        types: params.types || ['content'],
        page_size: params.page_size || 10,
        page_number: params.page_number || 1
      },
      metadata: params.metadata || {}
    };
    wsRef.current.send(JSON.stringify(message));
  }, []);

  return { connected, results, loading, error, search };
}
