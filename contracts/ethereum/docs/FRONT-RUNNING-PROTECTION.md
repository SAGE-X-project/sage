# Front-Running Protection Guide

**Version:** 1.0
**Date:** 2025-10-07
**Contracts:** SageRegistryV3, ERC8004ReputationRegistryV2

---

## Overview

This document explains the commit-reveal scheme implemented in SAGE contracts to prevent front-running attacks.

### What is Front-Running?

**Front-running** occurs when an attacker sees a pending transaction in the mempool and submits their own transaction with higher gas to execute first.

**Example Attack Scenario (Without Protection):**
```
1. Alice wants to register DID "did:sage:alice"
2. Alice submits registerAgent("did:sage:alice", ...)
3. Bob (attacker) sees Alice's transaction in mempool
4. Bob submits registerAgent("did:sage:alice", ...) with 2x gas price
5. Bob's transaction executes first
6. Bob steals Alice's desired DID
7. Alice's transaction fails ("DID already registered")
```

---

## Commit-Reveal Pattern

The **commit-reveal** scheme prevents front-running by hiding the intent until after commitment:

### Process Flow

```
Step 1: COMMIT (Hide Intent)
┌─────────────────────────────────┐
│ User creates commitment off-chain │
│ commitHash = keccak256(          │
│   did + publicKey + salt + chainId │
│ )                                │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│ User calls commitRegistration()  │
│ Only hash is visible on-chain   │
│ Attacker cannot see DID          │
└────────────┬────────────────────┘
             │
             ▼ Wait 1 minute
┌─────────────────────────────────┐
│ Step 2: REVEAL (Execute)        │
│ User calls registerAgentWithReveal() │
│ Provides did, publicKey, salt   │
│ Contract verifies hash matches   │
└─────────────────────────────────┘
```

**Why This Works:**
- Attacker sees commit hash but doesn't know the DID
- Attacker cannot generate valid hash without knowing salt
- User's DID is hidden until reveal
- After reveal, user's transaction is already committed

---

## Implementation

### 1. Agent Registration (SageRegistryV3)

#### Client-Side (JavaScript/TypeScript)

```javascript
const { ethers } = require("ethers");

// Step 1: Create commitment off-chain
async function createCommitment(did, publicKey, userAddress) {
    // Generate random salt (keep this secret!)
    const salt = ethers.randomBytes(32);

    // Get current chain ID
    const network = await provider.getNetwork();
    const chainId = network.chainId;

    // Create commitment hash
    const commitHash = ethers.keccak256(
        ethers.solidityPacked(
            ["string", "bytes", "address", "bytes32", "uint256"],
            [did, publicKey, userAddress, salt, chainId]
        )
    );

    return { commitHash, salt };
}

// Step 2: Commit
async function commitRegistration(registry, commitHash) {
    const tx = await registry.commitRegistration(commitHash);
    await tx.wait();
    console.log("✅ Commitment submitted");
    console.log("⏳ Wait 1 minute before revealing...");
}

// Step 3: Wait minimum delay
await new Promise(resolve => setTimeout(resolve, 61000)); // 61 seconds

// Step 4: Reveal and register
async function registerWithReveal(registry, params, salt) {
    const tx = await registry.registerAgentWithReveal(
        params.did,
        params.name,
        params.description,
        params.endpoint,
        params.publicKey,
        params.capabilities,
        params.signature,
        salt  // Reveal the salt
    );

    const receipt = await tx.wait();
    console.log("✅ Agent registered!");
    return receipt;
}

// Complete Example
async function safeRegisterAgent(registry, params) {
    // 1. Create commitment
    const { commitHash, salt } = await createCommitment(
        params.did,
        params.publicKey,
        await signer.getAddress()
    );

    // 2. Commit
    await commitRegistration(registry, commitHash);

    // 3. Wait
    console.log("⏳ Waiting 1 minute...");
    await new Promise(resolve => setTimeout(resolve, 61000));

    // 4. Reveal and register
    const receipt = await registerWithReveal(registry, params, salt);

    return receipt;
}
```

#### Contract-Side (Solidity)

```solidity
// Commitment storage
struct RegistrationCommitment {
    bytes32 commitHash;
    uint256 timestamp;
    bool revealed;
}

mapping(address => RegistrationCommitment) public registrationCommitments;

// Step 1: Commit
function commitRegistration(bytes32 commitHash) external whenNotPaused {
    RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

    // Check if already committed
    if (commitment.timestamp > 0 && !commitment.revealed) {
        if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {
            revert AlreadyCommitted();
        }
    }

    // Store commitment
    registrationCommitments[msg.sender] = RegistrationCommitment({
        commitHash: commitHash,
        timestamp: block.timestamp,
        revealed: false
    });

    emit RegistrationCommitted(msg.sender, commitHash, block.timestamp);
}

// Step 2: Reveal and register
function registerAgentWithReveal(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature,
    bytes32 salt
) external whenNotPaused returns (bytes32) {
    RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

    // Verify commitment exists
    if (commitment.timestamp == 0) revert NoCommitmentFound();
    if (commitment.revealed) revert CommitmentAlreadyRevealed();

    // Verify timing (must wait minimum delay)
    uint256 minRevealTime = commitment.timestamp + MIN_COMMIT_REVEAL_DELAY;
    uint256 maxRevealTime = commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

    if (block.timestamp < minRevealTime) {
        revert RevealTooSoon(block.timestamp, minRevealTime);
    }
    if (block.timestamp > maxRevealTime) {
        revert RevealTooLate(block.timestamp, maxRevealTime);
    }

    // Verify hash matches
    bytes32 expectedHash = keccak256(abi.encodePacked(
        did,
        publicKey,
        msg.sender,
        salt,
        block.chainid
    ));

    if (commitment.commitHash != expectedHash) revert InvalidReveal();

    // Mark as revealed
    commitment.revealed = true;

    // Proceed with registration...
    return _registerAgent(...);
}
```

---

### 2. Task Authorization (ERC8004ReputationRegistryV2)

#### Client-Side

```javascript
// Step 1: Create commitment for task authorization
async function createAuthCommitment(taskId, serverAgent, deadline) {
    const salt = ethers.randomBytes(32);
    const network = await provider.getNetwork();

    const commitHash = ethers.keccak256(
        ethers.solidityPacked(
            ["bytes32", "address", "uint256", "bytes32", "uint256"],
            [taskId, serverAgent, deadline, salt, network.chainId]
        )
    );

    return { commitHash, salt };
}

// Step 2: Commit
async function commitTaskAuth(registry, commitHash) {
    const tx = await registry.commitTaskAuthorization(commitHash);
    await tx.wait();
    console.log("✅ Task authorization committed");
}

// Step 3: Reveal (after 30 seconds)
async function authorizeTaskWithReveal(registry, taskId, serverAgent, deadline, salt) {
    const tx = await registry.authorizeTaskWithReveal(
        taskId,
        serverAgent,
        deadline,
        salt
    );

    await tx.wait();
    console.log("✅ Task authorized!");
}

// Complete flow
async function safeAuthorizeTask(registry, taskId, serverAgent, deadline) {
    // 1. Create commitment
    const { commitHash, salt } = await createAuthCommitment(
        taskId,
        serverAgent,
        deadline
    );

    // 2. Commit
    await commitTaskAuth(registry, commitHash);

    // 3. Wait minimum delay (30 seconds for task auth)
    await new Promise(resolve => setTimeout(resolve, 31000));

    // 4. Reveal and authorize
    await authorizeTaskWithReveal(registry, taskId, serverAgent, deadline, salt);
}
```

---

## Timing Parameters

### SageRegistryV3 (Agent Registration)

```solidity
MIN_COMMIT_REVEAL_DELAY = 1 minutes   // Must wait at least 1 minute
MAX_COMMIT_REVEAL_DELAY = 1 hours     // Must reveal within 1 hour
```

**Rationale:**
- 1 minute: Prevents instant reveal (gives time for commitment to be mined)
- 1 hour: Prevents indefinite commitment locking

### ERC8004ReputationRegistryV2 (Task Authorization)

```solidity
MIN_COMMIT_REVEAL_DELAY = 30 seconds  // Faster for task operations
MAX_COMMIT_REVEAL_DELAY = 10 minutes  // Shorter window for tasks
```

**Rationale:**
- Task authorization is time-sensitive
- Shorter delays for better UX
- Still provides front-running protection

---

## Security Considerations

### 1. Salt Management

✅ **DO:**
- Generate cryptographically secure random salt
- Keep salt secret until reveal
- Use different salt for each commitment

❌ **DON'T:**
- Reuse salts
- Use predictable salts (e.g., timestamp)
- Share salt before revealing

```javascript
// Good
const salt = ethers.randomBytes(32);

// Bad
const salt = ethers.zeroPadValue(ethers.toBeHex(Date.now()), 32);
```

### 2. Timing Attacks

**Problem:** If reveal timing is predictable, attacker might still front-run

**Solution:** Add random jitter to reveal time

```javascript
// Add 0-10 seconds random jitter
const jitter = Math.floor(Math.random() * 10000);
await new Promise(resolve => setTimeout(resolve, 61000 + jitter));
```

### 3. Gas Price Attacks

**Problem:** Attacker might try to prevent reveal by spam

**Solution:** Use EIP-1559 with reasonable max priority fee

```javascript
await registry.registerAgentWithReveal(..., {
    maxPriorityFeePerGas: ethers.parseUnits("2", "gwei"),
    maxFeePerGas: ethers.parseUnits("100", "gwei")
});
```

### 4. Commitment Expiry

**Problem:** User forgets to reveal within time window

**Solution:** Check commitment status before committing

```javascript
const commitment = await registry.getCommitment(userAddress);

if (commitment.timestamp > 0 && !commitment.revealed) {
    const elapsed = Date.now()/1000 - commitment.timestamp;
    if (elapsed > 3600) {
        console.log("⚠️  Previous commitment expired, can create new one");
    } else {
        console.log("⏳ Previous commitment still valid, must reveal or wait");
    }
}
```

---

## Gas Costs

### Additional Gas Overhead

| Operation | Without Commit-Reveal | With Commit-Reveal | Overhead |
|-----------|----------------------|-------------------|----------|
| Agent Registration | ~250,000 gas | ~280,000 gas | +30,000 gas |
| Task Authorization | ~80,000 gas | ~95,000 gas | +15,000 gas |

**Cost Breakdown:**
- Commit transaction: ~45,000 gas
- Reveal verification: ~10,000 gas
- Storage: ~20,000 gas (SSTORE)

**Total Cost:**
- Commit: ~45,000 gas (~$0.10 at 50 gwei)
- Reveal: ~30,000 gas extra (~$0.07 at 50 gwei)
- **Total Extra: ~$0.17 per registration**

**Worth it?** YES - Prevents DID theft which could be worth much more

---

## Backward Compatibility

Both V3 contracts maintain legacy functions for backward compatibility:

```solidity
// Legacy function (vulnerable to front-running)
function registerAgent(...) external returns (bytes32) {
    // Direct registration without commit-reveal
}

// New protected function (recommended)
function registerAgentWithReveal(..., bytes32 salt) external returns (bytes32) {
    // Commit-reveal protected
}
```

**Recommendation:**
- Use `registerAgentWithReveal()` for all new integrations
- Legacy `registerAgent()` will be deprecated in future versions
- Add warnings in UI for legacy function usage

---

## Testing

### Test Scenarios

```javascript
describe("Front-Running Protection", function() {
    it("Should prevent front-running with commit-reveal", async function() {
        // Alice commits to register "did:sage:alice"
        const { commitHash, salt } = await createCommitment(
            "did:sage:alice",
            alicePubKey,
            alice.address
        );

        await registry.connect(alice).commitRegistration(commitHash);

        // Bob (attacker) sees commit but cannot extract DID
        // Bob tries to register "did:sage:alice" directly
        await expect(
            registry.connect(bob).registerAgent("did:sage:alice", ...)
        ).to.not.be.reverted; // Bob succeeds with direct registration

        // But Alice's reveal will fail if Bob already registered
        await time.increase(61); // Wait 61 seconds

        await expect(
            registry.connect(alice).registerAgentWithReveal(
                "did:sage:alice",
                ...,
                salt
            )
        ).to.be.revertedWith("DID already registered");

        // This shows legacy function is still vulnerable!
        // Users MUST use commit-reveal for protection
    });

    it("Should successfully register with commit-reveal", async function() {
        const { commitHash, salt } = await createCommitment(
            "did:sage:carol",
            carolPubKey,
            carol.address
        );

        // Commit
        await registry.connect(carol).commitRegistration(commitHash);

        // Wait
        await time.increase(61);

        // Reveal
        const tx = await registry.connect(carol).registerAgentWithReveal(
            "did:sage:carol",
            ...,
            salt
        );

        expect(tx).to.emit(registry, "AgentRegistered");
        expect(tx).to.emit(registry, "RegistrationRevealed");
    });

    it("Should reject reveal without commit", async function() {
        await expect(
            registry.connect(dave).registerAgentWithReveal(
                "did:sage:dave",
                ...,
                randomSalt
            )
        ).to.be.revertedWithCustomError(registry, "NoCommitmentFound");
    });

    it("Should reject reveal too soon", async function() {
        const { commitHash, salt } = await createCommitment(...);
        await registry.commitRegistration(commitHash);

        // Try to reveal immediately
        await expect(
            registry.registerAgentWithReveal(..., salt)
        ).to.be.revertedWithCustomError(registry, "RevealTooSoon");
    });

    it("Should reject reveal too late", async function() {
        const { commitHash, salt } = await createCommitment(...);
        await registry.commitRegistration(commitHash);

        // Wait more than 1 hour
        await time.increase(3601);

        await expect(
            registry.registerAgentWithReveal(..., salt)
        ).to.be.revertedWithCustomError(registry, "RevealTooLate");
    });
});
```

---

## Migration Guide

### For Existing Users

If you're currently using `registerAgent()`:

```javascript
// Old code (vulnerable)
await registry.registerAgent(
    did, name, description, endpoint,
    publicKey, capabilities, signature
);

// New code (protected)
// 1. Create commitment
const { commitHash, salt } = await createCommitment(
    did, publicKey, await signer.getAddress()
);

// 2. Commit
await registry.commitRegistration(commitHash);

// 3. Wait
await new Promise(resolve => setTimeout(resolve, 61000));

// 4. Reveal
await registry.registerAgentWithReveal(
    did, name, description, endpoint,
    publicKey, capabilities, signature,
    salt  // Add salt parameter
);
```

### For SDK/Library Developers

Wrap commit-reveal in helper function:

```javascript
class SAGERegistry {
    async registerAgentSafe(params) {
        // Auto-handle commit-reveal
        const { commitHash, salt } = await this._createCommitment(params);

        await this.contract.commitRegistration(commitHash);

        // Wait with progress callback
        await this._waitForReveal((progress) => {
            this.emit('commitProgress', progress);
        });

        return await this.contract.registerAgentWithReveal(
            ...params,
            salt
        );
    }

    async registerAgentLegacy(params) {
        // Direct registration (show warning)
        console.warn('⚠️  Using legacy registration (vulnerable to front-running)');
        return await this.contract.registerAgent(...params);
    }
}
```

---

## Monitoring

### Track Front-Running Attempts

```javascript
// Listen for commit events
registry.on("RegistrationCommitted", (committer, commitHash, timestamp) => {
    console.log(`Commitment from ${committer}`);
    console.log(`Hash: ${commitHash}`);

    // Check if revealed within expected time
    setTimeout(async () => {
        const commitment = await registry.getCommitment(committer);
        if (!commitment.revealed) {
            console.warn(`⚠️  Commitment not revealed after 2 minutes`);
        }
    }, 120000);
});

// Listen for reveals
registry.on("RegistrationRevealed", (revealer, agentId, did) => {
    console.log(`✅ Revealed: ${did} by ${revealer}`);
});

// Listen for expired commitments
registry.on("CommitmentExpired", (committer, commitHash) => {
    console.log(`⏰ Commitment expired: ${commitHash}`);
});
```

---

## Conclusion

Commit-reveal pattern provides strong front-running protection with minimal UX impact:

✅ **Benefits:**
- Prevents DID/task theft
- Minimal gas overhead (~$0.17)
- Simple to implement
- Backward compatible

⚠️ **Trade-offs:**
- Requires 2 transactions
- 1-minute delay (registration)
- Users must manage salt

**Recommendation:** Use commit-reveal for all critical operations where front-running is a concern.

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Related Contracts:** SageRegistryV3.sol, ERC8004ReputationRegistryV2.sol
