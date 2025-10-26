/**
 * Governance Integration Test Suite
 *
 * Tests: 10 total
 * - G4.1: Full Governance Flow (5 tests)
 * - G4.2: Emergency Procedures (3 tests)
 * - G4.3: Complex Scenarios (2 tests)
 *
 * Integration: Multi-Sig → Timelock → Registry
 */

import { expect } from "chai";
import { parseEther, ZeroHash } from "ethers";
import { network } from "hardhat";

// Initialize ethers and network helpers from network connection
const { ethers, networkHelpers } = await network.connect();
const { time } = networkHelpers;

describe("Governance Integration", function () {
    let multiSig, timelock, registry, hook;
    let owner, signer1, signer2, signer3, attacker;

    const THRESHOLD = 2; // 2/3 multi-sig
    const MIN_DELAY = 2 * 24 * 60 * 60; // 48 hours
    const MIN_DELAY_EMERGENCY = 24 * 60 * 60; // 24 hours

    beforeEach(async function () {
        [owner, signer1, signer2, signer3, attacker] = await ethers.getSigners();

        // Deploy Multi-Sig Wallet
        const SimpleMultiSig = await ethers.getContractFactory("SimpleMultiSig");
        multiSig = await SimpleMultiSig.deploy(
            [signer1.address, signer2.address, signer3.address],
            THRESHOLD
        );
        await multiSig.waitForDeployment();

        // Deploy Timelock Controller
        const SAGETimelockController = await ethers.getContractFactory("SAGETimelockController");
        timelock = await SAGETimelockController.deploy(
            MIN_DELAY,
            [await multiSig.getAddress()],
            [await multiSig.getAddress()],
            ethers.ZeroAddress
        );
        await timelock.waitForDeployment();

        // Deploy AgentCardVerifyHook
        const Hook = await ethers.getContractFactory("AgentCardVerifyHook");
        hook = await Hook.deploy();
        await hook.waitForDeployment();

        // Deploy AgentCardRegistry with owner as initial owner
        const Registry = await ethers.getContractFactory("AgentCardRegistry");
        registry = await Registry.deploy(await hook.getAddress());
        await registry.waitForDeployment();
    });

    /**
     * Helper function to execute governance action
     */
    async function executeGovernanceAction(target, value, data, delay) {
        const timelockAddress = await timelock.getAddress();

        // Prepare timelock schedule call
        const scheduleData = timelock.interface.encodeFunctionData("schedule", [
            target,
            value,
            data,
            ZeroHash,
            ZeroHash,
            delay
        ]);

        // Propose through multi-sig
        await multiSig.connect(signer1).proposeTransaction(
            timelockAddress,
            0,
            scheduleData
        );

        const txId = (await multiSig.transactionCount()) - 1n;

        // Confirm by second signer (reaches threshold)
        await multiSig.connect(signer2).confirmTransaction(txId);

        // Wait for timelock delay
        await time.increase(delay + 1);

        // Prepare execute call
        const executeData = timelock.interface.encodeFunctionData("execute", [
            target,
            value,
            data,
            ZeroHash,
            ZeroHash
        ]);

        // Execute through multi-sig
        await multiSig.connect(signer1).proposeTransaction(
            timelockAddress,
            0,
            executeData
        );

        const executeTxId = (await multiSig.transactionCount()) - 1n;
        await multiSig.connect(signer2).confirmTransaction(executeTxId);
    }

    // ============================================================================
    // G4.1: Full Governance Flow (5 tests)
    // ============================================================================

    describe("G4.1: Full Governance Flow", function () {
        beforeEach(async function () {
            // Transfer ownership to timelock for governance tests
            await registry.transferOwnership(await timelock.getAddress());
            await hook.transferOwnership(await timelock.getAddress());

            // Accept ownership through governance mechanism
            const registryAcceptData = registry.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                registryAcceptData,
                MIN_DELAY
            );

            const hookAcceptData = hook.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await hook.getAddress(),
                0,
                hookAcceptData,
                MIN_DELAY
            );
        });
        /**
         * Test ID: G4.1.1
         * Verification: Update registry hook through governance
         */
        it("G4.1.1: Should update registry hook through governance", async function () {
            const newHook = await (await ethers.getContractFactory("AgentCardVerifyHook")).deploy();
            await newHook.waitForDeployment();

            const data = registry.interface.encodeFunctionData("setVerifyHook", [
                await newHook.getAddress()
            ]);

            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                data,
                MIN_DELAY
            );

            expect(await registry.verifyHook()).to.equal(await newHook.getAddress());
        });

        /**
         * Test ID: G4.1.2
         * Verification: Add address to whitelist through governance
         */
        it("G4.1.2: Should manage whitelist through governance", async function () {
            const data = hook.interface.encodeFunctionData("addToWhitelist", [
                signer1.address
            ]);

            await executeGovernanceAction(
                await hook.getAddress(),
                0,
                data,
                MIN_DELAY
            );

            expect(await hook.whitelisted(signer1.address)).to.be.true;
        });

        /**
         * Test ID: G4.1.3
         * Verification: Update stake amount through governance
         */
        it("G4.1.3: Should update stake amount through governance", async function () {
            const newStake = parseEther("0.02");
            const data = registry.interface.encodeFunctionData("setRegistrationStake", [
                newStake
            ]);

            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                data,
                MIN_DELAY
            );

            expect(await registry.registrationStake()).to.equal(newStake);
        });

        /**
         * Test ID: G4.1.4
         * Verification: Cannot bypass governance
         */
        it("G4.1.4: Should prevent direct updates without governance", async function () {
            const newHook = await (await ethers.getContractFactory("AgentCardVerifyHook")).deploy();
            await newHook.waitForDeployment();

            // Try to update directly (should fail)
            await expect(
                registry.connect(signer1).setVerifyHook(await newHook.getAddress())
            ).to.be.revert(ethers);

            // Owner of registry is timelock, not signer1
            await expect(
                registry.connect(owner).setVerifyHook(await newHook.getAddress())
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G4.1.5
         * Verification: Multi-sig threshold is enforced
         */
        it("G4.1.5: Should require threshold signatures", async function () {
            const data = hook.interface.encodeFunctionData("addToWhitelist", [
                attacker.address
            ]);

            const scheduleData = timelock.interface.encodeFunctionData("schedule", [
                await hook.getAddress(),
                0,
                data,
                ZeroHash,
                ZeroHash,
                MIN_DELAY
            ]);

            // Only 1 signer proposes (auto-confirms)
            await multiSig.connect(signer1).proposeTransaction(
                await timelock.getAddress(),
                0,
                scheduleData
            );

            const txId = (await multiSig.transactionCount()) - 1n;

            // Transaction not executed yet (need 2 confirmations)
            const [, , , executed] = await multiSig.getTransaction(txId);
            expect(executed).to.be.false;

            // Second confirmation executes
            await expect(
                multiSig.connect(signer2).confirmTransaction(txId)
            ).to.emit(multiSig, "TransactionExecuted");
        });
    });

    // ============================================================================
    // G4.2: Emergency Procedures (3 tests)
    // ============================================================================

    describe("G4.2: Emergency Procedures", function () {
        beforeEach(async function () {
            // Transfer ownership to timelock for governance tests
            await registry.transferOwnership(await timelock.getAddress());
            await hook.transferOwnership(await timelock.getAddress());

            // Accept ownership through governance mechanism
            const registryAcceptData = registry.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                registryAcceptData,
                MIN_DELAY
            );

            const hookAcceptData = hook.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await hook.getAddress(),
                0,
                hookAcceptData,
                MIN_DELAY
            );
        });

        /**
         * Test ID: G4.2.1
         * Verification: Emergency pause with reduced delay
         */
        it("G4.2.1: Should allow emergency pause with shorter delay", async function () {
            // Update timelock to allow shorter delay for emergency
            const updateDelayData = timelock.interface.encodeFunctionData("updateDelay", [
                MIN_DELAY_EMERGENCY
            ]);

            await executeGovernanceAction(
                await timelock.getAddress(),
                0,
                updateDelayData,
                MIN_DELAY
            );

            // Now pause with emergency delay
            const pauseData = registry.interface.encodeFunctionData("pause");

            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                pauseData,
                MIN_DELAY_EMERGENCY
            );

            expect(await registry.paused()).to.be.true;
        });

        /**
         * Test ID: G4.2.2
         * Verification: Registry operations blocked when paused
         */
        it("G4.2.2: Should block operations when paused", async function () {
            const pauseData = registry.interface.encodeFunctionData("pause");

            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                pauseData,
                MIN_DELAY
            );

            // Try to commit (should fail when paused)
            const commitHash = ethers.id("test-commitment");
            await expect(
                registry.connect(signer1).commitRegistration(commitHash, {
                    value: parseEther("0.01")
                })
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G4.2.3
         * Verification: Can unpause after issue resolved
         */
        it("G4.2.3: Should allow unpause through governance", async function () {
            // First pause
            const pauseData = registry.interface.encodeFunctionData("pause");
            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                pauseData,
                MIN_DELAY
            );

            expect(await registry.paused()).to.be.true;

            // Then unpause
            const unpauseData = registry.interface.encodeFunctionData("unpause");
            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                unpauseData,
                MIN_DELAY
            );

            expect(await registry.paused()).to.be.false;
        });
    });

    // ============================================================================
    // G4.3: Complex Scenarios (2 tests)
    // ============================================================================

    describe("G4.3: Complex Scenarios", function () {
        beforeEach(async function () {
            // Transfer ownership to timelock for governance tests
            await registry.transferOwnership(await timelock.getAddress());
            await hook.transferOwnership(await timelock.getAddress());

            // Accept ownership through governance mechanism
            const registryAcceptData = registry.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                registryAcceptData,
                MIN_DELAY
            );

            const hookAcceptData = hook.interface.encodeFunctionData("acceptOwnership");
            await executeGovernanceAction(
                await hook.getAddress(),
                0,
                hookAcceptData,
                MIN_DELAY
            );
        });

        /**
         * Test ID: G4.3.1
         * Verification: Multiple proposals in queue
         */
        it("G4.3.1: Should handle multiple queued proposals", async function () {
            // Proposal 1: Update hook
            const newHook1 = await (await ethers.getContractFactory("AgentCardVerifyHook")).deploy();
            await newHook1.waitForDeployment();
            const data1 = registry.interface.encodeFunctionData("setVerifyHook", [
                await newHook1.getAddress()
            ]);

            // Proposal 2: Update another hook
            const newHook = await (await ethers.getContractFactory("AgentCardVerifyHook")).deploy();
            await newHook.waitForDeployment();
            const data2 = registry.interface.encodeFunctionData("setVerifyHook", [
                await newHook.getAddress()
            ]);

            // Schedule both
            const timelockAddress = await timelock.getAddress();

            const scheduleData1 = timelock.interface.encodeFunctionData("schedule", [
                await registry.getAddress(),
                0,
                data1,
                ZeroHash,
                ethers.id("salt1"),
                MIN_DELAY
            ]);

            const scheduleData2 = timelock.interface.encodeFunctionData("schedule", [
                await registry.getAddress(),
                0,
                data2,
                ZeroHash,
                ethers.id("salt2"),
                MIN_DELAY
            ]);

            // Propose both through multi-sig
            await multiSig.connect(signer1).proposeTransaction(timelockAddress, 0, scheduleData1);
            let txId = (await multiSig.transactionCount()) - 1n;
            await multiSig.connect(signer2).confirmTransaction(txId);

            await multiSig.connect(signer1).proposeTransaction(timelockAddress, 0, scheduleData2);
            txId = (await multiSig.transactionCount()) - 1n;
            await multiSig.connect(signer2).confirmTransaction(txId);

            // Both should be pending
            const id1 = await timelock.hashOperation(
                await registry.getAddress(),
                0,
                data1,
                ZeroHash,
                ethers.id("salt1")
            );
            const id2 = await timelock.hashOperation(
                await registry.getAddress(),
                0,
                data2,
                ZeroHash,
                ethers.id("salt2")
            );

            expect(await timelock.isOperationPending(id1)).to.be.true;
            expect(await timelock.isOperationPending(id2)).to.be.true;
        });

        /**
         * Test ID: G4.3.2
         * Verification: Ownership transfer through governance
         */
        it("G4.3.2: Should handle ownership transfer correctly", async function () {
            // Registry owner is timelock
            expect(await registry.owner()).to.equal(await timelock.getAddress());

            // New owner (another timelock for example)
            const newOwner = signer3.address;

            // Transfer ownership (step 1)
            const transferData = registry.interface.encodeFunctionData("transferOwnership", [
                newOwner
            ]);

            await executeGovernanceAction(
                await registry.getAddress(),
                0,
                transferData,
                MIN_DELAY
            );

            // Pending owner should be set
            expect(await registry.pendingOwner()).to.equal(newOwner);

            // New owner must accept
            await registry.connect(signer3).acceptOwnership();

            expect(await registry.owner()).to.equal(newOwner);
        });
    });
});
