// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/ISageRegistryV4.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title SageRegistryV4
 * @notice SAGE AI Agent Registry Contract with Multi-Key Support
 * @dev Implements secure registration and management of AI agents with multiple public keys
 *      Supports Ed25519, ECDSA/secp256k1, and X25519 key types for multi-chain compatibility
 */
contract SageRegistryV4 is ISageRegistryV4, ReentrancyGuard {
    // State variables
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(bytes32 => AgentKey) private agentKeys;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(bytes32 => uint256) private agentNonce;

    address public immutable OWNER;
    address public beforeRegisterHook;
    address public afterRegisterHook;

    // Agent limits
    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MAX_KEYS_PER_AGENT = 10;

    // Public key length constants
    uint256 private constant ED25519_KEY_LENGTH = 32;
    uint256 private constant X25519_KEY_LENGTH = 32;
    uint256 private constant SECP256K1_COMPRESSED_LENGTH = 33;
    uint256 private constant SECP256K1_UNCOMPRESSED_LENGTH = 65;
    uint256 private constant SECP256K1_RAW_LENGTH = 64;

    // Signature constants
    uint256 private constant ECDSA_SIGNATURE_LENGTH = 65;
    uint256 private constant ED25519_SIGNATURE_LENGTH = 64;

    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == OWNER, "Only owner");
        _;
    }

    modifier onlyAgentOwner(bytes32 agentId) {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        _;
    }

    constructor() {
        OWNER = msg.sender;
    }

    /**
     * @notice Register a new AI agent with multiple keys
     * @dev Verifies signatures for ECDSA keys, requires pre-approval for Ed25519
     */
    function registerAgent(RegistrationParams calldata params)
        external
        nonReentrant
        returns (bytes32)
    {
        require(params.keyTypes.length == params.keyData.length, "Key arrays length mismatch");
        require(params.keyTypes.length == params.signatures.length, "Signature arrays length mismatch");
        require(params.keyTypes.length > 0 && params.keyTypes.length <= MAX_KEYS_PER_AGENT, "Invalid key count");

        // Input validation
        _validateRegistrationInputs(params.did, params.name);

        // Generate agent ID
        bytes32 agentId = _generateAgentId(params.did, params.keyData[0]);

        // Execute before hook
        _executeBeforeHook(agentId, params.did, params.keyData);

        // Process and verify each key
        bytes32[] memory keyHashes = new bytes32[](params.keyTypes.length);
        for (uint256 i = 0; i < params.keyTypes.length; i++) {
            bytes32 keyHash = _processAndStoreKey(
                agentId,
                params.keyTypes[i],
                params.keyData[i],
                params.signatures[i]
            );
            keyHashes[i] = keyHash;
        }

        // Store agent metadata
        _storeAgentMetadata(
            agentId,
            params.did,
            params.name,
            params.description,
            params.endpoint,
            params.capabilities,
            keyHashes
        );

        // Execute after hook
        _executeAfterHook(agentId, params.did, params.keyData);

        return agentId;
    }

    /**
     * @notice Add a new key to an existing agent
     */
    function addKey(
        bytes32 agentId,
        KeyType keyType,
        bytes calldata keyData,
        bytes calldata signature
    ) external onlyAgentOwner(agentId) returns (bytes32) {
        require(agents[agentId].active, "Agent not active");
        require(agents[agentId].keyHashes.length < MAX_KEYS_PER_AGENT, "Too many keys");

        bytes32 keyHash = _processAndStoreKey(agentId, keyType, keyData, signature);
        agents[agentId].keyHashes.push(keyHash);
        agents[agentId].updatedAt = block.timestamp;

        agentNonce[agentId]++;

        emit KeyAdded(agentId, keyHash, keyType, block.timestamp);

        return keyHash;
    }

    /**
     * @notice Revoke a key from an agent
     */
    function revokeKey(bytes32 agentId, bytes32 keyHash)
        external
        onlyAgentOwner(agentId)
    {
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");
        require(agents[agentId].keyHashes.length > 1, "Cannot revoke last key");

        agentKeys[keyHash].verified = false;
        agents[agentId].updatedAt = block.timestamp;

        emit KeyRevoked(agentId, keyHash, block.timestamp);
    }

    /**
     * @notice Contract owner approves an Ed25519 key
     * @dev Required for Ed25519 keys since on-chain verification is not available
     */
    function approveEd25519Key(bytes32 keyHash) external onlyOwner {
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");
        require(agentKeys[keyHash].keyType == KeyType.Ed25519, "Not Ed25519 key");

        agentKeys[keyHash].verified = true;

        emit Ed25519KeyApproved(keyHash, block.timestamp);
    }

    /**
     * @notice Update agent metadata
     */
    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external onlyAgentOwner(agentId) {
        require(agents[agentId].active, "Agent not active");
        require(bytes(name).length > 0, "Name required");

        // Verify signature with first verified ECDSA key
        bytes32 firstEcdsaKey = _getFirstVerifiedEcdsaKey(agentId);
        require(firstEcdsaKey != bytes32(0), "No verified ECDSA key");

        bytes32 messageHash = keccak256(abi.encode(
            agentId,
            name,
            description,
            endpoint,
            capabilities,
            msg.sender,
            agentNonce[agentId]
        ));

        require(
            _verifyEcdsaSignature(messageHash, signature, agentKeys[firstEcdsaKey].keyData, msg.sender),
            "Invalid signature"
        );

        // Update metadata
        agents[agentId].name = name;
        agents[agentId].description = description;
        agents[agentId].endpoint = endpoint;
        agents[agentId].capabilities = capabilities;
        agents[agentId].updatedAt = block.timestamp;

        agentNonce[agentId]++;

        emit AgentUpdated(agentId, msg.sender, block.timestamp);
    }

    /**
     * @notice Deactivate an agent
     */
    function deactivateAgent(bytes32 agentId) external onlyAgentOwner(agentId) {
        require(agents[agentId].active, "Agent already inactive");

        agents[agentId].active = false;
        agents[agentId].updatedAt = block.timestamp;

        emit AgentDeactivated(agentId, msg.sender, block.timestamp);
    }

    /**
     * @notice Get agent metadata by ID
     */
    function getAgent(bytes32 agentId) external view returns (AgentMetadata memory) {
        require(agents[agentId].registeredAt > 0, "Agent not found");
        return agents[agentId];
    }

    /**
     * @notice Get agent metadata by DID
     */
    function getAgentByDID(string calldata did) external view returns (AgentMetadata memory) {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        return agents[agentId];
    }

    /**
     * @notice Get key details by key hash
     */
    function getKey(bytes32 keyHash) external view returns (AgentKey memory) {
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");
        return agentKeys[keyHash];
    }

    /**
     * @notice Get all keys for an agent
     */
    function getAgentKeys(bytes32 agentId) external view returns (bytes32[] memory) {
        require(agents[agentId].registeredAt > 0, "Agent not found");
        return agents[agentId].keyHashes;
    }

    /**
     * @notice Get all agent IDs owned by an address
     */
    function getAgentsByOwner(address ownerAddress) external view returns (bytes32[] memory) {
        return ownerToAgents[ownerAddress];
    }

    /**
     * @notice Verify agent ownership
     */
    function verifyAgentOwnership(bytes32 agentId, address claimedOwner)
        external
        view
        returns (bool)
    {
        return agents[agentId].owner == claimedOwner;
    }

    /**
     * @notice Check if agent is active
     */
    function isAgentActive(bytes32 agentId) external view returns (bool) {
        return agents[agentId].active;
    }

    /**
     * @notice Set before register hook
     */
    function setBeforeRegisterHook(address hook) external onlyOwner {
        beforeRegisterHook = hook;
    }

    /**
     * @notice Set after register hook
     */
    function setAfterRegisterHook(address hook) external onlyOwner {
        afterRegisterHook = hook;
    }

    // ============ Internal Functions ============

    /**
     * @notice Internal function to validate registration inputs
     */
    function _validateRegistrationInputs(
        string memory did,
        string memory name
    ) private view {
        require(bytes(did).length > 0, "DID required");
        require(bytes(name).length > 0, "Name required");
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        require(ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER, "Too many agents");
    }

    /**
     * @notice Internal function to generate agent ID
     */
    function _generateAgentId(
        string memory did,
        bytes memory firstKey
    ) private pure returns (bytes32) {
        return keccak256(abi.encode(did, firstKey));
    }

    /**
     * @notice Internal function to process and store a key
     */
    function _processAndStoreKey(
        bytes32 agentId,
        KeyType keyType,
        bytes memory keyData,
        bytes memory signature
    ) private returns (bytes32) {
        // Validate key length
        _validateKeyLength(keyType, keyData);

        // Generate key hash
        bytes32 keyHash = keccak256(abi.encode(agentId, keyType, keyData));
        require(agentKeys[keyHash].registeredAt == 0, "Key already registered");

        // Determine verification status based on key type
        bool verified = false;
        if (keyType == KeyType.ECDSA) {
            // Verify ECDSA signature on-chain
            bytes32 messageHash = keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce[agentId]));
            verified = _verifyEcdsaSignature(messageHash, signature, keyData, msg.sender);
            require(verified, "ECDSA signature verification failed");
        } else if (keyType == KeyType.X25519) {
            // X25519 keys don't need verification (public key exchange)
            verified = true;
        }
        // Ed25519 keys remain unverified until owner approves

        // Store key
        agentKeys[keyHash] = AgentKey({
            keyType: keyType,
            keyData: keyData,
            signature: signature,
            verified: verified,
            registeredAt: block.timestamp
        });

        return keyHash;
    }

    /**
     * @notice Validate key length based on type
     */
    function _validateKeyLength(KeyType keyType, bytes memory keyData) private pure {
        if (keyType == KeyType.Ed25519) {
            require(keyData.length == ED25519_KEY_LENGTH, "Invalid Ed25519 key length");
        } else if (keyType == KeyType.ECDSA) {
            require(
                keyData.length == SECP256K1_RAW_LENGTH ||
                keyData.length == SECP256K1_UNCOMPRESSED_LENGTH ||
                keyData.length == SECP256K1_COMPRESSED_LENGTH,
                "Invalid ECDSA key length"
            );
        } else if (keyType == KeyType.X25519) {
            require(keyData.length == X25519_KEY_LENGTH, "Invalid X25519 key length");
        }
    }

    /**
     * @notice Verify ECDSA signature
     */
    function _verifyEcdsaSignature(
        bytes32 messageHash,
        bytes memory signature,
        bytes memory /* publicKey */,
        address expectedSigner
    ) private pure returns (bool) {
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );

        address recovered = _recoverSigner(ethSignedHash, signature);
        return recovered == expectedSigner;
    }

    /**
     * @notice Recover signer from signature
     */
    function _recoverSigner(bytes32 messageHash, bytes memory signature)
        private
        pure
        returns (address)
    {
        require(signature.length == ECDSA_SIGNATURE_LENGTH, "Invalid signature length");

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

    /**
     * @notice Get first verified ECDSA key for an agent
     */
    function _getFirstVerifiedEcdsaKey(bytes32 agentId) private view returns (bytes32) {
        bytes32[] memory keyHashes = agents[agentId].keyHashes;
        for (uint256 i = 0; i < keyHashes.length; i++) {
            AgentKey memory key = agentKeys[keyHashes[i]];
            if (key.keyType == KeyType.ECDSA && key.verified) {
                return keyHashes[i];
            }
        }
        return bytes32(0);
    }

    /**
     * @notice Internal function to store agent metadata
     */
    function _storeAgentMetadata(
        bytes32 agentId,
        string memory did,
        string memory name,
        string memory description,
        string memory endpoint,
        string memory capabilities,
        bytes32[] memory keyHashes
    ) private {
        agents[agentId] = AgentMetadata({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            keyHashes: keyHashes,
            capabilities: capabilities,
            owner: msg.sender,
            registeredAt: block.timestamp,
            updatedAt: block.timestamp,
            active: true
        });

        didToAgentId[did] = agentId;
        ownerToAgents[msg.sender].push(agentId);
        agentNonce[agentId]++;

        emit AgentRegistered(agentId, msg.sender, did, block.timestamp);
    }

    /**
     * @notice Internal function to execute before register hook
     */
    function _executeBeforeHook(
        bytes32 agentId,
        string memory did,
        bytes[] memory keyData
    ) private {
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, keyData);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);

            (bool success, string memory reason) = IRegistryHook(beforeRegisterHook)
                .beforeRegister(agentId, msg.sender, hookData);
            require(success, reason);
        }
    }

    /**
     * @notice Internal function to execute after register hook
     */
    function _executeAfterHook(
        bytes32 agentId,
        string memory did,
        bytes[] memory keyData
    ) private {
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, keyData);
            IRegistryHook(afterRegisterHook).afterRegister(agentId, msg.sender, hookData);
            emit AfterRegisterHook(agentId, msg.sender, hookData);
        }
    }
}
