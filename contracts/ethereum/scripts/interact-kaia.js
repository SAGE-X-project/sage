const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

async function main() {
  try {
    // Get network
    const network = hre.network.name;
    console.log(`\n🔗 Interacting with contracts on ${network}...`);
    
    // Load deployment info
    const deploymentPath = path.join(__dirname, `../deployments/${network}-latest.json`);
    
    if (!fs.existsSync(deploymentPath)) {
      throw new Error(`No deployment found for ${network}. Run 'npm run deploy:${network}' first.`);
    }
    
    const deployment = JSON.parse(fs.readFileSync(deploymentPath, 'utf8'));
    console.log(`\n Using deployment from: ${deployment.timestamp}`);
    
    // Get signer
    const [signer] = await hre.ethers.getSigners();
    console.log(`👤 Signer address: ${signer.address}`);
    
    // Connect to contracts
    const SageRegistry = await hre.ethers.getContractFactory("SageRegistry");
    const registry = SageRegistry.attach(deployment.contracts.SageRegistry.address);
    
    const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
    const verificationHook = SageVerificationHook.attach(deployment.contracts.SageVerificationHook.address);
    
    console.log(`\n📍 Contract Addresses:`);
    console.log(`   Registry: ${registry.address}`);
    console.log(`   Hook: ${verificationHook.address}`);
    
    // Check contract state
    console.log(`\n Checking contract state...`);
    
    const owner = await registry.owner();
    console.log(`   Registry Owner: ${owner}`);
    
    const beforeHook = await registry.beforeRegisterHook();
    console.log(`   Before Register Hook: ${beforeHook}`);
    
    const afterHook = await registry.afterRegisterHook();
    console.log(`   After Register Hook: ${afterHook || "Not set"}`);
    
    // Check if signer has any agents
    const agentIds = await registry.getAgentsByOwner(signer.address);
    console.log(`\n Your Agents: ${agentIds.length} registered`);
    
    if (agentIds.length > 0) {
      console.log("\n Agent Details:");
      for (const agentId of agentIds) {
        const agent = await registry.getAgent(agentId);
        console.log(`\n   Agent ID: ${agentId}`);
        console.log(`   - DID: ${agent.did}`);
        console.log(`   - Name: ${agent.name}`);
        console.log(`   - Active: ${agent.active}`);
        console.log(`   - Registered: ${new Date(agent.registeredAt * 1000).toISOString()}`);
      }
    }
    
    // Sample agent registration (commented out to avoid actual transaction)
    console.log("\n To register an agent, uncomment the code below:");
    console.log(`
    // Generate a keypair (you should use proper key generation)
    const wallet = hre.ethers.Wallet.createRandom();
    const publicKey = wallet.publicKey;
    
    // Prepare registration data
    const did = \`did:sage:\${network}:\${wallet.address}\`;
    const name = "Test Agent";
    const description = "A test AI agent";
    const endpoint = "https://example.com/agent";
    const capabilities = JSON.stringify(["chat", "code", "analysis"]);
    
    // Create message for signature
    const messageHash = hre.ethers.utils.keccak256(
      hre.ethers.utils.defaultAbiCoder.encode(
        ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
        [did, name, description, endpoint, publicKey, capabilities, signer.address, 0]
      )
    );
    
    // Sign the message
    const signature = await wallet.signMessage(hre.ethers.utils.arrayify(messageHash));
    
    // Register the agent
    const tx = await registry.registerAgent(
      did, name, description, endpoint, publicKey, capabilities, signature
    );
    
    console.log("Agent registration transaction:", tx.hash);
    await tx.wait();
    console.log("Agent registered successfully!");
    `);
    
    console.log("\n Interaction complete!\n");
    
  } catch (error) {
    console.error("\n Error:", error.message);
    process.exit(1);
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });