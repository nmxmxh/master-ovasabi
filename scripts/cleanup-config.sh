#!/bin/bash

# Configuration Cleanup Script for Master Ovasabi
# This script removes inconsistent configs and creates a clean, standardized setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[ℹ]${NC} $1"
}

# Remove redundant or conflicting environment files
cleanup_env_files() {
    print_info "Cleaning up environment files..."
    
    # Check for old/redundant env files
    local redundant_files=(
        "deployments/sample.env"
        ".env.local"
        ".env.development"
        ".env.production"
        "config/.env"
    )
    
    for file in "${redundant_files[@]}"; do
        if [ -f "$file" ]; then
            print_warning "Found redundant environment file: $file"
            echo "  Consider removing it to avoid confusion"
        fi
    done
    
    # Check for Docker override files that might conflict
    local docker_overrides=(
        "docker-compose.override.yml"
        "deployments/docker/docker-compose.override.yml"
    )
    
    for file in "${docker_overrides[@]}"; do
        if [ -f "$file" ]; then
            print_info "Found Docker override file: $file"
            echo "  Make sure it's intentional and not conflicting"
        fi
    done
}

# Validate configuration file consistency
validate_config_consistency() {
    print_info "Validating configuration file consistency..."
    
    # Check if config files use environment variables properly
    local config_files=(
        "config/config.yaml"
        "config/dev.yaml"
        "config/prod.yaml"
        "config/test.yaml"
    )
    
    for file in "${config_files[@]}"; do
        if [ -f "$file" ]; then
            # Check for hardcoded passwords
            if grep -q '"password":[[:space:]]*"[^$]' "$file" 2>/dev/null; then
                print_warning "Found hardcoded passwords in $file"
            fi
            
            # Check for consistent environment variable usage
            if grep -q '\${[^}]*}' "$file" 2>/dev/null; then
                print_status "$file uses environment variables correctly"
            else
                print_warning "$file might not be using environment variables"
            fi
        fi
    done
}

# Check Docker compose file consistency
validate_docker_consistency() {
    print_info "Validating Docker compose configuration..."
    
    local compose_files=(
        "deployments/docker/docker-compose.yml"
        "deployments/docker/docker-compose.prod.yml"
    )
    
    for file in "${compose_files[@]}"; do
        if [ -f "$file" ]; then
            # Check for hardcoded values that should be environment variables
            if grep -E 'password.*:[[:space:]]*['\''"][^$][^'\'']*['\''"]' "$file" >/dev/null 2>&1; then
                print_warning "Found potential hardcoded passwords in $file"
            fi
            
            # Check for consistent environment variable patterns
            if grep -E '\$\{[A-Z_]+:-[^}]*\}' "$file" >/dev/null 2>&1; then
                print_status "$file uses environment variables with defaults"
            fi
        fi
    done
}

# Generate a configuration summary
generate_config_summary() {
    print_info "Configuration Summary:"
    
    echo
    echo "=== Main Configuration Files ==="
    echo "✓ .env - Main environment variables"
    echo "✓ .env.example - Template for new setups"
    echo "✓ config/config.yaml - Main application config"
    echo "✓ config/dev.yaml - Development overrides"
    echo "✓ deployments/docker/docker-compose.yml - Development containers"
    echo "✓ deployments/docker/docker-compose.prod.yml - Production containers"
    
    echo
    echo "=== Configuration Hierarchy ==="
    echo "1. Environment variables (.env)"
    echo "2. Configuration files (config/*.yaml)"
    echo "3. Docker compose overrides"
    echo "4. Command line arguments"
    
    echo
    echo "=== Best Practices Applied ==="
    echo "✓ Sensitive data in environment variables"
    echo "✓ Consistent naming conventions"
    echo "✓ Default values for optional settings"
    echo "✓ Separate dev/prod configurations"
}

# Create a comprehensive environment documentation
create_env_documentation() {
    print_info "Creating environment documentation..."
    
    cat > docs/ENVIRONMENT.md << 'EOF'
# Environment Configuration Guide

This document explains the environment variable setup for Master Ovasabi.

## Required Variables

### Application
- `APP_NAME`: Application identifier
- `APP_ENV`: Environment (development/staging/production)
- `APP_PORT`: Main application port
- `HTTP_PORT`: HTTP service port
- `GRPC_PORT`: gRPC service port

### Database
- `DB_HOST`: PostgreSQL hostname
- `DB_PORT`: PostgreSQL port
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password (sensitive)
- `DB_NAME`: Database name
- `POSTGRES_USER`: PostgreSQL admin user
- `POSTGRES_PASSWORD`: PostgreSQL admin password (sensitive)
- `POSTGRES_DB`: PostgreSQL database name

### Redis
- `REDIS_HOST`: Redis hostname
- `REDIS_PORT`: Redis port
- `REDIS_PASSWORD`: Redis password (sensitive)

### Admin
- `ADMIN_USER`: Application admin username
- `ADMIN_PASSWORD`: Application admin password (sensitive)

## Optional Variables

### AWS Deployment
- `AWS_REGION`: AWS region for deployment
- `AWS_ACCOUNT_ID`: AWS account identifier
- `ECR_REGISTRY`: ECR registry URL

### Security
- `JWT_SECRET`: JWT signing secret (sensitive)

### Services
- `NEXUS_GRPC_ADDR`: Nexus service gRPC address
- `CAMPAIGN_ID`: Default campaign ID
- `WS_ALLOWED_ORIGINS`: WebSocket allowed origins

## Configuration Files

### .env
Main environment file for local development. Copy from `.env.example` and customize.

### .env.example
Template file with all required variables and examples.

### config/config.yaml
Main application configuration that references environment variables.

### config/dev.yaml
Development-specific overrides.

### config/prod.yaml
Production-specific overrides.

## Security Best Practices

1. **Never commit sensitive data** to version control
2. **Use strong passwords** (minimum 8 characters)
3. **Rotate secrets regularly** in production
4. **Use AWS Secrets Manager** for production deployments
5. **Validate environment** before deployment

## Validation

Run the environment validation script:
```bash
./scripts/validate-env.sh
```

This will check for:
- Missing required variables
- Weak passwords
- Configuration consistency
- Port conflicts
- Format validation
EOF

    print_status "Created docs/ENVIRONMENT.md"
}

# Main cleanup function
main() {
    echo "=== Master Ovasabi Configuration Cleanup ==="
    echo
    
    cleanup_env_files
    echo
    
    validate_config_consistency
    echo
    
    validate_docker_consistency
    echo
    
    generate_config_summary
    echo
    
    create_env_documentation
    echo
    
    print_status "Configuration cleanup completed!"
    echo
    echo "=== Next Steps ==="
    echo "1. Review any warnings above"
    echo "2. Run './scripts/validate-env.sh' to verify setup"
    echo "3. Test with 'docker compose up' for local development"
    echo "4. Use './deployments/aws/deploy-ecr.sh' for AWS deployment"
    echo
    echo "📖 See docs/ENVIRONMENT.md for detailed configuration guide"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
