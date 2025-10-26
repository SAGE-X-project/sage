// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import "../interfaces/IERC8004IdentityRegistry.sol";

/**
 * @title ERC8004IdentityRegistry
 * @notice Standalone implementation of ERC-8004 Identity Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * This is a STANDALONE implementation that does NOT depend on any project-specific contracts.
 * It can be used independently or integrated with other systems via adapters.
 *
 * Features:
 * - Self-contained agent registration and management
 * - Minimal on-chain storage (most data in off-chain AgentCard)
 * - DID-based agent identification
 * - Endpoint management for AgentCard resolution
 * - Owner-controlled agent lifecycle
 */
contract ERC8004IdentityRegistry is IERC8004IdentityRegistry {

    // ============================================
    // STATE VARIABLES
    // ============================================

    /// @dev Mapping from agentId (DID) to agent information
    mapping(string => AgentInfo) private agents;

    /// @dev Mapping from address to agentId (for reverse lookup)
    mapping(address => string) private addressToAgentId;

    /// @dev Mapping from agentId to owner address (for access control)
    mapping(string => address) private agentOwners;

    /// @dev Total number of registered agents
    uint256 public totalAgents;

    // ============================================
    // ERRORS
    // ============================================

    error AgentAlreadyRegistered(string agentId);
    error AgentNotFound(string agentId);
    error AgentNotActive(string agentId);
    error NotAgentOwner(string agentId, address caller);
    error InvalidAgentId();
    error InvalidEndpoint();
    error InvalidAddress();

    // ============================================
    // MODIFIERS
    // ============================================

    modifier onlyAgentOwner(string calldata agentId) {
        if (agentOwners[agentId] != msg.sender) {
            revert NotAgentOwner(agentId, msg.sender);
        }
        _;
    }

    modifier validAgentId(string calldata agentId) {
        if (bytes(agentId).length == 0) {
            revert InvalidAgentId();
        }
        _;
    }

    modifier validEndpoint(string calldata endpoint) {
        if (bytes(endpoint).length == 0) {
            revert InvalidEndpoint();
        }
        _;
    }

    // ============================================
    // EXTERNAL FUNCTIONS
    // ============================================

    /**
     * @notice Register a new agent
     * @dev Creates a new agent with the caller as owner
     * @param agentId Unique identifier for the agent (DID format recommended)
     * @param endpoint URL or IPFS hash pointing to AgentCard
     * @return success True if registration successful
     */
    function registerAgent(
        string calldata agentId,
        string calldata endpoint
    )
        external
        override
        validAgentId(agentId)
        validEndpoint(endpoint)
        returns (bool success)
    {
        // Check if agent already exists
        if (agents[agentId].registeredAt != 0) {
            revert AgentAlreadyRegistered(agentId);
        }

        // Check if address already has an agent
        if (bytes(addressToAgentId[msg.sender]).length != 0) {
            revert AgentAlreadyRegistered(addressToAgentId[msg.sender]);
        }

        // Create agent info
        agents[agentId] = AgentInfo({
            agentId: agentId,
            agentAddress: msg.sender,
            endpoint: endpoint,
            isActive: true,
            registeredAt: block.timestamp
        });

        // Store mappings
        addressToAgentId[msg.sender] = agentId;
        agentOwners[agentId] = msg.sender;

        // Increment counter
        totalAgents++;

        emit AgentRegistered(agentId, msg.sender, endpoint);

        return true;
    }

    /**
     * @notice Resolve agent information by agent ID
     * @param agentId The agent identifier to look up
     * @return info Agent information struct
     */
    function resolveAgent(string calldata agentId)
        external
        view
        override
        returns (AgentInfo memory info)
    {
        info = agents[agentId];
        // slither-disable-next-line incorrect-equality
        // Note: Checking registeredAt == 0 is safe - it's used to detect uninitialized structs, not for time comparison
        if (info.registeredAt == 0) {
            revert AgentNotFound(agentId);
        }
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
        if (agentAddress == address(0)) {
            revert InvalidAddress();
        }

        string memory agentId = addressToAgentId[agentAddress];
        if (bytes(agentId).length == 0) {
            revert AgentNotFound("address lookup failed");
        }

        info = agents[agentId];
        return info;
    }

    /**
     * @notice Check if an agent is active
     * @param agentId The agent identifier
     * @return isActive True if agent is active
     */
    function isAgentActive(string calldata agentId)
        external
        view
        override
        returns (bool)
    {
        AgentInfo storage agent = agents[agentId];
        if (agent.registeredAt == 0) {
            return false;
        }
        return agent.isActive;
    }

    /**
     * @notice Update agent endpoint (AgentCard location)
     * @dev Only callable by agent owner
     * @param agentId The agent identifier
     * @param newEndpoint New AgentCard URL or IPFS hash
     * @return success True if update successful
     */
    function updateAgentEndpoint(
        string calldata agentId,
        string calldata newEndpoint
    )
        external
        override
        onlyAgentOwner(agentId)
        validEndpoint(newEndpoint)
        returns (bool success)
    {
        AgentInfo storage agent = agents[agentId];

        if (agent.registeredAt == 0) {
            revert AgentNotFound(agentId);
        }

        if (!agent.isActive) {
            revert AgentNotActive(agentId);
        }

        string memory oldEndpoint = agent.endpoint;
        agent.endpoint = newEndpoint;

        emit AgentEndpointUpdated(agentId, oldEndpoint, newEndpoint);

        return true;
    }

    /**
     * @notice Deactivate an agent
     * @dev Only callable by agent owner
     * @param agentId The agent identifier
     * @return success True if deactivation successful
     */
    function deactivateAgent(string calldata agentId)
        external
        override
        onlyAgentOwner(agentId)
        returns (bool success)
    {
        AgentInfo storage agent = agents[agentId];

        if (agent.registeredAt == 0) {
            revert AgentNotFound(agentId);
        }

        if (!agent.isActive) {
            revert AgentNotActive(agentId);
        }

        agent.isActive = false;

        emit AgentDeactivated(agentId, agent.agentAddress);

        return true;
    }
}
