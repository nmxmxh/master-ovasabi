// State Architecture Definitions
// Clear separation between different types of state

export interface CampaignState {
  // Server-side campaign data (read-only from frontend perspective)
  id: string;
  slug: string;
  title: string;
  description: string;
  status: 'active' | 'inactive' | 'draft';
  features: string[];
  ui_content: any;
  theme: any;
  service_configs: any;
  metadata: any;

  // Campaign-specific configuration
  ranking_formula?: string;
  owner_id?: string;
  start_date?: string;
  end_date?: string;

  // Last updated timestamp
  last_updated: string;
}

export interface UserSessionState {
  // User-specific session data
  userId: string;
  username: string;
  authenticated: boolean;
  sessionId: string;
  guestId: string;

  // User preferences
  preferences: {
    theme: 'light' | 'dark' | 'auto';
    language: string;
    timezone: string;
  };

  // User activity
  lastActivity: string;
  sessionStart: string;
}

export interface DeviceState {
  // Device information
  deviceId: string;
  userAgent: string;
  language: string;
  timezone: string;

  // Device capabilities
  capabilities: {
    webgl: boolean;
    webgpu: boolean;
    wasm: boolean;
    websocket: boolean;
  };

  // Consent and privacy
  consentGiven: boolean;
  gdprConsentRequired: boolean;
}

export interface ConnectionState {
  // WebSocket connection
  connected: boolean;
  wasmReady: boolean;
  reason?: string;

  // Connection metadata
  url?: string;
  readyState?: number;
  lastConnected?: string;
  reconnectAttempts: number;
}

export interface UIState {
  // UI-specific state
  currentView: string;
  activeTab: number;
  showDetails: boolean;
  isLoading: boolean;

  // UI preferences
  theme: {
    primary: string;
    secondary: string;
    background: string;
    text: string;
  };

  // UI state
  modals: {
    [key: string]: boolean;
  };

  // Navigation state
  navigation: {
    currentPath: string;
    history: string[];
  };
}

export interface EventState {
  // Event system state
  events: any[];
  isWasmReady: boolean;
  queuedEvents: any[];

  // Event filtering
  filters: {
    types: string[];
    dateRange: {
      start: string;
      end: string;
    };
  };
}

// State Store Responsibilities
export interface StateStoreResponsibilities {
  // Campaign Store
  campaign: {
    // Manages server-side campaign data
    currentCampaign: CampaignState | null;
    availableCampaigns: CampaignState[];

    // Actions
    switchCampaign: (campaignId: string) => Promise<void>;
    refreshCampaigns: () => Promise<void>;
    updateCampaign: (updates: Partial<CampaignState>) => Promise<void>;
  };

  // Metadata Store
  metadata: {
    // Manages user session and device data
    user: UserSessionState;
    device: DeviceState;

    // Actions
    updateUser: (user: Partial<UserSessionState>) => void;
    updateDevice: (device: Partial<DeviceState>) => void;
    initializeFromWasm: () => Promise<void>;
  };

  // Connection Store
  connection: {
    // Manages WebSocket and WASM connection
    state: ConnectionState;

    // Actions
    connect: () => void;
    disconnect: () => void;
    reconnect: () => void;
  };

  // UI Store (new)
  ui: {
    // Manages UI-specific state
    state: UIState;

    // Actions
    setView: (view: string) => void;
    setTheme: (theme: Partial<UIState['theme']>) => void;
    toggleModal: (modal: string) => void;
  };

  // Event Store
  events: {
    // Manages event system
    state: EventState;

    // Actions
    emitEvent: (event: any) => void;
    filterEvents: (filters: any) => void;
    clearHistory: () => void;
  };
}

// State Flow Rules
export const STATE_FLOW_RULES = {
  // Campaign data flows FROM server TO frontend
  campaign: {
    source: 'server',
    destination: 'frontend',
    sync: 'unidirectional',
    responsibility: 'CampaignStore'
  },

  // User session data flows FROM WASM TO frontend
  user: {
    source: 'wasm',
    destination: 'frontend',
    sync: 'unidirectional',
    responsibility: 'MetadataStore'
  },

  // Device data flows FROM browser TO frontend
  device: {
    source: 'browser',
    destination: 'frontend',
    sync: 'unidirectional',
    responsibility: 'MetadataStore'
  },

  // UI state flows WITHIN frontend
  ui: {
    source: 'frontend',
    destination: 'frontend',
    sync: 'internal',
    responsibility: 'UIStore'
  },

  // Events flow BIDIRECTIONALLY between frontend and server
  events: {
    source: 'bidirectional',
    destination: 'bidirectional',
    sync: 'bidirectional',
    responsibility: 'EventStore'
  }
} as const;

// State Access Patterns
export const STATE_ACCESS_PATTERNS = {
  // Read campaign data
  getCampaignData: () => 'useCampaignStore.getState().currentCampaign',

  // Read user data
  getUserData: () => 'useMetadataStore.getState().metadata.user',

  // Read device data
  getDeviceData: () => 'useMetadataStore.getState().metadata.device',

  // Read UI state
  getUIState: () => 'useUIStore.getState().state',

  // Read connection state
  getConnectionState: () => 'useConnectionStore.getState()',

  // Read events
  getEvents: () => 'useEventStore.getState().events'
} as const;
