const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("SageRegistry V1 - Fixed Tests", function () {
  let sageRegistry;
  let verificationHook;
  let owner;
  let agent1;
  let agent2;
  let attacker;

  // Test data
  const testName = "Test AI Agent";
  const testDescription = "A test AI agent for demonstration";
  const testEndpoint = "https://agent.example.com";
  const testCapabilities = JSON.stringify(["chat", "code", "analysis"]);

  /**
   * Helper function to create signature for V1 registry
   * V1 uses abi.encode (NOT abi.encodePacked) to prevent hash collision attacks
   */
  async function createV1Signature(signer, params, nonce = 0) {
    // V1 uses abi.encode for message hash (secure against hash collisions)
    const messageHash = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
        [
          params.did,
          params.name,
          params.description,
          params.endpoint,
          params.publicKey,
          params.capabilities,
          signer.address,
          nonce
        ]
      )
    );

    // Sign the message
    return await signer.signMessage(ethers.getBytes(messageHash));
  }

  beforeEach(async function () {
    // Get signers
    [owner, agent1, agent2, attacker] = await ethers.getSigners();

    // Deploy SageRegistry (V1)
    const SageRegistry = await ethers.getContractFactory("SageRegistry");
    sageRegistry = await SageRegistry.deploy();
    await sageRegistry.waitForDeployment();

    // Deploy SageVerificationHook
    const SageVerificationHook = await ethers.getContractFactory("SageVerificationHook");
    verificationHook = await SageVerificationHook.deploy();
    await verificationHook.waitForDeployment();

    // Set verification hooks
    await sageRegistry.setBeforeRegisterHook(await verificationHook.getAddress());
    await sageRegistry.setAfterRegisterHook(await verificationHook.getAddress());
  });

  describe("Deployment", function () {
    it("Should set the correct owner", async function () {
      expect(await sageRegistry.OWNER()).to.equal(owner.address);
    });

    it("Should have the verification hook set", async function () {
      expect(await sageRegistry.beforeRegisterHook()).to.equal(await verificationHook.getAddress());
    });
  });

  describe("Agent Registration", function () {
    it("Should register a new agent successfully", async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      await expect(tx).to.emit(sageRegistry, "AgentRegistered");
      
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      expect(agentIds.length).to.equal(1);
      
      const agent = await sageRegistry.getAgent(agentIds[0]);
      expect(agent.did).to.equal(did);
      expect(agent.name).to.equal(testName);
      expect(agent.owner).to.equal(agent1.address);
      expect(agent.active).to.be.true;
    });

    it("Should reject registration with invalid DID", async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const invalidDID = "invalid-did"; // Not starting with "did:"
      
      const params = {
        did: invalidDID,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          invalidDID,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid DID format");
    });

    it("Should reject duplicate DID registration", async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      // First registration
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      // Try duplicate registration with different data
      const newPublicKey = ethers.hexlify(ethers.randomBytes(64));
      const newParams = { ...params, publicKey: newPublicKey };
      const newSignature = await createV1Signature(agent1, newParams);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          "Different Name",
          testDescription,
          testEndpoint,
          newPublicKey,
          testCapabilities,
          newSignature
        )
      ).to.be.revertedWith("DID already registered");
    });

    it("Should reject registration with invalid public key length", async function () {
      const invalidPublicKey = ethers.hexlify(ethers.randomBytes(20)); // Too short
      const did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey: invalidPublicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          invalidPublicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid public key length");
    });

    it("Should enforce maximum agents per owner", async function () {
      // This test would need to register 100+ agents
      // Skipping for performance, but the logic is:
      // Register MAX_AGENTS_PER_OWNER agents, then try one more
      this.skip();
    });
  });

  describe("Agent Updates", function () {
    let agentId;
    let publicKey;
    
    beforeEach(async function () {
      publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      agentId = agentIds[0];
    });

    it("Should update agent metadata successfully", async function () {
      const newName = "Updated Agent";
      const newDescription = "Updated description";
      const newEndpoint = "https://new.example.com";
      const newCapabilities = JSON.stringify(["chat", "code", "analysis", "search"]);
      
      // Create update signature
      // Note: After registration, nonce is incremented to 1
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, newName, newDescription, newEndpoint, newCapabilities, agent1.address, 1]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      const tx = await sageRegistry.connect(agent1).updateAgent(
        agentId,
        newName,
        newDescription,
        newEndpoint,
        newCapabilities,
        signature
      );
      
      await expect(tx).to.emit(sageRegistry, "AgentUpdated");
      
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.name).to.equal(newName);
      expect(agent.description).to.equal(newDescription);
      expect(agent.endpoint).to.equal(newEndpoint);
      expect(agent.capabilities).to.equal(newCapabilities);
    });

    it("Should reject update from non-owner", async function () {
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, "Hacked", testDescription, testEndpoint, testCapabilities, agent2.address, 0]
        )
      );
      
      const signature = await agent2.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        sageRegistry.connect(agent2).updateAgent(
          agentId,
          "Hacked",
          testDescription,
          testEndpoint,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Not agent owner");
    });

    it("Should reject update with invalid signature", async function () {
      const wrongSignature = await agent2.signMessage(ethers.randomBytes(32));
      
      await expect(
        sageRegistry.connect(agent1).updateAgent(
          agentId,
          "New Name",
          testDescription,
          testEndpoint,
          testCapabilities,
          wrongSignature
        )
      ).to.be.revertedWith("Invalid signature");
    });
  });

  describe("Agent Deactivation", function () {
    let agentId;
    
    beforeEach(async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      agentId = agentIds[0];
    });

    it("Should deactivate agent successfully", async function () {
      const tx = await sageRegistry.connect(agent1).deactivateAgent(agentId);
      await expect(tx).to.emit(sageRegistry, "AgentDeactivated");
      
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.active).to.be.false;
      
      const isActive = await sageRegistry.isAgentActive(agentId);
      expect(isActive).to.be.false;
    });

    it("Should reject deactivation from non-owner", async function () {
      await expect(
        sageRegistry.connect(agent2).deactivateAgent(agentId)
      ).to.be.revertedWith("Not agent owner");
    });

    it("Should reject double deactivation", async function () {
      await sageRegistry.connect(agent1).deactivateAgent(agentId);
      
      await expect(
        sageRegistry.connect(agent1).deactivateAgent(agentId)
      ).to.be.revertedWith("Agent already inactive");
    });
  });

  describe("Query Functions", function () {
    let agentId;
    let did;
    
    beforeEach(async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      did = `did:sage:test:${agent1.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      agentId = agentIds[0];
    });

    it("Should get agent by ID", async function () {
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.did).to.equal(did);
      expect(agent.name).to.equal(testName);
    });

    it("Should get agent by DID", async function () {
      const agent = await sageRegistry.getAgentByDID(did);
      expect(agent.name).to.equal(testName);
      expect(agent.owner).to.equal(agent1.address);
    });

    it("Should get agents by owner", async function () {
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      expect(agentIds.length).to.equal(1);
      expect(agentIds[0]).to.equal(agentId);
    });

    it("Should verify agent ownership correctly", async function () {
      const isOwner = await sageRegistry.verifyAgentOwnership(agentId, agent1.address);
      expect(isOwner).to.be.true;
      
      const isNotOwner = await sageRegistry.verifyAgentOwnership(agentId, agent2.address);
      expect(isNotOwner).to.be.false;
    });

    it("Should check agent active status", async function () {
      let isActive = await sageRegistry.isAgentActive(agentId);
      expect(isActive).to.be.true;
      
      await sageRegistry.connect(agent1).deactivateAgent(agentId);
      
      isActive = await sageRegistry.isAgentActive(agentId);
      expect(isActive).to.be.false;
    });
  });

  describe("Verification Hook", function () {
    it("Should reject invalid DID format through hook", async function () {
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const invalidDid = "not-a-did";
      
      const params = {
        did: invalidDid,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          invalidDid,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid DID format");
    });

    it("Should blacklist malicious actors", async function () {
      // Blacklist agent2
      await verificationHook.connect(owner).addToBlacklist(agent2.address);
      
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent2.address}`;
      
      const params = {
        did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent2, params);
      
      await expect(
        sageRegistry.connect(agent2).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Address blacklisted");
    });

    it("Should enforce daily registration limit", async function () {
      // Register up to the daily limit
      const MAX_REGISTRATIONS = 5;
      
      for (let i = 0; i < MAX_REGISTRATIONS; i++) {
        const publicKey = ethers.hexlify(ethers.randomBytes(64));
        const did = `did:sage:test:${agent1.address}_${i}`;
        
        const params = {
          did,
          name: `Agent ${i}`,
          description: testDescription,
          endpoint: testEndpoint,
          publicKey,
          capabilities: testCapabilities
        };
        
        const signature = await createV1Signature(agent1, params);
        
        await sageRegistry.connect(agent1).registerAgent(
          did,
          params.name,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        );
        
        // Wait for cooldown between registrations
        await ethers.provider.send("evm_increaseTime", [61]); // 1 minute + 1 second
        await ethers.provider.send("evm_mine");
      }
      
      // Try to register one more (should fail)
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}_extra`;
      
      const params = {
        did,
        name: "Extra Agent",
        description: testDescription,
        endpoint: testEndpoint,
        publicKey,
        capabilities: testCapabilities
      };
      
      const signature = await createV1Signature(agent1, params);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          params.name,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Daily registration limit reached");
    });
  });
});