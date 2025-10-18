const hre = require("hardhat");

async function main() {
  console.log("\n=== Deploying SageRegistryV4 to Local Network ===");
  console.log("=" .repeat(60));

  // Get signers
  const [deployer, agent1, agent2] = await hre.ethers.getSigners();

  console.log("Network: Hardhat Local");
  console.log("Deployer:", deployer.address);
  console.log("Test Agent 1:", agent1.address);
  console.log("Test Agent 2:", agent2.address);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Deployer balance:", hre.ethers.formatEther(balance), "ETH");
  console.log();

  // Deploy SageRegistryV4
  console.log("Deploying SageRegistryV4...");
  const SageRegistryV4 = await hre.ethers.getContractFactory("SageRegistryV4");
  const sageRegistry = await SageRegistryV4.deploy();
  await sageRegistry.waitForDeployment();

  const registryAddress = await sageRegistry.getAddress();
  console.log("SageRegistryV4 deployed to:", registryAddress);
  console.log();

  // Get chain ID and agent nonce (0 for new registrations)
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;
  const agentNonce = 0n;

  // Test single-key registration (ECDSA)
  console.log("=== Testing Single-Key Registration (ECDSA) ===");

  // Prepare ECDSA key (secp256k1) - compressed format (33 bytes)
  const randomPrivKey1 = hre.ethers.Wallet.createRandom();
  const ecdsaPublicKey = randomPrivKey1.publicKey; // Already in hex format (0x04...)

  const testAgent1 = {
    did: `did:sage:ethereum:${agent1.address.substring(2, 10)}`,
    name: "Test Agent Single-Key",
    description: "Single-key ECDSA agent",
    endpoint: "https://localhost:8080",
    capabilities: JSON.stringify(["chat", "code"])
  };

  // Calculate agentId same as contract: keccak256(abi.encode(did, firstKeyData))
  const abiCoder = hre.ethers.AbiCoder.defaultAbiCoder();
  const agentId1 = hre.ethers.keccak256(
    abiCoder.encode(
      ["string", "bytes"],
      [testAgent1.did, ecdsaPublicKey]
    )
  );

  console.log("Calculated Agent ID:", agentId1);

  // Create message hash for signature: keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce))
  const messageHash1 = hre.ethers.keccak256(
    abiCoder.encode(
      ["bytes32", "bytes", "address", "uint256"],
      [agentId1, ecdsaPublicKey, agent1.address, agentNonce]
    )
  );

  console.log("Message Hash:", messageHash1);

  // Sign with Ethereum personal sign (adds prefix automatically)
  const signature1 = await agent1.signMessage(hre.ethers.getBytes(messageHash1));

  console.log("Signature:", signature1);
  console.log("Signature length:", signature1.length);

  // Prepare registration params for V4 (using struct)
  const params1 = {
    did: testAgent1.did,
    name: testAgent1.name,
    description: testAgent1.description,
    endpoint: testAgent1.endpoint,
    keyTypes: [1], // KeyType.ECDSA
    keyData: [ecdsaPublicKey],
    signatures: [signature1],
    capabilities: testAgent1.capabilities
  };

  // Register agent with single key
  console.log("Registering agent with single ECDSA key...");
  let tx = await sageRegistry.connect(agent1).registerAgent(params1);

  let receipt = await tx.wait();
  console.log("Single-key agent registered!");
  console.log("Gas used:", receipt.gasUsed.toString());

  // Get agent from event
  let logs = await sageRegistry.queryFilter(
    sageRegistry.filters.AgentRegistered(),
    receipt.blockNumber,
    receipt.blockNumber
  );

  if (logs.length > 0) {
    const returnedAgentId = logs[0].args[0];
    console.log("Returned Agent ID:", returnedAgentId);

    const agent = await sageRegistry.getAgent(returnedAgentId);
    console.log("\nAgent Details:");
    console.log("  DID:", agent.did);
    console.log("  Name:", agent.name);
    console.log("  Owner:", agent.owner);
    console.log("  Active:", agent.active);
    console.log("  Keys Count:", agent.keyHashes.length);
  }
  console.log();

  // Test multi-key registration (ECDSA + X25519)
  console.log("=== Testing Multi-Key Registration (ECDSA + X25519) ===");

  // Prepare multiple keys
  const randomPrivKey2 = hre.ethers.Wallet.createRandom();
  const ecdsaKey2 = randomPrivKey2.publicKey;
  const x25519Key = hre.ethers.hexlify(hre.ethers.randomBytes(32)); // X25519 is 32 bytes

  const testAgent2 = {
    did: `did:sage:ethereum:${agent2.address.substring(2, 10)}`,
    name: "Test Agent Multi-Key",
    description: "Multi-key agent with ECDSA and X25519",
    endpoint: "https://localhost:8081",
    capabilities: JSON.stringify(["chat", "encryption"])
  };

  // Calculate agentId for agent2
  const agentId2 = hre.ethers.keccak256(
    abiCoder.encode(
      ["string", "bytes"],
      [testAgent2.did, ecdsaKey2]
    )
  );

  console.log("Calculated Agent ID:", agentId2);

  // Create signature for ECDSA key
  const messageHash2_1 = hre.ethers.keccak256(
    abiCoder.encode(
      ["bytes32", "bytes", "address", "uint256"],
      [agentId2, ecdsaKey2, agent2.address, agentNonce]
    )
  );
  const signature2_1 = await agent2.signMessage(hre.ethers.getBytes(messageHash2_1));

  // Create signature for X25519 key (same format)
  const messageHash2_2 = hre.ethers.keccak256(
    abiCoder.encode(
      ["bytes32", "bytes", "address", "uint256"],
      [agentId2, x25519Key, agent2.address, agentNonce]
    )
  );
  const signature2_2 = await agent2.signMessage(hre.ethers.getBytes(messageHash2_2));

  // Prepare registration params with multiple keys
  const params2 = {
    did: testAgent2.did,
    name: testAgent2.name,
    description: testAgent2.description,
    endpoint: testAgent2.endpoint,
    keyTypes: [1, 2], // KeyType.ECDSA, KeyType.X25519
    keyData: [ecdsaKey2, x25519Key],
    signatures: [signature2_1, signature2_2],
    capabilities: testAgent2.capabilities
  };

  // Register agent with multiple keys
  console.log("Registering agent with ECDSA + X25519 keys...");
  tx = await sageRegistry.connect(agent2).registerAgent(params2);

  receipt = await tx.wait();
  console.log("Multi-key agent registered!");
  console.log("Gas used:", receipt.gasUsed.toString());

  // Get agent from event
  logs = await sageRegistry.queryFilter(
    sageRegistry.filters.AgentRegistered(),
    receipt.blockNumber,
    receipt.blockNumber
  );

  if (logs.length > 0) {
    const returnedAgentId2 = logs[0].args[0];
    console.log("Returned Agent ID:", returnedAgentId2);

    const agent = await sageRegistry.getAgent(returnedAgentId2);
    console.log("\nAgent Details:");
    console.log("  DID:", agent.did);
    console.log("  Name:", agent.name);
    console.log("  Owner:", agent.owner);
    console.log("  Active:", agent.active);
    console.log("  Keys Count:", agent.keyHashes.length);

    // Get and display key details
    for (let i = 0; i < agent.keyHashes.length; i++) {
      const keyHash = agent.keyHashes[i];
      const keyData = await sageRegistry.getKey(keyHash);
      const keyTypeName = keyData.keyType === 0n ? "Ed25519" :
                          keyData.keyType === 1n ? "ECDSA" :
                          keyData.keyType === 2n ? "X25519" : "Unknown";
      console.log(`  Key ${i + 1}:`, keyTypeName, "- Verified:", keyData.verified);
    }
  }
  console.log();

  // Print summary
  console.log("=" .repeat(60));
  console.log("=== Deployment Complete! ===");
  console.log("=" .repeat(60));
  console.log("\nContract Address:");
  console.log("  SageRegistryV4:", registryAddress);
  console.log("\nNetwork Info:");
  console.log("  Chain ID:", chainId);
  console.log("  Network:", "localhost");
  console.log("\nTest Accounts:");
  console.log("  Deployer:", deployer.address);
  console.log("  Agent 1:", agent1.address);
  console.log("  Agent 2:", agent2.address);
  console.log("\nNext Steps:");
  console.log("  1. Use this contract address in Go SDK tests");
  console.log("  2. Contract is ready for multi-key agent registration");
  console.log("  3. Test with: SAGE_INTEGRATION_TEST=1 go test ./pkg/agent/did/ethereum -v");
  console.log();
}

main().catch((error) => {
  console.error("Deployment failed:", error);
  process.exit(1);
});
