const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

describe("SageRegistryV2 - Enhanced Public Key Validation", function () {
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
   * Helper function to create a valid registration signature
   */
  async function createRegistrationSignature(signer, publicKey) {
    const contractAddress = await sageRegistry.getAddress();
    const chainId = (await ethers.provider.getNetwork()).chainId;
    const keyHash = ethers.keccak256(publicKey);
    
    // Create challenge message matching the contract exactly
    // The contract uses abi.encodePacked without timestamp for predictability
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
    
    // Sign the challenge
    return await signer.signMessage(ethers.getBytes(challenge));
  }

  /**
   * Helper to create valid secp256k1 public key
   */
  function createValidPublicKey(compressed = false) {
    if (compressed) {
      // Compressed format: 0x02 or 0x03 + 32 bytes
      const prefix = Math.random() > 0.5 ? "0x02" : "0x03";
      return prefix + ethers.hexlify(ethers.randomBytes(32)).slice(2);
    } else {
      // Uncompressed format: 0x04 + 64 bytes
      return "0x04" + ethers.hexlify(ethers.randomBytes(64)).slice(2);
    }
  }

  beforeEach(async function () {
    // Get signers
    [owner, agent1, agent2, attacker] = await ethers.getSigners();

    // Deploy SageRegistryV2
    const SageRegistryV2 = await ethers.getContractFactory("SageRegistryV2");
    sageRegistry = await SageRegistryV2.deploy();
    await sageRegistry.waitForDeployment();

    // Deploy SageVerificationHook
    const SageVerificationHook = await ethers.getContractFactory("SageVerificationHook");
    verificationHook = await SageVerificationHook.deploy();
    await verificationHook.waitForDeployment();

    // Set verification hook
    await sageRegistry.setBeforeRegisterHook(await verificationHook.getAddress());
  });

  describe("Deployment", function () {
    it("Should set the correct owner", async function () {
      expect(await sageRegistry.owner()).to.equal(owner.address);
    });

    it("Should have the verification hook set", async function () {
      expect(await sageRegistry.beforeRegisterHook()).to.equal(await verificationHook.getAddress());
    });
  });

  describe("Enhanced Public Key Validation", function () {
    it("Should accept valid uncompressed public key with correct format", async function () {
      const publicKey = createValidPublicKey(false); // Uncompressed
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      await expect(tx).to.emit(sageRegistry, "KeyValidated");
      await expect(tx).to.emit(sageRegistry, "AgentRegistered");
      
      // Verify key is marked as valid
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.true;
    });

    it("Should accept valid compressed public key with correct format", async function () {
      const publicKey = createValidPublicKey(true); // Compressed
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      await expect(tx).to.emit(sageRegistry, "KeyValidated");
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.true;
    });

    it("Should reject public key with invalid uncompressed format", async function () {
      // Invalid prefix (should be 0x04)
      const invalidKey = "0x05" + ethers.hexlify(ethers.randomBytes(64)).slice(2);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, invalidKey);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          invalidKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid uncompressed key format");
    });

    it("Should reject public key with invalid compressed format", async function () {
      // Invalid prefix (should be 0x02 or 0x03)
      const invalidKey = "0x01" + ethers.hexlify(ethers.randomBytes(32)).slice(2);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, invalidKey);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          invalidKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid compressed key format");
    });

    it("Should reject zero public key", async function () {
      // All zeros key - create actual zero bytes, not character '0'
      const zeroKey = "0x04" + "00".repeat(64); // 0x04 followed by 64 zero bytes (128 hex chars)
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, zeroKey);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          zeroKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid zero key");
    });

    it("Should reject registration without proper key ownership proof", async function () {
      const publicKey = createValidPublicKey(false);
      const did = `did:sage:test:${agent1.address}`;
      
      // Wrong signature (signed by different account)
      const wrongSignature = await createRegistrationSignature(agent2, publicKey);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          wrongSignature
        )
      ).to.be.revertedWith("Key ownership not proven");
    });

    it("Should reject public key with invalid length", async function () {
      const tooShort = "0x" + ethers.hexlify(ethers.randomBytes(20)).slice(2);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, tooShort);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          tooShort,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Invalid public key length");
    });

    it("Should reject Ed25519 keys (32 bytes)", async function () {
      const ed25519Key = ethers.hexlify(ethers.randomBytes(32));
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, ed25519Key);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          did,
          testName,
          testDescription,
          testEndpoint,
          ed25519Key,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Ed25519 not supported on-chain");
    });
  });

  describe("Key Revocation", function () {
    let publicKey;
    let agentId;
    
    beforeEach(async function () {
      // Register an agent first
      publicKey = createValidPublicKey(false);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
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

    it("Should allow key owner to revoke their key", async function () {
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.true;
      
      const tx = await sageRegistry.connect(agent1).revokeKey(publicKey);
      await expect(tx).to.emit(sageRegistry, "KeyRevoked");
      
      expect(await sageRegistry.isKeyValid(publicKey)).to.be.false;
      
      // Agent should be deactivated
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.active).to.be.false;
    });

    it("Should not allow non-owner to revoke key", async function () {
      await expect(
        sageRegistry.connect(agent2).revokeKey(publicKey)
      ).to.be.revertedWith("Not key owner");
    });

    it("Should not allow double revocation", async function () {
      await sageRegistry.connect(agent1).revokeKey(publicKey);
      
      await expect(
        sageRegistry.connect(agent1).revokeKey(publicKey)
      ).to.be.revertedWith("Already revoked");
    });

    it("Should prevent registration with revoked key", async function () {
      // Revoke the key
      await sageRegistry.connect(agent1).revokeKey(publicKey);
      
      // Try to register new agent with same key
      const newDid = `did:sage:test:${agent1.address}_new`;
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      await expect(
        sageRegistry.connect(agent1).registerAgent(
          newDid,
          testName,
          testDescription,
          testEndpoint,
          publicKey,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Key has been revoked");
    });

    it("Should prevent updates with revoked key", async function () {
      // Revoke the key
      await sageRegistry.connect(agent1).revokeKey(publicKey);
      
      // Try to update agent
      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, "New Name", testDescription, testEndpoint, testCapabilities, agent1.address, 1]
        )
      );
      
      const signature = await agent1.signMessage(ethers.getBytes(messageHash));
      
      await expect(
        sageRegistry.connect(agent1).updateAgent(
          agentId,
          "New Name",
          testDescription,
          testEndpoint,
          testCapabilities,
          signature
        )
      ).to.be.revertedWith("Key has been revoked");
    });
  });

  describe("Integration with Hooks", function () {
    it("Should work with verification hook and DID validation", async function () {
      const publicKey = createValidPublicKey(false);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      await expect(tx).to.emit(sageRegistry, "BeforeRegisterHook");
      await expect(tx).to.emit(sageRegistry, "AgentRegistered");
    });

    it("Should reject invalid DID format through hook", async function () {
      const publicKey = createValidPublicKey(false);
      const invalidDid = "not-a-did";
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
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
  });

  describe("Gas Usage Comparison", function () {
    it("Should measure gas for registration with enhanced validation", async function () {
      const publicKey = createValidPublicKey(false);
      const did = `did:sage:test:${agent1.address}_gas`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      const tx = await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      const receipt = await tx.wait();
      console.log(`      Gas used for enhanced registration: ${receipt.gasUsed.toString()}`);
      
      // Should be reasonable (under 700k due to enhanced validation)
      // The additional validation adds significant gas cost but improves security
      expect(receipt.gasUsed).to.be.lt(700000);
    });
  });

  describe("Backwards Compatibility", function () {
    it("Should maintain all original registry functions", async function () {
      const publicKey = createValidPublicKey(false);
      const did = `did:sage:test:${agent1.address}`;
      
      const signature = await createRegistrationSignature(agent1, publicKey);
      
      // Register
      await sageRegistry.connect(agent1).registerAgent(
        did,
        testName,
        testDescription,
        testEndpoint,
        publicKey,
        testCapabilities,
        signature
      );
      
      // Get by DID
      const agentByDid = await sageRegistry.getAgentByDID(did);
      expect(agentByDid.name).to.equal(testName);
      
      // Get by owner
      const agentIds = await sageRegistry.getAgentsByOwner(agent1.address);
      expect(agentIds.length).to.equal(1);
      
      // Verify ownership
      const isOwner = await sageRegistry.verifyAgentOwnership(agentIds[0], agent1.address);
      expect(isOwner).to.be.true;
      
      // Check active status
      const isActive = await sageRegistry.isAgentActive(agentIds[0]);
      expect(isActive).to.be.true;
    });
  });
});