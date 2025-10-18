# Multi-Key Registry V4 - Feature Roadmap

This document outlines planned features and enhancements for the SageRegistryV4 multi-key agent system.

## Status: In Progress

**Current Branch:** `feature/multi-key-registry-v4`

**Completed Features:**
- ✅ SageRegistryV4 smart contract with multi-key support
- ✅ Unit tests for smart contract (30 tests, 100% pass)
- ✅ V4 types in Go backend (AgentMetadataV4, AgentKey)
- ✅ A2A Agent Card generation and validation
- ✅ Unit tests for Go backend (37+ tests, 77.6% coverage)
- ✅ CLI commands: `sage-did card generate/validate/show`
- ✅ All 85 feature tests passing

---

## Planned Features

### 1. Multi-Key Registration Support (Priority: HIGH)

**Description:** Enable simultaneous registration of multiple key types through CLI

**Current Limitation:**
- `sage-did register` only supports single key per chain
- Ethereum: ECDSA only
- Solana: Ed25519 only

**Proposed Enhancement:**
```bash
# Register with multiple keys
sage-did register \
  --chain ethereum \
  --name "Multi-Key Agent" \
  --keys ed25519.jwk,ecdsa.pem,x25519.key \
  --key-types ed25519,ecdsa,x25519 \
  --endpoint https://agent.example.com
```

**Implementation Tasks:**
- [ ] Extend CLI to accept multiple key files
- [ ] Support key type auto-detection from file format
- [ ] Generate proper signatures for each key type
- [ ] Call SageRegistryV4.registerAgent with multiple keys
- [ ] Handle Ed25519 pre-approval flow
- [ ] Update help documentation with examples

**Benefits:**
- Enables true multi-chain agent identity
- Single transaction for all keys
- Immediate A2A protocol compatibility

**Estimated Effort:** 60-90 minutes

**Files to Modify:**
- `cmd/sage-did/register.go`
- `pkg/agent/did/manager.go`
- `pkg/agent/did/ethereum/client.go`

---

### 2. Key Management CLI Commands (Priority: HIGH)

**Description:** Add commands for managing keys on registered agents

**New Commands:**
```bash
# Add a new key to existing agent
sage-did key add <did> <keyfile> \
  --key-type ed25519 \
  --signature <sig>

# List all keys for an agent
sage-did key list <did>
# Output:
# Key Hash: 0x123...
#   Type: Ed25519VerificationKey2020
#   Verified: true
#   Registered: 2025-01-15 10:30:00
#
# Key Hash: 0x456...
#   Type: EcdsaSecp256k1VerificationKey2019
#   Verified: true
#   Registered: 2025-01-15 10:30:00

# Revoke a key
sage-did key revoke <did> <keyhash> \
  --private-key <owner-key>

# Approve Ed25519 key (owner only)
sage-did key approve <keyhash> \
  --private-key <registry-owner-key>
```

**Implementation Tasks:**
- [ ] Create `cmd/sage-did/key.go` with subcommands
- [ ] Implement addKey, revokeKey wrapper functions
- [ ] Add key listing with formatted output
- [ ] Handle owner authentication for approve
- [ ] Add confirmation prompts for destructive operations

**Benefits:**
- Full lifecycle management of agent keys
- Support for key rotation without re-registration
- Enhanced security through key revocation

**Estimated Effort:** 40-60 minutes

**Files to Create:**
- `cmd/sage-did/key.go`

**Files to Modify:**
- `pkg/agent/did/ethereum/client.go` (add key management methods)

---

### 3. A2A Integration Examples (Priority: MEDIUM)

**Description:** Provide working examples demonstrating A2A protocol usage

**Proposed Structure:**
```
examples/a2a-integration/
├── README.md              # Setup and usage instructions
├── 01-register-agent/
│   ├── main.go           # Register multi-key agent
│   └── README.md
├── 02-generate-card/
│   ├── main.go           # Generate and export A2A card
│   └── README.md
├── 03-exchange-cards/
│   ├── main.go           # Agent-to-agent card exchange
│   └── README.md
└── 04-secure-message/
    ├── main.go           # End-to-end encrypted messaging
    └── README.md
```

**Example Content:**
- Multi-key agent registration
- A2A Agent Card generation and validation
- Card discovery and verification
- Establishing secure channels between agents
- Message signing with multiple key types

**Implementation Tasks:**
- [ ] Create example directory structure
- [ ] Write 01: Agent registration with multiple keys
- [ ] Write 02: A2A card generation and export to JSON
- [ ] Write 03: Card exchange and verification flow
- [ ] Write 04: Encrypted message exchange using session keys
- [ ] Add comprehensive README with prerequisites
- [ ] Create test script to verify all examples work

**Benefits:**
- Faster developer onboarding
- Reference implementation for integrators
- Test coverage for A2A workflows

**Estimated Effort:** 50-70 minutes

**Files to Create:**
- `examples/a2a-integration/` (multiple files)

---

### 4. Smart Contract Integration (Priority: HIGH)

**Description:** Deploy SageRegistryV4 and integrate with Go SDK

**Tasks:**

#### 4.1 Contract Deployment
- [ ] Create Hardhat deployment script
- [ ] Deploy to local hardhat network
- [ ] Deploy to Sepolia testnet
- [ ] Document contract addresses
- [ ] Verify on Etherscan

**Files to Create:**
- `contracts/ethereum/scripts/deploy_v4.js`
- `contracts/ethereum/scripts/verify_v4.js`
- `contracts/DEPLOYED_ADDRESSES.md`

#### 4.2 Go SDK Integration
- [ ] Generate Go bindings for SageRegistryV4
- [ ] Update `pkg/agent/did/ethereum/client.go` to use V4 contract
- [ ] Implement multi-key registration flow
- [ ] Add key management method wrappers
- [ ] Handle Ed25519 approval workflow

**Files to Modify:**
- `pkg/agent/did/ethereum/bindings.go` (regenerate)
- `pkg/agent/did/ethereum/client.go`
- `pkg/agent/did/manager.go`

#### 4.3 Integration Testing
- [ ] Test multi-key registration end-to-end
- [ ] Test key addition to existing agent
- [ ] Test key revocation
- [ ] Test A2A card generation from deployed contract
- [ ] Measure gas costs for various operations

**Files to Create:**
- `pkg/agent/did/ethereum/client_v4_test.go`
- `tests/integration/multi_key_test.go`

**Benefits:**
- Production-ready multi-key system
- Real blockchain interaction
- Gas cost optimization data
- Full integration test coverage

**Estimated Effort:** 90-120 minutes

**Dependencies:** Requires local blockchain or testnet access

---

### 5. Enhanced Validation (Priority: MEDIUM)

**Description:** Strengthen security with comprehensive validation

**Features:**

#### 5.1 A2A Card Signature Verification
```go
// Verify that the A2A card is signed by the DID controller
func VerifyA2ACardSignature(card *A2AAgentCard, signature []byte) error
```

**Implementation:**
- [ ] Add signature field to A2AAgentCard
- [ ] Implement card signing with agent's private key
- [ ] Add verification function using publicKey from card
- [ ] Update GenerateA2ACard to include optional signing
- [ ] Add signature validation to ValidateA2ACard

#### 5.2 DID Document Cross-Check
```go
// Verify A2A card matches on-chain DID document
func CrossCheckDIDDocument(did string, card *A2AAgentCard) error
```

**Implementation:**
- [ ] Resolve DID from blockchain
- [ ] Compare public keys
- [ ] Compare endpoints
- [ ] Compare capabilities
- [ ] Report discrepancies

#### 5.3 Key Proof-of-Possession
```go
// Verify key ownership through challenge-response
func VerifyKeyPossession(keyData []byte, challenge []byte, response []byte) error
```

**Implementation:**
- [ ] Generate random challenge
- [ ] Agent signs challenge with each key
- [ ] Verify signatures match public keys
- [ ] Integration with registration flow

**Implementation Tasks:**
- [ ] Add signature support to A2AAgentCard struct
- [ ] Implement card signing and verification functions
- [ ] Add DID document comparison logic
- [ ] Create challenge-response mechanism
- [ ] Add comprehensive tests for all validation

**Benefits:**
- Prevents card tampering
- Ensures DID authenticity
- Confirms key ownership
- Strengthens trust model

**Estimated Effort:** 40-60 minutes

**Files to Modify:**
- `pkg/agent/did/types_v4.go`
- `pkg/agent/did/a2a.go`
- `pkg/agent/did/verification.go`

**Files to Create:**
- `pkg/agent/did/a2a_validation.go`
- `pkg/agent/did/a2a_validation_test.go`

---

## Implementation Priority

### Phase 1 (Essential)
1. Multi-Key Registration Support
2. Key Management CLI Commands
3. Smart Contract Integration

### Phase 2 (Important)
4. A2A Integration Examples
5. Enhanced Validation

### Phase 3 (Nice to Have)
- Performance benchmarks
- Gas optimization
- Multi-chain deployment (Polygon, Avalanche, etc.)
- GraphQL API for agent discovery

---

## Success Metrics

- [ ] All multi-key agents can be registered with single transaction
- [ ] Keys can be rotated without disrupting agent identity
- [ ] A2A cards can be exchanged between agents
- [ ] Integration examples run successfully on first try
- [ ] Gas costs are documented and optimized
- [ ] 100% feature test pass rate maintained
- [ ] Test coverage remains >75%

---

## Notes

- All features should maintain backward compatibility with legacy single-key agents
- Each feature should include comprehensive tests before merge
- Documentation should be updated alongside code changes
- Security considerations should be documented for each feature

---

**Last Updated:** 2025-01-18
**Document Owner:** SAGE Development Team
**Related Documents:**
- `contracts/MULTI_KEY_DESIGN.md` - Design specification
- `docs/test/FEATURE_TEST_GUIDE_KR.md` - Test requirements
- `README.md` - Project overview
