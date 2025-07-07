#!/bin/bash
# Test that all MCP integration examples compile

set -e

echo "Testing MCP integration examples compilation..."

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Test basic demo
echo "Testing basic-demo..."
cd basic-demo
go build -o /dev/null .
cd ..

# Test simple standalone
echo "Testing simple-standalone..."
cd simple-standalone
go build -o /dev/null .
cd ..

# Test client
echo "Testing client..."
cd client  
go build -o /dev/null .
cd ..

# Test basic tool (requires imports)
echo "Testing basic-tool..."
cd basic-tool
go build -o /dev/null .
cd ..

# Test vulnerable chat
echo "Testing vulnerable chat..."
cd vulnerable-vs-secure/vulnerable-chat
go build -o /dev/null .
cd ../..

# Test secure chat
echo "Testing secure chat..."
cd vulnerable-vs-secure/secure-chat
go build -o /dev/null .
cd ../..

# Test attacker
echo "Testing attacker demo..."
cd vulnerable-vs-secure/attacker
go build -o /dev/null .
cd ../..

echo "✅ All examples compile successfully!"