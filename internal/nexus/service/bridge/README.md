# Protocol Bridge for Nexus

## Overview

The Protocol Bridge package enables Nexus to communicate with any external system, device, or
protocol via pluggable adapters. It provides:

- Pluggable adapters (MQTT, AMQP, WebSocket, CoAP, BLE, etc.)
- Dynamic discovery and registration (registry pattern)
- Metadata-driven routing with declarative policies
- Zero-trust security with identity federation and RBAC
- Observability, health checks, and connection pooling

## Core Components

- `Adapter` interface: Contract for all protocol adapters
- `Registry`: Dynamic registration and lookup of adapters
- `Router`: Metadata-driven routing and policy enforcement
- `Instrumentation`: Metrics and tracing wrappers for adapters
- `Security`: Identity, RBAC, and audit hooks
- `AdapterPool`: Connection pooling for scalable adapter use

## Directory Structure

- `adapter.go` — Adapter interface and types
- `registry.go` — Adapter registry
- `router.go` — Metadata-driven router
- `bridge.go` — Bridge service entry point and Nexus integration
- `security.go` — Security, RBAC, and audit hooks
- `instrumentation.go` — Metrics and tracing wrappers
- `adapterpool.go` — Adapter connection pooling
- `adapters/` — Directory for protocol adapter implementations (e.g., mqtt, amqp, websocket)

## Adding a New Adapter

1. Implement the `Adapter` interface in a new subdirectory under `adapters/`.
2. Register the adapter in its `init()` function using `bridge.RegisterAdapter()`.
3. Define metadata requirements and capabilities for routing and discovery.
4. Integrate security and observability hooks as needed.

## Example Use

```
// Register adapters at startup
mqttAdapter := adapters.NewMQTTAdapter(cfg)
bridge.RegisterAdapter(mqttAdapter)

// Initialize bridge service with routing rules
bridge.InitBridgeService(rules)
```

## Extensibility

- Add new protocols by implementing the Adapter interface
- Extend security, metrics, and tracing as needed
- Integrate with the knowledge graph for dynamic discovery

## References

- See architecture documentation for full design and best practices
