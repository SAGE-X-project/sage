// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/IRegistryHook.sol";

/**
 * @title SageVerificationHook
 * @notice Example hook implementation for additional verification
 */
contract SageVerificationHook is IRegistryHook {
    // Mapping to track registration attempts per address
    mapping(address => uint256) public registrationAttempts;
    mapping(address => uint256) public lastRegistrationTime;
    
    // Configuration
    uint256 public constant MAX_REGISTRATIONS_PER_DAY = 5;
    uint256 public constant REGISTRATION_COOLDOWN = 1 minutes;
    
    // Blacklist for malicious actors
    mapping(address => bool) public blacklisted;

    // Owner for blacklist management
    address public immutable owner;
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }
    
    constructor() {
        owner = msg.sender;
    }
    
    /**
     * @notice Pre-registration verification
     */
    function beforeRegister(
        bytes32, // agentId - unused but required by interface
        address agentOwner,
        bytes calldata data
    ) external override returns (bool success, string memory reason) {
        // Check if blacklisted
        if (blacklisted[agentOwner]) {
            return (false, "Address blacklisted");
        }
        
        // Check registration cooldown
        if (block.timestamp < lastRegistrationTime[agentOwner] + REGISTRATION_COOLDOWN) {
            return (false, "Registration cooldown active");
        }
        
        // Check daily limit
        if (_isNewDay(agentOwner)) {
            registrationAttempts[agentOwner] = 0;
        }
        
        if (registrationAttempts[agentOwner] >= MAX_REGISTRATIONS_PER_DAY) {
            return (false, "Daily registration limit reached");
        }
        
        // Decode and verify DID format
        (string memory did, ) = abi.decode(data, (string, bytes));
        if (!_isValidDID(did)) {
            return (false, "Invalid DID format");
        }
        
        return (true, "");
    }
    
    /**
     * @notice Post-registration actions
     */
    function afterRegister(
        bytes32, // agentId - unused but required by interface
        address agentOwner,
        bytes calldata // data - unused but required by interface
    ) external override {
        registrationAttempts[agentOwner]++;
        lastRegistrationTime[agentOwner] = block.timestamp;
        
        // Could emit events for monitoring
        // Could integrate with external verification services
        // Could trigger automated testing of the agent
    }
    
    /**
     * @notice Add address to blacklist
     */
    function addToBlacklist(address account) external onlyOwner {
        blacklisted[account] = true;
    }
    
    /**
     * @notice Remove address from blacklist
     */
    function removeFromBlacklist(address account) external onlyOwner {
        blacklisted[account] = false;
    }
    
    /**
     * @notice Check if it's a new day for the user
     */
    function _isNewDay(address user) private view returns (bool) {
        return block.timestamp / 1 days > lastRegistrationTime[user] / 1 days;
    }
    
    /**
     * @notice Validate DID format (basic check)
     */
    function _isValidDID(string memory did) private pure returns (bool) {
        bytes memory didBytes = bytes(did);
        if (didBytes.length < 10) return false;
        
        // Check if starts with "did:"
        if (didBytes[0] != 0x64 || didBytes[1] != 0x69 || didBytes[2] != 0x64 || didBytes[3] != 0x3A) {
            return false;
        }
        
        return true;
    }
}