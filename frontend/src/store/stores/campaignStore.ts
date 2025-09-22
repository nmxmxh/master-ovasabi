import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { CampaignMetadata, CampaignState } from '../types/campaign';
import type { EventEnvelope } from '../types/events';
import { useEventStore } from './eventStore';
import { useMetadataStore } from './metadataStore';

interface CampaignStore extends CampaignState {
  // Campaign-specific state
  currentCampaign?: CampaignMetadata;
  campaigns: any[];

  // Actions
  switchCampaign: (
    campaignId: number | string,
    slug?: string,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
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
  switchCampaignWithData: (campaignData: any, onResponse?: (event: EventEnvelope) => void) => void;
  updateCampaign: (
    updates: Record<string, any>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaignFeatures: (
    features: string[],
    action?: 'add' | 'remove' | 'set',
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaignConfig: (
    configType: 'ui_content' | 'scripts' | 'communication',
    config: Record<string, any>,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  requestCampaignState: (
    campaignId?: number | string,
    onResponse?: (event: EventEnvelope) => void
  ) => void;
  updateCampaignFromResponse: (campaignData: any) => void;
  updateCampaignsFromResponse: (responseData: any) => void;

  // Debugging & Monitoring
  getStateSnapshot: () => any; // Get complete state for debugging
  logStateChange: (action: string, details?: any) => void; // Log state changes
  getCampaignSwitchFlow: () => any; // Get campaign switching flow state
}

export const useCampaignStore = create<CampaignStore>()(
  devtools(
    (set, get) => ({
      // Initial campaign state
      campaignState: null,
      currentCampaign: undefined,
      campaigns: [],

      // Actions
      handleCampaignSwitchRequired: switchEvent => {
        console.log('[CampaignStore] Campaign switch required:', switchEvent);

        const { old_campaign_id, new_campaign_id, reason, timestamp } = switchEvent;

        // Show loading state during campaign switch
        set(
          {
            currentCampaign: {
              ...get().currentCampaign,
              campaignId: new_campaign_id,
              last_switched: timestamp || new Date().toISOString(),
              switch_reason: reason,
              switch_status: 'switching',
              features: get().currentCampaign?.features || []
            }
          },
          false,
          'handleCampaignSwitchRequired'
        );

        // Update metadata store
        const currentMetadata = useMetadataStore.getState().metadata?.campaign;
        useMetadataStore.getState().setMetadata({
          campaign: {
            ...currentMetadata,
            campaignId: new_campaign_id,
            last_switched: timestamp || new Date().toISOString(),
            switch_reason: reason,
            switch_status: 'switching',
            features: currentMetadata?.features || []
          }
        });

        // Notify user about the switch
        console.log(
          `[CampaignStore] Campaign switching from ${old_campaign_id} to ${new_campaign_id} (${reason})`
        );

        // Show switching notification
        if (typeof window !== 'undefined' && 'Notification' in window) {
          if (Notification.permission === 'granted') {
            new Notification('Switching Campaign', {
              body: `Switching to campaign ${new_campaign_id}...`,
              icon: '/favicon.ico'
            });
          }
        }
      },

      handleCampaignSwitchCompleted: switchEvent => {
        console.log('[CampaignStore] Campaign switch completed:', switchEvent);

        const { old_campaign_id, new_campaign_id, reason, timestamp } = switchEvent;

        // Update current campaign with completion status
        const currentCampaign = get().currentCampaign;
        set(
          {
            currentCampaign: {
              ...currentCampaign,
              campaignId: new_campaign_id,
              last_switched: timestamp || new Date().toISOString(),
              switch_reason: reason,
              switch_status: 'completed',
              features: currentCampaign?.features || []
            }
          },
          false,
          'handleCampaignSwitchCompleted'
        );

        // Update metadata store
        const currentMetadata = useMetadataStore.getState().metadata?.campaign;
        useMetadataStore.getState().setMetadata({
          campaign: {
            ...currentMetadata,
            campaignId: new_campaign_id,
            last_switched: timestamp || new Date().toISOString(),
            switch_reason: reason,
            switch_status: 'completed',
            features: currentMetadata?.features || []
          }
        });

        console.log(
          `[CampaignStore] Campaign switch completed from ${old_campaign_id} to ${new_campaign_id} (${reason})`
        );

        // Note: No need to request fresh state here as the switch success event already contains the state
        // The campaign:switch:v1:success event includes the complete campaign state in its payload

        // Show completion notification
        if (typeof window !== 'undefined' && 'Notification' in window) {
          if (Notification.permission === 'granted') {
            new Notification('Campaign Switch Completed', {
              body: `Successfully switched to campaign ${new_campaign_id}`,
              icon: '/favicon.ico'
            });
          }
        }
      },

      switchCampaign: (campaignId, slug, onResponse) => {
        console.log('[CampaignStore] Switching campaign:', { campaignId, slug });

        // Try to get campaign data from the campaigns list first
        // This will be populated by useCampaigns hook
        const campaignData = get().currentCampaign;

        // Create campaign metadata structure with actual data if available
        const campaignMetadata: CampaignMetadata = {
          campaignId,
          slug: slug || `campaign_${campaignId}`,
          campaignName:
            campaignData?.title || campaignData?.campaignName || `Campaign ${campaignId}`,
          features: campaignData?.features || [],
          status: campaignData?.status || 'active',
          title: campaignData?.title,
          description: campaignData?.description,
          tags: campaignData?.tags || [],
          about: campaignData?.about,
          ui_content: campaignData?.ui_content,
          broadcast_enabled: campaignData?.broadcast_enabled,
          channels: campaignData?.channels || [],
          i18n_keys: campaignData?.i18n_keys || [],
          focus: campaignData?.focus,
          inos_enabled: campaignData?.inos_enabled,
          ranking_formula: campaignData?.ranking_formula,
          start_date: campaignData?.start_date,
          end_date: campaignData?.end_date,
          owner_id: campaignData?.owner_id,
          master_id: campaignData?.master_id,
          master_uuid: campaignData?.master_uuid,
          serviceSpecific: {
            campaign: {
              campaignId,
              slug: slug || `campaign_${campaignId}`,
              status: campaignData?.status || 'active',
              last_switched: new Date().toISOString(),
              ...campaignData?.serviceSpecific?.campaign
            }
          }
        };

        // Update current campaign
        set(
          {
            currentCampaign: campaignMetadata
          },
          false,
          'switchCampaign'
        );

        // Also update metadata store
        useMetadataStore.getState().setMetadata({
          campaign: campaignMetadata
        });

        // Emit campaign switch event using canonical format
        // Use slug as the primary identifier for database lookups
        const campaignIdentifier = slug || `campaign_${campaignId}`;

        console.log('[CampaignStore] Campaign switch details:', {
          campaignId,
          slug,
          campaignIdentifier,
          note: 'Using slug as primary identifier for database lookup'
        });

        const event: Omit<
          EventEnvelope,
          'timestamp' | 'correlation_id' | 'version' | 'environment' | 'source'
        > = {
          type: 'campaign:switch:v1:requested', // Use switch event type for campaign switching
          payload: {
            campaignId: campaignIdentifier, // Use slug instead of numeric ID
            slug: campaignIdentifier,
            updates: {
              campaignId: campaignIdentifier,
              slug: campaignIdentifier,
              status: 'active',
              last_switched: new Date().toISOString()
            }
          },
          metadata: {
            global_context: {
              user_id:
                useMetadataStore.getState().userId ||
                useMetadataStore.getState().metadata?.user?.userId ||
                'loading',
              campaign_id: campaignIdentifier, // Use slug-based identifier
              correlation_id: `corr_${Date.now()}`,
              session_id: useMetadataStore.getState().metadata?.session?.sessionId || 'loading',
              device_id: useMetadataStore.getState().metadata?.device?.deviceId || 'loading',
              source: 'frontend'
            },
            envelope_version: '1.0.0',
            environment: process.env.NODE_ENV || 'development',
            ServiceSpecific: {
              campaign: { campaignId: campaignIdentifier, slug: campaignIdentifier }
            }
          }
        };

        // Emit event through event store
        console.log('[CampaignStore] Campaign switch event:', event);

        try {
          const eventStore = useEventStore.getState();
          eventStore.emitEvent(event, onResponse);
          console.log('[CampaignStore] Event emitted successfully');
        } catch (error) {
          console.error('[CampaignStore] Failed to emit event:', error);
          if (onResponse) {
            // Simulate response
            setTimeout(() => {
              const response: EventEnvelope = {
                ...event,
                type: 'campaign:update:v1:success',
                timestamp: new Date().toISOString(),
                correlation_id: `corr_${Date.now()}`,
                version: '1.0.0',
                environment: process.env.NODE_ENV || 'development',
                source: 'frontend'
              };
              onResponse(response);
            }, 100);
          }
        }
      },

      switchCampaignWithData: (campaignData, onResponse) => {
        console.log('[CampaignStore] Switching campaign with data:', {
          id: campaignData.id,
          campaignId: campaignData.campaignId,
          slug: campaignData.slug,
          title: campaignData.title,
          name: campaignData.name,
          status: campaignData.status,
          features: campaignData.features?.length || 0,
          hasUI: !!campaignData.ui_content,
          hasTheme: !!campaignData.theme,
          hasServiceConfigs: !!campaignData.service_configs
        });

        // Create campaign metadata structure with actual campaign data
        const campaignMetadata: CampaignMetadata = {
          campaignId: campaignData.id || campaignData.campaignId,
          slug: campaignData.slug || `campaign_${campaignData.id}`,
          campaignName: campaignData.title || campaignData.name || `Campaign ${campaignData.id}`,
          features: campaignData.features || [],
          status: campaignData.status || 'active',
          title: campaignData.title,
          description: campaignData.description,
          tags: campaignData.tags || [],
          about: campaignData.about,
          ui_content: campaignData.ui_content,
          broadcast_enabled: campaignData.broadcast_enabled,
          channels: campaignData.channels || [],
          i18n_keys: campaignData.i18n_keys || [],
          focus: campaignData.focus,
          inos_enabled: campaignData.inos_enabled,
          ranking_formula: campaignData.ranking_formula,
          start_date: campaignData.start_date,
          end_date: campaignData.end_date,
          owner_id: campaignData.owner_id,
          master_id: campaignData.master_id,
          master_uuid: campaignData.master_uuid,
          serviceSpecific: {
            campaign: {
              campaignId: campaignData.id || campaignData.campaignId,
              slug: campaignData.slug || `campaign_${campaignData.id}`,
              status: campaignData.status || 'active',
              last_switched: new Date().toISOString(),
              // Preserve nested structure from campaign.json
              ui_components:
                campaignData.ui_components ||
                campaignData.metadata?.service_specific?.campaign?.ui_components,
              theme: campaignData.theme || campaignData.metadata?.service_specific?.campaign?.theme,
              platform_type:
                campaignData.platform_type ||
                campaignData.metadata?.service_specific?.campaign?.platform_type,
              focus: campaignData.focus || campaignData.metadata?.service_specific?.campaign?.focus,
              target_audience:
                campaignData.target_audience ||
                campaignData.metadata?.service_specific?.campaign?.target_audience,
              service_configs:
                campaignData.service_configs ||
                campaignData.metadata?.service_specific?.campaign?.service_configs,
              // Spread other campaign data
              ...campaignData
            }
          }
        };

        console.log('[CampaignStore] Created campaign metadata:', {
          campaignId: campaignMetadata.campaignId,
          slug: campaignMetadata.slug,
          title: campaignMetadata.title,
          status: campaignMetadata.status,
          features: campaignMetadata.features.length,
          hasServiceSpecific: !!campaignMetadata.serviceSpecific,
          hasCampaign: !!campaignMetadata.serviceSpecific?.campaign
        });

        // Update current campaign
        set(
          {
            currentCampaign: campaignMetadata
          },
          false,
          'switchCampaignWithData'
        );

        // Also update metadata store with campaign data
        console.log('[CampaignStore] 🔄 Triggering metadata store update for campaign switch:', {
          campaignData: {
            id: campaignData.id || campaignData.campaignId,
            slug: campaignData.slug,
            title: campaignData.title,
            features: campaignData.features?.length || 0
          },
          currentMetadata: useMetadataStore.getState().metadata.campaign,
          userId: useMetadataStore.getState().userId
        });

        useMetadataStore.getState().updateCampaignMetadata(campaignData);

        // Emit campaign switch event
        const event: Omit<
          EventEnvelope,
          'timestamp' | 'correlation_id' | 'version' | 'environment' | 'source'
        > = {
          type: 'campaign:switch:v1:requested',
          payload: {
            campaignId: campaignData.id || campaignData.campaignId,
            slug: campaignData.slug,
            updates: {
              campaignId: campaignData.id || campaignData.campaignId,
              slug: campaignData.slug,
              status: 'active',
              last_switched: new Date().toISOString()
            }
          },
          metadata: {
            global_context: {
              user_id:
                useMetadataStore.getState().userId ||
                useMetadataStore.getState().metadata?.user?.userId ||
                'loading',
              campaign_id: campaignData.id || campaignData.campaignId,
              correlation_id: `corr_${Date.now()}`,
              session_id: useMetadataStore.getState().metadata?.session?.sessionId || 'loading',
              device_id: useMetadataStore.getState().metadata?.device?.deviceId || 'loading',
              source: 'frontend'
            },
            envelope_version: '1.0.0',
            environment: process.env.NODE_ENV || 'development',
            ServiceSpecific: {
              campaign: {
                campaignId: campaignData.id || campaignData.campaignId,
                slug: campaignData.slug
              }
            }
          }
        };

        // Emit event through event store
        console.log('[CampaignStore] Emitting campaign switch event:', {
          type: event.type,
          campaignId: event.payload.campaignId,
          slug: event.payload.slug,
          correlationId: event.metadata.global_context.correlation_id,
          hasOnResponse: !!onResponse
        });

        try {
          const eventStore = useEventStore.getState();
          eventStore.emitEvent(event, onResponse);
          console.log('[CampaignStore] Campaign switch event emitted successfully');
        } catch (error) {
          console.error('[CampaignStore] Failed to emit event:', error);
          if (onResponse) {
            console.log('[CampaignStore] Simulating response due to emit failure');
            // Simulate response
            setTimeout(() => {
              const response: EventEnvelope = {
                ...event,
                type: 'campaign:update:v1:success',
                timestamp: new Date().toISOString(),
                correlation_id: `corr_${Date.now()}`,
                version: '1.0.0',
                environment: process.env.NODE_ENV || 'development',
                source: 'frontend'
              };
              onResponse(response);
            }, 100);
          }
        }
      },

      updateCampaign: (updates, onResponse) => {
        console.log('[CampaignStore] Updating campaign:', updates);

        // Update both campaign state and current campaign
        set(
          state => ({
            campaignState: { ...state.campaignState, ...updates },
            currentCampaign: state.currentCampaign
              ? { ...state.currentCampaign, ...updates }
              : undefined
          }),
          false,
          'updateCampaign'
        );

        // Also update metadata store
        const currentCampaign = get().currentCampaign;
        if (currentCampaign) {
          useMetadataStore.getState().setMetadata({
            campaign: { ...currentCampaign, ...updates }
          });
        }

        // Emit campaign update event
        const event: Omit<EventEnvelope, 'timestamp'> = {
          type: 'campaign:update:v1:requested',
          correlation_id: `corr_${Date.now()}`,
          version: '1.0.0',
          environment: process.env.NODE_ENV || 'development',
          source: 'frontend',
          payload: { updates },
          metadata: {
            campaign: get().currentCampaign,
            user: { userId: 'current_user' },
            device: { deviceId: 'current_device' },
            session: { sessionId: 'current_session' }
          }
        };

        console.log('[CampaignStore] Campaign update event:', event);

        // Emit event through event store
        try {
          const eventStore = useEventStore.getState();
          eventStore.emitEvent(event, onResponse);
          console.log('[CampaignStore] Update event emitted successfully');
        } catch (error) {
          console.error('[CampaignStore] Failed to emit event:', error);
          if (onResponse) {
            setTimeout(() => {
              const response: EventEnvelope = {
                ...event,
                type: 'campaign:update:v1:success',
                timestamp: new Date().toISOString(),
                correlation_id: `corr_${Date.now()}`
              };
              onResponse(response);
            }, 100);
          }
        }
      },

      updateCampaignFeatures: (features, action = 'set', onResponse) => {
        console.log('[CampaignStore] Updating campaign features:', { features, action });

        const currentCampaign = get().currentCampaign;
        if (!currentCampaign) return;

        let newFeatures = [...currentCampaign.features];

        switch (action) {
          case 'add':
            features.forEach(feature => {
              if (!newFeatures.includes(feature)) {
                newFeatures.push(feature);
              }
            });
            break;
          case 'remove':
            newFeatures = newFeatures.filter(feature => !features.includes(feature));
            break;
          case 'set':
          default:
            newFeatures = [...features];
            break;
        }

        set(
          {
            currentCampaign: { ...currentCampaign, features: newFeatures }
          },
          false,
          'updateCampaignFeatures'
        );

        // Emit feature update event
        const event: Omit<EventEnvelope, 'timestamp'> = {
          type: 'campaign:feature:v1:requested',
          correlation_id: `corr_${Date.now()}`,
          version: '1.0.0',
          environment: process.env.NODE_ENV || 'development',
          source: 'frontend',
          payload: { features, action },
          metadata: {
            campaign: get().currentCampaign,
            user: { userId: 'current_user' },
            device: { deviceId: 'current_device' },
            session: { sessionId: 'current_session' }
          }
        };

        console.log('[CampaignStore] Feature update event:', event);

        if (onResponse) {
          setTimeout(() => {
            const response: EventEnvelope = {
              ...event,
              type: 'campaign:feature:v1:success',
              timestamp: new Date().toISOString(),
              correlation_id: `corr_${Date.now()}`
            };
            onResponse(response);
          }, 100);
        }
      },

      updateCampaignConfig: (configType, config, onResponse) => {
        console.log('[CampaignStore] Updating campaign config:', { configType, config });

        const currentCampaign = get().currentCampaign;
        if (!currentCampaign) return;

        set(
          {
            currentCampaign: {
              ...currentCampaign,
              serviceSpecific: {
                ...currentCampaign.serviceSpecific,
                [configType]: config
              }
            }
          },
          false,
          'updateCampaignConfig'
        );

        // Emit config update event
        const event: Omit<EventEnvelope, 'timestamp'> = {
          type: 'campaign:config:v1:requested',
          correlation_id: `corr_${Date.now()}`,
          version: '1.0.0',
          environment: process.env.NODE_ENV || 'development',
          source: 'frontend',
          payload: { configType, config },
          metadata: {
            campaign: get().currentCampaign,
            user: { userId: 'current_user' },
            device: { deviceId: 'current_device' },
            session: { sessionId: 'current_session' }
          }
        };

        console.log('[CampaignStore] Config update event:', event);

        if (onResponse) {
          setTimeout(() => {
            const response: EventEnvelope = {
              ...event,
              type: 'campaign:config:v1:success',
              timestamp: new Date().toISOString(),
              correlation_id: `corr_${Date.now()}`
            };
            onResponse(response);
          }, 100);
        }
      },

      requestCampaignState: (campaignId, onResponse) => {
        const currentCampaign = get().currentCampaign;
        const targetCampaignId = campaignId || currentCampaign?.campaignId || 0;

        // Convert numeric ID to slug format for backend compatibility
        const campaignSlug =
          typeof targetCampaignId === 'number' ? `campaign_${targetCampaignId}` : targetCampaignId;

        console.log('[CampaignStore] Requesting campaign state:', {
          campaignId: targetCampaignId,
          campaignSlug: campaignSlug
        });

        // Emit campaign state request event
        const event: Omit<EventEnvelope, 'timestamp'> = {
          type: 'campaign:state:v1:requested',
          correlation_id: `corr_${Date.now()}`,
          version: '1.0.0',
          environment: process.env.NODE_ENV || 'development',
          source: 'frontend',
          payload: {
            campaignId: campaignSlug, // Use slug for backend compatibility
            slug: campaignSlug, // Also include slug field
            fields: ['title', 'status', 'features', 'ui_content', 'communication']
          },
          metadata: {
            global_context: {
              user_id:
                useMetadataStore.getState().userId ||
                useMetadataStore.getState().metadata?.user?.userId ||
                'loading',
              campaign_id: campaignSlug, // Use slug in metadata too
              correlation_id: `corr_${Date.now()}`,
              session_id: useMetadataStore.getState().metadata?.session?.sessionId || 'loading',
              device_id: useMetadataStore.getState().metadata?.device?.deviceId || 'loading',
              source: 'frontend'
            },
            envelope_version: '1.0.0',
            environment: process.env.NODE_ENV || 'development',
            ServiceSpecific: {
              campaign: { campaignId: campaignSlug } // Use slug here too
            }
          }
        };

        console.log('[CampaignStore] Campaign state request event:', event);

        // Emit event through event store
        try {
          const eventStore = useEventStore.getState();
          eventStore.emitEvent(event, onResponse);
          console.log('[CampaignStore] State request event emitted successfully');
        } catch (error) {
          console.error('[CampaignStore] Failed to emit event:', error);
          if (onResponse) {
            setTimeout(() => {
              const response: EventEnvelope = {
                ...event,
                type: 'campaign:state:v1:success',
                timestamp: new Date().toISOString(),
                correlation_id: `corr_${Date.now()}`
              };
              onResponse(response);
            }, 100);
          }
        }
      },

      updateCampaignFromResponse: campaignData => {
        console.log('[CampaignStore] Updating campaign from response:', campaignData);

        // Extract campaign information from response
        const campaignId = campaignData.campaignId || campaignData.campaign_id || campaignData.id;
        const slug = campaignData.slug;
        const title = campaignData.title || campaignData.name;
        const features = Array.isArray(campaignData.features) ? campaignData.features : [];
        const status = campaignData.status || 'active';

        // Create full campaign metadata structure
        const campaignMetadata: CampaignMetadata = {
          campaignId,
          slug: slug || `campaign_${campaignId}`,
          campaignName: title || `Campaign ${campaignId}`,
          features,
          title,
          description: campaignData.description,
          status: status as 'active' | 'inactive' | 'draft',
          tags: campaignData.tags,
          createdAt: campaignData.createdAt || campaignData.created_at,
          updatedAt: campaignData.updatedAt || campaignData.updated_at,
          about: campaignData.about,
          ui_content: campaignData.ui_content,
          broadcast_enabled: campaignData.broadcast_enabled,
          channels: campaignData.channels,
          i18n_keys: campaignData.i18n_keys,
          focus: campaignData.focus,
          inos_enabled: campaignData.inos_enabled,
          ranking_formula: campaignData.ranking_formula,
          start_date: campaignData.start_date,
          end_date: campaignData.end_date,
          owner_id: campaignData.owner_id,
          master_id: campaignData.master_id,
          master_uuid: campaignData.master_uuid,
          serviceSpecific: {
            campaign: {
              campaignId,
              slug,
              title,
              status,
              features,
              about: campaignData.about,
              ui_content: campaignData.ui_content,
              broadcast_enabled: campaignData.broadcast_enabled,
              channels: campaignData.channels,
              i18n_keys: campaignData.i18n_keys,
              focus: campaignData.focus,
              inos_enabled: campaignData.inos_enabled,
              last_updated: new Date().toISOString()
            }
          }
        };

        // Update current campaign with full data
        set(
          {
            currentCampaign: campaignMetadata
          },
          false,
          'updateCampaignFromResponse'
        );

        // Sync with metadata store
        console.log('[CampaignStore] 🔄 Triggering metadata store sync for campaign response:', {
          campaignMetadata: {
            id: campaignMetadata.campaignId,
            slug: campaignMetadata.slug,
            title: campaignMetadata.title,
            status: campaignMetadata.status,
            features: campaignMetadata.features?.length || 0
          },
          currentMetadata: useMetadataStore.getState().metadata.campaign,
          userId: useMetadataStore.getState().userId
        });

        useMetadataStore.getState().syncWithCampaignState(campaignMetadata);
      },

      updateCampaignsFromResponse: responseData => {
        // Updating campaigns from response

        // Extract campaigns from the response
        const campaigns = responseData?.campaigns || responseData?.data?.campaigns || [];

        if (campaigns.length > 0) {
          // Processing campaigns

          // Update the campaigns in the store
          set(
            () => ({
              campaigns: campaigns.map((campaign: any, index: number) => {
                try {
                  const parsed = typeof campaign === 'string' ? JSON.parse(campaign) : campaign;

                  // Extract nested data from metadata.service_specific.campaign if it exists
                  const nestedCampaign = parsed.metadata?.service_specific?.campaign;

                  return {
                    id: parsed.id || parsed.campaign_id || index,
                    title: parsed.title || parsed.name || `Campaign ${index + 1}`,
                    slug: parsed.slug || `campaign-${index + 1}`,
                    description: parsed.description || '',
                    status: parsed.status || 'active',
                    features: parsed.features || [],
                    focus: parsed.focus || nestedCampaign?.focus || '',
                    inos_enabled: parsed.inos_enabled || false,
                    broadcast_enabled: parsed.broadcast_enabled || false,
                    channels: parsed.channels || [],
                    i18n_keys: parsed.i18n_keys || [],
                    ui_content: parsed.ui_content || nestedCampaign?.ui_content || {},
                    about: parsed.about || nestedCampaign?.about || {},
                    // Extract nested UI components and theme
                    ui_components: parsed.ui_components || nestedCampaign?.ui_components || {},
                    theme: parsed.theme || nestedCampaign?.theme || {},
                    platform_type:
                      parsed.platform_type ||
                      nestedCampaign?.platform_type ||
                      parsed.focus ||
                      'general',
                    service_configs:
                      parsed.service_configs || nestedCampaign?.service_configs || {},
                    target_audience:
                      parsed.target_audience || nestedCampaign?.target_audience || '',
                    last_updated: new Date().toISOString(),
                    // Preserve the full campaign data for nested access
                    ...parsed
                  };
                } catch (error) {
                  console.error('[CampaignStore] Error parsing campaign:', error, campaign);
                  return {
                    id: index,
                    title: `Campaign ${index + 1}`,
                    slug: `campaign-${index + 1}`,
                    description: '',
                    status: 'active',
                    features: [],
                    focus: '',
                    inos_enabled: false,
                    broadcast_enabled: false,
                    channels: [],
                    i18n_keys: [],
                    ui_content: {},
                    about: {},
                    ui_components: {},
                    theme: {},
                    platform_type: 'general',
                    service_configs: {},
                    target_audience: '',
                    last_updated: new Date().toISOString()
                  };
                }
              })
            }),
            false,
            'updateCampaignsFromResponse'
          );
        }
      },

      // Get complete state snapshot for debugging
      getStateSnapshot: () => {
        const state = get();
        return {
          currentCampaign: state.currentCampaign,
          campaigns: state.campaigns,
          metadata: useMetadataStore.getState().getStateSnapshot(),
          timestamp: new Date().toISOString()
        };
      },

      // Log state changes with context
      logStateChange: (action: string, details?: any) => {
        const snapshot = get().getStateSnapshot();
        console.log(`[CampaignStore] 📊 State Change: ${action}`, {
          action,
          details,
          currentState: snapshot,
          timestamp: new Date().toISOString()
        });
      },

      // Get campaign switching flow state
      getCampaignSwitchFlow: () => {
        const state = get();
        const metadata = useMetadataStore.getState().getStateSnapshot();

        return {
          currentCampaign: {
            id: state.currentCampaign?.campaignId,
            slug: state.currentCampaign?.slug,
            title: state.currentCampaign?.title,
            status: state.currentCampaign?.status,
            features: state.currentCampaign?.features?.length || 0
          },
          metadataCampaign: {
            id: metadata.metadata.campaign.campaignId,
            slug: metadata.metadata.campaign.slug,
            title: metadata.metadata.campaign.title,
            status: metadata.metadata.campaign.status,
            features: metadata.metadata.campaign.features?.length || 0,
            lastSwitched: metadata.metadata.campaign.last_switched
          },
          syncStatus: {
            campaignsMatch:
              state.currentCampaign?.campaignId === metadata.metadata.campaign.campaignId,
            titlesMatch: state.currentCampaign?.title === metadata.metadata.campaign.title,
            statusMatch: state.currentCampaign?.status === metadata.metadata.campaign.status,
            featuresMatch:
              JSON.stringify(state.currentCampaign?.features) ===
              JSON.stringify(metadata.metadata.campaign.features)
          },
          userId: metadata.userId,
          timestamp: new Date().toISOString()
        };
      }
    }),
    {
      name: 'campaign-store'
    }
  )
);
