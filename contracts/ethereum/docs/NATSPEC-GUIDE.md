# NatSpec Documentation Guide

**Version:** 1.0
**Date:** 2025-10-07
**Purpose:** Complete NatSpec documentation standard for SAGE contracts

---

## Overview

This guide defines the NatSpec (Natural Language Specification) documentation standards for all SAGE smart contracts. Comprehensive documentation improves:

1. **Developer Experience** - Easier to understand and integrate
2. **Security** - Clear intent reduces implementation bugs
3. **Auditability** - Auditors can verify code matches specifications
4. **Maintenance** - Future developers understand design decisions

---

## NatSpec Tags Reference

### Contract-Level Tags

```solidity
/**
 * @title Contract Name
 * @notice Short user-facing description (1-2 sentences)
 * @dev Detailed technical explanation for developers
 * @author Author name or team
 * @custom:security-note Special security considerations
 * @custom:upgrade-pattern Upgradeability information
 */
contract MyContract {
    // ...
}
```

### Function-Level Tags

```solidity
/**
 * @notice Short description for end users
 * @dev Detailed technical explanation
 * @param paramName Description of parameter
 * @return returnName Description of return value
 * @custom:throws ErrorName Explanation of when this error occurs
 * @custom:security-warning Security-critical behavior
 * @custom:gas-optimization Gas saving notes
 */
function myFunction(uint256 paramName) external returns (bool returnName) {
    // ...
}
```

### State Variable Tags

```solidity
/**
 * @notice User-facing description
 * @dev Technical details
 * @custom:invariant Description of invariants maintained
 */
uint256 public myVariable;
```

### Event Tags

```solidity
/**
 * @notice Description of when event is emitted
 * @param paramName Description of parameter
 * @dev Additional technical context
 */
event MyEvent(address indexed paramName);
```

### Error Tags

```solidity
/**
 * @notice Explanation of error condition
 * @dev When this error is thrown and how to avoid it
 */
error MyError(uint256 value, uint256 expected);
```

---

## Documentation Standards by Contract Type

### 1. Registry Contracts

**Required Documentation:**

```solidity
/**
 * @title SageRegistry
 * @notice Decentralized registry for AI agent identities
 * @dev ERC-8004 compliant registry with front-running protection
 * @author SAGE Team
 *
 * Architecture:
 * - DID-based agent identification
 * - Public key management with revocation
 * - Commit-reveal pattern for front-running protection
 * - Emergency pause mechanism
 *
 * Security Features:
 * - Ownable2Step for safe ownership transfer
 * - Pausable for emergency stops
 * - ReentrancyGuard on payable functions (if applicable)
 * - Cross-chain replay protection via chainId
 *
 * Invariants:
 * 1. Each DID maps to exactly one agent
 * 2. Each agent has exactly one owner
 * 3. Revoked keys cannot be reused
 * 4. Owners can have at most MAX_AGENTS_PER_OWNER agents
 *
 * @custom:security-contact security@sage.network
 * @custom:audit-status Pending external audit
 */
```

### 2. Governance Contracts

**Required Documentation:**

```solidity
/**
 * @title TEEKeyRegistry
 * @notice Community-governed registry for trusted TEE keys
 * @dev Implements proposal-voting-execution pattern with slashing
 * @author SAGE Team
 *
 * Governance Process:
 * 1. Proposer stakes ETH and proposes TEE key
 * 2. Community votes during votingPeriod (7 days)
 * 3. If approved (>66%), key becomes trusted
 * 4. If rejected, proposer loses slashingPercentage of stake
 *
 * Voting Mechanism:
 * - Weighted voting based on registered voter weight
 * - Minimum participation threshold (10%)
 * - One vote per address per proposal
 *
 * Economic Model:
 * - Proposal stake: 1 ETH (prevents spam)
 * - Slashing: 50% of stake for rejected proposals
 * - Treasury: Slashed funds go to contract treasury
 *
 * Supported TEE Types:
 * - Intel SGX
 * - AMD SEV
 * - ARM TrustZone
 * - AWS Nitro Enclaves
 *
 * @custom:security-contact security@sage.network
 */
```

### 3. Validation Contracts

**Required Documentation:**

```solidity
/**
 * @title ERC8004ValidationRegistry
 * @notice Decentralized validation through stake and TEE attestations
 * @dev ERC-8004 compliant validation with rewards and slashing
 * @author SAGE Team
 *
 * Validation Methods:
 * 1. Stake-based: Validators re-run computation, stake on result
 * 2. TEE-based: Cryptographic attestation from trusted hardware
 *
 * Economic Incentives:
 * - Validators stake minValidatorStake (0.1 ETH)
 * - Correct validators earn validatorRewardPercentage (10%)
 * - Wrong validators lose slashingPercentage (100%) of stake
 *
 * Consensus:
 * - Requires minValidatorsRequired responses (1)
 * - Consensus reached at consensusThreshold (66%)
 * - Finalization triggers rewards/slashing
 *
 * Security Features:
 * - ReentrancyGuard on all payable functions
 * - Pull payment pattern for withdrawals
 * - Maximum validators limit (DoS protection)
 * - Trusted TEE key verification
 *
 * @custom:security-contact security@sage.network
 */
```

---

## Function Documentation Examples

### Simple View Function

```solidity
/**
 * @notice Get agent information by DID
 * @dev Returns empty struct if agent doesn't exist
 * @param did The decentralized identifier
 * @return agent Agent metadata struct
 */
function getAgentByDID(string calldata did)
    external
    view
    returns (AgentMetadata memory agent);
```

### Complex State-Changing Function

```solidity
/**
 * @notice Submit stake-based validation for a request
 * @dev Validator must stake minValidatorStake ETH
 *
 * Process:
 * 1. Verify request exists and is pending
 * 2. Check validator hasn't already responded
 * 3. Verify validator has sufficient stake
 * 4. Record response
 * 5. Check if consensus reached
 * 6. Finalize if threshold met
 *
 * @param requestId Unique identifier for validation request
 * @param computedHash Hash of validator's computation result
 *
 * @return success True if validation accepted
 *
 * @custom:throws RequestNotFound if requestId doesn't exist
 * @custom:throws RequestNotPending if already finalized
 * @custom:throws ValidatorAlreadyResponded if already submitted
 * @custom:throws InsufficientValidatorStake if msg.value too low
 * @custom:throws MaxValidatorsReached if >100 validators
 *
 * @custom:security-warning This function handles funds. Uses ReentrancyGuard.
 * @custom:gas-cost Approximately 200,000 gas + finalization if triggered
 */
function submitStakeValidation(
    bytes32 requestId,
    bytes32 computedHash
) external payable nonReentrant whenNotPaused returns (bool success);
```

### Commit-Reveal Function

```solidity
/**
 * @notice Commit to future registration (Step 1 of 2)
 * @dev First phase of commit-reveal pattern for front-running protection
 *
 * Commitment Hash Formula:
 * ```
 * commitHash = keccak256(abi.encodePacked(
 *     did,        // The DID you want to register
 *     publicKey,  // Your public key
 *     msg.sender, // Your address
 *     salt,       // Random 32-byte salt (keep secret!)
 *     chainId     // Current chain ID (replay protection)
 * ))
 * ```
 *
 * Security:
 * - Salt must be cryptographically random
 * - Do not reveal salt until reveal phase
 * - Must wait MIN_COMMIT_REVEAL_DELAY (1 minute)
 * - Must reveal before MAX_COMMIT_REVEAL_DELAY (1 hour)
 *
 * @param commitHash Hash of registration parameters (see formula above)
 *
 * @custom:throws AlreadyCommitted if unexpired commitment exists
 *
 * @custom:next-step Call registerAgentWithReveal() after 1 minute
 *
 * Example:
 * ```javascript
 * const salt = ethers.randomBytes(32);
 * const commitHash = ethers.keccak256(
 *   ethers.solidityPacked(
 *     ["string", "bytes", "address", "bytes32", "uint256"],
 *     [did, publicKey, userAddress, salt, chainId]
 *   )
 * );
 * await registry.commitRegistration(commitHash);
 * ```
 */
function commitRegistration(bytes32 commitHash) external;
```

### Admin Function

```solidity
/**
 * @notice Update validator reward percentage
 * @dev Only callable by owner. Affects future validations only.
 *
 * @param newPercentage New reward percentage (0-100)
 *
 * @custom:throws InvalidPercentage if newPercentage > 100
 * @custom:throws OwnableUnauthorizedAccount if caller is not owner
 *
 * @custom:security-note This parameter affects economic incentives.
 *                        Changes should be discussed with community.
 *                        Consider using governance timelock.
 *
 * @custom:governance This should be controlled by TimelockController
 */
function setValidatorRewardPercentage(uint256 newPercentage)
    external
    onlyOwner;
```

---

## Error Documentation

### Custom Errors

```solidity
/**
 * @notice Thrown when reveal timing is too early
 * @dev Must wait at least MIN_COMMIT_REVEAL_DELAY after commitment
 * @param currentTime Current block timestamp
 * @param minTime Minimum allowed reveal time
 */
error RevealTooSoon(uint256 currentTime, uint256 minTime);

/**
 * @notice Thrown when validator stake is insufficient
 * @dev Validator must send at least minValidatorStake ETH
 * @param provided Amount of ETH sent (msg.value)
 * @param required Minimum required stake
 */
error InsufficientValidatorStake(uint256 provided, uint256 required);

/**
 * @notice Thrown when maximum validators limit is reached
 * @dev Prevents DoS via unbounded loop in finalization
 * @param requestId The validation request ID
 * @param maxAllowed Maximum validators allowed (typically 100)
 */
error MaxValidatorsReached(bytes32 requestId, uint256 maxAllowed);
```

---

## Event Documentation

```solidity
/**
 * @notice Emitted when agent is successfully registered
 * @param agentId Unique identifier for the agent
 * @param did Decentralized identifier
 * @param owner Address of agent owner
 * @param publicKey Agent's public key for authentication
 * @dev Indexed parameters (agentId, did, owner) for efficient filtering
 */
event AgentRegistered(
    bytes32 indexed agentId,
    string indexed did,
    address indexed owner,
    bytes publicKey
);

/**
 * @notice Emitted when validation request is finalized
 * @param requestId Unique identifier for request
 * @param success Whether consensus was successful
 * @param successCount Number of validators who agreed
 * @param totalResponses Total number of validator responses
 * @dev Finalization triggers reward distribution
 */
event ValidationFinalized(
    bytes32 indexed requestId,
    bool success,
    uint256 successCount,
    uint256 totalResponses
);
```

---

## Struct Documentation

```solidity
/**
 * @notice Agent metadata stored in registry
 * @dev Complete agent information including identity and status
 *
 * Fields:
 * - agentId: Unique identifier (keccak256 of DID)
 * - did: Decentralized identifier (e.g., "did:sage:alice")
 * - owner: Address that controls this agent
 * - name: Human-readable name
 * - description: Agent description/purpose
 * - endpoint: API endpoint URL
 * - publicKey: Cryptographic public key
 * - capabilities: JSON string of agent capabilities
 * - registeredAt: Block timestamp of registration
 * - lastUpdated: Block timestamp of last update
 * - active: Whether agent is active (false if key revoked)
 *
 * Invariants:
 * - agentId == keccak256(bytes(did))
 * - owner != address(0)
 * - publicKey.length >= MIN_PUBLIC_KEY_LENGTH
 * - publicKey.length <= MAX_PUBLIC_KEY_LENGTH
 */
struct AgentMetadata {
    bytes32 agentId;
    string did;
    address owner;
    string name;
    string description;
    string endpoint;
    bytes publicKey;
    string capabilities;
    uint256 registeredAt;
    uint256 lastUpdated;
    bool active;
}
```

---

## Modifier Documentation

```solidity
/**
 * @notice Restricts function to agent owner only
 * @dev Reverts if msg.sender is not the agent's owner
 * @param agentId The agent identifier to check ownership
 * @custom:throws "Not agent owner" if caller is not owner
 */
modifier onlyAgentOwner(bytes32 agentId) {
    require(agents[agentId].owner == msg.sender, "Not agent owner");
    _;
}

/**
 * @notice Restricts function to registered validators
 * @dev Checks if validator has minimum stake
 * @custom:throws "Not registered validator" if insufficient stake
 */
modifier onlyRegisteredValidator() {
    require(
        validatorStakes[msg.sender] >= minValidatorStake,
        "Not registered validator"
    );
    _;
}
```

---

## Constant Documentation

```solidity
/**
 * @notice Maximum number of agents one owner can register
 * @dev Prevents DoS via unbounded loops when iterating owner's agents
 * @custom:invariant ownerToAgents[owner].length <= MAX_AGENTS_PER_OWNER
 */
uint256 private constant MAX_AGENTS_PER_OWNER = 100;

/**
 * @notice Minimum time between commit and reveal
 * @dev Prevents instant reveal which would not protect against front-running
 * @custom:security Must be at least 1 block (15 seconds on mainnet)
 */
uint256 private constant MIN_COMMIT_REVEAL_DELAY = 1 minutes;

/**
 * @notice Maximum responses per query
 * @dev Prevents DoS via large return arrays
 * @custom:gas-optimization Limits gas cost of view functions
 */
uint256 private constant MAX_RESPONSES_PER_QUERY = 100;
```

---

## Inline Code Comments

### When to Use Inline Comments

✅ **DO comment:**
- Complex algorithms
- Non-obvious security checks
- Gas optimization tricks
- Invariant enforcement
- Business logic reasoning

❌ **DON'T comment:**
- Obvious code (`x = x + 1` doesn't need a comment)
- What the code does (use function NatSpec instead)
- Commented-out code (remove it)

### Example: Well-Commented Function

```solidity
function _checkAndFinalizeValidation(bytes32 requestId) private {
    ValidationRequest storage request = validationRequests[requestId];
    ValidationResponse[] storage responses = validationResponses[requestId];

    // DoS Protection: Limit iteration to prevent gas limit issues
    // If >100 responses, only check first 100 (see ARRAY-BOUNDS-CHECKING.md)
    uint256 maxCheck = responses.length > 100 ? 100 : responses.length;

    uint256 successCount = 0;
    uint256 failCount = 0;

    // Count successful vs failed validations
    for (uint256 i = 0; i < maxCheck; i++) {
        if (responses[i].success) {
            successCount++;
        } else {
            failCount++;
        }
    }

    // Calculate consensus: percentage of validators agreeing
    // Uses PRECISION_MULTIPLIER to avoid rounding errors
    uint256 totalVotes = successCount + failCount;
    uint256 successPercentage = (successCount * PERCENTAGE_BASE * PRECISION_MULTIPLIER)
                                 / (totalVotes * PRECISION_MULTIPLIER);

    // Determine if consensus reached (66% threshold)
    bool consensusReached = successPercentage >= consensusThreshold;

    // Mark as complete to prevent re-finalization
    validationComplete[requestId] = true;

    if (consensusReached) {
        // Consensus successful: distribute rewards
        _distributeRewardsAndSlashing(requestId, true);
        emit ValidationFinalized(requestId, true, successCount, totalVotes);
    } else {
        // Consensus failed: slash all validators
        _distributeRewardsAndSlashing(requestId, false);
        emit ValidationFinalized(requestId, false, successCount, totalVotes);
    }
}
```

---

## Documentation Checklist

### Per Contract

- [ ] Contract-level NatSpec with title, notice, dev
- [ ] Architecture overview
- [ ] Security features listed
- [ ] Invariants documented
- [ ] All public/external functions documented
- [ ] All events documented
- [ ] All custom errors documented
- [ ] All structs documented
- [ ] All constants documented
- [ ] Complex internal logic commented

### Per Function

- [ ] @notice for users
- [ ] @dev for developers
- [ ] @param for each parameter
- [ ] @return for each return value
- [ ] @custom:throws for each error
- [ ] @custom:security-warning if security-critical
- [ ] Example usage (if complex)
- [ ] Gas cost estimate (if high)

---

## Tools and Validation

### Generate Documentation

```bash
# Using Hardhat
npx hardhat docgen

# Using Foundry
forge doc
```

### Validation

```bash
# Check NatSpec completeness
npx hardhat check-natspec

# Lint Solidity documentation
npx solhint 'contracts/**/*.sol'
```

---

## Examples from SAGE Contracts

### ✅ Good Example: SageRegistryV3

```solidity
/**
 * @title SageRegistryV3
 * @notice SAGE AI Agent Registry with Front-Running Protection
 * @dev Adds commit-reveal scheme to prevent front-running of agent registration
 *
 * Key Features:
 * - Commit-reveal pattern for DID registration
 * - Enhanced public key validation
 * - Emergency pause mechanism
 * - Two-step ownership transfer
 * - Front-running protection
 *
 * Security Improvements from V2:
 * - MEDIUM-1: Front-running protection via commit-reveal
 * - MEDIUM-2: Cross-chain replay protection (chainId in signatures)
 */
```

### ✅ Good Example: Error with Context

```solidity
/**
 * @notice Thrown when reveal timing is too early
 * @dev Must wait at least MIN_COMMIT_REVEAL_DELAY after commitment
 * @param currentTime Current block timestamp
 * @param minTime Minimum allowed reveal time
 */
error RevealTooSoon(uint256 currentTime, uint256 minTime);
```

### ❌ Bad Example: Insufficient Documentation

```solidity
// Bad - no NatSpec
function updateAgent(bytes32 id, string memory data) external {
    agents[id].data = data;
}

// Good - with NatSpec
/**
 * @notice Update agent metadata
 * @dev Only callable by agent owner
 * @param agentId Unique agent identifier
 * @param data New metadata JSON string
 * @custom:throws "Not agent owner" if caller doesn't own agent
 */
function updateAgentMetadata(
    bytes32 agentId,
    string calldata data
) external onlyAgentOwner(agentId) {
    agents[agentId].data = data;
    emit AgentUpdated(agentId, msg.sender);
}
```

---

## Priority Order

### Phase 1: Critical Contracts (P0)
1. SageRegistryV3.sol
2. ERC8004ValidationRegistry.sol
3. ERC8004ReputationRegistryV2.sol
4. TEEKeyRegistry.sol

### Phase 2: Governance (P1)
5. SimpleMultiSig.sol (note: testnet only)
6. Transfer scripts documentation

### Phase 3: Interfaces (P2)
7. ISageRegistry.sol
8. IERC8004ValidationRegistry.sol
9. IERC8004ReputationRegistry.sol
10. IERC8004IdentityRegistry.sol

---

## Maintenance

### When to Update Documentation

- [ ] Before every pull request
- [ ] When adding new functions
- [ ] When changing function behavior
- [ ] When fixing bugs (document why)
- [ ] After security audits (add audit references)

### Review Process

1. Developer writes code + NatSpec
2. Peer review checks documentation completeness
3. Technical writer reviews clarity
4. Security team reviews security warnings
5. Merge only if all documentation complete

---

## References

- [NatSpec Format](https://docs.soliditylang.org/en/latest/natspec-format.html)
- [Solidity Style Guide](https://docs.soliditylang.org/en/latest/style-guide.html)
- [OpenZeppelin Documentation Standards](https://docs.openzeppelin.com/contracts/4.x/)
- [SAGE Front-Running Protection Guide](./FRONT-RUNNING-PROTECTION.md)
- [SAGE Array Bounds Checking Guide](./ARRAY-BOUNDS-CHECKING.md)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Next Review:** Before external audit

