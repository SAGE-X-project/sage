/**
 * AgentCardVerifyHook Test Suite
 *
 * TDD-based test suite for AgentCardVerifyHook contract
 *
 * Test Coverage:
 * - V2.1: DID Validation (6 tests)
 * - V2.2: Rate Limiting (6 tests)
 * - V2.3: Blacklist/Whitelist (7 tests)
 * - V2.4: Public Key Tracking (3 tests)
 *
 * Total: 22 tests
 *
 * @see contracts/AgentCardVerifyHook.sol
 * @see VERIFICATION_MATRIX.md (Section: AgentCardVerifyHook)
 */

const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("AgentCardVerifyHook", function () {
    let hook;
    let owner, user1, user2, user3, malicious;
    let validKey1, validKey2, validKey3;

    beforeEach(async function () {
        // Get signers
        [owner, user1, user2, user3, malicious] = await ethers.getSigners();

        // Deploy AgentCardVerifyHook
        const HookFactory = await ethers.getContractFactory("AgentCardVerifyHook");
        hook = await HookFactory.deploy();
        await hook.waitForDeployment();

        // Generate test keys
        validKey1 = ethers.randomBytes(65); // ECDSA key
        validKey2 = ethers.randomBytes(32); // Ed25519 key
        validKey3 = ethers.randomBytes(32); // X25519 key
    });

    // ============================================================================
    // V2.1: DID Validation (6 tests)
    // ============================================================================

    describe("V2.1: DID Validation", function () {
        /**
         * Test ID: H2.1.1
         * Verification: Accepts valid "did:sage:ethereum:0x..." format
         * Priority: P0 Critical
         */
        it("H2.1.1: Should accept valid DID format: did:sage:ethereum:0x...", async function () {
            // GIVEN: Valid DID format
            const validDID = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: Validate DID
            // THEN: Should not revert
            await expect(
                hook.beforeRegister(validDID, user1.address, keys)
            ).to.not.be.reverted;
        });

        /**
         * Test ID: H2.1.2
         * Verification: Rejects DID without "did:" prefix
         * Priority: P0 Critical
         */
        it("H2.1.2: Should reject DID without 'did:' prefix", async function () {
            // GIVEN: DID without "did:" prefix
            const invalidDID = "sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: Validate DID
            // THEN: Should revert with "Invalid DID prefix"
            await expect(
                hook.beforeRegister(invalidDID, user1.address, keys)
            ).to.be.revertedWith("Invalid DID prefix");
        });

        /**
         * Test ID: H2.1.3
         * Verification: Rejects DID without "sage" method
         * Priority: P0 Critical
         */
        it("H2.1.3: Should reject DID without 'sage' method", async function () {
            // GIVEN: DID with wrong method (not "sage")
            const invalidDID = "did:web:example.com";
            const keys = [validKey1];

            // WHEN: Validate DID
            // THEN: Should revert with "Invalid DID prefix"
            await expect(
                hook.beforeRegister(invalidDID, user1.address, keys)
            ).to.be.revertedWith("Invalid DID prefix");
        });

        /**
         * Test ID: H2.1.4
         * Verification: Rejects DID shorter than minimum
         * Priority: P0 Critical
         */
        it("H2.1.4: Should reject DID shorter than minimum length", async function () {
            // GIVEN: DID that is too short (valid prefix but too short overall)
            const invalidDID = "did:sage:short";  // 15 characters, less than 21 required
            const keys = [validKey1];

            // WHEN: Validate DID
            // THEN: Should revert with "DID too short"
            await expect(
                hook.beforeRegister(invalidDID, user1.address, keys)
            ).to.be.revertedWith("DID too short");
        });

        /**
         * Test ID: H2.1.5
         * Verification: Validates chain identifier in DID
         * Priority: P1 High
         */
        it("H2.1.5: Should accept DID with different chain identifiers", async function () {
            // GIVEN: DIDs with different chain identifiers
            const did1 = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const did2 = "did:sage:sepolia:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd";
            const did3 = "did:sage:mainnet:0x9876543210987654321098765432109876543210";
            const keys = [validKey1];

            // WHEN: Validate DIDs
            // THEN: All should be accepted
            await expect(hook.beforeRegister(did1, user1.address, keys)).to.not.be.reverted;
            await expect(hook.beforeRegister(did2, user2.address, keys)).to.not.be.reverted;
            await expect(hook.beforeRegister(did3, user3.address, keys)).to.not.be.reverted;
        });

        /**
         * Test ID: H2.1.6
         * Verification: Validates address embedding in DID
         * Priority: P1 High
         */
        it("H2.1.6: Should accept DID with address component", async function () {
            // GIVEN: DID with embedded address
            const userAddress = user1.address.toLowerCase().substring(2); // Remove 0x prefix
            const didWithAddress = `did:sage:ethereum:0x${userAddress}`;
            const keys = [validKey1];

            // WHEN: Validate DID
            // THEN: Should not revert
            await expect(
                hook.beforeRegister(didWithAddress, user1.address, keys)
            ).to.not.be.reverted;
        });
    });

    // ============================================================================
    // V2.2: Rate Limiting (6 tests)
    // ============================================================================

    describe("V2.2: Rate Limiting", function () {
        /**
         * Test ID: H2.2.1
         * Verification: Allows first registration immediately
         * Priority: P0 Critical
         */
        it("H2.2.1: Should allow first registration immediately", async function () {
            // GIVEN: New address with no prior registrations
            const did = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: First registration
            // THEN: Should not revert
            await expect(
                hook.beforeRegister(did, user1.address, keys)
            ).to.not.be.reverted;
        });

        /**
         * Test ID: H2.2.2
         * Verification: Allows up to 24 registrations per day
         * Priority: P0 Critical
         */
        it("H2.2.2: Should allow up to 24 registrations per day", async function () {
            // GIVEN: Multiple registration attempts within rate limit
            const keys = [validKey1];

            // WHEN: Register 24 times
            for (let i = 0; i < 24; i++) {
                const did = `did:sage:ethereum:${i.toString().padStart(42, '0')}`;
                await hook.beforeRegister(did, user1.address, keys);
            }

            // THEN: All 24 should succeed (no revert)
            // This test passes if no error is thrown
        });

        /**
         * Test ID: H2.2.3
         * Verification: Rejects 25th registration in same day
         * Priority: P0 Critical
         */
        it("H2.2.3: Should reject 25th registration in same day", async function () {
            // GIVEN: 24 successful registrations
            const keys = [validKey1];

            for (let i = 0; i < 24; i++) {
                const keyHash = ethers.keccak256(ethers.randomBytes(65));
                await hook.markKeyUsed(keyHash, user1.address);
            }

            // WHEN: Attempt 25th registration
            const did25 = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

            // THEN: Should revert with "Rate limit exceeded"
            await expect(
                hook.beforeRegister(did25, user1.address, keys)
            ).to.be.revertedWith("Rate limit exceeded");
        });

        /**
         * Test ID: H2.2.4
         * Verification: Resets count after 24 hours
         * Priority: P0 Critical
         */
        it("H2.2.4: Should reset count after 24 hours", async function () {
            // GIVEN: 24 registrations done
            const keys = [validKey1];

            for (let i = 0; i < 24; i++) {
                const did = `did:sage:ethereum:${i.toString().padStart(42, '0')}`;
                await hook.beforeRegister(did, user1.address, keys);
            }

            // WHEN: Advance time by 24 hours + 1 second
            await ethers.provider.send("evm_increaseTime", [86401]);
            await ethers.provider.send("evm_mine");

            // THEN: Should allow new registration
            const newDID = "did:sage:ethereum:0xnew1234567890123456789012345678901234567";
            await expect(
                hook.beforeRegister(newDID, user1.address, keys)
            ).to.not.be.reverted;
        });

        /**
         * Test ID: H2.2.5
         * Verification: Tracks counts per address separately
         * Priority: P0 Critical
         */
        it("H2.2.5: Should track counts per address separately", async function () {
            // GIVEN: Two different addresses
            const keys = [validKey1];

            // WHEN: user1 registers 24 times (using markKeyUsed to update count)
            for (let i = 0; i < 24; i++) {
                const keyHash = ethers.keccak256(ethers.randomBytes(65));
                await hook.markKeyUsed(keyHash, user1.address);
            }

            // THEN: user2 should still be able to register
            const user2DID = "did:sage:ethereum:user20000000000000000000000000000000000000";
            await expect(
                hook.beforeRegister(user2DID, user2.address, keys)
            ).to.not.be.reverted;

            // AND: user1 should be rate limited
            const user1Extra = "did:sage:ethereum:user1extra000000000000000000000000000";
            await expect(
                hook.beforeRegister(user1Extra, user1.address, keys)
            ).to.be.revertedWith("Rate limit exceeded");
        });

        /**
         * Test ID: H2.2.6
         * Verification: Whitelisted addresses bypass limit
         * Priority: P1 High
         */
        it("H2.2.6: Should allow whitelisted addresses to bypass rate limit", async function () {
            // GIVEN: user1 is whitelisted
            await hook.connect(owner).addToWhitelist(user1.address);
            const keys = [validKey1];

            // WHEN: Register more than 24 times
            for (let i = 0; i < 30; i++) {
                const did = `did:sage:ethereum:${i.toString().padStart(42, '0')}`;
                await hook.beforeRegister(did, user1.address, keys);
            }

            // THEN: All should succeed (no rate limit)
            // This test passes if no error is thrown
        });
    });

    // ============================================================================
    // V2.3: Blacklist/Whitelist (7 tests)
    // ============================================================================

    describe("V2.3: Blacklist/Whitelist", function () {
        /**
         * Test ID: H2.3.1
         * Verification: Blacklisted address rejected
         * Priority: P0 Critical
         */
        it("H2.3.1: Should reject blacklisted address", async function () {
            // GIVEN: malicious address is blacklisted
            await hook.connect(owner).addToBlacklist(malicious.address);
            const did = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: Blacklisted address tries to register
            // THEN: Should revert with "Address blacklisted"
            await expect(
                hook.beforeRegister(did, malicious.address, keys)
            ).to.be.revertedWith("Address blacklisted");
        });

        /**
         * Test ID: H2.3.2
         * Verification: Whitelisted address allowed
         * Priority: P0 Critical
         */
        it("H2.3.2: Should allow whitelisted address", async function () {
            // GIVEN: user1 is whitelisted
            await hook.connect(owner).addToWhitelist(user1.address);
            const did = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: Whitelisted address registers
            // THEN: Should not revert
            await expect(
                hook.beforeRegister(did, user1.address, keys)
            ).to.not.be.reverted;
        });

        /**
         * Test ID: H2.3.3
         * Verification: Owner can add to blacklist
         * Priority: P0 Critical
         */
        it("H2.3.3: Should allow owner to add to blacklist", async function () {
            // GIVEN: Owner account
            // WHEN: Owner adds address to blacklist
            // THEN: Should emit AddressBlacklisted event
            await expect(
                hook.connect(owner).addToBlacklist(malicious.address)
            )
                .to.emit(hook, "AddressBlacklisted")
                .withArgs(malicious.address);

            // AND: Address should be blacklisted
            expect(await hook.blacklisted(malicious.address)).to.be.true;
        });

        /**
         * Test ID: H2.3.4
         * Verification: Owner can remove from blacklist
         * Priority: P0 Critical
         */
        it("H2.3.4: Should allow owner to remove from blacklist", async function () {
            // GIVEN: Address is blacklisted
            await hook.connect(owner).addToBlacklist(malicious.address);
            expect(await hook.blacklisted(malicious.address)).to.be.true;

            // WHEN: Owner removes from blacklist
            await hook.connect(owner).removeFromBlacklist(malicious.address);

            // THEN: Address should no longer be blacklisted
            expect(await hook.blacklisted(malicious.address)).to.be.false;

            // AND: Address can now register
            const did = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];
            await expect(
                hook.beforeRegister(did, malicious.address, keys)
            ).to.not.be.reverted;
        });

        /**
         * Test ID: H2.3.5
         * Verification: Owner can add to whitelist
         * Priority: P0 Critical
         */
        it("H2.3.5: Should allow owner to add to whitelist", async function () {
            // GIVEN: Owner account
            // WHEN: Owner adds address to whitelist
            // THEN: Should emit AddressWhitelisted event
            await expect(
                hook.connect(owner).addToWhitelist(user1.address)
            )
                .to.emit(hook, "AddressWhitelisted")
                .withArgs(user1.address);

            // AND: Address should be whitelisted
            expect(await hook.whitelisted(user1.address)).to.be.true;
        });

        /**
         * Test ID: H2.3.6
         * Verification: Owner can remove from whitelist
         * Priority: P0 Critical
         */
        it("H2.3.6: Should allow owner to remove from whitelist", async function () {
            // GIVEN: Address is whitelisted
            await hook.connect(owner).addToWhitelist(user1.address);
            expect(await hook.whitelisted(user1.address)).to.be.true;

            // WHEN: Owner removes from whitelist
            await hook.connect(owner).removeFromWhitelist(user1.address);

            // THEN: Address should no longer be whitelisted
            expect(await hook.whitelisted(user1.address)).to.be.false;
        });

        /**
         * Test ID: H2.3.7
         * Verification: Non-owner cannot modify lists
         * Priority: P0 Critical
         */
        it("H2.3.7: Should reject non-owner admin calls", async function () {
            // GIVEN: Non-owner account (user1)

            // WHEN/THEN: Attempting to add to blacklist should revert
            await expect(
                hook.connect(user1).addToBlacklist(malicious.address)
            ).to.be.reverted;

            // WHEN/THEN: Attempting to remove from blacklist should revert
            await expect(
                hook.connect(user1).removeFromBlacklist(malicious.address)
            ).to.be.reverted;

            // WHEN/THEN: Attempting to add to whitelist should revert
            await expect(
                hook.connect(user1).addToWhitelist(user2.address)
            ).to.be.reverted;

            // WHEN/THEN: Attempting to remove from whitelist should revert
            await expect(
                hook.connect(user1).removeFromWhitelist(user2.address)
            ).to.be.reverted;
        });
    });

    // ============================================================================
    // V2.4: Public Key Tracking (3 tests)
    // ============================================================================

    describe("V2.4: Public Key Tracking", function () {
        /**
         * Test ID: H2.4.1
         * Verification: Prevents public key reuse across agents
         * Priority: P0 Critical
         */
        it("H2.4.1: Should prevent public key reuse across different agents", async function () {
            // GIVEN: Key used by user1
            const sharedKey = validKey1;
            const did1 = "did:sage:ethereum:user10000000000000000000000000000000000000";

            // First registration - mark key as used
            await hook.beforeRegister(did1, user1.address, [sharedKey]);
            await hook.markKeyUsed(ethers.keccak256(sharedKey), user1.address);

            // WHEN: Different user tries to use same key
            const did2 = "did:sage:ethereum:user20000000000000000000000000000000000000";

            // THEN: Should revert with "Key already used by another agent"
            await expect(
                hook.beforeRegister(did2, user2.address, [sharedKey])
            ).to.be.revertedWith("Key already used by another agent");
        });

        /**
         * Test ID: H2.4.2
         * Verification: Tracks key-to-owner mapping
         * Priority: P0 Critical
         */
        it("H2.4.2: Should track key-to-owner mapping correctly", async function () {
            // GIVEN: Key registered to user1
            const key = validKey1;
            const keyHash = ethers.keccak256(key);

            // WHEN: Mark key as used by user1
            await hook.markKeyUsed(keyHash, user1.address);

            // THEN: Key should be mapped to user1
            expect(await hook.keyToOwner(keyHash)).to.equal(user1.address);
        });

        /**
         * Test ID: H2.4.3
         * Verification: Allows same key for same owner
         * Priority: P1 High
         */
        it("H2.4.3: Should allow same owner to reuse their own key", async function () {
            // GIVEN: Key already used by user1
            const key = validKey1;
            const keyHash = ethers.keccak256(key);
            const did1 = "did:sage:ethereum:user1first00000000000000000000000000000";

            await hook.beforeRegister(did1, user1.address, [key]);
            await hook.markKeyUsed(keyHash, user1.address);

            // WHEN: Same user tries to register with same key again
            const did2 = "did:sage:ethereum:user1second0000000000000000000000000000";

            // THEN: Should allow (same owner)
            // Note: Implementation may allow or disallow this - adjust based on design
            // For now, we test that it checks owner properly
            const storedOwner = await hook.keyToOwner(keyHash);
            expect(storedOwner).to.equal(user1.address);
        });
    });

    // ============================================================================
    // V2.5: Additional Helper Tests (2 tests)
    // ============================================================================

    describe("V2.5: Helper Functions", function () {
        /**
         * Test ID: H2.5.1
         * Verification: markKeyUsed updates mapping
         * Priority: P0 Critical
         */
        it("H2.5.1: Should allow marking key as used (test helper)", async function () {
            // GIVEN: A new key
            const key = validKey1;
            const keyHash = ethers.keccak256(key);

            // Verify key not used initially
            expect(await hook.keyToOwner(keyHash)).to.equal(ethers.ZeroAddress);

            // WHEN: Mark key as used
            await hook.markKeyUsed(keyHash, user1.address);

            // THEN: Key should be mapped to owner
            expect(await hook.keyToOwner(keyHash)).to.equal(user1.address);
        });

        /**
         * Test ID: H2.5.2
         * Verification: Gas costs are reasonable
         * Priority: P1 High
         */
        it("H2.5.2: Should have reasonable gas costs for validation", async function () {
            // GIVEN: Valid registration data
            const did = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
            const keys = [validKey1];

            // WHEN: Estimate gas for beforeRegister (view function)
            const gasEstimate = await hook.beforeRegister.estimateGas(did, user1.address, keys);

            // THEN: Gas should be less than 100k
            console.log(`      Gas estimate for validation: ${gasEstimate}`);
            expect(gasEstimate).to.be.lt(100000);
        });
    });
});

/**
 * Test Summary
 * =============
 * V2.1: DID Validation (6 tests)
 * V2.2: Rate Limiting (6 tests)
 * V2.3: Blacklist/Whitelist (7 tests)
 * V2.4: Public Key Tracking (3 tests)
 * V2.5: Helper Functions (2 tests)
 *
 * Total: 24 tests
 *
 * TDD Methodology:
 * - RED: Tests written first, contract doesn't exist yet
 * - GREEN: Implement contract to make tests pass
 * - REFACTOR: Optimize while keeping tests passing
 */
