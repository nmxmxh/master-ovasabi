// Main store exports
export * from './types';

// Store exports
export { useConnectionStore } from './stores/connectionStore';
export { useEventStore } from './stores/eventStore';
export { useCampaignStore } from './stores/campaignStore';
export { useMetadataStore } from './stores/metadataStore';

// Hook exports
export * from './hooks/useConnection';
export * from './hooks/useEvents';
export * from './hooks/useCampaign';
export * from './hooks/useMetadata';
export * from './hooks/useGPU';
export * from './hooks/useWasmInitialization';

// Legacy compatibility exports (for gradual migration)
export { useConnectionStatus, useMediaStreamingState } from './hooks/useConnection';
export { useEmitEvent, useEventHistory, useEventState } from './hooks/useEvents';
export { useCampaignState, useCampaignUpdates } from './hooks/useCampaign';
export {
  useMetadata,
  useUserMetadata,
  useDeviceMetadata,
  useSessionMetadata
} from './hooks/useMetadata';
export { useGPUCapabilities } from './hooks/useGPU';
