// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./SageRegistryV2.sol";

/**
 * @title SageRegistryTest
 * @notice Test version of SageRegistryV2 that bypasses key validation for local testing
 * @dev DO NOT DEPLOY THIS CONTRACT TO PRODUCTION
 */
contract SageRegistryTest is SageRegistryV2 {
    
    bool public testMode = true;
    
    /**
     * @notice Toggle test mode on/off
     * @param _testMode Enable or disable test mode
     */
    function setTestMode(bool _testMode) external onlyOwner {
        testMode = _testMode;
    }
    
    /**
     * @notice Helper function to recover signer from signature
     */
    function recoverSigner(bytes32 hash, bytes memory signature) internal pure returns (address) {
        require(signature.length == 65, "Invalid signature length");
        
        bytes32 r;
        bytes32 s;
        uint8 v;
        
        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }
        
        if (v < 27) {
            v += 27;
        }
        
        require(v == 27 || v == 28, "Invalid signature recovery");
        return ecrecover(hash, v, r, s);
    }
    
    /**
     * @notice Override key validation to allow testing without real key pairs
     * @dev In test mode, accepts any signature from the sender
     */
    function _validatePublicKey(
        bytes calldata publicKey,
        bytes calldata signature
    ) internal override {
        if (!testMode) {
            // If not in test mode, use regular validation
            super._validatePublicKey(publicKey, signature);
            return;
        }
        
        // In test mode, just validate basic requirements
        require(publicKey.length == 65, "Invalid public key length");
        require(publicKey[0] == 0x04, "Invalid public key format");
        require(signature.length > 0, "Signature required");
        
        // Create a simplified validation that just checks msg.sender signed something
        bytes32 keyHash = keccak256(publicKey);
        bytes32 messageHash = keccak256(abi.encodePacked("TEST_MODE:", msg.sender, keyHash));
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );
        
        address recovered = recoverSigner(ethSignedHash, signature);
        require(recovered == msg.sender, "Invalid test signature");
        
        // Store simplified validation data
        // Just emit the event for test purposes
        emit KeyValidated(keyHash, msg.sender);
    }
}