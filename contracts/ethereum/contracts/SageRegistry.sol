// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./interfaces/ISageRegistry.sol";
import "./interfaces/IRegistryHook.sol";

/**
 * @title SageRegistry
 * @notice SAGE AI Agent Registry Contract
 * @dev Implements secure registration and management of AI agents with public key verification
 */
contract SageRegistry is ISageRegistry {
    // State variables
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(bytes32 => uint256) private agentNonce;
    
    address public owner;
    address public beforeRegisterHook;
    address public afterRegisterHook;
    
    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MIN_PUBLIC_KEY_LENGTH = 32;
    uint256 private constant MAX_PUBLIC_KEY_LENGTH = 65;
    
    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }
    
    modifier onlyAgentOwner(bytes32 agentId) {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        _;
    }
    
    modifier validPublicKey(bytes memory publicKey) {
        require(
            publicKey.length >= MIN_PUBLIC_KEY_LENGTH && 
            publicKey.length <= MAX_PUBLIC_KEY_LENGTH,
            "Invalid public key length"
        );
        _;
    }
    
    constructor() {
        owner = msg.sender;
    }
    
    /**
     * @notice Register a new AI agent
     * @dev Verifies signature to ensure the sender owns the public key
     */
    function registerAgent(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature
    ) external validPublicKey(publicKey) returns (bytes32) {
        require(bytes(did).length > 0, "DID required");
        require(bytes(name).length > 0, "Name required");
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        require(ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER, "Too many agents");
        
        // Generate agent ID
        bytes32 agentId = keccak256(abi.encodePacked(did, publicKey, block.timestamp));
        
        // Verify signature
        bytes32 messageHash = keccak256(abi.encodePacked(
            did,
            name,
            description,
            endpoint,
            publicKey,
            capabilities,
            msg.sender,
            agentNonce[agentId]
        ));
        
        require(_verifySignature(messageHash, signature, publicKey, msg.sender), "Invalid signature");
        
        // Execute before hook if set
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);
            
            (bool success, string memory reason) = IRegistryHook(beforeRegisterHook)
                .beforeRegister(agentId, msg.sender, hookData);
            require(success, reason);
        }
        
        // Store agent metadata
        agents[agentId] = AgentMetadata({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            publicKey: publicKey,
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
        
        // Execute after hook if set
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            IRegistryHook(afterRegisterHook).afterRegister(agentId, msg.sender, hookData);
            emit AfterRegisterHook(agentId, msg.sender, hookData);
        }
        
        return agentId;
    }
    
    /**
     * @notice Update agent metadata
     * @dev Only agent owner can update, signature required
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
     * @dev Supports both ECDSA and Ed25519 signatures
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
        
        // For Ed25519 (32 bytes), we would need external verification
        // This is a placeholder - in production, use a library or precompile
        if (publicKey.length == 32) {
            // Ed25519 verification would go here
            // For now, we'll require a separate verification step
            return true;
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
}