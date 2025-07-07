#!/bin/bash

# CLI Integration Test Script

set -e

echo "=== SAGE CLI Integration Tests ==="

# Build CLIs
echo "Building CLIs..."
make clean > /dev/null 2>&1
make build > /dev/null 2>&1

CRYPTO_BIN="./build/bin/sage-crypto"
DID_BIN="./build/bin/sage-did"

# Test 1: Generate Ed25519 key
echo -n "Test 1: Generate Ed25519 key... "
$CRYPTO_BIN generate --type ed25519 --format jwk --output test-ed25519.jwk > /dev/null 2>&1
if [ -f "test-ed25519.jwk" ]; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 2: Generate Secp256k1 key
echo -n "Test 2: Generate Secp256k1 key... "
$CRYPTO_BIN generate --type secp256k1 --format pem --output test-secp256k1.pem > /dev/null 2>&1
if [ -f "test-secp256k1.pem" ]; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 3: Sign and verify message
echo -n "Test 3: Sign and verify message... "
echo "Test message for signing" > test-message.txt
SIGNATURE=$($CRYPTO_BIN sign --key test-ed25519.jwk --message-file test-message.txt --base64 2>/dev/null)
if $CRYPTO_BIN verify --key test-ed25519.jwk --message-file test-message.txt --signature-b64 "$SIGNATURE" > /dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 4: Key storage
echo -n "Test 4: Key storage... "
mkdir -p test-storage
$CRYPTO_BIN generate --type ed25519 --format storage --storage-dir test-storage --key-id testkey > /dev/null 2>&1
if [ -f "test-storage/testkey.key" ]; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 5: List keys
echo -n "Test 5: List keys... "
if $CRYPTO_BIN list --storage-dir test-storage > /dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 6: Sign with stored key
echo -n "Test 6: Sign with stored key... "
SIGNATURE2=$($CRYPTO_BIN sign --storage-dir test-storage --key-id testkey --message "Hello SAGE" --base64 2>/dev/null)
if [ ! -z "$SIGNATURE2" ]; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 7: DID CLI help
echo -n "Test 7: DID CLI help... "
if $DID_BIN --help > /dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 8: Invalid signature verification
echo -n "Test 8: Invalid signature verification (should fail)... "
if ! $CRYPTO_BIN verify --key test-ed25519.jwk --message "Wrong message" --signature-b64 "$SIGNATURE" > /dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Clean up
echo -n "Cleaning up... "
rm -f test-ed25519.jwk test-secp256k1.pem test-message.txt
rm -rf test-storage
echo "DONE"

echo ""
echo "=== All tests passed! ==="