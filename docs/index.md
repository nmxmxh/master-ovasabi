# OVASABI Platform Documentation

Welcome to the official documentation for the OVASABI Platform.

## Overview

The OVASABI Platform is a production-ready Go gRPC service boilerplate with comprehensive
monitoring, concurrency management, and Kubernetes support. It follows a clean architecture approach
with clear separation of concerns.

## Key Components

- **Service Layer**: Business logic implementation
- **Repository Layer**: Data access and persistence
- **Infrastructure Layer**: Cross-cutting concerns

## Main Features

- **Monitoring & Observability**

  - Prometheus metrics collection
  - OpenTelemetry distributed tracing with Jaeger
  - Structured logging with Zap
  - Health checks and readiness probes

- **Service Implementation**

  - gRPC service with interceptors
  - Dependency injection
  - Interface-based design
  - Comprehensive error handling
  - Request context management

- **Amadeus Knowledge Graph**
  - Self-documenting architecture
  - Knowledge persistence
  - Programmatic accessibility
  - Impact analysis
  - Architectural compliance

## Getting Started

- [Architecture Overview](architecture/README.md)
- [Development Guide](development/README.md)
- [Deployment Guide](deployment/README.md)
- [Amadeus Overview](amadeus/index.md)

## Contributing

For information on how to contribute to this project, please see the CONTRIBUTING.md file in the
repository root.
