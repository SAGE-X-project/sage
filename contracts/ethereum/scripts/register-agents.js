const hre = require("hardhat");
const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

// Agent configurations for different environments
const AGENT_CONFIGS = {
  local: [
    {
      did: "did:sage:local:root",
      name: "Root Agent",
      description: "Main routing agent for SAGE system",
      endpoint: "http://localhost:3001",
      capabilities: ["routing", "management", "verification"],
      privateKey: null // Will be generated
    },
    {
      did: "did:sage:local:ordering",
      name: "Ordering Agent",
      description: "Handles order processing and management",
      endpoint: "http://localhost:3002",
      capabilities: ["ordering", "payment", "tracking"],
      privateKey: null
    },
    {
      did: "did:sage:local:planning",
      name: "Planning Agent",
      description: "Handles travel and event planning",
      endpoint: "http://localhost:3003",
      capabilities: ["booking", "scheduling", "recommendation"],
      privateKey: null
    }
  ],
  kairos: [
    {
      did: "did:sage:kairos:root",
      name: "Root Agent",
      description: "Main routing agent for SAGE testnet",
      endpoint: "https://root.sage-kairos.ai",
      capabilities: ["routing", "management", "verification"],
      privateKey: process.env.ROOT_AGENT_PRIVATE_KEY
    },
    {
      did: "did:sage:kairos:ordering",
      name: "Ordering Agent",
      description: "Handles order processing on testnet",
      endpoint: "https://ordering.sage-kairos.ai",
      capabilities: ["ordering", "payment", "tracking"],
      privateKey: process.env.ORDERING_AGENT_PRIVATE_KEY
    },
    {
      did: "did:sage:kairos:planning",
      name: "Planning Agent",
      description: "Handles planning services on testnet",
      endpoint: "https://planning.sage-kairos.ai",
      capabilities: ["booking", "scheduling", "recommendation"],
      privateKey: process.env.PLANNING_AGENT_PRIVATE_KEY
    }
  ],
  kaia: [
    {
      did: "did:sage:kaia:root",
      name: "SAGE Root Agent",
      description: "Production routing agent for SAGE protocol",
      endpoint: "https://root.sage.ai",
      capabilities: ["routing", "management", "verification", "security"],
      privateKey: process.env.PROD_ROOT_AGENT_PRIVATE_KEY
    },
    {
      did: "did:sage:kaia:ordering",
      name: "SAGE Ordering Agent",
      description: "Production order processing service",
      endpoint: "https://ordering.sage.ai",
      capabilities: ["ordering", "payment", "tracking", "refund"],
      privateKey: process.env.PROD_ORDERING_AGENT_PRIVATE_KEY
    },
    {
      did: "did:sage:kaia:planning",
      name: "SAGE Planning Agent",
      description: "Production planning and booking service",
      endpoint: "https://planning.sage.ai",
      capabilities: ["booking", "scheduling", "recommendation", "optimization"],
      privateKey: process.env.PROD_PLANNING_AGENT_PRIVATE_KEY
    }
  ]
};

// Generate secp256k1 keypair for agent
function generateAgentKeypair() {
  // For production, use proper secp256k1 library
  // For now, generate random bytes as placeholder
  const privateKey = crypto.randomBytes(32);
  const publicKey = crypto.randomBytes(64);
  
  // Add uncompressed public key prefix
  const uncompressedPublicKey = Buffer.concat([Buffer.from([0x04]), publicKey]);
  
  return {
    privateKey: "0x" + privateKey.toString("hex"),
    publicKey: "0x" + uncompressedPublicKey.toString("hex")
  };
}

async function registerAgents(options = {}) {
  const network = options.network || hre.network.name;
  const registryAddress = options.registryAddress;
  const agentConfigs = options.agents || AGENT_CONFIGS[network] || AGENT_CONFIGS.local;
  
  console.log("\n Agent Registration Script");
  console.log("=" .repeat(60));
  console.log(`📍 Network: ${network}`);
  console.log(` Registry: ${registryAddress || "Will load from deployment"}`);
  console.log(`👥 Agents to register: ${agentConfigs.length}`);
  console.log("=" .repeat(60));
  
  // Load deployment info if registry address not provided
  let contractAddress = registryAddress;
  if (!contractAddress) {
    try {
      const deploymentFile = path.join(__dirname, "..", "deployments", `${network}.json`);
      const deploymentData = JSON.parse(fs.readFileSync(deploymentFile, "utf8"));
      contractAddress = deploymentData.contracts.SageRegistryV2.address;
      console.log(` Loaded registry address from deployment: ${contractAddress}`);
    } catch (error) {
      console.error(" Failed to load deployment info:", error.message);
      console.log(" Please provide registry address or deploy contracts first");
      process.exit(1);
    }
  }
  
  // Connect to registry contract
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const registry = SageRegistryV2.attach(contractAddress);
  
  // Get signer
  const [signer] = await hre.ethers.getSigners();
  console.log(`\n👤 Registering with account: ${signer.address}`);
  
  const balance = await hre.ethers.provider.getBalance(signer.address);
  console.log(`💰 Account balance: ${hre.ethers.formatEther(balance)} ETH`);
  
  // Registration results
  const results = {
    network: network,
    registryAddress: contractAddress,
    timestamp: new Date().toISOString(),
    agents: [],
    totalGasUsed: BigInt(0)
  };
  
  // Register each agent
  console.log("\n Starting agent registration...\n");
  
  for (let i = 0; i < agentConfigs.length; i++) {
    const agentConfig = agentConfigs[i];
    console.log(`[${i+1}/${agentConfigs.length}] Registering ${agentConfig.name}...`);
    
    try {
      // Generate or use existing keypair
      let keypair;
      if (agentConfig.privateKey) {
        // Use provided private key (production)
        const wallet = new hre.ethers.Wallet(agentConfig.privateKey);
        // For this example, we'll generate a placeholder public key
        // In production, derive actual public key from private key
        keypair = {
          privateKey: agentConfig.privateKey,
          publicKey: "0x04" + crypto.randomBytes(64).toString("hex")
        };
      } else {
        // Generate new keypair (development)
        keypair = generateAgentKeypair();
        console.log(`  Generated new keypair for ${agentConfig.name}`);
      }
      
      // Prepare registration data
      const publicKey = keypair.publicKey;
      const capabilities = JSON.stringify(agentConfig.capabilities);
      
      // Create signature for key ownership verification
      const keyHash = hre.ethers.keccak256(publicKey);
      const chainId = (await hre.ethers.provider.getNetwork()).chainId;
      
      const challenge = hre.ethers.keccak256(
        hre.ethers.solidityPacked(
          ["string", "uint256", "address", "address", "bytes32"],
          ["SAGE Key Registration:", chainId, contractAddress, signer.address, keyHash]
        )
      );
      
      const signature = await signer.signMessage(hre.ethers.getBytes(challenge));
      
      // Register agent
      const tx = await registry.registerAgent(
        agentConfig.did,
        agentConfig.name,
        agentConfig.description,
        agentConfig.endpoint,
        publicKey,
        capabilities,
        signature
      );
      
      console.log(`  Transaction sent: ${tx.hash}`);
      const receipt = await tx.wait();
      console.log(`   Confirmed in block ${receipt.blockNumber}`);
      
      // Get agent ID from event
      const logs = await registry.queryFilter(
        registry.filters.AgentRegistered(),
        receipt.blockNumber,
        receipt.blockNumber
      );
      
      let agentId = null;
      if (logs.length > 0) {
        agentId = logs[0].args[0];
        console.log(`  Agent ID: ${agentId}`);
      }
      
      // Save result
      results.agents.push({
        id: agentId,
        did: agentConfig.did,
        name: agentConfig.name,
        description: agentConfig.description,
        endpoint: agentConfig.endpoint,
        owner: signer.address,
        publicKey: publicKey,
        privateKey: agentConfig.privateKey ? "REDACTED" : keypair.privateKey,
        transactionHash: tx.hash,
        blockNumber: receipt.blockNumber,
        gasUsed: receipt.gasUsed.toString()
      });
      
      results.totalGasUsed += receipt.gasUsed;
      console.log(`  Gas used: ${receipt.gasUsed.toString()}\n`);
      
    } catch (error) {
      console.error(`   Failed to register ${agentConfig.name}:`, error.message);
      results.agents.push({
        did: agentConfig.did,
        name: agentConfig.name,
        error: error.message
      });
    }
  }
  
  // Save registration results
  const resultsDir = path.join(__dirname, "..", "deployments", "agents");
  if (!fs.existsSync(resultsDir)) {
    fs.mkdirSync(resultsDir, { recursive: true });
  }
  
  const resultsFile = path.join(resultsDir, `${network}-agents-${Date.now()}.json`);
  fs.writeFileSync(resultsFile, JSON.stringify(results, null, 2));
  console.log(`\n💾 Registration results saved to: ${resultsFile}`);
  
  // Generate agent credentials file for development
  if (network === "local" || network === "hardhat") {
    const credentialsFile = path.join(resultsDir, `${network}-credentials.json`);
    const credentials = results.agents
      .filter(a => !a.error)
      .reduce((acc, agent) => {
        acc[agent.did] = {
          id: agent.id,
          privateKey: agent.privateKey,
          publicKey: agent.publicKey,
          endpoint: agent.endpoint
        };
        return acc;
      }, {});
    
    fs.writeFileSync(credentialsFile, JSON.stringify(credentials, null, 2));
    console.log(` Agent credentials saved to: ${credentialsFile}`);
    console.log("  Keep this file secure and never commit to version control!");
  }
  
  // Print summary
  console.log("\n" + "=" .repeat(60));
  console.log(" Registration Summary");
  console.log("=" .repeat(60));
  
  const successful = results.agents.filter(a => !a.error);
  const failed = results.agents.filter(a => a.error);
  
  console.log(` Successful: ${successful.length}`);
  console.log(` Failed: ${failed.length}`);
  console.log(`⛽ Total gas used: ${results.totalGasUsed.toString()}`);
  
  if (successful.length > 0) {
    console.log("\n Registered Agents:");
    successful.forEach(agent => {
      console.log(`  - ${agent.name} (${agent.did})`);
      console.log(`    ID: ${agent.id}`);
      console.log(`    Endpoint: ${agent.endpoint}`);
    });
  }
  
  if (failed.length > 0) {
    console.log("\n  Failed Registrations:");
    failed.forEach(agent => {
      console.log(`  - ${agent.name}: ${agent.error}`);
    });
  }
  
  // Verify registrations
  if (successful.length > 0 && options.verify !== false) {
    console.log("\n Verifying registrations...");
    for (const agent of successful) {
      try {
        const agentData = await registry.getAgent(agent.id);
        console.log(`   ${agent.name}: Active=${agentData.active}`);
      } catch (error) {
        console.log(`   ${agent.name}: Failed to verify`);
      }
    }
  }
  
  return results;
}

// Execute if run directly
if (require.main === module) {
  // Parse command line arguments
  const args = process.argv.slice(2);
  const options = {};
  
  for (let i = 0; i < args.length; i += 2) {
    const key = args[i].replace("--", "");
    const value = args[i + 1];
    options[key] = value;
  }
  
  registerAgents(options)
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(" Registration failed:", error);
      process.exit(1);
    });
}

module.exports = registerAgents;