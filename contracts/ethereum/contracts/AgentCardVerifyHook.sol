// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title AgentCardVerifyHook
 * @notice External verification hook for agent registration validation
 * @dev Implements advanced security checks and anti-fraud detection
 *
 * This contract provides pre-registration validation to prevent:
 * - Invalid DID formats (W3C non-compliance)
 * - Sybil attacks (rate limiting)
 * - Malicious actors (blacklist)
 * - Public key reuse (key tracking)
 * - Mass registration spam (daily limits)
 *
 * Key Features:
 * - DID format validation (did:sage:ethereum:0x...)
 * - Rate limiting (24 registrations per day per address)
 * - Blacklist/whitelist management
 * - Public key reuse prevention across different owners
 * - Owner-only administration
 *
 * @custom:security-contact security@sage.com
 */
contract AgentCardVerifyHook is Ownable2Step {
    // ============ State Variables ============

    /**
     * @notice Daily registration tracking
     * @dev Maps address → day number → count
     */
    mapping(address => uint256) public dailyRegistrationCount;
    mapping(address => uint256) public lastRegistrationDay;

    /**
     * @notice Maximum registrations per address per day
     * @dev Aligned with AgentCardStorage.MAX_DAILY_REGISTRATIONS
     */
    uint256 public constant MAX_REGISTRATIONS_PER_DAY = 24;

    /**
     * @notice Blacklist mapping
     * @dev Blacklisted addresses cannot register agents
     */
    mapping(address => bool) public blacklisted;

    /**
     * @notice Whitelist mapping
     * @dev Whitelisted addresses bypass rate limiting
     */
    mapping(address => bool) public whitelisted;

    /**
     * @notice Public key ownership tracking
     * @dev Maps key hash → owner address
     *      Prevents key reuse across different agents
     */
    mapping(bytes32 => address) public keyToOwner;

    // ============ Events ============

    /**
     * @notice Emitted when an address is added to blacklist
     * @param addr The blacklisted address
     */
    event AddressBlacklisted(address indexed addr);

    /**
     * @notice Emitted when an address is added to whitelist
     * @param addr The whitelisted address
     */
    event AddressWhitelisted(address indexed addr);

    /**
     * @notice Emitted when rate limit is exceeded
     * @param addr The address that exceeded the limit
     * @param count Current registration count
     */
    event RateLimitExceeded(address indexed addr, uint256 count);

    // ============ Constructor ============

    constructor() Ownable(msg.sender) {
    }

    // ============ Main Validation Function ============

    /**
     * @notice Pre-registration validation hook
     * @dev Called by AgentCardRegistry before registration
     *
     * Validation Steps:
     * 1. Check blacklist
     * 2. Validate DID format
     * 3. Check rate limiting (unless whitelisted)
     * 4. Validate public keys (no reuse)
     *
     * @param did DID identifier (e.g., "did:sage:ethereum:0x...")
     * @param owner Registration owner address
     * @param keys Array of public keys
     */
    function beforeRegister(
        string calldata did,
        address owner,
        bytes[] calldata keys
    ) external view {
        // 1. Check blacklist
        require(!blacklisted[owner], "Address blacklisted");

        // 2. Validate DID format
        _validateDIDFormat(did, owner);

        // 3. Check rate limiting (unless whitelisted)
        if (!whitelisted[owner]) {
            _checkRateLimit(owner);
        }

        // 4. Validate keys (no reuse across different owners)
        for (uint256 i = 0; i < keys.length; i++) {
            bytes32 keyHash = keccak256(keys[i]);
            address keyOwner = keyToOwner[keyHash];

            // Allow if key is new or belongs to same owner
            require(
                keyOwner == address(0) || keyOwner == owner,
                "Key already used by another agent"
            );
        }
    }

    // ============ Internal Validation Functions ============

    /**
     * @notice Validate W3C DID format
     * @dev Format: did:sage:{chain}:0x{address}
     *
     * Requirements:
     * - Minimum length: 21 characters
     * - Prefix: "did:sage:"
     * - Chain identifier (ethereum, sepolia, mainnet, etc.)
     * - Address component (optional but recommended)
     *
     * @param did DID identifier
     * @param owner Owner address (for future address validation)
     */
    function _validateDIDFormat(string calldata did, address owner) internal pure {
        bytes memory didBytes = bytes(did);

        // Check prefix "did:sage:" first (before length check)
        require(
            didBytes.length >= 9 &&
            didBytes[0] == 0x64 && // "d"
            didBytes[1] == 0x69 && // "i"
            didBytes[2] == 0x64 && // "d"
            didBytes[3] == 0x3a && // ":"
            didBytes[4] == 0x73 && // "s"
            didBytes[5] == 0x61 && // "a"
            didBytes[6] == 0x67 && // "g"
            didBytes[7] == 0x65 && // "e"
            didBytes[8] == 0x3a,   // ":"
            "Invalid DID prefix"
        );

        // Then check overall length
        require(didBytes.length > 20, "DID too short");

        // Additional validation could include:
        // - Chain identifier verification
        // - Address embedding verification
        // - Character set validation (alphanumeric + :-)
        // For now, we validate the prefix which is the critical part

        // Suppress unused variable warning
        owner; // Future use: validate owner address embedding in DID
    }

    /**
     * @notice Check rate limiting
     * @dev Enforces MAX_REGISTRATIONS_PER_DAY limit
     *
     * Rate Limit Logic:
     * - Day is calculated as: block.timestamp / 1 days
     * - Count resets when day changes
     * - Each address tracked independently
     *
     * @param owner Address to check
     */
    function _checkRateLimit(address owner) internal view {
        uint256 currentDay = block.timestamp / 1 days;
        uint256 lastDay = lastRegistrationDay[owner];

        // If same day, check count
        if (currentDay == lastDay) {
            uint256 count = dailyRegistrationCount[owner];
            require(
                count < MAX_REGISTRATIONS_PER_DAY,
                "Rate limit exceeded"
            );
        }
        // If different day, count resets automatically
    }

    // ============ Admin Functions ============

    /**
     * @notice Add address to blacklist
     * @dev Only owner can call
     * @param addr Address to blacklist
     */
    function addToBlacklist(address addr) external onlyOwner {
        blacklisted[addr] = true;
        emit AddressBlacklisted(addr);
    }

    /**
     * @notice Remove address from blacklist
     * @dev Only owner can call
     * @param addr Address to remove
     */
    function removeFromBlacklist(address addr) external onlyOwner {
        blacklisted[addr] = false;
    }

    /**
     * @notice Add address to whitelist
     * @dev Only owner can call
     *      Whitelisted addresses bypass rate limiting
     * @param addr Address to whitelist
     */
    function addToWhitelist(address addr) external onlyOwner {
        whitelisted[addr] = true;
        emit AddressWhitelisted(addr);
    }

    /**
     * @notice Remove address from whitelist
     * @dev Only owner can call
     * @param addr Address to remove
     */
    function removeFromWhitelist(address addr) external onlyOwner {
        whitelisted[addr] = false;
    }

    // ============ Helper Functions (For Testing) ============

    /**
     * @notice Mark a key as used by an owner
     * @dev This is a test helper function
     *      In production, this would be called by AgentCardRegistry
     *      after successful registration
     *
     * @param keyHash Hash of the public key
     * @param owner Owner address
     */
    function markKeyUsed(bytes32 keyHash, address owner) external {
        keyToOwner[keyHash] = owner;

        // Update rate limiting
        uint256 currentDay = block.timestamp / 1 days;
        uint256 lastDay = lastRegistrationDay[owner];

        if (currentDay == lastDay) {
            // Same day - increment count
            dailyRegistrationCount[owner]++;
        } else {
            // New day - reset count
            dailyRegistrationCount[owner] = 1;
            lastRegistrationDay[owner] = currentDay;
        }
    }
}
