const hre = require('hardhat');
const fs = require('fs');
const path = require('path');

async function verifyDeployment() {
  const network = hre.network.name;

  console.log('\n SAGE Deployment Verification Script');
  console.log('='.repeat(60));
  console.log(`📍 Network: ${network}`);

  try {
    // 1. Load deployment information
    const deploymentFile = path.join(__dirname, '..', 'deployments', `${network}.json`);
    if (!fs.existsSync(deploymentFile)) {
      console.error(` Deployment information file not found: ${deploymentFile}`);
      console.log(' Please run npm run deploy:unified first');
      return false;
    }

    const deployment = JSON.parse(fs.readFileSync(deploymentFile, 'utf8'));
    console.log(` Deployment information loaded: ${deploymentFile}`);

    // 2. Check contract addresses
    const registryAddress = deployment.contracts.SageRegistryV2?.address;
    const hookAddress = deployment.contracts.SageVerificationHook?.address;

    if (!registryAddress || !hookAddress) {
      console.error(' Contract addresses not found');
      return false;
    }

    console.log('\n Contract Addresses:');
    console.log(`  Registry: ${registryAddress}`);
    console.log(`  Hook: ${hookAddress}`);

    // 3. Connect and verify contracts
    const SageRegistry = await hre.ethers.getContractFactory(
      network === 'hardhat' || network === 'localhost' ? 'SageRegistryTest' : 'SageRegistryV2'
    );
    const registry = SageRegistry.attach(registryAddress);

    const SageVerificationHook = await hre.ethers.getContractFactory('SageVerificationHook');
    const hook = SageVerificationHook.attach(hookAddress);

    // 4. Check basic information
    console.log('\n Contract Status Check:');

    // Check Owner
    const owner = await registry.owner();
    console.log(`  Owner: ${owner}`);

    // Check Hook settings (direct call as public variables)
    const beforeHook = await registry.beforeRegisterHook();
    const afterHook = await registry.afterRegisterHook();
    console.log(`  Before Hook: ${beforeHook}`);
    console.log(`  After Hook: ${afterHook}`);

    // Check number of agents
    const agentCount = await registry.getAgentCount();
    console.log(`  Registered agents: ${agentCount}`);

    // 5. Check agent information
    if (deployment.agents && deployment.agents.length > 0) {
      console.log('\n Registered Agents Check:');

      for (const agentInfo of deployment.agents) {
        try {
          const agent = await registry.getAgent(agentInfo.id);
          console.log(`\n  ${agentInfo.name}:`);
          console.log(`    ID: ${agentInfo.id}`);
          console.log(`    DID: ${agent.did}`);
          console.log(`    Active: ${agent.active}`);
          console.log(`    Owner: ${agent.owner}`);
          console.log(`    Endpoint: ${agent.endpoint}`);

          // Public key information
          const keyInfo = await registry.getAgentPublicKey(agentInfo.id);
          console.log(`    Public key exists: ${keyInfo.length > 0 ? '' : ''}`);
        } catch (error) {
          console.log(`     Agent query failed: ${error.message}`);
        }
      }
    }

    // 6. Function tests
    console.log('\n Function Tests:');

    // Query agent by DID
    if (deployment.agents && deployment.agents.length > 0) {
      const testDid = deployment.agents[0].did;
      try {
        const agentId = await registry.getAgentByDID(testDid);
        console.log(`   DID query successful: ${testDid}`);
      } catch (error) {
        console.log(`   DID query failed: ${error.message}`);
      }
    }

    // 7. Check events
    console.log('\n Recent Events:');

    // Query events from recent blocks
    const currentBlock = await hre.ethers.provider.getBlockNumber();
    const events = await registry.queryFilter(registry.filters.AgentRegistered(), Math.max(0, currentBlock - 100), currentBlock);

    console.log(`  Registration events: ${events.length}`);
    events.slice(-3).forEach((event, i) => {
      console.log(`  Event ${i + 1}: Block ${event.blockNumber}, Agent ID: ${event.args[0].slice(0, 10)}...`);
    });

    // 8. Check gas usage
    if (deployment.gasUsed) {
      console.log('\n⛽ Gas Usage:');
      console.log(`  Total gas: ${deployment.gasUsed}`);
      console.log(`  Estimated cost: ${deployment.estimatedCostETH || 'N/A'} ETH`);
    }

    console.log('\n' + '='.repeat(60));
    console.log(' Verification Complete!');
    console.log('='.repeat(60));

    return true;
  } catch (error) {
    console.error('\n Verification Failed:', error);
    return false;
  }
}

// Additional verification functions

async function testAgentQuery(registryAddress, agentId) {
  const SageRegistry = await hre.ethers.getContractFactory('SageRegistryV2');
  const registry = SageRegistry.attach(registryAddress);

  console.log(`\n Agent Details Query: ${agentId}`);

  try {
    const agent = await registry.getAgent(agentId);
    console.log('  Basic Information:');
    console.log(`    Name: ${agent.name}`);
    console.log(`    Description: ${agent.description}`);
    console.log(`    DID: ${agent.did}`);
    console.log(`    Endpoint: ${agent.endpoint}`);
    console.log(`    Active: ${agent.active}`);

    const publicKey = await registry.getAgentPublicKey(agentId);
    console.log(`    Public key length: ${publicKey.length} bytes`);

    const capabilities = await registry.getAgentCapabilities(agentId);
    console.log(`    Capabilities: ${capabilities}`);

    return true;
  } catch (error) {
    console.error(`   Query failed: ${error.message}`);
    return false;
  }
}

async function testSignatureVerification(registryAddress) {
  console.log('\n Signature Verification Test');

  const [signer] = await hre.ethers.getSigners();

  // Generate test message
  const message = 'SAGE Protocol Test Message';
  const messageHash = hre.ethers.keccak256(hre.ethers.toUtf8Bytes(message));

  // Generate signature
  const signature = await signer.signMessage(hre.ethers.getBytes(messageHash));

  console.log(`  Signer: ${signer.address}`);
  console.log(`  Message: ${message}`);
  console.log(`  Signature: ${signature.slice(0, 20)}...`);

  // Signature verification (off-chain)
  const recoveredAddress = hre.ethers.verifyMessage(hre.ethers.getBytes(messageHash), signature);
  const isValid = recoveredAddress === signer.address;

  console.log(`  Recovered address: ${recoveredAddress}`);
  console.log(`  Verification result: ${isValid ? ' Valid' : ' Invalid'}`);

  return isValid;
}

// Main execution
if (require.main === module) {
  verifyDeployment()
    .then((success) => {
      if (!success) {
        process.exit(1);
      }

      // Run additional tests (optional)
      if (process.argv.includes('--detailed')) {
        return testSignatureVerification();
      }
    })
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });
}

module.exports = { verifyDeployment, testAgentQuery, testSignatureVerification };
