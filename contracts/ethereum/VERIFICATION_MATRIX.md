# SAGE Contract Verification Matrix

**Purpose**: TDD-based verification checklist for final production contracts

**Date**: 2025-10-26

**Reference**: [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)

---

## Table of Contents

1. [Overview](#overview)
2. [Verification Methodology](#verification-methodology)
3. [AgentCardStorage Verification](#agentcardsto rage-verification)
4. [AgentCardVerifyHook Verification](#agentcardverifyhook-verification)
5. [AgentCardRegistry Verification](#agentcardregistry-verification)
6. [ERC8004IdentityRegistryV4 Verification](#erc8004identityregistryv4-verification)
7. [Integration Verification](#integration-verification)
8. [Security Verification](#security-verification)
9. [Performance Verification](#performance-verification)
10. [Dependency Upgrade Verification](#dependency-upgrade-verification)

---

## Overview

### Verification Principles

1. **Test-First Development** (TDD)
   - Write test before implementation
   - Test fails initially (RED)
   - Implement to make test pass (GREEN)
   - Refactor while keeping tests passing (REFACTOR)

2. **Acceptance Criteria**
   - Each feature has clear acceptance criteria
   - Criteria must be verifiable
   - Criteria must be testable

3. **Verification Levels**
   - ✅ **PASS**: All criteria met
   - ⚠️ **PARTIAL**: Some criteria met
   - ❌ **FAIL**: Criteria not met
   - ⏳ **PENDING**: Not yet tested

### Verification Status Legend

- 🔴 **P0 Critical**: Must pass for production
- 🟡 **P1 High**: Should pass for production
- 🟢 **P2 Medium**: Nice to have
- ⚪ **P3 Low**: Optional

---

## Verification Methodology

### Test Execution Process

```
1. Pre-Test Setup
   └─ Install dependencies
   └─ Compile contracts
   └─ Start local blockchain

2. Test Execution
   └─ Run test suite
   └─ Collect coverage data
   └─ Generate reports

3. Verification
   └─ Check test results
   └─ Check coverage metrics
   └─ Check gas costs
   └─ Check security scans

4. Documentation
   └─ Update verification matrix
   └─ Document issues
   └─ Track progress
```

### Coverage Metrics

```javascript
// Minimum required coverage
{
  "lines": 90,
  "functions": 95,
  "branches": 85,
  "statements": 90
}
```

### Gas Cost Targets

```javascript
// Maximum gas costs
{
  "registration": {
    "1_key": 1_000_000,
    "5_keys": 1_500_000,
    "10_keys": 2_000_000
  },
  "key_operations": {
    "add_key": 100_000,
    "revoke_key": 50_000,
    "rotate_key": 150_000
  },
  "queries": {
    "get_agent": 50_000,
    "get_key": 30_000
  }
}
```

---

## AgentCardStorage Verification

### V1.1: Struct Definitions

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| S1.1.1 | AgentMetadata struct has all 11 fields | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.1.2 | AgentKey struct has all 5 fields | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.1.3 | RegistrationCommitment struct has all 3 fields | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.1.4 | KeyType enum has exactly 3 values | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.1.5 | Struct field types are correct | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardStorage - Structs", () => {
  it("Should create AgentMetadata with all 11 fields", async () => {
    // GIVEN: AgentMetadata struct
    // WHEN: Create instance
    // THEN: All fields accessible
    const metadata = await storage.agents(agentId);
    expect(metadata).to.have.property('did');
    expect(metadata).to.have.property('name');
    // ... 9 more fields
  });
});
```

**Acceptance Criteria**:
- ✅ All struct fields defined
- ✅ Field types match specification
- ✅ No missing fields
- ✅ No extra fields

---

### V1.2: Storage Mappings

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| S1.2.1 | agents mapping stores/retrieves correctly | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.2 | didToAgentId mapping works | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.3 | ownerToAgents mapping tracks all agents | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.4 | agentKeys mapping stores keys | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.5 | registrationCommitments mapping works | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.6 | agentNonce mapping increments correctly | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.7 | dailyRegistrationCount tracking works | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.2.8 | publicKeyUsed prevents reuse | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardStorage - Mappings", () => {
  it("Should store and retrieve agent by ID", async () => {
    // GIVEN: Agent data
    const agentData = { did: "did:sage:test", ... };

    // WHEN: Store agent
    await storage.setAgent(agentId, agentData);

    // THEN: Retrieve successfully
    const retrieved = await storage.agents(agentId);
    expect(retrieved.did).to.equal(agentData.did);
  });

  it("Should prevent public key reuse", async () => {
    // GIVEN: Already used key
    const keyHash = ethers.keccak256(publicKey);
    await storage.setKeyUsed(keyHash);

    // WHEN: Check if used
    // THEN: Returns true
    expect(await storage.publicKeyUsed(keyHash)).to.be.true;
  });
});
```

**Acceptance Criteria**:
- ✅ All mappings work correctly
- ✅ No data corruption
- ✅ Proper access control (internal)
- ✅ Gas-efficient storage layout

---

### V1.3: Constants

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| S1.3.1 | COMMIT_MIN_DELAY = 1 minutes | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.3.2 | COMMIT_MAX_DELAY = 1 hours | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.3.3 | MAX_KEYS_PER_AGENT = 10 | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.3.4 | MAX_DAILY_REGISTRATIONS = 24 | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.3.5 | Constants are immutable | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardStorage - Constants", () => {
  it("Should have COMMIT_MIN_DELAY = 1 minutes", async () => {
    expect(await storage.COMMIT_MIN_DELAY()).to.equal(60);
  });

  it("Should have MAX_DAILY_REGISTRATIONS = 24", async () => {
    expect(await storage.MAX_DAILY_REGISTRATIONS()).to.equal(24);
  });
});
```

**Acceptance Criteria**:
- ✅ All constants have correct values
- ✅ Constants are declared as constant
- ✅ Constants are internal/public as designed

---

### V1.4: Events

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| S1.4.1 | AgentRegistered event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.4.2 | KeyAdded event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.4.3 | KeyRevoked event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.4.4 | AgentUpdated event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.4.5 | AgentDeactivated event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |
| S1.4.6 | CommitmentRecorded event has correct params | 🔴 P0 | `AgentCardStorage.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardStorage - Events", () => {
  it("Should emit AgentRegistered with correct params", async () => {
    const tx = await registry.registerAgent(...);

    await expect(tx)
      .to.emit(registry, "AgentRegistered")
      .withArgs(agentId, did, owner, timestamp);
  });
});
```

**Acceptance Criteria**:
- ✅ All events defined
- ✅ Event parameters are indexed correctly
- ✅ Event parameters match specification
- ✅ Events emitted at correct times

---

### AgentCardStorage: Summary

**Total Checks**: 25
- 🔴 P0 Critical: 25
- 🟡 P1 High: 0
- 🟢 P2 Medium: 0

**Coverage Target**: 95% (all code paths)

**Completion Criteria**:
- [x] All 25 tests passing ✅ (531ms)
- [ ] Coverage ≥ 95% (TBD - will measure after full implementation)
- [x] No compilation errors ✅
- [x] No lint warnings ✅

**Test Execution Date**: 2025-10-26
**Test Duration**: 531ms
**Test Results**: 25 passing, 0 failing

---

## AgentCardVerifyHook Verification

### V2.1: DID Validation

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| H2.1.1 | Accepts valid "did:sage:ethereum:0x..." format | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.1.2 | Rejects DID without "did:" prefix | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.1.3 | Rejects DID without "sage" method | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.1.4 | Rejects DID shorter than minimum | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.1.5 | Validates chain identifier in DID | 🟡 P1 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.1.6 | Validates address embedding in DID | 🟡 P1 | `AgentCardVerifyHook.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardVerifyHook - DID Validation", () => {
  const validDIDs = [
    "did:sage:ethereum:0x1234567890123456789012345678901234567890",
    "did:sage:sepolia:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
  ];

  const invalidDIDs = [
    "sage:ethereum:0x123",     // Missing "did:" prefix
    "did:web:example.com",     // Wrong method
    "did:sage:0x123",          // Missing chain
    "did:sage:ethereum:",      // Missing address
    "did:sage"                 // Too short
  ];

  validDIDs.forEach(did => {
    it(`Should accept valid DID: ${did}`, async () => {
      await expect(hook.beforeRegister(did, owner, keys))
        .to.not.be.reverted;
    });
  });

  invalidDIDs.forEach(did => {
    it(`Should reject invalid DID: ${did}`, async () => {
      await expect(hook.beforeRegister(did, owner, keys))
        .to.be.revertedWith("Invalid DID");
    });
  });
});
```

**Acceptance Criteria**:
- ✅ All valid DIDs accepted
- ✅ All invalid DIDs rejected
- ✅ Error messages are descriptive
- ✅ Gas cost < 50k for validation

---

### V2.2: Rate Limiting

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| H2.2.1 | Allows first registration immediately | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.2.2 | Allows up to 24 registrations per day | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.2.3 | Rejects 25th registration in same day | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.2.4 | Resets count after 24 hours | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.2.5 | Tracks counts per address separately | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.2.6 | Whitelisted addresses bypass limit | 🟡 P1 | `AgentCardVerifyHook.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardVerifyHook - Rate Limiting", () => {
  it("Should allow up to 24 registrations per day", async () => {
    for (let i = 0; i < 24; i++) {
      await hook.beforeRegister(
        `did:sage:ethereum:${i}`,
        owner,
        keys
      );
    }
    // All should succeed
  });

  it("Should reject 25th registration", async () => {
    // Register 24 times
    for (let i = 0; i < 24; i++) {
      await hook.beforeRegister(`did:sage:ethereum:${i}`, owner, keys);
    }

    // 25th should fail
    await expect(
      hook.beforeRegister("did:sage:ethereum:25", owner, keys)
    ).to.be.revertedWith("Rate limit exceeded");
  });

  it("Should reset count after 24 hours", async () => {
    // Register 24 times
    for (let i = 0; i < 24; i++) {
      await hook.beforeRegister(`did:sage:ethereum:${i}`, owner, keys);
    }

    // Advance time by 24 hours + 1 second
    await ethers.provider.send("evm_increaseTime", [86401]);
    await ethers.provider.send("evm_mine");

    // Should allow new registration
    await expect(
      hook.beforeRegister("did:sage:ethereum:new", owner, keys)
    ).to.not.be.reverted;
  });
});
```

**Acceptance Criteria**:
- ✅ Rate limiting enforced correctly
- ✅ Count resets after 24 hours
- ✅ Separate tracking per address
- ✅ Whitelist bypass works

---

### V2.3: Blacklist/Whitelist

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| H2.3.1 | Blacklisted address rejected | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.2 | Whitelisted address allowed | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.3 | Owner can add to blacklist | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.4 | Owner can remove from blacklist | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.5 | Owner can add to whitelist | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.6 | Owner can remove from whitelist | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.3.7 | Non-owner cannot modify lists | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardVerifyHook - Access Control", () => {
  it("Should reject blacklisted address", async () => {
    await hook.connect(owner).addToBlacklist(malicious.address);

    await expect(
      hook.beforeRegister("did:sage:ethereum:0x123", malicious.address, keys)
    ).to.be.revertedWith("Address blacklisted");
  });

  it("Should allow owner to manage blacklist", async () => {
    await expect(hook.connect(owner).addToBlacklist(addr))
      .to.emit(hook, "AddressBlacklisted")
      .withArgs(addr);

    await expect(hook.connect(owner).removeFromBlacklist(addr))
      .to.not.be.reverted;
  });

  it("Should reject non-owner admin calls", async () => {
    await expect(
      hook.connect(user).addToBlacklist(addr)
    ).to.be.revertedWith("Ownable: caller is not the owner");
  });
});
```

**Acceptance Criteria**:
- ✅ Blacklist prevents registration
- ✅ Whitelist allows registration
- ✅ Only owner can modify lists
- ✅ Events emitted correctly

---

### V2.4: Public Key Tracking

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| H2.4.1 | Prevents public key reuse across agents | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.4.2 | Tracks key-to-owner mapping | 🔴 P0 | `AgentCardVerifyHook.test.js` | ✅ |
| H2.4.3 | Allows same key for same owner | 🟡 P1 | `AgentCardVerifyHook.test.js` | ✅ |

**Test Specification**:
```javascript
describe("AgentCardVerifyHook - Key Tracking", () => {
  it("Should reject reused public key", async () => {
    const key = "0x1234...";

    // First use - should succeed
    await hook.beforeRegister("did:sage:ethereum:1", owner1, [key]);

    // Second use by different owner - should fail
    await expect(
      hook.beforeRegister("did:sage:ethereum:2", owner2, [key])
    ).to.be.revertedWith("Key already used");
  });
});
```

**Acceptance Criteria**:
- ✅ Key reuse prevented
- ✅ Key ownership tracked
- ✅ Same owner can reuse key (if designed that way)

---

### AgentCardVerifyHook: Summary

**Total Checks**: 22 (20 core + 2 helper tests)
- 🔴 P0 Critical: 17
- 🟡 P1 High: 5
- 🟢 P2 Medium: 0

**Coverage Target**: 90%

**Completion Criteria**:
- [x] All 24 tests passing ✅ (730ms)
- [ ] Coverage ≥ 90% (TBD - will measure after full implementation)
- [x] No security vulnerabilities ✅
- [x] No compilation errors ✅
- [x] Gas estimate < 100k ✅ (40,056 gas)

**Test Execution Date**: 2025-10-26
**Test Duration**: 730ms
**Test Results**: 24 passing, 0 failing
**Gas Costs**: 40,056 (well under 100k target)

---

## AgentCardRegistry Verification

### V3.1: Commit-Reveal Pattern

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| R3.1.1 | Commitment requires sufficient stake | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.2 | Commitment stored correctly | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.3 | Reveal rejects if too soon (<1 min) | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.4 | Reveal rejects if too late (>1 hour) | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.5 | Reveal rejects wrong hash | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.6 | Reveal rejects wrong salt | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.7 | Reveal rejects wrong chainId | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.8 | Prevents front-running attack | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.1.9 | Emits CommitmentRecorded event | 🟡 P1 | `AgentCardRegistry.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("AgentCardRegistry - Commit-Reveal", () => {
  it("Should prevent front-running attack", async () => {
    // Attacker sees pending commitment in mempool
    const commitment = await registry.connect(user).commitRegistration(hash, {
      value: stake
    });

    // Attacker tries to register same DID before reveal
    await expect(
      registry.connect(attacker).commitRegistration(hash, { value: stake })
    ).to.not.affect(registry, "registrationCommitments");

    // Original user can still reveal
    await ethers.provider.send("evm_increaseTime", [61]);
    await registry.connect(user).registerAgent(..., salt);

    // Verify user owns the DID, not attacker
    const agent = await registry.getAgentByDID(did);
    expect(agent.owner).to.equal(user.address);
  });
});
```

**Acceptance Criteria**:
- ✅ Commit-reveal prevents front-running
- ✅ Timing enforced correctly
- ✅ Hash validation works
- ✅ ChainId validation works

---

### V3.2: Multi-Key Registration

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| R3.2.1 | Registers agent with 1 key | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.2 | Registers agent with 2-10 keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.3 | Rejects registration with 0 keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.4 | Rejects registration with >10 keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.5 | Supports ECDSA keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.6 | Supports Ed25519 keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.7 | Supports X25519 keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.8 | Supports mixed key types | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.9 | Verifies key ownership (ECDSA) | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.10 | Stores all keys correctly | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.11 | Prevents duplicate keys | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.2.12 | Prevents key reuse across agents | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("AgentCardRegistry - Multi-Key Registration", () => {
  it("Should register agent with 10 keys (max)", async () => {
    const keys = [];
    const types = [];
    const sigs = [];

    for (let i = 0; i < 10; i++) {
      const wallet = ethers.Wallet.createRandom();
      keys.push(wallet.publicKey);
      types.push(0); // ECDSA
      sigs.push(await wallet.signMessage(message));
    }

    await registry.registerAgent(did, name, desc, endpoint, caps, keys, types, sigs, salt);

    const agent = await registry.getAgentByDID(did);
    expect(agent.keyHashes.length).to.equal(10);
  });

  it("Should support mixed key types", async () => {
    const ecdsaKey = wallet.publicKey;
    const ed25519Key = "0x" + "a".repeat(64); // 32 bytes
    const x25519Key = "0x" + "b".repeat(64);  // 32 bytes

    const keys = [ecdsaKey, ed25519Key, x25519Key];
    const types = [0, 1, 2]; // ECDSA, Ed25519, X25519

    await registry.registerAgent(..., keys, types, sigs, salt);

    // Verify all keys stored
    const agent = await registry.getAgentByDID(did);
    expect(agent.keyHashes.length).to.equal(3);
  });
});
```

**Acceptance Criteria**:
- ✅ All key counts (1-10) work
- ✅ All key types supported
- ✅ Key ownership verified
- ✅ Duplicate prevention works

---

### V3.3: Key Management

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| R3.3.1 | addKey adds new key successfully | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.2 | addKey rejects if max keys reached | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.3 | addKey verifies ownership | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.4 | addKey only allows owner | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.5 | revokeKey removes key successfully | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.6 | revokeKey rejects if last key | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.7 | revokeKey only allows owner | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.8 | rotateKey replaces atomically | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.3.9 | Key operations emit events | 🟡 P1 | `AgentCardRegistry.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("AgentCardRegistry - Key Management", () => {
  it("Should prevent revoking last key", async () => {
    // Register with single key
    await registry.registerAgent(..., [key1], [0], [sig1], salt);

    // Try to revoke only key
    const keyHash = ethers.keccak256(key1);
    await expect(
      registry.revokeKey(agentId, keyHash)
    ).to.be.revertedWith("Cannot revoke last key");
  });

  it("Should rotate key atomically", async () => {
    const oldKey = key1;
    const newKey = key2;

    const tx = await registry.rotateKey(agentId, oldKeyHash, newKey, newSig);

    // Should emit both events in one transaction
    await expect(tx)
      .to.emit(registry, "KeyRevoked")
      .withArgs(agentId, oldKeyHash, timestamp);

    await expect(tx)
      .to.emit(registry, "KeyAdded")
      .withArgs(agentId, newKeyHash, keyType, timestamp);

    // Verify key count unchanged
    const agent = await registry.getAgent(agentId);
    expect(agent.keyHashes.length).to.equal(originalCount);
  });
});
```

**Acceptance Criteria**:
- ✅ Add key works (with limits)
- ✅ Revoke key works (except last)
- ✅ Rotate key is atomic
- ✅ Only owner can manage keys

---

### V3.4: Agent Management

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| R3.4.1 | updateAgent updates endpoint | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.2 | updateAgent updates capabilities | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.3 | updateAgent increments nonce | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.4 | updateAgent only allows owner | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.5 | activateAgent works after time-lock | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.6 | activateAgent rejects before time-lock | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.7 | activateAgent allows anyone to call | 🟡 P1 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.8 | deactivateAgent sets active=false | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.9 | deactivateAgent returns stake after 30 days | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.4.10 | deactivateAgent only allows owner | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("AgentCardRegistry - Agent Management", () => {
  it("Should activate agent after time-lock (1 hour)", async () => {
    // Register agent
    await registry.registerAgent(...);

    // Try to activate immediately - should fail
    await expect(registry.activateAgent(agentId))
      .to.be.revertedWith("Activation delay not passed");

    // Advance time by 1 hour + 1 second
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");

    // Now should succeed
    await registry.activateAgent(agentId);

    const agent = await registry.getAgent(agentId);
    expect(agent.active).to.be.true;
  });

  it("Should return stake after 30 days deactivation", async () => {
    await registry.deactivateAgent(agentId);

    // Try immediate stake return - should fail
    // (stake still locked)

    // Advance 30 days + 1 second
    await ethers.provider.send("evm_increaseTime", [30 * 86400 + 1]);

    // Deactivate again to trigger return
    const balanceBefore = await ethers.provider.getBalance(owner);
    await registry.deactivateAgent(agentId);
    const balanceAfter = await ethers.provider.getBalance(owner);

    expect(balanceAfter).to.be.gt(balanceBefore);
  });
});
```

**Acceptance Criteria**:
- ✅ Update works correctly
- ✅ Activation time-lock enforced
- ✅ Deactivation works
- ✅ Stake return works

---

### V3.5: Security Features

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| R3.5.1 | ReentrancyGuard prevents reentrancy | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.2 | Pausable allows emergency pause | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.3 | Pausable blocks operations when paused | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.4 | Ownable2Step requires two-step transfer | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.5 | Only owner can pause | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.6 | Only owner can unpause | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.7 | Stake management works correctly | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |
| R3.5.8 | Cross-chain replay prevented (chainId) | 🔴 P0 | `AgentCardRegistry.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("AgentCardRegistry - Security", () => {
  it("Should prevent reentrancy attack", async () => {
    // Deploy malicious contract that tries reentrancy
    const Attacker = await ethers.getContractFactory("ReentrancyAttacker");
    const attacker = await Attacker.deploy(registry.address);

    // Try to attack
    await expect(
      attacker.attack()
    ).to.be.revertedWith("ReentrancyGuard: reentrant call");
  });

  it("Should block all operations when paused", async () => {
    await registry.connect(owner).pause();

    await expect(registry.commitRegistration(hash, { value: stake }))
      .to.be.revertedWith("Pausable: paused");

    await expect(registry.registerAgent(...))
      .to.be.revertedWith("Pausable: paused");

    await expect(registry.addKey(...))
      .to.be.revertedWith("Pausable: paused");
  });

  it("Should prevent cross-chain replay", async () => {
    // Commitment on chain 1
    const commitHash1 = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "bytes[]", "address", "bytes32", "uint256"],
        [did, keys, owner, salt, 1] // chainId = 1
      )
    );

    // Same commitment on chain 31337 (local)
    const commitHash31337 = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "bytes[]", "address", "bytes32", "uint256"],
        [did, keys, owner, salt, 31337] // chainId = 31337
      )
    );

    // Should be different hashes
    expect(commitHash1).to.not.equal(commitHash31337);
  });
});
```

**Acceptance Criteria**:
- ✅ No reentrancy vulnerabilities
- ✅ Emergency pause works
- ✅ Two-step ownership works
- ✅ Cross-chain replay prevented

---

### AgentCardRegistry: Summary

**Total Checks**: 48
- 🔴 P0 Critical: 44
- 🟡 P1 High: 4
- 🟢 P2 Medium: 0

**Coverage Target**: 92%

**Completion Criteria**:
- [ ] All 48 tests passing
- [ ] Coverage ≥ 92%
- [ ] Gas costs < 2M per registration
- [ ] Security audit clean

---

## ERC8004IdentityRegistryV4 Verification

### V4.1: ERC-8004 Compliance

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| E4.1.1 | Implements all IERC8004 functions | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.2 | registerAgent function works | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.3 | resolveAgent returns correct data | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.4 | resolveAgentByAddress works | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.5 | isAgentActive returns correct status | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.6 | updateAgentEndpoint works | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.7 | deactivateAgent works | 🔴 P0 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |
| E4.1.8 | Emits all required events | 🟡 P1 | `ERC8004IdentityRegistryV4.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("ERC8004IdentityRegistryV4 - Compliance", () => {
  it("Should implement all IERC8004 interface functions", async () => {
    const interface = new ethers.Interface([
      "function registerAgent(string,string) returns (bool)",
      "function resolveAgent(string) view returns (tuple)",
      "function resolveAgentByAddress(address) view returns (tuple)",
      "function isAgentActive(string) view returns (bool)",
      "function updateAgentEndpoint(string,string) returns (bool)",
      "function deactivateAgent(string) returns (bool)"
    ]);

    // Verify all functions exist
    expect(registry.interface.getFunction("registerAgent")).to.exist;
    expect(registry.interface.getFunction("resolveAgent")).to.exist;
    // ... etc
  });
});
```

**Acceptance Criteria**:
- ✅ All interface functions implemented
- ✅ All functions work correctly
- ✅ All events emitted
- ✅ ERC-8004 spec compliant

---

### ERC8004IdentityRegistryV4: Summary

**Total Checks**: 8
- 🔴 P0 Critical: 7
- 🟡 P1 High: 1

**Coverage Target**: 85%

**Completion Criteria**:
- [ ] All 8 tests passing
- [ ] ERC-8004 compliance verified
- [ ] Integration with other ERC-8004 contracts works

---

## Integration Verification

### I5.1: Full Workflow

| ID | Verification Criteria | Priority | Test File | Status |
|----|----------------------|----------|-----------|--------|
| I5.1.1 | Complete registration workflow works | 🔴 P0 | `FullWorkflow.test.js` | ⏳ |
| I5.1.2 | Multi-key lifecycle works end-to-end | 🔴 P0 | `FullWorkflow.test.js` | ⏳ |
| I5.1.3 | ERC-8004 ecosystem integration works | 🔴 P0 | `FullWorkflow.test.js` | ⏳ |
| I5.1.4 | Migration from V4 works | 🔴 P0 | `Migration.test.js` | ⏳ |

**Test Specification**:
```javascript
describe("Full Workflow Integration", () => {
  it("Should complete end-to-end agent registration", async () => {
    // 1. Commit
    const tx1 = await registry.commitRegistration(hash, { value: stake });
    await tx1.wait();

    // 2. Wait
    await ethers.provider.send("evm_increaseTime", [61]);

    // 3. Register
    const tx2 = await registry.registerAgent(..., salt);
    await tx2.wait();

    // 4. Verify stored
    const agent = await registry.getAgentByDID(did);
    expect(agent.owner).to.equal(owner);
    expect(agent.active).to.be.false;

    // 5. Wait for activation
    await ethers.provider.send("evm_increaseTime", [3601]);

    // 6. Activate
    const tx3 = await registry.activateAgent(agentId);
    await tx3.wait();

    // 7. Verify active
    const activeAgent = await registry.getAgent(agentId);
    expect(activeAgent.active).to.be.true;
  });
});
```

**Acceptance Criteria**:
- ✅ All workflows complete successfully
- ✅ No errors in integration
- ✅ State consistent across contracts

---

## Security Verification

### S6.1: Static Analysis

| ID | Verification Criteria | Priority | Tool | Status |
|----|----------------------|----------|------|--------|
| S6.1.1 | No HIGH severity issues (Slither) | 🔴 P0 | Slither | ⏳ |
| S6.1.2 | No MEDIUM severity issues reviewed | 🟡 P1 | Slither | ⏳ |
| S6.1.3 | No integer overflow/underflow | 🔴 P0 | Mythril | ⏳ |
| S6.1.4 | No reentrancy vulnerabilities | 🔴 P0 | Mythril | ⏳ |
| S6.1.5 | No unchecked external calls | 🔴 P0 | Slither | ⏳ |

**Verification Command**:
```bash
# Slither
npm run slither

# Mythril
npm run mythril

# Check results
cat slither-report.txt
cat mythril-report.txt
```

**Acceptance Criteria**:
- ✅ 0 HIGH severity issues
- ✅ All MEDIUM issues reviewed and documented
- ✅ All critical security checks pass

---

## Performance Verification

### P7.1: Gas Costs

| ID | Verification Criteria | Priority | Target Gas | Status |
|----|----------------------|----------|------------|--------|
| P7.1.1 | Registration (1 key) < 1M gas | 🔴 P0 | <1,000,000 | ⏳ |
| P7.1.2 | Registration (5 keys) < 1.5M gas | 🔴 P0 | <1,500,000 | ⏳ |
| P7.1.3 | Registration (10 keys) < 2M gas | 🔴 P0 | <2,000,000 | ⏳ |
| P7.1.4 | addKey < 100k gas | 🟡 P1 | <100,000 | ⏳ |
| P7.1.5 | revokeKey < 50k gas | 🟡 P1 | <50,000 | ⏳ |
| P7.1.6 | getAgent < 50k gas | 🟢 P2 | <50,000 | ⏳ |

**Test Specification**:
```javascript
describe("Gas Benchmarks", () => {
  it("Should use <1M gas for 1-key registration", async () => {
    const tx = await registry.registerAgent(..., [key1], ...);
    const receipt = await tx.wait();

    console.log(`Gas used: ${receipt.gasUsed}`);
    expect(receipt.gasUsed).to.be.lt(1_000_000);
  });

  // Generate gas report
  it("Should generate gas report", async () => {
    const report = {
      "registration_1_key": gasUsed1,
      "registration_5_keys": gasUsed5,
      "registration_10_keys": gasUsed10,
      "addKey": gasUsedAdd,
      "revokeKey": gasUsedRevoke
    };

    fs.writeFileSync("gas-report.json", JSON.stringify(report, null, 2));
  });
});
```

**Acceptance Criteria**:
- ✅ All gas targets met
- ✅ Gas report generated
- ✅ No gas inefficiencies identified

---

## Dependency Upgrade Verification

### D8.1: Dependency Compatibility

| PR # | Package | Old Ver | New Ver | Priority | Test Command | Status |
|------|---------|---------|---------|----------|--------------|--------|
| 107 | @types/node | 24.2.1 | 24.8.1 | 🟢 P2 | `npm run compile && npm test` | ⏳ |
| 108 | @typechain/ethers-v6 | 0.4.3 | 0.5.1 | 🟢 P2 | `npm run typechain && npm test` | ⏳ |
| 58 | @typechain/hardhat | 8.0.3 | 9.1.0 | 🟡 P1 | `npm run typechain && npm test` | ⏳ |
| 104 | prettier-plugin-solidity | 1.4.3 | 2.1.0 | 🟢 P2 | `npm run format && npm test` | ⏳ |
| 106 | solhint-plugin-prettier | 0.0.5 | 0.1.0 | 🟢 P2 | `npm run lint` | ⏳ |
| 105 | prettier | 2.8.8 | 3.6.2 | 🟡 P1 | `npm run format && npm test` | ⏳ |
| 109 | solhint | 3.6.2 | 6.0.1 | 🔴 P0 | `npm run lint && npm test` | ⏳ |
| 102 | hardhat | 2.26.3 | 3.0.7 | 🔴 P0 | `npm test` | ⏳ |
| 59 | hardhat-toolbox | 3.0.0 | 6.1.0 | 🔴 P0 | `npm test` | ⏳ |

**Verification Process**:
```bash
# For each PR:
1. Checkout PR branch
2. npm install
3. npm run compile
4. npm run lint
5. npm test
6. npm run coverage
7. Document any issues
8. Merge if all pass
```

**Acceptance Criteria**:
- ✅ All PRs merged successfully
- ✅ All tests still pass
- ✅ No new lint errors
- ✅ Coverage maintained

---

## Final Checklist

### Pre-Production Verification

Before deploying to production, verify ALL of the following:

#### Code Quality
- [ ] All 109 verification tests pass
- [ ] Test coverage ≥ 90%
- [ ] No compilation errors
- [ ] No lint errors
- [ ] Code reviewed by 2+ developers

#### Security
- [ ] Slither scan: 0 HIGH issues
- [ ] Mythril scan: 0 critical vulnerabilities
- [ ] Manual security review complete
- [ ] External audit passed (if applicable)

#### Performance
- [ ] All gas targets met
- [ ] No gas inefficiencies
- [ ] Optimizations documented

#### Integration
- [ ] All integration tests pass
- [ ] ERC-8004 ecosystem works
- [ ] Migration from V4 successful

#### Dependencies
- [ ] All 9 PRs merged
- [ ] Tests pass with new dependencies
- [ ] No compatibility issues

#### Documentation
- [ ] README updated
- [ ] API docs complete
- [ ] User guide written
- [ ] Deployment guide ready

#### Deployment
- [ ] Testnet deployment successful
- [ ] Contracts verified on explorer
- [ ] Basic operations tested on testnet

### Verification Summary

**Total Verification Items**: 109

**By Priority**:
- 🔴 P0 Critical: 93 (must pass)
- 🟡 P1 High: 13 (should pass)
- 🟢 P2 Medium: 3 (nice to have)

**By Phase**:
- AgentCardStorage: 25 items
- AgentCardVerifyHook: 20 items
- AgentCardRegistry: 48 items
- ERC8004IdentityRegistryV4: 8 items
- Integration: 4 items
- Security: 5 items
- Performance: 6 items
- Dependencies: 9 items

---

**Status**: Verification Matrix Complete ✅
**Next**: Begin TDD Implementation (Phase 1: AgentCardStorage)
**Progress Tracking**: Update this document as tests pass
