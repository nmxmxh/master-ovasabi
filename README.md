# OVASABI Platform (Work in Progress)

A metadata-driven, extensible, and fair digital system for orchestrating services, relationships,
and value—built for the community, by the community.

> **Work in Progress:** OVASABI is a living project, continuously evolving with new paradigms,
> features, and best practices. See the [manifesto](docs/amadeus/manifesto.md) and
> [project preface](docs/amadeus/project_preface.md) for philosophy, roadmap, and intent.

## Philosophy

See [docs/amadeus/manifesto.md](docs/amadeus/manifesto.md) for the full manifesto and advice. In
short: simplicity, extensibility, fairness, and digital legacy are at the heart of everything we do.

## Latest Standards & Innovations

- **Metadata as System Currency:** Metadata is the universal ledger and currency of OVASABI,
  tracking value, reputation, and contributions across users, services, content, and tasks.
- **System-Wide Timezone Awareness:** All events, transactions, and metadata updates are timestamped
  and normalized for global consistency (TimezoneZ).
- **System Currency Explorer:** A tool (UI/API) for visualizing and analyzing the total value,
  contributions, and flows within the ecosystem.
- **User, Service, and Task Scores:** Every entity can have its own score, history, and value,
  contributing to the living system currency.
- **Dual Licensing:** OVASABI is available under the MIT License for open source use and a
  commercial license for enterprise features, support, and additional guarantees.
- **Canonical/Hosted Platform:** The hosted version of OVASABI is the de facto source of truth for
  standards, updates, and governance.

## Features

- Metadata-centric, self-documenting architecture
- Extensible connectors for people, services, compliance, and more
- Digital will pattern for legacy and allocation
- Accessibility and compliance built-in
- Automation with intention, transparency, and resilience
- Real-time, event-driven orchestration (Nexus, Redis, PostgreSQL)
- System-wide timezone and temporal intelligence (TimezoneZ)
- System Currency Explorer for value analytics and governance
- Tiered, programmable taxation and UBI encoded in metadata
- Graceful, symmetrical error and success orchestration
- Modular adapters and bridge layer for protocol extensibility
- Open source and commercial support options

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
├── explorer/           # System Currency Explorer (UI/API, WIP)
├── TimezoneZ/          # Timezone and temporal intelligence (WIP)
├── README.md           # This file
├── CONTRIBUTING.md     # How to contribute
├── CODE_OF_CONDUCT.md  # Community guidelines
├── LICENSE             # Open source license
```

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
- **Explorer**: System Currency Explorer (WIP)
- **TimezoneZ**: Temporal normalization (WIP)

## Metadata (System Currency & Extensibility)

- **Universal Ledger:** Metadata tracks every operation, service, and relationship, making it
  possible to value and reward all forms of participation—human or machine.
- **Lineage and Provenance:** Every entity, fork, and contribution is traceable via the `lineage`
  field and audit trails.
- **Extensible Patterns:** New fields, services, and value flows can be added without breaking
  existing data or requiring disruptive migrations.
- **System Currency:** The sum of all scores and values across the system forms a living, auditable
  measure of reputation, contribution, and impact.
- **See:** [docs/services/metadata.md](docs/services/metadata.md),
  [docs/amadeus/amadeus_context.md](docs/amadeus/amadeus_context.md)

## Graceful Orchestration (Error & Success)

- **Centralized Handling:** All errors and successes are handled via the `graceful` package,
  orchestrating post-action flows (logging, audit, cache, events) automatically and symmetrically.
- **Extensible Hooks:** Custom hooks and overrides allow for service-specific or advanced
  orchestration.
- **Auditability:** Every outcome is logged and auditable, supporting resilience and compliance.
- **See:** [pkg/graceful/](pkg/graceful/),
  [docs/amadeus/amadeus_context.md#platform-standard-graceful-error-and-success-handling](docs/amadeus/amadeus_context.md#platform-standard-graceful-error-and-success-handling)

## Database & Redis Practices

- **PostgreSQL:**
  - Use `jsonb` columns for rich, extensible metadata
  - Full-text search, GIN/partial indexes for performance
  - Partitioning, archiving, and audit trails for scalability
- **Redis:**
  - Hot data caching for low-latency access
  - Pub/sub for real-time events and notifications
  - Rate limiting, counters, and ephemeral state
- **See:** [docs/development/database_practices.md](docs/development/database_practices.md),
  [docs/development/redis_practices.md](docs/development/redis_practices.md)

## Handlers, Nexus, and Orchestration Patterns

- **Handlers:** REST, gRPC, and WebSocket handlers translate external requests into canonical
  metadata-driven actions.
- **Nexus:** The event bus and orchestration layer connect all services, adapters, and real-time
  flows, enabling dynamic, metadata-driven automation.
- **Bridge/Adapters:** Protocol adapters (MQTT, WebSocket, etc.) enable integration with any
  external system or device.
- **See:** [internal/nexus/service/pattern/](internal/nexus/service/pattern/),
  [internal/nexus/service/bridge/](internal/nexus/service/bridge/)

## Manifesto & Key Documentation

- [Manifesto](docs/amadeus/manifesto.md): Philosophy, values, and intent
- [Project Preface](docs/amadeus/project_preface.md): Roadmap, acknowledgments, learning philosophy
- [Amadeus Context](docs/amadeus/amadeus_context.md): System architecture, metadata, orchestration
- [Metadata Standard](docs/services/metadata.md): Canonical metadata pattern
- [Explorer](docs/amadeus/explorer.md): System Currency Explorer (WIP)

## Getting Started

1. Clone the repo
2. Install Go and dependencies (`go mod tidy`)
3. See `tax/ovasabi_default.go` for the digital will pattern
4. Explore the docs and services

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. All are welcome—code, docs, ideas, and
feedback!

## License

See [LICENSE](LICENSE) for details.

## Roadmap & Community Efforts

See the [Project Preface Roadmap](docs/amadeus/project_preface.md#coming-soon--community-roadmap)
for a detailed list of missing functionality, areas for improvement, and opportunities for community
contribution.

Highlights:

- Automated service discovery for connectors
- Dynamic relationship management (friend/family/lover/children blocks)
- Advanced orchestration and graceful error/success handling
- Accessibility and compliance automation
- Real-time knowledge graph updates and visualization
- Frontend reference implementation
- Internationalization, localization, and more

**Your ideas, feedback, and contributions are welcome!**

## Explore More

- **Experimental Features:** See `docs/architecture/experimental/` for cutting-edge ideas and
  prototypes.
- **Campaign Documentation:** See `docs/campaign/` for campaign scaffolding, best practices, and
  examples.
- **Articles:** See `docs/articles/` for in-depth explorations, technical deep-dives, and thought
  leadership.

For the full philosophy, advice, and roadmap, see
[docs/amadeus/manifesto.md](docs/amadeus/manifesto.md) and
[docs/amadeus/project_preface.md](docs/amadeus/project_preface.md).

## The OVASABI Thank You Tax

OVASABI is my digital will—a living system, a manifesto, and a gift to the world. If you've found
value here—if this project has inspired you, saved you time, or sparked new ideas—consider paying a
small "thank you tax."

This isn't a tax in the traditional sense. It's a gesture of gratitude—a way to say, _"I see you. I
appreciate the work. I want this legacy to grow."_

Your contribution helps me:

- Keep the lights on and the servers running
- Dedicate more time to building, documenting, and supporting the community
- Ensure OVASABI remains open, accessible, and evolving for everyone

**No amount is too small.** Every "thank you tax" is a vote for open knowledge, fairness, and
digital legacy.

If you'd like to contribute, you can do so here: [**Support OVASABI**](#)

<!-- Replace # with your funding link (GitHub Sponsors, Open Collective, etc.) -->

Thank you for being part of this journey. Let's keep building—together.

— Nobert Momoh (OVASABI Creator)

## Dual Licensing

OVASABI is dual-licensed:

- **MIT License:** Free and open source for community use, contributions, and research. See
  [LICENSE](LICENSE).
- **Commercial License:** For enterprises or organizations needing additional features, support,
  SLAs, or legal guarantees. Contact us for commercial licensing options.

## System Currency Explorer (Work in Progress)

The System Currency Explorer visualizes and analyzes the total value, contributions, and flows
within OVASABI. It enables:

- Viewing the total system currency (sum of all metadata scores)
- Drilling down into users, services, and tasks to see their value and history
- Tracking value flows, tax, and rewards
- Auditing provenance and digital legacy

## Canonical/Hosted Platform

The hosted version of OVASABI is the canonical source of truth for standards, updates, and
governance. Forks and integrations inherit the metadata lineage, but the hosted platform sets the
reference for the ecosystem.
