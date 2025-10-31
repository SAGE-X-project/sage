# SAGE Multi-Key Support & A2A Agent Card Integration

**Date**: 2025-10-18
**Status**: Design Document
**Version**: 1.0

---

## Executive Summary

This document proposes a comprehensive redesign of SAGE's smart contract and backend architecture to support:
1. **Multi-key storage** per agent (Ed25519, ECDSA, X25519)
2. **Chain-specific verification** strategies
3. **A2A Agent Card** integration for Google's A2A protocol
4. **Proper ownership proof** for each key type

---

## Table of Contents

1. [Current Architecture Analysis](#1-current-architecture-analysis)
2. [Problem Statement](#2-problem-statement)
3. [Requirements](#3-requirements)
4. [Proposed Solution](#4-proposed-solution)
5. [Smart Contract Design](#5-smart-contract-design)
6. [Go Backend Integration](#6-go-backend-integration)
7. [Verification Strategies](#7-verification-strategies)
8. [Migration Plan](#8-migration-plan)
9. [Security Considerations](#9-security-considerations)

---

## 1. Current Architecture Analysis

### 1.1 Current Solidity Structure

```solidity
struct AgentMetadata {
    string did;
    string name;
    string description;
    string endpoint;
    bytes publicKey;        // ❌ Single key only
    string capabilities;
    address owner;
    uint256 registeredAt;
    uint256 updatedAt;
    bool active;
}
```

**Limitations:**
- ❌ Only stores ONE public key per agent
- ❌ Cannot distinguish between Ed25519, ECDSA, X25519
- ❌ No chain-specific key support
- ❌ Ed25519 verification rejected (on-chain impossible)

### 1.2 Current Go Structure

```go
type AgentMetadata struct {
    DID          AgentDID
    PublicKey    interface{}     // ❌ Single key
    PublicKEMKey interface{}     // Separate KEM key
    // ... other fields
}
```

**Limitations:**
- ❌ Public key and KEM key stored separately (poor scaling)
- ❌ No support for multiple keys of same type
- ❌ No key type metadata

### 1.3 Current Use Cases

**SAGE's Multi-Chain Reality:**

| Chain | Signature Algorithm | Key Exchange | Use Case |
|-------|-------------------|--------------|----------|
| **Ethereum** | ECDSA (secp256k1) | X25519 | Smart contracts, transactions |
| **Solana** | Ed25519 | X25519 | On-chain programs |
| **Off-Chain** | Ed25519 | X25519 | HPKE handshake, RFC 9421 |

**Problem:** An agent operating across multiple chains needs BOTH Ed25519 AND ECDSA keys stored on-chain.

---

## 2. Problem Statement

### 2.1 The Core Issue

**Scenario:**
1. Agent wants to register on SAGE registry (Ethereum contract)
2. Agent has Ed25519 keys (for Solana, off-chain messaging)
3. Agent has ECDSA keys (for Ethereum transactions)
4. **Current contract ONLY accepts ONE key and REJECTS Ed25519**

### 2.2 Why Ed25519 Cannot Be Verified On-Chain (Ethereum)

```solidity
// Current problematic code
if (publicKey.length == 32) {  // Ed25519
    revert("Ed25519 not supported on-chain");  // ❌
}
```

**Reason:** Ethereum's `ecrecover` precompile ONLY supports ECDSA (secp256k1).
**Impact:** Agents using Ed25519 (Solana, off-chain) cannot register.

### 2.3 Real-World Agent Needs

```
Agent "Alice" Needs:
├── Ed25519 Key (32 bytes)
│   ├── For: Solana transactions
│   ├── For: RFC 9421 signatures (off-chain)
│   └── For: A2A protocol messaging
├── ECDSA Key (65 bytes)
│   ├── For: Ethereum transactions
│   └── For: On-chain verification
└── X25519 Key (32 bytes)
    └── For: HPKE key exchange
```

**Current Solution:** ❌ Pick ONE
**Needed Solution:** ✅ Store ALL with proper verification

---

## 3. Requirements

### 3.1 Functional Requirements

**FR1: Multi-Key Storage**
- Agent MUST be able to register multiple public keys
- Each key MUST have: type, data, verification proof

**FR2: Key Type Support**
- Ed25519 (32 bytes): Solana, off-chain signatures
- ECDSA/secp256k1 (33/65 bytes): Ethereum on-chain
- X25519 (32 bytes): HPKE key exchange

**FR3: Ownership Verification**
- Each key MUST prove ownership during registration
- ECDSA: On-chain verification via `ecrecover`
- Ed25519: Owner approval OR off-chain proof

**FR4: A2A Agent Card Integration**
- Agent metadata SHOULD map to A2A Agent Card format
- Support for multiple communication protocols

### 3.2 Non-Functional Requirements

**NFR1: Gas Efficiency**
- Multi-key registration SHOULD NOT exceed 500k gas
- Use batching where possible

**NFR2: Backward Compatibility**
- Existing agents SHOULD still work
- Migration path from single-key to multi-key

**NFR3: Security**
- No key type SHOULD be stored without proof
- Revocation MUST deactivate ALL associated agents

---

## 4. Proposed Solution

### 4.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent Registration Flow                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent (Go Backend)                                          │
│  ├── Generate Keys                                           │
│  │   ├── Ed25519 (Solana, off-chain)                        │
│  │   ├── ECDSA (Ethereum)                                   │
│  │   └── X25519 (HPKE)                                      │
│  │                                                           │
│  ├── Sign Ownership Proofs                                  │
│  │   ├── ECDSA signature (on-chain verifiable)             │
│  │   └── Ed25519 signature (off-chain or owner-approved)    │
│  │                                                           │
│  └── Submit to Contract                                     │
│      └── registerAgentWithKeys(did, keys[], proofs[])      │
│                                                              │
│  Smart Contract (Ethereum)                                   │
│  ├── Validate Each Key Type                                 │
│  │   ├── ECDSA → ecrecover (on-chain)                      │
│  │   ├── Ed25519 → approvedEd25519Keys[hash] (owner)       │
│  │   └── X25519 → store without verification (public)      │
│  │                                                           │
│  ├── Store Multi-Key Metadata                               │
│  │   └── AgentKeys[] = [{type, data, verified}]            │
│  │                                                           │
│  └── Emit Event                                             │
│      └── AgentRegisteredWithKeys(did, keyHashes[])         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Key Components

**Component 1: AgentKey Struct**
```solidity
struct AgentKey {
    KeyType keyType;        // Ed25519, ECDSA, X25519
    bytes keyData;          // Raw public key bytes
    bytes signature;        // Ownership proof signature
    bool verified;          // Verification status
    uint256 registeredAt;   // Timestamp
}
```

**Component 2: Multi-Key Storage**
```solidity
mapping(bytes32 => AgentKey[]) private agentKeys;  // agentId => keys
```

**Component 3: Verification Registry**
```solidity
mapping(bytes32 => bool) public approvedEd25519Keys;  // keyHash => approved
mapping(bytes32 => bool) public verifiedECDSAKeys;    // keyHash => verified
```

---

## 5. Smart Contract Design

### 5.1 Enhanced Data Structures

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

/**
 * @title SageRegistryV4 - Multi-Key Support
 * @notice Enhanced agent registry with support for multiple key types
 */
contract SageRegistryV4 {
    // Key type enumeration
    enum KeyType {
        Ed25519,      // 32 bytes, Solana & off-chain
        ECDSA,        // 33/65 bytes, Ethereum
        X25519        // 32 bytes, HPKE key exchange
    }

    // Individual key with metadata
    struct AgentKey {
        KeyType keyType;
        bytes keyData;
        bytes signature;      // Ownership proof
        bool verified;        // Verification status
        uint256 registeredAt;
    }

    // Agent metadata (updated)
    struct AgentMetadata {
        string did;
        string name;
        string description;
        string endpoint;
        bytes32[] keyHashes;  // References to keys
        string capabilities;  // JSON (includes A2A Agent Card)
        address owner;
        uint256 registeredAt;
        uint256 updatedAt;
        bool active;
    }

    // Storage
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(bytes32 => AgentKey) private keys;  // keyHash => key
    mapping(bytes32 => bytes32[]) private agentToKeys;  // agentId => keyHashes[]

    // Verification registries
    mapping(bytes32 => bool) public approvedEd25519Keys;
    mapping(bytes32 => address) public keyOwners;
}
```

### 5.2 Registration Function

```solidity
/**
 * @notice Register agent with multiple keys
 * @dev Supports Ed25519, ECDSA, and X25519 keys
 */
function registerAgentWithKeys(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    string calldata capabilities,
    KeyRegistration[] calldata keys
) external nonReentrant whenNotPaused returns (bytes32 agentId) {
    // Validate inputs
    require(bytes(did).length > 0, "DID required");
    require(keys.length > 0 && keys.length <= 10, "Invalid key count");

    // Generate agent ID
    agentId = keccak256(abi.encode(did, msg.sender, block.number));

    // Validate and store each key
    bytes32[] memory keyHashes = new bytes32[](keys.length);

    for (uint256 i = 0; i < keys.length; i++) {
        keyHashes[i] = _validateAndStoreKey(agentId, keys[i]);
    }

    // Store agent metadata
    agents[agentId] = AgentMetadata({
        did: did,
        name: name,
        description: description,
        endpoint: endpoint,
        keyHashes: keyHashes,
        capabilities: capabilities,
        owner: msg.sender,
        registeredAt: block.timestamp,
        updatedAt: block.timestamp,
        active: true
    });

    emit AgentRegisteredWithKeys(agentId, msg.sender, did, keyHashes);
}

struct KeyRegistration {
    KeyType keyType;
    bytes keyData;
    bytes signature;
}
```

### 5.3 Key Validation Strategies

```solidity
/**
 * @notice Validate and store a single key
 */
function _validateAndStoreKey(
    bytes32 agentId,
    KeyRegistration calldata keyReg
) private returns (bytes32 keyHash) {
    keyHash = keccak256(keyReg.keyData);

    // Verify ownership based on key type
    bool verified = false;

    if (keyReg.keyType == KeyType.ECDSA) {
        // On-chain verification via ecrecover
        verified = _verifyECDSAOwnership(
            keyReg.keyData,
            keyReg.signature,
            msg.sender
        );
        require(verified, "ECDSA verification failed");

    } else if (keyReg.keyType == KeyType.Ed25519) {
        // Owner pre-approval required
        verified = approvedEd25519Keys[keyHash];
        require(verified, "Ed25519 key not approved");

    } else if (keyReg.keyType == KeyType.X25519) {
        // No verification needed (public by design)
        verified = true;
    }

    // Store key
    keys[keyHash] = AgentKey({
        keyType: keyReg.keyType,
        keyData: keyReg.keyData,
        signature: keyReg.signature,
        verified: verified,
        registeredAt: block.timestamp
    });

    keyOwners[keyHash] = msg.sender;
    agentToKeys[agentId].push(keyHash);

    emit KeyRegistered(agentId, keyHash, keyReg.keyType);
}
```

### 5.4 Ed25519 Key Approval (Owner Function)

```solidity
/**
 * @notice Pre-approve Ed25519 keys for registration
 * @dev Only contract owner can approve Ed25519 keys
 *      This is necessary because Ethereum cannot verify Ed25519 on-chain
 */
function approveEd25519Keys(
    bytes[] calldata ed25519PublicKeys
) external onlyOwner {
    for (uint256 i = 0; i < ed25519PublicKeys.length; i++) {
        require(ed25519PublicKeys[i].length == 32, "Invalid Ed25519 key length");

        bytes32 keyHash = keccak256(ed25519PublicKeys[i]);
        approvedEd25519Keys[keyHash] = true;

        emit Ed25519KeyApproved(keyHash, ed25519PublicKeys[i]);
    }
}

/**
 * @notice Batch approve Ed25519 keys with proof
 * @dev Alternative: Provide off-chain verification proof
 */
function approveEd25519KeysWithProof(
    bytes[] calldata ed25519PublicKeys,
    bytes[] calldata proofs
) external onlyOwner {
    require(ed25519PublicKeys.length == proofs.length, "Length mismatch");

    for (uint256 i = 0; i < ed25519PublicKeys.length; i++) {
        // Store proof for transparency
        bytes32 keyHash = keccak256(ed25519PublicKeys[i]);

        // In production, verify proof via oracle or ZK proof
        // For now, trust owner
        approvedEd25519Keys[keyHash] = true;

        emit Ed25519KeyApprovedWithProof(keyHash, proofs[i]);
    }
}
```

---

## 6. Go Backend Integration

### 6.1 Updated AgentMetadata

```go
// pkg/agent/did/types.go

type KeyType string

const (
    KeyTypeEd25519 KeyType = "ed25519"
    KeyTypeECDSA   KeyType = "ecdsa"
    KeyTypeX25519  KeyType = "x25519"
)

type AgentKey struct {
    KeyType      KeyType       `json:"key_type"`
    KeyData      []byte        `json:"key_data"`
    Signature    []byte        `json:"signature"`
    Verified     bool          `json:"verified"`
    RegisteredAt time.Time     `json:"registered_at"`
}

type AgentMetadata struct {
    DID          AgentDID                `json:"did"`
    Name         string                  `json:"name"`
    Description  string                  `json:"description"`
    Endpoint     string                  `json:"endpoint"`
    Keys         []AgentKey              `json:"keys"`  // ✅ Multi-key support
    Capabilities map[string]interface{}  `json:"capabilities"`
    Owner        string                  `json:"owner"`
    IsActive     bool                    `json:"is_active"`
    CreatedAt    time.Time               `json:"created_at"`
    UpdatedAt    time.Time               `json:"updated_at"`
}
```

### 6.2 A2A Agent Card Generation

```go
// pkg/agent/did/agent_card.go

import "encoding/json"

// A2AAgentCard represents Google's A2A protocol Agent Card
type A2AAgentCard struct {
    ID          string            `json:"id"`           // Agent DID
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Endpoints   []A2AEndpoint     `json:"endpoints"`
    PublicKeys  []A2APublicKey    `json:"publicKeys"`   // ✅ Multiple keys
    Capabilities []string         `json:"capabilities"`
}

type A2AEndpoint struct {
    Protocol string `json:"protocol"`  // "grpc", "https", "ws"
    URI      string `json:"uri"`
}

type A2APublicKey struct {
    ID        string `json:"id"`         // Key identifier
    Type      string `json:"type"`       // "Ed25519", "EcdsaSecp256k1"
    PublicKey string `json:"publicKeyHex"` // Hex-encoded
    Purpose   string `json:"purpose"`    // "signing", "encryption"
}

// GenerateA2ACard creates an A2A Agent Card from AgentMetadata
func (m *AgentMetadata) GenerateA2ACard() (*A2AAgentCard, error) {
    card := &A2AAgentCard{
        ID:          string(m.DID),
        Name:        m.Name,
        Description: m.Description,
        Endpoints:   []A2AEndpoint{{Protocol: "https", URI: m.Endpoint}},
        PublicKeys:  make([]A2APublicKey, 0, len(m.Keys)),
    }

    // Convert SAGE keys to A2A format
    for _, key := range m.Keys {
        a2aKey := A2APublicKey{
            ID:        hex.EncodeToString(key.KeyData[:8]),  // First 8 bytes as ID
            PublicKey: hex.EncodeToString(key.KeyData),
        }

        switch key.KeyType {
        case KeyTypeEd25519:
            a2aKey.Type = "Ed25519VerificationKey2020"
            a2aKey.Purpose = "signing"
        case KeyTypeECDSA:
            a2aKey.Type = "EcdsaSecp256k1VerificationKey2019"
            a2aKey.Purpose = "signing"
        case KeyTypeX25519:
            a2aKey.Type = "X25519KeyAgreementKey2020"
            a2aKey.Purpose = "encryption"
        }

        card.PublicKeys = append(card.PublicKeys, a2aKey)
    }

    return card, nil
}

// ToJSON serializes the Agent Card to JSON
func (c *A2AAgentCard) ToJSON() (string, error) {
    bytes, err := json.MarshalIndent(c, "", "  ")
    return string(bytes), err
}
```

### 6.3 Multi-Key Registration Flow

```go
// pkg/agent/did/ethereum/client.go

type KeyRegistration struct {
    KeyType   KeyType
    KeyData   []byte
    Signature []byte
}

func (c *Client) RegisterAgentWithKeys(
    ctx context.Context,
    req *RegistrationRequest,
    keys []KeyRegistration,
) (*RegistrationResult, error) {
    // Validate all keys
    for _, key := range keys {
        if err := validateKeyFormat(key); err != nil {
            return nil, fmt.Errorf("invalid key: %w", err)
        }
    }

    // Convert to contract format
    contractKeys := make([]struct {
        KeyType   uint8
        KeyData   []byte
        Signature []byte
    }, len(keys))

    for i, key := range keys {
        contractKeys[i] = struct {
            KeyType   uint8
            KeyData   []byte
            Signature []byte
        }{
            KeyType:   keyTypeToUint8(key.KeyType),
            KeyData:   key.KeyData,
            Signature: key.Signature,
        }
    }

    // Call contract
    tx, err := c.contract.RegisterAgentWithKeys(
        c.auth,
        req.DID,
        req.Name,
        req.Description,
        req.Endpoint,
        capabilitiesJSON,
        contractKeys,
    )

    // ... transaction handling
}

func keyTypeToUint8(kt KeyType) uint8 {
    switch kt {
    case KeyTypeEd25519:
        return 0
    case KeyTypeECDSA:
        return 1
    case KeyTypeX25519:
        return 2
    default:
        return 255
    }
}
```

---

## 7. Verification Strategies

### 7.1 ECDSA (On-Chain Verification)

```solidity
function _verifyECDSAOwnership(
    bytes calldata publicKey,
    bytes calldata signature,
    address expectedOwner
) private pure returns (bool) {
    // Validate format
    require(
        publicKey.length == 65 && publicKey[0] == 0x04,
        "Invalid ECDSA key format"
    );

    // Create challenge
    bytes32 challenge = keccak256(abi.encodePacked(
        "SAGE Key Registration:",
        block.chainid,
        address(this),
        keccak256(publicKey)
    ));

    // Ethereum signed message prefix
    bytes32 ethSignedHash = keccak256(
        abi.encodePacked("\x19Ethereum Signed Message:\n32", challenge)
    );

    // Recover signer
    address recovered = _recoverSigner(ethSignedHash, signature);

    // Derive address from public key
    address keyAddress = _getAddressFromPublicKey(publicKey);

    return recovered == keyAddress && recovered == expectedOwner;
}
```

### 7.2 Ed25519 (Owner Approval Strategy)

**Option 1: Pre-Approval (Recommended)**

```solidity
// Step 1: Owner approves Ed25519 keys off-chain
function approveEd25519Keys(bytes[] calldata keys) external onlyOwner {
    for (uint256 i = 0; i < keys.length; i++) {
        bytes32 keyHash = keccak256(keys[i]);
        approvedEd25519Keys[keyHash] = true;
    }
}

// Step 2: Agent registers with approved key
function registerAgent(...) {
    if (keyType == KeyType.Ed25519) {
        require(approvedEd25519Keys[keyHash], "Key not approved");
    }
}
```

**Option 2: Oracle Verification (Future)**

```solidity
// Use Chainlink oracle for off-chain Ed25519 verification
function verifyEd25519WithOracle(
    bytes calldata publicKey,
    bytes calldata signature,
    bytes32 message
) external returns (bytes32 requestId) {
    // Request verification from oracle
    requestId = oracle.requestEd25519Verification(
        publicKey,
        signature,
        message
    );

    pendingVerifications[requestId] = keyHash;
}

function fulfillVerification(
    bytes32 requestId,
    bool verified
) external onlyOracle {
    bytes32 keyHash = pendingVerifications[requestId];
    approvedEd25519Keys[keyHash] = verified;
}
```

**Option 3: ZK Proof Verification (Advanced)**

```solidity
// Verify Ed25519 signature via ZK-SNARK proof
function verifyEd25519WithZKProof(
    bytes calldata publicKey,
    bytes calldata zkProof
) external view returns (bool) {
    // Verify ZK proof that Ed25519 signature is valid
    return zkVerifier.verifyProof(zkProof, publicKey);
}
```

### 7.3 X25519 (No Verification Needed)

```solidity
// X25519 is for HPKE key exchange (public by design)
function _storeX25519Key(
    bytes calldata keyData
) private {
    require(keyData.length == 32, "Invalid X25519 key length");

    // No signature verification needed
    // Store directly
    bytes32 keyHash = keccak256(keyData);
    keys[keyHash] = AgentKey({
        keyType: KeyType.X25519,
        keyData: keyData,
        signature: "", // No signature needed
        verified: true, // Always verified
        registeredAt: block.timestamp
    });
}
```

---

## 8. Migration Plan

### 8.1 Phase 1: Deploy V4 Contract (Week 1-2)

**Tasks:**
- [ ] Implement `SageRegistryV4.sol` with multi-key support
- [ ] Write comprehensive unit tests
- [ ] Deploy to testnet (Sepolia)
- [ ] Security audit

### 8.2 Phase 2: Update Go Backend (Week 3-4)

**Tasks:**
- [ ] Update `AgentMetadata` struct
- [ ] Implement `GenerateA2ACard()` function
- [ ] Update Ethereum client for multi-key registration
- [ ] Update DID resolver to handle multiple keys

### 8.3 Phase 3: Migration Tools (Week 5)

**Tasks:**
- [ ] Create migration script for existing agents
- [ ] Add backward compatibility layer
- [ ] Update CLI tools

### 8.4 Phase 4: Testing & Deployment (Week 6)

**Tasks:**
- [ ] Integration testing
- [ ] Load testing (gas optimization)
- [ ] Mainnet deployment
- [ ] Documentation updates

---

## 9. Security Considerations

### 9.1 Threats Addressed

✅ **Key Type Confusion**: Enum-based key types prevent confusion
✅ **Unauthorized Registration**: Each key requires ownership proof
✅ **Replay Attacks**: Challenge includes chain ID and contract address
✅ **Key Revocation**: Revoking one key doesn't affect others

### 9.2 Remaining Risks

⚠️ **Ed25519 Trust Model**: Relies on owner approval (centralization risk)
⚠️ **Gas Costs**: Multiple keys increase registration cost
⚠️ **Storage Bloat**: Agents with many keys increase state size

### 9.3 Mitigation Strategies

1. **Ed25519 Verification**: Implement oracle or ZK proof in Phase 2
2. **Gas Optimization**: Limit maximum keys per agent (10)
3. **Storage Management**: Use key hash references instead of full keys

---

## 10. Conclusion

This design provides a comprehensive solution for SAGE's multi-chain, multi-key requirements while integrating with Google's A2A protocol. The phased approach ensures security and backward compatibility.

**Next Steps:**
1. Review and approve design
2. Begin Phase 1 implementation
3. Schedule security audit

---

**Document Author**: Claude (SAGE Design Assistant)
**Reviewed By**: [Pending]
**Last Updated**: 2025-10-18
