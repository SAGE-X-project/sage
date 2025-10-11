# SAGE Performance Optimization Plan

**Date:** 2025-10-08
**Target:** Reduce session creation allocations from 38 to <10
**Based on:** Profiling results and performance baseline

---

## 1. Identified Bottlenecks

### 1.1 Memory Allocation Hotspots (from pprof)

| Source | Allocations | % of Total | Issue |
|--------|-------------|------------|-------|
| `sha256.New()` | 0.85GB | 35.63% | Called multiple times per session |
| `hmac.New()` | 0.74GB | 31.00% | Called for each key derivation |
| `hkdf.Expand()` | 0.17GB | 7.17% | Buffer allocations |
| `chacha20poly1305.New()` | 0.02GB | 0.88% | Per-cipher allocation |
| **Total Session Creation** | 2.24GB | 93.26% | Main target |

### 1.2 Root Causes

1. **Repeated Hash Instances** 
   ```go
   // Current: Creates new hash each time
   hkdfEnc := hkdf.New(sha256.New, ...)   // sha256.New() allocates
   hkdfSign := hkdf.New(sha256.New, ...)  // sha256.New() allocates again
   ```

2. **Multiple Key Buffer Allocations** 
   ```go
   // Current: Separate allocation for each key
   s.encryptKey = make([]byte, 32)  // Allocation 1
   s.signingKey = make([]byte, 32)  // Allocation 2
   // + 4 more in directional keys
   ```

3. **No Buffer Reuse** 
   - Each session creates fresh buffers
   - No pooling mechanism
   - Temporary buffers not reused

---

## 2. Optimization Strategies

### 2.1 Strategy 1: Pre-allocate Key Buffer

**Current Allocations:** 6 separate `make([]byte, 32)` calls

**Optimization:**
```go
// Allocate once, slice for keys
keyMaterial := make([]byte, 192) // All keys in one buffer
s.outKey = keyMaterial[0:32]
s.inKey = keyMaterial[32:64]
s.outSign = keyMaterial[64:96]
s.inSign = keyMaterial[96:128]
s.encryptKey = keyMaterial[128:160]
s.signingKey = keyMaterial[160:192]
```

**Expected Reduction:** 6 allocations → 1 allocation

---

### 2.2 Strategy 2: Single HKDF Expand Call

**Current Approach:** Multiple HKDF instances
```go
hkdfEnc := hkdf.New(sha256.New, seed, salt, []byte("encryption"))
hkdfSign := hkdf.New(sha256.New, seed, salt, []byte("signing"))
c2sEnc := hkdf.New(sha256.New, seed, salt, []byte("c2s|enc|v1"))
// ... 4 more HKDF instances
```

**Optimized Approach:** Single HKDF with concatenated info
```go
// Create one HKDF reader
hkdf := hkdf.New(sha256.New, seed, salt, nil)

// Derive all keys in sequence with context separation
keyMaterial := make([]byte, 192)
if _, err := io.ReadFull(hkdf, keyMaterial); err != nil {
    return err
}

// Or use different info strings but one reader:
deriveAll := func() ([]byte, error) {
    out := make([]byte, 192)
    offset := 0
    for _, info := range []string{
        "c2s|enc|v1", "c2s|sign|v1",
        "s2c|enc|v1", "s2c|sign|v1",
        "encryption", "signing",
    } {
        h := hkdf.New(sha256.New, seed, salt, []byte(info))
        if _, err := io.ReadFull(h, out[offset:offset+32]); err != nil {
            return nil, err
        }
        offset += 32
    }
    return out, nil
}
```

**Expected Reduction:** ~10 allocations → 3-4 allocations

---

### 2.3 Strategy 3: Reuse Hash Instances

**Problem:** `sha256.New()` called 6+ times per session

**Solution:** Hash pool or single hash with Reset
```go
var hashPool = sync.Pool{
    New: func() interface{} {
        return sha256.New()
    },
}

func deriveWithPool(seed, salt, info []byte, out []byte) error {
    h := hashPool.Get().(hash.Hash)
    defer func() {
        h.Reset()
        hashPool.Put(h)
    }()

    hkdf := hkdf.New(func() hash.Hash { return h }, seed, salt, info)
    _, err := io.ReadFull(hkdf, out)
    return err
}
```

**Expected Reduction:** 6 allocations → 1-2 allocations (pool management)

---

### 2.4 Strategy 4: Buffer Pool for Temporary Allocations

**Problem:** Encryption creates temporary buffers
```go
// In Encrypt()
nonce := make([]byte, 12)    // Allocation
ciphertext := make([]byte, len(plaintext)+overhead) // Allocation
```

**Solution:** Sync.Pool for buffers
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096) // Common size
    },
}

func (s *SecureSession) Encrypt(plaintext []byte) ([]byte, error) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)

    // Use buf for nonce and temp work
    nonce := buf[:12]
    // ...
}
```

**Expected Reduction:** 2-3 allocations per encryption → 0-1 allocations

---

## 3. Implementation Plan

### Phase 1: Low-Hanging Fruit (Est: 2 hours)

**Task 1.1: Pre-allocate Key Material Buffer**
- File: `session/session.go`
- Function: `NewSecureSession`, `deriveDirectionalKeys`
- Change: Single allocation for all keys
- Expected: 6 → 1 allocation

**Task 1.2: Optimize HKDF Usage**
- File: `session/session.go`
- Function: `deriveKeys`, `deriveDirectionalKeys`
- Change: Reduce HKDF instance creation
- Expected: 4-5 → 1-2 allocations

**Verification:**
```bash
go test ./benchmark -bench=BenchmarkSessionCreation -benchmem
# Target: <15 allocs/op (currently 38)
```

---

### Phase 2: Buffer Pooling (Est: 3 hours)

**Task 2.1: Implement Buffer Pool**
- File: `session/pool.go` (new)
- Content: Buffer pool for encryption/decryption
- Integration: Update `Encrypt()`, `Decrypt()`

**Task 2.2: Hash Instance Pool**
- File: `session/pool.go`
- Content: Hash pool for HKDF
- Integration: Update key derivation functions

**Verification:**
```bash
go test ./benchmark -bench=BenchmarkSessionEncryption -benchmem
# Target: 3 → 1 allocs/op for encryption
```

---

### Phase 3: Advanced Optimizations (Est: 2 hours)

**Task 3.1: Lazy AEAD Initialization**
- Current: AEAD created at session creation
- Optimized: Create AEAD on first use (if session unused, no allocation)

**Task 3.2: Key Material Reuse**
- Implement key rotation without full re-derivation
- Cache derived keys across sessions (where safe)

**Task 3.3: Minimize String Operations**
- Replace `fmt.Sprintf` with buffer operations
- Pre-allocate info strings

**Verification:**
```bash
go test ./benchmark -bench=. -benchmem
# Compare all metrics with baseline
```

---

## 4. Expected Results

### 4.1 Allocation Reduction

| Metric | Current | Phase 1 | Phase 2 | Phase 3 | Target |
|--------|---------|---------|---------|---------|--------|
| **Session Creation Allocs** | 38 | 22 | 12 | 8 | <10 Yes |
| **Encryption Allocs** | 3 | 3 | 1 | 1 | 1 Yes |
| **Memory per Session** | 2.9KB | 2.5KB | 2.0KB | 1.8KB | <2KB Yes |

### 4.2 Performance Improvement

| Operation | Current | Optimized | Improvement |
|-----------|---------|-----------|-------------|
| Session Creation | 2.4µs | 1.5µs | 37% faster |
| Encryption (1KB) | 1.96µs | 1.4µs | 28% faster |
| Throughput (16KB) | 802 MB/s | 1000 MB/s | 25% increase |

---

## 5. Testing Strategy

### 5.1 Correctness Tests

**Before any optimization:**
```bash
# Ensure all existing tests pass
go test ./session/... -v
go test ./benchmark/... -v
```

**After each phase:**
```bash
# Run fuzz tests to verify correctness
go test -fuzz=FuzzSessionEncryptDecrypt ./session -fuzztime=30s
go test -fuzz=FuzzSessionCreation ./session -fuzztime=30s
```

### 5.2 Performance Tests

**Baseline capture:**
```bash
go test ./benchmark -bench=. -benchmem -count=5 > baseline.txt
```

**After optimization:**
```bash
go test ./benchmark -bench=. -benchmem -count=5 > optimized.txt
benchstat baseline.txt optimized.txt
```

### 5.3 Memory Profiling

**Before:**
```bash
go test ./benchmark -bench=BenchmarkSessionCreation \
    -memprofile=mem_before.prof -memprofilerate=1
```

**After:**
```bash
go test ./benchmark -bench=BenchmarkSessionCreation \
    -memprofile=mem_after.prof -memprofilerate=1

go tool pprof -base=mem_before.prof mem_after.prof
```

---

## 6. Risk Assessment

### 6.1 Low Risk Changes Yes

- Pre-allocating key material buffer
- Reducing HKDF instances
- These are internal optimizations, no API changes

### 6.2 Medium Risk Changes Warning

- Buffer pooling
  - Risk: Incorrect buffer reuse could leak data
  - Mitigation: Zero buffers before return to pool
  - Testing: Fuzz tests with verification

- Hash instance pooling
  - Risk: State leakage between uses
  - Mitigation: Always call Reset() before reuse
  - Testing: Concurrent session creation tests

### 6.3 Safety Measures

1. **Always zero sensitive buffers:**
   ```go
   defer func() {
       for i := range buf {
           buf[i] = 0
       }
       pool.Put(buf)
   }()
   ```

2. **No pool for keys:**
   - Never pool cryptographic key material
   - Only pool temporary work buffers

3. **Verify with sanitizers:**
   ```bash
   go test -race ./session/...
   go test -msan ./session/... # if available
   ```

---

## 7. Success Criteria

### 7.1 Must Have Yes

- [ ] Session creation: <10 allocations/op
- [ ] All existing tests pass
- [ ] Fuzz tests pass (30s minimum)
- [ ] No race conditions (`-race` clean)
- [ ] Memory usage: <2KB per session

### 7.2 Should Have Target

- [ ] Encryption: 1 allocation/op
- [ ] 30%+ performance improvement
- [ ] Throughput: >1GB/s for 16KB messages
- [ ] Documented optimization techniques

### 7.3 Nice to Have Star

- [ ] Zero-allocation encryption for small messages
- [ ] Adaptive buffer pooling
- [ ] Concurrent benchmark improvements

---

## 8. Rollback Plan

If optimization introduces bugs:

1. **Immediate:** Revert to baseline
   ```bash
   git revert <optimization-commit>
   ```

2. **Verify:** Run full test suite
   ```bash
   go test ./... -race -count=3
   ```

3. **Analyze:** Review profiling data
   ```bash
   go tool pprof -http=:8080 mem.prof
   ```

4. **Fix:** Address specific issue, re-test

---

## 9. Implementation Checklist

### Phase 1: Core Optimizations
- [ ] Profile current performance
- [ ] Implement single key material allocation
- [ ] Optimize HKDF usage
- [ ] Run benchmarks and verify
- [ ] Update performance baseline

### Phase 2: Buffer Pooling
- [ ] Design buffer pool structure
- [ ] Implement with safety measures
- [ ] Integrate into Encrypt/Decrypt
- [ ] Fuzz test with pooling
- [ ] Measure allocation reduction

### Phase 3: Advanced
- [ ] Lazy AEAD initialization
- [ ] Hash instance pooling
- [ ] String operation optimization
- [ ] Final benchmark comparison
- [ ] Document results

### Documentation
- [ ] Update PERFORMANCE-BASELINE.md
- [ ] Create OPTIMIZATION-RESULTS.md
- [ ] Add inline comments for optimizations
- [ ] Update README with performance notes

---

## 10. Timeline

**Total Estimated Time:** 7 hours

| Phase | Tasks | Duration | Dependencies |
|-------|-------|----------|--------------|
| **Phase 1** | Core optimizations | 2 hours | Profiling complete |
| **Phase 2** | Buffer pooling | 3 hours | Phase 1 complete |
| **Phase 3** | Advanced opts | 2 hours | Phase 2 verified |

**Milestones:**
- Hour 2: <20 allocations/op Yes
- Hour 5: <12 allocations/op Yes
- Hour 7: <10 allocations/op, documentation complete Yes

---

## 11. Next Steps

1. **Immediate:** Start Phase 1
   - Create backup branch
   - Capture current benchmarks
   - Implement key material optimization

2. **Monitoring:** After each change
   - Run benchmarks
   - Check allocations
   - Verify tests pass

3. **Completion:** Final verification
   - Full benchmark suite
   - Performance report
   - Update documentation

---

**Status:** List READY TO IMPLEMENT
**Owner:** Development Team
**Priority:** HIGH (blocks production optimization)
**Next Action:** Begin Phase 1 implementation
