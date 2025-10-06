# ERC-8004 Implementation Plan for SAGE

## 📋 Executive Summary

This document outlines the implementation plan for adding ERC-8004 Trustless Agents standard to the SAGE project. The implementation will be **additive** - existing SageRegistryV2 will remain intact and serve as the Identity Registry, while we add new Reputation and Validation registries.

---

## 🎯 Implementation Goals

### Primary Objectives
1. ✅ **Leverage existing Identity Registry** - SageRegistryV2 already implements ERC-8004 Identity Registry
2. ✅ **Add Reputation Registry** - Enable feedback and reputation tracking
3. ✅ **Add Validation Registry** - Support stake-based and TEE-based validation
4. ✅ **Maintain backward compatibility** - No breaking changes to existing contracts
5. ✅ **Follow ERC-8004 standard** - Ensure compliance with the specification

### Secondary Objectives
- Minimize gas costs through efficient data structures
- Support off-chain aggregation for reputation scoring
- Enable modular validation protocols
- Provide comprehensive test coverage

---

## 🏗️ Architecture Overview

### Three-Registry System

```
┌─────────────────────────────────────────────────────────┐
│                  ERC-8004 Ecosystem                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  1. Identity Registry (EXISTING)             │      │
│  │     SageRegistryV2.sol                       │      │
│  │     - Agent registration (DID)               │      │
│  │     - Public key ownership proof             │      │
│  │     - Agent metadata                         │      │
│  └──────────────────────────────────────────────┘      │
│                         ↓                               │
│  ┌──────────────────────────────────────────────┐      │
│  │  2. Reputation Registry (NEW)                │      │
│  │     SageReputationRegistry.sol               │      │
│  │     - Feedback attestations                  │      │
│  │     - Pre-authorization checks               │      │
│  │     - Minimal on-chain data                  │      │
│  │     - Off-chain aggregation support          │      │
│  └──────────────────────────────────────────────┘      │
│                         ↓                               │
│  ┌──────────────────────────────────────────────┐      │
│  │  3. Validation Registry (NEW)                │      │
│  │     SageValidationRegistry.sol               │      │
│  │     - Stake-based validation                 │      │
│  │     - TEE attestation support                │      │
│  │     - Validator incentives                   │      │
│  │     - Slashing mechanism                     │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 📦 New Contracts to Implement

### 1. SageReputationRegistry.sol

**Purpose**: Track feedback and attestations for agent tasks

**Key Features**:
- Pre-authorized feedback submission
- Minimal on-chain storage
- Support for multiple feedback types (ratings, reviews, attestations)
- Query by agent address, task ID, or time range
- Integration with SageRegistryV2 for agent verification

**Interface**:
```solidity
interface ISageReputationRegistry {
    // Structs
    struct Feedback {
        bytes32 taskId;
        address clientAgent;
        address serverAgent;
        bytes32 dataHash;
        uint8 rating;
        string feedbackType;
        uint256 timestamp;
        bool verified;
    }

    struct TaskAuthorization {
        address client;
        address server;
        bytes32 taskId;
        uint256 deadline;
        bool used;
    }

    // Functions
    function authorizeTask(
        bytes32 taskId,
        address serverAgent,
        uint256 deadline
    ) external returns (bool);

    function submitFeedback(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        uint8 rating,
        string calldata feedbackType
    ) external returns (bytes32 feedbackId);

    function getFeedback(bytes32 feedbackId)
        external view returns (Feedback memory);

    function getAgentFeedback(
        address agentAddress,
        uint256 offset,
        uint256 limit
    ) external view returns (Feedback[] memory);

    function getTaskFeedback(bytes32 taskId)
        external view returns (Feedback[] memory);

    // Events
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
        uint8 rating
    );
}
```

---

### 2. SageValidationRegistry.sol

**Purpose**: Enable independent validation of task results

**Key Features**:
- Multiple validation types (stake-based, TEE attestation)
- Economic incentives for validators
- Slashing for dishonest validators
- TEE attestation verification
- Configurable validation parameters

**Interface**:
```solidity
interface ISageValidationRegistry {
    // Enums
    enum ValidationType { STAKE, TEE, HYBRID }
    enum ValidationStatus { PENDING, VALIDATED, FAILED, EXPIRED }

    // Structs
    struct ValidationRequest {
        bytes32 taskId;
        address requester;
        address serverAgent;
        bytes32 dataHash;
        ValidationType validationType;
        uint256 stake;
        uint256 deadline;
        ValidationStatus status;
        uint256 timestamp;
    }

    struct ValidationResponse {
        bytes32 requestId;
        address validator;
        bool success;
        bytes32 computedHash;
        bytes proof; // For TEE attestations
        uint256 timestamp;
    }

    struct ValidatorInfo {
        uint256 totalStake;
        uint256 successfulValidations;
        uint256 failedValidations;
        bool isActive;
    }

    // Functions
    function requestValidation(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        ValidationType validationType,
        uint256 deadline
    ) external payable returns (bytes32 requestId);

    function submitStakeValidation(
        bytes32 requestId,
        bytes32 computedHash
    ) external payable returns (bool);

    function submitTEEAttestation(
        bytes32 requestId,
        bytes calldata attestation,
        bytes calldata proof
    ) external returns (bool);

    function claimValidatorReward(bytes32 responseId)
        external returns (uint256 reward);

    function getValidationRequest(bytes32 requestId)
        external view returns (ValidationRequest memory);

    function getValidationResponses(bytes32 requestId)
        external view returns (ValidationResponse[] memory);

    function getValidatorInfo(address validator)
        external view returns (ValidatorInfo memory);

    // Configuration
    function setMinStake(uint256 _minStake) external;
    function setValidatorRewardPercentage(uint256 _percentage) external;
    function setSlashingPercentage(uint256 _percentage) external;

    // Events
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

    event ValidatorSlashed(
        address indexed validator,
        bytes32 indexed requestId,
        uint256 amount
    );

    event ValidatorRewarded(
        address indexed validator,
        bytes32 indexed responseId,
        uint256 amount
    );
}
```

---

### 3. Supporting Contracts

#### SageERC8004Registry.sol (Coordinator)

**Purpose**: Central coordinator that links all three registries

```solidity
interface ISageERC8004Registry {
    // Registry addresses
    function identityRegistry() external view returns (address);
    function reputationRegistry() external view returns (address);
    function validationRegistry() external view returns (address);

    // Agent status check across all registries
    function getAgentStatus(address agentAddress)
        external view returns (
            bool registered,
            bool active,
            uint256 reputationScore,
            uint256 validationCount
        );

    // Complete task lifecycle
    function completeTask(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        uint8 rating
    ) external returns (bool);
}
```

---

## 🔐 Security Considerations

### Pre-authorization Mechanism

**Problem**: Prevent spam and malicious feedback

**Solution**: Task-based authorization
```solidity
mapping(bytes32 => TaskAuthorization) private taskAuthorizations;

modifier onlyAuthorizedClient(bytes32 taskId) {
    TaskAuthorization memory auth = taskAuthorizations[taskId];
    require(auth.client == msg.sender, "Not authorized");
    require(!auth.used, "Already used");
    require(block.timestamp <= auth.deadline, "Expired");
    _;
}
```

### Validator Incentives

**Economic Model**:
```
Validation Success:
  - Validator receives: 10% of requester's stake
  - Requester receives: 90% stake refund

Validation Failure (Dishonest):
  - Validator loses: 100% of their stake
  - Requester receives: Validator's stake as compensation
```

### Slashing Conditions

```solidity
function _slashValidator(
    address validator,
    bytes32 requestId,
    uint256 amount
) private {
    // Deduct from validator's stake
    validatorStakes[validator] -= amount;

    // Transfer to requester as compensation
    ValidationRequest memory req = validationRequests[requestId];
    payable(req.requester).transfer(amount);

    // Update validator statistics
    validatorInfo[validator].failedValidations++;

    emit ValidatorSlashed(validator, requestId, amount);
}
```

---

## 📊 Data Structures

### Reputation Registry Storage

```solidity
contract SageReputationRegistry {
    // Minimal on-chain storage
    mapping(bytes32 => Feedback) private feedbacks;
    mapping(address => bytes32[]) private agentFeedbackIds;
    mapping(bytes32 => bytes32[]) private taskFeedbackIds;
    mapping(bytes32 => TaskAuthorization) private taskAuthorizations;

    // Counters for pagination
    mapping(address => uint256) private agentFeedbackCount;

    // Reference to Identity Registry
    ISageRegistry public identityRegistry;
}
```

### Validation Registry Storage

```solidity
contract SageValidationRegistry {
    // Validation requests
    mapping(bytes32 => ValidationRequest) private validationRequests;
    mapping(bytes32 => ValidationResponse[]) private validationResponses;

    // Validator management
    mapping(address => ValidatorInfo) private validatorInfo;
    mapping(address => uint256) private validatorStakes;

    // Configuration
    uint256 public minStake;
    uint256 public validatorRewardPercentage; // e.g., 10
    uint256 public slashingPercentage; // e.g., 100

    // TEE verification (future enhancement)
    mapping(bytes32 => bool) private trustedTEEKeys;
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**SageReputationRegistry.test.js**:
```javascript
describe("SageReputationRegistry", function() {
    describe("Task Authorization", function() {
        it("Should authorize task for client");
        it("Should reject unauthorized feedback");
        it("Should prevent double submission");
        it("Should expire old authorizations");
    });

    describe("Feedback Submission", function() {
        it("Should submit valid feedback");
        it("Should emit FeedbackSubmitted event");
        it("Should link to task and agent");
        it("Should verify agent exists in Identity Registry");
    });

    describe("Feedback Queries", function() {
        it("Should get feedback by ID");
        it("Should get agent feedback with pagination");
        it("Should get task feedback");
        it("Should filter by feedback type");
    });
});
```

**SageValidationRegistry.test.js**:
```javascript
describe("SageValidationRegistry", function() {
    describe("Stake-based Validation", function() {
        it("Should request validation with stake");
        it("Should accept correct validation");
        it("Should slash dishonest validator");
        it("Should reward honest validator");
    });

    describe("TEE Attestation", function() {
        it("Should submit TEE attestation");
        it("Should verify attestation signature");
        it("Should reject invalid attestation");
    });

    describe("Validator Management", function() {
        it("Should track validator statistics");
        it("Should prevent insufficient stake");
        it("Should allow stake withdrawal");
    });
});
```

### Integration Tests

**erc-8004-integration.test.js**:
```javascript
describe("ERC-8004 Full Lifecycle", function() {
    it("Should complete full agent task flow", async function() {
        // 1. Register agent (Identity Registry)
        const agentId = await identityRegistry.registerAgent(...);

        // 2. Authorize task (Reputation Registry)
        await reputationRegistry.authorizeTask(taskId, serverAgent, deadline);

        // 3. Request validation (Validation Registry)
        const requestId = await validationRegistry.requestValidation(
            taskId, serverAgent, dataHash, ValidationType.STAKE
        );

        // 4. Validator submits validation
        await validationRegistry.submitStakeValidation(requestId, dataHash);

        // 5. Client submits feedback
        await reputationRegistry.submitFeedback(
            taskId, serverAgent, dataHash, 5, "excellent"
        );

        // 6. Verify complete state
        const feedback = await reputationRegistry.getFeedback(feedbackId);
        expect(feedback.verified).to.be.true;
    });
});
```

---

## 📈 Gas Optimization Strategies

### 1. Minimal On-chain Storage
- Store only hashes and references on-chain
- Full data in events for off-chain indexing

### 2. Batch Operations
```solidity
function batchSubmitFeedback(
    bytes32[] calldata taskIds,
    address[] calldata serverAgents,
    bytes32[] calldata dataHashes,
    uint8[] calldata ratings
) external {
    require(taskIds.length == serverAgents.length, "Length mismatch");
    // ... batch processing
}
```

### 3. Efficient Data Structures
- Use `bytes32` for IDs instead of strings
- Pack related variables into single storage slots
- Use mappings over arrays when possible

### 4. Event-based Data Retrieval
```solidity
event FeedbackSubmitted(
    bytes32 indexed feedbackId,
    bytes32 indexed taskId,
    address indexed serverAgent,
    uint8 rating,
    bytes32 dataHash,
    string feedbackType,
    uint256 timestamp
);
```

---

## 🔄 Migration Strategy

### Phase 1: Deploy New Contracts ✅
```
1. Deploy SageReputationRegistry
2. Deploy SageValidationRegistry
3. Deploy SageERC8004Registry (coordinator)
4. Set cross-contract references
```

### Phase 2: Integration ✅
```
1. Update SageRegistryV2 to emit ERC-8004 compatible events
2. Add helper functions for cross-registry queries
3. Update documentation
```

### Phase 3: Testing ✅
```
1. Unit tests for each registry
2. Integration tests for full lifecycle
3. Gas optimization tests
4. Security audit
```

### Phase 4: Deployment ✅
```
1. Deploy to testnet (Kairos)
2. Community testing period
3. Deploy to mainnet
4. Announce ERC-8004 compliance
```

---

## 🛠️ Development Timeline

### Week 1-2: Contract Development
- [ ] Implement SageReputationRegistry.sol
- [ ] Implement SageValidationRegistry.sol
- [ ] Implement SageERC8004Registry.sol
- [ ] Write comprehensive interfaces

### Week 3: Testing
- [ ] Unit tests (100% coverage)
- [ ] Integration tests
- [ ] Gas optimization
- [ ] Security review

### Week 4: Documentation & Deployment
- [ ] API documentation
- [ ] Deployment scripts
- [ ] Testnet deployment
- [ ] User guide

---

## 📚 Dependencies

### Existing Contracts
- ✅ SageRegistryV2.sol (Identity Registry)
- ✅ ISageRegistry.sol (Interface)
- ✅ SageVerificationHook.sol (Hooks)

### New Dependencies
- OpenZeppelin Contracts (if not already included)
- TEE verification libraries (future)

### Development Tools
- Hardhat (existing)
- Ethers.js v6 (existing)
- Chai/Mocha (existing)

---

## 🎯 Success Metrics

### Technical Metrics
- [ ] 100% test coverage
- [ ] Gas costs < 500k per operation
- [ ] No critical security issues
- [ ] Full ERC-8004 compliance

### Ecosystem Metrics
- [ ] Integration with at least 3 agent platforms
- [ ] 100+ registered agents using reputation system
- [ ] 50+ validators in validation network

---

## 🔗 Integration Points

### With Existing SAGE Components

```
SageRegistryV2 (Identity)
    ↓
SageReputationRegistry
    ↓ (agent verification)
SageValidationRegistry
    ↓ (cross-check)
SAGE Message Protocol
    ↓ (real-time communication)
Off-chain Aggregation Service
    ↓ (reputation scoring)
UI/UX Components
```

---

## 📝 Notes

### Design Decisions

1. **Why separate registries?**
   - Modularity: Each registry can be upgraded independently
   - Gas efficiency: Only pay for features you use
   - Flexibility: Support different validation models

2. **Why minimal on-chain storage?**
   - Reduce gas costs
   - Enable off-chain innovation
   - Faster queries via events

3. **Why pre-authorization?**
   - Prevent spam
   - Ensure legitimate feedback
   - Maintain data quality

### Future Enhancements

1. **Off-chain Reputation Aggregation Service**
   - Calculate weighted reputation scores
   - Time-decay for old feedback
   - Multi-dimensional reputation (speed, quality, reliability)

2. **Advanced TEE Support**
   - Intel SGX attestation verification
   - AMD SEV integration
   - ARM TrustZone support

3. **Cross-chain Support**
   - Bridge to other EVM chains
   - L2 optimization
   - Multi-chain agent identity

---

*Document Version: 1.0*
*Last Updated: 2025-10-06*
*Author: SAGE Development Team*
