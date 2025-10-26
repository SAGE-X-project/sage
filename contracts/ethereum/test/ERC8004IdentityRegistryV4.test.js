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
describe("ERC8004IdentityRegistryV4", () => {
    let erc8004Registry;
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

        // Deploy AgentCardRegistry
        const Registry = await ethers.getContractFactory("AgentCardRegistry");
        agentRegistry = await Registry.deploy(await verifyHook.getAddress());
        await agentRegistry.waitForDeployment();
        registryAddress = await agentRegistry.getAddress();

        // Deploy ERC8004IdentityRegistryV4
        const ERC8004 = await ethers.getContractFactory("ERC8004IdentityRegistryV4");
        erc8004Registry = await ERC8004.deploy(registryAddress);
        await erc8004Registry.waitForDeployment();

        // Generate test keys
        validKey1 = ethers.randomBytes(33); // Compressed secp256k1
        validKey2 = ethers.randomBytes(33);

        // Create signatures
        validSig1 = await createSignature(user1);
        validSig2 = await createSignature(user2);

        // Register an agent via AgentCardRegistry for testing
        // This is needed because registerAgent() in ERC8004 reverts
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

        await agentRegistry.connect(user1).registerAgent(params1);

        // Activate agent
        const agentId1 = await agentRegistry.didToAgentId(validDID1);
        await ethers.provider.send("evm_increaseTime", [ACTIVATION_DELAY + 1]);
        await ethers.provider.send("evm_mine");
        await agentRegistry.activateAgent(agentId1);

        // Approve ERC8004 contract as operator
        const erc8004Address = await erc8004Registry.getAddress();
        await agentRegistry.connect(user1).setApprovalForAgent(agentId1, erc8004Address, true);
    });

    describe("V4.1: ERC-8004 Compliance", () => {
        describe("E4.1.1: Interface Implementation", () => {
            it("Should implement all IERC8004 interface functions", async () => {
                // Verify all required functions exist
                expect(erc8004Registry.interface.getFunction("registerAgent")).to.exist;
                expect(erc8004Registry.interface.getFunction("resolveAgent")).to.exist;
                expect(erc8004Registry.interface.getFunction("resolveAgentByAddress")).to.exist;
                expect(erc8004Registry.interface.getFunction("isAgentActive")).to.exist;
                expect(erc8004Registry.interface.getFunction("updateAgentEndpoint")).to.exist;
                expect(erc8004Registry.interface.getFunction("deactivateAgent")).to.exist;
            });
        });

        describe("E4.1.2: registerAgent Function", () => {
            it("Should revert with instruction message", async () => {
                await expect(
                    erc8004Registry.connect(user2).registerAgent(
                        validDID2,
                        "https://example.com"
                    )
                ).to.be.revertedWith("Use AgentCardRegistry.commitRegistration() for full flow");
            });
        });

        describe("E4.1.3: resolveAgent Function", () => {
            it("Should return correct agent info by DID", async () => {
                const agentInfo = await erc8004Registry.resolveAgent(validDID1);

                expect(agentInfo.agentId).to.equal(validDID1);
                expect(agentInfo.agentAddress).to.equal(user1.address);
                expect(agentInfo.endpoint).to.equal("https://example.com");
                expect(agentInfo.isActive).to.be.true;
                expect(agentInfo.registeredAt).to.be.gt(0);
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    erc8004Registry.resolveAgent(nonExistentDID)
                ).to.be.reverted;
            });
        });

        describe("E4.1.4: resolveAgentByAddress Function", () => {
            it("Should return correct agent info by address", async () => {
                const agentInfo = await erc8004Registry.resolveAgentByAddress(user1.address);

                expect(agentInfo.agentId).to.equal(validDID1);
                expect(agentInfo.agentAddress).to.equal(user1.address);
                expect(agentInfo.endpoint).to.equal("https://example.com");
                expect(agentInfo.isActive).to.be.true;
                expect(agentInfo.registeredAt).to.be.gt(0);
            });

            it("Should revert if no agent found for address", async () => {
                await expect(
                    erc8004Registry.resolveAgentByAddress(user2.address)
                ).to.be.revertedWith("No agent found");
            });
        });

        describe("E4.1.5: isAgentActive Function", () => {
            it("Should return true for active agent", async () => {
                const isActive = await erc8004Registry.isAgentActive(validDID1);
                expect(isActive).to.be.true;
            });

            it("Should return false after deactivation", async () => {
                // Deactivate via ERC8004 interface
                await erc8004Registry.connect(user1).deactivateAgent(validDID1);

                const isActive = await erc8004Registry.isAgentActive(validDID1);
                expect(isActive).to.be.false;
            });
        });

        describe("E4.1.6: updateAgentEndpoint Function", () => {
            it("Should update endpoint successfully", async () => {
                const newEndpoint = "https://new-endpoint.com";

                await expect(
                    erc8004Registry.connect(user1).updateAgentEndpoint(validDID1, newEndpoint)
                ).to.emit(erc8004Registry, "AgentEndpointUpdated")
                  .withArgs(validDID1, "https://example.com", newEndpoint);

                const agentInfo = await erc8004Registry.resolveAgent(validDID1);
                expect(agentInfo.endpoint).to.equal(newEndpoint);
            });

            it("Should revert if not agent owner", async () => {
                await expect(
                    erc8004Registry.connect(user2).updateAgentEndpoint(
                        validDID1,
                        "https://malicious.com"
                    )
                ).to.be.revertedWith("Not agent owner");
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    erc8004Registry.connect(user1).updateAgentEndpoint(
                        nonExistentDID,
                        "https://new.com"
                    )
                ).to.be.revertedWith("Agent not found");
            });
        });

        describe("E4.1.7: deactivateAgent Function", () => {
            it("Should deactivate agent successfully", async () => {
                await expect(
                    erc8004Registry.connect(user1).deactivateAgent(validDID1)
                ).to.emit(erc8004Registry, "AgentDeactivated")
                  .withArgs(validDID1, user1.address);

                const isActive = await erc8004Registry.isAgentActive(validDID1);
                expect(isActive).to.be.false;
            });

            it("Should revert if not agent owner", async () => {
                await expect(
                    erc8004Registry.connect(user2).deactivateAgent(validDID1)
                ).to.be.revertedWith("Not agent owner");
            });

            it("Should revert if agent not found", async () => {
                const nonExistentDID = "did:sage:ethereum:0x9999999999999999999999999999999999999999";

                await expect(
                    erc8004Registry.connect(user1).deactivateAgent(nonExistentDID)
                ).to.be.revertedWith("Agent not found");
            });
        });

        describe("E4.1.8: Event Emissions", () => {
            it("Should emit AgentEndpointUpdated event", async () => {
                const newEndpoint = "https://updated.com";

                await expect(
                    erc8004Registry.connect(user1).updateAgentEndpoint(validDID1, newEndpoint)
                ).to.emit(erc8004Registry, "AgentEndpointUpdated")
                  .withArgs(validDID1, "https://example.com", newEndpoint);
            });

            it("Should emit AgentDeactivated event", async () => {
                await expect(
                    erc8004Registry.connect(user1).deactivateAgent(validDID1)
                ).to.emit(erc8004Registry, "AgentDeactivated")
                  .withArgs(validDID1, user1.address);
            });

            it("Should not emit AgentRegistered (registration not supported)", async () => {
                // This test verifies that registerAgent reverts
                // and therefore no event is emitted
                await expect(
                    erc8004Registry.connect(user2).registerAgent(
                        validDID2,
                        "https://example.com"
                    )
                ).to.be.revertedWith("Use AgentCardRegistry.commitRegistration() for full flow");
            });
        });
    });
});
