# ERC-8004 Community Testing Guide

Welcome to the SAGE ERC-8004 community testing period! This guide will help you test the newly deployed contracts on Sepolia testnet.

**Testing Period**: October 6, 2025 - TBD
**Network**: Ethereum Sepolia Testnet
**Status**: 🔴 ACTIVE TESTING

---

## 🎯 Testing Objectives

1. Validate all ERC-8004 functionality
2. Test integration with existing SAGE infrastructure
3. Identify bugs and edge cases
4. Measure gas costs in real scenarios
5. Gather feedback for mainnet deployment

---

## 📋 Prerequisites

### Required

- [ ] Metamask or compatible Web3 wallet
- [ ] Sepolia ETH (0.1+ ETH recommended)
- [ ] Basic understanding of smart contracts
- [ ] GitHub account (for reporting issues)

### Get Sepolia ETH

Get testnet ETH from these faucets:

1. **Alchemy Faucet**: https://www.alchemy.com/faucets/ethereum-sepolia
2. **QuickNode Faucet**: https://faucet.quicknode.com/ethereum/sepolia
3. **Sepolia PoW Faucet**: https://sepolia-faucet.pk910.de/

---

## 📜 Deployed Contracts

### Contract Addresses

```
SageRegistryV2:            0xb25D5f59cA52532862dA92901a2A550A09d5b4c0
ERC8004IdentityRegistry:   0xffEE59C558544f5d62CaAb9cF9b5Cb134F8808a6
ERC8004ReputationRegistry: 0xb8a3Fd16eEbB27BE8Aa6baB176C6AEED77fABE5f
ERC8004ValidationRegistry: 0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE
```

### Etherscan Links

- [SageRegistryV2](https://sepolia.etherscan.io/address/0xb25D5f59cA52532862dA92901a2A550A09d5b4c0)
- [ERC8004IdentityRegistry](https://sepolia.etherscan.io/address/0xffEE59C558544f5d62CaAb9cF9b5Cb134F8808a6)
- [ERC8004ReputationRegistry](https://sepolia.etherscan.io/address/0xb8a3Fd16eEbB27BE8Aa6baB176C6AEED77fABE5f)
- [ERC8004ValidationRegistry](https://sepolia.etherscan.io/address/0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE)

---

## 🧪 Testing Scenarios

### Level 1: Basic Functionality (Easy)

#### Test 1.1: Agent Registration

**Goal**: Register a test agent in SageRegistryV2

**Steps**:
1. Go to [SageRegistryV2 on Etherscan](https://sepolia.etherscan.io/address/0xb25D5f59cA52532862dA92901a2A550A09d5b4c0#writeContract)
2. Connect your wallet
3. Call `registerAgent()` with test data
4. Verify registration in "Read Contract"

**Expected**: Transaction succeeds, agent appears in registry

**Report**: Screenshot + transaction hash

#### Test 1.2: Agent Resolution

**Goal**: Query registered agent information

**Steps**:
1. Go to [ERC8004IdentityRegistry](https://sepolia.etherscan.io/address/0xffEE59C558544f5d62CaAb9cF9b5Cb134F8808a6#readContract)
2. Call `resolveAgent()` with your agent DID
3. Call `resolveAgentByAddress()` with your wallet address
4. Verify returned information is correct

**Expected**: Correct agent info returned

**Report**: DID + returned data

#### Test 1.3: Check Agent Status

**Goal**: Verify agent active status

**Steps**:
1. Call `isAgentActive()` with your DID
2. Verify returns `true`
3. Call `deactivateAgent()`
4. Check status again, verify returns `false`

**Expected**: Status changes correctly

**Report**: Before/after status + tx hash

---

### Level 2: Reputation System (Medium)

#### Test 2.1: Task Authorization

**Goal**: Pre-authorize a task for feedback

**Steps**:
1. Go to [ERC8004ReputationRegistry](https://sepolia.etherscan.io/address/0xb8a3Fd16eEbB27BE8Aa6baB176C6AEED77fABE5f#writeContract)
2. Generate a unique task ID (use `ethers.id("task-yourname-1")`)
3. Call `authorizeTask()` with:
   - taskId: your task ID
   - serverAgent: another test agent address
   - deadline: timestamp 1 hour from now
4. Verify authorization with `isTaskAuthorized()`

**Expected**: Task authorized successfully

**Report**: Task ID + tx hash

#### Test 2.2: Submit Feedback

**Goal**: Submit feedback for authorized task

**Steps**:
1. Use task from Test 2.1
2. Call `submitFeedback()` with:
   - taskId: your authorized task
   - serverAgent: same as authorization
   - dataHash: `ethers.id("test-output")`
   - rating: 85 (0-100)
3. Check feedback count with `getAgentFeedbackCount()`

**Expected**: Feedback submitted, count increases

**Report**: Feedback ID + tx hash

#### Test 2.3: Query Feedback

**Goal**: Retrieve submitted feedback

**Steps**:
1. Call `getAgentFeedback()` with:
   - agentAddress: server agent address
   - offset: 0
   - limit: 10
2. Call `getTaskFeedback()` with your task ID
3. Verify feedback appears in results

**Expected**: Feedback retrieved correctly

**Report**: Query results screenshot

---

### Level 3: Validation System (Advanced)

#### Test 3.1: Request Validation

**Goal**: Create a validation request

**Steps**:
1. Go to [ERC8004ValidationRegistry](https://sepolia.etherscan.io/address/0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE#writeContract)
2. Call `requestValidation()` with:
   - taskId: your task ID
   - serverAgent: agent address
   - dataHash: `ethers.id("validation-test")`
   - validationType: 1 (STAKE)
   - deadline: timestamp 1 hour from now
   - value: 0.01 ETH (min stake)
3. Note the returned `requestId`

**Expected**: Validation request created

**Report**: Request ID + tx hash

#### Test 3.2: Submit Validator Response

**Goal**: Validate a task as a validator

**Steps**:
1. From a DIFFERENT wallet:
2. Call `submitStakeValidation()` with:
   - requestId: from Test 3.1
   - computedHash: same as request dataHash
   - value: 0.1 ETH (validator stake)
3. Check request status with `isValidationComplete()`

**Expected**: Validation submitted, status updated

**Report**: Response tx hash + final status

#### Test 3.3: Check Validator Stats

**Goal**: View validator performance metrics

**Steps**:
1. Call `getValidatorStats()` with validator address
2. Verify stats show:
   - Total validations: 1
   - Successful validations: 1
   - Rewards earned

**Expected**: Stats reflect completed validation

**Report**: Screenshot of stats

---

### Level 4: Integration Testing (Expert)

#### Test 4.1: Full Lifecycle

**Goal**: Complete end-to-end ERC-8004 flow

**Steps**:
1. Register two agents (client + server)
2. Authorize task
3. Request validation
4. Submit validator response
5. Submit client feedback
6. Query all results

**Expected**: Complete flow works seamlessly

**Report**: All transaction hashes in sequence

#### Test 4.2: Edge Cases

Test these scenarios:

- [ ] Expired task authorization
- [ ] Duplicate feedback submission
- [ ] Insufficient validator stake
- [ ] Incorrect validation hash
- [ ] Validation after deadline

**Expected**: Proper error messages

**Report**: Error scenarios + messages

#### Test 4.3: Gas Optimization

**Goal**: Measure real-world gas costs

Track gas used for:
- Agent registration
- Task authorization
- Feedback submission
- Validation request
- Validator response

**Report**: Gas usage table

---

## 🐛 Bug Reporting

### How to Report Issues

1. **Check Existing Issues**: https://github.com/SAGE-X-project/sage/issues
2. **Create New Issue** with labels: `erc-8004`, `testnet`, `bug`
3. **Include**:
   - Test scenario number
   - Expected vs actual behavior
   - Transaction hash
   - Wallet address (if relevant)
   - Screenshots
   - Error messages

### Issue Template

```markdown
### Test Scenario
Level X, Test X.X: [Test Name]

### Expected Behavior
[What should happen]

### Actual Behavior
[What actually happened]

### Transaction Hash
0x...

### Environment
- Network: Sepolia
- Wallet: Metamask X.X.X
- Browser: Chrome X.X.X

### Screenshots
[Attach screenshots]

### Additional Context
[Any other relevant information]
```

---

## 💡 Tips & Best Practices

### Testing Tips

1. **Start Simple**: Begin with Level 1 tests
2. **Document Everything**: Screenshot + tx hash for each test
3. **Test Failures**: Try to break things (within reason)
4. **Multiple Accounts**: Test with 2-3 different wallets
5. **Edge Cases**: Test boundary conditions

### Common Issues

**"Transaction Reverted"**
- Check you have enough ETH for gas
- Verify function parameters are correct
- Ensure prerequisites are met (e.g., task authorized)

**"Key ownership not proven"**
- Make sure you're using the wallet that created the agent
- Check signature is valid

**"Task already authorized"**
- Use a unique task ID for each test
- Format: `ethers.id("task-yourname-uniquenumber")`

---

## 📊 Testing Progress Tracker

Track your progress:

```
Level 1: Basic Functionality
[ ] 1.1 Agent Registration
[ ] 1.2 Agent Resolution
[ ] 1.3 Check Agent Status

Level 2: Reputation System
[ ] 2.1 Task Authorization
[ ] 2.2 Submit Feedback
[ ] 2.3 Query Feedback

Level 3: Validation System
[ ] 3.1 Request Validation
[ ] 3.2 Submit Validator Response
[ ] 3.3 Check Validator Stats

Level 4: Integration Testing
[ ] 4.1 Full Lifecycle
[ ] 4.2 Edge Cases
[ ] 4.3 Gas Optimization
```

---

## 🎁 Incentives

### Testing Rewards

TBD - Potential rewards for:
- Most thorough testing
- Critical bug discovery
- Best documentation
- Creative test scenarios

### Recognition

Contributors will be:
- Listed in release notes
- Acknowledged in documentation
- Considered for early mainnet access

---

## 📅 Timeline

### Week 1-2: Initial Testing
- Basic functionality tests
- Bug discovery and fixes
- Documentation improvements

### Week 3-4: Advanced Testing
- Integration testing
- Load testing
- Gas optimization

### Week 5+: Preparation
- Security audit (if scheduled)
- Mainnet deployment planning
- Final documentation review

---

## 🤝 Community

### Get Help

- **Discord**: [Join SAGE Discord](#)
- **GitHub Discussions**: https://github.com/SAGE-X-project/sage/discussions
- **Documentation**: `/contracts/ethereum/docs/`

### Stay Updated

- **GitHub**: Watch the repository for updates
- **Twitter**: Follow @SAGE_X_Project
- **Blog**: [SAGE Blog](#)

---

## 📚 Additional Resources

### Documentation

- [ERC-8004 Standard](https://eips.ethereum.org/EIPS/eip-8004)
- [Deployment Documentation](./SEPOLIA-DEPLOYMENT.md)
- [Implementation Summary](./ERC-8004-IMPLEMENTATION-SUMMARY.md)
- [Architecture](./ERC-8004-ARCHITECTURE.md)

### Tools

- **Etherscan**: Contract interaction UI
- **Hardhat**: Local testing framework
- **Ethers.js**: JavaScript library for Web3
- **Metamask**: Browser wallet

---

## ✅ Completion

After completing all tests:

1. **Submit Summary**: Create issue with "testing-complete" label
2. **Include**:
   - All transaction hashes
   - Test results summary
   - Bugs found (if any)
   - Suggestions for improvement
3. **Feedback**: Share your experience

---

**Thank you for helping test SAGE's ERC-8004 implementation!** 🙏

Your testing helps ensure a robust, secure mainnet deployment.

---

*Last Updated: 2025-10-06*
*Version: 1.0*
*Maintainer: SAGE Development Team*
