# Changelog

All notable changes to SAGE (Secure Agent Guarantee Engine) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

### 🎉 First Production Release

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

## [Unreleased]

### Planned Features

- Post-quantum cryptography support
- Multi-party sessions with shared session keys
- Automatic key rotation protocol
- Zero-knowledge proof integration
- Cross-chain bridges for L2 solutions
- Enhanced performance optimizations
- Additional language bindings (Java, Swift)

---

## Version History

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
