#!/bin/sh
# SAGE Docker Health Check Script
# Performs comprehensive health checks for the container

set -e

# Exit codes
EXIT_OK=0
EXIT_ERROR=1

# Check if sage-crypto binary exists and works
if ! command -v sage-crypto >/dev/null 2>&1; then
    echo "ERROR: sage-crypto binary not found"
    exit $EXIT_ERROR
fi

# Help check (basic liveness test)
if ! sage-crypto help >/dev/null 2>&1; then
    echo "ERROR: sage-crypto binary check failed"
    exit $EXIT_ERROR
fi

# Check if required directories exist
if [ ! -d "$HOME/.sage/keys" ]; then
    echo "ERROR: Keys directory not found"
    exit $EXIT_ERROR
fi

if [ ! -d "$HOME/.sage/data" ]; then
    echo "ERROR: Data directory not found"
    exit $EXIT_ERROR
fi

# Check blockchain connectivity (if configured)
if [ -n "$ETHEREUM_RPC_URL" ] && [ "$SAGE_NETWORK" != "none" ]; then
    if ! wget -q -O- --timeout=3 "$ETHEREUM_RPC_URL" >/dev/null 2>&1; then
        echo "WARNING: Blockchain RPC not responding"
        # Don't fail health check for RPC issues (might be temporary)
    fi
fi

# Check metrics endpoint (if running as server)
if [ -n "$SAGE_METRICS_PORT" ]; then
    if ! wget -q -O- --timeout=2 "http://localhost:${SAGE_METRICS_PORT}/health" >/dev/null 2>&1; then
        echo "WARNING: Metrics endpoint not responding"
        # Don't fail for metrics issues
    fi
fi

echo "Health check passed"
exit $EXIT_OK
