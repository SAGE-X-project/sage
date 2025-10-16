// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title SimpleMultiSig
 * @notice Simple multi-signature wallet for SAGE governance
 * @dev Simplified implementation for testing. For production, use Gnosis Safe.
 *
 * Features:
 * - M-of-N signature requirement
 * - Transaction proposal and execution
 * - Owner management
 * - Compatible with TimelockController
 *
 * Security Notes:
 * - This is a SIMPLIFIED implementation for testing
 * - For mainnet, use audited Gnosis Safe contracts
 * - This contract is NOT recommended for production use
 */
contract SimpleMultiSig is ReentrancyGuard {
    // Events
    event OwnerAdded(address indexed owner);
    event OwnerRemoved(address indexed owner);
    event ThresholdChanged(uint256 threshold);
    event TransactionProposed(uint256 indexed transactionId, address indexed proposer);
    event TransactionConfirmed(uint256 indexed transactionId, address indexed owner);
    event TransactionRevoked(uint256 indexed transactionId, address indexed owner);
    event TransactionExecuted(uint256 indexed transactionId, address indexed executor);
    event TransactionFailed(uint256 indexed transactionId, string reason);

    // Transaction structure
    struct Transaction {
        address to;
        uint256 value;
        bytes data;
        bool executed;
        uint256 confirmations;
    }

    // State variables
    mapping(address => bool) public isOwner;
    address[] public owners;
    uint256 public immutable threshold;

    mapping(uint256 => Transaction) public transactions;
    mapping(uint256 => mapping(address => bool)) public confirmations;
    uint256 public transactionCount;

    // Modifiers
    modifier onlyOwner() {
        require(isOwner[msg.sender], "Not an owner");
        _;
    }

    modifier transactionExists(uint256 transactionId) {
        require(transactionId < transactionCount, "Transaction does not exist");
        _;
    }

    modifier notExecuted(uint256 transactionId) {
        require(!transactions[transactionId].executed, "Transaction already executed");
        _;
    }

    modifier notConfirmed(uint256 transactionId) {
        require(!confirmations[transactionId][msg.sender], "Transaction already confirmed");
        _;
    }

    /**
     * @notice Constructor
     * @param _owners Array of owner addresses
     * @param _threshold Number of required confirmations
     */
    constructor(address[] memory _owners, uint256 _threshold) {
        require(_owners.length > 0, "Owners required");
        require(_threshold > 0 && _threshold <= _owners.length, "Invalid threshold");

        for (uint256 i = 0; i < _owners.length; i++) {
            address owner = _owners[i];

            require(owner != address(0), "Invalid owner address");
            require(!isOwner[owner], "Owner not unique");

            isOwner[owner] = true;
            owners.push(owner);
            emit OwnerAdded(owner);
        }

        threshold = _threshold;
        emit ThresholdChanged(_threshold);
    }

    /**
     * @notice Receive ETH
     */
    receive() external payable {}

    /**
     * @notice Get number of owners
     */
    function getOwnerCount() public view returns (uint256) {
        return owners.length;
    }

    /**
     * @notice Get all owners
     */
    function getOwners() public view returns (address[] memory) {
        return owners;
    }

    /**
     * @notice Propose a transaction
     * @param to Destination address
     * @param value Amount of ETH to send
     * @param data Transaction data
     * @return transactionId ID of the proposed transaction
     */
    function proposeTransaction(
        address to,
        uint256 value,
        bytes memory data
    ) public onlyOwner returns (uint256 transactionId) {
        transactionId = transactionCount;

        transactions[transactionId] = Transaction({
            to: to,
            value: value,
            data: data,
            executed: false,
            confirmations: 0
        });

        transactionCount++;

        emit TransactionProposed(transactionId, msg.sender);

        // Automatically confirm for proposer
        confirmTransaction(transactionId);

        return transactionId;
    }

    /**
     * @notice Confirm a transaction
     * @param transactionId ID of the transaction
     */
    function confirmTransaction(uint256 transactionId)
        public
        onlyOwner
        transactionExists(transactionId)
        notExecuted(transactionId)
        notConfirmed(transactionId)
    {
        confirmations[transactionId][msg.sender] = true;
        transactions[transactionId].confirmations++;

        emit TransactionConfirmed(transactionId, msg.sender);

        // Auto-execute if threshold reached
        if (transactions[transactionId].confirmations >= threshold) {
            executeTransaction(transactionId);
        }
    }

    /**
     * @notice Revoke confirmation
     * @param transactionId ID of the transaction
     */
    function revokeConfirmation(uint256 transactionId)
        public
        onlyOwner
        transactionExists(transactionId)
        notExecuted(transactionId)
    {
        require(confirmations[transactionId][msg.sender], "Transaction not confirmed");

        confirmations[transactionId][msg.sender] = false;
        transactions[transactionId].confirmations--;

        emit TransactionRevoked(transactionId, msg.sender);
    }

    /**
     * @notice Execute a transaction
     * @param transactionId ID of the transaction
     */
    function executeTransaction(uint256 transactionId)
        public
        onlyOwner
        transactionExists(transactionId)
        notExecuted(transactionId)
        nonReentrant
    {
        Transaction storage txn = transactions[transactionId];

        require(txn.confirmations >= threshold, "Insufficient confirmations");

        // Mark as executed BEFORE external call (Checks-Effects-Interactions pattern)
        txn.executed = true;

        // Load struct fields to avoid complex assembly for storage access
        address target = txn.to;
        uint256 value = txn.value;
        bytes memory data = txn.data;

        // Execute with return data size limit to prevent return bomb attack
        bool success;
        bytes memory returnData;
        uint256 maxReturnSize = 1024;

        assembly {
            // Allocate memory for return data (limit to prevent return bomb)
            returnData := mload(0x40)
            mstore(returnData, 0)
            mstore(0x40, add(returnData, add(maxReturnSize, 32)))

            // Call with limited return data size
            success := call(
                gas(),
                target,
                value,
                add(data, 32),
                mload(data),
                add(returnData, 32),
                maxReturnSize
            )

            // Store actual return data size (capped at maxReturnSize)
            let actualSize := returndatasize()
            if gt(actualSize, maxReturnSize) {
                actualSize := maxReturnSize
            }
            mstore(returnData, actualSize)
        }

        if (success) {
            emit TransactionExecuted(transactionId, msg.sender);
        } else {
            // Allow retry on failure
            txn.executed = false;

            // Extract revert reason if available (limited by maxReturnSize)
            string memory reason = "Execution failed";
            if (returnData.length > 68) {
                // Standard Error(string) has: selector(4) + offset(32) + length(32) + data
                assembly {
                    // Point to the string data (skip selector and offset)
                    let strPtr := add(returnData, 68)
                    let strLen := mload(strPtr)
                    // Cap string length to avoid issues
                    if gt(strLen, 256) {
                        strLen := 256
                    }
                    reason := strPtr
                    mstore(reason, strLen)
                }
            }

            emit TransactionFailed(transactionId, reason);
            revert(reason);
        }
    }

    /**
     * @notice Get transaction details
     * @param transactionId ID of the transaction
     */
    function getTransaction(uint256 transactionId)
        public
        view
        returns (
            address to,
            uint256 value,
            bytes memory data,
            bool executed,
            uint256 numConfirmations
        )
    {
        Transaction storage txn = transactions[transactionId];
        return (
            txn.to,
            txn.value,
            txn.data,
            txn.executed,
            txn.confirmations
        );
    }

    /**
     * @notice Get confirmation count
     * @param transactionId ID of the transaction
     */
    function getConfirmationCount(uint256 transactionId)
        public
        view
        returns (uint256 count)
    {
        uint256 ownersLength = owners.length;
        for (uint256 i = 0; i < ownersLength; i++) {
            if (confirmations[transactionId][owners[i]]) {
                count++;
            }
        }
    }

    /**
     * @notice Check if transaction is confirmed by owner
     */
    function isConfirmed(uint256 transactionId, address owner)
        public
        view
        returns (bool)
    {
        return confirmations[transactionId][owner];
    }

    /**
     * @notice Get pending transaction count
     */
    function getPendingTransactionCount() public view returns (uint256 count) {
        for (uint256 i = 0; i < transactionCount; i++) {
            if (!transactions[i].executed && transactions[i].confirmations < threshold) {
                count++;
            }
        }
    }

    /**
     * @notice Get executable transaction count
     */
    function getExecutableTransactionCount() public view returns (uint256 count) {
        for (uint256 i = 0; i < transactionCount; i++) {
            if (!transactions[i].executed && transactions[i].confirmations >= threshold) {
                count++;
            }
        }
    }
}
