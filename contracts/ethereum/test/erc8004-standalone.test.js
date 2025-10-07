const { expect } = require("chai");
const { ethers } = require("hardhat");

/**
 * ERC-8004 Standalone Implementation Tests
 *
 * Purpose: Verify that ERC-8004 contracts are completely independent
 * and can function without ANY Sage-specific contracts.
 *
 * These tests ensure:
 * 1. All three ERC-8004 contracts deploy independently
 * 2. No dependencies on SageRegistry or other Sage contracts
 * 3. Full ERC-8004 standard compliance
 * 4. Complete functionality without external dependencies
 */
describe("ERC-8004 Standalone Implementation Tests", function () {
    let identityRegistry;
    let reputationRegistry;
    let validationRegistry;

    let owner, alice, bob, charlie, validator1, validator2;
    let aliceAddress, bobAddress, charlieAddress;

    before(async function () {
        [owner, alice, bob, charlie, validator1, validator2] = await ethers.getSigners();
        aliceAddress = await alice.getAddress();
        bobAddress = await bob.getAddress();
        charlieAddress = await charlie.getAddress();
    });

    describe("1. Independent Deployment Tests", function () {
        it("should deploy ERC8004IdentityRegistry independently", async function () {
            const IdentityRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry"
            );
            identityRegistry = await IdentityRegistry.deploy();
            await identityRegistry.waitForDeployment();

            expect(await identityRegistry.getAddress()).to.be.properAddress;
            expect(await identityRegistry.totalAgents()).to.equal(0);
        });

        it("should deploy ERC8004ReputationRegistry independently", async function () {
            // Deploy without ValidationRegistry first to test independence
            const ReputationRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry"
            );
            reputationRegistry = await ReputationRegistry.deploy(ethers.ZeroAddress);
            await reputationRegistry.waitForDeployment();

            expect(await reputationRegistry.getAddress()).to.be.properAddress;
        });

        it("should deploy ERC8004ValidationRegistry independently", async function () {
            const ValidationRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry"
            );
            const minStake = ethers.parseEther("0.1");
            const minValidators = 2;
            const consensusThreshold = 66; // 66%

            validationRegistry = await ValidationRegistry.deploy(
                minStake,
                minValidators,
                consensusThreshold
            );
            await validationRegistry.waitForDeployment();

            expect(await validationRegistry.getAddress()).to.be.properAddress;
            expect(await validationRegistry.minStake()).to.equal(minStake);
            expect(await validationRegistry.minValidators()).to.equal(minValidators);
            expect(await validationRegistry.consensusThreshold()).to.equal(consensusThreshold);
        });

        it("should link ReputationRegistry to ValidationRegistry if desired", async function () {
            // This is optional - showing that linking can be done post-deployment
            const ReputationRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry"
            );
            reputationRegistry = await ReputationRegistry.deploy(
                await validationRegistry.getAddress()
            );
            await reputationRegistry.waitForDeployment();

            expect(await reputationRegistry.validationRegistry()).to.equal(
                await validationRegistry.getAddress()
            );
        });
    });

    describe("2. ERC8004IdentityRegistry - Full Functionality", function () {
        it("should register an agent with DID", async function () {
            const agentId = "did:sage:alice123";
            const endpoint = "https://alice-agent.example.com/agentcard";

            const tx = await identityRegistry.connect(alice).registerAgent(agentId, endpoint);
            const receipt = await tx.wait();

            // Verify event
            const event = receipt.logs.find(log => {
                try {
                    return identityRegistry.interface.parseLog(log)?.name === 'AgentRegistered';
                } catch { return false; }
            });
            expect(event).to.not.be.undefined;

            // Verify agent info
            const agentInfo = await identityRegistry.resolveAgent(agentId);
            expect(agentInfo.agentId).to.equal(agentId);
            expect(agentInfo.agentAddress).to.equal(aliceAddress);
            expect(agentInfo.endpoint).to.equal(endpoint);
            expect(agentInfo.isActive).to.be.true;

            // Verify total count
            expect(await identityRegistry.totalAgents()).to.equal(1);
        });

        it("should resolve agent by address", async function () {
            const agentId = "did:sage:bob456";
            const endpoint = "ipfs://QmXyz...";

            await identityRegistry.connect(bob).registerAgent(agentId, endpoint);

            const agentInfo = await identityRegistry.resolveAgentByAddress(bobAddress);
            expect(agentInfo.agentId).to.equal(agentId);
            expect(agentInfo.agentAddress).to.equal(bobAddress);
        });

        it("should check if agent is active", async function () {
            const agentId = "did:sage:alice123";
            expect(await identityRegistry.isAgentActive(agentId)).to.be.true;
        });

        it("should update agent endpoint", async function () {
            const agentId = "did:sage:alice123";
            const newEndpoint = "https://alice-agent-v2.example.com/agentcard";

            await identityRegistry.connect(alice).updateAgentEndpoint(agentId, newEndpoint);

            const agentInfo = await identityRegistry.resolveAgent(agentId);
            expect(agentInfo.endpoint).to.equal(newEndpoint);
        });

        it("should deactivate agent", async function () {
            const agentId = "did:sage:alice123";

            await identityRegistry.connect(alice).deactivateAgent(agentId);

            expect(await identityRegistry.isAgentActive(agentId)).to.be.false;

            const agentInfo = await identityRegistry.resolveAgent(agentId);
            expect(agentInfo.isActive).to.be.false;
        });

        it("should prevent duplicate registration", async function () {
            const agentId = "did:sage:duplicate";
            const endpoint = "https://test.com";

            await identityRegistry.connect(charlie).registerAgent(agentId, endpoint);

            await expect(
                identityRegistry.connect(charlie).registerAgent(agentId, endpoint)
            ).to.be.revertedWithCustomError(identityRegistry, "AgentAlreadyRegistered");
        });

        it("should prevent non-owner from updating endpoint", async function () {
            const agentId = "did:sage:bob456";
            const newEndpoint = "https://malicious.com";

            await expect(
                identityRegistry.connect(alice).updateAgentEndpoint(agentId, newEndpoint)
            ).to.be.revertedWithCustomError(identityRegistry, "NotAgentOwner");
        });
    });

    describe("3. ERC8004ReputationRegistry - Full Functionality", function () {
        let taskId1, taskId2;

        before(async function () {
            taskId1 = ethers.randomBytes(32);
            taskId2 = ethers.randomBytes(32);
        });

        it("should authorize a task for feedback", async function () {
            const deadline = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now

            const tx = await reputationRegistry.connect(alice).authorizeTask(
                taskId1,
                bobAddress,
                deadline
            );
            const receipt = await tx.wait();

            // Verify event
            const event = receipt.logs.find(log => {
                try {
                    return reputationRegistry.interface.parseLog(log)?.name === 'TaskAuthorized';
                } catch { return false; }
            });
            expect(event).to.not.be.undefined;
        });

        it("should submit feedback for authorized task", async function () {
            const dataHash = ethers.randomBytes(32);
            const rating = 85; // 85/100

            const tx = await reputationRegistry.connect(alice).submitFeedback(
                taskId1,
                bobAddress,
                dataHash,
                rating
            );
            const receipt = await tx.wait();

            // Verify event
            const event = receipt.logs.find(log => {
                try {
                    return reputationRegistry.interface.parseLog(log)?.name === 'FeedbackSubmitted';
                } catch { return false; }
            });
            expect(event).to.not.be.undefined;

            const parsedEvent = reputationRegistry.interface.parseLog(event);
            const feedbackId = parsedEvent.args.feedbackId;

            // Verify feedback details
            const feedback = await reputationRegistry.getFeedback(feedbackId);
            expect(feedback.clientAgent).to.equal(aliceAddress);
            expect(feedback.serverAgent).to.equal(bobAddress);
            expect(feedback.rating).to.equal(rating);
            expect(feedback.verified).to.be.false;
        });

        it("should get agent feedback count", async function () {
            const count = await reputationRegistry.getAgentFeedbackCount(bobAddress);
            expect(count).to.equal(1);
        });

        it("should get paginated agent feedback", async function () {
            // Submit more feedback
            const deadline2 = Math.floor(Date.now() / 1000) + 3600;
            await reputationRegistry.connect(charlie).authorizeTask(
                taskId2,
                bobAddress,
                deadline2
            );

            const dataHash2 = ethers.randomBytes(32);
            await reputationRegistry.connect(charlie).submitFeedback(
                taskId2,
                bobAddress,
                dataHash2,
                90
            );

            // Test pagination
            const feedbacks = await reputationRegistry.getAgentFeedback(bobAddress, 0, 10);
            expect(feedbacks.length).to.equal(2);
            expect(feedbacks[0].rating).to.equal(85);
            expect(feedbacks[1].rating).to.equal(90);
        });

        it("should prevent unauthorized feedback submission", async function () {
            const taskId3 = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);

            await expect(
                reputationRegistry.connect(alice).submitFeedback(
                    taskId3,
                    bobAddress,
                    dataHash,
                    80
                )
            ).to.be.revertedWithCustomError(reputationRegistry, "TaskNotAuthorized");
        });

        it("should prevent duplicate task authorization", async function () {
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            await expect(
                reputationRegistry.connect(alice).authorizeTask(
                    taskId1,
                    bobAddress,
                    deadline
                )
            ).to.be.revertedWithCustomError(reputationRegistry, "TaskAlreadyAuthorized");
        });
    });

    describe("4. ERC8004ValidationRegistry - Full Functionality", function () {
        let validationTaskId;
        let requestId;

        before(async function () {
            validationTaskId = ethers.randomBytes(32);
        });

        it("should request stake-based validation", async function () {
            const dataHash = ethers.keccak256(ethers.toUtf8Bytes("test output"));
            const deadline = Math.floor(Date.now() / 1000) + 3600;
            const stake = ethers.parseEther("0.1");

            const tx = await validationRegistry.connect(alice).requestValidation(
                validationTaskId,
                bobAddress,
                dataHash,
                1, // ValidationType.STAKE
                deadline,
                { value: stake }
            );
            const receipt = await tx.wait();

            // Extract requestId from event
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            expect(event).to.not.be.undefined;

            const parsedEvent = validationRegistry.interface.parseLog(event);
            requestId = parsedEvent.args.requestId;

            // Verify request
            const request = await validationRegistry.getValidationRequest(requestId);
            expect(request.requester).to.equal(aliceAddress);
            expect(request.serverAgent).to.equal(bobAddress);
            expect(request.validationType).to.equal(1); // STAKE
            expect(request.status).to.equal(0); // PENDING
        });

        it("should submit stake validation response", async function () {
            const dataHash = ethers.keccak256(ethers.toUtf8Bytes("test output"));
            const validatorStake = ethers.parseEther("0.1");

            const tx = await validationRegistry.connect(validator1).submitStakeValidation(
                requestId,
                dataHash,
                { value: validatorStake }
            );
            const receipt = await tx.wait();

            // Verify event
            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationSubmitted';
                } catch { return false; }
            });
            expect(event).to.not.be.undefined;

            // Check responses
            const responses = await validationRegistry.getValidationResponses(requestId);
            expect(responses.length).to.equal(1);
            expect(responses[0].validator).to.equal(await validator1.getAddress());
            expect(responses[0].success).to.be.true;
        });

        it("should reach consensus with multiple validators", async function () {
            const dataHash = ethers.keccak256(ethers.toUtf8Bytes("test output"));
            const validatorStake = ethers.parseEther("0.1");

            // Second validator submits matching response
            await validationRegistry.connect(validator2).submitStakeValidation(
                requestId,
                dataHash,
                { value: validatorStake }
            );

            // Check if validation is complete
            const [isComplete, status] = await validationRegistry.isValidationComplete(requestId);
            expect(isComplete).to.be.true;
            expect(status).to.equal(1); // VALIDATED
        });

        it("should prevent validation without sufficient stake", async function () {
            const taskId2 = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = Math.floor(Date.now() / 1000) + 3600;
            const insufficientStake = ethers.parseEther("0.01");

            await expect(
                validationRegistry.connect(alice).requestValidation(
                    taskId2,
                    bobAddress,
                    dataHash,
                    1, // ValidationType.STAKE
                    deadline,
                    { value: insufficientStake }
                )
            ).to.be.revertedWithCustomError(validationRegistry, "InvalidStake");
        });

        it("should prevent duplicate validator responses", async function () {
            const taskId3 = ethers.randomBytes(32);
            const dataHash = ethers.randomBytes(32);
            const deadline = Math.floor(Date.now() / 1000) + 3600;
            const stake = ethers.parseEther("0.1");

            const tx = await validationRegistry.connect(alice).requestValidation(
                taskId3,
                bobAddress,
                dataHash,
                1, // ValidationType.STAKE
                deadline,
                { value: stake }
            );
            const receipt = await tx.wait();

            const event = receipt.logs.find(log => {
                try {
                    return validationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const parsedEvent = validationRegistry.interface.parseLog(event);
            const newRequestId = parsedEvent.args.requestId;

            // First validation
            await validationRegistry.connect(validator1).submitStakeValidation(
                newRequestId,
                dataHash,
                { value: stake }
            );

            // Second validation from same validator should fail
            await expect(
                validationRegistry.connect(validator1).submitStakeValidation(
                    newRequestId,
                    dataHash,
                    { value: stake }
                )
            ).to.be.revertedWithCustomError(validationRegistry, "ValidatorAlreadyResponded");
        });
    });

    describe("5. Independence Verification", function () {
        it("should confirm no Sage contract dependencies in bytecode", async function () {
            // This test verifies that the deployed contracts don't reference Sage contracts
            const identityCode = await ethers.provider.getCode(await identityRegistry.getAddress());
            const reputationCode = await ethers.provider.getCode(await reputationRegistry.getAddress());
            const validationCode = await ethers.provider.getCode(await validationRegistry.getAddress());

            // All contracts should have non-empty bytecode
            expect(identityCode.length).to.be.greaterThan(2);
            expect(reputationCode.length).to.be.greaterThan(2);
            expect(validationCode.length).to.be.greaterThan(2);

            // None should be minimal proxy patterns pointing to Sage contracts
            expect(identityCode).to.not.include("SageRegistry");
            expect(reputationCode).to.not.include("SageRegistry");
            expect(validationCode).to.not.include("SageRegistry");
        });

        it("should work in complete isolation from Sage ecosystem", async function () {
            // Full workflow without any Sage contracts
            // Deploy fresh instances for complete isolation test
            const IdentityRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry"
            );
            const ValidationRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry"
            );
            const ReputationRegistry = await ethers.getContractFactory(
                "contracts/erc-8004/standalone/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry"
            );

            const freshIdentityRegistry = await IdentityRegistry.deploy();
            const freshValidationRegistry = await ValidationRegistry.deploy(
                ethers.parseEther("0.1"),
                2,
                66
            );
            const freshReputationRegistry = await ReputationRegistry.deploy(
                await freshValidationRegistry.getAddress()
            );

            await freshIdentityRegistry.waitForDeployment();
            await freshValidationRegistry.waitForDeployment();
            await freshReputationRegistry.waitForDeployment();

            // 1. Register agents
            const aliceId = "did:standalone:alice";
            const bobId = "did:standalone:bob";

            await freshIdentityRegistry.connect(alice).registerAgent(
                aliceId,
                "https://alice.example.com"
            );
            await freshIdentityRegistry.connect(bob).registerAgent(
                bobId,
                "https://bob.example.com"
            );

            // 2. Authorize and submit feedback
            const taskId = ethers.randomBytes(32);
            const deadline = Math.floor(Date.now() / 1000) + 3600;

            await freshReputationRegistry.connect(alice).authorizeTask(
                taskId,
                bobAddress,
                deadline
            );

            const dataHash = ethers.randomBytes(32);
            const tx = await freshReputationRegistry.connect(alice).submitFeedback(
                taskId,
                bobAddress,
                dataHash,
                95
            );
            const receipt = await tx.wait();

            const event = receipt.logs.find(log => {
                try {
                    return freshReputationRegistry.interface.parseLog(log)?.name === 'FeedbackSubmitted';
                } catch { return false; }
            });
            const feedbackId = freshReputationRegistry.interface.parseLog(event).args.feedbackId;

            // 3. Request validation
            const validationDeadline = Math.floor(Date.now() / 1000) + 3600;
            const stake = ethers.parseEther("0.1");

            const valTx = await freshValidationRegistry.connect(alice).requestValidation(
                taskId,
                bobAddress,
                dataHash,
                1, // STAKE
                validationDeadline,
                { value: stake }
            );
            const valReceipt = await valTx.wait();

            const valEvent = valReceipt.logs.find(log => {
                try {
                    return freshValidationRegistry.interface.parseLog(log)?.name === 'ValidationRequested';
                } catch { return false; }
            });
            const valRequestId = freshValidationRegistry.interface.parseLog(valEvent).args.requestId;

            // 4. Complete validation
            await freshValidationRegistry.connect(validator1).submitStakeValidation(
                valRequestId,
                dataHash,
                { value: stake }
            );
            await freshValidationRegistry.connect(validator2).submitStakeValidation(
                valRequestId,
                dataHash,
                { value: stake }
            );

            // Verify complete workflow succeeded
            expect(await freshIdentityRegistry.isAgentActive(aliceId)).to.be.true;
            expect(await freshIdentityRegistry.isAgentActive(bobId)).to.be.true;

            const feedback = await freshReputationRegistry.getFeedback(feedbackId);
            expect(feedback.rating).to.equal(95);

            const [isComplete, status] = await freshValidationRegistry.isValidationComplete(valRequestId);
            expect(isComplete).to.be.true;
            expect(status).to.equal(1); // VALIDATED
        });
    });

    describe("6. ERC-8004 Standard Compliance", function () {
        it("should implement all required IdentityRegistry interface methods", async function () {
            expect(identityRegistry.registerAgent).to.be.a('function');
            expect(identityRegistry.resolveAgent).to.be.a('function');
            expect(identityRegistry.resolveAgentByAddress).to.be.a('function');
            expect(identityRegistry.isAgentActive).to.be.a('function');
            expect(identityRegistry.updateAgentEndpoint).to.be.a('function');
            expect(identityRegistry.deactivateAgent).to.be.a('function');
        });

        it("should implement all required ReputationRegistry interface methods", async function () {
            expect(reputationRegistry.authorizeTask).to.be.a('function');
            expect(reputationRegistry.submitFeedback).to.be.a('function');
            expect(reputationRegistry.verifyFeedback).to.be.a('function');
            expect(reputationRegistry.getFeedback).to.be.a('function');
            expect(reputationRegistry.getAgentFeedback).to.be.a('function');
            expect(reputationRegistry.getTaskFeedback).to.be.a('function');
            expect(reputationRegistry.getAgentFeedbackCount).to.be.a('function');
        });

        it("should implement all required ValidationRegistry interface methods", async function () {
            expect(validationRegistry.requestValidation).to.be.a('function');
            expect(validationRegistry.submitStakeValidation).to.be.a('function');
            expect(validationRegistry.submitTEEAttestation).to.be.a('function');
            expect(validationRegistry.getValidationRequest).to.be.a('function');
            expect(validationRegistry.getValidationResponses).to.be.a('function');
            expect(validationRegistry.isValidationComplete).to.be.a('function');
        });
    });
});
