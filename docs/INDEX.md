# SAGE Documentation Index

Welcome to the SAGE (Secure Agent Guarantee Engine) documentation. This index provides an organized overview of all available documentation.

## Getting Started

- **[README](../README.md)** - Project overview, quick start, and basic usage
- **[Installation Guide](BUILD.md)** - Detailed installation and build instructions
- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute to SAGE

## Core Documentation

### Architecture

- **[Architecture Overview](ARCHITECTURE.md)** - System architecture, components, and design patterns
- **[API Reference](API.md)** - HTTP and gRPC API documentation
- **[Database Schema](DATABASE.md)** - Database structure and migrations

### Security

- **[Coding Guidelines](CODING_GUIDELINES.md)** - Security-focused coding standards
- **[Code Review Checklist](CODE_REVIEW_CHECKLIST.md)** - Review guidelines for pull requests
- **[Security Audit Reports](audit/)** - Third-party security audit findings

### Testing

- **[Testing Guide](TESTING.md)** - Testing strategies and best practices
- **[Integration Tests](../tests/integration/README.md)** - End-to-end integration testing
- **[Benchmark Guide](../tools/benchmark/README.md)** - Performance benchmarking

### Operations

- **[CI/CD Pipeline](CI-CD.md)** - Continuous integration and deployment
- **[Docker Deployment](../docker/README.md)** - Docker and Kubernetes deployment

### Project Management

- **[Go Version Requirements](GO_VERSION_REQUIREMENT.md)** - Go version compatibility and requirements
- **[Version Management](../pkg/version/README.md)** - Software version tracking and release management

## Module Documentation

### Cryptography (`pkg/agent/crypto/`)

- **[Key Management](crypto/)** - Ed25519, Secp256k1, X25519 implementations
- **[Formats](crypto/)** - JWK and PEM key format handling

### Identity (`pkg/agent/did/`)

- **[DID Management](did/)** - Decentralized identity operations
- Multi-chain resolver documentation

### Handshake (`pkg/agent/handshake/`)

- **[Handshake Protocol (EN)](handshake/HANDSHAKE-PROTOCOL.md)** - English documentation
- **[Handshake Protocol (KO)](handshake/HANDSHAKE-PROTOCOL-KO.md)** - Korean documentation
- **[HPKE Implementation](handshake/)** - RFC 9180 HPKE details

### Session Management (`pkg/agent/session/`)

- **[Session Lifecycle](../README.md#session-management)** - Session creation, encryption, expiration
- **[Nonce Management](../README.md#replay-protection)** - Replay attack prevention

### Transport Layer (`pkg/agent/transport/`)

- **[Transport Overview](../pkg/agent/transport/README.md)** - Protocol-agnostic message transport abstraction
- **[HTTP Transport](../pkg/agent/transport/http/README.md)** - REST/HTTP transport implementation
- **[WebSocket Transport](../pkg/agent/transport/websocket/README.md)** - WebSocket transport implementation

## Smart Contracts

### Ethereum

- **[Contract Documentation](../contracts/ethereum/README.md)** - Solidity contract overview
- **[Deployment Guide](../contracts/ethereum/docs/)** - Deployment instructions
- **[Sepolia Deployment](../contracts/ethereum/docs/PHASE7-SEPOLIA-DEPLOYMENT-COMPLETE.md)** - Live testnet contracts

### Solana

- **[Program Documentation](../contracts/solana/README.md)** - Solana program overview
- Anchor framework integration

## CLI Tools

### sage-crypto

Command-line tool for cryptographic operations:
- Key generation (Ed25519, Secp256k1, X25519)
- Message signing and verification
- Key format conversion (JWK ↔ PEM)

```bash
sage-crypto help
```

### sage-did

DID management tool:
- Agent registration
- DID resolution
- Multi-chain operations

```bash
sage-did help
```

### sage-verify

Message signature verification:
- RFC 9421 HTTP signature verification
- Batch verification support

```bash
sage-verify help
```

See [CLI Documentation](cli/) for detailed usage.

## Examples

### Basic Usage

- **[MCP Integration](../examples/mcp-integration/)** - Model Context Protocol integration
- **[Vulnerable vs Secure Chat](../examples/vulnerable-vs-secure/)** - Security comparison demo

### Advanced Topics

- **[Multi-Party Sessions](DETAILED_GUIDE_PART2_KO.md)** - Group messaging patterns
- **[Key Rotation](DETAILED_GUIDE_PART3_KO.md)** - Rotation strategies

## Reference

### RFCs and Standards

- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html) - Hybrid Public Key Encryption
- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [RFC 8032: Ed25519](https://www.rfc-editor.org/rfc/rfc8032.html)
- [RFC 7748: X25519](https://www.rfc-editor.org/rfc/rfc7748.html)
- [W3C DID Core](https://www.w3.org/TR/did-core/)

### External Resources

- [Go Documentation](https://pkg.go.dev/github.com/sage-x-project/sage)
- [GitHub Repository](https://github.com/sage-x-project/sage)
- [Issue Tracker](https://github.com/sage-x-project/sage/issues)
- [Discussions](https://github.com/sage-x-project/sage/discussions)

## Development Guides

### For New Contributors

1. Read [CONTRIBUTING.md](../CONTRIBUTING.md)
2. Review [Coding Guidelines](CODING_GUIDELINES.md)
3. Set up [Development Environment](BUILD.md)
4. Start with [Good First Issues](https://github.com/sage-x-project/sage/labels/good%20first%20issue)

### For Maintainers

- [Release Process](../CONTRIBUTING.md#release-process)
- [Code Review Guidelines](CODE_REVIEW_CHECKLIST.md)
- [CI/CD Maintenance](CI-CD.md)

## Detailed Guides (Korean)

Comprehensive guides in Korean for in-depth understanding:

- **[Part 1: Project Overview and Architecture](DETAILED_GUIDE_PART1_KO.md)**
  - What is SAGE?
  - Why SAGE is needed
  - Overall architecture
  - Core concepts

- **[Part 2: Development and Testing](DETAILED_GUIDE_PART2_KO.md)**
  - Development environment setup
  - Testing strategies
  - Debugging techniques

- **[Part 3: Advanced Features](DETAILED_GUIDE_PART3_KO.md)**
  - Advanced cryptographic operations
  - Multi-chain integration
  - Performance optimization

## Project History and Progress

### Active Planning

- **[Performance Optimization Roadmap](PERFORMANCE_OPTIMIZATION_ROADMAP.md)** - Performance improvement plan and goals
- **[Next Tasks Priority](NEXT_TASKS_PRIORITY.md)** - Future development priorities
- **[Optional Dependency Strategy](OPTIONAL_DEPENDENCY_STRATEGY.md)** - Build tags approach

### Archived Documentation

Historical implementation reports and refactoring plans have been moved to the archive:
- **[Refactoring Archive](archive/refactoring/)** - Architecture proposals and transport refactoring history
- **[Verification Reports](archive/)** - Feature and build verification reports

## Archive

Historical documentation and deprecated features:

- **[Archive](archive/)** - Deprecated documentation
- **[Legacy Contracts](archive/contracts/)** - Previous contract versions

## License Information

- **Backend (Go)**: [LGPL-3.0](../LICENSE)
- **Smart Contracts**: MIT License
- See [LICENSE_COMPLIANCE.md](../LICENSE_COMPLIANCE.md) for details

## Support

### Getting Help

- **Documentation Issues**: Open an issue with label `documentation`
- **Technical Questions**: Use [GitHub Discussions](https://github.com/sage-x-project/sage/discussions)
- **Bug Reports**: [Issue Tracker](https://github.com/sage-x-project/sage/issues/new?template=bug_report.md)
- **Feature Requests**: [Issue Tracker](https://github.com/sage-x-project/sage/issues/new?template=feature_request.md)

### Community

- **Discussions**: Ask questions and share ideas
- **Pull Requests**: Contribute improvements
- **Code Reviews**: Help review contributions

---

**Last Updated**: 2025-10-11

For suggestions to improve this documentation, please [open an issue](https://github.com/sage-x-project/sage/issues/new?title=Docs:+) or submit a pull request.
