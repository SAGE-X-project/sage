const hre = require('hardhat');
const fs = require('fs');
const path = require('path');

/**
 * Deploy ERC-8004 Contracts to Sepolia Testnet
 *
 * This script deploys the complete ERC-8004 Trustless Agents implementation:
 * 1. ERC8004IdentityRegistry (adapter for SageRegistryV2)
 * 2. ERC8004ReputationRegistry
 * 3. ERC8004ValidationRegistry
 */

async function main() {
  const network = hre.network.name;

  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║      ERC-8004 Trustless Agents Deployment Script          ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log(`\n Network: ${network}`);
  console.log(` Chain ID: ${hre.network.config.chainId || 'Unknown'}`);
  console.log('════════════════════════════════════════════════════════════\n');

  // Check if this is Sepolia
  if (network !== 'sepolia') {
    console.warn('⚠️  Warning: This script is designed for Sepolia testnet');
    console.log(`   Current network: ${network}`);
    console.log('   Continue? (Ctrl+C to cancel)');
  }

  // Check network connection
  try {
    const blockNumber = await hre.ethers.provider.getBlockNumber();
    console.log(`✓ Connected to network (block: ${blockNumber})`);
  } catch (error) {
    console.error('\n❌ Network connection failed:', error.message);
    process.exit(1);
  }

  // Get deployer account
  const [deployer] = await hre.ethers.getSigners();
  console.log(`\n📝 Deployer Account: ${deployer.address}`);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log(`💰 Balance: ${hre.ethers.formatEther(balance)} ETH`);

  // Check if deployer has sufficient balance
  const minBalance = hre.ethers.parseEther('0.1'); // Minimum 0.1 ETH
  if (balance < minBalance) {
    console.error('\n❌ Insufficient balance for deployment');
    console.log(`   Required: ~0.1 ETH`);
    console.log(`   Current: ${hre.ethers.formatEther(balance)} ETH`);
    console.log(`   Get Sepolia ETH from: https://sepoliafaucet.com/`);
    process.exit(1);
  }

  // Deployment info object
  const deploymentInfo = {
    network: network,
    chainId: hre.network.config.chainId,
    deployer: deployer.address,
    timestamp: new Date().toISOString(),
    contracts: {},
    gasUsed: BigInt(0),
  };

  try {
    // ==========================================
    // Step 0: Check for existing SageRegistryV2
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 0: Checking for existing SageRegistryV2...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    let sageRegistryAddress;
    let sageRegistry;

    // Check if there's an existing deployment
    const deploymentsDir = path.join(__dirname, '..', 'deployments');
    const deploymentFile = path.join(deploymentsDir, `${network}.json`);

    if (fs.existsSync(deploymentFile)) {
      const existingDeployment = JSON.parse(fs.readFileSync(deploymentFile, 'utf8'));
      if (existingDeployment.contracts?.SageRegistryV2?.address) {
        sageRegistryAddress = existingDeployment.contracts.SageRegistryV2.address;
        console.log(`✓ Found existing SageRegistryV2 at: ${sageRegistryAddress}`);

        // Verify contract exists
        const code = await hre.ethers.provider.getCode(sageRegistryAddress);
        if (code === '0x') {
          console.log('⚠️  Contract not found at address, will deploy new one');
          sageRegistryAddress = null;
        } else {
          sageRegistry = await hre.ethers.getContractAt('SageRegistryV2', sageRegistryAddress);
        }
      }
    }

    // Deploy SageRegistryV2 if not found
    if (!sageRegistryAddress) {
      console.log('\n  Deploying new SageRegistryV2...');
      const SageRegistryV2 = await hre.ethers.getContractFactory('SageRegistryV2');
      sageRegistry = await SageRegistryV2.deploy();
      await sageRegistry.waitForDeployment();

      sageRegistryAddress = await sageRegistry.getAddress();
      const deployTx = sageRegistry.deploymentTransaction();

      deploymentInfo.contracts.SageRegistryV2 = {
        address: sageRegistryAddress,
        transactionHash: deployTx.hash,
        blockNumber: deployTx.blockNumber,
        gasUsed: deployTx.gasUsed?.toString() || '0',
      };
      deploymentInfo.gasUsed += deployTx.gasUsed || BigInt(0);

      console.log(`  ✓ SageRegistryV2 deployed to: ${sageRegistryAddress}`);
      console.log(`  📜 Tx: ${deployTx.hash}`);
    }

    // ==========================================
    // Step 1: Deploy ERC8004IdentityRegistry
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 1: Deploying ERC8004IdentityRegistry...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const ERC8004IdentityRegistry = await hre.ethers.getContractFactory('ERC8004IdentityRegistry');
    const identityRegistry = await ERC8004IdentityRegistry.deploy(sageRegistryAddress);
    await identityRegistry.waitForDeployment();

    const identityAddress = await identityRegistry.getAddress();
    const identityDeployTx = identityRegistry.deploymentTransaction();

    deploymentInfo.contracts.ERC8004IdentityRegistry = {
      address: identityAddress,
      transactionHash: identityDeployTx.hash,
      blockNumber: identityDeployTx.blockNumber,
      gasUsed: identityDeployTx.gasUsed?.toString() || '0',
      constructorArgs: [sageRegistryAddress],
    };
    deploymentInfo.gasUsed += identityDeployTx.gasUsed || BigInt(0);

    console.log(`✓ ERC8004IdentityRegistry deployed to: ${identityAddress}`);
    console.log(`📜 Tx: ${identityDeployTx.hash}`);
    console.log(`⛽ Gas used: ${identityDeployTx.gasUsed?.toString() || 'Unknown'}`);

    // ==========================================
    // Step 2: Deploy ERC8004ReputationRegistry
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 2: Deploying ERC8004ReputationRegistry...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const ERC8004ReputationRegistry = await hre.ethers.getContractFactory('ERC8004ReputationRegistry');
    const reputationRegistry = await ERC8004ReputationRegistry.deploy(identityAddress);
    await reputationRegistry.waitForDeployment();

    const reputationAddress = await reputationRegistry.getAddress();
    const reputationDeployTx = reputationRegistry.deploymentTransaction();

    deploymentInfo.contracts.ERC8004ReputationRegistry = {
      address: reputationAddress,
      transactionHash: reputationDeployTx.hash,
      blockNumber: reputationDeployTx.blockNumber,
      gasUsed: reputationDeployTx.gasUsed?.toString() || '0',
      constructorArgs: [identityAddress],
    };
    deploymentInfo.gasUsed += reputationDeployTx.gasUsed || BigInt(0);

    console.log(`✓ ERC8004ReputationRegistry deployed to: ${reputationAddress}`);
    console.log(`📜 Tx: ${reputationDeployTx.hash}`);
    console.log(`⛽ Gas used: ${reputationDeployTx.gasUsed?.toString() || 'Unknown'}`);

    // ==========================================
    // Step 3: Deploy ERC8004ValidationRegistry
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 3: Deploying ERC8004ValidationRegistry...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const ERC8004ValidationRegistry = await hre.ethers.getContractFactory('ERC8004ValidationRegistry');
    const validationRegistry = await ERC8004ValidationRegistry.deploy(identityAddress, reputationAddress);
    await validationRegistry.waitForDeployment();

    const validationAddress = await validationRegistry.getAddress();
    const validationDeployTx = validationRegistry.deploymentTransaction();

    deploymentInfo.contracts.ERC8004ValidationRegistry = {
      address: validationAddress,
      transactionHash: validationDeployTx.hash,
      blockNumber: validationDeployTx.blockNumber,
      gasUsed: validationDeployTx.gasUsed?.toString() || '0',
      constructorArgs: [identityAddress, reputationAddress],
    };
    deploymentInfo.gasUsed += validationDeployTx.gasUsed || BigInt(0);

    console.log(`✓ ERC8004ValidationRegistry deployed to: ${validationAddress}`);
    console.log(`📜 Tx: ${validationDeployTx.hash}`);
    console.log(`⛽ Gas used: ${validationDeployTx.gasUsed?.toString() || 'Unknown'}`);

    // ==========================================
    // Step 4: Configure Registry Links
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 4: Configuring registry links...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    console.log('\n  Setting Validation Registry in Reputation Registry...');
    const setValidationTx = await reputationRegistry.setValidationRegistry(validationAddress);
    const setValidationReceipt = await setValidationTx.wait();
    deploymentInfo.gasUsed += setValidationReceipt.gasUsed;

    console.log(`  ✓ Validation Registry configured`);
    console.log(`  📜 Tx: ${setValidationTx.hash}`);
    console.log(`  ⛽ Gas used: ${setValidationReceipt.gasUsed.toString()}`);

    // ==========================================
    // Step 5: Save Deployment Info
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 5: Saving deployment information...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    // Calculate total gas cost
    const gasPrice = (await hre.ethers.provider.getFeeData()).gasPrice || BigInt(0);
    const totalGasUsed = deploymentInfo.gasUsed.toString();
    const estimatedCost = hre.ethers.formatEther(deploymentInfo.gasUsed * gasPrice);

    deploymentInfo.gasUsed = totalGasUsed;
    deploymentInfo.estimatedCostETH = estimatedCost;
    deploymentInfo.gasPriceGwei = hre.ethers.formatUnits(gasPrice, 'gwei');

    // Add Etherscan URLs
    const etherscanBase = 'https://sepolia.etherscan.io';
    deploymentInfo.explorer = {
      identityRegistry: `${etherscanBase}/address/${identityAddress}`,
      reputationRegistry: `${etherscanBase}/address/${reputationAddress}`,
      validationRegistry: `${etherscanBase}/address/${validationAddress}`,
      sageRegistryV2: `${etherscanBase}/address/${sageRegistryAddress}`,
    };

    // Save to file
    if (!fs.existsSync(deploymentsDir)) {
      fs.mkdirSync(deploymentsDir, { recursive: true });
    }

    const erc8004File = path.join(deploymentsDir, `${network}-erc8004.json`);
    fs.writeFileSync(erc8004File, JSON.stringify(deploymentInfo, null, 2));
    console.log(`✓ Deployment info saved to: deployments/${network}-erc8004.json`);

    // ==========================================
    // Step 6: Generate Environment Variables
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 6: Generating environment variables...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const envContent = `# ERC-8004 Deployment - Sepolia Testnet
# Generated: ${deploymentInfo.timestamp}
# Network: ${network} (Chain ID: ${deploymentInfo.chainId})

# ERC-8004 Contract Addresses
ERC8004_IDENTITY_REGISTRY=${identityAddress}
ERC8004_REPUTATION_REGISTRY=${reputationAddress}
ERC8004_VALIDATION_REGISTRY=${validationAddress}

# Base Registry
SAGE_REGISTRY_V2=${sageRegistryAddress}

# Network Configuration
NETWORK=${network}
CHAIN_ID=${deploymentInfo.chainId}
DEPLOYER=${deployer.address}

# Etherscan URLs
ETHERSCAN_IDENTITY=${deploymentInfo.explorer.identityRegistry}
ETHERSCAN_REPUTATION=${deploymentInfo.explorer.reputationRegistry}
ETHERSCAN_VALIDATION=${deploymentInfo.explorer.validationRegistry}
`;

    const envFile = path.join(deploymentsDir, `${network}-erc8004.env`);
    fs.writeFileSync(envFile, envContent);
    console.log(`✓ Environment variables saved to: deployments/${network}-erc8004.env`);

    // ==========================================
    // Step 7: Verify Contracts
    // ==========================================
    if (network === 'sepolia' && process.env.ETHERSCAN_API_KEY) {
      console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
      console.log('📋 Step 7: Verifying contracts on Etherscan...');
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

      console.log('\n⏳ Waiting 30 seconds for Etherscan to index...');
      await new Promise((resolve) => setTimeout(resolve, 30000));

      // Verify Identity Registry
      try {
        console.log('\n  Verifying ERC8004IdentityRegistry...');
        await hre.run('verify:verify', {
          address: identityAddress,
          constructorArguments: [sageRegistryAddress],
        });
        console.log('  ✓ ERC8004IdentityRegistry verified');
      } catch (error) {
        console.log(`  ⚠️  Verification failed: ${error.message}`);
      }

      // Verify Reputation Registry
      try {
        console.log('\n  Verifying ERC8004ReputationRegistry...');
        await hre.run('verify:verify', {
          address: reputationAddress,
          constructorArguments: [identityAddress],
        });
        console.log('  ✓ ERC8004ReputationRegistry verified');
      } catch (error) {
        console.log(`  ⚠️  Verification failed: ${error.message}`);
      }

      // Verify Validation Registry
      try {
        console.log('\n  Verifying ERC8004ValidationRegistry...');
        await hre.run('verify:verify', {
          address: validationAddress,
          constructorArguments: [identityAddress, reputationAddress],
        });
        console.log('  ✓ ERC8004ValidationRegistry verified');
      } catch (error) {
        console.log(`  ⚠️  Verification failed: ${error.message}`);
      }
    } else {
      console.log('\n⚠️  Skipping contract verification (ETHERSCAN_API_KEY not set)');
    }

    // ==========================================
    // Final Summary
    // ==========================================
    console.log('\n');
    console.log('╔════════════════════════════════════════════════════════════╗');
    console.log('║           DEPLOYMENT SUCCESSFUL! 🎉                        ║');
    console.log('╚════════════════════════════════════════════════════════════╝');

    console.log('\n📊 Deployment Summary:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`  Network:        ${network}`);
    console.log(`  Chain ID:       ${deploymentInfo.chainId}`);
    console.log(`  Deployer:       ${deployer.address}`);
    console.log(`  Total Gas:      ${totalGasUsed}`);
    console.log(`  Gas Price:      ${deploymentInfo.gasPriceGwei} gwei`);
    console.log(`  Estimated Cost: ${estimatedCost} ETH`);

    console.log('\n📜 Contract Addresses:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`  SageRegistryV2:            ${sageRegistryAddress}`);
    console.log(`  ERC8004IdentityRegistry:   ${identityAddress}`);
    console.log(`  ERC8004ReputationRegistry: ${reputationAddress}`);
    console.log(`  ERC8004ValidationRegistry: ${validationAddress}`);

    console.log('\n🔗 Etherscan Links:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`  Identity:   ${deploymentInfo.explorer.identityRegistry}`);
    console.log(`  Reputation: ${deploymentInfo.explorer.reputationRegistry}`);
    console.log(`  Validation: ${deploymentInfo.explorer.validationRegistry}`);

    console.log('\n📝 Next Steps:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('  1. Share deployment info with community');
    console.log('  2. Begin community testing period');
    console.log('  3. Monitor contract interactions');
    console.log('  4. Collect feedback for mainnet deployment');
    console.log('  5. Optional: Schedule security audit');
    console.log('');

    return deploymentInfo;
  } catch (error) {
    console.error('\n❌ Deployment failed:', error);
    console.error('\nError details:', error.message);

    // Save partial deployment info
    if (Object.keys(deploymentInfo.contracts).length > 0) {
      const failedFile = path.join(deploymentsDir, `${network}-erc8004-failed-${Date.now()}.json`);
      fs.writeFileSync(failedFile, JSON.stringify(deploymentInfo, null, 2));
      console.log(`\n💾 Partial deployment info saved to: ${failedFile}`);
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
