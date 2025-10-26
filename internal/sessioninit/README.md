# SessionInit Package

## Overview

The `sessioninit` package provides integration between the handshake protocol and session management. It implements the `handshake.Events` interface to automatically create secure sessions when handshakes complete.

This package acts as an adapter, translating handshake lifecycle events into session management operations.

## Purpose

- **Handshake-to-Session Bridge**: Automatically creates sessions from successful handshakes
- **Ephemeral Key Management**: Manages temporary X25519 keys during handshake
- **Key ID Generation**: Creates and binds opaque key identifiers to sessions
- **Deterministic Session Creation**: Ensures both peers create identical sessions
- **Context Tracking**: Maps handshake contexts to session IDs

## Architecture

```
┌─────────────────────────────────────────────┐
│    Handshake Protocol                       │
│    - Invitation                             │
│    - Request                                │
│    - Response                               │
│    - Complete                               │
└─────────────────┬───────────────────────────┘
                  │
                  │ Events
                  ▼
┌─────────────────────────────────────────────┐
│    Creator (implements handshake.Events)    │
│    - OnInvitation()                         │
│    - OnRequest()                            │
│    - OnResponse()                           │
│    - OnComplete()                           │
│    - AskEphemeral()                         │
│    - IssueKeyID()                           │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    Session Manager                          │
│    - EnsureSessionWithParams()              │
│    - BindKeyID()                            │
└─────────────────────────────────────────────┘
```

## Core Types

### Creator

```go
type Creator struct {
    sessionMgr   *session.Manager
    mu           sync.RWMutex
    ephPrivByCtx map[string]*keys.X25519KeyPair  // Context ID -> ephemeral private key
    sidByCtx     map[string]string                // Context ID -> session ID
    exporter     crypto.KeyExporter
}
```

The `Creator` maintains:
- **sessionMgr**: Target session manager for creating sessions
- **ephPrivByCtx**: Temporary storage for ephemeral private keys during handshake
- **sidByCtx**: Mapping from handshake context to created session ID
- **exporter**: JWK exporter for public key serialization

## Handshake Events Implementation

### OnInvitation

```go
func (c *Creator) OnInvitation(ctx context.Context, ctxID string, inv handshake.InvitationMessage) error
```

Called when a handshake invitation is received. Currently a no-op, but can be used for logging or metrics.

### OnRequest

```go
func (c *Creator) OnRequest(ctx context.Context, ctxID string, req handshake.RequestMessage, senderPub crypto.PublicKey) error
```

Called when a handshake request is received. Can be used for audit logging or access control checks.

### OnResponse

```go
func (c *Creator) OnResponse(ctx context.Context, ctxID string, res handshake.ResponseMessage, senderPub crypto.PublicKey) error
```

Called when a handshake response is received. Can be used for metrics or validation.

### OnComplete

```go
func (c *Creator) OnComplete(ctx context.Context, ctxID string, comp handshake.CompleteMessage, p session.Params) error
```

**Most important event** - called when handshake completes successfully:

1. Retrieves ephemeral private key for this context
2. Derives shared secret from peer's ephemeral public key
3. Creates session with deterministic parameters
4. Cleans up ephemeral private key
5. Stores session ID for later key ID binding

### AskEphemeral

```go
func (c *Creator) AskEphemeral(ctx context.Context, ctxID string) ([]byte, json.RawMessage, error)
```

Called when handshake needs an ephemeral key pair:

1. Generates fresh X25519 key pair
2. Stores private key in `ephPrivByCtx` map
3. Returns public key in both raw bytes and JWK format
4. Public key is sent to peer during handshake

### IssueKeyID

```go
func (c *Creator) IssueKeyID(ctxID string) (string, bool)
```

Called after handshake to create an opaque key identifier:

1. Retrieves session ID created in `OnComplete`
2. Generates random base64url-encoded key ID
3. Binds key ID to session in session manager
4. Returns key ID to be sent to peer
5. Cleans up context mapping

## Usage

### Basic Setup

```go
package main

import (
    "github.com/sage-x-project/sage/internal/sessioninit"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/session"
)

func main() {
    // Create session manager
    sessionMgr := session.NewManager()

    // Create handshake event handler
    creator := sessioninit.NewCreator(sessionMgr)

    // Create handshake protocol with event handler
    hs := handshake.New(creator)

    // Now handshakes will automatically create sessions
    err := hs.Initiate(peerDID)
    if err != nil {
        log.Fatal(err)
    }

    // After handshake completes, session is automatically created
    // and can be retrieved from sessionMgr
}
```

### Full Integration Example

```go
type Agent struct {
    sessionMgr *session.Manager
    handshake  *handshake.Protocol
    creator    *sessioninit.Creator
}

func NewAgent() *Agent {
    sessionMgr := session.NewManager()
    creator := sessioninit.NewCreator(sessionMgr)
    hs := handshake.New(creator)

    return &Agent{
        sessionMgr: sessionMgr,
        handshake:  hs,
        creator:    creator,
    }
}

func (a *Agent) ConnectToPeer(peerDID string) error {
    // Initiate handshake
    ctx := context.Background()
    invitation, err := a.handshake.CreateInvitation(ctx)
    if err != nil {
        return fmt.Errorf("create invitation: %w", err)
    }

    // Send invitation to peer...
    // Peer responds...
    // Handshake completes...

    // Session is now automatically created and ready to use
    // No manual session creation needed!

    return nil
}

func (a *Agent) SendMessage(sessionID string, plaintext []byte) ([]byte, error) {
    // Session was created automatically by creator
    return a.sessionMgr.Encrypt(sessionID, plaintext)
}
```

### Custom Event Handling

Extend the Creator to add custom logic:

```go
type CustomCreator struct {
    *sessioninit.Creator
    metrics *metrics.Collector
    logger  *logger.Logger
}

func NewCustomCreator(sm *session.Manager, m *metrics.Collector, l *logger.Logger) *CustomCreator {
    return &CustomCreator{
        Creator: sessioninit.NewCreator(sm),
        metrics: m,
        logger:  l,
    }
}

func (c *CustomCreator) OnComplete(ctx context.Context, ctxID string, comp handshake.CompleteMessage, p session.Params) error {
    start := time.Now()

    // Call base implementation
    err := c.Creator.OnComplete(ctx, ctxID, comp, p)

    // Add metrics
    c.metrics.RecordSessionCreation(time.Since(start), err == nil)

    // Add logging
    if err != nil {
        c.logger.Error("Session creation failed",
            logger.String("context_id", ctxID),
            logger.Error(err),
        )
    } else {
        c.logger.Info("Session created",
            logger.String("context_id", ctxID),
            logger.String("peer_did", p.PeerDID),
        )
    }

    return err
}
```

## Session Creation Flow

### Complete Handshake-to-Session Flow

```
Client                    Creator                  Session Manager
  |                          |                            |
  | 1. Handshake starts      |                            |
  |------------------------->|                            |
  |                          |                            |
  | 2. AskEphemeral()        |                            |
  |<-------------------------|                            |
  |   (generates X25519 kp)  |                            |
  |                          |                            |
  | 3. Exchange ephemeral    |                            |
  |    public keys           |                            |
  |                          |                            |
  | 4. OnComplete()          |                            |
  |------------------------->|                            |
  |                          | 5. Derive shared secret    |
  |                          |                            |
  |                          | 6. EnsureSessionWithParams |
  |                          |--------------------------->|
  |                          |                            |
  |                          |<---------------------------|
  |                          |   (session created)        |
  |                          |                            |
  |                          | 7. Cleanup ephemeral key   |
  |                          |                            |
  | 8. IssueKeyID()          |                            |
  |------------------------->|                            |
  |                          | 9. Generate key ID         |
  |                          |                            |
  |                          | 10. BindKeyID()            |
  |                          |--------------------------->|
  |                          |                            |
  |<-------------------------|                            |
  |   (key ID returned)      |                            |
  |                          |                            |
```

### Key Points

1. **Ephemeral keys are temporary** - created in `AskEphemeral`, used in `OnComplete`, then deleted
2. **Sessions are deterministic** - both peers create identical sessions from handshake parameters
3. **Key IDs are opaque** - random identifiers that hide session details from wire protocol
4. **Thread-safe** - all operations use mutex protection for concurrent handshakes

## Design Decisions

### Why Separate Package?

- **Separation of Concerns**: Handshake logic separate from session logic
- **Testability**: Can test handshake and session independently
- **Flexibility**: Easy to swap different session management implementations
- **Clarity**: Clear boundary between protocol and state management

### Why Ephemeral Key Storage?

Handshakes require both parties to:
1. Generate ephemeral X25519 key pair
2. Exchange public keys
3. Derive shared secret

The private key must be retained between:
- `AskEphemeral()` - when public key is generated
- `OnComplete()` - when shared secret is derived

Storage is temporary and cleaned up after handshake.

### Why Deterministic Sessions?

Both peers must create identical sessions:
- Same encryption keys
- Same sequence numbers
- Same session parameters

`EnsureSessionWithParams()` uses deterministic session ID generation based on:
- Peer DIDs (sorted)
- Handshake parameters
- Shared secret

This ensures both peers can find the same session.

### Why Key ID Binding?

Key IDs provide:
- **Abstraction**: Wire protocol doesn't expose session IDs
- **Security**: Session details hidden from eavesdroppers
- **Flexibility**: Can rotate keys without changing protocol
- **Privacy**: Multiple sessions can exist without revealing structure

## Security Considerations

### 1. Ephemeral Key Lifecycle

```go
// ✅ Correct - keys cleaned up after use
func (c *Creator) OnComplete(...) error {
    // Use key
    shared, err := ephKey.DeriveSharedSecret(peerPub)

    // Clean up immediately
    delete(c.ephPrivByCtx, ctxID)

    // ...
}

// ❌ Wrong - keys leaked
func (c *Creator) OnComplete(...) error {
    shared, err := ephKey.DeriveSharedSecret(peerPub)
    // Forgot to delete ephKey!
    return nil
}
```

### 2. Context ID Uniqueness

Each handshake must have unique context ID:

```go
// ✅ Correct - unique per handshake
ctxID1 := generateUniqueID()
ctxID2 := generateUniqueID()

// ❌ Wrong - reused context ID
ctxID := "fixed-id"  // Collisions!
```

### 3. Concurrent Handshakes

The Creator is thread-safe:

```go
// ✅ Safe - concurrent handshakes
go handleHandshake1(creator, ctxID1)
go handleHandshake2(creator, ctxID2)

// All internal maps are mutex-protected
```

### 4. Key ID Randomness

```go
func randBase64URL(length int) string {
    buf := make([]byte, length)
    if _, err := rand.Read(buf); err != nil {
        // Critical error - CSPRNG failure
        panic(err)
    }
    return base64.RawURLEncoding.EncodeToString(buf)
}
```

Uses `crypto/rand` for cryptographically secure key IDs.

## Error Handling

### Missing Ephemeral Key

```go
func (c *Creator) OnComplete(...) error {
    c.mu.RLock()
    ephKey := c.ephPrivByCtx[ctxID]
    c.mu.RUnlock()

    if ephKey == nil {
        return fmt.Errorf("no ephemeral private for ctx=%s", ctxID)
    }
    // ...
}
```

Indicates:
- `AskEphemeral()` was never called
- Context ID mismatch
- Concurrent cleanup issue

### Shared Secret Derivation Failure

```go
shared, err := ephKey.DeriveSharedSecret(peerPub)
if err != nil {
    return fmt.Errorf("derive shared: %w", err)
}
```

Can occur if:
- Peer public key is invalid
- Wrong key type (not X25519)
- Corrupted key data

### Session Creation Failure

```go
_, sid, _, err := sessionMgr.EnsureSessionWithParams(p, nil)
if err != nil {
    return fmt.Errorf("ensure session: %w", err)
}
```

Possible causes:
- Invalid session parameters
- Storage failure
- Duplicate session conflict

## Testing

### Unit Test Example

```go
package sessioninit_test

import (
    "context"
    "testing"

    "github.com/sage-x-project/sage/internal/sessioninit"
    "github.com/sage-x-project/sage/pkg/agent/session"
)

func TestCreator_AskEphemeral(t *testing.T) {
    sm := session.NewManager()
    creator := sessioninit.NewCreator(sm)

    rawPub, jwkPub, err := creator.AskEphemeral(context.Background(), "ctx-1")
    if err != nil {
        t.Fatal(err)
    }

    // Verify raw public key
    if len(rawPub) != 32 {
        t.Errorf("Expected 32-byte X25519 public key, got %d bytes", len(rawPub))
    }

    // Verify JWK format
    if len(jwkPub) == 0 {
        t.Error("JWK public key is empty")
    }
}

func TestCreator_IssueKeyID(t *testing.T) {
    sm := session.NewManager()
    creator := sessioninit.NewCreator(sm)

    // Simulate completed handshake
    ctx := context.Background()
    ctxID := "ctx-1"

    // Create ephemeral key
    _, _, err := creator.AskEphemeral(ctx, ctxID)
    if err != nil {
        t.Fatal(err)
    }

    // Complete handshake (creates session)
    params := session.Params{
        PeerDID:    "did:sage:test:123",
        PeerEph:    make([]byte, 32),  // Mock peer ephemeral
        MyRole:     "client",
        // ...
    }
    err = creator.OnComplete(ctx, ctxID, handshake.CompleteMessage{}, params)
    if err != nil {
        t.Fatal(err)
    }

    // Issue key ID
    keyID, ok := creator.IssueKeyID(ctxID)
    if !ok {
        t.Fatal("Failed to issue key ID")
    }

    if len(keyID) == 0 {
        t.Error("Key ID is empty")
    }

    // Verify key ID is bound to session
    // (would need to export sessionMgr.GetSessionByKeyID for this)
}
```

## Best Practices

### 1. Single Creator Per Session Manager

```go
// ✅ Correct - one creator per manager
sessionMgr := session.NewManager()
creator := sessioninit.NewCreator(sessionMgr)

// ❌ Wrong - multiple creators (causes conflicts)
creator1 := sessioninit.NewCreator(sessionMgr)
creator2 := sessioninit.NewCreator(sessionMgr)  // Don't do this
```

### 2. Unique Context IDs

```go
// ✅ Correct - unique per handshake
import "github.com/google/uuid"

ctxID := uuid.New().String()

// ❌ Wrong - predictable or reused
ctxID := "handshake-1"  // Not unique
```

### 3. Don't Access Internal Maps Directly

```go
// ✅ Correct - use public methods
keyID, ok := creator.IssueKeyID(ctxID)

// ❌ Wrong - internal access
sessionID := creator.sidByCtx[ctxID]  // Private field!
```

### 4. Handle All Errors

```go
// ✅ Correct - check all errors
if err := creator.OnComplete(ctx, ctxID, comp, params); err != nil {
    log.Error("Handshake completion failed", logger.Error(err))
    return err
}

// ❌ Wrong - ignore errors
creator.OnComplete(ctx, ctxID, comp, params)  // Error unchecked!
```

## File Structure

```
internal/sessioninit/
├── README.md           # This file (you're creating it)
└── session_creator.go  # Creator implementation
```

## Related Packages

- `pkg/agent/handshake` - Handshake protocol that uses this package
- `pkg/agent/session` - Session manager that this package wraps
- `pkg/agent/crypto/keys` - X25519 key generation

## References

- [RFC 7748 - Elliptic Curves for Security (X25519)](https://tools.ietf.org/html/rfc7748)
- [Adapter Pattern](https://en.wikipedia.org/wiki/Adapter_pattern)
- [Observer Pattern](https://en.wikipedia.org/wiki/Observer_pattern) (event-driven design)
