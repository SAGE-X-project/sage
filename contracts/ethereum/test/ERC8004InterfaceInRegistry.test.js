const { expect } = require("chai");
const { ethers } = require("hardhat");

/**
 * ERC8004IdentityRegistryV4 Test Suite
 *
 * Test Coverage:
 * - V4.1: ERC-8004 Compliance (8 tests)
 *   - E4.1.1: Interface implementation
 *   - E4.1.2: registerAgent function
 *   - E4.1.3: resolveAgent function
 *   - E4.1.4: resolveAgentByAddress function
 *   - E4.1.5: isAgentActive function
 *   - E4.1.6: updateAgentEndpoint function
 *   - E4.1.7: deactivateAgent function
 *   - E4.1.8: Event emissions
 *
 * Total: 8 tests
 */
describe("ERC8004 Interface in AgentCardRegistry", () => {
    let agentRegistry;
    let verifyHook;
    let owner;
    let user1;
    let user2;

    const STAKE = ethers.parseEther("0.01");
    const ACTIVATION_DELAY = 3600; // 1 hour
    let chainId;

    // Test DIDs
    const validDID1 = "did:sage:ethereum:0x1234567890123456789012345678901234567890";
    const validDID2 = "did:sage:ethereum:0x2234567890123456789012345678901234567890";

    // Test keys and signatures
    let validKey1, validKey2, validSig1, validSig2;
    let registryAddress;

    // Helper functions
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

    async function commitAndAdvanceTime(user, did, keys, salt) {
        const commitHash = ethers.keccak256(
            ethers.AbiCoder.defaultAbiCoder().encode(
                ["string", "bytes[]", "address", "bytes32", "uint256"],
                [did, keys, user.address, salt, chainId]
            )
        );
        await agentRegistry.connect(user).commitRegistration(commitHash, { value: STAKE });
        await ethers.provider.send("evm_increaseTime", [61]);
        await ethers.provider.send("evm_mine");
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

        // Deploy VerifyHook
        const VerifyHook = await ethers.getContractFactory("AgentCardVerifyHook");
        verifyHook = await VerifyHook.deploy();
        await verifyHook.waitForDeployment();

        // Deploy AgentCardRegistry (now includes ERC-8004 interface)
        const Registry = await ethers.getContractFactory("AgentCardRegistry");
        agentRegistry = await Registry.deploy(await verifyHook.getAddress());
        await agentRegistry.waitForDeployment();
        registryAddress = await agentRegistry.getAddress();

        // Generate test keys
        validKey1 = ethers.randomBytes(33); // Compressed secp256k1
        validKey2 = ethers.randomBytes(33);

        // Create signatures
        validSig1 = await createSignature(user1);
        validSig2 = await createSignature(user2);

        // Register an agent for testing
        const salt1 = ethers.randomBytes(32);
        await commitAndAdvanceTime(user1, validDID1, [validKey1], salt1);

        const params1 = createRegistrationParams(
            validDID1,
            "Test Agent 1",
            [validKey1],
            [0], // ECDSA
            [validSig1],
            salt1
        );

        await agentRegistry.connect(user1).registerAgentWithParams(params1);

        // Activate agent
        const agentId1 = await agentRegistry.didToAgentId(validDID1);
        await ethers.provider.send("evm_increaseTime", [ACTIVATION_DELAY + 1]);
        await ethers.provider.send("evm_mine");
        await agentRegistry.activateAgent(agentId1);
    });

    describe("V4.1: ERC-8004 Compliance", () => {
        describe("E4.1.1: Interface Implementation", () => {
            it("Should implement all IERC8004 interface functions", async () => {
                // Verify all required functions exist
                expect(agentRegistry.interface.getFunction("registerAgent")).to.exist;
                expect(agentRegistry.interface.getFunction("resolveAgent")).to.exist;
                expect(agentRegistry.interface.getFunction("resolveAgentByAddress")).to.exist;
                expect(agentRegistry.interface.getFunction("isAgentActive")).to.exist;
                expect(agentRegistry.interface.getFunction("updateAgentEndpoint")).to.exist;
                expect(agentRegistry.interface.getFunction("deactivateAgent")).to.exist;
            });
        });

        describe("E4.1.2: registerAgent Function", () => {
            it("Should revert with instruction message", async () => {
                await expect(
                    agentRegistry.connect(user2).registerAgent(
                        validDID2,
                        "https://example.com"
                    )
                ).to.be.revertedWith("Use commitRegistration() for secure registration");
            });
        });

        describe("E4.1.3: resolveAgent Function", () => {
            it("Should return correct agent info by DID", async () => {
                const agentInfo = await agentRegistry.resolveAgent(validDID1);

                expect(agentInfo.agentId).to.equal(validDID1);
                expect(agentInfo.agentAddress).to.equal(user1.address);
                expect(agentInfo.endpoint).to.equal("https://example.com");
                expect(agentInfo.isActive).to.be.true;
                expect(agentInfo.registeredAt).to.be.gt(0);
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    agentRegistry.resolveAgent(nonExistentDID)
                ).to.be.reverted;
            });
        });

        describe("E4.1.4: resolveAgentByAddress Function", () => {
            it("Should return correct agent info by address", async () => {
                const agentInfo = await agentRegistry.resolveAgentByAddress(user1.address);

                expect(agentInfo.agentId).to.equal(validDID1);
                expect(agentInfo.agentAddress).to.equal(user1.address);
                expect(agentInfo.endpoint).to.equal("https://example.com");
                expect(agentInfo.isActive).to.be.true;
                expect(agentInfo.registeredAt).to.be.gt(0);
            });

            it("Should revert if no agent found for address", async () => {
                await expect(
                    agentRegistry.resolveAgentByAddress(user2.address)
                ).to.be.revertedWith("No agent found");
            });
        });

        describe("E4.1.5: isAgentActive Function", () => {
            it("Should return true for active agent", async () => {
                const isActive = await agentRegistry.isAgentActive(validDID1);
                expect(isActive).to.be.true;
            });

            it("Should return false after deactivation", async () => {
                // Deactivate via ERC8004 interface
                await agentRegistry.connect(user1).deactivateAgent(validDID1);

                const isActive = await agentRegistry.isAgentActive(validDID1);
                expect(isActive).to.be.false;
            });
        });

        describe("E4.1.6: updateAgentEndpoint Function", () => {
            it("Should update endpoint successfully", async () => {
                const newEndpoint = "https://new-endpoint.com";

                await expect(
                    agentRegistry.connect(user1).updateAgentEndpoint(validDID1, newEndpoint)
                ).to.emit(agentRegistry, "AgentEndpointUpdated")
                  .withArgs(validDID1, "https://example.com", newEndpoint);

                const agentInfo = await agentRegistry.resolveAgent(validDID1);
                expect(agentInfo.endpoint).to.equal(newEndpoint);
            });

            it("Should revert if not agent owner", async () => {
                await expect(
                    agentRegistry.connect(user2).updateAgentEndpoint(
                        validDID1,
                        "https://malicious.com"
                    )
                ).to.be.revertedWith("Not authorized");
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    agentRegistry.connect(user1).updateAgentEndpoint(
                        nonExistentDID,
                        "https://new.com"
                    )
                ).to.be.revertedWith("Agent not found");
            });
        });

        describe("E4.1.7: deactivateAgent Function", () => {
            it("Should deactivate agent successfully", async () => {
                await expect(
                    agentRegistry.connect(user1).deactivateAgent(validDID1)
                ).to.emit(agentRegistry, "AgentDeactivated")
                  .withArgs(validDID1, user1.address);

                const isActive = await agentRegistry.isAgentActive(validDID1);
                expect(isActive).to.be.false;
            });

            it("Should revert if not agent owner", async () => {
                await expect(
                    agentRegistry.connect(user2).deactivateAgent(validDID1)
                ).to.be.revertedWith("Not authorized");
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    agentRegistry.connect(user1).deactivateAgent(nonExistentDID)
                ).to.be.revertedWith("Agent not found");
            });
        });

        describe("E4.1.8: Event Emissions", () => {
            it("Should emit AgentEndpointUpdated event", async () => {
                const newEndpoint = "https://updated.com";

                await expect(
                    agentRegistry.connect(user1).updateAgentEndpoint(validDID1, newEndpoint)
                ).to.emit(agentRegistry, "AgentEndpointUpdated")
                  .withArgs(validDID1, "https://example.com", newEndpoint);
            });

            it("Should emit AgentDeactivated event", async () => {
                await expect(
                    agentRegistry.connect(user1).deactivateAgent(validDID1)
                ).to.emit(agentRegistry, "AgentDeactivated")
                  .withArgs(validDID1, user1.address);
            });

            it("Should not emit AgentRegistered (registration not supported)", async () => {
                // This test verifies that registerAgent reverts
                // and therefore no event is emitted
                await expect(
                    agentRegistry.connect(user2).registerAgent(
                        validDID2,
                        "https://example.com"
                    )
                ).to.be.revertedWith("Use commitRegistration() for secure registration");
            });
        });
    });
});
