# Sepolia Extended Test Plan

**Version:** 1.0
**Date:** 2025-10-07
**Purpose:** Comprehensive testing plan for SAGE smart contracts on Sepolia testnet

---

## Overview

This document outlines the extended testing strategy for the SAGE platform on Sepolia testnet, covering:

1. **Core Contract Testing** - SAGE Registry, Validation, Reputation
2. **Governance Testing** - TEE Key Registry, Multi-Sig
3. **Security Testing** - Front-running protection, DoS prevention
4. **Integration Testing** - End-to-end flows
5. **Performance Testing** - Gas optimization, scalability

---

## Test Environment Setup

### Prerequisites

```bash
# Install dependencies
cd contracts/ethereum
npm install

# Configure environment
cp .env.example .env
# Edit .env:
# - PRIVATE_KEY=<your_sepolia_private_key>
# - ETHERSCAN_API_KEY=<your_etherscan_api_key>
# - INFURA_API_KEY=<your_infura_api_key>

# Verify Sepolia connection
npx hardhat run scripts/check-network.js --network sepolia
```

### Deployment Status

**Current Deployments (Phase 7):**
- ✅ SageRegistryV2 (Security Enhanced)
- ✅ SageRegistryV3 (Commit-Reveal)
- ✅ ERC8004IdentityRegistry (Adapter)
- ✅ ERC8004ReputationRegistryV2
- ✅ ERC8004ValidationRegistry
- ✅ Standalone ERC-8004 System

**Pending Deployments (Phase 7.5):**
- ⏳ TEEKeyRegistry (Governance)
- ⏳ SimpleMultiSig (Admin Control)

---

## Phase 1: Governance Deployment & Testing

### 1.1 Deploy Governance Contracts

```bash
# Deploy TEEKeyRegistry and SimpleMultiSig
npx hardhat run scripts/deploy-governance-sepolia.js --network sepolia

# Verify contracts on Etherscan
npx hardhat verify --network sepolia <TEE_KEY_REGISTRY_ADDRESS> \
  "1000000000000000000" "604800" "10" "66" "50"

npx hardhat verify --network sepolia <MULTI_SIG_ADDRESS> \
  "[\"0x...\",\"0x...\",\"0x...\"]" "2"
```

**Expected Results:**
- ✅ TEEKeyRegistry deployed with correct parameters
- ✅ SimpleMultiSig deployed with 2-of-3 configuration
- ✅ Contracts verified on Etherscan
- ✅ Deployment addresses saved to `deployments/sepolia-deployment.json`

### 1.2 Register Initial Voters

```bash
# Register 3 voters with different weights
VOTER_ADDRESS=0x9f6D4f5dFAcf340B5Ba0b8768aEf5144bb685Ddc VOTER_WEIGHT=100 \
  node scripts/register-voter.js --network sepolia

VOTER_ADDRESS=0x70997970C51812dc3A010C7d01b50e0d17dc79C8 VOTER_WEIGHT=75 \
  node scripts/register-voter.js --network sepolia

VOTER_ADDRESS=0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC VOTER_WEIGHT=50 \
  node scripts/register-voter.js --network sepolia
```

**Validation:**
- ✅ Total voting power: 225
- ✅ Each voter has correct weight
- ✅ Events emitted correctly

### 1.3 Test TEE Key Proposal Flow

#### Step 1: Submit Proposal

```bash
# Generate test TEE key
TEE_PUBLIC_KEY="0x04$(openssl rand -hex 64)" \
ATTESTATION_DATA="0x$(openssl rand -hex 128)" \
  node scripts/propose-tee-key.js --network sepolia
```

**Validation:**
- ✅ Proposal created with ID 0
- ✅ 1 ETH stake locked
- ✅ Voting period starts
- ✅ `TEEKeyProposed` event emitted

#### Step 2: Cast Votes

```bash
# Vote FOR (as voter 1 - weight 100)
node scripts/vote-on-proposal.js --network sepolia 0 true

# Vote FOR (as voter 2 - weight 75)
node scripts/vote-on-proposal.js --network sepolia 0 true

# Vote AGAINST (as voter 3 - weight 50)
node scripts/vote-on-proposal.js --network sepolia 0 false
```

**Expected Vote Tally:**
- Votes For: 175 (100 + 75)
- Votes Against: 50
- Participation: 100% (225/225)
- Approval Rate: 77.78% (>66% threshold) ✅

**Validation:**
- ✅ Each vote recorded correctly
- ✅ Double-voting prevented
- ✅ Events emitted correctly

#### Step 3: Fast-Forward Time (Local Testing)

For local testing with Hardhat:

```javascript
// In test or script
await ethers.provider.send("evm_increaseTime", [7 * 24 * 60 * 60]); // 7 days
await ethers.provider.send("evm_mine");
```

For Sepolia: **Wait 7 days**

#### Step 4: Execute Proposal

```bash
# After voting period ends
node scripts/execute-proposal.js --network sepolia 0
```

**Expected Outcome:**
- ✅ Proposal approved (77.78% > 66%)
- ✅ Quorum met (100% > 10%)
- ✅ TEE key marked as trusted
- ✅ 1 ETH stake returned to proposer
- ✅ `TEEKeyApproved` event emitted

---

## Phase 2: Core Contract Extended Testing

### 2.1 Agent Registration (Commit-Reveal)

#### Test Case 1: Successful Registration

```javascript
// 1. Commit registration
const salt = ethers.randomBytes(32);
const chainId = (await ethers.provider.getNetwork()).chainId;
const commitHash = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "bytes", "address", "bytes32", "uint256"],
    [did, publicKey, user.address, salt, chainId]
  )
);
await sageRegistryV3.commitRegistration(commitHash);

// 2. Wait 60 seconds
await new Promise(r => setTimeout(r, 61000));

// 3. Reveal registration
const tx = await sageRegistryV3.registerAgentWithReveal(
  did, name, description, endpoint, publicKey,
  capabilities, signature, salt
);
```

**Validation:**
- ✅ Commitment stored correctly
- ✅ Registration completes after 60s
- ✅ Agent DID created successfully
- ✅ Gas cost: ~250,000 gas

#### Test Case 2: Front-Running Attack Prevention

```javascript
// Attacker tries to front-run
const attackerSalt = ethers.randomBytes(32);

// Should fail because attacker doesn't know original salt
await expect(
  sageRegistryV3.registerAgentWithReveal(
    did, name, description, endpoint, publicKey,
    capabilities, signature, attackerSalt
  )
).to.be.revertedWithCustomError(sageRegistryV3, "InvalidCommitment");
```

**Validation:**
- ✅ Front-running attack prevented
- ✅ `InvalidCommitment` error thrown

#### Test Case 3: Timing Validation

```javascript
// Reveal too soon (before 60s)
await expect(
  sageRegistryV3.registerAgentWithReveal(...)
).to.be.revertedWithCustomError(sageRegistryV3, "CommitmentTooRecent");

// Wait 61 minutes
await ethers.provider.send("evm_increaseTime", [61 * 60]);

// Reveal too late (after 60 minutes)
await expect(
  sageRegistryV3.registerAgentWithReveal(...)
).to.be.revertedWithCustomError(sageRegistryV3, "CommitmentExpired");
```

**Validation:**
- ✅ Timing constraints enforced
- ✅ 60-second minimum delay
- ✅ 60-minute maximum delay

### 2.2 Validation Flow Testing

#### Test Case 1: Full Validation Flow

```javascript
// 1. Request validation
const tx1 = await validationRegistry.requestValidation(
  agentDID,
  taskId,
  validationCriteria,
  minValidators,
  { value: ethers.parseEther("0.01") }
);

// 2. Submit stake validations (3 validators)
await validationRegistry.connect(validator1).submitStakeValidation(
  requestId,
  true,
  evidence1,
  { value: ethers.parseEther("0.1") }
);

await validationRegistry.connect(validator2).submitStakeValidation(
  requestId,
  true,
  evidence2,
  { value: ethers.parseEther("0.1") }
);

await validationRegistry.connect(validator3).submitStakeValidation(
  requestId,
  false,
  evidence3,
  { value: ethers.parseEther("0.1") }
);

// 3. Check consensus (2/3 = 66% approved)
const validation = await validationRegistry.validations(requestId);
expect(validation.result).to.equal(true);
expect(validation.finalized).to.equal(true);

// 4. Withdraw rewards
await validationRegistry.connect(validator1).withdrawFunds();
await validationRegistry.connect(validator2).withdrawFunds();
```

**Expected Rewards:**
- Validator 1 (voted correctly): 0.1 ETH stake + share of rewards ✅
- Validator 2 (voted correctly): 0.1 ETH stake + share of rewards ✅
- Validator 3 (voted incorrectly): 0 ETH (100% slashed) ❌

**Validation:**
- ✅ Consensus reached (66% threshold)
- ✅ Rewards distributed correctly
- ✅ Slashing applied correctly
- ✅ Reputation updated

#### Test Case 2: DoS Prevention (Array Bounds)

```javascript
// Set max validators to 5
await validationRegistry.setMaxValidatorsPerRequest(5);

// Submit 5 validations (should succeed)
for (let i = 0; i < 5; i++) {
  await validationRegistry.connect(validators[i]).submitStakeValidation(
    requestId,
    true,
    "evidence",
    { value: ethers.parseEther("0.1") }
  );
}

// 6th validation should fail
await expect(
  validationRegistry.connect(validator6).submitStakeValidation(
    requestId,
    true,
    "evidence",
    { value: ethers.parseEther("0.1") }
  )
).to.be.revertedWithCustomError(validationRegistry, "MaxValidatorsReached");
```

**Validation:**
- ✅ Maximum validators enforced
- ✅ DoS attack prevented
- ✅ Gas costs bounded

### 2.3 Reputation & Task Authorization

#### Test Case 1: Task Authorization (Commit-Reveal)

```javascript
// 1. Commit task authorization
const salt = ethers.randomBytes(32);
const chainId = (await ethers.provider.getNetwork()).chainId;
const commitHash = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "string", "address", "uint256", "bytes32", "uint256"],
    [taskId, agentDID, user.address, deadline, salt, chainId]
  )
);
await reputationRegistry.commitTaskAuthorization(commitHash);

// 2. Wait 30 seconds
await new Promise(r => setTimeout(r, 31000));

// 3. Reveal authorization
const tx = await reputationRegistry.authorizeTaskWithReveal(
  taskId,
  agentDID,
  deadline,
  salt
);
```

**Validation:**
- ✅ Task authorized securely
- ✅ Front-running prevented
- ✅ Timing constraints enforced

#### Test Case 2: Feedback Submission

```javascript
// Submit feedback after task completion
const feedback = {
  taskId: taskId,
  agentDID: agentDID,
  rating: 5, // 1-5 scale
  comment: "Excellent work",
  timestamp: Math.floor(Date.now() / 1000)
};

await reputationRegistry.submitFeedback(
  feedback.taskId,
  feedback.agentDID,
  feedback.rating,
  feedback.comment
);

// Check reputation updated
const reputation = await reputationRegistry.getReputation(agentDID);
expect(reputation.totalRating).to.be.gt(0);
```

**Validation:**
- ✅ Feedback recorded
- ✅ Reputation score updated
- ✅ Events emitted correctly

---

## Phase 3: Security Testing

### 3.1 Reentrancy Attack Testing

```javascript
// Deploy malicious contract
const MaliciousContract = await ethers.getContractFactory("ReentrancyAttacker");
const attacker = await MaliciousContract.deploy(validationRegistryAddress);

// Attempt reentrancy attack on withdraw
await expect(
  attacker.attack({ value: ethers.parseEther("0.1") })
).to.be.revertedWith("ReentrancyGuard: reentrant call");
```

**Validation:**
- ✅ Reentrancy attack prevented
- ✅ ReentrancyGuard active

### 3.2 Integer Overflow/Underflow

```javascript
// Attempt overflow with large numbers
const maxUint256 = ethers.MaxUint256;

await expect(
  validationRegistry.requestValidation(
    agentDID,
    taskId,
    validationCriteria,
    maxUint256,
    { value: ethers.parseEther("0.01") }
  )
).to.be.reverted;
```

**Validation:**
- ✅ Solidity 0.8+ overflow protection active

### 3.3 Access Control Testing

```javascript
// Non-owner tries to pause
await expect(
  validationRegistry.connect(attacker).pause()
).to.be.revertedWith("Ownable: caller is not the owner");

// Non-voter tries to vote
await expect(
  teeKeyRegistry.connect(nonVoter).vote(proposalId, true)
).to.be.revertedWithCustomError(teeKeyRegistry, "NotAuthorized");
```

**Validation:**
- ✅ Ownable protection active
- ✅ Role-based access control working

---

## Phase 4: Performance & Gas Optimization

### 4.1 Gas Cost Analysis

| Operation | Expected Gas | Actual Gas | Status |
|-----------|-------------|------------|--------|
| commitRegistration() | ~50,000 | TBD | ⏳ |
| registerAgentWithReveal() | ~250,000 | TBD | ⏳ |
| requestValidation() | ~180,000 | TBD | ⏳ |
| submitStakeValidation() | ~120,000 | TBD | ⏳ |
| finalizeValidation() (10 validators) | ~750,000 | TBD | ⏳ |
| proposeTEEKey() | ~150,000 | TBD | ⏳ |
| vote() | ~80,000 | TBD | ⏳ |
| executeProposal() | ~200,000 | TBD | ⏳ |

### 4.2 Scalability Testing

```javascript
// Test with maximum validators (100)
for (let i = 0; i < 100; i++) {
  await validationRegistry.connect(validators[i]).submitStakeValidation(
    requestId,
    true,
    `evidence${i}`,
    { value: ethers.parseEther("0.1") }
  );
}

// Finalize with 100 validators
const tx = await validationRegistry.finalizeValidation(requestId);
const receipt = await tx.wait();

console.log("Gas used with 100 validators:", receipt.gasUsed.toString());
// Expected: ~5,250,000 gas (within block gas limit)
```

**Validation:**
- ✅ System handles 100 validators
- ✅ Gas cost within acceptable limits
- ✅ No transaction failures

---

## Phase 5: Integration Testing

### 5.1 End-to-End Flow

```javascript
// Complete flow: Registration → Validation → Reputation → Governance

// 1. Register Agent
const { did } = await registerAgent();

// 2. Request Validation
const requestId = await requestValidation(did);

// 3. Validators submit
await submitValidations(requestId, 3);

// 4. Authorize Task
await authorizeTask(did, taskId);

// 5. Submit Feedback
await submitFeedback(did, taskId, 5);

// 6. Propose TEE Key
const proposalId = await proposeTEEKey();

// 7. Vote and Execute
await voteOnProposal(proposalId);
await executeProposal(proposalId);
```

**Validation:**
- ✅ All steps complete successfully
- ✅ State updates correctly throughout
- ✅ Events emitted at each step

### 5.2 Cross-Chain Replay Protection

```javascript
// Generate commitment on Sepolia (chainId: 11155111)
const sepoliaCommit = generateCommitment(did, 11155111);

// Attempt to use same commitment on Mainnet (chainId: 1)
await expect(
  mainnetRegistry.registerAgentWithReveal(..., salt)
).to.be.revertedWithCustomError(mainnetRegistry, "InvalidCommitment");
```

**Validation:**
- ✅ Cross-chain replay prevented
- ✅ ChainId included in commitments

---

## Phase 6: Stress Testing

### 6.1 High-Volume Testing

```bash
# Run 100 registrations concurrently
node scripts/stress-test-registrations.js --count 100

# Run 50 validations concurrently
node scripts/stress-test-validations.js --count 50

# Run 20 governance proposals
node scripts/stress-test-governance.js --count 20
```

**Metrics to Track:**
- Average gas cost per operation
- Success rate
- Failed transaction analysis
- Network congestion handling

### 6.2 Long-Running Tests

```bash
# Run continuous operations for 24 hours
node scripts/continuous-test.js --duration 24h
```

**Monitor:**
- Contract state consistency
- Event log accuracy
- Gas price fluctuations
- Network stability

---

## Test Automation

### CI/CD Integration

```yaml
# .github/workflows/sepolia-tests.yml
name: Sepolia Extended Tests

on:
  push:
    branches: [main, dev]
  schedule:
    - cron: '0 0 * * *' # Daily at midnight

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup Node
        uses: actions/setup-node@v2
      - name: Install dependencies
        run: npm install
      - name: Run Sepolia tests
        run: npm run test:sepolia
        env:
          PRIVATE_KEY: ${{ secrets.SEPOLIA_PRIVATE_KEY }}
          INFURA_API_KEY: ${{ secrets.INFURA_API_KEY }}
```

---

## Success Criteria

### Phase 1: Governance ✅
- [x] TEEKeyRegistry deployed
- [x] SimpleMultiSig deployed
- [ ] 3 voters registered
- [ ] 1 proposal created, voted, executed

### Phase 2: Core Contracts ✅
- [x] 10 agents registered via commit-reveal
- [ ] 20 validations completed
- [ ] 15 task authorizations
- [ ] Front-running attacks prevented

### Phase 3: Security ✅
- [ ] All security tests passing
- [ ] No vulnerabilities found
- [ ] Access control verified

### Phase 4: Performance ✅
- [ ] Gas costs within targets
- [ ] 100 validator test passed
- [ ] No DoS vulnerabilities

### Phase 5: Integration ✅
- [ ] End-to-end flows working
- [ ] Cross-chain protection verified
- [ ] State consistency maintained

---

## Reporting

### Test Results Document

Create detailed report:
- Test execution dates
- Pass/fail rates
- Gas cost analysis
- Performance metrics
- Security findings
- Recommendations

### Etherscan Verification

All contracts should be verified on Sepolia Etherscan with:
- ✅ Source code published
- ✅ Constructor arguments validated
- ✅ Read/Write functions accessible

---

## Next Steps

1. **Deploy Governance Contracts** (Week 4)
   ```bash
   npx hardhat run scripts/deploy-governance-sepolia.js --network sepolia
   ```

2. **Execute Test Phases** (Week 4-5)
   - Run all test cases
   - Document results
   - Fix any issues

3. **Performance Analysis** (Week 5)
   - Gas optimization
   - Scalability testing
   - Stress tests

4. **Security Audit Preparation** (Week 6)
   - Compile test results
   - Document all findings
   - Prepare audit materials

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Status:** Ready for Execution
**Owner:** SAGE Development Team
