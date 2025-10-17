# SAGE Contracts - TODO

**Last Updated**: 2025-01-18
**Current Version**: SageRegistryV4 (Multi-Key Support)
**Branch**: `feature/multi-key-registry-v4`

---

## 🎯 Active Tasks

### High Priority

#### 1. Multi-Key Registration CLI Support
**Status**: 📋 Planned
**Owner**: TBD
**Effort**: 60-90 minutes

- [ ] Extend `sage-did register` to accept multiple key files
- [ ] Support key type auto-detection from file format
- [ ] Generate proper signatures for each key type
- [ ] Handle Ed25519 pre-approval workflow
- [ ] Update CLI documentation with examples

**See**: `ROADMAP.md` Feature #1

---

#### 2. Key Management CLI Commands
**Status**: 📋 Planned
**Owner**: TBD
**Effort**: 40-60 minutes

- [ ] Create `cmd/sage-did/key.go` with subcommands
- [ ] Implement `sage-did key add <did> <keyfile>`
- [ ] Implement `sage-did key list <did>`
- [ ] Implement `sage-did key revoke <did> <keyhash>`
- [ ] Implement `sage-did key approve <keyhash>` (owner only)

**See**: `ROADMAP.md` Feature #2

---

#### 3. Smart Contract Deployment & Integration
**Status**: 📋 Planned
**Owner**: TBD
**Effort**: 90-120 minutes

**Deployment**:
- [ ] Create Hardhat deployment script for V4
- [ ] Deploy to local hardhat network
- [ ] Deploy to Sepolia testnet
- [ ] Verify contract on Etherscan
- [ ] Document deployed addresses

**SDK Integration**:
- [ ] Generate Go bindings for SageRegistryV4
- [ ] Update `pkg/agent/did/ethereum/client.go` to use V4
- [ ] Implement multi-key registration flow
- [ ] Add key management wrappers
- [ ] Handle Ed25519 approval workflow

**Testing**:
- [ ] End-to-end multi-key registration test
- [ ] Test key addition/revocation
- [ ] Measure and document gas costs
- [ ] A2A card generation from deployed contract

**See**: `ROADMAP.md` Feature #4

---

### Medium Priority

#### 4. A2A Integration Examples
**Status**: 📋 Planned
**Owner**: TBD
**Effort**: 50-70 minutes

- [ ] Create `examples/a2a-integration/` directory structure
- [ ] Example 1: Multi-key agent registration
- [ ] Example 2: A2A card generation and export
- [ ] Example 3: Agent-to-agent card exchange
- [ ] Example 4: Encrypted message exchange
- [ ] Comprehensive README with setup instructions

**See**: `ROADMAP.md` Feature #3

---

#### 5. Enhanced Validation
**Status**: 📋 Planned
**Owner**: TBD
**Effort**: 40-60 minutes

- [ ] Add A2A card signature verification
- [ ] Implement DID document cross-check
- [ ] Create key proof-of-possession mechanism
- [ ] Add validation tests

**See**: `ROADMAP.md` Feature #5

---

### Low Priority (Future)

#### 6. Contract Cleanup
**Status**: 🔄 In Progress
**Owner**: Current Sprint

- [x] Archive legacy analysis documents
- [x] Create security audit summary
- [ ] Move legacy contracts (V1, V2, V3) to `deprecated/` directory
- [ ] Update contract documentation
- [ ] Create contracts index

**See**: Current cleanup tasks below

---

#### 7. Documentation Updates
**Status**: 🔄 In Progress
**Owner**: Current Sprint

- [x] Create `ROADMAP.md` with feature plans
- [x] Archive old `TODO.md` → `CODE_ANALYSIS_V1_V2_V3.md`
- [x] Extract security audit to `SECURITY_AUDIT_LEGACY.md`
- [ ] Update `contracts/README.md` with V4 information
- [ ] Create `CONTRACTS_INDEX.md` listing all contracts
- [ ] Update deployment guide with V4 instructions

---

#### 8. Gas Optimization
**Status**: 📋 Backlog

- [ ] Profile gas costs for V4 operations
- [ ] Benchmark multi-key registration (1, 2, 3 keys)
- [ ] Compare gas costs with V3
- [ ] Document optimization opportunities
- [ ] Implement top 3 optimizations

---

#### 9. Multi-Chain Support
**Status**: 📋 Future

- [ ] Design multi-chain DID architecture
- [ ] Deploy V4 to Polygon
- [ ] Deploy V4 to Avalanche
- [ ] Cross-chain agent resolution
- [ ] Document multi-chain deployment

---

## 🚀 Current Sprint (Week of 2025-01-18)

### In Progress
- [x] SageRegistryV4 implementation
- [x] Multi-key support (Ed25519, ECDSA, X25519)
- [x] Unit tests (30 tests, 100% pass)
- [x] V4 Go types and A2A card generation
- [x] A2A card CLI commands
- [x] Comprehensive unit tests (37+ tests, 77.6% coverage)
- [x] Feature test verification (85/85 passing)
- [x] Documentation cleanup (TODO.md restructure)

### Next Up
- [ ] Update contracts/README.md
- [ ] Organize contract directory structure
- [ ] Create contracts index documentation

---

## 📋 Completed

### SageRegistryV4 Development ✅
- [x] Design multi-key architecture (`MULTI_KEY_DESIGN.md`)
- [x] Implement SageRegistryV4.sol with multi-key support
- [x] Create ISageRegistryV4.sol interface
- [x] Write 30 comprehensive unit tests
- [x] Verify all tests pass (100%)
- [x] Commit to feature branch

### Go Backend V4 Support ✅
- [x] Create `types_v4.go` with AgentMetadataV4
- [x] Implement AgentKey struct and KeyType enum
- [x] Create A2AAgentCard types
- [x] Write conversion functions (V4 ↔ Legacy)
- [x] Implement `a2a.go` with card generation/validation
- [x] Write 17 tests for types_v4
- [x] Write 20+ tests for A2A functions
- [x] Achieve 77.6% test coverage

### CLI Enhancements ✅
- [x] Create `cmd/sage-did/card.go`
- [x] Implement `sage-did card generate` command
- [x] Implement `sage-did card validate` command
- [x] Implement `sage-did card show` command
- [x] Test CLI compilation and help texts
- [x] Fix .gitignore for cmd/ directories

### Documentation ✅
- [x] Create comprehensive `ROADMAP.md`
- [x] Archive legacy TODO.md to `CODE_ANALYSIS_V1_V2_V3.md`
- [x] Extract security audit to `SECURITY_AUDIT_LEGACY.md`
- [x] Create concise new TODO.md (this file)

---

## 🔗 Related Documents

- **[ROADMAP.md](./ROADMAP.md)** - Detailed feature roadmap with implementation plans
- **[MULTI_KEY_DESIGN.md](./MULTI_KEY_DESIGN.md)** - SageRegistryV4 design specification
- **[README.md](./README.md)** - Contracts overview (needs V4 update)
- **[DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)** - Deployment instructions
- **[archived/CODE_ANALYSIS_V1_V2_V3.md](./archived/CODE_ANALYSIS_V1_V2_V3.md)** - Legacy contract analysis
- **[archived/SECURITY_AUDIT_LEGACY.md](./archived/SECURITY_AUDIT_LEGACY.md)** - Security audit for V1/V2/V3

---

## 📊 Progress Tracking

### Overall Completion: 65%

| Area | Status | Completion |
|------|--------|-----------|
| **Smart Contracts** | ✅ Complete | 100% |
| **Go Backend** | ✅ Complete | 100% |
| **CLI Tools** | 🔄 Partial | 50% |
| **Examples** | 📋 Planned | 0% |
| **Deployment** | 📋 Planned | 0% |
| **Documentation** | 🔄 In Progress | 75% |

### Test Status: ✅ All Passing

- Unit Tests (Solidity): 30/30 passing
- Unit Tests (Go): 37+/37+ passing
- Feature Tests: 85/85 passing
- Coverage: 77.6% (Go), 100% (Solidity critical paths)

---

## 💡 Notes

- All feature development should maintain 85/85 feature test pass rate
- Test coverage should remain >75%
- Each feature requires comprehensive unit tests before merge
- Documentation should be updated alongside code changes
- Backward compatibility with legacy single-key agents must be maintained

---

**For detailed implementation plans, see [ROADMAP.md](./ROADMAP.md)**
