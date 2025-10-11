import { useState, useCallback, useRef, useEffect } from 'react';
import { useEventStore } from '../store/stores/eventStore';
import { useMetadata } from '../store/hooks/useMetadata';
import { useCampaignStore } from '../store/stores/campaignStore';

export interface Campaign {
  id: string | number;
  name: string;
  slug: string;
  title?: string;
  description?: string;
  status: 'active' | 'inactive' | 'draft' | 'archived';
  features: string[];
  tags: string[];
  createdAt?: string;
  updatedAt?: string;
  about?: {
    order?: Array<{
      title?: string;
      subtitle?: string;
      p?: string;
      list?: string[];
    }>;
  };
  ui_content?: {
    banner?: string;
    cta?: string;
    lead_form?: {
      fields?: string[];
      submit_text?: string;
    };
    // Additional UI content fields from actual data
    [key: string]: any;
  };
  broadcast_enabled?: boolean;
  channels?: string[];
  i18n_keys?: string[];
  focus?: string;
  inos_enabled?: boolean;
  ranking_formula?: string;
  start_date?: string;
  end_date?: string;
  owner_id?: string;
  master_id?: number;
  master_uuid?: string;
  // Additional fields from actual backend response
  campaign_id?: string | number;
  total?: number;
  limit?: number;
  offset?: number;
  source?: string;
  user_id?: string;
  correlationId?: string;
}

export interface UseCampaignsReturn {
  campaigns: Campaign[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  search: (query: string) => Promise<Campaign[]>;
  getActiveCampaigns: () => Promise<Campaign[]>;
}

export function useCampaigns(): UseCampaignsReturn {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { emitEvent } = useEventStore();
  const { metadata } = useMetadata();
  const { updateCampaignsFromResponse } = useCampaignStore();

  // Get campaigns from the campaign store
  const campaigns = useCampaignStore(state => state.campaigns);

  // Use ref to track actual loading state to prevent race conditions
  const isLoadingRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const loadCampaigns = useCallback(async () => {
    console.log('[useCampaigns] Starting dual-request approach: list + state');

    // Prevent duplicate requests using ref to avoid race conditions
    if (isLoadingRef.current) {
      console.log('[useCampaigns] Request already in progress, skipping duplicate');
      return;
    }

    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }

    try {
      isLoadingRef.current = true;
      setLoading(true);
      setError(null);

      // Generate correlation IDs for tracking
      const listCorrelationId = `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const stateCorrelationId = `corr_${Date.now() + 1}_${Math.random().toString(36).substr(2, 9)}`;

      let listCompleted = false;
      let stateCompleted = false;
      let hasSetCurrentCampaign = false;

      const checkCompletion = () => {
        if (listCompleted && stateCompleted && !hasSetCurrentCampaign) {
          console.log('[useCampaigns] Both requests completed, finalizing initialization');
          setLoading(false);
          setError(null);
          isLoadingRef.current = false;
        }
      };

      // Request 1: Campaign List
      emitEvent(
        {
          type: 'campaign:list:v1:requested',
          correlation_id: listCorrelationId,
          payload: {
            limit: 50,
            offset: 0
          },
          metadata
        },
        response => {
          console.log('[useCampaigns] Campaign list response:', response.type);

          if (response.type === 'campaign:list:v1:success') {
            console.log('[useCampaigns] Campaign list success, updating campaigns');
            updateCampaignsFromResponse(response.payload);
            listCompleted = true;
            checkCompletion();
          } else {
            console.error('[useCampaigns] Campaign list error:', response);
            listCompleted = true;
            checkCompletion();
          }
        }
      );

      // Request 2: Default Campaign State (always request this for robust initialization)
      const stateMetadata = {
        ...metadata,
        campaign: {
          ...(metadata.campaign || {}),
          campaignId: 'ovasabi_website', // Ensure we request the default campaign state
          slug: 'ovasabi_website'
        }
      };

      emitEvent(
        {
          type: 'campaign:state:v1:requested',
          correlation_id: stateCorrelationId,
          payload: {},
          metadata: stateMetadata
        },
        response => {
          console.log('[useCampaigns] Campaign state response:', response.type);

          if (response.type === 'campaign:state:v1:success') {
            console.log('[useCampaigns] Default campaign state received:', response.payload);

            // Set the campaign as current with full state data
            if (response.payload && !hasSetCurrentCampaign) {
              console.log('[useCampaigns] Setting default campaign as current with full state');
              useCampaignStore.getState().switchCampaignWithData(response.payload);
              hasSetCurrentCampaign = true;
            }

            stateCompleted = true;
            checkCompletion();
          } else {
            console.error('[useCampaigns] Default campaign state request failed:', response);
            stateCompleted = true;
            checkCompletion();
          }
        }
      );

      // Set timeout for both requests
      timeoutRef.current = setTimeout(() => {
        console.log('[useCampaigns] Dual request timeout after 10 seconds');
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

      console.error('[useCampaigns] Error in dual-request approach:', err);
      setError('Failed to load campaigns');
      setLoading(false);
      isLoadingRef.current = false;
    }
  }, [emitEvent, metadata, updateCampaignsFromResponse]);

  const refresh = useCallback(async () => {
    await loadCampaigns();
  }, [loadCampaigns]);

  const search = useCallback(
    async (query: string): Promise<Campaign[]> => {
      const lowerQuery = query.toLowerCase();
      return campaigns.filter(
        c =>
          c.name.toLowerCase().includes(lowerQuery) ||
          c.slug.toLowerCase().includes(lowerQuery) ||
          (c.title && c.title.toLowerCase().includes(lowerQuery)) ||
          (c.description && c.description.toLowerCase().includes(lowerQuery))
      );
    },
    [campaigns]
  );

  const getActiveCampaigns = useCallback(async (): Promise<Campaign[]> => {
    return campaigns.filter(c => c.status === 'active');
  }, [campaigns]);

  // Load campaigns on mount
  useEffect(() => {
    loadCampaigns();
  }, [loadCampaigns]);

  return {
    campaigns,
    loading,
    error,
    refresh,
    search,
    getActiveCampaigns
  };
}
