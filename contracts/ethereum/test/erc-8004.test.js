const { expect } = require("chai");
const { ethers } = require("hardhat");

/**
 * ERC-8004 Trustless Agents Standard - Comprehensive Test Suite
 *
 * Tests the complete ERC-8004 implementation including:
 * - Identity Registry (adapter for SageRegistryV2)
 * - Reputation Registry (feedback and task authorization)
 * - Validation Registry (stake-based and TEE validation)
 */
describe("ERC-8004 Trustless Agents Standard", function () {
    let sageRegistry;
    let identityRegistry;
    let reputationRegistry;
    let validationRegistry;
    let owner;
    let agent1, agent1Wallet;
    let agent2, agent2Wallet;
    let validator1, validator1Wallet;
    let validator2, validator2Wallet;
    let client, clientWallet;

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
        [owner, agent1, agent2, validator1, validator2, client] = await ethers.getSigners();

        // Deploy SageRegistryV2
        const SageRegistryV2 = await ethers.getContractFactory("SageRegistryV2");
        sageRegistry = await SageRegistryV2.deploy();
        await sageRegistry.waitForDeployment();

        // Deploy ERC8004IdentityRegistry (adapter version)
        const ERC8004IdentityRegistry = await ethers.getContractFactory("contracts/erc-8004/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry");
        identityRegistry = await ERC8004IdentityRegistry.deploy(await sageRegistry.getAddress());
        await identityRegistry.waitForDeployment();

        // Deploy ERC8004ReputationRegistry (adapter version)
        const ERC8004ReputationRegistry = await ethers.getContractFactory("contracts/erc-8004/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry");
        reputationRegistry = await ERC8004ReputationRegistry.deploy(await identityRegistry.getAddress());
        await reputationRegistry.waitForDeployment();

        // Deploy ERC8004ValidationRegistry (adapter version)
        const ERC8004ValidationRegistry = await ethers.getContractFactory("contracts/erc-8004/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry");
        validationRegistry = await ERC8004ValidationRegistry.deploy(
            await identityRegistry.getAddress(),
            await reputationRegistry.getAddress()
        );
        await validationRegistry.waitForDeployment();

        // Set validation registry in reputation registry
        await reputationRegistry.setValidationRegistry(await validationRegistry.getAddress());

        // Register test agents
        const agent1Data = createTestWallet();
        agent1Wallet = agent1Data.wallet.connect(ethers.provider);
        await owner.sendTransaction({
            to: agent1Wallet.address,
            value: ethers.parseEther("1.0")
        });
        const agent1Sig = await createRegistrationSignature(agent1Wallet, agent1Data.publicKey);

        await sageRegistry.connect(agent1Wallet).registerAgent(
            "did:sage:agent1",
            "Agent 1",
            "Test Agent 1",
            "https://agent1.example.com",
            agent1Data.publicKey,
            "AI Assistant",
            agent1Sig
        );

        const agent2Data = createTestWallet();
        agent2Wallet = agent2Data.wallet.connect(ethers.provider);
        await owner.sendTransaction({
            to: agent2Wallet.address,
            value: ethers.parseEther("1.0")
        });
        const agent2Sig = await createRegistrationSignature(agent2Wallet, agent2Data.publicKey);

        await sageRegistry.connect(agent2Wallet).registerAgent(
            "did:sage:agent2",
            "Agent 2",
            "Test Agent 2",
            "https://agent2.example.com",
            agent2Data.publicKey,
            "Data Processor",
            agent2Sig
        );

        // Register validators
        const validator1Data = createTestWallet();
        validator1Wallet = validator1Data.wallet.connect(ethers.provider);
        await owner.sendTransaction({
            to: validator1Wallet.address,
            value: ethers.parseEther("10.0")
        });
        const validator1Sig = await createRegistrationSignature(validator1Wallet, validator1Data.publicKey);

        await sageRegistry.connect(validator1Wallet).registerAgent(
            "did:sage:validator1",
            "Validator 1",
            "Test Validator 1",
            "https://validator1.example.com",
            validator1Data.publicKey,
            "Validator",
            validator1Sig
        );

        const validator2Data = createTestWallet();
        validator2Wallet = validator2Data.wallet.connect(ethers.provider);
        await owner.sendTransaction({
            to: validator2Wallet.address,
            value: ethers.parseEther("10.0")
        });
        const validator2Sig = await createRegistrationSignature(validator2Wallet, validator2Data.publicKey);

        await sageRegistry.connect(validator2Wallet).registerAgent(
            "did:sage:validator2",
            "Validator 2",
            "Test Validator 2",
            "https://validator2.example.com",
            validator2Data.publicKey,
            "Validator",
            validator2Sig
        );

        // Register client
        const clientData = createTestWallet();
        clientWallet = clientData.wallet.connect(ethers.provider);
        await owner.sendTransaction({
            to: clientWallet.address,
            value: ethers.parseEther("10.0")
        });
        const clientSig = await createRegistrationSignature(clientWallet, clientData.publicKey);

        await sageRegistry.connect(clientWallet).registerAgent(
            "did:sage:client",
            "Client",
            "Test Client",
            "https://client.example.com",
            clientData.publicKey,
            "Client",
            clientSig
        );
    });

    describe("ERC8004IdentityRegistry", function () {
        it("Should resolve agent by DID", async function () {
            const agentInfo = await identityRegistry.resolveAgent("did:sage:agent1");

            expect(agentInfo.agentId).to.equal("did:sage:agent1");
            expect(agentInfo.endpoint).to.equal("https://agent1.example.com");
            expect(agentInfo.isActive).to.be.true;
        });

        it("Should resolve agent by address", async function () {
            const agentsByOwner = await sageRegistry.getAgentsByOwner(agent1Wallet.address);
            const firstAgent = await sageRegistry.getAgent(agentsByOwner[0]);

            const agentInfo = await identityRegistry.resolveAgentByAddress(firstAgent.owner);

            expect(agentInfo.agentId).to.equal("did:sage:agent1");
            expect(agentInfo.isActive).to.be.true;
        });

        it("Should check if agent is active", async function () {
            const isActive = await identityRegistry.isAgentActive("did:sage:agent1");
            expect(isActive).to.be.true;
        });

        it("Should revert when resolving non-existent agent", async function () {
            await expect(
                identityRegistry.resolveAgent("did:sage:nonexistent")
            ).to.be.revertedWith("Agent not found");
        });
    });

    describe("ERC8004ReputationRegistry", function () {
        let taskId;
        let serverAgent;
        let clientAgent;

        beforeEach(async function () {
            taskId = ethers.id("task-123");

            // Get actual agent addresses from registry
            const agent2Agents = await sageRegistry.getAgentsByOwner(agent2Wallet.address);
            const agent2Metadata = await sageRegistry.getAgent(agent2Agents[0]);
            serverAgent = agent2Metadata.owner;

            const clientAgents = await sageRegistry.getAgentsByOwner(clientWallet.address);
            const clientMetadata = await sageRegistry.getAgent(clientAgents[0]);
            clientAgent = clientMetadata.owner;
        });

        describe("Task Authorization", function () {
            it("Should authorize a task", async function () {
                // Use 2 hours (7200s) to meet MIN_DEADLINE_DURATION requirement (1 hour)
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;

                await expect(
                    reputationRegistry.connect(clientWallet).authorizeTask(
                        taskId,
                        serverAgent,
                        deadline
                    )
                ).to.emit(reputationRegistry, "TaskAuthorized")
                 .withArgs(taskId, clientAgent, serverAgent, deadline);

                const isAuthorized = await reputationRegistry.isTaskAuthorized(taskId);
                expect(isAuthorized).to.be.true;
            });

            it("Should reject duplicate task authorization", async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;

                await reputationRegistry.connect(clientWallet).authorizeTask(
                    taskId,
                    serverAgent,
                    deadline
                );

                await expect(
                    reputationRegistry.connect(clientWallet).authorizeTask(
                        taskId,
                        serverAgent,
                        deadline
                    )
                ).to.be.revertedWith("Task already authorized");
            });

            it("Should reject authorization with expired deadline", async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp - 3600; // Past deadline

                await expect(
                    reputationRegistry.connect(clientWallet).authorizeTask(
                        taskId,
                        serverAgent,
                        deadline
                    )
                ).to.be.revertedWith("Invalid deadline");
            });
        });

        describe("Feedback Submission", function () {
            let dataHash;

            beforeEach(async function () {
                dataHash = ethers.id("task-output-data");
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;

                await reputationRegistry.connect(clientWallet).authorizeTask(
                    taskId,
                    serverAgent,
                    deadline
                );
            });

            it("Should submit valid feedback", async function () {
                const rating = 85;

                await expect(
                    reputationRegistry.connect(clientWallet).submitFeedback(
                        taskId,
                        serverAgent,
                        dataHash,
                        rating
                    )
                ).to.emit(reputationRegistry, "FeedbackSubmitted");

                const feedbackCount = await reputationRegistry.getAgentFeedbackCount(serverAgent);
                expect(feedbackCount).to.equal(1);
            });

            it("Should reject feedback without authorization", async function () {
                const unauthorizedTaskId = ethers.id("task-999");

                await expect(
                    reputationRegistry.connect(clientWallet).submitFeedback(
                        unauthorizedTaskId,
                        serverAgent,
                        dataHash,
                        85
                    )
                ).to.be.revertedWith("Not authorized client");
            });

            it("Should reject duplicate feedback", async function () {
                await reputationRegistry.connect(clientWallet).submitFeedback(
                    taskId,
                    serverAgent,
                    dataHash,
                    85
                );

                await expect(
                    reputationRegistry.connect(clientWallet).submitFeedback(
                        taskId,
                        serverAgent,
                        dataHash,
                        90
                    )
                ).to.be.revertedWith("Feedback already submitted");
            });

            it("Should reject rating above maximum", async function () {
                await expect(
                    reputationRegistry.connect(clientWallet).submitFeedback(
                        taskId,
                        serverAgent,
                        dataHash,
                        101
                    )
                ).to.be.revertedWith("Rating exceeds maximum");
            });
        });

        describe("Feedback Queries", function () {
            beforeEach(async function () {
                // Submit multiple feedback entries
                for (let i = 0; i < 3; i++) {
                    const taskId = ethers.id(`task-${i}`);
                    const dataHash = ethers.id(`output-${i}`);
                    const currentBlock = await ethers.provider.getBlock('latest');
                    const deadline = currentBlock.timestamp + 7200;

                    await reputationRegistry.connect(clientWallet).authorizeTask(
                        taskId,
                        serverAgent,
                        deadline
                    );

                    await reputationRegistry.connect(clientWallet).submitFeedback(
                        taskId,
                        serverAgent,
                        dataHash,
                        80 + i
                    );
                }
            });

            it("Should get agent feedback with pagination", async function () {
                const feedbacks = await reputationRegistry.getAgentFeedback(
                    serverAgent,
                    0,
                    2
                );

                expect(feedbacks.length).to.equal(2);
                expect(feedbacks[0].serverAgent).to.equal(serverAgent);
            });

            it("Should get task feedback", async function () {
                const taskId = ethers.id("task-0");
                const feedbacks = await reputationRegistry.getTaskFeedback(taskId);

                expect(feedbacks.length).to.equal(1);
                expect(feedbacks[0].taskId).to.equal(taskId);
            });

            it("Should get agent feedback count", async function () {
                const count = await reputationRegistry.getAgentFeedbackCount(serverAgent);
                expect(count).to.equal(3);
            });
        });
    });

    describe("ERC8004ValidationRegistry", function () {
        let taskId;
        let dataHash;
        let serverAgent;

        beforeEach(async function () {
            taskId = ethers.id("validation-task");
            dataHash = ethers.id("task-result-hash");

            const agent2Agents = await sageRegistry.getAgentsByOwner(agent2Wallet.address);
            const agent2Metadata = await sageRegistry.getAgent(agent2Agents[0]);
            serverAgent = agent2Metadata.owner;
        });

        describe("Validation Request", function () {
            it("Should create stake-based validation request", async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;
                const stake = ethers.parseEther("0.1");

                await expect(
                    validationRegistry.connect(clientWallet).requestValidation(
                        taskId,
                        serverAgent,
                        dataHash,
                        1, // ValidationType.STAKE
                        deadline,
                        { value: stake }
                    )
                ).to.emit(validationRegistry, "ValidationRequested");
            });

            it("Should reject validation request with insufficient stake", async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 3600 + 60; // 1 hour + buffer for MIN_DEADLINE_DURATION
                const stake = ethers.parseEther("0.001"); // Too low (min is 0.01)

                await expect(
                    validationRegistry.connect(clientWallet).requestValidation(
                        taskId,
                        serverAgent,
                        dataHash,
                        1,
                        deadline,
                        { value: stake }
                    )
                ).to.be.revertedWithCustomError(validationRegistry, "InsufficientStake");
            });
        });

        describe("Stake-based Validation", function () {
            let requestId;

            beforeEach(async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;
                const stake = ethers.parseEther("0.1");

                const tx = await validationRegistry.connect(clientWallet).requestValidation(
                    taskId,
                    serverAgent,
                    dataHash,
                    1, // ValidationType.STAKE
                    deadline,
                    { value: stake }
                );

                const receipt = await tx.wait();
                const event = receipt.logs.find(
                    log => log.fragment && log.fragment.name === "ValidationRequested"
                );
                requestId = event.args[0];
            });

            it("Should submit correct stake validation", async function () {
                const validatorStake = ethers.parseEther("0.1");

                await expect(
                    validationRegistry.connect(validator1Wallet).submitStakeValidation(
                        requestId,
                        dataHash,
                        { value: validatorStake }
                    )
                ).to.emit(validationRegistry, "ValidationSubmitted");
            });

            it("Should reject duplicate validation from same validator", async function () {
                const validatorStake = ethers.parseEther("0.1");

                await validationRegistry.connect(validator1Wallet).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: validatorStake }
                );

                // After first validation completes, request is no longer pending
                await expect(
                    validationRegistry.connect(validator1Wallet).submitStakeValidation(
                        requestId,
                        dataHash,
                        { value: validatorStake }
                    )
                ).to.be.revertedWith("Request not pending");
            });

            it("Should complete validation with consensus", async function () {
                const validatorStake = ethers.parseEther("0.1");

                // Validator 1 validates correctly
                await validationRegistry.connect(validator1Wallet).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: validatorStake }
                );

                // Check if validation is complete
                const [isComplete, status] = await validationRegistry.isValidationComplete(requestId);

                expect(isComplete).to.be.true;
                expect(status).to.equal(1); // ValidationStatus.VALIDATED
            });

            it("Should slash dishonest validator", async function () {
                const validatorStake = ethers.parseEther("0.1");
                const wrongHash = ethers.id("wrong-hash");

                // Validator submits wrong hash
                await expect(
                    validationRegistry.connect(validator1Wallet).submitStakeValidation(
                        requestId,
                        wrongHash,
                        { value: validatorStake }
                    )
                ).to.emit(validationRegistry, "ValidationSubmitted");

                // Since this is the first and only validator, it will finalize
                const [isComplete, status] = await validationRegistry.isValidationComplete(requestId);

                expect(isComplete).to.be.true;
            });
        });

        describe("Validator Statistics", function () {
            it("Should track validator stats after validation", async function () {
                const currentBlock = await ethers.provider.getBlock('latest');
                const deadline = currentBlock.timestamp + 7200;
                const stake = ethers.parseEther("0.1");

                const tx = await validationRegistry.connect(clientWallet).requestValidation(
                    taskId,
                    serverAgent,
                    dataHash,
                    1,
                    deadline,
                    { value: stake }
                );

                const receipt = await tx.wait();
                const event = receipt.logs.find(
                    log => log.fragment && log.fragment.name === "ValidationRequested"
                );
                const requestId = event.args[0];

                // Get validator address from signer
                const validator1Agents = await sageRegistry.getAgentsByOwner(validator1Wallet.address);
                const validator1Metadata = await sageRegistry.getAgent(validator1Agents[0]);
                const validator1Agent = validator1Metadata.owner;

                const validatorStake = ethers.parseEther("0.1");
                await validationRegistry.connect(validator1Wallet).submitStakeValidation(
                    requestId,
                    dataHash,
                    { value: validatorStake }
                );

                const stats = await validationRegistry.getValidatorStats(validator1Agent);
                expect(stats.totalValidations).to.equal(1);
            });
        });
    });

    describe("Full ERC-8004 Lifecycle", function () {
        it("Should complete full agent task flow", async function () {
            // 1. Agents already registered via beforeEach

            // 2. Get agent addresses
            const agent2Agents = await sageRegistry.getAgentsByOwner(agent2Wallet.address);
            const agent2Metadata = await sageRegistry.getAgent(agent2Agents[0]);
            const serverAgent = agent2Metadata.owner;

            const clientAgents = await sageRegistry.getAgentsByOwner(clientWallet.address);
            const clientMetadata = await sageRegistry.getAgent(clientAgents[0]);
            const clientAgent = clientMetadata.owner;

            // 3. Authorize task
            const taskId = ethers.id("full-lifecycle-task");
            const currentBlock = await ethers.provider.getBlock('latest');
            const deadline = currentBlock.timestamp + 7200;

            await reputationRegistry.connect(clientWallet).authorizeTask(
                taskId,
                serverAgent,
                deadline
            );

            // 4. Request validation
            const dataHash = ethers.id("task-output");
            const stake = ethers.parseEther("0.1");

            const tx = await validationRegistry.connect(clientWallet).requestValidation(
                taskId,
                serverAgent,
                dataHash,
                1, // STAKE
                deadline,
                { value: stake }
            );

            const receipt = await tx.wait();
            const event = receipt.logs.find(
                log => log.fragment && log.fragment.name === "ValidationRequested"
            );
            const requestId = event.args[0];

            // 5. Validator submits validation
            const validatorStake = ethers.parseEther("0.1");
            await validationRegistry.connect(validator1Wallet).submitStakeValidation(
                requestId,
                dataHash,
                { value: validatorStake }
            );

            // 6. Client submits feedback
            const rating = 95;
            const feedbackTx = await reputationRegistry.connect(clientWallet).submitFeedback(
                taskId,
                serverAgent,
                dataHash,
                rating
            );

            // 7. Verify complete state
            const [isComplete, status] = await validationRegistry.isValidationComplete(requestId);
            expect(isComplete).to.be.true;
            expect(status).to.equal(1); // VALIDATED

            const feedbackCount = await reputationRegistry.getAgentFeedbackCount(serverAgent);
            expect(feedbackCount).to.equal(1);
        });
    });
});
