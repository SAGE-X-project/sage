const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("SageRegistryV2 Integration Tests", function () {
  let sageRegistry;
  let verificationHook;
  let owner;
  let agent1;
  let agent2;

  // Helper to create registration signature
  async function createRegistrationSignature(signer, publicKey, registry) {
    const contractAddress = await registry.getAddress();
    const chainId = (await ethers.provider.getNetwork()).chainId;
    const keyHash = ethers.keccak256(publicKey);
    
    const packedData = ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      [
        "SAGE Key Registration:",
        chainId,
        contractAddress,
        signer.address,
        keyHash
      ]
    );
    
    const challenge = ethers.keccak256(packedData);
    return await signer.signMessage(ethers.getBytes(challenge));
  }

  // Helper to create valid public key
  function createValidPublicKey() {
    return "0x04" + ethers.hexlify(ethers.randomBytes(64)).slice(2);
  }

  beforeEach(async function () {
    [owner, agent1, agent2] = await ethers.getSigners();

    // Deploy contracts
    const SageRegistryV2 = await ethers.getContractFactory("SageRegistryV2");
    sageRegistry = await SageRegistryV2.deploy();
    await sageRegistry.waitForDeployment();

    const SageVerificationHook = await ethers.getContractFactory("SageVerificationHook");
    verificationHook = await SageVerificationHook.deploy();
    await verificationHook.waitForDeployment();

    // Configure hooks
    await sageRegistry.setBeforeRegisterHook(await verificationHook.getAddress());
  });

  describe("End-to-End Agent Lifecycle", function () {
    it("Should complete full agent lifecycle: register → update → revoke", async function () {
      const publicKey = createValidPublicKey();
      const did = `did:sage:test:${agent1.address}`;
      
      // 1. Register agent
      console.log("      1️⃣  Registering agent...");
      const registerSig = await createRegistrationSignature(agent1, publicKey, sageRegistry);
      
      const registerTx = await sageRegistry.connect(agent1).registerAgent(
        did,
        "Test Agent",
        "AI Assistant",
        "https://agent.example.com",
        publicKey,
        JSON.stringify(["chat", "code"]),
        registerSig
      );
      
      const receipt = await registerTx.wait();
      // Get agent ID from the event properly
      const agentRegisteredEvent = receipt.logs.find(log => {
        try {
          const parsed = sageRegistry.interface.parseLog(log);
          return parsed.name === 'AgentRegistered';
        } catch {
          return false;
        }
      });
      const agentId = agentRegisteredEvent.args[0];
      console.log("      ✅ Agent registered with ID:", agentId);
      
      // Verify registration
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.name).to.equal("Test Agent");
      expect(agent.active).to.be.true;
      
      // 2. Update agent
      console.log("      2️⃣  Updating agent...");
      // Note: After registration, nonce is incremented to 1
      const updateMessage = ethers.keccak256(
        ethers.solidityPacked(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, "Updated Agent", "Enhanced AI", "https://new.example.com", 
           JSON.stringify(["chat", "code", "analyze"]), agent1.address, 1]
        )
      );
      
      const updateSig = await agent1.signMessage(ethers.getBytes(updateMessage));
      
      await sageRegistry.connect(agent1).updateAgent(
        agentId,
        "Updated Agent",
        "Enhanced AI",
        "https://new.example.com",
        JSON.stringify(["chat", "code", "analyze"]),
        updateSig
      );
      console.log("      ✅ Agent updated");
      
      // Verify update
      const updatedAgent = await sageRegistry.getAgent(agentId);
      expect(updatedAgent.name).to.equal("Updated Agent");
      
      // 3. Revoke key
      console.log("      3️⃣  Revoking key...");
      await sageRegistry.connect(agent1).revokeKey(publicKey);
      console.log("      ✅ Key revoked");
      
      // Verify revocation
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.false;
      
      // Agent should be deactivated
      const revokedAgent = await sageRegistry.getAgent(agentId);
      expect(revokedAgent.active).to.be.false;
      console.log("      ✅ Agent automatically deactivated");
    });

    it("Should handle multiple agents with different keys", async function () {
      const key1 = createValidPublicKey();
      const key2 = createValidPublicKey();
      
      // Register first agent
      const did1 = `did:sage:test:${agent1.address}_1`;
      const sig1 = await createRegistrationSignature(agent1, key1, sageRegistry);
      
      await sageRegistry.connect(agent1).registerAgent(
        did1, "Agent 1", "Description 1", "https://agent1.com",
        key1, JSON.stringify(["chat"]), sig1
      );
      
      // Register second agent with different key
      const did2 = `did:sage:test:${agent1.address}_2`;
      const sig2 = await createRegistrationSignature(agent1, key2, sageRegistry);
      
      await sageRegistry.connect(agent1).registerAgent(
        did2, "Agent 2", "Description 2", "https://agent2.com",
        key2, JSON.stringify(["code"]), sig2
      );
      
      // Verify both agents exist
      const agents = await sageRegistry.getAgentsByOwner(agent1.address);
      expect(agents.length).to.equal(2);
      
      // Revoke first key - but we need to make sure key1 is actually owned by agent1
      // The issue is that addressToKeyHash mapping only stores one key per address
      // So we can only revoke the last registered key
      // Let's check which key is currently mapped to agent1
      const canRevokeKey1 = await sageRegistry.isKeyValid(key1);
      const canRevokeKey2 = await sageRegistry.isKeyValid(key2);
      
      // Since addressToKeyHash maps only the last registered key to an address,
      // revoking key2 will only affect the second agent
      await sageRegistry.connect(agent1).revokeKey(key2);
      
      // First agent should still be active, second should be inactive
      const agent1Data = await sageRegistry.getAgent(agents[0]);
      const agent2Data = await sageRegistry.getAgent(agents[1]);
      
      // First agent uses key1 and should still be active
      expect(agent1Data.active).to.be.true;
      // Second agent uses key2 which was revoked, so should be inactive
      expect(agent2Data.active).to.be.false;
    });
  });

  describe("Security Scenarios", function () {
    it("Should prevent registration with malicious zero key", async function () {
      const zeroKey = "0x04" + "00".repeat(64);
      const did = `did:sage:test:${agent1.address}`;
      const sig = await createRegistrationSignature(agent1, zeroKey, sageRegistry);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did, "Malicious", "Bad", "https://bad.com",
          zeroKey, "{}", sig
        )
      ).to.be.revertedWith("Invalid zero key");
    });

    it("Should prevent key reuse after revocation", async function () {
      const publicKey = createValidPublicKey();
      const did1 = `did:sage:test:${agent1.address}_1`;
      
      // Register and revoke
      const sig1 = await createRegistrationSignature(agent1, publicKey, sageRegistry);
      await sageRegistry.connect(agent1).registerAgent(
        did1, "Agent", "Desc", "https://agent.com",
        publicKey, "{}", sig1
      );
      
      await sageRegistry.connect(agent1).revokeKey(publicKey);
      
      // Try to register with revoked key
      const did2 = `did:sage:test:${agent1.address}_2`;
      const sig2 = await createRegistrationSignature(agent1, publicKey, sageRegistry);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did2, "New Agent", "Desc", "https://new.com",
          publicKey, "{}", sig2
        )
      ).to.be.revertedWith("Key has been revoked");
    });

    it("Should enforce proper access control", async function () {
      const publicKey = createValidPublicKey();
      const did = `did:sage:test:${agent1.address}`;
      
      // Agent1 registers
      const sig = await createRegistrationSignature(agent1, publicKey, sageRegistry);
      await sageRegistry.connect(agent1).registerAgent(
        did, "Agent", "Desc", "https://agent.com",
        publicKey, "{}", sig
      );
      
      // Agent2 tries to revoke agent1's key
      await expect(
        sageRegistry.connect(agent2).revokeKey(publicKey)
      ).to.be.revertedWith("Not key owner");
    });
  });

  describe("Hook Integration", function () {
    it("Should work seamlessly with verification hooks", async function () {
      const publicKey = createValidPublicKey();
      const validDid = `did:sage:test:${agent1.address}`;
      const invalidDid = "not-a-did";
      
      // Valid DID should pass
      const sig1 = await createRegistrationSignature(agent1, publicKey, sageRegistry);
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          validDid, "Agent", "Desc", "https://agent.com",
          publicKey, "{}", sig1
        )
      ).to.emit(sageRegistry, "BeforeRegisterHook");
      
      // Invalid DID should fail at hook
      const key2 = createValidPublicKey();
      const sig2 = await createRegistrationSignature(agent1, key2, sageRegistry);
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          invalidDid, "Agent", "Desc", "https://agent.com",
          key2, "{}", sig2
        )
      ).to.be.revertedWith("Invalid DID format");
    });
  });

  describe("Gas Optimization Check", function () {
    it("Should maintain reasonable gas costs for batch operations", async function () {
      const results = [];
      
      for (let i = 0; i < 3; i++) {
        const key = createValidPublicKey();
        const did = `did:sage:test:${agent1.address}_${i}`;
        const sig = await createRegistrationSignature(agent1, key, sageRegistry);
        
        const tx = await sageRegistry.connect(agent1).registerAgent(
          did, `Agent ${i}`, `Desc ${i}`, `https://agent${i}.com`,
          key, JSON.stringify([`capability${i}`]), sig
        );
        
        const receipt = await tx.wait();
        results.push(receipt.gasUsed);
      }
      
      console.log("      Gas usage for 3 registrations:");
      results.forEach((gas, i) => {
        console.log(`        Agent ${i}: ${gas.toString()} gas`);
      });
      
      // Average should be under 650K
      const avgGas = results.reduce((a, b) => a + b, 0n) / 3n;
      expect(avgGas).to.be.lt(650000n);
    });
  });
});