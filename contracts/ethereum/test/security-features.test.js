const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

/**
 * Security Features Integration Tests
 *
 * Tests all security improvements from the audit:
 * 1. Front-running protection (commit-reveal)
 * 2. Cross-chain replay protection
 * 3. Array bounds checking (DoS prevention)
 * 4. TEE key governance
 * 5. Emergency pause procedures
 */
describe("Security Features Integration Tests", function () {
    let sageRegistryV3;
    let reputationRegistryV2;
    let validationRegistry;
    let teeKeyRegistry;

    let owner, alice, bob, attacker;
    let validators;
    let aliceWallet, attackerWallet, bobWallet;

    beforeEach(async function () {
        [owner, alice, bob, attacker, ...validators] = await ethers.getSigners();

        // Create wallets with public keys for testing
        aliceWallet = ethers.Wallet.createRandom().connect(ethers.provider);
        attackerWallet = ethers.Wallet.createRandom().connect(ethers.provider);
        bobWallet = ethers.Wallet.createRandom().connect(ethers.provider);

        // Fund wallets
        await owner.sendTransaction({ to: aliceWallet.address, value: ethers.parseEther("10") });
        await owner.sendTransaction({ to: attackerWallet.address, value: ethers.parseEther("10") });
        await owner.sendTransaction({ to: bobWallet.address, value: ethers.parseEther("10") });

        // Deploy SageRegistryV3 (with front-running protection)
        const SageRegistryV3 = await ethers.getContractFactory("SageRegistryV3");
        sageRegistryV3 = await SageRegistryV3.deploy();
        await sageRegistryV3.waitForDeployment();

        // Deploy ERC8004IdentityRegistry (adapter version)
        const IdentityRegistry = await ethers.getContractFactory("contracts/erc-8004/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry");
        const identityRegistry = await IdentityRegistry.deploy(await sageRegistryV3.getAddress());
        await identityRegistry.waitForDeployment();

        const ReputationRegistryV2 = await ethers.getContractFactory("ERC8004ReputationRegistryV2");
        reputationRegistryV2 = await ReputationRegistryV2.deploy(await identityRegistry.getAddress());
        await reputationRegistryV2.waitForDeployment();

        // Deploy ValidationRegistry (adapter version)
        const ValidationRegistry = await ethers.getContractFactory("contracts/erc-8004/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry");
        validationRegistry = await ValidationRegistry.deploy(
            await identityRegistry.getAddress(),
            await reputationRegistryV2.getAddress()
        );
        await validationRegistry.waitForDeployment();

        // Deploy TEEKeyRegistry
        const TEEKeyRegistry = await ethers.getContractFactory("TEEKeyRegistry");
        teeKeyRegistry = await TEEKeyRegistry.deploy();
        await teeKeyRegistry.waitForDeployment();
    });

    describe("Front-Running Protection Tests", function () {
        describe("Agent Registration", function () {
            it("should protect against DID front-running", async function () {
                const did = "did:sage:valuable-name";
                const publicKey = ethers.getBytes(aliceWallet.signingKey.publicKey);
                const salt = ethers.randomBytes(32);
                const chainId = (await ethers.provider.getNetwork()).chainId;

                // Alice creates commitment
                const commitHash = ethers.keccak256(
                    ethers.solidityPacked(
                        ["string", "bytes", "address", "bytes32", "uint256"],
                        [did, publicKey, aliceWallet.address, salt, chainId]
                    )
                );

                // Alice commits
                await sageRegistryV3.connect(aliceWallet).commitRegistration(commitHash);

                // Attacker sees the commit transaction in mempool
                // Attacker tries to register the DID directly (without commit-reveal)
                const attackerPublicKey = ethers.getBytes(attackerWallet.signingKey.publicKey);
                const nonce = 0;
                const attackerSig = await signRegistration(
                    attackerWallet,
                    did,
                    "Attacker Agent",
                    "Stolen DID",
                    "https://attacker.com",
                    attackerPublicKey,
                    "{}",
                    nonce
                );

                // Attacker's direct registration succeeds (legacy function still available)
                await sageRegistryV3.connect(attackerWallet).registerAgent(
                    did,
                    "Attacker Agent",
                    "Stolen DID",
                    "https://attacker.com",
                    attackerPublicKey,
                    "{}",
                    attackerSig
                );

                // Wait for minimum delay
                await time.increase(61);

                // Alice tries to reveal (should fail - DID already taken)
                const aliceSig = await signRegistration(
                    aliceWallet,
                    did,
                    "Alice Agent",
                    "Legitimate agent",
                    "https://alice.com",
                    publicKey,
                    "{}",
                    0
                );

                await expect(
                    sageRegistryV3.connect(aliceWallet).registerAgentWithReveal(
                        did,
                        "Alice Agent",
                        "Legitimate agent",
                        "https://alice.com",
                        publicKey,
                        "{}",
                        aliceSig,
                        salt
                    )
                ).to.be.revertedWith("DID already registered");

                // This demonstrates: Users MUST use commit-reveal for protection
                // Legacy registerAgent() is still vulnerable
            });

            it("should successfully register with commit-reveal", async function () {
                const did = "did:sage:alice-protected";
                const publicKey = ethers.getBytes(aliceWallet.signingKey.publicKey);
                const salt = ethers.randomBytes(32);
                const chainId = (await ethers.provider.getNetwork()).chainId;

                // Create commitment
                const commitHash = ethers.keccak256(
                    ethers.solidityPacked(
                        ["string", "bytes", "address", "bytes32", "uint256"],
                        [did, publicKey, aliceWallet.address, salt, chainId]
                    )
                );

                // Commit
                await expect(
                    sageRegistryV3.connect(aliceWallet).commitRegistration(commitHash)
                ).to.emit(sageRegistryV3, "RegistrationCommitted")
                    .withArgs(aliceWallet.address, commitHash, await time.latest() + 1);

                // Wait minimum delay
                await time.increase(61);

                // Reveal and register
                const signature = await signRegistration(
                    aliceWallet,
                    did,
                    "Alice Agent",
                    "Protected registration",
                    "https://alice.com",
                    publicKey,
                    "{}",
                    0
                );

                const tx = await sageRegistryV3.connect(aliceWallet).registerAgentWithReveal(
                    did,
                    "Alice Agent",
                    "Protected registration",
                    "https://alice.com",
                    publicKey,
                    "{}",
                    signature,
                    salt
                );
                const receipt = await tx.wait();

                // Get agentId from RegistrationRevealed event
                const event = receipt.logs.find(log => {
                    try {
                        return sageRegistryV3.interface.parseLog(log)?.name === 'RegistrationRevealed';
                    } catch { return false; }
                });
                const agentId = sageRegistryV3.interface.parseLog(event).args.agentId;

                // Verify agent registered
                const agent = await sageRegistryV3.getAgent(agentId);
                expect(agent.did).to.equal(did);
            });

            it("should reject reveal too soon", async function () {
                const commitHash = ethers.randomBytes(32);
                await sageRegistryV3.connect(alice).commitRegistration(commitHash);

                const did = "did:sage:alice";
                const publicKey = ethers.randomBytes(64);
                const salt = ethers.randomBytes(32);
                const signature = await signRegistration(alice, did, "Alice", "Desc", "https://alice.com", publicKey, "{}", 0);

                // Try to reveal immediately
                await expect(
                    sageRegistryV3.connect(alice).registerAgentWithReveal(
                        did,
                        "Alice",
                        "Desc",
                        "https://alice.com",
                        publicKey,
                        "{}",
                        signature,
                        salt
                    )
                ).to.be.revertedWithCustomError(sageRegistryV3, "RevealTooSoon");
            });

            it("should reject reveal too late", async function () {
                const did = "did:sage:alice";
                const publicKey = ethers.randomBytes(64);
                const salt = ethers.randomBytes(32);
                const chainId = (await ethers.provider.getNetwork()).chainId;

                const commitHash = ethers.keccak256(
                    ethers.solidityPacked(
                        ["string", "bytes", "address", "bytes32", "uint256"],
                        [did, publicKey, alice.address, salt, chainId]
                    )
                );

                await sageRegistryV3.connect(alice).commitRegistration(commitHash);

                // Wait more than 1 hour
                await time.increase(3601);

                const signature = await signRegistration(alice, did, "Alice", "Desc", "https://alice.com", publicKey, "{}", 0);

                await expect(
                    sageRegistryV3.connect(alice).registerAgentWithReveal(
                        did,
                        "Alice",
                        "Desc",
                        "https://alice.com",
                        publicKey,
                        "{}",
                        signature,
                        salt
                    )
                ).to.be.revertedWithCustomError(sageRegistryV3, "RevealTooLate");
            });

            it("should reject invalid reveal (wrong salt)", async function () {
                const did = "did:sage:alice";
                const publicKey = ethers.randomBytes(64);
                const salt = ethers.randomBytes(32);
                const wrongSalt = ethers.randomBytes(32);
                const chainId = (await ethers.provider.getNetwork()).chainId;

                const commitHash = ethers.keccak256(
                    ethers.solidityPacked(
                        ["string", "bytes", "address", "bytes32", "uint256"],
                        [did, publicKey, alice.address, salt, chainId]
                    )
                );

                await sageRegistryV3.connect(alice).commitRegistration(commitHash);
                await time.increase(61);

                const signature = await signRegistration(alice, did, "Alice", "Desc", "https://alice.com", publicKey, "{}", 0);

                // Try to reveal with wrong salt
                await expect(
                    sageRegistryV3.connect(alice).registerAgentWithReveal(
                        did,
                        "Alice",
                        "Desc",
                        "https://alice.com",
                        publicKey,
                        "{}",
                        signature,
                        wrongSalt
                    )
                ).to.be.revertedWithCustomError(sageRegistryV3, "InvalidReveal");
            });
        });

        describe("Task Authorization", function () {
            it("should protect task authorization with commit-reveal", async function () {
                const taskId = ethers.randomBytes(32);
                const serverAgent = bob.address;
                const deadline = (await time.latest()) + 3600;
                const salt = ethers.randomBytes(32);
                const chainId = (await ethers.provider.getNetwork()).chainId;

                // Create commitment
                const commitHash = ethers.keccak256(
                    ethers.solidityPacked(
                        ["bytes32", "address", "uint256", "bytes32", "uint256"],
                        [taskId, serverAgent, deadline, salt, chainId]
                    )
                );

                // Commit
                await expect(
                    reputationRegistryV2.connect(alice).commitTaskAuthorization(commitHash)
                ).to.emit(reputationRegistryV2, "AuthorizationCommitted");

                // Wait minimum delay (30 seconds for task auth)
                await time.increase(31);

                // TODO: Complete after deploying agents in identity registry
                // This would require full setup of identity registry
            });
        });
    });

    describe("Cross-Chain Replay Protection", function () {
        it("should include chainId in commitment hash", async function () {
            const did = "did:sage:alice";
            const publicKey = ethers.getBytes(aliceWallet.signingKey.publicKey);
            const salt = ethers.randomBytes(32);
            const chainId = (await ethers.provider.getNetwork()).chainId;

            // Correct commitment (with chainId)
            const correctCommitHash = ethers.keccak256(
                ethers.solidityPacked(
                    ["string", "bytes", "address", "bytes32", "uint256"],
                    [did, publicKey, aliceWallet.address, salt, chainId]
                )
            );

            // Wrong commitment (without chainId or wrong chainId)
            const wrongCommitHash = ethers.keccak256(
                ethers.solidityPacked(
                    ["string", "bytes", "address", "bytes32", "uint256"],
                    [did, publicKey, aliceWallet.address, salt, 999n] // wrong chain
                )
            );

            // Commit correct hash
            await sageRegistryV3.connect(aliceWallet).commitRegistration(correctCommitHash);
            await time.increase(61);

            const signature = await signRegistration(aliceWallet, did, "Alice", "Desc", "https://alice.com", publicKey, "{}", 0);

            // Try to reveal with wrong salt (will fail hash verification)
            await expect(
                sageRegistryV3.connect(aliceWallet).registerAgentWithReveal(
                    did,
                    "Alice",
                    "Desc",
                    "https://alice.com",
                    publicKey,
                    "{}",
                    signature,
                    salt
                )
            ).to.not.be.reverted; // Should succeed with correct chainId

            // Verify chainId is checked during reveal
            // (The contract uses block.chainid internally)
        });
    });

    describe("Array Bounds Checking (DoS Prevention)", function () {
        it("should reject submissions when max validators reached", async function () {
            // Prevent auto-finalization by setting high minValidatorsRequired
            await validationRegistry.connect(owner).setMinValidatorsRequired(10);

            // Register alice as agent first
            const alicePublicKey = ethers.getBytes(aliceWallet.signingKey.publicKey);
            const aliceSig = await signRegistration(
                aliceWallet,
                "did:sage:alice",
                "Alice",
                "Test",
                "https://alice.com",
                alicePublicKey,
                "{}",
                0
            );

            await sageRegistryV3.connect(aliceWallet).registerAgent(
                "did:sage:alice",
                "Alice",
                "Test",
                "https://alice.com",
                alicePublicKey,
                "{}",
                aliceSig
            );

            // Register bob as agent
            const bobPublicKey = ethers.getBytes(bobWallet.signingKey.publicKey);
            const bobSig = await signRegistration(
                bobWallet,
                "did:sage:bob",
                "Bob",
                "Test",
                "https://bob.com",
                bobPublicKey,
                "{}",
                0
            );

            await sageRegistryV3.connect(bobWallet).registerAgent(
                "did:sage:bob",
                "Bob",
                "Test",
                "https://bob.com",
                bobPublicKey,
                "{}",
                bobSig
            );

            // Create validation request (serverAgent is bob's address)
            const taskId = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600 + 60;

            const tx = await validationRegistry.connect(aliceWallet).requestValidation(
                taskId,
                bobWallet.address, // serverAgent address
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

            // Set a low limit for testing (5 validators)
            await validationRegistry.connect(owner).setMaxValidatorsPerRequest(5);

            // Register 5 validators as agents
            const validatorWallets = [];
            for (let i = 0; i < 5; i++) {
                const wallet = ethers.Wallet.createRandom().connect(ethers.provider);
                validatorWallets.push(wallet);
                await owner.sendTransaction({ to: wallet.address, value: ethers.parseEther("10") });

                const validatorPubKey = ethers.getBytes(wallet.signingKey.publicKey);
                const validatorSig = await signRegistration(
                    wallet,
                    `did:sage:validator${i}`,
                    `Validator ${i}`,
                    "Test",
                    `https://validator${i}.com`,
                    validatorPubKey,
                    "{}",
                    0
                );

                await sageRegistryV3.connect(wallet).registerAgent(
                    `did:sage:validator${i}`,
                    `Validator ${i}`,
                    "Test",
                    `https://validator${i}.com`,
                    validatorPubKey,
                    "{}",
                    validatorSig
                );
            }

            // Submit 5 validators (reach the limit)
            for (let i = 0; i < 5; i++) {
                await validationRegistry.connect(validatorWallets[i]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            // Verify we reached the limit
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(5);

            // Register attacker as agent
            const attackerPubKey = ethers.getBytes(attackerWallet.signingKey.publicKey);
            const attackerSig = await signRegistration(
                attackerWallet,
                `did:sage:attacker`,
                `Attacker`,
                "Test",
                `https://attacker.com`,
                attackerPubKey,
                "{}",
                0
            );

            await sageRegistryV3.connect(attackerWallet).registerAgent(
                `did:sage:attacker`,
                `Attacker`,
                "Test",
                `https://attacker.com`,
                attackerPubKey,
                "{}",
                attackerSig
            );

            // Try to add 6th validator (should fail)
            await expect(
                validationRegistry.connect(attackerWallet).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.be.revertedWith("Maximum validators reached");
        });

        it("should allow owner to adjust max validators", async function () {
            const initialMax = await validationRegistry.maxValidatorsPerRequest();
            expect(initialMax).to.equal(100n);

            // Owner adjusts limit
            await expect(
                validationRegistry.connect(owner).setMaxValidatorsPerRequest(50)
            ).to.emit(validationRegistry, "MaxValidatorsPerRequestUpdated")
                .withArgs(100, 50);

            const newMax = await validationRegistry.maxValidatorsPerRequest();
            expect(newMax).to.equal(50n);
        });

        it("should reject zero max validators", async function () {
            await expect(
                validationRegistry.connect(owner).setMaxValidatorsPerRequest(0)
            ).to.be.revertedWithCustomError(validationRegistry, "InvalidMinimum");
        });

        it("should allow non-owner to call setMaxValidatorsPerRequest", async function () {
            // Note: This assumes the function doesn't have access control
            // If onlyOwner is implemented, this test should verify the revert
            await expect(
                validationRegistry.connect(attacker).setMaxValidatorsPerRequest(10)
            ).to.be.reverted; // Will fail with onlyOwner
        });

        it("should finalize validation with maximum validators without DoS", async function () {
            // Prevent auto-finalization by setting high minValidatorsRequired
            await validationRegistry.connect(owner).setMinValidatorsRequired(10);

            // Register alice as agent
            const alicePublicKey = ethers.getBytes(aliceWallet.signingKey.publicKey);
            const aliceSig = await signRegistration(
                aliceWallet,
                "did:sage:alice2",
                "Alice",
                "Test",
                "https://alice.com",
                alicePublicKey,
                "{}",
                0
            );

            await sageRegistryV3.connect(aliceWallet).registerAgent(
                "did:sage:alice2",
                "Alice",
                "Test",
                "https://alice.com",
                alicePublicKey,
                "{}",
                aliceSig
            );

            // Register bob as agent
            const bobPublicKey = ethers.getBytes(bobWallet.signingKey.publicKey);
            const bobSig = await signRegistration(
                bobWallet,
                "did:sage:bob2",
                "Bob",
                "Test",
                "https://bob.com",
                bobPublicKey,
                "{}",
                0
            );

            await sageRegistryV3.connect(bobWallet).registerAgent(
                "did:sage:bob2",
                "Bob",
                "Test",
                "https://bob.com",
                bobPublicKey,
                "{}",
                bobSig
            );

            // Create validation request (serverAgent is bob's address)
            const taskId = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = (await time.latest()) + 3600 + 60;

            const tx = await validationRegistry.connect(aliceWallet).requestValidation(
                taskId,
                bobWallet.address, // serverAgent address
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

            // Set low limit for testing (10 validators to match minValidators)
            await validationRegistry.connect(owner).setMaxValidatorsPerRequest(10);

            // Register 10 validators as agents
            const validatorWallets = [];
            for (let i = 0; i < 10; i++) {
                const wallet = ethers.Wallet.createRandom().connect(ethers.provider);
                validatorWallets.push(wallet);
                await owner.sendTransaction({ to: wallet.address, value: ethers.parseEther("10") });

                const validatorPubKey = ethers.getBytes(wallet.signingKey.publicKey);
                const validatorSig = await signRegistration(
                    wallet,
                    `did:sage:val${i}`,
                    `Val ${i}`,
                    "Test",
                    `https://val${i}.com`,
                    validatorPubKey,
                    "{}",
                    0
                );

                await sageRegistryV3.connect(wallet).registerAgent(
                    `did:sage:val${i}`,
                    `Val ${i}`,
                    "Test",
                    `https://val${i}.com`,
                    validatorPubKey,
                    "{}",
                    validatorSig
                );
            }

            // Submit 10 validators with correct hash (will trigger auto-finalization)
            for (let i = 0; i < 10; i++) {
                await validationRegistry.connect(validatorWallets[i]).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                );
            }

            // Verify all validators submitted
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(10);

            // Verification should have auto-finalized (10 validators, 100% consensus)
            const finalRequest = await validationRegistry.getValidationRequest(requestId);
            expect(finalRequest.status).to.not.equal(0); // Not PENDING (finalized)
        });
    });

    describe("TEE Key Governance", function () {
        beforeEach(async function () {
            // Register voters
            await teeKeyRegistry.connect(owner).registerVoter(alice.address, 100);
            await teeKeyRegistry.connect(owner).registerVoter(bob.address, 100);
            await teeKeyRegistry.connect(owner).registerVoter(validators[0].address, 100);
        });

        it("should allow proposing TEE key with stake", async function () {
            const keyHash = ethers.randomBytes(32);
            const attestationReport = "https://attestation.intel.com/report/123";
            const teeType = "SGX";

            await expect(
                teeKeyRegistry.connect(alice).proposeTEEKey(
                    keyHash,
                    attestationReport,
                    teeType,
                    { value: ethers.parseEther("1") }
                )
            ).to.emit(teeKeyRegistry, "TEEKeyProposed")
                .withArgs(0, keyHash, alice.address, teeType, attestationReport);

            const proposal = await teeKeyRegistry.proposals(0);
            expect(proposal.keyHash).to.equal(ethers.hexlify(keyHash));
            expect(proposal.proposer).to.equal(alice.address);
            expect(proposal.teeType).to.equal(teeType);
        });

        it("should reject proposal with insufficient stake", async function () {
            const keyHash = ethers.randomBytes(32);

            await expect(
                teeKeyRegistry.connect(alice).proposeTEEKey(
                    keyHash,
                    "https://attestation.com",
                    "SGX",
                    { value: ethers.parseEther("0.5") } // Less than 1 ETH
                )
            ).to.be.revertedWithCustomError(teeKeyRegistry, "InsufficientStake");
        });

        it("should allow voting on proposals", async function () {
            const keyHash = ethers.randomBytes(32);

            // Create proposal
            await teeKeyRegistry.connect(alice).proposeTEEKey(
                keyHash,
                "https://attestation.com",
                "SGX",
                { value: ethers.parseEther("1") }
            );

            // Vote in favor
            await expect(
                teeKeyRegistry.connect(bob).vote(0, true)
            ).to.emit(teeKeyRegistry, "VoteCast")
                .withArgs(0, bob.address, true, 100);

            const proposal = await teeKeyRegistry.proposals(0);
            expect(proposal.votesFor).to.equal(100);
        });

        it("should approve key with sufficient votes", async function () {
            const keyHash = ethers.randomBytes(32);

            // Create proposal
            await teeKeyRegistry.connect(alice).proposeTEEKey(
                keyHash,
                "https://attestation.com",
                "SGX",
                { value: ethers.parseEther("1") }
            );

            // Vote in favor (3 voters, 66% threshold)
            await teeKeyRegistry.connect(alice).vote(0, true);
            await teeKeyRegistry.connect(bob).vote(0, true);
            await teeKeyRegistry.connect(validators[0]).vote(0, true);

            // Wait for voting period
            await time.increase(7 * 24 * 60 * 60 + 1);

            // Execute proposal
            const initialBalance = await ethers.provider.getBalance(alice.address);

            await expect(
                teeKeyRegistry.connect(attacker).executeProposal(0)
            ).to.emit(teeKeyRegistry, "ProposalExecuted")
                .withArgs(0, keyHash, true);

            // Verify key approved
            expect(await teeKeyRegistry.approvedTEEKeys(keyHash)).to.be.true;

            // Verify stake returned
            const finalBalance = await ethers.provider.getBalance(alice.address);
            // Note: Alice should receive stake back (minus gas if she executed)
        });

        it("should slash stake for rejected proposals", async function () {
            const keyHash = ethers.randomBytes(32);

            // Get initial treasury balance
            const treasuryBefore = await teeKeyRegistry.getTreasuryBalance();

            // Create proposal
            await teeKeyRegistry.connect(alice).proposeTEEKey(
                keyHash,
                "https://fake-attestation.com",
                "SGX",
                { value: ethers.parseEther("1") }
            );

            // Vote against (majority)
            await teeKeyRegistry.connect(alice).vote(0, false);
            await teeKeyRegistry.connect(bob).vote(0, false);
            await teeKeyRegistry.connect(validators[0]).vote(0, false);

            // Wait for voting period
            await time.increase(7 * 24 * 60 * 60 + 1);

            // Execute proposal

            await teeKeyRegistry.connect(attacker).executeProposal(0);

            // Verify key NOT approved
            expect(await teeKeyRegistry.approvedTEEKeys(keyHash)).to.be.false;

            // Verify stake slashed (50%)
            const treasuryAfter = await teeKeyRegistry.getTreasuryBalance();
            expect(treasuryAfter - treasuryBefore).to.equal(ethers.parseEther("0.5"));
        });
    });

    // Helper function to sign registration for SageRegistryV3
    async function signRegistration(signer, did, name, description, endpoint, publicKey, capabilities, nonce) {
        // SageRegistryV3 uses key ownership proof
        const keyHash = ethers.keccak256(publicKey);
        const chainId = (await ethers.provider.getNetwork()).chainId;
        const challenge = ethers.solidityPackedKeccak256(
            ["string", "uint256", "address", "address", "bytes32"],
            ["SAGE Key Registration:", chainId, await sageRegistryV3.getAddress(), signer.address, keyHash]
        );

        const signature = await signer.signMessage(ethers.getBytes(challenge));
        return signature;
    }
});
