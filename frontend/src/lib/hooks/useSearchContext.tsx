import React, { createContext, useContext } from 'react';
import { useSearch } from './useSearch';

// Define the shape of the context
export const SearchContext = createContext<ReturnType<typeof useSearch> | null>(null);

export const SearchProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  // Centralized, single source of truth for search state
  const search = useSearch();
  return <SearchContext.Provider value={search}>{children}</SearchContext.Provider>;
};

// Custom hook for consuming the context with type safety
export function useSearchContext() {
  const ctx = useContext(SearchContext);
  if (!ctx) throw new Error('useSearchContext must be used within a SearchProvider');
  return ctx;
}
