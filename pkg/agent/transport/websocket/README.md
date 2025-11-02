# WebSocket Transport for SAGE

This package provides WebSocket transport implementation for SAGE secure messaging.

## Overview

The WebSocket transport enables persistent, bidirectional communication between SAGE agents. Unlike HTTP's request-response model, WebSocket maintains a long-lived connection for real-time message exchange.

## Features

-  Persistent bidirectional connections
-  Real-time message delivery
-  Automatic reconnection support
-  JSON wire format
-  Connection pooling
-  Configurable timeouts
-  Secure WebSocket (WSS) support

## Usage

### Client (Sending Messages)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/transport/websocket"
)

// Create WebSocket transport
transport := websocket.NewWSTransport("wss://agent.example.com/ws")
defer transport.Close()

// Use with handshake client
client := handshake.NewClient(transport, keyPair)

// Send messages over persistent connection
resp, err := client.Invitation(ctx, invitationMsg, targetDID)
```

### Server (Receiving Messages)

```go
import (
    "context"
    "net/http"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    wsTransport "github.com/sage-x-project/sage/pkg/agent/transport/websocket"
)

// Create message handler
handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Process message with handshake server
    return handshakeServer.HandleMessage(ctx, msg)
}

// Create WebSocket server
server := wsTransport.NewWSServer(handler)

// Register with HTTP router
http.Handle("/ws", server.Handler())

// Start HTTP server
http.ListenAndServe(":8080", nil)
```

### Custom Timeouts

```go
// Configure custom timeouts
transport := websocket.NewWSTransportWithTimeouts(
    "wss://agent.example.com/ws",
    30*time.Second,  // dial timeout
    60*time.Second,  // read timeout
    30*time.Second,  // write timeout
)
defer transport.Close()
```

## Wire Format

### Message (Client → Server)

```json
{
  "id": "uuid-123-456",
  "context_id": "ctx-789",
  "task_id": "task-abc",
  "payload": "base64-encoded-encrypted-payload",
  "did": "did:sage:ethereum:0x123...",
  "signature": "base64-encoded-signature",
  "metadata": {
    "custom-key": "custom-value"
  },
  "role": "user"
}
```

### Response (Server → Client)

```json
{
  "success": true,
  "message_id": "uuid-123-456",
  "task_id": "task-abc",
  "data": "base64-encoded-response-data",
  "error": ""
}
```

## Connection Management

### Automatic Connection

The WebSocket transport automatically establishes connection on first Send():

```go
transport := websocket.NewWSTransport("wss://agent.example.com/ws")
defer transport.Close()

// Connection established automatically on first send
resp, err := transport.Send(ctx, msg)
```

### Manual Connection Control

```go
transport := websocket.NewWSTransport("wss://agent.example.com/ws")

// Explicitly connect
if err := transport.Connect(ctx); err != nil {
    log.Fatal(err)
}

// Send messages
resp, err := transport.Send(ctx, msg)

// Close connection when done
transport.Close()
```

### Connection Persistence

```go
transport := websocket.NewWSTransport("wss://agent.example.com/ws")
defer transport.Close()

// Send multiple messages on same connection
for i := 0; i < 10; i++ {
    resp, err := transport.Send(ctx, msg)
    if err != nil {
        log.Printf("Message %d failed: %v", i, err)
    }
}
// Connection automatically reused
```

## Error Handling

### Network Errors

```go
resp, err := transport.Send(ctx, msg)
if err != nil {
    // Connection or network error
    log.Printf("Transport error: %v", err)

    // Transport will auto-reconnect on next send
    return
}
```

### Application Errors

```go
resp, err := transport.Send(ctx, msg)
if err != nil {
    log.Fatal(err)
}

if !resp.Success {
    // Application-level error
    log.Printf("Processing error: %v", resp.Error)
}
```

### Timeout Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := transport.Send(ctx, msg)
if err == context.DeadlineExceeded {
    log.Println("Request timed out")
}
```

## Security Considerations

### Use WSS (Secure WebSocket)

Always use `wss://` in production:

```go
// Production
transport := websocket.NewWSTransport("wss://agent.example.com/ws")

// Development only
transport := websocket.NewWSTransport("ws://localhost:8080/ws")
```

### Origin Checking

The server should implement proper origin checking in production:

```go
server := websocket.NewWSServer(handler)

// Configure upgrader for production
server.upgrader.CheckOrigin = func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    // Implement your origin validation
    return isAllowedOrigin(origin)
}
```

### Message Security

Like other transports, WebSocket does NOT provide message security:

- Messages are encrypted by SAGE session layer
- Signatures are verified by SAGE handshake layer
- DIDs are authenticated by SAGE DID resolver

WebSocket only provides transport-level security (WSS/TLS).

## Comparison with Other Transports

| Feature | WebSocket | HTTP | gRPC |
|---------|-----------|------|------|
| Persistent connection |  |  |  |
| Bidirectional |  |  |  |
| Real-time |  |  |  |
| Firewall-friendly |  |  |  |
| Load balancer support |  |  |  |
| Simple integration |  |  |  |
| Connection overhead | Low | High | Low |
| Browser support |  |  |  |

### When to Use WebSocket

**Use WebSocket when:**
- You need persistent connections
- You want real-time bidirectional communication
- You have frequent small messages
- You need server-initiated messages
- You want low connection overhead

**Use HTTP when:**
- You need firewall-friendly communication
- You want load balancer support
- You have infrequent request-response patterns
- You need REST API compatibility

**Use gRPC when:**
- You need maximum performance
- You want advanced streaming features
- You have low-latency requirements
- Browser support is not required

## Performance

### Connection Overhead

```
WebSocket: Connect once, send many
- First message: ~50-100ms (connection setup)
- Subsequent: ~1-10ms (no connection overhead)

HTTP: Connect per request
- Every message: ~20-50ms (new connection each time)
```

### Throughput

- **Single connection**: 1000-10000 msg/s
- **Multiple connections**: Scales linearly
- **Latency**: <10ms for small messages

### Best Practices

1. **Reuse Connections**:
```go
// Good: Reuse transport
transport := websocket.NewWSTransport(url)
defer transport.Close()
for _, msg := range messages {
    transport.Send(ctx, msg)
}

// Bad: New transport per message
for _, msg := range messages {
    transport := websocket.NewWSTransport(url)
    transport.Send(ctx, msg)
    transport.Close()
}
```

2. **Handle Disconnections**:
```go
for {
    resp, err := transport.Send(ctx, msg)
    if err != nil {
        log.Printf("Send failed: %v", err)
        // Transport will auto-reconnect on retry
        time.Sleep(time.Second)
        continue
    }
    break
}
```

3. **Set Appropriate Timeouts**:
```go
transport := websocket.NewWSTransportWithTimeouts(
    url,
    30*time.Second,  // Connect timeout
    60*time.Second,  // Read timeout (idle connection)
    10*time.Second,  // Write timeout
)
```

## Server Management

### Connection Tracking

```go
server := websocket.NewWSServer(handler)

// Get active connection count
count := server.GetConnectionCount()
log.Printf("Active connections: %d", count)
```

### Graceful Shutdown

```go
server := websocket.NewWSServer(handler)

// ... run server ...

// Graceful shutdown
if err := server.Close(); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

## Testing

Run tests:
```bash
go test ./pkg/agent/transport/websocket/... -v
```

## Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket) v1.5.3+

## Examples

See `examples/websocket-transport/` for complete examples:
- Simple WebSocket client/server
- WebSocket with reconnection
- WebSocket with load testing
- WebSocket with monitoring

## Related Documentation

- [Transport Interface](../README.md)
- [HTTP Transport](../http/README.md)
- [gRPC/A2A Transport](../a2a/README.md)
- [SAGE Architecture](../../../../docs/ARCHITECTURE.md)

## License

LGPL-3.0 - See LICENSE file for details.
