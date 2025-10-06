const { expect } = require("chai");
const { ethers } = require("hardhat");

/**
 * Security Test Suite: Reentrancy Protection
 *
 * Tests for CRITICAL-1 and CRITICAL-2 fixes:
 * - Reentrancy vulnerability in reward distribution
 * - Reentrancy in disputed case returns
 *
 * Reference: SECURITY-AUDIT-REPORT.md
 */
describe("Security: Reentrancy Protection", function () {
    let sageRegistry;
    let identityRegistry;
    let reputationRegistry;
    let validationRegistry;
    let reentrancyAttacker;
    let owner;
    let agent, agentWallet;
    let validator1, validator2;
    let client;

    // Helper function to create a test wallet with proper key format
    function createTestWallet() {
        const wallet = ethers.Wallet.createRandom();
        const publicKeyRaw = wallet.signingKey.publicKey;
        const publicKey = ethers.getBytes(publicKeyRaw);
        return { wallet, publicKey };
    }

    // Helper function to create registration signature
    async function createRegistrationSignature(wallet, publicKey) {
        const keyHash = ethers.keccak256(publicKey);
        const challenge = ethers.solidityPackedKeccak256(
            ["string", "uint256", "address", "address", "bytes32"],
            [
                "SAGE Key Registration:",
                (await ethers.provider.getNetwork()).chainId,
                await sageRegistry.getAddress(),
                wallet.address,
                keyHash
            ]
        );
        return await wallet.signMessage(ethers.getBytes(challenge));
    }

    beforeEach(async function () {
        [owner, client, validator1, validator2] = await ethers.getSigners();

        // Create test wallet for agent
        const agentData = createTestWallet();
        agentWallet = agentData.wallet;
        agent = await ethers.getImpersonatedSigner(agentWallet.address);

        // Fund agent
        await owner.sendTransaction({
            to: agentWallet.address,
            value: ethers.parseEther("10")
        });

        // Deploy SageRegistryV2
        const SageRegistryV2 = await ethers.getContractFactory("SageRegistryV2");
        sageRegistry = await SageRegistryV2.deploy();
        await sageRegistry.waitForDeployment();

        // Deploy ERC8004IdentityRegistry
        const ERC8004IdentityRegistry = await ethers.getContractFactory("ERC8004IdentityRegistry");
        identityRegistry = await ERC8004IdentityRegistry.deploy(await sageRegistry.getAddress());
        await identityRegistry.waitForDeployment();

        // Deploy ERC8004ReputationRegistry
        const ERC8004ReputationRegistry = await ethers.getContractFactory("ERC8004ReputationRegistry");
        reputationRegistry = await ERC8004ReputationRegistry.deploy(await identityRegistry.getAddress());
        await reputationRegistry.waitForDeployment();

        // Deploy ERC8004ValidationRegistry
        const ERC8004ValidationRegistry = await ethers.getContractFactory("ERC8004ValidationRegistry");
        validationRegistry = await ERC8004ValidationRegistry.deploy(
            await identityRegistry.getAddress(),
            await reputationRegistry.getAddress()
        );
        await validationRegistry.waitForDeployment();

        // Link reputation registry to validation registry
        await reputationRegistry.setValidationRegistry(await validationRegistry.getAddress());

        // Deploy ReentrancyAttacker
        const ReentrancyAttacker = await ethers.getContractFactory("ReentrancyAttacker");
        reentrancyAttacker = await ReentrancyAttacker.deploy(await validationRegistry.getAddress());
        await reentrancyAttacker.waitForDeployment();

        // Register agent
        const agentDid = "did:sage:test123";
        const agentName = "Test Agent";
        const agentDescription = "Security test agent";
        const agentEndpoint = "https://example.com/agent";
        const agentCapabilities = "validation,testing";
        const signature = await createRegistrationSignature(agentWallet, agentData.publicKey);

        await sageRegistry.connect(agent).registerAgent(
            agentDid,
            agentName,
            agentDescription,
            agentEndpoint,
            agentData.publicKey,
            agentCapabilities,
            signature
        );
    });

    describe("CRITICAL-1: Reentrancy in requestValidation", function () {
        it("Should prevent reentrancy attack during validation request", async function () {
            const taskId = ethers.id("test-task-1");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Fund attacker
            await owner.sendTransaction({
                to: await reentrancyAttacker.getAddress(),
                value: ethers.parseEther("1")
            });

            // Attempt reentrancy attack
            await expect(
                reentrancyAttacker.attackRequestValidation(
                    taskId,
                    agentWallet.address,
                    dataHash,
                    1, // STAKE (NONE=0, STAKE=1, TEE=2, HYBRID=3)
                    deadline,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.not.be.reverted;

            // Verify attackCount stayed at 0 (reentrancy was prevented)
            const attackCount = await reentrancyAttacker.attackCount();
            expect(attackCount).to.equal(0);
        });

        it("Should allow normal validation request", async function () {
            const taskId = ethers.id("test-task-2");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            await expect(
                validationRegistry.connect(client).requestValidation(
                    taskId,
                    agentWallet.address,
                    dataHash,
                    1, // STAKE
                    deadline,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.not.be.reverted;
        });
    });

    describe("CRITICAL-2: Reentrancy in submitStakeValidation", function () {
        let requestId;
        let dataHash;

        beforeEach(async function () {
            // Create a validation request
            const taskId = ethers.id("test-task-3");
            dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );
        });

        it("Should prevent reentrancy attack during stake validation submission", async function () {
            // Fund attacker
            await owner.sendTransaction({
                to: await reentrancyAttacker.getAddress(),
                value: ethers.parseEther("1")
            });

            // Attempt reentrancy attack
            await expect(
                reentrancyAttacker.attackSubmitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.not.be.reverted;

            // Verify attackCount stayed at 0 (reentrancy was prevented)
            const attackCount = await reentrancyAttacker.attackCount();
            expect(attackCount).to.equal(0);
        });

        it("Should allow normal stake validation submission", async function () {
            await expect(
                validationRegistry.connect(validator1).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.not.be.reverted;
        });

        it("Should prevent multiple submissions from same validator", async function () {
            // First submission should succeed
            await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Second submission should fail
            await expect(
                validationRegistry.connect(validator1).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: ethers.parseEther("0.1") }
                )
            ).to.be.revertedWith("Already responded");
        });
    });

    describe("CRITICAL: Complete Validation Flow Protection", function () {
        it("Should handle complete validation flow without reentrancy issues", async function () {
            const taskId = ethers.id("test-task-complete");
            const dataHash = ethers.id("test-data-complete");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Step 1: Request validation
            const requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            // Step 2: Validator 1 submits (correct)
            await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Step 3: Validator 2 submits (correct)
            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Step 4: Check validation completed
            const isComplete = await validationRegistry.isValidationComplete(requestId);
            expect(isComplete).to.be.true;

            // Step 5: Get validator stats (protected from reentrancy)
            const stats1 = await validationRegistry.getValidatorStats(validator1.address);
            expect(stats1.totalValidations).to.be.gt(0);
        });
    });

    describe("Gas Costs with ReentrancyGuard", function () {
        it("Should measure gas cost increase from ReentrancyGuard", async function () {
            const taskId = ethers.id("test-task-gas");
            const dataHash = ethers.id("test-data-gas");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Measure gas for requestValidation
            const tx1 = await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );
            const receipt1 = await tx1.wait();
            console.log(`      Gas for requestValidation: ${receipt1.gasUsed.toString()}`);

            // Get requestId
            const requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            // Measure gas for submitStakeValidation
            const tx2 = await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );
            const receipt2 = await tx2.wait();
            console.log(`      Gas for submitStakeValidation: ${receipt2.gasUsed.toString()}`);

            // ReentrancyGuard adds approximately 2,100-2,300 gas per protected function
            expect(receipt1.gasUsed).to.be.lt(200000); // Should be reasonable
            expect(receipt2.gasUsed).to.be.lt(300000); // Should be reasonable
        });
    });
});
