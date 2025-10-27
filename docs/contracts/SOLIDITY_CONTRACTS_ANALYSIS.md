# SAGE Smart Contracts Analysis

## Executive Summary

The SAGE project implements a comprehensive smart contract system for decentralized AI agent registration and management on Ethereum. The architecture consists of:

1. **Core Registry**: AgentCardRegistry - Production-ready agent registry with multi-key support
2. **Storage Layer**: AgentCardStorage - Isolated storage for future upgradability
3. **Hook System**: AgentCardVerifyHook - External verification and anti-fraud checks
4. **Governance**: TEEKeyRegistry, SimpleMultiSig, TimelockController
5. **Standards Compliance**: ERC-8004 Identity Registry interface implementation

---

## 1. Main Contract Files Overview

### 1.1 AgentCardRegistry.sol
**Location**: `contracts/ethereum/contracts/AgentCardRegistry.sol`

**Purpose**: Production SAGE registry combining V2, V3, V4 features with ERC-8004 compliance

**Inheritance Chain**:
- AgentCardStorage (data layer)
- IERC8004IdentityRegistry (standard interface)
- Pausable (OpenZeppelin)
- ReentrancyGuard (security)
- Ownable2Step (two-step ownership)

**Key Features**:
- Multi-key support (ECDSA, Ed25519, X25519)
- Commit-reveal pattern (prevents front-running)
- Cross-chain replay protection via chainId
- Rate limiting and anti-Sybil measures
- Emergency pause mechanism
- Stake requirement (0.01 ETH)
- Time-locked activation (1 hour delay)
- ERC-8004 compliant interface

---

### 1.2 AgentCardStorage.sol
**Location**: `contracts/ethereum/contracts/AgentCardStorage.sol`

**Purpose**: Isolated storage layer enabling future upgrades without data migration

**Key Structures**:

#### KeyType Enum (3 types)
```solidity
enum KeyType {
    ECDSA,      // secp256k1 for Ethereum compatibility
    Ed25519,    // EdDSA for high-performance signing
    X25519      // ECDH for encryption/key exchange
}
```

#### AgentMetadata Struct
```solidity
struct AgentMetadata {
    string did                 // W3C DID identifier
    string name                // Human-readable name
    string description         // Agent description
    string endpoint            // AgentCard URL or IPFS hash
    bytes32[] keyHashes        // Array of key hashes (max 10)
    string capabilities        // JSON-encoded capabilities
    address owner              // Agent owner address
    uint256 registeredAt       // Registration timestamp
    uint256 updatedAt          // Last update timestamp
    bool active                // Agent active status (time-locked)
    uint256 chainId            // Chain ID (replay protection)
}
```

#### AgentKey Struct
```solidity
struct AgentKey {
    KeyType keyType            // Type of cryptographic key
    bytes keyData              // Raw public key bytes
    bytes signature            // Ownership proof signature
    bool verified              // Verification status
    uint256 registeredAt       // Key registration timestamp
}
```

#### RegistrationParams Struct
```solidity
struct RegistrationParams {
    string did
    string name
    string description
    string endpoint
    string capabilities
    bytes[] keys               // Public key bytes
    KeyType[] keyTypes         // Type of each key
    bytes[] signatures         // Ownership proofs
    bytes32 salt               // Salt for commit-reveal
}
```

#### RegistrationCommitment Struct (Commit-Reveal Pattern)
```solidity
struct RegistrationCommitment {
    bytes32 commitHash         // keccak256(did, keys, owner, salt, chainId)
    uint256 timestamp          // Commitment timestamp
    bool revealed              // Whether commitment has been revealed
}
```

**Key Constants**:
- `COMMIT_MIN_DELAY = 1 minute` - Prevents instant reveal attacks
- `COMMIT_MAX_DELAY = 1 hour` - Prevents commitment squatting
- `MAX_KEYS_PER_AGENT = 10` - Prevents unbounded gas costs
- `MAX_DAILY_REGISTRATIONS = 24` - Rate limiting per address per day

**Storage Mappings**:
- `agents: bytes32 → AgentMetadata` - Main agent storage
- `didToAgentId: string → bytes32` - O(1) DID lookup
- `ownerToAgents: address → bytes32[]` - Find agents by owner
- `agentKeys: bytes32 → AgentKey` - Key storage by hash
- `registrationCommitments: address → RegistrationCommitment` - Commit-reveal tracking
- `agentNonce: bytes32 → uint256` - Replay protection per agent
- `dailyRegistrationCount: address → uint256` - Rate limiting
- `publicKeyUsed: bytes32 → bool` - Prevent key reuse
- `agentOperators: bytes32 → address → bool` - ERC-721 style operator pattern

---

### 1.3 AgentCardVerifyHook.sol
**Location**: `contracts/ethereum/contracts/AgentCardVerifyHook.sol`

**Purpose**: External verification hook for pre-registration validation and anti-fraud

**Inheritance**: Ownable2Step

**Key Validation Functions**:

#### `beforeRegister(did, owner, keys)`
Validates before registration:
1. Checks blacklist
2. Validates DID format (did:sage:chain:...)
3. Checks rate limiting (unless whitelisted)
4. Prevents public key reuse across different owners

**Security Features**:
- Blacklist/whitelist management
- DID format validation per W3C standard
- Rate limiting: 24 registrations per address per day
- Public key reuse prevention across owners
- Role-based access control (owner-only admin functions)

**Key Mappings**:
- `blacklisted: address → bool` - Prevent malicious addresses
- `whitelisted: address → bool` - Bypass rate limiting
- `keyToOwner: bytes32 → address` - Track key ownership
- `dailyRegistrationCount: address → uint256` - Daily quota
- `lastRegistrationDay: address → uint256` - Day tracking

---

### 1.4 Interface Files

#### ISageRegistry.sol
**Legacy interface** with single-key support:
```solidity
struct AgentMetadata {
    string did
    string name
    string description
    string endpoint
    bytes publicKey                // Single key only
    string capabilities
    address owner
    uint256 registeredAt
    uint256 updatedAt
    bool active
}

function registerAgent(
    string did,
    string name,
    string description,
    string endpoint,
    bytes publicKey,               // Single key
    string capabilities,
    bytes signature
) external returns (bytes32)

function updateAgent(
    bytes32 agentId,
    string name,
    string description,
    string endpoint,
    string capabilities,
    bytes signature
) external

function deactivateAgent(bytes32 agentId) external
function deactivateAgentByDID(string did) external
```

#### ISageRegistryV4.sol
**Current interface** with multi-key support:
```solidity
enum KeyType { Ed25519, ECDSA }  // Note: Different from V1

struct AgentKey {
    KeyType keyType
    bytes keyData
    bytes signature
    bool verified
    uint256 registeredAt
}

struct AgentMetadata {
    string did
    string name
    string description
    string endpoint
    bytes32[] keyHashes            // Multiple keys
    string capabilities
    address owner
    uint256 registeredAt
    uint256 updatedAt
    bool active
}

struct RegistrationParams {
    string did
    string name
    string description
    string endpoint
    KeyType[] keyTypes
    bytes[] keyData
    bytes[] signatures
    string capabilities
}

// Key management functions
function registerAgent(RegistrationParams) returns (bytes32)
function addKey(agentId, keyType, keyData, signature) returns (bytes32)
function revokeKey(agentId, keyHash) 
function rotateKey(agentId, oldKeyHash, newKeyType, newKeyData, newSignature) returns (bytes32)
function approveEd25519Key(keyHash)  // Owner approval for off-chain keys

// Query functions
function getAgent(agentId) returns (AgentMetadata)
function getAgentByDID(did) returns (AgentMetadata)
function getKey(keyHash) returns (AgentKey)
function getAgentKeys(agentId) returns (bytes32[])
function getAgentsByOwner(ownerAddress) returns (bytes32[])
function getNonce(agentId) returns (uint256)

// ERC-8004 interface
function updateAgentEndpoint(agentId, newEndpoint)
function deactivateAgent(agentId)
```

#### IERC8004IdentityRegistry.sol
**Standard interface** for agent identity:
```solidity
struct AgentInfo {
    string agentId           // DID
    address agentAddress     // Owner address
    string endpoint          // AgentCard location
    bool isActive           // Agent status
    uint256 registeredAt    // Registration timestamp
}

function registerAgent(agentId, endpoint) returns (bool)
function resolveAgent(agentId) returns (AgentInfo)
function resolveAgentByAddress(agentAddress) returns (AgentInfo)
function isAgentActive(agentId) returns (bool)
function updateAgentEndpoint(agentId, newEndpoint) returns (bool)
function deactivateAgent(agentId) returns (bool)
```

#### IRegistryHook.sol
**Simple hook interface** (legacy, not used by AgentCardRegistry):
```solidity
function beforeRegister(agentId, owner, data) 
    returns (bool success, string reason)
function afterRegister(agentId, owner, data)
```

---

## 2. Governance Contracts

### 2.1 TEEKeyRegistry.sol
**Location**: `contracts/ethereum/contracts/governance/TEEKeyRegistry.sol`

**Purpose**: Decentralized governance for TEE (Trusted Execution Environment) key approval

**Architecture**: Community voting with stake-based proposals

**Key Concepts**:

#### Proposal Flow
```
1. PROPOSAL PHASE
   - Proposer stakes 1 ETH
   - Submits TEE key + attestation report
   - 7-day voting period begins

2. VOTING PHASE
   - Registered voters cast weighted votes
   - Minimum 10% participation required
   - Votes tracked: FOR vs AGAINST

3. EXECUTION PHASE
   - After 7 days, anyone can execute
   - ≥66% approval + ≥10% participation required
   - APPROVED: Key trusted, stake returned
   - REJECTED: 50% stake slashed, 50% returned
```

#### ProposalStatus Enum
```solidity
enum ProposalStatus {
    PENDING,      // Voting in progress
    APPROVED,     // Reached approval threshold
    REJECTED,     // Voting failed
    EXECUTED,     // Approved and executed
    CANCELLED     // Cancelled by proposer
}
```

#### TEEKeyProposal Struct
```solidity
struct TEEKeyProposal {
    bytes32 keyHash
    address proposer
    string attestationReport       // URL to TEE attestation
    string teeType                // "SGX", "SEV", "TrustZone", "Nitro"
    uint256 proposalStake          // 1 ETH default
    uint256 votesFor
    uint256 votesAgainst
    uint256 createdAt
    uint256 votingDeadline
    ProposalStatus status
}
```

**Governance Parameters** (all adjustable by owner):
- `proposalStake = 1 ether` - Spam prevention
- `votingPeriod = 7 days` - Deliberation time
- `approvalThreshold = 66%` - Supermajority (⅔)
- `minVoterParticipation = 10%` - Prevent small groups
- `SLASHING_PERCENTAGE = 50%` - Penalty for rejected proposals

**Key Functions**:
```solidity
proposeTEEKey(keyHash, attestationReport, teeType) 
    - Requires 1 ETH stake
    - Returns proposalId

vote(proposalId, support)
    - Only registered voters
    - Weighted voting based on reputation

executeProposal(proposalId) returns (bool approved)
    - After voting deadline
    - Checks participation and approval thresholds
    - Returns or slashes stake

registerVoter(voter, weight)          // Owner only
updateVoterWeight(voter, newWeight)   // Owner only
removeVoter(voter)                    // Owner only
revokeTEEKey(keyHash, reason)         // Emergency revocation

getProposal(proposalId)
getProposalStatus(proposalId) returns (detailed voting info)
isTrustedTEEKey(keyHash) returns (bool)
```

**Security Model**:
- Byzantine Fault Tolerance: 66% threshold tolerates 33% malicious voters
- Stake-based spam prevention: 1 ETH proposal fee
- Sybil protection: Registered voter system with weights
- Emergency controls: Owner can revoke compromised keys

---

### 2.2 SimpleMultiSig.sol
**Location**: `contracts/ethereum/contracts/governance/SimpleMultiSig.sol`

**Purpose**: Simple M-of-N multi-signature wallet for SAGE governance

**Warning**: Simplified for testing; Gnosis Safe recommended for production

**Key Components**:

#### Transaction Struct
```solidity
struct Transaction {
    address to
    uint256 value
    bytes data
    bool executed
    uint256 confirmations
}
```

**Key Functions**:
```solidity
constructor(address[] owners, uint256 threshold)

proposeTransaction(to, value, data) returns (transactionId)
    - Any owner can propose
    - Auto-confirms for proposer
    - Auto-executes if threshold reached

confirmTransaction(transactionId)
    - Vote on pending transaction
    - Only owners
    - Auto-executes if threshold reached

revokeConfirmation(transactionId)
    - Withdraw vote before execution
    - Only owners

executeTransaction(transactionId)
    - Execute when threshold met
    - After all state changes (CEI pattern)
    - Return bomb protection (1KB limit)

getTransaction(transactionId) returns (to, value, data, executed, confirmations)
getOwners() returns (address[])
getConfirmationCount(transactionId)
getPendingTransactionCount()
getExecutableTransactionCount()
```

**Security Features**:
- Checks-Effects-Interactions (CEI) pattern
- Reentrancy guard
- Return bomb protection
- State changes before external calls

---

### 2.3 TimelockController (SAGETimelockController)
**Location**: `contracts/ethereum/contracts/governance/TimelockController.sol`

**Purpose**: OpenZeppelin TimelockController wrapper for delayed governance execution

**Usage**: Enables time-locked execution of governance actions

---

## 3. Key Changes from V1 to V4

### 3.1 Registration Flow Changes

**V1/ISageRegistry**:
- Single registration call: `registerAgent(did, name, ..., publicKey, ..., signature)`
- Single public key per agent
- No commit-reveal pattern
- No explicit time-lock activation

**V4/AgentCardRegistry**:
- Two-phase registration (commit-reveal):
  1. `commitRegistration(commitHash)` - Phase 1 (prevents front-running)
  2. `registerAgentWithParams(params)` - Phase 2 (verify commitment and register)
  3. `activateAgent(agentId)` - Phase 3 (time-locked activation after 1 hour)
- Multiple keys per agent (up to 10)
- Commit-reveal prevents DID front-running
- 1-hour time-lock before activation
- Stake requirement (0.01 ETH)

### 3.2 Key Management Changes

**V1**: Single key
```solidity
bytes publicKey  // Single key, no type information
```

**V4**: Multiple keys with types
```solidity
bytes32[] keyHashes          // Array of key references
mapping(bytes32 => AgentKey) // Detailed key information

struct AgentKey {
    KeyType keyType          // ECDSA, Ed25519, or X25519
    bytes keyData            // Raw public key
    bytes signature          // Ownership proof
    bool verified            // Verification status
    uint256 registeredAt
}
```

**New Key Management Functions**:
- `addKey()` - Add new key to existing agent
- `revokeKey()` - Revoke key (keeps at least 1)
- `rotateKey()` - Atomic key rotation (revoke old + add new)
- `approveEd25519Key()` - Owner approval for off-chain keys

### 3.3 KeyType Enum Changes

**ISageRegistry V4** (interface):
```solidity
enum KeyType {
    Ed25519,    // Solana, Cardano, Polkadot
    ECDSA       // Ethereum, Bitcoin
}
```

**AgentCardStorage** (actual implementation):
```solidity
enum KeyType {
    ECDSA,      // secp256k1 (index 0)
    Ed25519,    // EdDSA (index 1)
    X25519      // ECDH/Encryption (index 2)
}
```

**Important**: Different ordering! Storage has ECDSA at 0, but interface has Ed25519 at 0.

### 3.4 Security & Anti-Fraud Improvements

**Rate Limiting**:
- Max 24 registrations per address per day (configurable)
- Whitelist bypass available
- Tracked via hook system

**Anti-Sybil**:
- Stake requirement (0.01 ETH)
- Time-locked activation (1 hour)
- Public key reuse prevention across agents

**Replay Protection**:
- Chain ID stored in agent metadata
- Cross-chain safe due to chain ID in commitment hash
- Nonce incremented on updates

**Access Control**:
- Owner/operator pattern (ERC-721 style)
- `setApprovalForAgent()` for delegated management
- Hook-based external validation

### 3.5 Metadata Storage Changes

**V1 Metadata**:
```solidity
struct AgentMetadata {
    string did
    string name
    string description
    string endpoint
    bytes publicKey        // Single key
    string capabilities
    address owner
    uint256 registeredAt
    uint256 updatedAt
    bool active
}
```

**V4 Metadata**:
```solidity
struct AgentMetadata {
    string did
    string name
    string description
    string endpoint
    bytes32[] keyHashes    // Multiple keys
    string capabilities
    address owner
    uint256 registeredAt
    uint256 updatedAt
    bool active
    uint256 chainId        // NEW: Replay protection
}
```

### 3.6 ERC-8004 Integration

**New Functions** (required by ERC-8004):
```solidity
// Minimum ERC-8004 interface
registerAgent(agentId, endpoint)        // Simplified, bypasses security
resolveAgent(agentId)                   // Get agent by DID
resolveAgentByAddress(agentAddress)     // Get agent by owner
isAgentActive(agentId)                  // Check status
updateAgentEndpoint(agentId, newEndpoint)
deactivateAgent(agentId)
```

**Note**: ERC-8004 `registerAgent()` is provided for compatibility but bypasses the commit-reveal security. Production use should use `registerAgentWithParams()` + commit-reveal.

---

## 4. Function Signatures for Go Client Implementation

### 4.1 Registration Flow

```solidity
// Phase 1: Commit
function commitRegistration(bytes32 commitHash)
    external payable whenNotPaused nonReentrant
    - Input: keccak256(abi.encode(did, keys, owner, salt, chainId))
    - Value: >= 0.01 ETH (registration stake)
    - Emits: CommitmentRecorded(caller, commitHash, timestamp)

// Phase 2: Reveal & Register
function registerAgentWithParams(RegistrationParams calldata params)
    external whenNotPaused nonReentrant validDID returns (bytes32 agentId)
    - Input: RegistrationParams {
        did: string
        name: string
        description: string
        endpoint: string
        keys: bytes[]
        keyTypes: KeyType[]
        signatures: bytes[]
        capabilities: string
        salt: bytes32
      }
    - Returns: bytes32 agentId
    - Emits: AgentRegistered(agentId, did, owner, timestamp)

// Phase 3: Activate
function activateAgent(bytes32 agentId)
    external nonReentrant
    - Input: Agent ID (bytes32)
    - Requires: 1 hour after registration
    - Emits: AgentActivated(agentId, timestamp)
```

### 4.2 Key Management

```solidity
function addKey(
    bytes32 agentId,
    bytes calldata keyData,
    KeyType keyType,
    bytes calldata signature
) external onlyAgentOwner whenNotPaused nonReentrant
    - Returns: implicitly emits KeyAdded
    - Limits: Max 10 keys per agent

function revokeKey(bytes32 agentId, bytes32 keyHash)
    external onlyAgentOwner whenNotPaused nonReentrant
    - Requires: At least 1 key remaining

function rotateKey(
    bytes32 agentId,
    bytes32 oldKeyHash,
    KeyType newKeyType,
    bytes calldata newKeyData,
    bytes calldata newSignature
) external onlyAgentOwner whenNotPaused nonReentrant
    - Returns: bytes32 newKeyHash (not in V4 interface but implemented)
    - Atomic: Old key removed, new key added

function approveEd25519Key(bytes32 keyHash)
    external onlyOwner
    - Approves Ed25519 keys (can't be verified on-chain)
    - Emits: Ed25519KeyApproved(keyHash, timestamp)
```

### 4.3 Agent Management

```solidity
function updateAgent(
    bytes32 agentId,
    string calldata endpoint,
    string calldata capabilities
) external onlyAgentOwner whenNotPaused nonReentrant
    - Updates agent metadata
    - Increments agentNonce for replay protection
    - Emits: AgentUpdated(agentId, timestamp)

function deactivateAgentByHash(bytes32 agentId)
    external onlyAgentOwner nonReentrant
    - Deactivates agent
    - Returns stake after 30 days
    - Emits: AgentDeactivatedByHash(agentId, timestamp)

function setApprovalForAgent(
    bytes32 agentId,
    address operator,
    bool approved
) external
    - Grant/revoke operator permissions
    - Operators can manage agents on behalf of owner
    - Emits: ApprovalForAgent(agentId, owner, operator, approved)
```

### 4.4 Query Functions

```solidity
// Read-only (call)
function getAgent(bytes32 agentId) 
    public view returns (AgentMetadata)

function getAgentByDID(string calldata did)
    public view returns (AgentMetadata)

function getKey(bytes32 keyHash)
    public view returns (AgentKey)

function getAgentsByOwner(address owner)
    public view returns (bytes32[])

function getNonce(bytes32 agentId)
    external view returns (uint256)

function isAgentActive(bytes32 agentId)
    external view returns (bool)

function verifyAgentOwnership(bytes32 agentId, address claimedOwner)
    external view returns (bool)

function isApprovedOperator(bytes32 agentId, address operator)
    external view returns (bool)
```

### 4.5 Admin Functions

```solidity
function setRegistrationStake(uint256 newStake) external onlyOwner
function setActivationDelay(uint256 newDelay) external onlyOwner
function setVerifyHook(address newHook) external onlyOwner
function setBeforeRegisterHook(address hook) external onlyOwner
function setAfterRegisterHook(address hook) external onlyOwner
function pause() external onlyOwner
function unpause() external onlyOwner
```

---

## 5. Events Emitted

### Registration & Lifecycle
```solidity
event AgentRegistered(
    bytes32 indexed agentId,
    string indexed did,
    address indexed owner,
    uint256 timestamp
)

event AgentActivated(bytes32 indexed agentId, uint256 timestamp)

event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)

event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)

event AgentEndpointUpdated(
    string indexed agentId,
    string oldEndpoint,
    string newEndpoint
)

event AgentDeactivated(
    string indexed agentId,
    address indexed agentAddress
)
```

### Key Management
```solidity
event KeyAdded(
    bytes32 indexed agentId,
    bytes32 indexed keyHash,
    KeyType keyType,
    uint256 timestamp
)

event KeyRevoked(
    bytes32 indexed agentId,
    bytes32 indexed keyHash,
    uint256 timestamp
)

event KeyRotated(
    bytes32 indexed agentId,
    bytes32 indexed oldKeyHash,
    bytes32 indexed newKeyHash,
    uint256 timestamp
)

event Ed25519KeyApproved(bytes32 indexed keyHash, uint256 timestamp)
```

### Access Control
```solidity
event ApprovalForAgent(
    bytes32 indexed agentId,
    address indexed owner,
    address indexed operator,
    bool approved
)
```

### Hooks
```solidity
event BeforeRegisterHook(
    bytes32 indexed agentId,
    address indexed caller,
    bytes hookData
)

event AfterRegisterHook(
    bytes32 indexed agentId,
    address indexed caller,
    bytes hookData
)
```

### Commit-Reveal
```solidity
event CommitmentRecorded(
    address indexed committer,
    bytes32 commitHash,
    uint256 timestamp
)
```

---

## 6. Breaking Changes from V1

### 6.1 Type System
- **KeyType enum reordered**: V4 storage has ECDSA=0, but V4 interface has Ed25519=0
- **X25519 support added**: New key type for encryption
- **Multiple keys**: V1 had single `bytes publicKey`, V4 has `bytes32[] keyHashes`

### 6.2 Registration API
- **Two-phase registration required**: Commit-reveal pattern breaks backward compatibility
- **Parameters struct**: Stack depth issues solved by `RegistrationParams`
- **Salt parameter added**: Required for commit-reveal hash
- **Key arrays required**: Must provide multiple keys in `keys[]`, `keyTypes[]`, `signatures[]`

### 6.3 Query API
- **Agent ID is now bytes32**: V1 might have used strings, V4 uses hashes
- **Direct key queries**: `getKey()` returns `AgentKey` struct instead of just bytes
- **No single-key access**: Must iterate through key array

### 6.4 Activation Model
- **Time-lock activation**: Agents not active immediately after registration
- **Two-step: commit → register → activate**: Requires 3 transactions minimum
- **1-hour minimum wait**: Between registration and activation

### 6.5 Metadata Structure
- **Chain ID required**: V4 stores `chainId` for replay protection
- **Endpoint in agent**: V4 stores in metadata (ERC-8004 integration)
- **Removed single publicKey field**: All keys in keyHashes array

---

## 7. Integration Requirements for Go Client

### 7.1 Data Structure Mapping

Go struct for RegistrationParams:
```go
type RegistrationParams struct {
    Did          string      // DID identifier
    Name         string      // Agent name
    Description  string      // Agent description
    Endpoint     string      // AgentCard location
    KeyTypes     []uint8     // ECDSA=0, Ed25519=1, X25519=2
    KeyData      [][]byte    // Public key bytes
    Signatures   [][]byte    // Ownership proofs
    Capabilities string      // JSON capabilities
}
```

Go struct for AgentMetadata:
```go
type AgentMetadata struct {
    Did          string        // DID
    Name         string
    Description  string
    Endpoint     string
    KeyHashes    [][32]byte    // bytes32[] array
    Capabilities string
    Owner        common.Address
    RegisteredAt *big.Int      // uint256
    UpdatedAt    *big.Int
    Active       bool
    ChainId      *big.Int      // NEW: Chain ID
}
```

### 7.2 Key Verification Methods

**ECDSA (secp256k1)**:
- Signature verification on-chain via ecrecover
- Message format: Ethereum Signed Message (keccak256 wrapper)
- 65-byte signature: (r, s, v)

**Ed25519**:
- Cannot verify on-chain (no precompile)
- Requires owner pre-approval: `approveEd25519Key(keyHash)`
- 64-byte signature

**X25519**:
- Encryption-only key, no signature verification
- 32-byte key length validation only

### 7.3 Commit-Reveal Implementation

Go pseudocode:
```go
// Step 1: Prepare commitment
commitHash := keccak256(abi.encode(did, keys, owner, salt, chainId))

// Step 2: Send commitment (Phase 1)
tx1 := registry.CommitRegistration(commitHash, stake)
await tx1.Mined()

// Step 3: Wait minimum delay (1 minute)
time.Sleep(1 * time.Minute)

// Step 4: Reveal and register (Phase 2)
params := RegistrationParams{...}
tx2 := registry.RegisterAgentWithParams(params)
agentId := await tx2.GetAgentId()

// Step 5: Wait activation delay (1 hour)
time.Sleep(1 * time.Hour)

// Step 6: Activate agent (Phase 3)
tx3 := registry.ActivateAgent(agentId)
await tx3.Mined()
```

### 7.4 Error Handling

**Common Revert Reasons**:
- "Invalid commit hash" - commitHash is zero
- "Insufficient stake" - msg.value < registrationStake
- "Daily registration limit exceeded" - Rate limited
- "No commitment found" - commitRegistration() not called
- "Commitment expired" - Took > 1 hour to reveal
- "Reveal too soon" - Tried to reveal < 1 minute after commit
- "Invalid reveal" - Hash mismatch
- "Invalid key count" - 0 or > 10 keys
- "Key type mismatch" - keys.length != keyTypes.length
- "Signature mismatch" - keys.length != signatures.length
- "Public key already used" - Key reuse across agents
- "Invalid DID format" - DID doesn't match pattern
- "Address blacklisted" - Blocked by hook
- "Already revealed" - Already completed reveal phase
- "Activation delay not passed" - Trying to activate too early
- "Agent not active" - Agent still in time-lock period

---

## 8. Summary of Key Function Signatures

### Transactions (Modify State)

| Function | Parameters | Returns | Notes |
|----------|-----------|---------|-------|
| `commitRegistration` | bytes32 commitHash | void | Phase 1, requires stake |
| `registerAgentWithParams` | RegistrationParams | bytes32 agentId | Phase 2, commit-reveal |
| `activateAgent` | bytes32 agentId | void | Phase 3, time-locked |
| `addKey` | agentId, keyData, keyType, signature | void | Max 10 keys |
| `revokeKey` | agentId, keyHash | void | Min 1 key required |
| `rotateKey` | agentId, oldKeyHash, newKeyType, newKeyData, newSig | bytes32 | Atomic |
| `approveEd25519Key` | bytes32 keyHash | void | Owner only |
| `updateAgent` | agentId, endpoint, capabilities | void | Increments nonce |
| `deactivateAgentByHash` | bytes32 agentId | void | Returns stake after 30d |
| `setApprovalForAgent` | agentId, operator, approved | void | ERC-721 pattern |

### View Functions (Read-Only)

| Function | Parameters | Returns |
|----------|-----------|---------|
| `getAgent` | bytes32 agentId | AgentMetadata |
| `getAgentByDID` | string did | AgentMetadata |
| `getKey` | bytes32 keyHash | AgentKey |
| `getAgentsByOwner` | address owner | bytes32[] |
| `getNonce` | bytes32 agentId | uint256 |
| `isAgentActive` | bytes32 agentId | bool |
| `isApprovedOperator` | agentId, operator | bool |
| `verifyAgentOwnership` | agentId, owner | bool |

---

## 9. Differences Between V4 Interface and AgentCardRegistry Implementation

The actual `AgentCardRegistry` implementation differs from `ISageRegistryV4` in important ways:

### Missing from Interface but in Implementation

1. **Commit-Reveal Pattern**:
   - Interface doesn't define `commitRegistration()`
   - AgentCardRegistry has Phase 1 + 2 + 3 flow
   - ISageRegistryV4 only has `registerAgent(params)` (Phase 2 only)

2. **Time-Locked Activation**:
   - `activateAgent()` not in ISageRegistryV4
   - AgentCardRegistry requires activation after delay

3. **Operator Management**:
   - `setApprovalForAgent()` not in ISageRegistryV4
   - Enables ERC-721 style operator delegation

4. **StakeRequirement & Registration Parameters**:
   - ISageRegistryV4 doesn't define stake model
   - AgentCardRegistry requires 0.01 ETH + hook validation

### Different Implementations

1. **KeyType Enum Ordering**:
   - ISageRegistryV4: Ed25519=0, ECDSA=1
   - AgentCardStorage: ECDSA=0, Ed25519=1, X25519=2

2. **RotateKey Return Value**:
   - ISageRegistryV4: returns bytes32 newKeyHash
   - Actual implementation: likely same but verify

3. **Hook System**:
   - ISageRegistryV4: beforeRegisterHook, afterRegisterHook (simple interface)
   - AgentCardRegistry: AgentCardVerifyHook (detailed validation logic)

---

## 10. Governance Integration Points

### TEEKeyRegistry Integration

**Where Used**: ERC8004ValidationRegistry can call TEEKeyRegistry

```solidity
// In ValidationRegistry
require(
    TEEKeyRegistry(teeRegistry).isTrustedTEEKey(keyHash),
    "TEE key not trusted"
);
```

**For Go Client**: 
- Query `isTrustedTEEKey(bytes32 keyHash) returns (bool)`
- Subscribe to `TEEKeyApproved` and `TEEKeyRevoked` events

### SimpleMultiSig Integration

**Purpose**: Multi-signature governance for registry upgrades

**For Go Client**:
- Query transaction status: `getTransaction(id)`, `getConfirmationCount(id)`
- Monitor `TransactionProposed`, `TransactionExecuted` events

---

This comprehensive analysis shows SAGE's sophisticated contract architecture balancing security, decentralization, and user experience.
