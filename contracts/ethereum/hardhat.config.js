require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();

module.exports = {
  solidity: {
    version: "0.8.19",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
        details: {
          yul: true,
          yulDetails: {
            stackAllocation: true,
            optimizerSteps: "dhfoDgvulfnTUtnIf"
          }
        }
      },
      viaIR: true  // Enable IR-based compilation to avoid stack too deep errors
    }
  },
  networks: {
    // Local development network
    hardhat: {
      chainId: 31337
    },
    localhost: {
      url: "http://127.0.0.1:8545"
    },
    // Kaia Mainnet (Cypress)
    kaia: {
      url: process.env.KAIA_RPC_URL || "https://public-en.node.kaia.io",
      chainId: 8217,
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
      gasPrice: 250000000000, // 250 ston (fixed gas price in Kaia)
    },
    // Kaia Testnet (Kairos)
    kairos: {
      url: process.env.KAIROS_RPC_URL || "https://public-en-kairos.node.kaia.io",
      chainId: 1001,
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
      gasPrice: 250000000000, // 250 ston (fixed gas price in Kaia)
    }
  },
  etherscan: {
    // Kaia networks don't require API keys for verification
    // These are set to "unnecessary" for hardhat-verify plugin compatibility
    apiKey: {
      kairos: process.env.KAIROS_API_KEY || "unnecessary",
      kaia: process.env.KAIA_API_KEY || "unnecessary"
    },
    customChains: [
      {
        network: "kairos",
        chainId: 1001,
        urls: {
          // Kairos testnet explorer API and browser URLs
          apiURL: "https://api-kairos.klaytnscope.com/api",
          browserURL: "https://kairos.klaytnscope.com"
        }
      },
      {
        network: "kaia",
        chainId: 8217,
        urls: {
          // Kaia mainnet (Cypress) explorer API and browser URLs
          apiURL: "https://api.klaytnscope.com/api",
          browserURL: "https://klaytnscope.com"
        }
      }
    ]
  },
  // Sourcify verification support for Kaia networks
  sourcify: {
    enabled: true,
    apiUrl: "https://sourcify.dev/server",
    browserUrl: "https://sourcify.dev"
  },
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts"
  },
  mocha: {
    timeout: 40000
  }
};