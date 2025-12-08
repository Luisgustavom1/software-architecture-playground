#!/bin/bash
# Master test runner for Outbox Pattern implementation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "======================================"
echo "  Outbox Pattern Test Suite Runner"
echo "======================================"
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

run_test() {
    local test_name=$1
    local test_command=$2
    
    echo "----------------------------------------"
    echo "Running: $test_name"
    echo "----------------------------------------"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ PASSED${NC}: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo ""
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo ""
        return 1
    fi
}

echo "Phase 1: Unit Tests"
echo "======================================"
run_test "Relay Service Unit Tests" "go test ./cmd/relay -v"
run_test "Webhook Consumer Unit Tests" "go test ./cmd/webhook-consumer -v"

echo ""
echo "Phase 2: Configuration Validation"
echo "======================================"
run_test "Relay Dockerfile Validation" "./cmd/relay/Dockerfile_test.sh"
run_test "Webhook Consumer Dockerfile Validation" "./cmd/webhook-consumer/Dockerfile_test.sh"
run_test "Docker Compose Validation" "./docker-compose_test.sh"
run_test "SQL Migration Validation" "./migrations/migration_test.sh"

echo ""
echo "======================================"
echo "  Test Suite Summary"
echo "======================================"
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
else
    echo -e "${GREEN}Failed: $FAILED_TESTS${NC}"
fi
echo "======================================"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed! 🎉${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please review the output above.${NC}"
    exit 1
fi