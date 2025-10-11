/** @type import('hardhat/config').HardhatUserConfig */
export default {
  solidity: "0.8.19",
  networks: {
    hardhat: {
      type: "edr-simulated",
      chainId: 31337,
      accounts: {
        mnemonic: "test test test test test test test test test test test junk",
        count: 10,
        accountsBalance: "10000000000000000000000" // 10000 ETH
      }
    },
    localhost: {
      type: "http",
      url: "http://127.0.0.1:8545",
      chainId: 31337
    }
  },
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts"
  }
};
