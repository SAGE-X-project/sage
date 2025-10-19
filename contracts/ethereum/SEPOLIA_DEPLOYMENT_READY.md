# Sepolia Deployment - Ready to Deploy!

**Status**: ✅ All preparation complete - Ready for Sepolia testnet deployment
**Date**: 2025-01-19
**Last Update**: Local deployment tested and successful

---

## ✅ Completed Preparation

### 1. Documentation
- ✅ [DEPLOYMENT_GUIDE_V4.md](./DEPLOYMENT_GUIDE_V4.md) - Complete deployment guide
- ✅ [DEPLOYMENT_CHECKLIST.md](./DEPLOYMENT_CHECKLIST.md) - Quick reference checklist
- ✅ [.env.example](./.env.example) - Environment template ready

### 2. Deployment Scripts
- ✅ `scripts/deploy_v4.js` - Tested and working
  - Local deployment successful
  - Contract address: `0x5FbDB2315678afecb367f032d93F642f64180aa3`
  - Gas used: 3,433,564
  - Fixed Ownable pattern issues

- ✅ `scripts/verify_v4.js` - Etherscan verification ready

### 3. Contract Compilation
- ✅ SageRegistryV4.sol compiled successfully
- ✅ No compilation errors
- ✅ Optimizer enabled (200 runs)
- ✅ Solidity version: 0.8.19

---

## 🚀 Sepolia Deployment Instructions

### Prerequisites Checklist

Before deploying, ensure you have:

#### 1. Sepolia ETH (Test Funds)
- [ ] Minimum 0.5 Sepolia ETH in your wallet
- [ ] Get from faucets:
  - https://sepoliafaucet.com/
  - https://www.alchemy.com/faucets/ethereum-sepolia
  - https://faucet.quicknode.com/ethereum/sepolia

#### 2. RPC Endpoint
Choose one of:
- [ ] Alchemy (recommended): https://dashboard.alchemy.com/
- [ ] Infura: https://infura.io/
- [ ] Public RPC: https://rpc.sepolia.org (less reliable)

#### 3. Etherscan API Key (Optional, for verification)
- [ ] Get from: https://etherscan.io/myapikey

---

## 📝 Step-by-Step Deployment

### Step 1: Configure Environment

```bash
cd contracts/ethereum

# Create .env file if not exists
cp .env.example .env

# Edit .env file
nano .env
```

**Required .env configuration:**

```env
# Your wallet private key (WITHOUT 0x prefix)
# ⚠️ NEVER commit this file!
PRIVATE_KEY=your_sepolia_test_wallet_private_key_here

# Sepolia RPC URL
# Choose one:
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_ALCHEMY_KEY
# SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_INFURA_KEY
# SEPOLIA_RPC_URL=https://rpc.sepolia.org

# Etherscan API key (optional, for automatic verification)
ETHERSCAN_API_KEY=your_etherscan_api_key
```

**Security Notes:**
- ⚠️ Use a TEST wallet only (not your main wallet)
- ⚠️ NEVER commit .env file to git
- ⚠️ Keep your private key secure

### Step 2: Verify Configuration

```bash
# Check your Sepolia balance
npx hardhat run scripts/check-balance.js --network sepolia

# Expected output:
# Network: sepolia (Chain ID: 11155111)
# Account: 0xYOUR_ADDRESS
# Balance: 0.5 ETH  (or more)
```

If balance shows 0 ETH:
1. Visit https://sepoliafaucet.com/
2. Enter your wallet address
3. Wait 1-5 minutes for ETH to arrive
4. Run check-balance again

### Step 3: Deploy to Sepolia

```bash
# Deploy SageRegistryV4 to Sepolia testnet
npx hardhat run scripts/deploy_v4.js --network sepolia
```

**What will happen:**
1. Script shows deployment configuration
2. Estimates gas cost (~0.07-0.1 ETH)
3. Waits 5 seconds for confirmation
4. Deploys contract
5. Saves deployment info to `deployments/v4_sepolia.json`

**Expected output:**
```
╔═══════════════════════════════════════════════════════════╗
║     SageRegistryV4 Deployment Script                     ║
╚═══════════════════════════════════════════════════════════╝

📋 Deployment Configuration
────────────────────────────────────────────────────────────
Network:         sepolia
Chain ID:        11155111
Deployer:        0xYOUR_ADDRESS
Deployer Balance: 0.5 ETH

📊 Estimating deployment cost...
Gas Estimate:    3,500,000
Estimated Cost:  0.105 ETH

⚠️  You are about to deploy to SEPOLIA
   This will cost real funds.

Press Ctrl+C to cancel, or wait 5 seconds to continue...

🚀 Deploying SageRegistryV4...
✅ SageRegistryV4 deployed successfully!
   Address: 0xYOUR_CONTRACT_ADDRESS
   Time:    15.5 seconds

📝 Transaction Details
────────────────────────────────────────────────────────────
Transaction Hash: 0xTRANSACTION_HASH
Block Number:     5,xxx,xxx
Gas Used:         3,433,564
Actual Cost:      0.103 ETH

✅ Contract deployed with deployer as initial admin

💾 Deployment info saved to: deployments/v4_sepolia.json

╔═══════════════════════════════════════════════════════════╗
║     Deployment Complete!                                  ║
╚═══════════════════════════════════════════════════════════╝

📋 Next Steps:

1. Verify contract on Etherscan:
   npx hardhat run scripts/verify_v4.js --network sepolia

2. Update DEPLOYED_ADDRESSES.md

3. Test the deployment

🔍 View on Explorer:
   https://sepolia.etherscan.io/address/0xYOUR_CONTRACT_ADDRESS
```

**Record these values:**
- Contract Address: `___________________________`
- Transaction Hash: `___________________________`
- Block Number: `___________________________`
- Gas Used: `___________________________`
- Deployment Time: `___________________________`

### Step 4: Verify on Etherscan

```bash
# Automatic verification (if ETHERSCAN_API_KEY is set)
npx hardhat run scripts/verify_v4.js --network sepolia
```

**Expected output:**
```
🔍 Verifying SageRegistryV4 on Etherscan...

Contract: 0xYOUR_CONTRACT_ADDRESS
Network: sepolia

✅ Contract verified successfully!

View on Etherscan:
https://sepolia.etherscan.io/address/0xYOUR_CONTRACT_ADDRESS#code
```

**If automatic verification fails:**
```bash
# Manual verification
npx hardhat verify --network sepolia 0xYOUR_CONTRACT_ADDRESS

# Or use Etherscan UI:
# https://sepolia.etherscan.io/verifyContract
```

### Step 5: Update Documentation

```bash
# Edit DEPLOYED_ADDRESSES.md
nano ../../DEPLOYED_ADDRESSES.md
```

Add entry:
```markdown
### Sepolia Testnet (Ethereum)

**Deployed**: 2025-01-19

| Contract | Address | Transaction | Status |
|----------|---------|-------------|--------|
| SageRegistryV4 | `0xYOUR_CONTRACT_ADDRESS` | [View](https://sepolia.etherscan.io/tx/0xYOUR_TX) | ✅ Verified |

**Deployer**: `0xYOUR_WALLET_ADDRESS`
**Gas Used**: 3,433,564
**Cost**: 0.103 ETH
**Block**: 5,xxx,xxx
```

### Step 6: Test Registration

```bash
# Navigate to SAGE root
cd ../..

# Register a test agent
sage-did register \
  --chain ethereum \
  --rpc-url https://sepolia.infura.io/v3/YOUR_KEY \
  --contract 0xYOUR_CONTRACT_ADDRESS \
  --private-key YOUR_TEST_PRIVATE_KEY \
  --name "Test Agent V4" \
  --endpoint "https://test.example.com" \
  --keys test-key.pem

# Verify registration
sage-did key list did:sage:ethereum:0x...
```

### Step 7: Commit Deployment Info

```bash
# Commit (deployment JSON is gitignored, only update docs)
git add contracts/DEPLOYED_ADDRESSES.md
git commit -m "deploy: SageRegistryV4 to Sepolia testnet

- Contract: 0xYOUR_CONTRACT_ADDRESS
- Transaction: 0xYOUR_TX_HASH
- Gas used: 3,433,564
- Status: Verified on Etherscan"

git push origin dev
```

---

## 🔍 Post-Deployment Verification

### Contract Functions Test

```bash
# Test in Hardhat console
npx hardhat console --network sepolia
```

```javascript
// In console
const Registry = await ethers.getContractFactory("SageRegistryV4")
const registry = Registry.attach("0xYOUR_CONTRACT_ADDRESS")

// Test read functions
await registry.VERSION() // Should return "1.1.0"

// Get your address
const [deployer] = await ethers.getSigners()
console.log("Your address:", deployer.address)
```

### Integration Tests

```bash
# Run Go integration tests
cd ../..
SAGE_INTEGRATION_TEST=1 \
CONTRACT_ADDRESS=0xYOUR_CONTRACT_ADDRESS \
RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY \
PRIVATE_KEY=YOUR_TEST_KEY \
go test ./pkg/agent/did/ethereum/... -v
```

---

## 🎯 Success Criteria

Deployment is successful when:

- [x] Contract deployed to Sepolia
- [x] Transaction confirmed on Etherscan
- [x] Contract verified (source code visible)
- [x] Test agent registered successfully
- [x] Integration tests passing
- [x] Documentation updated
- [x] Changes committed to git

---

## 🆘 Troubleshooting

### Issue: "Insufficient funds"
```bash
# Get more Sepolia ETH from faucets
# Visit: https://sepoliafaucet.com/
```

### Issue: "Nonce too low"
```bash
# Reset your wallet nonce
# In MetaMask: Settings > Advanced > Reset Account
```

### Issue: "Connection refused"
```bash
# Try alternative RPC endpoint
# Update SEPOLIA_RPC_URL in .env
```

### Issue: "Verification failed"
```bash
# Wait 30 seconds after deployment
sleep 30

# Retry verification
npx hardhat run scripts/verify_v4.js --network sepolia
```

### Get Help
- GitHub Issues: https://github.com/SAGE-X-project/sage/issues
- Documentation: contracts/ethereum/DEPLOYMENT_GUIDE_V4.md

---

## 📊 Expected Costs

| Item | Amount | Notes |
|------|--------|-------|
| Deployment Gas | ~3,500,000 | Varies by network congestion |
| Gas Price | 20-50 gwei | Check https://sepolia.etherscan.io/gastracker |
| Estimated Cost | 0.07-0.175 ETH | At 20-50 gwei |
| Verification | Free | Using Etherscan API |
| Test Registration | ~200,000 gas | ~0.004-0.01 ETH |

**Total Estimated**: 0.08-0.2 Sepolia ETH (get 0.5 ETH to be safe)

---

## ✅ Ready to Deploy!

Everything is prepared. When you're ready:

```bash
cd contracts/ethereum
npx hardhat run scripts/deploy_v4.js --network sepolia
```

**Good luck with your deployment! 🚀**

---

**See Also:**
- [DEPLOYMENT_GUIDE_V4.md](./DEPLOYMENT_GUIDE_V4.md) - Detailed guide
- [DEPLOYMENT_CHECKLIST.md](./DEPLOYMENT_CHECKLIST.md) - Quick checklist
- [../../DEPLOYED_ADDRESSES.md](../../DEPLOYED_ADDRESSES.md) - Deployment tracking
