// State monitoring utilities for debugging and development
import { useCampaignStore } from '../store/stores/campaignStore';
import { useMetadataStore } from '../store/stores/metadataStore';
import { useEventStore } from '../store/stores/eventStore';
import { useConnectionStore } from '../store/stores/connectionStore';
import { useUIStore } from '../store/stores/uiStore';

export interface SystemStateSnapshot {
  timestamp: string;

  // Campaign State (Server-side data)
  campaign: {
    current: any;
    available: any[];
    syncStatus: {
      campaignsMatch: boolean;
      titlesMatch: boolean;
      statusMatch: boolean;
      featuresMatch: boolean;
    };
  };

  // User Session State (WASM-generated)
  metadata: {
    userId: string;
    campaignId: string | number;
    device: any;
    session: any;
    user: any;
  };

  // UI State (Frontend-only)
  ui: {
    currentView: string;
    activeTab: number;
    showDetails: boolean;
    isLoading: boolean;
    theme: any;
    modals: any;
    navigation: any;
  };

  // Event State (Bidirectional)
  events: {
    total: number;
    recent: any[];
    wasmReady: boolean;
  };

  // Connection State (Infrastructure)
  connection: {
    connected: boolean;
    wasmReady: boolean;
    reconnectAttempts: number;
  };

  // Performance Metrics
  performance: {
    memoryUsage?: any;
    renderTime?: number;
  };
}

export const getSystemStateSnapshot = (): SystemStateSnapshot => {
  const campaignStore = useCampaignStore.getState();
  const metadataStore = useMetadataStore.getState();
  const eventStore = useEventStore.getState();
  const connectionStore = useConnectionStore.getState();
  const uiStore = useUIStore.getState();

  const campaignFlow = campaignStore.getCampaignSwitchFlow();
  const metadataSnapshot = metadataStore.getStateSnapshot();
  const uiSnapshot = uiStore.getStateSnapshot();

  return {
    timestamp: new Date().toISOString(),

    // Campaign State (Server-side data)
    campaign: {
      current: campaignFlow.currentCampaign,
      available: campaignStore.campaigns,
      syncStatus: campaignFlow.syncStatus
    },

    // User Session State (WASM-generated)
    metadata: {
      userId: metadataSnapshot.userId,
      campaignId: metadataSnapshot.campaignId,
      device: metadataSnapshot.metadata.device,
      session: metadataSnapshot.metadata.session,
      user: metadataSnapshot.metadata.user
    },

    // UI State (Frontend-only)
    ui: {
      currentView: uiSnapshot.ui.currentView,
      activeTab: uiSnapshot.ui.activeTab,
      showDetails: uiSnapshot.ui.showDetails,
      isLoading: uiSnapshot.ui.isLoading,
      theme: uiSnapshot.ui.theme,
      modals: uiSnapshot.ui.modals,
      navigation: uiSnapshot.ui.navigation
    },

    // Event State (Bidirectional)
    events: {
      total: eventStore.events.length,
      recent: eventStore.events.slice(-5),
      wasmReady: eventStore.isWasmReady
    },

    // Connection State (Infrastructure)
    connection: {
      connected: connectionStore.connected,
      wasmReady: connectionStore.wasmReady,
      reconnectAttempts: connectionStore.reconnectAttempts
    },

    // Performance Metrics
    performance: {
      memoryUsage: (performance as any).memory
        ? {
            used: Math.round((performance as any).memory.usedJSHeapSize / 1024 / 1024),
            total: Math.round((performance as any).memory.totalJSHeapSize / 1024 / 1024),
            limit: Math.round((performance as any).memory.jsHeapSizeLimit / 1024 / 1024)
          }
        : undefined
    }
  };
};

export const logSystemState = (label: string = 'System State') => {
  const snapshot = getSystemStateSnapshot();
  console.group(`🔍 ${label}`);
  console.log('📊 Complete System State:', snapshot);
  console.log('🎯 Campaign Sync Status:', snapshot.campaign.syncStatus);
  console.log('👤 User Context:', snapshot.metadata);
  console.log('🔌 Connection Status:', snapshot.connection);
  console.log('📈 Performance:', snapshot.performance);
  console.groupEnd();
  return snapshot;
};

// Global state monitor for debugging
export const monitorCampaignSwitching = () => {
  const campaignStore = useCampaignStore.getState();

  console.group('🔄 Campaign Switching Monitor');

  // Log current state
  logSystemState('Before Switch');

  // Monitor for changes
  const unsubscribeCampaign = useCampaignStore.subscribe(state => {
    if (state.currentCampaign !== campaignStore.currentCampaign) {
      console.log('🎯 Campaign Store Changed:', {
        previous: campaignStore.currentCampaign,
        current: state.currentCampaign,
        timestamp: new Date().toISOString()
      });

      // Check sync status
      const flow = campaignStore.getCampaignSwitchFlow();
      console.log('🔄 Sync Status:', flow.syncStatus);
    }
  });

  const unsubscribeMetadata = useMetadataStore.subscribe(state => {
    if (state.metadata.campaign !== campaignStore.currentCampaign) {
      console.log('📊 Metadata Store Changed:', {
        previous: campaignStore.currentCampaign,
        current: state.metadata.campaign,
        timestamp: new Date().toISOString()
      });
    }
  });

  console.log('✅ Monitoring started. Call stopMonitoring() to stop.');

  return {
    stop: () => {
      unsubscribeCampaign();
      unsubscribeMetadata();
      console.log('🛑 Campaign switching monitoring stopped');
    }
  };
};

// Make monitoring functions available globally for debugging
if (typeof window !== 'undefined') {
  (window as any).debugCampaignSwitching = {
    getSystemState: getSystemStateSnapshot,
    logSystemState,
    monitorCampaignSwitching,
    getCampaignFlow: () => useCampaignStore.getState().getCampaignSwitchFlow(),
    getMetadataSnapshot: () => useMetadataStore.getState().getStateSnapshot()
  };

  console.log('🔧 Debug tools available: window.debugCampaignSwitching');
}
