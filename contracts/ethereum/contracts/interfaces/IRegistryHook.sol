// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

/**
 * @title IRegistryHook
 * @notice Interface for registry hooks
 */
interface IRegistryHook {
    /**
     * @notice Called before agent registration
     * @param agentId The agent ID
     * @param owner The owner address
     * @param data Additional data passed to the hook
     * @return success Whether to proceed with registration
     * @return reason Reason if registration should be blocked
     */
    function beforeRegister(
        bytes32 agentId,
        address owner,
        bytes calldata data
    ) external returns (bool success, string memory reason);

    /**
     * @notice Called after agent registration
     * @param agentId The agent ID
     * @param owner The owner address
     * @param data Additional data passed to the hook
     */
    function afterRegister(
        bytes32 agentId,
        address owner,
        bytes calldata data
    ) external;
}