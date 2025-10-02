import { useMemo } from 'react';
import { useCampaignStore } from '../stores/campaignStore';
import { useMetadataStore } from '../stores/metadataStore';

// Campaign state hook
export function useCampaignState() {
  const campaignState = useCampaignStore(state => state.campaignState);
  const currentCampaign = useCampaignStore(state => state.currentCampaign);
  const metadata = useMetadataStore(state => state.metadata?.campaign);

  // Memoize the merged state to prevent infinite loops
  return useMemo(() => {
    // Use currentCampaign as primary source, fallback to metadata
    const campaign = currentCampaign || metadata;
    const state = campaignState || {};

    return {
      state: {
        ...state,
        // Core campaign fields
        campaignId: campaign?.campaignId || 0,
        slug: campaign?.slug || 'default',
        features: campaign?.features || [],
        title: campaign?.title,
        description: campaign?.description,
        status: campaign?.status,
        tags: campaign?.tags || [],
        // Backend campaign fields
        about: campaign?.about,
        ui_content: campaign?.ui_content,
        broadcast_enabled: campaign?.broadcast_enabled,
        channels: campaign?.channels || [],
        i18n_keys: campaign?.i18n_keys || [],
        focus: campaign?.focus,
        inos_enabled: campaign?.inos_enabled,
        ranking_formula: campaign?.ranking_formula,
        start_date: campaign?.start_date,
        end_date: campaign?.end_date,
        owner_id: campaign?.owner_id,
        master_id: campaign?.master_id,
        master_uuid: campaign?.master_uuid,
        // Service-specific data
        serviceSpecific: campaign?.serviceSpecific || {}
      },
      metadata: campaign,
      currentCampaign
    };
  }, [campaignState, currentCampaign, metadata]);
}

// Campaign updates hook
export function useCampaignUpdates() {
  const {
    switchCampaign,
    updateCampaign,
    updateCampaignFeatures,
    updateCampaignConfig,
    requestCampaignState
  } = useCampaignStore();

  return {
    switchCampaign,
    updateCampaign,
    updateCampaignFeatures,
    updateCampaignConfig,
    requestCampaignState
  };
}

// Campaign operations hook
export function useCampaignOperations() {
  const campaignState = useCampaignState();
  const campaignUpdates = useCampaignUpdates();
  const createCampaign = useCampaignStore(state => state.createCampaign);

  return {
    ...campaignState,
    ...campaignUpdates,
    createCampaign
  };
}
