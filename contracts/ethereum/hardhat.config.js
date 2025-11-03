import hardhatEthers from "@nomicfoundation/hardhat-ethers";
import hardhatToolbox from "@nomicfoundation/hardhat-toolbox-mocha-ethers";
import "@nomicfoundation/hardhat-verify";
import dotenv from "dotenv";

dotenv.config();

// ============================================
// CONFIGURATION VALIDATION
// ============================================

// Helper function to validate environment variables
function getEnvVariable(key, defaultValue) {
  const value = process.env[key];
  if (!value && !defaultValue) {
    console.warn(`  Warning: ${key} is not set in environment variables`);
  }
  return value || defaultValue || "";
}

// Parse gas settings
const parseGasPrice = (envKey, defaultGwei) => {
  const gweiValue = getEnvVariable(envKey, defaultGwei);
  return gweiValue ? parseInt(gweiValue) * 1000000000 : undefined;
};

// Parse integer settings
const parseIntSetting = (envKey, defaultValue) => {
  return parseInt(getEnvVariable(envKey, defaultValue.toString()));
};

// ============================================
// NETWORK CONFIGURATION
// ============================================

const networks = {
  // Local Development Networks
  hardhat: {
    type: "edr-simulated",
    chainId: 31337,
    blockGasLimit: 30000000, // 30M gas limit for complex governance operations
    mining: {
      auto: true,
      interval: 0
    },
    allowBlocksWithSameTimestamp: true
  },

  localhost: {
    type: "http",
    url: getEnvVariable("LOCALHOST_RPC_URL", "http://127.0.0.1:8545"),
    chainId: 31337,
    timeout: 60000
  },

  // Kaia Testnet (Kairos)
  kairos: {
    type: "http",
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
    type: "http",
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
    type: "http",
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

  // Ethereum Mainnet
  mainnet: {
    type: "http",
    url: getEnvVariable("ETHEREUM_MAINNET_RPC_URL", "https://eth.llamarpc.com"),
    chainId: 1,
    accounts: process.env.MAINNET_PRIVATE_KEY
      ? [process.env.MAINNET_PRIVATE_KEY]
      : [],
    gasPrice: "auto",
    gas: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Ethereum Sepolia Testnet
  sepolia: {
    type: "http",
    url: getEnvVariable("ETHEREUM_SEPOLIA_RPC_URL", "https://ethereum-sepolia-rpc.publicnode.com"),
    chainId: 11155111,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: "auto",
    gas: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // BSC Mainnet
  bsc: {
    type: "http",
    url: getEnvVariable("BSC_MAINNET_RPC_URL", "https://bsc-dataseed1.binance.org"),
    chainId: 56,
    accounts: process.env.MAINNET_PRIVATE_KEY
      ? [process.env.MAINNET_PRIVATE_KEY]
      : [],
    gasPrice: parseGasPrice("BSC_GAS_PRICE_GWEI", "3"),
    timeout: 60000,
    confirmations: 2
  },

  // BSC Testnet
  bscTestnet: {
    type: "http",
    url: getEnvVariable("BSC_TESTNET_RPC_URL", "https://data-seed-prebsc-1-s1.binance.org:8545"),
    chainId: 97,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: parseGasPrice("BSC_GAS_PRICE_GWEI", "10"),
    timeout: 60000,
    confirmations: 2
  },

  // Base Mainnet
  base: {
    type: "http",
    url: getEnvVariable("BASE_MAINNET_RPC_URL", "https://mainnet.base.org"),
    chainId: 8453,
    accounts: process.env.MAINNET_PRIVATE_KEY
      ? [process.env.MAINNET_PRIVATE_KEY]
      : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Base Sepolia Testnet
  baseSepolia: {
    type: "http",
    url: getEnvVariable("BASE_TESTNET_RPC_URL", "https://sepolia.base.org"),
    chainId: 84532,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Arbitrum One
  arbitrum: {
    type: "http",
    url: getEnvVariable("ARBITRUM_MAINNET_RPC_URL", "https://arb1.arbitrum.io/rpc"),
    chainId: 42161,
    accounts: process.env.MAINNET_PRIVATE_KEY
      ? [process.env.MAINNET_PRIVATE_KEY]
      : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Arbitrum Sepolia
  arbitrumSepolia: {
    type: "http",
    url: getEnvVariable("ARBITRUM_TESTNET_RPC_URL", "https://sepolia-rollup.arbitrum.io/rpc"),
    chainId: 421614,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Optimism Mainnet
  optimism: {
    type: "http",
    url: getEnvVariable("OPTIMISM_MAINNET_RPC_URL", "https://mainnet.optimism.io"),
    chainId: 10,
    accounts: process.env.MAINNET_PRIVATE_KEY
      ? [process.env.MAINNET_PRIVATE_KEY]
      : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  },

  // Optimism Sepolia
  optimismSepolia: {
    type: "http",
    url: getEnvVariable("OPTIMISM_TESTNET_RPC_URL", "https://sepolia.optimism.io"),
    chainId: 11155420,
    accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    gasPrice: "auto",
    timeout: 60000,
    confirmations: 2
  }
};

// ============================================
// SOLIDITY COMPILER CONFIGURATION
// ============================================

// Adjust optimizer runs for coverage to handle deep stacks
const optimizerRuns = process.env.COVERAGE
  ? 800  // Higher runs for coverage to reduce stack depth
  : parseIntSetting("OPTIMIZER_RUNS", 200);

// viaIR setting: Enable only when needed to avoid verification issues
// Note: viaIR can cause Etherscan/block explorer verification failures
// Enable with VIA_IR=true environment variable if you need it for stack depth issues
const useViaIR = process.env.VIA_IR === "true" || process.env.COVERAGE === "true";

const solidityConfig = {
  version: "0.8.20",
  settings: {
    optimizer: {
      enabled: true,
      runs: optimizerRuns,
      details: {
        yul: true,
        yulDetails: {
          stackAllocation: true,
          optimizerSteps: "dhfoDgvulfnTUtnIf"
        }
      }
    },
    viaIR: useViaIR,  // Conditional: only enable when explicitly needed
    evmVersion: "shanghai",
    metadata: {
      bytecodeHash: "ipfs",
      appendCBOR: true
    },
    outputSelection: {
      "*": {
        "*": [
          "abi",
          "evm.bytecode",
          "evm.deployedBytecode",
          "evm.methodIdentifiers",
          "metadata"
        ],
        "": ["ast"]
      }
    }
  },
  // Exclude deprecated contracts and tests from compilation
  exclude: [
    "contracts/**/deprecated/**",
    "contracts/**/archive/**",
    "test/**/deprecated/**"
  ]
};

// ============================================
// ETHERSCAN/EXPLORER CONFIGURATION
// ============================================

// Hardhat v3 Etherscan configuration
// For multi-network projects, use object format with network-specific keys
const etherscanConfig = {
  apiKey: {
    // Ethereum networks - use same Etherscan API key
    mainnet: getEnvVariable("ETHERSCAN_API_KEY", ""),
    sepolia: getEnvVariable("ETHERSCAN_API_KEY", ""),

    // Kaia networks
    kairos: getEnvVariable("KAIROS_API_KEY", "unnecessary"),
    kaia: getEnvVariable("KAIA_API_KEY", "unnecessary"),
    cypress: getEnvVariable("KAIA_API_KEY", "unnecessary"),

    // BSC networks
    bsc: getEnvVariable("BSCSCAN_API_KEY", ""),
    bscTestnet: getEnvVariable("BSCSCAN_API_KEY", ""),

    // Base networks
    base: getEnvVariable("BASESCAN_API_KEY", ""),
    baseSepolia: getEnvVariable("BASESCAN_API_KEY", ""),

    // Arbitrum networks
    arbitrumOne: getEnvVariable("ARBISCAN_API_KEY", ""),
    arbitrum: getEnvVariable("ARBISCAN_API_KEY", ""),
    arbitrumSepolia: getEnvVariable("ARBISCAN_API_KEY", ""),

    // Optimism networks
    optimism: getEnvVariable("OPTIMISTIC_ETHERSCAN_API_KEY", ""),
    optimisticEthereum: getEnvVariable("OPTIMISTIC_ETHERSCAN_API_KEY", ""),
    optimismSepolia: getEnvVariable("OPTIMISTIC_ETHERSCAN_API_KEY", "")
  },

  // Custom chains for networks not natively supported
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
    },
    {
      network: "baseSepolia",
      chainId: 84532,
      urls: {
        apiURL: "https://api-sepolia.basescan.org/api",
        browserURL: "https://sepolia.basescan.org"
      }
    },
    {
      network: "arbitrumSepolia",
      chainId: 421614,
      urls: {
        apiURL: "https://api-sepolia.arbiscan.io/api",
        browserURL: "https://sepolia.arbiscan.io"
      }
    },
    {
      network: "optimismSepolia",
      chainId: 11155420,
      urls: {
        apiURL: "https://api-sepolia-optimistic.etherscan.io/api",
        browserURL: "https://sepolia-optimistic.etherscan.io"
      }
    }
  ]
};

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

const config = {
  plugins: [hardhatEthers, hardhatToolbox],

  defaultNetwork: "hardhat",

  solidity: solidityConfig,

  networks: networks,

  // Etherscan verification configuration (Hardhat v3)
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
    timeout: parseIntSetting("TEST_TIMEOUT", 60000),
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
  console.log("\n Configuration Status:");
  console.log("   No private key configured - using Hardhat default accounts");
  console.log("   Set PRIVATE_KEY in .env for testnet/mainnet deployment\n");
}

// Display active network configuration
const activeNetwork = process.env.HARDHAT_NETWORK || "hardhat";
if (activeNetwork !== "hardhat") {
  console.log(`\n Active Network: ${activeNetwork}`);
  if (networks[activeNetwork]) {
    console.log(`   Chain ID: ${networks[activeNetwork].chainId}`);
    console.log(`   RPC URL: ${networks[activeNetwork].url}\n`);
  }
}

export default config;
