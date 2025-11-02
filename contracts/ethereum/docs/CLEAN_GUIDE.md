# Ethereum Contracts - Clean Guide

Guide for cleaning generated files and build artifacts in the Ethereum contracts directory.

##  Available Clean Commands

### Basic Clean Commands

#### `npm run clean`
**Description**: Standard clean - removes build artifacts and generated files

**Removes**:
- `artifacts/` - Hardhat compilation artifacts
- `cache/` - Hardhat cache
- `typechain-types/` - TypeChain generated types
- `coverage/`, `coverage.json`, `.nyc_output/` - Coverage reports
- `test-results/` - Test result files
- `gas-report.txt` - Gas usage reports
- `*.log`, `logs/` - All log files
- `dist/`, `build/`, `out/` - Build directories
- `tmp/`, `temp/` - Temporary directories
- `*.tmp`, `*.temp`, `*.bak`, `*.backup`, `*.old` - Temporary files

**Usage**:
```bash
npm run clean
```

**When to use**:
- After running tests
- Before recompiling contracts
- When cleaning up workspace

---

#### `npm run clean:generated`
**Description**: Clean only generated files (no hardhat clean)

**Removes**: Same as `npm run clean` but without running `hardhat clean`

**Usage**:
```bash
npm run clean:generated
```

**When to use**:
- When you want to keep artifacts but clean other generated files
- Faster than full clean

---

### Advanced Clean Commands

#### `npm run clean:bindings`
**Description**: Remove all language binding generated files

**Removes**:
- **Go bindings**: `bindings/go/*.go`, `go.mod`, `go.sum`
- **Java bindings**: `bindings/java/target/`, `build/`, `*.class`
- **Python bindings**: `bindings/python/build/`, `dist/`, `__pycache__/`, `*.pyc`
- **Rust bindings**: `bindings/rust/target/`

**Preserves**:
- README.md files in each binding directory
- Example files

**Usage**:
```bash
npm run clean:bindings
```

**When to use**:
- Before regenerating bindings with `npm run generate:all`
- When bindings are corrupted
- To free up disk space

---

#### `npm run clean:deployments`
**Description**: Remove deployment JSON files

**Removes**:
- All `*.json` files in `deployments/` directory

**Preserves**:
- `example.json` - Example deployment file
- `*-latest.json` - Latest deployment records
- `*.md` - Documentation files

**Usage**:
```bash
npm run clean:deployments
```

**When to use**:
- Before fresh deployment testing
- To clean up old deployment records
- When switching networks

---

#### `npm run clean:all`
**Description**: Complete clean - all generated files

**Executes**:
1. `npm run clean`
2. `npm run clean:bindings`
3. `npm run clean:deployments`
4. Removes `abi/*.min.json`

**Usage**:
```bash
npm run clean:all
```

**When to use**:
- Starting fresh development
- Before major version updates
- Preparing for clean build
- Maximum disk space recovery

---

#### `npm run clean:deep`
**Description**: Deep clean - everything including node_modules

**Executes**:
1. `npm run clean:all`
2. Removes `node_modules/`

**Usage**:
```bash
npm run clean:deep
```

**- -  Warning**: After this command, you must run `npm install` to restore dependencies.

**When to use**:
- Resolving dependency conflicts
- Complete project reset
- Before archiving project
- Investigating npm package issues

**After running**:
```bash
npm install
```

---

##  Typical Workflows

### Development Workflow

```bash
# Daily development
npm run clean         # Clean before each compile

# When tests fail mysteriously
npm run clean:all     # Deep clean and retry
npm run compile
npm test

# Before switching git branches
npm run clean:all     # Avoid conflicts
git checkout <branch>
npm install           # Restore dependencies if needed
```

### Deployment Workflow

```bash
# Before deployment
npm run clean:all     # Start fresh
npm run compile       # Clean compile
npm test              # Verify tests pass
npm run deploy:sepolia
```

### Binding Generation Workflow

```bash
# Regenerate all bindings
npm run clean:bindings  # Remove old bindings
npm run generate:all    # Generate fresh bindings
```

### Troubleshooting Workflow

```bash
# When nothing works
npm run clean:deep    # Nuclear option
npm install           # Reinstall dependencies
npm run compile       # Fresh compile
npm test              # Test everything
```

---

## -  Disk Space Recovered

Approximate disk space recovered by each command:

| Command | Typical Size | Description |
|---------|-------------|-------------|
| `clean` | 50-200 MB | Build artifacts, cache |
| `clean:bindings` | 20-100 MB | Language bindings |
| `clean:deployments` | <1 MB | Deployment records |
| `clean:all` | 70-300 MB | All generated files |
| `clean:deep` | 400-800 MB | Including node_modules |

*Actual sizes vary based on project state and dependencies*

---

##  What's Preserved

These files are **never** deleted by clean commands:

### Source Files
- `contracts/**/*.sol` - Smart contracts
- `test/**/*.js` - Test files
- `scripts/**/*.js` - Deployment scripts

### Configuration
- `hardhat.config.js` - Hardhat configuration
- `package.json`, `package-lock.json` - NPM configuration
- `.env`, `.env.example` - Environment files
- `.gitignore` - Git configuration
- `*.md` - Documentation

### Important Generated Files
- `deployments/example.json` - Example deployment
- `deployments/*-latest.json` - Latest deployment records
- `bindings/**/README.md` - Binding documentation
- `bindings/**/example.*` - Example files
- `verification/VERIFICATION_GUIDE.md` - Verification guide
- `flattened/VERIFICATION_INSTRUCTIONS.md` - Flatten instructions

---

## -  Git Integration

The `.gitignore` is configured to automatically ignore all files cleaned by these commands:

```bash
# These are safe to commit after cleaning
git add .
git commit -m "Clean build"

# No generated files will be included
```

---

## 🆘 Troubleshooting

### "Permission denied" errors

Some files may be locked by running processes:

```bash
# Stop any running Hardhat node
npm run node:stop

# Or kill all node processes (macOS/Linux)
killall node

# Then retry clean
npm run clean:all
```

### "Directory not empty" errors

On Windows, some directories may be locked:

```bash
# Close all terminals and IDEs
# Wait 10 seconds
# Retry the command
npm run clean:all
```

### Clean script hangs

If clean script appears to hang:

```bash
# Press Ctrl+C to cancel
# Check for running processes
npm run node:status

# Clean up manually if needed
rm -rf artifacts cache typechain-types coverage
```

---

## -  Script Details

All clean scripts use:
- `rm -rf` for directory removal
- `find` for pattern matching
- `grep -v node_modules` to avoid node_modules
- `2>/dev/null || true` to ignore errors
- `! -name` to preserve specific files

This ensures:
-  Safe execution even if directories don't exist
-  No errors if files are already deleted
-  Preserved files are never touched
-  Cross-platform compatibility

---

**Last Updated**: 2025-10-26
**Version**: v1.3.1
