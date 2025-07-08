import { useWasmBridge } from './useWasmBridge';
import { useEffect, useCallback } from 'react';
// ...existing code...
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import deepmerge from 'deepmerge';
// ...existing code...

// --- Types ---
export interface UIState {
  theme?: string;
  locale?: string;
  direction?: string;
  contrast?: {
    prefersHighContrast?: boolean;
    colorContrastRatio?: string;
  };
  animationDirection?: string;
  orientation?: string;
  menu?: string;
  footerLink?: string;
  viewport?: {
    width?: number;
    height?: number;
    dvw?: number;
    dvh?: number;
  };
  motion?: {
    prefersReducedMotion?: boolean;
  };
  // ...extend as needed
}

export interface ServiceSpecific {
  waitlist?: any;
  referral?: any;
  quote?: any;
  // ...extend as needed
}

export interface CampaignSpecific {
  campaign_id?: number;
  slug?: string;
  ui?: Record<string, any>;
  rules?: Record<string, any>;
  // ...extend as needed
}

export interface UnifiedProfile {
  session_id: string;
  user_id?: number;
  waitlist_id?: number;
  guest_id?: string;
  email?: string;
  username?: string;
  display_name?: string;
  privileges?: string[];
  ui_state?: UIState;
  service_specific?: ServiceSpecific;
  campaign_specific?: CampaignSpecific;
  // ...extend as needed
}

type Status = 'connecting' | 'connected' | 'disconnected' | 'error';

interface ProfileStore {
  profile: UnifiedProfile | null;
  status: Status;
  error: string | null;
  setProfile: (p: UnifiedProfile) => void;
  mergeProfile: (p: Partial<UnifiedProfile>) => void;
  setStatus: (s: Status) => void;
  setError: (e: string | null) => void;
  reset: () => void;
}

export const useProfileStore = create<ProfileStore>()(
  persist(
    (set, get) => ({
      profile: null,
      status: 'disconnected',
      error: null,
      setProfile: (p: UnifiedProfile) => {
        // Normalize guest/user/waitlist
        let normalized: UnifiedProfile = { ...p };
        if (!normalized.session_id) {
          normalized.session_id = `guest_${Math.random().toString(36).slice(2)}`;
        }
        if (!normalized.privileges || normalized.privileges.length === 0) {
          normalized.privileges = ['guest'];
        }
        // If user_id exists, remove guest_id
        if (normalized.user_id && normalized.guest_id) {
          normalized.guest_id = undefined;
        }
        // If waitlist_id exists, add 'waitlist' privilege
        if (normalized.waitlist_id && !normalized.privileges.includes('waitlist')) {
          normalized.privileges = [...normalized.privileges, 'waitlist'];
        }
        set({ profile: normalized });
      },
      mergeProfile: (p: Partial<UnifiedProfile>) => {
        const current = get().profile;
        let merged: UnifiedProfile;
        if (current) {
          merged = deepmerge(current, p) as UnifiedProfile;
        } else {
          merged = p as UnifiedProfile;
        }
        // Normalize guest/user/waitlist
        if (!merged.session_id) {
          merged.session_id = `guest_${Math.random().toString(36).slice(2)}`;
        }
        if (!merged.privileges || merged.privileges.length === 0) {
          merged.privileges = ['guest'];
        }
        if (merged.user_id && merged.guest_id) {
          merged.guest_id = undefined;
        }
        if (merged.waitlist_id && !merged.privileges.includes('waitlist')) {
          merged.privileges = [...merged.privileges, 'waitlist'];
        }
        set({ profile: merged });
      },
      setStatus: (s: Status) => set({ status: s }),
      setError: (e: string | null) => set({ error: e }),
      reset: () =>
        set({
          profile: null,
          status: 'disconnected',
          error: null
        })
    }),
    {
      name: 'profile-store',
      partialize: (state: ProfileStore) => ({ profile: state.profile })
    }
  )
);

// --- WebSocket Hook ---
interface UseProfileOpts {
  campaignId: string | number;
  userId?: string | number | null;
  wsOrigin?: string; // e.g. ws://localhost:8090
  autoConnect?: boolean;
}

export function useProfile({ campaignId, userId, wsOrigin, autoConnect = true }: UseProfileOpts) {
  const { profile, status, error, setProfile, mergeProfile, setStatus, setError, reset } =
    useProfileStore();

  // --- WebSocket connection using useWebSocketConnection ---
  // --- WASM Bridge integration ---
  const { connected, send } = useWasmBridge({
    autoConnect,
    onMessage: msg => {
      try {
        if (msg.type === 'profile' && msg.payload) {
          mergeProfile(msg.payload);
        } else if (msg.type === 'error') {
          setError(msg.payload?.message || 'Server error');
          setStatus('error');
        }
      } catch (e) {
        setError('Malformed message from server');
        setStatus('error');
      }
    }
  });

  useEffect(() => {
    if (!autoConnect || !campaignId) {
      setStatus('disconnected');
      setError('No campaignId provided');
      return;
    }
    // Reset profile state on campaign/user change
    reset();
    setStatus('connecting');
    setError(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [campaignId, userId, wsOrigin, autoConnect]);

  // --- API Methods ---

  const updateProfile = useCallback(
    (partial: Partial<UnifiedProfile>) => {
      // Optimistically update local state
      mergeProfile(partial);
      // Send to backend
      if (connected) {
        try {
          send({
            type: 'profile.update',
            payload: partial
          });
        } catch (e) {
          setError('Failed to send update to server');
          setStatus('error');
        }
      } else {
        setError('WASM bridge not connected');
        setStatus('error');
      }
    },
    [mergeProfile, setError, setStatus]
  );

  function switchCampaign(newCampaignId: string | number, newUserId?: string | number) {
    // This function can be used to programmatically switch campaign/user and reset profile state
    // Usage: call this, then update the hook's params in your component
    reset();
    // Optionally, set a new guest profile for the new campaign/user
    let guestId = newUserId || `guest_${Math.random().toString(36).slice(2)}`;
    setProfile({
      session_id: typeof guestId === 'string' ? guestId : String(guestId),
      guest_id: typeof guestId === 'string' ? guestId : String(guestId),
      privileges: ['guest'],
      ui_state: {},
      service_specific: {},
      campaign_specific: { campaign_id: typeof newCampaignId === 'number' ? newCampaignId : 0 }
    });
    // The effect in useProfile will handle the actual reconnection
  }

  return {
    profile,
    status,
    error,
    updateProfile,
    switchCampaign, // for documentation; actual switching is via hook params
    reset
  };
}
