# Multi-Key Registry V4 - Feature Roadmap

This document outlines planned features and enhancements for the SageRegistryV4 multi-key agent system.

## Status: Phase 1 Complete, Phase 2 In Progress

**Current Branch:** `feature/multi-key-cli`

**Completed Features (Phase 1):**
- ✅ SageRegistryV4 smart contract with multi-key support
- ✅ Unit tests for smart contract (30 tests, 100% pass)
- ✅ V4 types in Go backend (AgentMetadataV4, AgentKey)
- ✅ A2A Agent Card generation and validation
- ✅ Unit tests for Go backend (37+ tests, 77.6% coverage)
- ✅ CLI commands: `sage-did card generate/validate/show`
- ✅ All 85 feature tests passing
- ✅ Multi-key registration CLI with auto-detection (Phase 1.1)
- ✅ Key management CLI commands: add/list/revoke/approve (Phase 1.2)
- ✅ Contract deployment automation (deploy_v4.js, verify_v4.js)
- ✅ Go SDK V4 integration with factory pattern (Phase 2.2)
- ✅ Comprehensive integration tests (TestV4AddKey, TestV4RevokeKey, TestV4ApproveEd25519Key)

**Current Work (Phase 2):**
- 🚧 A2A Integration Examples

---

## Planned Features

### 1. Multi-Key Registration Support (Priority: HIGH) ✅ COMPLETED

**Description:** Enable simultaneous registration of multiple key types through CLI

**Status:** ✅ Completed in Phase 1.1

**Implementation:**
```bash
# Register with multiple keys
sage-did register \
  --chain ethereum \
  --name "Multi-Key Agent" \
  --additional-keys ed25519.jwk,x25519.key \
  --endpoint https://agent.example.com
```

**Implementation Tasks:**
- [x] Extend CLI to accept multiple key files via `--additional-keys`
- [x] Support key type auto-detection from file format (.jwk, .pem, .ed25519, .x25519)
- [x] Generate proper signatures for each key type
- [x] Call SageRegistryV4.registerAgent with multiple keys
- [x] Handle Ed25519 pre-approval flow notification
- [x] Update help documentation with examples

**Completed Features:**
- Auto-detection from file extensions and content
- JWK and PEM parsing support
- Multiple key format handling
- Ed25519 approval workflow notices

**Benefits Achieved:**
- ✅ Enables true multi-chain agent identity
- ✅ Single transaction for all keys
- ✅ Immediate A2A protocol compatibility

**Actual Effort:** ~90 minutes

**Files Modified:**
- `cmd/sage-did/register.go` (+309 lines)

**Completed:** 2025-01-19 (Commit: d6282b0)

---

### 2. Key Management CLI Commands (Priority: HIGH) ✅ COMPLETED

**Description:** Add commands for managing keys on registered agents

**Status:** ✅ Completed in Phase 1.2

**Implemented Commands:**
```bash
# Add a new key to existing agent
sage-did key add <did> <keyfile> \
  --chain ethereum \
  --contract-address <addr> \
  --rpc-url <url> \
  --private-key <key>

# List all keys for an agent
sage-did key list <did> \
  --chain ethereum \
  --contract-address <addr> \
  --rpc-url <url>

# Revoke a key
sage-did key revoke <did> <keyhash> \
  --chain ethereum \
  --private-key <owner-key>

# Approve Ed25519 key (owner only)
sage-did key approve <keyhash> \
  --chain ethereum \
  --private-key <registry-owner-key>
```

**Implementation Tasks:**
- [x] Create `cmd/sage-did/key.go` with 4 subcommands
- [x] Implement AddKey, RevokeKey, ApproveEd25519Key in clientv4.go
- [x] Add RegistryV4 interface with key management methods
- [x] Add manager methods: AddKey, RevokeKey, ApproveEd25519Key
- [x] Automatic signature generation for ECDSA keys
- [x] Key type auto-detection support

**Completed Features:**
- Auto-detection of key types (optional --key-type flag)
- Automatic ECDSA signature generation
- Ed25519 approval workflow
- Formatted key listing output

**Benefits Achieved:**
- ✅ Full lifecycle management of agent keys
- ✅ Support for key rotation without re-registration
- ✅ Enhanced security through key revocation

**Actual Effort:** ~60 minutes

**Files Created:**
- `cmd/sage-did/key.go` (+400 lines)

**Files Modified:**
- `pkg/agent/did/manager.go` (added AddKey, RevokeKey, ApproveEd25519Key)
- `pkg/agent/did/registry.go` (added RegistryV4 interface)
- `pkg/agent/did/ethereum/clientv4.go` (+170 lines for key operations)

**Completed:** 2025-01-19 (Commit: c65b509)

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

### 4. Smart Contract Integration (Priority: HIGH) ✅ COMPLETED

**Description:** Deploy SageRegistryV4 and integrate with Go SDK

**Status:** ✅ Completed in Phase 2.1, 2.2, 2.3

#### 4.1 Contract Deployment ✅ COMPLETED (Phase 2.1)
- [x] Create Hardhat deployment script with gas estimation
- [x] Deploy to local hardhat network support
- [x] Deploy to Sepolia testnet support
- [x] Document contract addresses in markdown
- [x] Verify on Etherscan automation

**Files Created:**
- `contracts/ethereum/scripts/deploy_v4.js` (+223 lines)
- `contracts/ethereum/scripts/verify_v4.js` (+248 lines)
- `contracts/DEPLOYED_ADDRESSES.md` (comprehensive deployment guide)

**Features:**
- Automatic gas estimation and cost calculation
- Ownership transfer support
- Deployment info auto-saved to JSON
- Confirmation prompts for production networks
- Etherscan verification with auto-retry

**Completed:** 2025-01-19 (Commit: 956cbd3)

#### 4.2 Go SDK Integration ✅ COMPLETED (Phase 2.2)
- [x] Generate Go bindings for SageRegistryV4 (already existed)
- [x] Implement factory pattern to avoid import cycles
- [x] Auto-initialize V4 client in manager.Configure()
- [x] Implement multi-key registration flow
- [x] Add key management method wrappers
- [x] Handle Ed25519 approval workflow

**Files Modified:**
- `pkg/agent/did/manager.go` (factory pattern, auto-initialization)
- `pkg/agent/did/ethereum/clientv4.go` (init() registration)
- `cmd/sage-did/register.go` (default address documentation)

**Features:**
- EthereumV4ClientCreator factory pattern
- Automatic V4 client initialization on Configure()
- No manual SetClient() calls needed
- Clean separation of concerns

**Actual Effort:** ~40 minutes

**Completed:** 2025-01-19 (Commit: 921c121)

#### 4.3 Integration Testing ✅ COMPLETED (Phase 2.3)
- [x] Test multi-key registration end-to-end (existing tests)
- [x] Test key addition to existing agent (TestV4AddKey)
- [x] Test key revocation (TestV4RevokeKey)
- [x] Test Ed25519 approval (TestV4ApproveEd25519Key)
- [x] A2A card generation already tested in existing suite

**Files Created/Modified:**
- `pkg/agent/did/ethereum/clientv4_test.go` (+394 lines)
  - TestV4AddKey: ECDSA and Ed25519 key addition
  - TestV4RevokeKey: Complete revocation workflow
  - TestV4ApproveEd25519Key: Owner approval workflow

**Test Coverage:**
- Multi-key registration: ✅ TestV4MultiKeyRegistration
- Key addition: ✅ TestV4AddKey
- Key revocation: ✅ TestV4RevokeKey
- Ed25519 approval: ✅ TestV4ApproveEd25519Key
- Key rotation: ✅ TestV4KeyRotation
- Public key resolution: ✅ TestV4PublicKeyOwnershipVerification

**Actual Effort:** ~50 minutes

**Completed:** 2025-01-19 (Commit: 8023f54)

**Benefits Achieved:**
- ✅ Production-ready multi-key system
- ✅ Real blockchain interaction tested
- ✅ Full integration test coverage
- ⏳ Gas cost optimization data (pending analysis)

**Total Effort:** ~110 minutes

**Dependencies Met:** Local blockchain testing implemented

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

### Phase 1 (Essential) ✅ COMPLETED
1. ✅ Multi-Key Registration Support (Completed: 2025-01-19)
2. ✅ Key Management CLI Commands (Completed: 2025-01-19)
3. ✅ Smart Contract Integration (Completed: 2025-01-19)
   - ✅ 4.1 Contract Deployment
   - ✅ 4.2 Go SDK Integration
   - ✅ 4.3 Integration Testing

**Total Implementation Time:** ~290 minutes
**Branch:** `feature/multi-key-cli`
**Commits:** d6282b0, c65b509, 956cbd3, 921c121, 8023f54

### Phase 2 (Important) 🚧 IN PROGRESS
4. A2A Integration Examples (Next)
5. Enhanced Validation

### Phase 3 (Nice to Have)
- Performance benchmarks
- Gas optimization analysis
- Multi-chain deployment (Polygon, Avalanche, etc.)
- GraphQL API for agent discovery

---

## Success Metrics

- [x] All multi-key agents can be registered with single transaction ✅
- [x] Keys can be rotated without disrupting agent identity ✅
- [x] A2A cards can be generated from multi-key agents ✅
- [ ] Integration examples run successfully on first try (In Progress)
- [ ] Gas costs are documented and optimized (Pending analysis)
- [x] 100% feature test pass rate maintained ✅
- [x] Test coverage remains >75% ✅ (77.6% current)

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
