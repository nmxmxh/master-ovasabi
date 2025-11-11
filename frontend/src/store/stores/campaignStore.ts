import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { Campaign } from '../types/campaign';
import type { EventEnvelope } from '../types/events';
import { generateCorrelationIDSync } from '../../utils/wasmIdExtractor';
import { useEventStore } from './eventStore';
import { useMetadataStore } from './metadataStore';

// Helper to create a standardized event
const createEvent = (
  type: string,
  payload: Record<string, any>,
  campaignId?: string
): EventEnvelope => {
  const metadataStore = useMetadataStore.getState();
  const userId = metadataStore.userId || metadataStore.metadata?.user?.userId || 'anonymous';
  const sessionId = metadataStore.metadata?.session?.sessionId || 'unknown';
  const deviceId = metadataStore.metadata?.device?.deviceId || 'unknown';
  const currentCampaignId = campaignId || metadataStore.metadata?.campaign?.id || '0';
  const correlationId = generateCorrelationIDSync();
  const env = process.env.NODE_ENV || 'development';

  return {
    type,
    payload,
    correlation_id: correlationId,
    timestamp: new Date().toISOString(),
    version: '1.0.0',
    environment: env,
    source: 'frontend',
    metadata: {
      global_context: {
        user_id: userId,
        campaign_id: currentCampaignId,
        session_id: sessionId,
        device_id: deviceId,
        correlation_id: correlationId,
        source: 'frontend'
      },
      envelope_version: '1.0.0',
      environment: env,
      ServiceSpecific: {
        campaign: { campaignId: currentCampaignId }
      }
    }
  };
};

interface CampaignStore {
  currentCampaign?: Campaign;
  campaigns: Campaign[];
  loading: boolean;
  error: string | null;
  updateCount: number;

  // Actions
  switchCampaign: (campaignId: string, onResponse?: (event: EventEnvelope) => void) => void;
  handleCampaignSwitchRequired: (switchEvent: {
    old_campaign_id: string;
    new_campaign_id: string;
    reason: string;
    timestamp?: string;
  }) => void;
  handleCampaignSwitchCompleted: (switchEvent: {
    old_campaign_id: string;
    new_campaign_id: string;
    reason: string;
    timestamp?: string;
    status: string;
  }) => void;
  switchCampaignWithData: (
    campaignData: Campaign,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaign: (updates: Partial<Campaign>, onResponse?: (event: EventEnvelope) => void) => void;
  requestCampaignState: (campaignId: string) => Promise<Campaign>;
  updateCampaignFromResponse: (campaignData: any) => void;
  updateCampaignsFromResponse: (responseData: any) => void;
  createCampaign: (campaign: Partial<Campaign>) => Promise<Campaign>;
  requestCampaignList: () => Promise<void>;
  // Debugging helpers
  getCampaignSwitchFlow: () => {
    currentCampaign?: Campaign;
    syncStatus: {
      campaignsMatch: boolean;
      titlesMatch: boolean;
      statusMatch: boolean;
      featuresMatch: boolean;
    };
  };
}

export const useCampaignStore = create<CampaignStore>()(
  devtools(
    (set, get) => ({
      currentCampaign: undefined,
      campaigns: [],
      loading: true,
      error: null,
      updateCount: 0,

      handleCampaignSwitchRequired: switchEvent => {
        const { new_campaign_id, reason, timestamp } = switchEvent;
        set(
          state => ({
            currentCampaign: state.currentCampaign
              ? {
                  ...state.currentCampaign,
                  id: new_campaign_id,
                  last_switched: timestamp || new Date().toISOString(),
                  switch_reason: reason,
                  switch_status: 'switching'
                }
              : undefined
          }),
          false,
          'handleCampaignSwitchRequired'
        );
      },

      handleCampaignSwitchCompleted: switchEvent => {
        const { new_campaign_id, reason, timestamp } = switchEvent;
        set(
          state => ({
            currentCampaign: state.currentCampaign
              ? {
                  ...state.currentCampaign,
                  id: new_campaign_id,
                  last_switched: timestamp || new Date().toISOString(),
                  switch_reason: reason,
                  switch_status: 'completed'
                }
              : undefined
          }),
          false,
          'handleCampaignSwitchCompleted'
        );
        get().requestCampaignState(new_campaign_id);
      },

      switchCampaign: (campaignId, onResponse) => {
        const event = createEvent('campaign:switch:v1:requested', { campaignId }, campaignId);
        useEventStore.getState().emitEvent(event, onResponse);
      },

      switchCampaignWithData: (campaignData, onResponse) => {
        set({ currentCampaign: campaignData }, false, 'switchCampaignWithData');
        useMetadataStore.getState().updateCampaignMetadata(campaignData);
        const event = createEvent(
          'campaign:switch:v1:requested',
          {
            campaignId: campaignData.id,
            slug: campaignData.slug,
            updates: {
              status: 'active',
              last_switched: new Date().toISOString()
            }
          },
          campaignData.id
        );

        const responseHandler = (response: EventEnvelope) => {
          // After a switch, always request the latest state.
          if (response.type.includes('success') || response.type.includes('completed')) {
            get().requestCampaignState(campaignData.id);
          }
          // Pass the response to the original callback if it exists.
          if (onResponse) {
            onResponse(response);
          }
        };

        useEventStore.getState().emitEvent(event, responseHandler);
      },

      updateCampaign: (updates, onResponse) => {
        set(
          state => ({
            currentCampaign: state.currentCampaign
              ? { ...state.currentCampaign, ...updates }
              : undefined
          }),
          false,
          'updateCampaign'
        );
        const campaignId = get().currentCampaign?.id;
        if (campaignId) {
          const event = createEvent('campaign:update:v1:requested', { updates }, campaignId);
          useEventStore.getState().emitEvent(event, onResponse);
        }
      },

      requestCampaignState: campaignId => {
        return new Promise<Campaign>((resolve, reject) => {
          const event = createEvent(
            'campaign:state:v1:requested',
            {
              campaignId,
              fields: ['title', 'status', 'features', 'ui_content', 'communication']
            },
            campaignId
          );
          useEventStore.getState().emitEvent(event, (response: EventEnvelope) => {
            if (response.type === 'campaign:state:v1:success') {
              get().updateCampaignFromResponse(response.payload);
              resolve(response.payload as Campaign);
            } else if (response.type.includes('error')) {
              console.error('Campaign state error:', response.payload);
              reject(response.payload);
            }
          });
        });
      },

      updateCampaignFromResponse: campaignData => {
        const campaign = get().campaigns.find(c => c.id === campaignData.id);
        if (campaign) {
          const updatedCampaign = { ...campaign, ...campaignData };
          set(state => ({
            campaigns: state.campaigns.map(c =>
              c.id === updatedCampaign.id ? updatedCampaign : c
            ),
            currentCampaign:
              state.currentCampaign?.id === updatedCampaign.id
                ? updatedCampaign
                : state.currentCampaign
          }));
          // Sync with metadata store if this is the current campaign
          if (get().currentCampaign?.id === updatedCampaign.id) {
            useMetadataStore.getState().syncWithCampaignState(updatedCampaign);
          }
        } else {
          // if campaign is not in the list, add it
          set(state => ({
            campaigns: [...state.campaigns, campaignData]
          }));
        }
      },

      updateCampaignsFromResponse: responseData => {
        console.log('[CampaignStore] updateCampaignsFromResponse called with:', {
          responseData,
          hasCampaigns: !!responseData?.campaigns,
          hasDataCampaigns: !!responseData?.data?.campaigns,
          campaignsLength: responseData?.campaigns?.length || responseData?.data?.campaigns?.length || 0
        });
        
        const campaigns = responseData?.campaigns || responseData?.data?.campaigns || [];
        console.log('[CampaignStore] Extracted campaigns:', {
          count: campaigns.length,
          campaigns: campaigns.slice(0, 3) // Log first 3 for debugging
        });
        
        if (campaigns.length > 0) {
          set(
            state => {
              const existingCampaigns = new Map(state.campaigns.map((c: Campaign) => [c.id, c]));
              campaigns.forEach((c: Campaign) =>
                existingCampaigns.set(c.id, { ...existingCampaigns.get(c.id), ...c })
              );
              const updatedCampaigns = Array.from(existingCampaigns.values());
              console.log('[CampaignStore] Updated campaigns in store:', {
                previousCount: state.campaigns.length,
                newCount: updatedCampaigns.length,
                campaignIds: updatedCampaigns.map(c => c.id)
              });

              // Sync current campaign with metadata store
              const currentCampaignId = state.currentCampaign?.id;
              if (currentCampaignId) {
                const updatedCurrent = updatedCampaigns.find(c => c.id === currentCampaignId);
                if (updatedCurrent) {
                  useMetadataStore.getState().syncWithCampaignState(updatedCurrent);
                }
              }
              
              return { campaigns: updatedCampaigns };
            },
            false,
            'updateCampaignsFromResponse'
          );
        } else {
          console.warn('[CampaignStore] No campaigns found in response data:', responseData);
        }
      },

      createCampaign: campaign => {
        return new Promise((resolve, reject) => {
          const event = createEvent('campaign:create_campaign:v1:requested', { ...campaign });
          useEventStore.getState().emitEvent(event, (response: EventEnvelope) => {
            if (response.type === 'campaign:create_campaign:v1:success') {
              const newCampaign = response.payload?.campaign || response.payload;
              if (newCampaign) {
                set(state => ({ campaigns: [...state.campaigns, newCampaign] }));
                get().requestCampaignList();
                resolve(newCampaign);
              }
            } else {
              reject(response.payload);
            }
          });
        });
      },

      requestCampaignList: () => {
        return new Promise<void>((resolve, reject) => {
          set({ loading: true, error: null }, false, 'requestCampaignList');
          const event = createEvent('campaign:list:v1:requested', { limit: 50, offset: 0 });
          useEventStore.getState().emitEvent(event, (listResponse: EventEnvelope) => {
            if (listResponse.type === 'campaign:list:v1:success') {
              get().updateCampaignsFromResponse(listResponse.payload);
              set({ loading: false, error: null }, false, 'requestCampaignListSuccess');
              resolve();
            } else if (listResponse.type === 'campaign:error:v1:response') {
              const errorMessage = listResponse.payload?.message || 'Failed to load campaigns';
              console.error('Campaign error:', listResponse.payload);
              set({ loading: false, error: errorMessage }, false, 'requestCampaignListError');
              reject(new Error(errorMessage));
            } else {
              console.warn(
                `requestCampaignList received an unexpected event type: ${listResponse.type}`
              );
            }
          });
        });
      },

      // Debugging helpers
      getCampaignSwitchFlow: () => {
        const state = get();
        const current = state.currentCampaign;
        // Compare currentCampaign with metadata campaign if available
        const metadataCampaign = useMetadataStore.getState().metadata?.campaign as
          | Partial<Campaign>
          | undefined;
        const syncStatus = {
          campaignsMatch: !!(
            current &&
            metadataCampaign &&
            current.id === (metadataCampaign as any).id
          ),
          titlesMatch: !!(
            current &&
            metadataCampaign &&
            current.title === (metadataCampaign as any).title
          ),
          statusMatch: !!(
            current &&
            metadataCampaign &&
            current.status === (metadataCampaign as any).status
          ),
          featuresMatch: !!(
            current &&
            metadataCampaign &&
            JSON.stringify(current.features || []) ===
              JSON.stringify((metadataCampaign as any).features || [])
          )
        };
        return {
          currentCampaign: current,
          syncStatus
        };
      }
    }),
    {
      name: 'campaign-store'
    }
  )
);

// --- Subscribe to centralized event store ---
(() => {
  const { subscribe } = useEventStore.getState();

  subscribe('*', event => {
    const { type, payload } = event;
    const {
      handleCampaignSwitchRequired,
      handleCampaignSwitchCompleted,
      updateCampaignFromResponse,
      updateCampaignsFromResponse
    } = useCampaignStore.getState();

    switch (type) {
      case 'campaign:switch:v1:required':
        handleCampaignSwitchRequired(payload);
        break;
      case 'campaign:switch:v1:completed':
        handleCampaignSwitchCompleted(payload);
        break;
      case 'campaign:state:v1:success':
        updateCampaignFromResponse(payload);
        break;
      case 'campaign:list:v1:success':
        updateCampaignsFromResponse(payload);
        break;
      case 'campaign:create_campaign:v1:success':
        // Assuming the payload for create is a single campaign
        updateCampaignFromResponse(payload.campaign || payload);
        break;
      // Add other campaign-related event handlers here
    }
  });
})();
