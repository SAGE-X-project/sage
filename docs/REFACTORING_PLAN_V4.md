# SAGE Core Code Refactoring Plan for AgentCardRegistry Integration
## Version 1.4.0 → 1.5.0

**Date**: October 27, 2025
**Status**: Planning Phase
**Target**: Integrate new AgentCardRegistry (Commit-Reveal + Multi-Key + Governance)

---

## 📋 Executive Summary

### Current State
The Go client (`clientv4.go`) interfaces with `SageRegistryV4` which uses:
- ✅ Multi-key support (ECDSA, Ed25519, X25519)
- ✅ Simple one-call registration
- ✅ Update/nonce mechanisms
- ❌ **No commit-reveal pattern**
- ❌ **No time-locked activation**
- ❌ **No stake requirement**
- ❌ **No governance integration**

### Target State
The new `AgentCardRegistry` provides:
- ✅ **Three-phase registration** (commit → register → activate)
- ✅ **Anti-front-running** via commit-reveal (1 min - 1 hour)
- ✅ **Time-locked activation** (1 hour minimum)
- ✅ **Stake requirement** (0.01 ETH)
- ✅ **Operator approval system** (ERC-721 style)
- ✅ **Hook system** for external validation
- ✅ **Rate limiting** (24 registrations/address/day)
- ✅ **Governance integration** (TEEKeyRegistry, MultiSig, Timelock)

---

## 🚨 Critical Differences & Breaking Changes

### 1. Registration Flow Change

#### Current (SageRegistryV4)
```
Go Client → registerAgent() → ✅ Immediately active
```

#### New (AgentCardRegistry)
```
Go Client → commitRegistration()     [+0.01 ETH stake]
          ↓ Wait 1-60 minutes
          → registerAgentWithParams() [Reveal commitment]
          ↓ Wait 1+ hour
          → activateAgent()           [Agent becomes active]
```

### 2. KeyType Enum Mismatch

| Key Type | ISageRegistryV4 | AgentCardStorage | **Action Required** |
|----------|-----------------|------------------|---------------------|
| ECDSA    | 1               | 0                | ✅ Update mapping   |
| Ed25519  | 0               | 1                | ✅ Update mapping   |
| X25519   | N/A             | 2                | ✅ Add support      |

**Impact**: All key type conversions must be updated to match AgentCardStorage.

### 3. New Data Structures

#### RegistrationParams (commit-reveal)
```go
type RegistrationParams struct {
    DID          string
    Name         string
    Description  string
    Endpoint     string
    Capabilities string
    Keys         [][]byte  // Public key bytes
    KeyTypes     []uint8   // KeyType enum values
    Signatures   [][]byte  // Ownership proofs
    Salt         [32]byte  // For commit-reveal
}
```

#### AgentMetadata (new field: chainId)
```go
type AgentMetadata struct {
    // ... existing fields ...
    ChainID *big.Int  // NEW: Cross-chain replay protection
}
```

### 4. Stake & Gas Requirements

| Operation            | Current Cost | New Cost        | Notes                           |
|----------------------|--------------|-----------------|----------------------------------|
| Register             | ~620k gas    | 0.01 ETH + gas  | Stake refunded on activation    |
| Commit               | N/A          | ~50k gas        | Phase 1: Store commitment       |
| RegisterWithParams   | N/A          | ~650k gas       | Phase 2: Verify & register      |
| Activate             | N/A          | ~30k gas        | Phase 3: Activate after delay   |
| **Total**            | ~620k gas    | **0.01 ETH + ~730k gas** | Split across 3 transactions |

---

## 🏗️ Architecture Components to Modify

### Phase 1: Contract Bindings & Types (Week 1)

#### 1.1 Generate New Go Bindings
**Files to Create/Update**:
- `pkg/blockchain/ethereum/contracts/agentcardregistry/` (new package)
  - `AgentCardRegistry.go` - Generated via abigen
  - `AgentCardStorage.go` - Storage structs
  - `IRegistryHook.go` - Hook interface

**Command**:
```bash
cd contracts/ethereum
npm run compile
abigen --abi artifacts/contracts/AgentCardRegistry.sol/AgentCardRegistry.json \
       --pkg agentcardregistry \
       --out ../../pkg/blockchain/ethereum/contracts/agentcardregistry/AgentCardRegistry.go
```

**Verification**:
```bash
go test ./pkg/blockchain/ethereum/contracts/agentcardregistry -run TestBindings
```

#### 1.2 Update Type Mappings
**File**: `pkg/agent/did/types.go`

**Changes**:
```go
// KeyType enum (CRITICAL FIX)
const (
    KeyTypeECDSA   KeyType = 0  // Changed from 1
    KeyTypeEd25519 KeyType = 1  // Changed from 0
    KeyTypeX25519  KeyType = 2  // NEW
)

// RegistrationParams (new struct)
type RegistrationParams struct {
    DID          string
    Name         string
    Description  string
    Endpoint     string
    Capabilities string
    Keys         [][]byte
    KeyTypes     []KeyType
    Signatures   [][]byte
    Salt         [32]byte
}

// RegistrationStatus (new type for tracking)
type RegistrationStatus string

const (
    StatusCommitted  RegistrationStatus = "committed"
    StatusRegistered RegistrationStatus = "registered"
    StatusActivated  RegistrationStatus = "activated"
)
```

---

### Phase 2: Client Implementation (Week 2-3)

#### 2.1 Create AgentCardRegistry Client
**File**: `pkg/agent/did/ethereum/client_agentcard.go` (new file)

**Structure**:
```go
type AgentCardClient struct {
    client          *ethclient.Client
    contract        *agentcardregistry.AgentCardRegistry
    contractAddress common.Address
    privateKey      *ecdsa.PrivateKey
    chainID         *big.Int
    config          *did.RegistryConfig

    // Commitment tracking
    pendingCommitments map[common.Address]*CommitmentState
}

type CommitmentState struct {
    CommitHash     [32]byte
    Params         *RegistrationParams
    Timestamp      time.Time
    CommitTxHash   common.Hash
    RegisterTxHash common.Hash
    Status         RegistrationStatus
}
```

**Core Methods to Implement**:

##### 2.1.1 Three-Phase Registration
```go
// Phase 1: Commit
func (c *AgentCardClient) CommitRegistration(
    ctx context.Context,
    params *RegistrationParams,
) (commitHash [32]byte, txHash common.Hash, err error)

// Phase 2: Register (after 1-60 min delay)
func (c *AgentCardClient) RegisterAgent(
    ctx context.Context,
    commitHash [32]byte,
    params *RegistrationParams,
) (agentID [32]byte, txHash common.Hash, err error)

// Phase 3: Activate (after 1+ hour delay)
func (c *AgentCardClient) ActivateAgent(
    ctx context.Context,
    agentID [32]byte,
) (txHash common.Hash, err error)

// Convenience: Full registration flow
func (c *AgentCardClient) RegisterAgentFull(
    ctx context.Context,
    params *RegistrationParams,
) (agentID [32]byte, err error) {
    // 1. Commit
    commitHash, _, err := c.CommitRegistration(ctx, params)

    // 2. Wait minimum delay (1 minute)
    time.Sleep(61 * time.Second)

    // 3. Register
    agentID, _, err = c.RegisterAgent(ctx, commitHash, params)

    // 4. Wait activation delay (1 hour)
    time.Sleep(61 * time.Minute)

    // 5. Activate
    _, err = c.ActivateAgent(ctx, agentID)

    return agentID, err
}
```

##### 2.1.2 Commitment Hash Calculation
```go
func (c *AgentCardClient) ComputeCommitHash(
    params *RegistrationParams,
    owner common.Address,
) ([32]byte, error) {
    // Must match Solidity: keccak256(abi.encode(did, keys, owner, salt, chainId))

    // 1. Encode keys
    var keysEncoded []byte
    for _, key := range params.Keys {
        keyHash := crypto.Keccak256Hash(key)
        keysEncoded = append(keysEncoded, keyHash[:]...)
    }

    // 2. Build encoding
    encoded := []byte{}
    encoded = append(encoded, []byte(params.DID)...)
    encoded = append(encoded, keysEncoded...)
    encoded = append(encoded, owner.Bytes()...)
    encoded = append(encoded, params.Salt[:]...)
    encoded = append(encoded, c.chainID.Bytes()...)

    // 3. Hash
    return crypto.Keccak256Hash(encoded), nil
}
```

##### 2.1.3 Operator Management (NEW)
```go
// Approve operator to manage agent on behalf of owner
func (c *AgentCardClient) ApproveOperator(
    ctx context.Context,
    agentID [32]byte,
    operator common.Address,
) (txHash common.Hash, err error)

// Revoke operator approval
func (c *AgentCardClient) RevokeOperator(
    ctx context.Context,
    agentID [32]byte,
    operator common.Address,
) (txHash common.Hash, err error)

// Check if address is approved operator
func (c *AgentCardClient) IsApprovedOperator(
    ctx context.Context,
    agentID [32]byte,
    operator common.Address,
) (bool, error)
```

##### 2.1.4 Hook Management
```go
// Set verification hook (owner only)
func (c *AgentCardClient) SetVerifyHook(
    ctx context.Context,
    hookAddress common.Address,
) (txHash common.Hash, err error)

// Query current hook
func (c *AgentCardClient) GetVerifyHook(
    ctx context.Context,
) (common.Address, error)
```

#### 2.2 Update Existing ClientV4
**File**: `pkg/agent/did/ethereum/clientv4.go`

**Strategy**: Keep clientv4.go for backward compatibility, but mark as deprecated

```go
// DEPRECATED: Use AgentCardClient for new registrations with commit-reveal.
// This client interfaces with SageRegistryV4 which does not support:
// - Commit-reveal pattern (vulnerable to front-running)
// - Time-locked activation (no delay enforcement)
// - Stake requirements (no economic security)
// - Operator system (no delegated management)
//
// Migrate to AgentCardClient before SageRegistryV4 is phased out.
type EthereumClientV4 struct {
    // ... existing implementation ...
}
```

#### 2.3 Create Migration Guide
**File**: `pkg/agent/did/ethereum/MIGRATION_V4_TO_AGENTCARD.md`

**Content**: Step-by-step guide for migrating from V4 to AgentCard client

---

### Phase 3: CLI Tools Update (Week 4)

#### 3.1 Update sage-did CLI
**File**: `cmd/sage-did/main.go`

**New Commands**:
```bash
# Three-phase registration
sage-did commit --params params.json --stake 0.01
sage-did register --commit-hash 0x... --params params.json
sage-did activate --agent-id 0x...

# Full registration (wait automatically)
sage-did register-full --params params.json --wait

# Operator management
sage-did approve-operator --agent-id 0x... --operator 0x...
sage-did revoke-operator --agent-id 0x... --operator 0x...

# Query
sage-did get-commitment --address 0x...
sage-did get-activation-time --agent-id 0x...
```

**Implementation**:
- Add `--registry-type` flag: `v4` | `agentcard` (default: `agentcard`)
- Add `--wait` flag for automatic delay handling
- Add progress indicators for time-locked operations
- Store commitment state in local cache (~/.sage/commitments.json)

#### 3.2 Add Validation Helpers
**File**: `cmd/sage-did/validate.go` (new)

```go
// Validate registration params before commit
func ValidateRegistrationParams(params *RegistrationParams) error {
    // 1. DID format
    if !strings.HasPrefix(params.DID, "did:sage:") {
        return errors.New("invalid DID format")
    }

    // 2. Key limits
    if len(params.Keys) == 0 || len(params.Keys) > 10 {
        return errors.New("must provide 1-10 keys")
    }

    // 3. Key types match
    if len(params.Keys) != len(params.KeyTypes) {
        return errors.New("keys and keyTypes length mismatch")
    }

    // 4. Signature requirements (ECDSA only)
    ecdsaCount := 0
    for _, kt := range params.KeyTypes {
        if kt == KeyTypeECDSA {
            ecdsaCount++
        }
    }
    if ecdsaCount > 0 && len(params.Signatures) == 0 {
        return errors.New("ECDSA keys require signatures")
    }

    return nil
}
```

---

### Phase 4: Testing & Integration (Week 5-6)

#### 4.1 Unit Tests
**Files to Create**:
- `pkg/agent/did/ethereum/client_agentcard_test.go`
- `pkg/agent/did/ethereum/commitment_test.go`
- `pkg/agent/did/ethereum/operator_test.go`
- `pkg/agent/did/ethereum/hook_test.go`

**Test Coverage**:
- ✅ Commit hash calculation matches Solidity
- ✅ Three-phase registration flow
- ✅ Delay enforcement (min 1 min, max 1 hour for commit)
- ✅ Stake handling (send ETH, receive refund)
- ✅ Operator approval/revocation
- ✅ Hook integration
- ✅ KeyType enum conversion correctness
- ✅ Error handling for all failure modes

#### 4.2 Integration Tests
**File**: `tests/integration/agentcard_registration_test.go` (new)

**Scenarios**:
```go
func TestAgentCardRegistration_FullFlow(t *testing.T) {
    // Setup local hardhat node with AgentCardRegistry
    // Test full commit → register → activate flow
}

func TestAgentCardRegistration_FrontRunningPrevention(t *testing.T) {
    // Verify commit-reveal prevents front-running
}

func TestAgentCardRegistration_TimeDelays(t *testing.T) {
    // Test too-early registration rejection
    // Test too-late commitment expiration
    // Test activation before 1-hour delay
}

func TestAgentCardRegistration_StakeRefund(t *testing.T) {
    // Verify stake is refunded on activation
}

func TestAgentCardRegistration_RateLimit(t *testing.T) {
    // Test 24 registrations/day limit
}

func TestOperatorSystem(t *testing.T) {
    // Test operator approval
    // Test operator can register on behalf of owner
    // Test operator cannot transfer ownership
}
```

#### 4.3 E2E Tests
**File**: `tests/integration/agentcard_e2e_test.go` (new)

**Full System Tests**:
- Deploy AgentCardRegistry, Hook, Governance contracts
- Register agent with commit-reveal
- Update agent metadata (with operator)
- Deactivate agent
- Governance: Approve Ed25519 keys via TEEKeyRegistry
- Governance: Update registry via MultiSig + Timelock

---

### Phase 5: Documentation & Examples (Week 7)

#### 5.1 Update Documentation
**Files to Update**:
- `README.md` - Add AgentCardRegistry section
- `docs/ARCHITECTURE.md` - Update with commit-reveal design
- `docs/API.md` - Document new client methods
- `contracts/README.md` - Add AgentCardRegistry details

**New Files**:
- `docs/COMMIT_REVEAL_PATTERN.md` - Explain anti-front-running
- `docs/OPERATOR_SYSTEM.md` - Operator approval guide
- `docs/GOVERNANCE_INTEGRATION.md` - TEE key approval workflow

#### 5.2 Create Examples
**Directory**: `examples/agentcard-registration/`

**Files**:
```
examples/agentcard-registration/
├── basic/                 # Simple three-phase registration
│   └── main.go
├── operator/              # Operator delegation example
│   └── main.go
├── governance/            # TEE key approval via governance
│   └── main.go
└── migration/             # Migrate from V4 to AgentCard
    └── main.go
```

#### 5.3 Migration Guide
**File**: `docs/MIGRATION_V4_TO_V5.md`

**Sections**:
1. Why Migrate? (Security benefits)
2. Breaking Changes Summary
3. Step-by-Step Migration
4. Code Examples (before/after)
5. Testing Checklist
6. Rollback Plan

---

## 📊 Implementation Priority Matrix

### P0: Critical (Must Have for v1.5.0)
- ✅ Generate AgentCardRegistry Go bindings
- ✅ Fix KeyType enum mismatch
- ✅ Implement three-phase registration flow
- ✅ Update sage-did CLI for commit-reveal
- ✅ Add commitment state tracking
- ✅ Unit tests for core registration

### P1: High (Should Have for v1.5.0)
- ✅ Operator system implementation
- ✅ Hook management functions
- ✅ Integration tests (full flow)
- ✅ Stake handling & refund verification
- ✅ Rate limiting support
- ✅ Migration guide documentation

### P2: Medium (Nice to Have)
- 🔄 Governance integration (TEEKeyRegistry client)
- 🔄 E2E tests with governance
- 🔄 Advanced examples (operator, governance)
- 🔄 Performance benchmarks

### P3: Low (Future Enhancement)
- 📋 Multi-chain support (Solana, Cosmos)
- 📋 Gas optimization analysis
- 📋 Circuit breaker implementation
- 📋 Automated commitment retry logic

---

## 🚧 Risk Assessment

### High Risk
1. **KeyType Enum Mismatch**
   - **Impact**: Incorrect key type registration (ECDSA registered as Ed25519)
   - **Mitigation**: Extensive testing, update all type conversions
   - **Test**: Compare on-chain stored KeyType with Go client expectations

2. **Commitment Hash Calculation**
   - **Impact**: Registration fails if hash doesn't match Solidity
   - **Mitigation**: Match exact Solidity encoding (abi.encode)
   - **Test**: Cross-verify with Solidity test vectors

3. **Time Delay Handling**
   - **Impact**: User frustration if delays not handled properly
   - **Mitigation**: Clear UX messaging, automatic retries, progress indicators
   - **Test**: Simulate network delays, clock skew

### Medium Risk
4. **Stake Management**
   - **Impact**: Loss of funds if stake not refunded
   - **Mitigation**: Monitor stake refund in activation transaction
   - **Test**: Verify balance changes in all scenarios

5. **Backward Compatibility**
   - **Impact**: Breaking existing V4 integrations
   - **Mitigation**: Keep V4 client, provide migration path
   - **Test**: Run existing V4 tests against new codebase

### Low Risk
6. **Operator System Complexity**
   - **Impact**: Confusion about operator vs owner permissions
   - **Mitigation**: Clear documentation, example code
   - **Test**: Permission boundary tests

---

## 📅 Timeline & Milestones

### Week 1: Foundations (Nov 3-9, 2025)
- [ ] Generate contract bindings
- [ ] Fix KeyType enum mismatch
- [ ] Update type definitions
- [ ] Create AgentCardClient skeleton

### Week 2: Core Implementation (Nov 10-16, 2025)
- [ ] Implement three-phase registration
- [ ] Commitment hash calculation
- [ ] Stake handling
- [ ] Basic unit tests

### Week 3: Advanced Features (Nov 17-23, 2025)
- [ ] Operator system
- [ ] Hook management
- [ ] Rate limiting support
- [ ] Integration tests

### Week 4: CLI & Tooling (Nov 24-30, 2025)
- [ ] Update sage-did CLI
- [ ] Add validation helpers
- [ ] Commitment state tracking
- [ ] CLI tests

### Week 5: Testing (Dec 1-7, 2025)
- [ ] Full integration test suite
- [ ] E2E tests
- [ ] Performance benchmarks
- [ ] Bug fixes

### Week 6: Polish & QA (Dec 8-14, 2025)
- [ ] Documentation review
- [ ] Code review
- [ ] Security audit prep
- [ ] Final testing

### Week 7: Release (Dec 15-21, 2025)
- [ ] Documentation finalization
- [ ] Example code
- [ ] Migration guide
- [ ] Release v1.5.0

---

## 🔧 Development Tools & Scripts

### Binding Generation Script
**File**: `scripts/generate-agentcard-bindings.sh`

```bash
#!/bin/bash
set -e

echo "Generating AgentCardRegistry Go bindings..."

cd contracts/ethereum
npm run compile

# AgentCardRegistry
abigen --abi artifacts/contracts/AgentCardRegistry.sol/AgentCardRegistry.json \
       --pkg agentcardregistry \
       --out ../../pkg/blockchain/ethereum/contracts/agentcardregistry/AgentCardRegistry.go

# AgentCardVerifyHook
abigen --abi artifacts/contracts/AgentCardVerifyHook.sol/AgentCardVerifyHook.json \
       --pkg agentcardregistry \
       --out ../../pkg/blockchain/ethereum/contracts/agentcardregistry/AgentCardVerifyHook.go

# TEEKeyRegistry
abigen --abi artifacts/contracts/governance/TEEKeyRegistry.sol/TEEKeyRegistry.json \
       --pkg governance \
       --out ../../pkg/blockchain/ethereum/contracts/governance/TEEKeyRegistry.go

echo "✅ Bindings generated successfully"
```

### Testing Helper
**File**: `scripts/test-agentcard-flow.sh`

```bash
#!/bin/bash
# Test full AgentCardRegistry flow with local hardhat node

set -e

echo "Starting local hardhat node..."
cd contracts/ethereum
npx hardhat node &
HARDHAT_PID=$!
sleep 5

echo "Deploying contracts..."
npx hardhat run scripts/deploy-agentcard.js --network localhost

echo "Running Go integration tests..."
cd ../..
go test ./tests/integration/agentcard_registration_test.go -v

echo "Cleanup..."
kill $HARDHAT_PID

echo "✅ Tests complete"
```

---

## 📝 Code Review Checklist

### Before PR Submission
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] KeyType enum conversions verified
- [ ] Commitment hash matches Solidity
- [ ] Stake handling tested
- [ ] Time delays enforced correctly
- [ ] Operator permissions tested
- [ ] Documentation updated
- [ ] Examples work end-to-end
- [ ] Migration guide reviewed
- [ ] No breaking changes to V4 client
- [ ] Security considerations documented

### Security Review
- [ ] Commitment hash calculation reviewed
- [ ] Private key handling secure
- [ ] Stake refund logic verified
- [ ] Reentrancy protection (use client-side mutexes)
- [ ] Rate limiting tested
- [ ] Operator permission boundaries tested
- [ ] Front-running prevention validated

---

## 🎯 Success Criteria

### Technical Goals
- ✅ 100% unit test coverage for AgentCardClient
- ✅ All integration tests passing
- ✅ Zero breaking changes to existing V4 client
- ✅ Commitment hash matches Solidity 100%
- ✅ Operator system fully functional
- ✅ CLI supports full registration flow

### User Experience Goals
- ✅ Clear progress indicators during time-locked operations
- ✅ Helpful error messages for common failure modes
- ✅ Migration from V4 takes < 1 hour for typical project
- ✅ Documentation explains commit-reveal benefits clearly

### Performance Goals
- ✅ Three-phase registration completes in < 90 minutes (including waits)
- ✅ Commitment state tracking has < 10ms lookup time
- ✅ CLI responds in < 2 seconds for all non-blockchain operations

---

## 🔗 References

### Contracts
- `contracts/ethereum/contracts/AgentCardRegistry.sol`
- `contracts/ethereum/contracts/AgentCardStorage.sol`
- `contracts/ethereum/contracts/AgentCardVerifyHook.sol`
- `contracts/ethereum/contracts/governance/TEEKeyRegistry.sol`

### Go Packages
- `pkg/agent/did/ethereum/clientv4.go` (existing)
- `pkg/agent/did/ethereum/client_agentcard.go` (new)
- `pkg/agent/did/types.go` (update)

### Documentation
- `SOLIDITY_CONTRACTS_ANALYSIS.md` (analysis output)
- `docs/ARCHITECTURE.md` (update)
- `docs/MIGRATION_V4_TO_V5.md` (new)

### Testing
- `tests/integration/agentcard_registration_test.go` (new)
- `tests/integration/agentcard_e2e_test.go` (new)

---

## 📞 Contact & Support

**Project Lead**: SAGE Team
**Target Release**: v1.5.0 (December 2025)
**GitHub Issues**: https://github.com/SAGE-X-project/sage/issues
**Discussions**: https://github.com/SAGE-X-project/sage/discussions

---

**Last Updated**: October 27, 2025
**Next Review**: November 3, 2025 (Week 1 kickoff)
