// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../interfaces/IERC8004ValidationRegistry.sol";

/**
 * @title ERC8004ValidationRegistry
 * @notice Standalone implementation of ERC-8004 Validation Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * This is a STANDALONE implementation that does NOT depend on any project-specific contracts.
 * It can be used independently or integrated with other systems via adapters.
 *
 * Features:
 * - Stake-based validation (crypto-economic security)
 * - TEE attestation validation (crypto-verifiable security)
 * - Hybrid validation support
 * - Validator reward/slashing mechanism
 * - Consensus-based validation finalization
 */
contract ERC8004ValidationRegistry is IERC8004ValidationRegistry {

    // ============================================
    // STATE VARIABLES
    // ============================================

    /// @dev Mapping from requestId to validation request
    mapping(bytes32 => ValidationRequest) private validationRequests;

    /// @dev Mapping from requestId to array of responses
    mapping(bytes32 => ValidationResponse[]) private validationResponses;

    /// @dev Mapping from requestId to validator address to response index
    mapping(bytes32 => mapping(address => uint256)) private validatorResponseIndex;

    /// @dev Mapping from requestId to whether validator has responded
    mapping(bytes32 => mapping(address => bool)) private hasValidatorResponded;

    /// @dev Counter for generating unique request IDs
    uint256 private requestCounter;

    /// @dev Minimum stake required for validation
    uint256 public minStake;

    /// @dev Minimum number of validators required for consensus
    uint256 public minValidators;

    /// @dev Consensus threshold (percentage, 0-100)
    uint256 public consensusThreshold;

    /// @dev TEE public keys for attestation verification
    mapping(bytes32 => bool) public trustedTeeKeys;

    // ============================================
    // ERRORS
    // ============================================

    error InvalidDeadline(uint256 deadline);
    error InvalidStake(uint256 provided, uint256 required);
    error ValidationRequestNotFound(bytes32 requestId);
    error ValidationAlreadyComplete(bytes32 requestId);
    error ValidationExpired(bytes32 requestId);
    error ValidatorAlreadyResponded(address validator);
    error InvalidValidationType();
    error InvalidServerAgent();
    error InvalidProof();
    error UnauthorizedTeeKey(bytes32 keyHash);
    error InvalidConsensusThreshold(uint256 threshold);

    // ============================================
    // CONSTRUCTOR
    // ============================================

    constructor(
        uint256 _minStake,
        uint256 _minValidators,
        uint256 _consensusThreshold
    ) {
        if (_consensusThreshold > 100) {
            revert InvalidConsensusThreshold(_consensusThreshold);
        }

        minStake = _minStake;
        minValidators = _minValidators;
        consensusThreshold = _consensusThreshold;
    }

    // ============================================
    // EXTERNAL FUNCTIONS
    // ============================================

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
    )
        external
        payable
        override
        returns (bytes32 requestId)
    {
        // Validate inputs
        if (serverAgent == address(0)) {
            revert InvalidServerAgent();
        }
        if (deadline <= block.timestamp) {
            revert InvalidDeadline(deadline);
        }
        if (validationType == ValidationType.NONE) {
            revert InvalidValidationType();
        }
        if (validationType == ValidationType.STAKE || validationType == ValidationType.HYBRID) {
            if (msg.value < minStake) {
                revert InvalidStake(msg.value, minStake);
            }
        }

        // Generate unique request ID
        requestCounter++;
        requestId = keccak256(abi.encodePacked(
            taskId,
            serverAgent,
            msg.sender,
            block.timestamp,
            requestCounter
        ));

        // Create validation request
        validationRequests[requestId] = ValidationRequest({
            requestId: requestId,
            taskId: taskId,
            requester: msg.sender,
            serverAgent: serverAgent,
            dataHash: dataHash,
            validationType: validationType,
            stake: msg.value,
            deadline: deadline,
            status: ValidationStatus.PENDING,
            timestamp: block.timestamp
        });

        emit ValidationRequested(
            requestId,
            taskId,
            serverAgent,
            dataHash,
            validationType,
            msg.value
        );

        return requestId;
    }

    /**
     * @notice Submit stake-based validation response
     * @dev Validator re-executes task and submits result with stake
     * @param requestId The validation request identifier
     * @param computedHash Validator's computed output hash
     * @return success True if validation submission successful
     */
    function submitStakeValidation(
        bytes32 requestId,
        bytes32 computedHash
    )
        external
        payable
        override
        returns (bool success)
    {
        ValidationRequest storage request = validationRequests[requestId];

        // Validate request exists and is active
        if (request.timestamp == 0) {
            revert ValidationRequestNotFound(requestId);
        }
        if (request.status != ValidationStatus.PENDING) {
            revert ValidationAlreadyComplete(requestId);
        }
        if (block.timestamp > request.deadline) {
            revert ValidationExpired(requestId);
        }
        if (hasValidatorResponded[requestId][msg.sender]) {
            revert ValidatorAlreadyResponded(msg.sender);
        }

        // Validate stake requirement
        if (request.validationType == ValidationType.STAKE ||
            request.validationType == ValidationType.HYBRID) {
            if (msg.value < minStake) {
                revert InvalidStake(msg.value, minStake);
            }
        } else {
            revert InvalidValidationType();
        }

        // Generate response ID
        bytes32 responseId = keccak256(abi.encodePacked(
            requestId,
            msg.sender,
            block.timestamp
        ));

        // Create validation response
        ValidationResponse memory response = ValidationResponse({
            responseId: responseId,
            requestId: requestId,
            validator: msg.sender,
            success: computedHash == request.dataHash,
            computedHash: computedHash,
            proof: "",
            validatorStake: msg.value,
            timestamp: block.timestamp
        });

        // Store response
        validationResponses[requestId].push(response);
        validatorResponseIndex[requestId][msg.sender] = validationResponses[requestId].length - 1;
        hasValidatorResponded[requestId][msg.sender] = true;

        emit ValidationSubmitted(requestId, msg.sender, response.success, responseId);

        // Check if consensus reached
        _checkConsensus(requestId);

        return true;
    }

    /**
     * @notice Submit TEE attestation for validation
     * @dev Validator provides cryptographic proof of execution in TEE
     * @param requestId The validation request identifier
     * @param attestation TEE attestation data
     * @param proof Cryptographic proof (signature, etc.)
     * @return success True if TEE validation accepted
     */
    function submitTEEAttestation(
        bytes32 requestId,
        bytes calldata attestation,
        bytes calldata proof
    )
        external
        override
        returns (bool success)
    {
        ValidationRequest storage request = validationRequests[requestId];

        // Validate request exists and is active
        if (request.timestamp == 0) {
            revert ValidationRequestNotFound(requestId);
        }
        if (request.status != ValidationStatus.PENDING) {
            revert ValidationAlreadyComplete(requestId);
        }
        if (block.timestamp > request.deadline) {
            revert ValidationExpired(requestId);
        }
        if (hasValidatorResponded[requestId][msg.sender]) {
            revert ValidatorAlreadyResponded(msg.sender);
        }

        // Validate TEE requirement
        if (request.validationType != ValidationType.TEE &&
            request.validationType != ValidationType.HYBRID) {
            revert InvalidValidationType();
        }

        // Verify TEE attestation
        bool isValid = _verifyTeeAttestation(attestation, proof);
        if (!isValid) {
            revert InvalidProof();
        }

        // Extract computed hash from attestation
        bytes32 computedHash = _extractHashFromAttestation(attestation);

        // Generate response ID
        bytes32 responseId = keccak256(abi.encodePacked(
            requestId,
            msg.sender,
            block.timestamp
        ));

        // Create validation response
        ValidationResponse memory response = ValidationResponse({
            responseId: responseId,
            requestId: requestId,
            validator: msg.sender,
            success: computedHash == request.dataHash,
            computedHash: computedHash,
            proof: proof,
            validatorStake: 0,
            timestamp: block.timestamp
        });

        // Store response
        validationResponses[requestId].push(response);
        validatorResponseIndex[requestId][msg.sender] = validationResponses[requestId].length - 1;
        hasValidatorResponded[requestId][msg.sender] = true;

        emit ValidationSubmitted(requestId, msg.sender, response.success, responseId);

        // Check if consensus reached
        _checkConsensus(requestId);

        return true;
    }

    /**
     * @notice Get validation request details
     * @param requestId The request identifier
     * @return request Validation request structure
     */
    function getValidationRequest(bytes32 requestId)
        external
        view
        override
        returns (ValidationRequest memory request)
    {
        request = validationRequests[requestId];
        if (request.timestamp == 0) {
            revert ValidationRequestNotFound(requestId);
        }
        return request;
    }

    /**
     * @notice Get all responses for a validation request
     * @param requestId The request identifier
     * @return responses Array of validation responses
     */
    function getValidationResponses(bytes32 requestId)
        external
        view
        override
        returns (ValidationResponse[] memory responses)
    {
        return validationResponses[requestId];
    }

    /**
     * @notice Check if validation is complete
     * @param requestId The request identifier
     * @return isComplete True if validation finalized
     * @return status Final validation status
     */
    function isValidationComplete(bytes32 requestId)
        external
        view
        override
        returns (bool isComplete, ValidationStatus status)
    {
        ValidationRequest storage request = validationRequests[requestId];

        if (request.timestamp == 0) {
            revert ValidationRequestNotFound(requestId);
        }

        status = request.status;
        isComplete = (status != ValidationStatus.PENDING);

        return (isComplete, status);
    }

    // ============================================
    // INTERNAL FUNCTIONS
    // ============================================

    /**
     * @notice Check if consensus has been reached
     * @dev Called after each validator response
     * @param requestId The validation request identifier
     */
    function _checkConsensus(bytes32 requestId) internal {
        ValidationRequest storage request = validationRequests[requestId];
        ValidationResponse[] storage responses = validationResponses[requestId];

        // Check if minimum validators reached
        if (responses.length < minValidators) {
            return;
        }

        // Count successful validations
        uint256 successCount = 0;
        for (uint256 i = 0; i < responses.length; i++) {
            if (responses[i].success) {
                successCount++;
            }
        }

        // Calculate success rate
        uint256 successRate = (successCount * 100) / responses.length;

        // Check consensus
        if (successRate >= consensusThreshold) {
            request.status = ValidationStatus.VALIDATED;
            _distributeRewards(requestId, true);
        } else if (responses.length >= minValidators * 2) {
            // If we have 2x minimum validators and still no consensus, mark as disputed
            request.status = ValidationStatus.DISPUTED;
            _distributeRewards(requestId, false);
        }

        if (request.status != ValidationStatus.PENDING) {
            emit ValidationFinalized(requestId, request.status, successRate);
        }
    }

    /**
     * @notice Distribute rewards or slash stakes based on validation outcome
     * @param requestId The validation request identifier
     * @param consensusReached Whether consensus was reached
     */
    function _distributeRewards(bytes32 requestId, bool consensusReached) internal {
        ValidationRequest storage request = validationRequests[requestId];
        ValidationResponse[] storage responses = validationResponses[requestId];

        if (!consensusReached) {
            // Disputed - return stakes
            for (uint256 i = 0; i < responses.length; i++) {
                if (responses[i].validatorStake > 0) {
                    payable(responses[i].validator).transfer(responses[i].validatorStake);
                }
            }
            // Return requester stake
            if (request.stake > 0) {
                payable(request.requester).transfer(request.stake);
            }
            return;
        }

        // Consensus reached - reward honest validators, slash dishonest ones
        uint256 totalRewardPool = request.stake;
        uint256 honestValidatorCount = 0;

        // Count honest validators
        for (uint256 i = 0; i < responses.length; i++) {
            if (responses[i].success) {
                honestValidatorCount++;
                totalRewardPool += responses[i].validatorStake;
            } else {
                // Slash dishonest validator's stake
                totalRewardPool += responses[i].validatorStake;
                emit ValidatorSlashed(
                    responses[i].validator,
                    requestId,
                    responses[i].validatorStake
                );
            }
        }

        // Distribute rewards to honest validators
        if (honestValidatorCount > 0) {
            uint256 rewardPerValidator = totalRewardPool / honestValidatorCount;
            for (uint256 i = 0; i < responses.length; i++) {
                if (responses[i].success) {
                    payable(responses[i].validator).transfer(rewardPerValidator);
                    emit ValidatorRewarded(
                        responses[i].validator,
                        requestId,
                        rewardPerValidator
                    );
                }
            }
        }
    }

    /**
     * @notice Verify TEE attestation
     * @dev Placeholder for TEE verification logic
     * @param attestation TEE attestation data
     * @param proof Cryptographic proof
     * @return isValid True if attestation is valid
     */
    function _verifyTeeAttestation(
        bytes calldata attestation,
        bytes calldata proof
    ) internal view returns (bool isValid) {
        // Extract public key hash from attestation
        bytes32 keyHash = keccak256(abi.encodePacked(attestation, proof));

        // Check if TEE key is trusted
        return trustedTeeKeys[keyHash];
    }

    /**
     * @notice Extract computed hash from TEE attestation
     * @dev Placeholder for attestation parsing logic
     * @param attestation TEE attestation data
     * @return computedHash The hash extracted from attestation
     */
    function _extractHashFromAttestation(
        bytes calldata attestation
    ) internal pure returns (bytes32 computedHash) {
        // In a real implementation, this would parse the attestation format
        // For now, we assume the first 32 bytes contain the hash
        if (attestation.length >= 32) {
            return bytes32(attestation[0:32]);
        }
        return bytes32(0);
    }

    // ============================================
    // ADMIN FUNCTIONS
    // ============================================

    /**
     * @notice Add trusted TEE public key
     * @param keyHash Hash of TEE public key
     */
    function addTrustedTeeKey(bytes32 keyHash) external {
        trustedTeeKeys[keyHash] = true;
    }

    /**
     * @notice Remove trusted TEE public key
     * @param keyHash Hash of TEE public key
     */
    function removeTrustedTeeKey(bytes32 keyHash) external {
        trustedTeeKeys[keyHash] = false;
    }

    /**
     * @notice Update minimum stake requirement
     * @param newMinStake New minimum stake amount
     */
    function setMinStake(uint256 newMinStake) external {
        minStake = newMinStake;
    }

    /**
     * @notice Update minimum validators requirement
     * @param newMinValidators New minimum validator count
     */
    function setMinValidators(uint256 newMinValidators) external {
        minValidators = newMinValidators;
    }

    /**
     * @notice Update consensus threshold
     * @param newThreshold New consensus threshold (0-100)
     */
    function setConsensusThreshold(uint256 newThreshold) external {
        if (newThreshold > 100) {
            revert InvalidConsensusThreshold(newThreshold);
        }
        consensusThreshold = newThreshold;
    }
}
