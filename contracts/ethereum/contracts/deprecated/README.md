# Deprecated SAGE Registry Contracts

This directory contains legacy SAGE registry contracts that are no longer recommended for use in production environments.

**DO NOT USE THESE CONTRACTS FOR NEW DEPLOYMENTS**

---

## Deprecated Contracts

### SageRegistry.sol (V1)

**Status**: DEPRECATED - Critical Security Vulnerabilities

**Deprecated Date**: 2025-01-18

**Why Deprecated**:
- **CRITICAL**: Ed25519 signature bypass vulnerability
  - Line 375-379: Returns `true` without verification for 32-byte keys
  - Allows attackers to register agents with arbitrary Ed25519 keys
  - Identity theft and unauthorized agent registration possible
- Missing reentrancy protection
- Inadequate access control
- No key revocation capability
- Poor gas optimization

**Security Issues**:
- 1 Critical vulnerability
- 5 High-priority issues
- 8 Medium-priority issues
- 12 Low-priority issues

**Replacement**: Use SageRegistryV4.sol

**References**:
- [Security Audit](../../../archived/SECURITY_AUDIT_LEGACY.md)
- [Code Analysis](../../../archived/CODE_ANALYSIS_V1_V2_V3.md)

---

### SageRegistryV2.sol (V2)

**Status**: DEPRECATED - Superseded by V4

**Deprecated Date**: 2025-01-19

**Why Deprecated**:
- Superseded by SageRegistryV4 with superior multi-key architecture
- Single-key limitation (only one key per agent)
- No support for Ed25519 or X25519 keys
- Limited A2A protocol compatibility
- V4 provides all V2 features plus multi-key support and enhanced validation

**Key Features** (Historical):
- 5-step public key validation
- Signature-based ownership proof
- Key revocation capability
- Emergency pause mechanism

**Replacement**: Use SageRegistryV4.sol

**Legacy Deployments**:
- Sepolia testnet: 0x487d45a678eb947bbF9d8f38a67721b13a0209BF (no longer recommended)

---

### SageRegistryTest.sol (V2 Test Version)

**Status**: DEPRECATED - Testing Contract

**Deprecated Date**: 2025-01-19

**Why Deprecated**:
- Test version of SageRegistryV2 (now deprecated)
- Bypasses key validation for local testing
- **NEVER intended for production use**

**Replacement**: Use V4 test suite in `../test/` directory

---

### SageRegistryV3.sol (V3)

**Status**: DEPRECATED - Superseded by V4

**Deprecated Date**: 2025-01-18

**Why Deprecated**:
- Superseded by SageRegistryV4 with superior multi-key architecture
- Single-key limitation (only one key per agent)
- No support for multiple cryptographic key types
- Limited A2A protocol compatibility
- V4 provides all V3 features plus multi-key support

**Key Limitations**:
- Single key type support only
- No Ed25519 or X25519 support
- Limited interoperability with multi-chain systems
- Cannot support Google A2A protocol natively

**Replacement**: Use SageRegistryV4.sol

**Migration Path**:
- V4 maintains backward compatibility with V3 single-key agents
- Existing V3 agents can be migrated to V4 with additional keys
- See [ROADMAP.md](../../../ROADMAP.md) for migration guide

---

## Active Contracts

For all new deployments, use:

### SageRegistryV4.sol (Latest - Recommended)
- **Status**: ✅ Production Ready
- **Location**: `../SageRegistryV4.sol`
- **Features**:
  - Multi-key support (Ed25519, ECDSA, X25519)
  - A2A protocol compatibility with enhanced validation
  - W3C Verifiable Credentials
  - Proof-of-Possession mechanism
  - Backward compatible with single-key agents
- **Test Coverage**: 30 Solidity tests + 85+ Go tests, 100% passing
- **Performance**: 21 benchmarks, comprehensive gas analysis
- **Development Status**: Phase 1, 2 & 3 complete
- **Documentation**: Complete (ROADMAP.md, MULTI_KEY_DESIGN.md, GAS_COST_ANALYSIS.md)
- **Recommendation**: ⭐ Use for all new deployments

---

## Why These Contracts Are Kept

These deprecated contracts are retained in the repository for:

1. **Historical Reference**: Understanding the evolution of SAGE registry architecture
2. **Security Research**: Studying vulnerabilities and their fixes
3. **Audit Trail**: Maintaining complete version history
4. **Legacy Analysis**: Reference for understanding why V4 was designed as it is

**Important**: These contracts should NEVER be deployed to production environments.

---

## Version History

| Version | Status | Key Features | Issues | Deprecated Date |
|---------|--------|--------------|--------|-----------------|
| **V1** | DEPRECATED | Basic signature verification | Critical Ed25519 bypass | 2025-01-18 |
| **V2** | DEPRECATED | 5-step validation, key revocation | Single-key only | 2025-01-19 |
| **V3** | DEPRECATED | Commit-reveal pattern | Single-key only | 2025-01-18 |
| **V4** | ✅ PRODUCTION | Multi-key, A2A protocol, Enhanced Validation | None known | - |

---

## Migration Recommendations

### From V1 to V4

**DO NOT USE V1 IN ANY ENVIRONMENT**

1. Audit all V1 deployments for compromised agents
2. Deploy SageRegistryV4
3. Re-register all agents with proper key ownership proof
4. Decommission V1 contracts immediately

### From V2 to V4

1. Deploy SageRegistryV4
2. Re-register agents (V4 requires proof-of-possession)
3. Optionally add additional keys (Ed25519, X25519) for enhanced security
4. Update client applications to use V4 CLI tools (`sage-did`)
5. Verify enhanced validation features

### From V3 to V4

1. Deploy SageRegistryV4
2. Register agents with single key (backward compatible)
3. Optionally add additional keys (Ed25519, X25519) for multi-chain support
4. Verify A2A protocol compatibility with W3C Verifiable Credentials
5. Update client applications to use V4 endpoints

---

## Testing Deprecated Contracts

If you need to test deprecated contracts for research purposes:

```bash
# V1 tests (for historical reference only)
npx hardhat test test/SageRegistry.test.fixed.js

# DO NOT deploy to any network
# DO NOT use for production
# DO NOT register real agents
```

---

## Documentation

- [Main Contracts Documentation](../../../README.md)
- [Security Audit (Legacy)](../../../archived/SECURITY_AUDIT_LEGACY.md)
- [Code Analysis V1/V2/V3](../../../archived/CODE_ANALYSIS_V1_V2_V3.md)
- [Multi-Key Design (V4)](../../../MULTI_KEY_DESIGN.md)
- [Documentation Index](../../../CONTRACTS_INDEX.md)

---

## Support

For questions about deprecated contracts or migration assistance:
- Review [Security Audit](../../../archived/SECURITY_AUDIT_LEGACY.md)
- Check [Migration Guide](../../../README.md#migration-guide-v1--v2)
- Open GitHub issue with tag `migration`

---

**Last Updated**: 2025-01-19
**Maintained By**: SAGE Development Team
