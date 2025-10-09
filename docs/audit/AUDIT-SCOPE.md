# SAGE Security Audit Scope

**Version**: 1.0.0
**Date**: October 2025
**Audit Target**: SAGE v1.0 (Phase 7.5 Complete)

## Executive Summary

This document defines the scope of the security audit for SAGE (Secure Agent Guarantee Engine), a blockchain-based security framework for AI agent communication. The audit covers smart contracts, Go backend components, cryptographic implementations, and integration points.

---

## 1. Smart Contracts (Ethereum)

### 1.1 Core Contracts

#### SageRegistryV2
- **Location**: `contracts/ethereum/contracts/core/SageRegistryV2.sol`
- **Lines of Code**: ~600
- **Network**: Ethereum Sepolia (Live), Mainnet (Planned)
- **Contract Address**: `0x487d45a678eb947bbF9d8f38a67721b13a0209BF` (Sepolia)

**Critical Functions**:
```solidity
function registerAgent(
    string calldata name,
    string calldata endpoint,
    bytes calldata publicKey,
    string[] calldata capabilities
) external returns (string memory did)

function updatePublicKey(
    string calldata agentDID,
    bytes calldata newPublicKey,
    bytes calldata ownershipProof
) external

function revokeKey(string calldata agentDID) external

function deactivateAgent(string calldata agentDID) external
```

**Security Features to Audit**:
- 5-step public key validation process
- Challenge-response ownership verification
- Key revocation mechanism
- Access control (owner-only operations)
- Reentrancy protection
- Gas optimization patterns

#### ERC8004ValidationRegistry
- **Location**: `contracts/ethereum/contracts/erc-8004/core/ERC8004ValidationRegistry.sol`
- **Lines of Code**: ~400
- **Network**: Ethereum Sepolia
- **Contract Address**: `0x4D31A11DdE882D2B2cdFB9cCf534FaA55A519440` (Sepolia)

**Critical Functions**:
```solidity
function validateIdentity(
    bytes32 identityHash,
    bytes calldata proof
) external returns (bool)

function registerValidator(address validator) external

function revokeValidator(address validator) external
```

**Security Features to Audit**:
- Identity validation logic
- Validator management
- Proof verification
- Integration with SageRegistryV2

### 1.2 Supporting Contracts

#### SageVerificationHook
- **Location**: `contracts/ethereum/contracts/core/SageVerificationHook.sol`
- **Purpose**: Pre-registration validation hook
- **Lines of Code**: ~150

#### ERC8004IdentityRegistry (Standalone)
- **Location**: `contracts/ethereum/contracts/erc-8004/standalone/ERC8004IdentityRegistry.sol`
- **Purpose**: Standalone ERC-8004 implementation
- **Lines of Code**: ~350

### 1.3 Out of Scope (Smart Contracts)

- Test contracts in `contracts/ethereum/test/`
- Deployment scripts in `contracts/ethereum/scripts/`
- Mock contracts for testing
- Legacy V1 registry (deprecated)

---

## 2. Go Backend Components

### 2.1 Core Cryptography

#### Package: `crypto/`
- **Location**: `crypto/`
- **Lines of Code**: ~2,500

**Critical Components**:
1. **Key Management** (`crypto/keys/`)
   - Ed25519 implementation (digital signatures)
   - Secp256k1 implementation (Ethereum compatibility)
   - X25519 implementation (ECDH key exchange + HPKE)
   - RS256 implementation (RSA-PSS-SHA256, 2048-bit)
   - Key generation, storage, rotation

2. **Blockchain Providers** (`crypto/chain/`)
   - Ethereum provider
   - Solana provider
   - Transaction signing
   - Contract interaction

3. **Secure Storage** (`crypto/vault/`)
   - AES-256-GCM encrypted file storage
   - PBKDF2 key derivation (100,000 iterations, SHA-256)
   - Passphrase-based encryption
   - Secure permissions (0600 for key files)

**Security Focus**:
- Proper entropy sources for key generation
- Secure key storage mechanisms
- Memory cleanup after key operations
- Side-channel attack resistance

### 2.2 Session Management

#### Package: `session/`
- **Location**: `session/`
- **Lines of Code**: ~1,200

**Critical Components**:
1. **Session Manager** (`session/manager.go`)
   - Session lifecycle management
   - Session ID derivation (HKDF)
   - Key ID to Session ID mapping
   - Automatic cleanup

2. **Secure Session** (`session/session.go`)
   - ChaCha20-Poly1305 AEAD encryption
   - Directional keys (client→server, server→client)
   - Nonce management
   - Session expiration

3. **Nonce Cache** (`session/nonce.go`)
   - Replay attack prevention
   - Time-based nonce validation
   - Nonce cache cleanup

**Security Focus**:
- Session key derivation
- Replay attack prevention
- Session timeout handling
- Concurrent access safety
- Memory safety

### 2.3 Handshake Protocol

#### Package: `handshake/`
- **Location**: `handshake/`
- **Lines of Code**: ~800

**Critical Components**:
1. **Client** (`handshake/client.go`)
   - Invitation sending
   - Ephemeral key exchange
   - Session establishment

2. **Server** (`handshake/server.go`)
   - Invitation handling
   - Peer caching
   - Event-driven architecture
   - DID verification

**Security Focus**:
- HPKE encryption of ephemeral keys
- DID signature verification
- Man-in-the-middle prevention
- Peer identity validation

### 2.4 RFC 9421 Implementation

#### Package: `core/rfc9421/`
- **Location**: `core/rfc9421/`
- **Lines of Code**: ~1,500

**Critical Components**:
1. **Message Signing** (`core/rfc9421/signer.go`)
   - HTTP message canonicalization
   - Signature generation
   - Field selection

2. **Message Verification** (`core/rfc9421/verifier.go`)
   - Signature verification
   - Metadata validation
   - Timestamp checking

**Security Focus**:
- Canonicalization correctness
- Signature algorithm selection
- Timestamp validation
- Metadata verification

### 2.5 HPKE Implementation

#### Package: `hpke/`
- **Location**: `hpke/`
- **Lines of Code**: ~400

**Critical Components**:
- RFC 9180 compliance
- X25519 key agreement
- Key encapsulation/decapsulation
- Context export

**Security Focus**:
- Proper HPKE mode usage
- Key derivation
- Associated data handling

### 2.6 Out of Scope (Go Backend)

- CLI tools (`cmd/`)
- Examples (`examples/`)
- Test files (`*_test.go`)
- Internal utilities (`internal/`)
- Configuration loaders (`config/`)

---

## 3. Critical Security Areas

### 3.1 Authentication & Authorization

**Components**:
- DID-based authentication
- Smart contract access control
- Session-based authorization
- Key ownership verification

**Audit Focus**:
- Can an attacker impersonate another agent?
- Can unauthorized users modify agent data?
- Are challenge-response proofs forgeable?

### 3.2 Cryptographic Implementations

**Components**:
- Ed25519 signatures
- Secp256k1 signatures (Ethereum)
- X25519 HPKE
- ChaCha20-Poly1305 AEAD
- HMAC-SHA256

**Audit Focus**:
- Are standard algorithms used correctly?
- Is entropy sufficient for key generation?
- Are keys properly protected in memory?
- Are there timing attacks?

### 3.3 Replay Attack Prevention

**Components**:
- Nonce management
- Timestamp validation
- Message deduplication
- Session expiration

**Audit Focus**:
- Can messages be replayed?
- Is nonce cache properly sized?
- Are timestamps validated correctly?

### 3.4 Key Management

**Components**:
- Key generation
- Key storage
- Key rotation
- Key revocation

**Audit Focus**:
- Are keys generated with sufficient entropy?
- Are private keys encrypted at rest?
- Can revoked keys still be used?
- Is key rotation atomic?

### 3.5 Smart Contract Security

**Components**:
- Access control
- State management
- Gas optimization
- Upgrade mechanism

**Audit Focus**:
- Reentrancy vulnerabilities
- Integer overflow/underflow
- Access control bypasses
- Front-running risks
- Denial of service vectors

---

## 4. Testing Coverage

### 4.1 Current Test Coverage

**Smart Contracts**:
```
SageRegistryV2:              95% coverage
ERC8004ValidationRegistry:   92% coverage
SageVerificationHook:        88% coverage
```

**Go Backend**:
```
crypto/:      93.7% coverage  (recently increased)
crypto/keys:  95%+ coverage
crypto/vault: 92% coverage
session/:     85% coverage
handshake/:   82% coverage
core/rfc9421: 88% coverage
hpke/:        90% coverage
```

### 4.2 Test Types

- Unit tests
- Integration tests
- Fuzzing tests (Foundry)
- Random testing
- Gas optimization tests
- Sepolia testnet deployment tests

---

## 5. Known Issues & Limitations

### 5.1 Known Issues

1. **Cross-platform library builds**: Requires platform-specific C toolchains
   - **Impact**: Low (documented, workaround exists)
   - **Status**: Documented in BUILD.md

2. **Session cleanup timing**: Non-deterministic cleanup timing
   - **Impact**: Low (eventual cleanup guaranteed)
   - **Status**: Acceptable for current use case

### 5.2 Assumptions

1. **Network Assumptions**:
   - Ethereum/blockchain network is trusted
   - RPC endpoints are reliable
   - Block confirmations are sufficient

2. **Cryptographic Assumptions**:
   - Standard algorithms (Ed25519, Secp256k1, X25519, RS256) are secure
   - Go crypto library is trusted
   - File-based encrypted storage with strong passphrases is secure
   - PBKDF2 with 100K iterations provides sufficient key derivation

3. **Operational Assumptions**:
   - Agents act in good faith after registration
   - Private keys are kept secure by users
   - DID registry is properly maintained

---

## 6. Dependencies

### 6.1 Smart Contract Dependencies

```json
{
  "@openzeppelin/contracts": "^4.9.3",
  "@openzeppelin/contracts-upgradeable": "^4.9.3",
  "hardhat": "^2.19.0"
}
```

### 6.2 Go Dependencies

```go
github.com/ethereum/go-ethereum v1.13.5
github.com/cloudflare/circl v1.3.7
filippo.io/edwards25519 v1.1.0
golang.org/x/crypto v0.18.0
```

**Audit Focus**: Known vulnerabilities in dependencies

---

## 7. Deployment Information

### 7.1 Current Deployments

**Sepolia Testnet** (LIVE):
- SageRegistryV2: `0x487d45a678eb947bbF9d8f38a67721b13a0209BF`
- ERC8004ValidationRegistry: `0x4D31A11DdE882D2B2cdFB9cCf534FaA55A519440`
- Deployment Date: October 2025
- Deployer: `0xYourDeployerAddress`

**Planned**:
- Ethereum Mainnet (post-audit)
- Kaia Mainnet (post-audit)

### 7.2 Upgrade Mechanism

- UUPS Proxy pattern (OpenZeppelin)
- Admin-controlled upgrades
- Timelock on upgrades (planned)

---

## 8. Audit Deliverables

### 8.1 Expected from Auditors

1. **Vulnerability Report**:
   - Critical vulnerabilities
   - High-severity issues
   - Medium-severity issues
   - Low-severity issues
   - Informational findings

2. **Code Review Report**:
   - Best practice violations
   - Gas optimization opportunities
   - Code quality improvements

3. **Final Report**:
   - Executive summary
   - Detailed findings
   - Remediation recommendations
   - Re-audit results

### 8.2 Timeline

- **Audit Duration**: 3-4 weeks
- **Remediation**: 1-2 weeks
- **Re-audit**: 1 week
- **Total**: 5-7 weeks

---

## 9. Contact Information

**Project Lead**: SAGE Team
**Repository**: https://github.com/sage-x-project/sage
**Documentation**: https://github.com/sage-x-project/sage/tree/main/docs
**Deployment Guide**: See `contracts/ethereum/docs/PHASE7-SEPOLIA-DEPLOYMENT-COMPLETE.md`

---

## 10. Audit Checklist

### Smart Contracts
- [ ] Access control review
- [ ] Reentrancy analysis
- [ ] Integer overflow/underflow
- [ ] Gas optimization
- [ ] Event emission correctness
- [ ] Upgrade mechanism security
- [ ] Front-running risks
- [ ] DoS vectors

### Go Backend
- [ ] Cryptographic correctness
- [ ] Key management security
- [ ] Session handling
- [ ] Replay attack prevention
- [ ] Concurrency safety
- [ ] Memory safety
- [ ] Error handling
- [ ] Input validation

### Integration
- [ ] Smart contract ↔ Go backend interaction
- [ ] DID resolution correctness
- [ ] Signature verification flow
- [ ] Handshake protocol security
- [ ] End-to-end encryption

---

**Document Version**: 1.0
**Last Updated**: October 2025
**Status**: Ready for External Audit
