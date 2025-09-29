# MCP + SAGE Integration Examples

This directory contains examples showing how to integrate SAGE (Secure Agent Guarantee Engine) with MCP (Model Context Protocol) tools to add cryptographic security to AI agent interactions.

## Why SAGE?

Without SAGE, MCP tools are vulnerable to:
- 🚨 **Identity spoofing** - Any agent can pretend to be another
- 🚨 **Message tampering** - Requests can be modified in transit  
- 🚨 **Replay attacks** - Old requests can be resent
- 🚨 **Unauthorized access** - No verification of agent capabilities

SAGE solves these problems by adding:
-  **Cryptographic signatures** on every request
-  **Blockchain-verified agent identities** (DIDs)
-  **Capability-based access control**
-  **Replay attack protection**

## Examples

### 1. [basic-demo/](./basic-demo/) - Self-Contained Calculator Demo
A complete working demo that runs out of the box:
- Calculator tool with SAGE security
- Three demo agents (Alice, Bob, Eve)
- Automatic test scenarios
- Shows trusted vs untrusted agents

**Run it:**
```bash
cd basic-demo
go run main.go
# Server starts on :8080 and runs automatic demos
```

### 2. [simple-standalone/](./simple-standalone/) - Simple Standalone Example
A minimal example showing the security difference:
- Side-by-side comparison of insecure vs secure endpoints
- Automatic demo request after startup
- Clear demonstration of SAGE benefits

**Run it:**
```bash
cd simple-standalone
go run main.go
# Server starts on :8082
```

### 3. [basic-tool/](./basic-tool/) - Full Calculator Implementation
Complete MCP tool with production-ready features:
- Full DID integration with blockchain
- Capability-based access control  
- Request and response signing
- Tool definition for MCP compatibility

**Note:** This requires blockchain connection for DID resolution.

### 4. [vulnerable-vs-secure/](./vulnerable-vs-secure/) - Security Demonstration
Live demonstration showing:
- Vulnerable chat server without protection
- Attack scenarios (identity spoofing, injection, replay)
- Same server protected with SAGE
- How SAGE blocks all attacks

**Run the demo:**
```bash
# Terminal 1: Start vulnerable server
cd vulnerable-vs-secure/vulnerable-chat
go run .

# Terminal 2: Run attacks
cd ../attacker
go run .
# See how attacks succeed!

# Terminal 3: Start secure server  
cd ../secure-chat
go run .

# Terminal 4: Try attacks on secure server
cd ../attacker
go run . --secure
# All attacks blocked!
```

### 5. [client/](./client/) - AI Agent Client
Example AI agent that calls SAGE-protected tools:
- Key pair generation
- Request signing with RFC-9421
- Error handling
- Response verification

### 6. Additional Resources
- **multi-agent/** - Multi-agent communication patterns (planned)
- **QUICKSTART.md** - Quick start guide for getting started fast

## Quick Start

### For Tool Developers
Add SAGE to your existing MCP tool in seconds:

```go
// Add SAGE verification to any handler
func handleToolRequest(w http.ResponseWriter, r *http.Request) {
    // Check for required SAGE headers
    if r.Header.Get("X-Agent-DID") == "" {
        http.Error(w, "Missing X-Agent-DID", 400)
        return
    }
    if r.Header.Get("Signature") == "" {
        http.Error(w, "Missing signature", 401)
        return
    }
    
    // In production: verify signature with RFC-9421
    // See basic-demo for full example
    
    // Your existing code continues here
    processRequest(r)
}
```

### For AI Agent Developers
Make secure tool calls - see the [client example](./client/) for a complete implementation.

## Architecture

```
┌─────────────┐  Signed Request   ┌─────────────┐
│  AI Agent   │ ────────────────► │  MCP Tool   │
│ (has DID)   │                   │(SAGE-enabled)│
└─────────────┘ ◄──────────────── └─────────────┘
                 Signed Response

                   SAGE Layer 
         ┌─────────────────────────────┐
         │ • Verify signatures         │
         │ • Resolve DIDs              │
         │ • Check capabilities        │
         │ • Prevent replay attacks    │
         └─────────────────────────────┘
```

## Integration Patterns

### 1. Minimal Integration (3 lines)
```go
if err := sage.VerifyRequest(r); err != nil {
    http.Error(w, "Unauthorized", 401)
    return
}
```

### 2. With Capability Checking
```go
if err := sage.VerifyRequest(r); err != nil {
    http.Error(w, "Unauthorized", 401)
    return
}

agentDID := r.Header.Get("X-Agent-DID")
if !sage.HasCapability(agentDID, "execute_trades") {
    http.Error(w, "Forbidden", 403)
    return
}
```

### 3. Full Integration with Response Signing
```go
// Verify request
if err := sage.VerifyRequest(r); err != nil {
    http.Error(w, "Unauthorized", 401)
    return
}

// Process request
result := processRequest(r)

// Sign response
sage.SignResponse(w, r)

// Send response
json.NewEncoder(w).Encode(result)
```

## Security Benefits

| Attack Vector | Without SAGE | With SAGE |
|--------------|--------------|-----------|
| Identity Spoofing |  Any agent can claim any identity |  Cryptographically verified |
| Message Tampering |  Requests can be modified |  Signature verification |
| Replay Attacks |  Old requests can be resent |  Timestamp validation |
| Unauthorized Access |  No capability checking |  Blockchain-verified permissions |

## Next Steps

1. **Try the examples** - Start with the simple integration
2. **Run the security demo** - See the vulnerabilities SAGE prevents
3. **Integrate into your tools** - Use the provided wrapper
4. **Read the docs** - Check out the [SAGE documentation](../../../docs/)

## Questions?

- See the [SAGE documentation](../../../docs/did/)
- Check the [RFC-9421 implementation](../../../docs/core/rfc9421-en.md)
- Review the [architecture docs](../../../docs/architecture/)