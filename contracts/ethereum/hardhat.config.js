require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();

// ============================================
// CONFIGURATION VALIDATION
// ============================================

// Helper function to validate environment variables
function getEnvVariable(key, defaultValue) {
  const value = process.env[key];
  if (!value && !defaultValue) {
    console.warn(`⚠️  Warning: ${key} is not set in environment variables`);
  }
  return value || defaultValue;
}

// Parse gas settings
const parseGasPrice = (envKey, defaultGwei) => {
  const gweiValue = getEnvVariable(envKey, defaultGwei);
  return gweiValue ? parseInt(gweiValue) * 1000000000 : undefined;
};

// ============================================
// NETWORK CONFIGURATION
// ============================================

const networks = {
  // Local Development Networks
  hardhat: {
    chainId: 31337,
    mining: {
      auto: true,
      interval: 0
    }
  },
  
  localhost: {
    url: getEnvVariable("LOCALHOST_RPC_URL", "http://127.0.0.1:8545"),
    chainId: 31337,
    timeout: 60000
  },
  
  // Kaia Testnet (Kairos)
  kairos: {
    url: getEnvVariable("KAIROS_RPC_URL", "https://public-en-kairos.node.kaia.io"),
    chainId: 1001,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: parseGasPrice("GAS_PRICE_GWEI", "250"),
    gas: parseInt(getEnvVariable("GAS_LIMIT", "3000000")),
    timeout: 60000,
    confirmations: 2
  },
  
  // Kaia Mainnet (Cypress)
  kaia: {
    url: getEnvVariable("KAIA_RPC_URL", "https://public-en.node.kaia.io"),
    chainId: 8217,
    accounts: process.env.MAINNET_PRIVATE_KEY 
      ? [process.env.MAINNET_PRIVATE_KEY] 
      : (process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : []),
    gasPrice: parseGasPrice("GAS_PRICE_GWEI", "250"),
    gas: parseInt(getEnvVariable("GAS_LIMIT", "3000000")),
    timeout: 60000,
    confirmations: 2
  },
  
  // Alias for Kaia Mainnet
  cypress: {
    url: getEnvVariable("KAIA_RPC_URL", "https://public-en.node.kaia.io"),
    chainId: 8217,
    accounts: process.env.MAINNET_PRIVATE_KEY 
      ? [process.env.MAINNET_PRIVATE_KEY] 
      : (process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : []),
    gasPrice: parseGasPrice("GAS_PRICE_GWEI", "250"),
    gas: parseInt(getEnvVariable("GAS_LIMIT", "3000000")),
    timeout: 60000,
    confirmations: 2
  }
};

// Optional: Add Sepolia testnet support if configured
if (process.env.SEPOLIA_RPC_URL) {
  networks.sepolia = {
    url: process.env.SEPOLIA_RPC_URL,
    chainId: 11155111,
    accounts: process.env.SEPOLIA_PRIVATE_KEY 
      ? [process.env.SEPOLIA_PRIVATE_KEY]
      : (process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : []),
    gasPrice: "auto",
    gas: "auto",
    timeout: 60000,
    confirmations: 2
  };
}

// ============================================
// SOLIDITY COMPILER CONFIGURATION
// ============================================

const solidityConfig = {
  version: "0.8.19",
  settings: {
    optimizer: {
      enabled: true,
      runs: parseInt(getEnvVariable("OPTIMIZER_RUNS", "200")),
      details: {
        yul: true,
        yulDetails: {
          stackAllocation: true,
          optimizerSteps: "dhfoDgvulfnTUtnIf"
        }
      }
    },
    viaIR: true,  // Enable IR-based compilation to avoid stack too deep errors
    metadata: {
      bytecodeHash: "ipfs"
    }
  }
};

// ============================================
// ETHERSCAN/EXPLORER CONFIGURATION
// ============================================

const etherscanConfig = {
  apiKey: {
    kairos: getEnvVariable("KAIROS_API_KEY", "unnecessary"),
    kaia: getEnvVariable("KAIA_API_KEY", "unnecessary"),
    cypress: getEnvVariable("KAIA_API_KEY", "unnecessary")
  },
  customChains: [
    {
      network: "kairos",
      chainId: 1001,
      urls: {
        apiURL: "https://kairos-api.kaiascan.io/hardhat-verify",
        browserURL: "https://kairos.kaiascan.io"
      }
    },
    {
      network: "kaia",
      chainId: 8217,
      urls: {
        apiURL: "https://api.kaiascan.io/api",
        browserURL: "https://kaiascan.io"
      }
    },
    {
      network: "cypress",
      chainId: 8217,
      urls: {
        apiURL: "https://api.kaiascan.io/api",
        browserURL: "https://kaiascan.io"
      }
    }
  ]
};

// Add Sepolia to etherscan config if configured
if (process.env.ETHERSCAN_API_KEY && process.env.SEPOLIA_RPC_URL) {
  etherscanConfig.apiKey.sepolia = process.env.ETHERSCAN_API_KEY;
}

// ============================================
// GAS REPORTER CONFIGURATION
// ============================================

const gasReporterConfig = {
  enabled: process.env.REPORT_GAS === "true",
  currency: "USD",
  gasPrice: 250,  // in gwei for Kaia
  coinmarketcap: process.env.COINMARKETCAP_API_KEY,
  excludeContracts: [],
  src: "./contracts",
  outputFile: process.env.GAS_REPORT_FILE
};

// ============================================
// MAIN CONFIGURATION EXPORT
// ============================================

module.exports = {
  defaultNetwork: "hardhat",
  
  solidity: solidityConfig,
  
  networks: networks,
  
  etherscan: etherscanConfig,
  
  // Sourcify verification support
  sourcify: {
    enabled: process.env.SOURCIFY_ENABLED !== "false",
    apiUrl: "https://sourcify.dev/server",
    browserUrl: "https://sourcify.dev"
  },
  
  // Gas reporting
  gasReporter: gasReporterConfig,
  
  // Path configuration
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts",
    scripts: "./scripts"
  },
  
  // Test configuration
  mocha: {
    timeout: parseInt(getEnvVariable("TEST_TIMEOUT", "60000")),
    reporter: getEnvVariable("TEST_REPORTER", "spec")
  },
  
  // TypeChain configuration
  typechain: {
    outDir: "typechain-types",
    target: "ethers-v6",
    alwaysGenerateOverloads: false,
    externalArtifacts: ["externalArtifacts/*.json"],
    dontOverrideCompile: false
  }
};

// ============================================
// CONFIGURATION VALIDATION
// ============================================

// Validate critical configuration on startup
if (!process.env.PRIVATE_KEY && !process.env.TEST_MNEMONIC) {
  console.log("\n📋 Configuration Status:");
  console.log("   No private key configured - using Hardhat default accounts");
  console.log("   Set PRIVATE_KEY in .env for testnet/mainnet deployment\n");
}

// Display active network configuration
const activeNetwork = process.env.HARDHAT_NETWORK || "hardhat";
if (activeNetwork !== "hardhat") {
  console.log(`\n🌐 Active Network: ${activeNetwork}`);
  if (networks[activeNetwork]) {
    console.log(`   Chain ID: ${networks[activeNetwork].chainId}`);
    console.log(`   RPC URL: ${networks[activeNetwork].url}\n`);
  }
}
