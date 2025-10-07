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
    let clientWallet, val1Wallet, val2Wallet; // Actual registered wallets

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

        // Set minimum validators required to 2 for testing multiple validator scenarios
        await validationRegistry.setMinValidatorsRequired(2);

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

        // Register client as agent (using actual signer's key)
        clientWallet = ethers.Wallet.createRandom().connect(ethers.provider);
        const clientPubKey = ethers.getBytes(clientWallet.signingKey.publicKey);
        await owner.sendTransaction({ to: clientWallet.address, value: ethers.parseEther("10") });
        const clientSig = await createRegistrationSignature(clientWallet, clientPubKey);

        await sageRegistry.connect(clientWallet).registerAgent(
            "did:sage:client123",
            "Client Agent",
            "Client test agent",
            "https://example.com/client",
            clientPubKey,
            "client",
            clientSig
        );

        // Register validators as agents (using actual signer's keys)
        val1Wallet = ethers.Wallet.createRandom().connect(ethers.provider);
        const val1PubKey = ethers.getBytes(val1Wallet.signingKey.publicKey);
        await owner.sendTransaction({ to: val1Wallet.address, value: ethers.parseEther("10") });
        const val1Sig = await createRegistrationSignature(val1Wallet, val1PubKey);

        await sageRegistry.connect(val1Wallet).registerAgent(
            "did:sage:validator1",
            "Validator 1",
            "Validator test agent 1",
            "https://example.com/val1",
            val1PubKey,
            "validation",
            val1Sig
        );

        val2Wallet = ethers.Wallet.createRandom().connect(ethers.provider);
        const val2PubKey = ethers.getBytes(val2Wallet.signingKey.publicKey);
        await owner.sendTransaction({ to: val2Wallet.address, value: ethers.parseEther("10") });
        const val2Sig = await createRegistrationSignature(val2Wallet, val2PubKey);

        await sageRegistry.connect(val2Wallet).registerAgent(
            "did:sage:validator2",
            "Validator 2",
            "Validator test agent 2",
            "https://example.com/val2",
            val2PubKey,
            "validation",
            val2Sig
        );
    });

    describe("Pull Payment - Withdraw Function", function () {
        it("Should allow users to withdraw their pending balance", async function () {
            const taskId = ethers.id("test-task-withdraw");
            const dataHash = ethers.id("test-data");
            const currentBlock = await ethers.provider.getBlock('latest'); const deadline = currentBlock.timestamp + 3600 + 60;

            // Create validation request and get requestId from event
            const txRequest = await validationRegistry.connect(clientWallet).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );
            const receiptRequest = await txRequest.wait();
            const eventRequest = receiptRequest.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(eventRequest).args.requestId;

            // Validator submits correct validation
            await validationRegistry.connect(val1Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Second validator submits correct validation to complete
            await validationRegistry.connect(val2Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Check validator has pending withdrawal
            const pendingAmount = await validationRegistry.getWithdrawableAmount(val1Wallet.address);
            expect(pendingAmount).to.be.gt(0);

            // Get initial balance
            const initialBalance = await ethers.provider.getBalance(val1Wallet.address);

            // Withdraw funds
            const txWithdraw = await validationRegistry.connect(val1Wallet).withdraw();
            const receiptWithdraw = await txWithdraw.wait();
            const gasUsed = receiptWithdraw.gasUsed * receiptWithdraw.gasPrice;

            // Check new balance
            const finalBalance = await ethers.provider.getBalance(val1Wallet.address);
            expect(finalBalance).to.equal(initialBalance + pendingAmount - gasUsed);

            // Check pending withdrawal is now zero
            const newPending = await validationRegistry.getWithdrawableAmount(val1Wallet.address);
            expect(newPending).to.equal(0);
        });

        it("Should revert when withdrawing with zero balance", async function () {
            await expect(
                validationRegistry.connect(val1Wallet).withdraw()
            ).to.be.revertedWithCustomError(validationRegistry, "NoFundsToWithdraw");
        });

        it("Should handle multiple validators withdrawing independently", async function () {
            const taskId = ethers.id("test-task-multi-withdraw");
            const dataHash = ethers.id("test-data");
            const currentBlock = await ethers.provider.getBlock('latest'); const deadline = currentBlock.timestamp + 3600 + 60;

            // Create validation request and get requestId from event
            const tx = await validationRegistry.connect(clientWallet).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.2") }
            );
            const receipt = await tx.wait();
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            // Both validators submit
            await validationRegistry.connect(val1Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(val2Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Check both have pending withdrawals
            const pending1 = await validationRegistry.getWithdrawableAmount(val1Wallet.address);
            const pending2 = await validationRegistry.getWithdrawableAmount(val2Wallet.address);

            expect(pending1).to.be.gt(0);
            expect(pending2).to.be.gt(0);

            // Validator1 withdraws
            await validationRegistry.connect(val1Wallet).withdraw();

            // Validator1 should have 0 pending
            const newPending1 = await validationRegistry.getWithdrawableAmount(val1Wallet.address);
            expect(newPending1).to.equal(0);

            // Validator2 should still have pending amount
            const stillPending2 = await validationRegistry.getWithdrawableAmount(val2Wallet.address);
            expect(stillPending2).to.equal(pending2);

            // Validator2 withdraws
            await validationRegistry.connect(val2Wallet).withdraw();

            // Both should now have 0 pending
            const finalPending2 = await validationRegistry.getWithdrawableAmount(val2Wallet.address);
            expect(finalPending2).to.equal(0);
        });

        it("Should emit WithdrawalProcessed event", async function () {
            const taskId = ethers.id("test-task-event");
            const dataHash = ethers.id("test-data");
            const currentBlock = await ethers.provider.getBlock('latest'); const deadline = currentBlock.timestamp + 3600 + 60;

            // Create and complete validation - get requestId from event
            const tx = await validationRegistry.connect(clientWallet).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );
            const receipt = await tx.wait();
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(event).args.requestId;

            await validationRegistry.connect(val1Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            await validationRegistry.connect(val2Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Withdraw and check event
            const pendingAmount = await validationRegistry.getWithdrawableAmount(val1Wallet.address);

            await expect(validationRegistry.connect(val1Wallet).withdraw())
                .to.emit(validationRegistry, "WithdrawalProcessed")
                .withArgs(val1Wallet.address, pendingAmount);
        });
    });

    describe("Pull Payment - No Direct Transfers", function () {
        it("Should not send funds directly during validation completion", async function () {
            const taskId = ethers.id("test-task-no-direct");
            const dataHash = ethers.id("test-data");
            const currentBlock = await ethers.provider.getBlock('latest'); const deadline = currentBlock.timestamp + 3600 + 60;

            // Get initial balance
            const initialBalance = await ethers.provider.getBalance(val1Wallet.address);

            // Create validation request and get requestId from event
            const txReq = await validationRegistry.connect(clientWallet).requestValidation(
                taskId,
                agentWallet.address,
                dataHash,
                1, // STAKE
                deadline,
                { value: ethers.parseEther("0.1") }
            );
            const receiptReq = await txReq.wait();
            const eventReq = receiptReq.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const requestId = validationRegistry.interface.parseLog(eventReq).args.requestId;

            // Validator1 submits
            const tx1 = await validationRegistry.connect(val1Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );
            const receipt1 = await tx1.wait();
            const gas1 = receipt1.gasUsed * receipt1.gasPrice;

            // Validator2 completes validation
            await validationRegistry.connect(val2Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: ethers.parseEther("0.1") }
            );

            // Validator1 balance should only reflect gas costs (no direct transfer)
            const balanceAfterValidation = await ethers.provider.getBalance(val1Wallet.address);
            expect(balanceAfterValidation).to.equal(initialBalance - ethers.parseEther("0.1") - gas1);

            // Funds should be in pending withdrawals
            const pending = await validationRegistry.getWithdrawableAmount(val1Wallet.address);
            expect(pending).to.be.gt(0);
        });
    });
});
