const hre = require('hardhat');
const fs = require('fs');
const path = require('path');

/**
 * Deploy SAGE Core Contracts to Sepolia Testnet
 *
 * This script deploys SAGE's core infrastructure:
 * 1. SageRegistryV2 (if not already deployed)
 * 2. SageVerificationHook
 * 3. Configure hooks in SageRegistryV2
 */

async function main() {
  const network = hre.network.name;

  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║         SAGE Core Contracts Deployment Script              ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log(`\n Network: ${network}`);
  console.log(` Chain ID: ${hre.network.config.chainId || 'Unknown'}`);
  console.log('════════════════════════════════════════════════════════════\n');

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

  if (balance < hre.ethers.parseEther('0.05')) {
    console.error('\n❌ Insufficient balance for deployment');
    console.log(`   Required: ~0.05 ETH`);
    console.log(`   Current: ${hre.ethers.formatEther(balance)} ETH`);
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
    // Step 1: Check for existing SageRegistryV2
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 1: Checking for existing SageRegistryV2...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    let sageRegistryAddress;
    let sageRegistry;

    // Check if there's an existing deployment
    const deploymentsDir = path.join(__dirname, '..', 'deployments');
    const erc8004File = path.join(deploymentsDir, `${network}-erc8004.json`);
    const sageFile = path.join(deploymentsDir, `${network}-sage.json`);

    // Check ERC-8004 deployment file first
    if (fs.existsSync(erc8004File)) {
      const erc8004Deployment = JSON.parse(fs.readFileSync(erc8004File, 'utf8'));
      if (erc8004Deployment.contracts?.SageRegistryV2?.address) {
        sageRegistryAddress = erc8004Deployment.contracts.SageRegistryV2.address;
        console.log(`✓ Found SageRegistryV2 from ERC-8004 deployment: ${sageRegistryAddress}`);

        // Verify contract exists
        const code = await hre.ethers.provider.getCode(sageRegistryAddress);
        if (code === '0x') {
          console.log('⚠️  Contract not found at address, will deploy new one');
          sageRegistryAddress = null;
        } else {
          sageRegistry = await hre.ethers.getContractAt('SageRegistryV2', sageRegistryAddress);
          deploymentInfo.contracts.SageRegistryV2 = {
            address: sageRegistryAddress,
            status: 'existing',
            note: 'Reusing from ERC-8004 deployment',
          };
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
        status: 'newly_deployed',
      };
      deploymentInfo.gasUsed += deployTx.gasUsed || BigInt(0);

      console.log(`  ✓ SageRegistryV2 deployed to: ${sageRegistryAddress}`);
      console.log(`  📜 Tx: ${deployTx.hash}`);
    }

    // ==========================================
    // Step 2: Deploy SageVerificationHook
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 2: Deploying SageVerificationHook...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

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

    console.log(`✓ SageVerificationHook deployed to: ${hookAddress}`);
    console.log(`📜 Tx: ${hookDeployTx.hash}`);
    console.log(`⛽ Gas used: ${hookDeployTx.gasUsed?.toString() || 'Unknown'}`);

    // ==========================================
    // Step 3: Configure Hooks in SageRegistryV2
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 3: Configuring hooks in SageRegistryV2...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    console.log('\n  Setting BeforeRegisterHook...');
    const beforeTx = await sageRegistry.setBeforeRegisterHook(hookAddress);
    const beforeReceipt = await beforeTx.wait();
    deploymentInfo.gasUsed += beforeReceipt.gasUsed;

    console.log(`  ✓ BeforeRegisterHook configured`);
    console.log(`  📜 Tx: ${beforeTx.hash}`);
    console.log(`  ⛽ Gas used: ${beforeReceipt.gasUsed.toString()}`);

    console.log('\n  Setting AfterRegisterHook...');
    const afterTx = await sageRegistry.setAfterRegisterHook(hookAddress);
    const afterReceipt = await afterTx.wait();
    deploymentInfo.gasUsed += afterReceipt.gasUsed;

    console.log(`  ✓ AfterRegisterHook configured`);
    console.log(`  📜 Tx: ${afterTx.hash}`);
    console.log(`  ⛽ Gas used: ${afterReceipt.gasUsed.toString()}`);

    deploymentInfo.configuration = {
      beforeRegisterHook: hookAddress,
      afterRegisterHook: hookAddress,
    };

    // ==========================================
    // Step 4: Save Deployment Info
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 4: Saving deployment information...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    // Calculate total gas cost
    const gasPrice = (await hre.ethers.provider.getFeeData()).gasPrice || BigInt(0);
    const totalGasUsed = deploymentInfo.gasUsed.toString();
    const estimatedCost = hre.ethers.formatEther(deploymentInfo.gasUsed * gasPrice);

    deploymentInfo.gasUsed = totalGasUsed;
    deploymentInfo.estimatedCostETH = estimatedCost;
    deploymentInfo.gasPriceGwei = hre.ethers.formatUnits(gasPrice, 'gwei');

    // Add Etherscan URLs
    const etherscanBase = network === 'sepolia'
      ? 'https://sepolia.etherscan.io'
      : 'https://etherscan.io';

    deploymentInfo.explorer = {
      sageRegistryV2: `${etherscanBase}/address/${sageRegistryAddress}`,
      verificationHook: `${etherscanBase}/address/${hookAddress}`,
    };

    // Save to file
    if (!fs.existsSync(deploymentsDir)) {
      fs.mkdirSync(deploymentsDir, { recursive: true });
    }

    fs.writeFileSync(sageFile, JSON.stringify(deploymentInfo, null, 2));
    console.log(`✓ Deployment info saved to: deployments/${network}-sage.json`);

    // ==========================================
    // Step 5: Generate Environment Variables
    // ==========================================
    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📋 Step 5: Generating environment variables...');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const envContent = `# SAGE Core Deployment - ${network}
# Generated: ${deploymentInfo.timestamp}
# Network: ${network} (Chain ID: ${deploymentInfo.chainId})

# SAGE Core Contract Addresses
SAGE_REGISTRY_V2=${sageRegistryAddress}
SAGE_VERIFICATION_HOOK=${hookAddress}

# Network Configuration
NETWORK=${network}
CHAIN_ID=${deploymentInfo.chainId}
DEPLOYER=${deployer.address}

# Etherscan URLs
ETHERSCAN_SAGE_REGISTRY=${deploymentInfo.explorer.sageRegistryV2}
ETHERSCAN_VERIFICATION_HOOK=${deploymentInfo.explorer.verificationHook}
`;

    const envFile = path.join(deploymentsDir, `${network}-sage.env`);
    fs.writeFileSync(envFile, envContent);
    console.log(`✓ Environment variables saved to: deployments/${network}-sage.env`);

    // ==========================================
    // Step 6: Verify Contracts (if on Sepolia with API key)
    // ==========================================
    if (network === 'sepolia' && process.env.ETHERSCAN_API_KEY) {
      console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
      console.log('📋 Step 6: Verifying contracts on Etherscan...');
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

      console.log('\n⏳ Waiting 30 seconds for Etherscan to index...');
      await new Promise((resolve) => setTimeout(resolve, 30000));

      // Verify SageRegistryV2 (if newly deployed)
      if (deploymentInfo.contracts.SageRegistryV2.status === 'newly_deployed') {
        try {
          console.log('\n  Verifying SageRegistryV2...');
          await hre.run('verify:verify', {
            address: sageRegistryAddress,
            constructorArguments: [],
          });
          console.log('  ✓ SageRegistryV2 verified');
        } catch (error) {
          console.log(`  ⚠️  Verification failed: ${error.message}`);
        }
      }

      // Verify SageVerificationHook
      try {
        console.log('\n  Verifying SageVerificationHook...');
        await hre.run('verify:verify', {
          address: hookAddress,
          constructorArguments: [],
        });
        console.log('  ✓ SageVerificationHook verified');
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
    console.log(`  SageRegistryV2:         ${sageRegistryAddress}`);
    console.log(`  SageVerificationHook:   ${hookAddress}`);

    console.log('\n🔗 Etherscan Links:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`  Registry: ${deploymentInfo.explorer.sageRegistryV2}`);
    console.log(`  Hook:     ${deploymentInfo.explorer.verificationHook}`);

    console.log('\n📝 Next Steps:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('  1. SAGE core infrastructure is now complete');
    console.log('  2. ERC-8004 contracts will use this SageRegistryV2');
    console.log('  3. Test agent registration with verification hooks');
    console.log('  4. Update ERC-8004 deployment if needed');
    console.log('');

    return deploymentInfo;
  } catch (error) {
    console.error('\n❌ Deployment failed:', error);
    console.error('\nError details:', error.message);

    // Save partial deployment info
    if (Object.keys(deploymentInfo.contracts).length > 0) {
      const failedFile = path.join(deploymentsDir, `${network}-sage-failed-${Date.now()}.json`);
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
