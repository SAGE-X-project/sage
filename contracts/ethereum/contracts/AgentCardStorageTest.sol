// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import "./AgentCardStorage.sol";

/**
 * @title AgentCardStorageTest
 * @notice Test helper contract that exposes AgentCardStorage internals for testing
 * @dev This contract is ONLY for testing purposes and should NEVER be deployed to production
 *
 * Purpose:
 * - Exposes internal storage mappings via public getters
 * - Provides setter functions to manipulate storage for test scenarios
 * - Enables event emission testing
 * - Allows verification of constants and data structures
 *
 * TDD Methodology:
 * This contract enables the RED phase of TDD by providing a testable interface
 * to the abstract AgentCardStorage contract. It implements minimal functionality
 * to verify that the storage layer data structures are correctly defined.
 *
 * @custom:security-contact security@sage.com
 */
contract AgentCardStorageTest is AgentCardStorage {
    // ============ Getters for Internal Mappings ============

    /**
     * @notice Get agent metadata by agent ID
     * @param agentId The unique agent identifier
     * @return Agent metadata structure
     */
    function getAgentMetadata(bytes32 agentId) external view returns (AgentMetadata memory) {
        return agents[agentId];
    }

    /**
     * @notice Get agent key by key hash
     * @param keyHash The hash of the public key
     * @return Agent key structure
     */
    function getAgentKey(bytes32 keyHash) external view returns (AgentKey memory) {
        return agentKeys[keyHash];
    }

    /**
     * @notice Get registration commitment for an address
     * @param committer The address that made the commitment
     * @return Registration commitment structure
     */
    function getRegistrationCommitment(address committer) external view returns (RegistrationCommitment memory) {
        return registrationCommitments[committer];
    }

    /**
     * @notice Get agent ID by DID
     * @param did The W3C DID string
     * @return Agent ID (bytes32 hash)
     */
    function getAgentIdByDid(string calldata did) external view returns (bytes32) {
        return didToAgentId[did];
    }

    /**
     * @notice Get all agent IDs owned by an address
     * @param owner The owner address
     * @return Array of agent IDs
     */
    function getAgentsByOwner(address owner) external view returns (bytes32[] memory) {
        return ownerToAgents[owner];
    }

    /**
     * @notice Get current nonce for an agent
     * @param agentId The agent identifier
     * @return Current nonce value
     */
    function getAgentNonce(bytes32 agentId) external view returns (uint256) {
        return agentNonce[agentId];
    }

    /**
     * @notice Get daily registration count for an address
     * @param addr The address to check
     * @return Number of registrations today
     */
    function getDailyRegistrationCount(address addr) external view returns (uint256) {
        return dailyRegistrationCount[addr];
    }

    /**
     * @notice Get last registration day for an address
     * @param addr The address to check
     * @return Day number (block.timestamp / 1 days)
     */
    function getLastRegistrationDay(address addr) external view returns (uint256) {
        return lastRegistrationDay[addr];
    }

    /**
     * @notice Check if a public key has been used
     * @param keyHash The hash of the public key
     * @return True if key has been used
     */
    function isPublicKeyUsed(bytes32 keyHash) external view returns (bool) {
        return publicKeyUsed[keyHash];
    }

    // ============ Getters for Constants ============

    /**
     * @notice Get COMMIT_MIN_DELAY constant
     * @return Minimum delay between commit and reveal (seconds)
     */
    function getCommitMinDelay() external pure returns (uint256) {
        return COMMIT_MIN_DELAY;
    }

    /**
     * @notice Get COMMIT_MAX_DELAY constant
     * @return Maximum delay between commit and reveal (seconds)
     */
    function getCommitMaxDelay() external pure returns (uint256) {
        return COMMIT_MAX_DELAY;
    }

    /**
     * @notice Get MAX_KEYS_PER_AGENT constant
     * @return Maximum number of keys per agent
     */
    function getMaxKeysPerAgent() external pure returns (uint256) {
        return MAX_KEYS_PER_AGENT;
    }

    /**
     * @notice Get MAX_DAILY_REGISTRATIONS constant
     * @return Maximum daily registrations per address
     */
    function getMaxDailyRegistrations() external pure returns (uint256) {
        return MAX_DAILY_REGISTRATIONS;
    }

    // ============ Setters for Testing ============

    /**
     * @notice Set agent metadata (for testing)
     * @param agentId The agent identifier
     * @param metadata The metadata to store
     */
    function setAgentMetadata(bytes32 agentId, AgentMetadata calldata metadata) external {
        agents[agentId] = metadata;
    }

    /**
     * @notice Set agent key (for testing)
     * @param keyHash The key hash
     * @param key The key data to store
     */
    function setAgentKey(bytes32 keyHash, AgentKey calldata key) external {
        agentKeys[keyHash] = key;
    }

    /**
     * @notice Set registration commitment (for testing)
     * @param committer The committer address
     * @param commitment The commitment data to store
     */
    function setRegistrationCommitment(address committer, RegistrationCommitment calldata commitment) external {
        registrationCommitments[committer] = commitment;
    }

    /**
     * @notice Set DID to agent ID mapping (for testing)
     * @param did The DID string
     * @param agentId The agent ID
     */
    function setDidToAgentId(string calldata did, bytes32 agentId) external {
        didToAgentId[did] = agentId;
    }

    /**
     * @notice Add agent to owner's list (for testing)
     * @param owner The owner address
     * @param agentId The agent ID to add
     */
    function addAgentToOwner(address owner, bytes32 agentId) external {
        ownerToAgents[owner].push(agentId);
    }

    /**
     * @notice Increment nonce for an agent (for testing)
     * @param agentId The agent identifier
     */
    function incrementNonce(bytes32 agentId) external {
        agentNonce[agentId]++;
    }

    /**
     * @notice Increment daily registration count (for testing)
     * @param addr The address to increment
     */
    function incrementDailyRegistrationCount(address addr) external {
        dailyRegistrationCount[addr]++;
    }

    /**
     * @notice Set last registration day (for testing)
     * @param addr The address to set
     * @param day The day number
     */
    function setLastRegistrationDay(address addr, uint256 day) external {
        lastRegistrationDay[addr] = day;
    }

    /**
     * @notice Mark public key as used (for testing)
     * @param keyHash The key hash to mark
     */
    function markPublicKeyUsed(bytes32 keyHash) external {
        publicKeyUsed[keyHash] = true;
    }

    // ============ Event Emitters for Testing ============

    /**
     * @notice Emit AgentRegistered event (for testing)
     * @param agentId The agent identifier
     * @param did The DID string
     * @param owner The owner address
     * @param timestamp The timestamp
     */
    function emitAgentRegistered(
        bytes32 agentId,
        string calldata did,
        address owner,
        uint256 timestamp
    ) external {
        emit AgentRegistered(agentId, did, owner, timestamp);
    }

    /**
     * @notice Emit KeyAdded event (for testing)
     * @param agentId The agent identifier
     * @param keyHash The key hash
     * @param keyType The key type
     * @param timestamp The timestamp
     */
    function emitKeyAdded(
        bytes32 agentId,
        bytes32 keyHash,
        KeyType keyType,
        uint256 timestamp
    ) external {
        emit KeyAdded(agentId, keyHash, keyType, timestamp);
    }

    /**
     * @notice Emit KeyRevoked event (for testing)
     * @param agentId The agent identifier
     * @param keyHash The key hash
     * @param timestamp The timestamp
     */
    function emitKeyRevoked(
        bytes32 agentId,
        bytes32 keyHash,
        uint256 timestamp
    ) external {
        emit KeyRevoked(agentId, keyHash, timestamp);
    }

    /**
     * @notice Emit AgentUpdated event (for testing)
     * @param agentId The agent identifier
     * @param timestamp The timestamp
     */
    function emitAgentUpdated(bytes32 agentId, uint256 timestamp) external {
        emit AgentUpdated(agentId, timestamp);
    }

    /**
     * @notice Emit AgentDeactivatedByHash event (for testing)
     * @param agentId The agent identifier
     * @param timestamp The timestamp
     */
    function emitAgentDeactivatedByHash(bytes32 agentId, uint256 timestamp) external {
        emit AgentDeactivatedByHash(agentId, timestamp);
    }

    /**
     * @notice Emit CommitmentRecorded event (for testing)
     * @param committer The committer address
     * @param commitHash The commitment hash
     * @param timestamp The timestamp
     */
    function emitCommitmentRecorded(
        address committer,
        bytes32 commitHash,
        uint256 timestamp
    ) external {
        emit CommitmentRecorded(committer, commitHash, timestamp);
    }
}
