// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../interfaces/IERC8004ReputationRegistry.sol";

/**
 * @title ERC8004ReputationRegistry
 * @notice Standalone implementation of ERC-8004 Reputation Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * This is a STANDALONE implementation that does NOT depend on any project-specific contracts.
 * It can be used independently or integrated with other systems via adapters.
 *
 * Features:
 * - Task authorization and feedback submission
 * - Minimal on-chain storage (detailed feedback off-chain)
 * - Pagination support for feedback queries
 * - Verification integration hooks
 * - Spam prevention via pre-authorization
 */
contract ERC8004ReputationRegistry is IERC8004ReputationRegistry {

    // ============================================
    // STATE VARIABLES
    // ============================================

    /// @dev Mapping from feedbackId to feedback data
    mapping(bytes32 => Feedback) private feedbacks;

    /// @dev Mapping from taskId to task authorization
    mapping(bytes32 => TaskAuthorization) private taskAuthorizations;

    /// @dev Mapping from agent address to array of feedback IDs (for pagination)
    mapping(address => bytes32[]) private agentFeedbackIds;

    /// @dev Mapping from taskId to array of feedback IDs
    mapping(address => mapping(bytes32 => bytes32[])) private taskFeedbackIds;

    /// @dev Counter for generating unique feedback IDs
    uint256 private feedbackCounter;

    /// @dev Address authorized to verify feedback (typically ValidationRegistry)
    address public validationRegistry;

    // ============================================
    // ERRORS
    // ============================================

    error TaskNotAuthorized(bytes32 taskId);
    error TaskAuthorizationExpired(bytes32 taskId);
    error TaskAuthorizationAlreadyUsed(bytes32 taskId);
    error TaskAlreadyAuthorized(bytes32 taskId);
    error InvalidRating(uint8 rating);
    error InvalidDeadline(uint256 deadline);
    error FeedbackNotFound(bytes32 feedbackId);
    error FeedbackAlreadyVerified(bytes32 feedbackId);
    error UnauthorizedVerifier(address caller);
    error InvalidServerAgent();
    error InvalidPaginationParams(uint256 offset, uint256 limit);

    // ============================================
    // MODIFIERS
    // ============================================

    modifier onlyValidationRegistry() {
        if (msg.sender != validationRegistry && validationRegistry != address(0)) {
            revert UnauthorizedVerifier(msg.sender);
        }
        _;
    }

    // ============================================
    // CONSTRUCTOR
    // ============================================

    constructor(address _validationRegistry) {
        // Note: address(0) is allowed - registry can be set later via setValidationRegistry
        validationRegistry = _validationRegistry;
    }

    // ============================================
    // EXTERNAL FUNCTIONS
    // ============================================

    /**
     * @notice Authorize a task for future feedback submission
     * @dev Pre-authorization prevents spam and ensures legitimate task execution
     * @param taskId ERC-8004 task identifier
     * @param serverAgent The agent who will execute the task
     * @param deadline Authorization expiration timestamp
     * @return success True if authorization successful
     */
    function authorizeTask(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline
    )
        external
        override
        returns (bool success)
    {
        // Validate inputs
        if (serverAgent == address(0)) {
            revert InvalidServerAgent();
        }
        if (deadline <= block.timestamp) {
            revert InvalidDeadline(deadline);
        }
        if (taskAuthorizations[taskId].deadline != 0) {
            revert TaskAlreadyAuthorized(taskId);
        }

        // Create authorization
        taskAuthorizations[taskId] = TaskAuthorization({
            taskId: taskId,
            client: msg.sender,
            server: serverAgent,
            deadline: deadline,
            used: false
        });

        emit TaskAuthorized(taskId, msg.sender, serverAgent, deadline);

        return true;
    }

    /**
     * @notice Submit feedback for a completed task
     * @dev Requires valid task authorization from the client agent
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
    )
        external
        override
        returns (bytes32 feedbackId)
    {
        // Validate rating range
        if (rating > 100) {
            revert InvalidRating(rating);
        }

        // Check authorization
        TaskAuthorization storage auth = taskAuthorizations[taskId];
        if (auth.deadline == 0) {
            revert TaskNotAuthorized(taskId);
        }
        if (block.timestamp > auth.deadline) {
            revert TaskAuthorizationExpired(taskId);
        }
        if (auth.used) {
            revert TaskAuthorizationAlreadyUsed(taskId);
        }
        if (auth.client != msg.sender) {
            revert TaskNotAuthorized(taskId);
        }
        if (auth.server != serverAgent) {
            revert InvalidServerAgent();
        }

        // Mark authorization as used
        auth.used = true;

        // Generate unique feedback ID
        feedbackCounter++;
        feedbackId = keccak256(abi.encodePacked(
            taskId,
            msg.sender,
            serverAgent,
            block.timestamp,
            feedbackCounter
        ));

        // Create feedback record
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

        // Add to agent's feedback list
        agentFeedbackIds[serverAgent].push(feedbackId);

        // Add to task's feedback list
        taskFeedbackIds[serverAgent][taskId].push(feedbackId);

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
        Feedback storage feedback = feedbacks[feedbackId];

        if (feedback.timestamp == 0) {
            revert FeedbackNotFound(feedbackId);
        }
        if (feedback.verified) {
            revert FeedbackAlreadyVerified(feedbackId);
        }

        feedback.verified = true;

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
        feedback = feedbacks[feedbackId];
        if (feedback.timestamp == 0) {
            revert FeedbackNotFound(feedbackId);
        }
        return feedback;
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
    )
        external
        view
        override
        returns (Feedback[] memory)
    {
        bytes32[] storage feedbackIds = agentFeedbackIds[agentAddress];
        uint256 totalCount = feedbackIds.length;

        // Validate pagination parameters
        if (offset >= totalCount && totalCount > 0) {
            revert InvalidPaginationParams(offset, limit);
        }
        if (limit == 0) {
            revert InvalidPaginationParams(offset, limit);
        }

        // Calculate actual return size
        uint256 remaining = totalCount > offset ? totalCount - offset : 0;
        uint256 returnSize = remaining < limit ? remaining : limit;

        // Build return array
        Feedback[] memory result = new Feedback[](returnSize);
        for (uint256 i = 0; i < returnSize; i++) {
            result[i] = feedbacks[feedbackIds[offset + i]];
        }

        return result;
    }

    /**
     * @notice Get all feedback for a specific task
     * @dev This function is not fully implemented in standalone version
     *      as it requires server agent address mapping
     * @return feedbacks Array of feedback for the task
     */
    function getTaskFeedback(bytes32 /* taskId */)
        external
        view
        override
        returns (Feedback[] memory)
    {
        // Note: This requires knowing the server agent address
        // In a real implementation, you might want to maintain a separate mapping
        // For now, we'll return an empty array as we need the server address
        // This is a design tradeoff in the standalone version
        return new Feedback[](0);
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
        return agentFeedbackIds[agentAddress].length;
    }

    // ============================================
    // ADMIN FUNCTIONS
    // ============================================

    /**
     * @notice Update validation registry address
     * @dev Only callable by current validation registry or during initial setup
     * @param newValidationRegistry New validation registry address
     */
    function setValidationRegistry(address newValidationRegistry) external {
        if (validationRegistry != address(0) && msg.sender != validationRegistry) {
            revert UnauthorizedVerifier(msg.sender);
        }
        validationRegistry = newValidationRegistry;
    }
}
