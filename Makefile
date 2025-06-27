.PHONY: setup build test test-unit test-integration test-bench coverage benchmark clean proto docker-* k8s-* docs backup lint-fix docs-format docs-check-format docs-check-links docs-validate restore js-setup docs-all docs-site-setup docs-site docs-serve docs-deploy-github docs-prepare-hosting lint-focused docs-fix-links openapi-gen openapi-validate openapi-diff sync-openapi update-doc-dates validate-doc-dates openapi-json-diff docs-generate-tests wasm-build frontend-build frontend-dev wasm-dev docker-wasm-build docker-wasm-up docker-wasm-down wasm-threaded serve-wasm helm-install helm-upgrade helm-uninstall helm-status helm-dry-run

# Variables
BINARY_NAME=master-ovasabi
DOCKER_IMAGE=ovasabi/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty)
DOCKER_COMPOSE=COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 COMPOSE_BAKE=true docker-compose -f deployments/docker/docker-compose.yml
KUBECTL=kubectl
K8S_NAMESPACE=ovasabi
K8S_CONTEXT=docker-desktop

-include .env
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
	@yarn format:check -- --ignore-path .prettierignore
	@echo "Linting checks completed."

# Focused linting (excludes backups and amadeus knowledge graph)
lint-focused:
	@echo "Running Go linter checks (excluding backup files and amadeus utilities)..."
	golangci-lint run ./cmd/... ./internal/... ./pkg/...
	@echo "Checking Markdown documentation formatting..."
	@yarn format:check -- --ignore-path .prettierignore
	@echo "Linting checks completed."

# Lint-safe command that completely excludes amadeus and vendor directories (documentation/knowledge graph utilities only)
lint-safe:
	@echo "Running Go linter checks (excluding amadeus and vendor directories)..."
	golangci-lint run ./cmd/... ./internal/... ./pkg/... 
	@echo "Checking Markdown documentation formatting (excluding amadeus and vendor)..."
	@yarn format:check -- --ignore-path .prettierignore
	@echo "Linting checks completed."

# Run in development mode
dev:
	$(GOCMD) run ./cmd/server

# Docker Compose Commands
docker-build: proto
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

# Docker cleanup target
# Usage:
#   make docker-clean         # Safe cleanup: build cache, images, containers, volumes
#   make docker-clean ALL=1   # Aggressive: also does full system prune (removes ALL unused images, containers, networks, and volumes)

docker-clean:
	@echo "[docker-clean] Pruning Docker build cache (this will free the most space)..."
	docker builder prune -a -f
	@echo "[docker-clean] Pruning unused Docker images..."
	docker image prune -a -f
	@echo "[docker-clean] Pruning unused Docker containers..."
	docker container prune -f
	@echo "[docker-clean] Pruning unused Docker volumes..."
	docker volume prune -f
ifneq ($(ALL),)
	@echo "[docker-clean] WARNING: Running full system prune. This will remove ALL unused images, containers, networks, and volumes!"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		docker system prune -a -f --volumes; \
		echo "[docker-clean] Full system prune complete."; \
	else \
		echo "[docker-clean] Skipping full system prune."; \
	fi
endif
	@echo "[docker-clean] Docker cleanup complete!"

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

# Generate protobuf code using protoc
# This command generates Go code only for the latest version (v*) directory in each service directory under api/protos.
# This avoids generating code for old/unused proto versions and keeps generated code up-to-date.
# For each service, only the latest version's .proto files are processed.
# Always run this from the repo root.
proto:
	@echo "Generating protobuf code for latest proto versions only..."
	@for proto_dir in $(shell find $(PROTO_PATH) -mindepth 1 -maxdepth 1 -type d); do \
		latest_version_dir=$$(ls -d $$proto_dir/v*/ 2>/dev/null | sort -V | tail -n 1); \
		if [ -d "$$latest_version_dir" ]; then \
			echo "Processing protos in $$latest_version_dir..."; \
			for proto_file in $$(find $$latest_version_dir -name '*.proto'); do \
				protoc \
					-I=$(PROTO_PATH) \
					--go_out=$(PROTO_PATH) \
					--go_opt=paths=source_relative \
					--go-grpc_out=$(PROTO_PATH) \
					--go-grpc_opt=paths=source_relative \
					$$proto_file; \
				done; \
		fi \
	done
	@echo "Protobuf code generation complete"
# This ensures generated Go files are placed alongside their proto files (in-place)

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
	@echo "Renaming .go files in backups to .go.backup to avoid Go toolchain errors..."
	@find amadeus/backups -type f -name '*.go' -exec mv {} {}.backup \;
	@echo "Backup process completed successfully."

# Linting Fix Commands
lint-fix:
	@echo "Applying linting fixes to Go code..."
	golangci-lint run --fix
	@echo "Formatting Markdown documentation files..."
	@yarn format:docs --ignore-path .prettierignore
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
	@echo "Documentation & OpenAPI Commands:"
	@echo "  docs-all           - Update dates, generate all docs and OpenAPI schemas, validate everything (2025-05-14)"
	@echo "  update-doc-dates   - Update all docs/specs with current date and context block"
	@echo "  validate-doc-dates - Validate doc/spec recency and context block (TODO)"
	@echo "  docs               - Generate Markdown docs from code comments"
	@echo "  openapi-gen        - Generate OpenAPI schema from proto"
	@echo "  openapi-validate   - Validate OpenAPI schemas"
	@echo "  openapi-diff       - Diff generated and canonical OpenAPI schemas"
	@echo "  sync-openapi       - Sync generated OpenAPI schema to canonical location"
	@echo "  docs-format        - Format all Markdown documentation files with Prettier"
	@echo "  docs-check-format  - Check Markdown files formatting without making changes"
	@echo "  docs-check-links   - Check for broken links in documentation files"
	@echo "  docs-validate      - Run all documentation validation checks"
	@echo "  docs-all           - Generate, format and validate all documentation"
	@echo "  docs-site-setup    - Set up MkDocs for documentation site"
	@echo "  docs-site          - Generate documentation site"
	@echo "  docs-serve         - Serve documentation site locally"
	@echo "  docs-deploy-github - Deploy documentation to GitHub Pages"
	@echo "  docs-prepare-hosting - Prepare documentation for hosting"
	@echo "  docs-fix-links     - Fix documentation links"
	@echo "Linting Commands:"
	@echo "  lint               - Check Go code with golangci-lint and Markdown with Prettier (no fixes)"
	@echo "  lint-focused       - Same as lint but excludes backup files"
	@echo "  lint-fix           - Apply fixes to Go code and format Markdown documentation"
	@echo "  js-setup           - Install JavaScript dependencies with Yarn"
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
	$(DOCKER_COMPOSE) run --rm migrate -path=/migrations -database "postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-postgres}@postgres:5432/$${DB_NAME:-master_ovasabi}?sslmode=disable" $${ARGS:-up}

# Generate scenario-driven test documentation (auto-generated from test suites)
docs-generate-tests:
	go run scripts/gen_test_docs.go

# Main documentation workflow (add docs-generate-tests as a prerequisite)
docs: docs-generate-tests docs-format docs-validate
	@echo "All documentation generated and validated."

docs-all: docs-generate-tests docs-format docs-validate docs-serve
	@echo "Full documentation workflow complete."

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

# OpenAPI schema generation and validation
openapi-gen: update-doc-dates
	@echo "Generating OpenAPI schema from metadata.proto..."
	@go run tools/protoc/main.go metadata.proto api/protos/common/v1 docs/generated/openapi --openapi

openapi-validate:
	@echo "Validating OpenAPI schemas..."
	@npx swagger-cli validate docs/generated/openapi/metadata.swagger.json
	@npx swagger-cli validate docs/services/metadata_openapi.json

# Usage:
#   make openapi-gen      # Generate OpenAPI schema from proto
#   make openapi-validate # Validate OpenAPI schemas (requires swagger-cli)

openapi-diff:
	@echo "Diffing generated and canonical OpenAPI schemas..."
	@diff -u docs/generated/openapi/metadata.swagger.json docs/services/metadata_openapi.yaml || echo "Schemas differ (expected if updating standards)"

sync-openapi:
	# Convert JSON to YAML to avoid schema/linter errors (do not copy JSON as YAML)
	@yq -P eval '.' docs/generated/openapi/metadata.swagger.json > docs/services/metadata_openapi.yaml
	@echo "Canonical OpenAPI schema updated from generated output and converted to YAML. (2024-06-14)"

update-doc-dates:
	@./scripts/update-doc-dates.sh

validate-doc-dates:
	@echo "(TODO) Validate doc dates for recency and context block."

# Diff the generated OpenAPI JSON and the canonical metadata_openapi.json for review
openapi-json-diff:
	@echo "Diffing generated and canonical OpenAPI JSON schemas..."
	@diff -u docs/generated/openapi/metadata.swagger.json docs/services/metadata_openapi.json || echo "Schemas differ (expected if updating standards)"

# rest-point: Safely update all documentation, OpenAPI specs, lint code/docs, and back up the knowledge graph.
rest-point: docs-all
	-$(MAKE) lint
	$(MAKE) backup

# Run go vet for static analysis on all Go files
vet:
	go vet ./...

api-docs:
	swag init -g internal/server/handlers/docs.go -o docs/api

# --- WASM/Frontend Build (Vite-centric, progressive threads) ---

wasm-build:
	@./scripts/build-wasm.sh

wasm-threaded: wasm-build

# Start Vite dev server (ensure vite.config.js sets COOP/COEP headers for WASM threads)
vite-wasm:
	cd frontend && yarn dev
	@echo "[NOTE] If using WASM threads, ensure Vite is configured to set COOP/COEP headers. See vite.config.js example in docs."

# Remove old serve-wasm (python http.server) target, as Vite is used for serving

# Helm Chart Commands
helm-install:
	helm install ovasabi deployments/kubernetes

helm-upgrade:
	helm upgrade ovasabi deployments/kubernetes

helm-uninstall:
	helm uninstall ovasabi

helm-status:
	helm status ovasabi

helm-dry-run:
	helm install ovasabi deployments/kubernetes --dry-run --debug