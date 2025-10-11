// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "@openzeppelin/contracts/governance/TimelockController.sol";

/**
 * @title TimelockController
 * @notice Re-export OpenZeppelin's TimelockController for easier testing
 * @dev This contract simply re-exports the OpenZeppelin TimelockController
 *      so that Hardhat can generate artifacts for it.
 */
