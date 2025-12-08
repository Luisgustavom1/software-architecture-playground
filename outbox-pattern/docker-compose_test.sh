#!/bin/bash
# Docker Compose validation tests

set -e

echo "=== Docker Compose Configuration Validation Tests ==="

COMPOSE_FILE="outbox-pattern/docker-compose.yml"

# Test 1: File exists
echo "Test 1: Checking if docker-compose.yml exists..."
[ -f "$COMPOSE_FILE" ] && echo "✓ docker-compose.yml exists" || { echo "✗ Failed"; exit 1; }

# Test 2: Services exist
echo "Test 2: Checking required services..."
for service in postgres debezium relay webhook-consumer; do
    grep -q "^  $service:" "$COMPOSE_FILE" && echo "✓ Service '$service' defined" || { echo "✗ Failed"; exit 1; }
done

# Test 3: PostgreSQL WAL configuration
echo "Test 3: Checking PostgreSQL WAL configuration for CDC..."
grep -q "wal_level=logical" "$COMPOSE_FILE" && echo "✓ WAL level set to logical" || { echo "✗ Failed"; exit 1; }
grep -q "max_wal_senders" "$COMPOSE_FILE" && echo "✓ max_wal_senders configured" || { echo "✗ Failed"; exit 1; }
grep -q "max_replication_slots" "$COMPOSE_FILE" && echo "✓ max_replication_slots configured" || { echo "✗ Failed"; exit 1; }

# Test 4: Debezium configuration
echo "Test 4: Checking Debezium configuration..."
grep -q "DEBEZIUM_SINK_TYPE: http" "$COMPOSE_FILE" && echo "✓ Debezium sink type set to HTTP" || { echo "✗ Failed"; exit 1; }
grep -q "DEBEZIUM_SOURCE_CONNECTOR_CLASS.*PostgresConnector" "$COMPOSE_FILE" && echo "✓ PostgreSQL connector configured" || { echo "✗ Failed"; exit 1; }
grep -q "DEBEZIUM_SOURCE_TABLE_INCLUDE_LIST.*outbox" "$COMPOSE_FILE" && echo "✓ Outbox table included in CDC" || { echo "✗ Failed"; exit 1; }

# Test 5: Port mappings
echo "Test 5: Checking port mappings..."
grep -q "8081:8081" "$COMPOSE_FILE" && echo "✓ Relay port 8081 mapped" || { echo "✗ Failed"; exit 1; }
grep -q "8082:8082" "$COMPOSE_FILE" && echo "✓ Webhook consumer port 8082 mapped" || { echo "✗ Failed"; exit 1; }
grep -q "5432:5432" "$COMPOSE_FILE" && echo "✓ PostgreSQL port 5432 mapped" || { echo "✗ Failed"; exit 1; }

# Test 6: Network configuration
echo "Test 6: Checking network configuration..."
grep -q "outbox-network:" "$COMPOSE_FILE" && echo "✓ Custom network 'outbox-network' defined" || { echo "✗ Failed"; exit 1; }

# Test 7: Webhook URL configuration
echo "Test 7: Checking webhook URL configuration..."
grep -q "WEBHOOK_URL.*webhook-consumer:8082" "$COMPOSE_FILE" && echo "✓ Webhook URL correctly configured" || { echo "✗ Failed"; exit 1; }

# Test 8: Debezium sink URL
echo "Test 8: Checking Debezium sink URL..."
grep -q "DEBEZIUM_SINK_HTTP_URL.*relay:8081/debezium" "$COMPOSE_FILE" && echo "✓ Debezium sink URL correctly configured" || { echo "✗ Failed"; exit 1; }

# Test 9: Restart policies
echo "Test 9: Checking restart policies..."
restart_count=$(grep -c "restart: unless-stopped" "$COMPOSE_FILE")
[ "$restart_count" -ge 4 ] && echo "✓ Restart policies configured for all services" || { echo "✗ Failed"; exit 1; }

# Test 10: Debezium publication configuration
echo "Test 10: Checking Debezium publication configuration..."
grep -q "DEBEZIUM_SOURCE_PUBLICATION_NAME: outbox_pub" "$COMPOSE_FILE" && echo "✓ Debezium publication name configured" || { echo "✗ Failed"; exit 1; }
grep -q "DEBEZIUM_SOURCE_SLOT_NAME: outbox_slot" "$COMPOSE_FILE" && echo "✓ Debezium replication slot configured" || { echo "✗ Failed"; exit 1; }

echo ""
echo "=== All docker-compose validation tests passed! ==="