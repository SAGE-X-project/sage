const { expect } = require("chai");
const { ethers } = require("hardhat");

/**
 * Security Test Suite: Pull Payment Pattern
 *
 * Tests for CRITICAL-2 fix: Pull payment pattern implementation
 * Ensures users must withdraw their own funds instead of receiving automatic transfers
 *
 * Reference: SECURITY-AUDIT-REPORT.md, SECURITY-REMEDIATION-ROADMAP.md
 */
describe("Security: Pull Payment Pattern", function () {
    let sageRegistry;
    let identityRegistry;
    let reputationRegistry;
    let validationRegistry;
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

    describe("Pull Payment - Withdraw Function", function () {
        it("Should allow users to withdraw their pending balance", async function () {
            const taskId = ethers.id("test-task-withdraw");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Create validation request
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

            // Validator submits correct validation
            await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Second validator submits correct validation to complete
            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Check validator has pending withdrawal
            const pendingAmount = await validationRegistry.getWithdrawableAmount(validator1.address);
            expect(pendingAmount).to.be.gt(0);

            // Get initial balance
            const initialBalance = await ethers.provider.getBalance(validator1.address);

            // Withdraw funds
            const tx = await validationRegistry.connect(validator1).withdraw();
            const receipt = await tx.wait();
            const gasUsed = receipt.gasUsed * receipt.gasPrice;

            // Check new balance
            const finalBalance = await ethers.provider.getBalance(validator1.address);
            expect(finalBalance).to.equal(initialBalance + pendingAmount - gasUsed);

            // Check pending withdrawal is now zero
            const newPending = await validationRegistry.getWithdrawableAmount(validator1.address);
            expect(newPending).to.equal(0);
        });

        it("Should revert when withdrawing with zero balance", async function () {
            await expect(
                validationRegistry.connect(validator1).withdraw()
            ).to.be.revertedWith("No funds to withdraw");
        });

        it("Should handle multiple validators withdrawing independently", async function () {
            const taskId = ethers.id("test-task-multi-withdraw");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Create validation request
            await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.2") }
            );

            const requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.2") }
            );

            // Both validators submit
            await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Check both have pending withdrawals
            const pending1 = await validationRegistry.getWithdrawableAmount(validator1.address);
            const pending2 = await validationRegistry.getWithdrawableAmount(validator2.address);

            expect(pending1).to.be.gt(0);
            expect(pending2).to.be.gt(0);

            // Validator1 withdraws
            await validationRegistry.connect(validator1).withdraw();

            // Validator1 should have 0 pending
            const newPending1 = await validationRegistry.getWithdrawableAmount(validator1.address);
            expect(newPending1).to.equal(0);

            // Validator2 should still have pending amount
            const stillPending2 = await validationRegistry.getWithdrawableAmount(validator2.address);
            expect(stillPending2).to.equal(pending2);

            // Validator2 withdraws
            await validationRegistry.connect(validator2).withdraw();

            // Both should now have 0 pending
            const finalPending2 = await validationRegistry.getWithdrawableAmount(validator2.address);
            expect(finalPending2).to.equal(0);
        });

        it("Should emit WithdrawalProcessed event", async function () {
            const taskId = ethers.id("test-task-event");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Create and complete validation
            await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            const requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Withdraw and check event
            const pendingAmount = await validationRegistry.getWithdrawableAmount(validator1.address);

            await expect(validationRegistry.connect(validator1).withdraw())
                .to.emit(validationRegistry, "WithdrawalProcessed")
                .withArgs(validator1.address, pendingAmount);
        });
    });

    describe("Pull Payment - No Direct Transfers", function () {
        it("Should not send funds directly during validation completion", async function () {
            const taskId = ethers.id("test-task-no-direct");
            const dataHash = ethers.id("test-data");
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            // Get initial balance
            const initialBalance = await ethers.provider.getBalance(validator1.address);

            // Create validation request
            await validationRegistry.connect(client).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            const requestId = await validationRegistry.connect(client).requestValidation.staticCall(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );

            // Validator1 submits
            const tx1 = await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );
            const receipt1 = await tx1.wait();
            const gas1 = receipt1.gasUsed * receipt1.gasPrice;

            // Validator2 completes validation
            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Validator1 balance should only reflect gas costs (no direct transfer)
            const balanceAfterValidation = await ethers.provider.getBalance(validator1.address);
            expect(balanceAfterValidation).to.equal(initialBalance - ethers.parseEther("0.1") - gas1);

            // Funds should be in pending withdrawals
            const pending = await validationRegistry.getWithdrawableAmount(validator1.address);
            expect(pending).to.be.gt(0);
        });
    });
});
