# Graceful Orchestration & DI Container Pattern (Master by Ovasabi)

Version: 2025-06-01

---

## Overview

This document defines the canonical, platform-wide standards for:

- Dependency Injection (DI) container registration and resolution
- Graceful orchestration of all success and error flows
- Extensible orchestration hooks for caching, eventing, knowledge graph, scheduling, and more
- Handler-layer orchestration and error handling
- Best practices, onboarding, and compliance checklist

These patterns are required for all new and refactored services in Master by Ovasabi. They ensure modularity, testability, extensibility, and robust, DRY orchestration across the platform.

---

## 1. DI Container: Registration & Resolution

- **All services** (User, Notification, Referral, etc.) are registered in the DI container at startup using their canonical `Register` functions.
- **All handler layers** (REST, gRPC, WebSocket) resolve services from the DI container, never constructing them directly.
- **Supporting objects** (EventEmitter, Redis Cache, etc.) are also registered for orchestration.

**Example:**

```go
// Registering a service
if err := container.Register((*userpb.UserServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
    return userService, nil
}); err != nil {
    log.Error("Failed to register user service", zap.Error(err))
}

// Resolving in a handler
var userService userpb.UserServiceServer
if err := container.Resolve(&userService); err != nil {
    log.Error("Failed to resolve UserService", zap.Error(err))
    // Handle error gracefully
}
```

---

## 2. Graceful Orchestration: Success & Error Flows

- **All service methods** use the `graceful` package for error and success orchestration.
- **StandardOrchestrate** is called for both success and error, with hooks for:
  - Caching
  - Event emission
  - Knowledge graph enrichment
  - Scheduler registration
  - Custom hooks (for extensibility)

**Example:**

```go
// On success
success := graceful.WrapSuccess(ctx, codes.OK, "user created", resp, nil)
success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
    Log: log,
    Cache: cache,
    CacheKey: userID,
    CacheValue: resp,
    CacheTTL: 10 * time.Minute,
    Metadata: resp.Metadata,
    EventEmitter: eventEmitter,
    EventEnabled: true,
    EventType: "user_created",
    EventID: userID,
    PatternType: "user",
    PatternID: userID,
    PatternMeta: resp.Metadata,
    KnowledgeGraphHook: func(ctx context.Context) error {
        return pattern.EnrichKnowledgeGraph(ctx, log, "user", userID, resp.Metadata)
    },
    SchedulerHook: func(ctx context.Context) error {
        return pattern.RegisterSchedule(ctx, log, "user", userID, resp.Metadata)
    },
})

// On error
errResp := graceful.WrapErr(ctx, codes.Internal, "failed to create user", err)
errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
    Log: log,
    // Add hooks as needed
})
```

---

## 3. Pattern & Metadata Integration

- **All orchestration helpers** (e.g., `EnrichKnowledgeGraph`, `RegisterSchedule`) are called via hooks in the graceful orchestration config, not manually in service methods.
- **All metadata is passed through the orchestration pipeline** and used for caching, KG enrichment, scheduling, and event emission.

**Best Practice:**

- Never call orchestration helpers directly in service methods—always use hooks in the orchestration config.
- Normalize and validate metadata before orchestration.

---

## 4. Handler Layer: Error Handling & Orchestration

- **All HTTP/gRPC handlers** use graceful error handling and orchestration.
- **All errors are logged with context** and orchestrated for audit, alerting, and fallback.

**Example:**

```go
if err := container.Resolve(&userService); err != nil {
    log.Error("Failed to resolve UserService", zap.Error(err))
    errResp := graceful.WrapErr(ctx, codes.Internal, "internal error", err)
    errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
    return
}
```

---

## 5. Extensibility & Future-Proofing

- **All orchestration configs** support custom hooks for future extensibility.
- **All new services** must follow the same registration, orchestration, and error handling patterns.

**Checklist:**

- [ ] All services registered in DI container
- [ ] All handler/service dependencies resolved from DI
- [ ] All success/error flows use graceful orchestration
- [ ] All orchestration helpers called via hooks, not manually
- [ ] All metadata normalized and passed through orchestration
- [ ] All errors logged and orchestrated
- [ ] Health and metrics endpoints exposed
- [ ] Documentation and onboarding up-to-date

---

## 6. References & Onboarding

- See [Amadeus Context](amadeus_context.md) for canonical patterns and standards
- See [General Metadata Documentation](../services/metadata.md) for metadata patterns
- See [Service Onboarding Guides](../services/) for service-specific examples

---

**This document is a living reference. Update it as new orchestration patterns, hooks, or best practices are added to the platform.** 