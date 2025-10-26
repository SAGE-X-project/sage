const hre = require("hardhat");

async function main() {
  console.log("\n=== Deploying SageRegistryV4 to Local Network ===");
  console.log("=" .repeat(60));

  // Get deployer signer
  const [deployer] = await hre.ethers.getSigners();

  console.log("Network: Hardhat Local");
  console.log("Deployer:", deployer.address);

  const deployerBalance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Deployer balance:", hre.ethers.formatEther(deployerBalance), "ETH");
  console.log();

  // Deploy SageRegistryV4
  console.log("Deploying SageRegistryV4...");
  const SageRegistryV4 = await hre.ethers.getContractFactory("SageRegistryV4");
  const sageRegistry = await SageRegistryV4.deploy();
  await sageRegistry.waitForDeployment();

  const registryAddress = await sageRegistry.getAddress();
  console.log("SageRegistryV4 deployed to:", registryAddress);
  console.log();

  // Create agent wallets (agents generate their own keys)
  console.log("=== Creating Agent Wallets ===");
  const agent1Wallet = hre.ethers.Wallet.createRandom();
  const agent2Wallet = hre.ethers.Wallet.createRandom();

  console.log("Agent 1 Address:", agent1Wallet.address);
  console.log("Agent 2 Address:", agent2Wallet.address);
  console.log();

  // Fund agent wallets with gas (deployer sends ETH to agents)
  console.log("=== Funding Agent Wallets ===");
  const fundAmount = hre.ethers.parseEther("1.0");

  console.log("Sending", hre.ethers.formatEther(fundAmount), "ETH to Agent 1...");
  let fundTx = await deployer.sendTransaction({
    to: agent1Wallet.address,
    value: fundAmount
  });
  await fundTx.wait();

  console.log("Sending", hre.ethers.formatEther(fundAmount), "ETH to Agent 2...");
  fundTx = await deployer.sendTransaction({
    to: agent2Wallet.address,
    value: fundAmount
  });
  await fundTx.wait();

  const agent1Balance = await hre.ethers.provider.getBalance(agent1Wallet.address);
  const agent2Balance = await hre.ethers.provider.getBalance(agent2Wallet.address);
  console.log("Agent 1 balance:", hre.ethers.formatEther(agent1Balance), "ETH");
  console.log("Agent 2 balance:", hre.ethers.formatEther(agent2Balance), "ETH");
  console.log();

  // Connect wallets to provider
  const agent1Connected = agent1Wallet.connect(hre.ethers.provider);
  const agent2Connected = agent2Wallet.connect(hre.ethers.provider);

  // Get chain ID and agent nonce (0 for new registrations)
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;
  const agentNonce = 0n;

  // Test single-key registration (ECDSA)
  console.log("=== Testing Single-Key Registration (ECDSA) ===");

  // Prepare ECDSA key (secp256k1) - agent1's own public key
  // IMPORTANT: The public key MUST match the signer (agent1)
  const ecdsaPublicKey = agent1Wallet.signingKey.publicKey; // Already in hex format (0x04...)

  const testAgent1 = {
    did: `did:sage:ethereum:${agent1Wallet.address.substring(2, 10)}`,
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
      [agentId1, ecdsaPublicKey, agent1Wallet.address, agentNonce]
    )
  );

  console.log("Message Hash:", messageHash1);

  // Sign with Ethereum personal sign (adds prefix automatically)
  const signature1 = await agent1Connected.signMessage(hre.ethers.getBytes(messageHash1));

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

  // Register agent with single key (agent1 sends transaction with own key)
  console.log("Registering agent with single ECDSA key...");
  let tx = await sageRegistry.connect(agent1Connected).registerAgent(params1);

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

  // Test multi-key registration (Ed25519 + ECDSA)
  console.log("=== Testing Multi-Key Registration (Ed25519 + ECDSA) ===");

  // Prepare multiple keys
  // IMPORTANT: The ECDSA public key MUST match the signer (agent2)
  const ecdsaKey2 = agent2Wallet.signingKey.publicKey;
  const ed25519Key = hre.ethers.hexlify(hre.ethers.randomBytes(32)); // Ed25519 is 32 bytes

  const testAgent2 = {
    did: `did:sage:ethereum:${agent2Wallet.address.substring(2, 10)}`,
    name: "Test Agent Multi-Key",
    description: "Multi-key agent with Ed25519 and ECDSA",
    endpoint: "https://localhost:8081",
    capabilities: JSON.stringify(["chat", "multi-chain"])
  };

  // Calculate agentId for agent2 (using first key - Ed25519)
  const agentId2 = hre.ethers.keccak256(
    abiCoder.encode(
      ["string", "bytes"],
      [testAgent2.did, ed25519Key]
    )
  );

  console.log("Calculated Agent ID:", agentId2);

  // Create signature for Ed25519 key
  // Note: Ed25519 keys require owner approval, so we just create a dummy signature
  const messageHash2_1 = hre.ethers.keccak256(
    abiCoder.encode(
      ["bytes32", "bytes", "address", "uint256"],
      [agentId2, ed25519Key, agent2Wallet.address, agentNonce]
    )
  );
  const signature2_1 = await agent2Connected.signMessage(hre.ethers.getBytes(messageHash2_1));

  // Create signature for ECDSA key (will be verified on-chain)
  const messageHash2_2 = hre.ethers.keccak256(
    abiCoder.encode(
      ["bytes32", "bytes", "address", "uint256"],
      [agentId2, ecdsaKey2, agent2Wallet.address, agentNonce]
    )
  );
  const signature2_2 = await agent2Connected.signMessage(hre.ethers.getBytes(messageHash2_2));

  // Prepare registration params with multiple keys
  const params2 = {
    did: testAgent2.did,
    name: testAgent2.name,
    description: testAgent2.description,
    endpoint: testAgent2.endpoint,
    keyTypes: [0, 1], // KeyType.Ed25519, KeyType.ECDSA
    keyData: [ed25519Key, ecdsaKey2],
    signatures: [signature2_1, signature2_2],
    capabilities: testAgent2.capabilities
  };

  // Register agent with multiple keys (agent2 sends transaction with own key)
  console.log("Registering agent with Ed25519 + ECDSA keys...");
  tx = await sageRegistry.connect(agent2Connected).registerAgent(params2);

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
    console.log("\nKey Details:");
    for (let i = 0; i < agent.keyHashes.length; i++) {
      const keyHash = agent.keyHashes[i];
      const keyData = await sageRegistry.getKey(keyHash);
      const keyTypeName = keyData.keyType === 0n ? "Ed25519" :
                          keyData.keyType === 1n ? "ECDSA" : "Unknown";
      console.log(`  Key ${i + 1}:`, keyTypeName, "- Verified:", keyData.verified);

      // Approve Ed25519 keys (deployer is contract owner)
      if (keyData.keyType === 0n && !keyData.verified) {
        console.log(`    Approving Ed25519 key...`);
        const approveTx = await sageRegistry.connect(deployer).approveEd25519Key(keyHash);
        await approveTx.wait();
        console.log(`    Ed25519 key approved!`);
      }
    }

    // Display updated verification status
    console.log("\nUpdated Key Verification Status:");
    for (let i = 0; i < agent.keyHashes.length; i++) {
      const keyHash = agent.keyHashes[i];
      const keyData = await sageRegistry.getKey(keyHash);
      const keyTypeName = keyData.keyType === 0n ? "Ed25519" :
                          keyData.keyType === 1n ? "ECDSA" : "Unknown";
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
  console.log("  Agent 1:", agent1Wallet.address);
  console.log("  Agent 2:", agent2Wallet.address);
  console.log("\nAgent Private Keys (for testing):");
  console.log("  Agent 1:", agent1Wallet.privateKey);
  console.log("  Agent 2:", agent2Wallet.privateKey);
  console.log("\n" + "=" .repeat(60));
  console.log("=== Go Test Setup ===");
  console.log("=" .repeat(60));
  console.log("\nSet environment variable:");
  console.log("  export SAGE_V4_CONTRACT_ADDRESS=" + registryAddress);
  console.log("\nRun Go tests:");
  console.log("  SAGE_INTEGRATION_TEST=1 go test -v github.com/sage-x-project/sage/pkg/agent/did/ethereum \\");
  console.log("    -run 'TestV4DIDLifecycleWithFundedKey'");
  console.log();
}

main().catch((error) => {
  console.error("Deployment failed:", error);
  process.exit(1);
});
