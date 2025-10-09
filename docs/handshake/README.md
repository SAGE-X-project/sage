# SAGE Handshake Documentation

This folder contains documentation for SAGE's secure handshake protocols for establishing authenticated, encrypted sessions between AI agents.

## Quick Navigation

### For New Users

1. **Start here**: [Cryptographic Overview](./cryptographic-en.md) - Understand SAGE's security foundations
2. **Choose your approach**:
   - **Traditional**: [4-Phase Handshake Guide](./handshake-en.md) - Mature, battle-tested
   - **Modern**: [HPKE-Based Handshake Guide](./hpke-based-handshake-en.md) - 1-RTT, recommended for new projects

### For Developers

- **Implementation Guide**: [HPKE Detailed Tutorial](./hpke-detailed-en.md) - Step-by-step with code examples
- **API Reference**: See code documentation in `/handshake` and `/hpke` packages

---

## Two Handshake Protocols

SAGE supports two handshake protocols. Choose based on your requirements:

| Feature | Traditional (4-Phase) | HPKE-Based (2-Phase) |
|---------|----------------------|----------------------|
| **Packages** | `handshake/` | `hpke/` |
| **Round Trips** | 2 RTT (4 messages) | 1 RTT (2 messages) |
| **Key Exchange** | X25519 ECDH | HPKE Base + E2E X25519 |
| **Forward Secrecy** |  (ephemeral keys) |  (HPKE + E2E add-on) |
| **Maturity** | Stable | Stable |
| **Recommended For** | Existing integrations | New projects |

### Traditional 4-Phase Handshake

**Phases**: Invitation → Request → Response → Complete

```
Client                                    Server
  │                                          │
  │  1. Invitation (signed, plaintext)       │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  2. Request (bootstrap encrypted)        │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  3. Response (encrypted with session)    │
  │ <──────────────────────────────────────  │
  │                                          │
  │  4. Complete (final acknowledgment)      │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  Encrypted session established         │
```

**Use when**: Compatibility with existing A2A protocol integrations

**Documentation**: [handshake-en.md](./handshake-en.md)

### HPKE-Based 2-Phase Handshake (Recommended)

**Phases**: Initialize → Acknowledge

```
Client                                    Server
  │                                          │
  │  1. Init (HPKE enc + ephC)              │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  2. Ack (kid + ackTag + ephS)           │
  │ <──────────────────────────────────────  │
  │                                          │
  │  Encrypted session established         │
```

**Use when**: Starting new projects, need lower latency

**Documentation**: [hpke-based-handshake-en.md](./hpke-based-handshake-en.md)

---

## Document Descriptions

### Core Documentation

| Document | Language | Audience | Description |
|----------|----------|----------|-------------|
| **cryptographic-en.md** | English | All | Comprehensive cryptographic design and security model |
| **cryptographic-ko.md** | 한국어 | All | 암호학적 설계 및 보안 모델 (한국어) |
| **handshake-en.md** | English | Developers | Traditional 4-phase handshake implementation guide |
| **handshake-ko.md** | 한국어 | Developers | 전통적 4단계 핸드셰이크 구현 가이드 (한국어) |
| **hpke-based-handshake-en.md** | English | Developers | HPKE-based handshake implementation guide |
| **hpke-based-handshake-ko.md** | 한국어 | Developers | HPKE 기반 핸드셰이크 구현 가이드 (한국어) |

### Tutorials

| Document | Language | Audience | Description |
|----------|----------|----------|-------------|
| **hpke-detailed-en.md** | English | Beginners | Step-by-step HPKE tutorial with code examples |
| **hpke-detailed-ko.md** | 한국어 | Beginners | HPKE 단계별 튜토리얼 (코드 예제 포함, 한국어) |

---

## Learning Path

### Path 1: Traditional Handshake

1. Read [cryptographic-en.md](./cryptographic-en.md) - Security foundations
2. Read [handshake-en.md](./handshake-en.md) - 4-phase protocol
3. Review code in `/handshake` package
4. Build your integration

### Path 2: HPKE Handshake (Recommended)

1. Read [cryptographic-en.md](./cryptographic-en.md) - Security foundations (focus on HPKE section)
2. Read [hpke-based-handshake-en.md](./hpke-based-handshake-en.md) - 2-phase protocol
3. Follow [hpke-detailed-en.md](./hpke-detailed-en.md) - Hands-on tutorial
4. Review code in `/hpke` package
5. Build your integration

---

## Key Concepts

### DID Identity Binding

Both protocols use **Decentralized Identifiers (DIDs)** to bind sessions to agent identities:

- Ed25519 signing keys verify agent identity
- X25519 keys (derived from Ed25519) provide key exchange
- DID metadata stored on blockchain (Ethereum, Solana, Kaia)

### Forward Secrecy

Both protocols provide **forward secrecy**:

- **Traditional**: Ephemeral X25519 keys deleted after session establishment
- **HPKE**: HPKE Base mode + E2E ephemeral X25519 add-on

### Session Security

Once established, sessions use:

- **Encryption**: ChaCha20-Poly1305 AEAD (authenticated encryption)
- **MAC**: HMAC-SHA256 for additional integrity protection
- **Key Derivation**: HKDF-SHA256 with directional keys (C2S/S2C separation)
- **Nonce Management**: Per-session nonce tracking prevents replay attacks

---

## Security Properties

Both handshake protocols provide:

-  **Mutual Authentication**: Both agents verify each other's DIDs
-  **Forward Secrecy**: Past sessions remain secure even if long-term keys compromised
-  **Replay Protection**: Nonces and timestamps prevent message replay
-  **MitM Resistance**: DID signatures prevent man-in-the-middle attacks
-  **End-to-End Encryption**: Only communicating agents can decrypt messages

---

## Implementation Notes

### Choosing a Protocol

**Use Traditional (4-Phase) if**:
- Integrating with existing A2A protocol systems
- Need explicit invitation/acceptance flow
- Backward compatibility required

**Use HPKE-Based (2-Phase) if**:
- Starting a new project
- Need lower latency (1 RTT vs 2 RTT)
- Want modern cryptography (HPKE/RFC 9180)
- Prefer simpler state machine

### Code Packages

- **Traditional**: `github.com/sage-x-project/sage/handshake`
- **HPKE**: `github.com/sage-x-project/sage/hpke`
- **Session Management**: `github.com/sage-x-project/sage/session`
- **Cryptography**: `github.com/sage-x-project/sage/crypto/keys`

---

## Related Documentation

- **[RFC 9421 HTTP Message Signatures](../core/rfc9421-en.md)**: HTTP-level authentication
- **[Crypto Package Guide](../crypto/crypto-en.md)**: Cryptographic primitives
- **[DID Documentation](../did/)**: Decentralized identifier system

---

## Contributing

When updating handshake documentation:

1. Update both English and Korean versions
2. Ensure code examples match actual implementation
3. Run tests: `go test ./handshake/... ./hpke/...`
4. Update this README if adding new documents

---

## Support

For questions or issues:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://docs.sage-x-project.org
