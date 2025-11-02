# ADR-002: HPKE Selection for End-to-End Encryption

**Status:** Accepted

**Date:** 2024-10-26

**Decision Makers:** SAGE Core Team

**Technical Story:** [HPKE Implementation](https://github.com/sage-x-project/sage/tree/main/pkg/agent/hpke)

---

## Context

SAGE (Secure Agent Guarantee Engine) requires robust end-to-end encryption for agent-to-agent communication. Messages exchanged between agents contain potentially sensitive information and must be protected from eavesdropping, tampering, and unauthorized access. We needed to select a cryptographic protocol that provides:

1. **Strong Security**: Modern cryptography with proven security guarantees
2. **Efficiency**: Low computational and bandwidth overhead
3. **Standardization**: Well-defined, peer-reviewed protocol
4. **Simplicity**: Easy to implement correctly without subtle vulnerabilities
5. **Future-Proof**: Resistance to emerging threats (including quantum computing)
6. **Interoperability**: Cross-language and cross-platform support

### Problem Statement

Several encryption protocols exist for secure communication, each with different trade-offs:

1. **Complexity vs. Security**: More complex protocols (TLS, Signal) offer many features but increase attack surface and implementation difficulty.

2. **Performance Requirements**: AI agents may exchange thousands of messages per second. Encryption overhead must be minimal.

3. **Key Management**: Traditional PKI with certificates adds operational complexity. We prefer public-key-based authentication tied to DID (Decentralized Identifiers).

4. **Quantum Resistance**: While current quantum computers don't threaten existing cryptography, we want a protocol that can evolve toward post-quantum algorithms.

5. **Message Patterns**: Agent communication isn't always request-response. We need support for:
   - Unidirectional messages (agent → agent)
   - Asynchronous communication (no online key agreement)
   - Session establishment without multiple round trips

6. **Standards Compliance**: Using non-standard crypto makes security audits difficult and reduces trust.

### Requirements

**Must Have:**
- Authenticated encryption (confidentiality + integrity)
- Forward secrecy (compromise of long-term keys doesn't decrypt past messages)
- Public-key based (no shared secrets)
- Resistance to replay attacks
- Standardized and peer-reviewed

**Should Have:**
- Single round-trip key agreement
- Post-quantum migration path
- Efficient for small messages (<1KB typical)
- Native support in Go ecosystem

**Nice to Have:**
- Formal security proofs
- FIPS 140-2 compliance path
- Hardware acceleration support

---

## Decision

We decided to adopt **HPKE (Hybrid Public Key Encryption)** as specified in **RFC 9180** for all end-to-end encryption in SAGE.

### What is HPKE?

HPKE is a modern, standardized public-key encryption scheme that combines:

1. **KEM (Key Encapsulation Mechanism)**: X25519 (Curve25519 ECDH)
2. **KDF (Key Derivation Function)**: HKDF-SHA256
3. **AEAD (Authenticated Encryption with Associated Data)**: ChaCha20-Poly1305

### HPKE in SAGE Architecture

```
Agent A                                Agent B
  │                                      │
  │  1. Generate ephemeral key pair      │
  │  2. Encapsulate to Agent B's DID     │
  │     public key                        │
  ├───────── encapsulated_key ──────────►│
  │          + ciphertext                 │
  │                                      │  3. Decapsulate with private key
  │                                      │  4. Derive shared secret
  │                                      │  5. Decrypt message
  │◄──────── encrypted response ─────────┤
  │                                      │
```

### HPKE Configuration

**Cipher Suite:** DHKEM(X25519, HKDF-SHA256), HKDF-SHA256, ChaCha20Poly1305

- **KEM**: DHKEM with X25519 (Curve25519 ECDH)
- **KDF**: HKDF with SHA-256
- **AEAD**: ChaCha20-Poly1305 (authenticated encryption)

**Modes:**
- **Base Mode** (mode 0x00): Single-shot encryption, no sender authentication
  - Use case: Initial handshake messages
- **Auth Mode** (mode 0x02): Sender authenticates with their identity key
  - Use case: Authenticated agent-to-agent messages

### Implementation

```go
// Sender (Agent A)
import "github.com/sage-x-project/sage/pkg/agent/hpke"

client := hpke.NewClient(recipientPublicKey)
encapsulated, ciphertext, err := client.Seal(plaintext, nil)

// Receiver (Agent B)
server := hpke.NewServer(privateKey)
plaintext, err := server.Open(encapsulated, ciphertext, nil)
```

### Integration with SAGE Components

1. **Handshake Phase**: Agent A sends initial message to Agent B
   - HPKE Base Mode for first message
   - Establishes session keys

2. **Session Phase**: Subsequent messages use session keys
   - Derived from HPKE shared secret
   - ChaCha20-Poly1305 for message encryption

3. **DID Integration**: Public keys from DID documents
   - No separate certificate infrastructure
   - DID resolution provides recipient public keys

---

## Consequences

### Positive

1. **Standards-Based**
   - IETF RFC 9180 (published 2022)
   - Peer-reviewed by cryptography community
   - Active maintenance and security analysis
   - Reduces risk of implementation flaws

2. **Security Properties**
   - **IND-CCA2 secure** (indistinguishability under adaptive chosen-ciphertext attack)
   - **Forward secrecy**: Ephemeral keys for each session
   - **Authenticated encryption**: ChaCha20-Poly1305 provides confidentiality + integrity
   - **Replay protection**: Nonces prevent message replay

3. **Performance**
   - **Fast**: ChaCha20 is ~3x faster than AES on CPUs without AES-NI
   - **Small overhead**: ~48 bytes (X25519 public key + Poly1305 MAC)
   - **Single RTT**: One round-trip for key agreement
   - **No online handshake**: Async encryption with only recipient's public key

4. **Simplicity**
   - **Clean API**: Simple `Seal()` and `Open()` operations
   - **No complex state machine**: Easier to implement correctly
   - **Minimal configuration**: Few knobs to tune

5. **Future-Proof**
   - **Post-quantum ready**: HPKE framework supports post-quantum KEMs
   - **Hybrid mode**: Can combine classical + PQ algorithms
   - **Algorithm agility**: Easy to migrate to new ciphers

6. **Library Support**
   - **Go**: [`cloudflare/circl`](https://github.com/cloudflare/circl) (production-ready)
   - **Rust**: [`hpke-rs`](https://github.com/rozbb/hpke-rs)
   - **JavaScript**: [`hpke-js`](https://github.com/dajiaji/hpke-js)
   - **Python**: [`pyhpke`](https://github.com/dajiaji/pyhpke)

### Negative

1. **Relatively New Standard**
   - RFC published in 2022 (vs. TLS from 1999)
   - Less deployment experience than TLS
   - Fewer security audits
   - **Mitigation**: Use well-tested libraries (Cloudflare CIRCL)

2. **Limited FIPS 140-2 Validation**
   - ChaCha20-Poly1305 not in original FIPS 140-2
   - X25519 added to FIPS in recent update
   - **Mitigation**: FIPS mode can use AES-GCM instead

3. **No Built-in Session Management**
   - HPKE is single-message encryption
   - Must build session layer on top
   - **Mitigation**: SAGE implements session management separately

4. **Stateless Sender**
   - Sender doesn't maintain session state
   - Each message needs recipient's public key
   - **Mitigation**: Cache public keys from DID resolution

### Trade-offs Accepted

- **Newness vs. Modern Cryptography**: We accept that HPKE is newer, betting that its formal analysis and standardization make it safer than ad-hoc protocols.
- **Library Maturity vs. Future-Proof**: We accept some ecosystem immaturity in exchange for a protocol designed for the next decade.
- **Simplicity vs. Features**: We accept lack of built-in sessions/ratcheting, implementing these concerns separately for cleaner architecture.

---

## Alternatives Considered

### Alternative 1: TLS 1.3 + mTLS

**Approach:** Use TLS 1.3 for encryption and mutual authentication.

**Pros:**
- Extremely well-tested (billions of connections daily)
- Hardware acceleration widely available
- Strong ecosystem and tooling
- FIPS 140-2 validated implementations

**Cons:**
- **Certificate Management**: Requires PKI or self-signed certs
  - Doesn't integrate naturally with DID
  - Operational complexity (issuance, renewal, revocation)
- **Heavyweight**: Full TLS stack is complex
- **Connection-Oriented**: Requires persistent connections
- **Overhead**: TLS handshake adds latency
- **Overkill**: Most TLS features unnecessary for agent messaging

**Why Rejected:** TLS is designed for client-server web traffic, not peer-to-peer agent communication. Certificate management doesn't align with DID-based identity. Too complex for our needs.

---

### Alternative 2: Signal Protocol (Double Ratchet)

**Approach:** Use Signal's Double Ratchet algorithm for end-to-end encryption.

**Pros:**
- Battle-tested (billions of users via WhatsApp, Signal)
- Excellent forward secrecy (ratchets keys per message)
- Proven security against sophisticated adversaries
- Well-documented

**Cons:**
- **Mobile-Focused**: Designed for human messaging patterns
- **State Management**: Requires storing ratchet state for each conversation
- **Complexity**: Double ratchet + message ordering + lost messages
- **Synchronous Assumption**: Works best with online peers
- **Not Standardized**: No IETF RFC, protocol is WhatsApp/Signal specific

**Why Rejected:** Signal Protocol is optimized for mobile messaging apps (1:1 chats, small groups), not agent-to-agent bulk communication. State management complexity outweighs benefits for our use case.

---

### Alternative 3: Noise Protocol Framework

**Approach:** Use Noise Protocol for key agreement and encryption.

**Pros:**
- Flexible framework (many handshake patterns)
- Used successfully (WireGuard, WhatsApp, Lightning Network)
- Modern crypto (X25519, ChaCha20-Poly1305)
- Formal verification

**Cons:**
- **Not IETF Standard**: Community specification, not RFC
- **DIY Protocol**: Must choose handshake pattern, design session layer
- **Framework, Not Protocol**: Requires more design decisions
- **Less Documentation**: Compared to IETF RFCs

**Why Rejected:** Noise is a framework for building protocols, not a ready-to-use protocol. HPKE provides similar cryptography but with IETF standardization. We prefer not to design our own Noise-based protocol when HPKE exists.

---

### Alternative 4: NaCl / libsodium (Sealed Boxes)

**Approach:** Use libsodium's `crypto_box_seal()` for public-key encryption.

**Pros:**
- Simple API (`crypto_box_seal` is one function call)
- Battle-tested library (NaCl from Dan Bernstein)
- Fast (X25519 + XSalsa20 + Poly1305)
- Widely used

**Cons:**
- **Not Standardized**: No IETF RFC
- **Sealed Boxes Limitations**:
  - No sender authentication
  - No associated data support
  - No formal specification
- **Legacy Crypto**: XSalsa20 (24-byte nonce) vs. ChaCha20 (12-byte nonce)

**Why Rejected:** While libsodium is excellent, HPKE provides the same security with IETF standardization. We prefer RFC 9180 for long-term maintainability and interoperability.

---

### Alternative 5: Age Encryption Format

**Approach:** Use age (actually good encryption) for message encryption.

**Pros:**
- Simple, modern design
- X25519 + ChaCha20-Poly1305
- Good UX for file encryption
- Actively maintained

**Cons:**
- **File-Focused**: Designed for file encryption, not messaging
- **Not IETF Standard**: Personal project by Filippo Valsorda
- **Limited Scope**: Doesn't address session management, forward secrecy
- **Format Overhead**: Age format includes headers for file metadata

**Why Rejected:** Age is excellent for its intended use case (file encryption), but HPKE is better suited for message-level encryption in a communication protocol.

---

### Alternative 6: Custom Hybrid Encryption

**Approach:** Build our own hybrid encryption (ECDH + symmetric encryption).

**Pros:**
- Full control over protocol
- Can optimize for specific use case
- No external dependencies

**Cons:**
- **Security Risk**: Easy to make subtle mistakes
- **No Peer Review**: No cryptography community validation
- **Maintenance Burden**: Security updates, algorithm transitions
- **Trust Issues**: Users must trust our custom crypto

**Why Rejected:** "Don't roll your own crypto" is a fundamental security principle. HPKE provides exactly what we need with expert design and review.

---

## Migration Path

### Current Implementation

- **HPKE Library**: [`cloudflare/circl`](https://github.com/cloudflare/circl/tree/main/hpke)
- **Cipher Suite**: DHKEM(X25519, HKDF-SHA256), HKDF-SHA256, ChaCha20Poly1305
- **Modes**: Base (mode 0x00) and Auth (mode 0x02)

### Future Post-Quantum Migration

HPKE RFC 9180 explicitly supports post-quantum KEMs:

```go
// Current (2024)
DHKEM(X25519, HKDF-SHA256)

// Future (post-quantum era)
DHKEM(Kyber768, HKDF-SHA256)  // NIST PQC winner

// Hybrid (transitional)
DHKEM(X25519+Kyber768, HKDF-SHA256)  // Combined classical + PQ
```

**Migration Strategy:**
1. **Monitor NIST PQC standardization** (Kyber selected 2022, standardization 2024)
2. **Test hybrid mode** with X25519 + Kyber in development
3. **Deploy hybrid mode** when libraries mature
4. **Transition to PQ-only** when ecosystem ready

---

## Performance Characteristics

### Benchmarks (M1 Mac, Go 1.22)

```
Operation                    Latency    Throughput
---------------------------------------------------
X25519 Key Generation        15 μs      66,666 ops/sec
HPKE Seal (1KB message)      45 μs      22,222 msg/sec
HPKE Open (1KB message)      42 μs      23,809 msg/sec
ChaCha20-Poly1305 (1KB)      8 μs       125,000 msg/sec
```

**Comparison to AES-GCM:**
- ChaCha20-Poly1305: ~3x faster on CPUs without AES-NI
- ChaCha20-Poly1305: ~1.2x faster on ARM CPUs (M1, mobile)
- AES-GCM: Faster on Intel with AES-NI (~2x)

**Overhead Analysis:**
- Encapsulated Key: 32 bytes (X25519 public key)
- Authentication Tag: 16 bytes (Poly1305)
- **Total Overhead: 48 bytes per message** (~5% for 1KB messages)

---

## Security Considerations

### Threat Model

**Protected Against:**
-  Eavesdropping (confidentiality)
-  Message tampering (integrity)
-  Impersonation (authentication)
-  Replay attacks (nonces + timestamps)
-  Forward secrecy (ephemeral keys)

**Not Protected Against:**
-  Traffic analysis (message sizes visible)
-  Denial of service (must handle at application layer)
-  Key compromise (if attacker steals private keys)
-  Side-channel attacks (timing, cache, power analysis)

### Key Management

**Key Types:**
1. **Identity Keys** (Ed25519): Signing, long-lived, in DID document
2. **Encryption Keys** (X25519): HPKE, derived from identity or separate
3. **Session Keys** (ephemeral): Derived from HPKE shared secret

**Key Rotation:**
- Identity keys: Rotated via DID document updates
- Encryption keys: Can be rotated independently
- Session keys: New for each session, automatically rotated

---

## Related Documents

- [RFC 9180: Hybrid Public Key Encryption](https://www.rfc-editor.org/rfc/rfc9180.html)
- [HPKE Implementation](../../pkg/agent/hpke/)
- [Cryptography Documentation](../crypto/)
- [Session Management](../../pkg/agent/session/)

---

## References

- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [Cloudflare CIRCL Library](https://github.com/cloudflare/circl)
- [HPKE Security Analysis](https://eprint.iacr.org/2020/243.pdf)
- [Post-Quantum Cryptography](https://csrc.nist.gov/projects/post-quantum-cryptography)

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2024-10-26 | 1.0 | SAGE Team | Initial ADR |

---

## Approval

This ADR has been reviewed and accepted by the SAGE core team. HPKE implementation is complete and in production use.

**Acceptance Criteria Met:**
-  HPKE (RFC 9180) implemented with Cloudflare CIRCL
-  X25519 + ChaCha20-Poly1305 cipher suite
-  Both Base and Auth modes supported
-  Comprehensive test suite with RFC test vectors
-  Benchmarks demonstrate acceptable performance
-  Integration with DID-based key distribution
