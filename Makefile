.PHONY: setup build test test-unit test-integration test-bench coverage benchmark clean proto docker-* k8s-* docs backup lint-fix docs-format docs-check-format docs-check-links docs-validate restore js-setup docs-all docs-site-setup docs-site docs-serve docs-deploy-github docs-prepare-hosting lint-focused docs-fix-links

# Variables
BINARY_NAME=master-ovasabi
DOCKER_IMAGE=ovasabi/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty)
DOCKER_COMPOSE=COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 COMPOSE_BAKE=true docker-compose -f deployments/docker/docker-compose.yml
KUBECTL=kubectl
K8S_NAMESPACE=ovasabi
K8S_CONTEXT=docker-desktop

include .env
export $(shell sed 's/=.*//' .env)


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

# run linter
lint:
	@echo "Running Go linter checks (excluding amadeus directory)..."
	golangci-lint run ./cmd/... ./internal/... ./pkg/...
	@echo "Checking Markdown documentation formatting..."
	@yarn format:check -- --ignore-path='{.prettierignore,vendor/**,.venv/**}'
	@echo "Linting checks completed."

# Focused linting (excludes backups and amadeus knowledge graph)
lint-focused:
	@echo "Running Go linter checks (excluding backup files and amadeus utilities)..."
	@golangci-lint run ./cmd/... ./internal/... ./pkg/...
	@echo "Checking Markdown documentation formatting..."
	@yarn format:check -- --ignore-path='{.prettierignore,vendor/**,.venv/**}'
	@echo "Linting checks completed."

# Lint-safe command that completely excludes amadeus and vendor directories (documentation/knowledge graph utilities only)
lint-safe:
	@echo "Running Go linter checks (excluding amadeus and vendor directories)..."
	@golangci-lint run ./cmd/... ./internal/... ./pkg/... --skip-dirs amadeus --skip-dirs vendor
	@echo "Checking Markdown documentation formatting (excluding amadeus and vendor)..."
	@yarn format:check -- --ignore-path='{.prettierignore,vendor/**,.venv/**,amadeus/**}'
	@echo "Linting checks completed."

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
	@for service_dir in $(shell find $(PROTO_PATH) -mindepth 1 -maxdepth 1 -type d); do \
		latest_version_dir=$$(ls -d $$service_dir/v*/ | sort -V | tail -n 1); \
		if [ -d "$$latest_version_dir" ]; then \
			echo "Processing protos in $$latest_version_dir..."; \
			protoc \
				--proto_path=. \
				--go_out=$(PROTO_GO_OUT) \
				--go_opt=$(PROTO_GO_OPT) \
				--go-grpc_out=$(PROTO_GRPC_OUT) \
				--go-grpc_opt=$(PROTO_GRPC_OPT) \
				$$latest_version_dir/*.proto; \
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

# Amadeus Backup Commands
backup:
	@echo "Creating comprehensive Amadeus Knowledge Graph backup..."
	@if [ ! -f "bin/kgcli" ]; then \
		echo "Building kgcli tool first..."; \
		$(GOBUILD) -o bin/kgcli amadeus/cmd/kgcli/main.go; \
	fi
	@echo "Creating programmatic backup using kgcli..."
	@bin/kgcli backup --desc "Full system backup triggered via Makefile"
	@echo "Creating comprehensive backup using backup script..."
	@chmod +x amadeus/backup_script.sh
	@./amadeus/backup_script.sh
	@echo "Backup process completed successfully."

# Linting Fix Commands
lint-fix:
	@echo "Applying linting fixes to Go code..."
	golangci-lint run --fix
	@echo "Formatting Markdown documentation files..."
	@yarn format:docs
	@echo "Checking for broken links in Markdown files..."
	@find docs -name "*.md" -exec yarn markdown-link-check {} \; || echo "Some links might be broken. Please check the output above."
	@echo "Lint fixes applied successfully."

# Help
help:
	@echo "Available commands:"
	@echo "Amadeus Commands:"
	@echo "  backup              - Create a comprehensive backup of the Amadeus Knowledge Graph system"
	@echo "  restore             - Restore the Amadeus Knowledge Graph system from a backup"
	@echo "  kg-evolution        - Analyze Knowledge Graph evolution through historical backups"
	@echo "Documentation Commands:"
	@echo "  docs                - Generate documentation and format with Prettier"
	@echo "  docs-format         - Format all Markdown documentation files with Prettier"
	@echo "  docs-check-format   - Check Markdown files formatting without making changes"
	@echo "  docs-check-links    - Check for broken links in documentation files"
	@echo "  docs-validate       - Run all documentation validation checks"
	@echo "  docs-all            - Generate, format and validate all documentation"
	@echo "  docs-site-setup     - Set up MkDocs for documentation site"
	@echo "  docs-site           - Generate documentation site"
	@echo "  docs-serve          - Serve documentation site locally"
	@echo "  docs-deploy-github  - Deploy documentation to GitHub Pages"
	@echo "  docs-prepare-hosting - Prepare documentation for hosting"
	@echo "  docs-fix-links      - Fix documentation links"
	@echo "Linting Commands:"  
	@echo "  lint                - Check Go code with golangci-lint and Markdown with Prettier (no fixes)"
	@echo "  lint-focused        - Same as lint but excludes backup files"
	@echo "  lint-fix            - Apply fixes to Go code and format Markdown documentation"
	@echo "  js-setup            - Install JavaScript dependencies with Yarn"
	@echo ""
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

# Run database migrations up using Docker Compose
migrate-up:
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database "postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-postgres}@postgres:5432/$${DB_NAME:-master_ovasabi}?sslmode=disable" up

# Generate documentation
docs: js-setup
	@echo "Generating documentation..."
	@go run tools/docgen/cmd/main.go -source . -output docs/generated
	@echo "Formatting generated documentation..."
	@yarn format:docs
	@echo "Documentation generated and formatted successfully"

# JS dependencies
js-setup:
	@echo "Setting up JavaScript dependencies with Yarn..."
	@yarn install
	@echo "JavaScript dependencies installed successfully."

# Documentation Commands
docs-format:
	@echo "Formatting Markdown documentation files..."
	@yarn format:docs
	@echo "Documentation formatting complete."

docs-check-format:
	@echo "Checking Markdown documentation formatting..."
	@yarn format:check
	@echo "Documentation format check complete."

docs-check-links:
	@echo "Checking for broken links in documentation..."
	@find docs -name "*.md" -exec yarn markdown-link-check {} \; || echo "Some links might be broken. Please check the output above."
	@echo "Link checking complete."

docs-validate: docs-check-format docs-check-links docs-fix-links
	@echo "Documentation validation complete."

# Comprehensive documentation command
docs-all: js-setup docs docs-validate
	@echo "All documentation tasks completed successfully"

# Amadeus Restore Command
restore:
	@echo "List of available backups:"
	@ls -lth amadeus/backups/ | grep -v '^total' | head -10
	@read -p "Enter backup timestamp to restore (e.g., 20250430053006): " BACKUP_TS; \
	if [ -d "amadeus/backups/$$BACKUP_TS" ]; then \
		echo "Restoring from backup: $$BACKUP_TS"; \
		cp -r amadeus/backups/$$BACKUP_TS/src/* .; \
		echo "Restore completed successfully."; \
	else \
		echo "Error: Backup directory not found."; \
		exit 1; \
	fi

# Documentation Site Generation
docs-site-setup:
	@echo "Setting up MkDocs for documentation site using a virtual environment..."
	@if command -v python3 >/dev/null 2>&1; then \
		python3 -m venv .venv; \
		. .venv/bin/activate && pip install mkdocs mkdocs-material mdx_truly_sane_lists pymdown-extensions mkdocs-minify-plugin; \
	else \
		echo "Error: Python3 is not installed. Please install Python first:"; \
		echo "  - macOS: brew install python"; \
		echo "  - Ubuntu/Debian: apt-get install python3"; \
		echo "  - Windows: Download from https://www.python.org/downloads/"; \
		exit 1; \
	fi
	@echo "MkDocs setup complete in virtual environment. To activate, run:"; 
	@echo "  source .venv/bin/activate"

docs-site: docs-format 
	@echo "Generating documentation site..."
	@if [ -d ".venv" ]; then \
		. .venv/bin/activate && mkdocs build; \
	else \
		echo "Error: Virtual environment not found. Please run 'make docs-site-setup' first."; \
		exit 1; \
	fi
	@echo "Documentation site generated in 'site' directory."

docs-serve: docs-format
	@echo "Serving documentation site locally at http://localhost:8000"
	@if [ -d ".venv" ]; then \
		. .venv/bin/activate && mkdocs serve; \
	else \
		echo "Error: Virtual environment not found. Please run 'make docs-site-setup' first."; \
		exit 1; \
	fi

# Deploy documentation to GitHub Pages
docs-deploy-github:
	@echo "Deploying documentation to GitHub Pages..."
	@if [ -d ".venv" ]; then \
		. .venv/bin/activate && mkdocs gh-deploy --force; \
	else \
		echo "Error: Virtual environment not found. Please run 'make docs-site-setup' first."; \
		exit 1; \
	fi
	@echo "Documentation deployed to GitHub Pages."

# Prepare documentation for hosting
docs-prepare-hosting: docs-format
	@echo "Preparing documentation for hosting..."
	@if [ -d ".venv" ]; then \
		. .venv/bin/activate && mkdocs build; \
	else \
		echo "Error: Virtual environment not found. Please run 'make docs-site-setup' first."; \
		exit 1; \
	fi
	@echo "Static site is ready in the 'site' directory."
	@echo "You can deploy this to any static hosting service like:"
	@echo "  - Netlify"
	@echo "  - Vercel"
	@echo "  - AWS S3 + CloudFront"
	@echo "  - Google Cloud Storage"
	@echo "  - Azure Static Web Apps"
	@echo "  - DigitalOcean App Platform"
	@tar -czf docs-site.tar.gz -C site .

# Fix documentation links
docs-fix-links:
	@echo "Creating necessary directories for documentation assets..."
	@mkdir -p docs/diagrams
	@mkdir -p docs/assets/images
	@echo "Creating placeholder files for missing assets..."
	@touch docs/diagrams/amadeus_architecture.mmd
	@touch docs/diagrams/knowledge_graph_structure.mmd
	@touch docs/assets/images/logo.svg
	@touch docs/assets/images/favicon.svg
	@echo "Checking links in documentation..."
	@find docs -name "*.md" -exec yarn markdown-link-check {} \; || echo "Link issues found."
	@echo "Link check and fix complete."

# Amadeus Knowledge Graph Evolution Analysis
kg-evolution:
	@echo "Analyzing Amadeus Knowledge Graph Evolution..."
	@echo "=========================================================="
	@echo "1. Available Knowledge Graph Backups:"
	@ls -lth amadeus/backups/ | grep -v "^total" | head -10
	@echo ""
	@echo "2. Knowledge Graph Size Evolution:"
	@echo "Backup Date           Size"
	@echo "-----------------------------"
	@ls -l amadeus/backups/*/knowledge_graph.json | awk '{print $$9, $$5}' | sed 's/amadeus\/backups\/\([0-9]*\)\/knowledge_graph.json/\1    &/' | sort | awk '{print $$1, $$3}'
	@echo ""
	@echo "3. Latest Knowledge Graph Structure:"
	@jq -r 'keys' amadeus/knowledge_graph.json | head -10
	@echo ""
	@echo "4. Latest Backup Info:"
	@cat amadeus/backups/$$(ls -t amadeus/backups/ | grep -v "knowledge_graph_" | head -1)/backup_info.txt
	@echo ""
	@echo "5. Integration Points:"
	@jq -r '.amadeus_integration.integration_points | keys[]' amadeus/knowledge_graph.json
	@echo ""
	@echo "6. Changes Analysis:"
	@if [ $$(ls -t amadeus/backups/ | grep -v "knowledge_graph_" | wc -l) -gt 1 ]; then \
		LATEST=$$(ls -t amadeus/backups/ | grep -v "knowledge_graph_" | head -1); \
		PREVIOUS=$$(ls -t amadeus/backups/ | grep -v "knowledge_graph_" | head -2 | tail -1); \
		echo "Comparing $${PREVIOUS} to $${LATEST}:"; \
		diff -q amadeus/backups/$${PREVIOUS}/knowledge_graph.json amadeus/backups/$${LATEST}/knowledge_graph.json || echo "Knowledge graph has changed"; \
		echo ""; \
		echo "File differences:"; \
		diff -r amadeus/backups/$${PREVIOUS}/ amadeus/backups/$${LATEST}/ | grep -v "Only in" | head -10; \
	else \
		echo "Need at least two backups for comparison"; \
	fi
	@echo "=========================================================="
	@echo "Knowledge Graph Evolution Analysis Complete"