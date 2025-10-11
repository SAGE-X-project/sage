# Changelog

All notable changes to SAGE (Secure Agent Guarantee Engine) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

- **v1.0.0** (2025-10-11): First production release
  - Production-ready blockchain-based security framework
  - Complete RFC 9421 and RFC 9180 implementations
  - Multi-chain DID management
  - Comprehensive testing and documentation

---

**Note**: For migration guides between major versions, see [docs/MIGRATION.md](docs/MIGRATION.md) (will be created for future major releases).

For detailed development history, see the [commit history](https://github.com/SAGE-X-project/sage/commits/main).
