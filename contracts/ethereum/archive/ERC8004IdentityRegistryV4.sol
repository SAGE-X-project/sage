// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./erc-8004/interfaces/IERC8004IdentityRegistry.sol";
import "./AgentCardRegistry.sol";

/**
 * @title ERC8004IdentityRegistryV4
 * @notice ERC-8004 compliant adapter for AgentCardRegistry
 * @dev Production-ready ERC-8004 implementation
 *
 * This contract provides an ERC-8004 compliant interface to the SAGE
 * AgentCardRegistry system. It acts as an adapter, translating ERC-8004
 * calls to AgentCardRegistry operations.
 *
 * Key Design Decisions:
 * - Wraps AgentCardRegistry instead of reimplementing logic
 * - registerAgent() requires off-chain commit-reveal flow
 * - Supports multi-key agents via underlying registry
 * - Provides simplified ERC-8004 interface for ecosystem compatibility
 *
 * @custom:security-contact security@sage.com
 */
contract ERC8004IdentityRegistryV4 is IERC8004IdentityRegistry {
    // ============ State Variables ============

    /**
     * @notice Reference to the underlying AgentCardRegistry
     * @dev Immutable for security and gas optimization
     */
    AgentCardRegistry public immutable AGENT_REGISTRY;

    /**
     * @notice Mapping from DID to AgentDomain (future use)
     * @dev Reserved for future AgentDomain functionality
     */
    mapping(string => string) private agentDomains;

    // ============ Constructor ============

    /**
     * @notice Initialize the ERC-8004 registry adapter
     * @param registryAddress Address of deployed AgentCardRegistry
     */
    constructor(address registryAddress) {
        require(registryAddress != address(0), "Invalid registry address");
        AGENT_REGISTRY = AgentCardRegistry(registryAddress);
    }

    // ============ ERC-8004 Interface Implementation ============

    /**
     * @notice Register agent with simplified ERC-8004 interface
     * @dev This function requires the full commit-reveal flow
     *      Users should use AgentCardRegistry.commitRegistration() directly
     *      for production use with multi-key support
     *
     * @param agentId DID identifier (e.g., "did:sage:ethereum:0x...")
     * @param endpoint AgentCard URL or IPFS hash
     * @return success Always reverts with instruction message
     */
    function registerAgent(
        string calldata agentId,
        string calldata endpoint
    ) external override returns (bool success) {
        // Suppress unused variable warnings
        agentId;
        endpoint;

        // ERC-8004 simplified registration doesn't support commit-reveal
        // For full security and multi-key support, use AgentCardRegistry directly
        revert("Use AgentCardRegistry.commitRegistration() for full flow");
    }

    /**
     * @notice Resolve agent information by DID
     * @dev Returns ERC-8004 compliant AgentInfo struct
     *
     * @param agentId The DID to look up
     * @return info Agent information including DID, address, endpoint, status
     */
    function resolveAgent(string calldata agentId)
        external
        view
        override
        returns (AgentInfo memory info)
    {
        AgentCardRegistry.AgentMetadata memory metadata =
            AGENT_REGISTRY.getAgentByDID(agentId);

        // Check if agent exists (owner will be address(0) for non-existent agents)
        require(metadata.owner != address(0), "Agent not found");

        // Convert AgentMetadata to ERC-8004 AgentInfo
        info = AgentInfo({
            agentId: metadata.did,
            agentAddress: metadata.owner,
            endpoint: metadata.endpoint,
            isActive: metadata.active,
            registeredAt: metadata.registeredAt
        });
    }

    /**
     * @notice Resolve agent information by owner address
     * @dev Returns first agent owned by the address
     *      If multiple agents exist, only the first is returned
     *
     * @param agentAddress The owner address to look up
     * @return info Agent information for first agent owned by address
     */
    function resolveAgentByAddress(address agentAddress)
        external
        view
        override
        returns (AgentInfo memory info)
    {
        bytes32[] memory agentIds = AGENT_REGISTRY.getAgentsByOwner(agentAddress);
        require(agentIds.length > 0, "No agent found");

        // Get first agent (most common case: one agent per address)
        AgentCardRegistry.AgentMetadata memory metadata =
            AGENT_REGISTRY.getAgent(agentIds[0]);

        info = AgentInfo({
            agentId: metadata.did,
            agentAddress: metadata.owner,
            endpoint: metadata.endpoint,
            isActive: metadata.active,
            registeredAt: metadata.registeredAt
        });
    }

    /**
     * @notice Check if an agent is active
     * @dev Queries underlying registry for active status
     *
     * Note: Agents have a time-locked activation period after registration.
     * An agent is only active after the activation delay has passed and
     * activateAgent() has been called.
     *
     * @param agentId The DID to check
     * @return isActive True if agent is active, false otherwise
     */
    function isAgentActive(string calldata agentId)
        external
        view
        override
        returns (bool)
    {
        AgentCardRegistry.AgentMetadata memory metadata =
            AGENT_REGISTRY.getAgentByDID(agentId);

        // Check if agent exists
        require(metadata.owner != address(0), "Agent not found");

        return metadata.active;
    }

    /**
     * @notice Update agent's endpoint (AgentCard location)
     * @dev Only callable by agent owner
     *      Calls underlying registry's updateAgent function
     *
     * @param agentId The DID of the agent to update
     * @param newEndpoint New AgentCard URL or IPFS hash
     * @return success True if update successful
     */
    function updateAgentEndpoint(
        string calldata agentId,
        string calldata newEndpoint
    ) external override returns (bool success) {
        // Get agent metadata to verify ownership and get agentId hash
        bytes32 agentIdHash = AGENT_REGISTRY.didToAgentId(agentId);
        require(agentIdHash != bytes32(0), "Agent not found");

        AgentCardRegistry.AgentMetadata memory metadata =
            AGENT_REGISTRY.getAgent(agentIdHash);

        require(metadata.owner == msg.sender, "Not agent owner");

        // Store old endpoint for event
        string memory oldEndpoint = metadata.endpoint;

        // Update via AgentCardRegistry
        AGENT_REGISTRY.updateAgent(
            agentIdHash,
            newEndpoint,
            metadata.capabilities
        );

        emit AgentEndpointUpdated(agentId, oldEndpoint, newEndpoint);
        return true;
    }

    /**
     * @notice Deactivate an agent
     * @dev Only callable by agent owner
     *      Calls underlying registry's deactivateAgent function
     *
     * Note: Deactivation may trigger stake return after 30 days
     *
     * @param agentId The DID of the agent to deactivate
     * @return success True if deactivation successful
     */
    function deactivateAgent(string calldata agentId)
        external
        override
        returns (bool success)
    {
        // Get agent metadata
        bytes32 agentIdHash = AGENT_REGISTRY.didToAgentId(agentId);
        require(agentIdHash != bytes32(0), "Agent not found");

        AgentCardRegistry.AgentMetadata memory metadata =
            AGENT_REGISTRY.getAgent(agentIdHash);

        require(metadata.owner == msg.sender, "Not agent owner");

        // Deactivate via AgentCardRegistry
        AGENT_REGISTRY.deactivateAgent(agentIdHash);

        emit AgentDeactivated(agentId, msg.sender);
        return true;
    }
}
