// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../erc-8004/interfaces/IERC8004ValidationRegistry.sol";

/**
 * @title ReentrancyAttacker
 * @notice Malicious contract for testing reentrancy protection
 * @dev This contract attempts to reenter the ValidationRegistry during fund transfers
 */
contract ReentrancyAttacker {
    IERC8004ValidationRegistry public validationRegistry;
    bytes32 public attackRequestId;
    bytes32 public attackComputedHash;
    uint256 public attackCount;
    bool public attacking;

    constructor(address _validationRegistry) {
        validationRegistry = IERC8004ValidationRegistry(_validationRegistry);
    }

    /**
     * @notice Receive function that attempts reentrancy
     */
    receive() external payable {
        if (attacking && attackCount < 3) {
            attackCount++;

            // Attempt to reenter by submitting another validation
            // solhint-disable-next-line no-empty-blocks
            try validationRegistry.submitStakeValidation{value: msg.value / 2}(
                attackRequestId,
                attackComputedHash
            ) {
                // If this succeeds, reentrancy protection failed
                revert("Reentrancy attack succeeded");
            } catch { // solhint-disable-line no-empty-blocks
                // Expected - reentrancy should be prevented
            }
        }
    }

    /**
     * @notice Start reentrancy attack during validation submission
     */
    function attackSubmitStakeValidation(
        bytes32 requestId,
        bytes32 computedHash
    ) external payable {
        attacking = true;
        attackCount = 0;
        attackRequestId = requestId;
        attackComputedHash = computedHash;

        validationRegistry.submitStakeValidation{value: msg.value}(
            requestId,
            computedHash
        );

        attacking = false;
    }

    /**
     * @notice Start reentrancy attack during validation request
     */
    function attackRequestValidation(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        IERC8004ValidationRegistry.ValidationType validationType,
        uint256 deadline
    ) external payable returns (bytes32) {
        attacking = true;
        attackCount = 0;

        bytes32 requestId = validationRegistry.requestValidation{value: msg.value}(
            taskId,
            serverAgent,
            dataHash,
            validationType,
            deadline
        );

        attacking = false;
        return requestId;
    }

    /**
     * @notice Reset attack state
     */
    function reset() external {
        attacking = false;
        attackCount = 0;
        attackRequestId = bytes32(0);
        attackComputedHash = bytes32(0);
    }
}
