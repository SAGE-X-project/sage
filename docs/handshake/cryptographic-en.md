# Cryptographic

This document explains the cryptographic assurances across the SAGE protocol pipeline: DID identity keys → ephemeral key exchange → session key derivation → AEAD/HMAC protection → RFC 9421 messaging.

## Protecting the X25519 ephemeral key exchange with DID identity keys (Ed25519)

During the handshake phase the initiator encrypts the ephemeral key exchange payload with the peer agent’s Ed25519 public key obtained from the peer’s DID Document (`verificationMethod`).

The X25519 ephemeral public key payload exchanged during the handshake is:

- **Signed**: The sender signs with its Ed25519 identity key to preserve origin and integrity.
- **Bootstrap encrypted**: The sender converts the receiver’s Ed25519 public key to X25519 and applies Ephemeral-Static ECDH → HKDF → AEAD (AES-GCM) before sending.

**Security properties**

- **Confidentiality**: Bootstrap encryption prevents third parties from viewing the ephemeral key or handshake transcript (AES-GCM).
- **Integrity / origin**: Ed25519 signature verification plus the AEAD authentication tag blocks tampering and substitution.
- **Identity binding**: Verification is tied to the DID Document public key, so the peer’s identity is fixed.

### Assumptions (threat model prerequisites)

- **Valid DID → correct public key**: The peer’s DID Document is assumed to be current and unmodified (enforced through blockchain anchoring and validation procedures).
- **Healthy entropy source**: Adequate randomness exists for generating ephemeral keys and AEAD nonces.

### Risks and mitigations

#### 1. Man-in-the-middle (MitM)

- **Risk**  
  If an attacker swaps the ephemeral public key mid-handshake, all derived session keys can be anchored to the attacker.
- **Mitigation**
  - **Identity signature verification**: The upstream A2A layer signs and verifies the entire message with Ed25519 (DID plus signature in metadata) to stop spoofed senders.
  - **Bootstrap encryption integrity**: `EncryptWithEd25519Peer` uses AES-GCM and sets `transcript := appendPrefix(pubKey.Bytes(), peerX)` as AAD, binding the sender’s ephemeral key to the receiver’s converted key. Any swap destroys the GCM tag immediately.
- **Operational notes**
  - Expose minimal detail when A2A signature verification fails (signature mismatch, unknown DID, expiry, and so on).
  - Log only DID, context, and ephemeral key fingerprints—never plaintext handshake material.

#### 2. ECDH all-zero (RFC 7748 guidance)

- **Risk**  
  Malicious public keys (low-order or identity points) can force X25519 ECDH to produce an all-zero shared secret; using that as key material is unsafe.
- **Mitigation**
  - Always pass `privKey.ECDH(peerPubKey)` to `sharedSecret(dh, err)` and (a) require a 32-byte result, (b) compare against all-zero in constant time, rejecting on failure.
  - Enforce the RFC 7748 rule “all-zero shared secret MUST be rejected” in code.
- **Operational notes**
  - Record rejection fingerprints in audit logs and tie repeated events to peer-blocking policies.

#### 3. Bootstrap encryption nonce / entropy

- **Risk**  
  Reusing an AES-GCM nonce is catastrophic; weak randomness is also dangerous.
- **Mitigation**
  - Generate a fresh CSPRNG nonce every time (`nonce := make(...); io.ReadFull(rand.Reader, nonce)`).
  - Regenerate ephemeral keys per session during the handshake to reduce accidental key/nonce reuse.
- **Operational notes**
  - Consider counter-based nonces (per-session counters) in ultra-high-throughput environments.
  - Fail closed if randomness exhaustion or errors occur.

#### 4. DID public key freshness / integrity

- **Risk**  
  Using rotated or revoked keys, or stale DID Documents, leads to verification failures and confusion.
- **Mitigation**  
  Higher layers validate DID Documents (see the related documentation). Identity keys fetched via DIDs power A2A signature verification and bootstrap encryption.
- **Operational notes**  
  Apply TTL, refresh, and revocation policies to the DID resolver, verify blockchain anchors, and protect cache integrity (for example, signed caches).

## Protecting the X25519 handshake with DID identity keys & HPKE

This section describes the recommended way SAGE combines the **DID identity layer** with **HPKE (RFC 9180)** to safeguard the handshake-time X25519 key exchange. The highlights are:

- **Identity & integrity**: guaranteed by **DID signatures (Ed25519)**.
- **Session seed agreement**: secured by **HPKE (Base mode)**.
- **Mitigating static KEM key leakage**: **Ephemeral–ephemeral reinforcement (PFS add-on)** removes the long-term key exposure window.

### Payload recap

**Client → Server (Init)**

- `enc`: HPKE encapsulation (32 B, sent in the clear)
- `info`, `exportCtx`: context-binding strings (for example, `sage/hpke v1|ctx=...|init=...|resp=...`, `exporter:ctx`)
- `nonce`, `ts`: replay and freshness checks
- `ephC` _(optional, PFS add-on)_: client ephemeral X25519 public key (32 B)

> A2A metadata includes the **sender DID** and its **Ed25519 signature**, providing origin and integrity for the message.

**Server → Client (Ack)**

- `kid`: session key identifier
- `ackTagB64`: key-confirmation tag
  `ackKey = HKDF(seed, "ack-key", 32)`  
  `ackTag = HMAC(ackKey, "hpke-ack|" + ctxID + "|" + nonce + "|" + kid)`
- `ephS` _(optional, PFS add-on)_: server ephemeral X25519 public key (32 B)

### Seed derivation (HPKE Base + PFS add-on)

1. **HPKE Base** – both ends reproduce `exporterHPKE` using the recipient’s static X25519 KEM public key.
2. **PFS add-on (recommended)** – mix `ssE2E = X25519(ephC, ephS)` into the final seed.

```
Base only:
    seed = exporterHPKE

Base + PFS add-on:
    seed = HKDF-Extract( salt = exportCtx,
                         IKM  = exporterHPKE || ssE2E )
           → HKDF-Expand(info = "SAGE-HPKE+E2E-Combiner", L=32)
```

The client mirrors the same procedure, validates `ackTag`, and binds the resulting session to `kid`.

### Security properties

- **Confidentiality (post-handshake)**: derive session keys from `seed` via HKDF and protect application data with **AEAD** (e.g., ChaCha20-Poly1305).
- **Integrity / origin (handshake messages)**: Init/Ack messages are verified with **Ed25519 DID signatures**.
- **Key confirmation**: `ackTag` proves both sides derived the **same seed**.
- **Context binding**: embedding `ctxID`, `initDID`, and `respDID` inside `info/exportCtx` prevents **cross-context reuse or downgrades**.
- **Replay resistance**: Init `nonce` plus RFC 9421 request nonce checks through **ReplayGuard**.
- **Freshness**: `ts` is validated within the configured `±maxSkew` window.

### Threat model and assumptions

- **Valid DID → correct public key** – DID Documents are assumed current and authentic (enforced through blockchain anchoring and verification procedures).
- **Strong randomness** – X25519 ephemeral keys, AEAD nonces, and HPKE internals come from a secure CSPRNG.

### Key threats & mitigations

#### 1. MitM / downgrade

- **Risk**  
  Manipulating `ctxID`, peer DID, or `info` to force the exporter into a different context.
- **Mitigations**
  - Verify the full A2A message with **DID signatures** to block spoofed senders.
  - Check **`info/exportCtx` consistency** (both sides must use the same builder).
  - Validate **`ackTag`**—an HMAC over `ctxID`, `nonce`, `kid`, and the transcript hash—to confirm the actual key agreement.

#### 2. Replay

- **Risk**  
  Re-sending Init or HTTP requests to disturb state.
- **Mitigations**
  - Init: record `nonce` and enforce **freshness (`ts`)**.
  - Session phase: RFC 9421 nonces go through **ReplayGuardSeenOnce(kid, nonce)**.

#### 3. ECDH all-zero (RFC 7748)

- **Risk**: crafted public keys yield `0x00…00` as the X25519 shared secret.
- **Mitigation**: reject any all-zero result immediately after length and constant-time checks; log only fingerprints.

#### 4. Recipient static KEM private key leakage

- **Risk**: Base mode alone lets attackers reconstruct past **exporter** values from captured `enc`, exposing historical traffic.
- **Mitigations**
  - Prefer the **PFS add-on** so the seed depends on `exporterHPKE || ssE2E`, which cannot be reproduced with the static key alone.
  - Operational safeguards: shorten KEM key lifetimes, automate rotation, store secrets in HSMs, and, if possible, hide `enc` inside an additional encrypted channel.

#### 5. Insufficient request integrity / coverage

- **Mitigation**:  
  After the session is established, protect HTTP requests with **RFC 9421** (covering `@method`, `@path`, `@authority`, `content-digest`, `date`, etc.) to ensure header/body integrity and non-repudiation.

---

### Operational tips

- Keep **InfoBuilder** consistent between client and server with a service-specific but fixed format.
- Set **`maxSkew`** and nonce TTLs according to network latency.
- Session policy: adjust `MaxAge`, `IdleTimeout`, and `MaxMessages` to control lifetime and throughput; securely wipe keys when expiring sessions.
- Logging: record only DID, context, and key fingerprints—never plaintext, seeds, or key material.
- When strong forward secrecy is required, treat **Base + PFS add-on** as the default deployment mode.

## From X25519 shared secret → HKDF-SHA256 → session keys (AEAD/HMAC)

The shared secret obtained during the handshake produces the session encryption and signing keys.

### 1. X25519 ECDH → shared secret (`dh`)

- X25519, defined in RFC 7748, is a Montgomery-curve ECDH scheme based on the hardness of the discrete logarithm problem.
- Implementation rule: if the derived `dh` is all zeros (32 bytes of `0x00`), reject immediately (MUST reject) to block low-order point attacks. Check length and all-zero status in constant time.

### 2. Deriving a seed from the shared secret

Use HKDF-Extract (SHA-256) to produce a PRK (`sessionSeed`) from the shared secret.

```go
hkdf.Extract(sha256.New, ikm, salt)
```

- HKDF (Extract/Expand) behaves as a PRF; even with a biased `dh`, the Extract step normalizes it to a uniform PRK.
- Context binding: set `salt = H(label || contextID || sort(ephA, ephB))`, mixing the protocol label, session context, and both ordered ephemeral public keys. This yields session uniqueness and mitigates reflection or cross-protocol attacks.

### 3. HKDF-Expand for key separation

```go
hkdfEnc := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("encryption"))
s.encryptKey = make([]byte, 32)

hkdfSign := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("signing"))
s.signingKey = make([]byte, 32)
```

- Using distinct `info` strings (“encryption”, “signing”) produces independent encryption and HMAC keys. Compromise of one key does not affect the other.
- Additional isolation: reusing the session ID as the salt for all keys derived from the PRK lowers collision risk among related keys.

### Additional considerations and mitigations

#### 1. Context binding failure

- **Risk**  
  Reusing the same shared secret across different protocols or contexts can cause key/session collisions or cross-protocol attacks (for example, identical `ctxID` with mismatched labels, or poor salt/label design across contexts).
- **Mitigation**
  - `DeriveSessionSeed` binds the label, context, and ordered ephemeral keys before extracting the PRK.

  ```go
  h := sha256.New()
  h.Write([]byte(label))
  h.Write([]byte(p.ContextID))
  h.Write(lo)
  h.Write(hi)
  salt := h.Sum(nil)

  seed := hkdfExtractSHA256(sharedSecret, salt)
  ```

  - Compute the session ID as `ComputeSessionIDFromSeed(label || seed)` to keep protocol labels distinct.
- **Operational notes**
  - Include the protocol version in the label (for example, “a2a/handshake v1”) to reinforce domain separation.
  - When expanding to additional purposes, specialize HKDF `info` strings (for example, “encryption”, “signing”, “ack-key”, “traffic-key/c2s”, “traffic-key/s2c”).

#### 2. Insufficient key separation

- **Risk**  
  Pulling encryption and signing keys from the same OKM without domain separation weakens isolation.
- **Mitigation**  
  Keep HKDF `info` values distinct (“encryption” vs. “signing”) to enforce domain separation.

#### 3. AEAD nonce reuse (nonce misuse)

- **Risk**  
  ChaCha20-Poly1305 fails catastrophically under nonce reuse. Random nonces are safe but collision probability accumulates in very long sessions.
- **Mitigation**  
  Generate a 12-byte CSPRNG nonce for every encryption (`nonce := rand(12B)`), regenerate ephemeral handshake keys each session, and consider per-session counters for extremely high throughput.

#### 4. Key lifetime / usage limits

- **Risk**  
  Excessive messages or long-lived sessions enlarge the statistical attack surface.
- **Mitigation**
  - Configure `Config{MaxAge, IdleTimeout, MaxMessages}` to cap lifetime, idle time, and message count, and wipe keys/seeds via `Close()` when a session expires.
  - Regenerate ephemeral keys for each session to reduce key/nonce reuse.
- **Operational notes**
  - Tune `MaxMessages` / `MaxAge` to your environment (for example, ≤10^5 messages, tens of minutes to a few hours).
  - Introduce a re-key (session renegotiation) procedure when expiration is near.

## Session layer

### Key premise (HKDF-derived session keys)

- Session keys come from `sessionSeed = HKDF-Extract(SHA-256, shared_secret, salt)`. Using the shared secret plus a binding salt (context, label, ordered ephemeral keys) delivers key separation across sessions.
- Each session generates a fresh X25519 ephemeral key, yielding a new `dh` and therefore a new PRK and session keys. Once those keys are destroyed, previous sessions cannot be recovered—even if the long-term Ed25519 identity key leaks—providing forward secrecy at the session level.

### Payload protection: ChaCha20-Poly1305 (AEAD)

- Encryption key (32 B) → ChaCha20-Poly1305 AEAD

  ```go
  // Initialize AEAD cipher
  aead, err := chacha20poly1305.New(sess.encryptKey)
  if err != nil {
      return nil, fmt.Errorf("failed to create AEAD: %w", err)
  }
  sess.aead = aead

  ciphertext := sess.aead.Seal(nil, nonce, plaintext, nil)
  ```

- **Confidentiality**: 256-bit key, 96-bit nonce, stream cipher plus Poly1305 authentication provide IND-CPA confidentiality.
- **Integrity / authentication**: The 128-bit Poly1305 tag yields INT-CTXT guarantees; any tampering fails verification.
- **Random nonce**: Each encryption uses a CSPRNG 96-bit nonce, keeping collisions negligible. Because AEAD is nonce-sensitive:
  - Never reuse a nonce with the same key (doing so breaks confidentiality and integrity).
  - Consider per-session counter nonces in high-speed scenarios to eliminate collision probability.
- **Directional separation (optional)**: `EncryptOutbound` / `DecryptInbound` can operate with distinct keys for client→server and server→client, reducing blast radius if one direction leaks and lowering nonce-space collision risk.

### Metadata integrity: HMAC-SHA256

RFC 9421-style “covered components” signature:

```go
func (s *SecureSession) SignCovered(covered []byte) []byte {
    m := hmac.New(sha256.New, s.signingKey)
    m.Write(covered)
    s.UpdateLastUsed()
    return m.Sum(nil)
}
```

- **EUF-CMA security**: HMAC-SHA256 is modeled as a PRF, and no practical attacks exist. Chosen-message forgeries are infeasible unless the key leaks.
- **Covered components**: `@method`, `@path`, `host`, `date`, `content-digest`, and `@signature-params` are serialized in canonical order, binding headers, method, path, timestamp, and body digest to the signature. Header-only modifications, downgrade attacks, or date tampering are blocked.
- **Content-Digest binding**: `content-digest` hashes the transmitted body (ciphertext in this context); since HMAC signs that header, altering either header or body breaks the signature.
- **Verification implementation**: Use `hmac.Equal` for constant-time comparison to prevent timing side-channel attacks.

### Failure handling (security policy)

- Missing required headers or malformed values → HTTP 400
- Replay detected (`kid`, `nonce` reuse) → HTTP 401
- HMAC verification failure → HTTP 401
- AEAD tag failure / decryption failure → HTTP 401
- Session expired (`MaxAge`, `IdleTimeout`, `MaxMessages`) → HTTP 401 or a policy-specific status
- Logs should include only `kid`, `ctxID`, and fingerprints—never plaintext, keys, or seeds.

### Key lifetime and zeroization

- New keys per session: every handshake (ephemeral key exchange) yields a fresh `sessionSeed`, generating new encryption and signing keys and contributing to session-level forward secrecy.
- Zeroization: overwrite keys and seeds with zeros when the session ends to minimize forensic exposure.

## End-to-end (E2E) guarantees

- **Keys reside only at the endpoints**: Session keys are derived and stored exclusively by the two agents; proxies/gateways cannot decrypt or forge data. Both data and metadata are protected.
- **AEAD**: Ensures payload confidentiality and integrity (16-byte authentication tag).
- **HMAC signature**: Protects the covered components (`@method`, `@path`, `host`, `date`, `content-digest`, etc.) and resists replay.
- **Replay protection**: Combines `kid` + nonce cache, `Date` freshness, and session policies (`IdleTimeout`, `MaxMessages`) to stop reuse.

## Division of duties: AEAD tag and HMAC signature

- **AEAD authentication tag (ChaCha20-Poly1305 / GCM)**  
  Verified automatically during decryption; any ciphertext modification fails immediately, guaranteeing per-message integrity and authenticity.
- **HMAC-SHA256 (RFC 9421 style)**  
  Protects metadata—headers, method, path, timestamp, content digest—and provides replay resistance via nonce and `Date`.

This layered defense is well-suited to environments where adversaries may tamper with headers after payload encryption or attempt to interfere with proxy traffic.
