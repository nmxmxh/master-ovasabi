# Configuration

This directory contains configuration files for different environments.

## Environment Configurations

- `dev.yaml`: Development environment configuration
- `prod.yaml`: Production environment configuration
- `test.yaml`: Test environment configuration

## Configuration Structure

Each configuration file follows this structure:

```yaml
server:
  host: string       # Server host address
  port: number      # Server port
  debug: boolean    # Debug mode flag

database:
  host: string      # Database host
  port: number      # Database port
  name: string      # Database name
  user: string      # Database user
  password: string  # Database password (use env var in prod)

logging:
  level: string     # Logging level (debug/info/warn/error)
  format: string    # Log format (json/console)

auth:
  jwt_secret: string # JWT signing secret (use env var in prod)
  token_expiry: string # Token expiry duration

api:
  rate_limit: number # API rate limit (requests per minute)
  timeout: string    # API timeout duration

feature_flags:
  enable_new_features: boolean # Toggle new features
  maintenance_mode: boolean    # Toggle maintenance mode
```

## Environment Variables

The following environment variables are used in production:

- `DB_HOST`: Database host address
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `JWT_SECRET`: JWT signing secret

## Usage

1. For local development:

   ```bash
   export CONFIG_FILE=config/dev.yaml
   ```

2. For production:\

   ```bash
   export CONFIG_FILE=config/prod.yaml
   # Set required environment variables
   export DB_HOST=your-db-host
   export DB_USER=your-db-user
   export DB_PASSWORD=your-db-password
   export JWT_SECRET=your-jwt-secret
   ```

3. For testing:

   ```bash
   export CONFIG_FILE=config/test.yaml
   ```

## Best Practices

1. Never commit sensitive values directly in configuration files
2. Use environment variables for secrets in production
3. Keep development and test configurations simple
4. Document all configuration changes
5. Version control all configuration files except for local overrides
