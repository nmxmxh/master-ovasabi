# Testing Guide

This document outlines the testing strategy and organization for the OVASABI project.

## Test Organization

### Unit Tests

- Located next to the code they test
- Follow the pattern `*_test.go`
- Test individual components in isolation
- Use mocks for external dependencies

Example:

```go
pkg/logger/
  ├── logger.go
  └── logger_test.go
```

### Integration Tests

Located in `test/integration/`:

- API endpoint tests
- Database integration tests
- External service integration tests
- End-to-end workflows

### Performance Tests

Located in `test/benchmarks/`:

- Service benchmarks
- API performance tests
- Load tests
- Stress tests

### Test Data

Located in `test/data/`:

- Fixtures
- Mock responses
- Test configurations
- Sample data files

## Test Coverage Requirements

### Critical Packages (95%+ coverage)

- `pkg/errors`
- `pkg/logger`
- `pkg/metrics`
- `pkg/health`
- `internal/service/*`

### Core Packages (80%+ coverage)

- `internal/server`
- `pkg/models`
- `pkg/utils`

### Supporting Packages (70%+ coverage)

- `cmd/*`
- `tools/*`

## Testing Standards

### Unit Tests

1. Test file naming: `package_test.go`
2. Test function naming: `Test<Function>_<Scenario>`
3. Use table-driven tests where appropriate
4. Mock external dependencies
5. One assertion per test case

Example:

```go
func TestLogger_Info_WithFields(t *testing.T) {
    // Test implementation
}
```

### Integration Tests

1. Use test containers for dependencies
2. Clean up test data after each test
3. Run tests in isolation
4. Use realistic test data
5. Test complete workflows

### Benchmark Tests

1. Include baseline measurements
2. Test with varying loads
3. Document performance expectations
4. Include cleanup in timing
5. Use realistic data sizes

## Test Helpers

### Common Test Utilities

Located in `test/utils/`:

- Mock implementations
- Test fixtures
- Helper functions
- Assert functions

### Example Test Helper

```go
package utils

import "testing"

func SetupTestDB(t *testing.T) (*sql.DB, func()) {
    // Setup code
    return db, cleanup
}
```

## Running Tests

### All Tests

```bash
make test
```

### Unit Tests Only

```bash
make test-unit
```

### Integration Tests

```bash
make test-integration
```

### Benchmarks

```bash
make bench
```

## CI/CD Integration

### Pull Request Checks

- All unit tests must pass
- Integration tests must pass
- Coverage must not decrease
- Benchmarks must not degrade

### Performance Monitoring

- Track benchmark results over time
- Alert on significant regressions
- Store historical performance data
