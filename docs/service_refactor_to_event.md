# OVASABI Service Refactor to Event-Driven Pattern

## Overview

This document provides a comprehensive refactor plan for migrating all OVASABI services to the
canonical event emission ("emit event") pattern using the Nexus event bus and the robust metadata
standard. It includes a step-by-step checklist for each service and system-wide guidance. This is
the canonical reference for the event-driven refactor process.

---

## Why Refactor to the Emit Pattern?

- **Loose coupling:** Services communicate via events, not direct calls.
- **Extensibility:** New workflows can be added by subscribing to events.
- **Observability:** All actions are auditable and traceable.
- **Real-time UX:** Enables WebSocket and real-time updates.
- **Compliance:** Enforces metadata, versioning, and audit standards.

---

## Refactor Plan: High-Level Steps

1. **Inventory All Service Actions:** List all create, update, delete, and key business actions for
   each service.
2. **Define Event Types:** For each action, define a canonical event type (if not already present in
   `internal/service/nexus/events.go`).
3. **Standardize Payloads:** Ensure each event has a well-defined payload (proto/message struct).
4. **Adopt Metadata Pattern:** All events must include a `common.Metadata` object, with
   service-specific fields as needed.
5. **Emit Events:** Refactor service logic to emit events after key actions.
6. **Subscribe to Events:** Refactor downstream services to subscribe to relevant events instead of
   direct calls.
7. **Update Tests:** Ensure all new event flows are covered by tests.
8. **Document:** Update service and system documentation to reflect the new pattern.

---

## Service Refactor Checklist

For **each service**, follow this checklist:

### 1. Inventory Actions

- [ ] List all actions that should emit events (create, update, delete, business events).

### 2. Event Type Mapping

- [ ] Map each action to a canonical event type (see `internal/service/nexus/events.go`).
- [ ] If missing, add new event type constants.

### 3. Define Payloads

- [ ] Ensure each event has a clear, versioned payload (proto/message struct).
- [ ] Document the payload structure.

### 4. Integrate Metadata

- [ ] All events must include a `common.Metadata` object.
- [ ] Add service-specific fields under `metadata.service_specific.{service}`.
- [ ] Include versioning, audit, and orchestration fields as needed.

### 5. Refactor to Emit

- [ ] After each key action, emit the event to the Nexus event bus:
  ```go
  nexusBus.Emit(ctx, eventType, payload, metadata)
  ```
- [ ] Remove any direct service-to-service calls that can be replaced by event-driven flows.

### 6. Subscribe to Events

- [ ] For any downstream logic, subscribe to the relevant event type(s).
- [ ] Extract payload and metadata in the handler.
- [ ] Take action based on event data and orchestration hints.

### 7. Update Tests

- [ ] Add/modify tests to cover event emission and subscription.
- [ ] Test both successful and failure scenarios.

### 8. Documentation

- [ ] Update service README and API docs to describe new event flows.
- [ ] Document event types, payloads, and metadata fields.

---

## Example: User Service Refactor

| Step           | Example                                                                              |
| -------------- | ------------------------------------------------------------------------------------ |
| **Action**     | User created                                                                         |
| **Event Type** | `user.created`                                                                       |
| **Payload**    | `User` proto message                                                                 |
| **Metadata**   | `metadata.service_specific.user` with versioning, audit, etc.                        |
| **Emit**       | After user creation, call:<br>`nexusBus.Emit(ctx, EventUserCreated, user, metadata)` |
| **Subscribe**  | Notification, analytics, or audit services subscribe to `user.created`               |
| **Test**       | Test that event is emitted and handled correctly                                     |
| **Docs**       | Update README with event flow                                                        |

---

## System-Wide Refactor Steps

1. **Update All Service Protos:** Ensure all entities use `common.Metadata`.
2. **Centralize Event Types:** Use `internal/service/nexus/events.go` for all event type constants.
3. **Adopt Shared Helpers:** Use shared helpers for building event types and metadata.
4. **Update Orchestration Logic:** Move orchestration to event subscribers, not direct calls.
5. **Update Knowledge Graph:** Register all new event types and flows in the knowledge graph.
6. **CI/CD & Linting:** Add checks to enforce event emission and metadata standards.
7. **Documentation:** Update Amadeus context, service docs, and onboarding guides.

---

## Service Refactor Template

**For each service, document:**

- **Actions & Event Types:**  
  | Action | Event Type | Payload | Metadata Fields |
  |--------|------------|---------|----------------| | | | | |

- **Emit Example:**

  ```go
  nexusBus.Emit(ctx, EventType, payload, metadata)
  ```

- **Subscribe Example:**

  ```go
  func handleEvent(event Event) {
      // Extract payload, metadata, take action
  }
  ```

- **Checklist:**
  - [ ] All actions emit events
  - [ ] All events include metadata
  - [ ] All downstream logic subscribes to events
  - [ ] Tests updated
  - [ ] Docs updated

---

## References

- [Amadeus Context: Event Bus & Metadata Standards](docs/amadeus/amadeus_context.md)
- [Canonical Event Types](internal/service/nexus/events.go)
- [Metadata Pattern](docs/services/metadata.md)
- [Versioning Standard](docs/services/versioning.md)

---

**All new and refactored services must follow this pattern. This document is the canonical reference
for the event-driven refactor process in OVASABI.**
