# SAGE Complete Test Execution Guide

## IMPORTANT: Pre-test Requirements

### Port Status Check and Cleanup
```bash
# 1. Check port status
cd sage/contracts/ethereum
npm run node:status

# 2. Clean ports if needed
npm run port:clean
```

---

## Test Execution Methods

### Method 1: Automated Test Scripts (Recommended)

#### 1.1 Quick Test (1 minute)
```bash
# Run from project root
./quick-test.sh

# Success messages:
# Compilation completed
# Deployment completed
# Registry: 0x5FbDB2315678afecb367f032d93F642f64180aa3
# Registered agents: 3
```

#### 1.2 Basic Test (5 minutes)
```bash
# Full automated test (Start node -> Deploy -> Verify -> Cleanup)
./test-sage.sh

# Success message:
# All tests completed successfully!
```

#### 1.3 Full Test with Agents (10 minutes)
```bash
# Including multi-agent system
./test-sage.sh --with-agents

# Keep running after test (for debugging)
./test-sage.sh --with-agents --keep-alive
```

---

### Method 2: Manual Step-by-Step Testing

#### Step 1: Start Hardhat Node
```bash
# Terminal 1
cd sage/contracts/ethereum

# Check port
npm run node:status

# Start node
npm run node

# Verify output:
# Started HTTP and WebSocket JSON-RPC server at http://127.0.0.1:8545/
# 
# Accounts
# ========
# Account #0: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (10000 ETH)
```

#### Step 2: Deploy Contracts
```bash
# Terminal 2 (new terminal)
cd sage/contracts/ethereum

# Compile
npm run compile

# Deploy to localhost
npm run deploy:unified:local

# Success output:
# SageRegistryV2 deployed to: 0x5FbDB2315678afecb367f032d93F642f64180aa3
# SageVerificationHook deployed to: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
# 3 agents registered successfully
```

#### Step 3: Verify Deployment
```bash
# In the same terminal
npx hardhat run scripts/verify-deployment.js --network localhost

# Success output:
# Deployment info loaded successfully
# Owner: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
# Before Hook: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
# After Hook: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
# Number of registered agents: 3
```

#### Step 4: Query Agents
```bash
npx hardhat run scripts/query-agents.js --network localhost
```

#### Step 5: Test Go Application
```bash
# Set environment variables
export SAGE_REGISTRY_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
export SAGE_NETWORK=localhost
export SAGE_CHAIN_ID=31337

# Go verification
cd ../../
go run sage/cmd/sage-verify/main.go
```

#### Step 6: Test Cleanup and Termination
```bash
# In terminal 2
cd sage/contracts/ethereum
npm run node:stop

# Or press Ctrl+C in terminal 1
```

---

## Verification Checkpoints

### Required Verification Items

1. **Contract Deployment Verification**
```bash
cat sage/contracts/ethereum/deployments/localhost.json | grep -A1 "SageRegistryV2"
```

2. **Agent Registration Verification**
```bash
cat sage/contracts/ethereum/deployments/localhost.json | grep -c "did:"
# Result: 3
```

3. **Blockchain Connection Test**
```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

4. **Port Status Check**
```bash
lsof -i:8545  # Hardhat node
lsof -i:3001  # Root Agent (optional)
lsof -i:3002  # Ordering Agent (optional)
lsof -i:3003  # Planning Agent (optional)
```

---

## Troubleshooting Common Issues

### Issue 1: "Error: Network Error" or "could not decode result data"
**Cause**: Hardhat node is not running or connection failed

**Solution**:
```bash
# 1. Check port
npm run node:status

# 2. Restart node
npm run node:restart

# 3. Confirm using localhost network
# For deployment: --network localhost
# For verification: --network localhost
```

### Issue 2: "Port 8545 already in use"
**Cause**: Previous test node was not terminated

**Solution**:
```bash
# Method 1: npm script
npm run node:stop

# Method 2: Port manager
./scripts/port-manager.sh clean --hardhat

# Method 3: Manual
lsof -ti:8545 | xargs kill -9
```

### Issue 3: "Key ownership not proven"
**Cause**: Using production contract in local environment

**Solution**:
```bash
# SageRegistryTest is used for localhost or hardhat network
# deploy-unified.js handles this automatically
```

### Issue 4: "npm error code ENOENT"
**Cause**: Running from wrong directory

**Solution**:
```bash
# Move to correct directory
cd sage/contracts/ethereum
pwd  # Verify
```

---

## Test Result Interpretation

### Indicators of Successful Test

1. **Compilation Success**
   - "Compiled 2 Solidity files successfully"

2. **Deployment Success**
   - Registry address: 0x5FbDB2315678afecb367f032d93F642f64180aa3
   - Hook address: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
   - 3 agents registered

3. **Verification Success**
   - Owner verified
   - Hook configuration verified
   - Number of agents: 3

4. **Network Connection**
   - Block number increasing
   - Transaction processing

---

## Test Flow Diagram

```
Start
  ↓
Check ports (npm run node:status)
  ↓
Port in use? → Yes → Clean (npm run port:clean)
  ↓            No
Start Hardhat node (npm run node)
  ↓
Compile contracts (npm run compile)
  ↓
Deploy contracts (npm run deploy:unified:local)
  ↓
Verify deployment (npm run verify:deployment)
  ↓
Query agents (optional)
  ↓
Go test (optional)
  ↓
Stop node (npm run node:stop or Ctrl+C)
  ↓
Complete
```

---

## Additional Test Commands

### Unit Tests
```bash
cd sage/contracts/ethereum
npm test
```

### Test Coverage
```bash
npm run coverage
```

### Gas Usage Report
```bash
REPORT_GAS=true npm test
```

### Run Specific Tests
```bash
npm run test:v2  # V2 contracts only
npm run test:integration  # Integration tests only
```

---

## Tips and Best Practices

1. **Always use localhost network**
   - hardhat network is in-memory and resets on restart
   - localhost maintains state in separate node

2. **Check ports before testing**
   - Make `npm run node:status` a habit
   - Use `npm run port:clean` when needed

3. **Utilize log files**
   - hardhat.log: Node logs
   - deploy.log: Deployment logs
   - verify.log: Verification logs

4. **Backup environment variables**
   ```bash
   cp deployments/localhost.env ~/.sage-test.env
   source ~/.sage-test.env  # Restore when needed
   ```

---

## Summary: Quickest Test Method

```bash
# 1. Automated test (recommended)
./test-sage.sh

# 2. Manual quick test
cd sage/contracts/ethereum
npm run port:clean           # Clean ports
npm run node &               # Run in background
sleep 5                      # Wait
npm run deploy:unified:local # Deploy
npm run verify:deployment    # Verify
npm run node:stop           # Stop
```

---

**Last Updated**: 2025-09-27
**Version**: 1.1.0
**Author**: SAGE Development Team