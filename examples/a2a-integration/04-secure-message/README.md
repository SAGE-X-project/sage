# Example 04: Secure Message Exchange

This example demonstrates end-to-end encrypted messaging between two agents using SAGE's multi-key infrastructure.

## What This Example Does

1. **Registers two agents** (Agent A and Agent B) with multi-key support
2. **Encrypts a message** using HPKE (Hybrid Public Key Encryption)
3. **Signs the encrypted message** using Ed25519
4. **Transmits the secure message** (JSON format)
5. **Verifies the signature** at the recipient
6. **Decrypts the message** using the recipient's private key

## Security Properties

| Property | Implementation | Guarantee |
|----------|----------------|-----------|
| **Confidentiality** | HPKE with X25519 | Only recipient can decrypt |
| **Authentication** | Ed25519 signature | Proves sender identity |
| **Integrity** | Signature verification | Detects tampering |
| **Non-repudiation** | Digital signature | Sender can't deny |

## Prerequisites

- Local Hardhat node running
- SageRegistryV4 contract deployed
- Environment variables set

## Running the Example

```bash
cd examples/a2a-integration/04-secure-message
go run main.go
```

## Message Flow

```
┌──────────────┐                                  ┌──────────────┐
│   Agent A    │                                  │   Agent B    │
│              │                                  │              │
│ Private Keys:│                                  │ Private Keys:│
│  - Ed25519   │                                  │  - Ed25519   │
│  - X25519    │                                  │  - X25519    │
└──────┬───────┘                                  └──────┬───────┘
       │                                                 │
       │ 1. Encrypt with B's X25519 public key          │
       │    (HPKE)                                       │
       ├────────────────────────────────────────────────┤
       │                                                 │
       │ 2. Sign with A's Ed25519 private key           │
       ├────────────────────────────────────────────────┤
       │                                                 │
       │ 3. Send encrypted + signed message             │
       ├────────────────────────────────────────────────►
       │                                                 │
       │                         4. Verify signature    │
       │                            (A's Ed25519 pubkey)│
       │                                                 │
       │                         5. Decrypt message     │
       │                            (B's X25519 privkey)│
       │                                                 │
```

## Expected Output

```
╔═══════════════════════════════════════════════════════════╗
║     SAGE Example 04: Secure Message Exchange             ║
╚═══════════════════════════════════════════════════════════╝

👥 Setup: Creating Agent A and Agent B
═════════════════════════════════════════════════════════
Generating keys for Agent A...
Generating keys for Agent B...

Registering Agent A...
✓ Agent A registered: did:sage:ethereum:SecureAgent-A-123456

Registering Agent B...
✓ Agent B registered: did:sage:ethereum:SecureAgent-B-123456

📨 Step 1: Agent A Sends Encrypted Message
═════════════════════════════════════════════════════════

Plaintext message: Hello Agent B! This is a confidential message...
Message length: 89 bytes

🔐 Encrypting message with HPKE...
   Using Agent B's X25519 public key
✓ Message encrypted
  Ciphertext length: 89 bytes

✍️  Signing encrypted message...
   Using Agent A's Ed25519 private key
✓ Message signed
  Signature length: 64 bytes

📤 Message ready for transmission
  From:       did:sage:ethereum:SecureAgent-A-123456
  To:         did:sage:ethereum:SecureAgent-B-123456
  Timestamp:  2025-01-19T12:34:56Z
  Total size: 456 bytes

💾 Message saved to: secure-message.json

📬 Step 2: Agent B Receives and Processes Message
═════════════════════════════════════════════════════════

✓ Message received
  From:       did:sage:ethereum:SecureAgent-A-123456
  To:         did:sage:ethereum:SecureAgent-B-123456
  Timestamp:  2025-01-19T12:34:56Z

🔍 Verifying signature...
   Using Agent A's Ed25519 public key
✓ Signature verified!
  The message is authentic and from Agent A

🔓 Decrypting message...
   Using Agent B's X25519 private key
✓ Message decrypted!

💬 Step 3: Decrypted Message
═════════════════════════════════════════════════════════

From: did:sage:ethereum:SecureAgent-A-123456
Message: Hello Agent B! This is a confidential message...

╔═══════════════════════════════════════════════════════════╗
║     Secure Messaging Complete!                            ║
╚═══════════════════════════════════════════════════════════╝

🎉 Success! Agent A and Agent B exchanged a secure message.

Security guarantees achieved:
  1. ✓ Confidentiality - Only Agent B can decrypt
  2. ✓ Authentication - Signature proves sender is Agent A
  3. ✓ Integrity - Any tampering breaks the signature
  4. ✓ Non-repudiation - Agent A can't deny sending
```

## Message Format

The `SecureMessage` structure:

```go
type SecureMessage struct {
    From      string // Sender's DID
    To        string // Recipient's DID
    Timestamp string // ISO 8601 timestamp
    Content   []byte // Encrypted ciphertext
    Signature []byte // Ed25519 signature
    Nonce     []byte // HPKE nonce/encapsulated key
}
```

Serialized as JSON:

```json
{
  "from": "did:sage:ethereum:SecureAgent-A-123456",
  "to": "did:sage:ethereum:SecureAgent-B-123456",
  "timestamp": "2025-01-19T12:34:56Z",
  "content": "...", // base64 encoded ciphertext
  "signature": "...", // base64 encoded signature
  "nonce": "..." // base64 encoded HPKE nonce
}
```

## Cryptographic Operations

### 1. Encryption (Sender)

```go
// Get recipient's X25519 public key
recipientPubKey := agentBX25519.PublicKey()

// Encrypt plaintext with HPKE
ciphertext, nonce, err := hpke.Seal(recipientPubKey, plaintext, nil)
```

### 2. Signing (Sender)

```go
// Sign the ciphertext
signature, err := agentAEd25519.Sign(ciphertext)
```

### 3. Verification (Recipient)

```go
// Verify signature before decryption
valid, err := agentAEd25519.PublicKey().Verify(ciphertext, signature)
if !valid {
    return errors.New("signature verification failed")
}
```

### 4. Decryption (Recipient)

```go
// Decrypt with recipient's private key
plaintext, err := hpke.Open(agentBX25519PrivateKey, ciphertext, nonce, nil)
```

## Why This Matters

### Traditional Encryption (TLS/HTTPS)

```
Client ──[TLS]──► Proxy ──[TLS]──► Server
         ▲                  ▲
         │                  │
    Different keys   Proxy can read
```

Problems:
- Proxy can intercept and read
- Not true end-to-end encryption
- Trust in intermediaries required

### SAGE End-to-End Encryption

```
Agent A ══[E2E]══════════════════════════► Agent B
         ▲                                  ▲
         │                                  │
    Only A & B have keys
    No intermediary can decrypt
```

Benefits:
- True end-to-end encryption
- Zero-trust architecture
- No middleman can intercept

## Production Considerations

### 1. Key Management

```go
// Store private keys securely
// Never transmit private keys
// Use hardware security modules (HSM) for production
```

### 2. Message Transport

```go
// Use HTTPS for transport layer security
// Add retry logic for network failures
// Implement message queuing for reliability
```

### 3. Replay Protection

```go
// Check timestamp is recent
if time.Since(msg.Timestamp) > 5*time.Minute {
    return errors.New("message too old")
}

// Track processed message IDs
if seen(msg.ID) {
    return errors.New("replay detected")
}
```

### 4. Forward Secrecy

```go
// Rotate X25519 keys periodically
// Use ephemeral keys for each session
// Implement Perfect Forward Secrecy (PFS)
```

## Attack Scenarios

| Attack | Mitigation |
|--------|------------|
| Man-in-the-middle | HTTPS + signature verification |
| Message tampering | Ed25519 signature |
| Replay attack | Timestamp + nonce checking |
| Key compromise | Regular key rotation |
| Impersonation | DID verification on blockchain |

## Performance

Typical operation times (on modern hardware):

| Operation | Time | Notes |
|-----------|------|-------|
| HPKE Encryption | ~0.1ms | X25519 + AES-GCM |
| Ed25519 Signing | ~0.05ms | Very fast |
| Signature Verification | ~0.1ms | Slightly slower |
| HPKE Decryption | ~0.1ms | Similar to encryption |

**Total latency**: ~0.35ms for full encrypt-sign-verify-decrypt cycle

## Next Steps

1. **Implement message queuing**: Use Redis or RabbitMQ
2. **Add session management**: Reuse keys for multiple messages
3. **Build real-time chat**: WebSocket with E2E encryption
4. **Create agent marketplace**: Discovery + secure communication

## References

- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 8032: EdDSA (Ed25519)](https://www.rfc-editor.org/rfc/rfc8032.html)
- [RFC 7748: X25519](https://www.rfc-editor.org/rfc/rfc7748.html)
- [A2A Protocol](https://github.com/a2aproject/a2a)

## License

LGPL-v3 - See [LICENSE](../../../LICENSE)
