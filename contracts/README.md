# SAGE Smart Contracts

Smart contracts for SAGE (Secure Agent Guarantee Engine) AI Agent Registry on Ethereum-compatible blockchains.

## Overview

The SAGE contracts provide a decentralized registry for AI agents with ERC-8004 compliance and comprehensive security features.

**Current Status**: Production Ready - V4.1 (AgentCard Architecture)

### Core Features

- **ERC-8004 Compliant**: Full implementation of ERC-8004 Identity Registry standard
- **Multi-Chain Support**: Deployable on 12 EVM-compatible networks (Ethereum, Kaia, BSC, Base, Arbitrum, Optimism)
- **Multi-Key Architecture**: Support for ECDSA, Ed25519, and X25519 keys (up to 10 keys per agent)
- **Commit-Reveal Pattern**: Front-running protection for agent registration
- **Key Management**: Add, revoke, and rotate keys independently
- **Agent Lifecycle**: Complete lifecycle management (registration, updates, deactivation)
- **DID-Based Identity**: Decentralized identifiers with format validation
- **Gas Optimized**: Efficient storage patterns and streamlined operations
- **Comprehensive Testing**: 124 passing tests across 5 test phases

## Architecture

### Contract Structure

```
contracts/ethereum/
├── AgentCardRegistry.sol      # Main registry with ERC-8004 integration
├── AgentCardStorage.sol       # Storage layer for agent data
├── AgentCardVerifyHook.sol    # DID and key validation hook
├── erc-8004/                  # ERC-8004 standard interfaces
│   └── interfaces/
│       └── IERC8004IdentityRegistry.sol
└── deprecated/                # Legacy contracts (DO NOT USE)
```

### AgentCardRegistry

The core contract implementing ERC-8004 standard with SAGE-specific features:

- **Commit-Reveal Registration**: Two-phase registration to prevent front-running
- **Multi-Key Support**: Up to 10 keys per agent (ECDSA, Ed25519, X25519)
- **Key Management**: Add, revoke, and rotate keys
- **Agent Lifecycle**: Activation, updates, and deactivation
- **Security Features**: Pausable, reentrancy protection, two-step ownership

### AgentCardVerifyHook

Pre-registration validation hook enforcing:

- DID format validation (`did:sage:{chain}:{address}`)
- Public key format verification
- Key type validation
- Zero-key prevention

## Quick Start

### 1. Installation

```bash
cd contracts/ethereum

# Install dependencies
npm install

# Compile contracts
npm run compile

# Run tests (124 tests)
npm test
```

### 2. Environment Setup

```bash
# Copy example environment file
cp .env.example .env

# Edit .env and configure networks
nano .env
```

Required environment variables:

```env
# Private Keys
PRIVATE_KEY=your_testnet_private_key_without_0x
MAINNET_PRIVATE_KEY=your_mainnet_private_key_without_0x

# RPC URLs (configure networks you want to use)
ETHEREUM_SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR-KEY
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io

# API Keys for Contract Verification
ETHERSCAN_API_KEY=your_etherscan_api_key
```

## Deployment

### Supported Networks (12 Total)

**Testnets (6)**:
- Ethereum Sepolia
- Kaia Kairos
- BSC Testnet
- Base Sepolia
- Arbitrum Sepolia
- Optimism Sepolia

**Mainnets (6)**:
- Ethereum Mainnet
- Kaia Mainnet (Cypress)
- BSC Mainnet
- Base Mainnet
- Arbitrum One
- Optimism Mainnet

### Deploy to Testnet (Recommended)

```bash
# Check balance first
npm run check-balance

# Deploy to Kaia Kairos (recommended for testing)
npm run deploy:kaia:kairos

# Verify contracts
npm run verify:kaia:kairos
```

### Deploy to Other Networks

```bash
# Ethereum
npm run deploy:ethereum:sepolia
npm run deploy:ethereum:mainnet

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

# Local Hardhat Network
npm run node              # Terminal 1
npm run deploy:localhost  # Terminal 2
```

### Deployment Output

After successful deployment:

```
================================================================================
Deployment Complete!
================================================================================

Summary:
   Network:              kaia-kairos
   Chain ID:             1001
   AgentCardRegistry:    0x...
   AgentCardVerifyHook:  0x...
   Total Gas Used:       ~5,000,000

Next Steps:
   1. Verify contracts on block explorer
   2. Run: npm run verify:kaia:kairos
   3. Test agent registration
```

Deployment records saved in:
- `deployments/{network}-agentcard-{timestamp}.json`
- `deployments/{network}-latest.json`

## Security Features

### Core Security

- **Commit-Reveal Pattern**: Two-phase registration prevents front-running
- **Reentrancy Guards**: Protection against reentrancy attacks
- **Pausable Functionality**: Emergency pause mechanism
- **Two-Step Ownership**: Safe ownership transfer
- **DID Format Validation**: Enforces proper DID structure
- **Public Key Validation**: Multi-format key verification
- **Zero-Key Prevention**: Blocks invalid key submissions
- **Key Ownership Verification**: Proof-of-possession for keys

### Audit Status

- Code review: Complete
- Test coverage: 124/124 tests passing
- Security audit: Pending external audit

**Important**: Use testnets for initial deployment and testing. Deploy to mainnet only after thorough testing.

## Usage Examples

See `ethereum/README.md` for detailed usage examples including:

- Agent registration with commit-reveal pattern
- Multi-key management (add, revoke, rotate)
- Agent lifecycle operations (update, deactivate)
- ERC-8004 interface usage
- Query and resolution examples

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

## Testing

### Test Coverage (124 Tests Total)

```bash
cd contracts/ethereum

# Run all tests
npm test

# Expected output: 124 passing

# Run with coverage
npm run coverage
```

### Test Phases

| Phase | Test File | Tests | Coverage |
|-------|-----------|-------|----------|
| Phase 1 | AgentCardStorage.test.js | 25 | Storage layer, structs, events |
| Phase 2 | AgentCardVerifyHook.test.js | 36 | DID validation, key verification |
| Phase 3 | AgentCardRegistry.test.js | 47 | Registration, key mgmt, security |
| Phase 4 | ERC8004InterfaceInRegistry.test.js | 8 | ERC-8004 compliance |
| Phase 5 | FullWorkflow.test.js | 8 | Integration workflows |

### Key Test Areas

- Commit-reveal pattern (9 tests)
- Multi-key registration (12 tests)
- Key management (9 tests)
- Agent lifecycle (10 tests)
- Security features (8 tests)
- DID validation (10 tests)
- Key verification (13 tests)
- ERC-8004 interface (8 tests)
- Full workflows (8 tests)

## Available NPM Scripts

See `ethereum/README.md` for complete list of NPM scripts.

**Key Scripts:**

```bash
# Build & Test
npm run compile          # Compile contracts
npm test                 # Run all tests (124 passing)
npm run coverage         # Generate coverage report

# Deployment (Multi-Chain)
npm run deploy:ethereum:mainnet
npm run deploy:kaia:kairos
npm run deploy:bsc:mainnet
# ... (12 networks supported)

# Verification
npm run verify:ethereum:mainnet
npm run verify:kaia:kairos
# ... (same pattern for all networks)

# Development Tools
npm run node             # Start local Hardhat node
npm run check-balance    # Check deployer balance
npm run flatten          # Flatten contracts for verification
```

## Project Structure

```
contracts/
├── ethereum/                # Main contracts directory
│   ├── contracts/          # Solidity contracts
│   │   ├── AgentCardRegistry.sol
│   │   ├── AgentCardStorage.sol
│   │   ├── AgentCardVerifyHook.sol
│   │   ├── erc-8004/       # ERC-8004 interfaces
│   │   └── deprecated/     # Legacy contracts
│   ├── test/               # Test files (124 tests)
│   │   ├── AgentCardRegistry.test.js
│   │   ├── AgentCardStorage.test.js
│   │   ├── AgentCardVerifyHook.test.js
│   │   ├── ERC8004InterfaceInRegistry.test.js
│   │   └── FullWorkflow.test.js
│   ├── scripts/            # Deployment scripts
│   │   ├── deploy-agentcard.js
│   │   ├── verify-agentcard.js
│   │   └── check-balance.js
│   ├── deployments/        # Deployment records
│   ├── hardhat.config.js   # Network configuration
│   └── README.md           # Detailed documentation
└── README.md               # This file
```

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

## Troubleshooting

See `ethereum/README.md` for detailed troubleshooting guide including:

- Deployment issues
- Contract verification
- Network connection problems
- Test failures

## Resources

### Documentation

- [Ethereum Implementation Guide](ethereum/README.md)
- [ERC-8004 Standard](ethereum/contracts/erc-8004/README.md)
- [Deployment Records](ethereum/deployments/README.md)

### External Resources

- [Hardhat Documentation](https://hardhat.org/docs)
- [OpenZeppelin Contracts](https://docs.openzeppelin.com/contracts)
- [Solidity Documentation](https://docs.soliditylang.org/)
- [Kaia Developer Docs](https://docs.kaia.io)
- [Ethereum Development Resources](https://ethereum.org/developers)

### Network Faucets

- Ethereum Sepolia: https://sepoliafaucet.com/
- Kaia Kairos: https://kairos.wallet.kaia.io/faucet
- Base Sepolia: https://www.coinbase.com/faucets/base-sepolia-faucet
- Arbitrum Sepolia: https://faucet.quicknode.com/arbitrum/sepolia
- Optimism Sepolia: https://app.optimism.io/faucet

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new features
4. Ensure all tests pass (124/124)
5. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed guidelines.

## License

- **Smart Contracts**: MIT License
- **Go Code**: LGPL-v3 License

See [LICENSE](../LICENSE) for details.

## Support

- **GitHub Issues**: https://github.com/sage-x-project/sage/issues
- **Documentation**: https://github.com/sage-x-project/sage/tree/main/docs

---

**Last Updated**: 2025-10-31
**Version**: 4.1 (AgentCard Architecture)
**Status**: Production Ready
