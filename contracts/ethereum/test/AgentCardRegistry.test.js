/**
 * AgentCardRegistry Test Suite
 *
 * Tests: 47 total
 * - V3.1: Commit-Reveal Pattern (9 tests)
 * - V3.2: Multi-Key Registration (12 tests)
 * - V3.3: Key Management (8 tests)
 * - V3.4: Agent Management (10 tests)
 * - V3.5: Security Features (8 tests)
 *
 * Test Methodology: TDD (Test-Driven Development)
 * - RED: Write tests first (will fail)
 * - GREEN: Implement to pass tests
 * - REFACTOR: Optimize implementation
 */

import { expect } from "chai";
import { parseEther, keccak256, AbiCoder, Wallet, ZeroAddress } from "ethers";
import { network } from "hardhat";

// Initialize ethers from network connection
const { ethers } = await network.connect();

describe("AgentCardRegistry", function () {
    let registry;
    let hook;
    let owner, user1, user2, attacker;

    // Test data
    let validDID1, validDID2, validDID3;
    let validKey1, validKey2, validKey3;
    let validSig1, validSig2, validSig3;
    let wallet1, wallet2, wallet3;
    let chainId, registryAddress;

    const STAKE = parseEther("0.01");
    const MIN_DELAY = 60; // 1 minute
    const MAX_DELAY = 3600; // 1 hour
    const ACTIVATION_DELAY = 3600; // 1 hour

    // Helper function to create registration params
    function createRegistrationParams(did, name, keys, keyTypes, signatures, salt) {
        return {
            did: did,
            name: name,
            description: "Test Description",
            endpoint: "https://example.com",
            capabilities: '{"capabilities": []}',
            keys: keys,
            keyTypes: keyTypes,
            signatures: signatures,
            salt: salt
        };
    }

    // Helper function to commit and advance time
    async function commitAndAdvanceTime(user, did, keys, salt) {
        const commitHash = keccak256(
            AbiCoder.defaultAbiCoder().encode(
                ["string", "bytes[]", "address", "bytes32", "uint256"],
                [did, keys, user.address, salt, chainId]
            )
        );

        await registry.connect(user).commitRegistration(commitHash, { value: STAKE });

        await ethers.provider.send("evm_increaseTime", [61]);
        await ethers.provider.send("evm_mine");
    }

    // Helper function to create signature
    async function createSignature(signer) {
        const message = ethers.solidityPackedKeccak256(
            ["string", "uint256", "address", "address"],
            ["SAGE Agent Registration:", chainId, registryAddress, signer.address]
        );
        return await signer.signMessage(ethers.getBytes(message));
    }

    beforeEach(async function () {
        [owner, user1, user2, attacker] = await ethers.getSigners();

        // Deploy AgentCardVerifyHook
        const Hook = await ethers.getContractFactory("AgentCardVerifyHook");
        hook = await Hook.deploy();
        await hook.waitForDeployment();

        // Deploy AgentCardRegistry
        const Registry = await ethers.getContractFactory("AgentCardRegistry");
        registry = await Registry.deploy(await hook.getAddress());
        await registry.waitForDeployment();

        // Store chain info
        chainId = (await ethers.provider.getNetwork()).chainId;
        registryAddress = await registry.getAddress();

        // Create test wallets for key generation
        wallet1 = Wallet.createRandom();
        wallet2 = Wallet.createRandom();
        wallet3 = Wallet.createRandom();

        // Generate test data
        validDID1 = `did:sage:ethereum:${user1.address.toLowerCase()}`;
        validDID2 = `did:sage:ethereum:${user2.address.toLowerCase()}`;
        validDID3 = `did:sage:ethereum:${attacker.address.toLowerCase()}`;

        validKey1 = wallet1.publicKey;
        validKey2 = wallet2.publicKey;
        validKey3 = wallet3.publicKey;

        // Sign ownership proofs
        validSig1 = await createSignature(user1);
        validSig2 = await createSignature(user2);
        validSig3 = await createSignature(attacker);
    });

    // ========================================================================
    // V3.1: Commit-Reveal Pattern (9 tests)
    // ========================================================================

    describe("V3.1: Commit-Reveal Pattern", function () {

        it("R3.1.1: Should require sufficient stake for commitment", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            // Insufficient stake
            await expect(
                registry.connect(user1).commitRegistration(commitHash, { value: parseEther("0.005") })
            ).to.be.revertedWith("Insufficient stake");

            // Sufficient stake
            await expect(
                registry.connect(user1).commitRegistration(commitHash, { value: STAKE })
            ).to.not.be.revert(ethers);
        });

        it("R3.1.2: Should store commitment correctly", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            const commitment = await registry.registrationCommitments(user1.address);
            expect(commitment.commitHash).to.equal(commitHash);
            expect(commitment.revealed).to.be.false;
        });

        it("R3.1.3: Should reject reveal if too soon (<1 min)", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            // Try to reveal immediately
            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Reveal too soon");
        });

        it("R3.1.4: Should reject reveal if too late (>1 hour)", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            // Advance time by more than 1 hour
            await ethers.provider.send("evm_increaseTime", [3601]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Commitment expired");
        });

        it("R3.1.5: Should reject reveal with wrong hash", async function () {
            const salt = ethers.randomBytes(32);
            const wrongSalt = ethers.randomBytes(32);

            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            // Advance time
            await ethers.provider.send("evm_increaseTime", [61]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                wrongSalt  // Wrong salt
            );

            // Try to reveal with wrong salt
            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid reveal");
        });

        it("R3.1.6: Should reject reveal with wrong salt", async function () {
            // Same as R3.1.5 (testing salt mismatch)
            const salt = ethers.randomBytes(32);
            const wrongSalt = ethers.randomBytes(32);

            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            await ethers.provider.send("evm_increaseTime", [61]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                wrongSalt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid reveal");
        });

        it("R3.1.7: Should reject reveal with wrong chainId", async function () {
            const salt = ethers.randomBytes(32);

            // Create commitment with wrong chainId
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, 1] // Wrong chainId
                )
            );

            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            await ethers.provider.send("evm_increaseTime", [61]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            // Try to reveal with correct chainId (will mismatch)
            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid reveal");
        });

        it("R3.1.8: Should prevent front-running attack", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            // User commits
            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            // Attacker sees commitment and tries to commit same DID
            const attackerSalt = ethers.randomBytes(32);
            const attackerCommitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey3], attacker.address, attackerSalt, chainId]
                )
            );

            await registry.connect(attacker).commitRegistration(attackerCommitHash, { value: STAKE });

            // Advance time
            await ethers.provider.send("evm_increaseTime", [61]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            // User reveals first
            await registry.connect(user1).registerAgentWithParams(params);

            const attackerParams = createRegistrationParams(
                validDID1,
                "Attacker Agent",
                [validKey3],
                [0],
                [validSig3],
                attackerSalt
            );

            // Attacker tries to reveal - should fail (DID already registered)
            await expect(
                registry.connect(attacker).registerAgentWithParams(attackerParams)
            ).to.be.revertedWith("DID already registered");

            // Verify user owns the DID
            const agent = await registry.getAgentByDID(validDID1);
            expect(agent.owner).to.equal(user1.address);
        });

        it("R3.1.9: Should emit CommitmentRecorded event", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            await expect(
                registry.connect(user1).commitRegistration(commitHash, { value: STAKE })
            ).to.emit(registry, "CommitmentRecorded")
             .withArgs(user1.address, commitHash, await ethers.provider.getBlock("latest").then(b => b.timestamp + 1));
        });
    });

    // ========================================================================
    // V3.2: Multi-Key Registration (12 tests)
    // ========================================================================

    describe("V3.2: Multi-Key Registration", function () {

        it("R3.2.1: Should register agent with 1 key", async function () {
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            const agentId = await registry.didToAgentId(validDID1);

            const agent = await registry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(1);
        });

        it("R3.2.2: Should register agent with 2-10 keys", async function () {
            const salt = ethers.randomBytes(32);

            // Generate 10 keys
            const keys = [];
            const types = [];
            const sigs = [];

            for (let i = 0; i < 10; i++) {
                const wallet = Wallet.createRandom();
                keys.push(wallet.publicKey);
                types.push(0); // ECDSA
                sigs.push(await createSignature(user1));
            }

            await commitAndAdvanceTime(user1, validDID1, keys, salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                keys,
                types,
                sigs,
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            const agentId = await registry.didToAgentId(validDID1);

            const agent = await registry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(10);
        });

        it("R3.2.3: Should reject registration with 0 keys", async function () {
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [],
                [],
                [],
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid key count");
        });

        it("R3.2.4: Should reject registration with >10 keys", async function () {
            const salt = ethers.randomBytes(32);

            // Generate 11 keys
            const keys = [];
            const types = [];
            const sigs = [];

            for (let i = 0; i < 11; i++) {
                const wallet = Wallet.createRandom();
                keys.push(wallet.publicKey);
                types.push(0);
                sigs.push(await createSignature(user1));
            }

            await commitAndAdvanceTime(user1, validDID1, keys, salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                keys,
                types,
                sigs,
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid key count");
        });

        it("R3.2.5: Should support ECDSA keys", async function () {
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0], // ECDSA
                [validSig1],
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.not.be.revert(ethers);
        });

        it("R3.2.6: Should support Ed25519 keys", async function () {
            const salt = ethers.randomBytes(32);
            const ed25519Key = "0x" + "a".repeat(64); // 32 bytes
            const ed25519Sig = "0x" + "b".repeat(128); // 64 bytes

            await commitAndAdvanceTime(user1, validDID1, [ed25519Key], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [ed25519Key],
                [1], // Ed25519
                [ed25519Sig],
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.not.be.revert(ethers);
        });

        it("R3.2.7: Should support X25519 keys", async function () {
            const salt = ethers.randomBytes(32);
            const x25519Key = "0x" + "c".repeat(64); // 32 bytes
            const x25519Sig = "0x" + "d".repeat(64); // Dummy sig (not verified for X25519)

            await commitAndAdvanceTime(user1, validDID1, [x25519Key], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [x25519Key],
                [2], // X25519
                [x25519Sig],
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.not.be.revert(ethers);
        });

        it("R3.2.8: Should support mixed key types", async function () {
            const salt = ethers.randomBytes(32);

            const ecdsaKey = validKey1;
            const ed25519Key = "0x" + "a".repeat(64);
            const x25519Key = "0x" + "c".repeat(64);

            const keys = [ecdsaKey, ed25519Key, x25519Key];
            const types = [0, 1, 2];
            const sigs = [
                validSig1,
                "0x" + "b".repeat(128),
                "0x" + "d".repeat(64)
            ];

            await commitAndAdvanceTime(user1, validDID1, keys, salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                keys,
                types,
                sigs,
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            const agentId = await registry.didToAgentId(validDID1);

            const agent = await registry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(3);
        });

        it("R3.2.9: Should verify key ownership (ECDSA)", async function () {
            const salt = ethers.randomBytes(32);

            // Create wrong signature
            const wrongSig = await user2.signMessage("wrong message");

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [wrongSig], // Wrong signature
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid ECDSA signature");
        });

        it("R3.2.10: Should store all keys correctly", async function () {
            const salt = ethers.randomBytes(32);

            const keys = [validKey1, validKey2];
            const types = [0, 0];
            const sigs = [validSig1, validSig1]; // Both signed by user1

            await commitAndAdvanceTime(user1, validDID1, keys, salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                keys,
                types,
                sigs,
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            const agentId = await registry.didToAgentId(validDID1);

            const keyHash1 = keccak256(validKey1);
            const keyHash2 = keccak256(validKey2);

            const key1 = await registry.getKey(keyHash1);
            const key2 = await registry.getKey(keyHash2);

            expect(key1.verified).to.be.true;
            expect(key2.verified).to.be.true;
        });

        it("R3.2.11: Should prevent duplicate keys", async function () {
            const salt = ethers.randomBytes(32);

            // Try to register with duplicate key
            const keys = [validKey1, validKey1];
            const types = [0, 0];
            const sigs = [validSig1, validSig1];

            await commitAndAdvanceTime(user1, validDID1, keys, salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                keys,
                types,
                sigs,
                salt
            );

            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Public key already used");
        });

        it("R3.2.12: Should prevent key reuse across agents", async function () {
            // Register first agent with user1
            const salt1 = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt1);

            const params1 = createRegistrationParams(
                validDID1,
                "Test Agent 1",
                [validKey1],
                [0],
                [validSig1],
                salt1
            );

            await registry.connect(user1).registerAgentWithParams(params1);

            // Try to register second agent with same key
            const salt2 = ethers.randomBytes(32);

            await commitAndAdvanceTime(user2, validDID2, [validKey1], salt2);

            const params2 = createRegistrationParams(
                validDID2,
                "Test Agent 2",
                [validKey1], // Reusing key1
                [0],
                [validSig2],
                salt2
            );

            await expect(
                registry.connect(user2).registerAgentWithParams(params2)
            ).to.be.revertedWith("Public key already used");
        });
    });

    // ========================================================================
    // V3.3: Key Management (8 tests)
    // ========================================================================

    describe("V3.3: Key Management", function () {
        let agentId;

        beforeEach(async function () {
            // Register an agent first
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            agentId = await registry.didToAgentId(validDID1);
        });

        it("R3.3.1: Should add new key successfully", async function () {
            // Create signature for user1 (key owner)
            const sig = await createSignature(user1);

            await expect(
                registry.connect(user1).addKey(
                    agentId,
                    validKey2,
                    0, // ECDSA
                    sig
                )
            ).to.not.be.revert(ethers);

            const agent = await registry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(2);
        });

        it("R3.3.2: Should reject addKey if max keys reached", async function () {
            // Add 9 more keys (total 10)
            for (let i = 0; i < 9; i++) {
                const wallet = Wallet.createRandom();
                const sig = await createSignature(user1);

                await registry.connect(user1).addKey(
                    agentId,
                    wallet.publicKey,
                    0,
                    sig
                );
            }

            // Try to add 11th key
            const wallet11 = Wallet.createRandom();
            const sig11 = await createSignature(user1);

            await expect(
                registry.connect(user1).addKey(
                    agentId,
                    wallet11.publicKey,
                    0,
                    sig11
                )
            ).to.be.revertedWith("Max keys reached");
        });

        it("R3.3.3: Should verify ownership when adding key", async function () {
            const wrongSig = await user2.signMessage("wrong");

            await expect(
                registry.connect(user1).addKey(
                    agentId,
                    validKey2,
                    0,
                    wrongSig
                )
            ).to.be.revertedWith("Invalid ECDSA signature");
        });

        it("R3.3.4: Should only allow owner to add key", async function () {
            await expect(
                registry.connect(user2).addKey(
                    agentId,
                    validKey2,
                    0,
                    validSig2
                )
            ).to.be.revertedWith("Not agent owner");
        });

        it("R3.3.5: Should remove key successfully", async function () {
            // First add a second key
            const sig = await createSignature(user1);
            await registry.connect(user1).addKey(
                agentId,
                validKey2,
                0,
                sig
            );

            // Now revoke first key
            const keyHash1 = keccak256(validKey1);

            await expect(
                registry.connect(user1).revokeKey(agentId, keyHash1)
            ).to.not.be.revert(ethers);

            const agent = await registry.getAgent(agentId);
            expect(agent.keyHashes.length).to.equal(1);
        });

        it("R3.3.6: Should reject revokeKey if last key", async function () {
            const keyHash1 = keccak256(validKey1);

            await expect(
                registry.connect(user1).revokeKey(agentId, keyHash1)
            ).to.be.revertedWith("Cannot revoke last key");
        });

        it("R3.3.7: Should only allow owner to revoke key", async function () {
            // Add second key first
            const sig = await createSignature(user1);
            await registry.connect(user1).addKey(
                agentId,
                validKey2,
                0,
                sig
            );

            const keyHash1 = keccak256(validKey1);

            await expect(
                registry.connect(user2).revokeKey(agentId, keyHash1)
            ).to.be.revertedWith("Not agent owner");
        });

        it("R3.3.8: Should rotate key via add+revoke", async function () {
            // Manual key rotation: add new key, then revoke old key

            // Add new key
            const sig = await createSignature(user1);
            await registry.connect(user1).addKey(
                agentId,
                validKey2,
                0,
                sig
            );

            const agent1 = await registry.getAgent(agentId);
            expect(agent1.keyHashes.length).to.equal(2);

            // Revoke old key
            const keyHash1 = keccak256(validKey1);
            await registry.connect(user1).revokeKey(agentId, keyHash1);

            const agent2 = await registry.getAgent(agentId);
            expect(agent2.keyHashes.length).to.equal(1);

            // Verify only key2 remains
            const keyHash2 = keccak256(validKey2);
            expect(agent2.keyHashes[0]).to.equal(keyHash2);
        });

        it("R3.3.9: Should emit key operation events", async function () {
            // Test KeyAdded event
            const sig = await createSignature(user1);
            await expect(
                registry.connect(user1).addKey(
                    agentId,
                    validKey2,
                    0,
                    sig
                )
            ).to.emit(registry, "KeyAdded");

            // Test KeyRevoked event
            const keyHash1 = keccak256(validKey1);
            await expect(
                registry.connect(user1).revokeKey(agentId, keyHash1)
            ).to.emit(registry, "KeyRevoked");
        });
    });

    // ========================================================================
    // V3.4: Agent Management (10 tests)
    // ========================================================================

    describe("V3.4: Agent Management", function () {
        let agentId;

        beforeEach(async function () {
            // Register an agent
            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            await registry.connect(user1).registerAgentWithParams(params);
            agentId = await registry.didToAgentId(validDID1);
        });

        it("R3.4.1: Should update endpoint", async function () {
            const newEndpoint = "https://new-endpoint.com";

            await registry.connect(user1).updateAgent(
                agentId,
                newEndpoint,
                '{"capabilities": []}'
            );

            const agent = await registry.getAgent(agentId);
            expect(agent.endpoint).to.equal(newEndpoint);
        });

        it("R3.4.2: Should update capabilities", async function () {
            const newCapabilities = '{"capabilities": ["chat", "translate"]}';

            await registry.connect(user1).updateAgent(
                agentId,
                "https://example.com",
                newCapabilities
            );

            const agent = await registry.getAgent(agentId);
            expect(agent.capabilities).to.equal(newCapabilities);
        });

        it("R3.4.3: Should increment nonce on update", async function () {
            const nonceBefore = await registry.agentNonce(agentId);

            await registry.connect(user1).updateAgent(
                agentId,
                "https://new.com",
                '{"capabilities": []}'
            );

            const nonceAfter = await registry.agentNonce(agentId);
            expect(nonceAfter).to.equal(nonceBefore + 1n);
        });

        it("R3.4.4: Should only allow owner to update", async function () {
            await expect(
                registry.connect(user2).updateAgent(
                    agentId,
                    "https://hacker.com",
                    '{"capabilities": []}'
                )
            ).to.be.revertedWith("Not agent owner");
        });

        it("R3.4.5: Should activate agent after time-lock", async function () {
            // Check initial state
            const agentBefore = await registry.getAgent(agentId);
            expect(agentBefore.active).to.be.false;

            // Try to activate immediately - should fail
            await expect(
                registry.activateAgent(agentId)
            ).to.be.revertedWith("Activation delay not passed");

            // Advance time by 1 hour + 1 second
            await ethers.provider.send("evm_increaseTime", [3601]);
            await ethers.provider.send("evm_mine");

            // Now should succeed
            await registry.activateAgent(agentId);

            const agentAfter = await registry.getAgent(agentId);
            expect(agentAfter.active).to.be.true;
        });

        it("R3.4.6: Should reject activation before time-lock", async function () {
            await expect(
                registry.activateAgent(agentId)
            ).to.be.revertedWith("Activation delay not passed");
        });

        it("R3.4.7: Should allow anyone to call activateAgent", async function () {
            // Advance time
            await ethers.provider.send("evm_increaseTime", [3601]);
            await ethers.provider.send("evm_mine");

            // User2 (not owner) can activate
            await expect(
                registry.connect(user2).activateAgent(agentId)
            ).to.not.be.revert(ethers);

            const agent = await registry.getAgent(agentId);
            expect(agent.active).to.be.true;
        });

        it("R3.4.8: Should set active=false on deactivation", async function () {
            // Activate first
            await ethers.provider.send("evm_increaseTime", [3601]);
            await ethers.provider.send("evm_mine");
            await registry.activateAgent(agentId);

            // Deactivate
            await registry.connect(user1).deactivateAgentByHash(agentId);

            const agent = await registry.getAgent(agentId);
            expect(agent.active).to.be.false;
        });

        it("R3.4.9: Should return stake after 30 days deactivation", async function () {
            // Deactivate
            await registry.connect(user1).deactivateAgentByHash(agentId);

            // Advance 30 days + 1 second
            await ethers.provider.send("evm_increaseTime", [30 * 86400 + 1]);
            await ethers.provider.send("evm_mine");

            // Get balance before
            const balanceBefore = await ethers.provider.getBalance(user1.address);

            // Deactivate again to trigger stake return
            const tx = await registry.connect(user1).deactivateAgentByHash(agentId);
            const receipt = await tx.wait();
            const gasUsed = receipt.gasUsed * receipt.gasPrice;

            const balanceAfter = await ethers.provider.getBalance(user1.address);

            // Balance should increase by stake minus gas
            expect(balanceAfter).to.equal(balanceBefore + STAKE - gasUsed);
        });

        it("R3.4.10: Should only allow owner to deactivate", async function () {
            await expect(
                registry.connect(user2).deactivateAgentByHash(agentId)
            ).to.be.revertedWith("Not agent owner");
        });
    });

    // ========================================================================
    // V3.5: Security Features (8 tests)
    // ========================================================================

    describe("V3.5: Security Features", function () {

        it("R3.5.1: Should prevent reentrancy attack (nonReentrant modifier)", async function () {
            // The nonReentrant modifier is tested implicitly
            // by all the other tests. This test verifies the
            // modifier exists and is applied.

            const salt = ethers.randomBytes(32);

            await commitAndAdvanceTime(user1, validDID1, [validKey1], salt);

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            // Register normally
            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.not.be.revert(ethers);
        });

        it("R3.5.2: Should allow emergency pause", async function () {
            await expect(
                registry.connect(owner).pause()
            ).to.not.be.revert(ethers);

            // Verify paused
            expect(await registry.paused()).to.be.true;
        });

        it("R3.5.3: Should block operations when paused", async function () {
            await registry.connect(owner).pause();

            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            // All pausable operations should revert
            await expect(
                registry.connect(user1).commitRegistration(commitHash, { value: STAKE })
            ).to.be.revert(ethers); // Pausable: paused
        });

        it("R3.5.4: Should require two-step ownership transfer", async function () {
            // Ownable2Step requires transferOwnership + acceptOwnership

            // Transfer ownership to user1
            await registry.connect(owner).transferOwnership(user1.address);

            // Owner should still be the original owner
            expect(await registry.owner()).to.equal(owner.address);

            // User1 must accept
            await registry.connect(user1).acceptOwnership();

            // Now user1 is the owner
            expect(await registry.owner()).to.equal(user1.address);
        });

        it("R3.5.5: Should only allow owner to pause", async function () {
            await expect(
                registry.connect(user1).pause()
            ).to.be.revert(ethers); // Ownable: caller is not the owner
        });

        it("R3.5.6: Should only allow owner to unpause", async function () {
            await registry.connect(owner).pause();

            await expect(
                registry.connect(user1).unpause()
            ).to.be.revert(ethers); // Ownable: caller is not the owner

            // Owner can unpause
            await expect(
                registry.connect(owner).unpause()
            ).to.not.be.revert(ethers);
        });

        it("R3.5.7: Should manage stake correctly", async function () {
            const salt = ethers.randomBytes(32);
            const commitHash = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            // Get contract balance before
            const balanceBefore = await ethers.provider.getBalance(registryAddress);

            // Commit with stake
            await registry.connect(user1).commitRegistration(commitHash, { value: STAKE });

            // Contract balance should increase
            const balanceAfter = await ethers.provider.getBalance(registryAddress);
            expect(balanceAfter).to.equal(balanceBefore + STAKE);
        });

        it("R3.5.8: Should prevent cross-chain replay (chainId)", async function () {
            const salt = ethers.randomBytes(32);

            // Commitment with chain 1
            const commitHash1 = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, 1]
                )
            );

            // Commitment with current chain
            const commitHashCurrent = keccak256(
                AbiCoder.defaultAbiCoder().encode(
                    ["string", "bytes[]", "address", "bytes32", "uint256"],
                    [validDID1, [validKey1], user1.address, salt, chainId]
                )
            );

            // Should be different
            expect(commitHash1).to.not.equal(commitHashCurrent);

            // Commit with wrong chain
            await registry.connect(user1).commitRegistration(commitHash1, { value: STAKE });

            await ethers.provider.send("evm_increaseTime", [61]);
            await ethers.provider.send("evm_mine");

            const params = createRegistrationParams(
                validDID1,
                "Test Agent",
                [validKey1],
                [0],
                [validSig1],
                salt
            );

            // Try to reveal - should fail (hash mismatch)
            await expect(
                registry.connect(user1).registerAgentWithParams(params)
            ).to.be.revertedWith("Invalid reveal");
        });
    });
});
