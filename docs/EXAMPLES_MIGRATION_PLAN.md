# Examples Directory Migration Plan

**Date:** January 2025
**Status:** Analysis Complete - No Migration Required

## Summary

After analyzing the `examples/` directory, we determined that **no migration is required** for existing examples. The current examples do not use the handshake or HPKE protocols directly, and therefore are not affected by the transport layer refactoring.

## Analysis

### Current Examples Structure

```
examples/
├── README.md
├── config.yaml
└── mcp-integration/
    ├── basic-demo/          # MCP calculator tool demo
    ├── basic-tool/          # MCP tool implementation
    ├── client/              # MCP client
    ├── multi-agent/         # Multi-agent scenarios
    ├── performance-benchmark/ # Performance testing
    ├── simple-standalone/   # Standalone MCP server
    ├── vulnerable-vs-secure/ # Security comparison demos
    │   ├── attacker/
    │   ├── secure-chat/
    │   └── vulnerable-chat/
    ├── QUICKSTART.md
    ├── README.md
    └── test_compile.sh
```

### Dependency Analysis

Checked for transport-related dependencies:

```bash
# Check for A2A usage
$ grep -r "import.*a2a" examples/
# Result: No matches

# Check for handshake usage
$ grep -r "import.*handshake" examples/
# Result: No matches

# Check for HPKE usage
$ grep -r "import.*hpke" examples/
# Result: No matches
```

**Conclusion:** Examples use MCP (Model Context Protocol) and SAGE's crypto utilities (RFC9421 signatures), but do not use the transport layer.

### Example Code Pattern

Typical example structure (from `basic-demo/main.go`):

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

type Calculator struct {
    trustedAgents map[string]ed25519.PublicKey
}

func (c *Calculator) VerifyRequest(r *http.Request) error {
    // Uses RFC9421 signature verification
    verifier := rfc9421.NewHTTPVerifier()
    return verifier.VerifyRequest(r, publicKey, nil)
}
```

**Key Points:**
- Examples are HTTP-based MCP servers
- Use SAGE's signature verification (RFC9421)
- Do NOT use transport abstraction
- Do NOT use handshake or HPKE protocols

## Migration Status

| Example | Uses Transport? | Migration Needed? | Status |
|---------|----------------|-------------------|--------|
| basic-demo | No | No |  No change required |
| basic-tool | No | No |  No change required |
| client | No | No |  No change required |
| simple-standalone | No | No |  No change required |
| vulnerable-vs-secure | No | No |  No change required |
| multi-agent | No | No |  No change required |
| performance-benchmark | No | No |  No change required |

**Total:** 7 examples, 0 migrations needed

## Integration Tests

The integration tests in `test/integration/tests/session/` **are** affected by the transport refactoring, but they have already been updated:

### Handshake Integration Tests

**Location:** `test/integration/tests/session/handshake/`

**Changes Applied:**
- Server uses `a2a.NewA2AServerAdapter(hpkeServer)`
- Client uses `a2a.NewA2ATransport(conn)`
- Wire protocol unchanged (still A2A/gRPC)
- Build tags: `//go:build integration && a2a`

**Status:**  Already migrated (Phase 2)

### HPKE Integration Tests

**Location:** `test/integration/tests/session/hpke/`

**Changes Applied:**
- Server uses `a2a.NewA2AServerAdapter(hpkeServer)`
- Client uses `a2a.NewA2ATransport(conn)`
- Wire protocol unchanged (still A2A/gRPC)
- Build tags: `//go:build integration && a2a`

**Status:**  Already migrated (Phase 2)

### Verification

```bash
# Build integration tests
$ go build -tags=a2a,integration -o /tmp/hpke-server ./test/integration/tests/session/hpke/server
$ go build -tags=a2a,integration -o /tmp/hpke-client ./test/integration/tests/session/hpke/client

# Result: Build successful
```

## Future Examples Plan

When creating new examples that DO use the transport layer, follow these guidelines:

### 1. Use Transport Abstraction

**Good Example:**

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/transport/a2a"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

func main() {
    // Connect via A2A
    conn, _ := grpc.Dial(addr, opts...)
    transport := a2a.NewA2ATransport(conn)

    // Use HPKE client with transport
    client := hpke.NewClient(transport, resolver, keyPair, did, infoBuilder, sessMgr)

    // Initialize secure session
    kid, err := client.Initialize(ctx, contextID, clientDID, serverDID)
    // ...
}
```

**Bad Example:**

```go
// DON'T import A2A directly
import a2apb "github.com/a2aproject/a2a/grpc"

// DON'T use gRPC directly in security code
func main() {
    conn, _ := grpc.Dial(addr, opts...)
    client := a2apb.NewA2AServiceClient(conn)
    // ... directly calling A2A methods
}
```

### 2. Example Structure

Recommended structure for transport-using examples:

```
examples/
└── transport-examples/
    ├── README.md                  # Overview
    ├── a2a-client-server/         # A2A transport example
    │   ├── server/main.go
    │   └── client/main.go
    ├── http-client-server/        # Future: HTTP transport
    │   ├── server/main.go
    │   └── client/main.go
    └── multi-protocol/            # Future: Multiple transports
        └── main.go
```

### 3. Documentation Template

Each new transport example should include:

```markdown
# Example: [Name]

## Overview
Brief description of what this example demonstrates.

## Prerequisites
- Go 1.22+
- Dependencies...

## Architecture
```
┌─────────────┐      transport      ┌─────────────┐
│   Client    │ ───────────────────> │   Server    │
│             │   (A2A/HTTP/etc)     │             │
└─────────────┘                      └─────────────┘
```

## Running

```bash
# Terminal 1: Start server
$ go run server/main.go

# Terminal 2: Run client
$ go run client/main.go
```

## Key Concepts
- Transport abstraction
- Security layer separation
- ...
```

## Recommendations

### For MCP Examples (Current)

**No changes needed.** Current examples:
-  Focus on MCP protocol integration
-  Use SAGE signature verification
-  Don't need transport layer
-  Work as standalone HTTP servers

### For New Transport Examples (Future)

**Create separate directory:** `examples/transport-examples/`

**Include:**
1. A2A client-server example (show adapter usage)
2. HTTP transport example (when implemented)
3. Multi-protocol example (demonstrate flexibility)

**Benefits:**
- Clear separation between MCP examples and transport examples
- Easy to find transport-specific examples
- Demonstrate best practices for transport usage

### For Documentation

**Update these docs:**
1. `examples/README.md` - Add section on transport examples
2. `examples/mcp-integration/README.md` - Clarify MCP focus
3. Create `examples/transport-examples/README.md` - New transport examples guide

## Testing Strategy

### Current Examples (MCP)

```bash
# Existing test script works as-is
$ ./examples/mcp-integration/test_compile.sh
```

**Status:**  No changes needed

### Integration Tests

```bash
# A2A integration tests
$ go test -tags=a2a,integration ./test/integration/tests/session/...
```

**Status:**  Already working with A2A adapter

### Future Transport Examples

```bash
# Add to test script
$ go build ./examples/transport-examples/a2a-client-server/server
$ go build ./examples/transport-examples/a2a-client-server/client
```

## Migration Checklist

- [x] Analyze existing examples
- [x] Verify no transport dependencies
- [x] Confirm integration tests work
- [x] Document current state
- [ ] Create `examples/transport-examples/` directory (future)
- [ ] Add A2A example (future)
- [ ] Add HTTP example (future, after HTTP transport implemented)
- [ ] Update example documentation (future)

## Conclusion

### Current Status:  Complete

- **MCP Examples:** No migration needed (don't use transport)
- **Integration Tests:** Already migrated (use A2A adapter)
- **Documentation:** This plan documented

### Future Work: 📋 Planned

When adding transport-focused examples:
1. Create `examples/transport-examples/` directory
2. Add A2A client-server example
3. Add HTTP transport example (after Phase 4)
4. Document best practices

### Impact: ⚪ None

The transport refactoring does not affect existing examples. All examples continue to work as before.

---

## References

- [Transport Package README](../pkg/agent/transport/README.md)
- [Transport Refactoring Documentation](./TRANSPORT_REFACTORING.md)
- [MCP Integration Examples](../examples/mcp-integration/README.md)
- [Integration Tests](../test/integration/tests/session/README.md)

## Appendix: Example File Inventory

### MCP Integration Examples

```
examples/mcp-integration/
├── basic-demo/main.go              # 200 lines - RFC9421 demo
├── basic-tool/
│   ├── main.go                     # 180 lines - MCP server
│   └── calculator_tool.go          # 150 lines - Tool impl
├── client/
│   ├── main.go                     # 220 lines - MCP client
│   └── sage_client.go              # 190 lines - SAGE client
├── simple-standalone/main.go       # 160 lines - Minimal MCP
├── vulnerable-vs-secure/
│   ├── attacker/main.go            # 140 lines - Attack demo
│   ├── secure-chat/main.go         # 250 lines - Secure chat
│   └── vulnerable-chat/main.go     # 230 lines - Vulnerable chat
└── test_compile.sh                 # Build verification script
```

**Total:** ~1,700 lines of example code
**Transport Usage:** 0 files
**Migration Required:** 0 files

---

**Document Version:** 1.0
**Last Updated:** January 2025
**Status:** Complete - No Action Required
