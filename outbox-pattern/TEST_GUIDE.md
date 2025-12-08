# Test Guide for Outbox Pattern Implementation

This guide explains how to run the comprehensive test suite for the outbox pattern implementation.

## Test Coverage

The test suite includes:

### 1. Unit Tests (Go)
- **relay service** (`cmd/relay/main_test.go`): 14 test functions + 1 benchmark
- **webhook-consumer service** (`cmd/webhook-consumer/main_test.go`): 11 test functions + 1 benchmark

### 2. Configuration Validation Tests
- **Dockerfile tests** (relay and webhook-consumer)
- **Docker Compose validation**
- **SQL migration validation**

## Prerequisites

```bash
# Install Go dependencies
cd outbox-pattern
go mod tidy

# Install test dependencies (if not already installed)
go get github.com/DATA-DOG/go-sqlmock@v1.5.2
go get github.com/stretchr/testify@v1.11.1
```

## Running Unit Tests

### Run all unit tests
```bash
cd outbox-pattern
go test ./cmd/... -v
```

### Run tests with coverage
```bash
go test ./cmd/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run specific test
```bash
# Test relay service
go test ./cmd/relay -v -run TestGetEnv

# Test webhook consumer
go test ./cmd/webhook-consumer -v -run TestFinishOrderHandler
```

### Run benchmarks
```bash
go test ./cmd/webhook-consumer -bench=. -benchmem
```

## Running Configuration Validation Tests

### Validate Dockerfiles
```bash
# Relay service Dockerfile
./cmd/relay/Dockerfile_test.sh

# Webhook consumer Dockerfile  
./cmd/webhook-consumer/Dockerfile_test.sh
```

### Validate Docker Compose
```bash
./docker-compose_test.sh
```

### Validate SQL Migrations
```bash
./migrations/migration_test.sh
```

### Run all validation tests
```bash
./cmd/relay/Dockerfile_test.sh && \
./cmd/webhook-consumer/Dockerfile_test.sh && \
./docker-compose_test.sh && \
./migrations/migration_test.sh && \
echo "✓ All validation tests passed!"
```

## Test Structure

### Relay Service Tests (`cmd/relay/main_test.go`)

1. **TestGetEnv** - Environment variable handling
2. **TestExtractOutbox** - Outbox data extraction from Debezium events
3. **TestRelayServerCallWebhook** - Webhook calling functionality
4. **TestRelayServerCallWebhookNetworkError** - Network error handling
5. **TestDebeziumHandler** - Main CDC event handler
6. **TestDebeziumHandlerInvalidJSON** - Invalid JSON handling
7. **TestDebeziumHandlerReadBodyError** - Body reading errors
8. **TestDebeziumHandlerWebhookFailure** - Webhook failure scenarios
9. **TestDebeziumHandlerMissingOrderIDInPayload** - Missing data handling
10. **TestExtractOutboxEdgeCases** - Edge cases in extraction
11. **TestRelayServerCallWebhookConcurrency** - Concurrent webhook calls
12. **TestDebeziumChangeJSONSerialization** - JSON marshaling
13. **TestOutboxJSONSerialization** - Outbox struct serialization
14. **TestRelayServerCallWebhookTimeout** - Timeout handling

### Webhook Consumer Tests (`cmd/webhook-consumer/main_test.go`)

1. **TestGetEnv** - Environment variable handling
2. **TestFinishOrderHandler** - Order completion handler
3. **TestFinishOrderHandlerInvalidJSON** - Invalid JSON handling
4. **TestFinishOrderHandlerEmptyBody** - Empty body handling
5. **TestFinishOrderHandlerRowsAffectedError** - Database error handling
6. **TestOrderFinishRequestValidation** - Request validation
7. **TestFinishOrderHandlerConcurrency** - Concurrent requests
8. **TestOrderFinishRequestJSONSerialization** - JSON handling
9. **TestFinishOrderHandlerDifferentContentTypes** - Content type handling
10. **TestFinishOrderHandlerNilDatabase** - Nil database handling
11. **TestFinishOrderHandlerMultipleRowsAffected** - Multiple rows scenario

## Test Scenarios Covered

### Happy Path
- ✓ Successful CDC event processing
- ✓ Webhook calls succeed
- ✓ Database updates succeed
- ✓ Order status updates

### Edge Cases
- ✓ Empty/nil values
- ✓ Very large IDs (int64 max)
- ✓ Special characters in strings
- ✓ Long strings
- ✓ Invalid timestamps
- ✓ Multiple rows affected

### Error Handling
- ✓ Invalid JSON
- ✓ Missing required fields
- ✓ Database connection errors
- ✓ Network errors
- ✓ Timeout scenarios
- ✓ HTTP error responses (4xx, 5xx)
- ✓ Read errors

### Concurrency
- ✓ Concurrent webhook calls
- ✓ Concurrent HTTP requests
- ✓ Thread safety

### Configuration Validation
- ✓ Dockerfile best practices
- ✓ Multi-stage builds
- ✓ Security (CGO disabled, minimal base images)
- ✓ Docker Compose service dependencies
- ✓ Health checks
- ✓ Network configuration
- ✓ Environment variables
- ✓ PostgreSQL CDC configuration
- ✓ Debezium setup
- ✓ SQL migration idempotency

## Continuous Integration

Add to your CI pipeline:

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Install dependencies
        run: cd outbox-pattern && go mod download
      - name: Run unit tests
        run: cd outbox-pattern && go test ./cmd/... -v -cover
      - name: Run validation tests
        run: |
          cd outbox-pattern
          ./cmd/relay/Dockerfile_test.sh
          ./cmd/webhook-consumer/Dockerfile_test.sh
          ./docker-compose_test.sh
          ./migrations/migration_test.sh
```

## Test Metrics

- **Total Unit Tests**: 25+ test functions
- **Total Test Lines**: 1,563 lines
- **Coverage Target**: >80%
- **Validation Scripts**: 4 comprehensive scripts
- **Test Scenarios**: 50+ individual test cases

## Mocking Strategy

The tests use:
- **go-sqlmock** for database mocking
- **httptest** for HTTP server mocking
- **testify/assert** for assertions
- **testify/require** for required checks

## Best Practices

1. **Isolation**: Each test is independent
2. **Cleanup**: Resources are properly cleaned up
3. **Descriptive Names**: Test names clearly indicate purpose
4. **Coverage**: Happy paths, edge cases, and errors
5. **Fast**: Tests run quickly without external dependencies
6. **Deterministic**: No flaky tests

## Troubleshooting

### Tests fail with "module not found"
```bash
cd outbox-pattern
go mod tidy
```

### Validation scripts fail with "permission denied"
```bash
chmod +x cmd/relay/Dockerfile_test.sh
chmod +x cmd/webhook-consumer/Dockerfile_test.sh
chmod +x docker-compose_test.sh
chmod +x migrations/migration_test.sh
```

### Python not found for YAML validation
```bash
# Install Python 3 and PyYAML
sudo apt-get install python3 python3-yaml
```

## Contributing

When adding new features:
1. Write tests first (TDD)
2. Ensure >80% code coverage
3. Add validation tests for config changes
4. Update this guide with new test information