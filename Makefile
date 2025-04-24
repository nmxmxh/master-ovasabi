.PHONY: setup build test test-unit test-integration test-bench coverage benchmark clean

# Variables
BINARY_NAME=master-ovasabi
DOCKER_IMAGE=ovasabi/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Setup development environment
setup:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	$(GOTEST) -v -race ./... -run "^Test" -tags=unit

# Run integration tests
test-integration:
	$(GOTEST) -v -race ./test/integration/... -run "^Test" -tags=integration

# Run benchmark tests
test-bench:
	$(GOTEST) -v ./test/benchmarks/... -run=^$$ -bench=. -benchmem

# Generate test coverage report
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	$(GOCLEAN) coverage.out

# Run benchmarks
benchmark:
	$(GOTEST) -bench=. ./test/benchmarks/...

# Clean build files
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run in development mode
dev:
	$(GOCMD) run ./cmd/server

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

# Run Docker container
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(VERSION)

# Scan Docker image with Trivy
trivy-scan:
	@echo "Scanning Docker image with Trivy..."
	trivy image $(DOCKER_IMAGE):$(VERSION)

# Scan Docker image with Trivy and fail on critical vulnerabilities
trivy-scan-ci:
	@echo "Scanning Docker image with Trivy (CI mode)..."
	trivy image --exit-code 1 --severity CRITICAL $(DOCKER_IMAGE):$(VERSION)

# Build and scan Docker image
docker-build-scan: docker-build trivy-scan

# Build and scan Docker image (CI mode)
docker-build-scan-ci: docker-build trivy-scan-ci

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/protos/auth/*.proto \
		api/protos/broadcast/*.proto \
		api/protos/i18n/*.proto \
		api/protos/notification/*.proto \
		api/protos/quotes/*.proto \
		api/protos/referral/*.proto \
		api/protos/user/*.proto
	@echo "Protobuf code generation complete"

# Generate swagger documentation
swagger:
	swagger generate spec -o ./api/swagger/swagger.json

# Install dependencies
deps:
	$(GOGET) -u github.com/golang/protobuf/protoc-gen-go
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	$(GOGET) -u github.com/go-swagger/go-swagger/cmd/swagger

# Help
help:
	@echo "Available commands:"
	@echo "  setup           - Setup development environment"
	@echo "  build           - Build the binary"
	@echo "  test            - Run tests"
	@echo "  test-unit       - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-bench      - Run benchmark tests"
	@echo "  coverage        - Generate test coverage report"
	@echo "  benchmark       - Run benchmarks"
	@echo "  clean           - Clean build files"
	@echo "  dev             - Run in development mode"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  proto           - Generate protobuf code"
	@echo "  swagger         - Generate swagger documentation"
	@echo "  deps            - Install dependencies"

# Generate new service
new-service:
	@read -p "Enter service name (e.g., user, payment): " SERVICE_NAME; \
	mkdir -p api/protos/$$SERVICE_NAME internal/service; \
	echo 'syntax = "proto3";\n\npackage '$$SERVICE_NAME';\n\noption go_package = "github.com/ovasabi/master-ovasabi/api/protos/'$$SERVICE_NAME'";\n\nservice '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service {\n  // Add your RPC methods here\n}' > api/protos/$$SERVICE_NAME/$$SERVICE_NAME.proto; \
	echo 'package service\n\nimport (\n\t"context"\n\n\t"github.com/ovasabi/master-ovasabi/api/protos/'$$SERVICE_NAME'"\n\t"go.uber.org/zap"\n)\n\n// '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service implements the '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service gRPC service\ntype '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service struct {\n\t'$$SERVICE_NAME'.Unimplemented'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'ServiceServer\n\tlog *zap.Logger\n}\n\n// New'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service creates a new instance of '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service\nfunc New'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service(log *zap.Logger) *'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service {\n\treturn &'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service{\n\t\tlog: log,\n\t}\n}' > internal/service/$$SERVICE_NAME.go; \
	echo "Service $$SERVICE_NAME created successfully"; \
	echo "Run 'make proto' to generate protobuf code" 