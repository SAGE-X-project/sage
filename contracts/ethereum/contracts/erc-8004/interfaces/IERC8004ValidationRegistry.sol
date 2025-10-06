// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title IERC8004ValidationRegistry
 * @notice Interface for ERC-8004 Validation Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * The Validation Registry provides generic hooks for requesting and recording
 * independent checks through:
 * - Economic staking (validators re-running the job)
 * - Cryptographic proofs (TEE attestations)
 *
 * The interface allows any validation protocol to integrate seamlessly.
 */
interface IERC8004ValidationRegistry {
    /**
     * @notice Validation type enumeration
     * @dev As specified in ERC-8004
     */
    enum ValidationType {
        NONE,        // No validation required
        STAKE,       // Stake-secured inference validation (crypto-economic)
        TEE,         // Trusted Execution Environment attestation (crypto-verifiable)
        HYBRID       // Both stake and TEE
    }

    /**
     * @notice Validation status enumeration
     */
    enum ValidationStatus {
        PENDING,     // Awaiting validator responses
        VALIDATED,   // Successfully validated
        FAILED,      // Validation failed
        DISPUTED,    // Conflicting validator responses
        EXPIRED      // Validation deadline passed
    }

    /**
     * @notice Validation request structure
     * @dev Implements ERC-8004 ValidationRequest
     */
    struct ValidationRequest {
        bytes32 requestId;           // Unique request identifier
        bytes32 taskId;              // ERC-8004 task identifier
        address requester;           // Who requested validation
        address serverAgent;         // Agent being validated
        bytes32 dataHash;            // Expected output hash (from ERC-8004)
        ValidationType validationType;
        uint256 stake;               // Requester's stake
        uint256 deadline;            // Validation deadline
        ValidationStatus status;
        uint256 timestamp;
    }

    /**
     * @notice Validation response structure
     * @dev Implements ERC-8004 ValidationResponse
     */
    struct ValidationResponse {
        bytes32 responseId;          // Unique response identifier
        bytes32 requestId;           // Reference to request
        address validator;           // Validator address
        bool success;                // Validation result (0 or 100 in ERC-8004 spec)
        bytes32 computedHash;        // Validator's computed hash
        bytes proof;                 // TEE attestation or cryptographic proof
        uint256 validatorStake;      // Validator's stake amount
        uint256 timestamp;
    }

    /**
     * @notice Request validation for a task result
     * @dev Implements ERC-8004 ValidationRequest endpoint
     * @param taskId ERC-8004 task identifier
     * @param serverAgent Agent whose work is being validated
     * @param dataHash Hash of task output to validate
     * @param validationType Type of validation (STAKE, TEE, or HYBRID)
     * @param deadline Validation deadline timestamp
     * @return requestId Unique validation request identifier
     */
    function requestValidation(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        ValidationType validationType,
        uint256 deadline
    ) external payable returns (bytes32 requestId);

    /**
     * @notice Submit stake-based validation response
     * @dev Validator re-executes task and submits result with stake
     *      Implements crypto-economic validation from ERC-8004
     * @param requestId The validation request identifier
     * @param computedHash Validator's computed output hash
     * @return success True if validation submission successful
     */
    function submitStakeValidation(
        bytes32 requestId,
        bytes32 computedHash
    ) external payable returns (bool success);

    /**
     * @notice Submit TEE attestation for validation
     * @dev Validator provides cryptographic proof of execution in TEE
     *      Implements crypto-verifiable validation from ERC-8004
     * @param requestId The validation request identifier
     * @param attestation TEE attestation data
     * @param proof Cryptographic proof (signature, etc.)
     * @return success True if TEE validation accepted
     */
    function submitTEEAttestation(
        bytes32 requestId,
        bytes calldata attestation,
        bytes calldata proof
    ) external returns (bool success);

    /**
     * @notice Get validation request details
     * @param requestId The request identifier
     * @return request Validation request structure
     */
    function getValidationRequest(bytes32 requestId)
        external view returns (ValidationRequest memory request);

    /**
     * @notice Get all responses for a validation request
     * @param requestId The request identifier
     * @return responses Array of validation responses
     */
    function getValidationResponses(bytes32 requestId)
        external view returns (ValidationResponse[] memory responses);

    /**
     * @notice Check if validation is complete
     * @param requestId The request identifier
     * @return isComplete True if validation finalized
     * @return status Final validation status
     */
    function isValidationComplete(bytes32 requestId)
        external view returns (bool isComplete, ValidationStatus status);

    /**
     * @notice Emitted when validation is requested
     * @dev Implements ERC-8004 ValidationRequest event
     * @param requestId Unique request identifier
     * @param taskId ERC-8004 task identifier
     * @param serverAgent Agent being validated (AgentValidatorID in spec)
     * @param dataHash Hash to validate
     * @param validationType Type of validation
     * @param stake Requester's stake amount
     */
    event ValidationRequested(
        bytes32 indexed requestId,
        bytes32 indexed taskId,
        address indexed serverAgent,
        bytes32 dataHash,
        ValidationType validationType,
        uint256 stake
    );

    /**
     * @notice Emitted when a validator submits a response
     * @dev Implements ERC-8004 ValidationResponse event
     * @param requestId Validation request identifier
     * @param validator Validator address
     * @param success Validation result (maps to 0-100 in ERC-8004)
     * @param responseId Unique response identifier
     */
    event ValidationSubmitted(
        bytes32 indexed requestId,
        address indexed validator,
        bool success,
        bytes32 responseId
    );

    /**
     * @notice Emitted when validation is finalized
     * @param requestId Validation request identifier
     * @param status Final validation status
     * @param successRate Percentage of successful validations
     */
    event ValidationFinalized(
        bytes32 indexed requestId,
        ValidationStatus status,
        uint256 successRate
    );

    /**
     * @notice Emitted when a validator is rewarded
     * @param validator Validator address
     * @param requestId Validation request identifier
     * @param amount Reward amount
     */
    event ValidatorRewarded(
        address indexed validator,
        bytes32 indexed requestId,
        uint256 amount
    );

    /**
     * @notice Emitted when a validator is slashed for dishonest validation
     * @param validator Validator address
     * @param requestId Validation request identifier
     * @param amount Slashed amount
     */
    event ValidatorSlashed(
        address indexed validator,
        bytes32 indexed requestId,
        uint256 amount
    );
}
