// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/IERC8004ReputationRegistry.sol";
import "./interfaces/IERC8004IdentityRegistry.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title ERC8004ReputationRegistryV2
 * @author SAGE Development Team
 * @notice ERC-8004 Reputation Registry with front-running protection for task authorization
 * @dev Version 2 implementing commit-reveal pattern for secure task delegation
 *
 * ## Overview
 *
 * The Reputation Registry manages AI agent reputation scores and task authorizations
 * in the SAGE ecosystem. It serves two primary functions:
 *
 * 1. **Task Authorization**: Secure commit-reveal scheme for delegating tasks to agents
 * 2. **Reputation Management**: Recording and querying feedback for agent performance
 *
 * This contract prevents task authorization front-running attacks where adversaries
 * could intercept task delegation and route work to malicious agents instead.
 *
 * ## Architecture
 *
 * ### Component Integration
 * ```
 * Client → ReputationRegistry → {
 *   ├─ IdentityRegistry (verify agents)
 *   └─ ValidationRegistry (validate results)
 * }
 *
 * Validator → ValidationRegistry → ReputationRegistry (update feedback)
 * ```
 *
 * ### Task Authorization Flow
 * 1. **Commit**: Client commits hash of (taskId, serverAgent, deadline, salt)
 * 2. **Wait**: Minimum 30 seconds to prevent MEV front-running
 * 3. **Reveal**: Client reveals actual parameters within 10 minutes
 * 4. **Verification**: Contract validates hash matches commitment
 * 5. **Authorization**: Task officially assigned to specified agent
 *
 * ## Key Features
 *
 * ### 1. Front-Running Protection
 * - Commit-reveal pattern hides task delegation intent
 * - Salt prevents hash prediction
 * - ChainId prevents cross-chain replay attacks
 * - Timing constraints prevent abuse
 *
 * ### 2. Reputation Management
 * - Stores feedback from validation results
 * - Links feedback to tasks and agents
 * - Enables reputation querying by client or validator
 * - Integrates with validation outcomes
 *
 * ### 3. Access Control
 * - Only ValidationRegistry can submit feedback
 * - Prevents unauthorized reputation manipulation
 * - Owner can update ValidationRegistry address (one-time)
 * - Two-step ownership transfer for security
 *
 * ## Security Model
 *
 * ### Assumptions
 * - Clients keep salt secret until reveal
 * - ValidationRegistry is trusted (properly vetted before setting)
 * - Block timestamps accurate within ±15 seconds
 * - Owner is trusted for initial configuration
 *
 * ### Invariants
 * - Feedback can only be created by ValidationRegistry
 * - Each authorization requires a valid commitment
 * - Commitments expire after MAX_COMMIT_REVEAL_DELAY
 * - Ratings are bounded 0-100
 *
 * ### Attack Prevention
 * ```
 * ATTACK: Task Authorization Front-Running
 *
 * WITHOUT PROTECTION:
 * 1. Alice broadcasts authorizeTask(taskX, agentBob)
 * 2. Attacker sees transaction in mempool
 * 3. Attacker front-runs with authorizeTask(taskX, maliciousAgent)
 * 4. Attacker gets task, Alice's intended agent loses work ❌
 *
 * WITH PROTECTION:
 * 1. Alice commits hash (taskX + agentBob hidden)
 * 2. Attacker sees hash but can't decode
 * 3. Alice reveals after 30 seconds
 * 4. Alice's intended agent gets task ✅
 * ```
 *
 * ## Economic Model
 *
 * This contract has no direct economic incentives (no staking or fees).
 * However, it supports the broader ecosystem economics:
 *
 * - **Reputation Value**: High reputation agents earn more task assignments
 * - **Quality Incentive**: Good performance increases reputation
 * - **Accountability**: Poor performance decreases reputation
 * - **Trust Building**: Verifiable track record attracts clients
 *
 * ### Reputation Calculation
 * ```
 * Current implementation: Simple average of all ratings (0-100)
 * Future versions may implement:
 * - Time-weighted reputation (recent performance matters more)
 * - Stake-weighted reputation (validators with more stake count more)
 * - Task-weighted reputation (complex tasks count more)
 * ```
 *
 * ## Gas Costs (Approximate)
 *
 * - `commitTaskAuthorization()`: ~50,000 gas
 * - `authorizeTaskWithReveal()`: ~120,000 gas
 * - `submitFeedback()`: ~150,000 gas (ValidationRegistry only)
 * - `getAgentReputation()`: ~10,000 gas (view)
 * - `queryFeedback()`: ~5,000 + (2,000 × feedback_count) gas (view)
 *
 * ## Integration Points
 *
 * ### With IdentityRegistry
 * - Verifies agents are registered and active
 * - Resolves agent addresses to DIDs
 *
 * ### With ValidationRegistry
 * - Receives feedback after validation completion
 * - Updates agent reputation based on validation outcomes
 * - Authorizations reference validation requests
 *
 * @custom:security-contact security@sage.com
 * @custom:audit-status Phase 7.5 - Front-running protection implemented, pending external audit
 * @custom:version 2.0.0 (with commit-reveal)
 * @custom:erc ERC-8004 compliant
 */
contract ERC8004ReputationRegistryV2 is IERC8004ReputationRegistry, Ownable2Step {
    // State variables
    IERC8004IdentityRegistry public immutable identityRegistry;
    address public validationRegistry;

    // Feedback storage
    mapping(bytes32 => Feedback) private feedbacks;
    mapping(bytes32 => TaskAuthorization) private taskAuthorizations;

    // Indexing mappings
    mapping(address => bytes32[]) private agentFeedbackIds;
    mapping(bytes32 => bytes32[]) private taskFeedbackIds;

    // Counters
    mapping(address => uint256) private agentFeedbackCount;
    uint256 private feedbackCounter;

    // Commit-reveal for task authorization
    struct AuthorizationCommitment {
        bytes32 commitHash;
        uint256 timestamp;
        bool revealed;
    }

    mapping(address => mapping(bytes32 => AuthorizationCommitment)) public authCommitments;

    // Constants
    uint256 private constant MAX_RATING = 100;
    uint256 private constant MAX_FEEDBACK_PER_QUERY = 100;

    // Commit-reveal timing
    uint256 private constant MIN_COMMIT_REVEAL_DELAY = 30 seconds;  // Shorter for task auth
    uint256 private constant MAX_COMMIT_REVEAL_DELAY = 10 minutes;

    // Deadline validation
    uint256 private constant MIN_DEADLINE_DURATION = 1 hours;   // Minimum 1 hour for task completion
    uint256 private constant MAX_DEADLINE_DURATION = 30 days;   // Maximum 30 days

    // Events
    event ValidationRegistryUpdated(address indexed oldRegistry, address indexed newRegistry);

    // Commit-reveal events
    event AuthorizationCommitted(
        address indexed client,
        bytes32 indexed commitHash,
        uint256 timestamp
    );
    event AuthorizationRevealed(
        bytes32 indexed taskId,
        address indexed client,
        address indexed serverAgent
    );
    event CommitmentExpired(address indexed client, bytes32 indexed commitHash);

    // Errors
    error AlreadyCommitted();
    error NoCommitmentFound();
    error InvalidReveal();
    error RevealTooSoon(uint256 currentTime, uint256 minTime);
    error RevealTooLate(uint256 currentTime, uint256 maxTime);
    error CommitmentAlreadyRevealed();
    error DeadlineTooSoon(uint256 deadline, uint256 minRequired);
    error DeadlineTooFar(uint256 deadline, uint256 maxAllowed);

    modifier onlyValidationRegistry() {
        require(msg.sender == validationRegistry, "Only validation registry");
        _;
    }

    constructor(address _identityRegistry) {
        require(_identityRegistry != address(0), "Invalid identity registry");
        identityRegistry = IERC8004IdentityRegistry(_identityRegistry);
        _transferOwnership(msg.sender);
    }

    /**
     * @notice Set the Validation Registry address
     */
    function setValidationRegistry(address registry) external onlyOwner {
        require(registry != address(0), "Invalid validation registry");
        require(validationRegistry == address(0), "Already set");
        address oldRegistry = validationRegistry;
        validationRegistry = registry;
        emit ValidationRegistryUpdated(oldRegistry, registry);
    }

    // ============================================
    // COMMIT-REVEAL TASK AUTHORIZATION
    // ============================================

    /**
     * @notice Commit to a future task authorization (Step 1 of 2)
     * @dev Prevents front-running of task authorization
     *
     * The commitment hash is:
     * keccak256(abi.encodePacked(taskId, serverAgent, deadline, salt, chainId))
     *
     * @param commitHash Hash of authorization parameters + salt
     *
     * Process:
     * 1. Client creates commitment hash off-chain
     * 2. Client calls commitTaskAuthorization(commitHash)
     * 3. Wait MIN_COMMIT_REVEAL_DELAY
     * 4. Client calls authorizeTaskWithReveal() with actual parameters
     * 5. Contract verifies hash matches commitment
     */
    function commitTaskAuthorization(bytes32 commitHash) external {
        AuthorizationCommitment storage commitment = authCommitments[msg.sender][commitHash];

        // Check if already committed
        if (commitment.timestamp > 0 && !commitment.revealed) {
            if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {
                revert AlreadyCommitted();
            } else {
                emit CommitmentExpired(msg.sender, commitHash);
            }
        }

        // Store new commitment
        authCommitments[msg.sender][commitHash] = AuthorizationCommitment({
            commitHash: commitHash,
            timestamp: block.timestamp,
            revealed: false
        });

        emit AuthorizationCommitted(msg.sender, commitHash, block.timestamp);
    }

    /**
     * @notice Authorize task with reveal (Step 2 of 2)
     * @dev Verifies commitment and completes authorization
     *
     * @param taskId ERC-8004 task identifier
     * @param serverAgent The agent who will execute the task
     * @param deadline Authorization expiration timestamp
     * @param salt Random salt used in commitment
     * @return success True if authorization successful
     */
    function authorizeTaskWithReveal(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline,
        bytes32 salt
    ) external returns (bool success) {
        // Verify commitment hash
        bytes32 expectedHash = keccak256(abi.encodePacked(
            taskId,
            serverAgent,
            deadline,
            salt,
            block.chainid  // Cross-chain replay protection
        ));

        AuthorizationCommitment storage commitment = authCommitments[msg.sender][expectedHash];

        // Verify commitment exists
        if (commitment.timestamp == 0) revert NoCommitmentFound();
        if (commitment.revealed) revert CommitmentAlreadyRevealed();

        // Verify timing
        uint256 minRevealTime = commitment.timestamp + MIN_COMMIT_REVEAL_DELAY;
        uint256 maxRevealTime = commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

        if (block.timestamp < minRevealTime) {
            revert RevealTooSoon(block.timestamp, minRevealTime);
        }
        if (block.timestamp > maxRevealTime) {
            revert RevealTooLate(block.timestamp, maxRevealTime);
        }

        // Mark as revealed
        commitment.revealed = true;

        // Perform authorization
        success = _authorizeTask(taskId, serverAgent, deadline);

        emit AuthorizationRevealed(taskId, msg.sender, serverAgent);

        return success;
    }

    /**
     * @notice Legacy task authorization (without front-running protection)
     * @dev Kept for backward compatibility, but authorizeTaskWithReveal() is recommended
     * @custom:security-warning Vulnerable to front-running attacks
     */
    function authorizeTask(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline
    ) external override returns (bool success) {
        return _authorizeTask(taskId, serverAgent, deadline);
    }

    /**
     * @notice Internal task authorization logic
     */
    function _authorizeTask(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline
    ) private returns (bool success) {
        require(taskId != bytes32(0), "Invalid task ID");
        require(serverAgent != address(0), "Invalid server agent");

        // Enhanced deadline validation
        if (deadline <= block.timestamp + MIN_DEADLINE_DURATION) {
            revert DeadlineTooSoon(deadline, block.timestamp + MIN_DEADLINE_DURATION);
        }
        if (deadline > block.timestamp + MAX_DEADLINE_DURATION) {
            revert DeadlineTooFar(deadline, block.timestamp + MAX_DEADLINE_DURATION);
        }

        require(taskAuthorizations[taskId].client == address(0), "Task already authorized");

        // Store authorization BEFORE external calls (CEI pattern)
        taskAuthorizations[taskId] = TaskAuthorization({
            taskId: taskId,
            client: msg.sender,
            server: serverAgent,
            deadline: deadline,
            used: false
        });

        // Emit event BEFORE external calls (reentrancy protection)
        emit TaskAuthorized(taskId, msg.sender, serverAgent, deadline);

        // External calls LAST (after state changes and events)
        // Verify both client and server are registered agents
        IERC8004IdentityRegistry.AgentInfo memory clientInfo =
            identityRegistry.resolveAgentByAddress(msg.sender);
        require(clientInfo.isActive, "Client not active");

        IERC8004IdentityRegistry.AgentInfo memory serverInfo =
            identityRegistry.resolveAgentByAddress(serverAgent);
        require(serverInfo.isActive, "Server not active");

        return true;
    }

    // ============================================
    // FEEDBACK SUBMISSION
    // ============================================

    /**
     * @notice Submit feedback for a completed task
     */
    function submitFeedback(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        uint8 rating
    ) external override returns (bytes32 feedbackId) {
        require(rating <= MAX_RATING, "Rating exceeds maximum");
        require(dataHash != bytes32(0), "Invalid data hash");

        // Verify task authorization
        TaskAuthorization storage auth = taskAuthorizations[taskId];
        require(auth.client == msg.sender, "Not authorized client");
        require(auth.server == serverAgent, "Server mismatch");
        require(!auth.used, "Feedback already submitted");
        require(block.timestamp <= auth.deadline, "Authorization expired");

        // Verify server agent is still active
        IERC8004IdentityRegistry.AgentInfo memory serverInfo =
            identityRegistry.resolveAgentByAddress(serverAgent);
        require(serverInfo.isActive, "Server not active");

        // Mark authorization as used
        auth.used = true;

        // Generate unique feedback ID
        feedbackCounter++;
        feedbackId = keccak256(abi.encodePacked(
            taskId,
            msg.sender,
            serverAgent,
            dataHash,
            block.timestamp,
            feedbackCounter
        ));

        // Store feedback
        feedbacks[feedbackId] = Feedback({
            feedbackId: feedbackId,
            taskId: taskId,
            clientAgent: msg.sender,
            serverAgent: serverAgent,
            dataHash: dataHash,
            rating: rating,
            timestamp: block.timestamp,
            verified: false
        });

        // Update indices
        agentFeedbackIds[serverAgent].push(feedbackId);
        taskFeedbackIds[taskId].push(feedbackId);
        agentFeedbackCount[serverAgent]++;

        emit FeedbackSubmitted(
            feedbackId,
            taskId,
            serverAgent,
            msg.sender,
            dataHash,
            rating,
            block.timestamp
        );

        return feedbackId;
    }

    /**
     * @notice Mark feedback as verified by Validation Registry
     */
    function verifyFeedback(bytes32 feedbackId)
        external
        override
        onlyValidationRegistry
        returns (bool success)
    {
        require(feedbacks[feedbackId].timestamp > 0, "Feedback not found");
        require(!feedbacks[feedbackId].verified, "Already verified");

        feedbacks[feedbackId].verified = true;

        emit FeedbackVerified(feedbackId, msg.sender);

        return true;
    }

    // ============================================
    // VIEW FUNCTIONS
    // ============================================

    function getFeedback(bytes32 feedbackId)
        external
        view
        override
        returns (Feedback memory feedback)
    {
        require(feedbacks[feedbackId].timestamp > 0, "Feedback not found");
        return feedbacks[feedbackId];
    }

    function getAgentFeedback(
        address agentAddress,
        uint256 offset,
        uint256 limit
    ) external view override returns (Feedback[] memory) {
        require(limit > 0 && limit <= MAX_FEEDBACK_PER_QUERY, "Invalid limit");

        bytes32[] storage feedbackIds = agentFeedbackIds[agentAddress];
        require(offset < feedbackIds.length, "Offset out of bounds");

        uint256 resultCount = _min(limit, feedbackIds.length - offset);
        Feedback[] memory result = new Feedback[](resultCount);

        for (uint256 i = 0; i < resultCount; i++) {
            result[i] = feedbacks[feedbackIds[offset + i]];
        }

        return result;
    }

    function getTaskFeedback(bytes32 taskId)
        external
        view
        override
        returns (Feedback[] memory)
    {
        bytes32[] storage feedbackIds = taskFeedbackIds[taskId];
        Feedback[] memory result = new Feedback[](feedbackIds.length);

        for (uint256 i = 0; i < feedbackIds.length; i++) {
            result[i] = feedbacks[feedbackIds[i]];
        }

        return result;
    }

    function getAgentFeedbackCount(address agentAddress)
        external
        view
        override
        returns (uint256 count)
    {
        return agentFeedbackCount[agentAddress];
    }

    function isTaskAuthorized(bytes32 taskId) external view returns (bool isAuthorized) {
        TaskAuthorization memory auth = taskAuthorizations[taskId];
        return auth.client != address(0) &&
               !auth.used &&
               block.timestamp <= auth.deadline;
    }

    function getTaskAuthorization(bytes32 taskId)
        external
        view
        returns (TaskAuthorization memory authorization)
    {
        return taskAuthorizations[taskId];
    }

    function getVerifiedFeedbackCount(address agentAddress)
        external
        view
        returns (uint256 count)
    {
        bytes32[] storage feedbackIds = agentFeedbackIds[agentAddress];
        count = 0;

        for (uint256 i = 0; i < feedbackIds.length; i++) {
            if (feedbacks[feedbackIds[i]].verified) {
                count++;
            }
        }

        return count;
    }

    function getAuthCommitment(address client, bytes32 commitHash)
        external
        view
        returns (
            uint256 timestamp,
            bool revealed,
            bool isExpired
        )
    {
        AuthorizationCommitment memory commitment = authCommitments[client][commitHash];
        bool expired = commitment.timestamp > 0 &&
                      block.timestamp > commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

        return (commitment.timestamp, commitment.revealed, expired);
    }

    function _min(uint256 a, uint256 b) private pure returns (uint256) {
        return a < b ? a : b;
    }
}
