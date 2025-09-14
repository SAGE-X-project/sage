# SAGE - Secure Agent Guarantee Engine

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![Solidity Version](https://img.shields.io/badge/Solidity-0.8.19-red.svg)](https://soliditylang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

## Overview

SAGE (Secure Agent Guarantee Engine) is a comprehensive blockchain-based security framework for AI agent communication. It implements RFC 9421 HTTP Message Signatures to ensure secure, verifiable agent-to-agent interactions with decentralized identity (DID) management on multiple blockchains.

### Key Features

- **RFC 9421 Compliance**: Complete HTTP message signature implementation
- **Multi-Chain Support**: Ethereum, Solana, and Kaia network integration
- **Enhanced Security**: Public key ownership verification with on-chain validation
- **Multi-Algorithm Support**: Ed25519 and Secp256k1 cryptographic signatures
- **Smart Contract Registry**: Decentralized agent registry with revocation support
- **Key Management**: Secure key rotation and revocation mechanisms
- **Modular Architecture**: Clean separation of concerns with extensible design

## Project Structure

```
sage/
├── core/                    # Core RFC 9421 implementation
│   ├── rfc9421/            # HTTP message signatures
│   └── message/            # Message processing and validation
├── crypto/                  # Cryptographic operations
│   ├── keys/               # Key pair implementations
│   ├── chain/              # Blockchain-specific providers
│   └── storage/            # Key storage mechanisms
├── did/                     # Decentralized Identity
│   ├── ethereum/           # Ethereum DID client
│   └── solana/             # Solana DID client
├── contracts/               # Smart contracts
│   └── ethereum/           # Ethereum contracts and tests
├── cmd/                     # CLI applications
│   ├── sage-crypto/        # Cryptographic operations CLI
│   └── sage-did/           # DID management CLI
└── examples/               # Usage examples

```

## Installation

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ and npm (for smart contract development)
- Git

### Quick Start

1. **Clone the repository**

```bash
git clone https://github.com/sage-x-project/sage.git
cd sage
```

2. **Install Go dependencies**

```bash
go mod download
```

3. **Install smart contract dependencies**

```bash
cd contracts/ethereum
npm install
```

4. **Build the project**

```bash
# Build CLI tools
go build -o bin/sage-crypto ./cmd/sage-crypto
go build -o bin/sage-did ./cmd/sage-did

# Compile smart contracts
cd contracts/ethereum
npm run compile
```

## Configuration

### Environment Setup

Create a `.env` file in `contracts/ethereum/`:

```env
# Network RPC Endpoints
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
KAIA_RPC_URL=https://public-en.node.kaia.io
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io

# Private Keys (use test keys only!)
PRIVATE_KEY=your_private_key_here
MNEMONIC=your_twelve_word_mnemonic_phrase_here

# Contract Addresses
SAGE_REGISTRY_ADDRESS=0x...
```

## Usage

### 1. Generate Key Pairs

```bash
# Generate Ed25519 key pair
./bin/sage-crypto generate -t ed25519 -o agent.key

# Generate Secp256k1 key pair (for Ethereum)
./bin/sage-crypto generate -t secp256k1 -o ethereum.key
```

### 2. Register an AI Agent

```bash
# Register on Ethereum
./bin/sage-did register \
  --chain ethereum \
  --key ethereum.key \
  --name "My AI Agent" \
  --endpoint "https://api.myagent.com" \
  --capabilities "chat,code,analysis"
```

### 3. Create RFC 9421 Signed Messages

```go
import (
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
)

// Create a signed HTTP message
msg := &rfc9421.Message{
    Method:    "POST",
    Path:      "/api/v1/chat",
    Headers:   headers,
    Body:      []byte(requestBody),
    KeyID:     "did:sage:ethereum:0x...",
    Algorithm: rfc9421.AlgorithmEd25519,
}

signature, err := signer.Sign(msg)
```

## Testing

### Run Go Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./crypto/...
go test ./did/...
go test ./core/...
```

### Run Smart Contract Tests

```bash
cd contracts/ethereum

# Run all contract tests
npm test

# Run specific test suite
npm run test:v2

# Run with coverage
npm run coverage
```

## Smart Contract Features

### SageRegistryV2 - Enhanced Security Features

The latest version includes significant security enhancements:

- **Public Key Validation**: Comprehensive validation of secp256k1 keys
- **Ownership Proof**: Signature-based proof of key ownership
- **Key Revocation**: Ability to revoke compromised keys
- **Hook System**: Extensible validation through hook contracts
- **Gas Optimized**: Efficient storage patterns and operations

### Key Security Mechanisms

1. **Challenge-Response Authentication**

```solidity
bytes32 challenge = keccak256(abi.encodePacked(
    "SAGE Key Registration:",
    chainId,
    contractAddress,
    msg.sender,
    keyHash
));
```

2. **Format Validation**

- Uncompressed keys (65 bytes): `0x04` prefix
- Compressed keys (33 bytes): `0x02` or `0x03` prefix
- Ed25519 keys: Not supported on-chain

3. **Revocation System**

- Immediate key deactivation
- Automatic agent deactivation
- Prevention of key reuse

## Gas Usage

| Operation        | Gas Used | USD (@ 30 gwei) |
| ---------------- | -------- | --------------- |
| Register Agent   | ~653,000 | ~$50            |
| Update Agent     | ~85,000  | ~$7             |
| Revoke Key       | ~45,000  | ~$3.5           |
| Deactivate Agent | ~35,000  | ~$2.7           |

## Supported Networks

### Mainnet

- **Ethereum**: Full support with ENS integration
- **Kaia (Cypress)**: Production deployment
- **Solana**: In development

### Testnet

- **Sepolia**: Ethereum testnet
- **Kairos**: Kaia testnet
- **Solana Devnet**: Testing environment

## Multi-Language Bindings

SAGE provides bindings for multiple programming languages:

- **Go**: Native implementation
- **Python**: Web3.py based bindings
- **Rust**: Ethers-rs based bindings
- **JavaScript/TypeScript**: Ethers.js bindings
- **Java**: Web3j based bindings (coming soon)

Example usage in Python:

```python
from sage_contracts import SageRegistry

registry = SageRegistry(
    rpc_url="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
    contract_address="0x...",
    private_key="0x..."
)

# Register an agent
tx_hash = registry.register_agent(
    did="did:sage:ethereum:0x...",
    name="Python AI Agent",
    endpoint="https://api.example.com",
    public_key=public_key_bytes,
    capabilities=["chat", "analysis"]
)
```

## Security Considerations

1. **Private Key Management**

   - Never commit private keys to version control
   - Use hardware wallets for production
   - Implement key rotation policies

2. **Smart Contract Security**

   - Contracts are upgradeable through proxy pattern
   - Regular security audits recommended
   - Bug bounty program available

3. **Message Signature Verification**
   - Always verify signatures on the receiving end
   - Check signature expiration
   - Validate signer's DID status

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Code Style

- Go: Follow standard Go formatting (`gofmt`)
- Solidity: Follow Solidity style guide
- Use meaningful commit messages
- Add tests for new features

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Resources

- [RFC 9421 Specification](https://datatracker.ietf.org/doc/rfc9421/)
- [W3C DID Specification](https://www.w3.org/TR/did-core/)
- [Ethereum Development Docs](https://ethereum.org/developers)
- [Kaia Network Docs](https://docs.kaia.io)

## Support

- **Issues**: [GitHub Issues](https://github.com/sage-x-project/sage/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sage-x-project/sage/discussions)

## Acknowledgments

- RFC 9421 Working Group for HTTP Message Signatures specification
- Ethereum Foundation for blockchain infrastructure
- Kaia Network team for multi-chain support
- Open source community for continuous feedback and contributions

---

**Built by the SAGE Team**
