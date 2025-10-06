# Makefile Quick Reference

## Common Development Workflow

### 🚀 Starting Development

```bash
# Full setup from scratch
make setup                 # Install dependencies
make proto                # Generate protobuf code
make docker-build         # Build Docker images
make docker-up            # Start all services
```

### 🔧 Day-to-day Development

```bash
# Check service status
make docker-ps

# View logs
make docker-logs

# Restart services (when ws-gateway crashes)
make docker-restart

# Restart just the main app
make docker-restart-app
```

### 🐛 Debugging Compute Issues

```bash
# When ws-gateway is crashing/overloaded:
make docker-restart         # Restart all services
make docker-logs           # Check logs for errors

# Individual service management:
docker compose -f deployments/docker/docker-compose.yml restart ws-gateway
docker compose -f deployments/docker/docker-compose.yml logs ws-gateway
```

### 🧪 Testing & Quality

```bash
make test                  # Run all tests
make lint                  # Check code quality
make lint-fix             # Apply auto-fixes
```

### 🌐 Frontend Development

```bash
make wasm-build           # Build WASM components
make vite-wasm            # Start Vite dev server (frontend)
make ts-proto             # Generate TypeScript protos
make js-setup             # Install JavaScript dependencies
```

### 🧹 Cleanup

```bash
make docker-clean         # Clean Docker resources
make docker-cleanup-all   # Comprehensive cleanup
make clean                # Clean build artifacts
```

### 📚 Documentation

```bash
make docs                 # Generate and validate all docs
make docs-serve           # Serve docs locally (with hot reload)
make docs-format          # Format markdown files
make docs-deploy-github   # Deploy docs to GitHub Pages
```

### ☁️ AWS Deployment

```bash
make aws-validate         # Validate AWS credentials and config
make aws-deploy           # Deploy to AWS ECS
make aws-status           # Check deployment status
make aws-logs             # View AWS service logs
make aws-cleanup          # Remove AWS resources
make aws-infrastructure   # Setup AWS infrastructure
```

### 🗄️ Knowledge Graph Management

```bash
make kg-backup            # Backup knowledge graph
make kg-restore           # Restore from backup
make kg-list-backups      # List available backups
make kg-validate          # Validate knowledge graph
make kg-clean             # Clean knowledge graph cache
```

### 🛠️ Advanced Development

```bash
make new-service          # Generate new service scaffold
make backup               # Create full system backup
make restore              # Restore from backup
make deps                 # Update dependencies
make lint-focused         # Focused linting (faster)
make coverage             # Generate test coverage report
make benchmark            # Run performance benchmarks
```

## 🚨 Emergency Recovery

### When compute operations are failing

1. `make docker-ps` - Check which services are down
2. `make docker-restart` - Restart all services
3. `make docker-logs` - Check for startup errors
4. If still failing: `make docker-down && make docker-up`

### When builds are failing

1. `make clean` - Clean build artifacts
2. `make docker-clean` - Clean Docker cache
3. `make setup` - Reinstall dependencies
4. `make docker-build` - Rebuild images

## 📋 Service-Specific Commands

### ws-gateway (WebSocket Gateway for Compute)

- **Status**: `docker compose -f deployments/docker/docker-compose.yml ps ws-gateway`
- **Logs**: `docker compose -f deployments/docker/docker-compose.yml logs ws-gateway`
- **Restart**: `docker compose -f deployments/docker/docker-compose.yml restart ws-gateway`

### nexus (Event Bus)

- **Status**: `docker compose -f deployments/docker/docker-compose.yml ps nexus`
- **Logs**: `docker compose -f deployments/docker/docker-compose.yml logs nexus`

### Database Operations

- **Migrations**: `make migrate-up ARGS="up"`
- **Migrations Down**: `make migrate-up ARGS="down 1"`

### Kubernetes/EKS Operations

- **Create EKS Cluster**: `make eks-create-cluster`
- **Delete EKS Cluster**: `make eks-delete-cluster`
- **Update Kubeconfig**: `make eks-update-kubeconfig`
- **Deploy to EKS**: `make k8s-deploy-eks`
- **Check EKS Status**: `make k8s-status-eks`
- **EKS Logs**: `make k8s-logs-eks`

### Service Registration

- **Generate Registration**: `make generate-service-registration`
- **Check Registration**: `make check-service-registration`

## 🔧 Configuration

Always use the Makefile commands instead of direct Docker commands because:

- ✅ Proper dependency management
- ✅ Environment variable handling
- ✅ Build optimization
- ✅ Service registration validation
- ✅ Consistent error handling

## 💡 Pro Tips

1. **Always check service health**: `make docker-ps` before debugging
2. **Use build cache**: `make docker-build` is optimized for incremental builds
3. **Clean regularly**: `make docker-clean` when switching branches
4. **Check logs first**: `make docker-logs` shows all service logs together
5. **Restart services in order**: `make docker-restart` handles dependencies properly

## 🆘 When Things Go Wrong

**"ws-gateway is sending compute and crashing"**:

```bash
make docker-restart    # Restart all services
make docker-logs       # Check what caused the crash
```

**"Build failing"**:

```bash
make clean
make docker-clean
make setup
make docker-build
```

**"Frontend not connecting to WASM"**:

```bash
make wasm-build
make vite-wasm
# Check CORS/WebSocket connection in browser console
```
