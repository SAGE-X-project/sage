# Changelog

All notable changes to SAGE (Secure Agent Guarantee Engine) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **Dependabot Alerts Resolved**: Upgraded vulnerable dependencies (jwt, circl, pgx, cobra, crypto, edwards25519, go-ethereum)
- **Code Scanning Fixes**: Resolved gosec and Slither code scanning configuration errors

### Changed

- **Dependencies (Go)**:
  - Upgraded `github.com/ethereum/go-ethereum`, `golang.org/x/crypto`, `filippo.io/edwards25519`, `github.com/spf13/cobra`, `github.com/jackc/pgx`, `github.com/cloudflare/circl`, `github.com/golang-jwt/jwt`
- **Dependencies (npm)**:
  - Upgraded `@nomicfoundation/hardhat-ignition-ethers` and `@nomicfoundation/hardhat-ignition` to v3.0.5
  - Upgraded `prettier-plugin-solidity` and Hardhat ecosystem packages
- **Docker**: Updated base image to `golang:1.25.6-alpine`

### Fixed

- **CI/CD**: Fixed gosec SARIF output configuration and Slither empty SARIF handling
- **Lint**: Replaced `WriteString(fmt.Sprintf(...))` with `fmt.Fprintf` (QF1012)
- **Solhint**: Removed `ajv` override that broke solhint compatibility

## [1.5.2] - 2025-11-02

### Summary

Patch release fixing KME/KEM naming inconsistency and synchronizing HPKE client behavior.

### Fixed

- **KME → KEM Naming Correction**:
  - Fixed naming typo throughout codebase: KME (Key Management Extension) → KEM (Key Encapsulation Mechanism)
  - Aligns with RFC 9180 terminology for HPKE implementation
  - Updated smart contract field: `kmePublicKey` → `kemPublicKey`
  - Updated getter functions: `getKmePublicKey()` → `getKemPublicKey()`
  - Updated events: `KmePublicKeyUpdated` → `KemPublicKeyUpdated`

- **HPKE Client Synchronization**:
  - Synchronized HPKE client with corrected KEM key resolution
  - Fixed ineffassign lint error in `agentcard_client.go`
  - Improved error handling in agent ID extraction

### Changed

- **Smart Contract Interface Updates** (14 files changed):
  - `AgentCardRegistry.sol`: Updated KEM key registration logic
  - `AgentCardStorage.sol`: Corrected field and function names
  - Updated ABI JSON with correct naming conventions
  - Regenerated Go bindings for smart contracts

- **Go Package Updates**:
  - `pkg/agent/did/ethereum/agentcard_client.go`: Fixed struct fields (119 changes)
  - `pkg/agent/did/ethereum/client.go`: Updated getter/setter methods (352 changes)
  - `pkg/agent/did/ethereum/resolver.go`: Updated KEM key resolution (15 changes)
  - `pkg/agent/hpke/client.go`: Synchronized with resolver (42 changes)

- **Test Files**:
  - Updated 160+ contract test cases (AgentCardRegistry.test.js)
  - Updated 70+ storage test cases (AgentCardStorage.test.js)
  - All 202 tests passing

### Documentation

- Updated `docs/KME_PUBLIC_KEY_INTEGRATION.md` (219 changes)
  - Corrected all KME → KEM references
  - Updated code examples
  - Clarified RFC 9180 terminology

### Impact

- ✅ Improved code clarity and RFC 9180 compliance
- ✅ Enhanced maintainability with consistent naming
- ✅ All tests passing (100%)

## [1.5.1] - 2025-11-02

### Summary

Maintenance release focusing on Go 1.25.2 upgrade for security patches, dependency updates, and enhanced contract deployment tooling.

### Added

- **Enhanced Contract Deployment Tooling**:
  - **Multi-Network Deployment Scripts**: Comprehensive deployment automation for 12 blockchain networks
    - Ethereum: Mainnet, Sepolia testnet
    - Kaia: Cypress (Mainnet), Kairos (Testnet)
    - BSC: Mainnet, Testnet
    - Base: Mainnet, Sepolia
    - Arbitrum: One (Mainnet), Sepolia
    - Optimism: Mainnet, Sepolia
  - **deploy-all-contracts.js**: Complete contract suite deployment script (11,997 bytes)
    - Deploys AgentCardStorage, AgentCardVerifyHook, and AgentCardRegistry
    - Network detection and automatic RPC configuration
    - Deployment verification with comprehensive validation
    - JSON deployment records with timestamps
  - **test-deployed-contract-quick.js**: Fast contract validation (3,941 bytes)
    - Configuration verification (owner, hook address, min stake)
    - Read-only function testing without blockchain delays
    - Quick smoke testing for deployment validation
  - **QUICK_START.md**: Quick reference guide (9,385 bytes)
    - Network-specific deployment commands
    - Contract verification procedures
    - Common deployment scenarios
    - Troubleshooting guide

- **Enhanced Testing Scripts**:
  - **Progress Indicators**: Real-time progress bars during time-lock waits
    - Shows elapsed time during 61-second security delays
    - Improves developer experience with visual feedback
  - **NPM Script Additions** (package.json):
    - `deploy:all:*`: 12 network-specific deployment commands
    - `test:deployed`: Full integration test with security features
    - `test:deployed:quick`: Fast deployment validation

### Changed

- **Go Version Upgrade to 1.25.2**:
  - Upgraded from Go 1.23.0 to Go 1.25.2 for latest security patches and performance improvements
  - Updated all dependencies to Go 1.25.2-compatible versions
  - Updated CI/CD workflows to use Go 1.25.2
  - All tests passing with Go 1.25.2 (100% compatibility)

- **Dependency Updates (24 libraries)**:
  - `github.com/benbjohnson/clock`: v1.1.0 → v1.3.5
  - `github.com/bits-and-blooms/bitset`: v1.20.0 → v1.24.3
  - `github.com/consensys/gnark-crypto`: v0.18.1 → v0.19.2
  - `github.com/deckarep/golang-set/v2`: v2.6.0 → v2.8.0
  - `github.com/ethereum/c-kzg-4844/v2`: v2.1.3 → v2.1.5
  - `github.com/fatih/color`: v1.16.0 → v1.18.0
  - `github.com/fsnotify/fsnotify`: v1.6.0 → v1.9.0
  - `github.com/klauspost/compress`: v1.18.0 → v1.18.1
  - `github.com/mattn/go-colorable`: v0.1.13 → v0.1.14
  - `github.com/prometheus/common`: v0.66.1 → v0.67.2
  - `github.com/prometheus/procfs`: v0.16.1 → v0.19.2
  - `github.com/shirou/gopsutil`: v3.21.4 → v3.21.11
  - `github.com/spf13/pflag`: v1.0.9 → v1.0.10
  - `github.com/streamingfast/logging`: updated to v0.0.0-20250918142248
  - `github.com/supranational/blst`: v0.3.16-0.20250831170142 → v0.3.16
  - `github.com/tklauser/go-sysconf`: v0.3.12 → v0.3.15
  - `github.com/tklauser/numcpus`: v0.6.1 → v0.10.0
  - `go.mongodb.org/mongo-driver`: v1.12.2 → v1.17.6
  - `go.uber.org/atomic`: v1.7.0 → v1.11.0
  - `go.uber.org/multierr`: v1.6.0 → v1.11.0
  - `go.uber.org/ratelimit`: v0.2.0 → v0.3.1
  - `go.uber.org/zap`: v1.21.0 → v1.27.0
  - `go.yaml.in/yaml/v2`: v2.4.2 → v2.4.3
  - `golang.org/x/time`: v0.9.0 → v0.14.0

### Fixed
- **Code Scanning Alerts Resolution**:
  - Added `.gosec.json` configuration to exclude false-positive G115 alerts from CGO-generated code
  - Improved CI/CD workflow for better Slither error logging and debugging
  - All existing security issues were already resolved in previous commits
  - GitHub Code Scanning alerts are based on older commits and will be auto-closed upon next workflow run

### Documentation

- **Contract Documentation Cleanup**:
  - Removed duplicate DEPLOYMENT_GUIDE.md (80-90% overlap with existing guides)
  - Moved to archive: `contracts/ethereum/docs/archive/DEPLOYMENT_GUIDE.md.deprecated`
  - Updated contracts/ethereum/docs/README.md (v2.0 → v2.1)
  - Replaced verbose "Removed Documents" section with concise "Documentation Notes"
  - Improved documentation structure for better maintainability

- **Version Documentation Updates**:
  - Updated README.md "What's New" section for v1.5.1
  - Updated package.json version: 1.4.0 → 1.5.0 (contracts)
  - Updated version.go version: 1.5.0 → 1.5.1 (Go package)
  - Comprehensive CHANGELOG.md entry with all changes documented

## [1.5.0] - 2025-10-30

### Summary

Re-added KME (Key Management Extension) public key storage to AgentCardRegistry with enhanced security validation. This release restores the `kmePublicKey` field that was present in v1.3.1 but removed in v1.4.0, now with critical security improvements for HPKE (Hybrid Public Key Encryption) support per RFC 9180.

### Added

#### AgentCardRegistry: Three-Phase Secure Registration System

- **Three-Phase Registration Flow**: Enhanced security via commit-reveal pattern
  - **Phase 1 (Commit)**: Anti-front-running protection with commitment hash and 0.01 ETH stake
  - **Phase 2 (Register)**: Time-windowed reveal (1-60 minutes) to prevent immediate exploitation
  - **Phase 3 (Activate)**: Sybil-resistant activation delay (1+ hour) with stake refund
  - Economic security through stake requirement (refunded upon successful activation)

- **New CLI Commands**:
  - `sage-did commit`: Initialize registration with commitment hash and stake
  - `sage-did register`: Reveal commitment and register agent after delay
  - `sage-did activate`: Activate agent after time-lock period
  - Automatic commitment state management in `~/.sage/commitments/`

- **Enhanced Security Features**:
  - Commitment-based anti-front-running (prevents DID theft attacks)
  - Time-locked activation prevents rapid spam registration
  - Operator delegation system (ERC-721 style approval pattern)
  - Rate limiting: 24 registrations per address per day
  - Hook system for extensible pre/post-registration validation
  - Multi-key support: ECDSA (Ethereum), Ed25519 (Solana), X25519 (HPKE)

- **Smart Contract Infrastructure**:
  - `AgentCardRegistry.sol`: Production registry with commit-reveal pattern
  - `AgentCardStorage.sol`: Isolated storage layer for future upgradability
  - `AgentCardVerifyHook.sol`: External validation and anti-fraud checks
  - Gas-optimized storage patterns with efficient array operations

#### KME (Key Management Extension) Support

**Contract Layer:**
- Added `kmePublicKey` field to `AgentMetadata` struct (32-byte X25519 keys)
- Added `getKMEKey(bytes32 agentId)` view function for O(1) access
- Added `updateKMEKey(bytes32, bytes, bytes)` function with owner-only access
- Added `KMEKeyUpdated` event for key rotation tracking
- Enforced single X25519 key per agent policy

**Go Integration:**
- Added `PublicKEMKey` field to `AgentMetadataV4` struct
- Added `GetKMEKey(ctx, agentID)` client method
- Added `UpdateKMEKey(ctx, agentID, newKey, signature)` client method
- Updated `GetAgent()` to populate `PublicKEMKey` field
- `ResolveKEMKey()` already implemented in DID resolver

**X25519 Ownership Verification:**
- All X25519 keys must be proven owned by registering account
- ECDSA signature verification using ecrecover
- Signature includes chain ID and registry address (replay protection)
- Prevents Sybil attacks and key theft

**HPKE Integration:**
- KEM key resolution via DID
- Integration with HPKE client
- Support for RFC 9180 hybrid encryption
- Seamless integration with existing DID infrastructure

### Changed

#### Breaking Changes

**X25519 Signature Requirement** (HIGH Impact):

All X25519 key registrations now require ECDSA signatures to prevent key theft attacks.

**Before (v1.3.1 and earlier):**
```javascript
const params = {
    keys: [ecdsaKey, ed25519Key, x25519Key],
    keyTypes: [0, 1, 2],
    signatures: [ecdsaSig, ed25519Sig, "0x"]  // Empty X25519 signature
};
```

**After (v1.5.0):**
```javascript
const x25519Sig = await createX25519Signature(signer, x25519Key);
const params = {
    keys: [ecdsaKey, ed25519Key, x25519Key],
    keyTypes: [0, 1, 2],
    signatures: [ecdsaSig, ed25519Sig, x25519Sig]  // Required ECDSA signature
};
```

**Other Breaking Changes:**
- KeyType enum reordered (ECDSA=0, Ed25519=1, X25519=2)
- Registration flow changed from single-phase to three-phase
- New contract addresses required (AgentCardRegistry vs SageRegistryV4)
- See [AgentCardRegistry Migration Guide](docs/AGENTCARD_MIGRATION_GUIDE.md) for details

**Deprecated:**
- SageRegistryV4 single-phase registration (still functional but deprecated)
- Legacy registration CLI commands (replaced by commit/register/activate)

#### PR #118 Security Enhancements (PR #124 Implementation)

Comprehensive security improvements addressing body tampering, ECDSA support, and error handling (commit `620d6a0`, 2,541 lines added, 8 new files):

- **RFC9421 Body Integrity Validation**:
  - **Problem Solved**: Attackers could modify request body while leaving Content-Digest unchanged
  - **Solution**: New `BodyIntegrityValidator` validates Content-Digest matches actual body SHA-256 hash
  - **Files Added**:
    - `pkg/agent/core/rfc9421/body_integrity.go` (210 lines)
    - `pkg/agent/core/rfc9421/body_integrity_test.go` (283 lines)
    - `pkg/agent/core/rfc9421/body_integrity_edge_test.go` (318 lines)
  - **Edge Cases**: Nil requests, 10MB bodies, malformed headers, Unicode/binary data, timing attacks
  - **Test Coverage**: RFC9421: 83.7% → 84.5%
  - **Security Impact**: Prevents body tampering attacks in HTTP message signatures

- **HPKE ECDSA Signature Verification Support**:
  - **Problem Solved**: HPKE only supported Ed25519, blocking Ethereum agent communication
  - **Solution**: Strategy Pattern with `CompositeVerifier` (auto-selects ECDSAVerifier or Ed25519Verifier)
  - **Files Added**:
    - `pkg/agent/hpke/signature_verifier.go` (214 lines)
    - `pkg/agent/hpke/signature_verifier_test.go` (270 lines)
  - **Capabilities**: Ethereum-compatible Secp256k1 signatures, automatic algorithm selection
  - **Test Coverage**: HPKE: 73.4% → 74.8%
  - **Security Impact**: Enables secure HPKE handshakes for Ethereum agents

- **Enhanced HPKE Client Error Handling**:
  - **Improvements**: Check `resp.Success`, extract `resp.Error`, distinguish nil/failed/empty responses
  - **Files Added**: `pkg/agent/hpke/client_error_test.go` (329 lines)
  - **Scenarios**: Transport errors, nil responses, context cancellation, large data (1MB)

- **DID Utils X25519 Support**:
  - **Enhancement**: Full X25519 key support in `UnmarshalPublicKey` (32-byte validation, memory safety)
  - **Files Added**: `pkg/agent/did/utils_x25519_test.go` (252 lines)
  - **Validation**: Invalid sizes, nil inputs, concurrent access (100 operations), HPKE integration

- **Documentation**:
  - `docs/QUICKSTART_PR118.md` (596 lines): Comprehensive guide with end-to-end examples

- **Quality Assurance**:
  -  100+ new test cases
  -  golangci-lint: 0 issues
  -  gosec: 0 vulnerabilities
  -  Production-ready code quality

### Security

#### KME Security Enhancements

- **X25519 Ownership Verification**:
  - Prevents attackers from registering others' public keys
  - ECDSA signature required for all X25519 keys
  - Chain ID and registry address in signature (replay protection)
  - Signature format: `keccak256("SAGE X25519 Ownership:", x25519PublicKey, chainId, registryAddress, ownerAddress)`

- **Access Control**:
  - Only agent owner can update KME key
  - `onlyAgentOwner` modifier enforcement
  - Reentrancy protection on updates
  - Pause mechanism support

- **Inactive Agent Protection**:
  - `ResolveKEMKey()` rejects inactive agents
  - Prevents usage of compromised agents

#### Registry Security

- **Anti-Front-Running Protection**:
  - Commit-reveal pattern prevents attackers from stealing desired DIDs
  - Cryptographic commitment binding: `keccak256(did, keys, owner, salt, chainId)`
  - Time-window enforcement (1-60 minutes) prevents immediate attacks

- **Economic Security**:
  - 0.01 ETH stake requirement discourages spam registrations
  - Stake forfeiture for expired/invalid registrations
  - Automatic refund upon successful activation

#### Protocol Security

- **Body Integrity Validation** (PR #118/#124):
  - SHA-256 Content-Digest validation prevents body tampering
  - Timing attack resistance, collision resistance, replay attack prevention

- **ECDSA Signature Support** (PR #118/#124):
  - Ethereum-compatible Secp256k1 signatures for HPKE
  - Automatic algorithm selection based on key type

### Documentation

#### New Documentation

- **KME Integration Guide**: [KME Public Key Integration](docs/KME_PUBLIC_KEY_INTEGRATION.md)
  - Architecture overview
  - API reference (Solidity + Go)
  - Usage examples
  - Security considerations
  - Migration guide from v1.3.1
  - Troubleshooting

#### Updated Documentation

- **Migration Guide**: [AgentCardRegistry Migration Guide](docs/AGENTCARD_MIGRATION_GUIDE.md)
  - Step-by-step migration instructions from SageRegistryV4
  - Code examples for three-phase flow
  - CLI command reference (commit/register/activate)
  - X25519 signature requirement updates
  - Troubleshooting guide with common errors
  - Contract deployment instructions
  - Fixed broken references and file extensions (commit `ed3674d`)

- **Quick Start Guide**: [PR #118 Quick Start Guide](docs/QUICKSTART_PR118.md) (596 lines)
  - RFC9421 Body Integrity Validation examples
  - HPKE ECDSA Signature Support integration
  - Enhanced HPKE Error Handling patterns
  - DID X25519 Key Support usage
  - Complete end-to-end secure communication example

- **Architecture Documentation**:
  - [Solidity Contracts Analysis](docs/contracts/SOLIDITY_CONTRACTS_ANALYSIS.md): Moved from root to docs/contracts/ (commit `e2239ae`)
  - Three-phase registration flow diagrams
  - Security design rationale
  - Gas optimization strategies
  - Integration examples

#### Documentation Cleanup

- **Documentation Cleanup** (commits `e2239ae`, `333c15a`, `0107563`):
  - Removed obsolete guides: V4_UPDATE_DEPLOYMENT_GUIDE.md, REFACTORING_PLAN_V4.md, SAGE_A2A_GO_IMPLEMENTATION_GUIDE.md, OPTIONAL_DEPENDENCY_STRATEGY.md
  - Removed archive folder: docs/test/archive/ (15,712 lines, 12 files)
  - Total cleanup: 18,536 lines removed
  - Reorganized planning docs for better structure

### Performance

#### KME Key Retrieval Optimization

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| KME key retrieval | ~80,000 gas (O(N) array) | ~5,000 gas (O(1) field) | 94% reduction |
| Storage overhead | N/A | +32 bytes/agent | Minimal |
| Registration cost | ~449,000 gas | ~450,000 gas | +1,000 gas |

#### Benchmark Suite

- **Benchmark Suite** (Phase 6 - commit `be61927`):
  - Comprehensive performance testing for AgentCardRegistry
  - Gas usage analysis for all operations
  - Throughput measurements for concurrent registrations
  - Memory allocation profiling

### Testing

#### Test Coverage Summary

**Solidity Tests:** 202/202 passing (6s)
- 15 new KME-specific tests (R3.6.1-R3.6.15)
- 2 fixed legacy tests with X25519 signatures
- Comprehensive ownership verification coverage

**Go Tests:** All packages passing
- `pkg/agent/did`: KME client methods
- `pkg/agent/did/ethereum`: AgentCard client integration
- `pkg/agent/hpke`: End-to-end HPKE with KME resolution
- All other packages: 100% pass rate

#### New Tests Added

**Solidity (17 tests):**
- R3.6.1-R3.6.5: KME Key Registration (5 tests)
- R3.6.6-R3.6.8: X25519 Ownership Verification (3 tests)
- R3.6.9-R3.6.12: KME Key Retrieval (4 tests)
- R3.6.13-R3.6.15: KME Key Updates (3 tests)
- R3.2.7: Multi-key registration with proper X25519 signatures (fixed)
- R3.2.8: Invalid key type handling with X25519 signatures (fixed)

**Go Tests:**
- `TestAgentCardClient_GetKMEKey`: KME key retrieval
- `TestAgentCardClient_UpdateKMEKey`: KME key updates
- `TestAgentMetadataV4_PublicKEMKey`: Metadata field validation
- `TestMultiChainResolver_ResolveKEMKey`: 5 resolution scenarios
- `TestE2E_HPKE_KEMKeyResolution`: 4 end-to-end scenarios

#### Integration Tests

- **Phase 4-5 Integration Tests** (commit `2de23fe`):
  - End-to-end three-phase registration tests
  - Multi-key management test scenarios
  - Hook system integration tests
  - Rate limiting validation tests
  - Time-lock mechanism verification

## [1.1.0] - 2024-10-18

### Added

#### Phase 2: HIGH Priority Features - Multi-Key Infrastructure

- **Multi-Key Resolution Support** (Task 2.2)
  - `ResolveAllPublicKeys()`: Retrieve all verified public keys for an agent
  - `ResolvePublicKeyByType()`: Get specific key type (ECDSA or Ed25519)
  - Protocol-specific key selection (ECDSA for Ethereum, Ed25519 for Solana)
  - Comprehensive integration tests for multi-key scenarios
  - Filtering: Only verified keys are returned

- **Enhanced DID Format with Owner Validation** (Task 2.1)
  - New DID formats with owner address:
    - `did:sage:ethereum:0x{address}`
    - `did:sage:ethereum:0x{address}:{nonce}`
  - Benefits: Off-chain ownership verification, cross-chain traceability, DID collision prevention
  - Optional validation functions in SageRegistryV4.sol (backward compatible)
  - Go helper functions:
    - `generateAgentDIDWithAddress()`: Create DID with owner address
    - `generateAgentDIDWithNonce()`: Create DID with address and nonce
    - `deriveEthereumAddress()`: Derive Ethereum address from secp256k1 keypair
  - Comprehensive test coverage (201 contract tests, 13 Go tests)

#### Phase 1: CRITICAL Security Enhancements

- **Public Key Ownership Verification**
  - `_deriveAddressFromPublicKey()`: Verify ECDSA public key ownership
  - Prevents attackers from registering someone else's public key
  - Uses ecrecover to derive Ethereum address from public key
  - Dual verification: signature verification + address matching

- **Atomic Key Rotation**
  - `rotateKey()`: Transaction-level atomic key replacement
  - Prevents incomplete key rotation states
  - Swap-and-pop pattern for gas efficiency
  - Automatic nonce increment to invalidate old signatures
  - `KeyRotated` event for off-chain tracking

- **Complete Key Revocation**
  - Enhanced `revokeKey()`: Complete deletion from storage
  - Removes key from both keyHashes array and keys mapping
  - Prevents revoked keys from being used
  - Nonce increment invalidates old signatures
  - Auto-deactivation when last key is revoked

### Changed

#### Smart Contract Improvements

- **SageRegistryV4**: Enhanced with critical security features
  - Chain-specific signature verification design
    - ECDSA on Ethereum: On-chain verification via ecrecover (signature required)
    - Ed25519 on Ethereum: Off-chain verification + owner approval (no signature)
  - Multi-key support: Up to 10 keys per agent
  - Key type validation: ECDSA (65 bytes) and Ed25519 (32 bytes)
  - Improved error messages for better debugging

#### Test Coverage

- **Contract Tests**: 201 passing (up from 198)
  - 3 new DID format validation tests
  - Security verification tests for ownership, rotation, revocation
  - Multi-key resolution tests

- **Go Tests**: Comprehensive DID generation testing
  - `TestGenerateAgentDIDWithAddress`: 3 test cases
  - `TestGenerateAgentDIDWithNonce`: 4 test cases
  - `TestDeriveEthereumAddress`: 3 test cases
  - `TestDIDGenerationIntegration`: End-to-end flow

### Security

#### Critical Fixes (Phase 1)

- **CVE-SAGE-2025-001**: Public Key Theft Prevention
  - Severity: CRITICAL
  - Issue: Attackers could register with someone else's public key
  - Fix: `_deriveAddressFromPublicKey()` verifies ownership
  - Status: Resolved in SageRegistryV4

- **CVE-SAGE-2025-002**: Atomic Key Rotation
  - Severity: HIGH
  - Issue: Non-atomic key rotation could leave agent in inconsistent state
  - Fix: `rotateKey()` with transaction-level atomicity
  - Status: Resolved in SageRegistryV4

- **CVE-SAGE-2025-003**: Complete Key Revocation
  - Severity: HIGH
  - Issue: Soft-deleted keys could potentially be reused
  - Fix: Complete deletion from storage with nonce increment
  - Status: Resolved in SageRegistryV4

- **Bouncy Castle Dependency**: Updated to address CVE-2025-8916 (Dependabot Alert #11)
  - Updated bcprov-jdk18on from 1.78 to 1.79
  - Updated bcpkix-jdk18on from 1.78.1 to 1.79
  - Fixed excessive allocation vulnerability in PKIXCertPathReviewer
  - Affects Java SDK only (sdk/java/sage-client)

### Documentation

- **SAGE-A2A Integration Guide**: Comprehensive guide for sage-a2a-go project
  - Architecture separation (SAGE core vs sage-a2a-go)
  - Available SAGE APIs for DID resolution and multi-key support
  - Implementation guide for RFC9421 DID integration
  - Code examples and testing strategies
  - Migration path from v3 to v4

## [1.0.3] - 2025-10-18

### Security

- **Gosec Security Scan**: Addressed security findings from gosec static analysis (#93)
  - Fixed potential security issues in cryptographic operations
  - Resolved file permission and error handling vulnerabilities
  - Enhanced input validation and bounds checking
  - Added SARIF output format for better integration with security tools

- **Code Scanning Vulnerabilities**: Resolved critical security issues (#91)
  - Fixed 88 out of 128 code scanning vulnerabilities
  - Addressed high-priority security concerns
  - Improved code quality and security posture
  - Enhanced error handling and validation logic

### Fixed

- **Smart Contract Code Quality**: Slither analysis improvements (#92)
  - Resolved code quality and optimization issues
  - Enhanced gas efficiency in contract operations
  - Fixed potential edge cases and error conditions
  - Improved contract maintainability

- **CI/CD Pipeline**: Enhanced security scanning integration
  - Added SARIF output and upload for gosec results
  - Improved security scan reporting and visibility
  - Better integration with GitHub Code Scanning

### Changed

- **Architecture Refactoring**: Removed A2A implementation from SAGE core (#90)
  - Simplified core architecture by removing Agent-to-Agent (A2A) direct implementation
  - Maintained compatibility through adapter pattern
  - Improved code maintainability and clarity
  - Reduced complexity in core modules

## [1.0.2] - 2025-10-13

### Fixed

- **Coverage Issues**: Skip SageRegistryV3 in coverage to resolve stack depth issues (#70)
  - Resolved stack too deep compilation errors during coverage analysis
  - Added solcover.js configuration to exclude problematic contracts
  - Maintained test coverage for critical contract components

- **CI/CD Pipeline**: Multiple CI/CD improvements and fixes
  - Fixed Go formatting issues across codebase
  - Resolved docker-compose configuration problems
  - Fixed Solidity contract compilation issues in CI
  - Updated loadtest and release workflows to Go 1.23.0

- **License Compliance**: Added LGPL-v3 license headers and GitHub templates (#69)
  - Added proper license headers to all source files
  - Created GitHub issue and PR templates
  - Enhanced project documentation structure

### Security

- **HPKE Handshake Security**: Critical security improvements to handshake protocol (#66, #68)
  - Enhanced key exchange validation
  - Improved session establishment security
  - Strengthened authentication mechanisms
  - Added additional security checks for handshake phase

### Changed

- **Go Version**: Updated to Go 1.23.0 for better performance and security
  - Updated CI/CD workflows to use Go 1.23.0
  - Updated Docker base images
  - Resolved compatibility issues with Go 1.23.0

### Infrastructure

- **GitHub Actions**: Updated workflow configurations
  - Improved test reliability and coverage reporting
  - Enhanced security scanning integration
  - Optimized CI execution time

## [1.0.1] - 2025-10-12

### Changed

#### Dependency Updates

**GitHub Actions:**
- actions/checkout: v4 → v5
- actions/download-artifact: v4 → v5
- codecov/codecov-action: v4 → v5
- crytic/slither-action: v0.4.0 → v0.4.1
- golangci/golangci-lint-action: v6 → v8

**Docker:**
- golang: 1.24-alpine → 1.25-alpine

**Java SDK (Bouncy Castle):**
- bcprov-jdk18on: 1.77 → 1.78
- bcpkix-jdk18on: 1.77 → 1.78.1

**Go Dependencies:**
- github.com/ethereum/go-ethereum: 1.16.1 → 1.16.4
- github.com/gagliardetto/solana-go: 1.12.0 → 1.14.0
- golang.org/x/crypto: 0.41.0 → 0.43.0
- google.golang.org/grpc: 1.73.0 → 1.76.0
- google.golang.org/protobuf: 1.36.8 → 1.36.10
- filippo.io/edwards25519: 1.0.0-rc.1 → 1.1.0
- github.com/spf13/cobra: 1.9.1 → 1.10.1

### Fixed

- **Linter Migration**: Complete golangci-lint v2 migration with all linting errors resolved
  - Fixed 98 linting issues across the codebase
  - Updated to golangci-lint v8 with enhanced rule set
  - Improved code quality and maintainability

### Improved

- **Test Infrastructure**: Consolidated test directories and enhanced Hardhat setup documentation
  - Improved blockchain test setup documentation
  - Clarified dual Hardhat configuration purposes (Go integration tests vs contract deployment)
  - Enhanced test reliability and maintainability

### Security

- All dependencies updated to latest stable versions addressing known vulnerabilities
- Enhanced CI/CD security scanning with updated tools

### Testing

- All 85 feature tests passing with 100% success rate
- Zero linting issues after migration
- Complete test coverage maintained across all packages

## [1.0.0] - 2025-10-11

###  First Production Release

SAGE v1.0.0 marks the first production-ready release of the Secure Agent Guarantee Engine. This release provides a complete blockchain-based security framework for AI agent communication with end-to-end encryption, decentralized identity management, and RFC-compliant message signatures.

### Added

#### Core Features

- **RFC 9421 HTTP Message Signatures**: Complete implementation of HTTP message signature standard
  - Canonical request generation with signature component extraction
  - Support for Ed25519 and Secp256k1 signature algorithms
  - Timestamp and nonce validation for replay attack prevention
  - Signature verification with comprehensive error handling

- **RFC 9180 HPKE Implementation**: Hybrid Public Key Encryption for secure session establishment
  - X25519 key exchange with forward secrecy
  - ChaCha20-Poly1305 AEAD encryption for session messages
  - HKDF-based key derivation for session keys
  - Ephemeral key pairs for each handshake

- **Multi-Chain DID Management**: Decentralized identity across multiple blockchains
  - Ethereum DID registry with enhanced provider and retry logic
  - Solana DID client with on-chain program interface
  - Kaia network support for production deployment
  - DID document resolution with LRU caching

- **Secure Handshake Protocol**: Four-phase handshake for session establishment
  - Invitation phase with service discovery
  - Request phase with ephemeral key exchange
  - Response phase with mutual authentication
  - Complete phase with session key derivation
  - Peer caching for efficient subsequent connections

- **Session Management**: Comprehensive session lifecycle management
  - Deterministic session ID generation from shared secrets
  - Automatic session expiration and cleanup
  - Key ID binding for RFC 9421 verification
  - Nonce cache for replay attack prevention
  - Thread-safe session operations with proper locking

- **Transport Layer Abstraction**: Protocol-agnostic message transport
  - HTTP/REST transport implementation
  - WebSocket transport with persistent connections
  - MockTransport for unit testing
  - Automatic transport selection by URL scheme
  - A2A/gRPC adapter with build tags (optional)

#### Smart Contracts

- **SageRegistryV2**: Enhanced Ethereum smart contract with security features
  - 5-step public key validation (length, format, zero-key, ownership, revocation)
  - Challenge-response authentication for key registration
  - Key revocation system with auto-deactivation
  - Hook system for extensible validation
  - Gas-optimized storage patterns

- **ERC-8004 Identity Registry**: Standalone identity registry implementation
  - Compliant with ERC-8004 standard for Non-Fungible Decentralized Identifiers
  - Cross-contract interoperability

- **Live Deployments**: Production contracts on Sepolia testnet
  - SageRegistryV2: `0x487d45a678eb947bbF9d8f38a67721b13a0209BF`
  - ERC8004ValidationRegistry: `0x4D31A11DdE882D2B2cdFB9cCf534FaA55A519440`
  - ERC8004IdentityRegistry: `0x02439d8DA11517603d0DE1424B33139A90969517`

#### CLI Tools

- **sage-crypto**: Cryptographic operations CLI
  - Key pair generation (Ed25519, Secp256k1, X25519, RSA)
  - Message signing and verification
  - Key format conversion (PEM ↔ JWK)
  - Key storage management (file, memory, vault)

- **sage-did**: DID management CLI
  - Agent registration on multiple blockchains
  - DID resolution and lookup
  - Multi-chain operations
  - Batch DID operations

- **sage-verify**: Message signature verification CLI
  - RFC 9421 signature verification
  - Batch verification support
  - Detailed verification reports

#### Cryptography

- **Multi-Algorithm Support**:
  - Ed25519 for signing (RFC 8032)
  - Secp256k1 for Ethereum compatibility
  - X25519 for key exchange (RFC 7748)
  - RSA for legacy interoperability
  - ChaCha20-Poly1305 for AEAD encryption

- **Key Storage Backends**:
  - File-based encrypted storage with PEM format
  - In-memory storage for testing
  - OS keychain integration (macOS Keychain, Linux Secret Service, Windows Credential Manager)

- **Key Format Support**:
  - PEM encoding/decoding (RFC 7468)
  - JWK format conversion (RFC 7517)
  - Compressed/uncompressed public key conversion
  - Ethereum address generation from public keys

#### Testing & Quality Assurance

- **Comprehensive Test Suite**: 85+ feature tests with 100% pass rate
  - Unit tests for all core modules
  - Integration tests with blockchain nodes
  - End-to-end handshake tests with MockTransport
  - Random fuzzing tests for robustness
  - Benchmark tests for performance validation

- **Test Infrastructure**:
  - Automated test setup scripts
  - Blockchain node management (Hardhat/Anvil)
  - Test environment isolation
  - Comprehensive test documentation

- **Quality Gates**:
  - Minimum 70% code coverage requirement
  - 90%+ coverage for critical paths
  - Automated linting (golangci-lint, solhint)
  - Security scanning integration

#### Documentation

- **Comprehensive Documentation**:
  - Architecture guide with component diagrams
  - API reference for HTTP and gRPC endpoints
  - Build instructions for multiple platforms
  - Security design documentation
  - Testing guide with best practices

- **Developer Guides**:
  - Feature test guide (Korean)
  - CLI usage guides (English/Korean)
  - Handshake protocol specification (English/Korean)
  - HPKE implementation details (English/Korean)
  - Smart contract deployment guides

- **Multi-Language Support**:
  - English and Korean documentation
  - Localized CLI help messages
  - Code examples in multiple languages

#### Build & Distribution

- **Cross-Platform Support**:
  - Linux (x86_64, ARM64)
  - macOS (x86_64, ARM64 / Apple Silicon)
  - Windows (x86_64, ARM64)

- **Library Builds**:
  - Static libraries (.a)
  - Shared libraries (.so, .dylib, .dll)
  - C-compatible headers
  - Language bindings (C/C++, Python, Rust)

- **Release Automation**:
  - GitHub Actions CI/CD pipeline
  - Automated binary builds for all platforms
  - Release artifact generation with checksums
  - Automated security scanning

### Changed

- **Enhanced Provider**: Improved Ethereum RPC provider with retry logic and exponential backoff
- **Session Cleanup**: Optimized automatic session expiration with configurable intervals
- **DID Caching**: LRU cache for DID resolution with configurable TTL
- **Transport Selection**: Improved automatic transport selection based on URL schemes

### Fixed

- **Test Infrastructure**: Fixed blockchain integration test setup script bugs
  - Corrected Hardhat version detection command
  - Fixed undefined FORK_URL variable handling
  - Added explicit chain-id and port parameters

- **Setup Script**: Enhanced blockchain node management
  - Improved error handling for missing blockchain tools
  - Better process cleanup on shutdown
  - Clearer error messages for troubleshooting

- **Documentation**: Comprehensive updates to test documentation
  - Added blockchain test prerequisites section
  - Created troubleshooting guide for common failures
  - Added pre-test verification checklist

### Security

- **Challenge-Response Authentication**: Prevents unauthorized key registration in smart contracts
- **Key Revocation System**: Ability to revoke compromised keys with auto-deactivation
- **Nonce-Based Replay Protection**: 64-bit nonce cache prevents message replay attacks
- **Timestamp Validation**: Configurable time windows for message acceptance
- **Forward Secrecy**: Ephemeral X25519 keys provide forward secrecy in handshakes
- **AEAD Encryption**: ChaCha20-Poly1305 provides authenticated encryption for all messages

---

## Version History

- **v1.5.0** (2025-10-30): AgentCardRegistry - Major Security Release
  - Three-phase registration with commit-reveal pattern
  - Anti-front-running protection and economic security (0.01 ETH stake)
  - Time-locked activation (1+ hour delay) for Sybil resistance
  - New CLI commands: commit, register, activate
  - KME (Key Management Extension) support for X25519 keys
  - Breaking changes: Migration required from SageRegistryV4
  - Comprehensive migration guide and documentation

- **v1.1.0** (2024-10-18): Multi-Key Registry V4 - Major Feature Release
  - SageRegistryV4 with multi-key support (up to 10 keys per agent)
  - Multi-key resolution: ResolveAllPublicKeys() and ResolvePublicKeyByType()
  - Enhanced DID format with owner address validation
  - Critical security fixes: CVE-SAGE-2025-001, CVE-SAGE-2025-002, CVE-SAGE-2025-003
  - SAGE-A2A integration guide for cross-project compatibility
  - 201 passing contract tests, comprehensive Go test coverage
  - Bouncy Castle CVE-2025-8916 security update

- **v1.0.3** (2025-10-18): Security and code quality improvements
  - Gosec security scan findings addressed
  - Critical code scanning vulnerabilities resolved (88/128 issues)
  - Slither smart contract code quality improvements
  - Enhanced CI/CD security scanning integration
  - Removed A2A implementation from core architecture

- **v1.0.2** (2025-10-13): Patch release with security and CI/CD improvements
  - Critical HPKE handshake security enhancements
  - Stack depth issue resolution for test coverage
  - License compliance and documentation improvements
  - Go 1.23.0 update and CI/CD optimizations

- **v1.0.1** (2025-10-12): Maintenance release
  - Dependency updates across the board
  - Linter migration and code quality improvements
  - Test infrastructure enhancements

- **v1.0.0** (2025-10-11): First production release
  - Production-ready blockchain-based security framework
  - Complete RFC 9421 and RFC 9180 implementations
  - Multi-chain DID management
  - Comprehensive testing and documentation

---

**Note**: For migration guides between major versions, see [docs/MIGRATION.md](docs/MIGRATION.md) (will be created for future major releases).

For detailed development history, see the [commit history](https://github.com/SAGE-X-project/sage/commits/main).
