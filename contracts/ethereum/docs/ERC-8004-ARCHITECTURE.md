# ERC-8004 Architecture Design for SAGE

## 🏛️ System Architecture

### Overview

The SAGE ERC-8004 implementation consists of three interconnected registries that work together to provide a complete trust layer for AI agent interactions.

```
┌───────────────────────────────────────────────────────────────┐
│                    SAGE ERC-8004 Ecosystem                    │
└───────────────────────────────────────────────────────────────┘
                              │
                              ↓
        ┌─────────────────────┴─────────────────────┐
        │                                           │
        ↓                                           ↓
┌──────────────────┐                    ┌──────────────────┐
│  Off-chain       │                    │  On-chain        │
│  Components      │                    │  Smart Contracts │
└──────────────────┘                    └──────────────────┘
        │                                           │
        ↓                                           ↓
┌─────────────────┐                    ┌──────────────────────────┐
│ - Event Indexer │                    │ 1. Identity Registry     │
│ - Reputation    │←───────────────────│    (SageRegistryV2)      │
│   Aggregator    │      Events        │    - Agent DID           │
│ - UI/API        │                    │    - Public Key          │
│ - Analytics     │                    │    - Metadata            │
└─────────────────┘                    └──────────────────────────┘
        ↑                                           │
        │                                           ↓
        │                              ┌──────────────────────────┐
        │                              │ 2. Reputation Registry   │
        └──────────────────────────────│    (NEW)                 │
                 Read Events           │    - Task Authorization  │
                                       │    - Feedback            │
                                       │    - Attestations        │
                                       └──────────────────────────┘
                                                   │
                                                   ↓
                                       ┌──────────────────────────┐
                                       │ 3. Validation Registry   │
                                       │    (NEW)                 │
                                       │    - Stake Validation    │
                                       │    - TEE Attestation     │
                                       │    - Validator Incentives│
                                       └──────────────────────────┘
```

---

## 📦 Contract Specifications

### 1. Identity Registry (SageRegistryV2.sol) - EXISTING

**Status**: ✅ Already Implemented

**Responsibilities**:
- Register agent DIDs
- Validate public key ownership
- Store agent metadata
- Manage agent lifecycle (activate/deactivate)

**Key Functions**:
```solidity
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature
) external returns (bytes32 agentId);

function getAgent(bytes32 agentId)
    external view returns (AgentMetadata memory);

function isAgentActive(bytes32 agentId)
    external view returns (bool);
```

**ERC-8004 Compliance**:
- ✅ Unique agent identifier (bytes32 agentId)
- ✅ Agent metadata storage
- ✅ Public key verification
- ✅ Agent status tracking

---

### 2. Reputation Registry (SageReputationRegistry.sol) - NEW

**Purpose**: Track and manage agent reputation through feedback attestations

#### 2.1 Core Data Structures

```solidity
// Feedback attestation
struct Feedback {
    bytes32 feedbackId;        // Unique identifier
    bytes32 taskId;            // Associated task
    address clientAgent;       // Feedback provider
    address serverAgent;       // Feedback target
    bytes32 dataHash;          // Task result hash
    uint8 rating;              // 0-100 score
    string feedbackType;       // e.g., "quality", "speed", "accuracy"
    uint256 timestamp;         // Submission time
    bool verified;             // Validation status
    string ipfsHash;           // Off-chain detailed feedback (optional)
}

// Task authorization (pre-authorization mechanism)
struct TaskAuthorization {
    bytes32 taskId;            // Unique task ID
    address client;            // Authorized client
    address server;            // Server agent
    uint256 deadline;          // Authorization expiry
    bool used;                 // Prevent reuse
    uint256 timestamp;         // Authorization time
}

// Aggregated reputation (computed off-chain, stored as checkpoint)
struct ReputationCheckpoint {
    address agentAddress;      // Target agent
    uint256 totalFeedbacks;    // Count
    uint256 averageRating;     // Weighted average
    uint256 timestamp;         // Checkpoint time
    bytes32 merkleRoot;        // All feedbacks Merkle root
}
```

#### 2.2 State Variables

```solidity
contract SageReputationRegistry {
    // Core storage
    mapping(bytes32 => Feedback) private feedbacks;
    mapping(bytes32 => TaskAuthorization) private taskAuthorizations;

    // Indexes for efficient queries
    mapping(address => bytes32[]) private agentFeedbackIds;
    mapping(bytes32 => bytes32[]) private taskFeedbackIds;
    mapping(address => uint256) private agentFeedbackCount;

    // Checkpoints for aggregated reputation
    mapping(address => ReputationCheckpoint[]) private reputationHistory;

    // Configuration
    ISageRegistry public identityRegistry;
    uint256 public authorizationDuration;  // Default: 24 hours
    uint256 public minRating;              // 0
    uint256 public maxRating;              // 100

    // Counters
    uint256 private feedbackCounter;
    uint256 private authorizationCounter;
}
```

#### 2.3 Key Functions

```solidity
/**
 * @notice Authorize a task for future feedback
 * @dev Must be called before task execution
 */
function authorizeTask(
    bytes32 taskId,
    address serverAgent,
    uint256 customDeadline
) external returns (bool) {
    require(identityRegistry.isAgentActive(serverAgent), "Server not active");
    require(!taskAuthorizations[taskId].used, "Task already authorized");

    uint256 deadline = customDeadline > 0 ?
        customDeadline :
        block.timestamp + authorizationDuration;

    taskAuthorizations[taskId] = TaskAuthorization({
        taskId: taskId,
        client: msg.sender,
        server: serverAgent,
        deadline: deadline,
        used: false,
        timestamp: block.timestamp
    });

    emit TaskAuthorized(taskId, msg.sender, serverAgent, deadline);
    return true;
}

/**
 * @notice Submit feedback for a completed task
 * @dev Requires valid task authorization
 */
function submitFeedback(
    bytes32 taskId,
    address serverAgent,
    bytes32 dataHash,
    uint8 rating,
    string calldata feedbackType,
    string calldata ipfsHash
) external returns (bytes32 feedbackId) {
    // Validate authorization
    TaskAuthorization storage auth = taskAuthorizations[taskId];
    require(auth.client == msg.sender, "Not authorized client");
    require(auth.server == serverAgent, "Server mismatch");
    require(!auth.used, "Authorization already used");
    require(block.timestamp <= auth.deadline, "Authorization expired");
    require(rating >= minRating && rating <= maxRating, "Invalid rating");

    // Mark authorization as used
    auth.used = true;

    // Create feedback
    feedbackId = keccak256(abi.encodePacked(
        taskId,
        msg.sender,
        serverAgent,
        block.timestamp,
        feedbackCounter++
    ));

    feedbacks[feedbackId] = Feedback({
        feedbackId: feedbackId,
        taskId: taskId,
        clientAgent: msg.sender,
        serverAgent: serverAgent,
        dataHash: dataHash,
        rating: rating,
        feedbackType: feedbackType,
        timestamp: block.timestamp,
        verified: false,
        ipfsHash: ipfsHash
    });

    // Update indexes
    agentFeedbackIds[serverAgent].push(feedbackId);
    taskFeedbackIds[taskId].push(feedbackId);
    agentFeedbackCount[serverAgent]++;

    emit FeedbackSubmitted(feedbackId, taskId, serverAgent, rating);
    return feedbackId;
}

/**
 * @notice Mark feedback as verified (called by Validation Registry)
 */
function verifyFeedback(bytes32 feedbackId) external {
    require(
        msg.sender == address(validationRegistry),
        "Only validation registry"
    );
    feedbacks[feedbackId].verified = true;
    emit FeedbackVerified(feedbackId);
}

/**
 * @notice Get paginated feedback for an agent
 */
function getAgentFeedback(
    address agentAddress,
    uint256 offset,
    uint256 limit
) external view returns (Feedback[] memory) {
    bytes32[] storage ids = agentFeedbackIds[agentAddress];
    uint256 total = ids.length;

    if (offset >= total) {
        return new Feedback[](0);
    }

    uint256 end = offset + limit > total ? total : offset + limit;
    uint256 count = end - offset;

    Feedback[] memory results = new Feedback[](count);
    for (uint256 i = 0; i < count; i++) {
        results[i] = feedbacks[ids[offset + i]];
    }

    return results;
}

/**
 * @notice Store aggregated reputation checkpoint
 * @dev Called by off-chain aggregation service
 */
function storeReputationCheckpoint(
    address agentAddress,
    uint256 totalFeedbacks,
    uint256 averageRating,
    bytes32 merkleRoot
) external onlyOwner {
    reputationHistory[agentAddress].push(ReputationCheckpoint({
        agentAddress: agentAddress,
        totalFeedbacks: totalFeedbacks,
        averageRating: averageRating,
        timestamp: block.timestamp,
        merkleRoot: merkleRoot
    }));

    emit ReputationCheckpointStored(
        agentAddress,
        totalFeedbacks,
        averageRating
    );
}
```

#### 2.4 Events

```solidity
event TaskAuthorized(
    bytes32 indexed taskId,
    address indexed client,
    address indexed server,
    uint256 deadline
);

event FeedbackSubmitted(
    bytes32 indexed feedbackId,
    bytes32 indexed taskId,
    address indexed serverAgent,
    uint8 rating,
    bytes32 dataHash,
    string feedbackType,
    uint256 timestamp
);

event FeedbackVerified(bytes32 indexed feedbackId);

event ReputationCheckpointStored(
    address indexed agentAddress,
    uint256 totalFeedbacks,
    uint256 averageRating
);
```

---

### 3. Validation Registry (SageValidationRegistry.sol) - NEW

**Purpose**: Enable independent validation of task execution results

#### 3.1 Core Data Structures

```solidity
enum ValidationType {
    NONE,           // No validation required
    STAKE,          // Stake-based re-execution
    TEE,            // Trusted Execution Environment
    HYBRID          // Both stake and TEE
}

enum ValidationStatus {
    PENDING,        // Awaiting validation
    VALIDATED,      // Successfully validated
    FAILED,         // Validation failed
    DISPUTED,       // Conflicting validations
    EXPIRED         // Validation deadline passed
}

struct ValidationRequest {
    bytes32 requestId;         // Unique identifier
    bytes32 taskId;            // Associated task
    address requester;         // Who requested validation
    address serverAgent;       // Agent being validated
    bytes32 dataHash;          // Expected result hash
    ValidationType validationType;
    uint256 stake;             // Requester's stake
    uint256 deadline;          // Validation deadline
    ValidationStatus status;
    uint256 validatorCount;    // Number of validators
    uint256 timestamp;
}

struct ValidationResponse {
    bytes32 responseId;        // Unique identifier
    bytes32 requestId;         // Reference to request
    address validator;         // Who validated
    bool success;              // Match result
    bytes32 computedHash;      // Validator's computed hash
    bytes proof;               // TEE attestation or proof
    uint256 validatorStake;    // Validator's stake
    uint256 timestamp;
}

struct ValidatorInfo {
    address validatorAddress;
    uint256 totalStake;               // Current stake balance
    uint256 successfulValidations;    // Correct validations
    uint256 failedValidations;        // Incorrect validations
    uint256 totalRewards;             // Lifetime rewards
    uint256 totalSlashed;             // Lifetime slashed amount
    bool isActive;                    // Can validate
    uint256 registeredAt;
}
```

#### 3.2 State Variables

```solidity
contract SageValidationRegistry {
    // Core storage
    mapping(bytes32 => ValidationRequest) private validationRequests;
    mapping(bytes32 => ValidationResponse[]) private validationResponses;
    mapping(bytes32 => ValidationResponse) private responseById;

    // Validator management
    mapping(address => ValidatorInfo) private validators;
    mapping(address => uint256) private validatorStakes;
    address[] private validatorList;

    // Configuration
    uint256 public minStake;                    // 0.1 ETH
    uint256 public validatorRewardPercentage;   // 10%
    uint256 public slashingPercentage;          // 100%
    uint256 public validationTimeout;           // 24 hours
    uint256 public minValidators;               // 3 validators required

    // TEE verification (future)
    mapping(bytes32 => bool) private trustedTEEKeys;

    // References
    ISageRegistry public identityRegistry;
    ISageReputationRegistry public reputationRegistry;

    // Counters
    uint256 private requestCounter;
    uint256 private responseCounter;
}
```

#### 3.3 Key Functions

##### Stake-based Validation

```solidity
/**
 * @notice Request validation for a task result
 */
function requestValidation(
    bytes32 taskId,
    address serverAgent,
    bytes32 dataHash,
    ValidationType validationType,
    uint256 customDeadline
) external payable returns (bytes32 requestId) {
    require(msg.value >= minStake, "Insufficient stake");
    require(
        identityRegistry.isAgentActive(serverAgent),
        "Server not active"
    );

    uint256 deadline = customDeadline > 0 ?
        customDeadline :
        block.timestamp + validationTimeout;

    requestId = keccak256(abi.encodePacked(
        taskId,
        msg.sender,
        serverAgent,
        block.timestamp,
        requestCounter++
    ));

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
        validatorCount: 0,
        timestamp: block.timestamp
    });

    emit ValidationRequested(requestId, taskId, validationType, msg.value);
    return requestId;
}

/**
 * @notice Submit stake-based validation
 * @dev Validator re-executes task and submits result
 */
function submitStakeValidation(
    bytes32 requestId,
    bytes32 computedHash
) external payable returns (bool) {
    ValidationRequest storage request = validationRequests[requestId];
    require(request.status == ValidationStatus.PENDING, "Not pending");
    require(block.timestamp <= request.deadline, "Expired");
    require(msg.value >= minStake, "Insufficient validator stake");
    require(
        request.validationType == ValidationType.STAKE ||
        request.validationType == ValidationType.HYBRID,
        "Wrong validation type"
    );

    // Create response
    bytes32 responseId = keccak256(abi.encodePacked(
        requestId,
        msg.sender,
        block.timestamp,
        responseCounter++
    ));

    bool success = (computedHash == request.dataHash);

    ValidationResponse memory response = ValidationResponse({
        responseId: responseId,
        requestId: requestId,
        validator: msg.sender,
        success: success,
        computedHash: computedHash,
        proof: "",
        validatorStake: msg.value,
        timestamp: block.timestamp
    });

    validationResponses[requestId].push(response);
    responseById[responseId] = response;
    request.validatorCount++;

    // Update validator info
    if (validators[msg.sender].validatorAddress == address(0)) {
        _registerValidator(msg.sender);
    }

    if (success) {
        // Correct validation
        validators[msg.sender].successfulValidations++;
        validatorStakes[msg.sender] += msg.value;

        // Reward validator
        uint256 reward = (request.stake * validatorRewardPercentage) / 100;
        validatorStakes[msg.sender] += reward;
        validators[msg.sender].totalRewards += reward;

        emit ValidatorRewarded(msg.sender, responseId, reward);
    } else {
        // Incorrect validation - slash stake
        validators[msg.sender].failedValidations++;
        uint256 slashAmount = (msg.value * slashingPercentage) / 100;

        // Transfer slashed amount to requester
        payable(request.requester).transfer(slashAmount);
        validators[msg.sender].totalSlashed += slashAmount;

        emit ValidatorSlashed(msg.sender, requestId, slashAmount);
    }

    // Check if validation is complete
    if (request.validatorCount >= minValidators) {
        _finalizeValidation(requestId);
    }

    emit ValidationSubmitted(requestId, msg.sender, success);
    return success;
}

/**
 * @notice Finalize validation after sufficient validators
 */
function _finalizeValidation(bytes32 requestId) private {
    ValidationRequest storage request = validationRequests[requestId];
    ValidationResponse[] memory responses = validationResponses[requestId];

    uint256 successCount = 0;
    for (uint256 i = 0; i < responses.length; i++) {
        if (responses[i].success) {
            successCount++;
        }
    }

    // Majority consensus
    if (successCount * 2 > responses.length) {
        request.status = ValidationStatus.VALIDATED;

        // Mark feedback as verified
        reputationRegistry.verifyFeedback(request.taskId);

        // Refund requester
        uint256 refund = request.stake - (
            (request.stake * validatorRewardPercentage * successCount) / 100
        );
        payable(request.requester).transfer(refund);

        emit ValidationFinalized(requestId, true);
    } else {
        request.status = ValidationStatus.FAILED;
        emit ValidationFinalized(requestId, false);
    }
}
```

##### TEE Attestation Support

```solidity
/**
 * @notice Submit TEE attestation for validation
 * @dev Verifies cryptographic proof of execution in TEE
 */
function submitTEEAttestation(
    bytes32 requestId,
    bytes calldata attestation,
    bytes calldata proof
) external returns (bool) {
    ValidationRequest storage request = validationRequests[requestId];
    require(request.status == ValidationStatus.PENDING, "Not pending");
    require(
        request.validationType == ValidationType.TEE ||
        request.validationType == ValidationType.HYBRID,
        "Wrong validation type"
    );

    // Verify TEE attestation
    bool isValid = _verifyTEEAttestation(
        attestation,
        proof,
        request.dataHash
    );
    require(isValid, "Invalid TEE attestation");

    // Create response
    bytes32 responseId = keccak256(abi.encodePacked(
        requestId,
        msg.sender,
        block.timestamp,
        responseCounter++
    ));

    ValidationResponse memory response = ValidationResponse({
        responseId: responseId,
        requestId: requestId,
        validator: msg.sender,
        success: true,
        computedHash: request.dataHash,
        proof: proof,
        validatorStake: 0,
        timestamp: block.timestamp
    });

    validationResponses[requestId].push(response);
    responseById[responseId] = response;

    // TEE attestation is considered authoritative
    request.status = ValidationStatus.VALIDATED;
    reputationRegistry.verifyFeedback(request.taskId);

    emit ValidationSubmitted(requestId, msg.sender, true);
    emit ValidationFinalized(requestId, true);

    return true;
}

/**
 * @notice Verify TEE attestation (placeholder for future implementation)
 */
function _verifyTEEAttestation(
    bytes calldata attestation,
    bytes calldata proof,
    bytes32 expectedHash
) private view returns (bool) {
    // TODO: Implement TEE attestation verification
    // This would involve:
    // 1. Extract public key from attestation
    // 2. Verify signature on attestation
    // 3. Check attestation is from trusted TEE
    // 4. Extract measurement/hash from attestation
    // 5. Compare with expectedHash

    // For now, just check against trusted keys
    bytes32 attestationHash = keccak256(attestation);
    return trustedTEEKeys[attestationHash];
}

/**
 * @notice Add trusted TEE key (admin function)
 */
function addTrustedTEEKey(bytes32 keyHash) external onlyOwner {
    trustedTEEKeys[keyHash] = true;
    emit TrustedTEEKeyAdded(keyHash);
}
```

##### Validator Management

```solidity
/**
 * @notice Register as a validator
 */
function _registerValidator(address validator) private {
    validators[validator] = ValidatorInfo({
        validatorAddress: validator,
        totalStake: 0,
        successfulValidations: 0,
        failedValidations: 0,
        totalRewards: 0,
        totalSlashed: 0,
        isActive: true,
        registeredAt: block.timestamp
    });
    validatorList.push(validator);
    emit ValidatorRegistered(validator);
}

/**
 * @notice Withdraw validator stake
 */
function withdrawValidatorStake(uint256 amount) external {
    require(validators[msg.sender].isActive, "Not active validator");
    require(validatorStakes[msg.sender] >= amount, "Insufficient balance");

    validatorStakes[msg.sender] -= amount;
    validators[msg.sender].totalStake -= amount;

    payable(msg.sender).transfer(amount);
    emit ValidatorStakeWithdrawn(msg.sender, amount);
}

/**
 * @notice Get validator information
 */
function getValidatorInfo(address validator)
    external view returns (ValidatorInfo memory) {
    return validators[validator];
}
```

#### 3.4 Events

```solidity
event ValidationRequested(
    bytes32 indexed requestId,
    bytes32 indexed taskId,
    ValidationType validationType,
    uint256 stake
);

event ValidationSubmitted(
    bytes32 indexed requestId,
    address indexed validator,
    bool success
);

event ValidationFinalized(
    bytes32 indexed requestId,
    bool success
);

event ValidatorRewarded(
    address indexed validator,
    bytes32 indexed responseId,
    uint256 amount
);

event ValidatorSlashed(
    address indexed validator,
    bytes32 indexed requestId,
    uint256 amount
);

event ValidatorRegistered(address indexed validator);

event ValidatorStakeWithdrawn(
    address indexed validator,
    uint256 amount
);

event TrustedTEEKeyAdded(bytes32 indexed keyHash);
```

---

## 🔗 Contract Integration

### Cross-Registry Communication

```solidity
// In SageReputationRegistry
function submitFeedback(...) external returns (bytes32) {
    // ...
    // Automatically create validation request if configured
    if (autoValidationEnabled) {
        validationRegistry.requestValidation{value: validationStake}(
            taskId,
            serverAgent,
            dataHash,
            ValidationType.STAKE,
            0
        );
    }
    // ...
}

// In SageValidationRegistry
function _finalizeValidation(bytes32 requestId) private {
    // ...
    if (status == ValidationStatus.VALIDATED) {
        // Notify reputation registry
        reputationRegistry.verifyFeedback(request.taskId);
    }
    // ...
}
```

---

## 📈 Gas Optimization

### Storage Packing

```solidity
// Before (3 slots)
struct Feedback {
    uint256 timestamp;
    uint8 rating;
    bool verified;
}

// After (1 slot)
struct Feedback {
    uint256 timestamp;  // 256 bits
    uint8 rating;       // 8 bits   } Packed into same slot
    bool verified;      // 8 bits   }
}
```

### Batch Operations

```solidity
function batchAuthorizeTask(
    bytes32[] calldata taskIds,
    address[] calldata serverAgents,
    uint256[] calldata deadlines
) external returns (bool) {
    require(taskIds.length == serverAgents.length, "Length mismatch");
    require(taskIds.length == deadlines.length, "Length mismatch");

    for (uint256 i = 0; i < taskIds.length; i++) {
        _authorizeTask(taskIds[i], serverAgents[i], deadlines[i]);
    }

    return true;
}
```

---

## 🔐 Security Measures

### Reentrancy Protection

```solidity
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

contract SageValidationRegistry is ReentrancyGuard {
    function submitStakeValidation(...)
        external
        payable
        nonReentrant
        returns (bool) {
        // ...
    }
}
```

### Access Control

```solidity
import "@openzeppelin/contracts/access/AccessControl.sol";

contract SageReputationRegistry is AccessControl {
    bytes32 public constant ADMIN_ROLE = keccak256("ADMIN_ROLE");
    bytes32 public constant VALIDATOR_ROLE = keccak256("VALIDATOR_ROLE");

    modifier onlyAdmin() {
        require(hasRole(ADMIN_ROLE, msg.sender), "Not admin");
        _;
    }
}
```

---

*Document Version: 1.0*
*Last Updated: 2025-10-06*
*Author: SAGE Development Team*
