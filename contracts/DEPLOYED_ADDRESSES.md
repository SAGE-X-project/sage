# SAGE Contract Deployed Addresses

This document tracks all deployed SAGE smart contracts across different networks.

## Table of Contents

- [SageRegistryV4 (Multi-Key Support)](#sageregistryv4-multi-key-support)
- [Legacy Contracts](#legacy-contracts)
- [ERC-8004 Standalone Contracts](#erc-8004-standalone-contracts)
- [Verification Status](#verification-status)
- [Deployment Scripts](#deployment-scripts)

---

## SageRegistryV4 (Multi-Key Support)

**Current Version:** v4.0.0
**Contract:** `SageRegistryV4.sol`
**Features:** Multi-key support (Ed25519, ECDSA, X25519), A2A protocol compatible

### Mainnet

| Network | Chain ID | Contract Address | Deployer | Deployed | Verified | Explorer |
|---------|----------|------------------|----------|----------|----------|----------|
| Ethereum | 1 | `TBD` | - | - | - | - |

### Testnets

| Network | Chain ID | Contract Address | Deployer | Deployed | Verified | Explorer |
|---------|----------|------------------|----------|----------|----------|----------|
| Sepolia | 11155111 | `TBD` | - | - | - | [View](https://sepolia.etherscan.io) |
| Goerli | 5 | `TBD` | - | - | - | [View](https://goerli.etherscan.io) |

### Local Development

| Network | Chain ID | Contract Address | Notes |
|---------|----------|------------------|-------|
| Localhost | 31337 | *Dynamic* | Use `npx hardhat run scripts/deploy-v4-local.js` |
| Hardhat | 31337 | *Dynamic* | Ephemeral, resets on restart |

---

## Legacy Contracts

### SageRegistryV3

**Status:** Deprecated (use V4 for new deployments)
**Contract:** `SageRegistryV3.sol`
**Features:** Basic DID registration with single key

| Network | Chain ID | Contract Address | Notes |
|---------|----------|------------------|-------|
| Sepolia | 11155111 | `0x...` | Maintained for backward compatibility |

### SageRegistryV2

**Status:** Deprecated
**Contract:** `SageRegistryV2.sol`

| Network | Chain ID | Contract Address | Notes |
|---------|----------|------------------|-------|
| Sepolia | 11155111 | `0x...` | Legacy - Do not use for new agents |

---

## ERC-8004 Standalone Contracts

These contracts implement the ERC-8004 standard independently and can be used without the SAGE registry.

### ERC8004IdentityRegistry

**Contract:** `ERC8004IdentityRegistry.sol`
**Features:** Standalone agent identity management

| Network | Chain ID | Contract Address | Deployer | Deployed | Verified |
|---------|----------|------------------|----------|----------|----------|
| Sepolia | 11155111 | `TBD` | - | - | - |

### ERC8004ReputationRegistry

**Contract:** `ERC8004ReputationRegistry.sol`
**Features:** Task authorization and feedback submission

| Network | Chain ID | Contract Address | Deployer | Deployed | Verified |
|---------|----------|------------------|----------|----------|----------|
| Sepolia | 11155111 | `TBD` | - | - | - |

---

## Verification Status

All contracts deployed to public networks should be verified on Etherscan for transparency.

### Verification Commands

```bash
# Verify SageRegistryV4
npx hardhat run scripts/verify_v4.js --network sepolia

# Verify with explicit address
CONTRACT_ADDRESS=0x... npx hardhat run scripts/verify_v4.js --network sepolia
```

### Verification Checklist

- [ ] SageRegistryV4 on Sepolia
- [ ] SageRegistryV4 on Mainnet (when deployed)
- [ ] ERC8004IdentityRegistry on Sepolia
- [ ] ERC8004ReputationRegistry on Sepolia

---

## Deployment Scripts

### Deploy SageRegistryV4

```bash
# Local deployment with tests
npx hardhat run scripts/deploy-v4-local.js

# Sepolia testnet deployment
npx hardhat run scripts/deploy_v4.js --network sepolia

# Mainnet deployment (requires confirmation)
npx hardhat run scripts/deploy_v4.js --network mainnet
```

### Environment Variables

Create a `.env` file in `contracts/ethereum/`:

```bash
# Ethereum RPC endpoints
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_API_KEY
MAINNET_RPC_URL=https://mainnet.infura.io/v3/YOUR_API_KEY

# Private key for deployment (keep secure!)
DEPLOYER_PRIVATE_KEY=0x...

# Etherscan API key for verification
ETHERSCAN_API_KEY=YOUR_ETHERSCAN_API_KEY

# Optional: Initial owner address (defaults to deployer)
INITIAL_OWNER=0x...
```

### Post-Deployment Steps

1. **Verify Contract:**
   ```bash
   npx hardhat run scripts/verify_v4.js --network sepolia
   ```

2. **Update This File:**
   - Add contract address, deployer, and deployment date
   - Add Etherscan verification link
   - Update verification status

3. **Update Go SDK Defaults:**
   ```bash
   # Update default contract addresses in:
   - cmd/sage-did/register.go (getDefaultContractAddress)
   - pkg/agent/did/ethereum/client.go
   ```

4. **Test Deployment:**
   ```bash
   REGISTRY_ADDRESS=0x... go test ./pkg/agent/did/ethereum -v
   ```

5. **Announce Deployment:**
   - Update README.md
   - Update documentation
   - Notify team/community

---

## Network Information

### Ethereum Mainnet
- **Chain ID:** 1
- **RPC:** https://mainnet.infura.io/v3/YOUR_KEY
- **Explorer:** https://etherscan.io
- **Gas Token:** ETH

### Sepolia Testnet
- **Chain ID:** 11155111
- **RPC:** https://sepolia.infura.io/v3/YOUR_KEY
- **Explorer:** https://sepolia.etherscan.io
- **Faucet:** https://sepoliafaucet.com
- **Gas Token:** Sepolia ETH

### Goerli Testnet (Deprecated)
- **Chain ID:** 5
- **RPC:** https://goerli.infura.io/v3/YOUR_KEY
- **Explorer:** https://goerli.etherscan.io
- **Note:** Goerli is being phased out, use Sepolia instead

---

## Contract ABIs

Contract ABIs are automatically generated and stored in:

```
contracts/ethereum/abi/
├── SageRegistryV4.abi.json
├── ISageRegistryV4.abi.json
├── ERC8004IdentityRegistry.abi.json
└── ERC8004ReputationRegistry.abi.json
```

Go bindings are generated in:

```
pkg/blockchain/ethereum/contracts/
└── registryv4/registryv4.go
```

---

## Deployment History

### SageRegistryV4

| Date | Network | Version | Address | Deployer | Notes |
|------|---------|---------|---------|----------|-------|
| TBD | Sepolia | v4.0.0 | TBD | - | Initial V4 deployment |

### Legacy Deployments

See `contracts/ethereum/docs/` for historical deployment information.

---

## Support & Resources

- **Documentation:** [docs/](../docs/)
- **Contract Source:** [contracts/ethereum/contracts/](ethereum/contracts/)
- **Deployment Scripts:** [contracts/ethereum/scripts/](ethereum/scripts/)
- **Issues:** https://github.com/SAGE-X-project/sage/issues

---

**Last Updated:** 2025-01-19
**Maintainer:** SAGE Development Team
