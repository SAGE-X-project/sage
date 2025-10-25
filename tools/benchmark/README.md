## SAGE Performance Benchmarks

Comprehensive performance benchmarking suite for SAGE (Secure Agent Guarantee Engine).

## Overview

This benchmark suite measures and compares the performance of SAGE's security features against baseline (non-secure) implementations. It provides detailed metrics on throughput, latency, memory usage, and resource consumption.

### Benchmark Categories

1. **Cryptographic Operations** (`crypto_bench_test.go`)
   - Key generation (Ed25519, Secp256k1, X25519)
   - Message signing and verification
   - Key import/export (JWK, PEM)
   - Performance across different message sizes

2. **HPKE Operations** (`pkg/agent/hpke/hpke_bench_test.go`)
   - HPKE sender/receiver derivation
   - HPKE seal/open operations
   - Export secret derivation
   - Key derivation with various parameters
   - X25519 key generation

3. **Handshake Protocol** (`pkg/agent/handshake/handshake_bench_test.go`)
   - Key pair generation for handshake
   - Signature generation and verification
   - Session encryption/decryption
   - Complete handshake roundtrip
   - Message size scaling

4. **Session Management** (`session_bench_test.go`)
   - Session creation and lifecycle
   - Message encryption/decryption
   - Handshake protocol performance
   - Concurrent session operations
   - Nonce validation

5. **RFC 9421 HTTP Signatures** (`rfc9421_bench_test.go`)
   - HTTP message signing
   - Signature verification
   - Different signature components
   - HMAC-based signatures
   - Payload size variations

6. **Baseline Comparisons** (`comparison_bench_test.go`)
   - SAGE vs. no security
   - SAGE vs. simple hash-based integrity
   - Throughput measurements
   - Latency percentiles (p50, p95, p99)
   - Memory usage analysis

## Quick Start

### Run All Benchmarks

```bash
./scripts/run-benchmarks.sh
```

### Run Specific Category

```bash
# Crypto benchmarks only
./scripts/run-benchmarks.sh --type crypto

# Session benchmarks only
./scripts/run-benchmarks.sh --type session

# RFC 9421 benchmarks only
./scripts/run-benchmarks.sh --type rfc9421

# Comparison benchmarks only
./scripts/run-benchmarks.sh --type comparison
```

### Run with Custom Parameters

```bash
# Longer benchmark time for more accurate results
./scripts/run-benchmarks.sh --benchtime 30s --count 10

# Compare with previous results
./scripts/run-benchmarks.sh --compare tools/benchmark/results/benchmark_20250108_120000.json
```

### Manual Benchmark Execution

```bash
# Run all benchmarks
go test -bench=. -benchmem ./benchmark

# Run specific benchmark
go test -bench=BenchmarkKeyGeneration -benchmem ./benchmark

# Run with longer time
go test -bench=. -benchtime=30s -benchmem ./benchmark

# With CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof ./benchmark
go tool pprof cpu.prof

# With memory profiling
go test -bench=. -benchmem -memprofile=mem.prof ./benchmark
go tool pprof mem.prof
```

## Understanding Results

### Benchmark Output Format

```
BenchmarkKeyGeneration/Ed25519-8    5000    234567 ns/op    1024 B/op    12 allocs/op
```

- `BenchmarkKeyGeneration/Ed25519`: Benchmark name and sub-benchmark
- `-8`: Number of CPUs (GOMAXPROCS)
- `5000`: Number of iterations run
- `234567 ns/op`: Nanoseconds per operation (lower is better)
- `1024 B/op`: Bytes allocated per operation (lower is better)
- `12 allocs/op`: Number of allocations per operation (lower is better)

### Key Metrics

#### Throughput (MB/s)
- Measures data processing rate
- Higher is better
- Useful for encryption/decryption operations

#### Latency (ns/op)
- Time taken for single operation
- Lower is better
- Critical for real-time applications

#### Memory (B/op, allocs/op)
- Memory allocated per operation
- Lower is better
- Important for high-throughput scenarios

#### Percentiles (p50, p95, p99)
- p50: Median latency (50% of operations faster)
- p95: 95th percentile (95% of operations faster)
- p99: 99th percentile (99% of operations faster)
- Higher percentiles show worst-case performance

## Benchmark Categories Detail

### Cryptographic Operations

#### Key Generation
```bash
go test -bench=BenchmarkKeyGeneration -benchmem ./benchmark
```

**What it measures:**
- Ed25519 key pair generation
- Secp256k1 key pair generation
- X25519 key pair generation

**Typical Results:**
- Ed25519: ~50-100 μs per key pair
- Secp256k1: ~100-200 μs per key pair
- X25519: ~30-60 μs per key pair

#### Signing and Verification
```bash
go test -bench="BenchmarkSign|BenchmarkVerif" -benchmem ./benchmark
```

**What it measures:**
- Ed25519 signing and verification
- Secp256k1 signing and verification
- Performance with 1KB messages

**Typical Results:**
- Ed25519 signing: ~40-80 μs
- Ed25519 verification: ~100-200 μs
- Secp256k1 signing: ~60-120 μs
- Secp256k1 verification: ~150-300 μs

#### Message Size Scaling
```bash
go test -bench=BenchmarkMessageSizes -benchmem ./benchmark
```

**What it measures:**
- Signing performance: 64B, 256B, 1KB, 4KB, 16KB, 64KB
- Shows how performance scales with message size

### HPKE Operations

#### HPKE Sender/Receiver Derivation
```bash
go test -bench=BenchmarkHPKEDeriveSharedSecret -benchmem ./pkg/agent/hpke
go test -bench=BenchmarkHPKEOpenSharedSecret -benchmem ./pkg/agent/hpke
```

**What it measures:**
- HPKE Base sender-side key derivation
- HPKE Base receiver-side key derivation
- X25519 ECDH operations
- HKDF key derivation

**Typical Results:**
- Sender derivation: ~60-80 μs
- Receiver derivation: ~60-80 μs

#### HPKE Full Roundtrip
```bash
go test -bench=BenchmarkHPKEFullRoundtrip -benchmem ./pkg/agent/hpke
```

**What it measures:**
- Complete HPKE handshake (sender + receiver)
- Secret agreement verification
- End-to-end latency

**Typical Results:**
- Full roundtrip: ~120-160 μs

#### HPKE Export Lengths
```bash
go test -bench=BenchmarkHPKEExportLengths -benchmem ./pkg/agent/hpke
```

**What it measures:**
- Export secret derivation (16B, 32B, 64B, 128B, 256B)
- HKDF expansion performance

**Typical Results:**
- 32B export: ~60-80 μs

### Handshake Protocol

#### Key Generation
```bash
go test -bench=BenchmarkKeyGeneration -benchmem ./pkg/agent/handshake
```

**What it measures:**
- Ed25519 key pair generation for signing
- X25519 key pair generation for HPKE

**Typical Results:**
- Ed25519: ~25-30 μs per key pair
- X25519: ~40-50 μs per key pair

#### Signature Operations
```bash
go test -bench="BenchmarkSignature.*" -benchmem ./pkg/agent/handshake
```

**What it measures:**
- Ed25519 and Secp256k1 signing
- Ed25519 and Secp256k1 verification
- Performance with test messages

**Typical Results:**
- Ed25519 signing: ~40-50 μs
- Ed25519 verification: ~80-100 μs

#### Session Encryption/Decryption
```bash
go test -bench="BenchmarkSession.*" -benchmem ./pkg/agent/handshake
```

**What it measures:**
- AES-GCM session encryption
- AES-GCM session decryption
- Full roundtrip performance
- Scaling across message sizes (64B to 16KB)

**Typical Results:**
- Encryption (1KB): ~5-8 μs
- Decryption (1KB): ~5-8 μs
- Roundtrip (1KB): ~10-16 μs

### Session Management

#### Session Creation
```bash
go test -bench=BenchmarkSessionCreation -benchmem ./benchmark
```

**What it measures:**
- HPKE-based session establishment
- X25519 key agreement
- HKDF key derivation

**Typical Results:**
- Session creation: ~100-200 μs

#### Encryption/Decryption
```bash
go test -bench="BenchmarkSession.*cryption" -benchmem ./benchmark
```

**What it measures:**
- AES-GCM encryption/decryption
- Performance across message sizes (64B to 16KB)
- Throughput in MB/s

**Typical Results:**
- 1KB encryption: ~5-10 μs (~100-200 MB/s)
- 1KB decryption: ~5-10 μs (~100-200 MB/s)

#### Handshake Protocol
```bash
go test -bench=BenchmarkHandshakeProtocol -benchmem ./benchmark
```

**What it measures:**
- Complete handshake: initiation + response + finalization
- DID signature verification
- Ephemeral key exchange

**Typical Results:**
- Full handshake: ~500-1000 μs

#### Concurrent Operations
```bash
go test -bench=BenchmarkConcurrent -benchmem ./benchmark
```

**What it measures:**
- Parallel session creation
- Concurrent session access
- Lock contention
- Scalability with multiple goroutines

### RFC 9421 HTTP Signatures

#### HTTP Message Signing
```bash
go test -bench=BenchmarkHTTPSignature -benchmem ./benchmark
```

**What it measures:**
- HTTP request signing
- Signature verification
- Canonical form generation

**Typical Results:**
- HTTP signing: ~100-200 μs
- HTTP verification: ~150-300 μs

#### Signature Components
```bash
go test -bench=BenchmarkSignatureComponents -benchmem ./benchmark
```

**What it measures:**
- Performance with 3, 5, and 6 components
- Shows overhead of additional headers

#### HMAC Signatures
```bash
go test -bench=BenchmarkHMACSignature -benchmem ./benchmark
```

**What it measures:**
- HMAC-SHA256 signing/verification
- Faster than asymmetric signatures

**Typical Results:**
- HMAC signing: ~10-20 μs
- HMAC verification: ~10-20 μs

### Baseline Comparisons

#### SAGE vs. No Security
```bash
go test -bench=BenchmarkBaseline_vs_SAGE -benchmem ./benchmark
```

**What it measures:**
- Baseline (no security): Raw message passing
- Simple hash: SHA-256 integrity check
- SAGE full: Complete encryption/decryption

**Expected Overhead:**
- SAGE adds ~10-20 μs per message (1KB)
- ~100-200x slower than no security
- ~10-20x slower than simple hash
- **Trade-off**: Security vs. performance

#### Throughput Comparison
```bash
go test -bench=BenchmarkThroughput -benchmem ./benchmark
```

**What it measures:**
- Data processing rate (MB/s)
- Comparison across message sizes
- Baseline vs. SAGE throughput

**Typical Results:**
- Baseline: ~10,000 MB/s
- SAGE: ~100-200 MB/s
- Overhead: ~50-100x

#### Latency Percentiles
```bash
go test -bench=BenchmarkLatency -benchmem ./benchmark
```

**What it measures:**
- p50, p95, p99 latencies
- Baseline vs. SAGE round-trip time
- Full handshake latency

**Typical Results (1KB messages):**
- Baseline p50: ~0.01 ms
- SAGE p50: ~0.02 ms
- Handshake p50: ~1.0 ms

#### Memory Usage
```bash
go test -bench=BenchmarkMemoryUsage -benchmem ./benchmark
```

**What it measures:**
- Baseline: 1000 messages in memory
- SAGE: 1000 active sessions
- Memory overhead comparison

## Benchmark Results Analysis

### Running Analysis Tools

```bash
# Generate analysis report
go run ./tools/analyze/analyze.go \
  -input tools/benchmark/results/benchmark_20250108.json \
  -output tools/benchmark/results/analysis_20250108.md

# Compare with previous results
go run ./tools/analyze/analyze.go \
  -input tools/benchmark/results/current.json \
  -compare tools/benchmark/results/previous.json \
  -output tools/benchmark/results/comparison.md
```

### Interpreting Analysis

The analysis tool generates:

1. **Category Tables**: Organized by benchmark category
2. **Summary Statistics**: Overall performance metrics
3. **Extremes**: Fastest and slowest operations
4. **Comparison**: Performance changes vs. previous run

**Performance Change Indicators:**
- ✅ Green: Within 10% (acceptable)
- ⚠️ Yellow: 10-20% degradation (review needed)
- ❌ Red: >20% degradation (action required)
- ✨ New: New benchmark added

## Performance Targets

### Production Targets

| Operation | Target Latency | Target Throughput |
|-----------|---------------|-------------------|
| Key Generation | <200 μs | >5,000 ops/s |
| Session Creation | <500 μs | >2,000 ops/s |
| Message Encryption (1KB) | <20 μs | >50 MB/s |
| Message Decryption (1KB) | <20 μs | >50 MB/s |
| HTTP Signature Sign | <300 μs | >3,000 ops/s |
| HTTP Signature Verify | <400 μs | >2,500 ops/s |
| Full Handshake | <2 ms | >500 ops/s |

### Acceptable Overhead

| Scenario | Baseline | SAGE | Overhead |
|----------|----------|------|----------|
| Message Passing (1KB) | 0.01 ms | 0.02 ms | 2x |
| Throughput (1KB) | 10 GB/s | 100 MB/s | 100x |
| Session Setup | N/A | 1 ms | N/A |

## Continuous Benchmarking

### CI/CD Integration

Benchmarks run automatically on:
- Pull requests (comparison with main branch)
- Main branch commits (baseline updates)
- Weekly schedule (long-term trend analysis)

### GitHub Actions Workflow

```yaml
name: Benchmark

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: '0 0 * * 0'  # Weekly

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run benchmarks
        run: ./scripts/run-benchmarks.sh

      - name: Upload results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results
          path: tools/benchmark/results/
```

## Optimization Guidelines

### When Optimizing

1. **Profile First**: Use pprof to identify bottlenecks
2. **Measure**: Run benchmarks before and after changes
3. **Compare**: Use analysis tool for regression detection
4. **Document**: Note optimization rationale in commits

### Common Optimizations

1. **Reduce Allocations**
   - Reuse buffers
   - Use sync.Pool for temporary objects
   - Avoid unnecessary conversions

2. **Cache Expensive Operations**
   - Public key parsing
   - Cryptographic parameters
   - Session keys

3. **Parallelize When Possible**
   - Independent signature verifications
   - Batch operations
   - Use worker pools

### Performance Regression Prevention

1. **Set Thresholds**: Define acceptable degradation (e.g., 10%)
2. **Block PRs**: Fail CI on >20% regression
3. **Track Trends**: Monitor long-term performance changes
4. **Review Regularly**: Weekly performance review meetings

## Troubleshooting

### Inconsistent Results

```bash
# Run more iterations
go test -bench=. -benchtime=30s -count=10 ./benchmark

# Disable CPU frequency scaling
sudo cpupower frequency-set --governor performance

# Pin to specific CPUs
taskset -c 0-3 go test -bench=. ./benchmark
```

### High Variance

- Close background applications
- Run on dedicated benchmark machine
- Use longer benchmark times
- Increase iteration count

### Memory Issues

```bash
# Increase timeout
go test -bench=. -timeout=30m ./benchmark

# Reduce parallel jobs
GOMAXPROCS=1 go test -bench=. ./benchmark
```

## Additional Resources

- [Go Benchmark Documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
- [pprof Guide](https://go.dev/blog/pprof)
- [Performance Best Practices](https://go.dev/doc/effective_go#concurrency)
- [SAGE Documentation](../README.md)

## Contributing

When adding new benchmarks:

1. Follow existing naming conventions
2. Include baseline comparisons
3. Test across multiple message sizes
4. Document expected results
5. Update this README
