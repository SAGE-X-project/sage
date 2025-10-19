# SAGE Smart Contracts Index

Comprehensive index of all smart contracts in the SAGE project.

**Last Updated**: 2025-01-19
**Current Version**: V4 (Multi-Key Registry)

---

## Table of Contents

- [Active Contracts](#active-contracts)
- [Deprecated Contracts](#deprecated-contracts)
- [Interfaces](#interfaces)
- [Test Contracts](#test-contracts)
- [Contract Comparison](#contract-comparison)
- [Deployment Status](#deployment-status)

---

## Active Contracts

### SageRegistryV4.sol (LATEST - v1.1.0)

**Status**: ✅ Production Ready - Merged to dev
**Location**: `contracts/ethereum/contracts/SageRegistryV4.sol`
**Interface**: `ISageRegistryV4.sol`
**License**: MIT

**Purpose**: Multi-key AI agent registry with A2A protocol compatibility

**Key Features**:
- Multi-key architecture (Ed25519, ECDSA, X25519)
- Up to 10 keys per agent
- Key lifecycle management (add, revoke, rotate)
- Type-specific verification (ECDSA on-chain, Ed25519 owner-approved)
- A2A Agent Card generation
- Backward compatibility with V2/V3

**Gas Costs**:
| Operation | Gas Used | Notes |
|-----------|----------|-------|
| Register (1 key) | ~875,000 | Single ECDSA key |
| Register (3 keys) | ~1,300,000 | Mixed key types |
| Add Key | ~50,000 | To existing agent |
| Revoke Key | ~70,000 | Owner-controlled |
| Approve Ed25519 | ~55,000 | Owner only |

**Test Coverage**:
- 201 contract tests (100% core coverage)
- 85+ Go backend tests (77.6%+ coverage)
- All integration tests passing

**Related Files**:
- Go bindings: `pkg/agent/did/ethereum/clientv4.go`
- Types: `pkg/agent/did/types_v4.go`
- Tests: `contracts/ethereum/test/SageRegistryV4.test.js`
- CLI: `cmd/sage-did/register.go`, `cmd/sage-did/key.go`
- Deployment: `contracts/ethereum/scripts/deploy_v4.js`

---

### SageRegistryV2.sol (STABLE PRODUCTION)

**Status**: ✅ Production Stable
**Location**: `contracts/ethereum/contracts/SageRegistryV2.sol`
**Interface**: `ISageRegistry.sol`
**License**: MIT

**Purpose**: Enhanced single-key registry with 5-step validation

**Key Features**:
- 5-step public key validation
- Key revocation system
- Challenge-response ownership proof
- Hook system for extensibility
- Gas-optimized operations

**Gas Costs**:
| Operation | Gas Used | Notes |
|-----------|----------|-------|
| Register Agent | ~620,000 | With validation |
| Update Metadata | ~80,000 | Owner only |
| Revoke Key | ~66,000 | Auto-deactivate agent |
| Deactivate Agent | ~50,000 | Owner only |

**Related Files**:
- Hook implementation: `SageVerificationHook.sol`
- Interface: `ISageRegistry.sol`, `IRegistryHook.sol`
- Tests: `contracts/ethereum/test/SageRegistryV2.test.js`
- Deployment: `contracts/ethereum/scripts/deploy-v2.js`

---

### SageVerificationHook.sol

**Status**: ✅ Active
**Location**: `contracts/ethereum/contracts/SageVerificationHook.sol`
**Interface**: `IRegistryHook.sol`
**License**: MIT

**Purpose**: Extensible verification hook for V2 registry

**Key Features**:
- DID format validation
- Rate limiting (1 minute cooldown)
- Daily registration limits (5 per address)
- Blacklist management
- Before/after registration hooks

**Related Files**:
- Interface: `IRegistryHook.sol`
- Used by: `SageRegistryV2.sol`
- Tests: Integrated in V2 test suite

---

## Deprecated Contracts

### SageRegistryV3.sol

**Status**: 🗑️ DEPRECATED - Superseded by V4
**Location**: `contracts/ethereum/contracts/deprecated/SageRegistryV3.sol`
**Deprecated Date**: 2025-01-18

**Reason for Deprecation**:
- Limited to single key per agent
- Superseded by V4 multi-key architecture
- Missing A2A protocol compatibility

**Migration**: Use SageRegistryV4 for new deployments

---

### SageRegistry.sol (V1)

**Status**: 🗑️ DEPRECATED - Security Issues
**Location**: `contracts/ethereum/contracts/deprecated/SageRegistry.sol`
**Deprecated Date**: 2024-12-01

**Reason for Deprecation**:
- Insufficient public key validation
- No key revocation mechanism
- Security vulnerabilities identified in audit
- Missing ownership proof verification

**Critical Issues**:
- Weak signature validation
- No protection against key reuse attacks
- Missing revocation functionality

**Migration**: Use SageRegistryV2 or V4

**See**: `contracts/archived/SECURITY_AUDIT_LEGACY.md` for details

---

## Interfaces

### ISageRegistryV4.sol

**Purpose**: Interface for V4 multi-key registry
**Location**: `contracts/ethereum/contracts/interfaces/ISageRegistryV4.sol`

**Key Methods**:
```solidity
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    uint8[] calldata keyTypes,
    bytes[] calldata keyData,
    bytes[] calldata signatures,
    string calldata capabilities
) external returns (bytes32 agentId);

function addKey(
    bytes32 agentId,
    uint8 keyType,
    bytes calldata keyData,
    bytes calldata signature
) external returns (bytes32 keyHash);

function revokeKey(bytes32 agentId, bytes32 keyHash) external;

function approveEd25519Key(bytes32 keyHash) external;
```

---

### ISageRegistry.sol

**Purpose**: Interface for V2 registry
**Location**: `contracts/ethereum/contracts/interfaces/ISageRegistry.sol`

**Key Methods**:
```solidity
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature
) external returns (bytes32);

function revokeKey(bytes calldata publicKey) external;

function updateAgent(
    bytes32 agentId,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    string calldata capabilities
) external;
```

---

### IRegistryHook.sol

**Purpose**: Hook interface for V2 extensibility
**Location**: `contracts/ethereum/contracts/interfaces/IRegistryHook.sol`

**Methods**:
```solidity
function beforeRegister(
    address owner,
    string calldata did,
    bytes calldata publicKey
) external returns (bool);

function afterRegister(
    bytes32 agentId,
    address owner,
    string calldata did
) external;
```

---

## Test Contracts

### SageRegistryTest.sol

**Purpose**: V2 test version with additional debugging features
**Location**: `contracts/ethereum/contracts/SageRegistryTest.sol`
**Based On**: SageRegistryV2.sol

**Test Features**:
- Additional event logging
- Public test helpers
- State inspection methods

**Usage**: Local testing and development only

---

## Contract Comparison

| Feature | V1 | V2 | V4 |
|---------|----|----|-----|
| **Multi-Key Support** | ❌ | ❌ | ✅ Up to 10 keys |
| **Key Types** | ECDSA only | ECDSA only | ECDSA, Ed25519, X25519 |
| **Key Validation** | Basic | 5-step | Type-specific |
| **Key Revocation** | ❌ | ✅ | ✅ Per-key |
| **Key Rotation** | ❌ | ❌ | ✅ |
| **Ownership Proof** | ❌ | ✅ Challenge | ✅ Per-key |
| **Hook System** | ❌ | ✅ | ❌ (future) |
| **A2A Compatibility** | ❌ | ❌ | ✅ |
| **Gas Cost (register)** | ~400k | ~620k | ~875k (1 key) |
| **Status** | Deprecated | Production | Production Ready |
| **Test Coverage** | Basic | Comprehensive | Comprehensive |

---

## Deployment Status

### Mainnet Deployments

**V4**:
- Ethereum: Not yet deployed
- Kaia: Not yet deployed

**V2**:
- Ethereum: Not yet deployed
- Kaia: Not yet deployed

### Testnet Deployments

**V4**:
- Sepolia: Ready for deployment
- Kairos (Kaia): Ready for deployment
- Local: Tested on Hardhat/Anvil

**V2**:
- Sepolia: [To be deployed]
- Kairos (Kaia): [To be deployed]
- Local: `0x5FbDB2315678afecb367f032d93F642f64180aa3`

**See**: `contracts/DEPLOYED_ADDRESSES.md` for deployment tracking

---

## Gas Optimization Comparison

### V4 vs V2 Gas Analysis

**Single-Key Registration**:
- V2: ~620,000 gas
- V4: ~875,000 gas (+41%)
- V4 overhead: Multi-key infrastructure

**Multi-Key Advantage** (V4 only):
- 2 keys: ~1,200,000 gas (~600k per key)
- 3 keys: ~1,300,000 gas (~433k per key)
- Economies of scale for multi-key agents

**Key Management** (V4 only):
- Add key: ~50,000 gas (very efficient)
- Revoke key: ~70,000 gas
- Rotate key: 2 operations (~120,000 total)

**Recommendation**: Use V4 for multi-key agents, V2 for simple single-key use cases

---

## Security Considerations

### V4 Security Features

✅ **Strengths**:
- Type-specific key verification
- Per-key revocation (granular control)
- Owner-approved Ed25519 keys
- Multi-sig potential with multiple ECDSA keys
- A2A protocol compliance

⚠️ **Considerations**:
- Ed25519 verification requires off-chain approval
- Higher gas costs for single-key agents
- More complex key management

### V2 Security Features

✅ **Strengths**:
- 5-step key validation
- Challenge-response ownership proof
- Global key revocation
- Hook-based extensibility
- Battle-tested in production

⚠️ **Limitations**:
- Single key per agent (key rotation requires re-registration)
- No multi-chain key support
- Limited to ECDSA keys

---

## Related Documentation

- **[README.md](./README.md)** - Main contracts documentation
- **[ROADMAP.md](./ROADMAP.md)** - V4 feature roadmap and status
- **[TODO.md](./TODO.md)** - Active tasks and next steps
- **[DEPLOYED_ADDRESSES.md](./DEPLOYED_ADDRESSES.md)** - Deployment tracking
- **[MULTI_KEY_DESIGN.md](./MULTI_KEY_DESIGN.md)** - V4 design specification
- **[archived/CODE_ANALYSIS_V1_V2_V3.md](./archived/CODE_ANALYSIS_V1_V2_V3.md)** - Legacy analysis
- **[archived/SECURITY_AUDIT_LEGACY.md](./archived/SECURITY_AUDIT_LEGACY.md)** - V1 security audit

---

## Quick Reference

### Which Contract Should I Use?

**Use SageRegistryV4 if**:
- ✅ You need multi-chain agent identity
- ✅ You want A2A protocol compatibility
- ✅ You need key rotation without re-registration
- ✅ You have multiple key types (ECDSA, Ed25519, X25519)
- ✅ You're building new applications

**Use SageRegistryV2 if**:
- ✅ You only need single ECDSA key
- ✅ You want lower gas costs
- ✅ You need hook extensibility
- ✅ You're maintaining existing V2 deployments

**Don't Use V1/V3**:
- ❌ Security vulnerabilities (V1)
- ❌ Superseded by better alternatives (V3)

---

**For deployment procedures, see [DEPLOYED_ADDRESSES.md](./DEPLOYED_ADDRESSES.md)**
**For implementation details, see [README.md](./README.md)**
