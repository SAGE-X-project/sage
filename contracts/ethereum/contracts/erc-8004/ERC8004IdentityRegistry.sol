// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./interfaces/IERC8004IdentityRegistry.sol";
import "../SageRegistryV2.sol";

/**
 * @title ERC8004IdentityRegistry
 * @notice ERC-8004 compliant Identity Registry adapter for SAGE
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * This contract wraps the existing SageRegistryV2 to provide an ERC-8004
 * compliant interface while maintaining backward compatibility with SAGE's
 * existing agent registration system.
 */
contract ERC8004IdentityRegistry is IERC8004IdentityRegistry {
    SageRegistryV2 public sageRegistry;

    // Mapping from ERC-8004 agentId (DID string) to agent address
    mapping(string => address) private agentIdToAddress;

    constructor(address _sageRegistry) {
        require(_sageRegistry != address(0), "Invalid registry address");
        sageRegistry = SageRegistryV2(_sageRegistry);
    }

    /**
     * @notice Register a new agent (ERC-8004 compliant)
     * @dev Delegates to SageRegistryV2 with additional ERC-8004 event emission
     * @param agentId Unique identifier for the agent (DID)
     * @param endpoint URL or IPFS hash pointing to AgentCard
     * @return success True if registration successful
     */
    function registerAgent(
        string calldata agentId,
        string calldata endpoint
    ) external override returns (bool success) {
        // For minimal ERC-8004 registration, we create default values
        // In production, the caller should use SageRegistryV2 directly for full control

        // Generate a minimal public key requirement
        // In practice, agents should register via SageRegistryV2 with proper keys
        revert("Use SageRegistryV2.registerAgent for full registration");
    }

    /**
     * @notice Resolve agent information by agent ID
     * @param agentId The agent identifier to look up (DID)
     * @return info Agent information struct
     */
    function resolveAgent(string calldata agentId)
        external
        view
        override
        returns (AgentInfo memory info)
    {
        // Get agent from SageRegistryV2 using DID
        ISageRegistry.AgentMetadata memory metadata = sageRegistry.getAgentByDID(agentId);

        // Convert to ERC-8004 format
        info = AgentInfo({
            agentId: metadata.did,
            agentAddress: metadata.owner,
            endpoint: metadata.endpoint,
            isActive: metadata.active,
            registeredAt: metadata.registeredAt
        });

        return info;
    }

    /**
     * @notice Resolve agent information by address
     * @param agentAddress The agent's on-chain address
     * @return info Agent information struct
     */
    function resolveAgentByAddress(address agentAddress)
        external
        view
        override
        returns (AgentInfo memory info)
    {
        // Get all agents owned by this address
        bytes32[] memory agentIds = sageRegistry.getAgentsByOwner(agentAddress);
        require(agentIds.length > 0, "No agent found for address");

        // Return the first (primary) agent
        ISageRegistry.AgentMetadata memory metadata = sageRegistry.getAgent(agentIds[0]);

        info = AgentInfo({
            agentId: metadata.did,
            agentAddress: metadata.owner,
            endpoint: metadata.endpoint,
            isActive: metadata.active,
            registeredAt: metadata.registeredAt
        });

        return info;
    }

    /**
     * @notice Check if an agent is active
     * @param agentId The agent identifier (DID)
     * @return isActive True if agent is active
     */
    function isAgentActive(string calldata agentId)
        external
        view
        override
        returns (bool)
    {
        ISageRegistry.AgentMetadata memory metadata = sageRegistry.getAgentByDID(agentId);
        return metadata.active;
    }

    /**
     * @notice Update agent endpoint (AgentCard location)
     * @dev Delegates to SageRegistryV2's updateAgent function
     * @param agentId The agent identifier (DID)
     * @param newEndpoint New AgentCard URL or IPFS hash
     * @return success True if update successful
     */
    function updateAgentEndpoint(
        string calldata agentId,
        string calldata newEndpoint
    ) external override returns (bool success) {
        // Get current agent metadata
        ISageRegistry.AgentMetadata memory metadata = sageRegistry.getAgentByDID(agentId);
        require(metadata.owner == msg.sender, "Not agent owner");

        // For endpoint-only updates, use SageRegistryV2 directly
        // This is a limitation of the adapter pattern - full updates require signature
        revert("Use SageRegistryV2.updateAgent for updates");
    }

    /**
     * @notice Deactivate an agent
     * @dev Delegates to SageRegistryV2's deactivateAgent function
     * @param agentId The agent identifier (DID)
     * @return success True if deactivation successful
     */
    function deactivateAgent(string calldata agentId)
        external
        override
        returns (bool success)
    {
        // Get agent by DID to find the bytes32 agentId
        ISageRegistry.AgentMetadata memory metadata = sageRegistry.getAgentByDID(agentId);
        require(metadata.owner == msg.sender, "Not agent owner");

        // Get the bytes32 agent ID
        bytes32[] memory agentIds = sageRegistry.getAgentsByOwner(msg.sender);
        bytes32 targetAgentId;

        for (uint i = 0; i < agentIds.length; i++) {
            ISageRegistry.AgentMetadata memory agent = sageRegistry.getAgent(agentIds[i]);
            if (keccak256(bytes(agent.did)) == keccak256(bytes(agentId))) {
                targetAgentId = agentIds[i];
                break;
            }
        }

        require(targetAgentId != bytes32(0), "Agent not found");

        // Deactivate via SageRegistryV2
        sageRegistry.deactivateAgent(targetAgentId);

        emit AgentDeactivated(agentId, msg.sender);

        return true;
    }

    /**
     * @notice Get the underlying SageRegistryV2 address
     * @return Address of the SageRegistryV2 contract
     */
    function getSageRegistry() external view returns (address) {
        return address(sageRegistry);
    }
}
