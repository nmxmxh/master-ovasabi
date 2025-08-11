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
