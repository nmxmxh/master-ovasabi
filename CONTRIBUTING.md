# Contributing to INOS

Thank you for your interest in contributing! We welcome code, documentation, ideas, and feedback
from everyone.

## How to Contribute

- **Issues:** Report bugs, suggest features, or ask questions via GitHub Issues.
- **Pull Requests:** Fork the repo, create a branch, and submit a pull request. Please describe your
  changes clearly.
- **Code Style:** Follow Go best practices. Use clear, descriptive names (e.g., `fileStructure` not
  `file_structure`).
- **Documentation:** Update or add docs in the `docs/` directory as needed.

## Community Spirit

- Please read the [Manifesto and Advice](docs/amadeus/manifesto.md) for our philosophy and
  expectations.
- Be kind, inclusive, and constructive.
- Celebrate all contributions—code, art, feedback, and support.

## Questions?

Open an issue or start a discussion. We're here to help!

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
