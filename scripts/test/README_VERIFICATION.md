# SAGE Specification Verification Script

## Overview

`verify_all_specifications.sh` is a comprehensive test verification script that runs all tests defined in the SPECIFICATION_VERIFICATION_MATRIX.md document.

## Features

- Automated testing of all 9 chapters
- Colorful, real-time progress output
- Detailed logging for each chapter
- Comprehensive HTML-style report generation
- Performance timing for each chapter
- Pass/Fail/Skip status tracking
- Test data verification

## Usage

### Basic Usage

```bash
# Run all specification tests
./scripts/test/verify_all_specifications.sh
```

### Prerequisites

1. **Go** must be installed
2. **Project root**: Must run from SAGE project directory
3. **Hardhat node** (optional): For blockchain tests
   ```bash
   cd contracts/ethereum
   npx hardhat node
   ```

### What It Tests

| Chapter | Name | Package(s) |
|---------|------|-----------|
| 1 | RFC 9421 | `pkg/agent/core/rfc9421` |
| 2 | Key Management | `pkg/agent/crypto/{keys,rotation,storage,vault}` |
| 3 | DID Management | `pkg/agent/did/...` |
| 4 | Blockchain | `tests` (blockchain tests) |
| 5 | Message Processing | `pkg/agent/core/message/...`, `pkg/agent/transport/...` |
| 6 | CLI Tools | `cmd/...`, `tests/integration` (CLI tests) |
| 7 | Session Management | `pkg/agent/session`, `pkg/agent/handshake` |
| 8 | HPKE | `pkg/agent/hpke` |
| 9 | Health Check | `pkg/health`, `tests/integration` (health tests) |

## Output

### Console Output

The script provides colorful, real-time output:

```
╔══════════════════════════════════════════════════════════════╗
║  SAGE Specification Verification                             ║
╚══════════════════════════════════════════════════════════════╝

INFO: Starting verification at 2025-10-24 07:30:00
INFO: Project root: /path/to/sage
INFO: Log directory: /path/to/sage/logs/verification

═══ Checking Prerequisites ═══

PASS: Go go1.23.0
PASS: In SAGE project root
PASS: Hardhat node is running (Chain ID 31337)

═══ Chapter 1: RFC 9421 ═══

Running: RFC 9421 Implementation tests
PASS: Chapter 1: RFC 9421

...

╔══════════════════════════════════════════════════════════════╗
║  VERIFICATION SUMMARY                                         ║
╚══════════════════════════════════════════════════════════════╝

Results by Chapter:

  [PASS] Chapter 1 (RFC 9421): PASS (2s)
  [PASS] Chapter 2 (Key Management): PASS (3s)
  [PASS] Chapter 3 (DID Management): PASS (1s)
  ...

Overall Statistics:
  Total Chapters:  9
  Passed:         9
  Failed:         0
  Skipped:        0
  Total Tests:     234
  Passed Tests:   234
  Success Rate:    100.0%

Files Generated:
  Report: logs/verification/verification_report_20251024_073000.md
  Logs:   logs/verification
```

### Generated Files

#### 1. Verification Report

Location: `logs/verification/verification_report_YYYYMMDD_HHMMSS.md`

Contains:
- Executive summary with statistics
- Chapter-by-chapter results table
- Verification data file counts
- Next steps recommendations
- Links to detailed logs

Example:
```markdown
# SAGE Specification Verification Report

**Date**: 2025-10-24 07:30:00
**Project**: SAGE (Secure Agent Guarantee Engine)

## Executive Summary

╔══════════════════════════════════════════════════════════════╗
║  SAGE Specification Verification Results                     ║
╠══════════════════════════════════════════════════════════════╣
║  Total Chapters:     9                                       ║
║  Passed:          9                                          ║
║  Failed:          0                                          ║
║  Success Rate:       100.0%                                  ║
╚══════════════════════════════════════════════════════════════╝

## Chapter Results

| # | Chapter Name | Status | Time (s) | Log File |
|---|-------------|--------|----------|----------|
| 1 | RFC 9421 | PASS | 2s | chapter1_rfc9421.log |
| 2 | Key Management | PASS | 3s | chapter2_keys.log |
...
```

#### 2. Chapter Log Files

Location: `logs/verification/chapter<N>_<name>.log`

Contains verbose test output for each chapter:
- All test execution details
- Pass/fail/skip status for each test
- Error messages and stack traces
- Test data file paths

Example files:
- `chapter1_rfc9421.log`
- `chapter2_keys.log`
- `chapter3_did.log`
- `chapter4_blockchain.log`
- ... etc.

## Log Analysis

### View Specific Chapter Log

```bash
# View with less (recommended)
less logs/verification/chapter1_rfc9421.log

# View with cat
cat logs/verification/chapter1_rfc9421.log

# Search for failures
grep -i "fail" logs/verification/chapter1_rfc9421.log

# Count passed tests
grep -c "^--- PASS:" logs/verification/chapter1_rfc9421.log

# Show only test names
grep "^=== RUN" logs/verification/chapter1_rfc9421.log
```

### View Latest Report

```bash
# Find latest report
ls -lt logs/verification/verification_report_*.md | head -1

# View latest report
cat $(ls -t logs/verification/verification_report_*.md | head -1)
```

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed

## Troubleshooting

### Issue: Hardhat Node Not Running

**Symptom**: Chapter 4 is skipped

**Solution**:
```bash
# Terminal 1: Start Hardhat
cd contracts/ethereum
npx hardhat node

# Terminal 2: Run tests
./scripts/test/verify_all_specifications.sh
```

### Issue: Integration Tests Skipped

**Symptom**: Many tests in Chapter 3 show "SKIP"

**Solution**: Run with integration test flag:
```bash
SAGE_INTEGRATION_TEST=1 go test -v ./pkg/agent/did/ethereum
```

### Issue: Permission Denied

**Symptom**: `bash: permission denied`

**Solution**:
```bash
chmod +x scripts/test/verify_all_specifications.sh
```

## Advanced Usage

### Run Only Specific Chapters

Modify the script or run tests directly:

```bash
# Chapter 1 only
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421

# Chapter 4 only (blockchain)
go test -v ./tests -run "TestBlockchain"

# Chapters 1, 2, 3
go test -v \
  github.com/sage-x-project/sage/pkg/agent/core/rfc9421 \
  github.com/sage-x-project/sage/pkg/agent/crypto/keys \
  github.com/sage-x-project/sage/pkg/agent/did/...
```

### With Coverage

```bash
# Add coverage tracking
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Verbose Output

```bash
# Already verbose by default
# For even more detail, check individual logs
cat logs/verification/chapter*.log
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Specification Verification

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Set up Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install Hardhat
        run: |
          cd contracts/ethereum
          npm install
      
      - name: Start Hardhat node
        run: |
          cd contracts/ethereum
          npx hardhat node &
          sleep 10
      
      - name: Run Specification Tests
        run: ./scripts/test/verify_all_specifications.sh
      
      - name: Upload Test Reports
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-reports
          path: logs/verification/
```

## Maintenance

### Update Test Suites

When adding new tests to SPECIFICATION_VERIFICATION_MATRIX.md:

1. Update the corresponding `run_chapter_N()` function
2. Add new test packages to the `go test` command
3. Update the chapter name and description
4. Test the changes:
   ```bash
   ./scripts/test/verify_all_specifications.sh
   ```

### Cleanup Old Logs

```bash
# Remove logs older than 7 days
find logs/verification -name "*.log" -mtime +7 -delete
find logs/verification -name "verification_report_*.md" -mtime +7 -delete

# Keep only latest 10 reports
ls -t logs/verification/verification_report_*.md | tail -n +11 | xargs rm -f
```

## Related Documentation

- [SPECIFICATION_VERIFICATION_MATRIX.md](../../docs/test/SPECIFICATION_VERIFICATION_MATRIX.md) - Main specification document
- [CLAUDE.md](../../CLAUDE.md) - Development guidelines
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines

## Support

For issues or questions:
1. Check the generated logs in `logs/verification/`
2. Review the verification report
3. Check the [GitHub Issues](https://github.com/sage-x-project/sage/issues)
4. Open a new issue with:
   - Error message
   - Relevant log file
   - System information (OS, Go version)

---

**Last Updated**: 2025-10-24
**Script Version**: 1.0
