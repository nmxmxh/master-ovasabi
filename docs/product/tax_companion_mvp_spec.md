# Tax Companion MVP: Prototype & Data Layer Specification

**Version:** 1.2
**Date:** 2025-11-14
**Target:** Phase 1 (MVP) for Nigeria Tax Act, 2025

> **Changelog v1.2:**
> - Integrated **Dual ID Pattern** for database entities to enhance security.
> - Added a dedicated section on **Redis** for caching, session management, and real-time event queuing.
> - Standardized data models to follow `tax_{entity}` naming convention and incorporate the dual ID pattern.
> - Refined the overall structure for better clarity and alignment with project-wide best practices.

This document outlines the front-end prototype, system architecture, data models, and event structures for the Tax Companion MVP. It is intended for UI/UX designers, backend engineers, and AI agents.

---

## Part 1: Prototype Specification (UI/UX)

*(This section remains high-level, as the core UI/UX flow is unchanged. The implementation details are now driven by the event architecture in Part 2.)*

### 1.1. Major Screens & Modules
1.  **Onboarding:** User registration and profile setup.
2.  **Dashboard:** Real-time summary of the user's tax situation.
3.  **Income & Expense Trackers:** Screens for managing financial entries.
4.  **Smart Tax Calculator:** Interactive tool for estimating tax liability.
5.  **Reports:** Generation and download of financial summaries.
6.  **Tax Education Hub:** A repository of articles and guides.
7.  **E-Filing Preparation:** Module to export data for official tax filing.
8.  **Settings:** Profile management and preferences.

### 1.2. UI Component Library
-   **Forms, Charts, File Upload, Tooltips, PDF Viewer, Tables.**

---

## Part 2: System Architecture & Event Model

The application uses a modern, event-driven architecture designed for real-time updates, low latency, and clear separation of concerns. This makes the system easy to manage for both human developers and AI agents.

### 2.1. Component Overview

-   **Frontend (React/Vite):** A thin presentation layer responsible for rendering the UI based on state provided by the WASM client. It captures user interactions and forwards them as commands.
-   **WASM Client (Go):** Compiled to WebAssembly, this runs in the browser. It manages application state, performs client-side validation, handles complex calculations (like tax estimation), and communicates with the backend via WebSockets.
-   **Backend API (Go):** A gRPC-based server that handles business logic, database operations, and authentication. It exposes a WebSocket endpoint for the WASM client.
-   **Database (PostgreSQL):** The persistent data store.
-   **Cache & Message Queue (Redis):** Handles caching, session storage, and real-time event distribution.

### 2.2. Communication Flow

Communication is primarily handled over a single, persistent WebSocket connection between the WASM client and the Backend API.

1.  **UI Interaction:** The user clicks a button in the React UI (e.g., "Add Income").
2.  **Event Dispatch:** The UI dispatches a command to the WASM client.
3.  **WASM Processing:** The WASM client validates the input and constructs a standardized event.
4.  **WebSocket Communication:** The event is sent to the backend over the WebSocket connection.
5.  **Backend Processing:** The backend receives the event, performs the necessary business logic (e.g., saves to the database), and publishes one or more events to a Redis channel (e.g., `events:user-123`).
6.  **Real-time Push:** The backend, listening to the Redis channel, pushes the relevant events back to the appropriate client(s) over their WebSocket connection (e.g., `INCOME_ENTRY_CREATED`, `DASHBOARD_UPDATED`).
7.  **State Update:** The WASM client receives the events from the backend, updates its internal state, and the React UI automatically re-renders to reflect the new state.

### 2.3. Event Structure

All communication follows a standard event envelope to ensure consistency.

#### Event Envelope (JSON over WebSocket)
```json
{
  "event_type": "CREATE_INCOME_ENTRY",
  "payload": {
    // Base64-encoded Protobuf message for the specific event
    "data": "base64_string_of_proto_payload"
  },
  "metadata": {
    // Base64-encoded common.Metadata Protobuf message
    "data": "base64_string_of_proto_metadata"
  },
  "timestamp": "2025-11-14T10:00:00Z"
}
```

---

## Part 3: Data Layer & Database Schema

The schema follows the project's naming conventions (`tax_{entity}`) and incorporates the **Dual ID Pattern** for security and the standardized `metadata` column for advanced analytics.

### 3.1. Dual ID Pattern

As per `database_practices.md`, all core entities will use two identifiers:
-   **`id` (BIGSERIAL/INT):** An internal, auto-incrementing primary key for relational integrity (foreign keys).
-   **`public_id` (UUID):** An external, non-sequential identifier (UUID v4) exposed in APIs and URLs. This prevents resource enumeration and enhances security.

A "master table" is not required for this service, as the relationships between `users`, `profiles`, and their financial entries are direct.

### 3.2. Metadata Integration for Advanced Use Cases

The generic `JSONB` column for `metadata` in our tables is standardized to store the JSON representation of the `common.Metadata` message from `api/protos/common/v1/metadata.proto`. This provides a consistent, machine-readable structure for all metadata.

**Key Metadata Fields and Their Purpose:**

*   **`global_context`**: Contains essential session information like `user_id`, `session_id`, and `source` (e.g., "onboarding-flow"), crucial for audit trails and user behavior analysis.
*   **`tags`**: Used for categorization and filtering (e.g., `["individual", "resident", "PAYE"]`).
*   **`ai_confidence`**: A score from 0.0 to 1.0 indicating the confidence of an AI agent in a data entry or classification.
*   **`embedding_id`**: A reference to a vector embedding, allowing for semantic search on financial transactions.
*   **`categories`**: ML-inferred categories for transactions (e.g., `["transport", "rideshare"]`).
*   **`knowledge_graph`**: Links the data entity to a node in a larger knowledge graph for complex relationship analysis.

### 3.3. Core Data Models

**`tax_users`**
-   `id` (BIGSERIAL, PK)
-   `public_id` (UUID, Unique, Indexed)
-   `email` (VARCHAR, Unique)
-   `password_hash` (VARCHAR)
-   `full_name` (VARCHAR)
-   `role` (VARCHAR, e.g., 'individual', 'business', 'consultant')
-   `metadata` (JSONB, stores `common.Metadata`)

**`tax_profiles`**
-   `id` (BIGSERIAL, PK)
-   `public_id` (UUID, Unique, Indexed)
-   `user_id` (BIGINT, FK to `tax_users.id`)
-   `profile_type` (VARCHAR, 'individual' or 'business')
-   `residency_status_id` (BIGINT, FK to `tax_rule_residency_status.id`)
-   `metadata` (JSONB, stores `common.Metadata`)

**`tax_income_entries`**
-   `id` (BIGSERIAL, PK)
-   `public_id` (UUID, Unique, Indexed)
-   `profile_id` (BIGINT, FK to `tax_profiles.id`)
-   `amount` (NUMERIC)
-   `description` (TEXT)
-   `transaction_date` (TIMESTAMPTZ)
-   `metadata` (JSONB, stores `common.Metadata`)

**`tax_expense_entries`**
-   `id` (BIGSERIAL, PK)
-   `public_id` (UUID, Unique, Indexed)
-   `profile_id` (BIGINT, FK to `tax_profiles.id`)
-   `amount` (NUMERIC)
-   `description` (TEXT)
-   `transaction_date` (TIMESTAMPTZ)
-   `metadata` (JSONB, stores `common.Metadata`)

*(Other tables like `tax_rule_versions`, `tax_brackets`, etc., will also adopt the dual ID pattern where appropriate.)*

---

## Part 4: Data Flows & Integrations (Event-Driven)

The data flows are now explicitly mediated by Redis to decouple services and enhance real-time capabilities.

### 4.1. Core Data Flow (Create Income Entry)

1.  **Client -> Server:** The WASM client sends a `CREATE_INCOME_ENTRY` event over WebSocket.
2.  **Server Logic:** The backend validates the event and inserts a new record into `tax_income_entries`.
3.  **Publish to Redis:** The server publishes `INCOME_ENTRY_CREATED` and `DASHBOARD_METRICS_UPDATED` events to a user-specific Redis channel (e.g., `events:user_public_id_xyz`).
4.  **Push to Client:** A separate part of the backend service, subscribed to the Redis channel, receives the events and pushes them to the user's active WebSocket connection.
5.  **Client Logic:** The WASM client processes these events, updates its state, and the UI re-renders reactively.

---

## Part 5: Development & Tooling

This project is optimized for rapid, consistent development by both humans and AI agents through clear tooling and structure.

### 5.1. File Structure
```
/
├── api/protos/              # Protobuf definitions (gRPC, Events)
├── cmd/server/              # Backend Go application main package
├── frontend/                # React/Vite frontend application
├── internal/                # Backend Go service logic
├── wasm/                      # Go source for the WASM client
├── Makefile                 # Build automation script
├── Dockerfile               # Backend service container definition
└── docker-compose.yml       # Local development environment
```

### 5.2. Makefile
A `Makefile` simplifies common development tasks.
```makefile
# Makefile

.PHONY: all build-server build-wasm run test proto

# Generate Protobuf code
proto:
	@echo "Generating Protobuf code..."
	@protoc --go_out=. --go-grpc_out=. api/protos/common/v1/*.proto
	@protoc --go_out=. --go-grpc_out=. api/protos/tax/v1/*.proto

# Build all components
all: proto build-server build-wasm

# Build the backend Go server
build-server:
	@echo "Building backend server..."
	@go build -o bin/server ./cmd/server

# Build the WASM client
build-wasm:
	@echo "Building WASM client..."
	@GOOS=js GOARCH=wasm go build -o frontend/public/main.wasm ./wasm

# Run the backend server (requires Docker for DB and Redis)
run:
	@docker-compose up -d db redis
	@go run ./cmd/server

# Run tests
test:
	@go test ./...
```

---

## Part 6: Security, Analytics & Assumptions

*(This section remains largely the same but is now implemented within the new event-driven architecture.)*

### 6.1. Security & Compliance
-   **Authentication:** The WebSocket connection is authenticated on establishment using a JWT.
-   **Authorization:** The `GlobalContext` in each event's metadata is used to enforce user-specific data access rules.
-   **Audit Logs:** A dedicated logging service subscribes to the Redis event stream and records all actions to `tax_audit_logs`.

### 6.2. Analytics & KPIs
-   Dashboard KPIs are computed in real-time on the backend and pushed to the client via events.
-   The standardized `metadata` field allows for deep, cross-functional analytics.

### 6.3. Tax Rule Assumptions
-   The assumptions regarding tax brackets and reliefs remain unchanged and must be verified.

---

## Part 7: Caching, Queues & Real-time Layer (Redis)

In response to the query "will there be need for redis?", the answer is **yes**. Redis is a critical component of this architecture, serving multiple purposes as defined in `redis_practices.md`.

### 7.1. Use Cases for Redis

1.  **Real-time Event Bus (Pub/Sub):** As described in the data flow, Redis Pub/Sub is the backbone of our real-time architecture. It decouples the core business logic from the client-facing WebSocket service, allowing for better scalability and resilience.
    -   **Key Pattern:** `events:{user_public_id}`

2.  **Caching:** To reduce database load and improve response times, Redis will be used to cache frequently accessed, slowly changing data.
    -   **Tax Rules:** Tax brackets, reliefs, and residency status rules.
        -   **Key:** `cache:tax_rules:v1`
        -   **TTL:** 24 hours (or until a rule update event is published).
    -   **User Profiles:** Caching `tax_profiles` data.
        -   **Key:** `cache:user:{user_public_id}:profile`
        -   **TTL:** 1 hour (invalidated on profile update).

3.  **Session Management:** Storing short-lived user session data and authentication tokens.
    -   **Key:** `session:auth:{session_token}`
    -   **TTL:** 24 hours.

4.  **Distributed Locks:** To prevent race conditions in critical operations (e.g., processing a large file import).
    -   **Key:** `lock:import:{user_public_id}`
    -   **TTL:** 30-60 seconds.

### 7.2. Implementation Notes
-   All Redis keys must follow the naming convention `namespace:context:entity:id`.
-   The application must be designed to handle Redis unavailability gracefully (e.g., by fetching directly from the database if a cache miss occurs and Redis is down).
-   The Redis client will be managed via dependency injection throughout the backend service.


