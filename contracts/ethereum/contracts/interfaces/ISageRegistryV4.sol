// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

/**
 * @title ISageRegistryV4
 * @notice Interface for SAGE AI Agent Registry with Multi-Key Support
 */
interface ISageRegistryV4 {
    /**
     * @notice Supported key types for multi-chain compatibility
     * @dev Only signature keys are supported for DID registration
     *      Encryption keys (like X25519) should be generated ephemerally
     *      per session to ensure forward secrecy
     */
    enum KeyType {
        Ed25519,  // Solana, Cardano, Polkadot (32 bytes)
        ECDSA     // Ethereum, Bitcoin (33/65 bytes for secp256k1)
    }

    /**
     * @notice Agent key information
     * @param keyType Type of cryptographic key
     * @param keyData Raw public key bytes
     * @param signature Signature proving ownership during registration
     * @param verified Whether the key has been verified (on-chain or by owner)
     * @param registeredAt Timestamp when key was registered
     */
    struct AgentKey {
        KeyType keyType;
        bytes keyData;
        bytes signature;
        bool verified;
        uint256 registeredAt;
    }

    /**
     * @notice Agent metadata with multi-key support
     * @param did Decentralized Identifier
     * @param name Agent name
     * @param description Agent description
     * @param endpoint Service endpoint URL
     * @param keyHashes Array of key hashes (keccak256 of agentId + keyType + keyData)
     * @param capabilities JSON string with A2A Agent Card and capabilities
     * @param owner Ethereum address of agent owner
     * @param registeredAt Registration timestamp
     * @param updatedAt Last update timestamp
     * @param active Whether agent is active
     */
    struct AgentMetadata {
        string did;
        string name;
        string description;
        string endpoint;
        bytes32[] keyHashes;
        string capabilities;
        address owner;
        uint256 registeredAt;
        uint256 updatedAt;
        bool active;
    }

    /**
     * @notice Emitted when an agent is registered
     */
    event AgentRegistered(
        bytes32 indexed agentId,
        address indexed owner,
        string did,
        uint256 timestamp
    );

    /**
     * @notice Emitted when an agent is updated
     */
    event AgentUpdated(
        bytes32 indexed agentId,
        address indexed owner,
        uint256 timestamp
    );

    /**
     * @notice Emitted when an agent is deactivated
     */
    event AgentDeactivated(
        bytes32 indexed agentId,
        address indexed owner,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a key is added to an agent
     */
    event KeyAdded(
        bytes32 indexed agentId,
        bytes32 indexed keyHash,
        KeyType keyType,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a key is revoked
     */
    event KeyRevoked(
        bytes32 indexed agentId,
        bytes32 indexed keyHash,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a key is rotated (old key removed, new key added atomically)
     */
    event KeyRotated(
        bytes32 indexed agentId,
        bytes32 indexed oldKeyHash,
        bytes32 indexed newKeyHash,
        uint256 timestamp
    );

    /**
     * @notice Emitted when contract owner approves an Ed25519 key
     */
    event Ed25519KeyApproved(
        bytes32 indexed keyHash,
        uint256 timestamp
    );

    /**
     * @notice Emitted before registration hook is executed
     */
    event BeforeRegisterHook(
        bytes32 indexed agentId,
        address indexed caller,
        bytes hookData
    );

    /**
     * @notice Emitted after registration hook is executed
     */
    event AfterRegisterHook(
        bytes32 indexed agentId,
        address indexed caller,
        bytes hookData
    );

    /**
     * @notice Registration parameters struct to avoid stack too deep errors
     */
    struct RegistrationParams {
        string did;
        string name;
        string description;
        string endpoint;
        KeyType[] keyTypes;
        bytes[] keyData;
        bytes[] signatures;
        string capabilities;
    }

    /**
     * @notice Register a new AI agent with multiple keys
     * @param params Registration parameters
     * @return agentId Generated agent ID
     */
    function registerAgent(RegistrationParams calldata params) external returns (bytes32);

    /**
     * @notice Add a new key to an existing agent
     * @param agentId Agent identifier
     * @param keyType Type of key to add
     * @param keyData Public key data
     * @param signature Signature proving key ownership
     * @return keyHash Hash of the added key
     */
    function addKey(
        bytes32 agentId,
        KeyType keyType,
        bytes calldata keyData,
        bytes calldata signature
    ) external returns (bytes32);

    /**
     * @notice Revoke a key from an agent
     * @param agentId Agent identifier
     * @param keyHash Hash of key to revoke
     */
    function revokeKey(bytes32 agentId, bytes32 keyHash) external;

    /**
     * @notice Atomically rotate a key (remove old key and add new key)
     * @dev This is the recommended way to replace a compromised key
     *      The operation is atomic - if the new key fails verification,
     *      the old key is not removed
     * @param agentId Agent identifier
     * @param oldKeyHash Hash of key to remove
     * @param newKeyType Type of new key
     * @param newKeyData New public key data
     * @param newSignature Signature proving ownership of new key
     * @return newKeyHash Hash of the new key
     */
    function rotateKey(
        bytes32 agentId,
        bytes32 oldKeyHash,
        KeyType newKeyType,
        bytes calldata newKeyData,
        bytes calldata newSignature
    ) external returns (bytes32);

    /**
     * @notice Contract owner approves an Ed25519 key
     * @param keyHash Hash of Ed25519 key to approve
     */
    function approveEd25519Key(bytes32 keyHash) external;

    /**
     * @notice Update agent metadata
     * @param agentId Agent identifier
     * @param name New name
     * @param description New description
     * @param endpoint New endpoint
     * @param capabilities New capabilities
     * @param signature Signature from verified ECDSA key
     */
    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external;

    /**
     * @notice Deactivate an agent
     * @param agentId Agent identifier
     */
    function deactivateAgent(bytes32 agentId) external;

    /**
     * @notice Get agent metadata by ID
     * @param agentId Agent identifier
     * @return AgentMetadata struct
     */
    function getAgent(bytes32 agentId) external view returns (AgentMetadata memory);

    /**
     * @notice Get agent metadata by DID
     * @param did Decentralized Identifier
     * @return AgentMetadata struct
     */
    function getAgentByDID(string calldata did) external view returns (AgentMetadata memory);

    /**
     * @notice Get key details by key hash
     * @param keyHash Key identifier
     * @return AgentKey struct
     */
    function getKey(bytes32 keyHash) external view returns (AgentKey memory);

    /**
     * @notice Get all keys for an agent
     * @param agentId Agent identifier
     * @return Array of key hashes
     */
    function getAgentKeys(bytes32 agentId) external view returns (bytes32[] memory);

    /**
     * @notice Get all agent IDs owned by an address
     * @param ownerAddress Owner's Ethereum address
     * @return Array of agent IDs
     */
    function getAgentsByOwner(address ownerAddress) external view returns (bytes32[] memory);

    /**
     * @notice Get current nonce for an agent
     * @dev Nonce is used for replay protection in signed operations
     * @param agentId Agent identifier
     * @return Current nonce value
     */
    function getNonce(bytes32 agentId) external view returns (uint256);

    /**
     * @notice Verify agent ownership
     * @param agentId Agent identifier
     * @param claimedOwner Address claiming ownership
     * @return True if claimedOwner owns the agent
     */
    function verifyAgentOwnership(bytes32 agentId, address claimedOwner) external view returns (bool);

    /**
     * @notice Check if agent is active
     * @param agentId Agent identifier
     * @return True if agent is active
     */
    function isAgentActive(bytes32 agentId) external view returns (bool);

    /**
     * @notice Set before register hook address
     * @param hook Hook contract address
     */
    function setBeforeRegisterHook(address hook) external;

    /**
     * @notice Set after register hook address
     * @param hook Hook contract address
     */
    function setAfterRegisterHook(address hook) external;
}
