# SAGE Contract Implementation Plan

**Purpose**: Comprehensive implementation plan for final production contracts with TDD approach

**Date**: 2025-10-26

**Based On**:
- [REGISTRY_EVOLUTION_ANALYSIS.md](./REGISTRY_EVOLUTION_ANALYSIS.md)
- [ERC8004_COMPLIANCE_GAP_ANALYSIS.md](./ERC8004_COMPLIANCE_GAP_ANALYSIS.md)
- [FINAL_ARCHITECTURE_DESIGN.md](./FINAL_ARCHITECTURE_DESIGN.md)
- GitHub PRs: dependency upgrades (8 PRs pending)

---

## Table of Contents

1. [Overview](#overview)
2. [Current Status](#current-status)
3. [Dependency Upgrade Plan](#dependency-upgrade-plan)
4. [Implementation Phases](#implementation-phases)
5. [TDD Test Strategy](#tdd-test-strategy)
6. [Work Breakdown](#work-breakdown)
7. [Verification Strategy](#verification-strategy)
8. [Timeline](#timeline)

---

## Overview

### Goals

1. ✅ **Implement Final Production Contracts**
   - AgentCardStorage.sol
   - AgentCardRegistry.sol
   - AgentCardVerifyHook.sol
   - ERC8004IdentityRegistryV4.sol

2. ✅ **Upgrade Dependencies to Latest Stable Versions**
   - Hardhat 2.26.3 → 3.0.7
   - Solhint 3.6.2 → 6.0.1
   - Prettier 2.8.8 → 3.6.2
   - TypeChain packages → latest

3. ✅ **Achieve 100% Test Coverage**
   - Unit tests for all functions
   - Integration tests for workflows
   - Security tests (reentrancy, overflow, etc.)
   - Gas optimization tests

4. ✅ **Maintain ERC-8004 Compliance**
   - Identity Registry: DEPRECATED → PRODUCTION READY
   - Reputation Registry: Already compliant
   - Validation Registry: Already compliant

5. ✅ **Enable TDD Verification**
   - Pre-implementation test specifications
   - Test-driven development workflow
   - Automated verification checklist

### Success Criteria

- [ ] All new contracts compiled successfully
- [ ] All tests passing (unit + integration)
- [ ] Test coverage ≥ 90%
- [ ] Gas costs < 2M per registration
- [ ] ERC-8004 compliance verified
- [ ] Security audit passed (Slither + Mythril)
- [ ] All dependency upgrades applied
- [ ] Migration from V4 successful
- [ ] Documentation complete
- [ ] CLI tools updated

---

## Current Status

### Existing Contracts

| Contract | Status | Test Coverage | Production Ready |
|----------|--------|---------------|------------------|
| SageRegistryV2 | ✅ Deployed | ~80% | ⚠️ DEPRECATED |
| SageRegistryV3 | ✅ Deployed | ~75% | ⚠️ DEPRECATED |
| SageRegistryV4 | ✅ Deployed | ~85% | ⚠️ TO MIGRATE |
| ERC8004IdentityRegistry | ❌ Broken | 0% | ❌ DEPRECATED |
| ERC8004ReputationRegistry | ✅ Complete | 95% | ✅ PRODUCTION |
| ERC8004ValidationRegistry | ✅ Complete | 98% | ✅ PRODUCTION |

### New Contracts (To Be Implemented)

| Contract | Status | Priority | Estimated Hours |
|----------|--------|----------|-----------------|
| AgentCardStorage | 📝 Designed | 🔴 P0 | 8 hours |
| AgentCardRegistry | 📝 Designed | 🔴 P0 | 16 hours |
| AgentCardVerifyHook | 📝 Designed | 🔴 P0 | 8 hours |
| ERC8004IdentityRegistryV4 | 📝 Designed | 🟡 P1 | 6 hours |

**Total Estimated**: 38 hours (5 days)

---

## Dependency Upgrade Plan

### Pending GitHub PRs (8 PRs)

All these PRs are automated dependency updates from Dependabot.

#### Critical Upgrades (Breaking Changes)

1. **PR #102: Hardhat 2.26.3 → 3.0.7** 🔴 CRITICAL
   - **Breaking Changes**:
     - Hardhat 3.0 has major API changes
     - Config format may change
     - Plugin compatibility issues
   - **Action Required**:
     - Review Hardhat 3.0 migration guide
     - Test all hardhat tasks
     - Update hardhat.config.js if needed
   - **Risk Level**: HIGH
   - **Testing Priority**: P0

2. **PR #109: Solhint 3.6.2 → 6.0.1** 🔴 CRITICAL
   - **Breaking Changes**:
     - New lint rules may fail existing code
     - Rule severity changes
     - Configuration format changes
   - **Action Required**:
     - Review new lint rules
     - Fix lint errors in existing contracts
     - Update .solhint.json if needed
   - **Risk Level**: MEDIUM
   - **Testing Priority**: P0

3. **PR #105: Prettier 2.8.8 → 3.6.2** 🟡 MODERATE
   - **Breaking Changes**:
     - Formatting style changes
     - Config format updates
   - **Action Required**:
     - Reformat all code with new prettier
     - Verify no code logic changes
   - **Risk Level**: LOW
   - **Testing Priority**: P1

4. **PR #59: @nomicfoundation/hardhat-toolbox 3.0.0 → 6.1.0** 🟡 MODERATE
   - **Breaking Changes**:
     - Bundled plugin version changes
     - May require Hardhat 3.0
   - **Action Required**:
     - Review bundled plugins
     - Test all toolbox features
   - **Risk Level**: MEDIUM
   - **Testing Priority**: P0

#### Minor Upgrades (Compatible)

5. **PR #108: @typechain/ethers-v6 0.4.3 → 0.5.1**
   - **Risk Level**: LOW
   - **Testing Priority**: P2

6. **PR #58: @typechain/hardhat 8.0.3 → 9.1.0**
   - **Risk Level**: LOW
   - **Testing Priority**: P2

7. **PR #104: prettier-plugin-solidity 1.4.3 → 2.1.0**
   - **Risk Level**: LOW
   - **Testing Priority**: P2

8. **PR #106: solhint-plugin-prettier 0.0.5 → 0.1.0**
   - **Risk Level**: LOW
   - **Testing Priority**: P2

### Upgrade Strategy

**Option A: Sequential Upgrade (RECOMMENDED)**
```
Week 1: Implement new contracts with OLD dependencies
Week 2: Test thoroughly with OLD dependencies
Week 3: Upgrade dependencies ONE BY ONE
Week 4: Retest after each upgrade
```

**Option B: Bulk Upgrade (RISKY)**
```
Week 1: Merge all 8 PRs at once
Week 2-3: Fix all breaking changes
Week 4: Implement new contracts
```

**Option C: Deferred Upgrade**
```
Week 1-3: Implement new contracts with OLD dependencies
Week 4+: Upgrade dependencies separately in future release
```

**RECOMMENDATION**: **Option A (Sequential)** for safety and stability

### Upgrade Order (Sequential Approach)

**Phase 1: Low-Risk Upgrades** (Week 3, Day 1-2)
1. @types/node 24.2.1 → 24.8.1 (PR #107)
2. @typechain/ethers-v6 0.4.3 → 0.5.1 (PR #108)
3. @typechain/hardhat 8.0.3 → 9.1.0 (PR #58)

**Phase 2: Formatting Tools** (Week 3, Day 3)
4. prettier-plugin-solidity 1.4.3 → 2.1.0 (PR #104)
5. solhint-plugin-prettier 0.0.5 → 0.1.0 (PR #106)
6. prettier 2.8.8 → 3.6.2 (PR #105)
   - Run `npm run format` after upgrade
   - Commit all formatting changes

**Phase 3: Linting Tools** (Week 3, Day 4)
7. solhint 3.6.2 → 6.0.1 (PR #109)
   - Review new lint errors
   - Fix critical issues
   - Suppress non-critical warnings

**Phase 4: Critical Tools** (Week 3, Day 5)
8. hardhat 2.26.3 → 3.0.7 (PR #102)
9. @nomicfoundation/hardhat-toolbox 3.0.0 → 6.1.0 (PR #59)
   - Review Hardhat 3.0 migration guide
   - Update hardhat.config.js
   - Test ALL hardhat tasks
   - Run full test suite

---

## Implementation Phases

### Phase 0: Preparation (Week 0 - CURRENT)

**Duration**: 2 days

**Tasks**:
- [x] Analyze V2/V3/V4 evolution
- [x] Identify ERC-8004 gaps
- [x] Design final architecture
- [x] Create implementation plan
- [ ] Create verification matrix
- [ ] Create TDD test specifications

**Deliverables**:
- [x] REGISTRY_EVOLUTION_ANALYSIS.md
- [x] ERC8004_COMPLIANCE_GAP_ANALYSIS.md
- [x] FINAL_ARCHITECTURE_DESIGN.md
- [ ] IMPLEMENTATION_PLAN.md (this document)
- [ ] VERIFICATION_MATRIX.md
- [ ] TDD_TEST_SPECS.md

---

### Phase 1: Core Storage Layer (Week 1, Day 1-2)

**Duration**: 2 days (16 hours)

**Goal**: Implement `AgentCardStorage.sol` with complete test coverage

#### 1.1 Contract Implementation (8 hours)

**File**: `contracts/AgentCardStorage.sol`

**Tasks**:
1. [ ] Define all structs (AgentMetadata, AgentKey, RegistrationCommitment)
2. [ ] Define all mappings (agents, keys, commitments, nonces)
3. [ ] Define all constants (delays, limits)
4. [ ] Define all events
5. [ ] Add comprehensive NatSpec documentation
6. [ ] Solidity version: 0.8.19

**Code Structure**:
```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

abstract contract AgentCardStorage {
    // Enums
    enum KeyType { ECDSA, Ed25519, X25519 }

    // Structs (3 types)
    struct AgentMetadata { ... }
    struct AgentKey { ... }
    struct RegistrationCommitment { ... }

    // Storage mappings (10 mappings)
    mapping(bytes32 => AgentMetadata) internal agents;
    mapping(string => bytes32) internal didToAgentId;
    // ... 8 more

    // Constants (4 constants)
    uint256 internal constant COMMIT_MIN_DELAY = 1 minutes;
    // ... 3 more

    // Events (5 events)
    event AgentRegistered(...);
    // ... 4 more
}
```

**Verification**:
- [ ] Contract compiles without errors
- [ ] All structs defined correctly
- [ ] All mappings defined correctly
- [ ] All constants set correctly
- [ ] All events defined correctly
- [ ] NatSpec coverage 100%

#### 1.2 Test Implementation (8 hours)

**File**: `test/AgentCardStorage.test.js`

**Test Suites**:

1. **Struct Tests** (1 hour)
   ```javascript
   describe("AgentCardStorage - Structs", () => {
     it("Should create AgentMetadata with all fields");
     it("Should create AgentKey with all fields");
     it("Should create RegistrationCommitment with all fields");
     it("Should support all KeyType enums");
   });
   ```

2. **Mapping Tests** (2 hours)
   ```javascript
   describe("AgentCardStorage - Mappings", () => {
     it("Should store and retrieve agent by ID");
     it("Should map DID to agent ID");
     it("Should track agents by owner");
     it("Should store keys by hash");
     it("Should track commitments by address");
     it("Should manage nonces");
     it("Should track daily registration counts");
     it("Should prevent public key reuse");
   });
   ```

3. **Constant Tests** (1 hour)
   ```javascript
   describe("AgentCardStorage - Constants", () => {
     it("Should have COMMIT_MIN_DELAY = 1 minutes");
     it("Should have COMMIT_MAX_DELAY = 1 hours");
     it("Should have MAX_KEYS_PER_AGENT = 10");
     it("Should have MAX_DAILY_REGISTRATIONS = 24");
   });
   ```

4. **Event Tests** (2 hours)
   ```javascript
   describe("AgentCardStorage - Events", () => {
     it("Should emit AgentRegistered with correct params");
     it("Should emit KeyAdded with correct params");
     it("Should emit KeyRevoked with correct params");
     it("Should emit AgentUpdated with correct params");
     it("Should emit AgentDeactivated with correct params");
     it("Should emit CommitmentRecorded with correct params");
   });
   ```

5. **Gas Optimization Tests** (2 hours)
   ```javascript
   describe("AgentCardStorage - Gas Optimization", () => {
     it("Should use packed storage for structs");
     it("Should minimize storage slots");
     it("Should use internal/private for mappings");
     it("Should use constant for fixed values");
   });
   ```

**Verification**:
- [ ] All tests pass
- [ ] Test coverage ≥ 95%
- [ ] No compiler warnings
- [ ] Gas costs documented

---

### Phase 2: Verification Hook (Week 1, Day 3-4)

**Duration**: 2 days (16 hours)

**Goal**: Implement `AgentCardVerifyHook.sol` with anti-fraud features

#### 2.1 Contract Implementation (8 hours)

**File**: `contracts/AgentCardVerifyHook.sol`

**Tasks**:
1. [ ] Implement DID format validation (W3C compliance)
2. [ ] Implement rate limiting mechanism
3. [ ] Implement blacklist/whitelist system
4. [ ] Implement public key tracking
5. [ ] Implement beforeRegister hook
6. [ ] Add admin functions (onlyOwner)
7. [ ] Add comprehensive NatSpec

**Functions to Implement**:
```solidity
contract AgentCardVerifyHook is Ownable2Step {
    // Core validation
    function beforeRegister(string calldata did, address owner, bytes[] calldata keys) external view;

    // DID validation
    function _validateDIDFormat(string calldata did, address owner) internal pure;

    // Rate limiting
    function _checkRateLimit(address owner) internal view;

    // Admin functions
    function addToBlacklist(address addr) external onlyOwner;
    function removeFromBlacklist(address addr) external onlyOwner;
    function addToWhitelist(address addr) external onlyOwner;
    function removeFromWhitelist(address addr) external onlyOwner;
}
```

**Verification**:
- [ ] Contract compiles without errors
- [ ] All validation logic implemented
- [ ] Rate limiting works correctly
- [ ] Blacklist/whitelist functional
- [ ] Only owner can call admin functions

#### 2.2 Test Implementation (8 hours)

**File**: `test/AgentCardVerifyHook.test.js`

**Test Suites**:

1. **DID Validation Tests** (2 hours)
   ```javascript
   describe("AgentCardVerifyHook - DID Validation", () => {
     it("Should accept valid DID format: did:sage:ethereum:0x...");
     it("Should reject DID without 'did:' prefix");
     it("Should reject DID without 'sage' method");
     it("Should reject DID that is too short");
     it("Should validate address embedding in DID");
   });
   ```

2. **Rate Limiting Tests** (2 hours)
   ```javascript
   describe("AgentCardVerifyHook - Rate Limiting", () => {
     it("Should allow first registration immediately");
     it("Should allow up to 24 registrations per day");
     it("Should reject 25th registration in same day");
     it("Should reset count after 24 hours");
     it("Should track counts per address separately");
     it("Should bypass rate limit for whitelisted addresses");
   });
   ```

3. **Blacklist/Whitelist Tests** (2 hours)
   ```javascript
   describe("AgentCardVerifyHook - Access Control", () => {
     it("Should reject blacklisted address");
     it("Should allow whitelisted address");
     it("Should allow owner to add to blacklist");
     it("Should allow owner to remove from blacklist");
     it("Should allow owner to add to whitelist");
     it("Should allow owner to remove from whitelist");
     it("Should reject non-owner admin calls");
   });
   ```

4. **Public Key Tracking Tests** (1 hour)
   ```javascript
   describe("AgentCardVerifyHook - Key Tracking", () => {
     it("Should reject reused public key");
     it("Should track key to owner mapping");
     it("Should allow same key for same owner");
   });
   ```

5. **Integration Tests** (1 hour)
   ```javascript
   describe("AgentCardVerifyHook - Integration", () => {
     it("Should validate complete registration request");
     it("Should reject invalid registration request");
     it("Should work with AgentCardRegistry");
   });
   ```

**Verification**:
- [ ] All tests pass
- [ ] Test coverage ≥ 90%
- [ ] Edge cases covered
- [ ] Gas costs < 100k per validation

---

### Phase 3: Main Registry Logic (Week 1-2, Day 5-9)

**Duration**: 5 days (40 hours)

**Goal**: Implement `AgentCardRegistry.sol` with all features

#### 3.1 Contract Implementation (24 hours)

**File**: `contracts/AgentCardRegistry.sol`

**Sub-tasks**:

**Day 5-6: Core Registration (16 hours)**
1. [ ] Implement commit-reveal pattern (6 hours)
   - `commitRegistration(bytes32 commitHash)`
   - Commitment storage and validation
   - Timing enforcement (1 min - 1 hour)

2. [ ] Implement registerAgent function (10 hours)
   - Reveal and verify commitment
   - Validate all inputs
   - Call verify hook
   - Store agent metadata
   - Store all keys
   - Emit events
   - Handle stake

**Day 7: Key Management (8 hours)**
3. [ ] Implement addKey function (3 hours)
   - Verify ownership
   - Check max keys limit
   - Store key
   - Update agent

4. [ ] Implement revokeKey function (2 hours)
   - Check minimum keys (≥1)
   - Remove key from array (swap-and-pop)
   - Update key status

5. [ ] Implement rotateKey function (3 hours)
   - Atomic key replacement
   - Verify both old and new keys
   - Emit events

**Day 8: Agent Management (8 hours)**
6. [ ] Implement updateAgent function (2 hours)
   - Validate owner
   - Update endpoint/capabilities
   - Increment nonce

7. [ ] Implement deactivateAgent function (2 hours)
   - Set active = false
   - Handle stake return (after 30 days)

8. [ ] Implement activateAgent function (2 hours)
   - Check time-lock delay
   - Set active = true

9. [ ] Implement view functions (2 hours)
   - getAgent, getAgentByDID
   - getKey, getAgentsByOwner
   - All query functions

**Day 9: Security & Admin (8 hours)**
10. [ ] Implement security features (4 hours)
    - ReentrancyGuard on all state-changing functions
    - Pausable mechanism
    - Ownable2Step ownership
    - Stake management

11. [ ] Implement admin functions (2 hours)
    - setRegistrationStake
    - setActivationDelay
    - setVerifyHook
    - pause/unpause

12. [ ] Internal helper functions (2 hours)
    - _verifyKeyOwnership (ECDSA, Ed25519, X25519)
    - _recoverSigner
    - Error handling

**Code Structure**:
```solidity
contract AgentCardRegistry is
    AgentCardStorage,
    IAgentCardRegistry,
    Pausable,
    ReentrancyGuard,
    Ownable2Step
{
    // State variables
    AgentCardVerifyHook public verifyHook;
    uint256 public registrationStake = 0.01 ether;
    uint256 public activationDelay = 1 hours;

    // Commit-reveal functions
    function commitRegistration(bytes32 commitHash) external payable;
    function registerAgent(...) external returns (bytes32);

    // Key management
    function addKey(...) external;
    function revokeKey(...) external;
    function rotateKey(...) external;

    // Agent management
    function updateAgent(...) external;
    function deactivateAgent(...) external;
    function activateAgent(...) external;

    // View functions
    function getAgent(bytes32 agentId) external view returns (...);
    // ... more view functions

    // Admin functions
    function setRegistrationStake(uint256) external onlyOwner;
    function pause() external onlyOwner;
    function unpause() external onlyOwner;
}
```

**Verification**:
- [ ] Contract compiles without errors
- [ ] All functions implemented
- [ ] All security modifiers applied
- [ ] NatSpec coverage 100%

#### 3.2 Test Implementation (16 hours)

**File**: `test/AgentCardRegistry.test.js`

**Test Suites**:

**Day 5-6: Registration Tests (8 hours)**
1. **Commit-Reveal Tests** (3 hours)
   ```javascript
   describe("AgentCardRegistry - Commit-Reveal", () => {
     it("Should accept valid commitment with stake");
     it("Should reject commitment without stake");
     it("Should reject reveal before min delay");
     it("Should reject reveal after max delay");
     it("Should reject reveal with wrong hash");
     it("Should reject reveal with wrong salt");
     it("Should prevent front-running attacks");
   });
   ```

2. **Registration Tests** (5 hours)
   ```javascript
   describe("AgentCardRegistry - Registration", () => {
     it("Should register agent with single key");
     it("Should register agent with multiple keys (2-10)");
     it("Should register agent with ECDSA key");
     it("Should register agent with Ed25519 key");
     it("Should register agent with X25519 key");
     it("Should register agent with mixed key types");
     it("Should reject registration with 0 keys");
     it("Should reject registration with >10 keys");
     it("Should reject duplicate DID");
     it("Should reject invalid DID format");
     it("Should enforce daily registration limit (24)");
     it("Should emit AgentRegistered event");
     it("Should set agent to inactive initially");
     it("Should store stake correctly");
     it("Should call verify hook");
   });
   ```

**Day 7: Key Management Tests (4 hours)**
3. **Add Key Tests** (2 hours)
   ```javascript
   describe("AgentCardRegistry - Add Key", () => {
     it("Should add new key to agent");
     it("Should allow up to 10 keys total");
     it("Should reject 11th key");
     it("Should verify key ownership");
     it("Should emit KeyAdded event");
     it("Should prevent key reuse");
     it("Should only allow owner to add key");
   });
   ```

4. **Revoke Key Tests** (1 hour)
   ```javascript
   describe("AgentCardRegistry - Revoke Key", () => {
     it("Should revoke key successfully");
     it("Should reject revoking last key");
     it("Should emit KeyRevoked event");
     it("Should only allow owner to revoke");
     it("Should update key status to unverified");
   });
   ```

5. **Rotate Key Tests** (1 hour)
   ```javascript
   describe("AgentCardRegistry - Rotate Key", () => {
     it("Should rotate key atomically");
     it("Should maintain total key count");
     it("Should verify both keys");
     it("Should emit events for both operations");
   });
   ```

**Day 8: Management Tests (4 hours)**
6. **Update Agent Tests** (1 hour)
   ```javascript
   describe("AgentCardRegistry - Update Agent", () => {
     it("Should update endpoint");
     it("Should update capabilities");
     it("Should increment nonce");
     it("Should emit AgentUpdated event");
     it("Should only allow owner");
   });
   ```

7. **Activation Tests** (2 hours)
   ```javascript
   describe("AgentCardRegistry - Activation", () => {
     it("Should activate agent after time-lock");
     it("Should reject activation before time-lock");
     it("Should allow anyone to activate after delay");
     it("Should emit AgentActivated event");
     it("Should set active = true");
   });
   ```

8. **Deactivation Tests** (1 hour)
   ```javascript
   describe("AgentCardRegistry - Deactivation", () => {
     it("Should deactivate agent");
     it("Should return stake after 30 days");
     it("Should not return stake before 30 days");
     it("Should emit AgentDeactivated event");
   });
   ```

**Verification**:
- [ ] All tests pass
- [ ] Test coverage ≥ 90%
- [ ] All edge cases covered
- [ ] Gas costs documented

---

### Phase 4: ERC-8004 Adapter (Week 2, Day 10-11)

**Duration**: 2 days (16 hours)

**Goal**: Implement `ERC8004IdentityRegistryV4.sol` for spec compliance

#### 4.1 Contract Implementation (8 hours)

**File**: `contracts/erc-8004/ERC8004IdentityRegistryV4.sol`

**Tasks**:
1. [ ] Implement IERC8004IdentityRegistry interface
2. [ ] Wrap AgentCardRegistry functions
3. [ ] Add AgentDomain support
4. [ ] Map multi-key to primary key
5. [ ] Implement all required functions
6. [ ] Add comprehensive NatSpec

**Functions to Implement**:
```solidity
contract ERC8004IdentityRegistryV4 is IERC8004IdentityRegistry {
    AgentCardRegistry public immutable AGENT_REGISTRY;
    mapping(string => string) private agentDomains;

    function registerAgent(string calldata agentId, string calldata endpoint) external;
    function resolveAgent(string calldata agentId) external view;
    function resolveAgentByAddress(address) external view;
    function isAgentActive(string calldata agentId) external view;
    function updateAgentEndpoint(string calldata agentId, string calldata) external;
    function deactivateAgent(string calldata agentId) external;
}
```

**Verification**:
- [ ] Contract compiles without errors
- [ ] All interface functions implemented
- [ ] ERC-8004 compliant

#### 4.2 Test Implementation (8 hours)

**File**: `test/ERC8004IdentityRegistryV4.test.js`

**Test Suites**:

1. **ERC-8004 Compliance Tests** (4 hours)
   ```javascript
   describe("ERC8004IdentityRegistryV4 - Compliance", () => {
     it("Should implement all required functions");
     it("Should resolve agent by DID");
     it("Should resolve agent by address");
     it("Should check if agent is active");
     it("Should update agent endpoint");
     it("Should deactivate agent");
     it("Should emit all required events");
   });
   ```

2. **Multi-Key Mapping Tests** (2 hours)
   ```javascript
   describe("ERC8004IdentityRegistryV4 - Multi-Key", () => {
     it("Should return primary key for agent");
     it("Should handle agents with 1 key");
     it("Should handle agents with multiple keys");
     it("Should select first key as primary");
   });
   ```

3. **AgentDomain Tests** (1 hour)
   ```javascript
   describe("ERC8004IdentityRegistryV4 - Domain", () => {
     it("Should store agent domain");
     it("Should retrieve agent domain");
     it("Should update agent domain");
   });
   ```

4. **Integration Tests** (1 hour)
   ```javascript
   describe("ERC8004IdentityRegistryV4 - Integration", () => {
     it("Should work with AgentCardRegistry");
     it("Should work with ERC8004ReputationRegistry");
     it("Should work with ERC8004ValidationRegistry");
   });
   ```

**Verification**:
- [ ] All tests pass
- [ ] ERC-8004 spec compliance verified
- [ ] Integration with existing contracts works

---

### Phase 5: Integration & Security Testing (Week 3, Day 12-14)

**Duration**: 3 days (24 hours)

**Goal**: Comprehensive testing and security audit

#### 5.1 Integration Tests (8 hours)

**File**: `test/integration/FullWorkflow.test.js`

**Scenarios**:

1. **Complete Registration Workflow** (2 hours)
   ```javascript
   it("Should complete full agent registration workflow", async () => {
     // 1. Commit registration
     // 2. Wait for delay
     // 3. Reveal and register
     // 4. Verify agent stored
     // 5. Wait for activation delay
     // 6. Activate agent
     // 7. Verify agent active
   });
   ```

2. **Multi-Key Lifecycle** (2 hours)
   ```javascript
   it("Should manage complete key lifecycle", async () => {
     // 1. Register with 3 keys
     // 2. Add 2 more keys
     // 3. Revoke 1 key
     // 4. Rotate 1 key
     // 5. Verify final state
   });
   ```

3. **ERC-8004 Ecosystem** (2 hours)
   ```javascript
   it("Should integrate with full ERC-8004 stack", async () => {
     // 1. Register via IdentityRegistry
     // 2. Authorize task in ReputationRegistry
     // 3. Submit feedback
     // 4. Request validation in ValidationRegistry
     // 5. Submit validator response
     // 6. Verify reputation updated
   });
   ```

4. **Attack Scenarios** (2 hours)
   ```javascript
   it("Should prevent front-running attack");
   it("Should prevent Sybil attack with rate limiting");
   it("Should prevent key reuse attack");
   it("Should prevent reentrancy attack");
   it("Should prevent cross-chain replay attack");
   ```

#### 5.2 Security Testing (8 hours)

**Tools**:
- Slither (static analysis)
- Mythril (symbolic execution)
- Echidna (fuzzing)
- Manual review

**Tasks**:

1. **Static Analysis** (2 hours)
   ```bash
   npm run slither
   ```
   - [ ] Fix all HIGH severity issues
   - [ ] Review MEDIUM severity issues
   - [ ] Document LOW severity issues

2. **Symbolic Execution** (2 hours)
   ```bash
   npm run mythril
   ```
   - [ ] Check for integer overflow/underflow
   - [ ] Check for reentrancy
   - [ ] Check for unchecked calls

3. **Fuzzing** (2 hours)
   ```bash
   npm run echidna
   ```
   - [ ] Test invariants
   - [ ] Test edge cases
   - [ ] Test random inputs

4. **Manual Security Review** (2 hours)
   - [ ] Review all external calls
   - [ ] Review all state changes
   - [ ] Review all access controls
   - [ ] Review gas optimization

#### 5.3 Gas Optimization (8 hours)

**File**: `test/gas/GasBenchmark.test.js`

**Benchmarks**:

1. **Registration Costs** (2 hours)
   ```javascript
   it("Should measure gas for 1-key registration");
   it("Should measure gas for 5-key registration");
   it("Should measure gas for 10-key registration");
   it("Should verify costs < 2M gas");
   ```

2. **Key Management Costs** (2 hours)
   ```javascript
   it("Should measure gas for addKey");
   it("Should measure gas for revokeKey");
   it("Should measure gas for rotateKey");
   ```

3. **Query Costs** (2 hours)
   ```javascript
   it("Should measure gas for getAgent");
   it("Should measure gas for getAgentByDID");
   it("Should measure gas for getKey");
   it("Should verify query costs < 100k");
   ```

4. **Optimization Opportunities** (2 hours)
   - [ ] Identify expensive operations
   - [ ] Optimize storage layout
   - [ ] Optimize loop iterations
   - [ ] Document gas savings

**Verification**:
- [ ] All integration tests pass
- [ ] Security audit clean (no HIGH issues)
- [ ] Gas costs within targets
- [ ] Test coverage ≥ 90%

---

### Phase 6: Dependency Upgrades (Week 3, Day 15-17)

**Duration**: 3 days (24 hours)

**Goal**: Safely upgrade all dependencies

#### 6.1 Low-Risk Upgrades (Day 15, 8 hours)

**Tasks**:
1. [ ] Merge PR #107 (@types/node upgrade)
   - Test: `npm run compile`
   - Test: `npm test`

2. [ ] Merge PR #108 (@typechain/ethers-v6 upgrade)
   - Test: Regenerate typechain types
   - Test: Verify type safety

3. [ ] Merge PR #58 (@typechain/hardhat upgrade)
   - Test: Regenerate typechain types
   - Test: Run full test suite

**Verification**:
- [ ] All tests still pass
- [ ] No new compiler warnings
- [ ] TypeChain types correct

#### 6.2 Formatting Upgrades (Day 16, 8 hours)

**Tasks**:
1. [ ] Merge PR #104 (prettier-plugin-solidity upgrade)
   - Run: `npm run format`
   - Review: Formatting changes

2. [ ] Merge PR #106 (solhint-plugin-prettier upgrade)
   - Test: `npm run lint`

3. [ ] Merge PR #105 (prettier 3.6.2 upgrade)
   - Run: `npm run format`
   - Commit: All formatting changes
   - Test: Full test suite

**Verification**:
- [ ] Code formatted correctly
- [ ] No logic changes
- [ ] All tests pass

#### 6.3 Critical Upgrades (Day 17, 8 hours)

**Tasks**:
1. [ ] Review Hardhat 3.0 migration guide
   - Document breaking changes
   - Plan config updates

2. [ ] Merge PR #109 (solhint 6.0.1 upgrade)
   - Review new lint rules
   - Fix critical errors
   - Update .solhint.json if needed
   - Test: `npm run lint`

3. [ ] Merge PR #102 (hardhat 3.0.7 upgrade)
   - Update hardhat.config.js
   - Test all hardhat tasks
   - Run full test suite

4. [ ] Merge PR #59 (hardhat-toolbox 6.1.0 upgrade)
   - Verify bundled plugins
   - Test all toolbox features

**Verification**:
- [ ] Hardhat 3.0 working correctly
- [ ] All hardhat tasks functional
- [ ] All tests pass
- [ ] No lint errors

---

### Phase 7: Migration & Deployment (Week 4, Day 18-20)

**Duration**: 3 days (24 hours)

**Goal**: Migrate from V4 and deploy to testnet

#### 7.1 Migration Script (Day 18, 8 hours)

**File**: `scripts/migrate-v4-to-final.js`

**Tasks**:
1. [ ] Read all agents from V4 registry
2. [ ] For each agent:
   - Extract metadata
   - Extract all keys
   - Commit registration in new registry
   - Reveal and register
   - Verify migration successful
3. [ ] Create migration report
4. [ ] Handle failures gracefully

**Verification**:
- [ ] Migration script runs without errors
- [ ] All agents migrated
- [ ] All keys preserved
- [ ] All metadata correct

#### 7.2 Testnet Deployment (Day 19, 8 hours)

**Network**: Sepolia

**Tasks**:
1. [ ] Deploy AgentCardStorage
2. [ ] Deploy AgentCardVerifyHook
3. [ ] Deploy AgentCardRegistry
4. [ ] Deploy ERC8004IdentityRegistryV4
5. [ ] Configure contracts (set hooks, etc.)
6. [ ] Verify contracts on Etherscan
7. [ ] Test basic operations on testnet

**Verification**:
- [ ] All contracts deployed successfully
- [ ] Contract addresses recorded
- [ ] Contracts verified on Etherscan
- [ ] Basic operations work

#### 7.3 Documentation & CLI Updates (Day 20, 8 hours)

**Tasks**:
1. [ ] Update README.md with new contract addresses
2. [ ] Update deployment guide
3. [ ] Create user guide
4. [ ] Update CLI tools (sage-did)
   - Support new contracts
   - Support multi-key operations
   - Support commit-reveal flow
5. [ ] Create example scripts
6. [ ] Update API documentation

**Verification**:
- [ ] All documentation updated
- [ ] CLI tools work with new contracts
- [ ] Examples run successfully
- [ ] API docs complete

---

## TDD Test Strategy

### Test Pyramid

```
                    /\
                   /  \
                  /E2E \          10% (Integration tests)
                 /______\
                /        \
               / Integration\      30% (Contract integration)
              /____________\
             /              \
            /  Unit Tests    \    60% (Function-level tests)
           /__________________\
```

### Test Categories

1. **Unit Tests** (60% of tests)
   - Test each function in isolation
   - Mock external dependencies
   - Cover all branches and edge cases
   - Fast execution (<1s per test)

2. **Integration Tests** (30% of tests)
   - Test contract interactions
   - Test workflows across contracts
   - Test with real dependencies
   - Medium execution (1-5s per test)

3. **E2E Tests** (10% of tests)
   - Test complete user workflows
   - Test full ERC-8004 ecosystem
   - Test migration scenarios
   - Slow execution (>5s per test)

### Coverage Targets

| Category | Target | Minimum |
|----------|--------|---------|
| Line Coverage | 95% | 90% |
| Branch Coverage | 90% | 85% |
| Function Coverage | 100% | 95% |
| Statement Coverage | 95% | 90% |

### Test Execution

```bash
# Run all tests
npm test

# Run with coverage
npm run coverage

# Run specific test file
npm test test/AgentCardRegistry.test.js

# Run specific test suite
npm test -- --grep "Commit-Reveal"

# Run gas benchmarks
npm run test:gas

# Run security tests
npm run test:security
```

---

## Work Breakdown

### Estimated Hours by Phase

| Phase | Description | Hours | Days |
|-------|-------------|-------|------|
| 0 | Preparation | 16 | 2 |
| 1 | AgentCardStorage | 16 | 2 |
| 2 | AgentCardVerifyHook | 16 | 2 |
| 3 | AgentCardRegistry | 40 | 5 |
| 4 | ERC8004IdentityRegistryV4 | 16 | 2 |
| 5 | Integration & Security | 24 | 3 |
| 6 | Dependency Upgrades | 24 | 3 |
| 7 | Migration & Deployment | 24 | 3 |
| **Total** | | **176** | **22** |

### Resource Allocation

**Development**: 1 Senior Solidity Developer (full-time)
- Contract implementation
- Test writing
- Security review

**Testing**: 0.5 QA Engineer (part-time)
- Test execution
- Bug reporting
- Regression testing

**DevOps**: 0.25 DevOps Engineer (part-time)
- Deployment scripts
- CI/CD setup
- Monitoring

---

## Verification Strategy

### Pre-Implementation Checklist

Before starting each phase:
- [ ] Design document reviewed
- [ ] Test specifications written
- [ ] Dependencies identified
- [ ] Risk assessment complete

### During Implementation Checklist

For each function:
- [ ] Write test first (TDD)
- [ ] Implement function
- [ ] Test passes
- [ ] Code review
- [ ] Refactor if needed

### Post-Implementation Checklist

After completing each phase:
- [ ] All tests pass
- [ ] Coverage targets met
- [ ] Security scan clean
- [ ] Gas costs acceptable
- [ ] Documentation updated
- [ ] Code reviewed
- [ ] Verification matrix updated

### Final Verification Checklist

Before production deployment:
- [ ] All phases complete
- [ ] All tests passing (unit + integration + E2E)
- [ ] Test coverage ≥ 90%
- [ ] Security audit passed (external)
- [ ] Gas costs optimized
- [ ] ERC-8004 compliance verified
- [ ] All dependencies upgraded
- [ ] Migration tested
- [ ] Documentation complete
- [ ] CLI tools updated
- [ ] Testnet deployment successful
- [ ] Verification matrix 100% complete

---

## Timeline

### 4-Week Schedule

**Week 1: Core Implementation**
- Day 1-2: AgentCardStorage
- Day 3-4: AgentCardVerifyHook
- Day 5: AgentCardRegistry (start)

**Week 2: Main Registry & Adapter**
- Day 6-9: AgentCardRegistry (complete)
- Day 10-11: ERC8004IdentityRegistryV4

**Week 3: Testing & Upgrades**
- Day 12-14: Integration & Security Testing
- Day 15-17: Dependency Upgrades

**Week 4: Migration & Deployment**
- Day 18: Migration Script
- Day 19: Testnet Deployment
- Day 20: Documentation & CLI Updates
- Day 21-22: Buffer for issues

### Milestones

- [ ] **Milestone 1** (End of Week 1): Core contracts implemented and tested
- [ ] **Milestone 2** (End of Week 2): All contracts complete with unit tests
- [ ] **Milestone 3** (End of Week 3): Integration tested and dependencies upgraded
- [ ] **Milestone 4** (End of Week 4): Production deployment ready

---

## Risk Management

### High-Risk Items

1. **Hardhat 3.0 Upgrade** 🔴
   - **Risk**: Breaking changes may break build
   - **Mitigation**: Upgrade last, after all tests pass
   - **Fallback**: Stay on Hardhat 2.26.3 for v1.0

2. **Commit-Reveal Security** 🔴
   - **Risk**: Implementation bugs could allow attacks
   - **Mitigation**: Extensive testing, security audit
   - **Fallback**: Disable commit-reveal, use simple registration

3. **Migration from V4** 🔴
   - **Risk**: Data loss during migration
   - **Mitigation**: Thorough testing, backup, rollback plan
   - **Fallback**: Keep V4 running in parallel

### Medium-Risk Items

4. **Test Coverage** 🟡
   - **Risk**: May not reach 90% coverage
   - **Mitigation**: Write tests first (TDD)
   - **Fallback**: Accept 85% with manual review

5. **Gas Costs** 🟡
   - **Risk**: May exceed 2M gas target
   - **Mitigation**: Gas optimization in Phase 5
   - **Fallback**: Increase target to 2.5M

6. **Solhint 6.0 Compatibility** 🟡
   - **Risk**: New lint rules may require code changes
   - **Mitigation**: Upgrade early, fix issues
   - **Fallback**: Suppress non-critical warnings

### Low-Risk Items

7. **Prettier 3.0 Formatting** 🟢
   - **Risk**: Formatting changes may be extensive
   - **Mitigation**: Automated formatting
   - **Fallback**: Keep Prettier 2.8.8

8. **TypeChain Upgrades** 🟢
   - **Risk**: Type generation changes
   - **Mitigation**: Regenerate types
   - **Fallback**: Keep current versions

---

## Next Steps

1. **Create VERIFICATION_MATRIX.md**
   - Detailed checklist for each feature
   - Test specifications for TDD
   - Acceptance criteria

2. **Create TDD_TEST_SPECS.md**
   - Test-first specifications
   - Expected behaviors
   - Edge cases

3. **Set up development environment**
   - Install dependencies
   - Configure tools
   - Set up CI/CD

4. **Begin Phase 1: AgentCardStorage**
   - Write tests first
   - Implement contract
   - Verify and iterate

---

**Status**: Plan Complete ✅
**Next**: Create Verification Matrix and Test Specifications
**Start Date**: TBD
**Target Completion**: 4 weeks from start
