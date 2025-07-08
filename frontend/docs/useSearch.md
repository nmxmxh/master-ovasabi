# useSearch Hook Documentation

## Overview

The `useSearch` hook provides a comprehensive, type-safe interface for performing searches using WebSocket connections with protobuf support. It's been refactored to integrate seamlessly with the existing metadata and WebSocket infrastructure.

## Key Features

- **Type Safety**: Full TypeScript support with generated protobuf types
- **Metadata Integration**: Automatically uses campaign and user context from `useMetadata`
- **Binary WebSocket Support**: Handles both JSON and binary protobuf messages
- **Proto Reflection**: Dynamically discovers searchable fields from proto descriptors
- **Connection Management**: Auto-reconnection with retry capabilities
- **Flexible Configuration**: Customizable defaults and behavior

## Basic Usage

```typescript
import { useSearch } from '@/lib/hooks/useSearch';

function SearchComponent() {
  const {
    connected,
    loading,
    error,
    results,
    search,
    quickSearch,
    clearResults
  } = useSearch();

  const handleSearch = () => {
    search({
      query: 'example search',
      types: ['content', 'campaign'],
      pageSize: 20,
      pageNumber: 1
    });
  };

  return (
    <div>
      <button onClick={handleSearch} disabled={!connected || loading}>
        Search
      </button>
      {loading && <div>Searching...</div>}
      {error && <div>Error: {error}</div>}
      {results && <div>Found {results.results.length} results</div>}
    </div>
  );
}
```

## Advanced Configuration

```typescript
const searchHook = useSearch({
  defaultPageSize: 50,
  defaultTypes: ['content', 'user', 'talent'],
  wsPathTemplate: '/ws/search/{campaignId}/{userId}',
  descriptorsEndpoint: '/api/v2/proto/descriptors'
});
```

## API Reference

### Hook Options

```typescript
interface SearchConfig {
  defaultPageSize: number;        // Default: 20
  defaultTypes: string[];         // Default: ['content', 'campaign', 'user']
  wsPathTemplate: string;         // Default: '/ws/search/{campaignId}/{userId}'
  descriptorsEndpoint: string;    // Default: '/api/proto/descriptors'
}
```

### Search Request

```typescript
interface SearchRequest {
  query: string;                  // Search query string
  types: string[];               // Entity types to search
  pageSize: number;              // Results per page
  pageNumber: number;            // Page number (1-based)
  metadata?: Partial<ProtoMetadata>; // Optional metadata override
  campaignId?: string;           // Optional campaign override
}
```

### Search Response

```typescript
interface SearchResponse {
  results: SearchResult[];       // Array of search results
  total: number;                // Total number of results
  pageNumber: number;           // Current page number
  pageSize: number;             // Number of results per page
  metadata?: Metadata;          // Response metadata
}

interface SearchResult {
  id: string;                   // Entity ID
  entityType: string;           // Entity type (content, campaign, etc.)
  score: number;                // Relevance score
  fields?: { [key: string]: any }; // Key fields (title, snippet, etc.)
  metadata?: Metadata;          // Result metadata
}
```

### Return Value

```typescript
interface UseSearchReturn {
  // Connection state
  connected: boolean;
  loading: boolean;
  error: string | null;
  
  // Search state
  results: SearchResult | null;
  currentQuery: string;
  
  // Search actions
  search: (params: SearchRequest) => void;
  quickSearch: (query: string, types?: string[]) => void;
  searchPage: (pageNumber: number) => void;
  retry: () => void;
  clearResults: () => void;
  
  // Connection controls
  reconnect: () => void;
  close: () => void;
  
  // Proto reflection
  descriptorSet: ArrayBuffer | null;
  searchableFields: string[];
  
  // Context
  metadata: Metadata;
  config: SearchConfig;
}
```

## Search Methods

### `search(params: SearchRequest)`
Performs a full search with all parameters specified.

### `quickSearch(query: string, types?: string[])`
Convenience method for simple searches with default pagination.

### `searchPage(pageNumber: number)`
Searches the specified page using the last search parameters.

### `retry()`
Retries the last search request (useful after reconnection).

### `clearResults()`
Clears current search results and query.

## Proto Integration

The hook automatically:
- Loads proto descriptors via WebSocket or HTTP fallback
- Discovers searchable fields dynamically
- Builds type-safe requests using generated protobuf types
- Handles binary protobuf responses

## Metadata Context

The hook automatically includes:
- Campaign ID from metadata
- User/device identification
- Session context
- Device capabilities

## Error Handling

The hook handles:
- WebSocket connection failures
- Search request errors
- Proto descriptor loading failures
- Network disconnections with auto-retry

## Performance Optimizations

- Memoized WebSocket URL construction
- Request deduplication
- Automatic descriptor caching
- Connection state management

## Migration from Legacy Hook

### Before (Legacy)
```typescript
const { search, results, loading } = useOldSearch();
search({ query: 'test', context: { locale: 'en' } });
```

### After (New)
```typescript
const { search, results, loading } = useSearch();
search({ 
  query: 'test', 
  types: ['content'], 
  pageSize: 20, 
  pageNumber: 1 
});
```

## Best Practices

1. **Use quickSearch for simple searches**: It applies sensible defaults
2. **Handle connection state**: Check `connected` before searching
3. **Implement retry logic**: Use the `retry` method for failed requests
4. **Clear results when appropriate**: Prevent stale data display
5. **Configure for your use case**: Set appropriate defaults via config

## Example: Complete Search Component

```typescript
import React, { useState } from 'react';
import { useSearch } from '@/lib/hooks/useSearch';

export function SearchInterface() {
  const [query, setQuery] = useState('');
  const [selectedTypes, setSelectedTypes] = useState(['content']);

  const {
    connected,
    loading,
    error,
    results,
    currentQuery,
    search,
    quickSearch,
    searchPage,
    clearResults,
    retry
  } = useSearch({
    defaultPageSize: 15,
    defaultTypes: ['content', 'campaign']
  });

  const handleSearch = () => {
    if (query.trim()) {
      search({
        query: query.trim(),
        types: selectedTypes,
        pageSize: 15,
        pageNumber: 1
      });
    }
  };

  return (
    <div className="search-interface">
      {/* Connection indicator */}
      <div className={`status ${connected ? 'connected' : 'disconnected'}`}>
        {connected ? '🟢 Connected' : '🔴 Disconnected'}
        {!connected && (
          <button onClick={retry}>Retry</button>
        )}
      </div>

      {/* Search form */}
      <div className="search-form">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search..."
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button onClick={handleSearch} disabled={!connected || loading}>
          {loading ? 'Searching...' : 'Search'}
        </button>
        <button onClick={clearResults}>Clear</button>
      </div>

      {/* Type filters */}
      <div className="type-filters">
        {['content', 'campaign', 'user', 'talent'].map(type => (
          <label key={type}>
            <input
              type="checkbox"
              checked={selectedTypes.includes(type)}
              onChange={(e) => {
                if (e.target.checked) {
                  setSelectedTypes([...selectedTypes, type]);
                } else {
                  setSelectedTypes(selectedTypes.filter(t => t !== type));
                }
              }}
            />
            {type}
          </label>
        ))}
      </div>

      {/* Error display */}
      {error && (
        <div className="error">
          {error}
          <button onClick={retry}>Retry</button>
        </div>
      )}

      {/* Results */}
      {results && (
        <div className="results">
          <div className="results-header">
            Results for "{currentQuery}" ({results.results.length})
            {results.total && <span> of {results.total} total</span>}
          </div>
          <div className="results-list">
            {results.results.map((result, index) => (
              <div key={result.id || index} className="result">
                <div className="result-type">{result.entityType}</div>
                <div className="result-score">Score: {result.score}</div>
                {result.fields && (
                  <div className="result-fields">
                    {Object.entries(result.fields).map(([key, value]) => (
                      <div key={key}>{key}: {String(value)}</div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
          
          {/* Pagination */}
          {results.total > (results.pageNumber * results.pageSize) && (
            <button onClick={() => searchPage(results.pageNumber + 1)}>
              Load More (Page {results.pageNumber + 1})
            </button>
          )}
        </div>
      )}
    </div>
  );
}
```
