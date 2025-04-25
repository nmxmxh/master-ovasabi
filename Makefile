.PHONY: setup build test test-unit test-integration test-bench coverage benchmark clean proto docker-* k8s-*

# Variables
BINARY_NAME=master-ovasabi
DOCKER_IMAGE=ovasabi/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty)
DOCKER_COMPOSE=COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 COMPOSE_BAKE=true docker-compose -f deployments/docker/docker-compose.yml
KUBECTL=kubectl
K8S_NAMESPACE=ovasabi
K8S_CONTEXT=docker-desktop

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Protobuf parameters
PROTO_PATH=api/protos
PROTO_GO_OUT=.
PROTO_GO_OPT=paths=source_relative
PROTO_GRPC_OUT=.
PROTO_GRPC_OPT=paths=source_relative

# Setup development environment
setup: install-tools
	$(GOMOD) download
	$(GOMOD) tidy

# Install required tools
install-tools:
	$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Build the binary
build: proto
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
	find $(PROTO_PATH) -name "*.pb.go" -delete
	find $(PROTO_PATH) -name "*.pb.gw.go" -delete

# Run in development mode
dev:
	$(GOCMD) run ./cmd/server

# Docker Compose Commands
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

docker-restart:
	$(DOCKER_COMPOSE) restart

docker-restart-app:
	$(DOCKER_COMPOSE) restart app

docker-clean:
	$(DOCKER_COMPOSE) down -v --remove-orphans

docker-prune:
	docker system prune -af

# Scan Docker image with Trivy
trivy-scan:
	@echo "Scanning Docker images with Trivy..."
	@$(DOCKER_COMPOSE) images -q | xargs -I {} trivy image {}

# Scan Docker image with Trivy and fail on critical vulnerabilities
trivy-scan-ci:
	@echo "Scanning Docker images with Trivy (CI mode)..."
	@$(DOCKER_COMPOSE) images -q | xargs -I {} trivy image --exit-code 1 --severity CRITICAL {}

# Build and scan Docker image
docker-build-scan: docker-build trivy-scan

# Build and scan Docker image (CI mode)
docker-build-scan-ci: docker-build trivy-scan-ci

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@for dir in $(shell find $(PROTO_PATH) -type d); do \
		if ls $$dir/*.proto >/dev/null 2>&1; then \
			echo "Processing protos in $$dir..."; \
			protoc \
				--proto_path=. \
				--go_out=$(PROTO_GO_OUT) \
				--go_opt=$(PROTO_GO_OPT) \
				--go-grpc_out=$(PROTO_GRPC_OUT) \
				--go-grpc_opt=$(PROTO_GRPC_OPT) \
				$$dir/*.proto; \
		fi \
	done
	@echo "Protobuf code generation complete"

# Generate swagger documentation
swagger:
	swagger generate spec -o ./api/swagger/swagger.json

# Install dependencies
deps: install-tools
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	$(GOGET) -u github.com/go-swagger/go-swagger/cmd/swagger

# Kubernetes Commands
k8s-create-namespace:
	$(KUBECTL) create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -

k8s-set-context:
	$(KUBECTL) config use-context $(K8S_CONTEXT)

k8s-deploy: k8s-create-namespace
	$(KUBECTL) apply -f deployments/kubernetes/configmap.yaml -n $(K8S_NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/secret.yaml -n $(K8S_NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/deployment.yaml -n $(K8S_NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/service.yaml -n $(K8S_NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/ingress.yaml -n $(K8S_NAMESPACE)

k8s-deploy-monitoring:
	$(KUBECTL) apply -f deployments/kubernetes/monitoring/ -n $(K8S_NAMESPACE)

k8s-delete:
	$(KUBECTL) delete -f deployments/kubernetes/ -n $(K8S_NAMESPACE)

k8s-status:
	$(KUBECTL) get all -n $(K8S_NAMESPACE)

k8s-logs:
	$(KUBECTL) logs -f deployment/$(BINARY_NAME) -n $(K8S_NAMESPACE)

k8s-port-forward:
	$(KUBECTL) port-forward service/$(BINARY_NAME) 50051:50051 -n $(K8S_NAMESPACE)

k8s-dashboard:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml
	$(KUBECTL) create serviceaccount -n kubernetes-dashboard admin-user
	$(KUBECTL) create clusterrolebinding -n kubernetes-dashboard admin-user --clusterrole cluster-admin --serviceaccount=kubernetes-dashboard:admin-user
	@echo "Access token:"
	@$(KUBECTL) -n kubernetes-dashboard create token admin-user
	$(KUBECTL) proxy

# Development with Docker Desktop Kubernetes
dev-k8s: docker-build k8s-set-context k8s-deploy

# Help
help:
	@echo "Available commands:"
	@echo "Docker Compose Commands:"
	@echo "  docker-build      - Build Docker images"
	@echo "  docker-up         - Start all services"
	@echo "  docker-down       - Stop all services"
	@echo "  docker-logs       - View service logs"
	@echo "  docker-ps         - List running services"
	@echo "  docker-restart    - Restart all services"
	@echo "  docker-restart-app - Restart only the app service"
	@echo "  docker-clean      - Clean up all containers and volumes"
	@echo "  docker-prune      - Remove all unused Docker resources"
	@echo ""
	@echo "Kubernetes Commands:"
	@echo "  k8s-create-namespace  - Create Kubernetes namespace"
	@echo "  k8s-set-context      - Set Kubernetes context to docker-desktop"
	@echo "  k8s-deploy           - Deploy application to Kubernetes"
	@echo "  k8s-deploy-monitoring - Deploy monitoring stack to Kubernetes"
	@echo "  k8s-delete           - Delete application from Kubernetes"
	@echo "  k8s-status           - Show status of Kubernetes resources"
	@echo "  k8s-logs             - View application logs"
	@echo "  k8s-port-forward     - Forward application ports"
	@echo "  k8s-dashboard        - Deploy and access Kubernetes dashboard"
	@echo "  dev-k8s              - Build and deploy to local Kubernetes"
	@echo ""
	@echo "Development Commands:"
	@echo "  setup               - Setup development environment"
	@echo "  install-tools       - Install required tools"
	@echo "  build               - Build the binary"
	@echo "  test                - Run tests"
	@echo "  test-unit           - Run unit tests"
	@echo "  test-integration    - Run integration tests"
	@echo "  test-bench          - Run benchmark tests"
	@echo "  coverage            - Generate test coverage report"
	@echo "  benchmark           - Run benchmarks"
	@echo "  clean               - Clean build files"
	@echo "  dev                 - Run in development mode"
	@echo "  proto               - Generate protobuf code"
	@echo "  swagger             - Generate swagger documentation"
	@echo "  deps                - Install dependencies"

# Generate new service
new-service:
	@read -p "Enter service name: " SERVICE_NAME; \
	mkdir -p api/protos/$$SERVICE_NAME; \
	mkdir -p internal/service/$$SERVICE_NAME; \
	echo 'syntax = "proto3";\n\npackage '$$SERVICE_NAME';\n\noption go_package = "github.com/nmxmxh/master-ovasabi/api/protos/'$$SERVICE_NAME'";\n\nservice '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service {\n  // Add your RPC methods here\n}' > api/protos/$$SERVICE_NAME/$$SERVICE_NAME.proto; \
	echo 'package service\n\nimport (\n\t"context"\n\n\t"github.com/nmxmxh/master-ovasabi/api/protos/'$$SERVICE_NAME'"\n\t"go.uber.org/zap"\n)\n\n// '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service implements the '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service interface\ntype '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service struct {\n\tlogger *zap.Logger\n}\n\n// New'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service creates a new '$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service instance\nfunc New'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service(logger *zap.Logger) *'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service {\n\treturn &'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service{\n\t\tlogger: logger,\n\t}\n}' > internal/service/$$SERVICE_NAME/$$SERVICE_NAME.go; \
	echo 'package service\n\nimport (\n\t"context"\n\t"testing"\n\n\t"github.com/nmxmxh/master-ovasabi/api/protos/'$$SERVICE_NAME'"\n\t"github.com/stretchr/testify/assert"\n\t"go.uber.org/zap"\n)\n\nfunc TestNew'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service(t *testing.T) {\n\tlogger := zap.NewNop()\n\tsvc := New'$$(echo $$SERVICE_NAME | tr '[:lower:]' '[:upper:]')'Service(logger)\n\n\tassert.NotNil(t, svc)\n\tassert.Equal(t, logger, svc.logger)\n}' > internal/service/$$SERVICE_NAME/$$SERVICE_NAME_test.go 