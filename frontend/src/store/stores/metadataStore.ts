import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
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

  // Getters
  get campaignId(): string | number;
}

// Note: Secure ID generation functions are now imported from cryptoIds utility

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

    return {
      campaign: {
        campaignId: '0',
        features: [],
        slug: 'default'
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
        gdprConsentRequired: true
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
    // Fallback metadata in case of error
    return {
      campaign: {
        campaignId: '0',
        features: [],
        slug: 'default'
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
        gdprConsentRequired: true
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
    persist(
      (set, get) => ({
        // Initial metadata - will be populated asynchronously
        metadata: {
          campaign: {
            campaignId: '0',
            features: [],
            slug: 'default',
            campaignName: 'Default Campaign',
            status: 'active' as const
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
              typeof Intl !== 'undefined'
                ? Intl.DateTimeFormat().resolvedOptions().timeZone
                : 'UTC',
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
            console.log('[MetadataStore] Initializing with WASM user ID:', wasmUserId);
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
            return metadata?.campaign?.campaignId || metadata?.campaign?.slug || 0;
          } catch (error) {
            console.warn('[MetadataStore] Error accessing campaignId:', error);
            return 0;
          }
        }
      }),
      {
        name: 'metadata-store',
        partialize: () => {
          // Don't persist user IDs - always get them from WASM
          return {
            metadata: {
              campaign: { campaignId: '0', features: [], slug: 'default' },
              user: { userId: 'loading', username: 'Loading...' },
              device: {
                deviceId: 'loading',
                userAgent: '',
                language: 'en-US',
                timezone: 'UTC',
                consentGiven: false,
                gdprConsentRequired: true
              },
              session: { sessionId: 'loading', guestId: 'loading', authenticated: false },
              correlation_id: 'loading'
            },
            userId: 'loading'
          };
        }
      }
    ),
    {
      name: 'metadata-store'
    }
  )
);
