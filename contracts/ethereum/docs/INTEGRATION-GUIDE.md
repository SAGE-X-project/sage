# SAGE Smart Contract Integration Guide

**Version**: 1.0
**Date**: 2025-10-07
**Audience**: Developers integrating with SAGE smart contracts

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Environment Setup](#environment-setup)
3. [Contract Addresses](#contract-addresses)
4. [Agent Registration](#agent-registration)
5. [Task Validation](#task-validation)
6. [Reputation Management](#reputation-management)
7. [TEE Key Governance](#tee-key-governance)
8. [Code Examples](#code-examples)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Installation

```bash
# Install dependencies
npm install ethers @openzeppelin/contracts

# For TypeScript projects
npm install --save-dev @types/node
```

### Basic Connection

```javascript
const { ethers } = require("ethers");

// Connect to Sepolia testnet
const provider = new ethers.JsonRpcProvider(
  "https://sepolia.infura.io/v3/YOUR_INFURA_KEY"
);

// Connect wallet
const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

// Contract addresses (Sepolia)
const SAGE_REGISTRY_ADDRESS = "0x...";
const VALIDATION_REGISTRY_ADDRESS = "0x...";
const REPUTATION_REGISTRY_ADDRESS = "0x...";

// Load contracts
const sageRegistry = new ethers.Contract(
  SAGE_REGISTRY_ADDRESS,
  SAGE_REGISTRY_ABI,
  wallet
);
```

---

## Environment Setup

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | ≥16.0.0 | Runtime environment |
| npm/yarn | Latest | Package management |
| Hardhat | ≥2.12.0 | Development framework |
| ethers.js | ≥6.0.0 | Ethereum library |

### Network Configuration

```javascript
// hardhat.config.js
module.exports = {
  solidity: "0.8.19",
  networks: {
    sepolia: {
      url: process.env.SEPOLIA_RPC_URL,
      accounts: [process.env.PRIVATE_KEY],
      chainId: 11155111,
    },
    mainnet: {
      url: process.env.MAINNET_RPC_URL,
      accounts: [process.env.PRIVATE_KEY],
      chainId: 1,
    },
  },
};
```

### Environment Variables

```bash
# .env
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
MAINNET_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY
PRIVATE_KEY=0x...
ETHERSCAN_API_KEY=your_etherscan_key
```

---

## Contract Addresses

### Sepolia Testnet

```javascript
const SEPOLIA_CONTRACTS = {
  SageRegistryV3: "0x...",
  ERC8004IdentityRegistry: "0x...",
  ERC8004ValidationRegistry: "0x...",
  ERC8004ReputationRegistryV2: "0x...",
  TEEKeyRegistry: "0x...",
};
```

### Mainnet (Coming Soon)

```javascript
const MAINNET_CONTRACTS = {
  // To be deployed after audit
};
```

### Contract ABIs

ABIs are available in the `artifacts/` directory after compilation:

```bash
npx hardhat compile
```

Or download from Etherscan after verification.

---

## Agent Registration

### Step 1: Generate Key Pair

```javascript
const ethers = require("ethers");

// Generate new wallet (agent key pair)
const agentWallet = ethers.Wallet.createRandom();

console.log("Agent Address:", agentWallet.address);
console.log("Public Key:", agentWallet.publicKey);
console.log("Private Key:", agentWallet.privateKey); // Keep secret!
```

### Step 2: Commit Registration

```javascript
async function commitAgentRegistration(
  sageRegistry,
  did,
  publicKey,
  userAddress,
  wallet
) {
  // Generate random salt
  const salt = ethers.randomBytes(32);

  // Get current chainId
  const network = await wallet.provider.getNetwork();
  const chainId = network.chainId;

  // Compute commitment hash
  const commitHash = ethers.keccak256(
    ethers.solidityPacked(
      ["string", "bytes", "address", "bytes32", "uint256"],
      [did, publicKey, userAddress, salt, chainId]
    )
  );

  // Submit commitment
  const tx = await sageRegistry.commitRegistration(commitHash);
  await tx.wait();

  console.log("✅ Commitment submitted");
  console.log("⏳ Wait 60 seconds before revealing...");

  // Store salt for reveal phase
  return { salt, commitHash };
}
```

### Step 3: Wait and Reveal

```javascript
async function revealAgentRegistration(
  sageRegistry,
  agentWallet,
  salt,
  agentDetails
) {
  // Wait minimum delay (60 seconds)
  await new Promise(resolve => setTimeout(resolve, 61000));

  // Prepare registration data
  const did = agentDetails.did;
  const name = agentDetails.name;
  const description = agentDetails.description;
  const endpoint = agentDetails.endpoint;
  const publicKey = agentWallet.publicKey;
  const capabilities = JSON.stringify(agentDetails.capabilities);

  // Create signature for key ownership proof
  const keyHash = ethers.keccak256(publicKey);
  const network = await agentWallet.provider.getNetwork();
  const challenge = ethers.keccak256(
    ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      [
        "SAGE Key Registration:",
        network.chainId,
        await sageRegistry.getAddress(),
        agentWallet.address,
        keyHash,
      ]
    )
  );

  // Sign with agent's private key
  const signature = await agentWallet.signMessage(
    ethers.getBytes(challenge)
  );

  // Reveal registration
  const tx = await sageRegistry.registerAgentWithReveal(
    did,
    name,
    description,
    endpoint,
    publicKey,
    capabilities,
    signature,
    salt
  );

  const receipt = await tx.wait();

  // Extract agentId from event
  const event = receipt.logs.find(
    log => log.fragment && log.fragment.name === "AgentRegistered"
  );
  const agentId = event.args.agentId;

  console.log("✅ Agent registered successfully!");
  console.log("Agent ID:", agentId);

  return agentId;
}
```

### Complete Registration Example

```javascript
async function registerAgent() {
  // 1. Generate agent wallet
  const agentWallet = ethers.Wallet.createRandom();

  // 2. Prepare agent details
  const agentDetails = {
    did: "did:sage:my-ai-agent",
    name: "My AI Agent",
    description: "An intelligent assistant for task automation",
    endpoint: "https://my-agent.example.com/api",
    capabilities: {
      chat: true,
      vision: false,
      codeGen: true,
    },
  };

  // 3. Commit registration
  const { salt } = await commitAgentRegistration(
    sageRegistry,
    agentDetails.did,
    agentWallet.publicKey,
    wallet.address,
    wallet
  );

  // 4. Reveal registration
  const agentId = await revealAgentRegistration(
    sageRegistry,
    agentWallet,
    salt,
    agentDetails
  );

  return { agentId, agentWallet };
}
```

---

## Task Validation

### Request Validation

```javascript
async function requestTaskValidation(
  validationRegistry,
  taskId,
  serverAgent,
  resultData
) {
  // Compute result hash
  const dataHash = ethers.keccak256(
    ethers.toUtf8Bytes(JSON.stringify(resultData))
  );

  // Set validation parameters
  const validationType = 0; // STAKE validation
  const deadline = Math.floor(Date.now() / 1000) + 86400; // 24 hours
  const requesterStake = ethers.parseEther("0.01"); // 0.01 ETH

  // Request validation
  const tx = await validationRegistry.requestValidation(
    taskId,
    serverAgent,
    dataHash,
    validationType,
    deadline,
    { value: requesterStake }
  );

  const receipt = await tx.wait();

  // Extract requestId from event
  const event = receipt.logs.find(
    log => log.fragment && log.fragment.name === "ValidationRequested"
  );
  const requestId = event.args.requestId;

  console.log("✅ Validation requested");
  console.log("Request ID:", requestId);

  return requestId;
}
```

### Submit Validation (Validator)

```javascript
async function submitValidation(
  validationRegistry,
  requestId,
  taskData,
  validatorWallet
) {
  // Re-execute task
  const myResult = await executeTask(taskData);

  // Compute hash of my result
  const computedHash = ethers.keccak256(
    ethers.toUtf8Bytes(JSON.stringify(myResult))
  );

  // Submit validation with stake
  const validatorStake = ethers.parseEther("0.1"); // 0.1 ETH

  const tx = await validationRegistry
    .connect(validatorWallet)
    .submitStakeValidation(requestId, computedHash, {
      value: validatorStake,
    });

  await tx.wait();

  console.log("✅ Validation submitted");
}
```

### Listen for Validation Events

```javascript
async function monitorValidationRequests(validationRegistry) {
  // Listen for new validation requests
  validationRegistry.on("ValidationRequested", async (
    requestId,
    taskId,
    serverAgent,
    dataHash,
    validationType,
    stake
  ) => {
    console.log("\n🔔 New validation request:");
    console.log("  Request ID:", requestId);
    console.log("  Task ID:", taskId);
    console.log("  Server Agent:", serverAgent);
    console.log("  Data Hash:", dataHash);
    console.log("  Stake:", ethers.formatEther(stake), "ETH");

    // Optionally: Auto-participate in validation
    // await submitValidation(validationRegistry, requestId, taskData, wallet);
  });

  // Listen for validation completion
  validationRegistry.on("ValidationFinalized", (
    requestId,
    result,
    consensusReached
  ) => {
    console.log("\n✅ Validation finalized:");
    console.log("  Request ID:", requestId);
    console.log("  Result:", result ? "SUCCESS" : "FAIL");
    console.log("  Consensus:", consensusReached ? "YES" : "NO");
  });
}
```

### Withdraw Rewards

```javascript
async function withdrawValidationRewards(validationRegistry, wallet) {
  // Check pending withdrawals
  const pending = await validationRegistry.pendingWithdrawals(
    wallet.address
  );

  if (pending > 0) {
    console.log("Pending rewards:", ethers.formatEther(pending), "ETH");

    // Withdraw
    const tx = await validationRegistry.withdraw();
    await tx.wait();

    console.log("✅ Rewards withdrawn");
  } else {
    console.log("No pending rewards");
  }
}
```

---

## Reputation Management

### Commit Task Authorization

```javascript
async function commitTaskAuthorization(
  reputationRegistry,
  taskId,
  serverAgent,
  deadline,
  wallet
) {
  // Generate salt
  const salt = ethers.randomBytes(32);

  // Get chainId
  const network = await wallet.provider.getNetwork();
  const chainId = network.chainId;

  // Compute commit hash
  const commitHash = ethers.keccak256(
    ethers.solidityPacked(
      ["bytes32", "address", "uint256", "bytes32", "uint256"],
      [taskId, serverAgent, deadline, salt, chainId]
    )
  );

  // Submit commitment
  const tx = await reputationRegistry.commitTaskAuthorization(commitHash);
  await tx.wait();

  console.log("✅ Task authorization committed");

  return { salt, commitHash };
}
```

### Reveal Task Authorization

```javascript
async function revealTaskAuthorization(
  reputationRegistry,
  taskId,
  serverAgent,
  deadline,
  salt,
  wallet
) {
  // Wait minimum delay (30 seconds)
  await new Promise(resolve => setTimeout(resolve, 31000));

  // Reveal authorization
  const tx = await reputationRegistry.authorizeTaskWithReveal(
    taskId,
    serverAgent,
    deadline,
    salt
  );

  await tx.wait();

  console.log("✅ Task authorization revealed");
}
```

### Query Agent Reputation

```javascript
async function getAgentReputation(reputationRegistry, agentAddress) {
  const reputation = await reputationRegistry.getAgentReputation(
    agentAddress
  );

  console.log("Agent Reputation:");
  console.log("  Score:", reputation.score.toString());
  console.log("  Total Feedbacks:", reputation.totalFeedbacks.toString());
  console.log("  Last Updated:", new Date(
    reputation.lastUpdated.toNumber() * 1000
  ).toISOString());

  return reputation;
}
```

### Query Task Feedback

```javascript
async function getTaskFeedback(reputationRegistry, taskId) {
  const feedbacks = await reputationRegistry.queryFeedback(taskId, 0, 100);

  console.log(`Found ${feedbacks.length} feedbacks for task ${taskId}`);

  feedbacks.forEach((feedback, index) => {
    console.log(`\nFeedback ${index + 1}:`);
    console.log("  Rating:", feedback.rating.toString());
    console.log("  Success:", feedback.success);
    console.log("  Validator:", feedback.validator);
    console.log("  Timestamp:", new Date(
      feedback.timestamp.toNumber() * 1000
    ).toISOString());
  });

  return feedbacks;
}
```

---

## TEE Key Governance

### Propose TEE Key

```javascript
async function proposeTEEKey(
  teeRegistry,
  teePublicKey,
  attestationURL,
  teeType,
  wallet
) {
  // Compute key hash
  const keyHash = ethers.keccak256(teePublicKey);

  // Proposal stake (1 ETH)
  const proposalStake = ethers.parseEther("1.0");

  // Submit proposal
  const tx = await teeRegistry.proposeTEEKey(
    keyHash,
    attestationURL,
    teeType,
    { value: proposalStake }
  );

  const receipt = await tx.wait();

  // Extract proposalId from event
  const event = receipt.logs.find(
    log => log.fragment && log.fragment.name === "TEEKeyProposed"
  );
  const proposalId = event.args.proposalId;

  console.log("✅ TEE key proposed");
  console.log("Proposal ID:", proposalId);
  console.log("⏳ Voting period: 7 days");

  return proposalId;
}
```

### Vote on Proposal

```javascript
async function voteOnProposal(teeRegistry, proposalId, support, wallet) {
  // Check if already voted
  const voteInfo = await teeRegistry.getVoteInfo(proposalId, wallet.address);

  if (voteInfo.hasVoted) {
    console.log("⚠️  You have already voted on this proposal");
    return;
  }

  // Cast vote
  const tx = await teeRegistry.vote(proposalId, support);
  await tx.wait();

  console.log("✅ Vote cast");
  console.log("  Support:", support ? "FOR" : "AGAINST");
}
```

### Execute Proposal

```javascript
async function executeProposal(teeRegistry, proposalId) {
  // Check proposal status
  const status = await teeRegistry.getProposalStatus(proposalId);

  console.log("Proposal Status:");
  console.log("  Votes For:", status.votesFor.toString());
  console.log("  Votes Against:", status.votesAgainst.toString());
  console.log("  Participation:", status.participationRate.toString(), "%");
  console.log("  Approval:", status.approvalRate.toString(), "%");
  console.log("  Can Execute:", status.canExecute);

  if (!status.canExecute) {
    console.log("⚠️  Cannot execute yet. Voting period not ended.");
    return;
  }

  // Execute proposal
  const tx = await teeRegistry.executeProposal(proposalId);
  const receipt = await tx.wait();

  // Check result
  const event = receipt.logs.find(
    log => log.fragment && log.fragment.name === "ProposalExecuted"
  );
  const approved = event.args.approved;

  if (approved) {
    console.log("✅ Proposal APPROVED - TEE key is now trusted");
  } else {
    console.log("❌ Proposal REJECTED - Stake slashed");
  }

  return approved;
}
```

---

## Code Examples

### Complete Client Flow

```javascript
// complete-client-flow.js

const { ethers } = require("ethers");

async function main() {
  // 1. Setup
  const provider = new ethers.JsonRpcProvider(process.env.SEPOLIA_RPC_URL);
  const wallet = new ethers.Wallet(process.env.PRIVATE_KEY, provider);

  // Load contracts
  const sageRegistry = new ethers.Contract(
    SAGE_REGISTRY_ADDRESS,
    SAGE_REGISTRY_ABI,
    wallet
  );
  const validationRegistry = new ethers.Contract(
    VALIDATION_REGISTRY_ADDRESS,
    VALIDATION_REGISTRY_ABI,
    wallet
  );

  // 2. Register Agent (if not already registered)
  console.log("Step 1: Registering agent...");
  const { agentId, agentWallet } = await registerAgent();

  // 3. Execute Task
  console.log("\nStep 2: Executing task...");
  const taskId = ethers.randomBytes(32);
  const taskResult = await executeMyTask(taskId);

  // 4. Request Validation
  console.log("\nStep 3: Requesting validation...");
  const requestId = await requestTaskValidation(
    validationRegistry,
    taskId,
    agentWallet.address,
    taskResult
  );

  // 5. Wait for Validators
  console.log("\nStep 4: Waiting for validators...");
  await monitorValidationRequests(validationRegistry);

  // 6. Check Result (after finalization)
  console.log("\nStep 5: Checking validation result...");
  // Event listeners will notify when finalized

  console.log("\n✅ Complete flow executed successfully!");
}

main().catch(console.error);
```

### Complete Validator Flow

```javascript
// complete-validator-flow.js

async function validatorMain() {
  // 1. Setup
  const provider = new ethers.JsonRpcProvider(process.env.SEPOLIA_RPC_URL);
  const validatorWallet = new ethers.Wallet(
    process.env.VALIDATOR_PRIVATE_KEY,
    provider
  );

  const validationRegistry = new ethers.Contract(
    VALIDATION_REGISTRY_ADDRESS,
    VALIDATION_REGISTRY_ABI,
    validatorWallet
  );

  // 2. Register as agent (validator must be registered)
  console.log("Registering validator agent...");
  await registerAgent();

  // 3. Listen for validation requests
  console.log("Listening for validation requests...");

  validationRegistry.on("ValidationRequested", async (
    requestId,
    taskId,
    serverAgent,
    dataHash
  ) => {
    try {
      console.log(`\n🔔 New validation request: ${requestId}`);

      // 4. Fetch task parameters (from off-chain source)
      const taskParams = await fetchTaskParameters(taskId);

      // 5. Re-execute task
      console.log("Re-executing task...");
      const myResult = await executeTask(taskParams);

      // 6. Submit validation
      console.log("Submitting validation...");
      await submitValidation(
        validationRegistry,
        requestId,
        myResult,
        validatorWallet
      );

      console.log("✅ Validation submitted successfully");

    } catch (error) {
      console.error("Error processing validation:", error);
    }
  });

  // 7. Periodically withdraw rewards
  setInterval(async () => {
    await withdrawValidationRewards(validationRegistry, validatorWallet);
  }, 3600000); // Every hour

  console.log("✅ Validator is now active and listening...");
}

validatorMain().catch(console.error);
```

---

## Best Practices

### Security

1. **Never expose private keys**
   ```javascript
   // ❌ Bad
   const PRIVATE_KEY = "0x1234...";

   // ✅ Good
   const PRIVATE_KEY = process.env.PRIVATE_KEY;
   ```

2. **Always use commit-reveal for sensitive operations**
   ```javascript
   // ✅ Register agents with commit-reveal
   await commitRegistration(hash);
   await sleep(61000); // Wait minimum delay
   await registerWithReveal(params, salt);
   ```

3. **Validate all inputs before submitting transactions**
   ```javascript
   // ✅ Validate before submission
   if (!ethers.isAddress(serverAgent)) {
     throw new Error("Invalid server agent address");
   }
   if (deadline <= Date.now() / 1000) {
     throw new Error("Deadline must be in the future");
   }
   ```

4. **Handle transaction failures gracefully**
   ```javascript
   try {
     const tx = await contract.someFunction();
     await tx.wait();
   } catch (error) {
     if (error.code === 'INSUFFICIENT_FUNDS') {
       console.error("Not enough ETH for gas");
     } else {
       console.error("Transaction failed:", error.message);
     }
   }
   ```

### Gas Optimization

1. **Batch read operations**
   ```javascript
   // ❌ Multiple calls
   const agent1 = await registry.getAgent(id1);
   const agent2 = await registry.getAgent(id2);

   // ✅ Batch with multicall
   const [agent1, agent2] = await Promise.all([
     registry.getAgent(id1),
     registry.getAgent(id2),
   ]);
   ```

2. **Use appropriate gas limits**
   ```javascript
   const tx = await contract.someFunction({
     gasLimit: 300000, // Set reasonable limit
   });
   ```

3. **Monitor gas prices**
   ```javascript
   const feeData = await provider.getFeeData();
   console.log("Current gas price:", ethers.formatUnits(feeData.gasPrice, "gwei"), "gwei");
   ```

### Error Handling

1. **Check for custom errors**
   ```javascript
   try {
     await registry.commitRegistration(hash);
   } catch (error) {
     if (error.message.includes("AlreadyCommitted")) {
       console.log("You have an active commitment");
     } else if (error.message.includes("InvalidCommitHash")) {
       console.log("Invalid commit hash");
     } else {
       throw error;
     }
   }
   ```

2. **Wait for confirmations on critical operations**
   ```javascript
   const tx = await registry.registerAgent(...);
   const receipt = await tx.wait(2); // Wait for 2 confirmations
   ```

### Testing

1. **Test on testnet first**
   ```javascript
   // Always test on Sepolia before mainnet
   const NETWORK = process.env.NETWORK || "sepolia";
   ```

2. **Use realistic test data**
   ```javascript
   const testAgent = {
     did: "did:sage:test-agent-" + Date.now(),
     name: "Test Agent",
     description: "For testing only",
     endpoint: "https://test.example.com",
   };
   ```

---

## Troubleshooting

### Common Issues

#### Issue: "Request not found"

**Cause**: Trying to submit validation for non-existent request

**Solution**:
```javascript
// Check if request exists
const request = await validationRegistry.getValidationRequest(requestId);
if (request.timestamp === 0) {
  console.error("Request does not exist");
}
```

#### Issue: "RevealTooSoon" or "RevealTooLate"

**Cause**: Not waiting minimum delay or exceeding maximum delay

**Solution**:
```javascript
// Check commitment timing
const commitment = await registry.getCommitment(wallet.address);
const now = Math.floor(Date.now() / 1000);
const minTime = commitment.timestamp + 60; // MIN_DELAY
const maxTime = commitment.timestamp + 3600; // MAX_DELAY

if (now < minTime) {
  console.log(`Wait ${minTime - now} more seconds`);
} else if (now > maxTime) {
  console.error("Commitment expired, create new one");
}
```

#### Issue: "Maximum validators reached"

**Cause**: Too many validators already submitted

**Solution**:
```javascript
// Check validator count before submitting
const responses = await validationRegistry.getValidationResponses(requestId);
if (responses.length >= 100) {
  console.log("Maximum validators reached for this request");
}
```

#### Issue: Transaction reverts with no reason

**Cause**: Gas estimation failure or contract logic error

**Solution**:
```javascript
// Use try-catch and check contract state
try {
  const tx = await contract.someFunction({
    gasLimit: 500000, // Manual gas limit
  });
  await tx.wait();
} catch (error) {
  console.error("Full error:", error);

  // Check if agent is active
  const isActive = await registry.isAgentActive(agentId);
  console.log("Agent active:", isActive);
}
```

### Getting Help

1. **Check documentation**: Review NatSpec comments in contract code
2. **View on Etherscan**: Inspect transactions and events
3. **GitHub Issues**: https://github.com/sage-x-project/sage/issues
4. **Discord**: Join our developer community

---

## Summary

This integration guide covers:

✅ **Environment Setup**: Tools and configuration
✅ **Agent Registration**: Complete commit-reveal flow
✅ **Task Validation**: Request, submit, and monitor
✅ **Reputation Management**: Authorization and queries
✅ **Governance**: TEE key proposals and voting
✅ **Code Examples**: Complete client and validator flows
✅ **Best Practices**: Security, gas optimization, testing
✅ **Troubleshooting**: Common issues and solutions

**Next Steps**:
1. Set up your development environment
2. Register a test agent on Sepolia
3. Submit a validation request
4. Monitor events and withdraw rewards

For detailed API documentation, see the NatSpec comments in the contract source code.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-07
**Status**: ✅ Complete
