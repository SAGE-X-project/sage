# ADR-001: Transport Layer Abstraction

**Status:** Accepted

**Date:** 2024-10-26

**Decision Makers:** SAGE Core Team

**Technical Story:** [Transport Layer Implementation](https://github.com/sage-x-project/sage/tree/main/pkg/agent/transport)

---

## Context

SAGE (Secure Agent Guarantee Engine) provides secure, end-to-end encrypted communication between AI agents. The core security protocols (HPKE encryption, Ed25519 signatures, DID-based authentication, session management) need to transmit messages over a network, but should remain independent of the specific transport mechanism used.

### Problem Statement

We faced several challenges that required careful consideration:

1. **Protocol Diversity**: Different deployment scenarios require different transport protocols:
   - Web applications prefer HTTP/HTTPS for firewall compatibility
   - Real-time applications need WebSocket for bidirectional streaming
   - High-performance agent networks may use gRPC/A2A for efficiency
   - Testing environments need mock transports without network overhead

2. **Coupling Risk**: Tightly coupling security logic to a specific transport (e.g., gRPC) would:
   - Make it difficult to support additional transports later
   - Complicate testing (requiring full network infrastructure for unit tests)
   - Limit deployment flexibility
   - Create vendor lock-in to specific transport implementations

3. **Separation of Concerns**: Security operations (encryption, signing, verification) are fundamentally different from transport operations (serialization, network transmission, protocol handling).

4. **Future Extensibility**: We anticipated needing support for:
   - A2A (Agent-to-Agent) Protocol via gRPC
   - Custom transport protocols for specialized deployments
   - Multiple transports running simultaneously
   - Transport-specific optimizations (e.g., connection pooling, compression)

### Requirements

- Security layer must be transport-agnostic
- Easy to add new transport implementations
- Testable without network infrastructure
- Minimal performance overhead
- Support multiple concurrent transports
- Clear separation between security and transport concerns

---

## Decision

We decided to implement a **Transport Layer Abstraction** using Go interfaces, providing a clean separation between SAGE's security protocols and the underlying network transport.

### Core Design

#### 1. MessageTransport Interface

Define a simple, focused interface that all transports must implement:

```go
type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}
```

**Rationale:**
- Single-method interface follows Go best practices
- Context support for timeouts and cancellation
- Accepts transport-agnostic `SecureMessage`
- Returns standardized `Response`

#### 2. Transport-Agnostic Message Format

```go
type SecureMessage struct {
    ID        string            // Unique message ID (UUID)
    ContextID string            // Conversation context ID
    TaskID    string            // Task identifier
    Payload   []byte            // Already encrypted by security layer
    DID       string            // Sender DID
    Signature []byte            // Message signature
    Metadata  map[string]string // Custom headers
    Role      string            // "user" or "agent"
}
```

**Key Point:** The payload arrives already encrypted. Transports only handle transmission, not encryption.

#### 3. Implementation Strategy

```
SAGE Security Layer (handshake, hpke, session)
         ↓ depends on
MessageTransport interface
         ↓ implemented by
    ┌────┴────┬────────┬──────────┐
    ↓         ↓        ↓          ↓
HTTP    WebSocket   gRPC/A2A   Mock
```

#### 4. Transport Selector

Implement a transport selector that chooses the appropriate transport based on scheme:

```go
- http://  or https://  → HTTP Transport
- ws://    or wss://    → WebSocket Transport
- grpc://  or a2a://    → gRPC/A2A Transport (planned)
```

### Implementation Components

1. **Interface Definition** (`pkg/agent/transport/interface.go`)
   - `MessageTransport` interface
   - `SecureMessage` struct
   - `Response` struct

2. **HTTP Transport** (`pkg/agent/transport/http/`)
   - REST API implementation
   - POST `/v1/a2a:sendMessage`
   - Production-ready with retries and timeouts

3. **WebSocket Transport** (`pkg/agent/transport/websocket/`)
   - Bidirectional streaming
   - Connection management
   - Reconnection logic

4. **Mock Transport** (`pkg/agent/transport/mock/`)
   - In-memory message queue
   - No network overhead
   - Configurable success/failure for testing

5. **Transport Selector** (`pkg/agent/transport/selector.go`)
   - URL-based transport selection
   - Factory pattern for transport creation
   - Support for custom transports via registration

---

## Consequences

### Positive

1. **Transport Independence**
   - Security code has zero knowledge of HTTP, WebSocket, or gRPC specifics
   - Easy to swap transports without touching security logic
   - Each transport can be developed, tested, and optimized independently

2. **Testing Excellence**
   - Unit tests use `MockTransport` with no network infrastructure
   - Can simulate network failures, timeouts, and edge cases
   - Fast test execution (no actual network I/O)

3. **Flexibility**
   - Support multiple transports simultaneously
   - Choose transport per-connection based on requirements
   - Easy to add new transports (e.g., QUIC, custom protocols)

4. **Clean Architecture**
   - Clear separation of concerns
   - Each component has a single responsibility
   - Dependencies flow in one direction (security → transport)

5. **Future-Proof**
   - Can add A2A/gRPC without refactoring security layer
   - Transport-specific optimizations don't affect security code
   - Easy to deprecate old transports

### Negative

1. **Additional Abstraction Layer**
   - Extra interface adds slight cognitive overhead
   - One more layer to understand for new contributors
   - Requires understanding of Go interfaces

2. **Performance Overhead**
   - Interface dispatch has minimal cost (~1-2 nanoseconds)
   - Message conversion between formats adds slight overhead
   - Trade-off accepted for architectural benefits

3. **Maintenance Burden**
   - Each new transport requires full implementation
   - Interface changes affect all transport implementations
   - Need to maintain consistency across transports

4. **Initial Complexity**
   - More complex than direct gRPC/HTTP usage
   - Requires up-front design investment
   - Learning curve for contributors

### Trade-offs Accepted

- **Abstraction Overhead**: We accept the minimal performance cost (~microseconds per message) for the significant architectural benefits
- **Development Time**: Initial implementation took longer, but pays off in maintainability and testability
- **Complexity**: Added complexity is well-contained and justified by flexibility gains

---

## Alternatives Considered

### Alternative 1: Direct gRPC Usage

**Approach:** Implement all SAGE logic directly with gRPC.

**Pros:**
- Faster initial development
- No abstraction overhead
- Well-documented and supported

**Cons:**
- Tightly couples security to gRPC
- Difficult to support HTTP/WebSocket later
- Testing requires full gRPC infrastructure
- No flexibility for custom transports

**Why Rejected:** Too much coupling; limits future flexibility.

---

### Alternative 2: HTTP-Only

**Approach:** Support only HTTP/REST, no abstraction layer.

**Pros:**
- Simplest implementation
- Universally supported
- Firewall-friendly

**Cons:**
- No support for efficient bidirectional streaming
- Can't optimize for high-performance scenarios
- Locks us into REST paradigm
- Poor fit for real-time agent communication

**Why Rejected:** Insufficient for future A2A protocol requirements.

---

### Alternative 3: Protocol Buffers + Multi-Transport

**Approach:** Use protobuf for messages, implement each transport separately.

**Pros:**
- Efficient serialization
- Language-agnostic
- Well-defined schema

**Cons:**
- Heavier dependency
- Protobuf overhead for small messages
- Each security function would need transport knowledge
- More complex than our needs

**Why Rejected:** Overkill for current requirements; interface abstraction is simpler.

---

### Alternative 4: Message Broker (RabbitMQ/Kafka)

**Approach:** Use message broker for all agent communication.

**Pros:**
- Handles routing, persistence, retries
- Proven scalability
- Decouples senders and receivers

**Cons:**
- Heavy infrastructure dependency
- Latency overhead (broker intermediary)
- Operational complexity
- Overkill for direct agent-to-agent communication

**Why Rejected:** Too heavy; most deployments don't need broker semantics.

---

### Alternative 5: Per-Feature Transport Implementation

**Approach:** Each security feature (handshake, message send, etc.) implements its own transport.

**Pros:**
- Maximum flexibility per feature
- Can optimize each independently

**Cons:**
- Massive code duplication
- Inconsistent behavior across features
- Maintenance nightmare
- Violates DRY principle

**Why Rejected:** Unmaintainable; violates separation of concerns.

---

## Implementation Notes

### Adding a New Transport

To add a new transport (e.g., QUIC):

1. Implement `MessageTransport` interface:
   ```go
   type QUICTransport struct {
       // QUIC-specific fields
   }

   func (t *QUICTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
       // Convert SecureMessage to QUIC format
       // Transmit over QUIC connection
       // Return standardized Response
   }
   ```

2. Register with selector:
   ```go
   transport.RegisterScheme("quic", NewQUICTransport)
   ```

3. Add tests:
   ```go
   func TestQUICTransport(t *testing.T) {
       // Test implementation
   }
   ```

No changes to security layer required!

### Testing Strategy

```go
// Unit tests: Use MockTransport
mockTransport := transport.NewMock()
mockTransport.SetResponse(expectedResponse)
handshake := NewHandshake(mockTransport)

// Integration tests: Use real transport
httpTransport := http.NewTransport("http://localhost:8080")
handshake := NewHandshake(httpTransport)
```

---

## Related Documents

- [Transport Layer README](../../pkg/agent/transport/README.md)
- [HTTP Transport Documentation](../../pkg/agent/transport/http/README.md)
- [WebSocket Transport Documentation](../../pkg/agent/transport/websocket/README.md)
- [SAGE A2A Implementation Guide](../SAGE_A2A_GO_IMPLEMENTATION_GUIDE.md)

---

## References

- [Go Interfaces Best Practices](https://go.dev/doc/effective_go#interfaces)
- [Dependency Injection in Go](https://go.dev/blog/dependency-injection)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Interface Segregation Principle](https://en.wikipedia.org/wiki/Interface_segregation_principle)

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2024-10-26 | 1.0 | SAGE Team | Initial ADR |

---

## Approval

This ADR has been reviewed and accepted by the SAGE core team. Implementation is complete and in production use.

**Acceptance Criteria Met:**
- ✅ HTTP and WebSocket transports implemented
- ✅ Mock transport for testing
- ✅ All security tests pass with abstraction
- ✅ No performance regression
- ✅ Documentation complete
