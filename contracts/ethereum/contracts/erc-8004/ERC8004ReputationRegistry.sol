// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/IERC8004ReputationRegistry.sol";
import "./interfaces/IERC8004IdentityRegistry.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title ERC8004ReputationRegistry
 * @notice ERC-8004 compliant Reputation Registry implementation
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * The Reputation Registry enables task feedback between agents with minimal
 * on-chain storage. Scoring and aggregation occur off-chain, enabling an
 * ecosystem of specialized reputation services.
 *
 * Key Features:
 * - Pre-authorized feedback submission (prevents spam)
 * - Minimal on-chain data storage (gas efficient)
 * - Off-chain aggregation via events
 * - Integration with Identity Registry for agent verification
 * - Two-step ownership transfer for security
 */
contract ERC8004ReputationRegistry is IERC8004ReputationRegistry, Ownable2Step {
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

    // Constants
    uint256 private constant MAX_RATING = 100;
    uint256 private constant MAX_FEEDBACK_PER_QUERY = 100;

    // Events
    event ValidationRegistryUpdated(address indexed oldRegistry, address indexed newRegistry);

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
     * @dev Only callable by owner during initial setup
     * @param registry Address of the Validation Registry
     */
    function setValidationRegistry(address registry) external onlyOwner {
        require(registry != address(0), "Invalid validation registry");
        require(validationRegistry == address(0), "Already set");
        address oldRegistry = validationRegistry;
        validationRegistry = registry;
        emit ValidationRegistryUpdated(oldRegistry, registry);
    }

    /**
     * @notice Authorize a task for future feedback submission
     * @dev Must be called before task execution by client agent
     *      Implements pre-authorization mechanism from ERC-8004
     * @param taskId ERC-8004 task identifier
     * @param serverAgent The agent who will execute the task
     * @param deadline Authorization expiration timestamp
     * @return success True if authorization successful
     */
    function authorizeTask(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline
    ) external override returns (bool success) {
        require(taskId != bytes32(0), "Invalid task ID");
        require(serverAgent != address(0), "Invalid server agent");
        require(deadline > block.timestamp, "Invalid deadline");
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

    /**
     * @notice Submit feedback for a completed task
     * @dev Requires valid task authorization
     *      Emits FeedbackSubmitted event for off-chain indexing
     * @param taskId ERC-8004 task identifier
     * @param serverAgent The agent being rated
     * @param dataHash Hash of task output (from ERC-8004 response)
     * @param rating Rating score (0-100)
     * @return feedbackId Unique identifier for the feedback
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
     * @dev Only callable by the Validation Registry contract
     * @param feedbackId The feedback to verify
     * @return success True if verification successful
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

    /**
     * @notice Get feedback details by ID
     * @param feedbackId The feedback identifier
     * @return feedback Feedback structure
     */
    function getFeedback(bytes32 feedbackId)
        external
        view
        override
        returns (Feedback memory feedback)
    {
        require(feedbacks[feedbackId].timestamp > 0, "Feedback not found");
        return feedbacks[feedbackId];
    }

    /**
     * @notice Get paginated feedback for an agent
     * @dev Returns array of feedback for off-chain aggregation
     * @param agentAddress The agent's address
     * @param offset Starting index for pagination
     * @param limit Maximum number of results
     * @return feedbacks Array of feedback structures
     */
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

    /**
     * @notice Get all feedback for a specific task
     * @param taskId ERC-8004 task identifier
     * @return feedbacks Array of feedback for the task
     */
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

    /**
     * @notice Get feedback count for an agent
     * @param agentAddress The agent's address
     * @return count Total number of feedback entries
     */
    function getAgentFeedbackCount(address agentAddress)
        external
        view
        override
        returns (uint256)
    {
        return agentFeedbackCount[agentAddress];
    }

    /**
     * @notice Check if a task has been authorized
     * @param taskId The task identifier
     * @return isAuthorized True if task is authorized and not used
     */
    function isTaskAuthorized(bytes32 taskId) external view returns (bool) {
        TaskAuthorization memory auth = taskAuthorizations[taskId];
        return auth.client != address(0) &&
               !auth.used &&
               block.timestamp <= auth.deadline;
    }

    /**
     * @notice Get task authorization details
     * @param taskId The task identifier
     * @return Task authorization structure
     */
    function getTaskAuthorization(bytes32 taskId)
        external
        view
        returns (TaskAuthorization memory)
    {
        return taskAuthorizations[taskId];
    }

    /**
     * @notice Get verified feedback count for an agent
     * @param agentAddress The agent's address
     * @return count Number of verified feedback entries
     */
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

    /**
     * @notice Helper function to get minimum of two numbers
     */
    function _min(uint256 a, uint256 b) private pure returns (uint256) {
        return a < b ? a : b;
    }
}
