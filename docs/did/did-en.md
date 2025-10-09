# SAGE DID Package

A Go package providing Decentralized Identifier (DID) functionality for AI agents in the SAGE (Secure Agent Guarantee Engine) project.

## Key Features

- **Multi-chain Support**: Ethereum (deployed on Sepolia) and Solana (in development)
- **Agent Registration**: Register AI agents with unique DIDs on blockchain
- **DID Resolution**: Retrieve agent metadata and public keys from blockchain
- **Metadata Verification**: Verify agent information against on-chain data
- **Agent Management**: Update metadata and deactivate agents
- **Owner-based Discovery**: List all agents owned by an address
- **RFC-9421 Integration**: Works with SAGE's signature verification system
- **HPKE/KEM Support**: Stores both signing and key encapsulation public keys
- **Factory Pattern**: Flexible client creation for different blockchains

## Installation

```bash
go get github.com/sage-x-project/sage/did
```

## Architecture

### Package Structure

```
did/
├── types.go              # Core types and interfaces
├── did.go                # DID parsing and generation
├── client.go             # Client interface definition
├── manager.go            # DID manager (orchestrates registry/resolver/verifier)
├── factory.go            # ClientFactory for creating chain-specific clients
├── registry.go           # MultiChainRegistry implementation
├── resolver.go           # MultiChainResolver implementation
├── verification.go       # MetadataVerifier implementation
├── utils.go              # Utility functions
├── ethereum/             # Ethereum blockchain client
│   ├── client.go        # Ethereum DID operations
│   ├── resolver.go      # Ethereum-specific resolution
│   ├── abi.go           # Contract ABI definition
│   └── SageRegistryV2.abi.json # Contract ABI JSON
└── solana/              # Solana blockchain client (in development)
    ├── client.go        # Solana DID operations
    └── resolver.go      # Solana-specific resolution
```

### Integration with Core Module

The DID module is designed to work seamlessly with the SAGE core module:

1. **DID Module**: Retrieves agent metadata and public keys from blockchain
2. **Core Module**: Performs RFC-9421 signature verification using DID data
3. **Verification Service**: Orchestrates DID resolution and signature verification

## Build Instructions

### Building the CLI Tool

```bash
# Run from project root
go build -o sage-did ./cmd/sage-did

# Or use go install
go install ./cmd/sage-did
```

### Running Tests

```bash
# Run all tests
go test ./did/...

# Run tests with verbose output
go test -v ./did/...

# Test specific packages
go test ./did
go test ./did/ethereum
go test ./did/solana
```

## Usage

### 1. Programmatic Usage

#### Creating a DID Manager

```go
package main

import (
    "context"
    "github.com/sage-x-project/sage/did"
)

func main() {
    // Create DID manager
    manager := did.NewManager()
    
    // Configure for Ethereum
    ethConfig := &did.RegistryConfig{
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/your-api-key",
        ContractAddress: "0x1234567890abcdef...",
        PrivateKey:      "your-private-key", // For gas fees
    }
    manager.Configure(did.ChainEthereum, ethConfig)
    
    // Configure for Solana
    solConfig := &did.RegistryConfig{
        RPCEndpoint:     "https://api.mainnet-beta.solana.com",
        ContractAddress: "YourProgramID11111111111111111111",
        PrivateKey:      "your-private-key", // For transaction fees
    }
    manager.Configure(did.ChainSolana, solConfig)
}
```

#### Registering an AI Agent

```go
import (
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
)

// Generate key pair (Ed25519 for Solana, Secp256k1 for Ethereum)
keyPair, _ := keys.GenerateEd25519KeyPair()

// Create registration request
req := &did.RegistrationRequest{
    DID:         "did:sage:solana:agent001",
    Name:        "My AI Agent",
    Description: "An intelligent assistant",
    Endpoint:    "https://api.myagent.com",
    Capabilities: map[string]interface{}{
        "chat": true,
        "code": true,
        "search": false,
    },
    KeyPair: keyPair,
}

// Register agent
ctx := context.Background()
result, err := manager.RegisterAgent(ctx, did.ChainSolana, req)
if err != nil {
    panic(err)
}

fmt.Printf("Agent registered! TX: %s\n", result.TransactionHash)
```

#### Resolving Agent Metadata

```go
// Resolve agent DID
agentDID := did.AgentDID("did:sage:ethereum:agent001")
metadata, err := manager.ResolveAgent(ctx, agentDID)
if err != nil {
    panic(err)
}

fmt.Printf("Agent Name: %s\n", metadata.Name)
fmt.Printf("Endpoint: %s\n", metadata.Endpoint)
fmt.Printf("Active: %v\n", metadata.IsActive)
```

#### Integration with Verification Service

```go
import (
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
)

// Create verification service with DID resolver
verifier := core.NewVerificationService(manager)

// Verify an agent message
message := &rfc9421.Message{
    AgentDID:  "did:sage:ethereum:agent001",
    Body:      []byte("Hello from AI agent"),
    Signature: signature,
    // ... other fields
}

result, err := verifier.VerifyAgentMessage(ctx, message, opts)
if result.Valid {
    fmt.Println("Message verified successfully!")
}
```

### 2. CLI Tool Usage

#### Agent Registration

```bash
# Register an agent on Ethereum
./sage-did register \
    --chain ethereum \
    --name "My Assistant" \
    --endpoint "https://api.myagent.com" \
    --description "AI coding assistant" \
    --capabilities '{"chat":true,"code":true}' \
    --key agent-key.jwk \
    --private-key "0x..." # For gas fees

# Register on Solana with key from storage
./sage-did register \
    --chain solana \
    --name "Solana Agent" \
    --endpoint "https://api.solana-agent.com" \
    --storage-dir ./keys \
    --key-id my-agent-key \
    --rpc "https://api.devnet.solana.com" # Use devnet for testing
```

#### DID Resolution

```bash
# Resolve agent metadata
./sage-did resolve did:sage:ethereum:agent001

# Save metadata to file
./sage-did resolve did:sage:solana:agent002 \
    --output agent-metadata.json \
    --format json

# Custom RPC endpoint
./sage-did resolve did:sage:ethereum:agent001 \
    --rpc "https://eth-mainnet.g.alchemy.com/v2/your-key"
```

#### List Agents by Owner

```bash
# List all agents owned by an Ethereum address
./sage-did list \
    --chain ethereum \
    --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

# List Solana agents with JSON output
./sage-did list \
    --chain solana \
    --owner 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM \
    --format json \
    --output my-agents.json
```

#### Update Agent Metadata

```bash
# Update agent name and endpoint
./sage-did update did:sage:ethereum:agent001 \
    --name "Updated Agent Name" \
    --endpoint "https://new-api.myagent.com" \
    --key agent-key.jwk

# Update capabilities
./sage-did update did:sage:solana:agent002 \
    --capabilities '{"chat":true,"code":true,"image":true}' \
    --storage-dir ./keys \
    --key-id my-agent-key
```

#### Deactivate Agent

```bash
# Deactivate an agent (with confirmation)
./sage-did deactivate did:sage:ethereum:agent001 \
    --key agent-key.jwk

# Skip confirmation prompt
./sage-did deactivate did:sage:solana:agent002 \
    --storage-dir ./keys \
    --key-id my-agent-key \
    --yes
```

#### Verify Metadata

```bash
# Verify local metadata against blockchain
./sage-did verify did:sage:ethereum:agent001 \
    --metadata local-metadata.json

# Verify with custom RPC
./sage-did verify did:sage:solana:agent002 \
    --metadata agent-data.json \
    --rpc "https://api.mainnet-beta.solana.com"
```

## Blockchain Configuration

### Ethereum Configuration

| Network | RPC Endpoint | SageRegistryV2 Address | Status |
|---------|-------------|------------------------|--------|
| Mainnet | https://eth-mainnet.g.alchemy.com/v2/{key} | TBD | Planned |
| Sepolia | https://eth-sepolia.g.alchemy.com/v2/{key} | `0x487d45a678eb947bbF9d8f38a67721b13a0209BF` | **✅ Deployed** |
| Holesky | https://eth-holesky.g.alchemy.com/v2/{key} | TBD | Planned |

**Note**: Sepolia testnet is currently recommended for testing.

### Solana Configuration

| Network | RPC Endpoint | Program ID | Status |
|---------|-------------|------------|--------|
| Mainnet | https://api.mainnet-beta.solana.com | TBD | Planned |
| Devnet | https://api.devnet.solana.com | TBD | In Development |
| Testnet | https://api.testnet.solana.com | TBD | In Development |

**Note**: Solana integration is currently under development. Basic client implementation exists but requires on-chain program deployment.

## DID Format

SAGE DIDs follow this format:
```
did:sage:<chain>:<agent-id>
```

Examples:
- `did:sage:ethereum:agent001`
- `did:sage:solana:agent_abc123`

## Real-World Examples

### 1. Complete Agent Lifecycle

```bash
# 1. Generate appropriate key for the blockchain
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./keys --key-id solana-agent

# 2. Register agent on Solana
./sage-did register \
    --chain solana \
    --name "AI Assistant v1" \
    --endpoint "https://assistant.example.com/api" \
    --description "General purpose AI assistant" \
    --capabilities '{"chat":true,"code":true,"search":true}' \
    --storage-dir ./keys \
    --key-id solana-agent

# 3. Resolve and verify registration
./sage-did resolve did:sage:solana:agent_12345 --format json

# 4. Update endpoint after migration
./sage-did update did:sage:solana:agent_12345 \
    --endpoint "https://new.assistant.example.com/api" \
    --storage-dir ./keys \
    --key-id solana-agent

# 5. List all agents owned by the address
./sage-did list --chain solana \
    --owner YourSolanaAddress111111111111111111111111111

# 6. Deactivate agent when no longer needed
./sage-did deactivate did:sage:solana:agent_12345 \
    --storage-dir ./keys \
    --key-id solana-agent \
    --yes
```

### 2. Multi-Chain Agent Management

```bash
# Register same agent on multiple chains
# First on Ethereum
./sage-crypto generate --type secp256k1 --format storage \
    --storage-dir ./keys --key-id eth-agent

./sage-did register \
    --chain ethereum \
    --name "CrossChain AI" \
    --endpoint "https://api.crosschain-ai.com" \
    --storage-dir ./keys \
    --key-id eth-agent

# Then on Solana
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./keys --key-id sol-agent

./sage-did register \
    --chain solana \
    --name "CrossChain AI" \
    --endpoint "https://api.crosschain-ai.com" \
    --storage-dir ./keys \
    --key-id sol-agent
```

### 3. Programmatic Integration Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
)

func main() {
    ctx := context.Background()
    
    // Setup DID manager
    manager := did.NewManager()
    manager.Configure(did.ChainEthereum, &did.RegistryConfig{
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/key",
        ContractAddress: "0x...",
    })
    
    // Register an agent
    keyPair, _ := keys.GenerateSecp256k1KeyPair()
    req := &did.RegistrationRequest{
        DID:      did.GenerateDID(did.ChainEthereum, keyPair),
        Name:     "My Agent",
        Endpoint: "https://agent.example.com",
        KeyPair:  keyPair,
    }
    
    result, err := manager.RegisterAgent(ctx, did.ChainEthereum, req)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use with verification service
    verifier := core.NewVerificationService(manager)
    
    // Create and sign a message
    message := &rfc9421.Message{
        AgentDID: req.DID,
        Body:     []byte("Hello from agent"),
    }
    
    // Sign message
    signer := rfc9421.NewSigner()
    signature, _ := signer.SignMessage(keyPair, message)
    message.Signature = signature
    
    // Verify message
    verifyResult, _ := verifier.VerifyAgentMessage(ctx, message, nil)
    if verifyResult.Valid {
        log.Println("Message verified!")
    }
}
```

## Security Considerations

1. **Private Key Management**: Never expose private keys. Use environment variables or secure key management systems.

2. **Transaction Fees**: Both Ethereum and Solana require native tokens (ETH/SOL) for transaction fees.

3. **Agent Deactivation**: Deactivated agents cannot be reactivated. Ensure you want to deactivate before proceeding.

4. **Metadata Updates**: Only the agent owner (key holder) can update or deactivate an agent.

## Error Handling

### Common Errors

#### DID Not Found
```
Error: DID not found in registry
```
The specified DID does not exist on the blockchain.

#### Invalid Key Type
```
Error: Ethereum requires Secp256k1 keys, got Ed25519
```
Use the correct key type for each blockchain:
- Ethereum: Secp256k1
- Solana: Ed25519

#### Insufficient Balance
```
Error: insufficient funds for gas
```
Ensure the transaction signer has enough ETH/SOL for fees.

#### Permission Denied
```
Error: only agent owner can update metadata
```
Use the same key that registered the agent.

## Advanced Features

### Custom Contract Deployment

For private deployments, you can deploy your own DID registry contracts:

1. Deploy the appropriate contract for your blockchain
2. Configure the DID manager with your contract address
3. Use the same CLI commands with custom `--contract` flag

### Off-Chain Indexing

For better performance with large-scale queries:

1. Use event listeners to index DID registrations
2. Store indexed data in a database
3. Implement the `SearchAgents` functionality

## Implementation Status & Roadmap

### ✅ Completed
- Ethereum Sepolia integration (SageRegistryV2 deployed)
- Multi-chain architecture with factory pattern
- DID resolution and verification
- Agent registration and metadata management
- Integration with crypto package (Ed25519, Secp256k1, X25519)
- RFC-9421 algorithm mapping

### 🚧 In Progress
- Solana on-chain program development
- HPKE/KEM key integration for handshake protocol
- Off-chain indexing for efficient queries
- Enhanced search capabilities

### 📋 Planned
- Ethereum Mainnet deployment
- Kaia blockchain integration
- Multi-signature agent ownership
- Delegation and authorization framework
- Agent capability verification system

## Key Type Support

The DID package integrates with SAGE crypto package and supports:

| Blockchain | Signing Key | KEM Key (HPKE) | RFC 9421 Algorithm |
|------------|-------------|----------------|-------------------|
| Ethereum   | Secp256k1   | X25519         | es256k            |
| Solana     | Ed25519     | X25519         | ed25519           |

**Note**: `PublicKEMKey` field in `AgentMetadata` stores the X25519 public key used for HPKE-based secure handshake protocol.

## License

Provided as part of the SAGE project.