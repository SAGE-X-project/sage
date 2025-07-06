const ethers = require("ethers");
const fs = require("fs");

// Load contract ABIs
const registryABI = require("../ethereum/artifacts/contracts/SageRegistry.sol/SageRegistry.json").abi;

async function registerAgent() {
  // Setup provider and signer
  const provider = new ethers.providers.JsonRpcProvider(process.env.RPC_URL || "http://localhost:8545");
  const signer = new ethers.Wallet(process.env.PRIVATE_KEY || "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", provider);
  
  // Load deployment info
  const deploymentInfo = JSON.parse(fs.readFileSync("../ethereum/deployments/localhost.json", "utf8"));
  const registryAddress = deploymentInfo.contracts.SageRegistry;
  
  // Create contract instance
  const registry = new ethers.Contract(registryAddress, registryABI, signer);
  
  // Agent metadata
  const agentData = {
    did: "did:sage:agent001",
    name: "SAGE AI Assistant",
    description: "An AI agent for helping with smart contract development",
    endpoint: "https://api.sage.ai/agent/001",
    publicKey: ethers.utils.hexlify(ethers.utils.randomBytes(64)), // Secp256k1 public key
    capabilities: JSON.stringify({
      models: ["gpt-4", "claude-3"],
      skills: ["smart-contract-analysis", "code-generation"],
      languages: ["en", "ko"],
    }),
  };
  
  // Create message to sign
  const nonce = 0; // First registration
  const messageHash = ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
      [
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.publicKey,
        agentData.capabilities,
        signer.address,
        nonce
      ]
    )
  );
  
  // Sign the message
  const signature = await signer.signMessage(ethers.utils.arrayify(messageHash));
  
  console.log("Registering agent...");
  console.log("DID:", agentData.did);
  console.log("Owner:", signer.address);
  
  try {
    // Register the agent
    const tx = await registry.registerAgent(
      agentData.did,
      agentData.name,
      agentData.description,
      agentData.endpoint,
      agentData.publicKey,
      agentData.capabilities,
      signature
    );
    
    console.log("Transaction hash:", tx.hash);
    const receipt = await tx.wait();
    console.log("Transaction confirmed in block:", receipt.blockNumber);
    
    // Get the agent ID from events
    const event = receipt.events.find(e => e.event === "AgentRegistered");
    const agentId = event.args.agentId;
    console.log("Agent ID:", agentId);
    
    // Query the registered agent
    const agent = await registry.getAgent(agentId);
    console.log("\nRegistered agent:");
    console.log("- Name:", agent.name);
    console.log("- DID:", agent.did);
    console.log("- Active:", agent.active);
    console.log("- Owner:", agent.owner);
    
  } catch (error) {
    console.error("Registration failed:", error.message);
    if (error.data) {
      const decodedError = registry.interface.parseError(error.data);
      console.error("Error reason:", decodedError);
    }
  }
}

// Run the registration
registerAgent().catch(console.error);