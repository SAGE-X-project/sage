#!/bin/bash

echo "🚀 Starting SAGE contract full test"
echo "========================================"

# 1. Compile
echo ""
echo "📝 Step 1: Contract compilation"
npm run compile
if [ $? -ne 0 ]; then
    echo "❌ Compilation failed"
    exit 1
fi

# 2. Start local node (background)
echo ""
echo "🖥️ Step 2: Starting Hardhat node"
npx hardhat node &
NODE_PID=$!
echo "Node PID: $NODE_PID"
sleep 5  # Wait for node to start

# 3. Deploy
echo ""
echo "🚀 Step 3: Contract deployment"
npm run deploy:unified:local
if [ $? -ne 0 ]; then
    echo "❌ Deployment failed"
    kill $NODE_PID
    exit 1
fi

# 4. Verification
echo ""
echo "🔍 Step 4: Deployment verification"
npx hardhat run scripts/verify-deployment.js --network localhost
if [ $? -ne 0 ]; then
    echo "❌ Verification failed"
    kill $NODE_PID
    exit 1
fi

# 5. Agent query
echo ""
echo "📊 Step 5: Agent query"
npx hardhat run scripts/query-agents.js --network localhost
if [ $? -ne 0 ]; then
    echo "⚠️ Agent query failed (ignorable)"
fi

# 6. Run tests
echo ""
echo "🧪 Step 6: Unit tests"
npm test
if [ $? -ne 0 ]; then
    echo "❌ Tests failed"
    kill $NODE_PID
    exit 1
fi

# 7. Cleanup
echo ""
echo "🧹 Step 7: Cleanup"
kill $NODE_PID
echo "✅ Node terminated"

echo ""
echo "========================================"
echo "✅ All tests completed!"
echo ""
echo "📍 Deployment info: deployments/localhost.json"
echo "📍 Environment variables: deployments/localhost.env"
echo ""
echo "Next steps:"
echo "1. cp deployments/localhost.env ../../.env"
echo "2. Test Go application"