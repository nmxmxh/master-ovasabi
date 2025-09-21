/**
 * Hook for managing campaigns using the campaign store
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useEventStore } from '../store/stores/eventStore';
import { useMetadata } from '../store/hooks/useMetadata';
import { useCampaignStore } from '../store/stores/campaignStore';

export interface Campaign {
  id: number | string;
  name: string;
  slug: string;
  title?: string;
  description?: string;
  status?: 'active' | 'inactive' | 'draft';
  features?: string[];
  tags?: string[];
  createdAt?: string;
  updatedAt?: string;
  // Additional fields from backend - matching actual data structure
  about?: {
    order?: Array<{
      p?: string;
      title?: string;
      type?: 'content' | 'list' | 'image' | 'video';
      list?: string[];
      subtitle?: string;
    }>;
  };
  ui_content?: {
    banner?: string;
    cta?: string;
    architecture_overview?: {
      description?: string;
      sections?: string[];
    };
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
  const [loading, setLoading] = useState(false); // Start as false to allow initial load
  const [error, setError] = useState<string | null>(null);
  const { emitEvent } = useEventStore();
  const { metadata } = useMetadata();
  const { updateCampaignFromResponse, updateCampaignsFromResponse, requestCampaignState } =
    useCampaignStore();

  // Get campaigns from the campaign store
  const campaigns = useCampaignStore(state => state.campaigns);

  // Use ref to track actual loading state to prevent race conditions
  const isLoadingRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const loadCampaigns = useCallback(async () => {
    console.log('[useCampaigns] loadCampaigns called, current loading state:', loading);
    console.log('[useCampaigns] isLoadingRef.current:', isLoadingRef.current);

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
      console.log('[useCampaigns] Setting loading to true and clearing error');
      isLoadingRef.current = true;
      setLoading(true);
      setError(null);

      console.log('[useCampaigns] Loading campaigns...');

      // Generate correlation ID for tracking
      const correlationId = `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      console.log('[useCampaigns] Generated correlation ID:', correlationId);

      // Request campaign state immediately when loading campaigns
      // This ensures we get the full campaign state before any potential WebSocket closures
      if (requestCampaignState) {
        console.log('[useCampaigns] Requesting campaign state immediately');
        requestCampaignState(0, response => {
          console.log('[useCampaigns] Campaign state response:', response);
        });
      }

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
          console.log('[useCampaigns] Received response:', {
            type: response.type,
            correlationId: response.correlation_id,
            expectedCorrelationId: correlationId,
            payload: response.payload
          });

          // Clear timeout since we got a response
          if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
            console.log('[useCampaigns] Cleared timeout on successful response');
          }

          if (response.type === 'campaign:list:v1:success') {
            // Try multiple possible data structures for campaign data
            const rawCampaigns =
              response.payload?.campaigns || response.payload?.data?.campaigns || [];

            console.log('[useCampaigns] Received campaign response:', {
              type: response.type,
              payload: response.payload,
              rawCampaigns: rawCampaigns,
              campaignCount: rawCampaigns.length
            });

            // Parse and normalize campaign data
            const parsedCampaigns = rawCampaigns.map((campaign: any, index: number) => {
              try {
                // Handle different possible data structures
                const parsed = typeof campaign === 'string' ? JSON.parse(campaign) : campaign;

                // Extract description from about.order if no direct description
                const extractDescription = (about: any): string => {
                  if (about?.order && Array.isArray(about.order)) {
                    const firstContent = about.order.find((item: any) => item.p);
                    return firstContent?.p || '';
                  }
                  return '';
                };

                const campaignData: Campaign = {
                  id: parsed.id || parsed.campaign_id || index,
                  name: parsed.name || parsed.slug || `Campaign ${index}`,
                  slug: parsed.slug || parsed.name || `campaign-${index}`,
                  title: parsed.title || parsed.name,
                  description: parsed.description || extractDescription(parsed.about) || '',
                  status: parsed.status || 'active',
                  features: Array.isArray(parsed.features) ? parsed.features : [],
                  tags: Array.isArray(parsed.tags) ? parsed.tags : [],
                  createdAt: parsed.created_at || parsed.createdAt,
                  updatedAt: parsed.updated_at || parsed.updatedAt,
                  // Additional fields - preserve all backend data
                  about: parsed.about,
                  ui_content: parsed.ui_content,
                  broadcast_enabled: Boolean(parsed.broadcast_enabled),
                  channels: Array.isArray(parsed.channels) ? parsed.channels : [],
                  i18n_keys: Array.isArray(parsed.i18n_keys) ? parsed.i18n_keys : [],
                  focus: parsed.focus,
                  inos_enabled: Boolean(parsed.inos_enabled),
                  ranking_formula: parsed.ranking_formula,
                  start_date: parsed.start_date,
                  end_date: parsed.end_date,
                  owner_id: parsed.owner_id,
                  master_id: parsed.master_id,
                  master_uuid: parsed.master_uuid,
                  // Response metadata
                  campaign_id: parsed.campaign_id,
                  total: parsed.total,
                  limit: parsed.limit,
                  offset: parsed.offset,
                  source: parsed.source,
                  user_id: parsed.user_id,
                  correlationId: parsed.correlationId
                };

                console.log('[useCampaigns] Parsed campaign data:', {
                  id: campaignData.id,
                  name: campaignData.name,
                  title: campaignData.title,
                  features: campaignData.features,
                  inos_enabled: campaignData.inos_enabled,
                  focus: campaignData.focus,
                  about: campaignData.about,
                  ui_content: campaignData.ui_content
                });

                return campaignData;
              } catch (error) {
                console.error('[useCampaigns] Error parsing campaign:', error, campaign);
                return {
                  id: index,
                  name: `Campaign ${index}`,
                  slug: `campaign-${index}`,
                  title: `Campaign ${index}`,
                  description: '',
                  status: 'active' as const,
                  features: [],
                  tags: []
                };
              }
            });

            console.log('[useCampaigns] Parsed campaigns:', parsedCampaigns);

            // Update campaigns in the store with the full response payload
            updateCampaignsFromResponse(response.payload);

            // Update campaign store with the first active campaign if available
            const activeCampaign = parsedCampaigns.find((c: Campaign) => c.status === 'active');
            if (activeCampaign) {
              updateCampaignFromResponse(activeCampaign);
            }

            setLoading(false);
            setError(null); // Clear any previous errors
            isLoadingRef.current = false;
          } else if (response.type === 'campaign:list:v1:error') {
            console.error('[useCampaigns] Campaign list error:', response);
            setError('Failed to load campaigns');
            setLoading(false);
            isLoadingRef.current = false;
          } else {
            console.warn('[useCampaigns] Unexpected response type:', response.type, response);
            setLoading(false);
            isLoadingRef.current = false;
          }
        }
      );

      // Set timeout for request
      timeoutRef.current = setTimeout(() => {
        console.log('[useCampaigns] Request timeout after 10 seconds');
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
      console.error('[useCampaigns] Error loading campaigns:', err);
      setLoading(false);
      isLoadingRef.current = false;
    }
  }, [emitEvent]);

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
    console.log('[useCampaigns] useEffect triggered - loading campaigns on mount');
    console.log('[useCampaigns] Current state:', { loading, campaigns: campaigns.length, error });
    loadCampaigns();
  }, []); // Only run once on mount

  // Listen for campaign list responses that might not match pending requests
  useEffect(() => {
    const handleCampaignListResponse = (event: any) => {
      if (event.type === 'campaign:list:v1:success') {
        console.log(
          '[useCampaigns] Received campaign list response via event listener (fallback):',
          event
        );

        // Only process if we're still loading (avoid duplicate processing)
        if (isLoadingRef.current) {
          console.log('[useCampaigns] Processing fallback response - clearing loading state');

          // Clear timeout since we got a response
          if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
            console.log('[useCampaigns] Cleared timeout in fallback response handler');
          }

          // Update campaigns in the store with the response payload
          updateCampaignsFromResponse(event.payload);

          // Update loading state and clear any errors
          setLoading(false);
          setError(null);
          isLoadingRef.current = false;
        } else {
          console.log(
            '[useCampaigns] Ignoring fallback response - already processed or not loading'
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
  }, []);

  return {
    campaigns,
    loading,
    error,
    refresh,
    search,
    getActiveCampaigns
  };
}

/**
 * Hook for a specific campaign - takes campaigns as parameter to avoid duplicate requests
 */
export function useCampaign(id: number, campaigns: Campaign[]) {
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadCampaign = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Find campaign from the campaigns list
      const foundCampaign = campaigns.find(c => c.id === id) || null;
      setCampaign(foundCampaign);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load campaign';
      setError(errorMessage);
      console.error('[useCampaign] Error loading campaign:', err);
    } finally {
      setLoading(false);
    }
  }, [id, campaigns]);

  useEffect(() => {
    if (campaigns.length > 0) {
      loadCampaign();
    }
  }, [loadCampaign]);

  return {
    campaign,
    loading,
    error,
    refresh: loadCampaign
  };
}
