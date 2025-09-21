import React, { createContext, useContext, useEffect, useRef } from 'react';
import { useEventStore } from '../store/stores/eventStore';
import { useMetadataStore } from '../store/stores/metadataStore';
import { useCampaignStore } from '../store/stores/campaignStore';

interface CampaignProviderContextType {
  campaigns: any[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
  isInitialized: boolean;
}

const CampaignProviderContext = createContext<CampaignProviderContextType | null>(null);

interface CampaignProviderProps {
  children: React.ReactNode;
}

export function CampaignProvider({ children }: CampaignProviderProps) {
  const { emitEvent } = useEventStore();
  const { metadata } = useMetadataStore();
  const { updateCampaignsFromResponse } = useCampaignStore();

  // Get campaigns from the campaign store (single source of truth)
  const campaigns = useCampaignStore(state => state.campaigns);

  // Local state for loading and error
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [isInitialized, setIsInitialized] = React.useState(false);

  // Use ref to track actual loading state to prevent race conditions
  const isLoadingRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hasLoadedRef = useRef(false);

  const loadCampaigns = React.useCallback(async () => {
    console.log('[CampaignProvider] loadCampaigns called, current loading state:', loading);
    console.log('[CampaignProvider] isLoadingRef.current:', isLoadingRef.current);
    console.log('[CampaignProvider] hasLoadedRef.current:', hasLoadedRef.current);

    // If we already have campaigns and have loaded before, don't reload unless explicitly requested
    if (hasLoadedRef.current && campaigns.length > 0) {
      console.log('[CampaignProvider] Campaigns already loaded, skipping duplicate request');
      return;
    }

    // Prevent duplicate requests using ref to avoid race conditions
    if (isLoadingRef.current) {
      console.log('[CampaignProvider] Request already in progress, skipping duplicate');
      return;
    }

    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }

    try {
      console.log('[CampaignProvider] Setting loading to true and clearing error');
      isLoadingRef.current = true;
      setLoading(true);
      setError(null);

      console.log('[CampaignProvider] Loading campaigns...');

      // Generate correlation ID for tracking
      const correlationId = `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      console.log('[CampaignProvider] Generated correlation ID:', correlationId);

      // Note: Campaign state will be included in the campaign list response
      // No need to make separate state request to avoid duplication

      // Emit campaign list request event
      emitEvent(
        {
          type: 'campaign:list:v1:requested',
          payload: {
            limit: 50,
            offset: 0
          },
          metadata: {
            ...metadata,
            correlation_id: correlationId
          }
        },
        response => {
          console.log('[CampaignProvider] Received response:', {
            type: response.type,
            correlationId: response.correlation_id,
            expectedCorrelationId: correlationId,
            payload: response.payload
          });

          // Clear timeout since we got a response
          if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
            console.log('[CampaignProvider] Cleared timeout on successful response');
          }

          if (response.type === 'campaign:list:v1:success') {
            console.log('[CampaignProvider] Campaign list success, updating campaigns');

            // Update campaigns in the store with the response payload
            updateCampaignsFromResponse(response.payload);

            // Update loading state and clear any errors
            setLoading(false);
            setError(null);
            isLoadingRef.current = false;
            hasLoadedRef.current = true;
            setIsInitialized(true);

            console.log('[CampaignProvider] Campaigns loaded successfully');
          } else if (response.type === 'campaign:list:v1:error') {
            console.error('[CampaignProvider] Campaign list error:', response);
            setError('Failed to load campaigns');
            setLoading(false);
            isLoadingRef.current = false;
          } else {
            console.warn('[CampaignProvider] Unexpected response type:', response.type, response);
            setLoading(false);
            isLoadingRef.current = false;
          }
        }
      );

      // Set timeout for request
      timeoutRef.current = setTimeout(() => {
        console.log('[CampaignProvider] Request timeout after 10 seconds');
        setError('Request timeout');
        setLoading(false);
        isLoadingRef.current = false;
        timeoutRef.current = null;
      }, 10000);
    } catch (err) {
      // Clear timeout if there's an error
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }

      const errorMessage = err instanceof Error ? err.message : 'Failed to load campaigns';
      setError(errorMessage);
      console.error('[CampaignProvider] Error loading campaigns:', err);
      setLoading(false);
      isLoadingRef.current = false;
    }
  }, [emitEvent, metadata, updateCampaignsFromResponse, campaigns.length]);

  // Load campaigns on mount only once
  useEffect(() => {
    console.log('[CampaignProvider] useEffect triggered - loading campaigns on mount');
    console.log('[CampaignProvider] Current state:', {
      loading,
      campaigns: campaigns.length,
      error
    });
    loadCampaigns();
  }, []); // Only run once on mount

  // Listen for campaign list responses that might not match pending requests
  useEffect(() => {
    const handleCampaignListResponse = (event: any) => {
      if (event.type === 'campaign:list:v1:success') {
        console.log(
          '[CampaignProvider] Received campaign list response via event listener (fallback):',
          event
        );

        // Only process if we're still loading (avoid duplicate processing)
        if (isLoadingRef.current) {
          console.log('[CampaignProvider] Processing fallback response - clearing loading state');

          // Clear timeout since we got a response
          if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
            console.log('[CampaignProvider] Cleared timeout in fallback response handler');
          }

          // Update campaigns in the store with the response payload
          updateCampaignsFromResponse(event.payload);

          // Update loading state and clear any errors
          setLoading(false);
          setError(null);
          isLoadingRef.current = false;
          hasLoadedRef.current = true;
          setIsInitialized(true);
        } else {
          console.log(
            '[CampaignProvider] Ignoring fallback response - already processed or not loading'
          );
        }
      }
    };

    // Listen for campaign list responses
    const unsubscribe = useEventStore.subscribe(state => {
      const events = state.events;
      const latestEvent = events[events.length - 1];
      if (latestEvent && latestEvent.type === 'campaign:list:v1:success') {
        handleCampaignListResponse(latestEvent);
      }
    });

    return () => {
      unsubscribe();
    };
  }, [updateCampaignsFromResponse]);

  const refresh = React.useCallback(() => {
    console.log('[CampaignProvider] Manual refresh requested');
    hasLoadedRef.current = false; // Allow reload on manual refresh
    loadCampaigns();
  }, [loadCampaigns]);

  const contextValue: CampaignProviderContextType = {
    campaigns,
    loading,
    error,
    refresh,
    isInitialized
  };

  return (
    <CampaignProviderContext.Provider value={contextValue}>
      {children}
    </CampaignProviderContext.Provider>
  );
}

export function useCampaignData() {
  const context = useContext(CampaignProviderContext);
  if (!context) {
    throw new Error('useCampaignData must be used within a CampaignProvider');
  }
  return context;
}
