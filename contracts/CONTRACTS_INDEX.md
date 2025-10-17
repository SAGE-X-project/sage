# SAGE Contracts - Documentation Index

**Last Updated**: 2025-01-18
**Current Version**: SageRegistryV4 (Multi-Key Support)

This index provides an organized overview of all SAGE smart contract documentation.

---

## Quick Navigation

### For New Users
1. Start with [README.md](#readmemd) for overview
2. Read [MULTI_KEY_DESIGN.md](#multi_key_designmd) for V4 architecture
3. Check [ethereum/README.md](#ethereumreadmemd) for implementation details

### For Developers
1. Review [ROADMAP.md](#roadmapmd) for planned features
2. Check [TODO.md](#todomd) for current tasks
3. See [DEPLOYMENT_GUIDE.md](#deployment_guidemd) for deployment process

### For Auditors/Reviewers
1. Read [MULTI_KEY_DESIGN.md](#multi_key_designmd) for security design
2. Review [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd) for historical findings
3. Check [archived/CODE_ANALYSIS_V1_V2_V3.md](#archivedcode_analysis_v1_v2_v3md) for legacy analysis

---

## Active Documentation

### README.md
**Status**: Active - Recently Updated
**Version**: V4
**Purpose**: Main contracts overview and feature documentation

**Contents**:
- Overview of SAGE smart contracts
- V4 multi-key features (Ed25519, ECDSA, X25519)
- V2 stable features
- Architecture overview
- Contract addresses (testnet/mainnet)
- Installation and quick start
- Deployment process
- Testing guide
- Gas optimization details
- Migration guide (V1 → V2 → V4)
- Security features

**When to Read**: First document to read for understanding SAGE contracts

**Related Documents**:
- [ethereum/README.md](#ethereumreadmemd) - Implementation details
- [MULTI_KEY_DESIGN.md](#multi_key_designmd) - V4 architecture
- [DEPLOYMENT_GUIDE.md](#deployment_guidemd) - Deployment instructions

---

### TODO.md
**Status**: Active - Recently Restructured
**Version**: V4
**Purpose**: Track active tasks and current sprint progress

**Contents**:
- High priority tasks (Multi-key CLI, key management, contract deployment)
- Medium priority tasks (A2A integration, enhanced validation)
- Low priority tasks (contract cleanup, documentation updates)
- Current sprint status
- Completed work tracker
- Progress tracking (65% overall completion)
- Links to detailed implementation plans in ROADMAP.md

**When to Read**: When you want to see current development status and next steps

**Related Documents**:
- [ROADMAP.md](#roadmapmd) - Detailed feature plans
- [README.md](#readmemd) - Context for tasks

---

### ROADMAP.md
**Status**: Active
**Version**: V4
**Purpose**: Detailed feature roadmap with implementation plans

**Contents**:
- **Feature 1**: Multi-Key Registration CLI Support (60-90 min)
- **Feature 2**: Key Management CLI Commands (40-60 min)
- **Feature 3**: A2A Integration Examples (50-70 min)
- **Feature 4**: Smart Contract Deployment & Integration (90-120 min)
- **Feature 5**: Enhanced Validation (40-60 min)
- Implementation priorities (Phase 1, 2, 3)
- Success metrics
- Files to modify for each feature
- Effort estimates

**When to Read**: When planning implementation of specific features

**Related Documents**:
- [TODO.md](#todomd) - Current task status
- [MULTI_KEY_DESIGN.md](#multi_key_designmd) - Design specification

---

### MULTI_KEY_DESIGN.md
**Status**: Active
**Version**: V4
**Purpose**: Technical specification for SageRegistryV4 multi-key architecture

**Contents**:
- Design motivation and objectives
- Multi-key architecture overview
- Supported key types (Ed25519, ECDSA, X25519)
- Type-specific verification mechanisms
- Storage optimization strategies
- Security considerations
- Gas cost analysis
- API design
- Migration path from V3
- Implementation details

**When to Read**: When you need to understand V4 technical design and security model

**Related Documents**:
- [README.md](#readmemd) - Feature overview
- [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd) - Security context

---

### DEPLOYMENT_GUIDE.md
**Status**: Active - V2 Focused
**Version**: V2 (V4 update pending)
**Purpose**: Step-by-step deployment instructions

**Contents**:
- Quick start (local, testnet, production)
- Project structure
- Configuration (environment variables, networks)
- Deployment process
- Agent management
- Verification procedures
- Security considerations
- Troubleshooting
- Integration with Go applications

**When to Read**: When deploying contracts to any network

**Note**: Currently V2 focused, will be updated when V4 deployment is ready

**Related Documents**:
- [README.md](#readmemd) - General deployment overview
- [ethereum/README.md](#ethereumreadmemd) - Ethereum-specific details

---

### ethereum/README.md
**Status**: Active - Recently Updated
**Version**: V4
**Purpose**: Ethereum/EVM-specific implementation guide

**Contents**:
- Implementation status (V4, V2, V1)
- Quick start guide
- Project structure (contracts, scripts, tests)
- Available scripts (shell scripts, npm commands)
- Testing guide (local, testnet, integration)
- Deployment process
- Network configuration
- Security features (V4 and V2)
- Gas usage details
- Troubleshooting
- Cross-references to parent documentation

**When to Read**: When working with Ethereum contract implementation

**Related Documents**:
- [README.md](#readmemd) - Main contracts documentation
- [DEPLOYMENT_GUIDE.md](#deployment_guidemd) - Deployment details

---

## Archived Documentation

### archived/CODE_ANALYSIS_V1_V2_V3.md
**Status**: Archived - Historical Reference
**Version**: V1/V2/V3
**Original File**: contracts/TODO.md (3,103 lines)
**Purpose**: Comprehensive code analysis of legacy contracts

**Contents**:
- Security vulnerability analysis
  - Ed25519 signature bypass (V1) - CRITICAL
  - Reentrancy risks
  - Access control issues
- Clean code principles review
- SOLID principles violations
- Code quality assessment
- Gas optimization opportunities
- Refactoring recommendations
- Detailed code review of V1, V2, V3

**When to Read**: When researching historical issues or understanding why V4 was needed

**Related Documents**:
- [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd) - Security summary
- [MULTI_KEY_DESIGN.md](#multi_key_designmd) - How V4 addresses issues

---

### archived/SECURITY_AUDIT_LEGACY.md
**Status**: Archived - Historical Reference
**Version**: V1/V2/V3
**Purpose**: Security audit findings for legacy contracts

**Contents**:
- Executive summary
- Critical issue: Ed25519 signature bypass (V1)
- High priority issues (5 issues)
- Medium priority issues (8 issues)
- Low priority issues (12 issues)
- Migration path (V1/V2/V3 → V4)
- Status of all findings (26/26 addressed in V4)
- Recommendations

**When to Read**: When reviewing security considerations or audit history

**Related Documents**:
- [archived/CODE_ANALYSIS_V1_V2_V3.md](#archivedcode_analysis_v1_v2_v3md) - Detailed analysis
- [MULTI_KEY_DESIGN.md](#multi_key_designmd) - V4 security design

---

### archived/LOCAL_CONTRACT_INFO.md
**Status**: Archived - Outdated
**Version**: V2
**Original Location**: contracts/ethereum/LOCAL_CONTRACT_INFO.md
**Purpose**: Local deployment information (now outdated)

**Contents**:
- Network information (localhost:8545)
- Contract addresses (hardcoded from specific deployment)
- ABI file paths (hardcoded to specific user: /Users/0xtopaz/...)
- Test accounts and private keys
- Example code (JavaScript, Python, curl)
- Important notes about V2 requirements

**Why Archived**: Contains hardcoded absolute paths for specific user, V2 only, no V4 information

**When to Read**: Not recommended - use current deployment documentation instead

**Alternative**: Use [DEPLOYMENT_GUIDE.md](#deployment_guidemd) and [ethereum/README.md](#ethereumreadmemd)

---

## Contract Files Reference

### Smart Contracts (contracts/ethereum/contracts/)

#### V4 Contracts (Latest - In Development)
- **SageRegistryV4.sol** - Multi-key registry implementation
- **ISageRegistryV4.sol** - V4 interface
- Status: Complete, 30 unit tests passing, pending deployment

#### V2 Contracts (Stable Production)
- **SageRegistryV2.sol** - Enhanced registry with 5-step validation
- **SageVerificationHook.sol** - Hook implementation
- **ISageRegistry.sol** - V2 interface
- **IRegistryHook.sol** - Hook interface
- Status: Deployed on Sepolia testnet

#### V1 Contracts (Deprecated)
- **SageRegistry.sol** - Original implementation
- Status: Archived, not recommended for use

### Test Files (contracts/ethereum/test/)
- **SageRegistryV4.test.js** - V4 unit tests (30 tests)
- **SageRegistryV2.test.js** - V2 unit tests
- **integration-v2.test.js** - V2 integration tests
- **SageRegistry.test.fixed.js** - V1 compatibility tests

---

## Version History

### V4 (Current - In Development)
**Status**: Smart contract complete, pending deployment
**Documentation**: Complete
**Key Features**: Multi-key support (Ed25519, ECDSA, X25519), A2A protocol

**Key Documents**:
- [MULTI_KEY_DESIGN.md](#multi_key_designmd)
- [ROADMAP.md](#roadmapmd)
- [README.md](#readmemd) (V4 section)

### V3 (Legacy)
**Status**: Superseded by V4
**Key Features**: Commit-reveal pattern for front-running protection

### V2 (Stable Production)
**Status**: Production ready, deployed on Sepolia
**Key Features**: 5-step public key validation, key revocation

**Key Documents**:
- [README.md](#readmemd) (V2 section)
- [ethereum/README.md](#ethereumreadmemd)

### V1 (Deprecated)
**Status**: Archived, critical vulnerabilities
**Key Issues**: Ed25519 signature bypass

**Key Documents**:
- [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd)
- [archived/CODE_ANALYSIS_V1_V2_V3.md](#archivedcode_analysis_v1_v2_v3md)

---

## Recommended Reading Order

### For Understanding the System
1. [README.md](#readmemd) - Overview and features
2. [MULTI_KEY_DESIGN.md](#multi_key_designmd) - V4 architecture
3. [ethereum/README.md](#ethereumreadmemd) - Implementation details
4. [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd) - Security context

### For Contributing Development
1. [TODO.md](#todomd) - Current tasks
2. [ROADMAP.md](#roadmapmd) - Planned features
3. [MULTI_KEY_DESIGN.md](#multi_key_designmd) - Technical spec
4. [ethereum/README.md](#ethereumreadmemd) - Development setup

### For Deployment
1. [DEPLOYMENT_GUIDE.md](#deployment_guidemd) - Deployment process
2. [ethereum/README.md](#ethereumreadmemd) - Network configuration
3. [README.md](#readmemd) - Contract addresses and verification

### For Security Review
1. [MULTI_KEY_DESIGN.md](#multi_key_designmd) - Security design
2. [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd) - Historical findings
3. [archived/CODE_ANALYSIS_V1_V2_V3.md](#archivedcode_analysis_v1_v2_v3md) - Detailed analysis
4. [README.md](#readmemd) - Current security features

---

## Document Maintenance

### Update Frequency

**High Frequency** (Updated with each PR):
- [TODO.md](#todomd) - Task tracking
- [README.md](#readmemd) - Feature changes

**Medium Frequency** (Updated per feature):
- [ROADMAP.md](#roadmapmd) - Feature planning
- [ethereum/README.md](#ethereumreadmemd) - Implementation changes

**Low Frequency** (Updated per major version):
- [MULTI_KEY_DESIGN.md](#multi_key_designmd) - Architecture changes
- [DEPLOYMENT_GUIDE.md](#deployment_guidemd) - Deployment process changes

**Archived** (No updates):
- [archived/CODE_ANALYSIS_V1_V2_V3.md](#archivedcode_analysis_v1_v2_v3md)
- [archived/SECURITY_AUDIT_LEGACY.md](#archivedsecurity_audit_legacymd)
- [archived/LOCAL_CONTRACT_INFO.md](#archivedlocal_contract_infomd)

### Maintainers
- Documentation is maintained by SAGE development team
- All documentation changes should be included in PRs
- Follow the no-emoji policy established 2025-01-18

---

## External Resources

### SAGE Project
- Main Repository: [github.com/sage-x-project/sage](https://github.com/sage-x-project/sage)
- Main README: [../../README.md](../../README.md)

### Standards and Specifications
- Google A2A Protocol: Agent-to-Agent interoperability standard
- DID (Decentralized Identifiers): W3C standard
- HPKE (Hybrid Public Key Encryption): RFC 9180

### Development Tools
- Hardhat: [hardhat.org](https://hardhat.org)
- OpenZeppelin: [docs.openzeppelin.com](https://docs.openzeppelin.com)
- Solidity: [docs.soliditylang.org](https://docs.soliditylang.org)

---

**For questions or suggestions about this documentation index, please open a GitHub issue.**
