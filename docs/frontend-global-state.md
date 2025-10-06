# Canonical React Global State & Event Architecture

## Overview

This document describes the canonical approach for global state management in the OVASABI frontend,
ensuring full alignment with backend event standards, WebSocket/WASM integration, and
campaign-driven architecture. This pattern is required for all new features and services as of
July 2025.

---

## 1. WebSocket Broadcast Types

The backend (`ws-gateway`) supports three main broadcast types:

- **System Broadcast**: Sent to all connected clients (system-wide notifications).
- **Campaign Broadcast**: Sent to all clients in a specific campaign (campaign events, updates).
- **User Broadcast**: Sent to a specific user (personal notifications, direct responses).

All events use a canonical envelope:

```json
{
  "type": "{service}:{action}:v{version}:{state}",
  "payload": { ... },
  "metadata": { ... }
}
```

Metadata always includes campaign, user, device, and session context.

---

## 2. Frontend Hooks & Metadata Consistency

- `useMetadata.ts`: Centralizes device, campaign, owner, scheduler, and knowledge graph metadata.
  Syncs with profile/session.
- `useProfile.ts`: Manages user profile, campaign, privileges, and session state. Handles WASM
  bridge and WebSocket integration.
- Both hooks propagate and merge metadata, but must ensure all canonical fields are always present
  and up-to-date.

---

## 3. Canonical Global State Blueprint

### Types (Extended for Campaign, Scripts, Accessibility, GDPR)

````typescript
export interface EventEnvelope {
  type: string; // {service}:{action}:v{version}:{state}
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
  features: string[]; // e.g. ['waitlist', 'referral', ...]
  serviceSpecific?: {
    campaign?: Record<string, any>;
    localization?: {
      scripts?: Record<string, ScriptBlock>;
      scripts_translations?: Record<string, any>; // Raw translations per locale
      scripts_translated?: Record<string, ScriptBlock>; // Fully translated scripts, updated by localization service
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
  // GDPR/consent fields
  gdprConsentTimestamp?: string;
  gdprConsentRequired?: boolean;
}

export interface SessionMetadata {
  sessionId: string;
  guestId?: string;
  authenticated?: boolean;
}
### Zustand Store

```typescript
import { create } from 'zustand';

interface GlobalState {
  metadata: Metadata;
  events: EventEnvelope[];
  state: Record<string, string>; // eventType -> state
  setMetadata: (meta: Partial<Metadata>) => void;
  emitEvent: (event: Omit<EventEnvelope, 'timestamp'>) => void;
  updateState: (eventType: string, state: string) => void;
  reset: () => void;
}

export const useGlobalStore = create<GlobalState>((set, get) => ({
  metadata: /* initialMetadata */,
  events: [],
  state: {},
  setMetadata: meta => set({ metadata: { ...get().metadata, ...meta } }),
  emitEvent: event => {
    if (window.wasmBridge?.send) window.wasmBridge.send({ ...event, timestamp: Date.now() });
    set(state => ({
      events: [...state.events, { ...event, timestamp: Date.now() }],
      state: { ...state.state, [event.type]: event.type.split(':').pop() }
    }));
  },
  updateState: (eventType, newState) => set(state => ({ state: { ...state.state, [eventType]: newState } })),
  reset: () => set({ metadata: /* initialMetadata */, events: [], state: {} })
}));
````

### WebSocket/WASM Integration Hook

```typescript
import { useEffect } from 'react';
import { useWasmBridge } from './useWasmBridge';
import { useGlobalStore } from './useGlobalStore';

export function useGlobalEventSync() {
  const emitEvent = useGlobalStore(state => state.emitEvent);
  const setMetadata = useGlobalStore(state => state.setMetadata);

  const { connected, send, onMessage } = useWasmBridge({
    autoConnect: true,
    onMessage: (msg: any) => {
      if (msg?.type && msg?.metadata) {
        emitEvent(msg);
        setMetadata(msg.metadata);
      }
    }
  });

  // Optionally, handle direct WebSocket events here as well

  return { connected, send };
}
```

---

## 3a. Campaign, Scripts, Accessibility, and GDPR in Global State

- The `campaign` field in global state now includes all backend metadata: features, service-specific
  config, scheduling, versioning, and audit.
- `serviceSpecific.localization.scripts` holds all onboarding and dialogue flows, supporting dynamic
  UI and accessibility for different user types (business, talent, pioneer, hustler, etc).
- Accessibility metadata is included in each script/question for ARIA/alt text and semantic
  rendering.
- GDPR/consent is tracked in both device and campaign metadata, ensuring compliance and user
  transparency.

**Best Practice:**  
Always use the canonical campaign structure for all campaign-driven UI, onboarding, and event flows.
Scripts and accessibility metadata should be dynamically loaded and rendered. GDPR/consent state
must be respected in all user interactions.

---

## 3b. Localization Feedback Loop: scripts_translated

- The `scripts_translated` field in `serviceSpecific.localization` enables the localization service
  to inject fully translated dialogue flows directly into the global state.
- This creates a real-time feedback loop: backend/localization service updates translations,
  frontend state is updated, and UI reflects the latest localized content instantly.
- Use this for dynamic, multi-lingual onboarding, accessibility, and campaign flows.

**Best Practice:** Always render onboarding/dialogue UI from `scripts_translated` if available,
falling back to `scripts` as needed. Ensure the localization service can update this state via
events or API.

---

## 4. Usage & Best Practices

- Use `useGlobalStore()` anywhere in your app to access or update metadata, emit events, or listen
  for state changes.
- Call `useGlobalEventSync()` at the root of your app to keep state in sync with backend events.
- All event emission and handling is standards-compliant and future-proof.
- Never hardcode event types or metadata fields—always use the canonical format and shared
  constants.
- Always merge/propagate metadata on every event.
- Validate all event types and keys against the registry at build/startup.
- Use the global state for all orchestration, UI, and business logic.

---

## 5. Next Steps

- Scaffold the above store and hooks into your codebase (e.g., `frontend/src/lib/globalState.ts`).
- Refactor all event emission and metadata access to use the global store.
- Ensure all contributors follow this pattern for new features and services.

---

## References

- `docs/service-refactor.md`
- `docs/communication_standards.md`
- `frontend/src/lib/hooks/useMetadata.ts`
- `frontend/src/lib/hooks/useProfile.ts`
- `internal/server/ws-gateway/main.go`
- `wasm/main.go`

---

This document is the canonical reference for frontend global state and event architecture as of
July 2025.
