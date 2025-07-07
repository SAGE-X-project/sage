const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("SAGE Registry Integration Tests", function () {
  let registry;
  let verificationHook;
  let owner;
  let user1;
  let user2;

  beforeEach(async function () {
    [owner, user1, user2] = await ethers.getSigners();

    // Deploy contracts
    const SageRegistry = await ethers.getContractFactory("SageRegistry");
    registry = await SageRegistry.deploy();
    await registry.deployed();

    const SageVerificationHook = await ethers.getContractFactory("SageVerificationHook");
    verificationHook = await SageVerificationHook.deploy();
    await verificationHook.deployed();

    // Set verification hook
    await registry.setBeforeRegisterHook(verificationHook.address);
  });

  describe("Agent Registration Flow", function () {
    it("should successfully register an agent with valid signature", async function () {
      const agentData = {
        did: "did:sage:test001",
        name: "Test Agent",
        description: "A test AI agent",
        endpoint: "https://test.agent.ai",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: JSON.stringify({ skills: ["test"] }),
      };

      // Create message hash
      const nonce = 0;
      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [
            agentData.did,
            agentData.name,
            agentData.description,
            agentData.endpoint,
            agentData.publicKey,
            agentData.capabilities,
            user1.address,
            nonce
          ]
        )
      );

      // Sign message
      const signature = await user1.signMessage(ethers.utils.arrayify(messageHash));

      // Register agent
      const tx = await registry.connect(user1).registerAgent(
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.publicKey,
        agentData.capabilities,
        signature
      );

      const receipt = await tx.wait();
      const event = receipt.events.find(e => e.event === "AgentRegistered");
      
      expect(event).to.not.be.undefined;
      expect(event.args.owner).to.equal(user1.address);
      expect(event.args.did).to.equal(agentData.did);

      // Verify agent data
      const agentId = event.args.agentId;
      const agent = await registry.getAgent(agentId);
      
      expect(agent.name).to.equal(agentData.name);
      expect(agent.active).to.be.true;
      expect(agent.owner).to.equal(user1.address);
    });

    it("should reject registration with invalid signature", async function () {
      const agentData = {
        did: "did:sage:test002",
        name: "Test Agent 2",
        description: "Another test agent",
        endpoint: "https://test2.agent.ai",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: "{}",
      };

      // Create message with wrong data
      const wrongMessage = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["string"], 
          ["wrong message"]
        )
      );

      const signature = await user1.signMessage(ethers.utils.arrayify(wrongMessage));

      await expect(
        registry.connect(user1).registerAgent(
          agentData.did,
          agentData.name,
          agentData.description,
          agentData.endpoint,
          agentData.publicKey,
          agentData.capabilities,
          signature
        )
      ).to.be.revertedWith("Invalid signature");
    });

    it("should enforce rate limiting through hook", async function () {
      // Register maximum allowed agents
      for (let i = 0; i < 5; i++) {
        const agentData = {
          did: `did:sage:spam${i}`,
          name: `Spam Agent ${i}`,
          description: "Spam test",
          endpoint: "https://spam.ai",
          publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
          capabilities: "{}",
        };

        const messageHash = ethers.utils.keccak256(
          ethers.utils.defaultAbiCoder.encode(
            ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
            [agentData.did, agentData.name, agentData.description, agentData.endpoint, 
             agentData.publicKey, agentData.capabilities, user2.address, i]
          )
        );

        const signature = await user2.signMessage(ethers.utils.arrayify(messageHash));

        await registry.connect(user2).registerAgent(
          agentData.did,
          agentData.name,
          agentData.description,
          agentData.endpoint,
          agentData.publicKey,
          agentData.capabilities,
          signature
        );
      }

      // 6th registration should fail
      const spamData = {
        did: "did:sage:spam6",
        name: "Spam Agent 6",
        description: "Should fail",
        endpoint: "https://spam.ai",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: "{}",
      };

      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [spamData.did, spamData.name, spamData.description, spamData.endpoint,
           spamData.publicKey, spamData.capabilities, user2.address, 5]
        )
      );

      const signature = await user2.signMessage(ethers.utils.arrayify(messageHash));

      await expect(
        registry.connect(user2).registerAgent(
          spamData.did,
          spamData.name,
          spamData.description,
          spamData.endpoint,
          spamData.publicKey,
          spamData.capabilities,
          signature
        )
      ).to.be.revertedWith("Daily registration limit reached");
    });

    it("should allow blacklisting malicious actors", async function () {
      // Blacklist user2
      await verificationHook.addToBlacklist(user2.address);

      const agentData = {
        did: "did:sage:blacklisted",
        name: "Blacklisted Agent",
        description: "Should not register",
        endpoint: "https://bad.ai",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: "{}",
      };

      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [agentData.did, agentData.name, agentData.description, agentData.endpoint,
           agentData.publicKey, agentData.capabilities, user2.address, 0]
        )
      );

      const signature = await user2.signMessage(ethers.utils.arrayify(messageHash));

      await expect(
        registry.connect(user2).registerAgent(
          agentData.did,
          agentData.name,
          agentData.description,
          agentData.endpoint,
          agentData.publicKey,
          agentData.capabilities,
          signature
        )
      ).to.be.revertedWith("Address blacklisted");
    });
  });

  describe("Agent Management", function () {
    let agentId;

    beforeEach(async function () {
      // Register an agent first
      const agentData = {
        did: "did:sage:update-test",
        name: "Update Test Agent",
        description: "Will be updated",
        endpoint: "https://update.ai",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: JSON.stringify({ version: "1.0" }),
      };

      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
          [agentData.did, agentData.name, agentData.description, agentData.endpoint,
           agentData.publicKey, agentData.capabilities, user1.address, 0]
        )
      );

      const signature = await user1.signMessage(ethers.utils.arrayify(messageHash));

      const tx = await registry.connect(user1).registerAgent(
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.publicKey,
        agentData.capabilities,
        signature
      );

      const receipt = await tx.wait();
      const event = receipt.events.find(e => e.event === "AgentRegistered");
      agentId = event.args.agentId;
    });

    it("should allow owner to update agent", async function () {
      const updatedData = {
        name: "Updated Agent Name",
        description: "Updated description",
        endpoint: "https://updated.ai",
        capabilities: JSON.stringify({ version: "2.0" }),
      };

      // Get current agent to access public key
      const agent = await registry.getAgent(agentId);

      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, updatedData.name, updatedData.description, updatedData.endpoint,
           updatedData.capabilities, user1.address, 1] // nonce incremented
        )
      );

      const signature = await user1.signMessage(ethers.utils.arrayify(messageHash));

      await registry.connect(user1).updateAgent(
        agentId,
        updatedData.name,
        updatedData.description,
        updatedData.endpoint,
        updatedData.capabilities,
        signature
      );

      const updatedAgent = await registry.getAgent(agentId);
      expect(updatedAgent.name).to.equal(updatedData.name);
      expect(updatedAgent.description).to.equal(updatedData.description);
    });

    it("should allow owner to deactivate agent", async function () {
      await registry.connect(user1).deactivateAgent(agentId);

      const agent = await registry.getAgent(agentId);
      expect(agent.active).to.be.false;
    });

    it("should prevent non-owner from updating", async function () {
      const messageHash = ethers.utils.keccak256(
        ethers.utils.defaultAbiCoder.encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, "Hacked", "Hacked", "https://hacked.ai", "{}", user2.address, 1]
        )
      );

      const signature = await user2.signMessage(ethers.utils.arrayify(messageHash));

      await expect(
        registry.connect(user2).updateAgent(
          agentId,
          "Hacked",
          "Hacked",
          "https://hacked.ai",
          "{}",
          signature
        )
      ).to.be.revertedWith("Not agent owner");
    });
  });
});