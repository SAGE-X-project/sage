# SAGE Contract Deployment Guide

## 📋 Overview

This guide explains how to deploy and manage SAGE smart contracts across different networks.

## 🚀 Quick Start

### 1. Local Development

```bash
# Start local node (in separate terminal)
npm run node

# Deploy contracts
npm run deploy:unified

# Or deploy to running localhost
npm run deploy:unified:local
```

### 2. Testnet Deployment (Kairos)

```bash
# Set environment variables
cp .env.example .env
# Edit .env with your private key

# Deploy to Kairos testnet
npm run deploy:unified:kairos

# Verify contracts
npm run verify:kairos
```

### 3. Production Deployment

```bash
# Use production private key
export MAINNET_PRIVATE_KEY=your_key_here

# Deploy to mainnet
npx hardhat run scripts/deploy-unified.js --network kaia
```

## 📁 Project Structure

```
contracts/
├── ethereum/
│   ├── contracts/
│   │   ├── SageRegistryV2.sol      # Main registry contract
│   │   ├── SageRegistryTest.sol    # Test version with simplified validation
│   │   └── SageVerificationHook.sol # Hook for additional verification
│   ├── scripts/
│   │   ├── deploy-unified.js       # Main deployment script
│   │   └── register-agents.js      # Agent registration script
│   └── deployments/                # Deployment artifacts (auto-generated)
│       ├── {network}.json          # Network-specific deployment info
│       ├── {network}.env           # Environment variables
│       └── latest.json             # Most recent deployment
```

## 🔧 Configuration

### Environment Variables

Create a `.env` file:

```env
# Private Keys
PRIVATE_KEY=your_test_private_key
MAINNET_PRIVATE_KEY=your_production_key

# Network RPC (optional - uses defaults)
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io
KAIA_RPC_URL=https://public-en.node.kaia.io

# Gas Configuration
GAS_PRICE_GWEI=250
GAS_LIMIT=3000000

# Contract Addresses (set after deployment)
SAGE_REGISTRY_ADDRESS=0x...
SAGE_VERIFICATION_HOOK_ADDRESS=0x...
```

### Network Configuration

Networks are pre-configured in `hardhat.config.js`:

- **hardhat**: Local in-memory network
- **localhost**: External local node (port 8545)
- **kairos**: Kaia testnet (Chain ID: 1001)
- **kaia/cypress**: Kaia mainnet (Chain ID: 8217)

## 📝 Deployment Process

### Unified Deployment Script

The `deploy-unified.js` script handles:

1. **Contract Deployment**
   - Deploys SageRegistryV2 (or SageRegistryTest for local)
   - Deploys SageVerificationHook
   - Configures hooks

2. **Agent Registration** (for test networks)
   - Registers Root Agent
   - Registers Ordering Agent
   - Registers Planning Agent

3. **Information Management**
   - Saves deployment info to JSON
   - Generates environment variables
   - Creates verification data

4. **Contract Verification** (optional)
   - Verifies on block explorer

### Deployment Output

After successful deployment:

```
deployments/
├── {network}.json          # Complete deployment information
├── {network}.env          # Environment variables to use
├── latest.json            # Link to most recent deployment
└── agents/
    └── {network}-agents-*.json  # Registered agent details
```

## 🤖 Agent Management

### Register Agents Separately

```bash
# Register agents on deployed contracts
npx hardhat run scripts/register-agents.js --network {network}

# With specific registry address
npx hardhat run scripts/register-agents.js --registryAddress 0x...
```

### Agent Configuration

Agents are configured in `scripts/register-agents.js`:

```javascript
const AGENT_CONFIGS = {
  local: [
    {
      did: "did:sage:local:root",
      name: "Root Agent",
      endpoint: "http://localhost:3001",
      capabilities: ["routing", "management"]
    }
    // ...
  ]
}
```

## 🔍 Verification

### Verify Deployment

```bash
# Check deployment status
npx hardhat run scripts/interact-local.js --network {network}

# Query registered agents
npx hardhat run scripts/query-agents.js --network {network}
```

### Verify on Explorer

```bash
# Kairos testnet
npm run verify:kairos

# Mainnet
npm run verify:kaia
```

## 🔐 Security Considerations

### Local Development
- Uses SageRegistryTest with simplified validation
- Test mode allows easy agent registration
- NOT FOR PRODUCTION USE

### Production
- Uses full SageRegistryV2 with cryptographic validation
- Requires proper key ownership proof
- Implements registration cooldown
- Maximum agents per address limit

## 🛠️ Troubleshooting

### Common Issues

1. **"Key ownership not proven"**
   - Ensure signature matches the public key owner
   - Check you're using correct contract version

2. **"Registration cooldown active"**
   - Wait for cooldown period (default: 1 block)
   - Use different addresses for multiple agents

3. **"Maximum agents reached"**
   - Each address can register limited agents
   - Use multiple addresses if needed

4. **Compilation errors**
   - Run `npm run clean` then `npm run compile`
   - Check Solidity version compatibility

### Gas Optimization

- Batch agent registrations when possible
- Use appropriate gas limits:
  - Local: 3,000,000
  - Testnet: 5,000,000
  - Mainnet: 8,000,000

## 📊 Integration with Go Applications

### Load Deployment Info

```go
import "sage/config"

// Load blockchain configuration
cfg, err := config.LoadConfig("kairos")

// Contract address is loaded automatically from:
// 1. SAGE_REGISTRY_ADDRESS env var
// 2. deployments/{network}.json
// 3. Fallback to known addresses
```

### Update After Deployment

```bash
# Copy environment variables
cp sage/contracts/ethereum/deployments/{network}.env sage/.env

# Or export directly
export SAGE_REGISTRY_ADDRESS=0x...
```

## 📚 Additional Resources

- [Hardhat Documentation](https://hardhat.org/docs)
- [Kaia Network Docs](https://docs.kaia.io)
- [SAGE Protocol Specification](../README.md)

## 🆘 Support

For issues or questions:
1. Check deployment logs in `deployments/`
2. Review error messages in console
3. Verify network connectivity
4. Ensure sufficient balance for gas

---

**Last Updated**: 2025-09-27
**Version**: 1.0.0
