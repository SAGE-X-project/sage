const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("SageRegistry - Registration Status", function () {
  let registry;
  let owner;
  let user1;
  let user2;

  beforeEach(async function () {
    [owner, user1, user2] = await ethers.getSigners();

    const SageRegistry = await ethers.getContractFactory("SageRegistry");
    registry = await SageRegistry.deploy();
    await registry.deployed();
  });

  describe("isAgentRegistered", function () {
    it("should return false for unregistered DID", async function () {
      const result = await registry.isAgentRegistered("did:sage:test:001");
      expect(result).to.be.false;
    });

    it("should return true for registered DID", async function () {
      // Register an agent
      const agentData = {
        did: "did:sage:test:001",
        name: "Test Agent",
        description: "Test Description",
        endpoint: "https://test.example.com",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: JSON.stringify({ test: true }),
      };

      // Create signature
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
            0
          ]
        )
      );

      const signature = await user1.signMessage(ethers.utils.arrayify(messageHash));

      // Register
      await registry.connect(user1).registerAgent(
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.publicKey,
        agentData.capabilities,
        signature
      );

      // Check registration
      const result = await registry.isAgentRegistered(agentData.did);
      expect(result).to.be.true;
    });
  });

  describe("getAgentRegistrationStatus", function () {
    it("should return (false, false) for unregistered DID", async function () {
      const result = await registry.getAgentRegistrationStatus("did:sage:test:002");
      expect(result.registered).to.be.false;
      expect(result.active).to.be.false;
    });

    it("should return (true, true) for active registered DID", async function () {
      // Register an agent
      const agentData = {
        did: "did:sage:test:002",
        name: "Test Agent 2",
        description: "Test Description 2",
        endpoint: "https://test2.example.com",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: JSON.stringify({ test: true }),
      };

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
            0
          ]
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
      const agentId = event.args.agentId;

      // Check status
      const result = await registry.getAgentRegistrationStatus(agentData.did);
      expect(result.registered).to.be.true;
      expect(result.active).to.be.true;
    });

    it("should return (true, false) for deactivated DID", async function () {
      // Register an agent
      const agentData = {
        did: "did:sage:test:003",
        name: "Test Agent 3",
        description: "Test Description 3",
        endpoint: "https://test3.example.com",
        publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
        capabilities: JSON.stringify({ test: true }),
      };

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
            0
          ]
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
      const agentId = event.args.agentId;

      // Deactivate
      await registry.connect(user1).deactivateAgent(agentId);

      // Check status
      const result = await registry.getAgentRegistrationStatus(agentData.did);
      expect(result.registered).to.be.true;
      expect(result.active).to.be.false;
    });
  });

  describe("Edge cases", function () {
    it("should handle empty DID", async function () {
      await expect(registry.isAgentRegistered(""))
        .to.be.revertedWith("DID required");
    });

    it("should handle multiple registrations correctly", async function () {
      const dids = ["did:sage:test:multi1", "did:sage:test:multi2", "did:sage:test:multi3"];
      
      // Register multiple agents
      for (let i = 0; i < dids.length; i++) {
        const agentData = {
          did: dids[i],
          name: `Multi Agent ${i}`,
          description: `Multi Description ${i}`,
          endpoint: `https://multi${i}.example.com`,
          publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)),
          capabilities: JSON.stringify({ index: i }),
        };

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
              0
            ]
          )
        );

        const signature = await user1.signMessage(ethers.utils.arrayify(messageHash));

        await registry.connect(user1).registerAgent(
          agentData.did,
          agentData.name,
          agentData.description,
          agentData.endpoint,
          agentData.publicKey,
          agentData.capabilities,
          signature
        );
      }

      // Check all are registered
      for (const did of dids) {
        expect(await registry.isAgentRegistered(did)).to.be.true;
        const status = await registry.getAgentRegistrationStatus(did);
        expect(status.registered).to.be.true;
        expect(status.active).to.be.true;
      }

      // Check unregistered DID still returns false
      expect(await registry.isAgentRegistered("did:sage:test:nonexistent")).to.be.false;
    });
  });
});