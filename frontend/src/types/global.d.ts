import type { EventEnvelope } from '../store/global';

declare global {
  interface Window {
    /**
     * Exposed by the WASM bridge for sending events to the backend.
     */
    sendWasmMessage?: (event: EventEnvelope<any>) => void;

    /**
     * Exposed by WASM/JS to get initial guest session information.
     */
    getGuestInfo?: () => { guestId: string; sessionId: string };

    /**
     * Exposed by WASM/JS to trigger the migration from a guest to a full user.
     */
    migrateUser?: (userId: string) => void;

    /**
     * Set and updated only by WASM. Source of truth for user identity.
     */
    userID: string;

    /**
     * Called by WASM when userID changes (migration, etc).
     */
    onUserIDChanged?: (newId: string) => void;

    /**
     * Temporary status object for WASM readiness and connection, set by frontend before Zustand store is initialized.
     */
    __pendingWasmStatus?: { wasmReady?: boolean; connected?: boolean };
  }
}

// This is necessary to make the file a module, which is required for 'declare global'.
export {};
