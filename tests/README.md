# Rideshare Platform Testing Infrastructure

This directory contains all test suites for the platform:
- Unit tests for each microservice
- Integration tests for service interactions
- Contract tests for gRPC interfaces
- End-to-end scenario tests
- Load and performance tests

## How to Run All Tests

```bash
# Run all unit and integration tests
make test

# Run end-to-end tests
make e2e

# Run load tests
make load-test
```

See individual test files for details.
