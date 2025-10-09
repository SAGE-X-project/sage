# SAGE Security Considerations

**Version**: 1.0.0
**Date**: October 2025
**Purpose**: Security Audit Reference

## Table of Contents

1. [Critical Security Features](#1-critical-security-features)
2. [Attack Vectors & Mitigations](#2-attack-vectors--mitigations)
3. [Known Limitations](#3-known-limitations)
4. [Security Assumptions](#4-security-assumptions)
5. [Incident Response](#5-incident-response)

---

## 1. Critical Security Features

### 1.1 Public Key Validation (5-Step Process)

**Location**: `SageRegistryV2.sol::_validatePublicKey()`

**Steps**:
```solidity
1. Length Validation
   - Ed25519: exactly 32 bytes
   - Secp256k1: exactly 33 bytes (compressed)
   - Reject if wrong length

2. Format Validation
   - Secp256k1: First byte must be 0x02 or 0x03
   - Ed25519: No format requirements (all 32 bytes valid)

3. Zero-Key Check
   - Reject if all bytes are zero
   - Prevents placeholder/uninitialized keys

4. Ownership Proof
   - Challenge-response mechanism
   - Agent must sign challenge with private key
   - Signature verified on-chain

5. Revocation Status
   - Check if key has been previously revoked
   - Revoked keys cannot be re-registered
```

**Why Critical**:
- Prevents registration of invalid/malicious keys
- Ensures only key owner can register
- Prevents key reuse after revocation

**Potential Issues**:
- [ ] Can ownership proof be bypassed?
- [ ] Is challenge randomness sufficient?
- [ ] Can timing attacks reveal information?

### 1.2 Session Key Derivation

**Location**: `session/manager.go::CreateSession()`

**Process**:
```go
// 1. HPKE Key Agreement
sharedSecret := X25519(myEphemeralPrivate, peerEphemeralPublic)

// 2. Session ID Derivation
sessionID := HKDF(
    hash: SHA256,
    ikm: sharedSecret,
    salt: "SAGE-Session-v1",
    info: concat(myDID, peerDID),
    length: 32
)

// 3. Directional Keys
keyClientToServer := HKDF(
    hash: SHA256,
    ikm: sessionID,
    salt: "client-to-server",
    info: concat(clientDID, serverDID),
    length: 32
)

keyServerToClient := HKDF(
    hash: SHA256,
    ikm: sessionID,
    salt: "server-to-client",
    info: concat(serverDID, clientDID),
    length: 32
)
```

**Why Critical**:
- Session ID must be deterministic (both peers derive same ID)
- Keys must be directional (prevents reflection attacks)
- HKDF ensures cryptographic separation

**Potential Issues**:
- [ ] Is shared secret properly validated?
- [ ] Can session ID collide?
- [ ] Are ephemeral keys properly deleted?

### 1.3 Replay Attack Prevention

**Location**: `session/nonce.go::IsNonceUsed()`

**Mechanism**:
```go
type NonceCache struct {
    ttl  time.Duration
    data sync.Map        // keyid -> *sync.Map (nonce -> expiryUnix)
    tick *time.Ticker    // Background GC ticker
    stop chan struct{}   // Stop signal
}

// Seen returns true if (keyid, nonce) was seen before (replay attack)
func (n *NonceCache) Seen(keyid, nonce string) bool {
    if keyid == "" || nonce == "" {
        return false
    }
    exp := time.Now().Add(n.ttl).Unix()

    v, _ := n.data.LoadOrStore(keyid, &sync.Map{})
    m := v.(*sync.Map)

    if old, ok := m.Load(nonce); ok {
        if prevExp, _ := old.(int64); prevExp >= time.Now().Unix() {
            return true // Replay detected!
        }
    }
    m.Store(nonce, exp)
    return false
}

// Background GC removes expired nonces
func (n *NonceCache) gcLoop() {
    for {
        select {
        case <-n.tick.C:
            now := time.Now().Unix()
            n.data.Range(func(k, v any) bool {
                m := v.(*sync.Map)
                m.Range(func(nk, nv any) bool {
                    if exp, _ := nv.(int64); exp < now {
                        m.Delete(nk)
                    }
                    return true
                })
                return true
            })
        case <-n.stop:
            return
        }
    }
}
```

**Why Critical**:
- Prevents replay attacks
- Limits memory usage (TTL-based cleanup)
- Thread-safe for concurrent access

**Potential Issues**:
- [ ] Is nonce generation cryptographically secure?
- [ ] Can nonce cache overflow (DoS)?
- [ ] Is cleanup interval appropriate? (Currently: 1 minute)
- [ ] Lock-free sync.Map: Are there subtle race conditions?
- [ ] Per-keyid nonce maps: Memory overhead acceptable?

### 1.4 Message Signature Verification (RFC 9421)

**Location**: `core/rfc9421/verifier.go::VerifyHTTPSignature()`

**Process**:
```go
// 1. Canonicalize Message
canonical := canonicalizeMessage(
    method:  msg.Method,
    path:    msg.Path,
    headers: msg.Headers,
    body:    msg.Body,
    fields:  signedFields
)
// Output: "@method: POST\n@path: /api/chat\ncontent-digest: sha-256=...\n"

// 2. Lookup Session by Key ID
session := sessionManager.GetByKeyID(keyID)
if session == nil {
    return ErrSessionNotFound
}

// 3. Compute Expected Signature
expected := HMAC_SHA256(
    key: session.SigningKey,
    data: canonical
)

// 4. Constant-Time Comparison
if !subtle.ConstantTimeCompare(expected, provided) {
    return ErrSignatureInvalid
}

// 5. Timestamp Validation
if abs(msg.Timestamp - now()) > MAX_CLOCK_SKEW {
    return ErrTimestampOutOfRange
}
```

**Why Critical**:
- Ensures message integrity
- Prevents modification in transit
- Validates message freshness

**Potential Issues**:
- [ ] Is canonicalization correct per RFC 9421?
- [ ] Can canonicalization be bypassed?
- [ ] Is clock skew tolerance appropriate?
- [ ] Timing attacks in comparison?

---

## 2. Attack Vectors & Mitigations

### 2.1 Identity-Based Attacks

#### Attack 2.1.1: Agent Impersonation

**Scenario**: Attacker tries to impersonate legitimate agent

**Method**:
```
1. Attacker registers fake agent with similar name
   - "ChatGPT-Official" vs "ChatGPT_Official"
   - Homoglyph attacks (Cyrillic 'а' vs Latin 'a')

2. Attacker uses social engineering
   - Tricks users into trusting fake agent
   - Sends malicious messages as fake agent
```

**Mitigations**:
```
✓ DID is unique per agent (Ethereum address-based)
✓ Public key must be proven via ownership proof
✓ Agent names stored on-chain (immutable)
✓ Users should verify DID, not just name

Future:
- Name similarity detection
- Verified badge system
- ENS integration for human-readable names
```

**Severity**: HIGH
**Likelihood**: MEDIUM
**Risk**: HIGH

#### Attack 2.1.2: DID Spoofing

**Scenario**: Attacker claims fake DID in messages

**Method**:
```
1. Attacker sends message claiming:
   From: did:sage:ethereum:0xLEGIT...

2. If signature not verified, message appears legitimate
```

**Mitigations**:
```
✓ Every message MUST be signed with DID's private key
✓ Signature verified against on-chain public key
✓ Cannot forge signature without private key

Edge Cases:
- Compromised private key → key revocation needed
- Stolen session → session expiration limits damage
```

**Severity**: CRITICAL
**Likelihood**: LOW (requires private key)
**Risk**: MEDIUM

### 2.2 Cryptographic Attacks

#### Attack 2.2.1: Weak Key Generation

**Scenario**: Agent generates weak/predictable key

**Method**:
```
1. Attacker uses weak random source
   - time.Now() as seed
   - Sequential counter

2. Attacker can predict/brute-force key
```

**Mitigations**:
```
✓ Use crypto/rand for key generation
✓ 32 bytes minimum entropy
✓ Ed25519 keys from secure random source

Code Review Needed:
func GenerateEd25519() (*KeyPair, error) {
    // VERIFY: Uses crypto/rand, not math/rand
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    ...
}
```

**Severity**: CRITICAL
**Likelihood**: LOW (if crypto/rand used)
**Risk**: MEDIUM

#### Attack 2.2.2: Side-Channel Attacks

**Scenario**: Timing attacks reveal private key bits

**Method**:
```
1. Attacker measures signature generation time
2. Different execution paths leak key bits
3. After many samples, private key recovered
```

**Mitigations**:
```
✓ Use constant-time comparison (subtle.ConstantTimeCompare)
✓ Ed25519 library uses constant-time operations
? Need to verify: All signature operations constant-time

Audit Focus:
- Signature generation timing
- Verification timing
- Key derivation timing
```

**Severity**: HIGH
**Likelihood**: LOW (remote timing attacks difficult)
**Risk**: MEDIUM

### 2.3 Session-Based Attacks

#### Attack 2.3.1: Session Hijacking

**Scenario**: Attacker steals Key ID and uses it

**Method**:
```
1. Attacker intercepts message containing Key ID
   Signature-Input: sig1=("@method");keyid="abc123..."

2. Attacker tries to send message with same Key ID
```

**Mitigations**:
```
✓ Key ID is opaque (random 16 bytes)
✓ Session key never transmitted
✓ Cannot derive session key from Key ID
✓ Messages signed with HMAC (needs session key)
✓ Sessions expire (MaxAge, IdleTimeout)

Attack fails because:
- Attacker doesn't have session key
- Cannot forge valid HMAC without key
- Even if Key ID stolen, signature will fail
```

**Severity**: MEDIUM
**Likelihood**: LOW
**Risk**: LOW

#### Attack 2.3.2: Session Fixation

**Scenario**: Attacker forces victim to use known session

**Method**:
```
1. Attacker creates session with victim
2. Attacker sends Session ID to victim
3. Victim unknowingly uses attacker's session
```

**Mitigations**:
```
✓ Session ID derived from BOTH ephemeral keys
✓ Ephemeral keys generated independently
✓ Cannot force specific Session ID
✓ Handshake protocol prevents fixation

Why it fails:
SessionID = HKDF(X25519(myEphemeral, peerEphemeral))
- Attacker can't control victim's ephemeral key
- Attacker can't predict resulting Session ID
```

**Severity**: MEDIUM
**Likelihood**: VERY LOW
**Risk**: LOW

### 2.4 Smart Contract Attacks

#### Attack 2.4.1: Reentrancy

**Scenario**: Attacker exploits external calls to re-enter contract

**Method**:
```solidity
contract Attacker {
    SageRegistry registry;

    fallback() external {
        // Re-enter during registration
        registry.registerAgent(...);
    }
}
```

**Mitigations**:
```
✓ No external calls before state changes (CEI pattern)
✓ ReentrancyGuard on sensitive functions
✓ Checks-Effects-Interactions pattern

Audit Checklist:
- [ ] All state changes before external calls?
- [ ] External calls properly guarded?
- [ ] View functions don't modify state?
```

**Severity**: CRITICAL
**Likelihood**: LOW (no external calls in critical paths)
**Risk**: LOW

#### Attack 2.4.2: Integer Overflow/Underflow

**Scenario**: Arithmetic operations wrap around

**Method**:
```solidity
uint256 count = 2**256 - 1;
count++; // Wraps to 0
```

**Mitigations**:
```
✓ Solidity 0.8+ has built-in overflow checks
✓ All arithmetic operations revert on overflow
✓ No unchecked blocks in critical logic

Audit Checklist:
- [ ] Any unchecked{} blocks?
- [ ] Are they necessary?
- [ ] Safe arithmetic verified?
```

**Severity**: MEDIUM
**Likelihood**: VERY LOW (Solidity 0.8+)
**Risk**: LOW

#### Attack 2.4.3: Front-Running

**Scenario**: Attacker sees pending transaction and submits own with higher gas

**Method**:
```
1. User submits registerAgent() transaction
2. Attacker sees transaction in mempool
3. Attacker submits same registration with higher gas
4. Attacker's transaction mined first
5. User's transaction reverts (DID already registered)
```

**Mitigations**:
```
Current:
- First-come-first-served (unavoidable)
- User can retry with different parameters

Future (Post-Audit):
- Commit-reveal scheme
- Batch processing with fairness guarantees
- Flashbots/MEV protection
```

**Severity**: MEDIUM
**Likelihood**: MEDIUM (competitive registration)
**Risk**: MEDIUM

### 2.5 Denial of Service Attacks

#### Attack 2.5.1: Nonce Cache Overflow

**Scenario**: Attacker floods system with unique nonces to exhaust memory

**Method**:
```
for i := 0; i < 1000000; i++ {
    nonce := randomNonce()
    sendMessage(nonce)
}
```

**Mitigations**:
```
✓ TTL-based nonce expiration (e.g., 5 minutes)
✓ Periodic cleanup (e.g., every 30 seconds)
✓ Maximum nonce cache size limit

Calculations:
- 16 bytes per nonce + 8 bytes timestamp = 24 bytes
- 1M nonces = 24 MB memory
- TTL 5 minutes = max 5M nonces @ 1K req/sec
- Total: ~120 MB (acceptable)

Audit Focus:
- [ ] Is cleanup interval appropriate?
- [ ] Is TTL reasonable?
- [ ] Memory limits enforced?
```

**Severity**: MEDIUM
**Likelihood**: MEDIUM
**Risk**: MEDIUM

#### Attack 2.5.2: Session Exhaustion

**Scenario**: Attacker creates many sessions to exhaust resources

**Method**:
```
while true {
    createSession()  // Never use session
}
```

**Mitigations**:
```
✓ Session expiration (MaxAge, IdleTimeout)
✓ Automatic cleanup of expired sessions
✓ Rate limiting (future)

Current Limits:
- MaxAge: 1 hour (default)
- IdleTimeout: 10 minutes (default)
- Cleanup interval: 30 seconds

Attack Analysis:
- Attacker can create ~120 sessions/second (handshake time)
- Sessions expire in 10 minutes (idle)
- Max concurrent: 120 * 600 = 72K sessions
- Memory: 72K * 1KB = 72 MB (acceptable)

Future Improvements:
- Per-IP rate limiting
- CAPTCHA for registration
- Connection limits
```

**Severity**: MEDIUM
**Likelihood**: MEDIUM
**Risk**: MEDIUM

---

## 3. Known Limitations

### 3.1 Cross-Platform Library Builds

**Issue**: Building libraries for other platforms requires cross-compilers

**Impact**:
- Cannot build Linux libraries on macOS without tools
- Cannot build Windows DLLs without MinGW
- Developers need platform-specific toolchains

**Workaround**:
- Build natively on each platform
- Use Docker for cross-compilation
- CI/CD builds on multiple platforms

**Security Impact**: NONE (build-time only)

### 3.2 Smart Contract Upgrade Risk

**Issue**: UUPS proxy allows contract upgrades by admin

**Impact**:
- Admin could upgrade to malicious implementation
- User funds/data at risk if admin compromised

**Mitigations**:
- Multi-sig wallet for admin (planned)
- Timelock on upgrades (48-hour delay, planned)
- Transparent upgrade process
- Community notification before upgrades

**Security Impact**: MEDIUM (centralization risk)

### 3.3 Clock Synchronization

**Issue**: Timestamp validation assumes synchronized clocks

**Impact**:
- Agents with wrong clock may be rejected
- Clock skew tolerance: ±5 minutes

**Mitigations**:
- Document NTP requirement
- Graceful error messages for clock issues
- Monitoring for timestamp rejections

**Security Impact**: LOW (operational)

### 3.4 Nonce Cache Memory

**Issue**: Nonce cache grows with traffic

**Impact**:
- High traffic → large nonce cache
- Potential memory exhaustion

**Mitigations**:
- TTL-based expiration (5 minutes)
- Periodic cleanup (30 seconds)
- Maximum size limit (future)

**Security Impact**: MEDIUM (DoS risk)

---

## 4. Security Assumptions

### 4.1 Cryptographic Assumptions

```
1. Ed25519 is secure
   - No known attacks on signature scheme
   - Private key cannot be derived from public key

2. Secp256k1 is secure
   - Used by Bitcoin and Ethereum
   - ECDSA signature scheme is sound

3. X25519 is secure
   - ECDH key agreement is secure
   - Shared secret cannot be derived without private key

4. RSA-PSS-SHA256 (RS256) is secure
   - 2048-bit key provides adequate security
   - PSS padding prevents attacks

5. SHA-256 is collision-resistant
   - No known practical collisions
   - HMAC-SHA256 is unforgeable

6. ChaCha20-Poly1305 is secure
   - AEAD provides confidentiality and authenticity
   - No known attacks on algorithm

7. AES-256-GCM is secure
   - Used for vault encryption
   - Provides authenticated encryption

8. PBKDF2 with 100K iterations is sufficient
   - SHA-256 hash function
   - Resists brute-force attacks on passphrases
```

### 4.2 Network Assumptions

```
1. TLS protects transport
   - Man-in-the-middle prevented by TLS
   - Certificate validation assumed correct

2. Blockchain is Byzantine-fault-tolerant
   - Ethereum consensus is trusted
   - Transactions are immutable

3. RPC endpoints are trusted
   - Infura/Alchemy assumed honest
   - Can verify via multiple providers
```

### 4.3 Operational Assumptions

```
1. Private keys kept secure
   - Agents responsible for key security
   - No backup/recovery mechanism (yet)

2. Agents act rationally
   - Registered agents assumed non-malicious
   - Malicious behavior handled via revocation

3. Admin is trustworthy
   - Contract admin has significant power
   - Multi-sig and timelock to mitigate (planned)
```

### 4.4 Implementation Assumptions

```
1. Go standard library is secure
   - crypto/rand provides secure randomness
   - crypto/* packages are audited

2. Ethereum tooling is correct
   - go-ethereum client is trusted
   - Solidity compiler is correct

3. Dependencies are secure
   - Regular dependency updates
   - Known vulnerabilities patched
```

---

## 5. Incident Response

### 5.1 Security Incident Classification

#### Severity Levels

**P0 - Critical**
```
Examples:
- Private key compromise
- Smart contract exploit
- Mass agent impersonation

Response Time: Immediate (< 1 hour)
Actions:
- Pause smart contracts
- Revoke compromised keys
- Notify all users
- Emergency patch
```

**P1 - High**
```
Examples:
- Session hijacking
- Replay attack detected
- DoS attack

Response Time: < 4 hours
Actions:
- Isolate affected components
- Implement mitigations
- Monitor for spread
- Notify affected users
```

**P2 - Medium**
```
Examples:
- Individual key compromise
- Minor vulnerability
- Performance degradation

Response Time: < 24 hours
Actions:
- Investigate root cause
- Apply fixes
- Update documentation
```

**P3 - Low**
```
Examples:
- Configuration issues
- Non-security bugs
- Feature requests

Response Time: < 1 week
Actions:
- Standard development process
- Scheduled fixes
```

### 5.2 Emergency Procedures

#### Procedure 1: Key Compromise

```
1. User reports compromised key

2. Verification
   - Confirm identity of reporter
   - Verify compromise evidence

3. Immediate Actions
   - Call revokeKey() on-chain
   - Invalidate all sessions for that DID
   - Add to blocklist (if needed)

4. Post-Incident
   - Investigate how compromise occurred
   - User generates new key pair
   - Re-register with new key
   - Monitor for abuse of old key
```

#### Procedure 2: Smart Contract Vulnerability

```
1. Vulnerability discovered

2. Assessment
   - Severity classification
   - Exploit difficulty
   - Potential impact

3. Immediate Actions (if critical)
   - Pause contract (if possible)
   - Notify users immediately
   - Contact auditors

4. Remediation
   - Develop fix
   - Test thoroughly
   - Deploy upgrade
   - Unpause

5. Post-Mortem
   - Document incident
   - Update security measures
   - Improve testing
```

#### Procedure 3: DDoS Attack

```
1. Attack detected
   - Abnormal traffic patterns
   - Service degradation

2. Immediate Actions
   - Enable rate limiting
   - Block malicious IPs
   - Scale infrastructure

3. Mitigation
   - CloudFlare DDoS protection
   - IP whitelisting (if needed)
   - Session limits

4. Recovery
   - Monitor until normal
   - Analyze attack vectors
   - Implement permanent fixes
```

### 5.3 Contact Information

**Security Team**: security@sage-x-project.org
**Emergency Hotline**: +1-XXX-XXX-XXXX (24/7)
**PGP Key**: [Public Key Fingerprint]
**Bug Bounty**: https://sage-x-project.org/bug-bounty

---

## 6. Security Checklist (For Auditors)

### Smart Contracts
- [ ] Access control properly implemented
- [ ] No reentrancy vulnerabilities
- [ ] Integer arithmetic safe (Solidity 0.8+)
- [ ] Events emitted correctly
- [ ] Upgrade mechanism secure
- [ ] Gas optimization doesn't compromise security
- [ ] Front-running mitigated where possible

### Go Backend
- [ ] Cryptographic randomness (crypto/rand used)
- [ ] Keys properly generated and stored
- [ ] Session management correct
- [ ] Nonce cache prevents replay
- [ ] Signature verification correct
- [ ] Memory safely managed (no leaks)
- [ ] Concurrent access safe (mutexes used)
- [ ] Error handling doesn't leak information

### Integration
- [ ] DID resolution correct
- [ ] Handshake protocol sound
- [ ] Message flow end-to-end secure
- [ ] Session establishment verifiable
- [ ] Key rotation works correctly

### Operational
- [ ] Monitoring in place
- [ ] Logging sufficient (not excessive)
- [ ] Incident response procedures defined
- [ ] Backup and recovery tested
- [ ] Upgrade process documented

---

**Document Version**: 1.0
**Last Updated**: October 2025
**Status**: Ready for Audit
