# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

> **Standard:** This service must follow the
> [Unified Communication & Calculation Standard](../amadeus/amadeus_context.md#unified-communication--calculation-standard-grpc-rest-websocket-and-metadata-driven-orchestration).

## 1. Canonical Metadata Fields

- List and document all metadata fields used by this service (see:
  `internal/service/{service}/metadata.go`).

## 2. Calculation/Enrichment Endpoints

- List all gRPC/REST/WebSocket endpoints that accept/return canonical metadata.
- Example:
  ```proto
  rpc CalculateRisk(CalculateRiskRequest) returns (CalculateRiskResponse);
  message CalculateRiskRequest {
    string user_id = 1;
    common.Metadata metadata = 2;
  }
  message CalculateRiskResponse {
    common.Metadata metadata = 1;
  }
  ```

## 3. Communication Patterns

- gRPC: Calculation/enrichment endpoints, metadata chaining
- REST: Composable request/response with metadata
- WebSocket: Real-time metadata updates, UI state sync

## 4. UI State Handling

- How the UI hydrates from metadata (REST/gRPC/WebSocket)
- How real-time updates are handled

## 5. Orchestration & Knowledge Graph

- How this service participates in Nexus orchestration
- How metadata and calculation chains are tracked in the knowledge graph

## 6. Build & Deployment

- **Always use the Makefile and Docker configuration for proto/code generation and builds.**

## 7. Checklist

- [ ] Documents all metadata fields and calculation/enrichment chains
- [ ] Exposes calculation/enrichment endpoints using canonical metadata
- [ ] References the Amadeus context and unified standard
- [ ] Uses Makefile/Docker for all builds and proto generation
- [ ] Documents UI state handling and orchestration participation

---
