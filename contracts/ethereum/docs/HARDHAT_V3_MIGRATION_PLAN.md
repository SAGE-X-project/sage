# Hardhat v3 Migration Plan

## Executive Summary

This document outlines the migration plan from Hardhat v2.26.3 to Hardhat v3.0.6, which will resolve remaining security vulnerabilities (cookie and tmp packages) and provide improved performance and modern JavaScript support.

## Current State Analysis

### Current Version
- Hardhat: v2.26.3
- Node.js: v22.14.0
- Package Manager: npm

### Files Requiring Conversion (22 files total)

#### Configuration Files (1)
- `hardhat.config.js` - Main configuration using CommonJS require()

#### Script Files (21)
- `scripts/deploy.js`
- `scripts/verify.js`
- `scripts/check-balance.js`
- `scripts/deploy-kaia.js`
- `scripts/deploy-local.js`
- `scripts/deploy-v2.js`
- `scripts/interact-kaia.js`
- `scripts/quick-test.js`
- `scripts/verify-kaia.js`
- `scripts/query-agents.js`
- `scripts/interact-local.js`
- `scripts/register-production-agents.js`
- `scripts/deploy-kaia-v2.js`
- `scripts/deploy-kaia-v2-latest.js`
- `scripts/verify-contracts.js`
- `scripts/generate-verification-info.js`
- `scripts/extract-abi.js`
- `scripts/generate-go-bindings.js`
- `scripts/generate-java-bindings.js`
- `scripts/generate-python-bindings.js`
- `scripts/generate-rust-bindings.js`

#### Test Files (5)
- `test/SageRegistry.test.fixed.js`
- `test/SageRegistry.test.js`
- `test/integration-v2.test.js`
- `test/integration.test.js`
- `test/SageRegistryV2.test.js`

## Breaking Changes

### 1. ESM Module System (CRITICAL)
**Impact**: All JavaScript files must be converted to ESM syntax
- `require()` → `import`
- `module.exports` → `export`
- `__dirname` → `import.meta.url`

### 2. Configuration Changes
**Impact**: hardhat.config.js must use ESM syntax
- Add `"type": "module"` to package.json
- Convert all require statements to imports
- Use export default instead of module.exports

### 3. Plugin Loading
**Impact**: All Hardhat plugins must be ESM-compatible or have workarounds
- Update plugin imports in config
- Check compatibility of all plugins

### 4. Script Execution
**Impact**: All scripts must use ESM syntax
- Update ethers imports
- Update hardhat runtime environment access

## Migration Strategy

### Phase 1: Preparation (Day 1)
1. Create feature branch `feature/hardhat-v3-migration`
2. Backup current configuration
3. Document all current functionality
4. Create test checklist

### Phase 2: Conversion (Day 2-3)

#### Step 1: Package.json Update
```json
{
  "type": "module",
  "devDependencies": {
    "hardhat": "^3.0.6"
  }
}
```

#### Step 2: Config File Conversion
Convert `hardhat.config.js`:
```javascript
// Before (CommonJS)
require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config();
module.exports = { ... };

// After (ESM)
import "@nomicfoundation/hardhat-toolbox";
import dotenv from "dotenv";
dotenv.config();
export default { ... };
```

#### Step 3: Script Conversion Template
```javascript
// Before (CommonJS)
const hre = require("hardhat");
async function main() { ... }
main().catch((error) => { ... });

// After (ESM)
import hre from "hardhat";
async function main() { ... }
main().catch((error) => { ... });
```

#### Step 4: Test File Conversion
```javascript
// Before (CommonJS)
const { expect } = require("chai");
const { ethers } = require("hardhat");

// After (ESM)
import { expect } from "chai";
import { ethers } from "hardhat";
```

### Phase 3: Testing (Day 4)
1. Run compilation: `npm run compile`
2. Run all tests: `npm test`
3. Test all deployment scripts locally
4. Verify gas reporting functionality
5. Test coverage reports

### Phase 4: Validation (Day 5)
1. Deploy to test network (Kairos)
2. Verify contracts on block explorer
3. Run integration tests
4. Security audit with `npm audit`

## Risk Mitigation

### Rollback Plan
1. Maintain v2 configuration in separate branch
2. Keep package-lock.json backup
3. Document all changes for easy reversion

### Compatibility Concerns
1. **@nomicfoundation/hardhat-toolbox**: Check ESM support
2. **hardhat-gas-reporter**: May need update or replacement
3. **solidity-coverage**: Verify v0.8.x compatibility

### Testing Requirements
- All existing tests must pass
- Gas costs should remain consistent
- Deployment scripts must work on all networks
- Contract verification must function

## Benefits of Migration

1. **Security**: Eliminates cookie and tmp vulnerabilities
2. **Performance**: ~15-20% faster compilation and testing
3. **Modern JavaScript**: Native ESM support
4. **Future-Proof**: Better ecosystem compatibility
5. **Maintenance**: Cleaner, more maintainable code

## Timeline

| Phase | Duration | Tasks |
|-------|----------|-------|
| Preparation | 1 day | Backup, documentation, planning |
| Conversion | 2-3 days | Convert all files to ESM |
| Testing | 1 day | Run comprehensive tests |
| Validation | 1 day | Deploy and verify on testnet |
| **Total** | **5-6 days** | Complete migration |

## Success Criteria

- [ ] All 22 JavaScript files converted to ESM
- [ ] `npm audit` shows 0 vulnerabilities
- [ ] All tests pass with same coverage
- [ ] Deployment scripts work on all networks
- [ ] Gas reporting functions correctly
- [ ] Contract verification works
- [ ] No performance degradation

## Conversion Checklist

### Configuration
- [ ] Add `"type": "module"` to package.json
- [ ] Convert hardhat.config.js to ESM

### Scripts (21 files)
- [ ] deploy.js
- [ ] verify.js
- [ ] check-balance.js
- [ ] deploy-kaia.js
- [ ] deploy-local.js
- [ ] deploy-v2.js
- [ ] interact-kaia.js
- [ ] quick-test.js
- [ ] verify-kaia.js
- [ ] query-agents.js
- [ ] interact-local.js
- [ ] register-production-agents.js
- [ ] deploy-kaia-v2.js
- [ ] deploy-kaia-v2-latest.js
- [ ] verify-contracts.js
- [ ] generate-verification-info.js
- [ ] extract-abi.js
- [ ] generate-go-bindings.js
- [ ] generate-java-bindings.js
- [ ] generate-python-bindings.js
- [ ] generate-rust-bindings.js

### Tests (5 files)
- [ ] SageRegistry.test.fixed.js
- [ ] SageRegistry.test.js
- [ ] integration-v2.test.js
- [ ] integration.test.js
- [ ] SageRegistryV2.test.js

### Validation
- [ ] Compilation works
- [ ] All tests pass
- [ ] Local deployment works
- [ ] Testnet deployment works
- [ ] Contract verification works
- [ ] Gas reporting works
- [ ] Coverage reporting works

## Post-Migration Tasks

1. Update documentation
2. Update CI/CD pipelines if needed
3. Team training on ESM syntax
4. Monitor for any issues
5. Update development guidelines

## References

- [Hardhat v3 Release Notes](https://hardhat.org/hardhat-runner/docs/guides/migrating-from-v2)
- [Node.js ESM Documentation](https://nodejs.org/api/esm.html)
- [ESM Migration Guide](https://gist.github.com/sindresorhus/a39789f98801d908bbc7ff3ecc99d99c)

## Notes

- Consider using automated tools like `cjs-to-esm` for bulk conversion
- Test incrementally - convert and test one module at a time
- Keep detailed logs of any issues encountered
- Consider creating helper utilities for common patterns

---

Document Version: 1.0
Last Updated: 2024-12-27
Author: SAGE Development Team