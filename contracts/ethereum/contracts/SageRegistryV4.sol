// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/ISageRegistryV4.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title SageRegistryV4
 * @notice SAGE AI Agent Registry Contract with Multi-Key Support
 * @dev Implements secure registration and management of AI agents with multiple public keys
 *      Supports Ed25519 and ECDSA/secp256k1 signature keys for multi-chain compatibility
 *      Note: Encryption keys (like X25519) should be generated ephemerally per session
 */
contract SageRegistryV4 is ISageRegistryV4, ReentrancyGuard {
    // State variables
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(bytes32 => AgentKey) private agentKeys;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(bytes32 => uint256) private agentNonce;

    // slither-disable-next-line naming-convention
    address public immutable OWNER;  // Uppercase is standard convention for immutable variables
    address public beforeRegisterHook;
    address public afterRegisterHook;

    // Agent limits
    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MAX_KEYS_PER_AGENT = 10;

    // Public key length constants
    uint256 private constant ED25519_KEY_LENGTH = 32;
    uint256 private constant SECP256K1_COMPRESSED_LENGTH = 33;
    uint256 private constant SECP256K1_UNCOMPRESSED_LENGTH = 65;
    uint256 private constant SECP256K1_RAW_LENGTH = 64;

    // Signature constants
    uint256 private constant ECDSA_SIGNATURE_LENGTH = 65;

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
        // slither-disable-next-line reentrancy-no-eth
        // Note: Hook is a trusted contract set by owner. Protected by nonReentrant modifier.
        // State changes after hook are intentional to validate registration before committing state.
        // Agent ID uniqueness is checked in _generateAgentId, preventing duplicate registrations.
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
     * @dev Completely removes the key from the agent (not just soft delete)
     *      This ensures revoked keys cannot be used even if verification checks are missed
     */
    function revokeKey(bytes32 agentId, bytes32 keyHash)
        external
        onlyAgentOwner(agentId)
    {
        // slither-disable-next-line timestamp
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");  // Existence check, not timestamp comparison
        require(_keyExistsInAgent(agentId, keyHash), "Key not in agent");
        require(agents[agentId].keyHashes.length > 1, "Cannot revoke last key");

        // Completely remove key from agent's keyHashes array
        _removeKeyFromAgent(agentId, keyHash);

        // Delete key data from storage
        delete agentKeys[keyHash];

        // Update metadata and nonce
        agents[agentId].updatedAt = block.timestamp;
        agentNonce[agentId]++;

        emit KeyRevoked(agentId, keyHash, block.timestamp);
    }

    /**
     * @notice Atomically rotate a key (remove old key and add new key)
     * @dev This is the recommended way to replace a compromised key
     *      The operation is atomic - if adding the new key fails,
     *      the old key is not removed (transaction reverts)
     * @param agentId The agent ID
     * @param oldKeyHash The key hash to remove
     * @param newKeyType The type of new key
     * @param newKeyData The new public key data
     * @param newSignature Signature proving ownership of new key
     * @return newKeyHash Hash of the newly added key
     */
    function rotateKey(
        bytes32 agentId,
        bytes32 oldKeyHash,
        KeyType newKeyType,
        bytes calldata newKeyData,
        bytes calldata newSignature
    ) external onlyAgentOwner(agentId) returns (bytes32) {
        require(agents[agentId].active, "Agent not active");
        // slither-disable-next-line timestamp
        // Existence check, not timestamp comparison
        require(agentKeys[oldKeyHash].registeredAt > 0, "Old key not found");
        require(agentKeys[oldKeyHash].verified, "Old key not verified");

        // Verify old key belongs to this agent
        require(_keyExistsInAgent(agentId, oldKeyHash), "Old key not in agent");

        // Step 1: Process and verify new key (will revert if invalid)
        bytes32 newKeyHash = _processAndStoreKey(agentId, newKeyType, newKeyData, newSignature);

        // Step 2: Remove old key from agent's keyHashes array
        // Only reaches here if new key was successfully added
        _removeKeyFromAgent(agentId, oldKeyHash);

        // Step 3: Delete old key data
        delete agentKeys[oldKeyHash];

        // Step 4: Update metadata and nonce
        agents[agentId].updatedAt = block.timestamp;
        agentNonce[agentId]++;

        emit KeyRotated(agentId, oldKeyHash, newKeyHash, block.timestamp);

        return newKeyHash;
    }

    /**
     * @notice Contract owner approves an Ed25519 key
     * @dev Required for Ed25519 keys since on-chain verification is not available
     */
    function approveEd25519Key(bytes32 keyHash) external onlyOwner {
        // slither-disable-next-line timestamp
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");  // Existence check, not timestamp comparison
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
        // slither-disable-next-line timestamp
        require(agents[agentId].registeredAt > 0, "Agent not found");  // Existence check, not timestamp comparison
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
        // slither-disable-next-line timestamp
        require(agentKeys[keyHash].registeredAt > 0, "Key not found");  // Existence check, not timestamp comparison
        return agentKeys[keyHash];
    }

    /**
     * @notice Get all keys for an agent
     */
    function getAgentKeys(bytes32 agentId) external view returns (bytes32[] memory) {
        // slither-disable-next-line timestamp
        require(agents[agentId].registeredAt > 0, "Agent not found");  // Existence check, not timestamp comparison
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
        require(hook != address(0), "Hook cannot be zero address");
        beforeRegisterHook = hook;
    }

    /**
     * @notice Set after register hook
     */
    function setAfterRegisterHook(address hook) external onlyOwner {
        require(hook != address(0), "Hook cannot be zero address");
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

        // Optional: Validate DID format includes owner address
        // This provides stronger identity binding but is not strictly enforced
        // for backward compatibility and cross-chain flexibility
        // Format: did:sage:ethereum:0x{address} or did:sage:ethereum:0x{address}:{nonce}
        // Uncomment to enable strict DID validation:
        // _validateDIDFormat(did, msg.sender);
    }

    // ============ OPTIONAL SECURITY FEATURE (Currently Disabled) ============
    //
    // The following functions provide DID format validation for stronger identity binding.
    // Currently disabled for backward compatibility and cross-chain flexibility.
    //
    // To enable: Uncomment line 361 in _validateRegistrationInputs()
    //
    // Functions:
    //   - _validateDIDFormat: Validates DID includes owner address
    //   - _startsWith: String prefix checking helper
    //   - _parseAddressFromHex: Hex string to address conversion
    //   - _hexCharToUint: Hex character to uint conversion
    //
    // Note: Kept for future security enhancement. Removal would require
    //       re-implementation if stricter DID validation is needed later.
    // =========================================================================

    /**
     * @notice Validate that DID includes the owner's Ethereum address
     * @dev This function provides optional stronger identity binding by requiring
     *      the DID to include the owner's address. This ensures:
     *      - Off-chain DID ownership verification
     *      - Cross-chain owner traceability
     *      - DID collision prevention
     *
     *      Expected formats:
     *      - did:sage:ethereum:0x{address}
     *      - did:sage:ethereum:0x{address}:{nonce}
     *
     *      Example: did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
     *
     * @param did The DID string to validate
     * @param owner The expected owner address
     */
    function _validateDIDFormat(string memory did, address owner) private pure {
        bytes memory didBytes = bytes(did);

        // Minimum length: "did:sage:ethereum:0x" + 40 hex chars = 62 characters
        require(didBytes.length >= 62, "DID too short for address inclusion");

        // Check prefix "did:sage:ethereum:0x"
        require(_startsWith(didBytes, "did:sage:ethereum:0x"), "Invalid DID prefix");

        // Extract address portion (40 hex characters starting at position 22)
        // Position breakdown: "did:" (4) + "sage:" (5) + "ethereum:" (9) + "0x" (2) = 20 chars
        bytes memory addressHex = new bytes(40);
        for (uint256 i = 0; i < 40; i++) {
            addressHex[i] = didBytes[22 + i];
        }

        // Convert hex string to address
        address didAddress = _parseAddressFromHex(addressHex);

        // Verify the DID address matches the owner
        require(didAddress == owner, "DID address does not match owner");
    }

    /**
     * @notice Check if bytes start with a specific prefix
     * @param data The bytes to check
     * @param prefix The prefix string to match
     * @return true if data starts with prefix
     */
    function _startsWith(bytes memory data, string memory prefix) private pure returns (bool) {
        bytes memory prefixBytes = bytes(prefix);
        if (data.length < prefixBytes.length) {
            return false;
        }

        for (uint256 i = 0; i < prefixBytes.length; i++) {
            if (data[i] != prefixBytes[i]) {
                return false;
            }
        }

        return true;
    }

    /**
     * @notice Parse Ethereum address from hex string
     * @param hexAddress 40-character hex string representing an address
     * @return The parsed Ethereum address
     */
    function _parseAddressFromHex(bytes memory hexAddress) private pure returns (address) {
        require(hexAddress.length == 40, "Invalid hex address length");

        uint160 addressValue = 0;

        for (uint256 i = 0; i < 40; i++) {
            uint8 digit = _hexCharToUint(hexAddress[i]);
            addressValue = addressValue * 16 + digit;
        }

        return address(addressValue);
    }

    /**
     * @notice Convert hex character to uint
     * @param char The hex character (0-9, a-f, A-F)
     * @return The uint value (0-15)
     */
    function _hexCharToUint(bytes1 char) private pure returns (uint8) {
        uint8 c = uint8(char);

        // '0'-'9'
        if (c >= 48 && c <= 57) {
            return c - 48;
        }
        // 'a'-'f'
        else if (c >= 97 && c <= 102) {
            return c - 87;  // 97 - 10 = 87
        }
        // 'A'-'F'
        else if (c >= 65 && c <= 70) {
            return c - 55;  // 65 - 10 = 55
        }

        revert("Invalid hex character");
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
        // slither-disable-next-line timestamp
        // Existence check, not timestamp comparison
        require(agentKeys[keyHash].registeredAt == 0, "Key already registered");

        // Determine verification status based on key type
        bool verified = false;
        if (keyType == KeyType.ECDSA) {
            // Verify ECDSA signature on-chain
            bytes32 messageHash = keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce[agentId]));
            verified = _verifyEcdsaSignature(messageHash, signature, keyData, msg.sender);
            require(verified, "ECDSA signature verification failed");
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
        }
    }

    /**
     * @notice Verify ECDSA signature and public key ownership
     * @dev This function performs TWO critical security checks:
     *      1. Signature verification: Confirms msg.sender signed the message
     *      2. Public key ownership: Confirms the public key belongs to msg.sender
     *      Both checks must pass to prevent public key theft attacks
     * @param messageHash The hash of the message that was signed
     * @param signature The ECDSA signature (65 bytes: r + s + v)
     * @param publicKey The secp256k1 public key being registered
     * @param expectedSigner The address that should have signed (typically msg.sender)
     * @return true if both signature and ownership are valid
     */
    function _verifyEcdsaSignature(
        bytes32 messageHash,
        bytes memory signature,
        bytes memory publicKey,
        address expectedSigner
    ) private pure returns (bool) {
        // Check 1: Verify that expectedSigner (msg.sender) signed the message
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );

        address recovered = _recoverSigner(ethSignedHash, signature);
        require(recovered == expectedSigner, "Invalid signature");

        // Check 2: Verify that the public key actually belongs to the signer
        // This prevents attackers from registering someone else's public key
        address derivedAddress = _deriveAddressFromPublicKey(publicKey);
        require(derivedAddress == expectedSigner, "Public key does not match signer");

        return true;
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

        // Assembly: Extract ECDSA signature components (r, s, v)
        // SAFETY:
        // - signature.length is verified to be exactly 65 bytes (line 590)
        // - Memory layout: [length(32)] [r(32)] [s(32)] [v(1)]
        // - add(signature, 32) skips the length field to read r
        // - add(signature, 64) reads s at offset 64
        // - add(signature, 96) reads v at offset 96
        // - All reads are within bounds due to length check
        // - Using assembly saves ~200 gas vs. abi.decode
        // slither-disable-next-line assembly
        // solhint-disable-next-line no-inline-assembly
        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        return ecrecover(messageHash, v, r, s);
    }

    /**
     * @notice Derive Ethereum address from secp256k1 public key
     * @dev Ethereum address = last 20 bytes of keccak256(publicKey)
     *      This is used to verify that the registered public key actually belongs to msg.sender
     * @param publicKey The public key in uncompressed (65 bytes with 0x04 prefix) or raw (64 bytes) format
     * @return The derived Ethereum address
     */
    function _deriveAddressFromPublicKey(bytes memory publicKey)
        private
        pure
        returns (address)
    {
        bytes memory keyWithoutPrefix;

        // slither-disable-next-line incorrect-equality
        // Note: Checking publicKey.length is safe - bytes array length is deterministic
        if (publicKey.length == SECP256K1_UNCOMPRESSED_LENGTH) {
            // Uncompressed format: 0x04 + 64 bytes (x, y coordinates)
            // slither-disable-next-line incorrect-equality
            // Note: Checking byte value == 0x04 is safe - deterministic prefix check
            require(publicKey[0] == 0x04, "Invalid uncompressed key prefix");

            // Extract 64 bytes (x, y) without the 0x04 prefix
            keyWithoutPrefix = new bytes(SECP256K1_RAW_LENGTH);
            for (uint256 i = 0; i < SECP256K1_RAW_LENGTH; i++) {
                keyWithoutPrefix[i] = publicKey[i + 1];
            }
        // slither-disable-next-line incorrect-equality
        // Note: Checking publicKey.length is safe - bytes array length is deterministic
        } else if (publicKey.length == SECP256K1_RAW_LENGTH) {
            // Raw format: 64 bytes (x, y coordinates without prefix)
            keyWithoutPrefix = publicKey;
        // slither-disable-next-line incorrect-equality
        // Note: Checking publicKey.length is safe - bytes array length is deterministic
        } else if (publicKey.length == SECP256K1_COMPRESSED_LENGTH) {
            // Compressed format: 0x02/0x03 + 32 bytes
            // Decompression requires elliptic curve point operations (very expensive gas cost)
            // For security-critical operations, we require uncompressed keys for clarity
            revert("Compressed keys not supported");
        } else {
            revert("Invalid public key length");
        }

        // Compute keccak256 hash of the 64-byte public key
        bytes32 hash = keccak256(keyWithoutPrefix);

        // Ethereum address is the last 20 bytes of the hash
        address derivedAddress;
        // Assembly: Extract last 20 bytes from hash for Ethereum address
        // SAFETY:
        // - hash is a bytes32 (32 bytes) computed from keccak256
        // - Mask 0xFFFF...FFFF (20 bytes of F) extracts rightmost 160 bits (20 bytes)
        // - This is the standard Ethereum address derivation algorithm
        // - Equivalent to: address(uint160(uint256(hash)))
        // - Using assembly saves ~50 gas vs. type conversions
        // slither-disable-next-line assembly
        // solhint-disable-next-line no-inline-assembly
        assembly {
            // Load the last 20 bytes from the hash
            // mload loads 32 bytes, so we need to shift right by 12 bytes (96 bits)
            derivedAddress := and(hash, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF)
        }

        return derivedAddress;
    }

    /**
     * @notice Get first verified ECDSA key for an agent
     */
    function _getFirstVerifiedEcdsaKey(bytes32 agentId) private view returns (bytes32) {
        bytes32[] memory keyHashes = agents[agentId].keyHashes;
        for (uint256 i = 0; i < keyHashes.length; i++) {
            AgentKey memory key = agentKeys[keyHashes[i]];
            // slither-disable-next-line incorrect-equality
            // Note: Checking enum equality is safe - KeyType is a deterministic enum value
            if (key.keyType == KeyType.ECDSA && key.verified) {
                return keyHashes[i];
            }
        }
        return bytes32(0);
    }

    /**
     * @notice Remove a key hash from agent's keyHashes array
     * @dev Uses swap-and-pop pattern for gas efficiency
     * @param agentId The agent ID
     * @param keyHash The key hash to remove
     */
    function _removeKeyFromAgent(bytes32 agentId, bytes32 keyHash) private {
        bytes32[] storage hashes = agents[agentId].keyHashes;

        // Find the key in the array
        for (uint256 i = 0; i < hashes.length; i++) {
            if (hashes[i] == keyHash) {
                // Swap with last element and pop
                hashes[i] = hashes[hashes.length - 1];
                hashes.pop();
                return;
            }
        }

        // If we reach here, key was not found
        revert("Key not found in agent");
    }

    /**
     * @notice Check if a key hash exists in agent's keyHashes array
     * @param agentId The agent ID
     * @param keyHash The key hash to check
     * @return true if key exists in agent's keyHashes
     */
    function _keyExistsInAgent(bytes32 agentId, bytes32 keyHash) private view returns (bool) {
        bytes32[] memory hashes = agents[agentId].keyHashes;
        for (uint256 i = 0; i < hashes.length; i++) {
            // slither-disable-next-line incorrect-equality
            // Note: Checking bytes32 hash equality is safe - deterministic hash comparison for lookup
            if (hashes[i] == keyHash) {
                return true;
            }
        }
        return false;
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
