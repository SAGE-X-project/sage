// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./interfaces/ISageRegistry.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title SageRegistryV2
 * @notice Improved SAGE AI Agent Registry with Enhanced Public Key Validation
 * @dev Implements practical public key validation using signature-based ownership proof
 */
contract SageRegistryV2 is ISageRegistry, Ownable2Step {
    // Registration parameters struct to avoid stack too deep errors
    struct RegistrationParams {
        string did;
        string name;
        string description;
        string endpoint;
        bytes publicKey;
        string capabilities;
        bytes signature;
    }
    
    // Enhanced validation data
    struct KeyValidation {
        bytes32 keyHash;           // Hash of the public key
        uint256 registrationBlock;  // Block when key was registered
        bool isRevoked;            // Key revocation status
    }
    
    // State variables
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(address => uint256) private registrationNonce; // User-specific nonce for agent ID generation
    mapping(bytes32 => uint256) private agentNonce; // Agent-specific nonce for updates
    
    // Enhanced key validation
    mapping(bytes32 => KeyValidation) private keyValidations;
    mapping(address => bytes32) private addressToKeyHash;
    mapping(bytes32 => bytes32[]) private keyHashToAgentIds; // keyHash => agentIds

    address public beforeRegisterHook;
    address public afterRegisterHook;
    
    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MIN_PUBLIC_KEY_LENGTH = 32;
    uint256 private constant MAX_PUBLIC_KEY_LENGTH = 65;
    uint256 private constant HOOK_GAS_LIMIT = 50000; // Gas limit for hook calls
    
    // Events
    event KeyValidated(bytes32 indexed keyHash, address indexed owner);
    event KeyRevoked(bytes32 indexed keyHash, address indexed owner);
    event HookFailed(address indexed hook, string reason);

    // Modifiers
    modifier onlyAgentOwner(bytes32 agentId) {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        _;
    }

    constructor() {
        _transferOwnership(msg.sender);
    }
    
    /**
     * @notice Register a new AI agent with enhanced key validation
     * @dev Requires proof of public key ownership through signature
     */
    function registerAgent(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature
    ) external returns (bytes32) {
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
    
    /**
     * @notice Enhanced public key validation
     * @dev Validates format, non-zero, and ownership through signature
     */
    function _validatePublicKey(
        bytes calldata publicKey,
        bytes calldata signature
    ) internal virtual {
        // 1. Length validation
        require(
            publicKey.length >= MIN_PUBLIC_KEY_LENGTH && 
            publicKey.length <= MAX_PUBLIC_KEY_LENGTH,
            "Invalid public key length"
        );
        
        // 2. Format validation for secp256k1
        if (publicKey.length == 65) {
            // Uncompressed format: must start with 0x04
            require(publicKey[0] == 0x04, "Invalid uncompressed key format");
        } else if (publicKey.length == 33) {
            // Compressed format: must start with 0x02 or 0x03
            require(
                publicKey[0] == 0x02 || publicKey[0] == 0x03, 
                "Invalid compressed key format"
            );
        } else if (publicKey.length == 32) {
            // Ed25519 keys are 32 bytes - not supported on-chain
            revert("Ed25519 not supported on-chain");
        }
        
        // 3. Non-zero validation (prevent obviously invalid keys)
        bytes32 keyHash = keccak256(publicKey);
        // Check if the key is not all zeros (skip prefix byte for format check)
        bool isNonZero = false;
        uint startIdx = 0;
        if (publicKey.length == 65 && publicKey[0] == 0x04) {
            startIdx = 1; // Skip 0x04 prefix for uncompressed keys
        } else if (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) {
            startIdx = 1; // Skip 0x02/0x03 prefix for compressed keys
        }
        
        for (uint i = startIdx; i < publicKey.length; i++) {
            if (publicKey[i] != 0) {
                isNonZero = true;
                break;
            }
        }
        require(isNonZero, "Invalid zero key");
        
        // 4. Ownership proof through signature
        // Create a challenge message that includes context (without timestamp for predictability)
        bytes32 challenge = keccak256(abi.encodePacked(
            "SAGE Key Registration:",
            block.chainid,
            address(this),
            msg.sender,
            keyHash
        ));
        
        // Verify signature (this proves control of the private key)
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", challenge)
        );
        
        address recovered = _recoverSigner(ethSignedHash, signature);
        
        // Derive address from the public key and verify it matches the signer
        address keyAddress = _getAddressFromPublicKey(publicKey);
        require(recovered == keyAddress, "Key ownership not proven");
        require(recovered != address(0), "Invalid signature");
        
        // 5. Check if key has been revoked before
        if (keyValidations[keyHash].registrationBlock > 0) {
            require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        }
        
        // 6. Store validation data (only if not already registered)
        if (keyValidations[keyHash].registrationBlock == 0) {
            keyValidations[keyHash] = KeyValidation({
                keyHash: keyHash,
                registrationBlock: block.number,
                isRevoked: false
            });
        }
        
        // Store the key hash with the agent's address (derived from public key)
        addressToKeyHash[keyAddress] = keyHash;
        // Also store for msg.sender to maintain compatibility
        addressToKeyHash[msg.sender] = keyHash;
        
        emit KeyValidated(keyHash, msg.sender);
    }
    
    /**
     * @notice Verify if a public key is valid and not revoked
     */
    function isKeyValid(bytes calldata publicKey) external view returns (bool) {
        bytes32 keyHash = keccak256(publicKey);
        KeyValidation memory validation = keyValidations[keyHash];
        
        return validation.registrationBlock > 0 && !validation.isRevoked;
    }
    
    /**
     * @notice Revoke a compromised key
     */
    function revokeKey(bytes calldata publicKey) external {
        bytes32 keyHash = keccak256(publicKey);
        require(addressToKeyHash[msg.sender] == keyHash, "Not key owner");
        require(!keyValidations[keyHash].isRevoked, "Already revoked");

        keyValidations[keyHash].isRevoked = true;

        // Deactivate all agents using this key - O(n) where n is agents with this key
        // (not all agents of owner)
        bytes32[] memory agentIds = keyHashToAgentIds[keyHash];
        for (uint i = 0; i < agentIds.length; i++) {
            agents[agentIds[i]].active = false;
        }

        emit KeyRevoked(keyHash, msg.sender);
    }
    
    /**
     * @notice Internal register function
     */
    function _registerAgent(RegistrationParams memory params) private returns (bytes32) {
        // Input validation
        _validateRegistrationInputs(params.did, params.name);
        
        // Generate agent ID
        bytes32 agentId = _generateAgentId(params.did, params.publicKey);
        
        // Execute before hook
        _executeBeforeHook(agentId, params.did, params.publicKey);
        
        // Store agent metadata
        _storeAgentMetadata(agentId, params);
        
        // Execute after hook
        _executeAfterHook(agentId, params.did, params.publicKey);
        
        return agentId;
    }
    
    /**
     * @notice Update agent with signature verification
     */
    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external onlyAgentOwner(agentId) {
        require(bytes(name).length > 0, "Name required");
        
        // Verify key hasn't been revoked (check this first)
        bytes32 keyHash = keccak256(agents[agentId].publicKey);
        require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        
        // Then check if agent is active
        require(agents[agentId].active, "Agent not active");
        
        // Verify signature with stored public key
        bytes32 messageHash = keccak256(abi.encodePacked(
            agentId,
            name,
            description,
            endpoint,
            capabilities,
            msg.sender,
            agentNonce[agentId]
        ));
        
        require(
            _verifySignature(messageHash, signature, agents[agentId].publicKey, msg.sender),
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
    
    // [Previous helper functions remain the same...]
    
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
     * @dev Uses block.number instead of block.timestamp to prevent miner manipulation
     */
    function _generateAgentId(
        string memory did,
        bytes memory publicKey
    ) private returns (bytes32) {
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
    
    /**
     * @notice Internal function to execute before register hook with gas limits and error handling
     */
    function _executeBeforeHook(
        bytes32 agentId,
        string memory did,
        bytes memory publicKey
    ) private {
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);

            // Use try-catch with gas limit to prevent DoS and handle failures
            try IRegistryHook(beforeRegisterHook).beforeRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) returns (bool success, string memory reason) {
                // If hook returns false, revert with the reason
                require(success, reason);
            } catch Error(string memory reason) {
                // Catch revert with reason
                emit HookFailed(beforeRegisterHook, reason);
                revert(reason);
            } catch (bytes memory /* lowLevelData */) {
                // Catch other errors (out of gas, etc.)
                emit HookFailed(beforeRegisterHook, "Hook call failed");
                revert("Hook call failed");
            }
        }
    }
    
    /**
     * @notice Internal function to store agent metadata
     */
    function _storeAgentMetadata(
        bytes32 agentId,
        RegistrationParams memory params
    ) private {
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

        // Track agent by key hash for O(1) revocation
        bytes32 keyHash = keccak256(params.publicKey);
        keyHashToAgentIds[keyHash].push(agentId);

        emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);
    }
    
    /**
     * @notice Internal function to execute after register hook with gas limits and error handling
     * @dev After hooks are non-critical, so failures are logged but don't revert the transaction
     */
    function _executeAfterHook(
        bytes32 agentId,
        string memory did,
        bytes memory publicKey
    ) private {
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);

            // Use try-catch with gas limit - after hooks should not block registration
            try IRegistryHook(afterRegisterHook).afterRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) {
                // Hook executed successfully
                emit AfterRegisterHook(agentId, msg.sender, hookData);
            } catch Error(string memory reason) {
                // Log failure but don't revert - after hooks are non-critical
                emit HookFailed(afterRegisterHook, reason);
            } catch (bytes memory) {
                // Log failure for out of gas or other errors
                emit HookFailed(afterRegisterHook, "Hook call failed");
            }
        }
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
     * @notice Get all agent IDs owned by an address
     */
    function getAgentsByOwner(address _owner) external view returns (bytes32[] memory) {
        return ownerToAgents[_owner];
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
    
    /**
     * @notice Internal function to verify signature
     */
    function _verifySignature(
        bytes32 messageHash,
        bytes memory signature,
        bytes memory publicKey,
        address expectedSigner
    ) private pure returns (bool) {
        // For Ethereum (secp256k1), verify the signer address matches
        if (publicKey.length == 64 || publicKey.length == 65) {
            bytes32 ethSignedHash = keccak256(
                abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
            );
            
            address recovered = _recoverSigner(ethSignedHash, signature);
            return recovered == expectedSigner;
        }
        
        // For Ed25519 (32 bytes), would need external verification
        if (publicKey.length == 32) {
            // In production, use an oracle or ZK proof for Ed25519
            // For now, we'll require secp256k1 keys only
            revert("Ed25519 not supported on-chain");
        }
        
        return false;
    }
    
    /**
     * @notice Recover signer from signature
     */
    function _recoverSigner(bytes32 messageHash, bytes memory signature) 
        private 
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
        
        return ecrecover(messageHash, v, r, s);
    }
    
    /**
     * @notice Derive Ethereum address from public key
     */
    function _getAddressFromPublicKey(bytes memory publicKey) 
        private 
        pure 
        returns (address) 
    {
        // Handle uncompressed key (65 bytes with 0x04 prefix)
        if (publicKey.length == 65 && publicKey[0] == 0x04) {
            // Remove the 0x04 prefix and hash the remaining 64 bytes
            bytes memory keyWithoutPrefix = new bytes(64);
            for (uint i = 0; i < 64; i++) {
                keyWithoutPrefix[i] = publicKey[i + 1];
            }
            // Take the last 20 bytes of the keccak256 hash
            return address(uint160(uint256(keccak256(keyWithoutPrefix))));
        }
        
        // Handle compressed key (33 bytes) - would need elliptic curve decompression
        if (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) {
            // For now, we don't support compressed keys for address derivation
            // This would require implementing elliptic curve point decompression
            revert("Compressed key address derivation not supported");
        }
        
        revert("Invalid public key format for address derivation");
    }
}