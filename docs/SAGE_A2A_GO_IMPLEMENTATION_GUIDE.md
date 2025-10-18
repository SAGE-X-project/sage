# sage-a2a-go Implementation Guide

**Target Project**: `sage-a2a-go` (new repository)
**Purpose**: A2A (Agent-to-Agent) Protocol Integration Layer for SAGE
**SAGE Version Required**: v1.1.0+ (with DID Helper Functions)
**Created**: 2025-01-19
**For**: Implementation in separate session

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Prerequisites & Dependencies](#prerequisites--dependencies)
3. [Project Structure](#project-structure)
4. [Implementation Tasks](#implementation-tasks)
5. [Component Details](#component-details)
6. [Testing Strategy](#testing-strategy)
7. [Examples to Implement](#examples-to-implement)
8. [Implementation Order](#implementation-order)
9. [Reference Materials](#reference-materials)

---

## Project Overview

### What is sage-a2a-go?

sage-a2a-go is an **A2A Protocol Integration Layer** that enables AI agents using SAGE DIDs to communicate using the Google A2A (Agent-to-Agent) protocol with RFC9421 HTTP Message Signatures.

### Architecture

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
│  │   DIDVerifier (TO IMPLEMENT)      │ │
│  │   - RFC9421 integration           │ │
│  │   - Multi-key selection           │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
└──────────────────┼──────────────────────┘
                   │
                   │ (uses)
                   │
┌──────────────────▼──────────────────────┐
│         SAGE Core (COMPLETED)           │
│                                         │
│  ✅ DID Registry (SageRegistryV4)      │
│  ✅ Multi-key Resolution               │
│  ✅ DID Helper Functions               │
│  ✅ Crypto Primitives                  │
└─────────────────────────────────────────┘
```

### Key Responsibilities

sage-a2a-go is responsible for:
- ✅ RFC9421 HTTP Message Signatures with DID integration
- ✅ DIDVerifier implementation
- ✅ Multi-key selection based on protocol/chain
- ✅ A2A protocol message routing
- ✅ Integration examples and documentation

---

## Prerequisites & Dependencies

### Required SAGE APIs (Already Available)

From `github.com/sage-x-project/sage`:

#### 1. DID Helper Functions
```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// DID Generation
did.GenerateAgentDIDWithAddress(chain Chain, ownerAddress string) AgentDID
did.GenerateAgentDIDWithNonce(chain Chain, ownerAddress string, nonce uint64) AgentDID
did.DeriveEthereumAddress(keyPair crypto.KeyPair) (string, error)

// Public Key Marshaling
did.MarshalPublicKey(pubKey interface{}) ([]byte, error)
did.UnmarshalPublicKey(data []byte, keyType string) (interface{}, error)

// A2A Agent Cards
did.GenerateA2ACard(metadata *AgentMetadataV4) (*A2AAgentCard, error)
did.ValidateA2ACard(card *A2AAgentCard) error
did.MergeA2ACard(metadata *AgentMetadataV4, card *A2AAgentCard) error
```

#### 2. Multi-Key Resolution
```go
import "github.com/sage-x-project/sage/pkg/agent/did/ethereum"

// Resolve all verified public keys
client.ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error)

// Resolve public key by specific type
client.ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error)
```

#### 3. RFC9421 Verifier (SAGE Core)
```go
import "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"

// Existing RFC9421 verifier
verifier := rfc9421.NewVerifier()
verifier.VerifyHTTPRequest(req *http.Request, pubKey crypto.PublicKey) error
```

#### 4. Crypto Operations
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

crypto.GenerateSecp256k1KeyPair() (KeyPair, error)
crypto.GenerateEd25519KeyPair() (KeyPair, error)

keyPair.Sign(message []byte) ([]byte, error)
keyPair.Verify(message, signature []byte) error
```

### External Dependencies

Add to `go.mod`:
```go
module github.com/sage-x-project/sage-a2a-go

go 1.23

require (
    github.com/sage-x-project/sage v1.1.0  // SAGE Core APIs
    github.com/stretchr/testify v1.8.4      // Testing
    // Add other dependencies as needed
)
```

---

## Project Structure

### Recommended Directory Layout

```
sage-a2a-go/
├── go.mod
├── go.sum
├── README.md
├── LICENSE (MIT)
│
├── pkg/
│   ├── verifier/                    # DID-based signature verification
│   │   ├── did_verifier.go          # Main DIDVerifier implementation
│   │   ├── did_verifier_test.go     # Unit tests
│   │   ├── key_selector.go          # Multi-key selection logic
│   │   ├── key_selector_test.go     # Key selector tests
│   │   └── rfc9421_adapter.go       # RFC9421-DID adapter
│   │
│   ├── signer/                      # HTTP message signing
│   │   ├── a2a_signer.go            # A2A message signer
│   │   ├── a2a_signer_test.go       # Signer tests
│   │   └── types.go                 # Signature types
│   │
│   ├── protocol/                    # A2A protocol implementation (optional)
│   │   ├── a2a_client.go            # A2A client
│   │   ├── a2a_server.go            # A2A server
│   │   └── message_router.go        # Message routing
│   │
│   └── types/                       # Shared types
│       └── types.go                 # Common types and interfaces
│
├── examples/                        # Integration examples
│   ├── basic-did-resolver/          # Example 1: DID resolution
│   │   └── main.go
│   ├── multi-key-selection/         # Example 2: Protocol-based key selection
│   │   └── main.go
│   ├── rfc9421-integration/         # Example 3: RFC9421 + DID
│   │   ├── server.go
│   │   └── client.go
│   └── a2a-card-generator/          # Example 4: A2A Agent Card
│       └── main.go
│
├── test/                            # Integration tests
│   └── integration/
│       ├── did_verifier_test.go
│       └── end_to_end_test.go
│
└── docs/
    ├── QUICKSTART.md                # Quick start guide
    └── API.md                       # API documentation
```

---

## Implementation Tasks

### Task 1: Project Setup

**Priority**: High
**Estimated Time**: 30 minutes

#### 1.1 Create Repository
```bash
mkdir sage-a2a-go
cd sage-a2a-go
go mod init github.com/sage-x-project/sage-a2a-go
```

#### 1.2 Add SAGE Dependency
```bash
go get github.com/sage-x-project/sage@latest
```

#### 1.3 Create Directory Structure
```bash
mkdir -p pkg/verifier pkg/signer pkg/protocol pkg/types
mkdir -p examples/basic-did-resolver examples/multi-key-selection
mkdir -p examples/rfc9421-integration examples/a2a-card-generator
mkdir -p test/integration docs
```

#### 1.4 Create LICENSE
Use MIT License (compatible with SAGE's LGPL-v3):
```
MIT License

Copyright (c) 2025 SAGE-X-project

Permission is hereby granted, free of charge, to any person obtaining a copy...
```

---

### Task 2: Implement DIDVerifier

**Priority**: High
**Estimated Time**: 2-3 hours

#### 2.1 Create `pkg/verifier/did_verifier.go`

**Purpose**: Verify RFC9421 HTTP signatures using SAGE DIDs

**Interface to Implement**:
```go
package verifier

import (
    "context"
    "crypto"
    "fmt"
    "net/http"
    "strings"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
)

// DIDVerifier verifies HTTP message signatures using SAGE DIDs
type DIDVerifier struct {
    client   ethereum.EthereumClientV4
    selector KeySelector
    verifier *rfc9421.Verifier
}

// NewDIDVerifier creates a new DID-based signature verifier
func NewDIDVerifier(client ethereum.EthereumClientV4) *DIDVerifier {
    return &DIDVerifier{
        client:   client,
        selector: NewDefaultKeySelector(client),
        verifier: rfc9421.NewVerifier(),
    }
}

// VerifyHTTPSignature verifies an HTTP request signature using DID
//
// Steps:
// 1. Extract keyid (DID) from Signature-Input header
// 2. Resolve public key from SAGE DID registry
// 3. Verify signature using SAGE's RFC9421 verifier
func (v *DIDVerifier) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
    // Step 1: Extract signature parameters from headers
    // Signature-Input: sig1=("@method" "@target-uri");keyid="did:sage:ethereum:0x..."
    keyID, err := v.extractKeyID(req)
    if err != nil {
        return fmt.Errorf("failed to extract keyid: %w", err)
    }

    // Validate that extracted keyID matches expected agentDID
    if keyID != string(agentDID) {
        return fmt.Errorf("keyid mismatch: expected %s, got %s", agentDID, keyID)
    }

    // Step 2: Resolve public key from DID
    pubKey, err := v.ResolvePublicKey(ctx, agentDID, nil)
    if err != nil {
        return fmt.Errorf("failed to resolve public key for DID %s: %w", agentDID, err)
    }

    // Step 3: Verify signature using SAGE's RFC9421 verifier
    err = v.verifier.VerifyHTTPRequest(req, pubKey)
    if err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    return nil
}

// ResolvePublicKey resolves the public key for a DID
//
// If keyType is specified, uses that specific key type.
// Otherwise, uses the first verified key.
func (v *DIDVerifier) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error) {
    if keyType != nil {
        // Use specific key type
        return v.client.ResolvePublicKeyByType(ctx, agentDID, *keyType)
    }

    // Use first verified key
    keys, err := v.client.ResolveAllPublicKeys(ctx, agentDID)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve keys: %w", err)
    }

    if len(keys) == 0 {
        return nil, fmt.Errorf("no verified keys found for DID %s", agentDID)
    }

    // Return first key (default behavior)
    keyTypeStr := "secp256k1"
    if keys[0].Type == did.KeyTypeEd25519 {
        keyTypeStr = "ed25519"
    }

    return did.UnmarshalPublicKey(keys[0].KeyData, keyTypeStr)
}

// extractKeyID extracts the keyid parameter from the Signature-Input header
//
// Example header:
// Signature-Input: sig1=("@method" "@target-uri");keyid="did:sage:ethereum:0x123..."
func (v *DIDVerifier) extractKeyID(req *http.Request) (string, error) {
    sigInput := req.Header.Get("Signature-Input")
    if sigInput == "" {
        return "", fmt.Errorf("Signature-Input header not found")
    }

    // Parse keyid from header
    // Format: sig1=(...);keyid="value"
    parts := strings.Split(sigInput, ";keyid=")
    if len(parts) != 2 {
        return "", fmt.Errorf("invalid Signature-Input format: missing keyid")
    }

    keyid := strings.Trim(parts[1], "\"")
    return keyid, nil
}
```

#### 2.2 Implementation Notes

**Key Design Decisions**:

1. **Why separate KeySelector?**
   - Different protocols prefer different key types (Ethereum → ECDSA, Solana → Ed25519)
   - Extensible for future protocols

2. **Why use SAGE's RFC9421 Verifier?**
   - Already tested and compliant
   - No need to reimplement RFC9421 spec
   - sage-a2a-go focuses on DID integration layer

3. **Error Handling**:
   - Always wrap errors with context
   - Return specific error messages for debugging

---

### Task 3: Implement KeySelector

**Priority**: High
**Estimated Time**: 1-2 hours

#### 3.1 Create `pkg/verifier/key_selector.go`

**Purpose**: Select appropriate key based on protocol/chain

```go
package verifier

import (
    "context"
    "crypto"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// KeySelector selects the appropriate key for a given protocol
type KeySelector interface {
    SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error)
}

// DefaultKeySelector implements protocol-based key selection
type DefaultKeySelector struct {
    client ethereum.EthereumClientV4
}

// NewDefaultKeySelector creates a new default key selector
func NewDefaultKeySelector(client ethereum.EthereumClientV4) *DefaultKeySelector {
    return &DefaultKeySelector{
        client: client,
    }
}

// SelectKey selects the appropriate public key based on protocol
//
// Selection logic:
// - "ethereum" → prefer ECDSA (KeyTypeECDSA)
// - "solana"   → prefer Ed25519 (KeyTypeEd25519)
// - other      → use first verified key
//
// If preferred key type not available, falls back to first verified key.
func (s *DefaultKeySelector) SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error) {
    // Determine preferred key type based on protocol
    var preferredKeyType did.KeyType
    hasPreference := false

    switch protocol {
    case "ethereum":
        preferredKeyType = did.KeyTypeECDSA
        hasPreference = true
    case "solana":
        preferredKeyType = did.KeyTypeEd25519
        hasPreference = true
    default:
        // No preference, use first verified key
        return s.selectFirstKey(ctx, agentDID)
    }

    // Try to get preferred key type
    if hasPreference {
        pubKey, err := s.client.ResolvePublicKeyByType(ctx, agentDID, preferredKeyType)
        if err == nil {
            return pubKey.(crypto.PublicKey), preferredKeyType, nil
        }
        // Preferred key not found, fall back to first key
    }

    // Fallback to first available key
    return s.selectFirstKey(ctx, agentDID)
}

// selectFirstKey selects the first verified key from the agent's key list
func (s *DefaultKeySelector) selectFirstKey(ctx context.Context, agentDID did.AgentDID) (crypto.PublicKey, did.KeyType, error) {
    keys, err := s.client.ResolveAllPublicKeys(ctx, agentDID)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to resolve keys: %w", err)
    }

    if len(keys) == 0 {
        return nil, 0, fmt.Errorf("no verified keys found for DID %s", agentDID)
    }

    firstKey := keys[0]
    keyTypeStr := "secp256k1"
    if firstKey.Type == did.KeyTypeEd25519 {
        keyTypeStr = "ed25519"
    }

    pubKey, err := did.UnmarshalPublicKey(firstKey.KeyData, keyTypeStr)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to unmarshal public key: %w", err)
    }

    return pubKey.(crypto.PublicKey), firstKey.Type, nil
}
```

#### 3.2 Testing KeySelector

Create `pkg/verifier/key_selector_test.go`:

```go
package verifier_test

import (
    "context"
    "testing"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
    "github.com/stretchr/testify/assert"
)

func TestKeySelector_SelectKey(t *testing.T) {
    // Setup: Create mock Ethereum client or use testnet
    // This test requires a registered agent with multiple keys

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
        {
            name:         "Unknown protocol uses first key",
            protocol:     "unknown",
            expectedType: did.KeyTypeECDSA, // or Ed25519, depends on first key
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // TODO: Implement test with real or mock client
            // selector := verifier.NewDefaultKeySelector(client)
            // pubKey, keyType, err := selector.SelectKey(ctx, testDID, tt.protocol)
            // assert.NoError(t, err)
            // assert.Equal(t, tt.expectedType, keyType)
        })
    }
}
```

---

### Task 4: Implement A2ASigner

**Priority**: High
**Estimated Time**: 1-2 hours

#### 4.1 Create `pkg/signer/a2a_signer.go`

**Purpose**: Sign HTTP messages with agent's key and DID

```go
package signer

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// A2ASigner signs HTTP messages for Agent-to-Agent communication
type A2ASigner struct {
    signer *rfc9421.Signer
}

// NewA2ASigner creates a new A2A message signer
func NewA2ASigner() *A2ASigner {
    return &A2ASigner{
        signer: rfc9421.NewSigner(),
    }
}

// SignRequest signs an HTTP request with the agent's key
//
// The signature includes:
// - keyid: The agent's DID (for receiver to resolve public key)
// - algorithm: Based on key type (ecdsa-p256-sha256 or ed25519)
// - created: Timestamp for signature freshness
func (s *A2ASigner) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error {
    // Create signature parameters with DID as keyid
    params := &rfc9421.SignatureParams{
        KeyID:     string(agentDID), // Use DID as key identifier
        Algorithm: s.getAlgorithm(keyPair.Type()),
        Created:   time.Now().Unix(),
    }

    // Sign request using SAGE's RFC9421 signer
    err := s.signer.SignHTTPRequest(req, keyPair, params)
    if err != nil {
        return fmt.Errorf("failed to sign request: %w", err)
    }

    return nil
}

// getAlgorithm returns the RFC9421 algorithm identifier for a key type
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

#### 4.2 Testing A2ASigner

Create `pkg/signer/a2a_signer_test.go`:

```go
package signer_test

import (
    "context"
    "net/http"
    "testing"

    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestA2ASigner_SignRequest(t *testing.T) {
    // Generate a key pair
    keyPair, err := keys.GenerateSecp256k1KeyPair()
    require.NoError(t, err)

    // Create a test DID
    testDID := did.AgentDID("did:sage:ethereum:0xtest")

    // Create HTTP request
    req, err := http.NewRequest("POST", "https://agent.example.com/message", nil)
    require.NoError(t, err)

    // Sign request
    a2aSigner := signer.NewA2ASigner()
    ctx := context.Background()
    err = a2aSigner.SignRequest(ctx, req, testDID, keyPair)
    require.NoError(t, err)

    // Verify signature was added
    assert.NotEmpty(t, req.Header.Get("Signature"))
    assert.NotEmpty(t, req.Header.Get("Signature-Input"))

    // Verify keyid contains DID
    sigInput := req.Header.Get("Signature-Input")
    assert.Contains(t, sigInput, string(testDID))
}
```

---

### Task 5: Create Integration Examples

**Priority**: Medium
**Estimated Time**: 2-3 hours

#### Example 1: Basic DID Resolver

**File**: `examples/basic-did-resolver/main.go`

**Purpose**: Show how to resolve a DID and get its public keys

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func main() {
    fmt.Println("=== SAGE-A2A: Basic DID Resolver Example ===\n")

    // Step 1: Setup Ethereum client (connect to local testnet or Sepolia)
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9", // Local testnet
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    }

    client, err := ethereum.NewEthereumClientV4(config)
    if err != nil {
        log.Fatalf("Failed to create Ethereum client: %v", err)
    }

    // Step 2: Define test DID
    testDID := did.AgentDID("did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")

    fmt.Printf("Resolving DID: %s\n\n", testDID)

    // Step 3: Resolve all verified public keys
    ctx := context.Background()
    keys, err := client.ResolveAllPublicKeys(ctx, testDID)
    if err != nil {
        log.Fatalf("Failed to resolve keys: %v", err)
    }

    fmt.Printf("Found %d verified keys:\n", len(keys))
    for i, key := range keys {
        keyTypeName := "ECDSA"
        if key.Type == did.KeyTypeEd25519 {
            keyTypeName = "Ed25519"
        }
        fmt.Printf("  Key %d: Type=%s, Verified=%v\n", i+1, keyTypeName, key.Verified)
    }

    // Step 4: Resolve specific key types
    fmt.Println("\nResolving specific key types:")

    // Try to get ECDSA key
    ecdsaKey, err := client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeECDSA)
    if err == nil {
        fmt.Printf("✓ ECDSA key found: %T\n", ecdsaKey)
    } else {
        fmt.Printf("✗ ECDSA key not found: %v\n", err)
    }

    // Try to get Ed25519 key
    ed25519Key, err := client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeEd25519)
    if err == nil {
        fmt.Printf("✓ Ed25519 key found: %T\n", ed25519Key)
    } else {
        fmt.Printf("✗ Ed25519 key not found: %v\n", err)
    }

    fmt.Println("\n=== Example Complete ===")
}
```

**To Run**:
```bash
cd examples/basic-did-resolver
go run main.go
```

---

#### Example 2: Multi-Key Selection

**File**: `examples/multi-key-selection/main.go`

**Purpose**: Show protocol-based key selection

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

func main() {
    fmt.Println("=== SAGE-A2A: Multi-Key Selection Example ===\n")

    // Setup client
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    }

    client, err := ethereum.NewEthereumClientV4(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Create key selector
    selector := verifier.NewDefaultKeySelector(client)

    // Test DID with multiple keys
    testDID := did.AgentDID("did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")

    ctx := context.Background()

    // Test 1: Ethereum protocol (should prefer ECDSA)
    fmt.Println("Test 1: Ethereum protocol")
    pubKey, keyType, err := selector.SelectKey(ctx, testDID, "ethereum")
    if err != nil {
        log.Printf("  Error: %v\n", err)
    } else {
        fmt.Printf("  Selected key type: %v\n", keyType)
        fmt.Printf("  Public key type: %T\n", pubKey)
    }

    // Test 2: Solana protocol (should prefer Ed25519)
    fmt.Println("\nTest 2: Solana protocol")
    pubKey, keyType, err = selector.SelectKey(ctx, testDID, "solana")
    if err != nil {
        log.Printf("  Error: %v\n", err)
    } else {
        fmt.Printf("  Selected key type: %v\n", keyType)
        fmt.Printf("  Public key type: %T\n", pubKey)
    }

    // Test 3: Unknown protocol (should use first key)
    fmt.Println("\nTest 3: Unknown protocol")
    pubKey, keyType, err = selector.SelectKey(ctx, testDID, "custom-chain")
    if err != nil {
        log.Printf("  Error: %v\n", err)
    } else {
        fmt.Printf("  Selected key type: %v\n", keyType)
        fmt.Printf("  Public key type: %T\n", pubKey)
    }

    fmt.Println("\n=== Example Complete ===")
}
```

---

#### Example 3: RFC9421 Integration

**File**: `examples/rfc9421-integration/server.go`

**Purpose**: HTTP server that verifies DID-based signatures

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

var didVerifier *verifier.DIDVerifier

func main() {
    fmt.Println("=== SAGE-A2A: RFC9421 Integration Server ===")

    // Setup DID verifier
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    }

    client, err := ethereum.NewEthereumClientV4(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    didVerifier = verifier.NewDIDVerifier(client)

    // Setup HTTP server
    http.HandleFunc("/message", handleMessage)

    fmt.Println("Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
    fmt.Println("\n--- Incoming Request ---")

    // Extract DID from custom header (or from keyid in Signature-Input)
    agentDID := did.AgentDID(r.Header.Get("X-Agent-DID"))
    if agentDID == "" {
        http.Error(w, "Missing X-Agent-DID header", http.StatusBadRequest)
        return
    }

    fmt.Printf("Agent DID: %s\n", agentDID)

    // Verify HTTP signature
    ctx := context.Background()
    err := didVerifier.VerifyHTTPSignature(ctx, r, agentDID)
    if err != nil {
        fmt.Printf("Signature verification failed: %v\n", err)
        http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
        return
    }

    fmt.Println("✓ Signature verified successfully")

    // Process message
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Message received and verified"))
}
```

**File**: `examples/rfc9421-integration/client.go`

**Purpose**: HTTP client that signs requests with DID

```go
package main

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "log"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
)

func main() {
    fmt.Println("=== SAGE-A2A: RFC9421 Integration Client ===")

    // Step 1: Generate key pair
    keyPair, err := keys.GenerateSecp256k1KeyPair()
    if err != nil {
        log.Fatalf("Failed to generate keypair: %v", err)
    }

    // Step 2: Derive Ethereum address and create DID
    ownerAddr, err := did.DeriveEthereumAddress(keyPair)
    if err != nil {
        log.Fatalf("Failed to derive address: %v", err)
    }

    agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, ownerAddr)
    fmt.Printf("Agent DID: %s\n", agentDID)

    // Step 3: Create HTTP request
    body := []byte(`{"message": "Hello from A2A agent"}`)
    req, err := http.NewRequest("POST", "http://localhost:8080/message", bytes.NewReader(body))
    if err != nil {
        log.Fatalf("Failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Agent-DID", string(agentDID))

    // Step 4: Sign request
    a2aSigner := signer.NewA2ASigner()
    ctx := context.Background()
    err = a2aSigner.SignRequest(ctx, req, agentDID, keyPair)
    if err != nil {
        log.Fatalf("Failed to sign request: %v", err)
    }

    fmt.Println("✓ Request signed")

    // Step 5: Send request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    // Step 6: Read response
    respBody, _ := io.ReadAll(resp.Body)
    fmt.Printf("\nResponse Status: %s\n", resp.Status)
    fmt.Printf("Response Body: %s\n", string(respBody))
}
```

**To Run**:
```bash
# Terminal 1: Start server
cd examples/rfc9421-integration
go run server.go

# Terminal 2: Run client
go run client.go
```

---

#### Example 4: A2A Agent Card Generator

**File**: `examples/a2a-card-generator/main.go`

**Purpose**: Generate and validate A2A Agent Cards

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    fmt.Println("=== SAGE-A2A: Agent Card Generator Example ===\n")

    // Step 1: Create agent metadata
    metadata := &did.AgentMetadataV4{
        DID:         "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
        Name:        "Example A2A Agent",
        Description: "Multi-chain AI agent with ECDSA and Ed25519 keys",
        Endpoint:    "https://agent.example.com",
        Capabilities: map[string]interface{}{
            "capabilities": []string{
                "message-signing",
                "message-verification",
                "multi-chain-support",
            },
        },
        Keys: []did.AgentKey{
            {
                Type:      did.KeyTypeECDSA,
                KeyData:   []byte("mock-ecdsa-public-key-data"),
                Verified:  true,
                CreatedAt: time.Now(),
            },
            {
                Type:      did.KeyTypeEd25519,
                KeyData:   []byte("mock-ed25519-public-key-data"),
                Verified:  true,
                CreatedAt: time.Now(),
            },
        },
        Owner:     "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
        IsActive:  true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // Step 2: Generate A2A Agent Card
    card, err := did.GenerateA2ACard(metadata)
    if err != nil {
        log.Fatalf("Failed to generate A2A card: %v", err)
    }

    fmt.Println("Generated A2A Agent Card:")
    cardJSON, _ := json.MarshalIndent(card, "", "  ")
    fmt.Println(string(cardJSON))

    // Step 3: Validate the card
    err = did.ValidateA2ACard(card)
    if err != nil {
        log.Fatalf("Card validation failed: %v", err)
    }

    fmt.Println("\n✓ Card validation successful")

    // Step 4: Show key information
    fmt.Printf("\nAgent has %d verified keys:\n", len(card.PublicKeys))
    for i, key := range card.PublicKeys {
        fmt.Printf("  Key %d: %s (ID: %s)\n", i+1, key.Type, key.ID)
    }

    fmt.Println("\n=== Example Complete ===")
}
```

---

### Task 6: Write Integration Tests

**Priority**: High
**Estimated Time**: 2-3 hours

#### Integration Test Structure

**File**: `test/integration/did_verifier_test.go`

```go
// +build integration

package integration_test

import (
    "context"
    "net/http"
    "testing"

    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestDIDVerifier_EndToEnd tests the complete flow:
// 1. Generate keypair and register agent
// 2. Sign HTTP request with A2ASigner
// 3. Verify signature with DIDVerifier
func TestDIDVerifier_EndToEnd(t *testing.T) {
    // Setup: Create Ethereum client
    config := &did.RegistryConfig{
        ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
        RPCEndpoint:     "http://localhost:8545",
        PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    }

    client, err := ethereum.NewEthereumClientV4(config)
    require.NoError(t, err)

    // Step 1: Generate keypair
    keyPair, err := keys.GenerateSecp256k1KeyPair()
    require.NoError(t, err)

    // Step 2: Create DID
    ownerAddr, err := did.DeriveEthereumAddress(keyPair)
    require.NoError(t, err)

    testDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, ownerAddr)

    // Step 3: Register agent (assuming already registered in test setup)
    // In real test, you would call client.Register() here

    // Step 4: Create HTTP request
    req, err := http.NewRequest("POST", "https://agent.example.com/message", nil)
    require.NoError(t, err)

    // Step 5: Sign request
    a2aSigner := signer.NewA2ASigner()
    ctx := context.Background()
    err = a2aSigner.SignRequest(ctx, req, testDID, keyPair)
    require.NoError(t, err)

    // Step 6: Verify signature
    didVerifier := verifier.NewDIDVerifier(client)
    err = didVerifier.VerifyHTTPSignature(ctx, req, testDID)
    assert.NoError(t, err, "Signature verification should succeed")
}

func TestMultiKeySelection_Integration(t *testing.T) {
    // Test that agent with both ECDSA and Ed25519 keys
    // can be verified with either key type

    // TODO: Implement test
    // 1. Register agent with 2 keys (ECDSA + Ed25519)
    // 2. Sign message with ECDSA key
    // 3. Verify with protocol="ethereum" (should use ECDSA)
    // 4. Sign message with Ed25519 key
    // 5. Verify with protocol="solana" (should use Ed25519)
}
```

**To Run**:
```bash
# Start local testnet first
cd $SAGE_PROJECT
make blockchain-start

# Run integration tests
cd $SAGE_A2A_GO_PROJECT
go test -tags=integration ./test/integration/... -v
```

---

### Task 7: Write Documentation

**Priority**: Medium
**Estimated Time**: 1-2 hours

#### 7.1 Create `README.md`

```markdown
# sage-a2a-go

A2A (Agent-to-Agent) Protocol Integration for SAGE DID System

## Overview

sage-a2a-go enables AI agents using SAGE DIDs to communicate using the Google A2A protocol with RFC9421 HTTP Message Signatures.

## Features

- ✅ DID-based HTTP signature verification
- ✅ Multi-key selection based on protocol/chain
- ✅ RFC9421 compliant message signing
- ✅ A2A Agent Card support
- ✅ Integration examples

## Installation

```bash
go get github.com/sage-x-project/sage-a2a-go
```

## Quick Start

See [QUICKSTART.md](docs/QUICKSTART.md) for detailed instructions.

### Basic Usage

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

// Create DID verifier
client, _ := ethereum.NewEthereumClientV4(config)
didVerifier := verifier.NewDIDVerifier(client)

// Verify HTTP signature
err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
```

## Examples

See [examples/](examples/) directory:
- `basic-did-resolver/` - DID resolution
- `multi-key-selection/` - Protocol-based key selection
- `rfc9421-integration/` - Complete HTTP signing workflow
- `a2a-card-generator/` - A2A Agent Card generation

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [API Documentation](docs/API.md)
- [SAGE Integration Guide](https://github.com/sage-x-project/sage/blob/main/docs/SAGE_A2A_INTEGRATION_GUIDE.md)

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires local testnet)
go test -tags=integration ./test/integration/...
```

## License

MIT License - See [LICENSE](LICENSE)
```

#### 7.2 Create `docs/QUICKSTART.md`

```markdown
# Quick Start Guide

## Prerequisites

1. Go 1.23+
2. SAGE Core (v1.1.0+)
3. Local Ethereum testnet (for testing)

## Installation

```bash
go get github.com/sage-x-project/sage-a2a-go
```

## Step 1: Setup SAGE Client

```go
import "github.com/sage-x-project/sage/pkg/agent/did/ethereum"

config := &did.RegistryConfig{
    ContractAddress: "0xYourContractAddress",
    RPCEndpoint:     "http://localhost:8545",
    PrivateKey:      "your-private-key",
}

client, err := ethereum.NewEthereumClientV4(config)
```

## Step 2: Create DIDVerifier

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/verifier"

didVerifier := verifier.NewDIDVerifier(client)
```

## Step 3: Verify HTTP Signatures

```go
agentDID := did.AgentDID("did:sage:ethereum:0x...")
err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
if err != nil {
    // Signature verification failed
}
```

## Step 4: Sign HTTP Requests

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/signer"

a2aSigner := signer.NewA2ASigner()
err := a2aSigner.SignRequest(ctx, req, agentDID, keyPair)
```

## Complete Example

See [examples/rfc9421-integration/](../examples/rfc9421-integration/)

## Troubleshooting

### "DID not found"
- Ensure agent is registered on blockchain
- Check contract address and RPC endpoint

### "No verified keys found"
- Agent must have at least one verified public key
- Use `ResolveAllPublicKeys()` to check key status

### "Signature verification failed"
- Check that keyid in Signature-Input matches DID
- Verify signature was created with correct private key
```

---

## Implementation Order

### Phase 1: Core Components (Week 1)

**Day 1-2**:
- ✅ Task 1: Project setup
- ✅ Create directory structure
- ✅ Setup dependencies

**Day 3-4**:
- ✅ Task 2: Implement DIDVerifier
- ✅ Task 3: Implement KeySelector
- ✅ Write unit tests

**Day 5**:
- ✅ Task 4: Implement A2ASigner
- ✅ Write unit tests

### Phase 2: Examples & Testing (Week 2)

**Day 1-2**:
- ✅ Task 5: Create integration examples
- ✅ Example 1: Basic DID Resolver
- ✅ Example 2: Multi-Key Selection

**Day 3-4**:
- ✅ Example 3: RFC9421 Integration (client + server)
- ✅ Example 4: A2A Agent Card Generator

**Day 5**:
- ✅ Task 6: Write integration tests
- ✅ End-to-end testing

### Phase 3: Documentation (Week 2-3)

**Day 1-2**:
- ✅ Task 7: Write documentation
- ✅ README.md
- ✅ QUICKSTART.md
- ✅ API.md

---

## Testing Strategy

### Unit Tests

**Coverage Target**: 80%+

**Test Files**:
- `pkg/verifier/did_verifier_test.go`
- `pkg/verifier/key_selector_test.go`
- `pkg/signer/a2a_signer_test.go`

**What to Test**:
- ✅ DIDVerifier.VerifyHTTPSignature()
  - Valid signature → success
  - Invalid signature → error
  - Missing headers → error
  - DID mismatch → error
- ✅ KeySelector.SelectKey()
  - Protocol "ethereum" → ECDSA
  - Protocol "solana" → Ed25519
  - Unknown protocol → first key
  - No keys available → error
- ✅ A2ASigner.SignRequest()
  - ECDSA key → ecdsa-p256-sha256 algorithm
  - Ed25519 key → ed25519 algorithm
  - Signature headers added correctly

### Integration Tests

**Requirements**:
- Local Ethereum testnet (Hardhat/Anvil)
- Deployed SageRegistryV4 contract
- Test agent with multiple keys

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

**Metrics to Measure**:
- DID resolution time: < 100ms
- Signature verification time: < 50ms
- Key selection overhead: < 10ms

---

## Reference Materials

### SAGE Documentation

- [SAGE Architecture](https://github.com/sage-x-project/sage/blob/main/docs/ARCHITECTURE.md)
- [SAGE A2A Integration Guide](https://github.com/sage-x-project/sage/blob/main/docs/SAGE_A2A_INTEGRATION_GUIDE.md)
- [DID Registry V4 Contract](https://github.com/sage-x-project/sage/blob/main/contracts/ethereum/contracts/SageRegistryV4.sol)
- [Multi-Key Resolution Tests](https://github.com/sage-x-project/sage/blob/main/pkg/agent/did/ethereum/clientv4_multikey_resolution_test.go)
- [RFC9421 Verifier](https://github.com/sage-x-project/sage/blob/main/pkg/agent/core/rfc9421/verifier.go)

### External Specifications

- [RFC9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [A2A Protocol](https://github.com/a2aproject/a2a)
- [DID Core Specification](https://www.w3.org/TR/did-core/)

### SAGE API Examples

**DID Resolution**:
```go
client.ResolveAllPublicKeys(ctx, agentDID)
client.ResolvePublicKeyByType(ctx, agentDID, keyType)
```

**DID Generation**:
```go
did.GenerateAgentDIDWithAddress(chain, ownerAddress)
did.GenerateAgentDIDWithNonce(chain, ownerAddress, nonce)
did.DeriveEthereumAddress(keyPair)
```

**Key Marshaling**:
```go
did.MarshalPublicKey(pubKey)
did.UnmarshalPublicKey(data, keyType)
```

**A2A Agent Cards**:
```go
did.GenerateA2ACard(metadata)
did.ValidateA2ACard(card)
did.MergeA2ACard(metadata, card)
```

---

## Checklist

Before considering implementation complete, verify:

### Core Functionality
- [ ] DIDVerifier can verify RFC9421 signatures using DIDs
- [ ] KeySelector chooses correct key based on protocol
- [ ] A2ASigner creates valid RFC9421 signatures
- [ ] All unit tests pass
- [ ] Integration tests pass

### Examples
- [ ] All 4 examples run successfully
- [ ] Examples demonstrate key use cases
- [ ] Example code is well-commented

### Documentation
- [ ] README.md is complete
- [ ] QUICKSTART.md has step-by-step guide
- [ ] API.md documents all public interfaces
- [ ] Code has godoc comments

### Quality
- [ ] Code follows Go best practices
- [ ] No golangci-lint errors
- [ ] Test coverage > 80%
- [ ] All public functions documented

### Integration
- [ ] Works with SAGE v1.1.0+
- [ ] Compatible with SageRegistryV4 contract
- [ ] Handles both Ethereum and Solana DIDs

---

## Next Steps After Implementation

1. **Create GitHub Repository**
   ```bash
   gh repo create sage-x-project/sage-a2a-go --public
   ```

2. **Push Initial Code**
   ```bash
   git init
   git add .
   git commit -m "Initial commit: sage-a2a-go implementation"
   git branch -M main
   git remote add origin https://github.com/sage-x-project/sage-a2a-go.git
   git push -u origin main
   ```

3. **Setup CI/CD**
   - Add GitHub Actions for tests
   - Add code coverage reporting
   - Add linting workflow

4. **Release v0.1.0**
   - Tag first release
   - Publish to pkg.go.dev
   - Update SAGE documentation with link

---

**Document Version**: 1.0
**Last Updated**: 2025-01-19
**Author**: SAGE Development Team
**Contact**: https://github.com/sage-x-project/sage
