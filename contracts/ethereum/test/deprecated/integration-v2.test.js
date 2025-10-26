const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("SageRegistryV2 Integration Tests", function () {
  let sageRegistry;
  let verificationHook;
  let owner;
  let agent1;
  let agent2;

  // Helper to create registration signature
  async function createRegistrationSignature(wallet, publicKey) {
    const contractAddress = await sageRegistry.getAddress();
    const chainId = (await ethers.provider.getNetwork()).chainId;
    const keyHash = ethers.keccak256(publicKey);

    const packedData = ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      [
        "SAGE Key Registration:",
        chainId,
        contractAddress,
        wallet.address,
        keyHash
      ]
    );

    const challenge = ethers.keccak256(packedData);
    return await wallet.signMessage(ethers.getBytes(challenge));
  }

  // Helper to create test wallet with actual public key
  function createTestWallet() {
    const wallet = ethers.Wallet.createRandom();
    const publicKey = wallet.signingKey.publicKey;
    return { wallet, publicKey };
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
      // Create wallet with actual private key
      const { wallet: testWallet, publicKey } = createTestWallet();
      const connectedWallet = testWallet.connect(ethers.provider);
      const did = `did:sage:test:${testWallet.address}`;

      // Fund the wallet for gas
      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // 1. Register agent
      console.log("      1️⃣  Registering agent...");
      const registerSig = await createRegistrationSignature(connectedWallet, publicKey);

      const registerTx = await sageRegistry.connect(connectedWallet).registerAgent(
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
      console.log("       Agent registered with ID:", agentId);

      // Verify registration
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.name).to.equal("Test Agent");
      expect(agent.active).to.be.true;

      // 2. Update agent
      console.log("      2️⃣  Updating agent...");
      const updateMessage = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, "Updated Agent", "Enhanced AI", "https://new.example.com",
           JSON.stringify(["chat", "code", "analyze"]), testWallet.address, 1]
        )
      );

      const updateSig = await connectedWallet.signMessage(ethers.getBytes(updateMessage));

      await sageRegistry.connect(connectedWallet).updateAgent(
        agentId,
        "Updated Agent",
        "Enhanced AI",
        "https://new.example.com",
        JSON.stringify(["chat", "code", "analyze"]),
        updateSig
      );
      console.log("       Agent updated");

      // Verify update
      const updatedAgent = await sageRegistry.getAgent(agentId);
      expect(updatedAgent.name).to.equal("Updated Agent");

      // 3. Revoke key
      console.log("      3️⃣  Revoking key...");
      await sageRegistry.connect(connectedWallet).revokeKey(publicKey);
      console.log("       Key revoked");

      // Verify revocation
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.false;

      // Agent should be deactivated
      const revokedAgent = await sageRegistry.getAgent(agentId);
      expect(revokedAgent.active).to.be.false;
      console.log("       Agent automatically deactivated");
    });

    it("Should handle multiple agents with different keys", async function () {
      // Create two different wallets with different keys
      const { wallet: wallet1, publicKey: key1 } = createTestWallet();
      const { wallet: wallet2, publicKey: key2 } = createTestWallet();
      const connected1 = wallet1.connect(ethers.provider);
      const connected2 = wallet2.connect(ethers.provider);

      // Fund both wallets
      await owner.sendTransaction({
        to: wallet1.address,
        value: ethers.parseEther("1.0")
      });
      await owner.sendTransaction({
        to: wallet2.address,
        value: ethers.parseEther("1.0")
      });

      // Register first agent
      const did1 = `did:sage:test:${wallet1.address}`;
      const sig1 = await createRegistrationSignature(connected1, key1);

      const tx1 = await sageRegistry.connect(connected1).registerAgent(
        did1, "Agent 1", "Description 1", "https://agent1.com",
        key1, JSON.stringify(["chat"]), sig1
      );
      const receipt1 = await tx1.wait();
      const event1 = receipt1.logs.find(log => {
        try {
          return sageRegistry.interface.parseLog(log).name === 'AgentRegistered';
        } catch { return false; }
      });
      const agentId1 = event1.args[0];

      // Register second agent with different wallet
      const did2 = `did:sage:test:${wallet2.address}`;
      const sig2 = await createRegistrationSignature(connected2, key2);

      const tx2 = await sageRegistry.connect(connected2).registerAgent(
        did2, "Agent 2", "Description 2", "https://agent2.com",
        key2, JSON.stringify(["code"]), sig2
      );
      const receipt2 = await tx2.wait();
      const event2 = receipt2.logs.find(log => {
        try {
          return sageRegistry.interface.parseLog(log).name === 'AgentRegistered';
        } catch { return false; }
      });
      const agentId2 = event2.args[0];

      // Verify both agents exist
      const agents1 = await sageRegistry.getAgentsByOwner(wallet1.address);
      const agents2 = await sageRegistry.getAgentsByOwner(wallet2.address);
      expect(agents1.length).to.equal(1);
      expect(agents2.length).to.equal(1);

      // Revoke second key
      await sageRegistry.connect(connected2).revokeKey(key2);

      // First agent should still be active
      const agent1Data = await sageRegistry.getAgent(agentId1);
      expect(agent1Data.active).to.be.true;

      // Second agent should be inactive
      const agent2Data = await sageRegistry.getAgent(agentId2);
      expect(agent2Data.active).to.be.false;
    });
  });

  describe("Security Scenarios", function () {
    it("Should prevent registration with malicious zero key", async function () {
      const zeroKey = "0x04" + "00".repeat(64);
      const did = `did:sage:test:${agent1.address}`;
      const sig = await createRegistrationSignature(agent1, zeroKey);

      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did, "Malicious", "Bad", "https://bad.com",
          zeroKey, "{}", sig
        )
      ).to.be.revertedWith("Invalid zero key");
    });

    it("Should prevent key reuse after revocation", async function () {
      const { wallet: testWallet, publicKey } = createTestWallet();
      const connected = testWallet.connect(ethers.provider);
      const did1 = `did:sage:test:${testWallet.address}_1`;

      // Fund wallet
      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // Register and revoke
      const sig1 = await createRegistrationSignature(connected, publicKey);
      await sageRegistry.connect(connected).registerAgent(
        did1, "Agent", "Desc", "https://agent.com",
        publicKey, "{}", sig1
      );

      await sageRegistry.connect(connected).revokeKey(publicKey);

      // Try to register with revoked key
      const did2 = `did:sage:test:${testWallet.address}_2`;
      const sig2 = await createRegistrationSignature(connected, publicKey);

      await expect(
        sageRegistry.connect(connected).registerAgent(
          did2, "New Agent", "Desc", "https://new.com",
          publicKey, "{}", sig2
        )
      ).to.be.revertedWith("Key has been revoked");
    });

    it("Should enforce proper access control", async function () {
      const { wallet: testWallet, publicKey } = createTestWallet();
      const connected = testWallet.connect(ethers.provider);
      const did = `did:sage:test:${testWallet.address}`;

      // Fund wallet
      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // testWallet registers
      const sig = await createRegistrationSignature(connected, publicKey);
      await sageRegistry.connect(connected).registerAgent(
        did, "Agent", "Desc", "https://agent.com",
        publicKey, "{}", sig
      );

      // agent2 (different address) tries to revoke testWallet's key
      await expect(
        sageRegistry.connect(agent2).revokeKey(publicKey)
      ).to.be.revertedWith("Not key owner");
    });
  });

  describe("Hook Integration", function () {
    it("Should work seamlessly with verification hooks", async function () {
      const { wallet: testWallet, publicKey } = createTestWallet();
      const connected = testWallet.connect(ethers.provider);
      const validDid = `did:sage:test:${testWallet.address}`;
      const invalidDid = "not-a-did";

      // Fund wallet
      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // Valid DID should pass
      const sig1 = await createRegistrationSignature(connected, publicKey);
      await expect(
        sageRegistry.connect(connected).registerAgent(
          validDid, "Agent", "Desc", "https://agent.com",
          publicKey, "{}", sig1
        )
      ).to.emit(sageRegistry, "BeforeRegisterHook");

      // Invalid DID should fail at hook
      const { wallet: testWallet2, publicKey: key2 } = createTestWallet();
      const connected2 = testWallet2.connect(ethers.provider);

      await owner.sendTransaction({
        to: testWallet2.address,
        value: ethers.parseEther("1.0")
      });

      const sig2 = await createRegistrationSignature(connected2, key2);
      await expect(
        sageRegistry.connect(connected2).registerAgent(
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
        const { wallet, publicKey } = createTestWallet();
        const connected = wallet.connect(ethers.provider);
        const did = `did:sage:test:${wallet.address}`;

        await owner.sendTransaction({
          to: wallet.address,
          value: ethers.parseEther("1.0")
        });

        const sig = await createRegistrationSignature(connected, publicKey);

        const tx = await sageRegistry.connect(connected).registerAgent(
          did, `Agent ${i}`, `Desc ${i}`, `https://agent${i}.com`,
          publicKey, JSON.stringify([`capability${i}`]), sig
        );

        const receipt = await tx.wait();
        results.push(receipt.gasUsed);
      }

      console.log("      Gas usage for 3 registrations:");
      results.forEach((gas, i) => {
        console.log(`        Agent ${i}: ${gas.toString()} gas`);
      });

      // Average should be under 700K (Phase 3 security features add overhead)
      const avgGas = results.reduce((a, b) => a + b, 0n) / 3n;
      expect(avgGas).to.be.lt(700000n);
    });
  });
});
