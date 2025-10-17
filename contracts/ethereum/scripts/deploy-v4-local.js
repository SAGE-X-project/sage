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

  // Test single-key registration (ECDSA)
  console.log("=== Testing Single-Key Registration (ECDSA) ===");

  // Prepare ECDSA key (secp256k1)
  const randomKey = hre.ethers.randomBytes(64);
  const ecdsaPublicKey = hre.ethers.concat(["0x04", randomKey]); // Uncompressed format

  const testAgent1 = {
    did: `did:sage:ethereum:${agent1.address.substring(2, 10)}`,
    name: "Test Agent Single-Key",
    description: "Single-key ECDSA agent",
    endpoint: "https://localhost:8080",
    capabilities: JSON.stringify(["chat", "code"])
  };

  // Create key registration data for ECDSA
  const keyHash1 = hre.ethers.keccak256(ecdsaPublicKey);
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;

  // Challenge for ECDSA key ownership
  const challenge1 = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, agent1.address, keyHash1]
    )
  );

  const signature1 = await agent1.signMessage(hre.ethers.getBytes(challenge1));

  // Prepare key array with single ECDSA key
  const keys1 = [{
    publicKey: ecdsaPublicKey,
    keyType: 1, // KeyType.ECDSA
    signature: signature1
  }];

  // Register agent with single key
  console.log("Registering agent with single ECDSA key...");
  let tx = await sageRegistry.connect(agent1).registerAgent(
    testAgent1.did,
    testAgent1.name,
    testAgent1.description,
    testAgent1.endpoint,
    testAgent1.capabilities,
    keys1
  );

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
    const agentId1 = logs[0].args[0];
    console.log("Agent ID:", agentId1);

    const agent = await sageRegistry.getAgent(agentId1);
    console.log("\nAgent Details:");
    console.log("  Name:", agent.name);
    console.log("  Owner:", agent.owner);
    console.log("  Active:", agent.active);
    console.log("  Keys Count:", agent.keys.length);
    console.log("  Key Type:", agent.keys[0].keyType === 1n ? "ECDSA" : "Unknown");
    console.log("  Key Verified:", agent.keys[0].verified);
  }
  console.log();

  // Test multi-key registration (ECDSA + X25519)
  console.log("=== Testing Multi-Key Registration (ECDSA + X25519) ===");

  // Prepare multiple keys
  const ecdsaKey2 = hre.ethers.concat(["0x04", hre.ethers.randomBytes(64)]);
  const x25519Key = hre.ethers.randomBytes(32); // X25519 is 32 bytes

  const testAgent2 = {
    did: `did:sage:ethereum:${agent2.address.substring(2, 10)}`,
    name: "Test Agent Multi-Key",
    description: "Multi-key agent with ECDSA and X25519",
    endpoint: "https://localhost:8081",
    capabilities: JSON.stringify(["chat", "encryption"])
  };

  // ECDSA key signature
  const keyHash2_1 = hre.ethers.keccak256(ecdsaKey2);
  const challenge2_1 = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, agent2.address, keyHash2_1]
    )
  );
  const signature2_1 = await agent2.signMessage(hre.ethers.getBytes(challenge2_1));

  // X25519 key signature (owner signature, not key-specific)
  const keyHash2_2 = hre.ethers.keccak256(x25519Key);
  const challenge2_2 = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, agent2.address, keyHash2_2]
    )
  );
  const signature2_2 = await agent2.signMessage(hre.ethers.getBytes(challenge2_2));

  // Prepare key array with ECDSA and X25519
  const keys2 = [
    {
      publicKey: ecdsaKey2,
      keyType: 1, // KeyType.ECDSA
      signature: signature2_1
    },
    {
      publicKey: x25519Key,
      keyType: 2, // KeyType.X25519
      signature: signature2_2
    }
  ];

  // Register agent with multiple keys
  console.log("Registering agent with ECDSA + X25519 keys...");
  tx = await sageRegistry.connect(agent2).registerAgent(
    testAgent2.did,
    testAgent2.name,
    testAgent2.description,
    testAgent2.endpoint,
    testAgent2.capabilities,
    keys2
  );

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
    const agentId2 = logs[0].args[0];
    console.log("Agent ID:", agentId2);

    const agent = await sageRegistry.getAgent(agentId2);
    console.log("\nAgent Details:");
    console.log("  Name:", agent.name);
    console.log("  Owner:", agent.owner);
    console.log("  Active:", agent.active);
    console.log("  Keys Count:", agent.keys.length);

    for (let i = 0; i < agent.keys.length; i++) {
      const keyTypeName = agent.keys[i].keyType === 0n ? "Ed25519" :
                          agent.keys[i].keyType === 1n ? "ECDSA" :
                          agent.keys[i].keyType === 2n ? "X25519" : "Unknown";
      console.log(`  Key ${i + 1}:`, keyTypeName, "- Verified:", agent.keys[i].verified);
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
  console.log("  1. Save the contract address for Go SDK integration");
  console.log("  2. Generate Go bindings:");
  console.log("     cd contracts/ethereum");
  console.log("     npx hardhat run scripts/generate-go-bindings.js");
  console.log("  3. Update pkg/agent/did/ethereum/client.go with V4 contract");
  console.log();
}

main().catch((error) => {
  console.error("Deployment failed:", error);
  process.exit(1);
});
