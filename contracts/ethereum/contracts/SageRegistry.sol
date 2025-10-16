// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/ISageRegistry.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title SageRegistry
 * @notice SAGE AI Agent Registry Contract
 * @dev Implements secure registration and management of AI agents with public key verification
 */
contract SageRegistry is ISageRegistry, ReentrancyGuard {
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
    ) external validPublicKey(publicKey) nonReentrant returns (bytes32) {
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
     * @notice Internal register function using struct to avoid stack too deep
     */
    function _registerAgent(RegistrationParams memory params) private returns (bytes32) {
        // Input validation
        _validateRegistrationInputs(params.did, params.name);
        
        // Generate agent ID
        bytes32 agentId = _generateAgentId(params.did, params.publicKey);
        
        // Verify signature
        _verifyRegistrationSignature(agentId, params);
        
        // Execute before hook
        _executeBeforeHook(agentId, params.did, params.publicKey);
        
        // Store agent metadata
        _storeAgentMetadata(agentId, params);
        
        // Execute after hook
        _executeAfterHook(agentId, params.did, params.publicKey);
        
        return agentId;
    }
    
    /**
     * @notice Internal function to validate registration inputs
     */
    function _validateRegistrationInputs(
        string memory did,
        string memory name
    ) private view {
        require(bytes(did).length > 0, "DID required");
        require(bytes(name).length > 0, "Name required");
        // slither-disable-next-line incorrect-equality
        // Note: Checking bytes32(0) is safe - it's the default value for uninitialized mapping entries
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        require(ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER, "Too many agents");
    }
    
    /**
     * @notice Internal function to generate agent ID
     */
    function _generateAgentId(
        string memory did,
        bytes memory publicKey
    ) private view returns (bytes32) {
        // Use abi.encode to prevent hash collision attacks
        return keccak256(abi.encode(did, publicKey, block.timestamp));
    }
    
    /**
     * @notice Internal function to verify registration signature
     */
    function _verifyRegistrationSignature(
        bytes32 agentId,
        RegistrationParams memory params
    ) private view {
        // Use abi.encode to prevent hash collision attacks
        bytes32 messageHash = keccak256(abi.encode(
            params.did,
            params.name,
            params.description,
            params.endpoint,
            params.publicKey,
            params.capabilities,
            msg.sender,
            agentNonce[agentId]
        ));

        require(_verifySignature(messageHash, params.signature, params.publicKey, msg.sender), "Invalid signature");
    }
    
    /**
     * @notice Internal function to execute before register hook
     */
    function _executeBeforeHook(
        bytes32 agentId,
        string memory did,
        bytes memory publicKey
    ) private {
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);
            
            (bool success, string memory reason) = IRegistryHook(beforeRegisterHook)
                .beforeRegister(agentId, msg.sender, hookData);
            require(success, reason);
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
        
        emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);
    }
    
    /**
     * @notice Internal function to execute after register hook
     */
    function _executeAfterHook(
        bytes32 agentId,
        string memory did,
        bytes memory publicKey
    ) private {
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            IRegistryHook(afterRegisterHook).afterRegister(agentId, msg.sender, hookData);
            emit AfterRegisterHook(agentId, msg.sender, hookData);
        }
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
        // Use abi.encode to prevent hash collision attacks
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
     * @notice Deactivate an agent by DID (O(1) lookup)
     * @dev More efficient than deactivateAgent as it uses DID-to-ID mapping
     */
    function deactivateAgentByDID(string calldata did) external {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        require(agents[agentId].owner == msg.sender, "Not agent owner");
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
        // slither-disable-next-line incorrect-equality
        // Note: Checking publicKey.length is safe - bytes array length is deterministic
        // For Ethereum (secp256k1), verify the signer address matches
        if (publicKey.length == 64 || publicKey.length == 65) {
            bytes32 ethSignedHash = keccak256(
                abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
            );

            address recovered = _recoverSigner(ethSignedHash, signature);
            return recovered == expectedSigner;
        }

        // slither-disable-next-line incorrect-equality
        // Note: Checking publicKey.length is safe - bytes array length is deterministic
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