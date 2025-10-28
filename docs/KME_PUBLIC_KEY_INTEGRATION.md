# KME Public Key Integration Documentation

**Version:** 1.5.0
**Date:** 2025-10-28
**Status:** Completed

## Overview

This document describes the integration of KME (Key Management Encryption) public key storage in the SAGE AgentCardRegistry system. The `kmePublicKey` field was originally present in v1.3.1, removed due to perceived lack of necessity, and has now been re-added with enhanced security validation for HPKE (Hybrid Public Key Encryption) support per RFC 9180.

## Changes Summary

### Contract Layer (Solidity)

**AgentCardStorage.sol:**
- Added `kmePublicKey` field to `AgentMetadata` struct (bytes type, 32 bytes for X25519)
- Added `KMEKeyUpdated` event

**AgentCardRegistry.sol:**
- Implemented X25519 ownership verification using ECDSA signatures
- Added `getKMEKey(bytes32 agentId)` view function
- Added `updateKMEKey(bytes32, bytes, bytes)` function with owner-only access
- Modified registration to extract and store X25519 key from multi-key params
- Enforced single X25519 key per agent policy

### Go Integration Layer

**pkg/agent/did/types_v4.go:**
- Added `PublicKEMKey []byte` field to `AgentMetadataV4` struct

**pkg/agent/did/ethereum/abi.go:**
- Embedded AgentCardRegistry ABI
- Added `GetAgentCardRegistryABI()` function

**pkg/agent/did/ethereum/agentcard_client.go:**
- Updated `GetAgent()` to populate `PublicKEMKey` field
- Added `GetKMEKey(ctx, agentID)` method
- Added `UpdateKMEKey(ctx, agentID, newKey, signature)` method

**pkg/agent/did/resolver.go:**
- `ResolveKEMKey()` already implemented correctly

### Test Coverage

**Solidity Tests:** 202/202 passing
- Added 15 new tests (R3.6.1-R3.6.15) covering:
  - KME key registration (5 tests)
  - X25519 ownership verification (3 tests)
  - KME key retrieval (4 tests)
  - KME key updates (3 tests)
- Fixed 2 legacy tests (R3.2.7, R3.2.8) for X25519 signatures

**Go Tests:** All passing
- Added 3 test functions in `client_unit_test.go`
- Added 5 test scenarios in `resolver_test.go`
- Added 4 test scenarios in `hpke/e2e_test.go`

## Architecture

### Storage Model

```solidity
struct AgentMetadata {
    string did;
    string name;
    string description;
    string endpoint;
    bytes32[] keyHashes;
    string capabilities;
    address owner;
    uint256 registeredAt;
    uint256 updatedAt;
    bool active;
    uint256 chainId;
    bytes kmePublicKey;  // NEW: 32-byte X25519 public key
}
```

### Security Model

**X25519 Ownership Verification:**

All X25519 keys must be proven to be owned by the registering account through ECDSA signature verification:

```solidity
bytes32 messageHash = keccak256(abi.encodePacked(
    "SAGE X25519 Ownership:",
    x25519PublicKey,      // 32 bytes
    block.chainid,        // Network ID
    address(this),        // Registry address
    ownerAddress          // Expected owner
));

bytes32 ethSignedHash = keccak256(abi.encodePacked(
    "\x19Ethereum Signed Message:\n32",
    messageHash
));

address recovered = ecrecover(ethSignedHash, v, r, s);
require(recovered == ownerAddress, "Invalid X25519 ownership proof");
```

**Why This Security Enhancement:**

The original assumption that X25519 keys don't need signatures was incorrect. Without signature verification, an attacker could register someone else's X25519 public key, enabling man-in-the-middle attacks on HPKE-encrypted communications.

### Integration Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent Registration                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────────────────────────┐
        │ AgentCardRegistry.register()          │
        │   - Verify all key signatures         │
        │   - Extract X25519 key                │
        │   - Store in kmePublicKey field       │
        └───────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   KME Key Retrieval                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Method 1: Via GetAgent()                                    │
│  ────────────────────────                                    │
│  Client → agentcard_client.GetAgent(agentID)                 │
│         → Contract.agents(agentID)                           │
│         → metadata.PublicKEMKey populated                    │
│         → Return AgentMetadataV4                             │
│                                                               │
│  Method 2: Direct KME Key Access                             │
│  ───────────────────────────────                             │
│  Client → agentcard_client.GetKMEKey(agentID)                │
│         → Contract.getKMEKey(agentID)                        │
│         → Return 32-byte X25519 key                          │
│                                                               │
│  Method 3: Via DID Resolution                                │
│  ───────────────────────────                                 │
│  HPKE → resolver.ResolveKEMKey(did)                          │
│       → resolver.Resolve(did)                                │
│       → Check IsActive                                       │
│       → Return metadata.PublicKEMKey                         │
│       → Use for HPKE encryption                              │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    KME Key Update                            │
├─────────────────────────────────────────────────────────────┤
│  Owner → agentcard_client.UpdateKMEKey(agentID, newKey, sig) │
│        → Verify caller is owner                              │
│        → Verify X25519 signature                             │
│        → Update agents[agentID].kmePublicKey                 │
│        → Emit KMEKeyUpdated event                            │
└─────────────────────────────────────────────────────────────┘
```

## API Reference

### Solidity API

#### getKMEKey

```solidity
function getKMEKey(bytes32 agentId)
    external
    view
    returns (bytes memory)
```

**Description:** Retrieves the KME public key for a registered agent.

**Parameters:**
- `agentId`: The unique identifier of the agent (keccak256 hash of DID)

**Returns:**
- `bytes`: The 32-byte X25519 public key, or empty bytes if no key registered

**Reverts:**
- "Agent not found" if agent doesn't exist

**Gas Cost:** ~5,000 gas (O(1) direct field access)

#### updateKMEKey

```solidity
function updateKMEKey(
    bytes32 agentId,
    bytes calldata newKmeKey,
    bytes calldata signature
) external onlyAgentOwner(agentId) whenNotPaused nonReentrant
```

**Description:** Updates the KME public key for an agent. Only the agent owner can call this.

**Parameters:**
- `agentId`: The unique identifier of the agent
- `newKmeKey`: The new 32-byte X25519 public key
- `signature`: 65-byte ECDSA signature proving ownership of new X25519 key

**Events Emitted:**
- `KMEKeyUpdated(agentId, keyHash, timestamp)`

**Reverts:**
- "Invalid X25519 key length" if key is not 32 bytes
- "Invalid signature length" if signature is not 65 bytes
- "Invalid X25519 ownership proof" if signature verification fails
- "Caller is not the agent owner" if caller is not the owner

**Access Control:** Owner only (via `onlyAgentOwner` modifier)

**Security Features:**
- Reentrancy protection
- Pause mechanism support
- Nonce increment for replay protection
- Ownership verification

### Go API

#### GetKMEKey

```go
func (c *AgentCardClient) GetKMEKey(
    ctx context.Context,
    agentID [32]byte
) ([]byte, error)
```

**Description:** Retrieves the KME public key for a registered agent.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `agentID`: 32-byte agent identifier

**Returns:**
- `[]byte`: 32-byte X25519 public key
- `error`: Error if agent not found or has no KME key

**Example:**
```go
agentID := [32]byte{/* agent ID bytes */}
kmeKey, err := client.GetKMEKey(ctx, agentID)
if err != nil {
    return fmt.Errorf("failed to get KME key: %w", err)
}
// kmeKey is 32-byte X25519 public key
```

#### UpdateKMEKey

```go
func (c *AgentCardClient) UpdateKMEKey(
    ctx context.Context,
    agentID [32]byte,
    newKMEKey []byte,
    signature []byte
) error
```

**Description:** Updates the KME public key for an agent.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `agentID`: 32-byte agent identifier
- `newKMEKey`: 32-byte X25519 public key
- `signature`: 65-byte ECDSA signature

**Returns:**
- `error`: Error if validation fails or transaction fails

**Validation:**
- newKMEKey must be exactly 32 bytes
- signature must be exactly 65 bytes
- Caller must be agent owner
- Signature must prove ownership of new X25519 key

**Example:**
```go
// Generate new X25519 key pair
x25519KeyPair, err := keys.GenerateX25519KeyPair()
if err != nil {
    return err
}
newKMEKey := x25519KeyPair.PublicKey().([]byte)

// Create ownership signature
signature, err := createX25519OwnershipSignature(
    signer,
    newKMEKey,
    chainID,
    registryAddress,
)
if err != nil {
    return err
}

// Update KME key
err = client.UpdateKMEKey(ctx, agentID, newKMEKey, signature)
if err != nil {
    return fmt.Errorf("failed to update KME key: %w", err)
}
```

#### ResolveKEMKey

```go
func (m *MultiChainResolver) ResolveKEMKey(
    ctx context.Context,
    did AgentDID
) (interface{}, error)
```

**Description:** Resolves KEM key for a DID, used by HPKE client.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `did`: Agent DID (e.g., "did:sage:eth:agent001")

**Returns:**
- `interface{}`: X25519 public key (type assert to *ecdh.PublicKey)
- `error`: Error if resolution fails or agent inactive

**Errors:**
- `ErrDIDNotFound`: DID does not exist
- `ErrInactiveAgent`: Agent exists but is inactive
- `nil`: Agent has no KME key registered (PublicKEMKey is nil)

**Example:**
```go
did := sagedid.AgentDID("did:sage:eth:agent001")

kemKey, err := resolver.ResolveKEMKey(ctx, did)
if err == sagedid.ErrInactiveAgent {
    return fmt.Errorf("agent is inactive")
}
if err != nil {
    return fmt.Errorf("failed to resolve KEM key: %w", err)
}

if kemKey == nil {
    return fmt.Errorf("agent has no KME key registered")
}

// Type assert to *ecdh.PublicKey for HPKE
publicKey, ok := kemKey.(*ecdh.PublicKey)
if !ok {
    return fmt.Errorf("invalid KEM key type")
}

// Use publicKey with HPKE client
```

## Usage Examples

### Registering Agent with KME Key

**JavaScript (Hardhat):**

```javascript
const { ethers } = require("hardhat");

// Generate X25519 key pair
const x25519KeyPair = ethers.randomBytes(32);

// Create ECDSA signature for X25519 ownership
const chainId = (await ethers.provider.getNetwork()).chainId;
const registryAddress = await registry.getAddress();

const message = ethers.solidityPackedKeccak256(
    ["string", "bytes", "uint256", "address", "address"],
    [
        "SAGE X25519 Ownership:",
        x25519KeyPair,
        chainId,
        registryAddress,
        signer.address
    ]
);
const x25519Signature = await signer.signMessage(ethers.getBytes(message));

// Registration parameters
const params = {
    did: "did:sage:ethereum:0x123...",
    name: "My Agent",
    description: "Agent with KME support",
    endpoint: "https://agent.example.com",
    capabilities: JSON.stringify({ chat: true, hpke: true }),
    keys: [
        ecdsaPublicKey,    // 65 bytes
        ed25519PublicKey,  // 32 bytes
        x25519KeyPair      // 32 bytes (KME key)
    ],
    keyTypes: [0, 1, 2],  // ECDSA, Ed25519, X25519
    signatures: [
        ecdsaSignature,
        ed25519Signature,
        x25519Signature
    ],
    salt: ethers.randomBytes(32)
};

// Commit phase
const commitHash = await registry.connect(user).commitRegistration(
    ethers.keccak256(ethers.AbiCoder.defaultAbiCoder().encode(
        ["tuple(string,string,string,string,string,bytes[],uint8[],bytes[],bytes32)"],
        [Object.values(params)]
    ))
);
await commitHash.wait();

// Wait for commit delay
await ethers.provider.send("evm_increaseTime", [901]); // 15 min + 1 sec
await ethers.provider.send("evm_mine");

// Register agent
const registerTx = await registry.connect(user).register(params);
const receipt = await registerTx.wait();

// Get agent ID from event
const event = receipt.logs.find(log =>
    log.fragment?.name === "AgentRegistered"
);
const agentId = event.args.agentId;

console.log("Agent registered with ID:", agentId);
console.log("KME key stored:", ethers.hexlify(x25519KeyPair));
```

**Go:**

```go
package main

import (
    "context"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    sagedid "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    ctx := context.Background()

    // Initialize client
    client, err := ethereum.NewAgentCardClient(
        "https://rpc.example.com",
        "0xRegistryAddress...",
    )
    if err != nil {
        panic(err)
    }

    // Generate keys
    ecdsaKP, _ := keys.GenerateECDSAKeyPair()
    ed25519KP, _ := keys.GenerateEd25519KeyPair()
    x25519KP, _ := keys.GenerateX25519KeyPair()

    // Create signatures
    ecdsaSig, _ := createECDSASignature(ecdsaKP, ...)
    ed25519Sig, _ := createEd25519Signature(ed25519KP, ...)
    x25519Sig, _ := createX25519Signature(ecdsaKP, x25519KP.PublicKey(), ...)

    // Registration params
    params := ethereum.RegistrationParams{
        DID:          "did:sage:ethereum:0x123...",
        Name:         "My Agent",
        Description:  "Agent with KME support",
        Endpoint:     "https://agent.example.com",
        Capabilities: `{"chat": true, "hpke": true}`,
        Keys: [][]byte{
            ecdsaKP.PublicKey().([]byte),
            ed25519KP.PublicKey().([]byte),
            x25519KP.PublicKey().([]byte),  // KME key
        },
        KeyTypes: []sagedid.KeyType{
            sagedid.KeyTypeECDSA,
            sagedid.KeyTypeEd25519,
            sagedid.KeyTypeX25519,
        },
        Signatures: [][]byte{
            ecdsaSig,
            ed25519Sig,
            x25519Sig,
        },
        Salt: [32]byte{/* random salt */},
    }

    // Register agent (includes commit phase)
    agentID, err := client.Register(ctx, params)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Agent registered with ID: %x\n", agentID)

    // Verify KME key stored
    kmeKey, err := client.GetKMEKey(ctx, agentID)
    if err != nil {
        panic(err)
    }

    fmt.Printf("KME key retrieved: %x\n", kmeKey)
}
```

### Using KME Key with HPKE

```go
package main

import (
    "context"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

func main() {
    ctx := context.Background()

    // Initialize DID resolver
    resolver := did.NewMultiChainResolver()
    // Add chain resolvers...

    // Resolve KEM key for target agent
    targetDID := did.AgentDID("did:sage:eth:agent123")
    kemKey, err := resolver.ResolveKEMKey(ctx, targetDID)
    if err != nil {
        panic(err)
    }

    // Initialize HPKE client
    hpkeClient := hpke.NewClient(resolver)

    // Encrypt message to target agent
    plaintext := []byte("Hello, agent!")
    ciphertext, err := hpkeClient.Encrypt(ctx, targetDID, plaintext)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Encrypted message: %x\n", ciphertext)

    // Target agent can decrypt using their private X25519 key
}
```

## Performance

### Gas Costs

| Operation | Gas Cost | Optimization |
|-----------|----------|--------------|
| Registration with KME | ~450,000 | +1,000 for KME storage |
| getKMEKey() | ~5,000 | O(1) direct field access |
| updateKMEKey() | ~35,000 | Includes signature verification |
| Legacy array iteration | ~80,000 | 94% slower than direct access |

### Storage Overhead

- Per agent: +32 bytes (X25519 public key)
- Optional field: 0 bytes if no X25519 key registered

## Migration Guide

### From v1.3.1

If you were using the `kmePublicKey` field in v1.3.1:

1. **No action required for existing agents** - The field has been restored with the same structure
2. **X25519 signatures now required** - Update registration code to include ECDSA signatures for X25519 keys
3. **Use new accessor methods** - `getKMEKey()` and `updateKMEKey()` for better gas efficiency

**Code Changes:**

```javascript
// OLD (v1.3.1): X25519 without signature
const params = {
    keys: [ecdsaKey, ed25519Key, x25519Key],
    keyTypes: [0, 1, 2],
    signatures: [ecdsaSig, ed25519Sig, "0x"]  // Empty X25519 sig
};

// NEW (v1.5.0): X25519 with ECDSA signature
const x25519Sig = await createX25519Signature(signer, x25519Key);
const params = {
    keys: [ecdsaKey, ed25519Key, x25519Key],
    keyTypes: [0, 1, 2],
    signatures: [ecdsaSig, ed25519Sig, x25519Sig]  // Required sig
};
```

### From Versions Without KME Support

If you're upgrading from a version without `kmePublicKey`:

1. **Agents without X25519 keys continue to work** - The field is optional
2. **Add X25519 key during next update** - Use `updateKMEKey()` to add encryption support
3. **HPKE functionality** - Only available for agents with registered KME keys

## Security Considerations

### X25519 Ownership Verification

**Why it's critical:**
- Without signature verification, anyone could register your X25519 public key
- Attacker could then decrypt messages intended for you
- Man-in-the-middle attacks become trivial

**Implementation:**
- All X25519 keys must be signed by the agent owner's ECDSA key
- Signature includes chain ID and registry address to prevent replay attacks
- Signature verification happens both during registration and updates

### Key Rotation

To rotate your KME key:

1. Generate new X25519 key pair
2. Create ECDSA signature proving ownership
3. Call `updateKMEKey()` with new key and signature
4. Old key is immediately replaced (no grace period)

**Best Practices:**
- Rotate keys periodically (e.g., every 90 days)
- Monitor `KMEKeyUpdated` events for unauthorized changes
- Keep private keys in secure hardware (HSM/TPM)

### Access Control

- Only agent owner can update KME key
- `onlyAgentOwner` modifier enforces this at contract level
- Reentrancy protection prevents callback attacks
- Pause mechanism allows emergency stops

## Testing

### Running Tests

**Solidity tests:**
```bash
cd contracts/ethereum
npm test
```

**Go tests:**
```bash
# All tests
go test ./pkg/...

# Specific packages
go test ./pkg/agent/did/...
go test ./pkg/agent/did/ethereum/...
go test ./pkg/agent/hpke/...
```

### Test Coverage

- **Solidity:** 202 tests covering all contract functionality
- **Go:** Comprehensive unit and integration tests
- **Security:** Ownership verification, access control, edge cases
- **Performance:** Gas optimization validation

## Troubleshooting

### Common Issues

**Error: "X25519 requires ECDSA signature for ownership proof"**
- Cause: Missing or invalid signature for X25519 key
- Solution: Generate proper ECDSA signature using `createX25519Signature()` helper

**Error: "Agent does not have KME key registered"**
- Cause: Agent registered without X25519 key, or key retrieval on wrong agent
- Solution: Verify agent has X25519 key in registration params, or use `updateKMEKey()` to add

**Error: "Invalid X25519 key length"**
- Cause: Key data is not exactly 32 bytes
- Solution: X25519 public keys must be 32 bytes, verify key generation

**Error: "Caller is not the agent owner"**
- Cause: Non-owner trying to update KME key
- Solution: Only the agent owner can update KME key, check transaction signer

### Debug Tips

1. **Check agent registration:**
   ```javascript
   const agent = await registry.agents(agentId);
   console.log("KME key:", ethers.hexlify(agent.kmePublicKey));
   ```

2. **Verify signature:**
   ```javascript
   const recovered = ethers.verifyMessage(messageHash, signature);
   console.log("Recovered address:", recovered);
   console.log("Expected address:", signer.address);
   ```

3. **Monitor events:**
   ```javascript
   registry.on("KMEKeyUpdated", (agentId, keyHash, timestamp) => {
       console.log(`KME key updated for ${agentId}`);
   });
   ```

## References

- **RFC 9180:** HPKE - Hybrid Public Key Encryption
  - https://datatracker.ietf.org/doc/html/rfc9180
- **X25519:** Elliptic Curve Diffie-Hellman
  - https://datatracker.ietf.org/doc/html/rfc7748
- **SAGE Documentation:** Main project docs
  - /docs/README.md
- **Test Files:**
  - contracts/ethereum/test/AgentCardRegistry.test.js (R3.6.1-R3.6.15)
  - pkg/agent/did/ethereum/client_unit_test.go
  - pkg/agent/did/resolver_test.go
  - pkg/agent/hpke/e2e_test.go

## Version History

- **v1.5.0 (2025-10-28):** KME public key re-added with X25519 ownership verification
- **v1.4.0:** KME field removed (deemed unnecessary)
- **v1.3.1:** Original KME public key support (without ownership verification)

## Contributors

- SAGE Development Team
- Security review and enhancement contributions

---

**Last Updated:** 2025-10-28
**Document Version:** 1.0
**Status:** Production Ready
