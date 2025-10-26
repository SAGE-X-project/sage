// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import "@openzeppelin/contracts/governance/TimelockController.sol";

/**
 * @title SAGETimelockController
 * @notice Wrapper around OpenZeppelin's TimelockController for SAGE governance
 * @dev This contract re-exports the OpenZeppelin TimelockController so that
 *      Hardhat can generate artifacts for it and it can be used in tests.
 */
contract SAGETimelockController is TimelockController {
    constructor(
        uint256 minDelay,
        address[] memory proposers,
        address[] memory executors,
        address admin
    ) TimelockController(minDelay, proposers, executors, admin) {}
}
