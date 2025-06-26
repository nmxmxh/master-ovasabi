# Handler-Layer Orchestration Pattern

## Overview

This document describes the transition from service-layer to handler-layer orchestration in the
OVASABI backend. It explains the rationale, benefits, and best practices for this pattern, and
serves as a reference for all contributors.

---

## Background

**Previous Pattern:**

- Orchestration logic (caching, event emission, metadata enrichment, etc.) was implemented inside
  service methods.
- Service code was responsible for both business logic and infrastructure side effects.

**New Pattern:**

- All orchestration is now performed at the handler (API boundary) layer.
- Service methods are pure business logic: they process input and return results/errors.
- The handler is responsible for orchestrating all post-processing using the `graceful` package.

---

## Rationale for the Change

- **Separation of Concerns:**  
  Business logic is decoupled from infrastructure and orchestration.
- **Testability:**  
  Service methods are easier to test in isolation.
- **Extensibility:**  
  Orchestration logic can be easily modified, extended, or replaced at the API boundary.
- **Observability:**  
  All orchestration flows are explicit and visible at the handler level.
- **Performance:**  
  Reduces redundant data processing and event emission by centralizing orchestration.

---

## Pattern Details

### Service Layer

- Contains only business logic.
- No direct calls to cache, event bus, or orchestration helpers.
- Returns results or errors.

### Handler Layer

- Calls the service method.
- Passes all context, cache, event emitter, and metadata to the orchestration layer (`graceful`).
- Handles all post-processing: caching, event emission, knowledge graph enrichment, audit, etc.

### Orchestration Layer (`graceful`)

- Receives all dependencies and context from the handler.
- Executes orchestration steps in a standardized, symmetrical (success/error) manner.
- Ensures all events and side effects are metadata-rich and extensible.

---

## Benefits

- **Cleaner, more maintainable codebase**
- **Explicit orchestration flows**
- **Reduced data bloat and redundant processing**
- **Easier onboarding for new engineers**
- **Future-proof for event-driven and microservice architectures**

---

## Best Practices

- **Keep business logic and orchestration separate.**
- **Always perform orchestration at the handler/API boundary.**
- **Pass all required dependencies (cache, event emitter, etc.) to the orchestration config.**
- **Document orchestration flows in handler and API docs.**
- **Update onboarding guides to reflect this pattern.**

---

## Migration Checklist

- [ ] Remove orchestration logic from service methods.
- [ ] Refactor service methods to return only results/errors.
- [ ] Move all orchestration (cache, events, etc.) to the handler layer.
- [ ] Pass dependencies to `graceful` orchestration configs in handlers.
- [ ] Update documentation and onboarding materials.

---

## Example (Before & After)

**Before (Service-Layer Orchestration):**

```go
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
    // ... business logic ...
    s.cache.Set(ctx, ...)
    s.eventBus.Emit(ctx, ...)
    return resp, nil
}
```

**After (Handler-Layer Orchestration):**

```go
func UserHandler(container *di.Container) http.HandlerFunc {
    // ... parse request ...
    resp, err := userService.CreateUser(ctx, req)
    if err != nil {
        graceful.WrapErr(...).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{...})
        return
    }
    graceful.WrapSuccess(...).StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{...})
}
```

---

## References

- [Amadeus Context Documentation](../amadeus/amadeus_context.md)
- [Service Implementation Pattern](../services/implementation_pattern.md)
- [Graceful Orchestration Standard](../amadeus/amadeus_context.md)
- [Onboarding Guide](onboarding.md)

---

## Summary Table

| Aspect          | Service-Layer Orchestration | Handler-Layer Orchestration (Current) |
| --------------- | --------------------------- | ------------------------------------- |
| Business Logic  | Mixed with orchestration    | Pure, reusable                        |
| Orchestration   | Hidden, scattered           | Explicit, at boundary                 |
| Testability     | Harder                      | Easier                                |
| Data Processing | Can be redundant            | Targeted, efficient                   |
| Observability   | Harder                      | Clear, at API layer                   |
| Extensibility   | Rigid                       | Flexible                              |

---

## Changelog

- **2024-06-XX:** Adopted handler-layer orchestration as the platform standard.
- **2024-06-XX:** Updated all service and handler documentation to reflect the new pattern.

---

**All new and refactored services must follow this pattern. For questions, contact the platform
architecture team.**
