// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/ISageRegistry.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title SageRegistryV3
 * @notice SAGE AI Agent Registry with Front-Running Protection
 * @dev Adds commit-reveal scheme to prevent front-running of agent registration
 *
 * Key Features:
 * - Commit-reveal pattern for DID registration
 * - Enhanced public key validation
 * - Emergency pause mechanism
 * - Two-step ownership transfer
 * - Front-running protection
 *
 * Security Improvements from V2:
 * - MEDIUM-1: Front-running protection via commit-reveal
 * - MEDIUM-2: Cross-chain replay protection (chainId in signatures)
 */
contract SageRegistryV3 is ISageRegistry, Pausable, Ownable2Step {
    // ============================================
    // STRUCTS
    // ============================================

    struct RegistrationParams {
        string did;
        string name;
        string description;
        string endpoint;
        bytes publicKey;
        string capabilities;
        bytes signature;
    }

    struct KeyValidation {
        bytes32 keyHash;
        uint256 registrationBlock;
        bool isRevoked;
    }

    // Commit-reveal structure
    struct RegistrationCommitment {
        bytes32 commitHash;
        uint256 timestamp;
        bool revealed;
    }

    // ============================================
    // STATE VARIABLES
    // ============================================

    // Agent storage
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(address => uint256) private registrationNonce;
    mapping(bytes32 => uint256) private agentNonce;

    // Key validation
    mapping(bytes32 => KeyValidation) private keyValidations;
    mapping(address => bytes32) private addressToKeyHash;
    mapping(bytes32 => bytes32[]) private keyHashToAgentIds;

    // Commit-reveal for front-running protection
    mapping(address => RegistrationCommitment) public registrationCommitments;

    // Hooks
    address public beforeRegisterHook;
    address public afterRegisterHook;

    // ============================================
    // CONSTANTS
    // ============================================

    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MIN_PUBLIC_KEY_LENGTH = 32;
    uint256 private constant MAX_PUBLIC_KEY_LENGTH = 65;
    uint256 private constant HOOK_GAS_LIMIT = 50000;

    // Commit-reveal timing
    uint256 private constant MIN_COMMIT_REVEAL_DELAY = 1 minutes;  // Minimum wait after commit
    uint256 private constant MAX_COMMIT_REVEAL_DELAY = 1 hours;    // Maximum wait before expiry

    // ============================================
    // EVENTS
    // ============================================

    event KeyValidated(bytes32 indexed keyHash, address indexed owner);
    event KeyRevoked(bytes32 indexed keyHash, address indexed owner);
    event HookFailed(address indexed hook, string reason);
    event BeforeRegisterHookUpdated(address indexed oldHook, address indexed newHook);
    event AfterRegisterHookUpdated(address indexed oldHook, address indexed newHook);

    // Commit-reveal events
    event RegistrationCommitted(address indexed committer, bytes32 indexed commitHash, uint256 timestamp);
    event RegistrationRevealed(address indexed revealer, bytes32 indexed agentId, string did);
    event CommitmentExpired(address indexed committer, bytes32 indexed commitHash);

    // ============================================
    // ERRORS
    // ============================================

    error AlreadyCommitted();
    error NoCommitmentFound();
    error InvalidReveal();
    error RevealTooSoon(uint256 currentTime, uint256 minTime);
    error RevealTooLate(uint256 currentTime, uint256 maxTime);
    error CommitmentAlreadyRevealed();
    error InvalidCommitHash();

    // ============================================
    // MODIFIERS
    // ============================================

    modifier onlyAgentOwner(bytes32 agentId) {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        _;
    }

    // ============================================
    // CONSTRUCTOR
    // ============================================

    constructor() {
        _transferOwnership(msg.sender);
    }

    // ============================================
    // COMMIT-REVEAL FUNCTIONS
    // ============================================

    /**
     * @notice Commit to a future agent registration (Step 1 of 2)
     * @dev Prevents front-running by hiding registration intent
     *
     * The commitment hash is:
     * keccak256(abi.encodePacked(did, publicKey, msg.sender, salt, chainId))
     *
     * @param commitHash Hash of registration parameters + salt
     *
     * Process:
     * 1. User creates commitment hash off-chain
     * 2. User calls commitRegistration(commitHash)
     * 3. Wait MIN_COMMIT_REVEAL_DELAY (prevents instant reveal)
     * 4. User calls registerAgentWithReveal() with actual parameters
     * 5. Contract verifies hash matches commitment
     *
     * Security:
     * - Attacker cannot see which DID user wants to register
     * - Attacker cannot front-run because they don't know the salt
     * - User must wait minimum delay before reveal
     * - Commitment expires after maximum delay
     *
     * Example:
     * ```javascript
     * const did = "did:sage:alice";
     * const publicKey = "0x...";
     * const salt = ethers.randomBytes(32);
     * const chainId = await ethers.provider.getNetwork().chainId;
     *
     * const commitHash = ethers.keccak256(
     *   ethers.solidityPacked(
     *     ["string", "bytes", "address", "bytes32", "uint256"],
     *     [did, publicKey, userAddress, salt, chainId]
     *   )
     * );
     *
     * await registry.commitRegistration(commitHash);
     * // Wait 1 minute...
     * await registry.registerAgentWithReveal(did, name, ..., salt);
     * ```
     */
    function commitRegistration(bytes32 commitHash) external whenNotPaused {
        // Input validation
        if (commitHash == bytes32(0)) revert InvalidCommitHash();

        RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

        // Check if already committed and not expired
        if (commitment.timestamp > 0 && !commitment.revealed) {
            if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {
                revert AlreadyCommitted();
            } else {
                // Old commitment expired, emit event
                emit CommitmentExpired(msg.sender, commitment.commitHash);
            }
        }

        // Store new commitment
        registrationCommitments[msg.sender] = RegistrationCommitment({
            commitHash: commitHash,
            timestamp: block.timestamp,
            revealed: false
        });

        emit RegistrationCommitted(msg.sender, commitHash, block.timestamp);
    }

    /**
     * @notice Register agent with reveal (Step 2 of 2)
     * @dev Verifies commitment and completes registration
     *
     * @param did Decentralized Identifier
     * @param name Agent name
     * @param description Agent description
     * @param endpoint Agent API endpoint
     * @param publicKey Agent's public key
     * @param capabilities Agent capabilities
     * @param signature Ownership proof signature
     * @param salt Random salt used in commitment
     * @return agentId Unique identifier for the registered agent
     */
    function registerAgentWithReveal(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature,
        bytes32 salt
    ) external whenNotPaused returns (bytes32) {
        RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

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

        // Verify commitment hash matches revealed parameters
        bytes32 expectedHash = keccak256(abi.encodePacked(
            did,
            publicKey,
            msg.sender,
            salt,
            block.chainid  // Include chainId for cross-chain protection
        ));

        if (commitment.commitHash != expectedHash) revert InvalidReveal();

        // Mark as revealed
        commitment.revealed = true;

        // Validate public key format and ownership
        _validatePublicKey(publicKey, signature);

        // Proceed with registration
        RegistrationParams memory params = RegistrationParams({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            publicKey: publicKey,
            capabilities: capabilities,
            signature: signature
        });

        bytes32 agentId = _registerAgent(params);

        emit RegistrationRevealed(msg.sender, agentId, did);

        return agentId;
    }

    /**
     * @notice Legacy registration function (without front-running protection)
     * @dev Kept for backward compatibility, but registerAgentWithReveal() is recommended
     * @custom:security-warning Vulnerable to front-running attacks
     */
    function registerAgent(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature
    ) external whenNotPaused returns (bytes32) {
        // Validate public key format and ownership
        _validatePublicKey(publicKey, signature);

        RegistrationParams memory params = RegistrationParams({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            publicKey: publicKey,
            capabilities: capabilities,
            signature: signature
        });

        return _registerAgent(params);
    }

    // ============================================
    // VALIDATION FUNCTIONS
    // ============================================

    /**
     * @notice Enhanced public key validation with cross-chain protection
     * @dev Validates format, non-zero, and ownership through signature
     */
    function _validatePublicKey(
        bytes calldata publicKey,
        bytes calldata signature
    ) internal {
        // 1. Length validation
        require(
            publicKey.length >= MIN_PUBLIC_KEY_LENGTH &&
            publicKey.length <= MAX_PUBLIC_KEY_LENGTH,
            "Invalid public key length"
        );

        // 2. Format validation for secp256k1
        if (publicKey.length == 65) {
            require(publicKey[0] == 0x04, "Invalid uncompressed key format");
        } else if (publicKey.length == 33) {
            require(
                publicKey[0] == 0x02 || publicKey[0] == 0x03,
                "Invalid compressed key format"
            );
        } else if (publicKey.length == 32) {
            revert("Ed25519 not supported on-chain");
        }

        // 3. Non-zero validation
        bytes32 keyHash = keccak256(publicKey);
        bool isNonZero = false;
        uint startIdx = (publicKey.length == 65 && publicKey[0] == 0x04) ? 1 :
                        (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) ? 1 : 0;

        for (uint i = startIdx; i < publicKey.length; i++) {
            if (publicKey[i] != 0) {
                isNonZero = true;
                break;
            }
        }
        require(isNonZero, "Invalid zero key");

        // 4. Ownership proof through signature (with chainId for cross-chain protection)
        bytes32 challenge = keccak256(abi.encodePacked(
            "SAGE Key Registration:",
            block.chainid,          // Cross-chain replay protection
            address(this),
            msg.sender,
            keyHash
        ));

        // Verify signature
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", challenge)
        );

        address recovered = _recoverSigner(ethSignedHash, signature);
        address keyAddress = _getAddressFromPublicKey(publicKey);

        require(recovered == keyAddress, "Key ownership not proven");
        require(recovered != address(0), "Invalid signature");

        // 5. Check if key has been revoked
        if (keyValidations[keyHash].registrationBlock > 0) {
            require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        }

        // 6. Store validation data
        if (keyValidations[keyHash].registrationBlock == 0) {
            keyValidations[keyHash] = KeyValidation({
                keyHash: keyHash,
                registrationBlock: block.number,
                isRevoked: false
            });
        }

        addressToKeyHash[keyAddress] = keyHash;
        addressToKeyHash[msg.sender] = keyHash;

        emit KeyValidated(keyHash, msg.sender);
    }

    /**
     * @notice Validate DID format according to W3C DID spec
     */
    function _isValidDID(string memory did) private pure returns (bool) {
        bytes memory didBytes = bytes(did);
        uint256 len = didBytes.length;

        if (len < 7) return false;

        // Must start with "did:"
        if (didBytes[0] != 'd' || didBytes[1] != 'i' || didBytes[2] != 'd' || didBytes[3] != ':') {
            return false;
        }

        // Find second colon
        uint256 secondColonIndex = 0;
        for (uint256 i = 4; i < len; i++) {
            if (didBytes[i] == ':') {
                secondColonIndex = i;
                break;
            }
        }

        if (secondColonIndex == 0 || secondColonIndex == len - 1) {
            return false;
        }

        // Validate method (lowercase alphanumeric)
        for (uint256 i = 4; i < secondColonIndex; i++) {
            bytes1 char = didBytes[i];
            if (!((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'))) {
                return false;
            }
        }

        return true;
    }

    // ============================================
    // INTERNAL REGISTRATION LOGIC
    // ============================================

    function _registerAgent(RegistrationParams memory params) private returns (bytes32) {
        _validateRegistrationInputs(params.did, params.name);

        bytes32 agentId = _generateAgentId(params.did, params.publicKey);

        _executeBeforeHook(agentId, params.did, params.publicKey);
        _storeAgentMetadata(agentId, params);
        _executeAfterHook(agentId, params.did, params.publicKey);

        return agentId;
    }

    function _validateRegistrationInputs(string memory did, string memory name) private view {
        require(bytes(did).length > 0, "DID required");
        require(_isValidDID(did), "Invalid DID format");
        require(bytes(name).length > 0, "Name required");
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        require(ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER, "Too many agents");
    }

    function _generateAgentId(string memory did, bytes memory publicKey) private returns (bytes32) {
        uint256 nonce = registrationNonce[msg.sender];
        registrationNonce[msg.sender]++;

        return keccak256(abi.encodePacked(
            did,
            publicKey,
            msg.sender,
            block.number,
            nonce
        ));
    }

    function _storeAgentMetadata(bytes32 agentId, RegistrationParams memory params) private {
        agents[agentId] = AgentMetadata({
            did: params.did,
            name: params.name,
            description: params.description,
            endpoint: params.endpoint,
            publicKey: params.publicKey,
            capabilities: params.capabilities,
            owner: msg.sender,
            registeredAt: block.timestamp,
            updatedAt: block.timestamp,
            active: true
        });

        didToAgentId[params.did] = agentId;
        ownerToAgents[msg.sender].push(agentId);
        agentNonce[agentId]++;

        bytes32 keyHash = keccak256(params.publicKey);
        keyHashToAgentIds[keyHash].push(agentId);

        emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);
    }

    function _executeBeforeHook(bytes32 agentId, string memory did, bytes memory publicKey) private {
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);

            try IRegistryHook(beforeRegisterHook).beforeRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) returns (bool success, string memory reason) {
                require(success, reason);
            } catch Error(string memory reason) {
                emit HookFailed(beforeRegisterHook, reason);
                revert(reason);
            } catch (bytes memory) {
                emit HookFailed(beforeRegisterHook, "Hook call failed");
                revert("Hook call failed");
            }
        }
    }

    function _executeAfterHook(bytes32 agentId, string memory did, bytes memory publicKey) private {
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);

            try IRegistryHook(afterRegisterHook).afterRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) {
                emit AfterRegisterHook(agentId, msg.sender, hookData);
            } catch Error(string memory reason) {
                emit HookFailed(afterRegisterHook, reason);
            } catch (bytes memory) {
                emit HookFailed(afterRegisterHook, "Hook call failed");
            }
        }
    }

    // ============================================
    // AGENT MANAGEMENT
    // ============================================

    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external onlyAgentOwner(agentId) {
        require(bytes(name).length > 0, "Name required");

        bytes32 keyHash = keccak256(agents[agentId].publicKey);
        require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        require(agents[agentId].active, "Agent not active");

        // Include chainId in update signature for cross-chain protection
        bytes32 messageHash = keccak256(abi.encodePacked(
            agentId,
            name,
            description,
            endpoint,
            capabilities,
            msg.sender,
            agentNonce[agentId],
            block.chainid  // Cross-chain replay protection
        ));

        require(
            _verifySignature(messageHash, signature, agents[agentId].publicKey, msg.sender),
            "Invalid signature"
        );

        agents[agentId].name = name;
        agents[agentId].description = description;
        agents[agentId].endpoint = endpoint;
        agents[agentId].capabilities = capabilities;
        agents[agentId].updatedAt = block.timestamp;

        agentNonce[agentId]++;

        emit AgentUpdated(agentId, msg.sender, block.timestamp);
    }

    function deactivateAgent(bytes32 agentId) external onlyAgentOwner(agentId) {
        require(agents[agentId].active, "Agent already inactive");
        agents[agentId].active = false;
        agents[agentId].updatedAt = block.timestamp;
        emit AgentDeactivated(agentId, msg.sender, block.timestamp);
    }

    function deactivateAgentByDID(string calldata did) external {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        require(agents[agentId].active, "Agent already inactive");

        agents[agentId].active = false;
        agents[agentId].updatedAt = block.timestamp;

        emit AgentDeactivated(agentId, msg.sender, block.timestamp);
    }

    function revokeKey(bytes calldata publicKey) external {
        bytes32 keyHash = keccak256(publicKey);
        require(addressToKeyHash[msg.sender] == keyHash, "Not key owner");
        require(!keyValidations[keyHash].isRevoked, "Already revoked");

        keyValidations[keyHash].isRevoked = true;

        bytes32[] memory agentIds = keyHashToAgentIds[keyHash];
        for (uint i = 0; i < agentIds.length; i++) {
            agents[agentIds[i]].active = false;
        }

        emit KeyRevoked(keyHash, msg.sender);
    }

    // ============================================
    // VIEW FUNCTIONS
    // ============================================

    function getAgent(bytes32 agentId) external view returns (AgentMetadata memory) {
        require(agents[agentId].registeredAt > 0, "Agent not found");
        return agents[agentId];
    }

    function getAgentByDID(string calldata did) external view returns (AgentMetadata memory) {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        return agents[agentId];
    }

    function getAgentsByOwner(address _owner) external view returns (bytes32[] memory) {
        return ownerToAgents[_owner];
    }

    function verifyAgentOwnership(bytes32 agentId, address claimedOwner) external view returns (bool) {
        return agents[agentId].owner == claimedOwner;
    }

    function isAgentActive(bytes32 agentId) external view returns (bool) {
        return agents[agentId].active;
    }

    function isKeyValid(bytes calldata publicKey) external view returns (bool) {
        bytes32 keyHash = keccak256(publicKey);
        KeyValidation memory validation = keyValidations[keyHash];
        return validation.registrationBlock > 0 && !validation.isRevoked;
    }

    function getCommitment(address user) external view returns (
        bytes32 commitHash,
        uint256 timestamp,
        bool revealed,
        bool isExpired
    ) {
        RegistrationCommitment memory commitment = registrationCommitments[user];
        bool expired = commitment.timestamp > 0 &&
                      block.timestamp > commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

        return (
            commitment.commitHash,
            commitment.timestamp,
            commitment.revealed,
            expired
        );
    }

    // ============================================
    // ADMIN FUNCTIONS
    // ============================================

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    function setBeforeRegisterHook(address hook) external onlyOwner {
        address oldHook = beforeRegisterHook;
        beforeRegisterHook = hook;
        emit BeforeRegisterHookUpdated(oldHook, hook);
    }

    function setAfterRegisterHook(address hook) external onlyOwner {
        address oldHook = afterRegisterHook;
        afterRegisterHook = hook;
        emit AfterRegisterHookUpdated(oldHook, hook);
    }

    // ============================================
    // HELPER FUNCTIONS
    // ============================================

    function _verifySignature(
        bytes32 messageHash,
        bytes memory signature,
        bytes memory publicKey,
        address expectedSigner
    ) private pure returns (bool) {
        if (publicKey.length == 64 || publicKey.length == 65) {
            bytes32 ethSignedHash = keccak256(
                abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
            );

            address recovered = _recoverSigner(ethSignedHash, signature);
            return recovered == expectedSigner;
        }

        if (publicKey.length == 32) {
            revert("Ed25519 not supported on-chain");
        }

        return false;
    }

    function _recoverSigner(bytes32 messageHash, bytes memory signature)
        private pure returns (address)
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

        return ecrecover(messageHash, v, r, s);
    }

    function _getAddressFromPublicKey(bytes memory publicKey)
        private pure returns (address)
    {
        if (publicKey.length == 65 && publicKey[0] == 0x04) {
            bytes memory keyWithoutPrefix = new bytes(64);
            for (uint i = 0; i < 64; i++) {
                keyWithoutPrefix[i] = publicKey[i + 1];
            }
            return address(uint160(uint256(keccak256(keyWithoutPrefix))));
        }

        if (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) {
            revert("Compressed key address derivation not supported");
        }

        revert("Invalid public key format for address derivation");
    }
}
