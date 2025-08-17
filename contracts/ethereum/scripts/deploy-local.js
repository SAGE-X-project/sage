const hre = require("hardhat");

async function main() {
  console.log("\n🚀 Deploying SAGE Contracts to Local Network");
  console.log("=" .repeat(60));
  
  // Get signers
  const [deployer, agent1, agent2] = await hre.ethers.getSigners();
  
  console.log("📍 Network: Hardhat Local");
  console.log("👤 Deployer:", deployer.address);
  console.log("👤 Test Agent 1:", agent1.address);
  console.log("👤 Test Agent 2:", agent2.address);
  
  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("💰 Deployer balance:", hre.ethers.formatEther(balance), "ETH");
  console.log();

  // Deploy SageRegistryV2
  console.log("📝 Deploying SageRegistryV2...");
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const sageRegistry = await SageRegistryV2.deploy();
  await sageRegistry.waitForDeployment();
  
  const registryAddress = await sageRegistry.getAddress();
  console.log("✅ SageRegistryV2 deployed to:", registryAddress);

  // Deploy SageVerificationHook
  console.log("\n📝 Deploying SageVerificationHook...");
  const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
  const verificationHook = await SageVerificationHook.deploy();
  await verificationHook.waitForDeployment();
  
  const hookAddress = await verificationHook.getAddress();
  console.log("✅ SageVerificationHook deployed to:", hookAddress);

  // Configure hooks
  console.log("\n🔧 Configuring hooks...");
  let tx = await sageRegistry.setBeforeRegisterHook(hookAddress);
  await tx.wait();
  console.log("✅ BeforeRegisterHook configured");
  
  tx = await sageRegistry.setAfterRegisterHook(hookAddress);
  await tx.wait();
  console.log("✅ AfterRegisterHook configured");

  // Test registration
  console.log("\n🧪 Testing registration...");
  
  // Prepare test data
  // V2 requires proper public key format (0x04 prefix for uncompressed)
  const randomKey = hre.ethers.randomBytes(64);
  const publicKey = hre.ethers.concat(["0x04", randomKey]); // Add 0x04 prefix for uncompressed key
  
  const testAgent = {
    did: `did:sage:test:${agent1.address}`,
    name: "Test AI Agent",
    description: "A test agent for demonstration",
    endpoint: "https://localhost:8080",
    publicKey: publicKey,
    capabilities: JSON.stringify(["chat", "code", "analysis"])
  };

  // Create signature for V2 (needs key ownership proof)
  // V2 requires a special challenge signature for key ownership verification
  const keyHash = hre.ethers.keccak256(testAgent.publicKey);
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;
  
  // Create the challenge message (must match contract's _validatePublicKey)
  const challenge = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, agent1.address, keyHash]
    )
  );
  
  // Sign the challenge to prove key ownership
  const signature = await agent1.signMessage(hre.ethers.getBytes(challenge));
  
  // Register agent
  console.log("  Registering test agent...");
  tx = await sageRegistry.connect(agent1).registerAgent(
    testAgent.did,
    testAgent.name,
    testAgent.description,
    testAgent.endpoint,
    testAgent.publicKey,
    testAgent.capabilities,
    signature
  );
  
  const receipt = await tx.wait();
  console.log("  ✅ Test agent registered!");
  console.log("  Gas used:", receipt.gasUsed.toString());
  
  // Get agent ID from event
  const logs = await sageRegistry.queryFilter(
    sageRegistry.filters.AgentRegistered(),
    receipt.blockNumber,
    receipt.blockNumber
  );
  
  if (logs.length > 0) {
    const agentId = logs[0].args[0];
    console.log("  Agent ID:", agentId);
    
    // Verify agent data
    const agent = await sageRegistry.getAgent(agentId);
    console.log("\n📋 Registered Agent Details:");
    console.log("  Name:", agent.name);
    console.log("  Owner:", agent.owner);
    console.log("  Active:", agent.active);
    console.log("  DID:", agent.did);
  }

  // Print summary
  console.log("\n" + "=" .repeat(60));
  console.log("🎉 Deployment Complete!");
  console.log("=" .repeat(60));
  console.log("\n📍 Contract Addresses:");
  console.log("  SageRegistryV2:", registryAddress);
  console.log("  SageVerificationHook:", hookAddress);
  console.log("\n💡 Next Steps:");
  console.log("  1. Keep this terminal running");
  console.log("  2. In a new terminal, run:");
  console.log("     npx hardhat run scripts/interact-local.js --network localhost");
  console.log("  3. Use the contract addresses above when prompted");
  console.log("\n📚 Available Test Accounts:");
  console.log("  Account 0 (Owner):", deployer.address);
  console.log("  Account 1 (Agent1):", agent1.address);
  console.log("  Account 2 (Agent2):", agent2.address);
}

main().catch((error) => {
  console.error("❌ Deployment failed:", error);
  process.exit(1);
});