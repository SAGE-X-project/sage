// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

/**
 * @title ISageRegistry
 * @notice Interface for SAGE AI Agent Registry
 */
interface ISageRegistry {
    // Agent metadata structure
    struct AgentMetadata {
        string did;                  // Decentralized Identifier
        string name;                 // Agent name
        string description;          // Agent description
        string endpoint;             // Agent endpoint URL
        bytes publicKey;             // Agent's public key
        string capabilities;         // JSON string of capabilities
        address owner;               // Owner address
        uint256 registeredAt;        // Registration timestamp
        uint256 updatedAt;           // Last update timestamp
        bool active;                 // Active status
    }

    // Events
    event AgentRegistered(
        bytes32 indexed agentId,
        address indexed owner,
        string did,
        uint256 timestamp
    );

    event AgentUpdated(
        bytes32 indexed agentId,
        address indexed owner,
        uint256 timestamp
    );

    event AgentDeactivated(
        bytes32 indexed agentId,
        address indexed owner,
        uint256 timestamp
    );

    event BeforeRegisterHook(
        bytes32 indexed agentId,
        address indexed owner,
        bytes data
    );

    event AfterRegisterHook(
        bytes32 indexed agentId,
        address indexed owner,
        bytes data
    );

    // Core functions
    function registerAgent(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature
    ) external returns (bytes32);

    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external;

    function deactivateAgent(bytes32 agentId) external;

    function deactivateAgentByDID(string calldata did) external;

    function getAgent(bytes32 agentId) external view returns (AgentMetadata memory);

    function getAgentByDID(string calldata did) external view returns (AgentMetadata memory);

    function getAgentsByOwner(address owner) external view returns (bytes32[] memory);

    function verifyAgentOwnership(bytes32 agentId, address claimedOwner) external view returns (bool);

    function isAgentActive(bytes32 agentId) external view returns (bool);

    // Hook functions
    function setBeforeRegisterHook(address hook) external;
    function setAfterRegisterHook(address hook) external;
}