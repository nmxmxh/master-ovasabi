import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { Metadata } from '../types';
import { merge, cloneDeep } from 'lodash';
import { generateDeviceID, generateSessionID } from '../../utils/wasmIdExtractor';
// import { stateManager, type UserState } from '../../utils/stateManager';

interface MetadataStore {
  metadata: Metadata;
  userId: string; // Store userId as a state property instead of getter

  // Actions
  setMetadata: (meta: Partial<Metadata>) => void;
  updateMetadata: (updater: (current: Metadata) => Metadata) => void;
  handleUserIDChange: (newUserId: string) => void;
  initializeUserId: () => Promise<void>;
  initializeMetadata: () => Promise<void>; // Initialize metadata with WASM IDs
  updateCampaignMetadata: (campaignData: any) => void; // Update campaign-specific metadata
  syncWithCampaignState: (campaignState: any) => void; // Sync with campaign state changes

  // Getters
  get campaignId(): string | number;
  // Debugging helpers
  getStateSnapshot: () => { metadata: Metadata; userId: string; campaignId: string | number };
}

// Note: Secure ID generation functions are now imported from cryptoIds utility

// IP Address and Security Detection
const detectIPAndSecurity = async (): Promise<{
  ipAddress?: string;
  ipLocation?: any;
  securityFlags?: any;
}> => {
  try {
    // Try to get IP from multiple sources
    const ipPromises = [
      // Primary: ipify.org (most reliable)
      fetch('https://api.ipify.org?format=json')
        .then(res => res.json())
        .then(data => ({ ip: data.ip, source: 'ipify' }))
        .catch(() => null),

      // Fallback 1: ipapi.co (includes location)
      fetch('https://ipapi.co/json/')
        .then(res => res.json())
        .then(data => ({
          ip: data.ip,
          location: {
            country: data.country_name,
            region: data.region,
            city: data.city,
            latitude: data.latitude,
            longitude: data.longitude
          },
          source: 'ipapi'
        }))
        .catch(() => null)
    ];

    const results = await Promise.allSettled(ipPromises);
    const successfulResult = results.find(
      result => result.status === 'fulfilled' && result.value !== null
    );

    if (successfulResult && successfulResult.status === 'fulfilled' && successfulResult.value) {
      const data = successfulResult.value;

      // Basic security analysis
      const securityFlags = {
        isBot: false, // Would need more sophisticated detection
        isVPN: false, // Would need VPN detection service
        isProxy: false, // Would need proxy detection
        riskScore: 0, // 0-100 risk score
        suspiciousActivity: false
      };

      // Simple bot detection based on user agent
      if (typeof navigator !== 'undefined') {
        const userAgent = navigator.userAgent.toLowerCase();
        securityFlags.isBot =
          userAgent.includes('bot') ||
          userAgent.includes('crawler') ||
          userAgent.includes('spider') ||
          userAgent.includes('scraper');
      }

      return {
        ipAddress: data.ip,
        ipLocation: (data as any).location,
        securityFlags
      };
    }

    return {};
  } catch (error) {
    console.warn('[MetadataStore] Failed to detect IP address:', error);
    return {};
  }
};

// Get user ID from WASM (source of truth)
const getUserIdFromWasm = async (): Promise<string> => {
  // Try to get from WASM global (this is the authoritative source)
  if (typeof window !== 'undefined' && (window as any).userID) {
    return (window as any).userID;
  }

  // Try to get from WASM state manager as fallback
  if (typeof window !== 'undefined' && (window as any).initializeState) {
    try {
      const state = await (window as any).initializeState();
      if (state && state.user_id) {
        return state.user_id;
      }
    } catch (error) {
      console.warn('Failed to get state from WASM:', error);
    }
  }

  // If WASM userID is not available, wait for it with timeout
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      reject(new Error('WASM userID not available after timeout'));
    }, 5000); // 5 second timeout

    const checkForUserId = () => {
      if (typeof window !== 'undefined' && (window as any).userID) {
        clearTimeout(timeout);
        resolve((window as any).userID);
      } else {
        // Check again in 100ms
        setTimeout(checkForUserId, 100);
      }
    };
    checkForUserId();
  });
};

// Create initial metadata with fallback userId
const createInitialMetadata = async (fallbackUserId?: string): Promise<Metadata> => {
  try {
    // Use WASM ID generation if available, otherwise use fallback
    let userId: string;
    let deviceId: string;
    let sessionId: string;

    if (fallbackUserId) {
      userId = fallbackUserId;
    } else {
      // Use the proper WASM user ID getter that checks window.userID first
      userId = await getUserIdFromWasm();
    }

    try {
      deviceId = await generateDeviceID();
    } catch {
      // Fallback to WASM global if available
      if (typeof window !== 'undefined' && (window as any).deviceID) {
        deviceId = (window as any).deviceID;
      } else {
        deviceId = 'device_fallback_' + Date.now();
      }
    }

    try {
      sessionId = await generateSessionID();
    } catch {
      // Fallback to WASM global if available
      if (typeof window !== 'undefined' && (window as any).sessionID) {
        sessionId = (window as any).sessionID;
      } else {
        sessionId = 'session_fallback_' + Date.now();
      }
    }

    // Detect IP address and security information
    console.log('[MetadataStore] 🔍 Detecting IP address and security information...');
    const ipData = await detectIPAndSecurity();

    return {
      campaign: {
        id: '0',
        name: 'Default Campaign',
        title: 'Default Campaign',
        slug: 'default',
        description: 'Default campaign metadata context',
        status: 'active',
        features: [],
        tags: []
      },
      user: {
        userId: userId,
        username: 'Guest User'
      },
      device: {
        deviceId,
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
        language: typeof navigator !== 'undefined' ? navigator.language : 'en-US',
        timezone:
          typeof Intl !== 'undefined' ? Intl.DateTimeFormat().resolvedOptions().timeZone : 'UTC',
        consentGiven: false,
        gdprConsentRequired: true,
        // IP and security information
        ipAddress: ipData.ipAddress,
        ipLocation: ipData.ipLocation,
        securityFlags: ipData.securityFlags
      },
      session: {
        sessionId,
        guestId: userId,
        authenticated: false
      },
      correlation_id: `corr_${Date.now()}`
    };
  } catch (error) {
    console.warn('[MetadataStore] Error creating initial metadata:', error);

    // Try to get IP data even in fallback case
    let ipData: { ipAddress?: string; ipLocation?: any; securityFlags?: any } = {};
    try {
      ipData = await detectIPAndSecurity();
    } catch (ipError) {
      console.warn('[MetadataStore] Failed to get IP data in fallback:', ipError);
    }

    // Fallback metadata in case of error
    return {
      campaign: {
        id: '0',
        name: 'Default Campaign',
        title: 'Default Campaign',
        slug: 'default',
        description: 'Default campaign metadata context',
        status: 'active',
        features: [],
        tags: []
      },
      user: {
        userId: 'anonymous',
        username: 'Guest User'
      },
      device: {
        deviceId: 'fallback_device',
        userAgent: '',
        language: 'en-US',
        timezone: 'UTC',
        consentGiven: false,
        gdprConsentRequired: true,
        // Include IP data even in fallback
        ipAddress: ipData.ipAddress,
        ipLocation: ipData.ipLocation,
        securityFlags: ipData.securityFlags
      },
      session: {
        sessionId: 'fallback_session',
        guestId: 'anonymous',
        authenticated: false
      },
      correlation_id: 'fallback_corr'
    };
  }
};

export const useMetadataStore = create<MetadataStore>()(
  devtools(
    (set, get) => ({
      // Initial metadata - will be populated asynchronously
      metadata: {
        campaign: {
          id: '0',
          name: 'Default Campaign',
          title: 'Default Campaign',
          slug: 'default',
          description: 'Default campaign metadata context',
          status: 'active',
          features: [],
          tags: []
        },
        user: {
          userId: 'loading',
          username: 'Guest User'
        },
        device: {
          deviceId: 'loading',
          userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
          language: typeof navigator !== 'undefined' ? navigator.language : 'en-US',
          timezone:
            typeof Intl !== 'undefined' ? Intl.DateTimeFormat().resolvedOptions().timeZone : 'UTC',
          consentGiven: false,
          gdprConsentRequired: true
        },
        session: {
          sessionId: 'loading',
          guestId: 'loading',
          authenticated: false
        },
        correlation_id: `corr_${Date.now()}`
      },
      userId: 'loading', // Initial loading state

      // Actions
      setMetadata: newMeta => {
        set(
          state => ({
            metadata: merge(cloneDeep(state.metadata), newMeta, {
              correlation_id: newMeta.correlation_id || `corr_${Date.now()}`
            })
          }),
          false,
          'setMetadata'
        );
      },

      updateMetadata: updater => {
        set(
          state => ({
            metadata: updater(cloneDeep(state.metadata))
          }),
          false,
          'updateMetadata'
        );
      },

      // Initialize user ID from WASM
      initializeUserId: async () => {
        try {
          const wasmUserId = await getUserIdFromWasm();
          // Initializing with WASM user ID
          set({ userId: wasmUserId }, false, 'initializeUserId');

          // Also update the metadata with the new user ID
          set(
            state => ({
              metadata: {
                ...state.metadata,
                user: {
                  ...state.metadata.user,
                  userId: wasmUserId
                },
                session: {
                  ...state.metadata.session,
                  guestId: wasmUserId
                }
              }
            }),
            false,
            'updateUserIdInMetadata'
          );
        } catch (error) {
          console.warn('[MetadataStore] Failed to initialize userId from WASM:', error);
          // Use fallback IDs if WASM is not available
          console.log('[MetadataStore] Using fallback IDs...');
          const fallbackUserId = `guest_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
          const fallbackDeviceId = `device_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
          const fallbackSessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

          set({ userId: fallbackUserId }, false, 'initializeUserIdFallback');
          set(
            state => ({
              metadata: {
                ...state.metadata,
                user: {
                  ...state.metadata.user,
                  userId: fallbackUserId
                },
                device: {
                  ...state.metadata.device,
                  deviceId: fallbackDeviceId
                },
                session: {
                  ...state.metadata.session,
                  sessionId: fallbackSessionId,
                  guestId: fallbackUserId
                }
              }
            }),
            false,
            'updateFallbackMetadata'
          );
        }
      },

      // Initialize metadata with WASM-generated IDs
      initializeMetadata: async () => {
        try {
          const metadata = await createInitialMetadata();
          set({ metadata }, false, 'initializeMetadata');
          console.log('[MetadataStore] Metadata initialized with WASM IDs:', metadata);
        } catch (error) {
          console.warn('[MetadataStore] Failed to initialize metadata with WASM IDs:', error);
          // Keep the loading state if WASM is not available
        }
      },

      // Update campaign-specific metadata when campaign switches
      updateCampaignMetadata: (campaignData: any) => {
        console.log('[MetadataStore] 🔄 Updating campaign metadata:', {
          campaignData,
          currentCampaign: get().metadata.campaign,
          userId: get().userId,
          timestamp: new Date().toISOString()
        });

        const previousCampaign = get().metadata.campaign;

        set(
          state => ({
            metadata: {
              ...state.metadata,
              campaign: {
                id: campaignData.id || state.metadata.campaign.id,
                slug: campaignData.slug || state.metadata.campaign.slug,
                title: campaignData.title || state.metadata.campaign.name,
                status: campaignData.status || state.metadata.campaign.status,
                features: campaignData.features || state.metadata.campaign.features,
                last_switched: new Date().toISOString(),
                ...campaignData // Merge any additional campaign data
              }
            }
          }),
          false,
          'updateCampaignMetadata'
        );

        const newCampaign = get().metadata.campaign;
        console.log('[MetadataStore] ✅ Campaign metadata updated successfully:', {
          previous: {
            id: previousCampaign.id,
            slug: previousCampaign.slug,
            title: previousCampaign.name
          },
          new: {
            id: newCampaign.id,
            slug: newCampaign.slug,
            title: newCampaign.title
          },
          switchTime: newCampaign.last_switched,
          featuresCount: newCampaign.features?.length || 0
        });
      },

      // Sync with campaign state changes (called from campaign store)
      syncWithCampaignState: (campaignState: any) => {
        console.log('[MetadataStore] 🔄 Syncing with campaign state:', {
          campaignState,
          currentMetadata: get().metadata.campaign,
          userId: get().userId,
          timestamp: new Date().toISOString()
        });

        if (campaignState && campaignState.id) {
          const previousState = get().metadata.campaign;

          set(
            state => ({
              metadata: {
                ...state.metadata,
                campaign: {
                  ...state.metadata.campaign,
                  id: campaignState.id,
                  slug: campaignState.slug || state.metadata.campaign.slug,
                  title: campaignState.title || state.metadata.campaign.name,
                  status: campaignState.status || state.metadata.campaign.status,
                  features: campaignState.features || state.metadata.campaign.features,
                  last_switched: new Date().toISOString(),
                  ...campaignState // Merge any additional state data
                }
              }
            }),
            false,
            'syncWithCampaignState'
          );

          const newState = get().metadata.campaign;
          console.log('[MetadataStore] ✅ Campaign state synced successfully:', {
            previous: {
              id: previousState.id,
              status: previousState.status,
              featuresCount: previousState.features?.length || 0
            },
            new: {
              id: newState.id,
              status: newState.status,
              featuresCount: newState.features?.length || 0
            },
            changes: {
              idChanged: previousState.id !== newState.id,
              statusChanged: previousState.status !== newState.status,
              featuresChanged:
                JSON.stringify(previousState.features) !== JSON.stringify(newState.features)
            }
          });
        } else {
          console.warn('[MetadataStore] ⚠️ Invalid campaign state for sync:', campaignState);
        }
      },

      // Handle user ID changes from WASM (guest → authenticated migration)
      handleUserIDChange: (newUserId: string) => {
        set(
          state => ({
            userId: newUserId,
            metadata: {
              ...state.metadata,
              user: {
                ...state.metadata.user,
                userId: newUserId,
                username: state.metadata.user?.username || 'Guest User'
              },
              session: {
                ...state.metadata.session,
                guestId: newUserId,
                authenticated: !newUserId.startsWith('guest_')
              }
            }
          }),
          false,
          'handleUserIDChange'
        );
        console.log('[MetadataStore] User ID updated from WASM:', newUserId);
      },

      get campaignId(): string | number {
        try {
          const state = get();
          const metadata = state?.metadata;
          return metadata?.campaign?.id || metadata?.campaign?.slug || 0;
        } catch (error) {
          console.warn('[MetadataStore] Error accessing campaignId:', error);
          return 0;
        }
      },

      // Debugging helpers
      getStateSnapshot: () => {
        const state = get();
        return {
          metadata: state.metadata,
          userId: state.userId,
          campaignId: state.metadata?.campaign?.id || state.metadata?.campaign?.slug || 0
        };
      }
    }),
    {
      name: 'metadata-store'
    }
  )
);
