const hre = require('hardhat');
const fs = require('fs');
const path = require('path');
const readline = require('readline');

async function promptMnemonic() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    rl.question('Enter your mnemonic phrase: ', (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

async function main() {
  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║      Generate Private Key from Mnemonic (SECURE)          ║');
  console.log('╚════════════════════════════════════════════════════════════╝\n');

  // Get mnemonic from user input (not stored in code)
  const mnemonic = await promptMnemonic();

  if (!mnemonic || mnemonic.split(' ').length < 12) {
    console.error('❌ Invalid mnemonic. Please provide a valid 12 or 24 word phrase.');
    process.exit(1);
  }

  try {
    const wallet = hre.ethers.Wallet.fromPhrase(mnemonic);

    console.log('\n✅ Wallet generated successfully!\n');
    console.log('Address:     ', wallet.address);
    console.log('Private Key: ', wallet.privateKey);

    // Ask if user wants to create .env file
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });

    rl.question('\nDo you want to create/update .env file with this key? (yes/no): ', (answer) => {
      rl.close();

      if (answer.toLowerCase() === 'yes' || answer.toLowerCase() === 'y') {
        const envPath = path.join(__dirname, '..', '.env');
        const envExamplePath = path.join(__dirname, '..', '.env.example');

        let envContent = '';

        // Read existing .env or use .env.example as template
        if (fs.existsSync(envPath)) {
          envContent = fs.readFileSync(envPath, 'utf8');
          console.log('\n📝 Updating existing .env file...');
        } else if (fs.existsSync(envExamplePath)) {
          envContent = fs.readFileSync(envExamplePath, 'utf8');
          console.log('\n📝 Creating .env from .env.example template...');
        } else {
          console.error('❌ No .env.example found. Creating basic .env file...');
          envContent = '# SAGE Smart Contract Configuration\n\n';
        }

        // Update or add PRIVATE_KEY
        const privateKeyWithout0x = wallet.privateKey.startsWith('0x')
          ? wallet.privateKey.slice(2)
          : wallet.privateKey;

        if (envContent.includes('PRIVATE_KEY=')) {
          envContent = envContent.replace(
            /^PRIVATE_KEY=.*/m,
            `PRIVATE_KEY=${privateKeyWithout0x}`
          );
        } else {
          envContent += `\n# Generated from mnemonic\nPRIVATE_KEY=${privateKeyWithout0x}\n`;
        }

        // Write .env file
        fs.writeFileSync(envPath, envContent);
        console.log('✅ .env file updated successfully!');
        console.log(`   Location: ${envPath}`);
        console.log('\n⚠️  IMPORTANT: .env file is in .gitignore and will NOT be committed to git.');
      }

      console.log('\n⚠️  Security Reminders:');
      console.log('   1. Never share your mnemonic or private key');
      console.log('   2. Never commit .env file to git');
      console.log('   3. Use separate keys for testnet and mainnet');
      console.log('   4. Keep backups in secure offline storage\n');
    });
  } catch (error) {
    console.error('\n❌ Failed to generate wallet:', error.message);
    process.exit(1);
  }
}

// Only run if executed directly (not imported)
if (require.main === module) {
  main()
    .then(() => {
      // Don't exit immediately to allow readline to finish
    })
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });
}

module.exports = main;
