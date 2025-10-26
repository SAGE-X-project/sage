import { expect } from "chai";
import { parseEther, keccak256, AbiCoder, Wallet, ZeroAddress } from "ethers";
import { network } from "hardhat";

// Initialize ethers from network connection
const { ethers } = await network.connect();

/**
 * Full Workflow Integration Test Suite
 *
 * Test Coverage:
 * - I5.1.1: Complete registration workflow
 * - I5.1.2: Multi-key lifecycle end-to-end
 * - I5.1.3: ERC-8004 ecosystem integration
 *
 * Total: 3 main workflows with comprehensive sub-tests
 */
describe("Full Workflow Integration", () => {
    let agentRegistry;
    let verifyHook;
    let owner;
    let user1;
    let user2;

    const STAKE = parseEther("0.01");
    const ACTIVATION_DELAY = 3600; // 1 hour
    let chainId;
    let registryAddress;

    // Test DIDs
    const validDID1 = "did:sage:ethereum:0x1111111111111111111111111111111111111111";
    const validDID2 = "did:sage:ethereum:0x2222222222222222222222222222222222222222";

    // Helper functions
    function createRegistrationParams(did, name, keys, keyTypes, signatures, salt) {
        return {
            did: did,
            name: name,
            description: "Integration Test Agent",
            endpoint: "https://agent.example.com",
            capabilities: '{"capabilities": ["query", "execute"]}',
            keys: keys,
            keyTypes: keyTypes,
            signatures: signatures,
            salt: salt
        };
    }

    async function commitAndAdvanceTime(user, did, keys, salt) {
        const commitHash = keccak256(
            AbiCoder.defaultAbiCoder().encode(
                ["string", "bytes[]", "address", "bytes32", "uint256"],
                [did, keys, user.address, salt, chainId]
            )
        );
        const tx = await agentRegistry.connect(user).commitRegistration(commitHash, { value: STAKE });
        await tx.wait();

        await ethers.provider.send("evm_increaseTime", [61]);
        await ethers.provider.send("evm_mine");

        return commitHash;
    }

    async function createSignature(signer) {
        const message = ethers.solidityPackedKeccak256(
            ["string", "uint256", "address", "address"],
            ["SAGE Agent Registration:", chainId, registryAddress, signer.address]
        );
        return await signer.signMessage(ethers.getBytes(message));
    }

    beforeEach(async () => {
        [owner, user1, user2] = await ethers.getSigners();

        // Get chain ID
        chainId = (await ethers.provider.getNetwork()).chainId;

        // Deploy contracts
        const VerifyHook = await ethers.getContractFactory("AgentCardVerifyHook");
        verifyHook = await VerifyHook.deploy();
        await verifyHook.waitForDeployment();

        const Registry = await ethers.getContractFactory("AgentCardRegistry");
        agentRegistry = await Registry.deploy(await verifyHook.getAddress());
        await agentRegistry.waitForDeployment();
        registryAddress = await agentRegistry.getAddress();
    });

    describe("I5.1.1: Complete Registration Workflow", () => {
        it("Should complete end-to-end agent registration with 1 key", async () => {
            // Step 1: Generate key and signature
            const key1 = ethers.randomBytes(33);
            const sig1 = await createSignature(user1);
            const salt = ethers.randomBytes(32);

            // Step 2: Commit registration
            const commitHash = await commitAndAdvanceTime(user1, validDID1, [key1], salt);

            // Verify commitment stored
            const commitment = await agentRegistry.registrationCommitments(user1.address);
            expect(commitment.commitHash).to.equal(commitHash);
            expect(commitment.revealed).to.be.false;

            // Step 3: Register agent
            const params = createRegistrationParams(
                validDID1,
                "Test Agent 1",
                [key1],
                [0], // ECDSA
                [sig1],
                salt
            );

            const tx = await agentRegistry.connect(user1).registerAgentWithParams(params);
            const receipt = await tx.wait();

            // Verify event emitted
            const event = receipt.logs.find(
                log => log.fragment && log.fragment.name === "AgentRegistered"
            );
            expect(event).to.exist;

            // Step 4: Verify agent stored but not active
            const agentId = await agentRegistry.didToAgentId(validDID1);
            expect(agentId).to.not.equal(ethers.ZeroHash);

            const agent = await agentRegistry.getAgent(agentId);
            expect(agent.owner).to.equal(user1.address);
            expect(agent.did).to.equal(validDID1);
            expect(agent.active).to.be.false;
            expect(agent.keyHashes.length).to.equal(1);

            // Verify commitment revealed
            const revealedCommitment = await agentRegistry.registrationCommitments(user1.address);
            expect(revealedCommitment.revealed).to.be.true;

            // Step 5: Wait for activation delay
            await ethers.provider.send("evm_increaseTime", [ACTIVATION_DELAY + 1]);
            await ethers.provider.send("evm_mine");

            // Step 6: Activate agent
            const activateTx = await agentRegistry.activateAgent(agentId);
            const activateReceipt = await activateTx.wait();

            // Verify activation event
            const activateEvent = activateReceipt.logs.find(
                log => log.fragment && log.fragment.name === "AgentActivated"
            );
            expect(activateEvent).to.exist;

            // Step 7: Verify agent is now active
            const activeAgent = await agentRegistry.getAgent(agentId);
            expect(activeAgent.active).to.be.true;
        });

        it("Should complete end-to-end agent registration with multiple keys", async () => {
            // Generate 3 different key types
            const key1 = ethers.randomBytes(33); // ECDSA
            const key2 = ethers.randomBytes(32); // Ed25519
            const key3 = ethers.randomBytes(32); // X25519

            const sig1 = await createSignature(user1);
            const sig2 = ethers.randomBytes(64); // Ed25519 sig (placeholder)
            const sig3 = ethers.randomBytes(0);  // X25519 no sig needed

            const salt = ethers.randomBytes(32);

            // Commit
            await commitAndAdvanceTime(user1, validDID1, [key1, key2, key3], salt);

            // Register with 3 keys
            const params = createRegistrationParams(
                validDID1,
                "Multi-Key Agent",
                [key1, key2, key3],
                [0, 1, 2], // ECDSA, Ed25519, X25519
                [sig1, sig2, sig3],
                salt
            );

            await agentRegistry.connect(user1).registerAgentWithParams(params);

            // Verify all 3 keys stored
            const agentId = await agentRegistry.didToAgentId(validDID1);
            const agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(3);

            // Verify each key type
            for (let i = 0; i < 3; i++) {
                const keyInfo = await agentRegistry.getKey(agent.keyHashes[i]);
                expect(keyInfo.keyType).to.equal(i); // 0=ECDSA, 1=Ed25519, 2=X25519
                expect(keyInfo.verified).to.be.true;
            }
        });
    });

    describe("I5.1.2: Multi-Key Lifecycle End-to-End", () => {
        let agentId;
        let initialKeyHash;

        beforeEach(async () => {
            // Register agent with 1 key
            const key1 = ethers.randomBytes(33);
            const sig1 = await createSignature(user1);
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [key1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Lifecycle Agent",
                [key1],
                [0],
                [sig1],
                salt
            );

            await agentRegistry.connect(user1).registerAgentWithParams(params);
            agentId = await agentRegistry.didToAgentId(validDID1);

            const agent = await agentRegistry.getAgent(agentId);
            initialKeyHash = agent.keyHashes[0];

            // Activate
            await ethers.provider.send("evm_increaseTime", [ACTIVATION_DELAY + 1]);
            await ethers.provider.send("evm_mine");
            await agentRegistry.activateAgent(agentId);
        });

        it("Should complete full key lifecycle: add → rotate → revoke", async () => {
            // Verify initial state: 1 key
            let agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(1);

            // Step 1: Add new key
            const newKey = ethers.randomBytes(33);
            const newSig = await createSignature(user1);

            const addTx = await agentRegistry.connect(user1).addKey(
                agentId,
                newKey,
                0, // ECDSA
                newSig
            );
            await addTx.wait();

            // Verify: 2 keys now
            agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(2);
            const newKeyHash = agent.keyHashes[1];

            // Step 2: Rotate - remove old key
            const revokeTx = await agentRegistry.connect(user1).revokeKey(
                agentId,
                initialKeyHash
            );
            await revokeTx.wait();

            // Verify: back to 1 key (new one)
            agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(1);
            expect(agent.keyHashes[0]).to.equal(newKeyHash);

            // Verify old key revoked
            const oldKey = await agentRegistry.getKey(initialKeyHash);
            expect(oldKey.verified).to.be.false;

            // Step 3: Add another key for final test
            const thirdKey = ethers.randomBytes(33);
            const thirdSig = await createSignature(user1);

            await agentRegistry.connect(user1).addKey(
                agentId,
                thirdKey,
                0,
                thirdSig
            );

            // Verify: 2 keys again
            agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(2);
        });

        it("Should manage mixed key types throughout lifecycle", async () => {
            // Add Ed25519 key
            const ed25519Key = ethers.randomBytes(32);
            const ed25519Sig = ethers.randomBytes(64);

            await agentRegistry.connect(user1).addKey(
                agentId,
                ed25519Key,
                1, // Ed25519
                ed25519Sig
            );

            // Add X25519 key
            const x25519Key = ethers.randomBytes(32);
            const x25519Sig = ethers.randomBytes(0);

            await agentRegistry.connect(user1).addKey(
                agentId,
                x25519Key,
                2, // X25519
                x25519Sig
            );

            // Verify: 3 keys of different types
            const agent = await agentRegistry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(3);

            // Verify key types
            const key1 = await agentRegistry.getKey(agent.keyHashes[0]);
            const key2 = await agentRegistry.getKey(agent.keyHashes[1]);
            const key3 = await agentRegistry.getKey(agent.keyHashes[2]);

            expect(key1.keyType).to.equal(0); // ECDSA
            expect(key2.keyType).to.equal(1); // Ed25519
            expect(key3.keyType).to.equal(2); // X25519
        });
    });

    describe("I5.1.3: ERC-8004 Ecosystem Integration", () => {
        let agentId;

        beforeEach(async () => {
            // Register agent via AgentCardRegistry
            const key1 = ethers.randomBytes(33);
            const sig1 = await createSignature(user1);
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [key1], salt);

            const params = createRegistrationParams(
                validDID1,
                "ERC8004 Integration Agent",
                [key1],
                [0],
                [sig1],
                salt
            );

            await agentRegistry.connect(user1).registerAgentWithParams(params);
            agentId = await agentRegistry.didToAgentId(validDID1);

            // Activate
            await ethers.provider.send("evm_increaseTime", [ACTIVATION_DELAY + 1]);
            await ethers.provider.send("evm_mine");
            await agentRegistry.activateAgent(agentId);
        });

        it("Should register via AgentCardRegistry and resolve via ERC8004", async () => {
            // Query via ERC8004
            const agentInfo = await agentRegistry.resolveAgent(validDID1);

            expect(agentInfo.agentId).to.equal(validDID1);
            expect(agentInfo.agentAddress).to.equal(user1.address);
            expect(agentInfo.endpoint).to.equal("https://agent.example.com");
            expect(agentInfo.isActive).to.be.true;
            expect(agentInfo.registeredAt).to.be.gt(0);
        });

        it("Should resolve by address via ERC8004", async () => {
            const agentInfo = await agentRegistry.resolveAgentByAddress(user1.address);

            expect(agentInfo.agentId).to.equal(validDID1);
            expect(agentInfo.agentAddress).to.equal(user1.address);
        });

        it("Should check active status via ERC8004", async () => {
            const isActive = await agentRegistry.isAgentActive(validDID1);
            expect(isActive).to.be.true;
        });

        it("Should update endpoint via ERC8004 and verify in AgentCardRegistry", async () => {
            const newEndpoint = "https://new-agent-endpoint.com";

            // Update via ERC8004
            const tx = await agentRegistry.connect(user1).updateAgentEndpoint(
                validDID1,
                newEndpoint
            );
            await tx.wait();

            // Verify in AgentCardRegistry
            const agent = await agentRegistry.getAgent(agentId);
            expect(agent.endpoint).to.equal(newEndpoint);

            // Verify via ERC8004 query
            const agentInfo = await agentRegistry.resolveAgent(validDID1);
            expect(agentInfo.endpoint).to.equal(newEndpoint);
        });

        it("Should deactivate via ERC8004 and verify in AgentCardRegistry", async () => {
            // Deactivate via ERC8004
            const tx = await agentRegistry.connect(user1).deactivateAgent(validDID1);
            await tx.wait();

            // Verify in AgentCardRegistry
            const agent = await agentRegistry.getAgent(agentId);
            expect(agent.active).to.be.false;

            // Verify via ERC8004 query
            const isActive = await agentRegistry.isAgentActive(validDID1);
            expect(isActive).to.be.false;
        });

        it("Should maintain consistency across both interfaces", async () => {
            // Get data from both interfaces
            const agentFromRegistry = await agentRegistry.getAgent(agentId);
            const agentFromERC8004 = await agentRegistry.resolveAgent(validDID1);

            // Verify consistency
            expect(agentFromERC8004.agentId).to.equal(agentFromRegistry.did);
            expect(agentFromERC8004.agentAddress).to.equal(agentFromRegistry.owner);
            expect(agentFromERC8004.endpoint).to.equal(agentFromRegistry.endpoint);
            expect(agentFromERC8004.isActive).to.equal(agentFromRegistry.active);
            expect(agentFromERC8004.registeredAt).to.equal(agentFromRegistry.registeredAt);

            // Update via one interface
            const newEndpoint = "https://updated-via-test.com";
            await agentRegistry.connect(user1).updateAgentEndpoint(validDID1, newEndpoint);

            // Verify via both interfaces
            const updatedRegistry = await agentRegistry.getAgent(agentId);
            const updatedERC8004 = await agentRegistry.resolveAgent(validDID1);

            expect(updatedRegistry.endpoint).to.equal(newEndpoint);
            expect(updatedERC8004.endpoint).to.equal(newEndpoint);
        });
    });
});
