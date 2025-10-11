const hre = require('hardhat');
const fs = require('fs');
const path = require('path');

async function main() {
  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║      Setup Environment for Sepolia Deployment              ║');
  console.log('╚════════════════════════════════════════════════════════════╝\n');

  // Read mnemonic from command line argument
  const mnemonic = process.env.MNEMONIC || process.argv[2];

  if (!mnemonic) {
    console.error('❌ Error: Mnemonic not provided');
    console.log('\nUsage:');
    console.log('  MNEMONIC="your mnemonic phrase" npx hardhat run scripts/setup-env-for-deployment.js');
    console.log('  or');
    console.log('  npx hardhat run scripts/setup-env-for-deployment.js "your mnemonic phrase"');
    process.exit(1);
  }

  if (mnemonic.split(' ').length < 12) {
    console.error('❌ Invalid mnemonic. Please provide a valid 12 or 24 word phrase.');
    process.exit(1);
  }

  try {
    // Generate wallet from mnemonic
    const wallet = hre.ethers.Wallet.fromPhrase(mnemonic);

    console.log('✅ Wallet generated successfully!\n');
    console.log('📍 Address:', wallet.address);
    console.log('🔑 Private Key:', wallet.privateKey.slice(0, 10) + '...' + wallet.privateKey.slice(-8));

    // Check Sepolia balance
    console.log('\n📊 Checking Sepolia balance...');
    try {
      const provider = new hre.ethers.JsonRpcProvider(
        process.env.SEPOLIA_RPC_URL || 'https://eth-sepolia.g.alchemy.com/v2/v4TawV7y1l8GhqM_4_KZu5x7H9R6poNW'
      );
      const balance = await provider.getBalance(wallet.address);
      console.log('💰 Balance:', hre.ethers.formatEther(balance), 'ETH');

      if (balance === BigInt(0)) {
        console.log('\n⚠️  Warning: Account has 0 ETH on Sepolia');
        console.log('   Get testnet ETH from:');
        console.log('   - https://sepoliafaucet.com/');
        console.log('   - https://www.alchemy.com/faucets/ethereum-sepolia');
        console.log('   - https://faucet.quicknode.com/ethereum/sepolia');
      } else if (balance < hre.ethers.parseEther('0.1')) {
        console.log('\n⚠️  Warning: Low balance (< 0.1 ETH)');
        console.log('   Deployment may require ~0.05-0.1 ETH for gas');
        console.log('   Consider getting more testnet ETH from faucets');
      } else {
        console.log('✅ Sufficient balance for deployment');
      }
    } catch (error) {
      console.log('⚠️  Could not check balance:', error.message);
    }

    // Update .env file
    console.log('\n📝 Updating .env file...');
    const envPath = path.join(__dirname, '..', '.env');

    let envContent = '';
    if (fs.existsSync(envPath)) {
      envContent = fs.readFileSync(envPath, 'utf8');
    } else {
      console.log('⚠️  .env file not found, creating new one');
      envContent = '# SAGE Smart Contract Configuration\n\n';
    }

    // Update SEPOLIA_PRIVATE_KEY
    const privateKeyWithout0x = wallet.privateKey.startsWith('0x')
      ? wallet.privateKey.slice(2)
      : wallet.privateKey;

    if (envContent.includes('SEPOLIA_PRIVATE_KEY=')) {
      envContent = envContent.replace(
        /SEPOLIA_PRIVATE_KEY=.*/,
        `SEPOLIA_PRIVATE_KEY=${privateKeyWithout0x}`
      );
    } else {
      envContent += `\nSEPOLIA_PRIVATE_KEY=${privateKeyWithout0x}\n`;
    }

    // Ensure SEPOLIA_RPC_URL is set
    if (!envContent.includes('SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com')) {
      if (envContent.includes('SEPOLIA_RPC_URL=')) {
        envContent = envContent.replace(
          /SEPOLIA_RPC_URL=.*/,
          'SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/v4TawV7y1l8GhqM_4_KZu5x7H9R6poNW'
        );
      } else {
        envContent += '\nSEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/v4TawV7y1l8GhqM_4_KZu5x7H9R6poNW\n';
      }
    }

    fs.writeFileSync(envPath, envContent);
    console.log('✅ .env file updated successfully!');

    console.log('\n╔════════════════════════════════════════════════════════════╗');
    console.log('║                    SETUP COMPLETE! 🎉                      ║');
    console.log('╚════════════════════════════════════════════════════════════╝');

    console.log('\n📋 Next Steps:');
    console.log('   1. Make sure you have enough Sepolia ETH (at least 0.1 ETH)');
    console.log('   2. Deploy contracts:');
    console.log('      npx hardhat run scripts/deploy-erc8004-sepolia.js --network sepolia');
    console.log('\n⚠️  Security Reminders:');
    console.log('   - .env file is in .gitignore (will NOT be committed)');
    console.log('   - Never share your private key or mnemonic');
    console.log('   - Use separate keys for testnet and mainnet\n');

  } catch (error) {
    console.error('\n❌ Failed to setup environment:', error.message);
    process.exit(1);
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
