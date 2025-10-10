# HTTP Transport for SAGE

This package provides HTTP/REST transport implementation for SAGE secure messaging.

## Overview

The HTTP transport enables SAGE agents to communicate over standard HTTP protocols, making it easy to integrate with existing web infrastructure, load balancers, and API gateways.

## Features

- ✅ RESTful HTTP/HTTPS communication
- ✅ JSON wire format for messages
- ✅ Custom metadata via HTTP headers
- ✅ Configurable HTTP client (timeout, TLS, etc.)
- ✅ Standard HTTP error handling
- ✅ Compatible with any HTTP server framework

## Usage

### Client (Sending Messages)

```go
import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/transport/http"
)

// Create HTTP transport
transport := http.NewHTTPTransport("https://agent.example.com")

// Use with handshake client
client := handshake.NewClient(transport, keyPair)

// Send messages
resp, err := client.Invitation(ctx, invitationMsg, targetDID)
```

### Server (Receiving Messages)

```go
import (
    "context"
    "net/http"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    httpTransport "github.com/sage-x-project/sage/pkg/agent/transport/http"
)

// Create message handler
handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Process message with handshake server
    return handshakeServer.HandleMessage(ctx, msg)
}

// Create HTTP server
server := httpTransport.NewHTTPServer(handler)

// Register with HTTP router
mux := http.NewServeMux()
mux.Handle("/messages", server.MessagesHandler())

// Start HTTP server
http.ListenAndServe(":8080", mux)
```

### Custom HTTP Client

```go
// Configure custom HTTP client
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
        MaxIdleConns:    100,
    },
}

// Create transport with custom client
transport := http.NewHTTPTransportWithClient("https://agent.example.com", httpClient)
```

## Wire Format

### Request (POST /messages)

**Headers:**
```
Content-Type: application/json
X-SAGE-DID: did:sage:ethereum:0x123...
X-SAGE-Message-ID: uuid-123-456
X-SAGE-Context-ID: ctx-789
X-SAGE-Task-ID: task-abc
X-SAGE-Meta-CustomKey: custom-value
```

**Body:**
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

### Response (200 OK)

```json
{
  "success": true,
  "message_id": "uuid-123-456",
  "task_id": "task-abc",
  "data": "base64-encoded-response-data",
  "error": ""
}
```

## Error Handling

The HTTP transport returns errors in two ways:

1. **Network/HTTP Errors**: Returned as Go errors from `Send()`
2. **Application Errors**: Returned in the `Response.Error` field

```go
resp, err := transport.Send(ctx, msg)
if err != nil {
    // Network or HTTP protocol error
    log.Printf("Transport error: %v", err)
    return
}

if !resp.Success {
    // Application-level error
    log.Printf("Message processing error: %v", resp.Error)
    return
}

// Success - process response data
processData(resp.Data)
```

## Security Considerations

### TLS Configuration

Always use HTTPS in production:

```go
transport := http.NewHTTPTransport("https://agent.example.com")
```

### Custom TLS Config

```go
tlsConfig := &tls.Config{
    MinVersion:         tls.VersionTLS13,
    InsecureSkipVerify: false,
    ClientAuth:         tls.RequireAndVerifyClientCert,
}

httpClient := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
    },
}

transport := http.NewHTTPTransportWithClient("https://agent.example.com", httpClient)
```

### Message Authentication

The HTTP transport does NOT provide message authentication - this is handled by the SAGE security layer:

- Messages are encrypted by the SAGE session layer
- Signatures are verified by the SAGE handshake layer
- DIDs are authenticated by the SAGE DID resolver

The HTTP transport simply delivers the encrypted payloads.

## Comparison with Other Transports

| Feature | HTTP | gRPC (A2A) | WebSocket |
|---------|------|------------|-----------|
| Firewall-friendly | ✅ | ⚠️ | ✅ |
| Load balancer support | ✅ | ⚠️ | ⚠️ |
| Bidirectional streaming | ❌ | ✅ | ✅ |
| Simple integration | ✅ | ❌ | ⚠️ |
| Performance | ⚠️ | ✅ | ✅ |
| REST API compatible | ✅ | ❌ | ❌ |

## Performance

### Typical Latency
- Local network: ~1-5ms
- Internet: ~50-200ms
- High latency: Consider WebSocket for persistent connections

### Throughput
- Limited by HTTP request/response overhead
- Use HTTP/2 for multiplexing
- Consider gRPC for high-throughput scenarios

### Optimization Tips

1. **Enable HTTP/2**:
```go
transport := &http.Transport{
    ForceAttemptHTTP2: true,
}
```

2. **Connection Pooling**:
```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

3. **Compression** (if payload is large):
```go
// Server-side
handler := gzipHandler(server.MessagesHandler())
```

## Testing

Run tests:
```bash
go test ./pkg/agent/transport/http/... -v
```

## Examples

See `examples/http-transport/` for complete examples:
- Simple HTTP client/server
- HTTP with TLS
- HTTP with custom middleware
- Load balanced HTTP deployment

## Related Documentation

- [Transport Interface](../README.md)
- [gRPC/A2A Transport](../a2a/README.md)
- [WebSocket Transport](../websocket/README.md)
- [SAGE Architecture](../../../../docs/ARCHITECTURE.md)
