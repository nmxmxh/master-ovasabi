import React, { createContext, useContext, useEffect } from 'react';
import { useMetadataStore } from '../store/stores/metadataStore';
import { useCampaignStore } from '../store/stores/campaignStore';
import type { Campaign, EventEnvelope } from '../store/types';

interface CampaignProviderContextType {
  campaigns: Campaign[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

const CampaignProviderContext = createContext<CampaignProviderContextType | null>(null);

interface CampaignProviderProps {
  children: React.ReactNode;
}

export function CampaignProvider({ children }: CampaignProviderProps) {
  const userId = useMetadataStore(state => state.metadata?.user?.userId || state.userId);
  const {
    campaigns,
    requestCampaignList,
    requestCampaignState,
    updateCampaignFromResponse,
    loading,
    error
  } = useCampaignStore();

  useEffect(() => {
    if (userId && userId !== 'loading') {
      // Request the default campaign state first
      requestCampaignState('0', (response: EventEnvelope) => {
        if (response.type === 'campaign:state:v1:success') {
          updateCampaignFromResponse(response.payload);
        }
      });

      // Then, request the full list of campaigns
      requestCampaignList();
    }
  }, [userId, requestCampaignList, requestCampaignState]);

  const refresh = React.useCallback(() => {
    requestCampaignList();
  }, [requestCampaignList]);

  const contextValue: CampaignProviderContextType = {
    campaigns,
    loading,
    error,
    refresh
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
