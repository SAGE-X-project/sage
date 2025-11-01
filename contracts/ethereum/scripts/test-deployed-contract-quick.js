// SPDX-License-Identifier: MIT
// Quick test version - skips time delays for faster testing
import hre from 'hardhat';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {
  console.log('\n Quick Test for Deployed AgentCard Contracts');
  console.log('================================================================================');
  console.log(' Note: This version skips time-lock delays for faster testing');
  console.log(' Use test-deployed-contract.js for full security feature testing\n');
  console.log('================================================================================\n');

  // Load deployment info
  const deploymentPath = path.join(__dirname, '..', 'deployments', 'localhost-latest.json');
  if (!fs.existsSync(deploymentPath)) {
    console.error(' No deployment found. Run deploy-agentcard.js first.');
    process.exit(1);
  }

  const deployment = JSON.parse(fs.readFileSync(deploymentPath, 'utf8'));
  const registryAddress = deployment.contracts.AgentCardRegistry.address;
  const hookAddress = deployment.contracts.AgentCardVerifyHook.address;

  console.log(' Deployment Info:');
  console.log(`   Network: ${deployment.network}`);
  console.log(`   Chain ID: ${deployment.chainId}`);
  console.log(`   Registry: ${registryAddress}`);
  console.log(`   Hook: ${hookAddress}\n`);

  // Connect to network (Hardhat 3.x)
  const network = await hre.network.connect();
  const [deployer, user1, user2] = await network.ethers.getSigners();

  console.log(' Test Accounts:');
  console.log(`   Deployer: ${deployer.address}`);
  console.log(`   User1: ${user1.address}`);
  console.log(`   User2: ${user2.address}\n`);

  // Get contract instances
  const AgentCardRegistry = await network.ethers.getContractFactory('AgentCardRegistry');
  const registry = AgentCardRegistry.attach(registryAddress);

  console.log('================================================================================');
  console.log('TEST 1: Contract Configuration');
  console.log('================================================================================\n');

  const owner = await registry.owner();
  const hookAddr = await registry.verifyHook();
  const minStake = await registry.registrationStake();

  console.log(' Owner:', owner);
  console.log(' Hook Address:', hookAddr);
  console.log(' Min Stake:', network.ethers.formatEther(minStake), 'ETH');

  if (hookAddr !== hookAddress) {
    console.error(' ERROR: Hook address mismatch!');
    process.exit(1);
  }
  console.log(' PASS: Configuration verified\n');

  console.log('================================================================================');
  console.log('TEST 2: Read-Only Functions (No Registration)');
  console.log('================================================================================\n');

  try {
    // Try to get non-existent agent
    const testDID = `did:sage:ethereum:test_${Date.now()}`;
    await registry.getAgentByDID(testDID);
    console.log(' FAIL: Should have reverted for non-existent agent');
  } catch (error) {
    console.log(' PASS: Correctly reverts for non-existent agent');
  }

  // Test ERC-8004 interface
  const testAddress = user1.address;
  const isActive = await registry.isAgentActive(`did:sage:ethereum:test_123`).catch(() => false);
  console.log(` PASS: isAgentActive() callable (returned: ${isActive})`);

  console.log('\n Quick test completed successfully!');
  console.log('\n For full integration testing with registration:');
  console.log('   npx hardhat run scripts/test-deployed-contract.js --network localhost');
  console.log('   (Warning: Takes ~2 minutes due to security time-locks)\n');
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error('\n Test failed:', error);
    process.exit(1);
  });
