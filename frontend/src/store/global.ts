import { create } from 'zustand';
import { useEffect, useCallback } from 'react';
import { useWasmBridge } from '../lib/hooks/useWasmBridge';
import { useProfileStore } from '../lib/hooks/useProfile';
import { setConsentGiven as setConsentGivenLegacy } from '../lib/hooks/useMetadata';

// --- Types (from canonical doc) ---
export interface EventEnvelope {
  type: string;
  payload: any;
  metadata: Metadata;
  correlationId?: string;
  timestamp?: number;
}

export interface Metadata {
  campaign: CampaignMetadata;
  user: UserMetadata;
  device: DeviceMetadata;
  session: SessionMetadata;
  [key: string]: any;
}

export interface CampaignMetadata {
  campaignId: string;
  campaignName?: string;
  slug?: string;
  features: string[];
  serviceSpecific?: {
    campaign?: Record<string, any>;
    localization?: {
      scripts?: Record<string, ScriptBlock>;
      scripts_translations?: Record<string, any>;
      scripts_translated?: Record<string, ScriptBlock>;
    };
  };
  scheduling?: Record<string, any>;
  versioning?: Record<string, any>;
  audit?: Record<string, any>;
  gdpr?: {
    consentRequired: boolean;
    privacyPolicyUrl?: string;
    termsUrl?: string;
    consentGiven?: boolean;
    consentTimestamp?: string;
  };
}

export interface ScriptBlock {
  main_text: string;
  options_title: string;
  options_subtitle: string;
  question_subtitle: string;
  questions: Array<{
    question: string;
    why_this_matters: string;
    options: string[];
    accessibility?: {
      ariaLabel?: string;
      altText?: string;
    };
  }>;
}

export interface UserMetadata {
  userId?: string;
  username?: string;
  privileges?: string[];
  referralCode?: string;
}

export interface DeviceMetadata {
  deviceId: string;
  userAgent?: string;
  platform?: string;
  language?: string;
  timezone?: string;
  consentGiven: boolean;
  gdprConsentTimestamp?: string;
  gdprConsentRequired?: boolean;
}

export interface SessionMetadata {
  sessionId: string;
  guestId?: string;
  authenticated?: boolean;
}

// --- Zustand Global Store ---

// --- State snapshot for recall/history ---
export interface GlobalStateSnapshot {
  metadata: Metadata;
  events: EventEnvelope[];
  state: Record<string, string>;
}

interface GlobalState {
  metadata: Metadata;
  events: EventEnvelope[];
  state: Record<string, string>;
  history: GlobalStateSnapshot[];
  eventTypes: string[];
  setEventTypes: (types: string[]) => void;
  setMetadata: (meta: Partial<Metadata>) => void;
  emitEvent: (event: Omit<EventEnvelope, 'timestamp'>) => void;
  updateState: (eventType: string, state: string) => void;
  reset: () => void;
  setConsentGiven: (consent: boolean) => void;
  pushHistory: () => void;
  recallState: (index: number) => void;
}

const initialMetadata: Metadata = {
  campaign: { campaignId: '', features: [] },
  user: {},
  device: {
    deviceId: '',
    consentGiven: false
  },
  session: { sessionId: '' }
};

export const useGlobalStore = create<GlobalState>((set, get) => ({
  metadata: initialMetadata,
  events: [],
  state: {},
  history: [],
  eventTypes: [],
  setEventTypes: (types: string[]) => set({ eventTypes: types }),
  // Push a snapshot of the current state to history
  pushHistory: () => {
    const { metadata, events, state, history } = get();
    set({
      history: [
        ...history,
        {
          metadata: JSON.parse(JSON.stringify(metadata)),
          events: JSON.parse(JSON.stringify(events)),
          state: JSON.parse(JSON.stringify(state))
        }
      ]
    });
  },
  // Recall a previous state snapshot by index
  recallState: (index: number) => {
    const { history } = get();
    if (history[index]) {
      set({
        metadata: JSON.parse(JSON.stringify(history[index].metadata)),
        events: JSON.parse(JSON.stringify(history[index].events)),
        state: JSON.parse(JSON.stringify(history[index].state))
      });
    }
  },
  setMetadata: meta => {
    set(state => {
      const newMeta = { ...state.metadata, ...meta };
      // Push to history before changing
      get().pushHistory();
      return { metadata: newMeta };
    });
  },
  emitEvent: event => {
    if ((window as any).wasmBridge?.send)
      (window as any).wasmBridge.send({ ...event, timestamp: Date.now() });
    set(state => {
      get().pushHistory();
      const eventState = event.type.split(':').pop() || '';
      return {
        events: [...state.events, { ...event, timestamp: Date.now() }],
        state: { ...state.state, [event.type]: eventState }
      };
    });
  },
  updateState: (eventType, newState) =>
    set(state => {
      get().pushHistory();
      return { state: { ...state.state, [eventType]: newState } };
    }),
  reset: () => {
    get().pushHistory();
    set({ metadata: initialMetadata, events: [], state: {} });
  },
  setConsentGiven: consent => {
    setConsentGivenLegacy(consent);
    set(state => {
      get().pushHistory();
      return {
        metadata: {
          ...state.metadata,
          device: { ...state.metadata.device, consentGiven: consent }
        }
      };
    });
  }
}));

// --- Shared WASM/JS Memory State (read-only, polled via requestAnimationFrame, animation-optimized) ---

type SharedState = {
  buffer: ArrayBuffer | null;
  data: Float32Array | null;
};

export const useSharedState = create<SharedState>(() => ({
  buffer: null,
  data: null
}));

export function useWasmSharedState(selector?: (data: Float32Array) => any) {
  const setShared = useSharedState.setState;
  // Use a module-level variable instead of useRef for animation polling
  let lastData: Float32Array | null = null;

  const update = useCallback(() => {
    const buffer = typeof window.getSharedBuffer === 'function' ? window.getSharedBuffer() : null;
    if (buffer) {
      const view = new Float32Array(buffer);
      // Always copy to protect from mutation
      const safeCopy = new Float32Array(view);
      // Only update if changed (shallow compare)
      if (!lastData || safeCopy.some((v, i) => v !== lastData![i])) {
        setShared({ buffer, data: safeCopy });
        lastData = safeCopy;
      }
    }
    requestAnimationFrame(update);
  }, [setShared]);

  useEffect(() => {
    requestAnimationFrame(update);
    return () => {
      /* polling stops on unmount */
    };
  }, [update]);

  // Selector for consumers
  const data = useSharedState.getState().data;
  return selector && data ? selector(data) : data;
}

// --- Global Event Sync Hook ---
export function useGlobalEventSync() {
  const emitEvent = useGlobalStore(state => state.emitEvent);
  const setMetadata = useGlobalStore(state => state.setMetadata);
  const { profile, mergeProfile } = useProfileStore();
  const { connected, send } = useWasmBridge({
    autoConnect: true,
    onMessage: (msg: any) => {
      if (msg?.type && msg?.metadata) {
        emitEvent(msg);
        setMetadata(msg.metadata);
      }
      if (msg?.type === 'profile' && msg?.payload) {
        mergeProfile(msg.payload);
      }
    }
  });
  // Sync profile changes into global state
  useEffect(() => {
    if (profile) {
      setMetadata({
        user: {
          userId: profile.user_id ? String(profile.user_id) : undefined,
          username: profile.username,
          privileges: profile.privileges
        },
        session: {
          sessionId: profile.session_id,
          guestId: profile.guest_id,
          authenticated: !!profile.user_id
        },
        campaign: profile.campaign_specific
          ? {
              campaignId: String(profile.campaign_specific.campaign_id || ''),
              campaignName: profile.campaign_specific.slug,
              features: []
            }
          : { campaignId: '', features: [] }
      });
    }
  }, [profile, setMetadata]);
  return { connected, send };
}
