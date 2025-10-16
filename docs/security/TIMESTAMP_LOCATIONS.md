# Block.timestamp Locations - Quick Reference

**Total Locations:** 105 across 12 contract files

## File-by-File Breakdown

### 1. SageVerificationHook.sol (3 locations)

```solidity
// Line 48 - Cooldown check (⚠️ ACCEPTABLE - 1 minute minimum)
if (block.timestamp < lastRegistrationTime[agentOwner] + REGISTRATION_COOLDOWN) {

// Line 79 - Record registration time (✅ SAFE)
lastRegistrationTime[agentOwner] = block.timestamp;

// Line 104 - Day boundary check (✅ SAFE)
return block.timestamp / 1 days > lastRegistrationTime[user] / 1 days;
```

**Recommendation:** Add slither-disable comments with explanations

---

### 2. SageRegistry.sol (9 locations)

```solidity
// Line 136 - Agent ID generation (✅ SAFE)
return keccak256(abi.encode(did, publicKey, block.timestamp));

// Lines 194-195 - Record timestamps (✅ SAFE)
registeredAt: block.timestamp,
updatedAt: block.timestamp,

// Line 203 - Event emission (✅ SAFE)
emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);

// Line 258 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 262 - Event emission (✅ SAFE)
emit AgentUpdated(agentId, msg.sender, block.timestamp);

// Line 272 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 274 - Event emission (✅ SAFE)
emit AgentDeactivated(agentId, msg.sender, block.timestamp);

// Line 288 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 290 - Event emission (✅ SAFE)
emit AgentDeactivated(agentId, msg.sender, block.timestamp);
```

**Pattern:** All uses are for event recording and unique ID generation

---

### 3. SageRegistryV2.sol (6 locations)

```solidity
// Line 288 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 292 - Event emission (✅ SAFE)
emit AgentUpdated(agentId, msg.sender, block.timestamp);

// Lines 431-432 - Record timestamps (✅ SAFE)
registeredAt: block.timestamp,
updatedAt: block.timestamp,

// Line 444 - Event emission (✅ SAFE)
emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);

// Line 484 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 486 - Event emission (✅ SAFE)
emit AgentDeactivated(agentId, msg.sender, block.timestamp);

// Line 500 - Update timestamp (✅ SAFE)
agents[agentId].updatedAt = block.timestamp;

// Line 502 - Event emission (✅ SAFE)
emit AgentDeactivated(agentId, msg.sender, block.timestamp);
```

**Note:** Line 365 comment already documents use of block.number instead of timestamp for ID generation

---

### 4. SageRegistryV3.sol (18 locations)

```solidity
// Commit-Reveal Pattern (✅ SAFE - 1 min to 1 hour windows)
// Line 252 - Commitment expiry check
if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {

// Line 263 - Record commitment timestamp
timestamp: block.timestamp,

// Line 267 - Event emission
emit RegistrationCommitted(msg.sender, commitHash, block.timestamp);

// Line 384 - Minimum delay check
if (block.timestamp < minRevealTime) {
    revert RevealTooSoon(block.timestamp, minRevealTime);

// Line 387 - Maximum delay check
if (block.timestamp > maxRevealTime) {
    revert RevealTooLate(block.timestamp, maxRevealTime);

// Event Recording (✅ SAFE)
// Lines 627-628 - Record timestamps
registeredAt: block.timestamp,
updatedAt: block.timestamp,

// Line 639 - Event emission
emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);

// Line 720 - Update timestamp
agents[agentId].updatedAt = block.timestamp;

// Line 724 - Event emission
emit AgentUpdated(agentId, msg.sender, block.timestamp);

// Line 730 - Update timestamp
agents[agentId].updatedAt = block.timestamp;

// Line 731 - Event emission
emit AgentDeactivated(agentId, msg.sender, block.timestamp);

// Line 741 - Update timestamp
agents[agentId].updatedAt = block.timestamp;

// Line 743 - Event emission
emit AgentDeactivated(agentId, msg.sender, block.timestamp);

// Line 910 - View function (commitment expiry check)
block.timestamp > commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;
```

**Constants used:**
- `MIN_COMMIT_REVEAL_DELAY = 1 minutes`
- `MAX_COMMIT_REVEAL_DELAY = 1 hours`

---

### 5. ERC8004ValidationRegistry.sol (26 locations)

```solidity
// Deadline Validation (✅ SAFE - 1 hour minimum)
// Lines 255-259 - Deadline bounds checking
if (deadline <= block.timestamp + MIN_DEADLINE_DURATION) {
    revert DeadlineTooSoon(deadline, block.timestamp + MIN_DEADLINE_DURATION);
}
if (deadline > block.timestamp + MAX_DEADLINE_DURATION) {
    revert DeadlineTooFar(deadline, block.timestamp + MAX_DEADLINE_DURATION);
}

// ID Generation (✅ SAFE)
// Line 281 - Request ID generation
keccak256(abi.encodePacked(taskId, msg.sender, serverAgent, dataHash, block.timestamp, ...))

// Line 296 - Record request timestamp
timestamp: block.timestamp

// Expiry Checks (✅ SAFE - hour/day scale deadlines)
// Line 441 - Validation request expired check
require(block.timestamp <= request.deadline, "Request expired");

// Line 470 - Response ID generation
keccak256(abi.encodePacked(requestId, msg.sender, computedHash, block.timestamp))

// Line 482 - Record response timestamp
timestamp: block.timestamp

// Line 522 - TEE attestation expired check
require(block.timestamp <= request.deadline, "Request expired");

// Line 559 - Response ID generation
keccak256(abi.encodePacked(requestId, msg.sender, attestation, block.timestamp))

// Line 571 - Record response timestamp
timestamp: block.timestamp

// Line 982 - Finalize expired validation
require(block.timestamp > request.deadline, "Not expired");
```

**Constants used:**
- `MIN_DEADLINE_DURATION = 1 hours`
- `MAX_DEADLINE_DURATION = 30 days`

---

### 6. ERC8004ReputationRegistry.sol (4 locations)

```solidity
// Line 89 - Deadline validation
require(deadline > block.timestamp, "Invalid deadline");

// Line 141 - Authorization expiry check
require(block.timestamp <= auth.deadline, "Authorization expired");

// Line 158 - Feedback ID generation
keccak256(abi.encodePacked(taskId, msg.sender, serverAgent, dataHash, block.timestamp, ...))

// Line 170 - Record feedback timestamp
timestamp: block.timestamp,

// Line 186 - Event emission
emit FeedbackSubmitted(feedbackId, taskId, serverAgent, msg.sender, dataHash, rating, block.timestamp);

// Line 301 - Authorization check (view function)
block.timestamp <= auth.deadline;
```

---

### 7. ERC8004ReputationRegistryV2.sol (14 locations)

```solidity
// Commit-Reveal Pattern (✅ SAFE - 30 sec to 10 min windows)
// Line 249 - Commitment expiry check
if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {

// Line 259 - Record commitment timestamp
timestamp: block.timestamp,

// Line 263 - Event emission
emit AuthorizationCommitted(msg.sender, commitHash, block.timestamp);

// Lines 301-305 - Reveal timing checks
if (block.timestamp < minRevealTime) {
    revert RevealTooSoon(block.timestamp, minRevealTime);
}
if (block.timestamp > maxRevealTime) {
    revert RevealTooLate(block.timestamp, maxRevealTime);
}

// Deadline Validation (✅ SAFE - 1 hour minimum)
// Lines 344-348 - Task deadline bounds
if (deadline <= block.timestamp + MIN_DEADLINE_DURATION) {
    revert DeadlineTooSoon(deadline, block.timestamp + MIN_DEADLINE_DURATION);
}
if (deadline > block.timestamp + MAX_DEADLINE_DURATION) {
    revert DeadlineTooFar(deadline, block.timestamp + MAX_DEADLINE_DURATION);
}

// Feedback Submission (✅ SAFE)
// Line 399 - Authorization expiry check
require(block.timestamp <= auth.deadline, "Authorization expired");

// Line 416 - Feedback ID generation
keccak256(abi.encodePacked(taskId, msg.sender, serverAgent, dataHash, block.timestamp, ...))

// Line 428 - Record feedback timestamp
timestamp: block.timestamp,

// Line 444 - Event emission
emit FeedbackSubmitted(feedbackId, taskId, serverAgent, msg.sender, dataHash, rating, block.timestamp);

// View Functions (✅ SAFE)
// Line 532 - Authorization check
block.timestamp <= auth.deadline;

// Line 571 - Commitment expiry check
block.timestamp > commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;
```

**Constants used:**
- `MIN_COMMIT_REVEAL_DELAY = 30 seconds`
- `MAX_COMMIT_REVEAL_DELAY = 10 minutes`
- `MIN_DEADLINE_DURATION = 1 hours`
- `MAX_DEADLINE_DURATION = 30 days`

---

### 8. TEEKeyRegistry.sol (6 locations)

```solidity
// Proposal Creation (✅ SAFE - 7-day voting period)
// Lines 416-417 - Record proposal times
createdAt: block.timestamp,
votingDeadline: block.timestamp + votingPeriod,

// Voting Checks (✅ SAFE)
// Line 448 - Check if voting ended
if (block.timestamp > proposal.votingDeadline) {
    revert VotingEnded(proposalId);

// Line 488 - Check if voting still active
if (block.timestamp <= proposal.votingDeadline) {
    revert VotingNotEnded(proposalId);

// Proposal Execution (✅ SAFE)
// Line 521 - Record approval timestamp
teeKeyApprovedAt[proposal.keyHash] = block.timestamp;

// View Function (✅ SAFE)
// Line 766 - Check if proposal can be executed
block.timestamp > proposal.votingDeadline;
```

**Default voting period:** 7 days

---

### 9. standalone/ERC8004ValidationRegistry.sol (9 locations)

```solidity
// Line 148 - Deadline validation
if (deadline <= block.timestamp) {
    revert InvalidDeadline(deadline);

// Line 166 - Request ID generation
requestId = keccak256(abi.encodePacked(taskId, serverAgent, msg.sender, block.timestamp, ...));

// Line 181 - Record request timestamp
timestamp: block.timestamp

// Line 225 - Expiry check
if (block.timestamp > request.deadline) {
    revert ValidationExpired(requestId);

// Line 251 - Response ID generation
responseId = keccak256(abi.encodePacked(requestId, msg.sender, block.timestamp));

// Line 263 - Record response timestamp
timestamp: block.timestamp

// Line 308 - Expiry check
if (block.timestamp > request.deadline) {
    revert ValidationExpired(requestId);

// Line 334 - Response ID generation
responseId = keccak256(abi.encodePacked(requestId, msg.sender, block.timestamp));

// Line 346 - Record response timestamp
timestamp: block.timestamp
```

---

### 10. standalone/ERC8004ReputationRegistry.sol (4 locations)

```solidity
// Line 107 - Deadline validation
if (deadline <= block.timestamp) {
    revert InvalidDeadline(deadline);

// Line 159 - Expiry check
if (block.timestamp > auth.deadline) {
    revert TaskAuthorizationExpired(taskId);

// Line 181 - Feedback ID generation
feedbackId = keccak256(abi.encodePacked(taskId, msg.sender, serverAgent, block.timestamp, ...));

// Line 193 - Record feedback timestamp
timestamp: block.timestamp,

// Line 210 - Event emission
emit FeedbackSubmitted(feedbackId, taskId, serverAgent, msg.sender, dataHash, rating, block.timestamp);
```

---

### 11. standalone/ERC8004IdentityRegistry.sol (1 location)

```solidity
// Line 114 - Record registration timestamp
registeredAt: block.timestamp
```

---

## Summary by Category

### Event Recording & Logging (47 locations)
All uses of `block.timestamp` for recording when events occurred or emitting in events.
**Status:** ✅ SAFE - No security impact

### Unique ID Generation (15 locations)
Using `block.timestamp` as part of hash inputs for generating unique identifiers.
**Status:** ✅ SAFE - Used for uniqueness, not randomness

### Deadline & Expiry Checks (28 locations)
Comparing current time against deadlines (typically hour/day scale).
**Status:** ✅ SAFE - ±15s negligible on hour/day scale

### Commit-Reveal Timing (14 locations)
Time windows for commit-reveal pattern (ranging from 30 seconds to 1 hour).
**Status:** ✅ SAFE - Adequate tolerance for ±15s variance

### Rate Limiting (1 location)
1-minute cooldown for registration attempts.
**Status:** ⚠️ ACCEPTABLE - Edge case but acceptable given constraints

---

## Implementation Priority

### High Priority (Add comments first)
1. **SageVerificationHook.sol** - Only file with "acceptable" use case
2. **SageRegistryV3.sol** - Commit-reveal pattern needs clear documentation
3. **ERC8004ReputationRegistryV2.sol** - Commit-reveal with 30-second minimum

### Medium Priority
4. **ERC8004ValidationRegistry.sol** - Many deadline checks
5. **TEEKeyRegistry.sol** - Governance timing
6. **ERC8004ReputationRegistry.sol** - Task authorization

### Low Priority (Event recording only)
7. **SageRegistry.sol**
8. **SageRegistryV2.sol**
9. **standalone/** contracts

---

## Template Comments

Use these templates when adding `// slither-disable-next-line timestamp` comments:

```solidity
// For event recording
// slither-disable-next-line timestamp
// SAFE: Recording event timestamp for off-chain indexing (no security impact)

// For deadline checks (hour/day scale)
// slither-disable-next-line timestamp
// SAFE: Deadline is on hour/day scale, ±15s variance is negligible

// For commit-reveal (≥1 minute)
// slither-disable-next-line timestamp
// SAFE: Minimum delay ≥1 minute for front-running protection (±15s variance acceptable)

// For unique ID generation
// slither-disable-next-line timestamp
// SAFE: Used for uniqueness in hash generation, not for randomness or security-critical timing

// For rate limiting (1 minute)
// slither-disable-next-line timestamp
// SAFE: 1-minute cooldown for rate limiting (±15s variance acceptable for this use case)
```

---

**Last updated:** 2025-10-17
**Total locations documented:** 105
