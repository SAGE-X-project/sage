# SageRegistryV4 Deployment Guide

Complete guide for deploying SageRegistryV4 to Ethereum testnets and mainnet.

**Last Updated**: 2025-01-19
**Version**: V4 (Multi-Key Registry)

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Deployment Procedure](#deployment-procedure)
- [Verification](#verification)
- [Post-Deployment](#post-deployment)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

```bash
# Check Node.js version (18+ required)
node --version

# Check npm version
npm --version

# Check Hardhat installation
npx hardhat --version
```

### Required Resources

**For Sepolia Testnet:**
- Sepolia ETH (minimum 0.5 ETH recommended for deployment + testing)
- Etherscan API key (for automatic contract verification)
- Ethereum wallet with private key

**For Mainnet:**
- Real ETH (estimate: 0.05-0.1 ETH for deployment)
- Etherscan API key (required)
- Hardware wallet recommended
- Multi-sig wallet strongly recommended

### Get Test ETH

**Sepolia Faucets:**
- https://sepoliafaucet.com/
- https://www.alchemy.com/faucets/ethereum-sepolia
- https://faucet.quicknode.com/ethereum/sepolia
- https://faucet.chainstack.com/sepolia-faucet

**How to get Sepolia ETH:**
1. Visit any faucet above
2. Enter your wallet address
3. Complete verification (if required)
4. Wait for ETH to arrive (usually 1-5 minutes)
5. Verify balance: https://sepolia.etherscan.io/

### Get Etherscan API Key

1. Visit https://etherscan.io/
2. Sign up for free account
3. Go to API-KEYs section
4. Create new API key
5. Copy the key for later use

---

## Environment Setup

### 1. Install Dependencies

```bash
cd contracts/ethereum
npm install
```

### 2. Create Environment File

Create `.env` file in `contracts/ethereum/` directory:

```bash
# Copy template
cp .env.example .env

# Edit with your values
nano .env  # or use your preferred editor
```

### 3. Configure .env File

```env
# Network RPC URLs
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_ALCHEMY_KEY
# Or use Infura: https://sepolia.infura.io/v3/YOUR_INFURA_KEY
# Or use public endpoint: https://rpc.sepolia.org

# Deployer Private Key (NEVER commit this!)
# Get from MetaMask: Account Details > Export Private Key
PRIVATE_KEY=your_private_key_here_without_0x_prefix

# Etherscan API Key (for contract verification)
ETHERSCAN_API_KEY=your_etherscan_api_key_here

# Optional: Gas settings
GAS_PRICE=30  # in gwei (adjust based on network conditions)
```

**Security Warning:**
- ⚠️ NEVER commit `.env` file to git
- ⚠️ NEVER share your private key
- ⚠️ Use test wallets for testnet deployments
- ⚠️ For mainnet, use hardware wallet or multi-sig

### 4. Verify Configuration

```bash
# Check account balance
npx hardhat run scripts/check-balance.js --network sepolia

# Expected output:
# Network: sepolia (Chain ID: 11155111)
# Account: 0x1234...5678
# Balance: 0.5 ETH
```

If balance is 0, get Sepolia ETH from faucets (see Prerequisites).

---

## Deployment Procedure

### Step 1: Pre-Deployment Checks

```bash
# 1. Compile contracts
npx hardhat clean
npx hardhat compile

# 2. Run tests locally
npm test

# 3. Check gas estimation
npx hardhat run scripts/estimate-gas.js --network sepolia
```

**Expected Gas Costs:**
- V4 Deployment: ~2,500,000 gas
- At 30 gwei: ~0.075 ETH
- At 50 gwei: ~0.125 ETH

### Step 2: Deploy to Sepolia

```bash
# Deploy SageRegistryV4
npx hardhat run scripts/deploy_v4.js --network sepolia
```

**Deployment Output:**
```
🚀 Deploying SageRegistryV4 to sepolia...

Network: sepolia (Chain ID: 11155111)
Deployer: 0x1234...5678
Balance: 0.5 ETH

Estimated gas: 2,487,654
Estimated cost: 0.074 ETH @ 30 gwei

✅ Deploying in 5 seconds... (Press Ctrl+C to cancel)

📝 Deploying SageRegistryV4...
⏳ Waiting for confirmations...

✅ SageRegistryV4 deployed!
📍 Address: 0xAbCd...1234
💰 Gas used: 2,456,789
🔗 Transaction: 0x789a...bcde

💾 Deployment info saved to:
   deployments/v4/sepolia-deployment.json

Next steps:
1. Verify contract on Etherscan
2. Update DEPLOYED_ADDRESSES.md
3. Test registration with sage-did CLI
```

**Deployment Info Saved:**
```json
{
  "network": "sepolia",
  "chainId": 11155111,
  "contractAddress": "0xAbCd...1234",
  "transactionHash": "0x789a...bcde",
  "deployer": "0x1234...5678",
  "gasUsed": 2456789,
  "deployedAt": "2025-01-19T10:30:00.000Z",
  "blockNumber": 5123456
}
```

### Step 3: Verify Deployment

```bash
# Check contract on Etherscan
open https://sepolia.etherscan.io/address/0xAbCd...1234

# Test basic read function
npx hardhat console --network sepolia
> const Registry = await ethers.getContractFactory("SageRegistryV4")
> const registry = Registry.attach("0xAbCd...1234")
> await registry.VERSION()
'1.1.0'
```

---

## Verification

### Automatic Verification

```bash
# Verify on Etherscan (automatic)
npx hardhat run scripts/verify_v4.js --network sepolia
```

**Expected Output:**
```
🔍 Verifying SageRegistryV4 on Etherscan...

Contract: 0xAbCd...1234
Network: sepolia

Checking if already verified...
✅ Contract verified successfully!

View on Etherscan:
https://sepolia.etherscan.io/address/0xAbCd...1234#code
```

### Manual Verification (if automatic fails)

```bash
# Method 1: Using Hardhat verify
npx hardhat verify --network sepolia 0xAbCd...1234

# Method 2: Via Etherscan UI
# 1. Go to https://sepolia.etherscan.io/verifyContract
# 2. Enter contract address
# 3. Select compiler version: v0.8.19
# 4. Select optimization: Yes (200 runs)
# 5. Paste contract source code
# 6. Click "Verify and Publish"
```

### Verification Troubleshooting

**Issue: "Already Verified"**
```
✓ Contract already verified
```
No action needed.

**Issue: "Invalid API Key"**
```bash
# Check .env file has correct Etherscan API key
cat .env | grep ETHERSCAN_API_KEY

# Get new key from https://etherscan.io/myapikey
```

**Issue: "Rate Limited"**
```
Wait 30 seconds and try again
```

---

## Post-Deployment

### 1. Update Deployment Records

```bash
# Update DEPLOYED_ADDRESSES.md
nano ../../DEPLOYED_ADDRESSES.md
```

Add entry:
```markdown
### Sepolia Testnet

**Deployed**: 2025-01-19

| Contract | Address | Transaction | Status |
|----------|---------|-------------|--------|
| SageRegistryV4 | `0xAbCd...1234` | [View](https://sepolia.etherscan.io/tx/0x789a...bcde) | ✅ Verified |

**Deployer**: `0x1234...5678`
**Gas Used**: 2,456,789
**Block**: 5,123,456
```

### 2. Test Registration

```bash
# Generate test keys
cd ../../
./scripts/generate-test-keys.sh

# Register test agent
sage-did register \
  --chain ethereum \
  --rpc-url https://sepolia.infura.io/v3/YOUR_KEY \
  --contract 0xAbCd...1234 \
  --name "Test Agent V4" \
  --endpoint "https://test.agent.com" \
  --keys test-keys/ecdsa.pem,test-keys/ed25519.jwk

# Verify registration
sage-did key list did:sage:ethereum:0x...
```

### 3. Configure Go Backend

Update `pkg/agent/did/manager.go` or config file:

```go
config := &did.RegistryConfig{
    ContractAddress: "0xAbCd...1234",  // Sepolia V4 address
    RPCEndpoint:     "https://sepolia.infura.io/v3/YOUR_KEY",
    PrivateKey:      os.Getenv("PRIVATE_KEY"),
    ChainID:         11155111, // Sepolia
}
```

### 4. Integration Tests

```bash
# Run integration tests against Sepolia
cd contracts/ethereum
SEPOLIA_CONTRACT=0xAbCd...1234 npm run test:integration

# Run Go integration tests
cd ../../
SAGE_INTEGRATION_TEST=1 \
CONTRACT_ADDRESS=0xAbCd...1234 \
RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY \
go test ./pkg/agent/did/ethereum/... -v
```

### 5. Commit Deployment Info

```bash
git add deployments/v4/sepolia-deployment.json
git add contracts/DEPLOYED_ADDRESSES.md
git commit -m "deploy: SageRegistryV4 to Sepolia testnet

- Deployed at: 0xAbCd...1234
- Transaction: 0x789a...bcde
- Gas used: 2,456,789
- Verified on Etherscan"

git push origin dev
```

---

## Mainnet Deployment

### Additional Precautions for Mainnet

**Before Mainnet Deployment:**

1. **Security Audit**
   - [ ] Complete external security audit
   - [ ] Fix all critical and high severity issues
   - [ ] Document all acknowledged risks

2. **Testing**
   - [ ] Run full test suite (100% pass)
   - [ ] Complete Sepolia deployment and testing
   - [ ] Run for at least 2 weeks on testnet
   - [ ] Test all critical paths

3. **Code Freeze**
   - [ ] No changes after final testnet deployment
   - [ ] Tag release version in git
   - [ ] Document exact commit hash

4. **Preparation**
   - [ ] Use hardware wallet (Ledger/Trezor)
   - [ ] Consider multi-sig for ownership
   - [ ] Prepare pause mechanism
   - [ ] Set up monitoring and alerts
   - [ ] Prepare incident response plan

5. **Verification**
   - [ ] Multiple team members review deployment plan
   - [ ] Dry run deployment process
   - [ ] Verify contract bytecode matches source

### Mainnet Deployment Steps

```bash
# 1. Final compilation
npx hardhat clean
npx hardhat compile

# 2. Verify bytecode matches
npx hardhat run scripts/verify-bytecode.js

# 3. Deploy to mainnet (requires confirmation)
npx hardhat run scripts/deploy_v4.js --network mainnet

# 4. Immediate verification
npx hardhat run scripts/verify_v4.js --network mainnet

# 5. Smoke test (read-only)
npx hardhat run scripts/test-deployed.js --network mainnet

# 6. Transfer ownership (if using multi-sig)
npx hardhat run scripts/transfer-ownership.js --network mainnet
```

**Post-Mainnet Checklist:**
- [ ] Contract verified on Etherscan
- [ ] Ownership transferred to multi-sig
- [ ] Test agent registered successfully
- [ ] Monitoring enabled
- [ ] Documentation updated
- [ ] Announcement prepared

---

## Troubleshooting

### Common Issues

#### 1. Insufficient Funds

**Error:**
```
Error: insufficient funds for intrinsic transaction cost
```

**Solution:**
```bash
# Check balance
npx hardhat run scripts/check-balance.js --network sepolia

# Get more Sepolia ETH from faucets
# Need at least 0.1 ETH for deployment + gas buffer
```

#### 2. Nonce Too Low

**Error:**
```
Error: nonce too low
```

**Solution:**
```bash
# Reset nonce in MetaMask:
# Settings > Advanced > Reset Account

# Or specify nonce manually in deployment script
```

#### 3. Gas Price Too Low

**Error:**
```
Error: replacement transaction underpriced
```

**Solution:**
```bash
# Check current gas prices
# https://sepolia.etherscan.io/gastracker

# Update .env
GAS_PRICE=50  # increase if needed
```

#### 4. RPC Connection Issues

**Error:**
```
Error: could not detect network
```

**Solution:**
```bash
# Test RPC endpoint
curl -X POST https://sepolia.infura.io/v3/YOUR_KEY \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Try alternative RPC:
# - Alchemy: https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY
# - Public: https://rpc.sepolia.org
```

#### 5. Contract Already Deployed

**Error:**
```
Error: contract creation code storage out of gas
```

**Solution:**
```bash
# Check if contract already deployed
npx hardhat run scripts/check-deployment.js --network sepolia

# If needed, use existing deployment
# Or deploy with different deployer address
```

#### 6. Verification Fails

**Error:**
```
Error: Failed to verify contract
```

**Solutions:**
```bash
# Wait 30 seconds after deployment
sleep 30

# Try manual verification
npx hardhat verify --network sepolia 0xCONTRACT_ADDRESS

# Check Etherscan API status
# https://etherscan.io/apis

# Use Etherscan GUI verification as fallback
```

### Getting Help

**Resources:**
- Hardhat Docs: https://hardhat.org/docs
- Etherscan API: https://docs.etherscan.io/
- SAGE Issues: https://github.com/SAGE-X-project/sage/issues

**Support Channels:**
- GitHub Issues (technical problems)
- GitHub Discussions (questions)
- Project documentation

---

## Deployment Checklist

Use this checklist for each deployment:

### Pre-Deployment
- [ ] Code compiled without errors
- [ ] All tests passing (local)
- [ ] .env file configured correctly
- [ ] Deployer account has sufficient balance
- [ ] RPC endpoint accessible
- [ ] Etherscan API key valid

### Deployment
- [ ] Gas estimation reviewed
- [ ] Deployment transaction confirmed
- [ ] Contract address recorded
- [ ] Transaction hash saved
- [ ] Deployment info file created

### Verification
- [ ] Contract verified on Etherscan
- [ ] Source code visible
- [ ] Read/Write functions accessible
- [ ] Contract matches local compiled bytecode

### Post-Deployment
- [ ] DEPLOYED_ADDRESSES.md updated
- [ ] Test agent registered successfully
- [ ] Integration tests passing
- [ ] Go backend configured
- [ ] Deployment committed to git
- [ ] Team notified

---

## Network Details

### Sepolia Testnet

| Property | Value |
|----------|-------|
| Chain ID | 11155111 |
| Currency | SepoliaETH (test ETH) |
| Block Explorer | https://sepolia.etherscan.io |
| RPC (Public) | https://rpc.sepolia.org |
| RPC (Alchemy) | https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY |
| RPC (Infura) | https://sepolia.infura.io/v3/YOUR_KEY |
| Faucets | Multiple (see Prerequisites) |

### Ethereum Mainnet

| Property | Value |
|----------|-------|
| Chain ID | 1 |
| Currency | ETH |
| Block Explorer | https://etherscan.io |
| RPC (Public) | https://eth.llamarpc.com |
| RPC (Alchemy) | https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY |
| RPC (Infura) | https://mainnet.infura.io/v3/YOUR_KEY |

---

## Security Best Practices

1. **Private Keys**
   - Never share or commit private keys
   - Use environment variables only
   - Consider hardware wallets for mainnet

2. **RPC Endpoints**
   - Use authenticated endpoints (Alchemy/Infura)
   - Avoid public RPC for production
   - Monitor API usage limits

3. **Gas Management**
   - Set reasonable gas limits
   - Monitor gas prices
   - Use gas estimation before deployment

4. **Contract Verification**
   - Always verify immediately after deployment
   - Keep source code backup
   - Document compiler settings

5. **Access Control**
   - Transfer ownership to multi-sig
   - Use timelock for critical operations
   - Document admin procedures

---

**Next Steps:**
1. Complete this deployment guide
2. Deploy to Sepolia testnet
3. Test thoroughly
4. Prepare for mainnet deployment

**See Also:**
- [DEPLOYED_ADDRESSES.md](../../DEPLOYED_ADDRESSES.md) - Deployment tracking
- [README.md](../README.md) - Main documentation
- [ROADMAP.md](../../ROADMAP.md) - Feature roadmap
