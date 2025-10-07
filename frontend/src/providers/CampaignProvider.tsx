import React, { createContext, useContext, useEffect } from 'react';
import { useMetadataStore } from '../store/stores/metadataStore';
import { useCampaignStore } from '../store/stores/campaignStore';
import type { Campaign } from '../store/types';

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
  const { campaigns, requestCampaignList, loading, error } = useCampaignStore();

  useEffect(() => {
    if (userId && userId !== 'loading') {
      requestCampaignList();
    }
  }, [userId, requestCampaignList]);

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
