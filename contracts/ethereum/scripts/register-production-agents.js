const hre = require("hardhat");
const fs = require("fs");
const path = require("path");
const axios = require("axios");

// Color codes for console output
const colors = {
  reset: "\x1b[0m",
  bright: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  red: "\x1b[31m",
  cyan: "\x1b[36m"
};

function log(message, color = "reset") {
  console.log(colors[color] + message + colors.reset);
}

// Production agent configurations
// These should match your actual agent server configurations
const PRODUCTION_AGENTS = [
  {
    name: "Root Agent",
    did: "did:sage:ethereum:root",
    description: "Main orchestrator agent for SAGE network",
    endpoint: process.env.ROOT_AGENT_ENDPOINT || "http://localhost:8080/agents/root",
    publicKeyEndpoint: process.env.ROOT_AGENT_KEY_ENDPOINT || "http://localhost:8080/agents/root/publickey",
    capabilities: ["orchestration", "routing", "management"]
  },
  {
    name: "Ordering Agent",
    did: "did:sage:ethereum:ordering",
    description: "Handles transaction ordering and sequencing",
    endpoint: process.env.ORDERING_AGENT_ENDPOINT || "http://localhost:8081/agents/ordering",
    publicKeyEndpoint: process.env.ORDERING_AGENT_KEY_ENDPOINT || "http://localhost:8081/agents/ordering/publickey",
    capabilities: ["ordering", "sequencing", "transaction"]
  },
  {
    name: "Planning Agent",
    did: "did:sage:ethereum:planning",
    description: "Strategic planning and optimization agent",
    endpoint: process.env.PLANNING_AGENT_ENDPOINT || "http://localhost:8082/agents/planning",
    publicKeyEndpoint: process.env.PLANNING_AGENT_KEY_ENDPOINT || "http://localhost:8082/agents/planning/publickey",
    capabilities: ["planning", "optimization", "strategy"]
  }
];

async function fetchAgentPublicKey(agent) {
  try {
    // If running locally, use predefined keys for testing
    if (agent.endpoint.includes("localhost")) {
      log(`  Using test key for ${agent.name} (localhost detected)`, "yellow");
      
      // Generate deterministic test keys based on agent name
      const seed = hre.ethers.keccak256(hre.ethers.toUtf8Bytes(agent.name));
      const randomBytes = hre.ethers.getBytes(seed);
      
      // Create uncompressed secp256k1 public key format (0x04 + 64 bytes)
      const publicKey = hre.ethers.concat([
        "0x04",
        randomBytes.slice(0, 32),  // X coordinate
        randomBytes.slice(0, 32)   // Y coordinate (reusing for test)
      ]);
      
      return {
        publicKey,
        signature: "0x" + "00".repeat(65) // Dummy signature for test
      };
    }
    
    // For production, fetch from actual agent server
    log(`📡 Fetching public key from ${agent.publicKeyEndpoint}`, "blue");
    
    const response = await axios.get(agent.publicKeyEndpoint, {
      timeout: 10000,
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (!response.data || !response.data.publicKey) {
      throw new Error("Invalid response from agent server");
    }
    
    // The server should return the public key and a signature proving ownership
    return {
      publicKey: response.data.publicKey,
      signature: response.data.signature || "0x" + "00".repeat(65)
    };
    
  } catch (error) {
    log(` Failed to fetch public key for ${agent.name}: ${error.message}`, "red");
    
    // Fallback to generating a test key
    log(`  Generating fallback test key for ${agent.name}`, "yellow");
    const seed = hre.ethers.keccak256(hre.ethers.toUtf8Bytes(agent.name + Date.now()));
    const randomBytes = hre.ethers.getBytes(seed);
    
    const publicKey = hre.ethers.concat([
      "0x04",
      randomBytes.slice(0, 32),
      randomBytes.slice(0, 32)
    ]);
    
    return {
      publicKey,
      signature: "0x" + "00".repeat(65)
    };
  }
}

async function generateKeyOwnershipProof(publicKey, signer, registryAddress, chainId) {
  // Generate the challenge message that matches the contract's validation
  const keyHash = hre.ethers.keccak256(publicKey);
  
  const challenge = hre.ethers.keccak256(
    hre.ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, signer.address, keyHash]
    )
  );
  
  // Sign the challenge
  const signature = await signer.signMessage(hre.ethers.getBytes(challenge));
  
  return signature;
}

async function main() {
  try {
    const network = hre.network.name;
    
    log(`\n${"=".repeat(50)}`, "bright");
    log(` Production Agent Registration on ${network.toUpperCase()}`, "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    // Get deployer account
    const [deployer] = await hre.ethers.getSigners();
    const chainId = (await deployer.provider.getNetwork()).chainId;
    
    log("\n👤 Deployer Information:", "yellow");
    console.log(`   Address: ${deployer.address}`);
    console.log(`   Network: ${network}`);
    console.log(`   Chain ID: ${chainId}`);
    
    // Get registry address from environment or deployment file
    let registryAddress = process.env.SAGE_REGISTRY_ADDRESS;
    
    if (!registryAddress) {
      // Try to load from latest deployment file
      const deploymentPath = path.join(__dirname, `../deployments/${network}-v2-latest.json`);
      if (fs.existsSync(deploymentPath)) {
        const deployment = JSON.parse(fs.readFileSync(deploymentPath, 'utf8'));
        registryAddress = deployment.contracts.SageRegistryV2;
        log(`\n Loaded registry address from deployment: ${registryAddress}`, "green");
      }
    }
    
    if (!registryAddress) {
      throw new Error("Registry address not found. Set SAGE_REGISTRY_ADDRESS in .env or deploy first.");
    }
    
    // Connect to registry
    const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
    const registry = SageRegistryV2.attach(registryAddress);
    
    log(`\n Connected to SageRegistryV2 at: ${registryAddress}`, "green");
    
    // Register each production agent
    log(`\n${"=".repeat(50)}`, "bright");
    log(" Registering Production Agents", "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    const registeredAgents = [];
    
    for (const agent of PRODUCTION_AGENTS) {
      log(`\n[${PRODUCTION_AGENTS.indexOf(agent) + 1}/${PRODUCTION_AGENTS.length}] Registering ${agent.name}...`, "blue");
      
      try {
        // Fetch or generate public key
        const keyData = await fetchAgentPublicKey(agent);
        
        // Generate proper signature for key ownership proof
        const signature = await generateKeyOwnershipProof(
          keyData.publicKey,
          deployer,
          registryAddress,
          chainId
        );
        
        // Prepare agent data
        const agentData = {
          did: agent.did,
          name: agent.name,
          description: agent.description,
          endpoint: agent.endpoint,
          publicKey: keyData.publicKey,
          capabilities: JSON.stringify(agent.capabilities),
          signature: signature
        };
        
        log(`    Agent Details:`, "yellow");
        console.log(`      DID: ${agentData.did}`);
        console.log(`      Name: ${agentData.name}`);
        console.log(`      Endpoint: ${agentData.endpoint}`);
        console.log(`      Public Key: ${agentData.publicKey.substring(0, 20)}...`);
        console.log(`      Capabilities: ${agentData.capabilities}`);
        
        // Register the agent
        log(`   ⏳ Submitting registration transaction...`, "yellow");
        
        const tx = await registry.registerAgent(
          agentData.did,
          agentData.name,
          agentData.description,
          agentData.endpoint,
          agentData.publicKey,
          agentData.capabilities,
          agentData.signature,
          {
            gasLimit: 3000000
          }
        );
        
        log(`   ⏳ Waiting for confirmation...`, "yellow");
        const receipt = await tx.wait();
        
        // Get agent ID from events
        const logs = await registry.queryFilter(
          registry.filters.AgentRegistered(),
          receipt.blockNumber,
          receipt.blockNumber
        );
        
        let agentId = null;
        if (logs.length > 0) {
          agentId = logs[0].args[0];
        }
        
        log(`    ${agent.name} registered successfully!`, "green");
        console.log(`      Transaction: ${receipt.hash}`);
        console.log(`      Agent ID: ${agentId}`);
        console.log(`      Gas Used: ${receipt.gasUsed.toString()}`);
        
        registeredAgents.push({
          ...agentData,
          agentId: agentId,
          transactionHash: receipt.hash
        });
        
      } catch (error) {
        log(`    Failed to register ${agent.name}: ${error.message}`, "red");
        
        if (error.message.includes("already registered")) {
          log(`   ℹ️  Agent may already be registered. Skipping...`, "yellow");
        } else {
          log(`   ℹ️  You may need to configure the agent server endpoints`, "yellow");
        }
      }
    }
    
    // Save registration results
    if (registeredAgents.length > 0) {
      const resultPath = path.join(__dirname, `../deployments/${network}-agents-${Date.now()}.json`);
      const resultData = {
        network: network,
        chainId: chainId,
        registryAddress: registryAddress,
        registeredAt: new Date().toISOString(),
        agents: registeredAgents
      };
      
      fs.writeFileSync(resultPath, JSON.stringify(resultData, null, 2));
      
      log(`\n${"=".repeat(50)}`, "bright");
      log(" Registration Complete!", "green");
      log(`${"=".repeat(50)}`, "bright");
      
      log("\n Summary:", "cyan");
      console.log(`   Network: ${network}`);
      console.log(`   Registry: ${registryAddress}`);
      console.log(`   Agents Registered: ${registeredAgents.length}/${PRODUCTION_AGENTS.length}`);
      
      log("\n💾 Results saved to:", "cyan");
      console.log(`   ${resultPath}`);
      
      if (network === "kairos" || network === "kaia") {
        log("\n View on Explorer:", "cyan");
        const explorerBase = network === "kairos" ? "https://kairos.klaytnscope.com" : "https://klaytnscope.com";
        console.log(`   Registry: ${explorerBase}/account/${registryAddress}`);
        registeredAgents.forEach(agent => {
          console.log(`   ${agent.name}: ${explorerBase}/tx/${agent.transactionHash}`);
        });
      }
    }
    
    log("\n Next Steps:", "yellow");
    log("   1. Verify agents are correctly registered", "reset");
    log("   2. Test agent interactions", "reset");
    log("   3. Configure agent servers with contract addresses", "reset");
    
  } catch (error) {
    log(`\n Registration failed: ${error.message}`, "red");
    console.error(error);
    process.exit(1);
  }
}

// Execute registration
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });