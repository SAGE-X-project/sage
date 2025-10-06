const hre = require('hardhat');
const fs = require('fs');
const path = require('path');

// Configuration
const CONFIG = {
  local: {
    name: 'Local Network',
    testAgents: true,
    verifyContracts: false,
  },
  kairos: {
    name: 'Kaia Testnet (Kairos)',
    testAgents: true,
    verifyContracts: true,
    explorerUrl: 'https://kairos.kaiascan.io',
  },
  kaia: {
    name: 'Kaia Mainnet',
    testAgents: false,
    verifyContracts: true,
    explorerUrl: 'https://kaiascan.io',
  },
  cypress: {
    name: 'Kaia Mainnet (Cypress)',
    testAgents: false,
    verifyContracts: true,
    explorerUrl: 'https://kaiascan.io',
  },
};

async function main() {
  const network = hre.network.name;
  const config = CONFIG[network] || CONFIG.local;

  console.log('\n SAGE Unified Deployment Script');
  console.log('='.repeat(60));
  console.log(` Network: ${config.name} (${network})`);
  console.log(` Chain ID: ${hre.network.config.chainId || 31337}`);
  console.log('='.repeat(60));

  // Check network connection
  try {
    const blockNumber = await hre.ethers.provider.getBlockNumber();
    console.log(` Current block: ${blockNumber}`);
  } catch (error) {
    console.error('\n Network connection failed:', error.message);
    console.log('\n Solutions:');
    console.log('  1. Check if Hardhat node is running: npx hardhat node');
    console.log('  2. Check if port 8545 is open: lsof -i:8545');
    console.log('  3. Check network settings: hardhat.config.js');
    process.exit(1);
  }

  // Get deployer account
  const [deployer] = await hre.ethers.getSigners();
  console.log('\n Deployer Account:', deployer.address);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log(' Balance:', hre.ethers.formatEther(balance), 'ETH');

  // Deployment info object
  const deploymentInfo = {
    network: network,
    chainId: hre.network.config.chainId || 31337,
    deployer: deployer.address,
    timestamp: new Date().toISOString(),
    contracts: {},
    agents: [],
    gasUsed: BigInt(0),
    configuration: {},
  };

  try {
    // ==========================================
    // 1. Deploy SageRegistryV2 (or Test version for local)
    // ==========================================
    const isLocal = network === 'local' || network === 'localhost' || network === 'hardhat';
    const contractName = isLocal ? 'SageRegistryTest' : 'SageRegistryV2';

    console.log(`\n Step 1: Deploying ${contractName}...`);
    const RegistryContract = await hre.ethers.getContractFactory(contractName);
    const sageRegistry = await RegistryContract.deploy();
    await sageRegistry.waitForDeployment();

    const registryAddress = await sageRegistry.getAddress();
    const registryDeployTx = sageRegistry.deploymentTransaction();

    deploymentInfo.contracts.SageRegistryV2 = {
      address: registryAddress,
      transactionHash: registryDeployTx.hash,
      blockNumber: registryDeployTx.blockNumber,
      gasUsed: registryDeployTx.gasUsed?.toString() || '0',
    };
    deploymentInfo.gasUsed += registryDeployTx.gasUsed || BigInt(0);

    console.log(' SageRegistryV2 deployed to:', registryAddress);

    // ==========================================
    // 2. Deploy SageVerificationHook
    // ==========================================
    console.log('\n Step 2: Deploying SageVerificationHook...');
    const SageVerificationHook = await hre.ethers.getContractFactory('SageVerificationHook');
    const verificationHook = await SageVerificationHook.deploy();
    await verificationHook.waitForDeployment();

    const hookAddress = await verificationHook.getAddress();
    const hookDeployTx = verificationHook.deploymentTransaction();

    deploymentInfo.contracts.SageVerificationHook = {
      address: hookAddress,
      transactionHash: hookDeployTx.hash,
      blockNumber: hookDeployTx.blockNumber,
      gasUsed: hookDeployTx.gasUsed?.toString() || '0',
    };
    deploymentInfo.gasUsed += hookDeployTx.gasUsed || BigInt(0);

    console.log(' SageVerificationHook deployed to:', hookAddress);

    // ==========================================
    // 3. Configure Hooks
    // ==========================================
    console.log('\n Step 3: Configuring hooks...');

    let tx = await sageRegistry.setBeforeRegisterHook(hookAddress);
    let receipt = await tx.wait();
    deploymentInfo.configuration.beforeRegisterHook = {
      address: hookAddress,
      transactionHash: tx.hash,
      gasUsed: receipt.gasUsed.toString(),
    };
    deploymentInfo.gasUsed += receipt.gasUsed;
    console.log(' BeforeRegisterHook configured');

    tx = await sageRegistry.setAfterRegisterHook(hookAddress);
    receipt = await tx.wait();
    deploymentInfo.configuration.afterRegisterHook = {
      address: hookAddress,
      transactionHash: tx.hash,
      gasUsed: receipt.gasUsed.toString(),
    };
    deploymentInfo.gasUsed += receipt.gasUsed;
    console.log(' AfterRegisterHook configured');

    // ==========================================
    // 4. Register Test Agents (if applicable)
    // ==========================================
    if (config.testAgents) {
      console.log('\n Step 4: Registering test agents...');

      const agents = [
        {
          did: `did:sage:${network}:root`,
          name: 'Root Agent',
          description: 'Main routing agent for SAGE system',
          endpoint: network === 'local' ? 'http://localhost:3001' : `https://root.sage-${network}.ai`,
          capabilities: JSON.stringify(['routing', 'management', 'verification']),
        },
        {
          did: `did:sage:${network}:ordering`,
          name: 'Ordering Agent',
          description: 'Handles order processing and management',
          endpoint: network === 'local' ? 'http://localhost:3002' : `https://ordering.sage-${network}.ai`,
          capabilities: JSON.stringify(['ordering', 'payment', 'tracking']),
        },
        {
          did: `did:sage:${network}:planning`,
          name: 'Planning Agent',
          description: 'Handles travel and event planning',
          endpoint: network === 'local' ? 'http://localhost:3003' : `https://planning.sage-${network}.ai`,
          capabilities: JSON.stringify(['booking', 'scheduling', 'recommendation']),
        },
      ];

      // Get multiple signers for testing (to avoid cooldown)
      const signers = await hre.ethers.getSigners();

      for (let i = 0; i < agents.length; i++) {
        const agent = agents[i];
        const signer = signers[i] || deployer; // Use different signer for each agent if available

        console.log(`  Registering ${agent.name} with ${signer.address}...`);

        // Generate a test public key (for production, use actual agent keys)
        const randomKey = hre.ethers.randomBytes(64);
        const publicKey = hre.ethers.concat(['0x04', randomKey]);

        // Create signature for key ownership verification
        const keyHash = hre.ethers.keccak256(publicKey);

        let signature;
        if (isLocal) {
          // For test mode, use simplified signature
          const messageHash = hre.ethers.keccak256(
            hre.ethers.solidityPacked(['string', 'address', 'bytes32'], ['TEST_MODE:', signer.address, keyHash])
          );
          signature = await signer.signMessage(hre.ethers.getBytes(messageHash));
        } else {
          // For production, use full validation
          const chainId = (await hre.ethers.provider.getNetwork()).chainId;
          const challenge = hre.ethers.keccak256(
            hre.ethers.solidityPacked(
              ['string', 'uint256', 'address', 'address', 'bytes32'],
              ['SAGE Key Registration:', chainId, registryAddress, deployer.address, keyHash]
            )
          );
          signature = await deployer.signMessage(hre.ethers.getBytes(challenge));
        }

        // Register agent
        tx = await sageRegistry
          .connect(signer)
          .registerAgent(agent.did, agent.name, agent.description, agent.endpoint, publicKey, agent.capabilities, signature);

        receipt = await tx.wait();
        deploymentInfo.gasUsed += receipt.gasUsed;

        // Get agent ID from event
        const logs = await sageRegistry.queryFilter(sageRegistry.filters.AgentRegistered(), receipt.blockNumber, receipt.blockNumber);

        if (logs.length > 0) {
          const agentId = logs[0].args[0];
          deploymentInfo.agents.push({
            id: agentId,
            ...agent,
            owner: signer.address,
            transactionHash: tx.hash,
            gasUsed: receipt.gasUsed.toString(),
          });
          console.log(`   ${agent.name} registered with ID: ${agentId}`);
        }
      }
    }

    // ==========================================
    // 5. Save Deployment Info
    // ==========================================
    console.log('\n💾 Step 5: Saving deployment information...');

    // Calculate total gas cost
    const gasPrice = await hre.ethers.provider.getFeeData();
    const totalGasUsed = deploymentInfo.gasUsed.toString();
    const estimatedCost = hre.ethers.formatEther(deploymentInfo.gasUsed * (gasPrice.gasPrice || BigInt(0)));

    deploymentInfo.gasUsed = totalGasUsed;
    deploymentInfo.estimatedCostETH = estimatedCost;

    // Add explorer URLs if available
    if (config.explorerUrl) {
      deploymentInfo.explorer = {
        registry: `${config.explorerUrl}/address/${registryAddress}`,
        hook: `${config.explorerUrl}/address/${hookAddress}`,
      };
    }

    // Save to file
    const deploymentsDir = path.join(__dirname, '..', 'deployments');
    if (!fs.existsSync(deploymentsDir)) {
      fs.mkdirSync(deploymentsDir, { recursive: true });
    }

    const deploymentFile = path.join(deploymentsDir, `${network}.json`);
    fs.writeFileSync(
      deploymentFile,
      JSON.stringify(
        deploymentInfo,
        (key, value) => {
          if (typeof value === 'bigint') {
            return value.toString();
          }
          return value;
        },
        2
      )
    );
    console.log(` Deployment info saved to: deployments/${network}.json`);

    // Save latest deployment for easy access
    const latestFile = path.join(deploymentsDir, 'latest.json');
    fs.writeFileSync(
      latestFile,
      JSON.stringify(
        deploymentInfo,
        (key, value) => {
          if (typeof value === 'bigint') {
            return value.toString();
          }
          return value;
        },
        2
      )
    );

    // ==========================================
    // 6. Generate Environment Variables
    // ==========================================
    console.log('\n Step 6: Generating environment variables...');

    const envContent = `# SAGE Deployment - ${config.name}
# Generated: ${deploymentInfo.timestamp}
# Network: ${network} (Chain ID: ${deploymentInfo.chainId})

# Contract Addresses
SAGE_REGISTRY_ADDRESS=${registryAddress}
SAGE_VERIFICATION_HOOK_ADDRESS=${hookAddress}

# Network Configuration
SAGE_NETWORK=${network}
SAGE_CHAIN_ID=${deploymentInfo.chainId}
SAGE_DEPLOYER=${deployer.address}

# For Go applications
DEPLOYED_CONTRACT_ADDRESS=${registryAddress}
`;

    const envFile = path.join(deploymentsDir, `${network}.env`);
    fs.writeFileSync(envFile, envContent);
    console.log(` Environment variables saved to: deployments/${network}.env`);

    // ==========================================
    // 7. Verify Contracts (if applicable)
    // ==========================================
    if (config.verifyContracts && network !== 'local' && network !== 'hardhat') {
      console.log('\n Step 7: Verifying contracts on explorer...');

      try {
        await hre.run('verify:verify', {
          address: registryAddress,
          constructorArguments: [],
        });
        console.log(' SageRegistryV2 verified');
      } catch (error) {
        console.log('  SageRegistryV2 verification failed:', error.message);
      }

      try {
        await hre.run('verify:verify', {
          address: hookAddress,
          constructorArguments: [],
        });
        console.log(' SageVerificationHook verified');
      } catch (error) {
        console.log('  SageVerificationHook verification failed:', error.message);
      }
    }

    // ==========================================
    // 8. Print Summary
    // ==========================================
    console.log('\n' + '='.repeat(60));
    console.log(' DEPLOYMENT SUCCESSFUL!');
    console.log('='.repeat(60));
    console.log('\n Deployment Summary:');
    console.log(`  Network: ${config.name} (${network})`);
    console.log(`  Chain ID: ${deploymentInfo.chainId}`);
    console.log(`  Deployer: ${deployer.address}`);
    console.log(`  Total Gas Used: ${totalGasUsed}`);
    console.log(`  Estimated Cost: ${estimatedCost} ETH`);

    console.log('\n Contract Addresses:');
    console.log(`  SageRegistryV2: ${registryAddress}`);
    console.log(`  SageVerificationHook: ${hookAddress}`);

    if (deploymentInfo.agents.length > 0) {
      console.log('\n Registered Agents:');
      deploymentInfo.agents.forEach((agent) => {
        console.log(`  - ${agent.name} (ID: ${agent.id})`);
      });
    }

    if (config.explorerUrl) {
      console.log('\n Explorer Links:');
      console.log(`  Registry: ${deploymentInfo.explorer.registry}`);
      console.log(`  Hook: ${deploymentInfo.explorer.hook}`);
    }

    console.log('\n Next Steps:');
    console.log('  1. Copy environment variables:');
    console.log(`     cp deployments/${network}.env ../../.env`);
    console.log('  2. Update Go applications to use new addresses');
    console.log('  3. Test the deployment:');
    console.log(`     npx hardhat run scripts/interact-local.js --network ${network}`);

    return deploymentInfo;
  } catch (error) {
    console.error('\n Deployment failed:', error);

    // Save partial deployment info if available
    if (Object.keys(deploymentInfo.contracts).length > 0) {
      const failedFile = path.join(__dirname, '..', 'deployments', `${network}-failed-${Date.now()}.json`);
      fs.writeFileSync(
        failedFile,
        JSON.stringify(
          deploymentInfo,
          (key, value) => {
            if (typeof value === 'bigint') {
              return value.toString();
            }
            return value;
          },
          2
        )
      );
      console.log(`\n Partial deployment info saved to: ${failedFile}`);
    }

    process.exit(1);
  }
}

// Execute if run directly
if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });
}

module.exports = main;
