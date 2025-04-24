# Tools Documentation

## Overview

This documentation covers the various tools and utilities used in the OVASABI platform for development, testing, and deployment.

## Build Tools

1. **Make**

   ```makefile
   # Example from Makefile
   .PHONY: build test clean
   
   build:
       go build -o bin/server cmd/server/main.go
   
   test:
       go test ./... -v
   
   clean:
       rm -rf bin/
   ```

2. **Go Modules**

   ```bash
   # Initialize module
   go mod init github.com/ovasabi/master-ovasabi
   
   # Add dependency
   go get github.com/example/dependency
   
   # Tidy dependencies
   go mod tidy
   ```

## Development Tools

1. **Code Generation**

   ```go
   // Example from tools/codegen/main.go
   func main() {
       // Generate protobuf code
       if err := protoc.Generate(); err != nil {
           log.Fatalf("failed to generate protobuf code: %v", err)
       }
       
       // Generate mocks
       if err := mockgen.Generate(); err != nil {
           log.Fatalf("failed to generate mocks: %v", err)
       }
   }
   ```

2. **Linting**

   ```bash
   # Run linter
   golangci-lint run
   
   # Fix issues
   golangci-lint run --fix
   ```

## Testing Tools

1. **Unit Testing**

   ```go
   // Example from tools/test/main.go
   func main() {
       // Run tests
       if err := test.Run(); err != nil {
           log.Fatalf("tests failed: %v", err)
       }
       
       // Generate coverage
       if err := test.Coverage(); err != nil {
           log.Fatalf("failed to generate coverage: %v", err)
       }
   }
   ```

2. **Benchmarking**

   ```go
   // Example from tools/bench/main.go
   func main() {
       // Run benchmarks
       if err := bench.Run(); err != nil {
           log.Fatalf("benchmarks failed: %v", err)
       }
       
       // Generate report
       if err := bench.Report(); err != nil {
           log.Fatalf("failed to generate report: %v", err)
       }
   }
   ```

## Deployment Tools

1. **Docker**

   ```dockerfile
   # Example from Dockerfile
   FROM golang:1.21-alpine AS builder
   
   WORKDIR /app
   COPY . .
   RUN go build -o server cmd/server/main.go
   
   FROM alpine:latest
   COPY --from=builder /app/server /server
   CMD ["/server"]
   ```

2. **Kubernetes**

   ```yaml
   # Example from k8s/deployment.yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: ovasabi-server
   spec:
     replicas: 3
     template:
       spec:
         containers:
         - name: server
           image: ovasabi/server:latest
           ports:
           - containerPort: 8080
   ```

## Monitoring Tools

1. **Metrics Collection**

   ```go
   // Example from tools/metrics/main.go
   func main() {
       // Start metrics server
       if err := metrics.Start(); err != nil {
           log.Fatalf("failed to start metrics server: %v", err)
       }
       
       // Collect metrics
       if err := metrics.Collect(); err != nil {
           log.Fatalf("failed to collect metrics: %v", err)
       }
   }
   ```

2. **Logging**

   ```go
   // Example from tools/logging/main.go
   func main() {
       // Configure logging
       if err := logging.Configure(); err != nil {
           log.Fatalf("failed to configure logging: %v", err)
       }
       
       // Start log collection
       if err := logging.Start(); err != nil {
           log.Fatalf("failed to start log collection: %v", err)
       }
   }
   ```

## Security Tools

1. **Vulnerability Scanning**

   ```bash
   # Run security scan
   go list -json -m all | nancy sleuth
   
   # Check dependencies
   go mod verify
   ```

2. **Static Analysis**

   ```bash
   # Run static analysis
   gosec ./...
   
   # Check for common issues
   staticcheck ./...
   ```

## Documentation Tools

1. **API Documentation**

   ```go
   // Example from tools/docs/main.go
   func main() {
       // Generate API docs
       if err := docs.Generate(); err != nil {
           log.Fatalf("failed to generate API docs: %v", err)
       }
       
       // Generate OpenAPI spec
       if err := docs.OpenAPI(); err != nil {
           log.Fatalf("failed to generate OpenAPI spec: %v", err)
       }
   }
   ```

2. **Code Documentation**

   ```bash
   # Generate godoc
   godoc -http=:6060
   
   # Check documentation
   golint ./...
   ```

## Utility Tools

1. **Database Migration**

   ```go
   // Example from tools/migrate/main.go
   func main() {
       // Run migrations
       if err := migrate.Up(); err != nil {
           log.Fatalf("failed to run migrations: %v", err)
       }
       
       // Verify schema
       if err := migrate.Verify(); err != nil {
           log.Fatalf("failed to verify schema: %v", err)
       }
   }
   ```

2. **Data Import/Export**

   ```go
   // Example from tools/data/main.go
   func main() {
       // Import data
       if err := data.Import(); err != nil {
           log.Fatalf("failed to import data: %v", err)
       }
       
       // Export data
       if err := data.Export(); err != nil {
           log.Fatalf("failed to export data: %v", err)
       }
   }
   ```
