import { useWasmBridge } from './useWasmBridge';
import { useProtoDescriptorDiscovery } from './useProtoDescriptorDiscovery';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { Metadata as ProtoMetadata } from '../../../protos/common/v1/metadata';
import type {
  SearchRequest as ProtoSearchRequest,
  SearchResponse as ProtoSearchResponse
} from '../../../protos/search/v1/search';
// ...existing code...
import { useMetadata } from './useMetadata';
// ...existing code...

// --- Enhanced search types ---
export type SearchRequest = Omit<ProtoSearchRequest, 'metadata' | 'campaignId'> & {
  /** Optional metadata override */
  metadata?: Partial<ProtoMetadata>;
  /** Optional campaign override */
  campaignId?: string;
};

export type SearchResult = ProtoSearchResponse;

// --- Search configuration ---
interface SearchConfig {
  /** Default page size for search results */
  defaultPageSize: number;
  /** Default search types */
  defaultTypes: string[];
  /** WebSocket endpoint path template */
  wsPathTemplate: string;
  /** HTTP descriptors endpoint */
  descriptorsEndpoint: string;
}

const DEFAULT_CONFIG: SearchConfig = {
  defaultPageSize: 20,
  defaultTypes: ['content', 'campaign', 'user'],
  wsPathTemplate: '/ws/search/{campaignId}/{userId}',
  descriptorsEndpoint: '/api/proto/descriptors'
};

// --- Enhanced search hook ---
export function useSearch(config: Partial<SearchConfig> = {}) {
  const searchConfig = { ...DEFAULT_CONFIG, ...config };

  // Get metadata context
  const metadata = useMetadata();

  // Search state
  const [results, setResults] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentQuery, setCurrentQuery] = useState<string>('');

  // Proto reflection state (from WASM/WS/HTTP)
  const [descriptorSet, setDescriptorSet] = useState<ArrayBuffer | null>(null);
  const [searchableFields, setSearchableFields] = useState<string[]>([]);
  const [, setDescriptorError] = useState<any>(null);

  // Use the new proto descriptor discovery hook (WASM/WS/HTTP)
  useProtoDescriptorDiscovery({
    // wsUrl: 'ws://localhost:8081/ws/proto/descriptors', // Optionally pass wsUrl for ws-gateway
    onBinary: data => {
      setDescriptorSet(data);
      setSearchableFields(extractSearchableFields(data));
    },
    onError: err => {
      setDescriptorError(err);
    },
    auto: true
  });

  // Request tracking
  const lastRequestRef = useRef<SearchRequest | null>(null);
  const requestIdRef = useRef<number>(0);

  // Build WebSocket URL from metadata
  // --- WASM Bridge integration ---
  const { connected, send } = useWasmBridge({
    autoConnect: true,
    onMessage: data => {
      // Handle search responses
      if (data.type === 'search.result' || data.type === 'search.completed') {
        setResults(data.payload || data);
        setLoading(false);
      } else if (data.type === 'search.error') {
        setError(data.message || 'Search failed');
        setLoading(false);
      } else if (data.type === 'get_proto_descriptors' && data.payload) {
        setDescriptorSet(data.payload);
        setSearchableFields(extractSearchableFields(data.payload));
      }
    }
  });

  // WebSocket connection with binary support
  // WASM bridge handles connection

  // WASM bridge handles proto descriptor discovery; HTTP fallback removed

  // Extract searchable fields from descriptor set
  const extractSearchableFields = (_descriptorData: ArrayBuffer): string[] => {
    // Placeholder implementation - in production, parse the FileDescriptorSet
    // and extract field names for SearchRequest message
    return ['query', 'types', 'pageSize', 'pageNumber', 'metadata', 'campaignId'];
  };

  // Build complete search request
  const buildSearchRequest = useCallback(
    (params: SearchRequest): ProtoSearchRequest => {
      const campaignId = params.campaignId || metadata.campaign?.campaignId || 'default';

      // Merge metadata with any overrides
      const requestMetadata: ProtoMetadata = {
        // Default metadata structure
        features: [],
        tags: [],
        categories: [],
        aiConfidence: 0,
        embeddingId: '',
        nexusChannel: 'search',
        sourceUri: '',
        // Merge with any provided metadata
        ...params.metadata
      };

      return {
        query: params.query,
        types: params.types.length > 0 ? params.types : searchConfig.defaultTypes,
        pageSize: params.pageSize || searchConfig.defaultPageSize,
        pageNumber: params.pageNumber || 1,
        metadata: requestMetadata,
        campaignId
      };
    },
    [metadata, searchConfig]
  );

  // Perform search
  const search = useCallback(
    (params: SearchRequest) => {
      if (!connected) {
        setError('WASM bridge not connected');
        return;
      }
      const requestId = ++requestIdRef.current;
      setLoading(true);
      setError(null);
      setCurrentQuery(params.query);
      const fullRequest = buildSearchRequest(params);
      lastRequestRef.current = params;
      const message = {
        type: 'search',
        id: requestId,
        payload: fullRequest,
        timestamp: Date.now()
      };
      send(message);
    },
    [connected, buildSearchRequest, send]
  );

  // Quick search with defaults
  const quickSearch = useCallback(
    (query: string, types?: string[]) => {
      search({
        query,
        types: types || searchConfig.defaultTypes,
        pageSize: searchConfig.defaultPageSize,
        pageNumber: 1
      });
    },
    [search, searchConfig]
  );

  // Paginated search
  const searchPage = useCallback(
    (pageNumber: number) => {
      if (lastRequestRef.current) {
        search({
          ...lastRequestRef.current,
          pageNumber
        });
      }
    },
    [search]
  );

  // Retry last search
  const retry = useCallback(() => {
    if (lastRequestRef.current) {
      search(lastRequestRef.current);
    }
  }, [search]);

  // Auto-retry on reconnection
  useEffect(() => {
    if (connected && lastRequestRef.current && error) {
      retry();
    }
  }, [connected, retry, error]);

  // Clear results
  const clearResults = useCallback(() => {
    setResults(null);
    setCurrentQuery('');
    lastRequestRef.current = null;
  }, []);

  return {
    // Connection state
    connected,
    loading,
    error,

    // Search state
    results,
    currentQuery,

    // Search actions
    search,
    quickSearch,
    searchPage,
    retry,
    clearResults,

    // Connection controls
    // reconnect, // WASM bridge handles reconnection internally
    close,

    // Proto reflection
    descriptorSet,
    searchableFields,

    // Context
    metadata,

    // Configuration
    config: searchConfig
  };
}
