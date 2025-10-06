
.PHONY: setup build test test-unit test-integration test-bench coverage benchmark clean proto docker-* k8s-* docs backup lint-fix docs-format docs-check-format docs-check-links docs-validate restore js-setup docs-all docs-site-setup docs-site docs-serve docs-deploy-github docs-prepare-hosting lint-focused docs-fix-links openapi-gen openapi-validate openapi-diff sync-openapi update-doc-dates validate-doc-dates openapi-json-diff docs-generate-tests wasm-build frontend-build frontend-dev wasm-dev docker-wasm-build docker-wasm-up docker-wasm-down wasm-threaded serve-wasm helm-install helm-upgrade helm-uninstall helm-status helm-dry-run aws-validate aws-deploy aws-infrastructure aws-images aws-status aws-logs aws-scale aws-cleanup aws-info

# Variables
 BINARY_NAME=master-ovasabi
 DOCKER_IMAGE=ovasabi/$(BINARY_NAME)
 VERSION=$(shell git describe --tags --always --dirty)
 DOCKER_COMPOSE=COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 COMPOSE_BAKE=true docker-compose -f deployments/docker/docker-compose.yml
 KUBECTL=kubectl
 K8S_NAMESPACE=ovasabi
 K8S_CONTEXT=arn:aws:eks:$(AWS_REGION):$(AWS_ACCOUNT_ID):cluster/$(K8S_NAMESPACE)
 K8S_KUBERNETES_PATH=deployments/kubernetes

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

## Python proto generation parameters (output in internal/ai/python/protos for clear separation)
PY_PROTO_OUT=internal/ai/python
PY_PROTO_FILES=api/protos/ai/v1/model.proto api/protos/common/v1/metadata.proto api/protos/common/v1/orchestration.proto api/protos/common/v1/entity.proto api/protos/common/v1/patterns.proto api/protos/common/v1/payload.proto api/protos/nexus/v1/nexus.proto

# Setup development environment
setup: install-tools py-proto
	$(GOMOD) download
	$(GOMOD) tidy

# Install required tools
install-tools:
	$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Build the binary
build: proto
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) ./cmd/server

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


# Only rebuild the builder image if go.mod, go.sum, or Dockerfile.builder change
ovasabi-go-builder: go.mod go.sum deployments/docker/Dockerfile.builder
	docker build -f deployments/docker/Dockerfile.builder -t ovasabi-go-builder:latest .

.PHONY: ovasabi-go-builder

# docker-build depends on the builder image target, not always running the build
docker-build: ovasabi-go-builder proto check-service-registration
	$(DOCKER_COMPOSE) build

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

# --- TypeScript Proto Codegen for Frontend ---
# Usage: make ts-proto
ts-proto:
	@echo "Generating TypeScript code from protobufs for frontend (using ts-proto)..."
	@cd frontend && yarn install --silent
	@mkdir -p frontend/protos
	@for proto_dir in $(shell find $(PROTO_PATH) -mindepth 1 -maxdepth 1 -type d); do \
		latest_version_dir=$$(ls -d $$proto_dir/v*/ 2>/dev/null | sort -V | tail -n 1); \
		if [ -d "$$latest_version_dir" ]; then \
			echo "Processing protos in $$latest_version_dir..."; \
			for proto_file in $$(find $$latest_version_dir -name '*.proto'); do \
				npx -y protoc \
					-I=$(PROTO_PATH) \
					--plugin=protoc-gen-ts_proto=frontend/node_modules/.bin/protoc-gen-ts_proto \
					--ts_proto_out=frontend/protos \
					--ts_proto_opt=esModuleInterop=true,forceLong=string,useOptionals=messages \
					$$proto_file; \
			done; \
		fi \
	done
	@echo "TypeScript proto generation complete. Output in frontend/protos"

# Docker Compose Commands
docker-slim-all:
	$(MAKE) -f deployments/docker/docker-slim.mk docker-slim-all

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

docker-clean: check-service-registration
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

# Comprehensive Docker cleanup - removes orphaned volumes, build cache, and unused resources
docker-cleanup-all: check-service-registration
	@echo "🧹 Starting comprehensive Docker cleanup..."
	@echo "📊 Before cleanup:"
	@docker system df
	@echo ""
	@echo "🗑️  Removing orphaned volumes..."
	docker volume prune -f
	@echo "🗑️  Removing build cache (this will free the most space)..."
	docker builder prune -a -f
	@echo "🗑️  Removing unused images..."
	docker image prune -a -f
	@echo "🗑️  Removing unused containers..."
	docker container prune -f
	@echo "🗑️  Removing unused networks..."
	docker network prune -f
	@echo "🗑️  Removing docker-slim artifacts..."
	docker images | grep '\.slim$$' | awk '{print $$1 ":" $$2}' | xargs -r docker rmi || true
	@echo ""
	@echo "📊 After cleanup:"
	@docker system df
	@echo "✅ Docker cleanup complete!"

# Generate Go protobuf code only (no Python)
proto:
	@echo "Generating Go protobuf code for latest proto versions only..."
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
	@echo "Go protobuf code generation complete"
# This ensures generated Go files are placed alongside their proto files (in-place)

# Install dependencies
deps: install-tools
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	$(GOGET) -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	$(GOGET) -u github.com/go-swagger/go-swagger/cmd/swagger

# Generate Python protobufs for AI enrichment module (output in internal/ai/python/protos)
py-proto:
	@echo "Generating Python protobufs for AI and common contracts (Python output: $(PY_PROTO_OUT))..."
	@mkdir -p $(PY_PROTO_OUT)
	@for proto_file in $(PY_PROTO_FILES); do \
		python -m grpc_tools.protoc -I=$(PROTO_PATH) \
			--python_out=$(PY_PROTO_OUT) \
			--grpc_python_out=$(PY_PROTO_OUT) \
			$$proto_file; \
	done
	@echo "Ensuring __init__.py files for Python proto importability..."
	@find $(PY_PROTO_OUT) -type d \( -path '*/ai*' -o -path '*/common*' -o -path '*/nexus*' \) -exec touch {}/__init__.py \;
	@echo "Python protobuf generation complete. Output in $(PY_PROTO_OUT)"

# Amadeus Backup Commands
backup:
	@echo "Ensuring wasm/config directory exists and copying config/service_registration.json for WASM embedding..."
	@mkdir -p wasm/config
	@cp -f config/service_registration.json wasm/config/service_registration.json
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
	@echo "Master Ovasabi Build System"
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  setup              - Set up development environment"
	@echo "  build              - Build the application"
	@echo "  dev                - Run in development mode"
	@echo "  test               - Run all tests"
	@echo "  lint               - Run linters"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       - Build Docker images"
	@echo "  docker-up          - Start Docker containers"
	@echo "  docker-down        - Stop Docker containers"
	@echo ""
	@echo "Environment:"
	@echo "  validate-env       - Validate environment variables"
	@echo "  cleanup-config     - Clean up configuration files"
	@echo ""
	@echo "AWS Deployment:"
	@echo "  aws-ecr-deploy     - Build and push images to ECR"
	@echo ""
	@echo "Documentation:"
	@echo "  docs               - Generate documentation"

# Environment validation and cleanup
validate-env:
	@./scripts/validate-env.sh

cleanup-config:
	@./scripts/cleanup-config.sh

# AWS ECR deployment
aws-ecr-deploy:
	@./deployments/aws/deploy-ecr.sh

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
docs: docs-format docs-validate
	@echo "All documentation generated and validated."

docs-all: docs-format docs-validate docs-serve
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
	@tar -czf docs-site.tar.gz -C site .
docs-fix-links:
	@touch docs/assets/images/logo.svg
	@find docs -name "*.md" -exec yarn markdown-link-check {} \; || echo "Link issues found."
# Amadeus Knowledge Graph Evolution Analysis
	@echo "=========================================================="
	@echo ""
	@echo "-----------------------------"

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


# rest-point: Safely update all documentation, OpenAPI specs, lint code/docs, and back up the knowledge graph.
rest-point: docs-all
	-$(MAKE) lint
	$(MAKE) backup

# Run go vet for static analysis on all Go files
vet:
	go vet ./...

wasm-build:
	@./scripts/build-wasm.sh

wasm-threaded: wasm-build

# Start Vite dev server (ensure vite.config.js sets COOP/COEP headers for WASM threads)
vite-wasm:
	cd frontend && yarn dev
	@echo "[NOTE] If using WASM threads, ensure Vite is configured to set COOP/COEP headers. See vite.config.js example in docs."

# Knowledge Graph CLI Commands
.PHONY: kg-backup kg-list-backups kg-restore kg-clean kg-validate kg-sync-backup

# Create a backup of the knowledge graph (prunes and validates before saving)
kg-backup:
	@bin/kgcli backup --desc "Manual backup via Makefile"

# List all available knowledge graph backups
kg-list-backups:
	@bin/kgcli list-backups --format text

# Restore the main knowledge graph from a specified backup file
# Usage: make kg-restore BACKUP=amadeus/backups/knowledge_graph_YYYYMMDD_HHMMSS.json
kg-restore:
	@if [ -z "$(BACKUP)" ]; then \
		echo "Usage: make kg-restore BACKUP=amadeus/backups/knowledge_graph_YYYYMMDD_HHMMSS.json"; \
		exit 1; \
	fi; \
	bin/kgcli restore --path $(BACKUP)

# Clean/prune the knowledge graph (removes empty/obsolete fields)
kg-clean:
	@bin/kgcli clean

# Validate the knowledge graph schema and required fields
kg-validate:
	@bin/kgcli validate

# Sync the main knowledge graph from the latest backup
kg-sync-backup:
	@bin/kgcli sync-backup

# Help descriptions for KG CLI
kg-help:
	   @echo "Knowledge Graph CLI Commands:";
	   @echo "  kg-backup        - Create a backup of the knowledge graph (prunes and validates before saving)";
	   @echo "  kg-list-backups  - List all available knowledge graph backups";
	   @echo "  kg-restore       - Restore the main knowledge graph from a specified backup file (use BACKUP=...)";
	   @echo "  kg-clean         - Clean/prune the knowledge graph (removes empty/obsolete fields)";
	   @echo "  kg-validate      - Validate the knowledge graph schema and required fields";
	   @echo "  kg-sync-backup   - Sync the main knowledge graph from the latest backup";
	   @echo "  kg-describe      - Output a summary of the knowledge graph structure (agent/AI-friendly)";
	   @echo "  kg-validate-full - Perform deep validation of the knowledge graph (cross-references, etc)";
	   @echo "  kg-list-services - List all service names in the knowledge graph";
	   @echo "  kg-list-patterns - List all pattern names in the knowledge graph";
	   @echo "  kg-get-service   - Output a specific service by name (use NAME=...)";
	   @echo "  kg-get-pattern   - Output a specific pattern by name (use NAME=...)";
	   @echo "  kg-delete-service - Delete a service by name (use NAME=...)";
	   @echo "  kg-delete-pattern - Delete a pattern by name (use NAME=...)";
# Generate service registration config
generate-service-registration:
	go run cmd/service-registration/main.go

# Check that config/service_registration.json is a file (not a directory) before building Docker images
check-service-registration:
	@if [ ! -f config/service_registration.json ]; then \
	  echo "ERROR: config/service_registration.json must be a file, not a directory!"; \
	  exit 1; \
	fi

eks-create-cluster:
	eksctl create cluster --name $(K8S_NAMESPACE) --region $(AWS_REGION) --nodes 2 --node-type t3.medium

eks-delete-cluster:
	eksctl delete cluster --name $(K8S_NAMESPACE) --region $(AWS_REGION)

# Update kubeconfig for EKS
eks-update-kubeconfig:
	aws eks update-kubeconfig --region $(AWS_REGION) --name $(K8S_NAMESPACE)

# Get EKS cluster info
eks-info:
	aws eks describe-cluster --name $(K8S_NAMESPACE) --region $(AWS_REGION) --query "cluster.{name:name,endpoint:endpoint,status:status}"

# List EKS clusters
eks-list:
	aws eks list-clusters --region $(AWS_REGION)

# Get EKS nodes
eks-nodes:
	kubectl get nodes

# Set context to EKS
k8s-set-context-eks: eks-update-kubeconfig
	kubectl config use-context arn:aws:eks:$(AWS_REGION):$(AWS_ACCOUNT_ID):cluster/$(K8S_NAMESPACE)

# Create a new EKS nodegroup (edit parameters as needed)
eks-create-nodegroup:
	eksctl create nodegroup --cluster $(K8S_NAMESPACE) --region $(AWS_REGION) --name $(K8S_NAMESPACE) --node-type t3.medium --nodes 2 --nodes-min 1 --nodes-max 3

# Deploy to EKS (use correct namespace and context)
k8s-deploy-eks: k8s-create-namespace k8s-set-context-eks
	kubectl apply -f kubernetes/ -n $(K8S_NAMESPACE)

k8s-status-eks:
	kubectl get all -n $(K8S_NAMESPACE)

k8s-logs-eks:
	kubectl logs -f deployment/$(BINARY_NAME) -n $(K8S_NAMESPACE)

k8s-port-forward-eks:
	kubectl port-forward service/$(BINARY_NAME) 50051:50051 -n $(K8S_NAMESPACE)

# ======================
# AWS ECS Deployment
# ======================

# Validate environment before AWS deployment
aws-validate:
	@echo "🔍 Validating AWS deployment prerequisites..."
	@./scripts/validate-env.sh
	@echo "✅ Environment validation complete"

# Deploy to AWS ECS (one-command deployment)
aws-deploy: aws-validate
	@echo "🚀 Starting AWS ECS deployment..."
	@./deployments/aws/deploy-full.sh
	@echo "✅ AWS deployment complete!"

# Deploy just the infrastructure
aws-infrastructure:
	@echo "🏗️ Deploying AWS infrastructure..."
	@aws cloudformation deploy \
		--template-file deployments/aws/infrastructure.yaml \
		--stack-name master-ovasabi-infrastructure \
		--parameter-overrides Environment=production \
		--capabilities CAPABILITY_IAM \
		--region $(AWS_REGION)

# Build and push images to ECR
aws-images:
	@echo "🐳 Building and pushing images to ECR..."
	@./deployments/aws/deploy-ecr.sh

# Check AWS deployment status
aws-status:
	@echo "📊 AWS Deployment Status:"
	@echo "========================="
	@echo "Stack Status:"
	@aws cloudformation describe-stacks \
		--stack-name master-ovasabi-infrastructure \
		--query 'Stacks[0].StackStatus' \
		--output text --region $(AWS_REGION) 2>/dev/null || echo "Stack not found"
	@echo ""
	@echo "ECS Service Status:"
	@aws ecs describe-services \
		--cluster master-ovasabi-cluster \
		--services master-ovasabi-service \
		--query 'services[0].{Status:status,RunningCount:runningCount,DesiredCount:desiredCount}' \
		--output table --region $(AWS_REGION) 2>/dev/null || echo "Service not found"
	@echo ""
	@echo "Application URL:"
	@aws cloudformation describe-stacks \
		--stack-name master-ovasabi-infrastructure \
		--query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
		--output text --region $(AWS_REGION) 2>/dev/null | sed 's/^/http:\/\//' || echo "Load balancer not found"

# View AWS logs
aws-logs:
	@echo "📋 Viewing AWS CloudWatch logs..."
	@aws logs tail /ecs/master-ovasabi --follow --region $(AWS_REGION)

# Scale AWS service
aws-scale:
	@read -p "Enter desired task count (current: $$(aws ecs describe-services --cluster master-ovasabi-cluster --services master-ovasabi-service --query 'services[0].desiredCount' --output text --region $(AWS_REGION) 2>/dev/null || echo '0')): " count; \
	aws ecs update-service \
		--cluster master-ovasabi-cluster \
		--service master-ovasabi-service \
		--desired-count $$count \
		--region $(AWS_REGION)

# Clean up AWS resources
aws-cleanup:
	@echo "🗑️ Cleaning up AWS resources..."
	@echo "⚠️  This will delete ALL AWS infrastructure!"
	@read -p "Are you sure? Type 'DELETE' to confirm: " confirm; \
	if [ "$$confirm" = "DELETE" ]; then \
		aws cloudformation delete-stack \
			--stack-name master-ovasabi-infrastructure \
			--region $(AWS_REGION); \
		echo "🗑️ Deletion initiated. Stack will be removed in a few minutes."; \
	else \
		echo "❌ Deletion cancelled."; \
	fi

# Show AWS deployment info
aws-info:
	@echo "📋 AWS Deployment Information:"
	@echo "=============================="
	@echo "Region: $(AWS_REGION)"
	@echo "Account ID: $(AWS_ACCOUNT_ID)"
	@echo "Stack Name: master-ovasabi-infrastructure"
	@echo ""
	@echo "Quick Commands:"
	@echo "  Deploy:     make aws-deploy"
	@echo "  Status:     make aws-status"
	@echo "  Logs:       make aws-logs"
	@echo "  Scale:      make aws-scale"
	@echo "  Cleanup:    make aws-cleanup"
	@echo ""
	@echo "Documentation:"
	@echo "  Quick Start: deployments/aws/QUICK-START.md"
	@echo "  Full Guide:  deployments/aws/SETUP-GUIDE.md"