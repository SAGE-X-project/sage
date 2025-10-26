# SAGE Session Management

The `session` package provides secure, authenticated communication sessions between AI agents using AEAD encryption (ChaCha20-Poly1305) and replay protection. It manages session lifecycle, key derivation, automatic cleanup, and nonce tracking for secure agent-to-agent messaging.

## Overview

After agents complete a handshake (using HPKE key agreement), they establish a secure session for ongoing communication. The session package handles symmetric encryption, message authentication, replay attack prevention, and automatic session expiration.

### Key Benefits

- **AEAD Encryption**: ChaCha20-Poly1305 for authenticated encryption
- **Replay Protection**: Nonce-based defense against message replay attacks
- **Automatic Cleanup**: Background expiration of idle and aged sessions
- **HPKE Integration**: Direct derivation from HPKE exporter secrets
- **Direction-Separated Keys**: Independent inbound/outbound encryption keys
- **Zero Allocation**: Object pooling for high-performance scenarios

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SAGE Components (handshake, messaging)                 │
│  - Post-handshake communication                          │
│  - Encrypted message exchange                            │
│  - Replay attack prevention                              │
└────────────────────┬────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────┐
│  session.Manager (session lifecycle)                    │
│  - CreateSession()                                       │
│  - GetSession()                                          │
│  - BindKeyID()                                           │
│  - Cleanup expired sessions                              │
└────────────────────┬────────────────────────────────────┘
                     │ manages
          ┌──────────┴──────────┬──────────────────┐
          ▼                     ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐
│ SecureSession    │  │ NonceCache       │  │ Metadata     │
├──────────────────┤  ├──────────────────┤  ├──────────────┤
│ • ChaCha20-      │  │ • TTL-based      │  │ • Created    │
│   Poly1305 AEAD  │  │ • keyid+nonce    │  │ • LastUsed   │
│ • HKDF key       │  │ • Auto GC        │  │ • MsgCount   │
│   derivation     │  │                  │  │ • Expiration │
│ • Direction keys │  │                  │  │              │
└──────────────────┘  └──────────────────┘  └──────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────┐
│  Cryptographic Primitives                               │
│  ├─ ChaCha20-Poly1305: Authenticated encryption         │
│  ├─ HKDF-SHA256: Key derivation from HPKE secrets       │
│  └─ HMAC-SHA256: Message signatures for covered data    │
└─────────────────────────────────────────────────────────┘
```

## Cryptographic Algorithms

### ChaCha20-Poly1305 AEAD

**Properties:**
- **Cipher**: ChaCha20 stream cipher (256-bit key)
- **Authentication**: Poly1305 MAC (128-bit tag)
- **Nonce Size**: 12 bytes (96 bits)
- **Key Size**: 32 bytes (256 bits)
- **Performance**: ~3-5 cycles/byte (software), faster than AES-GCM without hardware

**Standards:**
- RFC 8439 (ChaCha20 and Poly1305)
- IETF variant (96-bit nonce)

**Why ChaCha20-Poly1305?**
- ✅ Constant-time (side-channel resistant)
- ✅ Fast on all platforms (no hardware dependency)
- ✅ Widely deployed (TLS 1.3, WireGuard, Signal)
- ✅ Proven security (formal verification)
- ❌ Not hardware-accelerated like AES-GCM

### HKDF-SHA256 (Key Derivation)

**Properties:**
- **Hash Function**: SHA-256
- **Input**: HPKE exporter secret or ECDH shared secret
- **Output**: Multiple independent keys (encryption, signing)
- **Info**: Context string for domain separation

**Standards:**
- RFC 5869 (HKDF)

**Key Derivation Hierarchy:**
```
HPKE Exporter Secret (from handshake)
    │
    ├─ HKDF-Extract(salt="sage/hpke v1")
    │      │
    │      ├─ Session Encryption Key (32 bytes)
    │      ├─ Session Signing Key (32 bytes)
    │      ├─ C2S (Client→Server) Enc Key (32 bytes)
    │      ├─ C2S Signing Key (32 bytes)
    │      ├─ S2C (Server→Client) Enc Key (32 bytes)
    │      └─ S2C Signing Key (32 bytes)
    │
    └─ Total: 192 bytes of independent key material
```

### Direction-Separated Keys

**Initiator (Client) Perspective:**
- **Outbound**: C2S encryption + C2S signing
- **Inbound**: S2C encryption + S2C signing

**Responder (Server) Perspective:**
- **Outbound**: S2C encryption + S2C signing
- **Inbound**: C2S encryption + C2S signing

**Why separate directions?**
- ✅ Prevent key reuse across different contexts
- ✅ Enable unidirectional rate limiting
- ✅ Support asymmetric security levels
- ✅ Simplify concurrent read/write

## Core Components

### Manager

Session lifecycle management:

```go
type Manager struct {
    sessions      map[string]Session  // sessionID -> Session
    byKeyID       map[string]string   // keyID -> sessionID
    nonceCache    *NonceCache         // Replay protection
}

// Core operations
func NewManager() *Manager
func (m *Manager) CreateSession(sessionID string, sharedSecret []byte) (Session, error)
func (m *Manager) CreateSessionWithConfig(sessionID string, sharedSecret []byte, config Config) (Session, error)
func (m *Manager) GetSession(sessionID string) (Session, bool)
func (m *Manager) DeleteSession(sessionID string) error
func (m *Manager) BindKeyID(keyID, sessionID string) error
func (m *Manager) GetSessionByKeyID(keyID string) (Session, bool)
func (m *Manager) GetStatus() Status
```

**Features:**
- Automatic background cleanup (every 30 seconds)
- Session expiration (max age, idle timeout)
- KeyID binding for quick lookups
- Replay attack prevention (nonce cache)
- Session pooling for zero allocation

### Session Interface

Core session operations:

```go
type Session interface {
    // Identification
    GetID() string
    GetCreatedAt() time.Time
    GetLastUsedAt() time.Time

    // Lifecycle
    IsExpired() bool
    UpdateLastUsed()
    Close() error

    // Cryptographic operations
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    EncryptAndSign(plaintext []byte, covered []byte) ([]byte, []byte, error)
    DecryptAndVerify(cipher []byte, covered []byte, mac []byte) ([]byte, error)
    SignCovered(covered []byte) []byte
    VerifyCovered(covered, sig []byte) error

    // Statistics
    GetMessageCount() int
    GetConfig() Config
}
```

### Config

Session policies and limits:

```go
type Config struct {
    MaxAge      time.Duration `json:"maxAge"`      // Absolute expiration (e.g., 1 hour)
    IdleTimeout time.Duration `json:"idleTimeout"` // Idle timeout (e.g., 10 minutes)
    MaxMessages int           `json:"maxMessages"` // Message limit per session
}

// Default configuration
Config{
    MaxAge:      time.Hour,        // 1 hour absolute
    IdleTimeout: 10 * time.Minute, // 10 minutes idle
    MaxMessages: 1000,             // 1000 messages max
}
```

**Expiration Logic:**
- **MaxAge**: Session expires after this duration since creation (absolute limit)
- **IdleTimeout**: Session expires if no messages for this duration
- **MaxMessages**: Session expires after sending/receiving this many messages
- **Any condition triggers expiration**

### SecureSession

ChaCha20-Poly1305 AEAD implementation:

```go
type SecureSession struct {
    id           string
    createdAt    time.Time
    lastUsedAt   time.Time
    messageCount int
    config       Config

    // Cryptographic materials
    sessionSeed  []byte
    keyMaterial  []byte      // Pre-allocated 192 bytes

    // Direction-separated keys
    outKey       []byte      // Outbound encryption key
    inKey        []byte      // Inbound encryption key
    outSign      []byte      // Outbound signing key
    inSign       []byte      // Inbound signing key
    aeadOut      cipher.AEAD // Outbound AEAD cipher
    aeadIn       cipher.AEAD // Inbound AEAD cipher
}
```

**Key Properties:**
- Zero-copy key derivation (pre-allocated buffer)
- Separate AEAD ciphers for each direction
- Thread-safe with RWMutex
- Automatic last-used timestamp updates

### NonceCache

Replay attack prevention:

```go
type NonceCache struct {
    ttl  time.Duration
    data sync.Map // keyid -> sync.Map (nonce -> expiryUnix)
}

func NewNonceCache(ttl time.Duration) *NonceCache
func (n *NonceCache) Seen(keyid, nonce string) bool
func (n *NonceCache) DeleteKey(keyid string)
```

**Features:**
- TTL-based expiration (default: 10 minutes)
- Per-keyID nonce tracking
- Automatic garbage collection (every 1 minute)
- Thread-safe (sync.Map)

**Usage Pattern:**
```go
if nonceCache.Seen(keyID, nonce) {
    return fmt.Errorf("replay attack detected")
}
// Process message...
```

## Session Lifecycle

```
┌─────────────┐
│  Handshake  │  (HPKE key agreement)
│  Complete   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  1. Create Session                      │
│     manager.CreateSession(sid, secret)  │
│     - Derive encryption keys (HKDF)     │
│     - Initialize AEAD ciphers           │
│     - Start lifecycle tracking          │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  2. Bind KeyID (optional)               │
│     manager.BindKeyID(keyID, sid)       │
│     - Enable lookup by keyID            │
│     - Associate with nonce cache        │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  3. Message Exchange                    │
│     session.Encrypt(plaintext)          │
│     session.Decrypt(ciphertext)         │
│     - ChaCha20-Poly1305 AEAD            │
│     - Update last-used timestamp        │
│     - Increment message counter         │
└──────┬──────────────────────────────────┘
       │
       ├─── (continue messaging)
       │
       ▼
┌─────────────────────────────────────────┐
│  4. Session Expiration (any of):       │
│     - MaxAge exceeded                   │
│     - IdleTimeout exceeded              │
│     - MaxMessages reached               │
│     - Manual close()                    │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│  5. Cleanup                             │
│     - Remove from manager               │
│     - Delete nonce cache                │
│     - Zero sensitive memory             │
└─────────────────────────────────────────┘
```

## Usage Examples

### Basic Session Creation

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/session"
)

// Create session manager
manager := session.NewManager()

// Shared secret from handshake (ECDH, HPKE, etc.)
sharedSecret := []byte{...} // 32 bytes

// Create session
sess, err := manager.CreateSession("session-123", sharedSecret)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Session created: %s\n", sess.GetID())
```

### Session with Custom Configuration

```go
import (
    "time"
    "github.com/sage-x-project/sage/pkg/agent/session"
)

manager := session.NewManager()

// Custom configuration
config := session.Config{
    MaxAge:      2 * time.Hour,    // 2-hour absolute limit
    IdleTimeout: 15 * time.Minute, // 15-minute idle timeout
    MaxMessages: 5000,             // 5000 messages max
}

sess, err := manager.CreateSessionWithConfig("session-456", sharedSecret, config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Custom session created with %v max age\n", config.MaxAge)
```

### Encrypt and Decrypt

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Encrypt message
plaintext := []byte("Hello, secure agent!")
ciphertext, err := sess.Encrypt(plaintext)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Encrypted: %x\n", ciphertext)

// Decrypt message
decrypted, err := sess.Decrypt(ciphertext)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Decrypted: %s\n", string(decrypted))
// Output: Decrypted: Hello, secure agent!
```

### Encrypt with Additional Authenticated Data (AAD)

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Message content
plaintext := []byte("Transfer 100 ETH")

// Additional authenticated data (not encrypted, but signed)
// Example: message metadata, headers, timestamps
covered := []byte(`{"from":"agent1","to":"agent2","timestamp":"2025-10-25T10:00:00Z"}`)

// Encrypt plaintext + sign covered data
ciphertext, mac, err := sess.EncryptAndSign(plaintext, covered)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Ciphertext: %x\n", ciphertext)
fmt.Printf("MAC: %x\n", mac)

// Receiver: Decrypt and verify
decrypted, err := sess.DecryptAndVerify(ciphertext, covered, mac)
if err != nil {
    log.Fatal("Decryption or verification failed:", err)
}

fmt.Printf("Decrypted: %s\n", string(decrypted))
// Both plaintext and covered data are authenticated
```

### HPKE Integration (Recommended)

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/session"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

// After HPKE handshake...
hpkeSender, _ := hpke.NewSender(recipientPublicKey)
encapsulatedKey, err := hpkeSender.Encapsulate()

// Derive exporter secret (64 bytes recommended)
exporterSecret, err := hpkeSender.ExportSecret("sage/hpke v1", 64)
if err != nil {
    log.Fatal(err)
}

// Create session from exporter secret
sess, sid, existed, err := manager.EnsureSessionFromExporterWithRole(
    exporterSecret,
    "sage/hpke v1",
    true, // initiator=true (HPKE sender)
    nil,  // use default config
)
if err != nil {
    log.Fatal(err)
}

if existed {
    fmt.Println("Reusing existing session:", sid)
} else {
    fmt.Println("Created new session:", sid)
}

// Use session for encrypted communication
ciphertext, _ := sess.Encrypt([]byte("Hello from HPKE session!"))
```

### KeyID Binding for Quick Lookup

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Create session
sess, _ := manager.CreateSession("session-789", sharedSecret)

// Bind to a key ID (e.g., agent DID or public key fingerprint)
keyID := "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
err := manager.BindKeyID(keyID, sess.GetID())
if err != nil {
    log.Fatal(err)
}

// Later: Look up session by keyID
retrievedSess, found := manager.GetSessionByKeyID(keyID)
if !found {
    log.Fatal("Session not found for keyID")
}

fmt.Printf("Found session: %s\n", retrievedSess.GetID())
```

### Replay Attack Prevention

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// KeyID from incoming message
keyID := "agent-key-123"
nonce := "unique-nonce-456"

// Check if nonce was seen before (replay detection)
if manager.nonceCache.Seen(keyID, nonce) {
    log.Fatal("Replay attack detected! Nonce already used.")
}

// Nonce is fresh, process message
sess, _ := manager.GetSessionByKeyID(keyID)
plaintext, err := sess.Decrypt(ciphertext)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message processed: %s\n", string(plaintext))
// Nonce is automatically recorded in cache
```

### Session Status Monitoring

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Get manager status
status := manager.GetStatus()

fmt.Printf("Total sessions: %d\n", status.TotalSessions)
fmt.Printf("Active sessions: %d\n", status.ActiveSessions)
fmt.Printf("Expired sessions: %d\n", status.ExpiredSessions)

// Check individual session
sess, _ := manager.GetSession("session-123")
fmt.Printf("Session ID: %s\n", sess.GetID())
fmt.Printf("Created at: %v\n", sess.GetCreatedAt())
fmt.Printf("Last used: %v\n", sess.GetLastUsedAt())
fmt.Printf("Message count: %d\n", sess.GetMessageCount())
fmt.Printf("Expired: %v\n", sess.IsExpired())
```

### Manual Session Cleanup

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Close specific session
err := sess.Close()
if err != nil {
    log.Fatal(err)
}

// Delete from manager
err = manager.DeleteSession(sess.GetID())
if err != nil {
    log.Fatal(err)
}

fmt.Println("Session cleaned up manually")

// Note: Background cleanup runs automatically every 30 seconds
// Manual cleanup only needed for immediate resource release
```

### Bidirectional Communication

```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Initiator (client)
clientSess, _, _, _ := manager.EnsureSessionFromExporterWithRole(
    exporterSecret,
    "sage/hpke v1",
    true, // initiator=true
    nil,
)

// Send message (uses C2S keys)
ciphertext, _ := clientSess.Encrypt([]byte("Hello from client"))

// -------- Network transmission --------

// Responder (server)
serverSess, _, _, _ := manager.EnsureSessionFromExporterWithRole(
    exporterSecret,
    "sage/hpke v1",
    false, // initiator=false
    nil,
)

// Receive message (uses C2S keys for decryption)
plaintext, _ := serverSess.Decrypt(ciphertext)
fmt.Printf("Server received: %s\n", string(plaintext))

// Reply (uses S2C keys)
reply, _ := serverSess.Encrypt([]byte("Hello from server"))

// -------- Network transmission --------

// Client receives reply (uses S2C keys for decryption)
replyPlain, _ := clientSess.Decrypt(reply)
fmt.Printf("Client received: %s\n", string(replyPlain))
```

## Performance

### Benchmark Results (Apple M1, Go 1.24)

| Operation | Message Size | Time | Throughput |
|-----------|-------------|------|------------|
| Session Creation | - | ~60-80 μs | ~15k/s |
| Encrypt | 64 B | ~3-5 μs | ~15 MB/s |
| Encrypt | 1 KB | ~5-8 μs | ~125 MB/s |
| Encrypt | 16 KB | ~50-80 μs | ~200 MB/s |
| Decrypt | 64 B | ~3-5 μs | ~15 MB/s |
| Decrypt | 1 KB | ~5-8 μs | ~125 MB/s |
| Decrypt | 16 KB | ~50-80 μs | ~200 MB/s |
| EncryptAndSign | 1 KB | ~10-15 μs | ~70 MB/s |
| DecryptAndVerify | 1 KB | ~10-15 μs | ~70 MB/s |
| Nonce Check | - | ~0.5-1 μs | ~1M ops/s |

### Memory Usage

| Component | Memory per Session |
|-----------|-------------------|
| SecureSession struct | ~512 bytes |
| Key material (pre-allocated) | 192 bytes |
| AEAD ciphers | ~200 bytes |
| Total | ~900 bytes/session |

**1000 active sessions**: ~900 KB memory

### Optimization Tips

1. **Reuse sessions**: Avoid creating new sessions for each message
2. **Enable session pooling**: Manager has built-in `sync.Pool`
3. **Batch messages**: Send multiple messages per session
4. **Use direction keys**: Enable concurrent read/write
5. **Monitor expiration**: Adjust `MaxAge` and `IdleTimeout` based on workload

## Security Considerations

### Key Derivation

**Best Practices:**
- ✅ Use HPKE exporter secrets (not raw ECDH output)
- ✅ Separate encryption and signing keys
- ✅ Use direction-separated keys (C2S, S2C)
- ❌ Never reuse keys across different sessions
- ❌ Never use encryption keys for signing

### Replay Attack Prevention

**Nonce Requirements:**
- ✅ Unique per message
- ✅ Unpredictable (cryptographically random)
- ✅ Checked before processing message
- ✅ TTL: 5-10 minutes (configurable)

**Example Secure Nonce:**
```go
import "crypto/rand"

nonce := make([]byte, 16)
rand.Read(nonce)
nonceStr := base64.StdEncoding.EncodeToString(nonce)
```

### Session Expiration

**Recommended Settings:**
- **Short-lived sessions** (chat, API): `MaxAge: 15 minutes`, `IdleTimeout: 5 minutes`
- **Long-lived sessions** (file transfer): `MaxAge: 1 hour`, `IdleTimeout: 10 minutes`
- **Production servers**: `MaxAge: 30 minutes`, `IdleTimeout: 5 minutes`

**Why expire sessions?**
- ✅ Limit key exposure time
- ✅ Prevent stale session accumulation
- ✅ Force re-authentication periodically
- ✅ Reduce memory usage

### Direction-Separated Keys

**Why separate keys?**
- ✅ Prevent key reuse across contexts
- ✅ Enable asymmetric security levels (e.g., client stronger than server)
- ✅ Support unidirectional rate limiting
- ✅ Simplify concurrent access (no lock contention)

### Memory Safety

**Sensitive Data Handling:**
```go
// Session automatically zeroes key material on Close()
sess.Close()

// Manual zeroing (if needed)
for i := range keyMaterial {
    keyMaterial[i] = 0
}
```

## Testing

### Unit Tests

```bash
# Run all session tests
go test ./pkg/agent/session/...

# Run with coverage
go test -cover ./pkg/agent/session/...

# Run specific tests
go test ./pkg/agent/session -run TestSessionCreation
go test ./pkg/agent/session -run TestReplayPrevention
```

### Fuzz Testing

```bash
# Run fuzzing (10 seconds)
./tools/scripts/run-fuzz.sh --time 10s --type go

# Run session-specific fuzzer
go test -fuzz=FuzzSessionEncryption -fuzztime=1m ./pkg/agent/session
```

### Performance Tests

```bash
# Run session benchmarks
go test -bench=. -benchmem ./pkg/agent/session

# Expected results:
# - Session creation: ~60-80 μs
# - Encryption (1KB): ~5-8 μs
# - Decryption (1KB): ~5-8 μs
# - Nonce check: ~0.5-1 μs
```

## Directory Structure

```
pkg/agent/session/
├── README.md                    # This file
├── manager.go                   # Session lifecycle manager
├── manager_test.go              # Manager tests
├── session.go                   # SecureSession implementation
├── session_test.go              # Session tests
├── types.go                     # Interfaces and types
├── nonce.go                     # NonceCache (replay prevention)
├── metadata.go                  # Session metadata tracking
├── metadata_test.go             # Metadata tests
└── fuzz_test.go                 # Fuzzing tests
```

## FAQ

### Q: When should I create a new session?

A: **After each handshake**:
- Initial agent connection (HPKE handshake)
- Session expiration (MaxAge, IdleTimeout, MaxMessages)
- Security breach (compromise detected)
- Manual re-authentication

**Don't create sessions for each message** - reuse existing sessions.

### Q: How do I handle session expiration?

A: **Automatic expiration** (recommended):
```go
// Manager automatically cleans up expired sessions every 30 seconds
// No action needed
```

**Manual check** (if needed):
```go
if sess.IsExpired() {
    // Re-establish handshake
    newSess, _ := performHandshake()
}
```

### Q: What's the difference between Encrypt() and EncryptAndSign()?

**Encrypt(plaintext)**:
- Encrypts plaintext with ChaCha20-Poly1305 AEAD
- Provides confidentiality + authentication
- Use for simple messages

**EncryptAndSign(plaintext, covered)**:
- Encrypts plaintext with ChaCha20-Poly1305
- **Additionally** signs `covered` data with HMAC-SHA256
- Use when you need to authenticate additional data (e.g., headers, metadata)

**Example use case:**
```go
// Message body (encrypted)
plaintext := []byte("Transfer 100 ETH")

// Message headers (not encrypted, but signed)
headers := []byte(`{"from":"agent1","to":"agent2","nonce":"123"}`)

ciphertext, mac, _ := sess.EncryptAndSign(plaintext, headers)
// Both plaintext and headers are authenticated
```

### Q: How do I prevent replay attacks?

A: **Use NonceCache** (built into Manager):
```go
// Check nonce before processing
if manager.nonceCache.Seen(keyID, nonce) {
    return fmt.Errorf("replay attack")
}

// Process message...
```

**Nonce best practices:**
- ✅ 16+ bytes of randomness
- ✅ Include in every message
- ✅ Check before decryption
- ✅ TTL: 5-10 minutes

### Q: Can I use the same session for multiple agents?

A: **No**. Each session is for a **pair** of agents:
- One initiator (client)
- One responder (server)

For multiple agents, create separate sessions per pair:
```go
sessAB, _ := manager.CreateSession("A-to-B", secretAB)
sessAC, _ := manager.CreateSession("A-to-C", secretAC)
```

### Q: How do I integrate with HPKE?

A: **Use EnsureSessionFromExporterWithRole()**:
```go
// After HPKE handshake
exporterSecret, _ := hpkeSender.ExportSecret("sage/hpke v1", 64)

sess, sid, existed, _ := manager.EnsureSessionFromExporterWithRole(
    exporterSecret,
    "sage/hpke v1",
    true, // initiator (HPKE sender)
    nil,
)
```

See [HPKE documentation](../hpke/) for complete integration guide.

## See Also

- [HPKE Package](../hpke/) - Key agreement and session bootstrap
- [Handshake Protocol](../handshake/) - Agent authentication
- [Crypto Package](../crypto/README.md) - Key management
- [Transport Layer](../transport/README.md) - Message transport

## References

- [RFC 8439](https://www.rfc-editor.org/rfc/rfc8439) - ChaCha20-Poly1305 AEAD
- [RFC 5869](https://www.rfc-editor.org/rfc/rfc5869) - HKDF
- [RFC 9180](https://www.rfc-editor.org/rfc/rfc9180) - HPKE
- [WireGuard](https://www.wireguard.com/papers/wireguard.pdf) - Noise protocol (similar design)

## License

LGPL-3.0 - See LICENSE file for details.
