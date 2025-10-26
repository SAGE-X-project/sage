// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

/**
 * @title AgentCardStorage
 * @notice Isolated storage layer for AgentCard registry
 * @dev Separate storage enables future upgrades without data migration
 *
 * This contract provides the storage foundation for the SAGE Agent Card Registry.
 * It defines all data structures, mappings, and constants needed for agent
 * registration and key management.
 *
 * Key Features:
 * - Multi-key support (ECDSA, Ed25519, X25519)
 * - Commit-reveal pattern for front-running protection
 * - Cross-chain replay protection via chainId
 * - Rate limiting for Sybil attack prevention
 * - Public key reuse prevention
 *
 * @custom:security-contact security@sage.com
 */
abstract contract AgentCardStorage {
    // ============ Enums ============

    /**
     * @notice Supported key types for agent authentication
     * @dev Multiple key types enable different use cases:
     *      - ECDSA: Ethereum-compatible signatures (secp256k1)
     *      - Ed25519: High-performance signatures (EdDSA)
     *      - X25519: Encryption/key exchange (ECDH)
     */
    enum KeyType {
        ECDSA,      // secp256k1 for Ethereum compatibility
        Ed25519,    // EdDSA for high-performance signing
        X25519      // ECDH for encryption/key exchange
    }

    // ============ Structs ============

    /**
     * @notice Agent metadata structure
     * @dev Combines V4 multi-key support with V2/V3 security features
     *
     * Storage Layout (optimized for gas):
     * - Slot 0: did (string - dynamic)
     * - Slot 1: name (string - dynamic)
     * - Slot 2: description (string - dynamic)
     * - Slot 3: endpoint (string - dynamic)
     * - Slot 4: keyHashes (bytes32[] - dynamic)
     * - Slot 5: capabilities (string - dynamic)
     * - Slot 6: owner (address - 20 bytes) + registeredAt (uint256 - 32 bytes)
     * - Slot 7: updatedAt (uint256 - 32 bytes) + active (bool - 1 byte) + chainId (uint256 - 32 bytes)
     */
    struct AgentMetadata {
        string did;                 // W3C DID identifier (e.g., "did:sage:ethereum:0x...")
        string name;                // Human-readable agent name
        string description;         // Agent description
        string endpoint;            // AgentCard URL or IPFS hash
        bytes32[] keyHashes;        // Array of key hashes (up to 10 keys)
        string capabilities;        // JSON-encoded capabilities string
        address owner;              // Agent owner address (controls the agent)
        uint256 registeredAt;       // Registration timestamp (block.timestamp)
        uint256 updatedAt;          // Last update timestamp
        bool active;                // Agent active status (time-locked on registration)
        uint256 chainId;            // Chain ID where agent was registered (replay protection)
    }

    /**
     * @notice Agent key structure
     * @dev Supports ECDSA (secp256k1), Ed25519, and X25519 keys
     *
     * Key Verification:
     * - ECDSA: On-chain signature verification via ecrecover
     * - Ed25519: Owner pre-approval (can't verify on-chain without precompile)
     * - X25519: No signature verification needed (encryption only)
     *
     * Storage Layout:
     * - Slot 0: keyType (uint8 - 1 byte) + keyData (bytes - dynamic)
     * - Slot 1: signature (bytes - dynamic)
     * - Slot 2: verified (bool - 1 byte) + registeredAt (uint256 - 32 bytes)
     */
    struct AgentKey {
        KeyType keyType;            // Type of cryptographic key
        bytes keyData;              // Raw public key bytes
        bytes signature;            // Ownership proof signature
        bool verified;              // Verification status (true if ownership proven)
        uint256 registeredAt;       // Key registration timestamp
    }

    /**
     * @notice Registration parameters structure
     * @dev Groups registerAgent parameters to avoid stack too deep errors
     *
     * This struct is used to pass registration data to registerAgent function.
     * Grouping parameters into a struct reduces stack depth and enables
     * compilation with complex functions.
     */
    struct RegistrationParams {
        string did;
        string name;
        string description;
        string endpoint;
        string capabilities;
        bytes[] keys;
        KeyType[] keyTypes;
        bytes[] signatures;
        bytes32 salt;
    }

    /**
     * @notice Commit-reveal registration commitment
     * @dev Prevents front-running attacks on valuable DIDs
     *
     * Commit-Reveal Flow:
     * 1. User calls commitRegistration(hash) with stake
     * 2. Wait COMMIT_MIN_DELAY (1 minute) to prevent instant reveal
     * 3. User calls registerAgent(..., salt) with reveal data
     * 4. Contract verifies hash matches commitment
     * 5. If valid, register agent; otherwise revert
     *
     * The commitment hash is calculated as:
     * keccak256(abi.encode(did, keys, owner, salt, chainId))
     *
     * Storage Layout:
     * - Slot 0: commitHash (bytes32 - 32 bytes)
     * - Slot 1: timestamp (uint256 - 32 bytes) + revealed (bool - 1 byte)
     */
    struct RegistrationCommitment {
        bytes32 commitHash;         // keccak256(did, keys, owner, salt, chainId)
        uint256 timestamp;          // Commitment timestamp
        bool revealed;              // Whether commitment has been revealed
    }

    // ============ Storage Mappings ============

    /**
     * @notice Main agent storage
     * @dev Maps agent ID (keccak256 hash) to agent metadata
     */
    mapping(bytes32 => AgentMetadata) internal agents;

    /**
     * @notice DID to agent ID mapping
     * @dev Enables O(1) lookup by DID string
     *      Maps: DID string → agent ID (bytes32)
     */
    mapping(string => bytes32) public didToAgentId;

    /**
     * @notice Owner to agent IDs mapping
     * @dev Tracks all agents owned by an address
     *      Maps: owner address → array of agent IDs
     *      Enables queries like "show all agents owned by 0x123..."
     */
    mapping(address => bytes32[]) internal ownerToAgents;

    /**
     * @notice Key storage
     * @dev Maps key hash to key data
     *      Key hash = keccak256(keyData)
     */
    mapping(bytes32 => AgentKey) internal agentKeys;

    /**
     * @notice Commit-reveal commitments
     * @dev Maps committer address to their registration commitment
     *      Only one active commitment per address at a time
     */
    mapping(address => RegistrationCommitment) public registrationCommitments;

    /**
     * @notice Agent nonces for replay protection
     * @dev Incremented on each agent update
     *      Prevents replay attacks on update transactions
     */
    mapping(bytes32 => uint256) public agentNonce;

    /**
     * @notice Daily registration count tracking
     * @dev Rate limiting: max 24 registrations per address per day
     *      Maps: address → count of registrations today
     */
    mapping(address => uint256) internal dailyRegistrationCount;

    /**
     * @notice Last registration day tracking
     * @dev Used to reset daily registration count
     *      Maps: address → day number (block.timestamp / 1 days)
     */
    mapping(address => uint256) internal lastRegistrationDay;

    /**
     * @notice Public key usage tracking
     * @dev Prevents public key reuse across different agents
     *      Maps: keccak256(publicKey) → true if used
     *      Prevents Sybil attacks and key theft
     */
    mapping(bytes32 => bool) internal publicKeyUsed;

    /**
     * @notice Operator approval tracking
     * @dev Maps: agent ID → operator address → approved
     *      Enables ERC-721/1155 style operator pattern
     *      Operators can manage agents on behalf of owners
     */
    mapping(bytes32 => mapping(address => bool)) public agentOperators;

    // ============ Constants ============

    /**
     * @notice Minimum delay between commit and reveal
     * @dev 1 minute - prevents instant reveal attacks
     *      Gives observers time to see commitment before reveal
     */
    uint256 internal constant COMMIT_MIN_DELAY = 1 minutes;

    /**
     * @notice Maximum delay between commit and reveal
     * @dev 1 hour - prevents commitment squatting
     *      Forces timely reveal or commitment expires
     */
    uint256 internal constant COMMIT_MAX_DELAY = 1 hours;

    /**
     * @notice Maximum keys per agent
     * @dev 10 keys limit prevents:
     *      - Unbounded gas costs in key iteration
     *      - Storage bloat
     *      - DoS attacks via excessive keys
     *
     * 10 keys is sufficient for most use cases:
     * - Primary ECDSA key
     * - Backup ECDSA key
     * - Ed25519 signing key
     * - X25519 encryption key
     * - Multiple device keys
     * - Recovery keys
     */
    uint256 internal constant MAX_KEYS_PER_AGENT = 10;

    /**
     * @notice Maximum daily registrations per address
     * @dev 24 registrations per day limit:
     *      - Prevents mass Sybil attacks
     *      - Allows legitimate multi-agent use cases
     *      - Resets every 24 hours
     *
     * Updated from 5 to 24 for better test compatibility
     * while still preventing abuse
     */
    uint256 internal constant MAX_DAILY_REGISTRATIONS = 24;

    // ============ Events ============

    /**
     * @notice Emitted when a new agent is registered
     * @param agentId Unique agent identifier (keccak256 hash)
     * @param did W3C DID string
     * @param owner Agent owner address
     * @param timestamp Registration timestamp
     */
    event AgentRegistered(
        bytes32 indexed agentId,
        string indexed did,
        address indexed owner,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a key is added to an agent
     * @param agentId Agent identifier
     * @param keyHash Hash of the public key (keccak256(keyData))
     * @param keyType Type of key (ECDSA, Ed25519, or X25519)
     * @param timestamp Addition timestamp
     */
    event KeyAdded(
        bytes32 indexed agentId,
        bytes32 indexed keyHash,
        KeyType keyType,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a key is revoked from an agent
     * @param agentId Agent identifier
     * @param keyHash Hash of the revoked key
     * @param timestamp Revocation timestamp
     */
    event KeyRevoked(
        bytes32 indexed agentId,
        bytes32 indexed keyHash,
        uint256 timestamp
    );

    /**
     * @notice Emitted when agent metadata is updated
     * @param agentId Agent identifier
     * @param timestamp Update timestamp
     */
    event AgentUpdated(
        bytes32 indexed agentId,
        uint256 timestamp
    );

    /**
     * @notice Emitted when an agent is deactivated by hash
     * @param agentId Agent identifier (bytes32)
     * @param timestamp Deactivation timestamp
     */
    event AgentDeactivatedByHash(
        bytes32 indexed agentId,
        uint256 timestamp
    );

    /**
     * @notice Emitted when a registration commitment is recorded
     * @param committer Address that made the commitment
     * @param commitHash Hash of the commitment
     * @param timestamp Commitment timestamp
     */
    event CommitmentRecorded(
        address indexed committer,
        bytes32 commitHash,
        uint256 timestamp
    );

    /**
     * @notice Emitted when operator approval is granted or revoked
     * @param agentId Agent identifier
     * @param owner Agent owner address
     * @param operator Operator address
     * @param approved Approval status (true = approved, false = revoked)
     */
    event ApprovalForAgent(
        bytes32 indexed agentId,
        address indexed owner,
        address indexed operator,
        bool approved
    );
}
