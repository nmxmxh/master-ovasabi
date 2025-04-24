# Contributing to OVASABI

This document describes the project structure and guidelines for contributing to the codebase.

## Project Structure

```go
.
├── api/           # Public API definitions and documentation
│   ├── docs/      # API documentation
│   ├── openapi/   # OpenAPI/Swagger specifications
│   └── protos/    # Protocol buffer definitions
├── cmd/           # Main applications of the project
├── config/        # Configuration files and templates
├── deployments/   # Deployment configurations and scripts
├── docs/          # Project documentation
├── internal/      # Private application and library code
├── pkg/           # Public library code
├── test/          # Integration and performance tests
└── tools/         # Project tools and utilities
```

## Directory Guidelines

### `/api`

- Contains all external API definitions
- Each API version should have its own directory
- Include OpenAPI/Swagger documentation

### `/cmd`

- Main applications for this project
- Directory name should match the executable name
- Keep the code minimal - only wire up components

### `/internal`

- Private code that you don't want others importing
- Includes:
  - `config/`: Internal configuration code
  - `middleware/`: HTTP/gRPC middleware
  - `repository/`: Data access layer
  - `service/`: Business logic
  - `server/`: Server implementations

### `/pkg`

- Code that's safe to use as libraries
- Should have stable APIs
- Other projects should be able to import these

### `/test`

- Integration and performance tests
- Test data and test utilities
- Unit tests should stay with the code they test

## Development Guidelines

1. **Code Organization**
   - Keep packages focused and cohesive
   - Follow standard Go project layout
   - Use meaningful package names

2. **Testing**
   - Write unit tests alongside the code
   - Integration tests go in `/test`
   - Aim for high test coverage

3. **Documentation**
   - Document all exported functions and types
   - Keep READMEs up to date
   - Use examples in documentation

4. **Configuration**
   - Use environment variables for secrets
   - Keep configuration files in `/config`
   - Document all configuration options

## Getting Started

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `make test`
4. Start development server: `make dev`

For more detailed information, see the documentation in `/docs`.
