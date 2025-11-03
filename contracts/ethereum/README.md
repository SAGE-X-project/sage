# SAGE AgentCard Contracts

## Overview

This directory contains the AgentCard smart contracts for SAGE (Secure Agent Guarantee Engine) - a multi-chain EVM implementation for decentralized AI agent identity management with ERC-8004 compliance.

**Current Version**: V4.1 (AgentCard Architecture)

### Key Features

- **ERC-8004 Compliant**: Full implementation of ERC-8004 Identity Registry standard
- **Multi-Chain Support**: Deployable on 12 EVM-compatible networks
- **Multi-Key Architecture**: Support for ECDSA, Ed25519, and X25519 keys
- **Commit-Reveal Pattern**: Front-running protection for agent registration
- **Comprehensive Testing**: 124 passing tests across 5 test phases

## Architecture

### Contracts

```
contracts/
├── AgentCardRegistry.sol      # Main registry with ERC-8004 integration
├── AgentCardStorage.sol       # Storage layer for agent data
├── AgentCardVerifyHook.sol    # DID and key validation hook
├── erc-8004/                  # ERC-8004 standard interfaces
│   └── interfaces/
│       └── IERC8004IdentityRegistry.sol
└── deprecated/                # Legacy contracts (DO NOT USE)
```

### AgentCardRegistry

The core contract that implements both SAGE-specific functionality and ERC-8004 standard interface:

- **Commit-Reveal Registration**: Two-phase registration to prevent front-running
- **Multi-Key Support**: Up to 10 keys per agent (ECDSA, Ed25519, X25519)
- **Key Management**: Add, revoke, and rotate keys
- **Agent Lifecycle**: Activation, updates, and deactivation
- **Security Features**: Pausable, reentrancy protection, two-step ownership

### AgentCardVerifyHook

Pre-registration validation hook that enforces:

- DID format validation (`did:sage:{chain}:{address}`)
- Public key format verification
- Key type validation (ECDSA, Ed25519, X25519)
- Zero-key prevention

## Supported Networks

### Production Mainnets (6)

| Network | Chain ID | RPC Configuration | Explorer |
|---------|----------|-------------------|----------|
| Ethereum Mainnet | 1 | `ETHEREUM_MAINNET_RPC_URL` | etherscan.io |
| Kaia Mainnet (Cypress) | 8217 | `KAIA_RPC_URL` | kaiascan.io |
| BSC Mainnet | 56 | `BSC_MAINNET_RPC_URL` | bscscan.com |
| Base Mainnet | 8453 | `BASE_MAINNET_RPC_URL` | basescan.org |
| Arbitrum One | 42161 | `ARBITRUM_MAINNET_RPC_URL` | arbiscan.io |
| Optimism Mainnet | 10 | `OPTIMISM_MAINNET_RPC_URL` | optimistic.etherscan.io |

### Testnets (6)

| Network | Chain ID | RPC Configuration | Explorer |
|---------|----------|-------------------|----------|
| Ethereum Sepolia | 11155111 | `ETHEREUM_SEPOLIA_RPC_URL` | sepolia.etherscan.io |
| Kaia Kairos | 1001 | `KAIROS_RPC_URL` | kairos.kaiascan.io |
| BSC Testnet | 97 | `BSC_TESTNET_RPC_URL` | testnet.bscscan.com |
| Base Sepolia | 84532 | `BASE_TESTNET_RPC_URL` | sepolia.basescan.org |
| Arbitrum Sepolia | 421614 | `ARBITRUM_TESTNET_RPC_URL` | sepolia.arbiscan.io |
| Optimism Sepolia | 11155420 | `OPTIMISM_TESTNET_RPC_URL` | sepolia-optimistic.etherscan.io |

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

### 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit .env and configure:
nano .env
```

Required environment variables:

```bash
# Private Keys
PRIVATE_KEY=your_testnet_private_key_without_0x
MAINNET_PRIVATE_KEY=your_mainnet_private_key_without_0x

# RPC URLs (configure networks you want to use)
ETHEREUM_SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR-KEY
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io

# API Keys for Contract Verification
ETHERSCAN_API_KEY=your_etherscan_api_key
```

### 3. Compile Contracts

```bash
npm run compile
```

### 4. Run Tests

```bash
npm test
```

Expected output: **124 passing tests** - 

## Deployment

### Deploy to Testnet

#### Kaia Kairos (Recommended for Testing)

```bash
# Check your balance first
npm run check-balance

# Deploy to Kairos
npm run deploy:kaia:kairos

# Verify contracts
npm run verify:kaia:kairos
```

#### Ethereum Sepolia

```bash
npm run deploy:ethereum:sepolia
npm run verify:ethereum:sepolia
```

### Deploy to Other Networks

```bash
# BSC
npm run deploy:bsc:testnet
npm run deploy:bsc:mainnet

# Base
npm run deploy:base:sepolia
npm run deploy:base:mainnet

# Arbitrum
npm run deploy:arbitrum:sepolia
npm run deploy:arbitrum:mainnet

# Optimism
npm run deploy:optimism:sepolia
npm run deploy:optimism:mainnet
```

### Deploy to Local Hardhat Network

```bash
# Terminal 1: Start local node
npm run node

# Terminal 2: Deploy
npm run deploy:localhost
```

## Deployment Output

After successful deployment, you'll see:

```
================================================================================
-  Deployment Complete!
================================================================================

 Summary:
   Network:              kaia-kairos
   Chain ID:             1001
   AgentCardRegistry:    0x...
   AgentCardVerifyHook:  0x...
   Total Gas Used:       ~5,000,000

-  Next Steps:
   1. Verify contracts on block explorer
   2. Run: npx hardhat run scripts/verify-agentcard.js --network kairos
   3. Test agent registration
```

Deployment information is saved in:
- `deployments/{network}-agentcard-{timestamp}.json` - Timestamped deployment
- `deployments/{network}-latest.json` - Latest deployment (for verification)

## Deployed Contracts

### Ethereum Sepolia Testnet

#### AgentCard Contracts
- **AgentCardRegistry**: [`0xC7eCF7Ad6ee71CB0d94f0eb00F46f1DDf432a808`](https://sepolia.etherscan.io/address/0xC7eCF7Ad6ee71CB0d94f0eb00F46f1DDf432a808)
- **AgentCardVerifyHook**: [`0xf3be150cd4EC0819bef95890DeeE0B71d9C94F6b`](https://sepolia.etherscan.io/address/0xf3be150cd4EC0819bef95890DeeE0B71d9C94F6b)

#### ERC-8004 Standalone Contracts
- **ERC8004IdentityRegistry**: [`0x5B0763c3649eee889966dF478a73e53Df0420C84`](https://sepolia.etherscan.io/address/0x5B0763c3649eee889966dF478a73e53Df0420C84)
- **ERC8004ReputationRegistry**: [`0xE953B278fd2378BA4987FE07f71575dd3353C9a8`](https://sepolia.etherscan.io/address/0xE953B278fd2378BA4987FE07f71575dd3353C9a8)
- **ERC8004ValidationRegistry**: [`0x97291e2D3023d166878ed45BBD176F92E5Fda098`](https://sepolia.etherscan.io/address/0x97291e2D3023d166878ed45BBD176F92E5Fda098)

**Deployment Date**: November 3, 2025
**Network**: Sepolia Testnet (Chain ID: 11155111)
**Verification Status**: ✅ All contracts verified on Etherscan and Blockscout

**ERC8004ValidationRegistry Configuration**:
- Minimum Stake: 0.01 ETH
- Minimum Validators: 3
- Consensus Threshold: 66%

## Project Structure

```
contracts/ethereum/
├── contracts/               # Smart contracts
│   ├── AgentCardRegistry.sol       -  Main registry with ERC-8004
│   ├── AgentCardStorage.sol        -  Storage layer
│   ├── AgentCardVerifyHook.sol     -  Validation hook
│   ├── erc-8004/                   -  ERC-8004 interfaces
│   └── deprecated/                 - -   Legacy contracts
│
├── scripts/                 # Deployment scripts
│   ├── deploy-agentcard.js         -  Multi-chain deployment
│   ├── verify-agentcard.js         -  Contract verification
│   ├── check-balance.js            -  Balance checker
│   ├── flatten-contracts.sh        -  Contract flattening
│   ├── generate-key-from-mnemonic.js -  Key generation
│   ├── port-manager.sh             -  Local node management
│   └── deprecated/                 - -   Legacy scripts
│
├── test/                    # Test files
│   ├── AgentCardRegistry.test.js   -  47 tests
│   ├── AgentCardStorage.test.js    -  25 tests
│   ├── AgentCardVerifyHook.test.js -  36 tests
│   ├── ERC8004InterfaceInRegistry.test.js -  8 tests
│   ├── FullWorkflow.test.js        -  8 tests
│   └── deprecated/                 - -   Legacy tests (22 failing - expected)
│
├── deployments/             # Deployment records
│   ├── README.md                   -  Multi-chain deployment guide
│   ├── {network}-latest.json        Latest deployment per network
│   └── {network}-agentcard-{timestamp}.json  Historical deployments
│
├── hardhat.config.js        # Hardhat configuration
├── package.json             # NPM scripts and dependencies
└── README.md                # This file
```

## Available NPM Scripts

### Build & Test

```bash
npm run compile          # Compile contracts
npm test                 # Run all tests (124 passing)
npm run test:kairos      # Run tests on Kairos network
npm run coverage         # Generate coverage report
npm run lint             # Lint Solidity code
npm run lint:fix         # Fix linting issues
npm run size             # Check contract sizes
```

### Deployment (Multi-Chain)

```bash
# Ethereum
npm run deploy:ethereum:mainnet
npm run deploy:ethereum:sepolia

# Kaia
npm run deploy:kaia:mainnet
npm run deploy:kaia:kairos

# BSC
npm run deploy:bsc:mainnet
npm run deploy:bsc:testnet

# Base
npm run deploy:base:mainnet
npm run deploy:base:sepolia

# Arbitrum
npm run deploy:arbitrum:mainnet
npm run deploy:arbitrum:sepolia

# Optimism
npm run deploy:optimism:mainnet
npm run deploy:optimism:sepolia

# Local
npm run deploy:localhost
```

### Verification

```bash
# Verify on respective block explorers
npm run verify:ethereum:mainnet
npm run verify:kaia:kairos
npm run verify:bsc:mainnet
# ... (same pattern for all networks)
```

### Development Tools

```bash
npm run node              # Start local Hardhat node
npm run node:stop         # Stop local node
npm run node:restart      # Restart local node
npm run node:status       # Check node status

npm run console           # Open Hardhat console
npm run console:kairos    # Console on Kairos network

npm run check-balance     # Check deployer balance
npm run flatten           # Flatten contracts for verification
```

### Cleanup

```bash
npm run clean             # Clean artifacts and cache
npm run clean:deployments # Remove deployment files
npm run clean:all         # Clean everything
npm run clean:deep        # Clean + remove node_modules
```

## Test Coverage

### Test Phases (124 Tests Total)

| Phase | Test File | Tests | Coverage |
|-------|-----------|-------|----------|
| Phase 1 | AgentCardStorage.test.js | 25 | Storage layer, structs, events |
| Phase 2 | AgentCardVerifyHook.test.js | 36 | DID validation, key verification |
| Phase 3 | AgentCardRegistry.test.js | 47 | Registration, key mgmt, security |
| Phase 4 | ERC8004InterfaceInRegistry.test.js | 8 | ERC-8004 compliance |
| Phase 5 | FullWorkflow.test.js | 8 | Integration workflows |

**Total**: 124/124 passing - 

### Key Test Areas

- -  Commit-reveal pattern (9 tests)
- -  Multi-key registration (12 tests)
- -  Key management (9 tests)
- -  Agent lifecycle (10 tests)
- -  Security features (8 tests)
- -  DID validation (10 tests)
- -  Key verification (13 tests)
- -  ERC-8004 interface (8 tests)
- -  Full workflows (8 tests)

## Gas Costs

Approximate gas costs on Kaia network:

| Operation | Gas Used | Description |
|-----------|----------|-------------|
| Deploy Hook | ~1,000,000 | AgentCardVerifyHook deployment |
| Deploy Registry | ~4,000,000 | AgentCardRegistry deployment |
| Commit Registration | ~100,000 | Phase 1 commitment |
| Reveal Registration | ~500,000 | Phase 2 reveal (1 key) |
| Add Key | ~100,000 | Add additional key |
| Revoke Key | ~50,000 | Revoke a key |
| Update Agent | ~80,000 | Update endpoint/capabilities |
| Deactivate Agent | ~60,000 | Deactivate agent |

**Total deployment cost**: ~5,000,000 gas

## ERC-8004 Compliance

AgentCardRegistry implements the full ERC-8004 Identity Registry interface:

```solidity
interface IERC8004IdentityRegistry {
    function registerAgent(string calldata agentId, string calldata endpoint)
        external returns (bool success);

    function resolveAgent(string calldata agentId)
        external view returns (AgentInfo memory info);

    function resolveAgentByAddress(address agentAddress)
        external view returns (AgentInfo memory info);

    function isAgentActive(string calldata agentId)
        external view returns (bool);

    function updateAgentEndpoint(string calldata agentId, string calldata newEndpoint)
        external returns (bool success);

    function deactivateAgent(string calldata agentId)
        external returns (bool success);
}
```

**Note**: For SAGE-specific features (commit-reveal, multi-key), use the full `AgentCardRegistry` interface.

## Migration from Legacy Contracts

If you're migrating from SageRegistryV2 or SageRegistryV4:

1. **Deprecated Contracts** are in `contracts/deprecated/` directory
2. **Deprecated Scripts** are in `scripts/deprecated/` directory
3. **Deprecated Tests** are in `test/deprecated/` directory

**Do not use legacy contracts for new deployments**. Use AgentCardRegistry instead.

## Security

### Security Features

- -  Commit-reveal pattern (front-running protection)
- -  Reentrancy guards
- -  Pausable functionality
- -  Two-step ownership transfer
- -  DID format validation
- -  Public key validation
- -  Zero-key prevention
- -  Key ownership verification

### Audit Status

- Code review: -  Complete
- Test coverage: -  124/124 tests passing
- Security audit:  Pending external audit

**Important**: Use testnets for initial deployment and testing. Deploy to mainnet only after thorough testing.

## Troubleshooting

### Deployment Fails

```bash
# Check balance
npm run check-balance

# Check network configuration
cat hardhat.config.js | grep -A 10 "kairos:"

# Check .env file
cat .env | grep PRIVATE_KEY
```

### Verification Fails

```bash
# Ensure correct API key
cat .env | grep ETHERSCAN_API_KEY

# Wait 1-2 minutes after deployment
# Block explorers need time to index the contract

# Retry verification
npm run verify:kaia:kairos
```

### Tests Failing

```bash
# Clean and recompile
npm run clean
npm run compile

# Run tests
npm test

# Check for specific test failures
npm test -- --grep "AgentCard"
```

## Resources

### Documentation

- [ERC-8004 Standard](../erc-8004/README.md)
- [Deployment Guide](deployments/README.md)
- [Test Verification Matrix](VERIFICATION_MATRIX.md)
- [Main Contracts README](../README.md)

### Network Faucets

- Ethereum Sepolia: https://sepoliafaucet.com/
- Kaia Kairos: https://kairos.wallet.kaia.io/faucet
- Base Sepolia: https://www.coinbase.com/faucets/base-sepolia-faucet
- Arbitrum Sepolia: https://faucet.quicknode.com/arbitrum/sepolia
- Optimism Sepolia: https://app.optimism.io/faucet

### Block Explorers

- Ethereum: https://etherscan.io / https://sepolia.etherscan.io
- Kaia: https://kaiascan.io / https://kairos.kaiascan.io
- BSC: https://bscscan.com / https://testnet.bscscan.com
- Base: https://basescan.org / https://sepolia.basescan.org
- Arbitrum: https://arbiscan.io / https://sepolia.arbiscan.io
- Optimism: https://optimistic.etherscan.io / https://sepolia-optimistic.etherscan.io

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new features
4. Ensure all tests pass
5. Submit a pull request

## License

- Smart Contracts: MIT License
- Go Code: LGPL-v3 License

See [LICENSE](../../LICENSE) for details.

## Support

- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs

---

**Last Updated**: 2025-11-03
**Version**: 4.1 (AgentCard Architecture)
