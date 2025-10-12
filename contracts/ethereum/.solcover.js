module.exports = {
  skipFiles: [
    'test/',
    'governance/TimelockController.sol', // Skip complex contract with deep stack
    'SageRegistryV3.sol' // Skip due to stack too deep in coverage
  ],
  configureYulOptimizer: true,
  solcOptimizerDetails: {
    peephole: true,
    inliner: true,
    jumpdestRemover: true,
    orderLiterals: true,
    deduplicate: true,
    cse: true,
    constantOptimizer: true,
    yul: true,
    yulDetails: {
      stackAllocation: true,
      optimizerSteps: "dhfoDgvulfnTUtnIf"
    }
  },
  compileCommand: '../node_modules/.bin/hardhat compile',
  testCommand: '../node_modules/.bin/hardhat test',
  providerOptions: {
    default_balance_ether: '10000000000',
    total_accounts: 40,
    gasLimit: 0xfffffffffff,
    gasPrice: 0x01
  },
  istanbulReporter: ['html', 'lcov', 'text', 'json'],
  mocha: {
    timeout: 60000
  }
};
