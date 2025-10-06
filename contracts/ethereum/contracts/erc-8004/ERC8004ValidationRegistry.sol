// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./interfaces/IERC8004ValidationRegistry.sol";
import "./interfaces/IERC8004IdentityRegistry.sol";
import "./interfaces/IERC8004ReputationRegistry.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title ERC8004ValidationRegistry
 * @notice ERC-8004 compliant Validation Registry implementation
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * The Validation Registry provides generic hooks for requesting and recording
 * independent checks through:
 * - Economic staking (validators re-running the job)
 * - Cryptographic proofs (TEE attestations)
 *
 * Key Features:
 * - Stake-based validation with crypto-economic incentives
 * - TEE attestation support for cryptographic verification
 * - Validator rewards and slashing mechanism
 * - Integration with Reputation Registry for feedback verification
 * - Reentrancy protection on all payable functions
 */
contract ERC8004ValidationRegistry is IERC8004ValidationRegistry, ReentrancyGuard {
    // State variables
    IERC8004IdentityRegistry public identityRegistry;
    IERC8004ReputationRegistry public reputationRegistry;
    address public owner;

    // Validation storage
    mapping(bytes32 => ValidationRequest) private validationRequests;
    mapping(bytes32 => ValidationResponse[]) private validationResponses;
    mapping(bytes32 => bool) private validationComplete;

    // Validator management
    mapping(address => uint256) private validatorStakes;
    mapping(address => ValidatorStats) private validatorStats;

    // Response tracking
    mapping(bytes32 => mapping(address => bool)) private hasValidatorResponded;
    uint256 private requestCounter;

    struct ValidatorStats {
        uint256 totalValidations;
        uint256 successfulValidations;
        uint256 failedValidations;
        uint256 totalRewards;
        uint256 totalSlashed;
        bool isActive;
    }

    // Configuration parameters
    uint256 public minStake = 0.01 ether;
    uint256 public minValidatorStake = 0.1 ether;
    uint256 public validatorRewardPercentage = 10; // 10% of requester stake
    uint256 public slashingPercentage = 100; // 100% of validator stake
    uint256 public minValidatorsRequired = 1;
    uint256 public consensusThreshold = 66; // 66% agreement required

    // Trusted TEE keys (for production, use a more sophisticated verification system)
    mapping(bytes32 => bool) private trustedTEEKeys;

    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }

    constructor(address _identityRegistry, address _reputationRegistry) {
        require(_identityRegistry != address(0), "Invalid identity registry");
        require(_reputationRegistry != address(0), "Invalid reputation registry");

        identityRegistry = IERC8004IdentityRegistry(_identityRegistry);
        reputationRegistry = IERC8004ReputationRegistry(_reputationRegistry);
        owner = msg.sender;
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
    ) external payable override nonReentrant returns (bytes32 requestId) {
        require(taskId != bytes32(0), "Invalid task ID");
        require(serverAgent != address(0), "Invalid server agent");
        require(dataHash != bytes32(0), "Invalid data hash");
        require(deadline > block.timestamp, "Invalid deadline");
        require(msg.value >= minStake, "Insufficient stake");
        require(validationType != ValidationType.NONE, "Invalid validation type");

        // Verify requester is a registered agent
        IERC8004IdentityRegistry.AgentInfo memory requesterInfo =
            identityRegistry.resolveAgentByAddress(msg.sender);
        require(requesterInfo.isActive, "Requester not active");

        // Verify server agent is registered
        IERC8004IdentityRegistry.AgentInfo memory serverInfo =
            identityRegistry.resolveAgentByAddress(serverAgent);
        require(serverInfo.isActive, "Server not active");

        // Generate unique request ID
        requestCounter++;
        requestId = keccak256(abi.encodePacked(
            taskId,
            msg.sender,
            serverAgent,
            dataHash,
            block.timestamp,
            requestCounter
        ));

        // Store validation request
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
     *      Implements crypto-economic validation from ERC-8004
     * @param requestId The validation request identifier
     * @param computedHash Validator's computed output hash
     * @return success True if validation submission successful
     */
    function submitStakeValidation(
        bytes32 requestId,
        bytes32 computedHash
    ) external payable override nonReentrant returns (bool success) {
        ValidationRequest storage request = validationRequests[requestId];
        require(request.timestamp > 0, "Request not found");
        require(request.status == ValidationStatus.PENDING, "Request not pending");
        require(block.timestamp <= request.deadline, "Request expired");
        require(!hasValidatorResponded[requestId][msg.sender], "Already responded");
        require(msg.value >= minValidatorStake, "Insufficient validator stake");
        require(
            request.validationType == ValidationType.STAKE ||
            request.validationType == ValidationType.HYBRID,
            "Invalid validation type"
        );

        // Verify validator is a registered agent
        IERC8004IdentityRegistry.AgentInfo memory validatorInfo =
            identityRegistry.resolveAgentByAddress(msg.sender);
        require(validatorInfo.isActive, "Validator not active");

        // Determine validation result
        bool validationSuccess = (computedHash == request.dataHash);

        // Generate response ID
        bytes32 responseId = keccak256(abi.encodePacked(
            requestId,
            msg.sender,
            computedHash,
            block.timestamp
        ));

        // Store validation response
        ValidationResponse memory response = ValidationResponse({
            responseId: responseId,
            requestId: requestId,
            validator: msg.sender,
            success: validationSuccess,
            computedHash: computedHash,
            proof: new bytes(0), // No proof for stake-based validation
            validatorStake: msg.value,
            timestamp: block.timestamp
        });

        validationResponses[requestId].push(response);
        hasValidatorResponded[requestId][msg.sender] = true;
        validatorStakes[msg.sender] += msg.value;

        // Update validator stats
        if (!validatorStats[msg.sender].isActive) {
            validatorStats[msg.sender].isActive = true;
        }
        validatorStats[msg.sender].totalValidations++;

        emit ValidationSubmitted(requestId, msg.sender, validationSuccess, responseId);

        // Check if we can finalize validation
        _checkAndFinalizeValidation(requestId);

        return true;
    }

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
    ) external override nonReentrant returns (bool success) {
        ValidationRequest storage request = validationRequests[requestId];
        require(request.timestamp > 0, "Request not found");
        require(request.status == ValidationStatus.PENDING, "Request not pending");
        require(block.timestamp <= request.deadline, "Request expired");
        require(!hasValidatorResponded[requestId][msg.sender], "Already responded");
        require(attestation.length > 0, "Empty attestation");
        require(proof.length > 0, "Empty proof");
        require(
            request.validationType == ValidationType.TEE ||
            request.validationType == ValidationType.HYBRID,
            "Invalid validation type"
        );

        // Verify validator is a registered agent
        IERC8004IdentityRegistry.AgentInfo memory validatorInfo =
            identityRegistry.resolveAgentByAddress(msg.sender);
        require(validatorInfo.isActive, "Validator not active");

        // Verify TEE attestation
        // In production, this should verify Intel SGX, AMD SEV, or ARM TrustZone attestations
        // For now, we check against trusted TEE keys
        bytes32 teeKeyHash = keccak256(proof);
        require(trustedTEEKeys[teeKeyHash], "Untrusted TEE key");

        // Extract computed hash from attestation
        // In production, parse the attestation structure properly
        bytes32 computedHash = keccak256(attestation);

        // Determine validation result
        bool validationSuccess = (computedHash == request.dataHash);

        // Generate response ID
        bytes32 responseId = keccak256(abi.encodePacked(
            requestId,
            msg.sender,
            attestation,
            block.timestamp
        ));

        // Store validation response
        ValidationResponse memory response = ValidationResponse({
            responseId: responseId,
            requestId: requestId,
            validator: msg.sender,
            success: validationSuccess,
            computedHash: computedHash,
            proof: proof,
            validatorStake: 0, // No stake for TEE validation
            timestamp: block.timestamp
        });

        validationResponses[requestId].push(response);
        hasValidatorResponded[requestId][msg.sender] = true;

        // Update validator stats
        if (!validatorStats[msg.sender].isActive) {
            validatorStats[msg.sender].isActive = true;
        }
        validatorStats[msg.sender].totalValidations++;

        emit ValidationSubmitted(requestId, msg.sender, validationSuccess, responseId);

        // Check if we can finalize validation
        _checkAndFinalizeValidation(requestId);

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
        require(validationRequests[requestId].timestamp > 0, "Request not found");
        return validationRequests[requestId];
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
        ValidationRequest memory request = validationRequests[requestId];
        require(request.timestamp > 0, "Request not found");

        return (validationComplete[requestId], request.status);
    }

    /**
     * @notice Internal function to check and finalize validation
     * @dev Called after each response to check if consensus reached
     */
    function _checkAndFinalizeValidation(bytes32 requestId) private {
        ValidationRequest storage request = validationRequests[requestId];
        ValidationResponse[] storage responses = validationResponses[requestId];

        // Check if minimum validators have responded
        if (responses.length < minValidatorsRequired) {
            return;
        }

        // Calculate consensus
        uint256 successCount = 0;
        uint256 failCount = 0;

        for (uint256 i = 0; i < responses.length; i++) {
            if (responses[i].success) {
                successCount++;
            } else {
                failCount++;
            }
        }

        uint256 totalResponses = successCount + failCount;
        uint256 successRate = (successCount * 100) / totalResponses;

        // Determine final status
        ValidationStatus finalStatus;

        if (successRate >= consensusThreshold) {
            finalStatus = ValidationStatus.VALIDATED;
        } else if (successRate <= (100 - consensusThreshold)) {
            finalStatus = ValidationStatus.FAILED;
        } else {
            finalStatus = ValidationStatus.DISPUTED;
        }

        // Update request status
        request.status = finalStatus;
        validationComplete[requestId] = true;

        // Distribute rewards/slashing
        _distributeRewardsAndSlashing(requestId, finalStatus);

        emit ValidationFinalized(requestId, finalStatus, successRate);
    }

    /**
     * @notice Distribute rewards to honest validators and slash dishonest ones
     */
    function _distributeRewardsAndSlashing(
        bytes32 requestId,
        ValidationStatus finalStatus
    ) private {
        ValidationRequest storage request = validationRequests[requestId];
        ValidationResponse[] storage responses = validationResponses[requestId];

        if (finalStatus == ValidationStatus.DISPUTED) {
            // In disputed cases, return stakes without rewards/slashing
            for (uint256 i = 0; i < responses.length; i++) {
                if (responses[i].validatorStake > 0) {
                    validatorStakes[responses[i].validator] -= responses[i].validatorStake;
                    payable(responses[i].validator).transfer(responses[i].validatorStake);
                }
            }
            // Return requester stake
            payable(request.requester).transfer(request.stake);
            return;
        }

        bool expectedSuccess = (finalStatus == ValidationStatus.VALIDATED);
        uint256 totalReward = (request.stake * validatorRewardPercentage) / 100;
        uint256 honestValidatorCount = 0;

        // Count honest validators
        for (uint256 i = 0; i < responses.length; i++) {
            if (responses[i].success == expectedSuccess) {
                honestValidatorCount++;
            }
        }

        require(honestValidatorCount > 0, "No honest validators");
        uint256 rewardPerValidator = totalReward / honestValidatorCount;

        // Distribute rewards and slash dishonest validators
        for (uint256 i = 0; i < responses.length; i++) {
            ValidationResponse storage response = responses[i];

            if (response.success == expectedSuccess) {
                // Honest validator - reward
                if (response.validatorStake > 0) {
                    uint256 totalPayout = response.validatorStake + rewardPerValidator;
                    validatorStakes[response.validator] -= response.validatorStake;
                    payable(response.validator).transfer(totalPayout);

                    validatorStats[response.validator].successfulValidations++;
                    validatorStats[response.validator].totalRewards += rewardPerValidator;

                    emit ValidatorRewarded(response.validator, requestId, rewardPerValidator);
                } else {
                    // TEE validator - small reward
                    payable(response.validator).transfer(rewardPerValidator);
                    validatorStats[response.validator].totalRewards += rewardPerValidator;
                    emit ValidatorRewarded(response.validator, requestId, rewardPerValidator);
                }
            } else {
                // Dishonest validator - slash
                if (response.validatorStake > 0) {
                    uint256 slashAmount = (response.validatorStake * slashingPercentage) / 100;
                    validatorStakes[response.validator] -= response.validatorStake;

                    // Transfer slashed amount to requester as compensation
                    payable(request.requester).transfer(slashAmount);

                    validatorStats[response.validator].failedValidations++;
                    validatorStats[response.validator].totalSlashed += slashAmount;

                    emit ValidatorSlashed(response.validator, requestId, slashAmount);
                }
            }
        }

        // Return remaining stake to requester
        uint256 remainingStake = request.stake - totalReward;
        if (remainingStake > 0) {
            payable(request.requester).transfer(remainingStake);
        }
    }

    /**
     * @notice Get validator statistics
     * @param validator Validator address
     * @return stats Validator statistics
     */
    function getValidatorStats(address validator)
        external
        view
        returns (ValidatorStats memory stats)
    {
        return validatorStats[validator];
    }

    /**
     * @notice Add a trusted TEE key
     * @dev Only callable by owner
     * @param teeKeyHash Hash of the TEE public key
     */
    function addTrustedTEEKey(bytes32 teeKeyHash) external onlyOwner {
        trustedTEEKeys[teeKeyHash] = true;
    }

    /**
     * @notice Remove a trusted TEE key
     * @dev Only callable by owner
     * @param teeKeyHash Hash of the TEE public key
     */
    function removeTrustedTEEKey(bytes32 teeKeyHash) external onlyOwner {
        trustedTEEKeys[teeKeyHash] = false;
    }

    /**
     * @notice Update configuration parameters
     */
    function setMinStake(uint256 _minStake) external onlyOwner {
        minStake = _minStake;
    }

    function setMinValidatorStake(uint256 _minValidatorStake) external onlyOwner {
        minValidatorStake = _minValidatorStake;
    }

    function setValidatorRewardPercentage(uint256 _percentage) external onlyOwner {
        require(_percentage <= 100, "Invalid percentage");
        validatorRewardPercentage = _percentage;
    }

    function setSlashingPercentage(uint256 _percentage) external onlyOwner {
        require(_percentage <= 100, "Invalid percentage");
        slashingPercentage = _percentage;
    }

    function setConsensusThreshold(uint256 _threshold) external onlyOwner {
        require(_threshold > 50 && _threshold <= 100, "Invalid threshold");
        consensusThreshold = _threshold;
    }

    function setMinValidatorsRequired(uint256 _minValidators) external onlyOwner {
        require(_minValidators > 0, "Invalid minimum");
        minValidatorsRequired = _minValidators;
    }
}
