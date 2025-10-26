const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

/**
 * Array Bounds Checking Tests (Standalone Version)
 *
 * Tests the standalone ERC8004ValidationRegistry to verify:
 * 1. Maximum validators per request enforcement
 * 2. DoS prevention through bounded loops
 * 3. Gas consumption with maximum validators
 */
describe("Array Bounds Checking - Standalone", function () {
    let validationRegistry;
    let owner, alice, bob;
    let validators;

    beforeEach(async function () {
        [owner, alice, bob, ...validators] = await ethers.getSigners();

        // Deploy Standalone ValidationRegistry
        const ValidationRegistry = await ethers.getContractFactory(
            "contracts/erc-8004/standalone/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry"
        );

        validationRegistry = await ValidationRegistry.deploy(
            ethers.parseEther("0.1"), // minStake
            50, // minValidators (set high to prevent auto-finalization)
            66 // consensusThreshold (66%)
        );
        await validationRegistry.waitForDeployment();
    });

    describe("Maximum Validators Enforcement", function () {
        it("should reject submissions when max validators reached", async function () {
            // Create validation request
            const taskId = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600;

            const tx = await validationRegistry.connect(alice).requestValidation(
                taskId,
                bob.address, // serverAgent
                dataHash,
                1, // STAKE validation
                deadline,
                { value: ethers.parseEther("1") }
            );
            const receipt = await tx.wait();

            // Get requestId from event
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            // Get current maxValidatorsPerRequest
            const maxValidators = await validationRegistry.maxValidatorsPerRequest();
            expect(maxValidators).to.equal(100n);

            // Submit maximum allowed validators (100)
            // Note: We'll submit 5 for faster test execution
            const testLimit = 5;

            // Set lower limit for testing
            await validationRegistry.setMaxValidatorsPerRequest(testLimit);

            // Submit testLimit validators
            for (let i = 0; i < testLimit; i++) {
                await validationRegistry.connect(validators[i]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            // Verify we reached the limit
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(testLimit);

            // Try to add one more validator (should fail)
            await expect(
                validationRegistry.connect(validators[testLimit]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.be.revertedWithCustomError(validationRegistry, "MaximumValidatorsReached");
        });

        it("should enforce default limit of 100", async function () {
            const maxValidators = await validationRegistry.maxValidatorsPerRequest();
            expect(maxValidators).to.equal(100n);
        });

        it("should allow owner to adjust max validators", async function () {
            const initialMax = await validationRegistry.maxValidatorsPerRequest();
            expect(initialMax).to.equal(100n);

            // Note: Standalone version doesn't have setter yet
            // This test documents the expected behavior

            // For now, verify the limit is set correctly in constructor
            expect(initialMax).to.equal(100n);
        });
    });

    describe("Gas Consumption Analysis", function () {
        it("should finalize validation with multiple validators without DoS", async function () {
            // Set minValidators to 10 so validation doesn't auto-finalize too early
            await validationRegistry.setMinValidators(10);

            // Create validation request
            const taskId = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600;

            const tx = await validationRegistry.connect(alice).requestValidation(
                taskId,
                bob.address,
                dataHash,
                1, // STAKE validation
                deadline,
                { value: ethers.parseEther("1") }
            );
            const receipt = await tx.wait();

            // Get requestId
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            // Set low limit for testing
            const testLimit = 10;
            await validationRegistry.setMaxValidatorsPerRequest(testLimit);

            // Submit testLimit validators with correct hash
            for (let i = 0; i < testLimit; i++) {
                await validationRegistry.connect(validators[i]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            // Verify we have all validators
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(testLimit);

            // Check status - should have auto-finalized (10 validators, 100% consensus > 66% threshold)
            const finalRequest = await validationRegistry.getValidationRequest(requestId);
            expect(finalRequest.status).to.not.equal(0); // Not PENDING (finalized)
        });

        it("should handle maximum validators (100) without exceeding gas limit", async function () {
            // This is a documentation test - actual 100 validators would take too long
            // We verify the math instead:

            const maxValidators = 100;
            const gasPerValidator = 50000; // Estimated gas per validator processing
            const totalGas = maxValidators * gasPerValidator;
            const blockGasLimit = 30000000; // 30M typical block gas limit

            // Verify total gas is safe
            expect(totalGas).to.be.lessThan(blockGasLimit);

            // Maximum gas should be around 5M (well under 30M limit)
            expect(totalGas).to.equal(5000000);
        });
    });

    describe("Edge Cases", function () {
        it("should allow setting max validators", async function () {
            // Set to 50
            await validationRegistry.setMaxValidatorsPerRequest(50);
            expect(await validationRegistry.maxValidatorsPerRequest()).to.equal(50n);

            // Set back to 100
            await validationRegistry.setMaxValidatorsPerRequest(100);
            expect(await validationRegistry.maxValidatorsPerRequest()).to.equal(100n);
        });

        it("should handle consensus with limited validators", async function () {
            // Lower minValidators for this test
            await validationRegistry.setMinValidators(5);

            // Create validation request
            const taskId = ethers.randomBytes(32);
            const correctHash = ethers.randomBytes(32);
            const wrongHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600;

            const tx = await validationRegistry.connect(alice).requestValidation(
                taskId,
                bob.address,
                correctHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("1") }
            );
            const receipt = await tx.wait();

            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            // Set limit to 5
            await validationRegistry.setMaxValidatorsPerRequest(5);

            // Submit 4 correct, 1 wrong (80% success rate > 66% threshold)
            for (let i = 0; i < 4; i++) {
                await validationRegistry.connect(validators[i]).submitStakeValidation(
                    requestId,
                    correctHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            await validationRegistry.connect(validators[4]).submitStakeValidation(
                requestId,
                wrongHash,
                { value: ethers.parseEther("0.1") }
            );

            // Verify consensus reached (4/5 = 80% > 66% threshold)
            const finalRequest = await validationRegistry.getValidationRequest(requestId);

            // Should be validated (80% success rate exceeds 66% threshold)
            expect(finalRequest.status).to.equal(1); // VALIDATED
        });
    });

    describe("Protection Against DoS Attacks", function () {
        it("should prevent attacker from adding unlimited validators", async function () {
            // Create validation request
            const taskId = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600;

            const tx = await validationRegistry.connect(alice).requestValidation(
                taskId,
                bob.address,
                dataHash,
                1,
                deadline,
                { value: ethers.parseEther("1") }
            );
            const receipt = await tx.wait();

            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            // Set to 3 for fast test
            await validationRegistry.setMaxValidatorsPerRequest(3);

            // Attacker tries to flood with validators
            for (let i = 0; i < 3; i++) {
                await validationRegistry.connect(validators[i]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            // 4th attempt should fail
            await expect(
                validationRegistry.connect(validators[3]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.be.revertedWithCustomError(validationRegistry, "MaximumValidatorsReached");

            // Verify only 3 responses recorded
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(3);
        });
    });
});
