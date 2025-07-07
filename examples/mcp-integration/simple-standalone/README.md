# Simple Standalone SAGE Integration Example

This is a self-contained example showing how to add SAGE security to an MCP tool.

## Run the Example

```bash
go run main.go
```

The server will start on port 8082 and automatically make a signed demo request after 2 seconds.

## Test Manually

### Insecure Endpoint (Vulnerable)
```bash
curl -X POST http://localhost:8082/weather-insecure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"San Francisco"}}'
```

This works without any authentication - DANGEROUS!

### Secure Endpoint (Protected)
```bash
curl -X POST http://localhost:8082/weather-secure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"San Francisco"}}'
```

This will fail with "Unauthorized" because it's missing the SAGE signature.

## Key Differences

### Without SAGE (3 lines):
```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Process request directly - NO SECURITY!
    processRequest(r)
}
```

### With SAGE (6 lines):
```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := verifySAGERequest(r); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }
    processRequest(r)
}
```

That's it! Just 3 extra lines add complete cryptographic security.