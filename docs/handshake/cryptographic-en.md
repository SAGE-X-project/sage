# Cryptographic Overview

This document explains the cryptographic assurances across the SAGE protocol flow: protecting DID identity keys, exchanging ephemeral keys, deriving session keys, applying AEAD/HMAC protection, and finally communicating with RFC 9421.

## Protecting the X25519 ephemeral key exchange with DID identity keys (Ed25519)

During the handshake phase the protocol encrypts payloads with the peer agent's Ed25519 public key obtained from its DID Document (verificationMethod) to secure the ephemeral key exchange.

The ephemeral X25519 public key payload exchanged in the handshake is:

- **Signed**: The sender signs with its Ed25519 identity key to guarantee origin and integrity.
- **Bootstrap encrypted**: The sender converts the receiver's Ed25519 public key into X25519, performs Ephemeral-Static ECDH → HKDF → AEAD (AES-GCM), and sends the encrypted result.

**Security properties**

- **Confidentiality**: The bootstrap encryption ensures third parties cannot observe the ephemeral key or handshake content (AES-GCM).
- **Integrity & origin**: Ed25519 signature verification plus the AEAD authentication tag prevents tampering or substitution.
- **Identity binding**: Because verification relies on the DID Document's public key, the communicating peer is firmly bound to its identity.

### Assumptions (threat model prerequisites)

- **Valid DID → correct public key**: The peer's DID Document is current and unaltered, secured by blockchain anchoring and validation procedures.
- **Healthy randomness source**: Sufficient entropy is available for generating ephemeral keys and AEAD nonces.

### Risks and mitigations

#### 1. Man-in-the-middle (MitM)

- **Risk**  
  An attacker could replace the ephemeral public key during the handshake, letting the attacker control the derived session keys.
- **Mitigation**
  - **Identity signature verification**: The higher A2A layer signs/verifies the entire message with Ed25519 (DID + signature in metadata) to block spoofed identities.
  - **Bootstrap encryption integrity**: `EncryptWithEd25519Peer` uses AES-GCM and binds the sender's ephemeral key to the receiver's converted key via AAD `transcript := appendPrefix(pubKey.Bytes(), peerX)`. Any swap of the ephemeral key breaks the GCM tag immediately.
- **Operational notes**
  - Limit exposure when reporting signature verification failures (mismatch, unregistered DID, expiration).
  - Store only DID, context, and key fingerprints in handshake logs—never the plaintext payload.

#### 2. ECDH all-zero (RFC 7748 guidance)

- **Risk**  
  Malicious public keys (low-order/identity points) can force X25519 ECDH to produce an all-zero shared secret; using it in key derivation reintroduces vulnerabilities.
- **Mitigation**
  - Pass the result of `privKey.ECDH(peerPubKey)` into `sharedSecret(dh, err)` and (a) ensure a 32-byte result, (b) compare against all-zero in constant time, rejecting immediately.
  - Honor the RFC 7748 rule "all-zero shared secret MUST be rejected" at the implementation level.
- **Operational notes**
  - Log rejection events with fingerprints only and tie repeated offenses to peer blocking policies.

#### 3. Bootstrap encryption nonce/entropy

- **Risk**  
  Reusing AES-GCM nonces is catastrophic; poor randomness quality also weakens security.
- **Mitigation**
  - Generate a fresh, CSPRNG-based nonce each time (`nonce := make(...); io.ReadFull(rand.Reader, nonce)`).
  - Refresh ephemeral keys per session during the handshake, further reducing any key/nonce reuse risk.
- **Operational notes**
  - Extremely high-throughput environments may consider a per-session counter nonce.
  - Adopt a fail-closed policy if random number generation stalls or errors.

#### 4. DID public key freshness/integrity

- **Risk**  
  Using rotated or revoked keys, or stale DID Documents, causes verification failures or confusion.
- **Mitigation**  
  The protocol assumes upstream layers already validate the DID Document (as documented elsewhere). The DID is resolved to fetch the identity key for A2A signature verification and bootstrap encryption.
- **Operational notes**  
  Apply TTL/refresh/revocation policies to the DID resolver; protect cache integrity (for example, signed caches) and verify blockchain anchors.

## From X25519 shared secret → HKDF-SHA256 → session keys (AEAD/HMAC)

The shared secret produced during the handshake seeds the session encryption and signing keys.

1. **Extract a session seed from the shared secret**

   ```go
   hkdf.Extract(sha256.New, ikm, salt)
   ```

   The salt is derived from the label, context ID, and ordered ephemeral keys, yielding a PRK (`sessionSeed`).

2. **Expand the session seed into encryption and signing keys**

   ```go
   hkdfEnc := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("encryption"))
   s.encryptKey = make([]byte, 32)

   hkdfSign := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("signing"))
   s.signingKey = make([]byte, 32)
   ```

   The salt is the session ID. Distinct `info` strings ("encryption" / "signing") guarantee key separation.

**HKDF-Expand usage**

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

- Signing key (32 B) → HMAC-SHA256 (RFC 9421-style covered components)

  ```go
  func (s *SecureSession) SignCovered(covered []byte) []byte {
      m := hmac.New(sha256.New, s.signingKey)
      m.Write(covered)
      s.UpdateLastUsed()
      return m.Sum(nil)
  }
  ```

## Session layer

### Key derivation premise (HKDF-derived session keys)

- `sessionSeed = HKDF-Extract(SHA-256, shared_secret, salt)` combines the handshake shared secret with a salt that encodes the protocol label, context ID, and ordered ephemeral keys, enforcing key separation across sessions.
- Subsequent HKDF-Expand calls with distinct `info` strings ("encryption", "signing", etc.) produce purpose-specific keys so disclosure of one key does not compromise the others.

### Payload protection: ChaCha20-Poly1305 (Encrypt / Decrypt)

- **Confidentiality**: 256-bit keys with 96-bit nonces offer IND-CPA confidentiality.
- **Integrity/authentication**: Poly1305 provides a 128-bit authentication tag for INT-CTXT guarantees. Any tampering causes decryption failure.
- **Random nonces**: Each encryption call uses a CSPRNG-derived 96-bit nonce. Because AEAD is fragile to nonce reuse:
  - Never reuse a nonce with the same key; doing so breaks secrecy and integrity.
  - High-speed deployments may perform per-session counter-based nonces to avoid collision probabilities altogether.
- **Directional separation (optional)**: Using separate key slots for EncryptOutbound/DecryptInbound yields unique keys for client→server and server→client, containing the blast radius if one direction is compromised and reducing nonce-space collisions.

### Metadata integrity: HMAC-SHA256 (SignCovered / VerifyCovered)

- **EUF-CMA security**: HMAC modeled as a PRF with SHA-256 resists practical attacks; signatures remain unforgeable under chosen-message attacks unless the key leaks.
- **Covered components**: The normalized serialization includes `@method`, `@path`, `host`, `date`, `content-digest`, and `@signature-params`, binding headers, method, path, timestamp, and body digest to the signature. Downgrade or tampering attempts are therefore detected.
- **Content-Digest binding**: The `content-digest` header hashes the transmitted body (the ciphertext). Because HMAC signs that header, body/header tampering breaks verification.
- **Verification implementation**: Use `hmac.Equal` for constant-time comparison to avoid timing attacks.

### Additional considerations and mitigations

#### 1. Context binding failure

- **Risk**  
  Reusing the same shared secret across different protocols or contexts can trigger key/session collisions or cross-protocol attacks (for example, identical `ctxID` but different labels).
- **Mitigation**

  ```go
  h := sha256.New()
  h.Write([]byte(label))
  h.Write([]byte(p.ContextID))
  h.Write(lo)
  h.Write(hi)
  salt := h.Sum(nil)

  seed := hkdfExtractSHA256(sharedSecret, salt)
  ```

  The session ID is computed from `ComputeSessionIDFromSeed(label || seed)` so protocol labels remain distinct.

- **Operational notes**
  - Include the protocol version in the label (for example, "a2a/handshake v1") to maintain domain separation.
  - If new key material is required, refine HKDF `info` strings ("encryption", "signing", "ack-key", "traffic-key/c2s", `traffic-key/s2c`, and so on).

#### 2. Insufficient key separation

- **Risk**  
  Pulling encryption and signing keys from the same OKM without domain separation weakens isolation.
- **Mitigation**  
  Distinguish HKDF `info` strings ("encryption" vs. "signing") so each purpose gets a unique derived key.

#### 3. AEAD nonce reuse (nonce misuse)

- **Risk**  
  ChaCha20-Poly1305 breaks under nonce reuse. Random nonces are safe but collision probability grows with very long sessions.
- **Mitigation**  
  Generate a 12-byte nonce from a CSPRNG for every encryption; consider per-session counters when throughput requires it; abort sessions if entropy sources fail.

#### 4. Session lifetime/usage limits

- **Risk**  
  Excessive messages or long-lived sessions increase exposure.
- **Mitigation**
  - Configure `Config{MaxAge, IdleTimeout, MaxMessages}` to cap lifetime, idle periods, and message counts, and wipe keys/seeds on expiration via `Close()`.
  - Regenerate ephemeral keys per session during the handshake to minimize accidental key/nonce reuse.
- **Operational notes**
  - Tailor `MaxMessages`/`MaxAge` to your environment (for example, ≤10^5 messages, tens of minutes to a few hours).
  - Introduce a re-key (session renegotiation) workflow when expiration nears.

### Failure handling (security policy)

- Missing required headers or malformed values → HTTP 400
- Replay detection (`kid`, `nonce` reuse) → HTTP 401
- HMAC verification failure → HTTP 401
- AEAD tag failure/decryption failure → HTTP 401
- Session expired (`MaxAge`/`IdleTimeout`/`MaxMessages`) → HTTP 401 or an equivalent policy status
- Log only `kid`, `ctxID`, and fingerprints—never plaintext, keys, or seeds.

### Key lifetime and zeroization

- Each session derives fresh keys: the handshake negotiation yields a new `sessionSeed` per session, leading to unique encryption/signing keys. This provides forward secrecy at the session level.
- On session termination, zeroize keys and seeds in memory to minimize forensic risk.

## End-to-end guarantees

- **Only endpoints hold the keys**: Session keys are derived and stored solely by the two agents; proxies or gateways cannot decrypt or forge traffic. Both data and metadata are protected.
- **AEAD**: Ensures payload confidentiality and integrity (16-byte authentication tag).
- **HMAC signature**: Guards the covered components (method, path, headers, content digest) and enforces freshness through nonce/date usage.
- **Replay protection**: Combines `kid` + nonce caching, `Date` freshness, and session policies (`IdleTimeout`, `MaxMessages`) to block replays.

## Division of responsibility: AEAD tag and HMAC signature

- **AEAD authentication tag (ChaCha20-Poly1305/GCM)**  
  Verified automatically during decryption; any modification of the ciphertext causes immediate failure. Guarantees per-message integrity and authenticity.

- **HMAC-SHA256 (RFC 9421 style)**  
  Enforces integrity of headers, method, path, timestamp, and the content digest, and helps detect replays via nonce and `Date`.

This layered defense is well-suited to resist adversaries who can observe or alter headers after payload encryption.
