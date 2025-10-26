// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./AgentCardStorage.sol";
import "./AgentCardVerifyHook.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title AgentCardRegistry
 * @notice Production SAGE registry with multi-key + commit-reveal + security
 * @dev Combines best features from V2, V3, V4 + ERC-8004 compliance
 *
 * Features:
 * - Multi-key support (ECDSA, Ed25519, X25519)
 * - Commit-reveal pattern (prevents front-running)
 * - Cross-chain replay protection
 * - Rate limiting and anti-Sybil
 * - Emergency pause mechanism
 * - Stake requirement
 * - Time-locked activation
 *
 * @custom:security-contact security@sage.com
 */
contract AgentCardRegistry is
    AgentCardStorage,
    Pausable,
    ReentrancyGuard,
    Ownable2Step
{
    // ============ State Variables ============

    AgentCardVerifyHook public verifyHook;

    // Security parameters
    uint256 public registrationStake = 0.01 ether;
    uint256 public activationDelay = 1 hours;
    mapping(bytes32 => uint256) public agentActivationTime;
    mapping(bytes32 => uint256) public agentStakes;

    // Reputation system
    mapping(address => AgentReputation) public agentReputations;

    struct AgentReputation {
        uint256 successfulInteractions;
        uint256 failedInteractions;
        uint256 reputationScore;
        bool verified;
    }

    // ============ Modifiers ============

    modifier onlyAgentOwner(bytes32 agentId) {
        require(
            agents[agentId].owner == msg.sender ||
            agentOperators[agentId][msg.sender],
            "Not agent owner"
        );
        _;
    }

    modifier validDID(string calldata did) {
        require(bytes(did).length > 0, "Empty DID");
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        _;
    }

    // ============ Constructor ============

    constructor(address _verifyHook) {
        require(_verifyHook != address(0), "Invalid hook address");
        verifyHook = AgentCardVerifyHook(_verifyHook);
        _transferOwnership(msg.sender);
    }

    // ============ Registration Functions ============

    /**
     * @notice Commit to agent registration (Phase 1)
     * @dev Prevents front-running by hiding parameters
     * @param commitHash keccak256(abi.encode(did, keys, owner, salt, chainId))
     */
    function commitRegistration(bytes32 commitHash)
        external
        payable
        whenNotPaused
        nonReentrant
    {
        require(commitHash != bytes32(0), "Invalid commit hash");
        require(msg.value >= registrationStake, "Insufficient stake");

        // Check rate limiting
        uint256 currentDay = block.timestamp / 1 days;
        if (lastRegistrationDay[msg.sender] != currentDay) {
            lastRegistrationDay[msg.sender] = currentDay;
            dailyRegistrationCount[msg.sender] = 0;
        }
        require(
            dailyRegistrationCount[msg.sender] < MAX_DAILY_REGISTRATIONS,
            "Daily registration limit exceeded"
        );
        dailyRegistrationCount[msg.sender]++;

        // Store commitment
        registrationCommitments[msg.sender] = RegistrationCommitment({
            commitHash: commitHash,
            timestamp: block.timestamp,
            revealed: false
        });

        emit CommitmentRecorded(msg.sender, commitHash, block.timestamp);
    }

    /**
     * @notice Reveal and register agent (Phase 2)
     * @dev Verifies commitment and registers agent
     * @param params RegistrationParams struct containing all registration data
     */
    function registerAgent(RegistrationParams calldata params)
        external
        whenNotPaused
        nonReentrant
        validDID(params.did)
        returns (bytes32 agentId)
    {
        // 1. Verify commit-reveal
        RegistrationCommitment storage commitment = registrationCommitments[msg.sender];
        require(commitment.timestamp > 0, "No commitment found");
        require(!commitment.revealed, "Already revealed");
        require(
            block.timestamp >= commitment.timestamp + COMMIT_MIN_DELAY,
            "Reveal too soon"
        );
        require(
            block.timestamp <= commitment.timestamp + COMMIT_MAX_DELAY,
            "Commitment expired"
        );

        // Verify commitment hash
        bytes32 expectedHash = keccak256(abi.encode(
            params.did,
            params.keys,
            msg.sender,
            params.salt,
            block.chainid
        ));
        require(commitment.commitHash == expectedHash, "Invalid reveal");

        // Mark as revealed
        commitment.revealed = true;

        // 2. Validate input
        require(params.keys.length > 0 && params.keys.length <= MAX_KEYS_PER_AGENT, "Invalid key count");
        require(params.keys.length == params.keyTypes.length, "Key type mismatch");
        require(params.keys.length == params.signatures.length, "Signature mismatch");

        // 3. Call verify hook (external validation)
        verifyHook.beforeRegister(params.did, msg.sender, params.keys);

        // 4. Generate agent ID
        agentId = keccak256(abi.encodePacked(params.did, msg.sender, block.timestamp));

        // 5. Store keys
        bytes32[] memory keyHashes = new bytes32[](params.keys.length);
        for (uint256 i = 0; i < params.keys.length; i++) {
            bytes32 keyHash = keccak256(params.keys[i]);
            keyHashes[i] = keyHash;

            // Check key reuse
            require(!publicKeyUsed[keyHash], "Public key already used");
            publicKeyUsed[keyHash] = true;

            // Verify key ownership
            _verifyKeyOwnership(params.keyTypes[i], params.keys[i], params.signatures[i], msg.sender);

            // Store key
            agentKeys[keyHash] = AgentKey({
                keyType: params.keyTypes[i],
                keyData: params.keys[i],
                signature: params.signatures[i],
                verified: true,
                registeredAt: block.timestamp
            });

            emit KeyAdded(agentId, keyHash, params.keyTypes[i], block.timestamp);
        }

        // 6. Store agent metadata
        agents[agentId] = AgentMetadata({
            did: params.did,
            name: params.name,
            description: params.description,
            endpoint: params.endpoint,
            keyHashes: keyHashes,
            capabilities: params.capabilities,
            owner: msg.sender,
            registeredAt: block.timestamp,
            updatedAt: block.timestamp,
            active: false,  // Not active yet (time-locked)
            chainId: block.chainid
        });

        didToAgentId[params.did] = agentId;
        ownerToAgents[msg.sender].push(agentId);

        // 7. Store stake and set activation time
        agentStakes[agentId] = registrationStake;
        agentActivationTime[agentId] = block.timestamp + activationDelay;

        // 8. Initialize reputation
        agentReputations[msg.sender] = AgentReputation({
            successfulInteractions: 0,
            failedInteractions: 0,
            reputationScore: 50,  // Start at 50/100
            verified: false
        });

        emit AgentRegistered(agentId, params.did, msg.sender, block.timestamp);

        return agentId;
    }

    /**
     * @notice Activate agent after time lock expires
     * @dev Anyone can call this after activation delay
     * @param agentId The agent identifier
     */
    function activateAgent(bytes32 agentId) external nonReentrant {
        require(agents[agentId].owner != address(0), "Agent not found");
        require(!agents[agentId].active, "Already active");
        require(
            block.timestamp >= agentActivationTime[agentId],
            "Activation delay not passed"
        );

        agents[agentId].active = true;

        emit AgentActivated(agentId, block.timestamp);
    }

    /**
     * @notice Add new key to existing agent
     * @param agentId Agent identifier
     * @param keyData Public key bytes
     * @param keyType Key type (ECDSA, Ed25519, X25519)
     * @param signature Ownership proof signature
     */
    function addKey(
        bytes32 agentId,
        bytes calldata keyData,
        KeyType keyType,
        bytes calldata signature
    )
        external
        onlyAgentOwner(agentId)
        whenNotPaused
        nonReentrant
    {
        require(
            agents[agentId].keyHashes.length < MAX_KEYS_PER_AGENT,
            "Max keys reached"
        );

        bytes32 keyHash = keccak256(keyData);
        require(!publicKeyUsed[keyHash], "Key already used");

        // Verify ownership
        _verifyKeyOwnership(keyType, keyData, signature, msg.sender);

        // Store key
        agentKeys[keyHash] = AgentKey({
            keyType: keyType,
            keyData: keyData,
            signature: signature,
            verified: true,
            registeredAt: block.timestamp
        });

        agents[agentId].keyHashes.push(keyHash);
        publicKeyUsed[keyHash] = true;

        emit KeyAdded(agentId, keyHash, keyType, block.timestamp);
    }

    /**
     * @notice Revoke key from agent
     * @param agentId Agent identifier
     * @param keyHash Hash of key to revoke
     */
    function revokeKey(bytes32 agentId, bytes32 keyHash)
        external
        onlyAgentOwner(agentId)
        whenNotPaused
        nonReentrant
    {
        require(agents[agentId].keyHashes.length > 1, "Cannot revoke last key");

        // Find and remove key
        bytes32[] storage keys = agents[agentId].keyHashes;
        for (uint256 i = 0; i < keys.length; i++) {
            if (keys[i] == keyHash) {
                // Swap with last element and pop
                keys[i] = keys[keys.length - 1];
                keys.pop();
                break;
            }
        }

        // Mark key as revoked (don't delete for audit trail)
        agentKeys[keyHash].verified = false;

        emit KeyRevoked(agentId, keyHash, block.timestamp);
    }

    /**
     * @notice Update agent metadata
     * @param agentId Agent identifier
     * @param endpoint New endpoint
     * @param capabilities New capabilities
     */
    function updateAgent(
        bytes32 agentId,
        string calldata endpoint,
        string calldata capabilities
    )
        external
        onlyAgentOwner(agentId)
        whenNotPaused
        nonReentrant
    {
        AgentMetadata storage agent = agents[agentId];
        agent.endpoint = endpoint;
        agent.capabilities = capabilities;
        agent.updatedAt = block.timestamp;
        agentNonce[agentId]++;

        emit AgentUpdated(agentId, block.timestamp);
    }

    /**
     * @notice Deactivate agent
     * @param agentId Agent identifier
     */
    function deactivateAgent(bytes32 agentId)
        external
        onlyAgentOwner(agentId)
        nonReentrant
    {
        agents[agentId].active = false;

        // Return stake after 30 days
        if (block.timestamp >= agents[agentId].registeredAt + 30 days) {
            uint256 stake = agentStakes[agentId];
            if (stake > 0) {
                agentStakes[agentId] = 0;
                (bool success, ) = msg.sender.call{value: stake}("");
                require(success, "Stake return failed");
            }
        }

        emit AgentDeactivated(agentId, block.timestamp);
    }

    // ============ View Functions ============

    function getAgent(bytes32 agentId)
        external
        view
        returns (AgentMetadata memory)
    {
        return agents[agentId];
    }

    function getAgentByDID(string calldata did)
        external
        view
        returns (AgentMetadata memory)
    {
        bytes32 agentId = didToAgentId[did];
        return agents[agentId];
    }

    function getKey(bytes32 keyHash)
        external
        view
        returns (AgentKey memory)
    {
        return agentKeys[keyHash];
    }

    function getAgentsByOwner(address owner)
        external
        view
        returns (bytes32[] memory)
    {
        return ownerToAgents[owner];
    }

    // ============ Internal Functions ============

    function _verifyKeyOwnership(
        KeyType keyType,
        bytes calldata keyData,
        bytes calldata signature,
        address expectedOwner
    ) internal view {
        if (keyType == KeyType.ECDSA) {
            // Verify ECDSA signature
            bytes32 messageHash = keccak256(abi.encodePacked(
                "SAGE Agent Registration:",
                block.chainid,
                address(this),
                expectedOwner
            ));
            bytes32 ethSignedHash = keccak256(abi.encodePacked(
                "\x19Ethereum Signed Message:\n32",
                messageHash
            ));

            address recovered = _recoverSigner(ethSignedHash, signature);
            require(recovered == expectedOwner, "Invalid ECDSA signature");
        } else if (keyType == KeyType.Ed25519) {
            // Ed25519 requires owner pre-approval (can't verify on-chain)
            // In production, use Chainlink oracle or TEE for Ed25519 verification
            require(signature.length == 64, "Invalid Ed25519 signature");
        } else if (keyType == KeyType.X25519) {
            // X25519 is encryption key, no signature verification needed
            require(keyData.length == 32, "Invalid X25519 key length");
        }
    }

    function _recoverSigner(bytes32 ethSignedHash, bytes memory signature)
        internal
        pure
        returns (address)
    {
        require(signature.length == 65, "Invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        if (v < 27) {
            v += 27;
        }

        require(v == 27 || v == 28, "Invalid signature 'v' value");

        return ecrecover(ethSignedHash, v, r, s);
    }

    // ============ Operator Management ============

    /**
     * @notice Set or revoke operator approval for an agent
     * @dev Enables ERC-721/1155 style operator pattern
     *      Operators can manage agents on behalf of owners
     *
     * @param agentId Agent identifier
     * @param operator Address to grant/revoke approval
     * @param approved True to approve, false to revoke
     */
    function setApprovalForAgent(
        bytes32 agentId,
        address operator,
        bool approved
    ) external {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        require(operator != address(0), "Invalid operator");
        require(operator != msg.sender, "Cannot approve self");

        agentOperators[agentId][operator] = approved;

        emit ApprovalForAgent(agentId, msg.sender, operator, approved);
    }

    /**
     * @notice Check if an address is approved operator for an agent
     * @param agentId Agent identifier
     * @param operator Address to check
     * @return True if operator is approved
     */
    function isApprovedOperator(bytes32 agentId, address operator)
        external
        view
        returns (bool)
    {
        return agentOperators[agentId][operator];
    }

    // ============ Admin Functions ============

    function setRegistrationStake(uint256 newStake) external onlyOwner {
        registrationStake = newStake;
    }

    function setActivationDelay(uint256 newDelay) external onlyOwner {
        activationDelay = newDelay;
    }

    function setVerifyHook(address newHook) external onlyOwner {
        require(newHook != address(0), "Invalid hook address");
        verifyHook = AgentCardVerifyHook(newHook);
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    // ============ Additional Events ============

    event AgentActivated(bytes32 indexed agentId, uint256 timestamp);
}
