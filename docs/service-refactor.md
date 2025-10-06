# Search Service Refactor: Communication & Event Standards Compliance (Canonical Standard, July 2025)

## Overview

This document details the July 2025 refactor of the Search Service, focusing on:

- Canonical event emission via centralized `graceful.Handler` orchestration
- Generic action handler pattern (single handler, action/state dispatch)
- Provider-driven event registration and routing
- Registry-driven event constants and validation
- Robust metadata propagation and observability
- Support for partial updates and state transitions
- Security, access control, and backward compatibility

---

## 1. Event & Channel Naming

`{service}:{action}:v{version}:{state}`  
 Example: `search:search:v1:requested`, `search:suggest:v1:success`

---

- All service actions (e.g., `search`, `suggest`) are handled by a single, generic event handler.
- The handler parses `{action}` and `{state}` from the event type and dispatches to business logic
  via an `actionHandlers` map.
- Per-action handler functions are removed; all logic is implemented in generic handler functions
  and registered in the map.

### Event/Action Naming (Canonical, Search-Referenced)

All event/action names must use the canonical `service:action:version:state` pattern, referencing
the search service standard. Allowed states: `requested`, `started`, `success`, `failed`,
`completed`.

#### Example:

```go
handler.Success(ctx, "admin:user:create:v1:completed", ...)
handler.Error(ctx, "admin:user:create:v1:failed", ...)
handler.Success(ctx, "search:search:v1:completed", ...)
handler.Error(ctx, "search:search:v1:failed", ...)
```

Do not use underscores, dots, or 'metadata' as an action/state. Actions should be verbs, state
should be one of the allowed states above.

Reference: See `internal/service/search/search.go` for canonical event naming and orchestration
patterns.

- Defensive filtering ensures only relevant events are processed by each handler.
- Event bus (Nexus) routes events to subscribers based on explicit event type registration.
- Only relevant handlers receive events, preventing cross-action processing and improving isolation.

---

## 3. Metadata Handling

- All emitted events include a `correlation_id` for end-to-end tracing
- Event payloads include relevant metadata fields such as `campaign_id`, `user_id`, and any
  domain-specific context
- Metadata is propagated through all layers (API, business logic, orchestration, and storage)
- All logs include the full event type/key and correlation ID

---

## 4. Centralized Orchestration via graceful.Handler

- All event emission, error handling, and state transitions use the handler, which emits canonical
  envelopes and logs all actions.
- Legacy event emission methods are removed; only `EmitEventEnvelopeWithLogging` is used.
- Handler is injected into all services and orchestrates all event flows.

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

- Partial search result updates emit events with the same canonical format, e.g.,
  `search:search:v1:success` with a partial payload and a `partial: true` flag in metadata
- Consumers can distinguish between full and partial updates via the event payload
- Partial updates are supported in both event emission and state management logic

---

## 7. Registry-Driven Constants & Validation

- All event, channel, and key constants are generated from the service registration registry.
- Go/TypeScript constants and JSON docs are auto-generated and referenced in code.
- Startup validation ensures all event types used in code are present in the registry.

---

## 8. Event Routing

## 8. Provider-Driven Event Registration & Routing

- Provider registers all canonical event types and handlers at startup, using the registry as the
  source of truth.
- Event bus routes events to subscribers based on explicit event type registration.
- Only relevant handlers receive events, preventing cross-action processing.
- All event types and keys are validated against the registry at build/startup.

---

## 9. Validation & Linting

- At build/startup, all emitted/subscribed event types and Redis keys are validated against the
  registry
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
- See `events/constants.go` and `events/event_types.json` for the list of valid event types and key
  patterns

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

- Both frontend (WASM) and backend use the canonical `service_registration.json` as the source of
  truth for all event/action types.
- The WASM code dynamically loads and logs all valid event types at runtime for verification (see
  `wasm/main.go`, `loadServiceEventTypes`).
- The backend generator parses the same file and outputs:
  - Go constants for use in code (compile-time safety).
  - A JSON file for runtime validation and documentation.
- At backend startup, a validation step should ensure all event types used in code are present in
  the generated registry, preventing drift.
- This guarantees that all event emission, subscription, and key usage are always up-to-date and
  compliant with the canonical registry.

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

This document is the canonical standard for service functionality as of July 2025. All contributors
must read and follow these standards.
