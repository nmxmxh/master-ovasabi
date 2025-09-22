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

// Removed verbose logging

// Global loading state to prevent multiple instances from loading simultaneously
let globalLoadingState = false;

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
    // If we already have campaigns and have loaded before, don't reload unless explicitly requested
    if (hasLoadedRef.current && campaigns.length > 0) {
      console.log('[CampaignProvider] Campaigns already loaded, skipping reload');
      return;
    }

    // Prevent duplicate requests using ref to avoid race conditions
    if (isLoadingRef.current) {
      console.log('[CampaignProvider] Request already in progress, skipping duplicate');
      return;
    }

    // Additional check: if we're already in the process of loading, don't start another load
    if (loading) {
      console.log('[CampaignProvider] Already loading, skipping duplicate request');
      return;
    }

    // Global loading state check to prevent multiple instances
    if (globalLoadingState) {
      console.log('[CampaignProvider] Global loading in progress, skipping duplicate');
      return;
    }

    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }

    try {
      // Set global loading state
      globalLoadingState = true;
      isLoadingRef.current = true;
      setLoading(true);
      setError(null);

      console.log('[CampaignProvider] Starting dual-request approach: list + state');

      // Generate correlation IDs for tracking
      const listCorrelationId = `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const stateCorrelationId = `corr_${Date.now() + 1}_${Math.random().toString(36).substr(2, 9)}`;

      let listCompleted = false;
      let stateCompleted = false;
      let hasSetCurrentCampaign = false;

      const checkCompletion = () => {
        if (listCompleted && stateCompleted && !hasSetCurrentCampaign) {
          console.log('[CampaignProvider] Both requests completed, finalizing initialization');
          setLoading(false);
          setError(null);
          isLoadingRef.current = false;
          hasLoadedRef.current = true;
          setIsInitialized(true);
          // Clear global loading state
          globalLoadingState = false;
        }
      };

      // Request 1: Campaign List
      emitEvent(
        {
          type: 'campaign:list:v1:requested',
          payload: {
            limit: 50,
            offset: 0
          },
          metadata: {
            ...metadata,
            correlation_id: listCorrelationId
          }
        },
        response => {
          console.log('[CampaignProvider] Campaign list response:', response.type);

          if (response.type === 'campaign:list:v1:success') {
            console.log('[CampaignProvider] Campaign list success, updating campaigns');
            updateCampaignsFromResponse(response.payload);
            listCompleted = true;
            checkCompletion();
          } else {
            console.error('[CampaignProvider] Campaign list error:', response);
            listCompleted = true;
            checkCompletion();
          }
        }
      );

      // Request 2: Default Campaign State (always request this for robust initialization)
      emitEvent(
        {
          type: 'campaign:state:v1:requested',
          payload: {},
          metadata: {
            ...metadata,
            correlation_id: stateCorrelationId,
            global_context: {
              campaign_id: 'ovasabi_website',
              user_id: metadata?.user?.userId || metadata?.userId || 'guest_unknown',
              device_id: metadata?.device?.deviceId || metadata?.deviceId || 'device_unknown',
              session_id: metadata?.session?.sessionId || metadata?.sessionId || 'session_unknown',
              correlation_id: stateCorrelationId,
              source: 'frontend'
            }
          }
        },
        response => {
          console.log('[CampaignProvider] Campaign state response:', response.type);

          if (response.type === 'campaign:state:v1:success') {
            console.log('[CampaignProvider] Default campaign state received:', response.payload);

            // Set the campaign as current with full state data
            if (response.payload && !hasSetCurrentCampaign) {
              console.log('[CampaignProvider] Setting default campaign as current with full state');
              useCampaignStore.getState().switchCampaignWithData(response.payload);
              hasSetCurrentCampaign = true;
            }

            stateCompleted = true;
            checkCompletion();
          } else {
            console.error('[CampaignProvider] Default campaign state request failed:', response);
            stateCompleted = true;
            checkCompletion();
          }
        }
      );

      // Set timeout for both requests
      timeoutRef.current = setTimeout(() => {
        console.log('[CampaignProvider] Dual request timeout after 10 seconds');
        setError('Request timeout');
        setLoading(false);
        isLoadingRef.current = false;
        timeoutRef.current = null;
        // Clear global loading state on timeout
        globalLoadingState = false;
      }, 10000);
    } catch (err) {
      // Clear timeout if there's an error
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }

      console.error('[CampaignProvider] Error in dual-request approach:', err);
      setError('Failed to load campaigns');
      setLoading(false);
      isLoadingRef.current = false;
      // Clear global loading state on error
      globalLoadingState = false;
    }
  }, [emitEvent, metadata, updateCampaignsFromResponse]);

  // Load campaigns when metadata is ready and not in loading state
  useEffect(() => {
    // Only load campaigns if metadata is properly initialized (not in loading state)
    const userId = metadata?.user?.userId || metadata?.userId;
    if (metadata && userId && userId !== 'loading') {
      console.log('[CampaignProvider] Metadata ready, loading campaigns');
      loadCampaigns();
    } else {
      console.log('[CampaignProvider] Metadata not ready yet, waiting...', {
        hasMetadata: !!metadata,
        userId: userId,
        isInitialized: isInitialized
      });
    }
  }, [metadata?.user?.userId, metadata?.userId, isInitialized]); // Remove loadCampaigns to prevent circular dependency

  // Listen for campaign list responses that might not match pending requests
  useEffect(() => {
    const handleCampaignListResponse = (event: any) => {
      if (event.type === 'campaign:list:v1:success') {
        // Received campaign list response via event listener

        // Only process if we're still loading (avoid duplicate processing)
        if (isLoadingRef.current) {
          // Processing fallback response

          // Clear timeout since we got a response
          if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
            // Timeout cleared in fallback
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
          // Ignoring fallback response - already processed
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
    // Manual refresh requested
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
