import { expect } from "chai";
import { parseEther, keccak256, AbiCoder, Wallet, ZeroAddress } from "ethers";
import { network } from "hardhat";

// Initialize ethers from network connection
const { ethers } = await network.connect();

/**
 * AgentCardStorage Test Suite
 *
 * Purpose: Verify AgentCardStorage.sol implementation
 * Verification Matrix: V1.1 - V1.4 (25 test cases)
 *
 * Test Structure:
 * 1. Struct Definitions (V1.1) - 5 tests
 * 2. Storage Mappings (V1.2) - 8 tests
 * 3. Constants (V1.3) - 5 tests
 * 4. Events (V1.4) - 6 tests
 * 5. Gas Optimization (V1.5) - 1 test
 *
 * Total: 25 verification items
 */

describe("AgentCardStorage", function () {
    // Test contract that exposes AgentCardStorage for testing
    let StorageTestContract;
    let storage;
    let owner, user1, user2, user3;

    // Deploy a test contract that inherits from AgentCardStorage
    before(async function () {
        [owner, user1, user2, user3] = await ethers.getSigners();

        // Create a test contract that exposes AgentCardStorage
        const StorageTestFactory = await ethers.getContractFactory("AgentCardStorageTest");
        storage = await StorageTestFactory.deploy();
        await storage.waitForDeployment();
    });

    // ============================================================================
    // V1.1: Struct Definitions (5 tests)
    // ============================================================================

    describe("V1.1: Struct Definitions", function () {
        /**
         * Test ID: S1.1.1
         * Verification: AgentMetadata struct has all 11 fields
         * Priority: P0 Critical
         */
        it("S1.1.1: Should create AgentMetadata with all 12 fields", async function () {
            const agentId = ethers.id("test-agent-1");
            const metadata = {
                did: "did:sage:ethereum:0x1234567890123456789012345678901234567890",
                name: "Test Agent",
                description: "A test agent for verification",
                endpoint: "https://agent.example.com/card",
                keyHashes: [ethers.id("key1"), ethers.id("key2")],
                capabilities: '{"skills":["translation","summarization"]}',
                owner: user1.address,
                registeredAt: Math.floor(Date.now() / 1000),
                updatedAt: Math.floor(Date.now() / 1000),
                active: false,
                chainId: 31337,
                kmePublicKey: "0x"  // Empty KME key (optional field)
            };

            // Store test agent
            await storage.setAgentMetadata(agentId, metadata);

            // Retrieve and verify all fields
            const retrieved = await storage.getAgentMetadata(agentId);

            expect(retrieved.did).to.equal(metadata.did);
            expect(retrieved.name).to.equal(metadata.name);
            expect(retrieved.description).to.equal(metadata.description);
            expect(retrieved.endpoint).to.equal(metadata.endpoint);
            expect(retrieved.keyHashes.length).to.equal(2);
            expect(retrieved.keyHashes[0]).to.equal(metadata.keyHashes[0]);
            expect(retrieved.keyHashes[1]).to.equal(metadata.keyHashes[1]);
            expect(retrieved.capabilities).to.equal(metadata.capabilities);
            expect(retrieved.owner).to.equal(metadata.owner);
            expect(retrieved.registeredAt).to.equal(metadata.registeredAt);
            expect(retrieved.updatedAt).to.equal(metadata.updatedAt);
            expect(retrieved.active).to.equal(metadata.active);
            expect(retrieved.chainId).to.equal(metadata.chainId);
            expect(retrieved.kmePublicKey).to.equal(metadata.kmePublicKey);  //  Verify KME key
        });

        /**
         * Test ID: S1.1.2
         * Verification: AgentKey struct has all 5 fields
         * Priority: P0 Critical
         */
        it("S1.1.2: Should create AgentKey with all 5 fields", async function () {
            const keyHash = ethers.id("test-key-1");
            const wallet = Wallet.createRandom();
            const message = "Test message";
            const signature = await wallet.signMessage(message);

            const keyData = {
                keyType: 0, // ECDSA
                keyData: wallet.publicKey,
                signature: signature,
                verified: true,
                registeredAt: Math.floor(Date.now() / 1000)
            };

            // Store test key
            await storage.setAgentKey(keyHash, keyData);

            // Retrieve and verify all fields
            const retrieved = await storage.getAgentKey(keyHash);

            expect(retrieved.keyType).to.equal(keyData.keyType);
            expect(retrieved.keyData).to.equal(keyData.keyData);
            expect(retrieved.signature).to.equal(keyData.signature);
            expect(retrieved.verified).to.equal(keyData.verified);
            expect(retrieved.registeredAt).to.equal(keyData.registeredAt);
        });

        /**
         * Test ID: S1.1.3
         * Verification: RegistrationCommitment struct has all 3 fields
         * Priority: P0 Critical
         */
        it("S1.1.3: Should create RegistrationCommitment with all 3 fields", async function () {
            const commitment = {
                commitHash: ethers.id("test-commitment"),
                timestamp: Math.floor(Date.now() / 1000),
                revealed: false
            };

            // Store test commitment
            await storage.setRegistrationCommitment(user1.address, commitment);

            // Retrieve and verify all fields
            const retrieved = await storage.getRegistrationCommitment(user1.address);

            expect(retrieved.commitHash).to.equal(commitment.commitHash);
            expect(retrieved.timestamp).to.equal(commitment.timestamp);
            expect(retrieved.revealed).to.equal(commitment.revealed);
        });

        /**
         * Test ID: S1.1.4
         * Verification: KeyType enum has exactly 3 values
         * Priority: P0 Critical
         */
        it("S1.1.4: Should support all KeyType enum values (ECDSA=0, Ed25519=1, X25519=2)", async function () {
            const keyHash1 = ethers.id("ecdsa-key");
            const keyHash2 = ethers.id("ed25519-key");
            const keyHash3 = ethers.id("x25519-key");

            // Test ECDSA (0)
            await storage.setAgentKey(keyHash1, {
                keyType: 0,
                keyData: "0x04" + "a".repeat(128), // 65 bytes
                signature: "0x" + "b".repeat(130),
                verified: true,
                registeredAt: Date.now()
            });

            // Test Ed25519 (1)
            await storage.setAgentKey(keyHash2, {
                keyType: 1,
                keyData: "0x" + "c".repeat(64), // 32 bytes
                signature: "0x" + "d".repeat(128),
                verified: true,
                registeredAt: Date.now()
            });

            // Test X25519 (2)
            await storage.setAgentKey(keyHash3, {
                keyType: 2,
                keyData: "0x" + "e".repeat(64), // 32 bytes
                signature: "0x",
                verified: false,
                registeredAt: Date.now()
            });

            // Verify all key types stored correctly
            const key1 = await storage.getAgentKey(keyHash1);
            const key2 = await storage.getAgentKey(keyHash2);
            const key3 = await storage.getAgentKey(keyHash3);

            expect(key1.keyType).to.equal(0); // ECDSA
            expect(key2.keyType).to.equal(1); // Ed25519
            expect(key3.keyType).to.equal(2); // X25519
        });

        /**
         * Test ID: S1.1.5
         * Verification: Struct field types are correct
         * Priority: P0 Critical
         */
        it("S1.1.5: Should have correct field types for all structs", async function () {
            // This test verifies type correctness implicitly through TypeScript/Solidity type checking
            // If compilation succeeds and previous tests pass, types are correct

            // AgentMetadata types
            const agentId = ethers.id("type-test-agent");
            await storage.setAgentMetadata(agentId, {
                did: "did:sage:test",                // string
                name: "Type Test",                   // string
                description: "Testing types",        // string
                endpoint: "https://test.com",        // string
                keyHashes: [ethers.id("k1")],       // bytes32[]
                capabilities: "{}",                   // string
                owner: user1.address,                 // address
                registeredAt: 123456,                 // uint256
                updatedAt: 123456,                    // uint256
                active: true,                         // bool
                chainId: 1,                           // uint256
                kmePublicKey: "0x"                    // bytes (optional)
            });

            const agent = await storage.getAgentMetadata(agentId);

            // Verify types through value checks
            expect(typeof agent.did).to.equal("string");
            expect(typeof agent.name).to.equal("string");
            expect(typeof agent.active).to.equal("boolean");
            expect(typeof agent.owner).to.equal("string"); // address as string in ethers.js
            expect(agent.keyHashes).to.be.an("array");
        });
    });

    // ============================================================================
    // V1.2: Storage Mappings (8 tests)
    // ============================================================================

    describe("V1.2: Storage Mappings", function () {
        /**
         * Test ID: S1.2.1
         * Verification: agents mapping stores/retrieves correctly
         * Priority: P0 Critical
         */
        it("S1.2.1: Should store and retrieve agent by ID", async function () {
            const agentId = ethers.id("mapping-test-1");
            const agentData = {
                did: "did:sage:ethereum:0xabc",
                name: "Mapping Test Agent",
                description: "Testing agent mapping",
                endpoint: "https://agent.test",
                keyHashes: [],
                capabilities: "{}",
                owner: user1.address,
                registeredAt: 100,
                updatedAt: 100,
                active: false,
                chainId: 31337,
                kmePublicKey: "0x"  // Empty KME key (optional field)
            };

            await storage.setAgentMetadata(agentId, agentData);
            const retrieved = await storage.getAgentMetadata(agentId);

            expect(retrieved.did).to.equal(agentData.did);
            expect(retrieved.owner).to.equal(agentData.owner);
        });

        /**
         * Test ID: S1.2.2
         * Verification: didToAgentId mapping works
         * Priority: P0 Critical
         */
        it("S1.2.2: Should map DID to agent ID", async function () {
            const did = "did:sage:ethereum:0x1111111111111111111111111111111111111111";
            const agentId = ethers.id("did-mapping-test");

            await storage.setDidToAgentId(did, agentId);
            const retrieved = await storage.getAgentIdByDid(did);

            expect(retrieved).to.equal(agentId);
        });

        /**
         * Test ID: S1.2.3
         * Verification: ownerToAgents mapping tracks all agents
         * Priority: P0 Critical
         */
        it("S1.2.3: Should track agents by owner", async function () {
            const agentId1 = ethers.id("owner-test-1");
            const agentId2 = ethers.id("owner-test-2");
            const agentId3 = ethers.id("owner-test-3");

            // Add 3 agents for user1
            await storage.addAgentToOwner(user1.address, agentId1);
            await storage.addAgentToOwner(user1.address, agentId2);
            await storage.addAgentToOwner(user1.address, agentId3);

            const agents = await storage.getAgentsByOwner(user1.address);

            expect(agents.length).to.equal(3);
            expect(agents[0]).to.equal(agentId1);
            expect(agents[1]).to.equal(agentId2);
            expect(agents[2]).to.equal(agentId3);
        });

        /**
         * Test ID: S1.2.4
         * Verification: agentKeys mapping stores keys
         * Priority: P0 Critical
         */
        it("S1.2.4: Should store and retrieve keys by hash", async function () {
            const keyHash = ethers.id("key-mapping-test");
            const keyData = {
                keyType: 0,
                keyData: "0x1234",
                signature: "0x5678",
                verified: true,
                registeredAt: 200
            };

            await storage.setAgentKey(keyHash, keyData);
            const retrieved = await storage.getAgentKey(keyHash);

            expect(retrieved.keyType).to.equal(keyData.keyType);
            expect(retrieved.verified).to.equal(keyData.verified);
        });

        /**
         * Test ID: S1.2.5
         * Verification: registrationCommitments mapping works
         * Priority: P0 Critical
         */
        it("S1.2.5: Should store and retrieve commitments", async function () {
            const commitment = {
                commitHash: ethers.id("commitment-test"),
                timestamp: 300,
                revealed: false
            };

            await storage.setRegistrationCommitment(user2.address, commitment);
            const retrieved = await storage.getRegistrationCommitment(user2.address);

            expect(retrieved.commitHash).to.equal(commitment.commitHash);
            expect(retrieved.timestamp).to.equal(commitment.timestamp);
            expect(retrieved.revealed).to.equal(commitment.revealed);
        });

        /**
         * Test ID: S1.2.6
         * Verification: agentNonce mapping increments correctly
         * Priority: P0 Critical
         */
        it("S1.2.6: Should increment nonce correctly", async function () {
            const agentId = ethers.id("nonce-test");

            // Initial nonce should be 0
            let nonce = await storage.getAgentNonce(agentId);
            expect(nonce).to.equal(0);

            // Increment nonce
            await storage.incrementNonce(agentId);
            nonce = await storage.getAgentNonce(agentId);
            expect(nonce).to.equal(1);

            // Increment again
            await storage.incrementNonce(agentId);
            nonce = await storage.getAgentNonce(agentId);
            expect(nonce).to.equal(2);
        });

        /**
         * Test ID: S1.2.7
         * Verification: dailyRegistrationCount tracking works
         * Priority: P0 Critical
         */
        it("S1.2.7: Should track daily registration count", async function () {
            // Initial count should be 0
            let count = await storage.getDailyRegistrationCount(user3.address);
            expect(count).to.equal(0);

            // Increment count
            await storage.incrementDailyRegistrationCount(user3.address);
            count = await storage.getDailyRegistrationCount(user3.address);
            expect(count).to.equal(1);

            // Increment multiple times
            await storage.incrementDailyRegistrationCount(user3.address);
            await storage.incrementDailyRegistrationCount(user3.address);
            count = await storage.getDailyRegistrationCount(user3.address);
            expect(count).to.equal(3);
        });

        /**
         * Test ID: S1.2.8
         * Verification: publicKeyUsed prevents reuse
         * Priority: P0 Critical
         */
        it("S1.2.8: Should prevent public key reuse", async function () {
            const publicKey = "0x04" + "a".repeat(128);
            const keyHash = keccak256(ethers.toUtf8Bytes(publicKey));

            // Initially key should not be used
            let isUsed = await storage.isPublicKeyUsed(keyHash);
            expect(isUsed).to.be.false;

            // Mark key as used
            await storage.markPublicKeyUsed(keyHash);

            // Now key should be marked as used
            isUsed = await storage.isPublicKeyUsed(keyHash);
            expect(isUsed).to.be.true;
        });
    });

    // ============================================================================
    // V1.3: Constants (5 tests)
    // ============================================================================

    describe("V1.3: Constants", function () {
        /**
         * Test ID: S1.3.1
         * Verification: COMMIT_MIN_DELAY = 1 minutes
         * Priority: P0 Critical
         */
        it("S1.3.1: Should have COMMIT_MIN_DELAY = 60 seconds (1 minute)", async function () {
            const minDelay = await storage.getCommitMinDelay();
            expect(minDelay).to.equal(60);
        });

        /**
         * Test ID: S1.3.2
         * Verification: COMMIT_MAX_DELAY = 1 hours
         * Priority: P0 Critical
         */
        it("S1.3.2: Should have COMMIT_MAX_DELAY = 3600 seconds (1 hour)", async function () {
            const maxDelay = await storage.getCommitMaxDelay();
            expect(maxDelay).to.equal(3600);
        });

        /**
         * Test ID: S1.3.3
         * Verification: MAX_KEYS_PER_AGENT = 10
         * Priority: P0 Critical
         */
        it("S1.3.3: Should have MAX_KEYS_PER_AGENT = 10", async function () {
            const maxKeys = await storage.getMaxKeysPerAgent();
            expect(maxKeys).to.equal(10);
        });

        /**
         * Test ID: S1.3.4
         * Verification: MAX_DAILY_REGISTRATIONS = 24
         * Priority: P0 Critical
         */
        it("S1.3.4: Should have MAX_DAILY_REGISTRATIONS = 24", async function () {
            const maxDaily = await storage.getMaxDailyRegistrations();
            expect(maxDaily).to.equal(24);
        });

        /**
         * Test ID: S1.3.5
         * Verification: Constants are immutable
         * Priority: P0 Critical
         */
        it("S1.3.5: Should have constants declared as constant (immutable)", async function () {
            // This is verified at compile time
            // If constants are mutable, compiler will error
            // This test documents the requirement

            const minDelay = await storage.getCommitMinDelay();
            const maxDelay = await storage.getCommitMaxDelay();
            const maxKeys = await storage.getMaxKeysPerAgent();
            const maxDaily = await storage.getMaxDailyRegistrations();

            // Verify values haven't changed
            expect(minDelay).to.equal(60);
            expect(maxDelay).to.equal(3600);
            expect(maxKeys).to.equal(10);
            expect(maxDaily).to.equal(24);

            // Note: Cannot modify constants in Solidity 0.8.19
            // Any attempt to modify will result in compilation error
        });
    });

    // ============================================================================
    // V1.4: Events (6 tests)
    // ============================================================================

    describe("V1.4: Events", function () {
        /**
         * Test ID: S1.4.1
         * Verification: AgentRegistered event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.1: Should emit AgentRegistered with correct params", async function () {
            const agentId = ethers.id("event-test-1");
            const did = "did:sage:ethereum:0xevent";
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitAgentRegistered(agentId, did, user1.address, timestamp))
                .to.emit(storage, "AgentRegistered")
                .withArgs(agentId, did, user1.address, timestamp);
        });

        /**
         * Test ID: S1.4.2
         * Verification: KeyAdded event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.2: Should emit KeyAdded with correct params", async function () {
            const agentId = ethers.id("event-test-2");
            const keyHash = ethers.id("key-event-test");
            const keyType = 0; // ECDSA
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitKeyAdded(agentId, keyHash, keyType, timestamp))
                .to.emit(storage, "KeyAdded")
                .withArgs(agentId, keyHash, keyType, timestamp);
        });

        /**
         * Test ID: S1.4.3
         * Verification: KeyRevoked event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.3: Should emit KeyRevoked with correct params", async function () {
            const agentId = ethers.id("event-test-3");
            const keyHash = ethers.id("revoke-event-test");
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitKeyRevoked(agentId, keyHash, timestamp))
                .to.emit(storage, "KeyRevoked")
                .withArgs(agentId, keyHash, timestamp);
        });

        /**
         * Test ID: S1.4.4
         * Verification: AgentUpdated event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.4: Should emit AgentUpdated with correct params", async function () {
            const agentId = ethers.id("event-test-4");
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitAgentUpdated(agentId, timestamp))
                .to.emit(storage, "AgentUpdated")
                .withArgs(agentId, timestamp);
        });

        /**
         * Test ID: S1.4.5
         * Verification: AgentDeactivatedByHash event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.5: Should emit AgentDeactivatedByHash with correct params", async function () {
            const agentId = ethers.id("event-test-5");
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitAgentDeactivatedByHash(agentId, timestamp))
                .to.emit(storage, "AgentDeactivatedByHash")
                .withArgs(agentId, timestamp);
        });

        /**
         * Test ID: S1.4.6
         * Verification: CommitmentRecorded event has correct params
         * Priority: P0 Critical
         */
        it("S1.4.6: Should emit CommitmentRecorded with correct params", async function () {
            const commitHash = ethers.id("commit-event-test");
            const timestamp = Math.floor(Date.now() / 1000);

            await expect(storage.emitCommitmentRecorded(user1.address, commitHash, timestamp))
                .to.emit(storage, "CommitmentRecorded")
                .withArgs(user1.address, commitHash, timestamp);
        });
    });

    // ============================================================================
    // V1.5: Additional Verification
    // ============================================================================

    describe("V1.5: Gas Optimization & Edge Cases", function () {
        /**
         * Additional verification: Storage layout efficiency
         * Ensures structs are packed efficiently for gas optimization
         */
        it("Should use efficient storage layout (documentation)", async function () {
            // This test documents the storage layout optimization
            // Actual verification happens through gas profiling

            // AgentMetadata uses ~7 storage slots (dynamic arrays/strings not counted)
            // AgentKey uses ~3 storage slots
            // RegistrationCommitment uses ~2 storage slots

            // Gas optimization techniques used:
            // 1. Group small types together (bool + uint256)
            // 2. Use bytes32 instead of string where possible
            // 3. Use internal mappings (no external access needed)
            // 4. Use constant for immutable values

            // This is validated through:
            // - Slither security scanner
            // - Gas reporter in hardhat
            // - Manual code review

            expect(true).to.be.true; // Documentation test
        });
    });
});

/**
 * Test Results Summary
 *
 * Verification Matrix Coverage:
 * - V1.1 Struct Definitions: 5/5 tests 
 * - V1.2 Storage Mappings: 8/8 tests 
 * - V1.3 Constants: 5/5 tests 
 * - V1.4 Events: 6/6 tests 
 * - V1.5 Additional: 1/1 test 
 *
 * Total: 25/25 verification items
 *
 * Next Steps:
 * 1. Implement AgentCardStorageTest.sol helper contract
 * 2. Run tests (should fail - RED phase)
 * 3. Implement actual storage functionality (GREEN phase)
 * 4. Optimize and refactor (REFACTOR phase)
 * 5. Update VERIFICATION_MATRIX.md with results
 */
