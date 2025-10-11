# SAGE Contracts Improvement Roadmap
## From MEDIUM Risk to LOW Risk (Mainnet Ready)

**Created:** 2025-10-07
**Current Risk Level:** 🟡 MEDIUM
**Target Risk Level:** 🟢 LOW
**Estimated Timeline:** 10-16 weeks

---

## Executive Summary

This roadmap outlines the remaining work required to move from MEDIUM risk (testnet ready) to LOW risk (mainnet ready). The work is organized into 5 parallel tracks that can be executed concurrently to optimize timeline.

**Total Tasks:** 15 major items
**Priority Levels:** P0 (Blocking), P1 (Critical), P2 (Important)

---

## Track 1: Governance Infrastructure (P0 - BLOCKING)
**Timeline:** 2-3 weeks
**Owner:** DevOps + Smart Contract Team

### Task 1.1: Deploy Multi-sig Wallet ⏳ PENDING
**Priority:** P0 - BLOCKING MAINNET
**Estimated Time:** 3-5 days

**Requirements:**
- Deploy Gnosis Safe on target network
- Configure 3/5 signature threshold
- Identify 5 trusted signers
- Distribute hardware wallets
- Test signature collection flow
- Document emergency procedures

**Steps:**
```bash
# 1. Deploy Gnosis Safe
npm install @gnosis.pm/safe-contracts

# 2. Create deployment script
npx hardhat run scripts/deploy-multisig.js --network sepolia

# 3. Verify contract
npx hardhat verify --network sepolia <SAFE_ADDRESS>

# 4. Test transaction flow
npx hardhat run scripts/test-multisig.js
```

**Deliverables:**
- [ ] Gnosis Safe deployed and verified
- [ ] 5 signers identified with hardware wallets
- [ ] Multi-sig test transactions successful
- [ ] Emergency contact list created
- [ ] Documentation: `/docs/MULTISIG-OPERATIONS.md`

**Acceptance Criteria:**
- 3/5 threshold enforced
- Test transaction executed successfully
- All signers confirmed access
- Emergency procedures documented

---

### Task 1.2: Deploy Timelock Controller ⏳ PENDING
**Priority:** P0 - BLOCKING MAINNET
**Estimated Time:** 2-3 days

**Requirements:**
- Deploy OpenZeppelin TimelockController
- Configure 48-hour delay for parameter changes
- Configure 24-hour delay for emergency actions
- Set multi-sig as proposer and executor
- Test timelock flow

**Steps:**
```solidity
// 1. Deploy Timelock
import "@openzeppelin/contracts/governance/TimelockController.sol";

TimelockController timelock = new TimelockController(
    2 days,                    // minDelay for normal operations
    [multiSigAddress],         // proposers (multi-sig)
    [multiSigAddress],         // executors (multi-sig)
    address(0)                 // admin (renounced after setup)
);

// 2. Grant roles
bytes32 PROPOSER_ROLE = timelock.PROPOSER_ROLE();
bytes32 EXECUTOR_ROLE = timelock.EXECUTOR_ROLE();
bytes32 TIMELOCK_ADMIN_ROLE = timelock.TIMELOCK_ADMIN_ROLE();

// 3. Renounce deployer admin after setup
timelock.renounceRole(TIMELOCK_ADMIN_ROLE, deployerAddress);
```

**Deliverables:**
- [ ] TimelockController deployed
- [ ] 48-hour delay configured
- [ ] Multi-sig roles granted
- [ ] Admin role renounced
- [ ] Test proposal executed successfully
- [ ] Documentation: `/docs/TIMELOCK-OPERATIONS.md`

**Acceptance Criteria:**
- Cannot execute changes before delay expires
- Multi-sig can propose and execute
- Emergency delay works (24h for pause)
- No admin backdoors remain

---

### Task 1.3: Transfer Contract Ownership ⏳ PENDING
**Priority:** P0 - BLOCKING MAINNET
**Estimated Time:** 1-2 days

**Requirements:**
- Transfer SageRegistryV2 ownership to Timelock
- Transfer ERC8004ValidationRegistry ownership to Timelock
- Transfer ERC8004ReputationRegistry ownership to Timelock
- Verify ownership transfer complete
- Test admin functions through timelock

**Steps:**
```javascript
// 1. Prepare ownership transfer
const timelock = await ethers.getContractAt("TimelockController", TIMELOCK_ADDRESS);
const sageRegistry = await ethers.getContractAt("SageRegistryV2", REGISTRY_ADDRESS);

// 2. Transfer ownership (two-step process)
await sageRegistry.transferOwnership(timelock.address);
await timelock.acceptOwnership(); // Called through timelock

// 3. Verify ownership
const newOwner = await sageRegistry.owner();
assert(newOwner === timelock.address);

// 4. Test admin function through timelock
const setHookCalldata = sageRegistry.interface.encodeFunctionData(
    "setBeforeRegisterHook",
    [newHookAddress]
);

// Must go through timelock with delay
await timelock.schedule(
    sageRegistry.address,
    0,
    setHookCalldata,
    ethers.constants.HashZero,
    ethers.constants.HashZero,
    2 * 24 * 60 * 60 // 2 days
);

// Wait 2 days...
await timelock.execute(
    sageRegistry.address,
    0,
    setHookCalldata,
    ethers.constants.HashZero,
    ethers.constants.HashZero
);
```

**Deliverables:**
- [ ] All 3 contracts owned by Timelock
- [ ] Test admin operation successful
- [ ] Ownership verification script
- [ ] Documentation: `/docs/OWNERSHIP-TRANSFER.md`

**Acceptance Criteria:**
- No EOA owns contracts
- All admin functions require timelock
- 48-hour delay enforced
- Multi-sig controls timelock

---

## Track 2: Security Improvements (P1 - CRITICAL)
**Timeline:** 1-2 weeks
**Owner:** Smart Contract Team

### Task 2.1: Implement Front-Running Protection ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 3-4 days

**Current Issue:**
```solidity
// Agent registration can be front-run
function registerAgent(string did, ...) external {
    // Attacker sees this in mempool
    // Front-runs with higher gas price
    // Steals desired DID
}
```

**Solution: Commit-Reveal Scheme**
```solidity
// Step 1: Commit (hide intent)
mapping(bytes32 => Commitment) public commitments;

struct Commitment {
    bytes32 commitHash;
    uint256 timestamp;
    bool revealed;
}

function commitRegistration(bytes32 commitHash) external {
    require(commitments[msg.sender].timestamp == 0, "Already committed");

    commitments[msg.sender] = Commitment({
        commitHash: commitHash,
        timestamp: block.timestamp,
        revealed: false
    });

    emit RegistrationCommitted(msg.sender, commitHash);
}

// Step 2: Reveal (after min delay)
function registerAgent(
    string calldata did,
    bytes calldata publicKey,
    bytes32 salt,
    ...
) external {
    Commitment storage commitment = commitments[msg.sender];

    // Verify commitment
    bytes32 expectedHash = keccak256(abi.encodePacked(
        did,
        publicKey,
        msg.sender,
        salt
    ));
    require(commitment.commitHash == expectedHash, "Invalid reveal");
    require(!commitment.revealed, "Already revealed");
    require(block.timestamp >= commitment.timestamp + 1 minutes, "Too soon");
    require(block.timestamp <= commitment.timestamp + 1 hours, "Expired");

    commitment.revealed = true;

    // Proceed with registration (front-run protected)
    _registerAgent(...);
}
```

**Deliverables:**
- [ ] Commit-reveal implemented in SageRegistryV2
- [ ] Commit-reveal implemented in ERC8004ReputationRegistry (authorizeTask)
- [ ] Tests for front-running scenarios
- [ ] Gas cost analysis
- [ ] Documentation: `/docs/FRONT-RUNNING-PROTECTION.md`

**Acceptance Criteria:**
- Front-running tests fail (attacker cannot steal DID)
- Normal registration flow works
- Gas overhead < 50k
- 1-minute minimum delay enforced

---

### Task 2.2: Add Cross-Chain Replay Protection ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 2 days

**Current Issue:**
```solidity
// Signature from testnet could work on mainnet
bytes32 challenge = keccak256(abi.encodePacked(
    "SAGE Key Registration:",
    // Missing chainId in some signatures
    address(this),
    msg.sender,
    keyHash
));
```

**Solution: Enforce ChainId Everywhere**
```solidity
// Update all signature verification
function _validatePublicKey(bytes calldata publicKey, bytes calldata signature) internal {
    bytes32 challenge = keccak256(abi.encodePacked(
        "SAGE Key Registration:",
        block.chainid,              // ✅ Already present
        address(this),
        msg.sender,
        keyHash
    ));
    // ...
}

// Add to updateAgent signature
function updateAgent(bytes32 agentId, ..., bytes calldata signature) external {
    bytes32 messageHash = keccak256(abi.encodePacked(
        agentId,
        name,
        description,
        endpoint,
        capabilities,
        msg.sender,
        agentNonce[agentId],
        block.chainid              // ✅ ADD THIS
    ));
    // ...
}

// Add to all other signature verifications
// - Task authorization
// - Validation submission
// - Any future signed operations
```

**Deliverables:**
- [ ] Audit all signature verification functions
- [ ] Add chainId to all signature messages
- [ ] Update tests for different chainIds
- [ ] Documentation: `/docs/SIGNATURE-STANDARDS.md`

**Acceptance Criteria:**
- All signatures include chainId
- Testnet signatures fail on mainnet
- Mainnet signatures fail on testnet
- Tests verify cross-chain protection

---

### Task 2.3: Add Array Bounds Checking ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 1-2 days

**Current Issue:**
```solidity
// Some loops don't validate array length
for (uint256 i = 0; i < responses.length; i++) {
    // Could exceed gas limit if responses.length is huge
}
```

**Solution: Add Maximum Limits**
```solidity
// Add constants
uint256 public constant MAX_RESPONSES_PER_VALIDATION = 100;
uint256 public constant MAX_AGENTS_PER_QUERY = 50;
uint256 public constant MAX_FEEDBACK_PER_QUERY = 100; // Already exists

// Enforce limits
function _checkAndFinalizeValidation(bytes32 requestId) private {
    ValidationResponse[] storage responses = validationResponses[requestId];
    require(responses.length <= MAX_RESPONSES_PER_VALIDATION, "Too many responses");

    for (uint256 i = 0; i < responses.length; i++) {
        // Safe iteration
    }
}

// Add to public query functions
function getAgentsByOwner(address owner, uint256 offset, uint256 limit)
    external
    view
    returns (bytes32[] memory)
{
    require(limit <= MAX_AGENTS_PER_QUERY, "Limit too high");
    // ... pagination logic
}
```

**Deliverables:**
- [ ] Audit all loops in contracts
- [ ] Add maximum limits to all unbounded operations
- [ ] Convert unbounded queries to paginated
- [ ] Gas limit tests
- [ ] Documentation: `/docs/GAS-LIMITS.md`

**Acceptance Criteria:**
- No unbounded loops remain
- All queries have maximum limits
- Gas limit tests pass
- DoS via large arrays prevented

---

### Task 2.4: Implement Decentralized TEE Key Registry ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 3-5 days

**Current Issue:**
```solidity
// Owner controls all trusted TEE keys
mapping(bytes32 => bool) private trustedTEEKeys;

function addTrustedTEEKey(bytes32 keyHash) external onlyOwner {
    trustedTEEKeys[keyHash] = true;
}
```

**Solution: Community Governance for TEE Keys**
```solidity
contract TEEKeyRegistry {
    struct TEEKeyProposal {
        bytes32 keyHash;
        address proposer;
        string attestationReport; // URL to attestation
        uint256 votesFor;
        uint256 votesAgainst;
        uint256 deadline;
        bool executed;
    }

    mapping(bytes32 => TEEKeyProposal) public proposals;
    mapping(bytes32 => mapping(address => bool)) public hasVoted;
    mapping(bytes32 => bool) public approvedKeys;

    // Minimum stake to propose
    uint256 public constant PROPOSAL_STAKE = 1 ether;

    // Voting period
    uint256 public constant VOTING_PERIOD = 7 days;

    // Approval threshold
    uint256 public constant APPROVAL_THRESHOLD = 66; // 66%

    function proposeTEEKey(
        bytes32 keyHash,
        string calldata attestationReport
    ) external payable {
        require(msg.value >= PROPOSAL_STAKE, "Insufficient stake");
        require(proposals[keyHash].proposer == address(0), "Already proposed");

        proposals[keyHash] = TEEKeyProposal({
            keyHash: keyHash,
            proposer: msg.sender,
            attestationReport: attestationReport,
            votesFor: 0,
            votesAgainst: 0,
            deadline: block.timestamp + VOTING_PERIOD,
            executed: false
        });

        emit TEEKeyProposed(keyHash, msg.sender, attestationReport);
    }

    function vote(bytes32 keyHash, bool support) external {
        // Voting logic (could be token-weighted)
        require(!hasVoted[keyHash][msg.sender], "Already voted");
        TEEKeyProposal storage proposal = proposals[keyHash];
        require(block.timestamp < proposal.deadline, "Voting ended");

        // Simple 1-address-1-vote (could be improved with token weighting)
        hasVoted[keyHash][msg.sender] = true;

        if (support) {
            proposal.votesFor++;
        } else {
            proposal.votesAgainst++;
        }

        emit Voted(keyHash, msg.sender, support);
    }

    function executeProposal(bytes32 keyHash) external {
        TEEKeyProposal storage proposal = proposals[keyHash];
        require(block.timestamp >= proposal.deadline, "Voting ongoing");
        require(!proposal.executed, "Already executed");

        uint256 totalVotes = proposal.votesFor + proposal.votesAgainst;
        require(totalVotes > 0, "No votes");

        uint256 approvalRate = (proposal.votesFor * 100) / totalVotes;

        if (approvalRate >= APPROVAL_THRESHOLD) {
            approvedKeys[keyHash] = true;
            // Return stake
            payable(proposal.proposer).transfer(PROPOSAL_STAKE);
            emit TEEKeyApproved(keyHash);
        } else {
            // Slash stake for failed proposal
            emit TEEKeyRejected(keyHash);
        }

        proposal.executed = true;
    }

    function isTrustedTEEKey(bytes32 keyHash) external view returns (bool) {
        return approvedKeys[keyHash];
    }
}
```

**Deliverables:**
- [ ] TEEKeyRegistry contract implemented
- [ ] Voting mechanism tested
- [ ] Integration with ERC8004ValidationRegistry
- [ ] Migration plan for existing keys
- [ ] Documentation: `/docs/TEE-KEY-GOVERNANCE.md`

**Acceptance Criteria:**
- No single party controls TEE keys
- Community can propose and vote
- Malicious keys can be rejected
- Existing legitimate keys migrated

---

## Track 3: Documentation & Testing (P1 - CRITICAL)
**Timeline:** 2-3 weeks
**Owner:** Documentation Team + QA Team

### Task 3.1: Complete NatSpec Documentation ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 5-7 days

**Current State:**
- Some functions have partial NatSpec
- Many functions missing `@param` and `@return`
- No usage examples
- Error conditions not documented

**Requirements:**
```solidity
/**
 * @title ERC8004ValidationRegistry
 * @author SAGE Team
 * @notice ERC-8004 compliant Validation Registry for AI agent output verification
 * @dev Implements stake-based and TEE-based validation mechanisms
 *
 * This contract provides validation services for AI agent outputs through:
 * - Economic staking: Validators re-execute tasks and stake funds
 * - Cryptographic proofs: TEE attestations provide cryptographic verification
 * - Consensus mechanism: Multiple validators reach agreement
 *
 * Security Features:
 * - ReentrancyGuard on all payable functions
 * - Pull payment pattern for fund distribution
 * - Pausable for emergency situations
 * - Ownable2Step for secure ownership transfer
 *
 * @custom:security-contact security@sage-project.io
 */
contract ERC8004ValidationRegistry { }

/**
 * @notice Request validation for a completed AI task
 * @dev Requires payment of minimum stake and creates validation request
 *
 * The requester (client agent) pays a stake which will be used to:
 * 1. Reward honest validators who correctly verify the output
 * 2. Cover the cost of the validation process
 * 3. Slash dishonest validators who provide false responses
 *
 * @param taskId Unique ERC-8004 task identifier (must be non-zero)
 * @param serverAgent Address of the agent whose work is being validated (must be registered and active)
 * @param dataHash Keccak256 hash of the task output to validate (must be non-zero)
 * @param validationType Type of validation required (STAKE, TEE, or HYBRID)
 * @param deadline Timestamp by which validation must complete (must be 1 hour to 30 days in future)
 *
 * @return requestId Unique identifier for this validation request
 *
 * Requirements:
 * - msg.value >= minStake (currently 0.01 ether)
 * - taskId must be non-zero
 * - serverAgent must be registered and active
 * - deadline must be between 1 hour and 30 days in the future
 * - validationType cannot be NONE
 * - caller must be a registered and active agent
 *
 * Emits:
 * - ValidationRequested(requestId, taskId, serverAgent, dataHash, validationType, stake)
 *
 * Example:
 * ```solidity
 * bytes32 taskId = keccak256("task-123");
 * address server = 0x1234...;
 * bytes32 output = keccak256("result data");
 * uint256 deadline = block.timestamp + 2 hours;
 *
 * bytes32 requestId = validationRegistry.requestValidation{value: 0.01 ether}(
 *     taskId,
 *     server,
 *     output,
 *     ValidationType.STAKE,
 *     deadline
 * );
 * ```
 *
 * @custom:security-note Uses nonReentrant and whenNotPaused modifiers
 */
function requestValidation(
    bytes32 taskId,
    address serverAgent,
    bytes32 dataHash,
    ValidationType validationType,
    uint256 deadline
) external payable override nonReentrant whenNotPaused returns (bytes32 requestId) {
    // Implementation...
}
```

**Deliverables:**
- [ ] Complete NatSpec for all 3 contracts
- [ ] Usage examples for all public functions
- [ ] Error condition documentation
- [ ] Security notes for critical functions
- [ ] Generate documentation with `forge doc` or `hardhat-docgen`
- [ ] Published docs at `/docs/api/`

**Acceptance Criteria:**
- Every public/external function has complete NatSpec
- All parameters documented
- All return values documented
- Usage examples provided
- Error conditions explained
- Generated docs build successfully

---

### Task 3.2: Write Integration Tests for Multi-sig ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 3-4 days

**Test Scenarios:**
```javascript
describe("Multi-sig + Timelock Integration", function() {
    describe("Parameter Changes", function() {
        it("Should require 3/5 signatures to propose parameter change", async function() {
            // Attempt with 2 signatures - should fail
            // Attempt with 3 signatures - should succeed
        });

        it("Should enforce 48-hour delay for parameter changes", async function() {
            // Propose setMinStake(0.02 ether)
            // Try to execute immediately - should fail
            // Wait 48 hours
            // Execute - should succeed
        });

        it("Should allow community to detect malicious proposals", async function() {
            // Malicious actor gets 3/5 keys
            // Proposes setMinStake(0) to drain funds
            // Proposal visible in mempool/events
            // Community has 48h to respond
            // Can trigger emergency pause
        });
    });

    describe("Emergency Procedures", function() {
        it("Should allow faster emergency pause (24h delay)", async function() {
            // Critical bug discovered
            // Multi-sig proposes pause()
            // 24-hour delay (not 48-hour)
            // Execute pause
            // All operations stopped
        });

        it("Should prevent operations while paused", async function() {
            // Contract paused
            // Try requestValidation - should revert
            // Try submitStakeValidation - should revert
            // Try registerAgent - should revert
        });

        it("Should allow unpause after fix", async function() {
            // Contract paused
            // Bug fixed in new deployment
            // Multi-sig proposes unpause
            // 24-hour delay
            // Execute unpause
            // Operations resume
        });
    });

    describe("Ownership Transfer", function() {
        it("Should prevent EOA from changing parameters", async function() {
            // Try setMinStake from deployer - should fail
            // Only timelock can call admin functions
        });

        it("Should allow hook updates through timelock", async function() {
            // Propose setBeforeRegisterHook(newHook)
            // Wait 48 hours
            // Execute
            // Verify hook updated
        });
    });

    describe("Attack Scenarios", function() {
        it("Should prevent single key holder from acting alone", async function() {
            // Attacker compromises 1 key
            // Attempts to propose malicious change
            // Needs 2 more signatures
            // Cannot execute
        });

        it("Should prevent bypassing timelock", async function() {
            // Attacker tries to call admin function directly
            // Should revert (not owner)
            // Attacker tries to create malicious proposal
            // Community detects during delay period
        });
    });
});
```

**Deliverables:**
- [ ] 20+ integration test scenarios
- [ ] Multi-sig simulation tests
- [ ] Timelock delay tests
- [ ] Emergency procedure tests
- [ ] Attack scenario tests
- [ ] Documentation: `/test/integration-multisig.test.js`

**Acceptance Criteria:**
- All multi-sig scenarios tested
- Emergency procedures verified
- Attack scenarios fail as expected
- 100% coverage of governance flows

---

### Task 3.3: Deploy to Sepolia Testnet ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 2-3 days

**Deployment Steps:**
```bash
# 1. Prepare deployment scripts
npx hardhat run scripts/deploy-all.js --network sepolia

# 2. Verify all contracts
npx hardhat verify --network sepolia <ADDRESS> <CONSTRUCTOR_ARGS>

# 3. Set up initial configuration
npx hardhat run scripts/configure-contracts.js --network sepolia

# 4. Transfer ownership to multi-sig + timelock
npx hardhat run scripts/transfer-ownership.js --network sepolia

# 5. Run smoke tests on testnet
npx hardhat test --network sepolia
```

**Contracts to Deploy:**
1. SageRegistryV2
2. ERC8004IdentityRegistry
3. ERC8004ReputationRegistry
4. ERC8004ValidationRegistry
5. Gnosis Safe Multi-sig
6. TimelockController
7. TEEKeyRegistry (if ready)

**Deliverables:**
- [ ] All contracts deployed to Sepolia
- [ ] All contracts verified on Etherscan
- [ ] Multi-sig and timelock configured
- [ ] Initial parameters set
- [ ] Ownership transferred
- [ ] Deployment addresses documented
- [ ] Documentation: `/deployments/sepolia.json`

**Acceptance Criteria:**
- All contracts verified on Sepolia Etherscan
- Multi-sig controls all admin functions
- Test transactions successful
- Public can interact with contracts
- Deployment is reproducible

---

## Track 4: Monitoring & Operations (P2 - IMPORTANT)
**Timeline:** 1-2 weeks
**Owner:** DevOps Team

### Task 4.1: Set Up 24/7 Monitoring System ⏳ PENDING
**Priority:** P2 - IMPORTANT
**Estimated Time:** 5-7 days

**Monitoring Requirements:**

**1. On-Chain Monitoring:**
```javascript
// Use The Graph or Tenderly for event monitoring

// Critical events to monitor:
- ValidationRequested (track volume, stake amounts)
- ValidationFinalized (track success/failure rates)
- ValidatorSlashed (alert immediately)
- WithdrawalProcessed (track fund movements)
- Paused/Unpaused (critical alerts)
- OwnershipTransferred (critical alerts)
- TEEKeyAdded/Removed (alert on changes)
- HookFailed (investigate immediately)

// Metrics to track:
- Total validation requests per day
- Average stake amount
- Validator success rates
- Slashing events per day
- Total value locked (TVL)
- Active validators count
- Gas costs per operation
```

**2. Smart Contract State Monitoring:**
```javascript
// Monitor critical state variables
setInterval(async () => {
    // Check if contracts are paused
    const isPaused = await validationRegistry.paused();
    if (isPaused) {
        alert("🚨 CRITICAL: ValidationRegistry is paused!");
    }

    // Check contract balance vs expected
    const balance = await ethers.provider.getBalance(validationRegistry.address);
    const expectedBalance = calculateExpectedBalance();
    if (balance < expectedBalance * 0.9) {
        alert("⚠️ WARNING: Contract balance lower than expected");
    }

    // Check multi-sig ownership
    const owner = await validationRegistry.owner();
    if (owner !== TIMELOCK_ADDRESS) {
        alert("🚨 CRITICAL: Ownership changed unexpectedly!");
    }
}, 60000); // Every minute
```

**3. Alerting System:**
```yaml
# alerts.yml
alerts:
  critical:
    - name: "Contract Paused"
      condition: "Paused event emitted"
      channels: [pagerduty, telegram, email]

    - name: "Large Withdrawal"
      condition: "WithdrawalProcessed > 10 ETH"
      channels: [telegram, email]

    - name: "Ownership Changed"
      condition: "OwnershipTransferred event"
      channels: [pagerduty, telegram, email, sms]

  warning:
    - name: "High Slashing Rate"
      condition: "Slashing rate > 20% in 1 hour"
      channels: [telegram, email]

    - name: "Low Validator Participation"
      condition: "Validators < 5 for 6 hours"
      channels: [email]
```

**Tools:**
- **The Graph:** On-chain data indexing
- **Tenderly:** Real-time monitoring and alerts
- **OpenZeppelin Defender:** Automated operations and monitoring
- **Grafana:** Metrics dashboard
- **PagerDuty:** Critical alerts
- **Telegram Bot:** Real-time notifications

**Deliverables:**
- [ ] The Graph subgraph deployed
- [ ] Tenderly monitoring configured
- [ ] Grafana dashboard created
- [ ] Alert rules configured
- [ ] PagerDuty integration
- [ ] Telegram alert bot
- [ ] Documentation: `/docs/MONITORING.md`

**Acceptance Criteria:**
- All critical events trigger alerts
- Dashboard shows real-time metrics
- Alerts delivered within 1 minute
- 99.9% monitoring uptime
- Team can respond within 15 minutes

---

### Task 4.2: Create Emergency Response Procedures ⏳ PENDING
**Priority:** P2 - IMPORTANT
**Estimated Time:** 2-3 days

**Emergency Playbooks:**

**1. Critical Bug Discovery:**
```markdown
# CRITICAL BUG RESPONSE PLAYBOOK

## Severity Assessment (5 minutes)
- [ ] Can funds be stolen? → CRITICAL
- [ ] Can contracts be locked? → HIGH
- [ ] Can operations be disrupted? → MEDIUM

## Immediate Actions (CRITICAL)
1. Activate incident response team (Slack #incidents)
2. Assess if pause is needed
3. If YES → Multi-sig initiates pause proposal
4. Notify community via Twitter, Discord, Telegram

## Multi-sig Pause Procedure (24 hours)
1. Signer 1: Propose pause() transaction
2. Signers 2-3: Review and approve (need 3/5)
3. Submit to timelock (24-hour emergency delay)
4. Monitor for exploitation during delay
5. Execute pause after delay

## Post-Pause Actions
- [ ] Analyze vulnerability in detail
- [ ] Develop fix
- [ ] Deploy fix to testnet
- [ ] Test thoroughly
- [ ] External audit of fix
- [ ] Prepare migration plan
- [ ] Community communication
- [ ] Multi-sig proposes unpause
- [ ] Execute unpause

## Communication Template
"We have discovered [severity] issue in [contract].
Out of abundance of caution, we are pausing [operations].
No funds are at risk [if true].
Timeline: [estimate]
Updates: [frequency]"
```

**2. Validator Collusion Detected:**
```markdown
# VALIDATOR COLLUSION RESPONSE PLAYBOOK

## Detection Signals
- Multiple validators from same IP
- Coordinated incorrect responses
- Abnormal success rates
- Timing patterns

## Immediate Actions
1. Flag suspicious validators
2. Review on-chain evidence
3. Calculate potential damage
4. Assess if pause needed

## Mitigation
- [ ] Increase consensus threshold
- [ ] Increase minimum validators
- [ ] Blacklist malicious validators
- [ ] Adjust economic parameters
- [ ] Improve validator reputation system

## Long-term
- [ ] Implement KYC for large validators
- [ ] Add stake lockup periods
- [ ] Improve detection algorithms
```

**3. Smart Contract Upgrade:**
```markdown
# CONTRACT UPGRADE PLAYBOOK

## Pre-Upgrade Checklist
- [ ] New contract fully tested
- [ ] External audit completed
- [ ] Testnet deployment successful
- [ ] Migration script tested
- [ ] Rollback plan prepared
- [ ] Community notified (7 days advance)

## Upgrade Process
1. Multi-sig proposes deployment of new contract
2. Community review period (7 days)
3. Deploy new contract
4. Verify on Etherscan
5. Test new contract
6. Multi-sig proposes migration
7. Timelock delay (48 hours)
8. Execute migration
9. Verify migration successful
10. Monitor for issues (24 hours)

## Rollback Triggers
- Migration fails
- Critical bug in new contract
- Unexpected behavior
- Community consensus for rollback
```

**Deliverables:**
- [ ] 10+ emergency playbooks
- [ ] Contact list with roles
- [ ] Communication templates
- [ ] Decision tree flowcharts
- [ ] Practice drills completed
- [ ] Documentation: `/docs/EMERGENCY-PROCEDURES.md`

**Acceptance Criteria:**
- All critical scenarios covered
- Response times defined
- Roles and responsibilities clear
- Team trained on procedures
- Practice drill successful

---

## Track 5: External Audit & Community (P0 - BLOCKING)
**Timeline:** 6-8 weeks
**Owner:** Project Management + Community Team

### Task 5.1: Contact External Audit Firms ⏳ PENDING
**Priority:** P0 - BLOCKING MAINNET
**Estimated Time:** 1 week

**Audit Firm Options:**

**Tier 1 (Premium):**
- **Trail of Bits** - $100k-150k, 4-6 weeks
- **ConsenSys Diligence** - $80k-120k, 4-6 weeks
- **OpenZeppelin Security** - $90k-130k, 4-5 weeks

**Tier 2 (Reputable):**
- **Certik** - $60k-100k, 3-4 weeks
- **Quantstamp** - $50k-90k, 3-4 weeks
- **Hacken** - $40k-80k, 3-4 weeks

**Selection Criteria:**
- Experience with similar protocols
- Quality of previous reports
- Turnaround time
- Budget fit
- Availability

**Steps:**
1. Contact 3-5 firms for quotes
2. Compare proposals
3. Check references
4. Negotiate terms
5. Sign contract
6. Schedule audit dates

**Deliverables:**
- [ ] RFPs sent to 5 audit firms
- [ ] Proposals received and evaluated
- [ ] Audit firm selected
- [ ] Contract signed
- [ ] Audit scheduled
- [ ] Documentation: `/docs/AUDIT-SELECTION.md`

**Acceptance Criteria:**
- Reputable firm selected
- Within budget
- Timeline works for mainnet launch
- Contract terms favorable

---

### Task 5.2: Prepare Audit Materials ⏳ PENDING
**Priority:** P0 - BLOCKING AUDIT
**Estimated Time:** 1 week

**Required Materials:**

**1. Code Package:**
```
/audit-package/
├── contracts/
│   ├── SageRegistryV2.sol
│   ├── ERC8004IdentityRegistry.sol
│   ├── ERC8004ReputationRegistry.sol
│   ├── ERC8004ValidationRegistry.sol
│   └── interfaces/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── coverage-report/
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY-FEATURES.md
│   ├── THREAT-MODEL.md
│   └── API-REFERENCE.md
├── audit/
│   ├── SCOPE.md
│   ├── KNOWN-ISSUES.md
│   └── FOCUS-AREAS.md
└── README.md
```

**2. Architecture Diagrams:**
```mermaid
# System Architecture
# Contract Interactions
# Validation Flow
# Economic Model
# Threat Vectors
```

**3. Security Documentation:**
- Known issues and limitations
- Previous audit reports
- Security assumptions
- Trust boundaries
- Economic security model

**4. Test Coverage:**
- Unit test coverage > 95%
- Integration test coverage > 90%
- Security test coverage > 85%

**Deliverables:**
- [ ] Complete code package
- [ ] Architecture diagrams
- [ ] Security documentation
- [ ] Test coverage reports
- [ ] Scope definition document
- [ ] Documentation: `/audit/README.md`

**Acceptance Criteria:**
- All requested materials prepared
- Documentation clear and complete
- Code frozen for audit
- Audit scope agreed upon

---

### Task 5.3: Set Up Bug Bounty Program ⏳ PENDING
**Priority:** P1 - CRITICAL
**Estimated Time:** 1 week

**Platform: Immunefi**

**Bounty Structure:**
```yaml
Critical (Funds at Risk):
  reward: $50,000 - $100,000
  examples:
    - Drain contract funds
    - Steal user stakes
    - Bypass ReentrancyGuard
    - Ownership takeover

High (Logic Errors):
  reward: $10,000 - $50,000
  examples:
    - Validator collusion
    - Economic exploits
    - Reputation gaming
    - Validation manipulation

Medium (Operational Issues):
  reward: $2,000 - $10,000
  examples:
    - DoS attacks
    - Gas griefing
    - Front-running
    - Access control issues

Low (Minor Issues):
  reward: $500 - $2,000
  examples:
    - Input validation
    - Error messages
    - Gas optimizations
    - Code quality
```

**Program Rules:**
- Responsible disclosure (48-hour window)
- No public disclosure before fix
- No exploitation on mainnet
- Must provide PoC
- First reporter gets reward

**Setup Steps:**
1. Create Immunefi account
2. Allocate bounty fund ($200k+)
3. Define scope and rules
4. Prepare disclosure policy
5. Set up response team
6. Launch program
7. Announce to community

**Deliverables:**
- [ ] Immunefi program launched
- [ ] $200k+ bounty fund allocated
- [ ] Disclosure policy published
- [ ] Response team trained
- [ ] Community announcement
- [ ] Documentation: `/docs/BUG-BOUNTY.md`

**Acceptance Criteria:**
- Program live on Immunefi
- Clear rules and scope
- Response process defined
- Fund adequately allocated
- Community aware

---

## Timeline Summary

### Parallel Execution (Optimized: 10 weeks)

```
Week 1-2:  Track 1 (Governance) + Track 2 (Security)
           ├── Multi-sig deployment
           ├── Timelock deployment
           ├── Front-running protection
           └── Cross-chain protection

Week 2-3:  Track 2 (Security) + Track 3 (Documentation)
           ├── Array bounds checking
           ├── TEE key registry
           ├── NatSpec documentation
           └── Integration tests

Week 3-4:  Track 3 (Testing) + Track 4 (Monitoring)
           ├── Testnet deployment
           ├── Monitoring setup
           └── Emergency procedures

Week 4-5:  Track 5 (Audit Prep)
           ├── Contact audit firms
           ├── Prepare materials
           └── Bug bounty setup

Week 6-10: External Audit
           ├── Audit execution (4 weeks)
           ├── Fix findings (1 week)
           └── Second review (if needed)

Week 11:   Final Prep
           ├── Community testing
           ├── Monitoring verification
           └── Launch preparation

Week 12:   MAINNET LAUNCH 🚀
```

### Sequential Execution (Conservative: 16 weeks)
```
Week 1-2:   Governance Infrastructure
Week 3-4:   Security Improvements
Week 5-6:   Documentation & Testing
Week 7-8:   Monitoring & Operations
Week 9-10:  Audit Preparation
Week 11-16: External Audit + Fixes
Week 17:    Community Testing
Week 18:    MAINNET LAUNCH 🚀
```

---

## Risk Mitigation

### If Timeline Slips

**Backup Plan 1: Phased Launch**
- Week 10: Launch with limited features
- Week 12: Enable full features after audit

**Backup Plan 2: Extended Testing**
- Continue testnet for extra 2-4 weeks
- Larger bug bounties
- Community stress testing

**Backup Plan 3: Second Audit**
- If first audit finds critical issues
- Quick second review by another firm
- Adds 2-3 weeks

---

## Success Criteria

### Track 1: Governance ✅
- [ ] Multi-sig controls all contracts
- [ ] 48-hour timelock enforced
- [ ] Emergency pause tested
- [ ] No EOA admin access

### Track 2: Security ✅
- [ ] Front-running protected
- [ ] Cross-chain protected
- [ ] All arrays bounded
- [ ] TEE keys decentralized

### Track 3: Documentation ✅
- [ ] 100% NatSpec coverage
- [ ] All tests passing
- [ ] Testnet deployed
- [ ] Integration tests complete

### Track 4: Operations ✅
- [ ] 24/7 monitoring live
- [ ] Alerts configured
- [ ] Emergency procedures tested
- [ ] Team trained

### Track 5: Audit ✅
- [ ] External audit complete
- [ ] All findings fixed
- [ ] Bug bounty active
- [ ] Audit report published

---

## Budget Estimate

| Item | Cost | Timeline |
|------|------|----------|
| External Audit | $80,000 - $120,000 | 4-6 weeks |
| Bug Bounty Fund | $200,000 | Ongoing |
| Monitoring Tools | $500/month | Ongoing |
| Multi-sig Hardware | $5,000 | One-time |
| DevOps Resources | $20,000 | 2 months |
| **Total** | **$305,500 - $345,500** | **10-16 weeks** |

---

## Next Steps

1. **Immediate (This Week):**
   - [ ] Review and approve this roadmap
   - [ ] Assign owners to each track
   - [ ] Set up project management board
   - [ ] Begin Track 1 (Multi-sig deployment)

2. **Short-term (Week 2-3):**
   - [ ] Complete Track 1 and 2
   - [ ] Begin Track 3 and 4
   - [ ] Contact audit firms

3. **Medium-term (Week 4-10):**
   - [ ] Complete all implementation
   - [ ] Execute external audit
   - [ ] Fix all findings

4. **Long-term (Week 11-12):**
   - [ ] Final testing and verification
   - [ ] Community announcement
   - [ ] MAINNET LAUNCH

---

**Document Owner:** SAGE Project Management
**Created:** 2025-10-07
**Last Updated:** 2025-10-07
**Next Review:** Weekly during execution
