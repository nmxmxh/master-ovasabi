# Inos – The Internet Native Operating System

> **🚧 Work in Progress (WIP):**  
> INOS is a fast-evolving, open platform for distributed, AI-powered, and WASM-enabled systems. We welcome contributors of all backgrounds—Go developers, AI/ML engineers, database and distributed systems specialists, QA/testers, frontend and WASM enthusiasts, and anyone passionate about building resilient, extensible digital infrastructure. See the Contributing section below to get involved!

Welcome to **Inos**—a self-documenting, AI-ready, and community-driven platform for orchestrating digital services, relationships, and value.

---

## What is Inos?

**Inos** (Internet Native Operating System) is a modular, event-driven platform for orchestrating digital services, relationships, and value across the internet. It provides a unified substrate for identity, data, and interface interoperability—bridging backend, real-time, and frontend layers.

---

## Architecture Overview

INOS is built on a layered, modern architecture:

```text
Go Services (Domain Logic, gRPC APIs)
        │
        ▼
gRPC Service Mesh (Internal APIs, Service-to-Service)
        │
        ▼
Event Bus (Redis + Custom) & WebSockets (Real-Time)
        │
        ▼
Multithreaded WASM (AI/ML, Compute, Browser/Edge)
        │
        ▼
Frontend (React, TypeScript, SPA, PWA)
```

### **Key Components**

- **Go Services:** Modular, domain-driven microservices (user, campaign, content, analytics, etc.) expose gRPC APIs and business logic.
- **gRPC Mesh:** High-performance, strongly-typed APIs for internal and external service communication.
- **Event Bus & WebSockets:** Real-time orchestration, pub/sub, and campaign/user-specific messaging. Enables live updates, notifications, and cross-service workflows.
- **Multithreaded WASM:** High-performance AI/ML and compute modules running in the browser or edge, interoperating with backend services.
- **Frontend:** Modern, reactive UI (React/TypeScript) consuming real-time data and WASM-powered features.

---

## Metadata: The System’s DNA

- **Universal Ledger:** Metadata is the core currency and audit trail of INOS, tracking every operation, relationship, and value flow—across users, services, content, and tasks.
- **Extensibility:** All services, events, and entities are described and orchestrated via extensible metadata, enabling dynamic evolution without breaking existing data.
- **Lineage & Provenance:** Every entity and action is traceable, supporting audit, compliance, and digital legacy.
- **System Currency:** The sum of all scores and values across the system forms a living, auditable measure of reputation, contribution, and impact.

---

## Graceful Orchestration: Error & Success as First-Class Citizens

- **Centralized Handling:** All errors and successes are orchestrated via the `graceful` package, ensuring that every outcome—positive or negative—is logged, auditable, and can trigger further automation.
- **Extensible Hooks:** Custom hooks and overrides allow for service-specific or advanced orchestration, supporting resilience and compliance.
- **Symmetry:** Both error and success flows are handled with equal care, enabling robust, predictable, and transparent system behavior.

---

## Features

- Metadata-centric, self-documenting architecture
- Extensible connectors for people, services, compliance, and more
- Digital will pattern for legacy and allocation
- Accessibility and compliance built-in
- Automation with intention, transparency, and resilience
- Real-time, event-driven orchestration (Nexus, Redis, PostgreSQL)
- Tiered, programmable taxation and UBI encoded in metadata
- Graceful, symmetrical error and success orchestration
- Modular adapters and bridge layer for protocol extensibility
- Open source and commercial support options

---

## Directory Structure (Work in Progress)

```go
.
├── api/                # Protobuf definitions for all services
├── internal/           # Service implementations, handlers, business logic
│   ├── blueprints/     # Service blueprints and patterns
│   ├── bootstrap/      # System bootstrap logic
│   ├── config/         # Configuration management
│   ├── health/         # Health checks and metrics
│   ├── metrics/        # Metrics collection
│   ├── nexus/          # Orchestration, event bus, bridge, adapters
│   │   ├── service/
│   │   │   ├── pattern/    # Orchestration patterns
│   │   │   ├── bridge/     # Protocol bridge, adapters, registry
│   │   │   └── adapters/   # Protocol adapters (MQTT, WebSocket, etc.)
│   ├── repository/     # Data access and caching
│   ├── server/         # API, WebSocket, REST, gRPC handlers
│   ├── service/        # All core and extension services
│   └── shared/         # Shared utilities and interfaces
├── pkg/                # Shared packages (graceful, utils, logger, etc.)
├── tax/                # Digital will, allocation, and taxation logic
├── docs/               # Documentation, manifesto, advice, explorer, patterns
│   └── amadeus/
│       ├── manifesto.md
│       ├── project_preface.md
│       ├── amadeus_context.md
│       └── explorer.md
├── README.md           # This file
├── CONTRIBUTING.md     # How to contribute
├── CODE_OF_CONDUCT.md  # Community guidelines
├── LICENSE             # Open source license
```

---

## Services List (Work in Progress)

- **User**: Identity, authentication, RBAC, audit
- **Notification**: Multi-channel, templates, real-time
- **Campaign**: Campaign management, analytics
- **Referral**: Referral, rewards, fraud detection
- **Security**: Policies, audit, compliance
- **Content**: Articles, micro-posts, video, comments, reactions
- **Commerce**: Orders, payments, billing
- **Localization**: i18n, translation, compliance
- **Search**: Full-text, fuzzy, entity search
- **Admin**: Admin user management, roles, audit
- **Analytics**: Event, usage, reporting
- **ContentModeration**: Moderation, compliance
- **Talent**: Talent profiles, bookings
- **Nexus**: Orchestration, event bus, bridge, adapters
- **Adapters/Bridge**: MQTT, WebSocket, AMQP, HTTP, and more
- **Scheduler**: Time-based orchestration (WIP)

---

## Database & Redis Practices

- **PostgreSQL:**  
  - Uses `jsonb` columns for rich, extensible metadata  
  - Full-text search, GIN/partial indexes for performance  
  - Partitioning, archiving, and audit trails for scalability
- **Redis:**  
  - Hot data caching for low-latency access  
  - Pub/sub for real-time events and notifications  
  - Rate limiting, counters, and ephemeral state

---

## Handlers, Nexus, and Orchestration Patterns

- **Handlers:** REST, gRPC, and WebSocket handlers translate external requests into canonical metadata-driven actions.
- **Nexus:** The event bus and orchestration layer connect all services, adapters, and real-time flows, enabling dynamic, metadata-driven automation.
- **Bridge/Adapters:** Protocol adapters (MQTT, WebSocket, etc.) enable integration with any external system or device.

---

## Getting Started

1. Clone the repo
2. Install Go and dependencies (`go mod tidy`)
3. See `tax/ovasabi_default.go` for the digital will pattern
4. Explore the docs and services

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. All are welcome—code, docs, ideas, and feedback!

---

## License

INOS is dual-licensed:

- **MIT License:** Free and open source for community use, contributions, and research. See [LICENSE](LICENSE).
- **Enterprise License (AGPL/BUSL):** For advanced features, enterprise support, and legal guarantees. See [LICENSE](LICENSE).

**Why this license?**  
We believe in open innovation and community-driven development, while also supporting enterprise needs for advanced features, support, and compliance.  
This dual-licensing model ensures INOS remains open and accessible, while enabling sustainable growth and stewardship.

---

## Why INOS is the Internet Native Operating System

- The internet lacks a shared user context — INOS fixes this.
- Users own their profiles and move seamlessly across apps.
- Developers build interoperable frontends and shared backends.
- INOS provides standards for memory, interfaces, and control in network-native environments.
- Metadata is the backbone: every action, relationship, and value flow is tracked, auditable, and programmable.
- Graceful orchestration ensures robust, transparent, and resilient system behavior for both errors and successes.

**INOS is the OS for the programmable, AI-native, internet-scale future.**

---

## Explore More

- [Manifesto](docs/amadeus/manifesto.md): Philosophy, values, and intent
- [Project Preface](docs/amadeus/project_preface.md): Roadmap, acknowledgments, learning philosophy
- [Amadeus Context](docs/amadeus/amadeus_context.md): System architecture, metadata, orchestration
- [Metadata Standard](docs/services/metadata.md): Canonical metadata pattern
- [Explorer](docs/amadeus/explorer.md): System Currency Explorer (WIP)

---

**Thank you for being part of this journey. Let’s keep building—together.**
