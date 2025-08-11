#!/bin/bash

# Environment Variables Validation Script for Master Ovasabi
# This script validates that all required environment variables are set and consistent

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

# Load environment variables from .env if it exists
load_env() {
    if [ -f ".env" ]; then
        print_info "Loading environment variables from .env"
        set -a
        source .env
        set +a
    else
        print_error ".env file not found. Please create one based on .env.example"
        exit 1
    fi
}

# Required environment variables
REQUIRED_VARS=(
    "APP_NAME"
    "APP_ENV"
    "DB_HOST"
    "DB_PORT"
    "DB_USER"
    "DB_PASSWORD"
    "DB_NAME"
    "POSTGRES_USER"
    "POSTGRES_PASSWORD"
    "POSTGRES_DB"
    "REDIS_HOST"
    "REDIS_PORT"
    "REDIS_PASSWORD"
    "ADMIN_USER"
    "ADMIN_PASSWORD"
    "HTTP_PORT"
    "GRPC_PORT"
    "NEXUS_GRPC_ADDR"
)

# Optional but recommended variables
OPTIONAL_VARS=(
    "AWS_REGION"
    "AWS_ACCOUNT_ID"
    "ECR_REGISTRY"
    "JWT_SECRET"
    "LOG_LEVEL"
    "CAMPAIGN_ID"
    "WS_ALLOWED_ORIGINS"
)

# Sensitive variables that should not be empty or default
SENSITIVE_VARS=(
    "DB_PASSWORD"
    "REDIS_PASSWORD"
    "ADMIN_PASSWORD"
    "JWT_SECRET"
)

check_required_vars() {
    print_info "Checking required environment variables..."
    local missing_vars=()
    
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
            print_error "$var is not set"
        else
            print_status "$var is set"
        fi
    done
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        print_error "Missing required variables: ${missing_vars[*]}"
        print_info "Please set these variables in your .env file"
        return 1
    fi
    
    return 0
}

check_optional_vars() {
    print_info "Checking optional environment variables..."
    
    for var in "${OPTIONAL_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            print_warning "$var is not set (optional)"
        else
            print_status "$var is set"
        fi
    done
}

check_sensitive_vars() {
    print_info "Checking sensitive variables..."
    local weak_vars=()
    
    for var in "${SENSITIVE_VARS[@]}"; do
        local value="${!var}"
        if [ -z "$value" ]; then
            print_error "$var is empty"
            weak_vars+=("$var")
        elif [ "$value" = "password" ] || [ "$value" = "secret" ] || [ "$value" = "admin" ]; then
            print_warning "$var uses a weak default value"
            weak_vars+=("$var")
        elif [ ${#value} -lt 8 ]; then
            print_warning "$var is too short (less than 8 characters)"
            weak_vars+=("$var")
        else
            print_status "$var appears secure"
        fi
    done
    
    if [ ${#weak_vars[@]} -gt 0 ]; then
        print_warning "Weak or missing sensitive variables: ${weak_vars[*]}"
        print_info "Consider using stronger passwords and secrets"
    fi
}

check_database_consistency() {
    print_info "Checking database variable consistency..."
    
    # Check if DB_* and POSTGRES_* variables match
    if [ "$DB_USER" != "$POSTGRES_USER" ]; then
        print_error "DB_USER ($DB_USER) != POSTGRES_USER ($POSTGRES_USER)"
        return 1
    fi
    
    if [ "$DB_PASSWORD" != "$POSTGRES_PASSWORD" ]; then
        print_error "DB_PASSWORD != POSTGRES_PASSWORD"
        return 1
    fi
    
    if [ "$DB_NAME" != "$POSTGRES_DB" ]; then
        print_error "DB_NAME ($DB_NAME) != POSTGRES_DB ($POSTGRES_DB)"
        return 1
    fi
    
    print_status "Database variables are consistent"
}

check_port_conflicts() {
    print_info "Checking for port conflicts..."
    
    local ports=("$HTTP_PORT" "$GRPC_PORT" "$DB_PORT" "$REDIS_PORT")
    local unique_ports=($(printf "%s\n" "${ports[@]}" | sort -u))
    
    if [ ${#ports[@]} -ne ${#unique_ports[@]} ]; then
        print_error "Port conflicts detected in: ${ports[*]}"
        return 1
    fi
    
    print_status "No port conflicts detected"
}

check_url_formats() {
    print_info "Checking URL formats..."
    
    # Check DATABASE_URL format
    if [[ ! "$DATABASE_URL" =~ ^postgres://.*@.*:.*/.*\?.*$ ]]; then
        print_warning "DATABASE_URL format may be incorrect: $DATABASE_URL"
    else
        print_status "DATABASE_URL format looks correct"
    fi
    
    # Check NEXUS_GRPC_ADDR format
    if [[ ! "$NEXUS_GRPC_ADDR" =~ ^.*:[0-9]+$ ]]; then
        print_warning "NEXUS_GRPC_ADDR format may be incorrect: $NEXUS_GRPC_ADDR"
    else
        print_status "NEXUS_GRPC_ADDR format looks correct"
    fi
}

validate_aws_config() {
    print_info "Checking AWS configuration..."
    
    if [ -n "$AWS_ACCOUNT_ID" ] && [ -n "$AWS_REGION" ]; then
        if [ -z "$ECR_REGISTRY" ]; then
            local expected_ecr="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
            print_warning "ECR_REGISTRY not set. Expected: $expected_ecr"
        else
            print_status "AWS configuration appears complete"
        fi
    else
        print_info "AWS configuration is incomplete (optional for local development)"
    fi
}

generate_summary() {
    print_info "Environment Summary:"
    echo "  Application: $APP_NAME ($APP_ENV)"
    echo "  Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
    echo "  Redis: $REDIS_HOST:$REDIS_PORT"
    echo "  Admin User: $ADMIN_USER"
    echo "  Ports: HTTP=$HTTP_PORT, gRPC=$GRPC_PORT"
    if [ -n "$AWS_REGION" ]; then
        echo "  AWS: $AWS_REGION (Account: ${AWS_ACCOUNT_ID:-not set})"
    fi
}

main() {
    echo "=== Master Ovasabi Environment Validation ==="
    echo
    
    load_env
    
    local errors=0
    
    check_required_vars || ((errors++))
    echo
    
    check_optional_vars
    echo
    
    check_sensitive_vars
    echo
    
    check_database_consistency || ((errors++))
    echo
    
    check_port_conflicts || ((errors++))
    echo
    
    check_url_formats
    echo
    
    validate_aws_config
    echo
    
    generate_summary
    echo
    
    if [ $errors -eq 0 ]; then
        print_status "Environment validation completed successfully!"
        echo "Your environment is ready for deployment."
    else
        print_error "Environment validation failed with $errors error(s)."
        echo "Please fix the issues above before proceeding."
        exit 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
