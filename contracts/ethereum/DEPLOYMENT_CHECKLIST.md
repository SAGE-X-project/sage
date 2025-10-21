# SageRegistryV4 Deployment Checklist

Quick reference checklist for deploying SageRegistryV4 to Sepolia testnet.

**Date**: _____________
**Deployer**: _____________
**Network**: Sepolia Testnet

---

## Pre-Deployment Checklist

### Environment Setup
- [ ] Node.js 18+ installed
- [ ] Dependencies installed (`npm install`)
- [ ] `.env` file created from `.env.example`
- [ ] Private key configured in `.env`
- [ ] Sepolia RPC URL configured
- [ ] Etherscan API key configured

### Funding & Access
- [ ] Deployer wallet has Sepolia ETH (minimum 0.5 ETH)
  - Faucets: https://sepoliafaucet.com/
  - Check balance: `npx hardhat run scripts/check-balance.js --network sepolia`
- [ ] RPC endpoint accessible
  - Test: `curl https://sepolia.infura.io/v3/YOUR_KEY`
- [ ] Etherscan API key valid
  - Get from: https://etherscan.io/myapikey

### Code Preparation
- [ ] Latest code pulled from `dev` branch
- [ ] All tests passing locally
  - Run: `npm test`
- [ ] Contracts compiled successfully
  - Run: `npx hardhat compile`
- [ ] No pending git changes (clean working tree)

---

## Deployment Steps

### Step 1: Final Pre-Flight Checks

```bash
# Navigate to contracts directory
cd contracts/ethereum

# Clean and recompile
npx hardhat clean
npx hardhat compile

# Run full test suite
npm test

# Check deployer balance
npx hardhat run scripts/check-balance.js --network sepolia
```

**Expected Results:**
- [ ] All tests pass (201 tests)
- [ ] Balance shows ≥ 0.5 Sepolia ETH
- [ ] No compilation errors

### Step 2: Deploy Contract

```bash
# Deploy SageRegistryV4 to Sepolia
npx hardhat run scripts/deploy_v4.js --network sepolia
```

**Record Deployment Info:**
- Contract Address: `_______________________`
- Transaction Hash: `_______________________`
- Gas Used: `_______________________`
- Block Number: `_______________________`
- Timestamp: `_______________________`

**Verification:**
- [ ] Deployment transaction confirmed on Etherscan
  - URL: https://sepolia.etherscan.io/tx/YOUR_TX_HASH
- [ ] Contract address visible
- [ ] No errors in transaction
- [ ] Deployment JSON saved to `deployments/v4/sepolia-deployment.json`

### Step 3: Verify Contract

```bash
# Automatic verification on Etherscan
npx hardhat run scripts/verify_v4.js --network sepolia
```

**Alternative Manual Verification:**
```bash
npx hardhat verify --network sepolia <CONTRACT_ADDRESS>
```

**Verification Checks:**
- [ ] Contract source code visible on Etherscan
- [ ] Contract marked as "Verified"
- [ ] Read/Write functions accessible
- [ ] Contract name shows "SageRegistryV4"

---

## Post-Deployment Checklist

### Documentation Updates

- [ ] Update `contracts/DEPLOYED_ADDRESSES.md`
  - Add Sepolia deployment entry
  - Include contract address, tx hash, timestamp
  - Mark as verified

- [ ] Create deployment commit
  ```bash
  git add deployments/v4/sepolia-deployment.json
  git add contracts/DEPLOYED_ADDRESSES.md
  git commit -m "deploy: SageRegistryV4 to Sepolia testnet"
  git push origin dev
  ```

### Testing & Validation

- [ ] Test basic contract read functions
  ```bash
  npx hardhat console --network sepolia
  > const Registry = await ethers.getContractFactory("SageRegistryV4")
  > const registry = Registry.attach("<CONTRACT_ADDRESS>")
  > await registry.VERSION()
  ```

- [ ] Register test agent via CLI
  ```bash
  sage-did register \
    --chain ethereum \
    --rpc-url https://sepolia.infura.io/v3/YOUR_KEY \
    --contract <CONTRACT_ADDRESS> \
    --name "Test Agent" \
    --endpoint "https://test.com"
  ```

- [ ] Verify agent registration
  ```bash
  sage-did key list did:sage:ethereum:0x...
  ```

### Integration Setup

- [ ] Update Go backend configuration
  - File: `pkg/agent/did/manager.go` or config
  - Set Sepolia contract address
  - Set RPC endpoint

- [ ] Run Go integration tests
  ```bash
  SAGE_INTEGRATION_TEST=1 \
  CONTRACT_ADDRESS=<CONTRACT_ADDRESS> \
  RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY \
  go test ./pkg/agent/did/ethereum/... -v
  ```

- [ ] All integration tests passing

---

## Validation Checklist

### Contract Functionality

- [ ] `VERSION()` returns "1.1.0"
- [ ] `registerAgent()` works with single ECDSA key
- [ ] `registerAgent()` works with multiple keys
- [ ] `addKey()` adds key to existing agent
- [ ] `revokeKey()` revokes key successfully
- [ ] `approveEd25519Key()` approves Ed25519 key (owner only)
- [ ] `getAgent()` returns correct agent data
- [ ] `getAgentKeys()` returns all keys
- [ ] `resolvePublicKey()` returns first verified key
- [ ] `resolveAllPublicKeys()` returns all verified keys
- [ ] `resolvePublicKeyByType()` filters by key type

### Events Verification

- [ ] `AgentRegistered` event emitted on registration
- [ ] `KeyAdded` event emitted on key addition
- [ ] `KeyRevoked` event emitted on revocation
- [ ] `Ed25519KeyApproved` event emitted on approval
- [ ] Events indexed correctly on Etherscan

### Security Checks

- [ ] Only owner can call admin functions
- [ ] Non-owner calls revert with correct error
- [ ] Signature validation working correctly
- [ ] Key hash calculation matches expected format
- [ ] Agent ID calculation correct

---

## Rollback Plan

In case of issues after deployment:

### Option 1: Deploy New Version
```bash
# Fix issues in code
# Deploy new contract
npx hardhat run scripts/deploy_v4.js --network sepolia

# Update references to new address
```

### Option 2: Use Previous Deployment
```bash
# Revert to previous stable contract
# Update DEPLOYED_ADDRESSES.md
# Notify team of rollback
```

### Option 3: Pause Operations
```bash
# If contract has pause functionality
# Contact users to stop operations
# Investigate and fix issues
```

---

## Troubleshooting

### Common Issues

**Issue: Insufficient funds**
```bash
# Get more Sepolia ETH
# Visit: https://sepoliafaucet.com/
# Or: https://www.alchemy.com/faucets/ethereum-sepolia
```

**Issue: Nonce too low**
```bash
# Reset MetaMask account
# Settings > Advanced > Reset Account
```

**Issue: Gas price too low**
```bash
# Check current gas prices
# https://sepolia.etherscan.io/gastracker

# Update .env
GAS_PRICE=50
```

**Issue: Verification fails**
```bash
# Wait 30 seconds
sleep 30

# Retry verification
npx hardhat run scripts/verify_v4.js --network sepolia

# Or use manual verification
```

**Issue: RPC connection fails**
```bash
# Try alternative RPC endpoints:
# - Alchemy: https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY
# - Infura: https://sepolia.infura.io/v3/YOUR_KEY
# - Public: https://rpc.sepolia.org
```

---

## Success Criteria

Deployment is considered successful when:

- [x] Contract deployed to Sepolia
- [x] Contract verified on Etherscan
- [x] All read functions working
- [x] Test agent registered successfully
- [x] Integration tests passing
- [x] Documentation updated
- [x] Changes committed to git

---

## Team Notification

After successful deployment, notify:

- [ ] Development team (Slack/Discord)
- [ ] QA team for testing
- [ ] Documentation team
- [ ] Project stakeholders

**Notification Template:**
```
🚀 SageRegistryV4 Deployed to Sepolia!

Contract: 0x... (link to Etherscan)
Transaction: 0x...
Gas Used: X,XXX,XXX
Status: ✅ Verified

Next steps:
- Integration testing
- Update client SDKs
- Prepare mainnet deployment

Details: contracts/DEPLOYED_ADDRESSES.md
```

---

## Signature

**Deployed by**: _________________ (Name)
**Date**: _________________ (YYYY-MM-DD)
**Time**: _________________ (HH:MM UTC)
**Verified by**: _________________ (Reviewer)

---

**See Also:**
- [DEPLOYMENT_GUIDE_V4.md](./DEPLOYMENT_GUIDE_V4.md) - Detailed deployment guide
- [DEPLOYED_ADDRESSES.md](../../DEPLOYED_ADDRESSES.md) - Deployment tracking
- [README.md](../README.md) - Main documentation
