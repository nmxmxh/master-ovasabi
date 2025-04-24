# Protocol Buffers API Definitions

This directory contains the gRPC service definitions using Protocol Buffers.

## Directory Structure

```go
.
├── auth/       # Authentication and authorization services
├── broadcast/  # Broadcasting and notification services
├── i18n/       # Internationalization services
├── quotes/     # Quote management services
└── referral/   # Referral system services
```

## Service Domains

### Authentication (`auth/`)

- User authentication
- Token management
- Permission control

### Broadcasting (`broadcast/`)

- Push notifications
- Real-time updates
- Event broadcasting

### Internationalization (`i18n/`)

- Language management
- Translation services
- Locale handling

### Quotes (`quotes/`)

- Quote creation and management
- Quote retrieval
- Quote search and filtering

### Referral (`referral/`)

- Referral code management
- Reward tracking
- Referral analytics

## Development Guidelines

1. **Proto File Organization**
   - One service per file
   - Related messages in the same file
   - Clear package naming

2. **Versioning**
   - Use semantic versioning
   - Maintain backward compatibility
   - Document breaking changes

3. **Documentation**
   - Document all services and RPCs
   - Include usage examples
   - Explain error codes

4. **Best Practices**
   - Use well-defined types
   - Follow Protocol Buffers style guide
   - Keep services focused and cohesive

## Generation

To regenerate the Go code from proto files:

```bash
make proto
```

See the `tools/protoc/` directory for protoc configuration and plugins.
