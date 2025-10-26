/**
 * TimelockController Test Suite
 *
 * Tests: 18 total
 * - G3.1: Basic Configuration (3 tests)
 * - G3.2: Operation Scheduling (5 tests)
 * - G3.3: Operation Execution (5 tests)
 * - G3.4: Role Management (3 tests)
 * - G3.5: Emergency Features (2 tests)
 */

import { expect } from "chai";
import { parseEther, keccak256, toUtf8Bytes, ZeroHash } from "ethers";
import { network } from "hardhat";

// Initialize ethers and network helpers from network connection
const { ethers, networkHelpers } = await network.connect();
const { time } = networkHelpers;

describe("TimelockController", function () {
    let timelock;
    let targetContract;
    let owner, proposer, executor, attacker;

    const MIN_DELAY = 2 * 24 * 60 * 60; // 48 hours
    const MIN_DELAY_EMERGENCY = 24 * 60 * 60; // 24 hours

    // Role hashes
    let PROPOSER_ROLE, EXECUTOR_ROLE, CANCELLER_ROLE, DEFAULT_ADMIN_ROLE;

    beforeEach(async function () {
        [owner, proposer, executor, attacker] = await ethers.getSigners();

        const SAGETimelockController = await ethers.getContractFactory("SAGETimelockController");
        timelock = await SAGETimelockController.deploy(
            MIN_DELAY,
            [proposer.address],
            [executor.address],
            owner.address
        );
        await timelock.waitForDeployment();

        // Get role hashes
        PROPOSER_ROLE = await timelock.PROPOSER_ROLE();
        EXECUTOR_ROLE = await timelock.EXECUTOR_ROLE();
        CANCELLER_ROLE = await timelock.CANCELLER_ROLE();
        DEFAULT_ADMIN_ROLE = await timelock.DEFAULT_ADMIN_ROLE();

        // Deploy target contract for testing
        const Target = await ethers.getContractFactory("AgentCardStorageTest");
        targetContract = await Target.deploy();
        await targetContract.waitForDeployment();
    });

    // ============================================================================
    // G3.1: Basic Configuration (3 tests)
    // ============================================================================

    describe("G3.1: Basic Configuration", function () {
        /**
         * Test ID: G3.1.1
         * Verification: Minimum delay is set correctly
         */
        it("G3.1.1: Should have correct minimum delay", async function () {
            const delay = await timelock.getMinDelay();
            expect(delay).to.equal(MIN_DELAY);
        });

        /**
         * Test ID: G3.1.2
         * Verification: Roles are assigned correctly
         */
        it("G3.1.2: Should assign roles correctly", async function () {
            expect(await timelock.hasRole(PROPOSER_ROLE, proposer.address)).to.be.true;
            expect(await timelock.hasRole(EXECUTOR_ROLE, executor.address)).to.be.true;
            expect(await timelock.hasRole(DEFAULT_ADMIN_ROLE, owner.address)).to.be.true;
        });

        /**
         * Test ID: G3.1.3
         * Verification: Can receive ETH
         */
        it("G3.1.3: Should receive ETH", async function () {
            await owner.sendTransaction({
                to: await timelock.getAddress(),
                value: parseEther("1.0")
            });

            const balance = await ethers.provider.getBalance(await timelock.getAddress());
            expect(balance).to.equal(parseEther("1.0"));
        });
    });

    // ============================================================================
    // G3.2: Operation Scheduling (5 tests)
    // ============================================================================

    describe("G3.2: Operation Scheduling", function () {
        let target, value, data, predecessor, salt;

        beforeEach(async function () {
            target = await targetContract.getAddress();
            value = 0;
            data = targetContract.interface.encodeFunctionData("incrementNonce", [
                ethers.id("test-agent")
            ]);
            predecessor = ZeroHash;
            salt = ZeroHash;
        });

        /**
         * Test ID: G3.2.1
         * Verification: Proposer can schedule operation
         */
        it("G3.2.1: Should allow proposer to schedule operation", async function () {
            await expect(
                timelock.connect(proposer).schedule(
                    target,
                    value,
                    data,
                    predecessor,
                    salt,
                    MIN_DELAY
                )
            ).to.emit(timelock, "CallScheduled");

            const id = await timelock.hashOperation(target, value, data, predecessor, salt);
            expect(await timelock.isOperationPending(id)).to.be.true;
        });

        /**
         * Test ID: G3.2.2
         * Verification: Non-proposer cannot schedule
         */
        it("G3.2.2: Should reject scheduling from non-proposer", async function () {
            await expect(
                timelock.connect(attacker).schedule(
                    target,
                    value,
                    data,
                    predecessor,
                    salt,
                    MIN_DELAY
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.2.3
         * Verification: Cannot schedule with delay less than minimum
         */
        it("G3.2.3: Should reject delay less than minimum", async function () {
            await expect(
                timelock.connect(proposer).schedule(
                    target,
                    value,
                    data,
                    predecessor,
                    salt,
                    MIN_DELAY / 2
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.2.4
         * Verification: Cannot schedule duplicate operation
         */
        it("G3.2.4: Should reject duplicate operation", async function () {
            await timelock.connect(proposer).schedule(
                target,
                value,
                data,
                predecessor,
                salt,
                MIN_DELAY
            );

            await expect(
                timelock.connect(proposer).schedule(
                    target,
                    value,
                    data,
                    predecessor,
                    salt,
                    MIN_DELAY
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.2.5
         * Verification: Get operation timestamp
         */
        it("G3.2.5: Should return correct operation timestamp", async function () {
            const currentTime = await time.latest();

            await timelock.connect(proposer).schedule(
                target,
                value,
                data,
                predecessor,
                salt,
                MIN_DELAY
            );

            const id = await timelock.hashOperation(target, value, data, predecessor, salt);
            const timestamp = await timelock.getTimestamp(id);

            expect(timestamp).to.be.closeTo(currentTime + MIN_DELAY, 5);
        });
    });

    // ============================================================================
    // G3.3: Operation Execution (5 tests)
    // ============================================================================

    describe("G3.3: Operation Execution", function () {
        let target, value, data, predecessor, salt;

        beforeEach(async function () {
            target = await targetContract.getAddress();
            value = 0;
            data = targetContract.interface.encodeFunctionData("incrementNonce", [
                ethers.id("test-agent")
            ]);
            predecessor = ZeroHash;
            salt = ZeroHash;

            // Schedule operation
            await timelock.connect(proposer).schedule(
                target,
                value,
                data,
                predecessor,
                salt,
                MIN_DELAY
            );
        });

        /**
         * Test ID: G3.3.1
         * Verification: Execute operation after delay
         */
        it("G3.3.1: Should execute operation after delay", async function () {
            await time.increase(MIN_DELAY + 1);

            const nonceBefore = await targetContract.getAgentNonce(ethers.id("test-agent"));

            await expect(
                timelock.connect(executor).execute(
                    target,
                    value,
                    data,
                    predecessor,
                    salt
                )
            ).to.emit(timelock, "CallExecuted");

            const nonceAfter = await targetContract.getAgentNonce(ethers.id("test-agent"));
            expect(nonceAfter).to.equal(nonceBefore + 1n);

            const id = await timelock.hashOperation(target, value, data, predecessor, salt);
            expect(await timelock.isOperationDone(id)).to.be.true;
        });

        /**
         * Test ID: G3.3.2
         * Verification: Cannot execute before delay
         */
        it("G3.3.2: Should reject execution before delay", async function () {
            await expect(
                timelock.connect(executor).execute(
                    target,
                    value,
                    data,
                    predecessor,
                    salt
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.3.3
         * Verification: Non-executor cannot execute
         */
        it("G3.3.3: Should reject execution from non-executor", async function () {
            await time.increase(MIN_DELAY + 1);

            await expect(
                timelock.connect(attacker).execute(
                    target,
                    value,
                    data,
                    predecessor,
                    salt
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.3.4
         * Verification: Cannot execute twice
         */
        it("G3.3.4: Should prevent double execution", async function () {
            await time.increase(MIN_DELAY + 1);

            await timelock.connect(executor).execute(
                target,
                value,
                data,
                predecessor,
                salt
            );

            await expect(
                timelock.connect(executor).execute(
                    target,
                    value,
                    data,
                    predecessor,
                    salt
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G3.3.5
         * Verification: Can cancel pending operation
         */
        it("G3.3.5: Should allow cancelling pending operation", async function () {
            const id = await timelock.hashOperation(target, value, data, predecessor, salt);

            await expect(
                timelock.connect(proposer).cancel(id)
            ).to.emit(timelock, "Cancelled");

            expect(await timelock.isOperationPending(id)).to.be.false;

            // Cannot execute cancelled operation
            await time.increase(MIN_DELAY + 1);
            await expect(
                timelock.connect(executor).execute(
                    target,
                    value,
                    data,
                    predecessor,
                    salt
                )
            ).to.be.revert(ethers);
        });
    });

    // ============================================================================
    // G3.4: Role Management (3 tests)
    // ============================================================================

    describe("G3.4: Role Management", function () {
        /**
         * Test ID: G3.4.1
         * Verification: Admin can grant roles
         */
        it("G3.4.1: Should allow admin to grant roles", async function () {
            await expect(
                timelock.connect(owner).grantRole(PROPOSER_ROLE, attacker.address)
            ).to.emit(timelock, "RoleGranted");

            expect(await timelock.hasRole(PROPOSER_ROLE, attacker.address)).to.be.true;
        });

        /**
         * Test ID: G3.4.2
         * Verification: Admin can revoke roles
         */
        it("G3.4.2: Should allow admin to revoke roles", async function () {
            await expect(
                timelock.connect(owner).revokeRole(PROPOSER_ROLE, proposer.address)
            ).to.emit(timelock, "RoleRevoked");

            expect(await timelock.hasRole(PROPOSER_ROLE, proposer.address)).to.be.false;
        });

        /**
         * Test ID: G3.4.3
         * Verification: Non-admin cannot manage roles
         */
        it("G3.4.3: Should reject role management from non-admin", async function () {
            await expect(
                timelock.connect(attacker).grantRole(PROPOSER_ROLE, attacker.address)
            ).to.be.revert(ethers);

            await expect(
                timelock.connect(attacker).revokeRole(PROPOSER_ROLE, proposer.address)
            ).to.be.revert(ethers);
        });
    });

    // ============================================================================
    // G3.5: Emergency Features (2 tests)
    // ============================================================================

    describe("G3.5: Emergency Features", function () {
        /**
         * Test ID: G3.5.1
         * Verification: Admin can update minimum delay
         */
        it("G3.5.1: Should allow updating minimum delay", async function () {
            const newDelay = MIN_DELAY_EMERGENCY;
            const target = await timelock.getAddress();
            const value = 0;
            const data = timelock.interface.encodeFunctionData("updateDelay", [newDelay]);
            const predecessor = ZeroHash;
            const salt = ZeroHash;

            // Schedule delay update
            await timelock.connect(proposer).schedule(
                target,
                value,
                data,
                predecessor,
                salt,
                MIN_DELAY
            );

            await time.increase(MIN_DELAY + 1);

            await timelock.connect(executor).execute(
                target,
                value,
                data,
                predecessor,
                salt
            );

            expect(await timelock.getMinDelay()).to.equal(newDelay);
        });

        /**
         * Test ID: G3.5.2
         * Verification: Batch operations work
         */
        it("G3.5.2: Should support batch operations", async function () {
            const targets = [
                await targetContract.getAddress(),
                await targetContract.getAddress()
            ];
            const values = [0, 0];
            const datas = [
                targetContract.interface.encodeFunctionData("incrementNonce", [
                    ethers.id("agent-1")
                ]),
                targetContract.interface.encodeFunctionData("incrementNonce", [
                    ethers.id("agent-2")
                ])
            ];
            const predecessor = ZeroHash;
            const salt = ZeroHash;

            await timelock.connect(proposer).scheduleBatch(
                targets,
                values,
                datas,
                predecessor,
                salt,
                MIN_DELAY
            );

            await time.increase(MIN_DELAY + 1);

            await expect(
                timelock.connect(executor).executeBatch(
                    targets,
                    values,
                    datas,
                    predecessor,
                    salt
                )
            ).to.emit(timelock, "CallExecuted");

            // Verify both operations executed
            expect(await targetContract.getAgentNonce(ethers.id("agent-1"))).to.equal(1);
            expect(await targetContract.getAgentNonce(ethers.id("agent-2"))).to.equal(1);
        });
    });
});
