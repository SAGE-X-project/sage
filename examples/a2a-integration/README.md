# A2A Integration Examples

This directory contains working examples demonstrating how to use SAGE with the A2A (Agent-to-Agent) protocol for secure multi-key agent communication.

## Overview

The examples progress from basic agent registration to advanced secure messaging:

1. **01-register-agent**: Register a multi-key agent with ECDSA, Ed25519, and X25519 keys
2. **02-generate-card**: Generate and export an A2A Agent Card for agent discovery
3. **03-exchange-cards**: Exchange and verify A2A cards between agents
4. **04-secure-message**: Establish secure channels and exchange encrypted messages

## Prerequisites

### 1. Local Blockchain (Hardhat)

```bash
cd contracts/ethereum
npm install
npx hardhat node
```

Keep this running in a separate terminal.

### 2. Deploy SageRegistryV4

In another terminal:

```bash
cd contracts/ethereum
npx hardhat run scripts/deploy-v4-local.js --network localhost
```

Note the deployed contract address (e.g., `0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9`).

### 3. Build SAGE CLI

```bash
go build -o build/bin/sage-did ./cmd/sage-did
```

### 4. Set Environment Variables

```bash
export REGISTRY_ADDRESS="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
export RPC_URL="http://localhost:8545"
export PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
```

## Running the Examples

Each example is self-contained and can be run independently:

```bash
# Example 01: Register a multi-key agent
cd 01-register-agent
go run main.go

# Example 02: Generate an A2A card
cd 02-generate-card
go run main.go

# Example 03: Exchange and verify cards
cd 03-exchange-cards
go run main.go

# Example 04: Secure messaging
cd 04-secure-message
go run main.go
```

## Example Details

### Example 01: Multi-Key Agent Registration

**Purpose:** Demonstrates registering an agent with multiple cryptographic keys

**What it does:**
- Generates ECDSA (secp256k1) key for Ethereum
- Generates Ed25519 key for signing
- Generates X25519 key for encryption
- Registers all keys in a single transaction
- Shows Ed25519 approval workflow

**Key concepts:**
- Multi-key registration
- Automatic signature generation
- Key type handling

### Example 02: A2A Card Generation

**Purpose:** Shows how to generate and export A2A Agent Cards

**What it does:**
- Resolves agent metadata from blockchain
- Converts to A2A Agent Card format
- Exports to JSON file
- Validates card structure

**Key concepts:**
- A2A card structure
- Public key formatting
- Card validation

### Example 03: Card Exchange and Verification

**Purpose:** Demonstrates agent-to-agent card discovery and verification

**What it does:**
- Agent A generates and shares card
- Agent B receives and validates card
- Verifies DID ownership
- Cross-checks with blockchain data

**Key concepts:**
- Card exchange protocol
- DID verification
- Trust establishment

### Example 04: Secure Message Exchange

**Purpose:** End-to-end encrypted messaging between agents

**What it does:**
- Establishes secure channel using X25519 keys
- Performs ECDH key exchange
- Encrypts messages with HPKE
- Signs messages with Ed25519
- Verifies signatures

**Key concepts:**
- ECDH key agreement
- HPKE encryption
- Message signing
- End-to-end security

## Architecture

```
┌─────────────┐         ┌─────────────┐
│   Agent A   │         │   Agent B   │
│             │         │             │
│  ECDSA Key  │         │  ECDSA Key  │
│ Ed25519 Key │         │ Ed25519 Key │
│ X25519 Key  │         │ X25519 Key  │
└──────┬──────┘         └──────┬──────┘
       │                       │
       │   A2A Card Exchange   │
       ├──────────────────────►│
       │◄──────────────────────┤
       │                       │
       │  Secure Channel (HPKE)│
       ├══════════════════════►│
       │◄══════════════════════┤
       │                       │
       └───────────┬───────────┘
                   │
                   ▼
         ┌──────────────────┐
         │  SageRegistryV4  │
         │   (Ethereum)     │
         └──────────────────┘
```

## Key Technologies

- **SAGE**: Secure Agent Guarantee Engine
- **A2A**: Agent-to-Agent protocol
- **DID**: Decentralized Identifiers (did:sage)
- **HPKE**: Hybrid Public Key Encryption (RFC 9180)
- **HTTP Signatures**: Message signing (RFC 9421)
- **Ethereum**: Smart contract platform

## Testing

Run all examples in sequence:

```bash
./run_all_examples.sh
```

Or test individually:

```bash
cd 01-register-agent
go test -v
```

## Troubleshooting

### "Connection refused" error

Ensure Hardhat node is running:
```bash
npx hardhat node
```

### "Contract not found" error

Deploy the V4 contract:
```bash
npx hardhat run scripts/deploy-v4-local.js --network localhost
```

### "Invalid key format" error

Check that key files are in the correct format:
- ECDSA: PEM or JWK
- Ed25519: .ed25519 or JWK
- X25519: .x25519 or raw bytes

## Further Reading

- [SAGE Architecture](../../docs/ARCHITECTURE.md)
- [Multi-Key Design](../../contracts/MULTI_KEY_DESIGN.md)
- [A2A Protocol Specification](https://github.com/a2aproject/a2a)
- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)

## License

LGPL-v3 - See [LICENSE](../../LICENSE) for details
