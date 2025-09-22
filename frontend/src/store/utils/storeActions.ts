// Shared store actions to break circular dependencies
// This file contains utility functions that can be used by multiple stores
// without creating circular dependencies

// EventEnvelope type is used in interface definitions
import { withErrorHandling } from './errorHandling';

// Store action types
export interface StoreActions {
  handleConnectionStatus?: (connected: boolean, reason: string) => void;
  handleWasmMessage?: (msg: any) => void;
  setWasmReady?: (ready: boolean) => void;
  handleUserIDChange?: (newUserId: string) => void;
  updateCampaignMetadata?: (campaignData: any) => void;
  syncWithCampaignState?: (campaignState: any) => void;
}

// Store registry to avoid circular imports
class StoreRegistry {
  private stores: Map<string, StoreActions> = new Map();

  register(storeName: string, actions: StoreActions) {
    this.stores.set(storeName, actions);
  }

  get(storeName: string): StoreActions | undefined {
    return this.stores.get(storeName);
  }

  // Helper methods for common actions with error handling
  notifyConnectionStatus(connected: boolean, reason: string) {
    withErrorHandling(
      'connection',
      () => {
        const connectionStore = this.get('connection');
        if (connectionStore?.handleConnectionStatus) {
          connectionStore.handleConnectionStatus(connected, reason);
        }
      },
      { connected, reason }
    );
  }

  notifyWasmMessage(msg: any) {
    withErrorHandling(
      'event',
      () => {
        const eventStore = this.get('event');
        if (eventStore?.handleWasmMessage) {
          eventStore.handleWasmMessage(msg);
        }
      },
      { messageType: msg?.type }
    );
  }

  notifyWasmReady(ready: boolean) {
    withErrorHandling(
      'event',
      () => {
        const eventStore = this.get('event');
        if (eventStore?.setWasmReady) {
          eventStore.setWasmReady(ready);
        }
      },
      { wasmReady: ready }
    );
  }

  notifyUserIDChange(newUserId: string) {
    withErrorHandling(
      'metadata',
      () => {
        const metadataStore = this.get('metadata');
        if (metadataStore?.handleUserIDChange) {
          metadataStore.handleUserIDChange(newUserId);
        }
      },
      { newUserId }
    );
  }

  notifyCampaignMetadataUpdate(campaignData: any) {
    withErrorHandling(
      'metadata',
      () => {
        const metadataStore = this.get('metadata');
        if (metadataStore?.updateCampaignMetadata) {
          metadataStore.updateCampaignMetadata(campaignData);
        }
      },
      { campaignId: campaignData?.campaignId }
    );
  }

  notifyCampaignStateSync(campaignState: any) {
    withErrorHandling(
      'metadata',
      () => {
        const metadataStore = this.get('metadata');
        if (metadataStore?.syncWithCampaignState) {
          metadataStore.syncWithCampaignState(campaignState);
        }
      },
      { campaignId: campaignState?.campaignId }
    );
  }
}

// Export singleton instance
export const storeRegistry = new StoreRegistry();

// Utility functions for common store operations
export const storeUtils = {
  // Safe store action execution
  safeExecute: (action: (() => void) | undefined, context: string) => {
    try {
      if (action) {
        action();
      }
    } catch (error) {
      console.error(`[StoreUtils] Error in ${context}:`, error);
    }
  },

  // Batch store updates
  batchUpdate: (updates: Array<() => void>) => {
    try {
      updates.forEach(update => update());
    } catch (error) {
      console.error('[StoreUtils] Error in batch update:', error);
    }
  },

  // Store state validation
  validateStoreState: (state: any, requiredFields: string[]) => {
    const missingFields = requiredFields.filter(field => !(field in state));
    if (missingFields.length > 0) {
      console.warn('[StoreUtils] Missing required fields:', missingFields);
      return false;
    }
    return true;
  }
};
