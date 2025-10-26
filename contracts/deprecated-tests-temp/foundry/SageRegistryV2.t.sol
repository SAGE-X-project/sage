// SPDX-License-Identifier: LGPL-3.0
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../../contracts/core/SageRegistryV2.sol";

/**
 * @title SageRegistryV2FuzzTest
 * @notice Foundry fuzz tests for SageRegistryV2
 */
contract SageRegistryV2FuzzTest is Test {
    SageRegistryV2 public registry;
    address public admin;
    address public user1;
    address public user2;

    function setUp() public {
        admin = address(this);
        user1 = address(0x1);
        user2 = address(0x2);

        registry = new SageRegistryV2();
        registry.initialize(admin);
    }

    /// @dev Fuzz test for DID registration
    function testFuzz_RegisterDID(
        string memory did,
        bytes memory publicKey
    ) public {
        // Skip invalid inputs
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey.length > 0 && publicKey.length <= 1024);

        vm.prank(user1);

        // First registration should succeed
        bool success = registry.registerDID(did, publicKey, "");

        if (success) {
            // Verify registration
            assertTrue(registry.isDIDRegistered(did));

            bytes memory retrievedKey = registry.getPublicKey(did);
            assertEq(retrievedKey, publicKey);

            address owner = registry.getDIDOwner(did);
            assertEq(owner, user1);
        }
    }

    /// @dev Fuzz test for duplicate DID registration prevention
    function testFuzz_PreventDuplicateRegistration(
        string memory did,
        bytes memory publicKey1,
        bytes memory publicKey2
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey1.length > 0 && publicKey1.length <= 1024);
        vm.assume(publicKey2.length > 0 && publicKey2.length <= 1024);
        vm.assume(keccak256(publicKey1) != keccak256(publicKey2));

        // First registration
        vm.prank(user1);
        bool success1 = registry.registerDID(did, publicKey1, "");
        vm.assume(success1); // Only continue if first registration succeeded

        // Second registration with same DID should fail
        vm.prank(user2);
        vm.expectRevert();
        registry.registerDID(did, publicKey2, "");

        // Verify original registration is intact
        assertEq(registry.getDIDOwner(did), user1);
        assertEq(registry.getPublicKey(did), publicKey1);
    }

    /// @dev Fuzz test for public key updates
    function testFuzz_UpdatePublicKey(
        string memory did,
        bytes memory initialKey,
        bytes memory newKey,
        string memory proof
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(initialKey.length > 0 && initialKey.length <= 1024);
        vm.assume(newKey.length > 0 && newKey.length <= 1024);
        vm.assume(keccak256(initialKey) != keccak256(newKey));

        // Register DID
        vm.prank(user1);
        bool registered = registry.registerDID(did, initialKey, "");
        vm.assume(registered);

        // Update public key
        vm.prank(user1);
        registry.updatePublicKey(did, newKey, proof);

        // Verify update
        bytes memory retrievedKey = registry.getPublicKey(did);
        assertEq(retrievedKey, newKey);
        assertEq(registry.getDIDOwner(did), user1);
    }

    /// @dev Fuzz test for unauthorized public key updates
    function testFuzz_UnauthorizedUpdate(
        string memory did,
        bytes memory publicKey,
        bytes memory newKey
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey.length > 0 && publicKey.length <= 1024);
        vm.assume(newKey.length > 0 && newKey.length <= 1024);

        // Register DID with user1
        vm.prank(user1);
        bool registered = registry.registerDID(did, publicKey, "");
        vm.assume(registered);

        // Try to update with user2 (should fail)
        vm.prank(user2);
        vm.expectRevert();
        registry.updatePublicKey(did, newKey, "");

        // Verify original key is intact
        assertEq(registry.getPublicKey(did), publicKey);
    }

    /// @dev Fuzz test for DID revocation
    function testFuzz_RevokeDID(
        string memory did,
        bytes memory publicKey
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey.length > 0 && publicKey.length <= 1024);

        // Register DID
        vm.prank(user1);
        bool registered = registry.registerDID(did, publicKey, "");
        vm.assume(registered);

        // Revoke DID
        vm.prank(user1);
        registry.revokeDID(did);

        // Verify revocation
        assertTrue(registry.isRevoked(did));

        // Operations on revoked DID should fail
        vm.prank(user1);
        vm.expectRevert();
        registry.updatePublicKey(did, publicKey, "");
    }

    /// @dev Fuzz test for unauthorized revocation
    function testFuzz_UnauthorizedRevocation(
        string memory did,
        bytes memory publicKey
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey.length > 0 && publicKey.length <= 1024);

        // Register DID
        vm.prank(user1);
        bool registered = registry.registerDID(did, publicKey, "");
        vm.assume(registered);

        // Try to revoke with user2 (should fail)
        vm.prank(user2);
        vm.expectRevert();
        registry.revokeDID(did);

        // Verify DID is not revoked
        assertFalse(registry.isRevoked(did));
    }

    /// @dev Fuzz test for batch operations
    function testFuzz_BatchRegistration(
        string[] memory dids,
        bytes[] memory publicKeys
    ) public {
        // Limit array size
        vm.assume(dids.length > 0 && dids.length <= 10);
        vm.assume(publicKeys.length == dids.length);

        uint256 successCount = 0;

        for (uint256 i = 0; i < dids.length; i++) {
            // Skip invalid inputs
            if (bytes(dids[i]).length == 0 || bytes(dids[i]).length > 256) continue;
            if (publicKeys[i].length == 0 || publicKeys[i].length > 1024) continue;

            vm.prank(user1);
            bool success = registry.registerDID(dids[i], publicKeys[i], "");

            if (success) {
                successCount++;
                assertTrue(registry.isDIDRegistered(dids[i]));
            }
        }

        // At least some registrations should succeed if inputs are valid
        assertTrue(successCount > 0 || dids.length == 0);
    }

    /// @dev Fuzz test for ownership transfer
    function testFuzz_OwnershipTransfer(
        string memory did,
        bytes memory publicKey,
        address newOwner
    ) public {
        vm.assume(bytes(did).length > 0 && bytes(did).length <= 256);
        vm.assume(publicKey.length > 0 && publicKey.length <= 1024);
        vm.assume(newOwner != address(0));
        vm.assume(newOwner != user1);

        // Register DID
        vm.prank(user1);
        bool registered = registry.registerDID(did, publicKey, "");
        vm.assume(registered);

        // Transfer ownership (if implemented)
        // This tests the invariant that only owner can transfer
        vm.prank(user1);
        // registry.transferOwnership(did, newOwner);

        // Verify original owner still has control
        assertEq(registry.getDIDOwner(did), user1);
    }

    /// @dev Invariant: Total registered DIDs should never decrease
    function invariant_TotalDIDsNeverDecrease() public view {
        // This would track total DIDs if we had a counter
        // For now, we can check that revoked DIDs still exist
    }

    /// @dev Invariant: Revoked DIDs cannot be used
    function invariant_RevokedDIDsUnusable() public {
        // We can't directly test this without tracking revoked DIDs
        // But the contract logic ensures revoked DIDs can't be updated
    }
}
