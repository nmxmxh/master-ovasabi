import { create } from 'zustand';
import { useEffect } from 'react';
// import { shallow } from 'zustand/shallow';
import { useProfile } from './useProfile';

// --- Types ---
export interface DeviceMetadata {
  deviceId: string;
  sessionId: string;
  userAgent?: string;
  platform?: string;
  language?: string;
  timezone?: string;
  consentGiven: boolean;
  [key: string]: any;
}

export interface CampaignMetadata {
  campaignId?: string;
  campaignName?: string;
  [key: string]: any;
}

export interface OwnerMetadata {
  ownerId?: string;
  referralCode?: string;
  [key: string]: any;
}

export interface SchedulerMetadata {
  scheduleId?: string;
  scheduleType?: string;
  [key: string]: any;
}

export interface KnowledgeGraphMetadata {
  graphId?: string;
  nodeIds?: string[];
  [key: string]: any;
}

export interface Metadata {
  device: DeviceMetadata;
  campaign?: CampaignMetadata;
  owner?: OwnerMetadata;
  scheduler?: SchedulerMetadata;
  knowledgeGraph?: KnowledgeGraphMetadata;
  [key: string]: any;
}

// --- Zustand Store ---
interface MetadataState {
  metadata: Metadata;
  setMetadata: (meta: Partial<Metadata>) => void;
  mergeMetadata: (meta: Partial<Metadata>) => void;
  resetMetadata: () => void;
}

const defaultDeviceMetadata = (): DeviceMetadata => {
  let deviceId = localStorage.getItem('device_id');
  if (!deviceId) {
    deviceId = crypto.randomUUID();
    localStorage.setItem('device_id', deviceId);
  }
  let sessionId = sessionStorage.getItem('session_id');
  if (!sessionId) {
    sessionId = crypto.randomUUID();
    sessionStorage.setItem('session_id', sessionId);
  }
  const userAgent = typeof window !== 'undefined' ? window.navigator.userAgent : '';
  const userAgentData = typeof window !== 'undefined' && (window.navigator as any).userAgentData;
  const deviceType =
    typeof window !== 'undefined'
      ? /Mobi|Android|iPhone|iPad|iPod|Mobile|Tablet/i.test(userAgent)
        ? 'mobile'
        : 'desktop'
      : 'unknown';
  const isTouch = typeof window !== 'undefined' ? 'ontouchstart' in window : false;
  const isHeadless =
    typeof window !== 'undefined' && /HeadlessChrome|puppeteer|phantomjs|slimerjs/i.test(userAgent);

  return {
    deviceId,
    sessionId,
    userAgent,
    deviceType,
    isTouch,
    isHeadless,
    platform: userAgentData?.platform || undefined, // Only if available, else undefined
    language: typeof window !== 'undefined' ? window.navigator.language : '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    consentGiven: !!localStorage.getItem('consent')
  };
};

const initialMetadata: Metadata = {
  device: defaultDeviceMetadata()
};

export const useMetadataStore = create<MetadataState>((set, get) => ({
  metadata: initialMetadata,
  setMetadata: meta => set({ metadata: { ...get().metadata, ...meta } }),
  mergeMetadata: meta =>
    set({
      metadata: {
        ...get().metadata,
        ...meta,
        device: { ...get().metadata.device, ...(meta.device || {}) },
        campaign: { ...get().metadata.campaign, ...(meta.campaign || {}) },
        owner: { ...get().metadata.owner, ...(meta.owner || {}) },
        scheduler: { ...get().metadata.scheduler, ...(meta.scheduler || {}) },
        knowledgeGraph: { ...get().metadata.knowledgeGraph, ...(meta.knowledgeGraph || {}) }
      }
    }),
  resetMetadata: () => set({ metadata: initialMetadata })
}));

// --- Hook ---
export function useMetadata(select?: (meta: Metadata) => any) {
  // Zustand selector returns unknown by default, so cast to Metadata
  const metadata = useMetadataStore(state => state.metadata) as Metadata;
  const mergeMetadata = useMetadataStore(state => state.mergeMetadata);

  // Sync with profile/session changes
  const { profile } = useProfile({
    campaignId: metadata.campaign?.campaignId || '',
    userId: metadata.owner?.ownerId
  });
  useEffect(() => {
    if (profile) {
      mergeMetadata({
        owner: profile.user_id ? { ownerId: String(profile.user_id) } : undefined,
        campaign: profile.campaign_specific
          ? {
              campaignId: String(profile.campaign_specific.campaign_id || ''),
              campaignName: profile.campaign_specific.slug
            }
          : undefined,
        device: {
          ...metadata.device,
          sessionId: profile.session_id || metadata.device.sessionId
        }
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profile]);

  // Optionally select a slice
  return select ? select(metadata) : metadata;
}

// --- GDPR/Consent helpers ---

export function setConsentGiven(consent: boolean) {
  localStorage.setItem('consent', consent ? '1' : '');
  const device = useMetadataStore.getState().metadata.device;
  useMetadataStore.getState().mergeMetadata({
    device: { ...device, consentGiven: consent }
  });
}

export function eraseMetadata() {
  localStorage.removeItem('device_id');
  sessionStorage.removeItem('session_id');
  localStorage.removeItem('consent');
  useMetadataStore.getState().resetMetadata();
}
