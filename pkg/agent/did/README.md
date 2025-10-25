# SAGE Decentralized Identity (DID) Management

The `did` package provides comprehensive decentralized identity management for AI agents using blockchain-based registries. It implements the W3C DID standard with multi-chain support for Ethereum and Solana, enabling verifiable agent authentication and discovery.

## Overview

SAGE agents use blockchain-anchored DIDs to establish trust and identity. Each agent receives a unique, verifiable identifier (DID) registered on-chain with cryptographic keys, metadata, and capabilities. This enables secure peer-to-peer agent communication without centralized identity providers.

### Key Benefits

- **Decentralized Trust**: No central identity authority required
- **Blockchain Anchored**: Immutable registration on Ethereum/Solana
- **Multi-Key Support**: Up to 10 keys per agent (SageRegistryV4)
- **W3C Compliant**: Follows DID Core 1.0 specification
- **Multi-Chain**: Ethereum and Solana support
- **A2A Integration**: Google A2A Agent Card compatibility
- **Verifiable Metadata**: On-chain agent capabilities and endpoints

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SAGE Components (handshake, session, messaging)        │
│  - Agent authentication                                  │
│  - Peer discovery                                        │
│  - Trust establishment                                   │
└────────────────────┬────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────┐
│  did.Manager (multi-chain DID operations)               │
│  - RegisterAgent()                                       │
│  - ResolveAgent()                                        │
│  - ValidateAgent()                                       │
│  - UpdateAgent() [V4]                                    │
└────────────────────┬────────────────────────────────────┘
                     │ manages
          ┌──────────┴──────────┬──────────────────┐
          ▼                     ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐
│ Multi-Chain      │  │ Multi-Chain      │  │ Metadata     │
│ Registry         │  │ Resolver         │  │ Verifier     │
├──────────────────┤  ├──────────────────┤  ├──────────────┤
│ • Ethereum V2    │  │ • On-chain query │  │ • DID format │
│ • Ethereum V4    │  │ • Caching layer  │  │ • Signature  │
│ • Solana         │  │ • Fallback       │  │ • Ownership  │
└──────────────────┘  └──────────────────┘  └──────────────┘
          │                     │
          ▼                     ▼
┌─────────────────────────────────────────────────────────┐
│  Blockchain Smart Contracts                             │
│  ├─ Ethereum: SageRegistryV2 (legacy)                   │
│  ├─ Ethereum: SageRegistryV4 (multi-key) [RECOMMENDED] │
│  └─ Solana: Program (Ed25519 keys)                      │
└─────────────────────────────────────────────────────────┘
```

## DID Format

SAGE follows the W3C DID specification with blockchain-specific identifiers:

### Ethereum DIDs

```
did:sage:ethereum:<ethereum-address>
```

**Example:**
```
did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
```

**Components:**
- `did` - DID scheme
- `sage` - DID method (SAGE protocol)
- `ethereum` - Chain identifier
- `0xf39f...` - Ethereum address (derived from public key)

### Solana DIDs

```
did:sage:solana:<base58-public-key>
```

**Example:**
```
did:sage:solana:CuieVDEDtLo7FypA9SbLM9saXFdb1dsshEkyErMqkRQq
```

**Components:**
- `did` - DID scheme
- `sage` - DID method
- `solana` - Chain identifier
- `CuieV...` - Base58-encoded Ed25519 public key

## Supported Blockchains

### Ethereum

**Registry Versions:**

#### SageRegistryV2 (Legacy)
- **Status**: Deployed on Sepolia testnet
- **Features**: Single key per agent, basic metadata
- **Contract**: [`0x487d45a678eb947bbF9d8f38a67721b13a0209BF`](https://sepolia.etherscan.io/address/0x487d45a678eb947bbF9d8f38a67721b13a0209BF)
- **Use Case**: Legacy systems, simple deployments

#### SageRegistryV4 (Recommended)
- **Status**: Production ready, awaiting deployment
- **Features**:
  - Multi-key support (up to 10 keys per agent)
  - Ed25519 and ECDSA/secp256k1
  - Agent metadata updates with nonce-based replay protection
  - Atomic key rotation
  - Public key ownership verification (CVE fixes)
- **Use Case**: Production deployments, multi-chain agents
- **Documentation**: [V4 Deployment Guide](../../../docs/V4_UPDATE_DEPLOYMENT_GUIDE.md)

**Key Algorithm:**
- Secp256k1 (ECDSA) - Ethereum-compatible
- Ed25519 - Supported in V4

**Address Derivation:**
- Keccak256 hash of uncompressed public key
- Last 20 bytes as Ethereum address

**Gas Costs:**
- Registration: ~653,000 gas (~$20-50 at 50 gwei)
- Resolution: Free (view function)
- Update (V4): ~100,000 gas

### Solana

**Registry:**
- Solana program (not yet deployed)
- Native Ed25519 support
- Base58 address encoding

**Key Algorithm:**
- Ed25519 only

**Address Derivation:**
- Base58-encoded Ed25519 public key (32 bytes)

**Transaction Costs:**
- Registration: ~5,000 lamports (~$0.0001)
- Resolution: Free (account query)

## Core Components

### Manager

Unified multi-chain DID management:

```go
type Manager struct {
    registry *MultiChainRegistry
    resolver *MultiChainResolver
    verifier *MetadataVerifier
}

// Core operations
func NewManager() *Manager
func (m *Manager) Configure(chain Chain, config *RegistryConfig) error
func (m *Manager) RegisterAgent(ctx context.Context, chain Chain, req *RegistrationRequest) (*RegistrationResult, error)
func (m *Manager) ResolveAgent(ctx context.Context, did AgentDID) (*AgentMetadata, error)
func (m *Manager) ValidateAgent(ctx context.Context, did AgentDID, opts *ValidationOptions) (*AgentMetadata, error)
func (m *Manager) UpdateAgent(ctx context.Context, did AgentDID, updates *UpdateRequest) error  // V4 only
```

**Features:**
- Chain-agnostic API
- Automatic chain detection from DID
- Built-in caching
- Verification and validation

### AgentMetadata

Complete agent information:

```go
type AgentMetadata struct {
    DID          AgentDID               `json:"did"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    Endpoint     string                 `json:"endpoint"`       // HTTPS endpoint
    PublicKey    interface{}            `json:"public_key"`     // Signing key
    PublicKEMKey interface{}            `json:"public_kem_key"` // HPKE key (optional)
    Capabilities map[string]interface{} `json:"capabilities"`
    Owner        string                 `json:"owner"`          // Blockchain address
    IsActive     bool                   `json:"is_active"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

### AgentMetadataV4 (Multi-Key Support)

Extended metadata for SageRegistryV4:

```go
type AgentMetadataV4 struct {
    DID          AgentDID               `json:"did"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    Endpoint     string                 `json:"endpoint"`
    Keys         []AgentKey             `json:"keys"`          // Multiple keys
    Capabilities map[string]interface{} `json:"capabilities"`
    Owner        string                 `json:"owner"`
    Nonce        uint64                 `json:"nonce"`         // Replay protection
    IsActive     bool                   `json:"is_active"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}

type AgentKey struct {
    Type     string `json:"type"`      // "Ed25519" or "ECDSA"
    KeyData  []byte `json:"key_data"`  // Raw public key bytes
    Verified bool   `json:"verified"`  // Ownership verification
}
```

### Registry Configuration

```go
type RegistryConfig struct {
    ContractAddress string `json:"contract_address"` // Smart contract address
    RPCEndpoint     string `json:"rpc_endpoint"`     // Blockchain RPC URL
    PrivateKey      string `json:"private_key"`      // Owner private key
    RegistryVersion string `json:"registry_version"` // "v2" or "v4"
}
```

### Resolver

DID resolution with caching:

```go
type MultiChainResolver struct {
    cache *lru.Cache // LRU cache for performance
}

func (r *MultiChainResolver) Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error)
func (r *MultiChainResolver) ResolveWithOptions(ctx context.Context, did AgentDID, opts *ResolutionOptions) (*AgentMetadata, error)
```

**Features:**
- LRU caching (default 1000 entries)
- Cache TTL: 5 minutes
- Automatic cache invalidation
- Fallback to on-chain query

### Verifier

Metadata and signature verification:

```go
type MetadataVerifier struct {
    resolver *MultiChainResolver
}

func (v *MetadataVerifier) VerifySignature(message []byte, signature []byte, did AgentDID) error
func (v *MetadataVerifier) VerifyOwnership(did AgentDID, ownerAddress string) error
func (v *MetadataVerifier) VerifyCapabilities(did AgentDID, requiredCapabilities []string) error
```

## Usage Examples

### Basic DID Generation

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"
)

// Generate Ethereum key pair
ethProvider := ethereum.NewProvider()
keyPair, _ := ethProvider.GenerateKeyPair()

// Derive Ethereum address
address := ethProvider.DeriveAddress(keyPair.PublicKey())

// Generate DID
agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, address)
fmt.Println(agentDID)
// Output: did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
```

### Multiple Agents Per Owner (Nonce-based)

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Same owner can have multiple agents
did1 := did.GenerateAgentDIDWithNonce(did.ChainEthereum, ownerAddr, 0)
did2 := did.GenerateAgentDIDWithNonce(did.ChainEthereum, ownerAddr, 1)
did3 := did.GenerateAgentDIDWithNonce(did.ChainEthereum, ownerAddr, 2)

// Each DID is unique but verifiable to the same owner
fmt.Println(did1) // did:sage:ethereum:0xabc...001
fmt.Println(did2) // did:sage:ethereum:0xdef...002
fmt.Println(did3) // did:sage:ethereum:0x123...003
```

### Agent Registration (V2 - Single Key)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// Create manager
manager := did.NewManager()

// Configure Ethereum V2 registry
config := &did.RegistryConfig{
    ContractAddress: "0x487d45a678eb947bbF9d8f38a67721b13a0209BF", // Sepolia V2
    RPCEndpoint:     "https://sepolia.infura.io/v3/YOUR-PROJECT-ID",
    PrivateKey:      "0x...", // Owner private key
    RegistryVersion: "v2",
}
err := manager.Configure(did.ChainEthereum, config)

// Generate key pair
cryptoManager := crypto.NewManager()
keyPair, _ := cryptoManager.GenerateKeyPair(crypto.KeyTypeSecp256k1)

// Create registration request
req := &did.RegistrationRequest{
    DID:          "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    Name:         "Trading Agent",
    Description:  "Automated trading AI agent",
    Endpoint:     "https://agent.example.com",
    Capabilities: map[string]interface{}{
        "trading": true,
        "analysis": true,
    },
    KeyPair: keyPair,
}

// Register agent
result, err := manager.RegisterAgent(context.Background(), did.ChainEthereum, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Agent registered! TX: %s\n", result.TransactionHash)
fmt.Printf("Gas used: %d\n", result.GasUsed)
```

### Agent Registration (V4 - Multi-Key)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// Configure Ethereum V4 registry
config := &did.RegistryConfig{
    ContractAddress: "0x...", // V4 contract address
    RPCEndpoint:     "https://sepolia.infura.io/v3/YOUR-PROJECT-ID",
    PrivateKey:      "0x...",
    RegistryVersion: "v4",
}
manager.Configure(did.ChainEthereum, config)

// Generate multiple keys
ed25519Key, _ := cryptoManager.GenerateKeyPair(crypto.KeyTypeEd25519)
secp256k1Key, _ := cryptoManager.GenerateKeyPair(crypto.KeyTypeSecp256k1)

// Prepare keys for V4 registration
keys := []did.AgentKey{
    {
        Type:     "Ed25519",
        KeyData:  ed25519Key.PublicKey().([]byte),
        Verified: true,
    },
    {
        Type:     "ECDSA",
        KeyData:  secp256k1Key.PublicKey().([]byte),
        Verified: true,
    },
}

req := &did.RegistrationRequest{
    DID:          "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    Name:         "Multi-Chain Agent",
    Description:  "Agent with Ed25519 and ECDSA keys",
    Endpoint:     "https://agent.example.com",
    Capabilities: map[string]interface{}{
        "ethereum": true,
        "solana":   true,
    },
    Keys:    keys,
    KeyPair: secp256k1Key, // Primary key for transaction signing
}

result, err := manager.RegisterAgent(context.Background(), did.ChainEthereum, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Multi-key agent registered! TX: %s\n", result.TransactionHash)
```

### Agent Resolution

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Resolve agent by DID
agentDID := did.AgentDID("did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
metadata, err := manager.ResolveAgent(context.Background(), agentDID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Agent: %s\n", metadata.Name)
fmt.Printf("Endpoint: %s\n", metadata.Endpoint)
fmt.Printf("Owner: %s\n", metadata.Owner)
fmt.Printf("Active: %v\n", metadata.IsActive)
fmt.Printf("Capabilities: %+v\n", metadata.Capabilities)
```

### Multi-Key Resolution (V4)

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Resolve with V4 metadata
metadataV4, err := manager.ResolveAgentV4(context.Background(), agentDID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Agent has %d keys:\n", len(metadataV4.Keys))
for i, key := range metadataV4.Keys {
    fmt.Printf("  Key %d: %s (verified: %v)\n", i+1, key.Type, key.Verified)
}

// Resolve specific key type
ethKey, err := manager.ResolvePublicKeyByType(context.Background(), agentDID, "ECDSA")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Ethereum key: %x\n", ethKey.KeyData)

solanaKey, err := manager.ResolvePublicKeyByType(context.Background(), agentDID, "Ed25519")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Solana key: %x\n", solanaKey.KeyData)
```

### Agent Validation

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Validate DID format and on-chain registration
opts := &did.ValidationOptions{
    CheckOnChain:     true,  // Verify blockchain registration
    RequireActive:    true,  // Agent must be active
    RequireEndpoint:  true,  // Endpoint must be set
    RequiredCapabilities: []string{"trading"}, // Must have trading capability
}

metadata, err := manager.ValidateAgent(context.Background(), agentDID, opts)
if err != nil {
    log.Fatal("Validation failed:", err)
}

fmt.Println("Agent validated successfully!")
```

### Signature Verification

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Message to verify
message := []byte("Hello, SAGE!")
signature := []byte{...} // Signature from agent

// Verify signature against DID
verifier := did.NewMetadataVerifier(resolver)
err := verifier.VerifySignature(message, signature, agentDID)
if err != nil {
    log.Fatal("Signature verification failed:", err)
}

fmt.Println("Signature verified! Agent identity confirmed.")
```

### Agent Metadata Update (V4)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Update agent metadata (name, description, endpoint, capabilities)
updates := &did.UpdateRequest{
    Name:        "Trading Agent v2.0",
    Description: "Enhanced trading agent with ML",
    Endpoint:    "https://agent-v2.example.com",
    Capabilities: map[string]interface{}{
        "trading":  true,
        "analysis": true,
        "ml-predictions": true,
    },
}

// Sign and submit update
err := manager.UpdateAgent(context.Background(), agentDID, updates)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Agent metadata updated successfully!")

// Nonce is automatically incremented to prevent replay attacks
```

### A2A Agent Card Integration

```go
import (
    "context"
    "encoding/json"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Generate A2A Agent Card from SAGE metadata
metadataV4, _ := manager.ResolveAgentV4(context.Background(), agentDID)
a2aCard, err := did.GenerateA2ACard(metadataV4)
if err != nil {
    log.Fatal(err)
}

// Export as JSON
cardJSON, _ := json.MarshalIndent(a2aCard, "", "  ")
fmt.Printf("A2A Agent Card:\n%s\n", cardJSON)

// Validate incoming A2A card
err = did.ValidateA2ACard(a2aCard)
if err != nil {
    log.Fatal("Invalid A2A card:", err)
}

// Merge capabilities from A2A card
err = did.MergeA2ACard(metadataV4, a2aCard)
if err != nil {
    log.Fatal(err)
}

fmt.Println("A2A capabilities merged successfully!")
```

### Key Rotation (V4)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Generate new key
newKey, _ := cryptoManager.GenerateKeyPair(crypto.KeyTypeEd25519)

// Prepare key rotation request
rotationReq := &did.KeyRotationRequest{
    OldKeyIndex: 0, // Index of key to replace
    NewKey: did.AgentKey{
        Type:     "Ed25519",
        KeyData:  newKey.PublicKey().([]byte),
        Verified: true,
    },
}

// Perform atomic key rotation
err := manager.RotateKey(context.Background(), agentDID, rotationReq)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Key rotated successfully!")
// Old key is automatically deactivated
// New key is immediately active
```

## V4 vs V2 Comparison

| Feature | V2 (Legacy) | V4 (Recommended) |
|---------|-------------|------------------|
| Keys per agent | 1 | Up to 10 |
| Key types | Secp256k1 only | Ed25519 + ECDSA |
| Metadata updates | ❌ Not supported | ✅ Full support |
| Replay protection | ❌ None | ✅ Nonce-based |
| Key rotation | ❌ Manual (new agent) | ✅ Atomic rotation |
| Ownership verification | ⚠️ Basic | ✅ Enhanced (CVE fixes) |
| Multi-chain agents | ⚠️ Workaround | ✅ Native support |
| Gas cost (registration) | ~500k | ~653k |
| Production ready | ✅ Yes (Sepolia) | ✅ Yes (pending deployment) |

**Migration Guide**: See [V4 Deployment Guide](../../../docs/V4_UPDATE_DEPLOYMENT_GUIDE.md)

## Directory Structure

```
pkg/agent/did/
├── README.md                    # This file
├── manager.go                   # Multi-chain DID manager
├── did.go                       # DID generation and utilities
├── types.go                     # Core types (V2)
├── types_v4.go                  # V4 types (multi-key)
├── registry.go                  # Multi-chain registry
├── resolver.go                  # Multi-chain resolver with caching
├── verification.go              # Signature and metadata verification
├── utils.go                     # Utility functions (Marshal/Unmarshal)
├── a2a.go                       # Google A2A Agent Card integration
├── a2a_proof.go                 # A2A proof of possession
├── key_proof.go                 # Public key ownership proofs
├── factory.go                   # DID client factory
├── client.go                    # Generic client interface
│
├── ethereum/                    # Ethereum DID client
│   ├── client.go                # V2 client (legacy)
│   ├── clientv4.go              # V4 client (multi-key)
│   ├── resolver.go              # Ethereum resolver
│   ├── abi.go                   # Smart contract ABI
│   └── *_test.go                # Comprehensive tests
│
└── solana/                      # Solana DID client
    ├── client.go                # Solana program client
    ├── resolver.go              # Solana resolver
    └── *_test.go                # Solana tests
```

## Testing

### Unit Tests

```bash
# Run all DID tests
go test ./pkg/agent/did/...

# Run with coverage
go test -cover ./pkg/agent/did/...

# Run specific tests
go test ./pkg/agent/did -run TestDIDGeneration
go test ./pkg/agent/did/ethereum -run TestRegisterAgent
```

### Integration Tests

```bash
# Ethereum integration (requires local node)
go test ./tests/integration -run TestDIDIntegration

# Start local Ethereum node first
# Hardhat: npx hardhat node
# Ganache: ganache-cli
```

### Performance Tests

```bash
# DID performance benchmarks
go test ./pkg/agent/did -run=^$ -bench=BenchmarkDID

# Expected results:
# - DID generation: ~0.1-1 μs
# - DID parsing: ~0.5-2 μs
# - Resolution (cached): ~10-50 μs
# - Resolution (on-chain): ~100-500 ms
```

## Security Considerations

### DID Generation

**Best Practices:**
- ✅ Use cryptographically secure key generation
- ✅ Derive DID from owner address for verification
- ✅ Use nonces for multiple agents per owner
- ❌ Never reuse keys across different agents
- ❌ Never share private keys

### Registration

**Security Checklist:**
- ✅ Verify DID ownership before registration
- ✅ Sign registration with private key
- ✅ Use V4 for production (enhanced security)
- ✅ Enable replay protection (V4 nonces)
- ✅ Verify public key ownership (CVE-SAGE-2025-001 fix)

### Resolution

**Cache Safety:**
- ✅ Cache TTL: 5 minutes (configurable)
- ✅ Invalidate cache on updates
- ✅ Fall back to on-chain query on cache miss
- ⚠️ Don't cache inactive agents

### Verification

**Signature Verification:**
```go
// Always verify signatures before trusting messages
err := verifier.VerifySignature(message, signature, agentDID)
if err != nil {
    // Reject message - invalid signature
    return fmt.Errorf("invalid signature: %w", err)
}

// Message is authentic and from the claimed agent
processMessage(message)
```

**Ownership Verification:**
```go
// Verify DID ownership before sensitive operations
err := verifier.VerifyOwnership(agentDID, expectedOwner)
if err != nil {
    // Reject - agent not owned by expected address
    return fmt.Errorf("ownership mismatch: %w", err)
}
```

## Performance

### Benchmark Results (Apple M1, Go 1.24)

| Operation | Time | Notes |
|-----------|------|-------|
| DID Generation | ~0.1-1 μs | In-memory computation |
| DID Parsing | ~0.5-2 μs | String parsing |
| DID Validation | ~1-5 μs | Format check only |
| Resolution (cached) | ~10-50 μs | LRU cache hit |
| Resolution (on-chain) | ~100-500 ms | RPC call + network |
| Registration (Ethereum) | ~2-5 seconds | Blockchain confirmation |
| Signature Verification | ~100-200 μs | Ed25519/ECDSA |

### Optimization Tips

1. **Use caching**: Enable resolver cache (default: enabled)
2. **Batch resolutions**: Resolve multiple DIDs in parallel
3. **Pre-fetch**: Load frequently used DIDs at startup
4. **Use V4 multi-key**: Avoid multiple agents per use case

## FAQ

### Q: When should I use V4 vs V2?

A: **Use V4 for all new deployments**:
- Multi-key support (Ethereum + Solana)
- Metadata updates without re-registration
- Enhanced security (CVE fixes)
- Nonce-based replay protection

Use V2 only if:
- Maintaining legacy integration
- Single-key simple use case
- Already deployed on Sepolia V2

### Q: How do I migrate from V2 to V4?

A: See [V4 Deployment Guide](../../../docs/V4_UPDATE_DEPLOYMENT_GUIDE.md) for step-by-step migration.

**Summary:**
1. Deploy SageRegistryV4 contract
2. Re-register agents with multiple keys
3. Update client code to use V4 manager
4. Deprecate V2 contract access

### Q: Can one owner have multiple agents?

A: **Yes**, use nonce-based DID generation:
```go
did1 := did.GenerateAgentDIDWithNonce(did.ChainEthereum, owner, 0)
did2 := did.GenerateAgentDIDWithNonce(did.ChainEthereum, owner, 1)
```

Each agent has a unique DID but verifiable to the same owner.

### Q: How do I support both Ethereum and Solana?

A: **Use V4 multi-key registration**:
```go
keys := []did.AgentKey{
    {Type: "ECDSA", KeyData: ethKey},    // Ethereum
    {Type: "Ed25519", KeyData: solanaKey}, // Solana
}
```

One agent, two blockchain identities.

### Q: What are the gas costs?

**Ethereum (Sepolia):**
- V2 Registration: ~500,000 gas (~$15-40 at 50 gwei)
- V4 Registration: ~653,000 gas (~$20-50 at 50 gwei)
- V4 Update: ~100,000 gas (~$3-8 at 50 gwei)
- Resolution: Free (view function)

**Mainnet:** Costs 10-100x higher depending on gas prices.

### Q: How do I test without spending ETH?

A: **Use local node or testnet**:
1. **Hardhat local node**: `npx hardhat node` (free, instant)
2. **Sepolia testnet**: Get free ETH from faucet
3. **Mock registry**: Use in-memory registry for unit tests

### Q: What is A2A integration?

A: **Google A2A (Agent-to-Agent)** is a protocol for AI agent interoperability. SAGE provides:
- `GenerateA2ACard()` - Export SAGE agent as A2A card
- `ValidateA2ACard()` - Validate incoming A2A cards
- `MergeA2ACard()` - Import A2A capabilities

See [A2A Integration Guide](../../../docs/SAGE_A2A_INTEGRATION_GUIDE.md)

## See Also

- [Crypto Package](../crypto/README.md) - Key generation and signing
- [Session Management](../session/README.md) - Secure agent sessions
- [Handshake Protocol](../handshake/) - Session establishment
- [V4 Deployment Guide](../../../docs/V4_UPDATE_DEPLOYMENT_GUIDE.md) - V4 migration
- [A2A Integration Guide](../../../docs/SAGE_A2A_INTEGRATION_GUIDE.md) - A2A protocol

## References

- [W3C DID Core 1.0](https://www.w3.org/TR/did-core/) - DID specification
- [EIP-1056](https://eips.ethereum.org/EIPS/eip-1056) - Ethereum DID
- [Google A2A Protocol](https://github.com/a2aproject/a2a) - Agent interoperability
- [SageRegistryV4 Contract](../../../contracts/ethereum/SageRegistryV4.sol) - Smart contract
- [Sepolia Etherscan](https://sepolia.etherscan.io/) - Testnet explorer

## License

LGPL-3.0 - See LICENSE file for details.
