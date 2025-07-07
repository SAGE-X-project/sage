# Quick Start Guide - MCP + SAGE Integration

## 🚀 Fastest Way to See SAGE in Action

### 1. Run the Basic Demo (Recommended)
This demo runs completely standalone with no external dependencies:

```bash
cd basic-demo
go run main.go
```

You'll see:
- A calculator tool secured with SAGE
- Three demo agents (Alice, Bob, Eve) 
- Automatic test scenarios showing:
  - ✅ Trusted agents succeed
  - ❌ Untrusted agents fail
  - ❌ Anonymous requests fail

### 2. Try the Simple Standalone Example
Shows the difference between insecure and secure endpoints:

```bash
cd simple-standalone  
go run main.go
```

Then test with curl:
```bash
# Insecure endpoint (works but dangerous!)
curl -X POST http://localhost:8082/weather-insecure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"NYC"}}'

# Secure endpoint (requires SAGE signature)
curl -X POST http://localhost:8082/weather-secure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"NYC"}}'
# This fails with "Unauthorized" - as it should!
```

## 📚 More Examples

### Security Demonstration
See real attacks and how SAGE stops them:

```bash
# Terminal 1: Start vulnerable server
cd vulnerable-vs-secure/vulnerable-chat
go run .

# Terminal 2: Run attacks (they succeed - bad!)
cd ../attacker
go run .

# Terminal 3: Start secure server
cd ../secure-chat  
go run .

# Terminal 4: Run attacks on secure server (they fail - good!)
cd ../attacker
go run . --secure
```

## 🔧 Integration Guide

### Adding SAGE to Your Tool (3 lines)

Before (insecure):
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    processRequest(r) // Anyone can call this!
}
```

After (secure):
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := verifySAGERequest(r); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }
    processRequest(r) // Only verified agents!
}
```

## 🎯 Key Concepts

1. **DIDs** - Every agent has a decentralized identifier
2. **Signatures** - All requests are cryptographically signed
3. **Verification** - Tools verify signatures before processing
4. **Trust** - Only registered agents are accepted

## 📖 Learn More

- [Full Documentation](../../../docs/)
- [RFC-9421 Details](../../../docs/core/rfc9421-en.md)
- [DID System](../../../docs/did/)