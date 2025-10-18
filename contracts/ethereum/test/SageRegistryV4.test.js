const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

describe("SageRegistryV4 - Multi-Key Support", function () {
  let sageRegistry;
  let owner;
  let agent1;
  let agent2;
  let attacker;

  // Test data
  const testName = "Test AI Agent";
  const testDescription = "A multi-chain AI agent";
  const testEndpoint = "https://agent.example.com";

  // A2A Agent Card example
  const testCapabilities = JSON.stringify({
    id: "did:sage:test:agent1",
    name: "Test Agent",
    description: "A multi-chain AI agent",
    endpoints: [
      {
        type: "grpc",
        uri: "https://agent.example.com:443"
      }
    ],
    publicKeys: [],  // Will be populated with actual keys
    capabilities: ["chat", "code-generation", "analysis"]
  });

  /**
   * Helper function to get the actual public key from a wallet
   * Returns the uncompressed public key (65 bytes with 0x04 prefix)
   */
  function getPublicKeyFromWallet(wallet) {
    return wallet.signingKey.publicKey;
  }

  /**
   * Helper function to calculate agent ID
   */
  function calculateAgentId(did, firstKeyData) {
    return ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "bytes"],
        [did, firstKeyData]
      )
    );
  }

  /**
   * Helper function to create a registration signature for ECDSA key
   */
  async function createEcdsaRegistrationSignature(wallet, agentId, keyData) {
    const messageHash = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["bytes32", "bytes", "address", "uint256"],
        [agentId, keyData, wallet.address, 0]  // nonce is 0 for initial registration
      )
    );

    return await wallet.signMessage(ethers.getBytes(messageHash));
  }

  /**
   * Helper function to create Ed25519 key (32 bytes)
   */
  function createEd25519Key() {
    return ethers.hexlify(ethers.randomBytes(32));
  }

  /**
   * Helper function to create a mock Ed25519 signature
   */
  function createMockEd25519Signature() {
    return ethers.hexlify(ethers.randomBytes(64));
  }

  /**
   * Helper function to get KeyType enum value
   */
  const KeyType = {
    Ed25519: 0,
    ECDSA: 1
  };

  beforeEach(async function () {
    // Get signers
    [owner, agent1, agent2, attacker] = await ethers.getSigners();

    // Deploy SageRegistryV4
    const SageRegistryV4 = await ethers.getContractFactory("SageRegistryV4");
    sageRegistry = await SageRegistryV4.deploy();
    await sageRegistry.waitForDeployment();
  });

  describe("Deployment", function () {
    it("Should set the correct owner", async function () {
      expect(await sageRegistry.OWNER()).to.equal(owner.address);
    });

    it("Should have no hooks set initially", async function () {
      expect(await sageRegistry.beforeRegisterHook()).to.equal(ethers.ZeroAddress);
      expect(await sageRegistry.afterRegisterHook()).to.equal(ethers.ZeroAddress);
    });
  });

  describe("Single Key Registration - ECDSA", function () {
    it("Should register agent with single ECDSA key", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const did = `did:sage:test:${testWallet.address}`;

      // Fund the test wallet
      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // Calculate agent ID
      const agentId = calculateAgentId(did, ecdsaKey);

      const signature = await createEcdsaRegistrationSignature(testWallet, agentId, ecdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      // Get the agent ID from the event
      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      const returnedAgentId = registerEvent.args[0];

      // Verify agent was registered
      const agent = await sageRegistry.getAgent(returnedAgentId);
      expect(agent.did).to.equal(did);
      expect(agent.name).to.equal(testName);
      expect(agent.owner).to.equal(testWallet.address);
      expect(agent.active).to.be.true;
      expect(agent.keyHashes.length).to.equal(1);

      // Verify key was stored
      const keyHash = agent.keyHashes[0];
      const storedKey = await sageRegistry.getKey(keyHash);
      expect(storedKey.keyType).to.equal(KeyType.ECDSA);
      expect(storedKey.keyData).to.equal(ecdsaKey);
      expect(storedKey.verified).to.be.true;  // ECDSA keys are verified on-chain
    });

    it("Should reject ECDSA registration with invalid signature", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const invalidSignature = ethers.hexlify(ethers.randomBytes(65));

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [invalidSignature],
        capabilities: testCapabilities
      };

      await expect(
        sageRegistry.connect(testWallet).registerAgent(params)
      ).to.be.revertedWith("Invalid signature");
    });
  });

  describe("Multi-Key Registration", function () {
    it("Should register agent with ECDSA and Ed25519 keys", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const ed25519Key = createEd25519Key();
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // Calculate agent ID using first key (ECDSA in this case)
      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const ecdsaSignature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);
      const ed25519Signature = createMockEd25519Signature();

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA, KeyType.Ed25519],
        keyData: [ecdsaKey, ed25519Key],
        signatures: [ecdsaSignature, ed25519Signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      const agentId = registerEvent.args[0];

      // Verify agent has 2 keys
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.keyHashes.length).to.equal(2);

      // Verify ECDSA key is verified
      const ecdsaKeyHash = agent.keyHashes[0];
      const ecdsaStoredKey = await sageRegistry.getKey(ecdsaKeyHash);
      expect(ecdsaStoredKey.keyType).to.equal(KeyType.ECDSA);
      expect(ecdsaStoredKey.verified).to.be.true;

      // Verify Ed25519 key is NOT verified (needs owner approval)
      const ed25519KeyHash = agent.keyHashes[1];
      const ed25519StoredKey = await sageRegistry.getKey(ed25519KeyHash);
      expect(ed25519StoredKey.keyType).to.equal(KeyType.Ed25519);
      expect(ed25519StoredKey.verified).to.be.false;
    });

    it("Should reject registration with mismatched array lengths", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA, KeyType.Ed25519],
        keyData: [getPublicKeyFromWallet(testWallet)],  // Only 1 key
        signatures: [ethers.hexlify(ethers.randomBytes(65))],
        capabilities: testCapabilities
      };

      await expect(
        sageRegistry.connect(testWallet).registerAgent(params)
      ).to.be.revertedWith("Key arrays length mismatch");
    });

    it("Should reject registration with too many keys", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      // Create 11 keys (exceeds MAX_KEYS_PER_AGENT = 10)
      const keyTypes = Array(11).fill(KeyType.Ed25519);
      const keyData = Array(11).fill(0).map(() => createEd25519Key());
      const signatures = Array(11).fill(0).map(() => createMockEd25519Signature());

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: keyTypes,
        keyData: keyData,
        signatures: signatures,
        capabilities: testCapabilities
      };

      await expect(
        sageRegistry.connect(testWallet).registerAgent(params)
      ).to.be.revertedWith("Invalid key count");
    });
  });

  describe("Ed25519 Key Approval", function () {
    let agentId;
    let ed25519KeyHash;
    let testWallet;

    beforeEach(async function () {
      testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const ed25519Key = createEd25519Key();
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const ecdsaSignature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);
      const ed25519Signature = createMockEd25519Signature();

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA, KeyType.Ed25519],
        keyData: [ecdsaKey, ed25519Key],
        signatures: [ecdsaSignature, ed25519Signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      agentId = registerEvent.args[0];

      const agent = await sageRegistry.getAgent(agentId);
      ed25519KeyHash = agent.keyHashes[1];
    });

    it("Should allow owner to approve Ed25519 key", async function () {
      // Verify key is not verified initially
      let key = await sageRegistry.getKey(ed25519KeyHash);
      expect(key.verified).to.be.false;

      // Owner approves the key
      const tx = await sageRegistry.connect(owner).approveEd25519Key(ed25519KeyHash);
      await expect(tx).to.emit(sageRegistry, "Ed25519KeyApproved");

      // Verify key is now verified
      key = await sageRegistry.getKey(ed25519KeyHash);
      expect(key.verified).to.be.true;
    });

    it("Should reject non-owner approval attempt", async function () {
      await expect(
        sageRegistry.connect(agent1).approveEd25519Key(ed25519KeyHash)
      ).to.be.revertedWith("Only owner");
    });

    it("Should reject approval of non-Ed25519 key", async function () {
      const agent = await sageRegistry.getAgent(agentId);
      const ecdsaKeyHash = agent.keyHashes[0];  // First key is ECDSA

      await expect(
        sageRegistry.connect(owner).approveEd25519Key(ecdsaKeyHash)
      ).to.be.revertedWith("Not Ed25519 key");
    });
  });

  describe("Key Management", function () {
    let agentId;
    let testWallet;

    beforeEach(async function () {
      testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const signature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      agentId = registerEvent.args[0];
    });

    it("Should allow agent owner to add new key", async function () {
      const newEd25519Key = createEd25519Key();
      const signature = createMockEd25519Signature();

      const tx = await sageRegistry.connect(testWallet).addKey(
        agentId,
        KeyType.Ed25519,
        newEd25519Key,
        signature
      );

      await expect(tx).to.emit(sageRegistry, "KeyAdded");

      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.keyHashes.length).to.equal(2);
    });

    it("Should reject non-owner adding key", async function () {
      const newEd25519Key = createEd25519Key();
      const signature = createMockEd25519Signature();

      await expect(
        sageRegistry.connect(agent1).addKey(
          agentId,
          KeyType.Ed25519,
          newEd25519Key,
          signature
        )
      ).to.be.revertedWith("Not agent owner");
    });

    it("Should allow agent owner to revoke key", async function () {
      // First add a second key
      const newEd25519Key = createEd25519Key();
      const signature = createMockEd25519Signature();
      await sageRegistry.connect(testWallet).addKey(
        agentId,
        KeyType.Ed25519,
        newEd25519Key,
        signature
      );

      const agent = await sageRegistry.getAgent(agentId);
      const keyHashToRevoke = agent.keyHashes[1];

      const tx = await sageRegistry.connect(testWallet).revokeKey(agentId, keyHashToRevoke);
      await expect(tx).to.emit(sageRegistry, "KeyRevoked");

      // Key should be completely deleted
      const updatedAgent = await sageRegistry.getAgent(agentId);
      expect(updatedAgent.keyHashes.length).to.equal(1);

      // Trying to get deleted key should revert with "Key not found"
      await expect(
        sageRegistry.getKey(keyHashToRevoke)
      ).to.be.revertedWith("Key not found");
    });

    it("Should prevent revoking last key", async function () {
      const agent = await sageRegistry.getAgent(agentId);
      const onlyKeyHash = agent.keyHashes[0];

      await expect(
        sageRegistry.connect(testWallet).revokeKey(agentId, onlyKeyHash)
      ).to.be.revertedWith("Cannot revoke last key");
    });
  });

  describe("Key Length Validation", function () {
    it("Should reject Ed25519 key with invalid length", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const invalidEd25519Key = ethers.hexlify(ethers.randomBytes(31));  // Wrong length
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.Ed25519],
        keyData: [invalidEd25519Key],
        signatures: [createMockEd25519Signature()],
        capabilities: testCapabilities
      };

      await expect(
        sageRegistry.connect(testWallet).registerAgent(params)
      ).to.be.revertedWith("Invalid Ed25519 key length");
    });

    it("Should reject ECDSA key with invalid length", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const invalidEcdsaKey = ethers.hexlify(ethers.randomBytes(40));  // Wrong length
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, invalidEcdsaKey);
      const signature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, invalidEcdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [invalidEcdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      await expect(
        sageRegistry.connect(testWallet).registerAgent(params)
      ).to.be.revertedWith("Invalid ECDSA key length");
    });
  });

  describe("Query Functions", function () {
    let agentId;
    let testWallet;
    let did;

    beforeEach(async function () {
      testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const signature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      agentId = registerEvent.args[0];
    });

    it("Should get agent by ID", async function () {
      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.did).to.equal(did);
      expect(agent.name).to.equal(testName);
      expect(agent.active).to.be.true;
    });

    it("Should get agent by DID", async function () {
      const agent = await sageRegistry.getAgentByDID(did);
      expect(agent.name).to.equal(testName);
      expect(agent.owner).to.equal(testWallet.address);
    });

    it("Should get agent keys", async function () {
      const keyHashes = await sageRegistry.getAgentKeys(agentId);
      expect(keyHashes.length).to.equal(1);
    });

    it("Should get agents by owner", async function () {
      const agentIds = await sageRegistry.getAgentsByOwner(testWallet.address);
      expect(agentIds.length).to.equal(1);
      expect(agentIds[0]).to.equal(agentId);
    });

    it("Should verify agent ownership", async function () {
      const isOwner = await sageRegistry.verifyAgentOwnership(agentId, testWallet.address);
      expect(isOwner).to.be.true;

      const isNotOwner = await sageRegistry.verifyAgentOwnership(agentId, agent1.address);
      expect(isNotOwner).to.be.false;
    });

    it("Should check if agent is active", async function () {
      const isActive = await sageRegistry.isAgentActive(agentId);
      expect(isActive).to.be.true;
    });
  });

  describe("Agent Update and Deactivation", function () {
    let agentId;
    let testWallet;

    beforeEach(async function () {
      testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const did = `did:sage:test:${testWallet.address}`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const signature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      const registerEvent = receipt.logs.find(
        log => log.fragment && log.fragment.name === "AgentRegistered"
      );
      agentId = registerEvent.args[0];
    });

    it("Should allow agent owner to update metadata", async function () {
      const newName = "Updated Agent Name";
      const newDescription = "Updated description";

      const messageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
          ["bytes32", "string", "string", "string", "string", "address", "uint256"],
          [agentId, newName, newDescription, testEndpoint, testCapabilities, testWallet.address, 1]
        )
      );

      const signature = await testWallet.signMessage(ethers.getBytes(messageHash));

      const tx = await sageRegistry.connect(testWallet).updateAgent(
        agentId,
        newName,
        newDescription,
        testEndpoint,
        testCapabilities,
        signature
      );

      await expect(tx).to.emit(sageRegistry, "AgentUpdated");

      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.name).to.equal(newName);
      expect(agent.description).to.equal(newDescription);
    });

    it("Should allow agent owner to deactivate agent", async function () {
      const tx = await sageRegistry.connect(testWallet).deactivateAgent(agentId);
      await expect(tx).to.emit(sageRegistry, "AgentDeactivated");

      const agent = await sageRegistry.getAgent(agentId);
      expect(agent.active).to.be.false;
    });

    it("Should reject non-owner deactivation attempt", async function () {
      await expect(
        sageRegistry.connect(agent1).deactivateAgent(agentId)
      ).to.be.revertedWith("Not agent owner");
    });
  });

  describe("Hook Management", function () {
    it("Should allow owner to set hooks", async function () {
      const mockHookAddress = agent1.address;

      await sageRegistry.connect(owner).setBeforeRegisterHook(mockHookAddress);
      expect(await sageRegistry.beforeRegisterHook()).to.equal(mockHookAddress);

      await sageRegistry.connect(owner).setAfterRegisterHook(mockHookAddress);
      expect(await sageRegistry.afterRegisterHook()).to.equal(mockHookAddress);
    });

    it("Should reject non-owner setting hooks", async function () {
      const mockHookAddress = agent1.address;

      await expect(
        sageRegistry.connect(agent1).setBeforeRegisterHook(mockHookAddress)
      ).to.be.revertedWith("Only owner");

      await expect(
        sageRegistry.connect(agent1).setAfterRegisterHook(mockHookAddress)
      ).to.be.revertedWith("Only owner");
    });
  });

  describe("Gas Usage", function () {
    it("Should measure gas for single ECDSA key registration", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const did = `did:sage:test:${testWallet.address}_gas`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const signature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA],
        keyData: [ecdsaKey],
        signatures: [signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      console.log(`      Gas used for single ECDSA key: ${receipt.gasUsed.toString()}`);
      expect(receipt.gasUsed).to.be.lt(1000000);
    });

    it("Should measure gas for multi-key registration", async function () {
      const testWallet = ethers.Wallet.createRandom().connect(ethers.provider);
      const ecdsaKey = testWallet.signingKey.publicKey;
      const ed25519Key = createEd25519Key();
      const did = `did:sage:test:${testWallet.address}_multikey`;

      await owner.sendTransaction({
        to: testWallet.address,
        value: ethers.parseEther("1.0")
      });

      const agentIdCalc = calculateAgentId(did, ecdsaKey);
      const ecdsaSignature = await createEcdsaRegistrationSignature(testWallet, agentIdCalc, ecdsaKey);
      const ed25519Signature = createMockEd25519Signature();

      const params = {
        did: did,
        name: testName,
        description: testDescription,
        endpoint: testEndpoint,
        keyTypes: [KeyType.ECDSA, KeyType.Ed25519],
        keyData: [ecdsaKey, ed25519Key],
        signatures: [ecdsaSignature, ed25519Signature],
        capabilities: testCapabilities
      };

      const tx = await sageRegistry.connect(testWallet).registerAgent(params);
      const receipt = await tx.wait();

      console.log(`      Gas used for 2-key registration: ${receipt.gasUsed.toString()}`);
      expect(receipt.gasUsed).to.be.lt(1500000);
    });
  });
});
