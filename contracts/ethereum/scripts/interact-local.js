const hre = require("hardhat");
const readline = require("readline");

// Create readline interface for user input
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

// Helper function to get user input
function prompt(question) {
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      resolve(answer);
    });
  });
}

async function main() {
  console.log("🚀 SageRegistryV2 Local Interaction Script");
  console.log("=" .repeat(60));
  
  // Get signers
  const [owner, agent1, agent2] = await hre.ethers.getSigners();
  
  console.log("📍 Connected to local network");
  console.log("👤 Owner address:", owner.address);
  console.log("👤 Agent1 address:", agent1.address);
  console.log("👤 Agent2 address:", agent2.address);
  console.log();

  // Get contract addresses from deployment
  const registryAddress = await prompt("Enter SageRegistryV2 contract address: ");
  const hookAddress = await prompt("Enter SageVerificationHook contract address (optional): ");
  
  // Connect to contracts
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const registry = SageRegistryV2.attach(registryAddress);
  
  console.log("\n✅ Connected to SageRegistryV2 at:", registryAddress);
  
  if (hookAddress) {
    const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
    const hook = SageVerificationHook.attach(hookAddress);
    console.log("✅ Connected to SageVerificationHook at:", hookAddress);
  }
  
  // Main interaction loop
  while (true) {
    console.log("\n" + "=" .repeat(60));
    console.log("Choose an action:");
    console.log("1. Register a new agent");
    console.log("2. View agent details");
    console.log("3. Update agent");
    console.log("4. Deactivate agent");
    console.log("5. Revoke public key");
    console.log("6. List all agents for an owner");
    console.log("7. Check hook configuration");
    console.log("8. Test signature verification");
    console.log("9. Exit");
    console.log();
    
    const choice = await prompt("Enter your choice (1-9): ");
    
    switch(choice) {
      case "1":
        await registerAgent(registry, agent1);
        break;
      case "2":
        await viewAgent(registry);
        break;
      case "3":
        await updateAgent(registry, agent1);
        break;
      case "4":
        await deactivateAgent(registry, agent1);
        break;
      case "5":
        await revokeKey(registry, agent1);
        break;
      case "6":
        await listAgents(registry);
        break;
      case "7":
        await checkHooks(registry);
        break;
      case "8":
        await testSignature(agent1);
        break;
      case "9":
        console.log("👋 Goodbye!");
        rl.close();
        process.exit(0);
      default:
        console.log("❌ Invalid choice");
    }
  }
}

async function registerAgent(registry, signer) {
  console.log("\n📝 Register New Agent");
  console.log("-" .repeat(40));
  
  const did = `did:sage:test:${signer.address}_${Date.now()}`;
  const name = await prompt("Agent name: ");
  const description = await prompt("Agent description: ");
  const endpoint = await prompt("Endpoint URL: ");
  
  // Generate a random public key with proper format (0x04 prefix for uncompressed secp256k1)
  const randomKey = hre.ethers.randomBytes(64);
  const publicKey = hre.ethers.concat(["0x04", randomKey]);
  const capabilities = JSON.stringify(["chat", "code", "analysis"]);
  
  console.log("\n📋 Agent Details:");
  console.log("  DID:", did);
  console.log("  Name:", name);
  console.log("  Description:", description);
  console.log("  Endpoint:", endpoint);
  console.log("  Public Key:", publicKey.substring(0, 20) + "...");
  console.log("  Capabilities:", capabilities);
  
  // Create signature for V2 registration (needs key ownership proof)
  const keyHash = hre.ethers.keccak256(publicKey);
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;
  const registryAddress = await registry.getAddress();
  
  // Create the challenge message (must match contract's _validatePublicKey)
  const challenge = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, signer.address, keyHash]
    )
  );
  
  // Sign the challenge to prove key ownership
  const signature = await signer.signMessage(hre.ethers.getBytes(challenge));
  
  console.log("\n🔐 Signature generated");
  console.log("\n⏳ Registering agent...");
  
  try {
    const tx = await registry.connect(signer).registerAgent(
      did,
      name,
      description,
      endpoint,
      publicKey,
      capabilities,
      signature
    );
    
    const receipt = await tx.wait();
    console.log("✅ Agent registered successfully!");
    console.log("  Transaction hash:", receipt.hash);
    
    // Get the agent ID from events
    const logs = await registry.queryFilter(
      registry.filters.AgentRegistered(),
      receipt.blockNumber,
      receipt.blockNumber
    );
    
    if (logs.length > 0) {
      console.log("  Agent ID:", logs[0].args[0]);
    }
  } catch (error) {
    console.log("❌ Registration failed:", error.message);
  }
}

async function viewAgent(registry) {
  console.log("\n🔍 View Agent Details");
  console.log("-" .repeat(40));
  
  const agentId = await prompt("Enter agent ID (0x...): ");
  
  try {
    const agent = await registry.getAgent(agentId);
    
    console.log("\n📋 Agent Information:");
    console.log("  DID:", agent.did);
    console.log("  Name:", agent.name);
    console.log("  Description:", agent.description);
    console.log("  Owner:", agent.owner);
    console.log("  Endpoint:", agent.endpoint);
    console.log("  Public Key:", agent.publicKey.substring(0, 20) + "...");
    console.log("  Capabilities:", agent.capabilities);
    console.log("  Active:", agent.active);
    console.log("  Created:", new Date(Number(agent.createdAt) * 1000).toLocaleString());
    console.log("  Updated:", new Date(Number(agent.updatedAt) * 1000).toLocaleString());
  } catch (error) {
    console.log("❌ Failed to get agent:", error.message);
  }
}

async function updateAgent(registry, signer) {
  console.log("\n✏️  Update Agent");
  console.log("-" .repeat(40));
  
  const agentId = await prompt("Enter agent ID to update: ");
  const name = await prompt("New name: ");
  const description = await prompt("New description: ");
  const endpoint = await prompt("New endpoint: ");
  const capabilities = await prompt("New capabilities (JSON): ");
  
  // Get current agent to verify ownership
  try {
    const agent = await registry.getAgent(agentId);
    
    if (agent.owner.toLowerCase() !== signer.address.toLowerCase()) {
      console.log("❌ You are not the owner of this agent");
      return;
    }
    
    // Create update signature (nonce is 1 after registration)
    const messageHash = hre.ethers.keccak256(
      hre.ethers.solidityPacked(
        ["bytes32", "string", "string", "string", "string", "address", "uint256"],
        [agentId, name, description, endpoint, capabilities, signer.address, 1]
      )
    );
    
    const signature = await signer.signMessage(hre.ethers.getBytes(messageHash));
    
    console.log("\n⏳ Updating agent...");
    
    const tx = await registry.connect(signer).updateAgent(
      agentId,
      name,
      description,
      endpoint,
      capabilities,
      signature
    );
    
    await tx.wait();
    console.log("✅ Agent updated successfully!");
  } catch (error) {
    console.log("❌ Update failed:", error.message);
  }
}

async function deactivateAgent(registry, signer) {
  console.log("\n🔴 Deactivate Agent");
  console.log("-" .repeat(40));
  
  const agentId = await prompt("Enter agent ID to deactivate: ");
  
  try {
    console.log("\n⏳ Deactivating agent...");
    
    const tx = await registry.connect(signer).deactivateAgent(agentId);
    await tx.wait();
    
    console.log("✅ Agent deactivated successfully!");
  } catch (error) {
    console.log("❌ Deactivation failed:", error.message);
  }
}

async function revokeKey(registry, signer) {
  console.log("\n🔑 Revoke Public Key");
  console.log("-" .repeat(40));
  
  const publicKey = await prompt("Enter public key to revoke (0x...): ");
  
  try {
    console.log("\n⏳ Revoking key...");
    
    const tx = await registry.connect(signer).revokeKey(publicKey);
    await tx.wait();
    
    console.log("✅ Key revoked successfully!");
    console.log("⚠️  All agents using this key are now deactivated");
  } catch (error) {
    console.log("❌ Revocation failed:", error.message);
  }
}

async function listAgents(registry) {
  console.log("\n📋 List Agents by Owner");
  console.log("-" .repeat(40));
  
  const ownerAddress = await prompt("Enter owner address: ");
  
  try {
    const agentIds = await registry.getAgentsByOwner(ownerAddress);
    
    console.log(`\n📊 Found ${agentIds.length} agent(s) for ${ownerAddress}:`);
    
    for (let i = 0; i < agentIds.length; i++) {
      const agent = await registry.getAgent(agentIds[i]);
      console.log(`\n  ${i + 1}. Agent ID: ${agentIds[i]}`);
      console.log(`     Name: ${agent.name}`);
      console.log(`     Active: ${agent.active}`);
      console.log(`     DID: ${agent.did}`);
    }
  } catch (error) {
    console.log("❌ Failed to list agents:", error.message);
  }
}

async function checkHooks(registry) {
  console.log("\n🔧 Hook Configuration");
  console.log("-" .repeat(40));
  
  try {
    const beforeHook = await registry.beforeRegisterHook();
    const afterHook = await registry.afterRegisterHook();
    
    console.log("\n📍 Configured Hooks:");
    console.log("  Before Register Hook:", beforeHook || "Not set");
    console.log("  After Register Hook:", afterHook || "Not set");
  } catch (error) {
    console.log("❌ Failed to check hooks:", error.message);
  }
}

async function testSignature(signer) {
  console.log("\n🔐 Test Signature Generation");
  console.log("-" .repeat(40));
  
  const message = await prompt("Enter message to sign: ");
  
  const messageHash = hre.ethers.keccak256(hre.ethers.toUtf8Bytes(message));
  const signature = await signer.signMessage(hre.ethers.getBytes(messageHash));
  
  console.log("\n📝 Signature Details:");
  console.log("  Message:", message);
  console.log("  Message Hash:", messageHash);
  console.log("  Signature:", signature);
  console.log("  Signer Address:", signer.address);
}

// Error handling
main().catch((error) => {
  console.error("❌ Error:", error);
  process.exit(1);
});