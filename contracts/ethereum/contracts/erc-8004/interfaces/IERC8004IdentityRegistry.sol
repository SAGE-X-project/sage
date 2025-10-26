// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

/**
 * @title IERC8004IdentityRegistry
 * @notice Interface for ERC-8004 Identity Registry
 * @dev Part of ERC-8004: Trustless Agents standard
 *      https://eips.ethereum.org/EIPS/eip-8004
 *
 * The Identity Registry provides a unique identifier for agents and resolves
 * to their off-chain AgentCard containing capabilities and connection information.
 */
interface IERC8004IdentityRegistry {
    /**
     * @notice Agent metadata structure
     * @dev Minimal on-chain storage, most data in off-chain AgentCard
     */
    struct AgentInfo {
        string agentId;        // Unique agent identifier (e.g., DID)
        address agentAddress;  // On-chain address
        string endpoint;       // Off-chain AgentCard URL or IPFS hash
        bool isActive;         // Agent status
        uint256 registeredAt;  // Registration timestamp
    }

    /**
     * @notice Register a new agent
     * @dev Emits AgentRegistered event
     * @param agentId Unique identifier for the agent
     * @param endpoint URL or IPFS hash pointing to AgentCard
     * @return success True if registration successful
     */
    function registerAgent(
        string calldata agentId,
        string calldata endpoint
    ) external returns (bool success);

    /**
     * @notice Resolve agent information by agent ID
     * @param agentId The agent identifier to look up
     * @return info Agent information struct
     */
    function resolveAgent(string calldata agentId)
        external view returns (AgentInfo memory info);

    /**
     * @notice Resolve agent information by address
     * @param agentAddress The agent's on-chain address
     * @return info Agent information struct
     */
    function resolveAgentByAddress(address agentAddress)
        external view returns (AgentInfo memory info);

    /**
     * @notice Check if an agent is active
     * @param agentId The agent identifier
     * @return isActive True if agent is active
     */
    function isAgentActive(string calldata agentId)
        external view returns (bool isActive);

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
    ) external returns (bool success);

    /**
     * @notice Deactivate an agent
     * @dev Only callable by agent owner
     * @param agentId The agent identifier
     * @return success True if deactivation successful
     */
    function deactivateAgent(string calldata agentId)
        external returns (bool success);

    /**
     * @notice Emitted when a new agent is registered
     * @param agentId The unique agent identifier
     * @param agentAddress The agent's on-chain address
     * @param endpoint The AgentCard location
     */
    event AgentRegistered(
        string indexed agentId,
        address indexed agentAddress,
        string endpoint
    );

    /**
     * @notice Emitted when an agent's endpoint is updated
     * @param agentId The agent identifier
     * @param oldEndpoint Previous endpoint
     * @param newEndpoint New endpoint
     */
    event AgentEndpointUpdated(
        string indexed agentId,
        string oldEndpoint,
        string newEndpoint
    );

    /**
     * @notice Emitted when an agent is deactivated
     * @param agentId The agent identifier
     * @param agentAddress The agent's address
     */
    event AgentDeactivated(
        string indexed agentId,
        address indexed agentAddress
    );
}
