#!/bin/bash
# Dockerfile validation tests for webhook-consumer service

set -e

echo "=== Dockerfile Validation Tests for Webhook Consumer Service ==="

DOCKERFILE="outbox-pattern/cmd/webhook-consumer/Dockerfile"

# Test 1: Dockerfile exists
echo "Test 1: Checking if Dockerfile exists..."
if [ -f "$DOCKERFILE" ]; then
    echo "✓ Dockerfile exists"
else
    echo "✗ Dockerfile does not exist"
    exit 1
fi

# Test 2: Multi-stage build verification
echo "Test 2: Verifying multi-stage build..."
if grep -q "FROM golang:1.24-alpine AS builder" "$DOCKERFILE" && grep -q "FROM alpine:latest" "$DOCKERFILE"; then
    echo "✓ Multi-stage build detected"
else
    echo "✗ Multi-stage build not properly configured"
    exit 1
fi

# Test 3: Verify Go version consistency
echo "Test 3: Checking Go version..."
if grep -q "golang:1.24" "$DOCKERFILE"; then
    echo "✓ Go 1.24 specified"
else
    echo "✗ Go version not 1.24"
    exit 1
fi

# Test 4: Verify CGO is disabled
echo "Test 4: Checking CGO_ENABLED=0..."
if grep -q "CGO_ENABLED=0" "$DOCKERFILE"; then
    echo "✓ CGO disabled for static binary"
else
    echo "✗ CGO not disabled"
    exit 1
fi

# Test 5: Verify GOOS=linux
echo "Test 5: Checking GOOS=linux..."
if grep -q "GOOS=linux" "$DOCKERFILE"; then
    echo "✓ GOOS set to linux"
else
    echo "✗ GOOS not set to linux"
    exit 1
fi

# Test 6: Verify build output path
echo "Test 6: Checking build output..."
if grep -q "go build -o webhook-consumer ./cmd/webhook-consumer" "$DOCKERFILE"; then
    echo "✓ Build output correctly specified"
else
    echo "✗ Build output path incorrect"
    exit 1
fi

# Test 7: Verify CA certificates
echo "Test 7: Checking CA certificates installation..."
if grep -q "apk --no-cache add ca-certificates" "$DOCKERFILE"; then
    echo "✓ CA certificates will be installed"
else
    echo "✗ CA certificates not configured"
    exit 1
fi

# Test 8: Verify exposed port
echo "Test 8: Checking exposed port..."
if grep -q "EXPOSE 8082" "$DOCKERFILE"; then
    echo "✓ Port 8082 exposed"
else
    echo "✗ Port 8082 not exposed"
    exit 1
fi

# Test 9: Verify CMD instruction
echo "Test 9: Checking CMD instruction..."
if grep -q 'CMD \["./webhook-consumer"\]' "$DOCKERFILE"; then
    echo "✓ CMD instruction correct"
else
    echo "✗ CMD instruction missing or incorrect"
    exit 1
fi

# Test 10: Verify dependency layer optimization
echo "Test 10: Checking dependency caching optimization..."
if grep -q "COPY go.mod go.sum ./" "$DOCKERFILE"; then
    echo "✓ Dependencies copied separately for better caching"
else
    echo "✗ Dependencies not optimally cached"
    exit 1
fi

echo ""
echo "=== All Dockerfile validation tests passed! ==="