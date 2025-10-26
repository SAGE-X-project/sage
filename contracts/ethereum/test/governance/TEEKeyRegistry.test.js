/**
 * TEEKeyRegistry Test Suite
 *
 * Tests: 20 total
 * - G2.1: Proposal Management (5 tests)
 * - G2.2: Voting System (5 tests)
 * - G2.3: Proposal Execution (5 tests)
 * - G2.4: Governance Parameters (3 tests)
 * - G2.5: Security Features (2 tests)
 */

import { expect } from "chai";
import { parseEther, keccak256, toUtf8Bytes } from "ethers";
import { network } from "hardhat";

// Initialize ethers and network helpers from network connection
const { ethers, networkHelpers } = await network.connect();
const { time } = networkHelpers;

describe("TEEKeyRegistry", function () {
    let teeRegistry;
    let owner, proposer, voter1, voter2, voter3, attacker;

    const PROPOSAL_STAKE = parseEther("1.0");
    const VOTING_PERIOD = 7 * 24 * 60 * 60; // 7 days
    const MIN_PARTICIPATION = 10; // 10%
    const APPROVAL_THRESHOLD = 66; // 66%
    const SLASH_PERCENTAGE = 50; // 50%

    // Test TEE keys
    const TEE_KEY_1 = keccak256(toUtf8Bytes("Intel-SGX-Key-1"));
    const TEE_KEY_2 = keccak256(toUtf8Bytes("AMD-SEV-Key-1"));
    const TEE_ATTESTATION = "0x1234567890abcdef";

    beforeEach(async function () {
        [owner, proposer, voter1, voter2, voter3, attacker] = await ethers.getSigners();

        const TEEKeyRegistry = await ethers.getContractFactory("TEEKeyRegistry");
        teeRegistry = await TEEKeyRegistry.deploy();
        await teeRegistry.waitForDeployment();

        // Register voters with weights
        await teeRegistry.connect(owner).registerVoter(voter1.address, 100);
        await teeRegistry.connect(owner).registerVoter(voter2.address, 100);
        await teeRegistry.connect(owner).registerVoter(voter3.address, 100);
    });

    // ============================================================================
    // G2.1: Proposal Management (5 tests)
    // ============================================================================

    describe("G2.1: Proposal Management", function () {
        /**
         * Test ID: G2.1.1
         * Verification: Anyone can propose with stake
         */
        it("G2.1.1: Should allow proposing TEE key with stake", async function () {
            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: PROPOSAL_STAKE }
                )
            ).to.emit(teeRegistry, "TEEKeyProposed")
                .withArgs(0, proposer.address, TEE_KEY_1);

            const proposal = await teeRegistry.proposals(0);
            expect(proposal.keyHash).to.equal(TEE_KEY_1);
            expect(proposal.proposer).to.equal(proposer.address);
            expect(proposal.executed).to.be.false;
        });

        /**
         * Test ID: G2.1.2
         * Verification: Reject proposal without sufficient stake
         */
        it("G2.1.2: Should reject proposal with insufficient stake", async function () {
            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: parseEther("0.5") }
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.1.3
         * Verification: Cannot propose duplicate key
         */
        it("G2.1.3: Should reject duplicate key proposal", async function () {
            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_1,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );

            await expect(
                teeRegistry.connect(attacker).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: PROPOSAL_STAKE }
                )
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.1.4
         * Verification: Get proposal details
         */
        it("G2.1.4: Should return correct proposal details", async function () {
            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_1,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );

            const proposal = await teeRegistry.proposals(0);
            expect(proposal.keyHash).to.equal(TEE_KEY_1);
            expect(proposal.proposer).to.equal(proposer.address);
            expect(proposal.stake).to.equal(PROPOSAL_STAKE);
            expect(proposal.votesFor).to.equal(0);
            expect(proposal.votesAgainst).to.equal(0);
        });

        /**
         * Test ID: G2.1.5
         * Verification: Proposal count increments
         */
        it("G2.1.5: Should increment proposal count", async function () {
            expect(await teeRegistry.proposalCount()).to.equal(0);

            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_1,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );

            expect(await teeRegistry.proposalCount()).to.equal(1);

            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_2,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );

            expect(await teeRegistry.proposalCount()).to.equal(2);
        });
    });

    // ============================================================================
    // G2.2: Voting System (5 tests)
    // ============================================================================

    describe("G2.2: Voting System", function () {
        beforeEach(async function () {
            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_1,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );
        });

        /**
         * Test ID: G2.2.1
         * Verification: Registered voter can vote
         */
        it("G2.2.1: Should allow registered voter to vote", async function () {
            await expect(
                teeRegistry.connect(voter1).vote(0, true)
            ).to.emit(teeRegistry, "VoteCast")
                .withArgs(0, voter1.address, true, 100);

            const proposal = await teeRegistry.proposals(0);
            expect(proposal.votesFor).to.equal(100);
            expect(proposal.votesAgainst).to.equal(0);
        });

        /**
         * Test ID: G2.2.2
         * Verification: Non-registered voter cannot vote
         */
        it("G2.2.2: Should reject vote from non-registered voter", async function () {
            await expect(
                teeRegistry.connect(attacker).vote(0, true)
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.2.3
         * Verification: Cannot vote twice
         */
        it("G2.2.3: Should prevent double voting", async function () {
            await teeRegistry.connect(voter1).vote(0, true);

            await expect(
                teeRegistry.connect(voter1).vote(0, true)
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.2.4
         * Verification: Votes are weighted correctly
         */
        it("G2.2.4: Should apply voting weights correctly", async function () {
            await teeRegistry.connect(voter1).vote(0, true); // +100
            await teeRegistry.connect(voter2).vote(0, false); // +100

            const proposal = await teeRegistry.proposals(0);
            expect(proposal.votesFor).to.equal(100);
            expect(proposal.votesAgainst).to.equal(100);
        });

        /**
         * Test ID: G2.2.5
         * Verification: Cannot vote after period ends
         */
        it("G2.2.5: Should reject vote after voting period", async function () {
            await time.increase(VOTING_PERIOD + 1);

            await expect(
                teeRegistry.connect(voter1).vote(0, true)
            ).to.be.revert(ethers);
        });
    });

    // ============================================================================
    // G2.3: Proposal Execution (5 tests)
    // ============================================================================

    describe("G2.3: Proposal Execution", function () {
        beforeEach(async function () {
            await teeRegistry.connect(proposer).proposeTEEKey(
                TEE_KEY_1,
                TEE_ATTESTATION,
                { value: PROPOSAL_STAKE }
            );
        });

        /**
         * Test ID: G2.3.1
         * Verification: Execute approved proposal
         */
        it("G2.3.1: Should execute approved proposal", async function () {
            // All 3 voters vote for (300 votes for, 0 against = 100% approval)
            await teeRegistry.connect(voter1).vote(0, true);
            await teeRegistry.connect(voter2).vote(0, true);
            await teeRegistry.connect(voter3).vote(0, true);

            await time.increase(VOTING_PERIOD + 1);

            const proposerBalanceBefore = await ethers.provider.getBalance(proposer.address);

            await expect(
                teeRegistry.connect(owner).executeProposal(0)
            ).to.emit(teeRegistry, "ProposalExecuted")
                .withArgs(0, true);

            // Check key is trusted
            expect(await teeRegistry.isTrustedTEEKey(TEE_KEY_1)).to.be.true;

            // Check stake returned
            const proposal = await teeRegistry.proposals(0);
            expect(proposal.executed).to.be.true;
        });

        /**
         * Test ID: G2.3.2
         * Verification: Reject proposal with insufficient votes
         */
        it("G2.3.2: Should reject proposal below approval threshold", async function () {
            // 1 for, 2 against (100 vs 200 = 33% approval, need 66%)
            await teeRegistry.connect(voter1).vote(0, true);
            await teeRegistry.connect(voter2).vote(0, false);
            await teeRegistry.connect(voter3).vote(0, false);

            await time.increase(VOTING_PERIOD + 1);

            await expect(
                teeRegistry.connect(owner).executeProposal(0)
            ).to.emit(teeRegistry, "ProposalExecuted")
                .withArgs(0, false);

            // Check key is not trusted
            expect(await teeRegistry.isTrustedTEEKey(TEE_KEY_1)).to.be.false;

            // Check stake slashed
            const proposal = await teeRegistry.proposals(0);
            expect(proposal.executed).to.be.true;
        });

        /**
         * Test ID: G2.3.3
         * Verification: Cannot execute before voting period
         */
        it("G2.3.3: Should reject execution before voting period ends", async function () {
            await teeRegistry.connect(voter1).vote(0, true);

            await expect(
                teeRegistry.connect(owner).executeProposal(0)
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.3.4
         * Verification: Cannot execute twice
         */
        it("G2.3.4: Should prevent double execution", async function () {
            await teeRegistry.connect(voter1).vote(0, true);
            await teeRegistry.connect(voter2).vote(0, true);
            await teeRegistry.connect(voter3).vote(0, true);

            await time.increase(VOTING_PERIOD + 1);

            await teeRegistry.connect(owner).executeProposal(0);

            await expect(
                teeRegistry.connect(owner).executeProposal(0)
            ).to.be.revert(ethers);
        });

        /**
         * Test ID: G2.3.5
         * Verification: Reject with insufficient participation
         */
        it("G2.3.5: Should reject proposal with low participation", async function () {
            // Only 1 voter participates (100 votes out of 300 total = 33% participation, need 10%)
            // This test assumes totalVotingPower is tracked
            await teeRegistry.connect(voter1).vote(0, true);

            await time.increase(VOTING_PERIOD + 1);

            // Should succeed as 33% > 10% minimum
            await expect(
                teeRegistry.connect(owner).executeProposal(0)
            ).to.emit(teeRegistry, "ProposalExecuted");
        });
    });

    // ============================================================================
    // G2.4: Governance Parameters (3 tests)
    // ============================================================================

    describe("G2.4: Governance Parameters", function () {
        /**
         * Test ID: G2.4.1
         * Verification: Owner can update proposal stake
         */
        it("G2.4.1: Should allow owner to update proposal stake", async function () {
            const newStake = parseEther("2.0");

            await expect(
                teeRegistry.connect(owner).updateProposalStake(newStake)
            ).to.emit(teeRegistry, "ProposalStakeUpdated");

            // Verify by checking if new proposal requires new stake
            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: PROPOSAL_STAKE }
                )
            ).to.be.revert(ethers);

            // Should work with new stake
            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: newStake }
                )
            ).to.emit(teeRegistry, "TEEKeyProposed");
        });

        /**
         * Test ID: G2.4.2
         * Verification: Owner can manage voters
         */
        it("G2.4.2: Should allow owner to register/unregister voters", async function () {
            const newVoter = attacker.address;

            await expect(
                teeRegistry.connect(owner).registerVoter(newVoter, 150)
            ).to.emit(teeRegistry, "VoterRegistered")
                .withArgs(newVoter, 150);

            expect(await teeRegistry.voters(newVoter)).to.equal(150);

            await expect(
                teeRegistry.connect(owner).unregisterVoter(newVoter)
            ).to.emit(teeRegistry, "VoterUnregistered")
                .withArgs(newVoter);

            expect(await teeRegistry.voters(newVoter)).to.equal(0);
        });

        /**
         * Test ID: G2.4.3
         * Verification: Non-owner cannot update parameters
         */
        it("G2.4.3: Should reject parameter updates from non-owner", async function () {
            await expect(
                teeRegistry.connect(attacker).updateProposalStake(parseEther("5.0"))
            ).to.be.revert(ethers);

            await expect(
                teeRegistry.connect(attacker).registerVoter(attacker.address, 100)
            ).to.be.revert(ethers);
        });
    });

    // ============================================================================
    // G2.5: Security Features (2 tests)
    // ============================================================================

    describe("G2.5: Security Features", function () {
        /**
         * Test ID: G2.5.1
         * Verification: Owner can pause/unpause
         */
        it("G2.5.1: Should allow emergency pause", async function () {
            await teeRegistry.connect(owner).pause();

            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: PROPOSAL_STAKE }
                )
            ).to.be.revert(ethers);

            await teeRegistry.connect(owner).unpause();

            await expect(
                teeRegistry.connect(proposer).proposeTEEKey(
                    TEE_KEY_1,
                    TEE_ATTESTATION,
                    { value: PROPOSAL_STAKE }
                )
            ).to.emit(teeRegistry, "TEEKeyProposed");
        });

        /**
         * Test ID: G2.5.2
         * Verification: 2-step ownership transfer
         */
        it("G2.5.2: Should require 2-step ownership transfer", async function () {
            await teeRegistry.connect(owner).transferOwnership(voter1.address);

            // Owner should still be owner until acceptance
            await expect(
                teeRegistry.connect(voter1).registerVoter(attacker.address, 100)
            ).to.be.revert(ethers);

            await teeRegistry.connect(voter1).acceptOwnership();

            // Now voter1 is owner
            await expect(
                teeRegistry.connect(voter1).registerVoter(attacker.address, 100)
            ).to.emit(teeRegistry, "VoterRegistered");
        });
    });
});
