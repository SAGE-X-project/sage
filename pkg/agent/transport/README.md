# SAGE Transport Layer

The `transport` package provides a transport-agnostic abstraction layer for SAGE's secure messaging infrastructure. This allows SAGE security protocols (handshake, HPKE, session management) to remain independent of specific transport implementations.

## Overview

SAGE's security layer needs to send and receive encrypted messages between agents, but should not be tightly coupled to any particular transport protocol (gRPC, HTTP, WebSocket, etc.). The transport package solves this by defining a simple, clean interface that any transport can implement.

### Key Benefits

- **Transport Independence**: Security logic doesn't depend on gRPC, HTTP, or any specific protocol
- **Easy Testing**: Use `MockTransport` for unit tests without network infrastructure
- **Flexibility**: Swap transport implementations without changing security code
- **Protocol Agnostic**: Support multiple transports simultaneously (A2A, HTTP, custom protocols)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SAGE Security Layer (handshake, hpke, session)         │
│  - Encryption/Decryption                                 │
│  - Signature verification                                │
│  - Session management                                    │
└────────────────────┬────────────────────────────────────┘
                     │ depends on
                     ▼
┌─────────────────────────────────────────────────────────┐
│  transport.MessageTransport interface                    │
│  - Send(ctx, SecureMessage) → Response                  │
└────────────────────┬────────────────────────────────────┘
                     │ implemented by
          ┌──────────┴──────────┬──────────────────┬──────────┐
          ▼                     ▼                  ▼          ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  ┌──────────────┐
│ A2ATransport     │  │ MockTransport    │  │ HTTPTransport│  │ WSTransport  │
│ (gRPC/A2A)       │  │ (unit tests)     │  │ (HTTP/REST)  │  │ (WebSocket)  │
└──────────────────┘  └──────────────────┘  └──────────────┘  └──────────────┘
```

## Core Interfaces

### MessageTransport

The main transport interface that all implementations must satisfy:

```go
type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}
```

**Responsibilities:**
- Convert `SecureMessage` to transport-specific format
- Handle network transmission
- Return standardized `Response`

### SecureMessage

The transport-agnostic message format prepared by SAGE security layer:

```go
type SecureMessage struct {
    ID        string            // Unique message ID (UUID)
    ContextID string            // Conversation context ID
    TaskID    string            // Task identifier
    Payload   []byte            // Encrypted message content
    DID       string            // Sender DID
    Signature []byte            // Message signature
    Metadata  map[string]string // Custom headers
    Role      string            // "user" or "agent"
}
```

**Note:** The payload is already encrypted by SAGE security layer. Transport implementations only handle transmission.

### Response

Standardized response format:

```go
type Response struct {
    Success   bool   // Delivery success
    MessageID string // Echo of message ID
    TaskID    string // Echo of task ID
    Data      []byte // Response payload
    Error     error  // Transport error (if any)
}
```

## Available Transports

### HTTP/HTTPS (REST)
- **Status:** ✅ Available
- **Package:** `github.com/sage-x-project/sage/pkg/agent/transport/http`
- **Use Case:** Web-friendly, firewall-friendly, load balancer support
- **Documentation:** [HTTP Transport README](./http/README.md)

### gRPC (A2A Protocol)
- **Status:** 🚧 Planned
- **Package:** `github.com/sage-x-project/sage/pkg/agent/transport/a2a` (not yet implemented)
- **Use Case:** High-performance agent-to-agent communication
- **Documentation:** A2A Protocol specification (coming soon)

### WebSocket
- **Status:** ✅ Available
- **Package:** `github.com/sage-x-project/sage/pkg/agent/transport/websocket`
- **Use Case:** Bidirectional streaming, persistent connections, real-time communication
- **Documentation:** [WebSocket Transport README](./websocket/README.md)

### Mock (Testing)
- **Status:** ✅ Available
- **Package:** `github.com/sage-x-project/sage/pkg/agent/transport`
- **Use Case:** Unit testing without network

## Transport Selector

The transport selector allows automatic selection of transport based on URL scheme:

```go
import "github.com/sage-x-project/sage/pkg/agent/transport"
import _ "github.com/sage-x-project/sage/pkg/agent/transport/http" // Auto-register HTTP

// Automatic selection based on URL
transport, err := transport.SelectByURL("https://agent.example.com")
if err != nil {
    log.Fatal(err)
}

// Use with any SAGE component
client := handshake.NewClient(transport, keyPair)
```

Supported URL schemes:
- `http://` → HTTP transport
- `https://` → HTTPS transport (same as HTTP with TLS)
- `ws://` → WebSocket transport
- `wss://` → WebSocket Secure

**Note:** gRPC transport (`grpc://`) is planned but not yet implemented.

## Usage Examples

### Unit Testing with MockTransport

For unit tests, use `MockTransport` to avoid network infrastructure:

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestHandshake(t *testing.T) {
    // Create mock transport
    mockTransport := &transport.MockTransport{}

    // Create server
    server := handshake.NewServer(serverKeyPair, events, resolver, nil, 0, nil)

    // Route messages to server
    mockTransport.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
        return server.HandleMessage(ctx, msg)
    }

    // Create client with mock transport
    client := handshake.NewClient(mockTransport, clientKeyPair)

    // Test without network
    resp, err := client.Invitation(ctx, invMsg, did)
    require.NoError(t, err)
}
```

**Key Points:**
- No gRPC, no network, no ports
- Direct method calls through `SendFunc`
- Fast, deterministic tests
- Easy to simulate errors

### Production with HTTP Transport

For HTTP/REST communication:

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/transport/http"
)

func main() {
    // Create HTTP transport
    transport := http.NewHTTPTransport("https://agent.example.com")

    // Use with handshake client
    client := handshake.NewClient(transport, keyPair)

    // Send invitation
    resp, err := client.Invitation(ctx, invMsg, targetDID)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: %+v", resp)
}
```

**Server-side HTTP:**

```go
import (
    "net/http"
    httpTransport "github.com/sage-x-project/sage/pkg/agent/transport/http"
)

func main() {
    // Create message handler
    handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
        // Process with handshake server
        return handshakeServer.HandleMessage(ctx, msg)
    }

    // Create HTTP server
    server := httpTransport.NewHTTPServer(handler)

    // Register endpoint
    mux := http.NewServeMux()
    mux.Handle("/messages", server.MessagesHandler())

    // Start server
    http.ListenAndServe(":8080", mux)
}
```

### Production with gRPC (A2A) - Coming Soon

gRPC transport for high-performance agent-to-agent communication is planned for a future release. The design will provide:
- High throughput with HTTP/2
- Bidirectional streaming
- Low-latency communication
- Full backward compatibility

Stay tuned for updates!

## Implementing Custom Transports

To implement a custom transport (HTTP, WebSocket, etc.):

### 1. Implement MessageTransport Interface

```go
package mytransport

import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

type MyCustomTransport struct {
    endpoint string
    client   *http.Client
}

func NewMyCustomTransport(endpoint string) *MyCustomTransport {
    return &MyCustomTransport{
        endpoint: endpoint,
        client:   &http.Client{},
    }
}

func (t *MyCustomTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // 1. Convert SecureMessage to your protocol format
    request := convertToMyFormat(msg)

    // 2. Send over network
    httpResp, err := t.client.Do(request)
    if err != nil {
        return &transport.Response{
            Success: false,
            Error:   err,
        }, err
    }
    defer httpResp.Body.Close()

    // 3. Convert response to transport.Response
    data, _ := io.ReadAll(httpResp.Body)
    return &transport.Response{
        Success:   httpResp.StatusCode == 200,
        MessageID: msg.ID,
        TaskID:    msg.TaskID,
        Data:      data,
    }, nil
}
```

### 2. Test Your Implementation

```go
func TestMyTransport(t *testing.T) {
    transport := NewMyCustomTransport("http://localhost:8080")

    msg := &transport.SecureMessage{
        ID:        uuid.NewString(),
        ContextID: "test-ctx",
        TaskID:    "test-task",
        Payload:   []byte("encrypted-data"),
        DID:       "did:sage:ethereum:test",
        Signature: []byte("signature"),
        Role:      "user",
    }

    resp, err := transport.Send(context.Background(), msg)
    require.NoError(t, err)
    require.True(t, resp.Success)
}
```

### 3. Use in SAGE Components

```go
func main() {
    myTransport := mytransport.NewMyCustomTransport("http://server:8080")

    client := handshake.NewClient(myTransport, keyPair)
    // ... use normally
}
```

## Directory Structure

```
pkg/agent/transport/
├── README.md           # This file
├── interface.go        # Core interfaces (MessageTransport, SecureMessage, Response)
├── interface_test.go   # Interface compliance tests
├── mock.go            # MockTransport for unit testing
├── selector.go        # Transport selector (auto-select by URL)
├── selector_test.go   # Selector tests
├── http/              # HTTP/REST transport
│   ├── README.md      # HTTP transport documentation
│   ├── client.go      # HTTP client transport
│   ├── server.go      # HTTP server handler
│   ├── register.go    # Auto-registration with selector
│   └── http_test.go   # HTTP transport tests
└── websocket/         # WebSocket transport
    ├── README.md      # WebSocket transport documentation
    ├── client.go      # WebSocket client transport
    ├── server.go      # WebSocket server handler
    ├── register.go    # Auto-registration with selector
    └── websocket_test.go # WebSocket transport tests
```

## Design Principles

### 1. Separation of Concerns

**Security Layer** (handshake, hpke, session):
- Encryption/decryption
- Signature generation/verification
- Session management
- DID operations

**Transport Layer** (this package):
- Network transmission
- Protocol conversion
- Connection management
- Error handling

### 2. Protocol Independence

The security layer never imports:
- `google.golang.org/grpc`
- `net/http`
- Any transport-specific packages

It only imports `github.com/sage-x-project/sage/pkg/agent/transport`.

### 3. Testability First

All SAGE components that need transport:
- Accept `transport.MessageTransport` interface
- Can be tested with `MockTransport`
- Don't require network for unit tests

## Design Benefits

The transport abstraction layer provides:

**Protocol Independence:**
```go
import "github.com/sage-x-project/sage/pkg/agent/transport"

// Security components accept transport interface
func NewServer(..., t transport.MessageTransport) *Server {
    // Works with any transport implementation
}

// Handle messages without knowing the underlying protocol
func (s *Server) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Pure security logic, no protocol coupling
}
```

**Flexible Deployment:**
- Swap HTTP for WebSocket without code changes
- Test with MockTransport
- Add new transports without modifying core logic

## FAQ

### Q: Why not use gRPC directly?

A: Direct gRPC usage creates tight coupling. The transport abstraction allows:
- Testing without gRPC infrastructure
- Supporting multiple protocols (HTTP, WebSocket, custom)
- Easier mocking and simulation
- Protocol evolution without changing security code

### Q: Does this add performance overhead?

A: Minimal. The interface is a thin abstraction:
- No extra serialization (payload is already encrypted bytes)
- No data copying (pass by reference)
- Inline function calls (Go compiler optimization)

### Q: Can I use multiple transports simultaneously?

A: Yes! Create multiple transport instances:

```go
import _ "github.com/sage-x-project/sage/pkg/agent/transport/http"
import _ "github.com/sage-x-project/sage/pkg/agent/transport/websocket"

// Use selector for different endpoints
httpTransport, _ := transport.SelectByURL("https://agent1.example.com")
wsTransport, _ := transport.SelectByURL("wss://agent2.example.com/ws")

// Use different transports for different purposes
client1 := handshake.NewClient(httpTransport, kp1)
client2 := handshake.NewClient(wsTransport, kp2)
```

### Q: How do I choose between HTTP and WebSocket transport?

A: Consider your use case:

**Use HTTP when:**
- You need firewall-friendly communication (port 80/443)
- You want load balancer support
- You need REST API compatibility
- You're integrating with web infrastructure
- You have request-response patterns

**Use WebSocket when:**
- You need persistent connections
- You want bidirectional real-time communication
- You have frequent small messages
- You need server-initiated messages
- You want low connection overhead
- You have streaming data patterns

## See Also

- [HPKE Documentation](../hpke/README.md)
- [Handshake Protocol](../handshake/README.md)
- [Session Management](../session/README.md)
- [Transport Refactoring Architecture](../../../docs/architecture/TRANSPORT_REFACTORING.md)

## License

LGPL-3.0 - See LICENSE file for details.
