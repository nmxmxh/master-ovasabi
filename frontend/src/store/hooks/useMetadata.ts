import { useMemo } from 'react';
import { useMetadataStore } from '../stores/metadataStore';

// Metadata hook
export function useMetadata() {
  const { metadata, setMetadata, updateMetadata, userId, campaignId } = useMetadataStore();

  return useMemo(
    () => ({
      metadata,
      setMetadata,
      updateMetadata,
      userId,
      campaignId
    }),
    [metadata, setMetadata, updateMetadata, userId, campaignId]
  );
}

// User metadata hook
export function useUserMetadata() {
  const metadata = useMetadataStore(state => state.metadata);

  return useMemo(
    () => ({
      user: metadata?.user,
      userId: metadata?.user?.userId || metadata?.session?.guestId || 'anonymous',
      username: metadata?.user?.username,
      privileges: metadata?.user?.privileges,
      referralCode: metadata?.user?.referralCode
    }),
    [metadata?.user, metadata?.session?.guestId]
  );
}

// Device metadata hook
export function useDeviceMetadata() {
  const metadata = useMetadataStore(state => state.metadata);

  return useMemo(
    () => ({
      device: metadata?.device,
      deviceId: metadata?.device?.deviceId,
      userAgent: metadata?.device?.userAgent,
      language: metadata?.device?.language,
      timezone: metadata?.device?.timezone,
      consentGiven: metadata?.device?.consentGiven,
      gpuCapabilities: metadata?.device?.gpuCapabilities,
      wasmGPUBridge: metadata?.device?.wasmGPUBridge
    }),
    [metadata?.device]
  );
}

// Session metadata hook
export function useSessionMetadata() {
  const metadata = useMetadataStore(state => state.metadata);

  return useMemo(
    () => ({
      session: metadata?.session,
      sessionId: metadata?.session?.sessionId,
      guestId: metadata?.session?.guestId,
      authenticated: metadata?.session?.authenticated
    }),
    [metadata?.session]
  );
}
