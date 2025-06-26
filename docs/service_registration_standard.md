# OVASABI Platform Service Registration & Inter-Service Communication Standard

This document defines the canonical pattern for service registration and inter-service communication
in the OVASABI platform. All new and refactored services must follow this standard.

---

## Singleton gRPC Client Pattern

- Only one gRPC connection per external service is created and reused (singleton).
- The connection and client are stored in the Provider struct and registered in the DI container.
- All inter-service calls must use the DI-resolved client, never instantiate directly.
- If a client cannot be resolved, log an error and degrade gracefully (do not crash the system).

---

## Concurrency & Scaling Pattern (for High-Intensity Services)

For services with high concurrency or throughput requirements (e.g., Nexus, Analytics, Search):

- **gRPC Client Pool:**

  - The Provider maintains a pool (slice or channel) of gRPC client connections for the service.
  - On Provider initialization, N connections/clients are created and appended to the pool.
  - Requests are distributed across the pool using round-robin, random, or worker pool patterns.
  - The DI container can register a pool resolver, or the Provider exposes a helper to get a client
    from the pool.

- **Pattern Example:**

  ```go
  // In Provider struct:
  nexusClientPool []nexuspb.NexusServiceClient
  // In NewProvider: for i := 0; i < N; i++ { ... append to pool ... }
  // In helper: select a client from the pool (e.g., round-robin)
  func (p *Provider) NextNexusClient() nexuspb.NexusServiceClient { ... }
  ```

- **Dynamic Scaling:**

  - The pool size can be made configurable (via env/config).
  - For bursty workloads, the Provider can expand/shrink the pool at runtime.
  - Nexus or other orchestrators can trigger pool expansion based on load/metrics.

- **Orchestration Integration:**
  - Nexus can use the Provider's pool helpers to parallelize orchestration flows.
  - For high-throughput chains, use goroutine worker pools to fan out requests across the client
    pool.
  - All orchestration and automation flows should use the DI-resolved client or pool helper.

---

## Dependency Injection (DI) Registration

- All services and clients must be registered in the DI container.
- Use the same pattern for all gRPC clients and service servers.
- Always resolve dependencies via the DI container in service constructors.

---

## Graceful Error Handling

- Never call `log.Fatal` or panic on service registration or resolution failure.
- Always log errors and allow the system to continue running, limiting only the features that depend
  on the missing service.

---

## Provider Helper Methods

- For every gRPC client, provide a helper method on the provider for easy access and to encourage
  consistent usage.

---

## Template for New gRPC Client Registration

```go
// In Provider struct:
notificationConn *grpc.ClientConn
notificationClient notificationpb.NotificationServiceClient

// In NewProvider:
conn, err := grpc.Dial("notification:8080", grpc.WithInsecure())
if err != nil {
    log.Error("Failed to connect to NotificationService", zap.Error(err))
} else {
    p.notificationConn = conn
    p.notificationClient = notificationpb.NewNotificationServiceClient(conn)
}

// In registerServices:
if err := p.container.Register((*notificationpb.NotificationServiceClient)(nil), func(_ *di.Container) (interface{}, error) {
    if p.notificationClient != nil {
        return p.notificationClient, nil
    }
    return nil, fmt.Errorf("NotificationServiceClient unavailable: gRPC connection not established")
}); err != nil {
    p.log.Error("Failed to register NotificationServiceClient", zap.Error(err))
    return err
}

// In Provider helper:
func (p *Provider) NotificationClient() notificationpb.NotificationServiceClient {
    var client notificationpb.NotificationServiceClient
    if err := p.container.Resolve(&client); err != nil {
        p.log.Error("Failed to resolve notification client", zap.Error(err))
        return nil
    }
    return client
}

// In Close():
if p.notificationConn != nil {
    if err := p.notificationConn.Close(); err != nil {
        p.log.Error("Failed to close notification gRPC connection", zap.Error(err))
    }
}
```

---

## Nexus and Orchestration

- Nexus requires access to all service clients for orchestration.
- The Provider registers all service clients (including pools for high-intensity services) and
  exposes helpers for orchestration flows.
- Nexus can use these helpers to chain, parallelize, or fan out requests as needed.
- For new orchestration patterns, always use the DI-resolved client or pool helper.

---

## Best Practices

- Register all required factories before any dependent services.
- Never call `log.Fatal` or panic on registration/resolution failure; log and degrade gracefully.
- Document this pattern in all new service/provider files.

---

**This document is the authoritative reference for all service registration and inter-service
communication in the OVASABI platform.**

---

## References

- See `internal/service/provider.go` for the canonical implementation.
- See `docs/amadeus/amadeus_context.md` for platform-wide standards and integration points.
- See `internal/nexus/service/pattern_store.go` for orchestration patterns.
