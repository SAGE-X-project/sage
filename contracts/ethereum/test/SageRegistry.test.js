const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

describe("SageRegistry", function () {
  let sageRegistry;
  let verificationHook;
  let owner;
  let agent1;
  let agent2;
  let attacker;

  // Test data
  const testDID = "did:sage:test:0x1234567890123456789012345678901234567890";
  const testName = "Test AI Agent";
  const testDescription = "A test AI agent for demonstration";
  const testEndpoint = "https://agent.example.com";
  const testCapabilities = JSON.stringify(["chat", "code", "analysis"]);

  beforeEach(async function () {
    // Get signers
    [owner, agent1, agent2, attacker] = await ethers.getSigners();

    // Deploy SageRegistry
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
      expect(await sageRegistry.owner()).to.equal(owner.address);
    });

    it("Should have the verification hook set", async function () {
      expect(await sageRegistry.beforeRegisterHook()).to.equal(await verificationHook.getAddress());
    });
  });

  describe("Agent Registration", function () {
    it("Should register a new agent successfully", async function () {
      // Use agent1's address as the public key owner
      const publicKey = ethers.hexlify(ethers.randomBytes(64)); // 64 bytes for uncompressed secp256k1
      const did = `did:sage:test:${agent1.address}`;
      
      // Create message hash for signature (using abi.encode to match contract)
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );
      
      // Sign with agent1's account
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      // Register agent
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      // Check event emission
      await expect(tx).to.emit(sageRegistry, "AgentRegistered");
      
      // Verify agent was registered
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
      
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [invalidDID, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
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
      
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );

      const signature = await agent1.signMessage(ethers.getBytes(messageHash));

      // First registration should succeed
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );

      // Second registration with same DID should fail (from different owner)
      const messageHash2 = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent2.address, 0]
        )
      );
      
      const signature2 = await agent2.signMessage(ethers.getBytes(messageHash2));
      
      await expect(
        sageRegistry.connect(agent2).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature2
        )
      ).to.be.revertedWith("DID already registered");
    });

    it("Should enforce registration cooldown", async function () {
      const publicKey1 = ethers.hexlify(ethers.randomBytes(64));
      const publicKey2 = ethers.hexlify(ethers.randomBytes(64));
      
      // First registration
      const did1 = `did:sage:test:${agent1.address}_1`;
      const messageHash1 = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did1, testName, testDescription, testEndpoint, publicKey1, testCapabilities, agent1.address, 0]
        )
      );
      const signature1 = await agent1.signMessage(ethers.getBytes(messageHash1));

      await sageRegistry.connect(agent1).registerAgent(
        did1,
        testName,
        testDescription,
        testEndpoint,
        publicKey1,
        testCapabilities,
        signature1
      );

      // Try second registration immediately
      const did2 = `did:sage:test:${agent1.address}_2`;
      const messageHash2 = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did2, testName, testDescription, testEndpoint, publicKey2, testCapabilities, agent1.address, 0]
        )
      );
      const signature2 = await agent1.signMessage(ethers.getBytes(messageHash2));
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did2,
          testName,
          testDescription,
          testEndpoint,
          publicKey2,
          testCapabilities,
          signature2
        )
      ).to.be.revertedWith("Registration cooldown active");
    });
  });

  describe("Agent Management", function () {
    let agentId;
    let publicKey;
    
    beforeEach(async function () {
      // Register an agent for testing
      publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}`;

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
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

    it("Should update agent metadata", async function () {
      const newName = "Updated Agent Name";
      const newDescription = "Updated description";
      const newEndpoint = "https://new-endpoint.example.com";
      const newCapabilities = JSON.stringify(["chat", "code", "analysis", "vision"]);
      
      // Get current nonce (should be 1 after registration)
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, newName, newDescription, newEndpoint, newCapabilities, agent1.address, 1]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      await sageRegistry.connect(agent1).updateAgent(
        agentId,
        newName,
        newDescription,
        newEndpoint,
        newCapabilities,
        signature
      );
      
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.name).to.equal(newName);
      expect(agent.description).to.equal(newDescription);
      expect(agent.endpoint).to.equal(newEndpoint);
      expect(agent.capabilities).to.equal(newCapabilities);
    });

    it("Should only allow owner to update agent", async function () {
      const newName = "Hacked Name";
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, newName, testDescription, testEndpoint, testCapabilities, attacker.address, 1]
        )
      );
      
      const signature = await attacker.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        sageRegistry.connect(attacker).updateAgent(
          agentId,
          newName,
          testDescription,
          testEndpoint,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Not agent owner");
    });

    it("Should deactivate agent", async function () {
      await sageRegistry.connect(agent1).deactivateAgent(agentId);
      
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.active).to.be.false;
      
      const isActive = await sageRegistry.isAgentActive(agentId);
      expect(isActive).to.be.false;
    });

    it("Should only allow owner to deactivate agent", async function () {
      await expect(
        sageRegistry.connect(attacker).deactivateAgent(agentId)
      ).to.be.revertedWith("Not agent owner");
    });
  });

  describe("Query Functions", function () {
    let did;
    let agentId;
    
    beforeEach(async function () {
      // Register an agent
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      did = `did:sage:test:${agent1.address}`;

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
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

    it("Should verify agent ownership", async function () {
      const isOwner = await sageRegistry.verifyAgentOwnership(agentId, agent1.address);
      expect(isOwner).to.be.true;
      
      const isNotOwner = await sageRegistry.verifyAgentOwnership(agentId, attacker.address);
      expect(isNotOwner).to.be.false;
    });

    it("Should get all agents by owner", async function () {
      // Wait for cooldown
      await time.increase(61); // 61 seconds to pass 1 minute cooldown
      
      // Register another agent
      const publicKey2 = ethers.hexlify(ethers.randomBytes(64));
      const did2 = `did:sage:test:${agent1.address}_second`;

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did2, "Second Agent", testDescription, testEndpoint, publicKey2, testCapabilities, agent1.address, 0]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      await sageRegistry.connect(agent1).registerAgent(
        did2,
        "Second Agent",
        testDescription,
        testEndpoint,
        publicKey2,
        testCapabilities,
        signature
      );
      
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      expect(agentIds.length).to.equal(2);
    });
  });

  describe("Verification Hook", function () {
    it("Should blacklist malicious actors", async function () {
      // Blacklist attacker
      await verificationHook.addToBlacklist(attacker.address);
      
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${attacker.address}`;

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, testName, testDescription, testEndpoint, publicKey, testCapabilities, attacker.address, 0]
        )
      );
      
      const signature = await attacker.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        sageRegistry.connect(attacker).registerAgent(
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
      // Register multiple agents quickly (after cooldown periods)
      for (let i = 0; i < 5; i++) {
        const publicKey = ethers.hexlify(ethers.randomBytes(64));
        const did = `did:sage:test:${agent1.address}_${i}`;
        
        const messageHash = ethers.keccak256(
          ethers.AbiCoder.defaultAbiCoder().encode(
            ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
            [did, `Agent ${i}`, testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
          )
        );
        
        const signature = await agent1.signMessage(ethers.getBytes(messageHash));
        
        await sageRegistry.connect(agent1).registerAgent(
          did,
          `Agent ${i}`,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        );
        
        // Wait for cooldown between registrations
        if (i < 4) {
          await time.increase(61);
        }
      }
      
      // Try to register 6th agent (should fail)
      await time.increase(61);
      
      const publicKey = ethers.hexlify(ethers.randomBytes(64));
      const did = `did:sage:test:${agent1.address}_6`;

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [did, "Agent 6", testDescription, testEndpoint, publicKey, testCapabilities, agent1.address, 0]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          "Agent 6",
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Daily registration limit reached");
    });
  });

  describe("Access Control", function () {
    it("Should only allow owner to set hooks", async function () {
      const newHook = ethers.Wallet.createRandom().address;
      
      await expect(
        sageRegistry.connect(attacker).setBeforeRegisterHook(newHook)
      ).to.be.revertedWith("Only owner");
      
      await expect(
        sageRegistry.connect(attacker).setAfterRegisterHook(newHook)
      ).to.be.revertedWith("Only owner");
    });

    it("Should allow owner to change hooks", async function () {
      const newHook = ethers.Wallet.createRandom().address;
      
      await sageRegistry.setBeforeRegisterHook(newHook);
      expect(await sageRegistry.beforeRegisterHook()).to.equal(newHook);
      
      await sageRegistry.setAfterRegisterHook(newHook);
      expect(await sageRegistry.afterRegisterHook()).to.equal(newHook);
    });
  });
});