#!/usr/bin/env node

/**
 * Extract ABI from compiled contracts
 * Usage: node scripts/extract-abi.js
 */

const fs = require('fs');
const path = require('path');

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

async function extractABIs() {
  try {
    log("\n📜 Extracting Contract ABIs", "cyan");
    log("=" .repeat(50), "bright");

    // Create abi directory if it doesn't exist
    const abiDir = path.join(__dirname, '..', 'abi');
    if (!fs.existsSync(abiDir)) {
      fs.mkdirSync(abiDir, { recursive: true });
      log(" Created abi directory", "green");
    }

    // Contracts to extract
    const contracts = [
      {
        name: 'SageRegistryV2',
        path: 'contracts/SageRegistryV2.sol/SageRegistryV2.json'
      },
      {
        name: 'SageRegistryV4',
        path: 'contracts/SageRegistryV4.sol/SageRegistryV4.json'
      },
      {
        name: 'SageVerificationHook',
        path: 'contracts/SageVerificationHook.sol/SageVerificationHook.json'
      },
      {
        name: 'ISageRegistry',
        path: 'contracts/interfaces/ISageRegistry.sol/ISageRegistry.json'
      },
      {
        name: 'ISageRegistryV4',
        path: 'contracts/interfaces/ISageRegistryV4.sol/ISageRegistryV4.json'
      },
      {
        name: 'IRegistryHook',
        path: 'contracts/interfaces/IRegistryHook.sol/IRegistryHook.json'
      }
    ];

    // Extract ABIs
    for (const contract of contracts) {
      const artifactPath = path.join(__dirname, '..', 'artifacts', contract.path);
      
      if (!fs.existsSync(artifactPath)) {
        log(`  Artifact not found for ${contract.name}`, "yellow");
        log(`   Please run 'npm run compile' first`, "yellow");
        continue;
      }

      // Read artifact
      const artifact = JSON.parse(fs.readFileSync(artifactPath, 'utf8'));
      
      // Extract ABI
      const abi = artifact.abi;
      
      // Save ABI to file
      const abiPath = path.join(abiDir, `${contract.name}.abi.json`);
      fs.writeFileSync(abiPath, JSON.stringify(abi, null, 2));
      
      // Save minified version
      const abiMinPath = path.join(abiDir, `${contract.name}.abi.min.json`);
      fs.writeFileSync(abiMinPath, JSON.stringify(abi));
      
      log(` Extracted ${contract.name}`, "green");
      log(`   📄 ${abiPath}`, "blue");
      log(`   📄 ${abiMinPath} (minified)`, "blue");
      
      // Show stats
      const functionCount = abi.filter(item => item.type === 'function').length;
      const eventCount = abi.filter(item => item.type === 'event').length;
      log(`    ${functionCount} functions, ${eventCount} events`, "cyan");
    }

    // Create index file for easy imports
    const indexContent = `// Auto-generated ABI exports
// Generated: ${new Date().toISOString()}

module.exports = {
  SageRegistryV2: require('./SageRegistryV2.abi.json'),
  SageRegistryV4: require('./SageRegistryV4.abi.json'),
  SageVerificationHook: require('./SageVerificationHook.abi.json'),
  ISageRegistry: require('./ISageRegistry.abi.json'),
  ISageRegistryV4: require('./ISageRegistryV4.abi.json'),
  IRegistryHook: require('./IRegistryHook.abi.json')
};
`;

    fs.writeFileSync(path.join(abiDir, 'index.js'), indexContent);
    log("\n Created index.js for easy imports", "green");

    // Create TypeScript definitions
    const tsContent = `// Auto-generated TypeScript definitions
// Generated: ${new Date().toISOString()}

export const SageRegistryV2: any[];
export const SageRegistryV4: any[];
export const SageVerificationHook: any[];
export const ISageRegistry: any[];
export const ISageRegistryV4: any[];
export const IRegistryHook: any[];
`;

    fs.writeFileSync(path.join(abiDir, 'index.d.ts'), tsContent);
    log(" Created TypeScript definitions", "green");

    log("\n" + "=".repeat(50), "bright");
    log(" ABI extraction complete!", "green");
    log("\nUsage example:", "yellow");
    log("  const { SageRegistryV2 } = require('./abi');", "cyan");
    log("  // or", "cyan");
    log("  import SageRegistryV2ABI from './abi/SageRegistryV2.abi.json';", "cyan");

  } catch (error) {
    log("\n Error extracting ABIs:", "red");
    console.error(error);
    process.exit(1);
  }
}

// Run extraction
extractABIs();