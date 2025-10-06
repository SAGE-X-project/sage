// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title IERC8004ReputationRegistry
 * @notice Interface for ERC-8004 Reputation Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * The Reputation Registry enables task feedback between agents with minimal
 * on-chain storage. Scoring and aggregation occur off-chain, enabling an
 * ecosystem of specialized reputation services.
 */
interface IERC8004ReputationRegistry {
    /**
     * @notice Feedback attestation structure
     * @dev Minimal on-chain data, detailed feedback stored off-chain
     */
    struct Feedback {
        bytes32 feedbackId;      // Unique feedback identifier
        bytes32 taskId;          // Associated task ID (from ERC-8004 request)
        address clientAgent;     // Feedback provider
        address serverAgent;     // Feedback recipient
        bytes32 dataHash;        // Hash of task output for verification
        uint8 rating;            // Rating score (0-100)
        uint256 timestamp;       // Submission timestamp
        bool verified;           // Verified by Validation Registry
    }

    /**
     * @notice Task authorization for pre-authorized feedback
     * @dev Prevents spam and ensures legitimate task execution
     */
    struct TaskAuthorization {
        bytes32 taskId;          // ERC-8004 task identifier
        address client;          // Authorized client agent
        address server;          // Server agent
        uint256 deadline;        // Authorization expiry
        bool used;               // Prevents reuse
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
    ) external returns (bool success);

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
    ) external returns (bytes32 feedbackId);

    /**
     * @notice Mark feedback as verified by Validation Registry
     * @dev Only callable by the Validation Registry contract
     * @param feedbackId The feedback to verify
     * @return success True if verification successful
     */
    function verifyFeedback(bytes32 feedbackId)
        external returns (bool success);

    /**
     * @notice Get feedback details by ID
     * @param feedbackId The feedback identifier
     * @return feedback Feedback structure
     */
    function getFeedback(bytes32 feedbackId)
        external view returns (Feedback memory feedback);

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
    ) external view returns (Feedback[] memory feedbacks);

    /**
     * @notice Get all feedback for a specific task
     * @param taskId ERC-8004 task identifier
     * @return feedbacks Array of feedback for the task
     */
    function getTaskFeedback(bytes32 taskId)
        external view returns (Feedback[] memory feedbacks);

    /**
     * @notice Get feedback count for an agent
     * @param agentAddress The agent's address
     * @return count Total number of feedback entries
     */
    function getAgentFeedbackCount(address agentAddress)
        external view returns (uint256 count);

    /**
     * @notice Emitted when a task is authorized for feedback
     * @param taskId ERC-8004 task identifier
     * @param client Client agent address
     * @param server Server agent address
     * @param deadline Authorization deadline
     */
    event TaskAuthorized(
        bytes32 indexed taskId,
        address indexed client,
        address indexed server,
        uint256 deadline
    );

    /**
     * @notice Emitted when feedback is submitted
     * @dev Contains all data needed for off-chain reputation aggregation
     * @param feedbackId Unique feedback identifier
     * @param taskId ERC-8004 task identifier
     * @param serverAgent Agent being rated
     * @param clientAgent Agent providing feedback
     * @param dataHash Hash of task output
     * @param rating Rating score (0-100)
     * @param timestamp Submission time
     */
    event FeedbackSubmitted(
        bytes32 indexed feedbackId,
        bytes32 indexed taskId,
        address indexed serverAgent,
        address clientAgent,
        bytes32 dataHash,
        uint8 rating,
        uint256 timestamp
    );

    /**
     * @notice Emitted when feedback is verified by Validation Registry
     * @param feedbackId The feedback identifier
     * @param verifier Address that verified (Validation Registry)
     */
    event FeedbackVerified(
        bytes32 indexed feedbackId,
        address indexed verifier
    );
}
