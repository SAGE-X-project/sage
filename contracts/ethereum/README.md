# SAGE Smart Contracts for Kaia Network

## Overview
SAGE Registry smart contracts for managing AI agent identities on Kaia blockchain.

### 🆕 Version 2 Features
- **Enhanced Public Key Validation**: 5-step verification process
- **Key Revocation**: Ability to revoke compromised keys
- **Zero-Key Prevention**: Protection against invalid keys
- **Ownership Proof**: Signature-based key ownership verification
- **Ed25519 Rejection**: Explicit rejection of unsupported key types

## 🚀 Quick Start

### 1. Install Dependencies
```bash
npm install
```

### 2. Set Up Environment
```bash
# Copy example environment file
cp .env.example .env

# Edit .env and add your private key (without 0x prefix)
nano .env
```

### 3. Get Test KLAY
Get test KLAY from Kairos faucet: https://kairos.wallet.kaia.io/faucet

### 4. Run Quick Start Script
```bash
# Interactive setup and deployment
./bin/quick-start.sh
```

## 📁 Project Structure

```
sage/contracts/ethereum/
├── contracts/               # Smart contracts
│   ├── SageRegistry.sol     # V1 Registry (deprecated)
│   ├── SageRegistryV2.sol   # V2 Registry with enhanced validation
│   ├── SageVerificationHook.sol
│   └── interfaces/
├── scripts/                 # Deployment and utility scripts
│   ├── deploy-v2.js        # Main deployment script
│   ├── quick-test.js       # Quick testing script
│   └── check-balance.js   # Balance checker
├── test/                   # Test files
│   ├── SageRegistryV2.test.js
│   └── integration-v2.test.js
├── bin/                    # Shell scripts
│   ├── quick-start.sh      # Interactive setup
│   ├── deploy-v2.sh        # Deployment helper
│   ├── compile.sh          # Compilation script
│   ├── test.sh             # Test runner
│   └── test-v2.sh          # V2-specific tests
├── deployments/            # Deployment records
├── hardhat.config.js       # Hardhat configuration
└── package.json

```

## 🛠 Available Scripts

### Shell Scripts (in `bin/` directory)

#### **quick-start.sh** - Interactive Setup Assistant
```bash
./bin/quick-start.sh
```
Features:
- Checks prerequisites
- Installs dependencies
- Configures environment
- Runs tests
- Provides deployment options

#### **deploy-v2.sh** - Deployment Helper
```bash
./bin/deploy-v2.sh
```
Options:
1. Local Hardhat Network
2. Kaia Testnet (Kairos)
3. Kaia Mainnet

#### **compile.sh** - Compile Contracts
```bash
./bin/compile.sh
```
Compiles all contracts with size report.

#### **test-v2.sh** - Run V2 Tests
```bash
./bin/test-v2.sh
```
Runs comprehensive tests for SageRegistryV2.

### NPM Commands

#### Development
- `npm run compile` - Compile smart contracts
- `npm run clean` - Clean build artifacts
- `npm run test` - Run all tests
- `npm run test:v2` - Run V2 tests only
- `npm run coverage` - Generate coverage report
- `npm run size` - Check contract sizes

#### Deployment
- `npm run deploy:local` - Deploy to local network
- `npm run deploy:kairos` - Deploy to Kairos testnet
- `npm run deploy:kaia` - Deploy to Kaia mainnet

#### Utilities
- `npm run node` - Start local Hardhat node
- `npm run console` - Open Hardhat console
- `npm run quick-test` - Run quick test script

## 📝 Testing Guide

### Local Testing
```bash
# Start local node (Terminal 1)
npm run node

# Run tests (Terminal 2)
npm run test

# Quick test with deployment
npx hardhat run scripts/quick-test.js --network localhost
```

### Testnet Testing
```bash
# Check balance first
npx hardhat run scripts/check-balance.js --network kairos

# Deploy to testnet
npm run deploy:kairos

# Test deployed contract
npx hardhat run scripts/test-deployed.js --network kairos
```

### Specific Test Suites
```bash
# V2 tests only
./bin/test-v2.sh

# Integration tests
npx hardhat test test/integration-v2.test.js

# With gas reporting
REPORT_GAS=true npx hardhat test
```

## 🚀 Deployment Process

### Step 1: Prepare Environment
```bash
# Check prerequisites
./bin/quick-start.sh

# Verify configuration
cat .env | grep -v PRIVATE_KEY
```

### Step 2: Deploy Contracts
```bash
# Interactive deployment
./bin/deploy-v2.sh

# Or direct deployment
npx hardhat run scripts/deploy-v2.js --network kairos
```

### Step 3: Verify Contracts
```bash
# Verify on block explorer
npx hardhat verify --network kairos CONTRACT_ADDRESS
```

### Step 4: Test Deployment
```bash
# Quick test
npx hardhat run scripts/test-deployed.js --network kairos

# Interactive console
npx hardhat console --network kairos
```

## 🔧 Configuration

### Network Configuration

#### Kairos Testnet
- **Chain ID**: 1001
- **RPC URL**: https://public-en-kairos.node.kaia.io
- **Explorer**: https://kairos.kaiascope.com
- **Faucet**: https://kairos.wallet.kaia.io/faucet

#### Kaia Mainnet
- **Chain ID**: 8217
- **RPC URL**: https://public-en-cypress.klaytn.net
- **Explorer**: https://kaiascope.com

### Environment Variables (.env)
```env
# Required
PRIVATE_KEY=your_private_key_without_0x

# Network RPCs
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io
CYPRESS_RPC_URL=https://public-en-cypress.klaytn.net

# Optional
KAIASCOPE_API_KEY=your_api_key
```

## 🔐 Security Features

### SageRegistryV2 Enhancements
1. **Public Key Validation**
   - Format checking (0x04, 0x02, 0x03)
   - Zero-key prevention
   - Length validation (32-65 bytes)

2. **Ownership Verification**
   - Challenge-response signature
   - Private key possession proof
   - Replay attack protection

3. **Key Management**
   - Key revocation capability
   - Automatic agent deactivation
   - Revoked key tracking

### Gas Usage
- Registration: ~620K gas
- Update: ~50K gas
- Revocation: ~30K gas

## 🐛 Troubleshooting

### Common Issues

#### Compilation Warnings
✅ Fixed in latest version - no warnings

#### "Insufficient funds" Error
```bash
# Check balance
npx hardhat run scripts/check-balance.js --network kairos

# Get test KLAY
# Visit: https://kairos.wallet.kaia.io/faucet
```

#### "Key ownership not proven" Error
Ensure signature generation matches contract expectations:
```javascript
const challenge = keccak256(solidityPacked(
  ["string", "uint256", "address", "address", "bytes32"],
  ["SAGE Key Registration:", chainId, contractAddress, signerAddress, keyHash]
));
```

#### Network Connection Issues
```bash
# Test connection
npx hardhat run scripts/test-connection.js --network kairos

# Check RPC status
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  https://public-en-kairos.node.kaia.io
```

## 📚 Documentation

- [Deployment Guide](DEPLOYMENT_GUIDE.md) - Detailed deployment instructions
- [Migration Guide](MIGRATION_GUIDE_V2.md) - V1 to V2 migration
- [Code Review](CODE_REVIEW_V2.md) - Security analysis
- [Key Validation](KEY_VALIDATION_IMPROVEMENTS.md) - Technical improvements

## 🌐 Resources

- [Kaia Documentation](https://docs.kaia.io)
- [Kaia Explorer](https://kaiascope.com)
- [Hardhat Documentation](https://hardhat.org)
- [Ethers.js Documentation](https://docs.ethers.org)

## 📄 License
MIT