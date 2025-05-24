# OVASABI Platform: Provider Refactor & Event-Driven Orchestration

## Overview

The OVASABI backend now adopts a **metadata-driven, event-oriented architecture**. The main service
provider and all core services are refactored to:

- Use canonical provider/DI patterns for construction and registration.
- Leverage **Nexus as a central event bus** for emitting and consuming events.
- Rely on the **shared metadata pattern** for all event payloads and service communication.
- Enable orchestration, automation, and extensibility across all services.

---

## 1. Provider Refactor: What Changed

### a. Canonical Provider Usage

- All services are now constructed using their canonical provider functions (e.g.,
  `user.NewUserServiceProvider`, `content.NewContentServiceProvider`).
- Dependencies (including Nexus, Redis, other services) are injected explicitly.

### b. Nexus as Event Bus

- The Nexus service is constructed first and injected into all other services that need to emit or
  consume events.
- The provider struct holds references to both the Nexus server and client(s).

### c. Service Registration

- All services (servers and clients) are registered in the DI container.
- Nexus is registered as both a gRPC server and client, and is available for orchestration/eventing.

---

## 2. New Features to Implement

### a. Event Emission

- All services should emit events to Nexus after key actions (create, update, delete, etc.).
- Events must include a `common.Metadata` payload, following the shared metadata pattern.

### b. Event Consumption

- Services can subscribe to and handle events from Nexus.
- This enables cross-service workflows, automation, and dynamic orchestration.

### c. Metadata-Driven Communication

- All event payloads and service-to-service messages use the `common.Metadata` struct.
- Service-specific extensions go under `metadata.service_specific.{service}`.
- Versioning, audit, and compliance fields are always included.

### d. Orchestration Patterns

- Nexus can chain, enrich, or transform events, enabling complex workflows.
- New orchestration patterns can be added without changing existing services.

---

## 3. Required Changes in Services

### a. Emit Events

- After important actions, call Nexus to emit an event:
  ```go
  event := &nexuspb.Event{
      Type: "user.created",
      EntityId: userID,
      Metadata: user.Metadata, // must use shared metadata pattern
      Timestamp: timestamppb.Now(),
  }
  if err := s.Nexus.EmitEvent(ctx, event); err != nil {
      s.log.Warn("failed to emit event to Nexus", zap.Error(err))
  }
  ```
- Ensure all emitted events use the `common.Metadata` struct.

### b. Consume Events

- Implement handlers or subscriptions for relevant event types.
- Use the metadata in the event to drive business logic, orchestration, or side effects.

### c. Constructor Signature

- Update service constructors to accept Nexus as a dependency if they emit/consume events:
  ```go
  func NewUserServiceProvider(log *zap.Logger, db *sql.DB, redisProvider *redis.Provider, notificationClient notificationpb.NotificationServiceClient, nexus nexuspb.NexusServiceServer) userpb.UserServiceServer
  ```

### d. Metadata Compliance

- All service logic, event payloads, and database writes must use the shared metadata pattern.
- Service-specific fields must be namespaced under `metadata.service_specific.{service}`.

---

## 4. Benefits

- **Loose Coupling:** Services interact via events, not direct calls.
- **Extensibility:** New features/services can be added by subscribing to events.
- **Observability:** All actions are tracked centrally with rich metadata.
- **Orchestration:** Nexus enables dynamic, metadata-driven workflows and automation.

---

## 5. Migration/Onboarding Checklist

- [ ] Refactor all service providers to use canonical constructors and inject Nexus.
- [ ] Update all service logic to emit events to Nexus after key actions.
- [ ] Implement event handlers/subscriptions as needed.
- [ ] Ensure all event payloads use the shared metadata pattern.
- [ ] Document new event types and orchestration patterns in the knowledge graph and onboarding
      docs.

---

## 6. References

- [docs/amadeus/amadeus_context.md](docs/amadeus/amadeus_context.md) (Provider/DI, metadata, event
  bus, orchestration)
- [internal/service/nexus/](internal/service/nexus/) (Nexus service/event bus)
- [internal/service/pattern/metadata_pattern.go](internal/service/pattern/metadata_pattern.go)
  (metadata helpers)

---

**Summary:**  
The provider and all services are now event-driven, metadata-compliant, and orchestrated via Nexus.
This enables robust, extensible, and observable microservice workflows across the OVASABI platform.

---

**For onboarding, always reference this documentation and the Amadeus context.**
