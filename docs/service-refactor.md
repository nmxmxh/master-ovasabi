# Search Service Refactor: Communication & Event Standards Compliance (Canonical Standard, July 2025)

## Overview

This document details the refactor of the Search Service to comply with the OVASABI Communication & Event Naming Standards. The refactor ensures consistent event naming, robust metadata handling, seamless orchestration with `graceful`, correct state event emission, support for partial updates, and proper integration with system and shared packages.

---

## 1. Event & Channel Naming

- All events, channels, and keys now use the canonical format:  
  `{service}:{action}:v{version}:{state}`  
  Example: `search:search:v1:requested`, `search:suggest:v1:success`
- All event types and keys use only the allowed states: `requested`, `started`, `success`, `failed`, `completed`
- All event emission, subscription, and key usage reference generated constants (see `events/constants.go` or equivalent in TypeScript)

---

## 2. Generic Canonical Event Handling (2025 Pattern)

- **All search service actions (e.g., `search`, `suggest`, etc.) are handled by a single, generic event handler.**
- The event handler parses the `{action}` and `{state}` from the canonical event type and dispatches to the correct business logic using a map (e.g., `actionHandlers`).
- **All per-action event handler functions (e.g., `HandleSearchRequestedEvent`, `HandleSuggestRequestedEvent`) have been removed.**
- All business logic for each action is implemented in a generic handler function (e.g., `handleSearchAction`) and registered in the `actionHandlers` map in `events.go`.
- All orchestration, error handling, and state transitions use canonical event types and key patterns, and are performed via the generic handler.
- Adding a new action only requires registering a new business logic handler in the `actionHandlers` map.
- All event types and keys are validated and loaded from the registry at startup.

---

## 3. Metadata Handling

- All emitted events include a `correlation_id` for end-to-end tracing
- Event payloads include relevant metadata fields such as `campaign_id`, `user_id`, and any domain-specific context
- Metadata is propagated through all layers (API, business logic, orchestration, and storage)
- All logs include the full event type/key and correlation ID

---

## 4. Integration with `graceful` Orchestration

- All orchestration, error handling, and state transitions use canonical event types and key patterns
- State transitions (e.g., from `requested` to `started` to `success`/`failed`/`completed`) are explicitly emitted as events
- Graceful workflows emit and listen for events using the `{service}:{action}:v{version}:{state}` format
- All orchestration logic references generated event constants

---

## 5. State Event Emission

- Every significant state change in the search workflow emits a corresponding event:
  - `search:search:v1:requested` (search initiated)
  - `search:search:v1:started` (processing started)
  - `search:search:v1:success` (results found)
  - `search:search:v1:failed` (error occurred)
  - `search:search:v1:completed` (finalization)
- Event emission is centralized and uses shared utilities from `pkg/events` or similar
- All event emissions are validated against the registry at build/startup

---

## 6. Partial Updates

- Partial search result updates emit events with the same canonical format, e.g., `search:search:v1:success` with a partial payload and a `partial: true` flag in metadata
- Consumers can distinguish between full and partial updates via the event payload
- Partial updates are supported in both event emission and state management logic

---

## 7. System, Pkg, and Service Registration Integration

- All event, channel, and key constants are imported from shared packages (e.g., `pkg/events`, `pkg/constants`).
- Shared utilities for event emission, metadata propagation, and validation are used throughout the service.
- System-level events (e.g., health checks, system-wide notifications) also follow the canonical format and are handled in the same way.

### Service Registration Generator as Source of Truth

The `pkg/registration/generator.go` provides a dynamic, registry-driven mechanism for generating all canonical event types, key patterns, and Go/TypeScript constants for every service, including the search service. This ensures:

- **Automatic Event Type Generation:**
  - The generator analyzes service registration configs (from proto/service definitions) and produces all valid event types in the `{service}:{action}:v{version}:{state}` format, using the canonical state vocabulary.
- **Go/TS Constant Generation:**
  - Outputs Go constants (for use in code) and JSON (for docs/validation) via `WriteEventTypesGo` and `WriteEventTypesJSON`.
- **Registry-Driven:**
  - All event types, keys, and patterns are derived from the service registration registry, ensuring a single source of truth.
- **Validation:**
  - At build/startup, emitted/subscribed event types and Redis keys can be validated against the generated registry.

#### How to Use in Service Refactor

1. **Update Service Registration:**
   Ensure your service and its methods are correctly defined in the service registration config (e.g., `service_registration.json`) or proto files.
2. **Run the Generator:**
   Use the generator to produce event type constants and key patterns for your service:
   - Go: `WriteEventTypesGo` → outputs Go constants for use in your service code.
   - JSON: `WriteEventTypesJSON` → outputs for documentation and validation.
3. **Reference Generated Constants:**
   In your service code, always use the generated constants for event emission, subscription, and key usage.
4. **CI/Validation:**
   Integrate validation to ensure all event types used in code are present in the generated registry.

#### Service Registration Validity

- The current `service_registration.json` entry for the search service is valid and includes:
  - `name`: "search"
  - `version`: "v1"
  - `schema.methods`: ["Search", "Suggest"]
  - `action_map` for both `search` and `suggest` actions, mapping to proto methods and request/response models.
- This structure enables the generator to produce all canonical event types and key patterns for the search service, ensuring compliance with the July 2025 standards.

This approach guarantees that all event, channel, and key usage in the codebase is always up-to-date, consistent, and validated against the canonical service registration schema.

---

## 8. Event Routing

- Event payloads include `campaign_id` and/or `user_id` as appropriate for routing
- Routing is determined by payload fields, not event type
- No per-event-type routing logic; the gateway and Nexus use generic handlers

---

## 9. Validation & Linting

- At build/startup, all emitted/subscribed event types and Redis keys are validated against the registry
- CI linter checks for naming convention violations

---

## 10. Backward Compatibility

- Legacy event/key names are aliased or migrated to the canonical format
- Transitional support is provided for existing consumers

---

## 11. Security & Access Control

- Namespaces and prefixes are used to prevent cross-service collisions
- Access control is enforced for event publication and subscription

---

## 12. References

- See `docs/communication_standards.md` for the full standard
- See `events/constants.go` and `events/event_types.json` for the list of valid event types and key patterns

---

## 13. Testing & Validation

- All event emissions and subscriptions are covered by unit and integration tests
- Startup validation ensures all event types are registered and conform to standards

---

## 14. Contributor Guidance

- All new code and services must comply with these standards
- PRs introducing non-canonical event types or key patterns will be rejected

---

## 15. Canonical Event Type Discovery and Validation

- Both frontend (WASM) and backend use the canonical `service_registration.json` as the source of truth for all event/action types.
- The WASM code dynamically loads and logs all valid event types at runtime for verification (see `wasm/main.go`, `loadServiceEventTypes`).
- The backend generator parses the same file and outputs:
  - Go constants for use in code (compile-time safety).
  - A JSON file for runtime validation and documentation.
- At backend startup, a validation step should ensure all event types used in code are present in the generated registry, preventing drift.
- This guarantees that all event emission, subscription, and key usage are always up-to-date and compliant with the canonical registry.

### Feedback on the Process

- **Strengths:**
  - Single source of truth (`service_registration.json`).
  - Both frontend and backend are always in sync.
  - Easy to add new event types—just update the registration and re-run the generator.
  - Compile-time safety for Go, runtime safety for WASM/JS.
- **Improvements:**
  - Automate generator runs in CI to prevent stale constants.
  - Add startup validation in Go to catch any non-canonical event usage.
  - Optionally, expose the event type list via an admin endpoint for observability.
- **Risks:**
  - If the generator is not run after changes, drift can occur.
  - If event types are hardcoded anywhere, they may become non-canonical.

---

This document is the canonical standard for service functionality as of July 2025. All contributors must read and follow these standards.
