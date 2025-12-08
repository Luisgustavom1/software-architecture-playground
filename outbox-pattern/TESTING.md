# Testing the Outbox Pattern Implementation

## Quick Start

Run all tests with a single command:

```bash
./run_all_tests.sh
```

This will execute:
- ✓ Unit tests for relay service
- ✓ Unit tests for webhook-consumer service  
- ✓ Dockerfile validation (both services)
- ✓ Docker Compose configuration validation
- ✓ SQL migration validation

## Individual Test Execution

### Unit Tests Only
```bash
# All unit tests
go test ./cmd/... -v

# Specific service
go test ./cmd/relay -v
go test ./cmd/webhook-consumer -v

# With coverage
go test ./cmd/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Configuration Validation Only
```bash
# All validation tests
./cmd/relay/Dockerfile_test.sh
./cmd/webhook-consumer/Dockerfile_test.sh
./docker-compose_test.sh
./migrations/migration_test.sh
```

### Specific Test Function
```bash
# Run a single test
go test ./cmd/relay -v -run TestExtractOutbox
go test ./cmd/webhook-consumer -v -run TestFinishOrderHandler
```

### Benchmarks
```bash
go test ./cmd/webhook-consumer -bench=. -benchmem
```

## What's Tested

### Relay Service (cmd/relay/main_test.go)
- ✓ Debezium CDC event handling
- ✓ Webhook HTTP calls with retries
- ✓ Outbox data extraction
- ✓ Database updates
- ✓ Error handling (network, JSON, database)
- ✓ Concurrency safety
- ✓ Timeout handling

### Webhook Consumer (cmd/webhook-consumer/main_test.go)
- ✓ Order completion endpoint
- ✓ Database transaction handling
- ✓ Request validation
- ✓ Error responses
- ✓ Concurrency handling
- ✓ Edge cases (nil values, large IDs, special chars)

### Configuration Files
- ✓ Dockerfiles follow best practices
- ✓ Multi-stage builds configured
- ✓ Security settings (CGO disabled)
- ✓ Docker Compose orchestration
- ✓ CDC configuration (WAL, replication)
- ✓ SQL migration idempotency

## Test Coverage

- **25+ test functions**
- **50+ test scenarios**
- **1,563 lines of test code**
- **6 validation scripts**

## Prerequisites

```bash
# Install dependencies
go mod download

# Verify Go version
go version  # Should be 1.24+
```

## CI/CD Integration

Add to your CI pipeline:

```yaml
# .github/workflows/test.yml
- name: Test
  run: |
    cd outbox-pattern
    ./run_all_tests.sh
```

## Troubleshooting

### "module not found" error
```bash
go mod tidy
```

### "permission denied" on scripts
```bash
chmod +x run_all_tests.sh
chmod +x cmd/relay/Dockerfile_test.sh
chmod +x cmd/webhook-consumer/Dockerfile_test.sh
chmod +x docker-compose_test.sh
chmod +x migrations/migration_test.sh
```

### Tests pass locally but fail in CI
- Check Go version (needs 1.24+)
- Verify all dependencies are available
- Check file paths are relative to repo root

## More Information

- **TEST_GUIDE.md** - Comprehensive testing guide
- **TEST_SUMMARY.md** - Detailed test suite overview

## Test Philosophy

Tests are designed to be:
1. **Fast** - Complete in seconds
2. **Isolated** - No external dependencies
3. **Deterministic** - No flaky tests
4. **Comprehensive** - Cover happy paths, edge cases, and errors
5. **Maintainable** - Clear structure and naming

---

**Need Help?** Check TEST_GUIDE.md for detailed documentation.