# Master Ovasabi

A production-ready Go gRPC service boilerplate with comprehensive monitoring, concurrency
management, and Kubernetes support.

## Architecture Overview

The project follows a clean architecture approach with clear separation of concerns:

- **Service Layer**: Business logic implementation
- **Repository Layer**: Data access and persistence
- **Infrastructure Layer**: Cross-cutting concerns (monitoring, concurrency, etc.)

### Key Features

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

- Modular, concurrent service registration using dependency injection (DI)
- Each service is automatically registered as a pattern in the Nexus orchestrator for orchestration
  and introspection
- Robust error handling for all registration and orchestration steps

The Provider struct now includes a patternStore field, which manages pattern orchestration
registration in Nexus.

## Canonical Metadata Pattern

All services now use a central, extensible `common.Metadata` message for all metadata fields. This
enables:

- Consistent, discoverable metadata across all services
- Service-specific extensibility via the `service_specific` field
- Efficient storage and querying with Postgres `jsonb`
- Intrinsic scheduling and orchestration via the Scheduler service and Postgres triggers
- Knowledge graph and AI/ML integration

See the Amadeus context for full documentation and best practices.

## Getting Started

1. **Prerequisites**

   - Go 1.22.2+
   - Docker & Kubernetes
   - Make
   - Protocol Buffers compiler (protoc)

2. **Installation**

   ```bash
   # Install protobuf generators
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

   # Setup project
   make setup
   make build
   ```

3. **Development**

   ```bash
   # Generate protobuf code
   protoc --go_out=. --go_opt=paths=source_relative \
          --go-grpc_out=. --go-grpc_opt=paths=source_relative \
          api/protos/*.proto

   # Run tests
   make test
   ```

4. **Docker Build & Run**

   ```bash
   # Build and run with Docker Compose
   cd deployments/docker
   docker-compose up --build -d

   # Or build Docker image manually
   docker build -t ovasabi/master-ovasabi:latest -f deployments/docker/Dockerfile .

   # Run container manually
   docker run -p 50051:50051 -p 9090:9090 ovasabi/master-ovasabi:latest
   ```

5. **Kubernetes Deployment**

   ```bash
   kubectl apply -f deployments/kubernetes/deployment.yaml
   ```

## Directory Structure

```text
.
├── api/                # API definitions (protobuf)
├── cmd/                # Application entry points
├── config/             # Configuration files
├── deployments/        # Deployment configurations
├── docs/              # Documentation
├── internal/          # Private application code
│   ├── server/       # gRPC server implementation
│   └── service/      # Service implementations (User, Notification, Content, Commerce, etc.)
├── pkg/               # Public packages
│   ├── logger/       # Logging package
│   ├── metrics/      # Metrics package
│   └── tracing/      # Tracing package
└── test/              # Test suites
```

## Documentation

- [Architecture](docs/architecture.md)
- [API Documentation](docs/api.md)
- [Development Guide](docs/development.md)
- [Deployment Guide](docs/deployment.md)
- [Documentation Tooling](docs/tools/documentation-tooling.md)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License

## Protobuf Code Generation

### Prerequisites

Before generating protobuf code, ensure you have the following installed:

1. Protocol Buffers Compiler (protoc)
2. Go plugins for protoc:

   ```bash
   make deps
   ```

   This will install:

   - protoc-gen-go (for generating Go code)
   - protoc-gen-grpc-gateway (for gRPC-Gateway)
   - protoc-gen-swagger (for Swagger/OpenAPI)
   - go-swagger (for API documentation)

### Generating Code

To generate code from your .proto files:

```bash
make proto
```

This command will generate Go code for all proto files in the following directories:

- api/protos/auth/
- api/protos/broadcast/
- api/protos/i18n/
- api/protos/notification/
- api/protos/quotes/
- api/protos/referral/
- api/protos/user/
- api/protos/content/

### Creating a New Service

To create a new service with proto files:

```bash
make new-service
```

When prompted, enter your service name (e.g., payment, inventory, content). This will:

1. Create a new directory in api/protos/<service_name>
2. Generate a basic proto file template
3. Create a service implementation template in internal/service/

After creating a new service, run `make proto` to generate the Go code.

### Available Make Commands

- `make deps` - Install all required protobuf dependencies
- `make proto` - Generate Go code from proto files
- `make swagger` - Generate Swagger/OpenAPI documentation
- `make help` - Show all available make commands
