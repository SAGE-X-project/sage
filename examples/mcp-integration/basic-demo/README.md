# Basic SAGE + MCP Demo

This is a self-contained demo showing how SAGE secures MCP tools using cryptographic signatures and agent verification.

## What This Demo Shows

1. **Calculator Tool** - A simple MCP tool that performs arithmetic
2. **SAGE Security** - All requests must be signed by trusted agents
3. **Agent Trust** - Only registered agents can use the tool
4. **Attack Prevention** - Unauthorized requests are rejected

## Run the Demo

```bash
go run main.go
```

The demo will:
1. Start a calculator tool server on port 8080
2. Create three demo agents (Alice, Bob, and Eve)
3. Register Alice and Bob as trusted (Eve remains untrusted)
4. Run automatic test requests showing:
   -  Alice's request succeeds (trusted)
   -  Bob's request succeeds (trusted)
   -  Eve's request fails (untrusted)
   -  Anonymous request fails (no signature)

## Key Security Features

### Without SAGE (Vulnerable)
```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Anyone can call this!
    processRequest(r)
}
```

### With SAGE (Secure)
```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := verifyRequest(r); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }
    // Only verified agents can proceed
    processRequest(r)
}
```

## How It Works

1. **Agent Registration**: Each agent has a cryptographic key pair and DID
2. **Request Signing**: Agents sign their requests using RFC-9421
3. **Signature Verification**: The tool verifies signatures before processing
4. **Trust Management**: Only pre-registered agents are accepted

## Test Manually

### Trusted Agent Request (will succeed)
The demo automatically makes signed requests. To test manually, you need to:
1. Generate an Ed25519 key pair
2. Register the public key with the calculator
3. Sign your request using RFC-9421
4. Include the signature in headers

### Untrusted Request (will fail)
```bash
curl -X POST http://localhost:8080/calculator \
  -H "Content-Type: application/json" \
  -d '{"tool":"calculator","operation":"add","arguments":{"a":5,"b":3}}'
```

This fails because it's missing the required SAGE signature.

## Next Steps

- See the [client example](../client/) for how to build SAGE-enabled agents
- Check the [simple wrapper](../sage-wrapper/) for easy integration
- Read about [RFC-9421](../../../docs/core/rfc9421-en.md) for signature details