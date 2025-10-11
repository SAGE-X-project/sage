# SAGE Fuzzing Guide

Comprehensive fuzzing and property-based testing for SAGE.

## Overview

SAGE uses multiple fuzzing approaches to ensure robustness and security:

1. **Go Native Fuzzing**: Built-in Go 1.18+ fuzzing for backend code
2. **Foundry Fuzzing**: Property-based testing for Solidity smart contracts
3. **Invariant Testing**: Continuous property validation
4. **Differential Testing**: Compare implementations for consistency

## Go Fuzzing

### What is Fuzzing?

Fuzzing automatically generates random inputs to test code paths and find bugs that traditional tests might miss. Go's native fuzzing:

- Generates diverse inputs automatically
- Learns from crashes to create interesting test cases
- Maintains a corpus of test inputs
- Provides coverage-guided fuzzing

### Running Go Fuzz Tests

```bash
# Run all fuzz tests
./scripts/run-fuzz.sh

# Run for specific duration
./scripts/run-fuzz.sh --time 5m

# Run only Go fuzzing
./scripts/run-fuzz.sh --type go --time 1m

# Run specific fuzzer
go test -fuzz=FuzzKeyPairGeneration -fuzztime=30s ./crypto

# Run with multiple parallel workers
go test -fuzz=FuzzSignAndVerify -fuzztime=1m -parallel=4 ./crypto
```

### Available Go Fuzz Tests

#### Crypto Package

**FuzzKeyPairGeneration**
- Tests key generation for Ed25519, Secp256k1, X25519
- Validates key properties
- Ensures no panics with random key types

```bash
go test -fuzz=FuzzKeyPairGeneration -fuzztime=30s ./crypto
```

**FuzzSignAndVerify**
- Tests message signing and verification
- Validates signature correctness
- Ensures modified messages/signatures fail
- Tests with random message sizes

```bash
go test -fuzz=FuzzSignAndVerify -fuzztime=30s ./crypto
```

**FuzzKeyExportImport**
- Tests JWK and PEM export/import
- Validates round-trip consistency
- Tests with different key types

```bash
go test -fuzz=FuzzKeyExportImport -fuzztime=30s ./crypto
```

**FuzzSignatureWithDifferentKeys**
- Ensures signatures fail with wrong keys
- Validates key isolation
- Tests cross-contamination prevention

```bash
go test -fuzz=FuzzSignatureWithDifferentKeys -fuzztime=30s ./crypto
```

**FuzzInvalidSignatureData**
- Tests behavior with invalid signature data
- Ensures no panics with garbage input
- Validates error handling

```bash
go test -fuzz=FuzzInvalidSignatureData -fuzztime=30s ./crypto
```

**FuzzKeyDerivation**
- Tests HPKE key derivation
- Validates determinism
- Tests with random contexts

```bash
go test -fuzz=FuzzKeyDerivation -fuzztime=30s ./crypto
```

#### Session Package

**FuzzSessionCreation**
- Tests session creation with random parameters
- Validates session properties
- Tests various timeout values

```bash
go test -fuzz=FuzzSessionCreation -fuzztime=30s ./session
```

**FuzzSessionEncryptDecrypt**
- Tests encryption/decryption round-trips
- Validates ciphertext integrity
- Tests with various message sizes (0 to 64KB)

```bash
go test -fuzz=FuzzSessionEncryptDecrypt -fuzztime=30s ./session
```

**FuzzNonceValidation**
- Tests nonce validation and replay prevention
- Validates timestamp handling
- Ensures duplicate nonces are rejected

```bash
go test -fuzz=FuzzNonceValidation -fuzztime=30s ./session
```

**FuzzSessionExpiration**
- Tests session expiration logic
- Validates max age and idle timeout
- Tests cleanup behavior

```bash
go test -fuzz=FuzzSessionExpiration -fuzztime=30s ./session
```

**FuzzConcurrentSessionAccess**
- Tests thread safety
- Validates concurrent encryption/decryption
- Ensures no data races

```bash
go test -fuzz=FuzzConcurrentSessionAccess -fuzztime=30s ./session
```

**FuzzInvalidSessionData**
- Tests behavior with invalid data
- Ensures no panics
- Validates error handling

```bash
go test -fuzz=FuzzInvalidSessionData -fuzztime=30s ./session
```

### Understanding Fuzz Results

#### Success Output

```
fuzz: elapsed: 0s, gathering baseline coverage: 0/2 completed
fuzz: elapsed: 3s, gathering baseline coverage: 2/2 completed, now fuzzing with 8 workers
fuzz: elapsed: 6s, execs: 12345 (4115/sec), new interesting: 5 (total: 7)
fuzz: elapsed: 9s, execs: 24567 (4074/sec), new interesting: 2 (total: 9)
PASS
```

- `execs`: Number of fuzzing iterations
- `new interesting`: Inputs that increase coverage
- `PASS`: No crashes found

#### Crash Output

```
--- FAIL: FuzzSignAndVerify (0.03s)
    --- FAIL: FuzzSignAndVerify (0.00s)
        testing.go:1319: panic: runtime error: index out of range [0] with length 0

    Failing input written to testdata/fuzz/FuzzSignAndVerify/a1b2c3d4e5f6
    To re-run:
    go test -run=FuzzSignAndVerify/a1b2c3d4e5f6
FAIL
```

When a crash occurs:
1. Crash input saved to `testdata/fuzz/FuzzSignAndVerify/`
2. Reproduce with: `go test -run=FuzzSignAndVerify/CRASHHASH`
3. Minimize with: `go test -fuzz=FuzzSignAndVerify -run=FuzzSignAndVerify/CRASHHASH`
4. Fix the bug
5. Re-run fuzzer to verify fix

### Corpus Management

Fuzz tests maintain a corpus of interesting inputs:

```
testdata/
└── fuzz/
    ├── FuzzKeyPairGeneration/
    │   ├── a1b2c3d4e5f6
    │   └── f6e5d4c3b2a1
    └── FuzzSignAndVerify/
        ├── 123456789abc
        └── cba987654321
```

**Benefits:**
- Tests run with corpus inputs first
- Corpus grows with interesting inputs
- Commit corpus to Git for regression testing
- Share corpus across team

**Managing Corpus:**

```bash
# Clear corpus for fresh start
rm -rf testdata/fuzz/FuzzTestName

# Seed corpus manually
mkdir -p testdata/fuzz/FuzzTestName
echo -n "test input" > testdata/fuzz/FuzzTestName/seed1

# Merge corpus from multiple runs
go test -fuzz=FuzzTestName -fuzzminimizetime=0 -run=^$ ./package
```

## Foundry Fuzzing (Solidity)

### Setup Foundry

```bash
# Install Foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Verify installation
forge --version
```

### Running Solidity Fuzz Tests

```bash
# Run all fuzz tests
./scripts/run-fuzz.sh --type solidity

# Run with Foundry directly
cd contracts/ethereum
forge test --match-test "testFuzz_"

# Run specific test
forge test --match-test "testFuzz_RegisterDID" -vv

# Run invariant tests
forge test --match-test "invariant_"

# Increase fuzz runs
forge test --match-test "testFuzz_" --fuzz-runs 1000
```

### Available Solidity Fuzz Tests

**testFuzz_RegisterDID**
- Tests DID registration with random inputs
- Validates string length limits
- Tests public key size constraints

**testFuzz_PreventDuplicateRegistration**
- Ensures duplicate DIDs are rejected
- Validates ownership preservation
- Tests with random DID/key combinations

**testFuzz_UpdatePublicKey**
- Tests public key updates
- Validates ownership requirements
- Tests with random keys

**testFuzz_UnauthorizedUpdate**
- Ensures only owners can update
- Tests access control
- Validates with random users

**testFuzz_RevokeDID**
- Tests DID revocation
- Validates revocation state
- Ensures revoked DIDs can't be used

**testFuzz_UnauthorizedRevocation**
- Tests access control for revocation
- Validates only owners can revoke

**testFuzz_BatchRegistration**
- Tests batch operations
- Validates array handling
- Tests with random array sizes

**testFuzz_OwnershipTransfer**
- Tests ownership transfer (if implemented)
- Validates access control

### Invariant Testing

Invariants are properties that should always be true:

```solidity
function invariant_TotalDIDsNeverDecrease() public view {
    // Total registered DIDs should never decrease
}

function invariant_RevokedDIDsUnusable() public {
    // Revoked DIDs cannot be used for operations
}
```

Run invariant tests:

```bash
cd contracts/ethereum
forge test --match-test "invariant_"
```

### Foundry Configuration

See `contracts/ethereum/foundry.toml`:

```toml
[profile.default]
fuzz = { runs = 256 }
invariant = { runs = 256, depth = 15 }
```

Adjust for longer tests:

```toml
fuzz = { runs = 10000 }
invariant = { runs = 1000, depth = 50 }
```

## Best Practices

### Writing Fuzz Tests

1. **Use `vm.assume()` to filter invalid inputs:**

```solidity
function testFuzz_Example(uint256 x) public {
    vm.assume(x > 0 && x < 1000);
    // Test logic
}
```

2. **Test invariants, not specifics:**

```go
// Good: Test property
if len(encrypted) <= len(plaintext) {
    t.Fatal("Ciphertext should be larger than plaintext")
}

// Bad: Test specific value
if len(encrypted) != len(plaintext) + 16 {
    t.Fatal("Unexpected ciphertext length")
}
```

3. **Validate error handling:**

```go
// Ensure no panics
defer func() {
    if r := recover(); r != nil {
        t.Errorf("Panic: %v", r)
    }
}()
```

4. **Test edge cases in corpus:**

```go
f.Add([]byte(""))           // Empty input
f.Add([]byte("a"))          // Single byte
f.Add(make([]byte, 65536))  // Large input
```

### Coverage Goals

Target coverage levels:

- **Crypto**: 95%+ (critical security code)
- **Session**: 90%+ (high-value code)
- **Contracts**: 95%+ (immutable code)
- **Other**: 85%+

Check coverage:

```bash
# Go coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Solidity coverage
cd contracts/ethereum
forge coverage
```

### Continuous Fuzzing

Run fuzzing in CI/CD:

```yaml
# .github/workflows/fuzz.yml
name: Fuzz Tests

on:
  schedule:
    - cron: '0 0 * * *'  # Daily

jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run fuzz tests
        run: ./scripts/run-fuzz.sh --time 10m

      - name: Upload crash files
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: fuzz-crashes
          path: testdata/fuzz/
```

## Troubleshooting

### Fuzz Test Hangs

If a fuzz test hangs:

1. Reduce fuzz time: `-fuzztime=10s`
2. Check for infinite loops in test logic
3. Use timeout: `go test -timeout=5m -fuzz=...`

### Out of Memory

If fuzzer runs out of memory:

1. Reduce parallel workers: `-parallel=1`
2. Limit input size with `vm.assume()`
3. Clear corpus: `rm -rf testdata/fuzz/`

### False Positives

If fuzzer finds invalid "crashes":

1. Review the crash input
2. Add input validation: `vm.assume(condition)`
3. Update test logic to handle edge cases

### Slow Fuzzing

To speed up fuzzing:

1. Increase workers: `-parallel=8`
2. Use faster hardware
3. Focus on high-value tests
4. Run long fuzzing sessions overnight

## Additional Resources

- [Go Fuzzing Documentation](https://go.dev/security/fuzz/)
- [Foundry Book - Fuzzing](https://book.getfoundry.sh/forge/fuzz-testing)
- [Property-Based Testing Guide](https://hypothesis.works/articles/what-is-property-based-testing/)
- [SAGE Testing Documentation](./TESTING.md)

## Contributing

When adding new fuzz tests:

1. Follow naming convention: `FuzzTestName` (Go), `testFuzz_TestName` (Solidity)
2. Add seed corpus with `f.Add()` or in testdata
3. Test invariants, not implementation details
4. Document what properties are being tested
5. Run locally before committing
6. Include corpus in commits for regression testing
