const { ethers } = require("ethers");
const fs = require("fs");

// Configuration
const RPC_URL = process.env.RPC_URL || "http://localhost:8545";
const REGISTRY_ADDRESS = process.env.CONTRACT_ADDRESS || "0xc5a5C42992dECbae36851359345FE25997F5C42d";
const ABI_PATH = "./artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json";

async function main() {
  console.log("🔍 Querying SAGE Registry Agents");
  console.log("=" .repeat(60));
  
  // Connect to network
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const abi = JSON.parse(fs.readFileSync(ABI_PATH, "utf8")).abi;
  const registry = new ethers.Contract(REGISTRY_ADDRESS, abi, provider);
  
  console.log("📍 Registry Address:", REGISTRY_ADDRESS);
  console.log("🌐 RPC URL:", RPC_URL);
  console.log();

  // Get command line arguments
  const args = process.argv.slice(2);
  const command = args[0] || "list-all";
  
  switch(command) {
    case "list-all":
      await listAllAgents(registry);
      break;
    
    case "by-owner":
      const owner = args[1];
      if (!owner) {
        console.log("❌ Please provide owner address: node query-agents.js by-owner 0x...");
        break;
      }
      await listAgentsByOwner(registry, owner);
      break;
    
    case "by-id":
      const agentId = args[1];
      if (!agentId) {
        console.log("❌ Please provide agent ID: node query-agents.js by-id 0x...");
        break;
      }
      await getAgentById(registry, agentId);
      break;
    
    case "by-did":
      const did = args[1];
      if (!did) {
        console.log("❌ Please provide DID: node query-agents.js by-did did:sage:...");
        break;
      }
      await getAgentByDID(registry, did);
      break;
    
    case "events":
      await getRegistrationEvents(registry);
      break;
    
    case "stats":
      await getRegistryStats(registry);
      break;
    
    default:
      console.log("Available commands:");
      console.log("  list-all              - List all registered agents");
      console.log("  by-owner <address>    - List agents by owner address");
      console.log("  by-id <agentId>       - Get agent by ID");
      console.log("  by-did <did>          - Get agent by DID");
      console.log("  events                - Show recent registration events");
      console.log("  stats                 - Show registry statistics");
  }
}

async function listAllAgents(registry) {
  console.log("📋 Fetching all registered agents...\n");
  
  // Get registration events to find all agents
  const filter = registry.filters.AgentRegistered();
  const events = await registry.queryFilter(filter, 0, "latest");
  
  if (events.length === 0) {
    console.log("No agents registered yet.");
    return;
  }
  
  console.log(`Found ${events.length} agent(s):\n`);
  
  for (let i = 0; i < events.length; i++) {
    const event = events[i];
    const agentId = event.args[0];
    const owner = event.args[1];
    const did = event.args[2];
    
    console.log(`${i + 1}. Agent #${i + 1}`);
    console.log("   📌 ID:", agentId);
    console.log("   👤 Owner:", owner);
    console.log("   🆔 DID:", did);
    
    // Get full agent details
    try {
      const agent = await registry.getAgent(agentId);
      console.log("   📝 Name:", agent.name);
      console.log("   📄 Description:", agent.description);
      console.log("   🌐 Endpoint:", agent.endpoint);
      console.log("   ⚡ Active:", agent.active);
      console.log("   📅 Registered:", new Date(Number(agent.registeredAt) * 1000).toLocaleString());
      
      // Check if key is valid
      const isKeyValid = await registry.isKeyValid(agent.publicKey);
      console.log("   🔑 Key Valid:", isKeyValid);
    } catch (error) {
      console.log("   ⚠️  Could not fetch full details");
    }
    console.log();
  }
}

async function listAgentsByOwner(registry, ownerAddress) {
  console.log(`📋 Fetching agents for owner: ${ownerAddress}\n`);
  
  try {
    const agentIds = await registry.getAgentsByOwner(ownerAddress);
    
    if (agentIds.length === 0) {
      console.log("No agents found for this owner.");
      return;
    }
    
    console.log(`Found ${agentIds.length} agent(s):\n`);
    
    for (let i = 0; i < agentIds.length; i++) {
      const agentId = agentIds[i];
      const agent = await registry.getAgent(agentId);
      
      console.log(`${i + 1}. ${agent.name}`);
      console.log("   📌 ID:", agentId);
      console.log("   🆔 DID:", agent.did);
      console.log("   📄 Description:", agent.description);
      console.log("   🌐 Endpoint:", agent.endpoint);
      console.log("   ⚡ Active:", agent.active);
      console.log("   🔧 Capabilities:", agent.capabilities);
      console.log("   📅 Registered:", new Date(Number(agent.registeredAt) * 1000).toLocaleString());
      console.log("   🔄 Updated:", new Date(Number(agent.updatedAt) * 1000).toLocaleString());
      console.log();
    }
  } catch (error) {
    console.log("❌ Error:", error.message);
  }
}

async function getAgentById(registry, agentId) {
  console.log(`📋 Fetching agent with ID: ${agentId}\n`);
  
  try {
    const agent = await registry.getAgent(agentId);
    
    console.log("📊 Agent Details:");
    console.log("   📝 Name:", agent.name);
    console.log("   🆔 DID:", agent.did);
    console.log("   📄 Description:", agent.description);
    console.log("   👤 Owner:", agent.owner);
    console.log("   🌐 Endpoint:", agent.endpoint);
    console.log("   🔑 Public Key:", agent.publicKey.substring(0, 20) + "...");
    console.log("   🔧 Capabilities:", agent.capabilities);
    console.log("   ⚡ Active:", agent.active);
    console.log("   📅 Registered:", new Date(Number(agent.registeredAt) * 1000).toLocaleString());
    console.log("   🔄 Updated:", new Date(Number(agent.updatedAt) * 1000).toLocaleString());
    
    // Check if owner
    const isOwner = await registry.verifyAgentOwnership(agentId, agent.owner);
    console.log("   ✅ Ownership Verified:", isOwner);
    
    // Check if key is valid
    const isKeyValid = await registry.isKeyValid(agent.publicKey);
    console.log("   🔑 Key Valid:", isKeyValid);
  } catch (error) {
    console.log("❌ Error:", error.message);
  }
}

async function getAgentByDID(registry, did) {
  console.log(`📋 Fetching agent with DID: ${did}\n`);
  
  try {
    const agent = await registry.getAgentByDID(did);
    
    console.log("📊 Agent Details:");
    console.log("   📝 Name:", agent.name);
    console.log("   📄 Description:", agent.description);
    console.log("   👤 Owner:", agent.owner);
    console.log("   🌐 Endpoint:", agent.endpoint);
    console.log("   🔧 Capabilities:", agent.capabilities);
    console.log("   ⚡ Active:", agent.active);
    console.log("   📅 Registered:", new Date(Number(agent.registeredAt) * 1000).toLocaleString());
    console.log("   🔄 Updated:", new Date(Number(agent.updatedAt) * 1000).toLocaleString());
  } catch (error) {
    console.log("❌ Error:", error.message);
  }
}

async function getRegistrationEvents(registry) {
  console.log("📋 Recent Registration Events\n");
  
  // Get last 10 blocks
  const currentBlock = await registry.provider.getBlockNumber();
  const fromBlock = Math.max(0, currentBlock - 100);
  
  const filter = registry.filters.AgentRegistered();
  const events = await registry.queryFilter(filter, fromBlock, "latest");
  
  if (events.length === 0) {
    console.log("No recent registration events.");
    return;
  }
  
  console.log(`Found ${events.length} recent registration(s):\n`);
  
  for (const event of events) {
    const block = await event.getBlock();
    console.log(`📍 Block #${event.blockNumber}`);
    console.log("   🕐 Time:", new Date(block.timestamp * 1000).toLocaleString());
    console.log("   📌 Agent ID:", event.args[0]);
    console.log("   👤 Owner:", event.args[1]);
    console.log("   🆔 DID:", event.args[2]);
    console.log("   🔥 TX Hash:", event.transactionHash);
    console.log();
  }
}

async function getRegistryStats(registry) {
  console.log("📊 Registry Statistics\n");
  
  // Get total agents from events
  const filter = registry.filters.AgentRegistered();
  const events = await registry.queryFilter(filter, 0, "latest");
  
  console.log("📈 Overall Stats:");
  console.log("   Total Registrations:", events.length);
  
  // Count by owner
  const ownerCount = {};
  const activeCount = { active: 0, inactive: 0 };
  
  for (const event of events) {
    const owner = event.args[1];
    ownerCount[owner] = (ownerCount[owner] || 0) + 1;
    
    // Check if active
    try {
      const agent = await registry.getAgent(event.args[0]);
      if (agent.active) {
        activeCount.active++;
      } else {
        activeCount.inactive++;
      }
    } catch (e) {
      activeCount.inactive++;
    }
  }
  
  console.log("   Unique Owners:", Object.keys(ownerCount).length);
  console.log("   Active Agents:", activeCount.active);
  console.log("   Inactive Agents:", activeCount.inactive);
  console.log();
  
  console.log("👥 Agents per Owner:");
  for (const [owner, count] of Object.entries(ownerCount)) {
    console.log(`   ${owner}: ${count} agent(s)`);
  }
  
  // Get hooks
  console.log();
  console.log("🔧 Hook Configuration:");
  const beforeHook = await registry.beforeRegisterHook();
  const afterHook = await registry.afterRegisterHook();
  console.log("   Before Hook:", beforeHook || "Not set");
  console.log("   After Hook:", afterHook || "Not set");
}

// Run the script
main().catch((error) => {
  console.error("❌ Error:", error);
  process.exit(1);
});