# SAGE-A2A Integration Guide

**Target Project**: `sage-a2a-go`
**Purpose**: Integrate SAGE DID system with A2A (Agent-to-Agent) protocol
**SAGE Version**: v4 (Multi-key Registry)
**Date**: 2025-01-18

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Separation](#architecture-separation)
3. [SAGE Core APIs Available](#sage-core-apis-available)
4. [Task 2.3: DID-RFC9421 Integration](#task-23-did-rfc9421-integration)
5. [Implementation Steps](#implementation-steps)
6. [Code Examples](#code-examples)
7. [Testing Strategy](#testing-strategy)
8. [References](#references)

---

## Overview

### What SAGE Provides

SAGE is a **DID Registry and Cryptographic Infrastructure** that provides:

- ✅ Decentralized Identity (DID) registration on blockchain
- ✅ Multi-key support (ECDSA for Ethereum, Ed25519 for Solana)
- ✅ Public key resolution by DID and key type
- ✅ Cryptographic operations (sign/verify)
- ✅ Owner address validation in DID format

### What sage-a2a-go Will Provide

sage-a2a-go is an **A2A Protocol Integration Layer** that will provide:

- ⏸️ RFC9421 HTTP Message Signatures for A2A
- ⏸️ DIDVerifier (SAGE DID → RFC9421 integration)
- ⏸️ Agent-to-Agent communication routing
- ⏸️ A2A protocol implementation
- ⏸️ Multi-key selection for different chains/protocols

### Integration Point

```
┌─────────────────────────────────────────┐
│         sage-a2a-go Project             │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   A2A Protocol Layer              │ │
│  │   - HTTP Message Signatures       │ │
│  │   - Agent Communication           │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
│  ┌───────────────▼───────────────────┐ │
│  │   DIDVerifier (NEW)               │ │
│  │   - RFC9421 integration           │ │
│  │   - Multi-key selection           │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
└──────────────────┼──────────────────────┘
                   │
                   │ (uses)
                   │
┌──────────────────▼──────────────────────┐
│         SAGE Core (this project)        │
│                                         │
│  ✅ DID Registry (SageRegistryV4)      │
│  ✅ Multi-key Resolution               │
│  ✅ Crypto Primitives                  │
│  ✅ Blockchain Integration             │
└─────────────────────────────────────────┘
```

---

## Architecture Separation

### SAGE Core Responsibilities

| Component | Description | Status |
|-----------|-------------|--------|
| DID Registry | Smart contract for DID registration | ✅ Complete |
| Multi-key Support | ECDSA + Ed25519 key management | ✅ Complete |
| Public Key Resolution | `ResolveAllPublicKeys()`, `ResolvePublicKeyByType()` | ✅ Complete |
| Owner Validation | DID format with owner address | ✅ Complete |
| Crypto Operations | Sign/Verify with different key types | ✅ Complete |
| Blockchain Client | Ethereum V4 client | ✅ Complete |

### sage-a2a-go Responsibilities

| Component | Description | Status |
|-----------|-------------|--------|
| DIDVerifier | RFC9421 integration for SAGE DIDs | ⏸️ To Implement |
| Multi-key Selection | Choose appropriate key for chain/protocol | ⏸️ To Implement |
| A2A Protocol | Agent-to-Agent communication | ⏸️ To Implement |
| HTTP Message Signing | RFC9421 implementation | ⏸️ To Implement |
| Agent Routing | Message routing between agents | ⏸️ To Implement |

---

## SAGE Core APIs Available

### 1. DID Resolution

**Package**: `github.com/sage-x-project/sage/pkg/agent/did`

```go
// Manager interface
type Manager interface {
    RegisterAgent(ctx context.Context, chain Chain, req *RegistrationRequest) (*RegistrationResult, error)
    ResolveAgent(ctx context.Context, chain Chain, agentDID AgentDID) (*AgentMetadata, error)
}

// AgentMetadata returned from resolution
type AgentMetadata struct {
    DID          AgentDID
    Name         string
    Description  string
    Endpoint     string
    PublicKey    interface{} // First key (backward compatibility)
    Capabilities map[string]interface{}
    Owner        string
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 2. Multi-Key Resolution (NEW in V4)

**Package**: `github.com/sage-x-project/sage/pkg/agent/did/ethereum`

```go
// EthereumClientV4 interface
type EthereumClientV4 interface {
    // Resolve all verified public keys for an agent
    ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error)

    // Resolve public key by specific type
    // Returns: *ecdsa.PublicKey for ECDSA, ed25519.PublicKey for Ed25519
    ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error)
}

// AgentKey structure
type AgentKey struct {
    Type      KeyType
    KeyData   []byte
    Signature []byte
    Verified  bool
    CreatedAt time.Time
}

// KeyType enum
const (
    KeyTypeEd25519 KeyType = 0
    KeyTypeECDSA   KeyType = 1
)
```

### 3. DID Format (NEW in V4)

**Enhanced DID Format with Owner Validation**:

```
Format 1: did:sage:ethereum:0x{address}
Format 2: did:sage:ethereum:0x{address}:{nonce}
```

**Helper Functions** (in `cmd/sage-did/register.go` - can be moved to package):

```go
// Generate DID with owner address
func generateAgentDIDWithAddress(chain did.Chain, ownerAddress string) did.AgentDID

// Generate DID with owner address and nonce
func generateAgentDIDWithNonce(chain did.Chain, ownerAddress string, nonce uint64) did.AgentDID

// Derive Ethereum address from secp256k1 keypair
func deriveEthereumAddress(keyPair crypto.KeyPair) (string, error)
```

### 4. Crypto Operations

**Package**: `github.com/sage-x-project/sage/pkg/agent/crypto`

```go
// KeyPair interface
type KeyPair interface {
    PublicKey() crypto.PublicKey
    PrivateKey() crypto.PrivateKey
    Type() KeyType
    Sign(message []byte) ([]byte, error)
    Verify(message, signature []byte) error
}

// Key generation
func GenerateSecp256k1KeyPair() (KeyPair, error)
func GenerateEd25519KeyPair() (KeyPair, error)
```

### 5. DID Marshaling/Unmarshaling

**Package**: `github.com/sage-x-project/sage/pkg/agent/did`

```go
// Marshal public key to bytes
func MarshalPublicKey(pubKey interface{}) ([]byte, error)

// Unmarshal public key from bytes
// keyType: "secp256k1" or "ed25519"
func UnmarshalPublicKey(data []byte, keyType string) (interface{}, error)
```

---

## Task 2.3: DID-RFC9421 Integration

### Objective

Integrate SAGE's DID system with RFC9421 HTTP Message Signatures for Agent-to-Agent communication.

### Components to Implement

#### 1. DIDVerifier

**Purpose**: Verify RFC9421 HTTP signatures using SAGE DIDs

**Location**: `sage-a2a-go/pkg/verifier/did_verifier.go`

**Responsibilities**:
- Accept DID as key identifier in HTTP signature
- Resolve public key from SAGE DID registry
- Support multi-key selection (ECDSA vs Ed25519)
- Integrate with existing RFC9421 verifier

**Interface**:
```go
type DIDVerifier interface {
    // Verify HTTP message signature using DID
    VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error

    // Resolve public key for DID (with optional key type preference)
    ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error)
}
```

#### 2. Multi-Key Selector

**Purpose**: Select appropriate key based on chain/protocol

**Location**: `sage-a2a-go/pkg/verifier/key_selector.go`

**Logic**:
```
If protocol = "ethereum" → prefer ECDSA
If protocol = "solana" → prefer Ed25519
If keyType specified → use that type
Otherwise → use first verified key
```

**Interface**:
```go
type KeySelector interface {
    SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error)
}
```

#### 3. A2A Signer

**Purpose**: Sign HTTP messages for Agent-to-Agent communication

**Location**: `sage-a2a-go/pkg/signer/a2a_signer.go`

**Responsibilities**:
- Sign HTTP requests using agent's private key
- Include DID in signature parameters
- Support both ECDSA and Ed25519 signatures

**Interface**:
```go
type A2ASigner interface {
    // Sign HTTP request with agent's key
    SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error
}
```

---

## Implementation Steps

### Phase 1: Setup and Basic Integration (Week 1)

#### Step 1.1: Project Setup
```bash
# Create sage-a2a-go project
mkdir sage-a2a-go
cd sage-a2a-go
go mod init github.com/sage-x-project/sage-a2a-go

# Add SAGE as dependency
go get github.com/sage-x-project/sage
```

#### Step 1.2: Create DIDVerifier Skeleton
**File**: `pkg/verifier/did_verifier.go`

```go
package verifier

import (
    "context"
    "crypto"
    "fmt"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

type DIDVerifier struct {
    client ethereum.EthereumClientV4
    selector KeySelector
}

func NewDIDVerifier(client ethereum.EthereumClientV4) *DIDVerifier {
    return &DIDVerifier{
        client: client,
        selector: NewDefaultKeySelector(client),
    }
}

// Implement VerifyHTTPSignature
func (v *DIDVerifier) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
    // TODO: Extract signature from HTTP headers
    // TODO: Resolve public key from DID
    // TODO: Verify signature
    return fmt.Errorf("not implemented")
}

// Implement ResolvePublicKey
func (v *DIDVerifier) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error) {
    if keyType != nil {
        // Use specific key type
        return v.client.ResolvePublicKeyByType(ctx, agentDID, *keyType)
    }

    // Use first verified key
    keys, err := v.client.ResolveAllPublicKeys(ctx, agentDID)
    if err != nil {
        return nil, err
    }

    if len(keys) == 0 {
        return nil, fmt.Errorf("no verified keys found")
    }

    // Return first key (default behavior)
    keyTypeStr := "secp256k1"
    if keys[0].Type == did.KeyTypeEd25519 {
        keyTypeStr = "ed25519"
    }

    return did.UnmarshalPublicKey(keys[0].KeyData, keyTypeStr)
}
```

#### Step 1.3: Create KeySelector
**File**: `pkg/verifier/key_selector.go`

```go
package verifier

import (
    "context"
    "crypto"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

type KeySelector interface {
    SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error)
}

type DefaultKeySelector struct {
    client ethereum.EthereumClientV4
}

func NewDefaultKeySelector(client ethereum.EthereumClientV4) *DefaultKeySelector {
    return &DefaultKeySelector{client: client}
}

func (s *DefaultKeySelector) SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error) {
    // Determine preferred key type based on protocol
    var preferredKeyType did.KeyType

    switch protocol {
    case "ethereum":
        preferredKeyType = did.KeyTypeECDSA
    case "solana":
        preferredKeyType = did.KeyTypeEd25519
    default:
        // No preference, use first verified key
        return s.selectFirstKey(ctx, agentDID)
    }

    // Try to get preferred key type
    pubKey, err := s.client.ResolvePublicKeyByType(ctx, agentDID, preferredKeyType)
    if err == nil {
        return pubKey.(crypto.PublicKey), preferredKeyType, nil
    }

    // Fallback to first available key
    return s.selectFirstKey(ctx, agentDID)
}

func (s *DefaultKeySelector) selectFirstKey(ctx context.Context, agentDID did.AgentDID) (crypto.PublicKey, did.KeyType, error) {
    keys, err := s.client.ResolveAllPublicKeys(ctx, agentDID)
    if err != nil {
        return nil, 0, err
    }

    if len(keys) == 0 {
        return nil, 0, fmt.Errorf("no verified keys found")
    }

    firstKey := keys[0]
    keyTypeStr := "secp256k1"
    if firstKey.Type == did.KeyTypeEd25519 {
        keyTypeStr = "ed25519"
    }

    pubKey, err := did.UnmarshalPublicKey(firstKey.KeyData, keyTypeStr)
    if err != nil {
        return nil, 0, err
    }

    return pubKey.(crypto.PublicKey), firstKey.Type, nil
}
```

### Phase 2: RFC9421 Integration (Week 2)

#### Step 2.1: Integrate with Existing RFC9421 Verifier

SAGE already has RFC9421 verifier at:
- `pkg/agent/core/rfc9421/verifier.go`
- `pkg/agent/core/rfc9421/verifier_http.go`

**Option A**: Extend existing verifier to support DID

**File**: `sage-a2a-go/pkg/verifier/rfc9421_did_adapter.go`

```go
package verifier

import (
    "context"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// RFC9421DIDAdapter adapts SAGE's RFC9421 verifier to work with DIDs
type RFC9421DIDAdapter struct {
    verifier     *rfc9421.Verifier
    didVerifier  *DIDVerifier
}

func NewRFC9421DIDAdapter(didVerifier *DIDVerifier) *RFC9421DIDAdapter {
    return &RFC9421DIDAdapter{
        verifier:    rfc9421.NewVerifier(),
        didVerifier: didVerifier,
    }
}

func (a *RFC9421DIDAdapter) VerifyHTTPMessage(ctx context.Context, req *http.Request) error {
    // Step 1: Extract key-id (DID) from Signature-Input header
    // Format: sig1=("@method" "@target-uri");keyid="did:sage:ethereum:0x..."
    keyID, err := a.extractKeyID(req)
    if err != nil {
        return err
    }

    // Step 2: Parse DID
    agentDID := did.AgentDID(keyID)

    // Step 3: Resolve public key
    pubKey, err := a.didVerifier.ResolvePublicKey(ctx, agentDID, nil)
    if err != nil {
        return err
    }

    // Step 4: Verify using RFC9421 verifier with resolved key
    return a.verifier.VerifyHTTPRequest(req, pubKey)
}

func (a *RFC9421DIDAdapter) extractKeyID(req *http.Request) (string, error) {
    // Parse Signature-Input header to extract keyid
    // Implementation details...
    return "", nil
}
```

#### Step 2.2: Implement HTTP Message Signing

**File**: `sage-a2a-go/pkg/signer/a2a_signer.go`

```go
package signer

import (
    "context"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

type A2ASigner struct {
    signer *rfc9421.Signer
}

func NewA2ASigner() *A2ASigner {
    return &A2ASigner{
        signer: rfc9421.NewSigner(),
    }
}

func (s *A2ASigner) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error {
    // Step 1: Create signature parameters with DID as keyid
    params := &rfc9421.SignatureParams{
        KeyID:     string(agentDID), // Use DID as key identifier
        Algorithm: s.getAlgorithm(keyPair.Type()),
        Created:   time.Now().Unix(),
    }

    // Step 2: Sign request
    return s.signer.SignHTTPRequest(req, keyPair, params)
}

func (s *A2ASigner) getAlgorithm(keyType crypto.KeyType) string {
    switch keyType {
    case crypto.KeyTypeSecp256k1:
        return "ecdsa-p256-sha256"
    case crypto.KeyTypeEd25519:
        return "ed25519"
    default:
        return ""
    }
}
```

### Phase 3: Testing (Week 3)

#### Step 3.1: Integration Test

**File**: `sage-a2a-go/pkg/verifier/integration_test.go`

```go
package verifier_test

import (
    "context"
    "net/http"
    "testing"

    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

func TestDIDVerifierIntegration(t *testing.T) {
    // Setup: Create Ethereum client (connects to local testnet)
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    }

    client, err := ethereum.NewEthereumClientV4(config)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    // Generate keypair
    keyPair, err := crypto.GenerateSecp256k1KeyPair()
    if err != nil {
        t.Fatalf("Failed to generate keypair: %v", err)
    }

    // Register agent
    testDID := did.AgentDID("did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
    // ... registration code ...

    // Create signer and verifier
    a2aSigner := signer.NewA2ASigner()
    didVerifier := verifier.NewDIDVerifier(client)

    // Create HTTP request
    req, err := http.NewRequest("POST", "https://agent.example.com/message", nil)
    if err != nil {
        t.Fatalf("Failed to create request: %v", err)
    }

    // Sign request
    ctx := context.Background()
    err = a2aSigner.SignRequest(ctx, req, testDID, keyPair)
    if err != nil {
        t.Fatalf("Failed to sign request: %v", err)
    }

    // Verify signature
    err = didVerifier.VerifyHTTPSignature(ctx, req, testDID)
    if err != nil {
        t.Fatalf("Failed to verify signature: %v", err)
    }

    t.Log("DID-based HTTP signature verification successful!")
}
```

#### Step 3.2: Multi-Key Selection Test

```go
func TestMultiKeySelection(t *testing.T) {
    // Register agent with both ECDSA and Ed25519 keys
    // Test that KeySelector chooses correct key based on protocol

    tests := []struct {
        name         string
        protocol     string
        expectedType did.KeyType
    }{
        {
            name:         "Ethereum prefers ECDSA",
            protocol:     "ethereum",
            expectedType: did.KeyTypeECDSA,
        },
        {
            name:         "Solana prefers Ed25519",
            protocol:     "solana",
            expectedType: did.KeyTypeEd25519,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test key selection logic
            // ...
        })
    }
}
```

### Phase 4: A2A Protocol Integration (Week 4)

This phase involves integrating with the actual A2A protocol specification.

**Reference**: [A2A Protocol Specification](https://github.com/a2aproject/a2a)

**Implementation files**:
- `sage-a2a-go/pkg/protocol/a2a_client.go` - A2A client implementation
- `sage-a2a-go/pkg/protocol/a2a_server.go` - A2A server implementation
- `sage-a2a-go/pkg/protocol/message_router.go` - Message routing

---

## Code Examples

### Example 1: Agent Registration with Multi-Key

```go
package main

import (
    "context"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func main() {
    // Setup
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "your-private-key",
    }

    client, _ := ethereum.NewEthereumClientV4(config)

    // Generate keys
    ecdsaKey, _ := crypto.GenerateSecp256k1KeyPair()
    ed25519Key, _ := crypto.GenerateEd25519KeyPair()

    // Marshal public keys
    ecdsaPubKey, _ := did.MarshalPublicKey(ecdsaKey.PublicKey())
    ed25519PubKey, _ := did.MarshalPublicKey(ed25519Key.PublicKey())

    // Create DID
    testDID := did.AgentDID("did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")

    // Register with multiple keys
    req := &did.RegistrationRequest{
        DID:         testDID,
        Name:        "Multi-Chain Agent",
        Description: "Agent with ECDSA and Ed25519 keys",
        Endpoint:    "https://agent.example.com",
        Keys: []did.AgentKey{
            {Type: did.KeyTypeECDSA, KeyData: ecdsaPubKey},
            {Type: did.KeyTypeEd25519, KeyData: ed25519PubKey},
        },
    }

    ctx := context.Background()
    result, _ := client.Register(ctx, req)

    fmt.Printf("Agent registered: %s\n", result.TransactionHash)
}
```

### Example 2: Resolving Multi-Key Agent

```go
func resolveAgent(client ethereum.EthereumClientV4, agentDID did.AgentDID) {
    ctx := context.Background()

    // Get all verified keys
    keys, err := client.ResolveAllPublicKeys(ctx, agentDID)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Agent has %d verified keys:\n", len(keys))
    for i, key := range keys {
        fmt.Printf("  Key %d: Type=%s, Verified=%v\n",
            i, key.Type, key.Verified)
    }

    // Get ECDSA key specifically (for Ethereum operations)
    ecdsaKey, err := client.ResolvePublicKeyByType(ctx, agentDID, did.KeyTypeECDSA)
    if err != nil {
        panic(err)
    }
    fmt.Printf("ECDSA key: %T\n", ecdsaKey)

    // Get Ed25519 key specifically (for Solana operations)
    ed25519Key, err := client.ResolvePublicKeyByType(ctx, agentDID, did.KeyTypeEd25519)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Ed25519 key: %T\n", ed25519Key)
}
```

### Example 3: Using DIDVerifier (sage-a2a-go)

```go
package main

import (
    "context"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

func handleAgentRequest(w http.ResponseWriter, r *http.Request) {
    // Extract DID from request (e.g., from header or signature)
    agentDID := did.AgentDID(r.Header.Get("X-Agent-DID"))

    // Setup DID verifier
    client, _ := ethereum.NewEthereumClientV4(config)
    didVerifier := verifier.NewDIDVerifier(client)

    // Verify HTTP signature
    ctx := context.Background()
    err := didVerifier.VerifyHTTPSignature(ctx, r, agentDID)
    if err != nil {
        http.Error(w, "Signature verification failed", http.StatusUnauthorized)
        return
    }

    // Signature verified, process request
    w.Write([]byte("Request verified successfully"))
}
```

---

## Testing Strategy

### Unit Tests

**Location**: `sage-a2a-go/pkg/verifier/*_test.go`

1. **DIDVerifier Tests**
   - Test public key resolution
   - Test signature verification with ECDSA
   - Test signature verification with Ed25519
   - Test error handling (invalid DID, no keys, etc.)

2. **KeySelector Tests**
   - Test protocol-based key selection
   - Test fallback to first key
   - Test when preferred key type not available

3. **A2ASigner Tests**
   - Test request signing with ECDSA
   - Test request signing with Ed25519
   - Test signature format compliance with RFC9421

### Integration Tests

**Location**: `sage-a2a-go/test/integration/*_test.go`

**Requirements**:
- Local Ethereum testnet (Hardhat/Anvil)
- Deployed SageRegistryV4 contract

**Test Scenarios**:

1. **End-to-End Agent Communication**
   ```
   Agent A (ECDSA) → Signs message → Agent B verifies using DID
   ```

2. **Multi-Key Scenario**
   ```
   Agent with ECDSA + Ed25519
   → Ethereum operation uses ECDSA
   → Solana operation uses Ed25519
   ```

3. **Cross-Protocol Communication**
   ```
   Ethereum Agent (ECDSA) ↔ Solana Agent (Ed25519)
   → Both verify each other's signatures
   ```

### Performance Tests

**Benchmarks**:
- DID resolution time
- Signature verification time
- Multi-key selection overhead

**Target Metrics**:
- DID resolution: < 100ms
- Signature verification: < 50ms
- Key selection: < 10ms

---

## References

### SAGE Documentation

- [SAGE Architecture](../ARCHITECTURE.md)
- [DID Registry V4 Contract](../contracts/ethereum/contracts/SageRegistryV4.sol)
- [Multi-Key Resolution Tests](../pkg/agent/did/ethereum/clientv4_multikey_resolution_test.go)
- [RFC9421 Verifier](../pkg/agent/core/rfc9421/verifier.go)

### External Specifications

- [RFC9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [A2A Protocol](https://github.com/a2aproject/a2a)
- [DID Core Specification](https://www.w3.org/TR/did-core/)
- [Multikey Specification](https://www.w3.org/TR/controller-document/#multikey)

### SAGE API Examples

**DID Registration**:
```bash
# Using sage-did CLI
sage-did register \
  --chain ethereum \
  --name "My Agent" \
  --endpoint "https://agent.example.com" \
  --key ./key.jwk
```

**Multi-Key Registration**:
```bash
sage-did register \
  --chain ethereum \
  --name "Multi-Key Agent" \
  --endpoint "https://agent.example.com" \
  --key ./ecdsa-key.jwk \
  --additional-keys ./ed25519-key.jwk \
  --key-types ecdsa,ed25519
```

---

## Migration Path

### From SAGE v3 to v4

If you have existing code using SAGE v3:

**Old (v3)**:
```go
// Single key only
agent, _ := manager.ResolveAgent(ctx, chain, agentDID)
pubKey := agent.PublicKey // Only one key
```

**New (v4)**:
```go
// Multi-key support
client := ethereum.NewEthereumClientV4(config)

// Get all keys
keys, _ := client.ResolveAllPublicKeys(ctx, agentDID)

// Or get specific key type
ecdsaKey, _ := client.ResolvePublicKeyByType(ctx, agentDID, did.KeyTypeECDSA)
```

---

## FAQ

### Q1: Can I use SAGE DIDs without blockchain?

**A**: No, SAGE DIDs are anchored on blockchain (Ethereum V4 contract). However, once registered, the DID can be resolved off-chain.

### Q2: Which key type should I use for A2A?

**A**:
- For Ethereum-based agents: ECDSA (secp256k1)
- For Solana-based agents: Ed25519
- For multi-chain agents: Register both and use KeySelector

### Q3: How do I handle key rotation?

**A**: SAGE V4 supports atomic key rotation:
```go
client.RotateKey(ctx, agentID, oldKeyHash, newKeyType, newKeyData, newSignature)
```

### Q4: Can agents have multiple DIDs?

**A**: Yes, use the nonce format:
```
did:sage:ethereum:0x{address}:0
did:sage:ethereum:0x{address}:1
...
```

### Q5: Is Ed25519 verification automatic?

**A**: No, Ed25519 keys require contract owner approval on Ethereum due to lack of native precompile:
```go
client.ApproveEd25519Key(ctx, keyHash)
```

---

## Next Steps

1. **Create sage-a2a-go repository**
   ```bash
   gh repo create sage-x-project/sage-a2a-go --public
   ```

2. **Implement Phase 1** (Setup & DIDVerifier skeleton)

3. **Test with local testnet**
   ```bash
   # Terminal 1: Start local Ethereum testnet
   npx hardhat node

   # Terminal 2: Deploy contract
   npx hardhat run scripts/deploy.js --network localhost

   # Terminal 3: Run integration tests
   go test ./test/integration/...
   ```

4. **Iterate and integrate with A2A protocol**

---

**Document Version**: 1.0
**Last Updated**: 2025-01-18
**Author**: SAGE Development Team
**Contact**: https://github.com/sage-x-project/sage
