import { useState } from 'react';
import PublicLayout from '../components/shared/layouts/public';
import { useWebSocketSearch } from '../lib/hooks/useWebSocketSearch';
import styled from 'styled-components';

const WS_URL = import.meta.env.VITE_WS_URL;

export default function SearchDemoPage() {
  const [query, setQuery] = useState('');
  const [typed, setTyped] = useState(false);
  const { connected, results, loading, error, search } = useWebSocketSearch(WS_URL);

  // Suggestive search: send search on input change (debounced in real app)
  const handleInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setQuery(value);
    setTyped(true);
    if (value.trim().length > 1) {
      search({ query: value, page_size: 5 });
    }
  };

  return (
    <PublicLayout>
      <Style.Container>
        <h1>Suggestive Search Demo</h1>
        <Style.SearchBox>
          <input
            type="text"
            placeholder="Type to search..."
            value={query}
            onChange={handleInput}
            disabled={!connected}
            autoFocus
          />
          {loading && <span className="loading">Searching...</span>}
        </Style.SearchBox>
        {error && <Style.Error>{error}</Style.Error>}
        {typed && results && results.results && results.results.length > 0 && (
          <Style.Suggestions>
            {results.results.map((item: any, idx: number) => (
              <li key={item.id || idx}>
                <strong>{item.fields?.title || item.id || 'Untitled'}</strong>
                {item.fields?.snippet && <p>{item.fields.snippet}</p>}
              </li>
            ))}
          </Style.Suggestions>
        )}
        {typed && results && results.results && results.results.length === 0 && !loading && (
          <Style.NoResults>No results found.</Style.NoResults>
        )}
      </Style.Container>
    </PublicLayout>
  );
}

const Style = {
  Container: styled.div`
    margin-top: 12dvh;
    width: 100%;
    max-width: 480px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2rem;
  `,
  SearchBox: styled.div`
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    input {
      width: 100%;
      padding: 1rem;
      font-size: 1.2rem;
      border-radius: 8px;
      border: 1px solid #ccc;
      margin-bottom: 0.5rem;
    }
    .loading {
      font-size: 0.95rem;
      color: #888;
    }
  `,
  Suggestions: styled.ul`
    width: 100%;
    list-style: none;
    padding: 0;
    margin: 0;
    border-radius: 8px;
    background: #f8f8fa;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
    li {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #eee;
      strong {
        font-size: 1.05rem;
        color: #222;
      }
      p {
        margin: 0.25rem 0 0 0;
        color: #666;
        font-size: 0.98rem;
      }
      &:last-child {
        border-bottom: none;
      }
    }
  `,
  Error: styled.div`
    color: #c00;
    margin-top: 1rem;
  `,
  NoResults: styled.div`
    color: #888;
    margin-top: 1rem;
  `
};
