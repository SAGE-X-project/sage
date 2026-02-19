# SAGE - Secure Agent Guarantee Engine

[![Go Version](https://img.shields.io/badge/Go-1.25.2-blue.svg)](https://golang.org/dl/)
[![Solidity Version](https://img.shields.io/badge/Solidity-0.8.20-red.svg)](https://soliditylang.org/)
[![License](https://img.shields.io/badge/License-LGPL--3.0-blue.svg)](LICENSE)

[![Tests](https://github.com/sage-x-project/sage/workflows/Test/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/test.yml)
[![Integration Tests](https://github.com/sage-x-project/sage/workflows/Integration%20Tests/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/integration-test.yml)
[![Security](https://github.com/sage-x-project/sage/workflows/Security/badge.svg)](https://github.com/sage-x-project/sage/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/sage-x-project/sage/branch/main/graph/badge.svg)](https://codecov.io/gh/sage-x-project/sage)

A blockchain-based security framework for AI agent communication — providing end-to-end encrypted, authenticated channels between AI agents using decentralized identity (DID), HPKE key agreement (RFC 9180), and HTTP Message Signatures (RFC 9421).

> For release history and recent changes, see [CHANGELOG.md](CHANGELOG.md).

## Table of Contents

- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Usage](#usage)
- [Testing](#testing)
- [Supported Networks](#supported-networks)
- [Live Deployments](#live-deployments)
- [Multi-Language Bindings](#multi-language-bindings)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)
- [Support & Acknowledgments](#support--acknowledgments)

## Key Features

- **End-to-End Encrypted Handshake** — HPKE (RFC 9180) with X25519 key agreement
- **RFC 9421 HTTP Message Signatures** — Verifiable agent-to-agent communication
- **Multi-Chain DID** — Ethereum, Kaia, and Solana network integration
- **AgentCardRegistry** — Three-phase commit-reveal registration with ERC-8004 compliance
- **Multi-Key Support** — Ed25519, Secp256k1, and X25519 cryptographic operations
- **Session Management** — Automatic key rotation, nonce tracking, and replay protection
- **A2A Protocol Integration** — Native support for Google Agent-to-Agent protocol
- **Protocol-Agnostic Transport** — HTTP, WebSocket, and pluggable architecture
- **Zero External Dependencies** — Self-contained core for better maintainability

## Quick Start

### Prerequisites

- **Go 1.25.2+** (see [docs/GO_VERSION_REQUIREMENT.md](docs/GO_VERSION_REQUIREMENT.md))
- **Node.js 22+** and npm (for smart contract development)
- **Git**

### Install & Build

```bash
# Clone
git clone https://github.com/SAGE-X-project/sage.git
cd sage

# Go dependencies
go mod download

# Smart contract dependencies
cd contracts/ethereum && npm install && cd ../..

# Build all CLI tools
make build

# Compile smart contracts
cd contracts/ethereum && npm run compile
```

### Run Tests

```bash
# Go tests
make test

# Smart contract tests (202 passing)
cd contracts/ethereum && npm test

# Integration tests
make test-integration
```

See [docs/BUILD.md](docs/BUILD.md) for cross-platform compilation, library builds, and language integration details.

## Architecture

### Project Structure

```
sage/
├── core/                    # Core RFC 9421 implementation
│   ├── rfc9421/            # HTTP message signatures (canonicalization, signing, verification)
│   └── message/            # Message processing, validation, ordering, and deduplication
├── crypto/                  # Cryptographic operations
│   ├── keys/               # Ed25519, Secp256k1, X25519 key pair implementations
│   ├── chain/              # Blockchain-specific providers (Ethereum, Solana)
│   ├── storage/            # Secure key storage (file, memory)
│   ├── vault/              # Hardware-backed secure storage with OS keychain integration
│   └── formats/            # JWK, PEM key format converters
├── did/                     # Decentralized Identity
│   ├── ethereum/           # Ethereum DID client with enhanced provider
│   ├── solana/             # Solana DID client
│   ├── manager.go          # Multi-chain DID management
│   └── resolver.go         # DID document resolution with caching
├── handshake/               # Secure session establishment
│   ├── client.go           # Handshake initiator implementation
│   ├── server.go           # Handshake responder with peer caching
│   └── types.go            # Invitation, Request, Response, Complete messages
├── hpke/                    # HPKE (RFC 9180) implementation
│   ├── client.go           # HPKE sender (encapsulation)
│   ├── server.go           # HPKE receiver (decapsulation)
│   └── common.go           # Shared HPKE utilities
├── session/                 # Session and key management
│   ├── manager.go          # Session lifecycle, cleanup, and key ID binding
│   ├── session.go          # Secure session with ChaCha20-Poly1305 AEAD
│   ├── nonce.go            # Replay attack prevention with nonce cache
│   └── metadata.go         # Session state and expiration tracking
├── transport/               # Protocol-agnostic transport layer
│   ├── interface.go        # MessageTransport interface
│   ├── mock.go             # MockTransport for testing
│   ├── selector.go         # Runtime transport selection
│   ├── http/               # HTTP/REST transport implementation
│   ├── websocket/          # WebSocket transport implementation
│   └── a2a/                # A2A adapter for backward compatibility
├── health/                  # Health monitoring system
│   ├── checker.go          # Component health checks
│   └── server.go           # HTTP health endpoint
├── config/                  # Configuration management
│   ├── config.go           # Unified configuration loader
│   ├── blockchain.go       # Blockchain-specific settings
│   └── validator.go        # Configuration validation
├── contracts/               # Smart contracts
│   └── ethereum/           # Ethereum contracts, tests, deployment scripts
├── cmd/                     # CLI applications
│   ├── sage-crypto/        # Cryptographic operations CLI
│   ├── sage-did/           # DID management CLI
│   └── deployment-verify/  # Blockchain deployment verification CLI
├── examples/                # Usage examples
│   └── mcp-integration/    # Model Context Protocol integration examples
├── tests/                   # Testing infrastructure
│   ├── integration/        # End-to-end integration tests
│   ├── random/             # Randomized fuzzing tests
│   └── handshake/          # Handshake integration tests
├── docs/                    # Documentation
│   ├── handshake/          # Handshake protocol documentation (EN/KO)
│   ├── dev/                # Developer guides and security design
│   └── assets/             # Architecture diagrams
├── scripts/                 # Test and deployment scripts
└── internal/                # Internal utilities and helpers
```

### Handshake Protocol (HPKE, 1-RTT / 2-Phase)

SAGE uses a server static X25519 KEM, a client ephemeral KEM (enc), plus Ed25519 signatures, an ackTag (key-confirmation), and optional cookies for DoS control.

1.  **Initialize** (Client → Server)
    - **Sends**: enc (HPKE encapsulation), ephC (client X25519 for PFS), info / exportCtx, nonce / ts, DID signature, and (optional) cookie.
    - **Server**: verify cookie early → verify DID signature → check replay/clock-skew/context → HPKE Open → generate ephS → compute ssE2E → derive seed → **create session**.

2.  **Acknowledge** (Server → Client)
    - **Sends**: kid, ackTagB64 (key confirmation), ephS, and a signed server envelope.
    - **Client**: verify ackTag → verify server signature → **bind kid ↔ session** → derive c2s/s2c AEAD keys and start the channel.

    <img src="docs/assets/SAGE-hpke-handshake.png" width="450" height="550"/>

See [docs/handshake/hpke-based-handshake-en.md](docs/handshake/hpke-based-handshake-en.md) for the full protocol specification.

## Usage

### 1. Generate Key Pairs

```bash
# Generate Ed25519 key pair (for DID signatures)
./build/bin/sage-crypto generate -t ed25519 -o keys/agent.key

# Generate Secp256k1 key pair (for Ethereum)
./build/bin/sage-crypto generate -t secp256k1 -o keys/ethereum.key

# Generate X25519 key pair (for HPKE encryption)
./build/bin/sage-crypto generate -t x25519 -o keys/hpke.key

# List all keys
./build/bin/sage-crypto list -d keys/
```

### 2. Register an AI Agent (Three-Phase Flow)

```bash
# Phase 1: Commit with stake
./build/bin/sage-did commit \
  --chain ethereum \
  --key keys/ethereum.key \
  --name "My AI Agent" \
  --endpoint "https://api.myagent.com"

# Phase 2: Register (after 1–60 min)
./build/bin/sage-did register \
  --chain ethereum \
  --key keys/ethereum.key

# Phase 3: Activate (after 1+ hour)
./build/bin/sage-did activate \
  --chain ethereum \
  --key keys/ethereum.key

# Resolve a DID
./build/bin/sage-did resolve did:sage:ethereum:0x...
```

### 3. Secure Handshake & Encrypted Communication

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/hpke"
    "github.com/sage-x-project/sage/pkg/agent/session"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Client side (Agent A)
client := hpke.NewClient(transport, resolver, myKeyPair, string(myDID), infoBuilder, sessionManager)

// Initialize session
ctxID := "ctx-" + uuid.NewString()
kid, _ := client.Initialize(ctx, ctxID, clientDID, serverDID)

// Get session and encrypt/decrypt
sess, _ := sessionManager.GetByKeyID(kid)
cipher, _ := sess.Encrypt(body)
plain, _ := sess.Decrypt(cipher)
```

### 4. RFC 9421 Signed Messages

```go
import "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"

// Build and sign HTTP message
builder := rfc9421.NewMessageBuilder()
msg := builder.
    Method("POST").
    Authority("api.example.com").
    Path("/api/v1/chat").
    Header("Content-Type", "application/json").
    Body([]byte(cipherRequestBody)).
    Build()

verifier := rfc9421.NewHTTPVerifier(sess, sessionManager)
signature, _ := verifier.SignRequest(msg, sigName, []string{
    "@method", "@authority", "@path", "content-type", "content-digest",
}, privKey)
```

### 5. A2A Protocol Integration

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Generate a key pair and derive Ethereum address
keyPair, _ := crypto.GenerateSecp256k1KeyPair()
ownerAddr, _ := did.DeriveEthereumAddress(keyPair)

// Create DID with owner address
agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, ownerAddr)

// Export as A2A-compliant agent card
card := did.GenerateA2ACard(agentDID, metadata)
```

For detailed A2A integration, see [SAGE A2A Integration Guide](docs/SAGE_A2A_INTEGRATION_GUIDE.md) and [sage-a2a-go](https://github.com/sage-x-project/sage-a2a-go).

### Configuration

SAGE supports YAML configuration files and environment variables:

```yaml
blockchain:
  ethereum:
    rpc_url: "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
    contract_address: "0x..."
    chain_id: 1

crypto:
  key_dir: "./keys"
  default_algorithm: "ed25519"

session:
  max_age: "1h"
  idle_timeout: "10m"
  cleanup_interval: "30s"
```

See environment variable options and Hardhat setup details in the [full configuration docs](docs/BUILD.md).

## Testing

### Go Tests

```bash
make test                # All tests
make test-quick          # Exclude slow integration tests
make test-integration    # Integration tests (starts Hardhat node)
make bench               # Benchmark tests
make random-test         # Random fuzzing tests
go test -race ./...      # Race detection
go test -cover ./...     # Coverage
```

### Smart Contract Tests

```bash
cd contracts/ethereum
npm test                 # All 202 contract tests
npm run coverage         # Coverage report
npm run test:integration # Integration tests
```

### Feature Verification

```bash
./tools/scripts/verify_all_features.sh   # 85+ feature tests
./tools/scripts/quick_verify.sh           # Quick 5-check verification
```

## Supported Networks

### Mainnet

- **Ethereum** — Full support with ENS integration
- **Kaia (Cypress)** — Production deployment
- **BSC, Base, Arbitrum, Optimism** — EVM-compatible deployment ready
- **Solana** — In development

### Testnet

- **Sepolia** (Ethereum), **Kairos** (Kaia), **BSC Testnet**, **Base Sepolia**, **Arbitrum Sepolia**, **Optimism Sepolia**, **Solana Devnet**

## Live Deployments

### Sepolia Testnet — AgentCard Contracts

| Contract | Address |
|----------|---------|
| AgentCardRegistry | [`0xC7eCF7Ad6ee71CB0d94f0eb00F46f1DDf432a808`](https://sepolia.etherscan.io/address/0xC7eCF7Ad6ee71CB0d94f0eb00F46f1DDf432a808) |
| AgentCardVerifyHook | [`0xf3be150cd4EC0819bef95890DeeE0B71d9C94F6b`](https://sepolia.etherscan.io/address/0xf3be150cd4EC0819bef95890DeeE0B71d9C94F6b) |
| ERC8004IdentityRegistry | [`0x5B0763c3649eee889966dF478a73e53Df0420C84`](https://sepolia.etherscan.io/address/0x5B0763c3649eee889966dF478a73e53Df0420C84) |
| ERC8004ReputationRegistry | [`0xE953B278fd2378BA4987FE07f71575dd3353C9a8`](https://sepolia.etherscan.io/address/0xE953B278fd2378BA4987FE07f71575dd3353C9a8) |
| ERC8004ValidationRegistry | [`0x97291e2D3023d166878ed45BBD176F92E5Fda098`](https://sepolia.etherscan.io/address/0x97291e2D3023d166878ed45BBD176F92E5Fda098) |

See [contracts/ethereum/README.md](contracts/ethereum/README.md) for deployment details and verification status.

### Sepolia Testnet — Legacy Contracts

| Contract | Address |
|----------|---------|
| SageRegistryV2 | [`0x487d45a678eb947bbF9d8f38a67721b13a0209BF`](https://sepolia.etherscan.io/address/0x487d45a678eb947bbF9d8f38a67721b13a0209BF) |
| ERC8004ValidationRegistry | [`0x4D31A11DdE882D2B2cdFB9cCf534FaA55A519440`](https://sepolia.etherscan.io/address/0x4D31A11DdE882D2B2cdFB9cCf534FaA55A519440) |
| ERC8004IdentityRegistry | [`0x02439d8DA11517603d0DE1424B33139A90969517`](https://sepolia.etherscan.io/address/0x02439d8DA11517603d0DE1424B33139A90969517) |

> **Note**: Legacy contracts are deprecated. Use AgentCardRegistry for new deployments.

## Multi-Language Bindings

SAGE provides bindings for multiple programming languages:

| Language | Type | Details |
|----------|------|---------|
| Go | Native | Primary implementation |
| C/C++ | Static/shared library | `.a`, `.so`/`.dylib`/`.dll` |
| Python | Web3.py + ctypes | Smart contract + library bindings |
| Rust | FFI | Via static library |
| JavaScript/TypeScript | Ethers.js | Smart contract bindings |

See [docs/BUILD.md](docs/BUILD.md) for library build instructions and integration examples.

## Documentation

### Core

- **[Documentation Index](docs/INDEX.md)** — Complete documentation catalog
- **[Architecture Guide](docs/ARCHITECTURE.md)** — System architecture and design patterns
- **[Build Instructions](docs/BUILD.md)** — Compilation, cross-platform, and library builds
- **[API Reference](docs/API.md)** — HTTP and gRPC API documentation

### Protocol & Security

- **[Handshake Protocol](docs/handshake/hpke-based-handshake-en.md)** — HPKE handshake specification
- **[Security Design](docs/dev/security-design.md)** — Security architecture
- **[KEM Key Integration](docs/KME_PUBLIC_KEY_INTEGRATION.md)** — X25519 KEM key support

### Smart Contracts

- **[Contracts README](contracts/ethereum/README.md)** — AgentCard contracts, deployment, testing
- **[AgentCard Migration Guide](docs/AGENTCARD_MIGRATION_GUIDE.md)** — Migrating from legacy registries

### Development

- **[Contributing Guide](CONTRIBUTING.md)** — How to contribute
- **[Testing Guide](docs/test/TESTING.md)** — Testing strategies and best practices
- **[CI/CD Pipeline](docs/CI-CD.md)** — Continuous integration workflows
- **[Coding Guidelines](docs/CODING_GUIDELINES.md)** — Code quality standards

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines.

```bash
# Quick development workflow
git checkout -b feature/my-feature
# ... make changes ...
make test && make lint
git commit -m "feat(scope): description"
git push origin feature/my-feature
# Open a Pull Request
```

## License

This project is licensed under **GNU Lesser General Public License v3.0** — see [LICENSE](LICENSE).

**You CAN**: Use SAGE in commercial/proprietary applications, modify and distribute it.

**You MUST**: Provide SAGE source code if distributed, allow library relinking, maintain LGPL-3.0 notices.

**You DON'T need to**: Open-source your application that uses SAGE.

**Smart Contracts** (`contracts/ethereum/`) are separately licensed under **MIT License** — see [contracts/ethereum/LICENSE](contracts/ethereum/LICENSE).

See also: [LGPL-3.0 Full Text](https://www.gnu.org/licenses/lgpl-3.0.html) | [INSTALL.md](INSTALL.md) | [NOTICE](NOTICE)

## Support & Acknowledgments

- **Issues**: [GitHub Issues](https://github.com/SAGE-X-project/sage/issues)
- **Discussions**: [GitHub Discussions](https://github.com/SAGE-X-project/sage/discussions)

### Resources

- [RFC 9421 — HTTP Message Signatures](https://datatracker.ietf.org/doc/rfc9421/)
- [RFC 9180 — HPKE](https://datatracker.ietf.org/doc/rfc9180/)
- [A2A Protocol](https://a2a-protocol.org/)
- [W3C DID Specification](https://www.w3.org/TR/did-core/)
- [Ethereum Development Docs](https://ethereum.org/developers)
- [Kaia Network Docs](https://docs.kaia.io)

### Acknowledgments

- RFC 9421 and RFC 9180 Working Groups
- A2A Protocol team
- Ethereum Foundation and Kaia Network
- Cloudflare CIRCL library
- Open source community

---

**Built by the SAGE Team**
