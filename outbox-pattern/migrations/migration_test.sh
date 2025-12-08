#!/bin/bash
# SQL migration validation tests

set -e

echo "=== SQL Migration Validation Tests ==="

MIGRATION_FILE="outbox-pattern/migrations/001_init_schema.sql"

# Test 1: File exists
echo "Test 1: Checking if migration file exists..."
if [ -f "$MIGRATION_FILE" ]; then
    echo "✓ Migration file exists"
else
    echo "✗ Migration file does not exist"
    exit 1
fi

# Test 2: Check for orders table
echo "Test 2: Checking orders table definition..."
if grep -q "CREATE TABLE.*orders" "$MIGRATION_FILE"; then
    echo "✓ Orders table creation found"
else
    echo "✗ Orders table not defined"
    exit 1
fi

# Test 3: Check for outbox table
echo "Test 3: Checking outbox table definition..."
if grep -q "CREATE TABLE.*outbox" "$MIGRATION_FILE"; then
    echo "✓ Outbox table creation found"
else
    echo "✗ Outbox table not defined"
    exit 1
fi

# Test 4: Check for outbox status index
echo "Test 4: Checking status index on outbox table..."
if grep -q "CREATE INDEX.*idx_outbox_status" "$MIGRATION_FILE"; then
    echo "✓ Status index defined"
else
    echo "✗ Status index missing"
    exit 1
fi

# Test 5: Check for outbox aggregate index
echo "Test 5: Checking aggregate index on outbox table..."
if grep -q "CREATE INDEX.*idx_outbox_aggregate" "$MIGRATION_FILE"; then
    echo "✓ Aggregate index defined"
else
    echo "✗ Aggregate index missing"
    exit 1
fi

# Test 6: Check for publication
echo "Test 6: Checking Debezium publication..."
if grep -q "CREATE PUBLICATION.*outbox_pub" "$MIGRATION_FILE"; then
    echo "✓ Publication for CDC defined"
else
    echo "✗ Publication not defined"
    exit 1
fi

# Test 7: Check publication targets outbox table
echo "Test 7: Verifying publication includes outbox table..."
if grep -q "FOR TABLE outbox" "$MIGRATION_FILE"; then
    echo "✓ Publication targets outbox table"
else
    echo "✗ Publication does not target outbox table"
    exit 1
fi

# Test 8: Check IF NOT EXISTS clauses
echo "Test 8: Checking idempotency (IF NOT EXISTS)..."
if_not_exists_count=$(grep -c "IF NOT EXISTS" "$MIGRATION_FILE")
if [ "$if_not_exists_count" -ge 3 ]; then
    echo "✓ Idempotent migration (IF NOT EXISTS used)"
else
    echo "✗ Migration may not be idempotent"
    exit 1
fi

# Test 9: Check for proper data types
echo "Test 9: Checking data types..."
if grep -q "BIGSERIAL\|SERIAL" "$MIGRATION_FILE"; then
    echo "✓ Auto-incrementing ID columns found"
else
    echo "✗ No auto-incrementing columns found"
    exit 1
fi

# Test 10: Check for timestamp columns
echo "Test 10: Checking timestamp columns..."
if grep -q "created_at.*TIMESTAMP" "$MIGRATION_FILE" && \
   grep -q "updated_at.*TIMESTAMP" "$MIGRATION_FILE"; then
    echo "✓ Timestamp columns defined"
else
    echo "✗ Timestamp columns missing or incorrect"
    exit 1
fi

# Test 11: Check for DEFAULT values
echo "Test 11: Checking DEFAULT constraints..."
if grep -q "DEFAULT NOW()" "$MIGRATION_FILE" || grep -q "DEFAULT CURRENT_TIMESTAMP" "$MIGRATION_FILE"; then
    echo "✓ DEFAULT timestamp values defined"
else
    echo "✗ DEFAULT values not properly set"
    exit 1
fi

# Test 12: Check for status field in outbox
echo "Test 12: Checking outbox status field..."
if grep -A 10 "CREATE TABLE.*outbox" "$MIGRATION_FILE" | grep -q "status.*VARCHAR"; then
    echo "✓ Outbox status field defined"
else
    echo "✗ Outbox status field missing"
    exit 1
fi

# Test 13: Check for aggregate_id in outbox
echo "Test 13: Checking aggregate_id field..."
if grep -A 10 "CREATE TABLE.*outbox" "$MIGRATION_FILE" | grep -q "aggregate_id"; then
    echo "✓ Aggregate ID field defined"
else
    echo "✗ Aggregate ID field missing"
    exit 1
fi

# Test 14: Check for payload field in outbox
echo "Test 14: Checking payload field..."
if grep -A 10 "CREATE TABLE.*outbox" "$MIGRATION_FILE" | grep -q "payload"; then
    echo "✓ Payload field defined"
else
    echo "✗ Payload field missing"
    exit 1
fi

# Test 15: Verify no SQL syntax errors (basic check)
echo "Test 15: Basic SQL syntax validation..."
if grep -q ";" "$MIGRATION_FILE"; then
    echo "✓ SQL statements properly terminated"
else
    echo "✗ Missing statement terminators"
    exit 1
fi

echo ""
echo "=== All SQL migration validation tests passed! ==="